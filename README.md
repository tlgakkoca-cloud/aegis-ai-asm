# Aegis AI Attack Surface Manager (ASM)

Aegis AI ASM, siber güvenlik ekiplerinin bulut ve SaaS varlıklarını otomatik olarak keşfedip skorlamasını, politika ihlallerini tespit etmesini ve operatörlere aksiyon alınabilir içgörüler vermesini sağlayan Go tabanlı bir platformdur.

## Özellikler
- **Varlık keşfi:** Çoklu kaynaklardan gelen envanteri normalize eder ve tekil graf yapısında tutar.
- **Attack surface skorlaması:** Maruziyet, zafiyet, keşfedilebilirlik ve blast radius metriklerini hesaplar.
- **Politika gardiyanları:** YAML tabanlı politikalarla yanlış yapılandırmaları yakalar.
- **Telemetri omurgası:** Her bileşen audit-grade olaylar üretir ve güvenli şekilde kuyruğa iletir.
- **CLI operatör deneyimi:** `aegisctl` aracı yüksek öncelikli risk raporlarını sunar.

## Başlangıç

### Önkoşullar
- Go 1.22+
- Make

### Kurulum
```bash
cp .env.example .env
make deps
```
> `.env` dosyasında **TARGET_DOMAIN** (keşfedilecek kök alan adı) ve **API_KEY** alanlarının dolu olduğundan emin ol; boş bırakılırsa uygulama başlangıçta hata verip kapanır.

### Çalıştırma / Geliştirme
| Komut | Açıklama |
|-------|---------|
| `make run` | `cmd/aegisctl` giriş noktasını `go run` ile çalıştırır. |
| `make build` | Tüm modülleri derler. |
| `make test` | Paket testlerini çalıştırır. |
| `make lint` | `go fmt` + `go vet` ile temel statik kontrolleri uygular. |
| `make fmt` | Tüm Go dosyalarını `gofmt` ile formatlar. |

`make run` sonrasında CLI, `.env` dosyasındaki ayarları yükler ve temel başlangıç mesajını basar. İlerleyen sprintlerde servis bootstrap süreci buradan genişletilecektir.

## Yapı
```
cmd/           # CLI giriş noktaları
internal/      # Domain mantığı ve uygulama paketleri
docs/          # Roadmap ve teknik dökümanlar
pkg/           # Paylaşılan yardımcı paketler (harici kullanım için güvenli)
```

## Katkı Beklentileri
- Kod PR’ları, Aegis-TL tarafından tanımlanan clean-code standartlarına uyacak.
- Yeni bağımlılıklar güvenlik taramasından geçirilecek.
- Her modül için test + dokümantasyon şart.
