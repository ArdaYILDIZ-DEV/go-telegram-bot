<h1 align="center">🤖 Go Telegram Asistan Botu</h1>
<p align="center">
  <strong>Go ile yazılmış, sunucunuzu Telegram üzerinden yönetmenizi sağlayan yüksek performanslı ve modüler bir otomasyon asistanı.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white">
  <img src="https://img.shields.io/badge/FFmpeg-00780B?style=for-the-badge&logo=ffmpeg&logoColor=white">
  <img src="https://img.shields.io/badge/yt--dlp-838383?style=for-the-badge&logo=youtube&logoColor=white">
  <a href="https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/blob/main/LICENSE"><img src="https://img.shields.io/github/license/ArdaYILDIZ-DEV/go-telegram-asistan?style=for-the-badge&color=informational"></a>
</p>

---

## 📖 Hakkında
**Go Telegram Asistan Botu**, sadece komut çalıştıran basit bir botun çok ötesindedir.  
Arka planda çalışan zamanlayıcılar, dosya sistemi izleyicileri ve Go’nun eşzamanlılık gücü sayesinde,  
sunucunuzla proaktif bir şekilde etkileşim kurar. Dosyalarınızı organize eder, ağ sorunlarını bildirir  
ve uzun süren görevleri arka planda halleder.

> Amaç: Sunucu kontrolü, güvenlik ve otomasyonu doğrudan Telegram üzerinden sağlamak.

---

## ✨ Öne Çıkan Özellikler

### 📂 Gelişmiş Dosya Yönetimi
- **Listeleme ve Arama**
  - `/liste` → Ana dizindeki dosyaları listeler.
  - `/klasor <kategori>` → Kategori altındaki dosyaları gösterir.
  - `/ara <kelime>` → Tüm alt klasörlerde arama yapar.
- **Dosya Transferi & Manipülasyonu**
  - `/getir <dosya>` → Sunucudaki dosyayı gönderir.
  - `/sil <dosya>` → Inline buton ile güvenli silme.
  - `/yenidenadlandir <eski> <yeni>` → Dosya adını değiştirir, açıklamayı korur.
- **Metadata Sistemi**
  - `/aciklama_ekle <dosya> <açıklama>` → Kalıcı açıklama ekler.
  - `/aciklama_ara <kelime>` → Açıklamalar içinde arama yapar.

---

### 🖥️ Kapsamlı Sistem Kontrolü
- **Görev Yöneticisi**
  - `/gorevler` → Sayfalı, sıralanabilir işlem listesi.
  - `/kapat <PID>` → İşlemi durdurur.
- **Güvenli Komut Çalıştırma**
  - `/calistir <yol> <süre>` → Timeout ile güvenli betik çalıştırma.
- **Ekran Görüntüsü ve Kayıt**
  - `/ss` → Anlık ekran görüntüsü.
  - `/kayit_al` / `/kayit_durdur` → FFmpeg ile ekran kaydı.

---

### 🎬 Medya İndirme ve İşleme
- **Akıllı İndirme Motoru**
  - `/indir <URL> [kalite] [format]` → Video indirme.
  - `/indir_ses <URL> [format]` → Sadece ses indirir.
- **Video Düzenleme**
  - `/kes <dosya> <baş> <bitiş>` → Yeniden kodlamadan hızlı kesme.
- **GIF Üretimi**
  - `/gif_yap <dosya> <baş> <bitiş>` → Optimize edilmiş GIF oluşturur.

---

### ⚙️ Akıllı Otomasyonlar
- **Magic Folder**
  - `TelegramaGonder` klasörüne atılan dosyaları otomatik gönderir ve siler.
- **Otomatik Raporlama**
  - Saatlik sistem + hız testi raporu.
- **Ağ İzleme**
  - `/izle` → İnternet kesildiğinde ve geri geldiğinde bildirim.
  - **Port Monitörü** → `/portlar` ile izleme.

---

## 🏗️ Mimari ve Tasarım Felsefesi
> "Sorumlulukların ayrılması" ve "engellemesiz operasyon" prensipleri ile geliştirilmiştir.
- **Modülerlik** → Her dosya tek bir sorumluluk alanına odaklanır.
- **Eşzamanlılık** → `goroutine` ve `channel` ile aynı anda birden çok işlem.
- **Durum Güvenliği** → `sync.Mutex` ile veri erişim güvenliği.
- **Harici Araç Entegrasyonu** → `yt-dlp`, `ffmpeg` vb. araçlarla güçlü entegrasyon.

---

## 🚀 Kurulum

### Gereksinimler
- **Go** → `1.18` veya üzeri
- **Harici Araçlar**
  - [yt-dlp](https://github.com/yt-dlp/yt-dlp)
  - [FFmpeg](https://ffmpeg.org/download.html)
  - [Speedtest CLI](https://www.speedtest.net/apps/cli)


### Adımlar
```bash
# Projeyi klonla
git clone https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan.git
cd go-telegram-asistan

# Go modüllerini indir
go mod tidy

# .env dosyasını oluştur ve aşağıdaki formatta doldur
BOT_TOKEN=123456:ABC-DEF
ADMIN_CHAT_ID=9876543210
ALLOWED_IDS=112233,445566
BASE_DIR=C:/BotDosyalari
MONITORED_PORTS=80,443,3306

# Botu çalıştır
go run .

# Veya derleyip çalıştır
go build -o asistan && ./asistan

```

📜 Tüm Komutlar
<details> <summary>Komut Listesi</summary>
| Komut                               | Açıklama                    |
| :---------------------------------- | :-------------------------- |
| `/start`                            | Hoş geldin mesajı           |
| `/help`                             | Komut listesi               |
| `/getir <dosya>`                    | Dosya gönder                |
| `/sil <dosya>`                      | Dosya sil (onay ile)        |
| `/yenidenadlandir <eski> <yeni>`    | Dosya adı değiştir          |
| `/tasi <dosya> <klasör>`            | Dosya taşı                  |
| `/ara <kelime>`                     | Dosya arama                 |
| `/liste`                            | Ana klasördeki dosyalar     |
| `/klasor <kategori>`                | Kategori klasörünü listeler |
| `/aciklama_ekle <dosya> <açıklama>` | Açıklama ekle               |
| `/aciklama_sil <dosya>`             | Açıklama sil                |
| `/aciklamalar`                      | Tüm açıklamaları listeler   |
| `/aciklama_ara <kelime>`            | Açıklama arama              |
| `/indir <URL> [kalite] [format]`    | Video/dosya indir           |
| `/indir_ses <URL> [format]`         | Ses indir                   |
| `/kes <dosya> <baş> <bitiş>`        | Video kes                   |
| `/gif_yap <dosya> <baş> <bitiş>`    | GIF yap                     |
| `/gorevler`                         | Görev yöneticisi            |
| `/calistir <yol> <süre>`            | Betik çalıştır              |
| `/kapat <PID>`                      | İşlem durdur                |
| `/duzenle`                          | Dosyaları düzenle           |
| `/durum`                            | Sistem durumu               |
| `/sistem_bilgisi`                   | Ayrıntılı sistem bilgisi    |
| `/hiz_testi`                        | İnternet hızı testi         |
| `/portlar`                          | Port kontrol                |
| `/ss`                               | Ekran görüntüsü             |
| `/kayit_al` / `/kayit_durdur`       | Ekran kaydı                 |
| `/izle`                             | Ağ izleme                   |

🛠️ Kullanılan Teknolojiler
<p> <img src="https://skillicons.dev/icons?i=go,git,github,powershell,vscode" /> <img src="https://repository-images.githubusercontent.com/947861912/79d2548e-a5dc-420e-8fda-3e9368a7b668" alt="FFmpeg" height="48"> <img src="https://repository-images.githubusercontent.com/307260205/b6a8d716-9c7b-40ec-bc44-6422d8b741a0" alt="yt-dlp" height="48"> </p>

🤝 Katkıda Bulunma

1.Fork’la

2.Yeni dal oluştur → git checkout -b feature/YeniOzellik

3.Değişiklikleri yap ve commit’le

4.Dalını push’la

5.Pull Request gönder

📜 Lisans

Bu proje MIT Lisansı ile lisanslanmıştır.
