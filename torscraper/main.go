package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/net/proxy"
)

var (
	active   []string
	inactive []string
	logFile  *os.File
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Kullanım:")
		fmt.Println("  go run main.go <targets.yaml>")
		fmt.Println("  go run main.go <url>")
		os.Exit(1)
	}

	os.MkdirAll("output/html", 0755)
	logFile, _ = os.Create("output/scan_report.log")
	defer logFile.Close()

	input := strings.TrimSpace(os.Args[1])
	var urls []string

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		urls = []string{input}
		fmt.Printf("[INFO] Tek URL modu: %s\n", input)
	} else {
		data, err := os.ReadFile(input)
		if err != nil {
			log.Fatal(err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				urls = append(urls, line)
			}
		}
		fmt.Printf("[INFO] Dosya modu: %s (Toplam %d URL bulundu)\n", input, len(urls))
	}

	writeLog(fmt.Sprintf("=== Tarama Başladı: %s ===\n", time.Now().Format("2006-01-02 15:04:05")))
	writeLog(fmt.Sprintf("Toplam URL sayısı: %d\n\n", len(urls)))

	client := setupTor()
	checkIP(client)

	for i, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		fmt.Printf("[%d/%d] Tarama: %s\n", i+1, len(urls), url)
		scan(client, url)
		time.Sleep(2 * time.Second)
	}

	printReport()
}

func setupTor() *http.Client {
	ports := []string{"127.0.0.1:9050", "127.0.0.1:9150"}

	for i, p := range ports {
		conn, err := net.DialTimeout("tcp", p, 2*time.Second)
		if err == nil {
			conn.Close()
			if i == 0 {
				fmt.Printf("[WARN] 9050 portu çalışmıyor, 9150 deneniyor...\n")
			} else {
				fmt.Printf("[INFO] Tor Browser proxy (9150) kullanılıyor\n")
			}
			d, _ := proxy.SOCKS5("tcp", p, nil, proxy.Direct)
			return &http.Client{
				Transport: &http.Transport{
					DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						return d.Dial(network, addr)
					},
				},
				Timeout: 30 * time.Second,
			}
		}
	}

	log.Fatal("Tor proxy bağlantısı kurulamadı (9050 ve 9150 denendi).\n\nÇözüm:\n1. Tor Browser'ı açın ve arka planda çalıştığından emin olun\n2. Veya Tor daemon'ı başlatın\n\nKontrol için:\n  netstat -an | findstr \"9150\"\n  netstat -an | findstr \"9050\"")
	return nil
}

func checkIP(client *http.Client) {
	fmt.Println("\n[INFO] Tor IP kontrolü yapılıyor...")
	resp, err := client.Get("https://check.torproject.org/api/ip")
	if err != nil {
		fmt.Printf("[WARN] IP kontrolü yapılamadı: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[INFO] Tor IP Yanıtı: %s\n\n", string(body))
}

func scan(client *http.Client, url string) {
	start := time.Now()

	resp, err := client.Get(url)
	if err != nil {
		msg := fmt.Sprintf("[ERR] Tarama: %s -> HATA: %v", url, err)
		fmt.Println(msg)
		writeLog(msg + "\n")
		inactive = append(inactive, url)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("[ERR] Tarama: %s -> HTTP %d", url, resp.StatusCode)
		fmt.Println(msg)
		writeLog(msg + "\n")
		inactive = append(inactive, url)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("[ERR] Tarama: %s -> İçerik okunamadı: %v", url, err)
		fmt.Println(msg)
		writeLog(msg + "\n")
		inactive = append(inactive, url)
		return
	}

	fname := makeFilename(url)
	fpath := filepath.Join("output/html", fname+".html")
	os.WriteFile(fpath, body, 0644)

	screenshot(url, fname)

	elapsed := time.Since(start)
	msg := fmt.Sprintf("[SUCCESS] Tarama: %s -> %d byte, %v sürede kaydedildi", url, len(body), elapsed)
	fmt.Println(msg)
	writeLog(fmt.Sprintf("[SUCCESS] %s -> %s (%d byte)\n", url, fpath, len(body)))
	active = append(active, url)
}

func screenshot(url, fname string) {
	proxyAddr := ""
	ports := []string{"127.0.0.1:9050", "127.0.0.1:9150"}

	for _, p := range ports {
		conn, err := net.DialTimeout("tcp", p, 2*time.Second)
		if err == nil {
			conn.Close()
			proxyAddr = "socks5://" + p
			break
		}
	}

	if proxyAddr == "" {
		fmt.Printf("[WARN] Screenshot için Tor proxy bulunamadı, atlanıyor\n")
		return
	}

	ctx, cancel := chromedp.NewExecAllocator(context.Background(),
		chromedp.Flag("proxy-server", proxyAddr),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-logging", true),
		chromedp.Flag("log-level", "3"),
	)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.FullScreenshot(&buf, 90),
	)

	if err != nil {
		fmt.Printf("[WARN] Screenshot alınamadı (%s): %v\n", url, err)
		return
	}

	spath := filepath.Join("output/html", fname+".png")
	os.WriteFile(spath, buf, 0644)
	fmt.Printf("[INFO] Screenshot kaydedildi: %s\n", spath)
	writeLog(fmt.Sprintf("[SCREENSHOT] %s -> %s\n", url, spath))
}

func makeFilename(url string) string {
	f := strings.TrimPrefix(url, "http://")
	f = strings.TrimPrefix(f, "https://")

	for _, ch := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		f = strings.ReplaceAll(f, ch, "_")
	}

	if len(f) > 100 {
		f = f[:100]
	}

	ts := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s", f, ts)
}

func writeLog(msg string) {
	if logFile != nil {
		logFile.WriteString(msg)
	}
}

func printReport() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("TARAMA ÖZETİ")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Aktif URL'ler: %d\n", len(active))
	fmt.Printf("Pasif/Hatalı URL'ler: %d\n", len(inactive))
	fmt.Printf("Toplam: %d\n", len(active)+len(inactive))
	fmt.Println(strings.Repeat("=", 50))

	writeLog("\n=== Tarama Özeti ===\n")
	writeLog(fmt.Sprintf("Aktif URL'ler: %d\n", len(active)))
	writeLog(fmt.Sprintf("Pasif/Hatalı URL'ler: %d\n", len(inactive)))
	writeLog(fmt.Sprintf("Toplam: %d\n", len(active)+len(inactive)))

	if len(active) > 0 {
		writeLog("\nAktif URL'ler:\n")
		for _, url := range active {
			writeLog(fmt.Sprintf("  - %s\n", url))
		}
	}

	if len(inactive) > 0 {
		writeLog("\nPasif/Hatalı URL'ler:\n")
		for _, url := range inactive {
			writeLog(fmt.Sprintf("  - %s\n", url))
		}
	}

	writeLog(fmt.Sprintf("\n=== Tarama Bitti: %s ===\n", time.Now().Format("2006-01-02 15:04:05")))
	fmt.Printf("\n[INFO] Detaylı rapor: output/scan_report.log\n")
	fmt.Printf("[INFO] HTML dosyaları: output/html/\n")
}
