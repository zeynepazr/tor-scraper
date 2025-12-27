# TOR Scraper

Go (Golang) ile yazılmış, Tor ağı üzerinden .onion adreslerini tarayan otomasyon aracı.

## Özellikler

- ✅ Tor SOCKS5 proxy desteği (9050 ve 9150 portları)
- ✅ Toplu URL tarama (YAML dosyasından)
- ✅ IP sızıntısı önleme
- ✅ Hata toleransı (ölü siteler programı durdurmaz)
- ✅ HTML içeriği kaydetme
- ✅ Screenshot (PNG) kaydetme
- ✅ Detaylı loglama
- ✅ Tarama özet raporu

## Gereksinimler

1. **Go (Golang)**: 1.21 veya üzeri
2. **Tor Servisi**: 
   - Tor daemon (port 9050) veya
   - Tor Browser (port 9150)
3. **Chrome/Chromium**: Screenshot özelliği için gerekli (otomatik olarak indirilir veya sisteminizde yüklü olmalı)

## Kurulum

1. Projeyi klonlayın veya indirin:
```bash
cd torscraper
```

2. Go bağımlılıklarını yükleyin:
```bash
go mod download
```

## Kullanım

1. Tor servisinin çalıştığından emin olun:
   - Tor daemon için: `tor` komutu veya systemd servisi
   - Tor Browser için: Tor Browser'ı açık tutun

2. `targets.yaml` dosyasını düzenleyin ve taramak istediğiniz .onion adreslerini ekleyin:
```yaml
http://example1.onion
http://example2.onion
http://example3.onion
```

3. Programı çalıştırın:

**Dosyadan URL listesi okuma:**
```bash
go run main.go targets.yaml
```

**Tek bir URL tarama:**
```bash
go run main.go http://example.onion
```

## Çıktılar

Program çalıştıktan sonra aşağıdaki dosyalar oluşturulur:

- `output/html/`: Her başarılı taramadan alınan HTML dosyaları
- `output/screenshots/`: Her başarılı taramadan alınan PNG screenshot dosyaları
- `output/scan_report.log`: Detaylı tarama raporu (aktif/pasif URL'ler, hatalar, vs.)

**Not:** Sonuçları görüntülemek için:
```powershell
explorer output\html      # HTML dosyaları
explorer output\screenshots  # Screenshot dosyaları
```

HTML dosyalarını tarayıcıda açabilir veya PNG screenshot dosyalarını doğrudan görüntüleyebilirsiniz.

## Örnek Çıktı

```
[INFO] Toplam 5 URL bulundu
[INFO] Tor daemon proxy (9050) kullanılıyor

[INFO] Tor IP kontrolü yapılıyor...
[INFO] Tor IP Yanıtı: {"IP":"123.456.789.0",...}

[1/5] Tarama: http://example1.onion
[SUCCESS] Tarama: http://example1.onion -> 45678 byte, 2.3s sürede kaydedildi

[2/5] Tarama: http://example2.onion
[ERR] Tarama: http://example2.onion -> HATA: timeout

...

==================================================
TARAMA ÖZETİ
==================================================
Aktif URL'ler: 3
Pasif/Hatalı URL'ler: 2
Toplam: 5
==================================================
```

## Güvenlik Notları

- Bu araç sadece eğitim ve yasal amaçlar için kullanılmalıdır
- Tor ağında bulunan içeriklerin birçoğu yasadışı olabilir
- Sadece kendi sahip olduğunuz veya izin aldığınız siteleri tarayın
- IP sızıntısını önlemek için özel HTTP transport yapılandırması kullanılmaktadır

## Gelişmiş Kullanım

### Binary Olarak Derleme

Windows için:
```bash
go build -o torscraper.exe main.go
```

Linux/Mac için:
```bash
go build -o torscraper main.go
```

Derlenmiş binary'i kullanma:
```bash
./torscraper targets.yaml
```

## Modüller

1. **Dosya Okuma Modülü**: `targets.yaml` dosyasını okur ve URL listesini çıkarır
2. **Tor Proxy Yönetimi**: SOCKS5 proxy üzerinden HTTP istekleri yapar
3. **İstek ve Hata Yönetimi**: Hataları yakalar ve bir sonraki URL'e geçer
4. **Veri Kayıt Modülü**: HTML içeriğini dosyalara kaydeder ve log tutar

## Teknik Detaylar

- **HTTP Timeout**: 30 saniye
- **Rate Limiting**: Her istek arasında 2 saniye bekleme
- **Proxy Portları**: Önce 9050 (Tor daemon), sonra 9150 (Tor Browser) denenir
- **Dosya Adlandırma**: URL'den türetilmiş güvenli dosya adları + timestamp


