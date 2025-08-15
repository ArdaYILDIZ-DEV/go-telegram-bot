# Go Telegram Asistan Botu

<p align="center">
  <strong>Go ile yazılmış, sunucunuzu Telegram üzerinden yönetmek için tasarlanmış, yüksek performanslı ve modüler bir otomasyon aracı.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/FFmpeg-00780B?style=for-the-badge&logo=ffmpeg&logoColor=white" alt="FFmpeg">
  <img src="https://img.shields.io/badge/yt--dlp-838383?style=for-the-badge&logo=youtube&logoColor=white" alt="yt-dlp">
  <img src="https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=for-the-badge">
</p>

---


### İçindekiler
1. [Proje Hakkında](#-proje-hakkında)
2. [Özelliklerin Derinlemesine İncelenmesi](#-özelliklerin-derinlemesine-incelenmesi)
   - [Dosya Yönetimi](#-gelişmiş-dosya-yönetimi)
   - [Sistem Kontrolü](#-kapsamlı-sistem-kontrolü)
   - [Medya İşlemleri](#-medya-indirme-ve-işleme)
   - [Akıllı Otomasyonlar](#-akıllı-otomasyonlar)
3. [Mimarî ve Tasarım Felsefesi](#-mimarî-ve-tasarım-felsefesi)
4. [Kurulum ve Başlangıç](#-kurulum-ve-başlangıç)
   - [Ön Gereksinimler](#ön-gereksinimler)
   - [Kurulum Adımları](#kurulum-adımları)
5. [Yapılandırma Detayları](#-yapılandırma-detayları)
6. [Kod Mimarisine Genel Bakış](#-kod-mimarisine-genel-bakış)
7. [Katkıda Bulunma](#-katkıda-bulunma)
8. [Lisans](#-lisans)

---

### 📖 Proje Hakkında

**Go Telegram Asistan Botu**, sadece komut çalıştıran basit bir botun çok ötesindedir. Arka planda çalışan zamanlayıcılar, dosya sistemi olaylarını anlık olarak dinleyen izleyiciler ve Go'nun eşzamanlılık (concurrency) gücü sayesinde, sunucunuzla proaktif bir şekilde etkileşim kuran kişisel bir asistandır. Dosyalarınızı organize eder, ağ sorunlarını size bildirir ve uzun süren görevleri sizi engellemeden arka planda halleder.

> Bu proje, bir sunucu üzerindeki kontrolü, güvenliği ve otomasyonu doğrudan Telegram arayüzüne taşıyarak uzaktan yönetimi kolaylaştırmak amacıyla geliştirilmiştir.

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>

---

## ✨ Özelliklerin Derinlemesine İncelenmesi

### 📁 Gelişmiş Dosya Yönetimi
- **Listeleme ve Arama:**
  - `/liste` & `/klasor`: Standart listeleme.
  - `/ara <kelime>`: `filepath.Walk` kullanarak **tüm alt klasörlerde** rekürsif bir arama yapar ve bulunan dosyaların göreceli yollarını listeler.
- **Güvenli Dosya Operasyonları:**
  - `/sil <dosya>`: Silme gibi geri alınamaz bir işlem için **inline butonlar ile onay mekanizması** sunar. Bu, yanlışlıkla veri kaybını önler.
  - `/yenidenadlandir <eski> <yeni>`: Bir dosyanın adını değiştirirken, o dosyaya ait metadata (açıklama) varsa, bu veriyi kaybetmeden yeni dosyaya aktarır.
- **Kalıcı Metadata Sistemi:**
  - `/aciklama_ekle`: Dosyalara eklenen açıklamalar, bot yeniden başlasa bile kaybolmayacak şekilde `metadata.json` dosyasında kalıcı olarak saklanır.
  - `/aciklama_ara`: Aramayı sadece dosya adlarıyla sınırlamaz, **tüm açıklamaların içeriğinde** de arama yaparak güçlü bir içerik tabanlı bulma yeteneği sunar.

### 🖥️ Kapsamlı Sistem Kontrolü
- **İnteraktif Görev Yöneticisi:**
  - `/gorevler`: `gopsutil` kütüphanesini kullanarak sistemdeki işlemleri listeler. Arayüz, **sayfalı (paginated)** ve **sıralanabilir** yapıdadır. Her buton tıklaması, yeni bir `CallbackQuery` ile botun mesajı düzenlemesini tetikleyerek dinamik bir kullanıcı deneyimi sağlar.
- **Zaman Aşımı Korumalı Betik Yürütme:**
  - `/calistir <yol> <süre>`: Bu komut, Go'nun `context.WithTimeout` özelliğini kullanarak çalıştırılan betiğe bir "yaşam süresi" tanımlar. Eğer betik belirlenen sürede tamamlanmazsa, işlem otomatik olarak sonlandırılır. Bu, sunucuyu kilitleyebilecek sonsuz döngüdeki betiklere karşı koruma sağlar.
- **Platforma Özel Ekran Kontrolü:**
  - `/ss`: Windows üzerinde, harici bir programa ihtiyaç duymadan, doğrudan PowerShell'in .NET kütüphanelerine erişim yeteneğini kullanarak anlık ekran görüntüsü alır.
  - `/kayit_al`: Arka planda `FFmpeg` işlemini başlatır ve işlemin standart girdisine (`stdin`) bir `pipe` bağlar. `/kayit_durdur` komutu, bu `pipe`'a 'q' karakterini yazarak FFmpeg'in kaydı düzgün bir şekilde sonlandırıp dosyayı tamamlamasını sağlar.

### 🌐 Medya İndirme ve İşleme
- **Akıllı ve Esnek İndirme Motoru:**
  - `/indir`: Kullanıcının belirttiği kalite ve format tercihlerini bir "öncelik zinciri" haline getirerek `yt-dlp`'ye `-f` parametresi olarak sunar. Örneğin, `bestvideo[height<=?1080][ext=mp4]+bestaudio/best`. Bu, en uygun formatın bulunmasını garanti altına alır. İndirme süresince, `yt-dlp`'nin `--progress` çıktısı anlık olarak okunur, parse edilir ve Telegram mesajı düzenlenerek kullanıcıya ilerleme durumu bildirilir.
- **Yüksek Hızlı, Kayıpsız Video Düzenleme:**
  - `/kes`: Bu komutun hızı, FFmpeg'in `-c copy` (codec: copy) parametresinden gelir. Bu parametre, videoyu yeniden kodlamak (re-encoding) yerine, mevcut video ve ses akışlarını belirtilen zaman aralığında doğrudan kopyalar. Bu sayede işlem, CPU yükü olmadan saniyeler içinde tamamlanır.
- **Stüdyo Kalitesinde GIF Üretimi:**
  - `/gif_yap`: Bu komut, basit bir format dönüştürme işleminden daha fazlasını yapar. İki aşamalı bir `filtergraph` kullanır:
    1.  `palettegen`: Önce videonun belirtilen bölümünü analiz ederek en uygun 256 renkten oluşan özel bir renk paleti oluşturur.
    2.  `paletteuse`: Ardından, bu özel paleti kullanarak GIF'i oluşturur. Bu yöntem, renk geçişlerinde oluşan bozulmaları (dithering) en aza indirir ve çok daha canlı, yüksek kaliteli sonuçlar üretir.

### ⚙️ Akıllı Otomasyonlar
- **Olay Tabanlı "Magic Folder":**
  - `TelegramaGonder` klasörü, Go'nun `fsnotify` kütüphanesi ile sürekli izlenir. Bir dosya oluşturulduğunda veya üzerine yazıldığında (`fsnotify.Create`/`fsnotify.Write` olayları), zamanlayıcı anında tetiklenir, dosyayı bir `goroutine` içinde işlemeye alır, gönderir ve siler.
- **Durum (Stateful) Odaklı Bildirimler:**
  - `/izle` ve Port Monitörü, sadece "durum değiştiğinde" bildirim gönderir. Örneğin, internet durumu bir önceki kontrolde "var" iken şimdiki kontrolde de "var" ise, gereksiz bir mesaj gönderilmez. Sadece "yok" durumuna geçtiğinde bildirim atılır. Bu, botun "geveze" olmasını engeller.

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>

---

### 🏛️ Mimarî ve Tasarım Felsefesi
- **Modülerlik:** Her dosya (`auth.go`, `file_manager.go`, `scheduler.go` vb.) tek bir sorumluluk alanına odaklanır. Bu, kodun okunabilirliğini ve bakımını kolaylaştırır.
- **Eşzamanlılık (Concurrency):** `goroutine` ve `channel`'lar, indirme, betik çalıştırma gibi uzun süren işlemlerin botun ana akışını engellemesini önler. Bot, aynı anda birden çok komuta yanıt verebilir.
- **Durum Güvenliği (State Safety):** Paylaşılan verilere (metadata, kayıt durumu vb.) erişim, `sync.Mutex` kilitleri ile korunarak "race condition" hatalarının önüne geçilir.
- **Harici Araç Entegrasyonu:** `yt-dlp`, `ffmpeg` gibi kendini kanıtlamış, güçlü komut satırı araçlarını bir arayüz arkasında birleştirir ve bu araçların en güçlü özelliklerini ortaya çıkarır.

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>

---

## 🚀 Kurulum ve Başlangıç

### Ön Gereksinimler
- **Go:** `1.18` veya üstü.
- **Harici Araçlar:** Botun tüm özelliklerini kullanabilmek için aşağıdaki CLI araçlarının sisteminizde kurulu ve **PATH** ortam değişkenine eklenmiş olması gerekir:
  - **[yt-dlp](https://github.com/yt-dlp/yt-dlp):** Video ve ses indirmek için.
  - **[FFmpeg](https://ffmpeg.org/download.html):** Medya işleme (kesme, GIF yapma, ekran kaydı) için.
  - **[Speedtest CLI](https://www.speedtest.net/apps/cli):** `/hiz_testi` komutu için.

### Kurulum Adımları
1. **Projeyi Klonlayın:**
   ```bash
   git clone https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan.git
   cd go-telegram-asistan
   ```
2. **Go Modüllerini İndirin:**
   ```bash
   go mod tidy
   ```
3. **Yapılandırma Dosyasını Oluşturun:**
   Proje dizinine `.env.example` adında bir şablon dosya eklemeniz önerilir. Kullanıcılar bu dosyayı kopyalayarak kendi `.env` dosyalarını oluşturabilirler.
   ```bash
   # Örnek: cp .env.example .env
   ```
   Ardından `.env` dosyasını kendi bilgilerinizle doldurun.

4. **Botu Çalıştırın:**
   ```bash
   # Sadece çalıştırmak için
   go run .

   # Derleyip kalıcı bir dosya oluşturmak için
   # Windows:
   go build -o asistan.exe && ./asistan.exe
   # Linux/macOS:
   go build -o asistan && ./asistan
   ```

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>

---

### ⚙️ Yapılandırma Detayları
`.env` dosyasında kullanılabilecek değişkenler:

| Değişken | Gerekli? | Açıklama |
| :--- | :---: | :--- |
| `BOT_TOKEN` | ✅ **Evet** | Telegram'da `@BotFather`'dan alacağınız API token'ı. |
| `ADMIN_CHAT_ID` | ✅ **Evet** | Botun tam yetkili yöneticisinin Telegram ID'si. Bu ID olmadan zamanlayıcı gibi özellikler çalışmaz. |
| `ALLOWED_IDS` | ❌ Hayır | Botu kullanabilecek diğer kullanıcıların ID'leri (virgülle ayırın). |
| `BASE_DIR`| ❌ Hayır | Botun çalışacağı ana klasör. Varsayılan olarak `Gelenler` klasörünü oluşturur. |
| `MONITORED_PORTS` | ❌ Hayır | Periyodik olarak izlenecek portlar (virgülle ayırın). |

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>

---

### 🏗️ Kod Mimarisine Genel Bakış

<details>
  <summary><strong>Proje Dosya Yapısı ve Sorumlulukları</strong></summary>

  | Dosya | Sorumluluk Alanı |
  | :--- | :--- |
  | `main.go` | Uygulamanın giriş noktası. Loglama, yapılandırma ve botun ilk başlatma işlemlerini yapar. |
  | `telegram_bot.go` | Botun ana yönlendiricisi (router). Gelen tüm güncellemeleri karşılar, yetki kontrolü yapar ve komutları ilgili işleyicilere dağıtır. |
  | `config.go` | `.env` dosyasından yapılandırmayı okur ve global bir `config` nesnesine yükler. |
  | `auth.go` | Kullanıcı yetkilendirme mantığını içerir (`isUserAdmin`, `isUserAllowed`). |
  | `command_handlers.go` | Komutların ana iş mantığını içeren fonksiyonları barındırır. |
  | `scheduler.go` | Zamanlanmış görevleri (saatlik rapor, port kontrolü) ve dosya sistemi izleyicisini (`fsnotify`) yönetir. |
  | `system_monitor.go` | `gopsutil` ve `speedtest` gibi araçlarla sistem kaynaklarını izleyen fonksiyonları içerir. |
  | `file_manager.go` | Dosya organizasyonu, kategori belirleme ve dosya bulma gibi temel dosya sistemi işlemlerini yönetir. |
  | `metadata_manager.go` | Dosya açıklamalarının `metadata.json` dosyasına yazılması, okunması ve aranmasından sorumludur. |
  | `downloader_command.go`| `yt-dlp` ve standart HTTP indirme işlemlerini, ilerleme takibiyle birlikte yönetir. |
  | `video_processor.go` | `FFmpeg` kullanarak yerel video dosyaları üzerinde kesme ve GIF yapma işlemlerini gerçekleştirir. |
  | `executor_command.go` | Harici betiklerin güvenli bir şekilde (zaman aşımı ile) çalıştırılması ve işlemlerin sonlandırılmasından sorumludur. |
  | `extra_commands.go` | `FFmpeg` ve `PowerShell` gibi harici araçlara dayalı ekran kaydı ve ekran görüntüsü alma komutlarını içerir. |
  
</details>

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>

---
## 🛠️ Kullanılan Teknolojiler

<p align="left">
  <!-- skillicons.dev ile gelenler (yönlendirmesiz) -->
  <img src="https://skillicons.dev/icons?i=go,git,github,powershell,vscode" />
  
  <!-- Manuel olarak eklenen ve yönlendirmesi kaldırılan logolar -->
  <img src="https://repository-images.githubusercontent.com/947861912/79d2548e-a5dc-420e-8fda-3e9368a7b668" alt="FFmpeg" height="48">
  <img src="https://repository-images.githubusercontent.com/307260205/b6a8d716-9c7b-40ec-bc44-6422d8b741a0" alt="yt-dlp" height="48">
</p>

---
### 🤝 Katkıda Bulunma

Bu proje kişisel kullanım için geliştirilmiştir, ancak her türlü fikir, öneri ve katkıya açıktır. Bir hata bulursanız veya yeni bir özellik önermek isterseniz, lütfen bir **[Issue](https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/issues)** açmaktan veya **[Pull Request](https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/pulls)** göndermekten çekinmeyin.

1. Projeyi **Fork**'layın.
2. Yeni bir özellik dalı oluşturun (`git checkout -b feature/YeniHarikaOzellik`).
3. Değişikliklerinizi yapın ve **Commit**'leyin (`git commit -m 'feat: Yeni bir harika özellik eklendi'`).
4. Dalınızı **Push**'layın (`git push origin feature/YeniHarikaOzellik`).
5. Bir **Pull Request** oluşturun.

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>

---

### 📜 Lisans

Bu proje [MIT Lisansı](https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/blob/main/LICENSE) ile lisanslanmıştır.

<p align="right">(<a href="#go-telegram-asistan-botu">başa dön</a>)</p>