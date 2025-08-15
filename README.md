# Go Telegram Asistan Botu

<p align="center">
  <strong>Go ile yazılmış, sunucunuzu Telegram üzerinden yönetmek için tasarlanmış, yüksek performanslı ve modüler bir otomasyon aracı.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/FFmpeg-00780B?style=for-the-badge&logo=ffmpeg&logoColor=white" alt="FFmpeg">
  <img src="https://img.shields.io/badge/yt--dlp-838383?style=for-the-badge&logo=youtube&logoColor=white" alt="yt-dlp">
</p>

---

**Go Telegram Asistan Botu**, sadece komut çalıştıran basit bir botun çok ötesindedir. Arka planda çalışan zamanlayıcılar, dosya sistemi olaylarını anlık olarak dinleyen izleyiciler ve Go'nun eşzamanlılık (concurrency) gücü sayesinde, sunucunuzla proaktif bir şekilde etkileşim kuran kişisel bir asistandır. Dosyalarınızı organize eder, ağ sorunlarını size bildirir ve uzun süren görevleri sizi engellemeden arka planda halleder.

> Bu proje, bir sunucu üzerindeki kontrolü, güvenliği ve otomasyonu doğrudan Telegram arayüzüne taşıyarak uzaktan yönetimi kolaylaştırmak amacıyla geliştirilmiştir.

---

## ✨ Öne Çıkan Özellikler

<br/>
<p align="center">
  <img src="https://img.shields.io/badge/-Gelişmiş%20Dosya%20Yönetimi-007ACC?style=for-the-badge" alt="Dosya Yönetimi">
</p>

- **Listeleme ve Arama:**
  - `/liste`: Ana dizindeki dosyaları listeler.
  - `/klasor <kategori>`: Belirli bir kategori altındaki dosyaları gösterir.
  - `/ara <kelime>`: **Tüm alt klasörlerde** dosya adlarına göre arama yapar ve konumlarıyla birlikte raporlar.
- **Dosya Transferi ve Manipülasyonu:**
  - `/getir <dosya>`: Sunucudaki herhangi bir dosyayı, açıklamasıyla birlikte Telegram'a gönderir.
  - `/sil <dosya>`: Yanlışlıkla silmeleri önlemek için **inline butonlar ile onay isteyen** güvenli bir silme mekanizması sunar.
  - `/yenidenadlandir <eski> <yeni>`: Bir dosyanın adını değiştirir; eğer dosyanın bir açıklaması varsa, bu açıklama yeni dosyaya taşınır.
- **Metadata (Açıklama) Sistemi:**
  - `/aciklama_ekle <dosya> <açıklama>`: Dosyalara kalıcı olarak `metadata.json` dosyasında saklanan açıklamalar ekler.
  - `/aciklama_ara <kelime>`: Sadece dosya adlarında değil, **dosya açıklamalarının içinde de** arama yapar.

<br/>
<p align="center">
  <img src="https://img.shields.io/badge/-Kapsamlı%20Sistem%20Kontrolü-007ACC?style=for-the-badge" alt="Sistem Kontrolü">
</p>

- **İnteraktif Görev Yöneticisi:**
  - `/gorevler`: Sunucuda çalışan tüm işlemleri **sayfalı ve sıralanabilir** bir arayüzde sunar. CPU veya RAM kullanımına göre artan/azalan şekilde sıralama yapabilirsiniz.
  - `/kapat <PID>`: Görev yöneticisinden veya manuel olarak belirlediğiniz bir işlemi anında sonlandırır.
- **Güvenli Komut/Betik Çalıştırma:**
  - `/calistir <yol> <süre>`: `.bat`, `.ps1` veya diğer çalıştırılabilir dosyaları, belirtilen **zaman aşımı (timeout)** süresiyle çalıştırır. Eğer betik bu süre içinde tamamlanmazsa, otomatik olarak sonlandırılır ve o ana kadarki çıktısı size gönderilir.
- **Ekran Görüntüsü ve Kaydı:**
  - `/ss`: Windows üzerinde PowerShell kullanarak anlık, yüksek çözünürlüklü bir ekran görüntüsü alır.
  - `/kayit_al` & `/kayit_durdur`: `FFmpeg` kullanarak ekran kaydı yapar. Kayıt durdurulduğunda, video dosyası işlenir ve otomatik olarak size gönderilir.

<br/>
<p align="center">
  <img src="https://img.shields.io/badge/-Medya%20İndirme%20ve%20İşleme-007ACC?style=for-the-badge" alt="Medya İşlemleri">
</p>

- **Akıllı İndirme Motoru:**
  - `/indir <URL> [kalite] [format]`: `yt-dlp`'nin esnek format seçimi (`-f`) yeteneğini kullanarak, "en iyi video (<=1080p, mp4) + en iyi ses" gibi karmaşık kurallarla indirme yapar. İlerleme durumu anlık olarak mesaj düzenlenerek size bildirilir.
  - `/indir_ses <URL> [format]`: Videoyu tamamen atlayarak sadece en iyi ses akışını indirir ve `opus`, `mp3`, `flac` gibi formatlara dönüştürür.
- **Yüksek Hızlı Video Düzenleme:**
  - `/kes <dosya> <baş> <bitiş>`: FFmpeg'in `-c copy` parametresini kullanarak videoyu yeniden kodlamadan **saniyeler içinde** keser. Bu, saatler sürebilecek işlemleri anlık hale getirir.
- **Optimize Edilmiş GIF Üretimi:**
  - `/gif_yap <dosya> <baş> <bitiş>`: Standart GIF oluşturmanın ötesinde, videodan önce bir renk paleti çıkarıp sonra bu paleti kullanarak GIF'i oluşturan iki aşamalı bir `filtergraph` kullanır. Bu, çok daha yüksek renk doğruluğu ve daha küçük dosya boyutu sağlar.

<br/>
<p align="center">
  <img src="https://img.shields.io/badge/-Akıllı%20Otomasyonlar-007ACC?style=for-the-badge" alt="Otomasyonlar">
</p>

- **Magic Folder (`TelegramaGonder`):** Bu klasöre sürükleyip bıraktığınız herhangi bir dosya, `fsnotify` dosya sistemi izleyicisi tarafından anında algılanır, size gönderilir ve ardından sunucudan temizlenir.
- **Otomatik Raporlama ve Bakım:**
  - **Saatlik Sistem Raporu:** Her saat başı, `/sistem_bilgisi` ve `/hiz_testi` komutlarının birleşiminden oluşan detaylı bir raporu otomatik olarak size gönderir.
  - **Otomatik Kategorizasyon:** `/duzenle` komutu, periyodik olarak çalışarak "Gelenler" klasörünü düzenli tutar.
- **Proaktif Ağ İzleme:**
  - `/izle`: İnternet bağlantısını `ping` ile sürekli izler. Bağlantı kesildiğinde ve geri geldiğinde **sadece durum değiştiğinde** bildirim gönderir, gereksiz mesajları önler.
  - **Port Monitörü:** `/portlar` komutunun izlediği portların durumu değiştiğinde (örn: bir servis çöktüğünde veya başladığında) anında bildirim alırsınız.

---

### Mimarî ve Tasarım Felsefesi
> Bu bot, "sorumlulukların ayrılması" ve "engellemesiz operasyon" prensipleri üzerine kurulmuştur.
-   **Modülerlik:** Her dosya (`auth.go`, `file_manager.go`, `scheduler.go` vb.) tek bir sorumluluk alanına odaklanır.
-   **Eşzamanlılık (Concurrency):** `goroutine` ve `channel`'lar, indirme, betik çalıştırma gibi uzun süren işlemlerin botun ana akışını engellemesini önler. Bot, aynı anda birden çok komuta yanıt verebilir.
-   **Durum Güvenliği (State Safety):** Paylaşılan verilere (metadata, kayıt durumu vb.) erişim, `sync.Mutex` kilitleri ile korunarak "race condition" hatalarının önüne geçilir.
-   **Harici Araç Entegrasyonu:** `yt-dlp`, `ffmpeg` gibi kendini kanıtlamış, güçlü komut satırı araçlarını bir arayüz arkasında birleştirir.

---

## 🚀 Hızlı Başlangıç

### Gereksinimler

- **Go:** `1.18` veya üstü.
- **Harici Araçlar:** Botun tüm özelliklerini kullanabilmek için aşağıdaki CLI araçlarının sisteminizde kurulu ve **PATH** ortam değişkenine eklenmiş olması gerekir:
  - **[yt-dlp](https://github.com/yt-dlp/yt-dlp):** Video ve ses indirmek için.
  - **[FFmpeg](https://ffmpeg.org/download.html):** Medya işleme (kesme, GIF yapma, ekran kaydı) için.
  - **[Speedtest CLI](https://www.speedtest.net/apps/cli):** `/hiz_testi` komutu için.

### Kurulum Adımları

1.  **Projeyi Klonlayın:**
    ```bash
    git clone https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan.git
    cd go-telegram-asistan
    ```
2.  **Go Modüllerini İndirin:**
    ```bash
    go mod tidy
    ```
3.  **Yapılandırma Dosyasını (`.env`) Oluşturun:**
    Proje ana dizininde `.env` adında bir dosya oluşturun ve aşağıdaki tabloya göre doldurun.

    | Değişken | Gerekli? | Açıklama | Örnek |
    | :--- | :---: | :--- | :--- |
    | `BOT_TOKEN` | ✅ **Evet** | Telegram'da `@BotFather`'dan alacağınız API token'ı. | `123456:ABC-DEF...` |
    | `ADMIN_CHAT_ID` | ✅ **Evet** | Botun tam yetkili yöneticisinin Telegram ID'si. | `9876543210` |
    | `ALLOWED_IDS` | ❌ Hayır | Botu kullanabilecek diğer kullanıcıların ID'leri (virgülle ayırın). | `112233,445566` |
    | `BASE_DIR`| ❌ Hayır | Botun çalışacağı ana klasör. *Varsayılan: `Gelenler`*. | `C:/BotDosyalari` |
    | `MONITORED_PORTS` | ❌ Hayır | Periyodik olarak izlenecek portlar (virgülle ayırın). | `80,443,3306` |

4.  **Botu Çalıştırın:**
    ```bash
    go run .
    ```
    Veya daha performanslı bir şekilde derleyip çalıştırmak için:
    ```bash
    # Windows için
    go build -o asistan.exe && ./asistan.exe

    # Linux/macOS için
    go build -o asistan && ./asistan
    ```

---

<details>
  <summary><strong>Tüm Komutların Listesi ve Açıklamaları</strong></summary>
  
  | Komut | Açıklama |
  | :--- | :--- |
  | `/start` | Bota hoş geldin mesajı ve genel bir bakış sunar. |
  | `/help` | Bu komut listesini gösterir. |
  | `/getir <dosya>` | Sunucudan belirtilen dosyayı gönderir. |
  | `/sil <dosya>` | Belirtilen dosyayı onay alarak siler. |
  | `/yenidenadlandir <eski> <yeni>` | Bir dosyanın adını değiştirir. |
  | `/tasi <dosya> <klasör>` | Bir dosyayı başka bir klasöre taşır. |
  | `/ara <kelime>` | Dosya adlarında arama yapar. |
  | `/liste` | Ana klasördeki dosyaları gösterir. |
  | `/klasor <kategori>` | Belirli bir kategori klasörünü listeler. |
  | `/aciklama_ekle <dosya> <açıklama>` | Bir dosyaya açıklama ekler. |
  | `/aciklama_sil <dosya>` | Bir dosyanın açıklamasını siler. |
  | `/aciklamalar` | Tüm açıklamaları listeler. |
  | `/aciklama_ara <kelime>` | Açıklamaların içinde arama yapar. |
  | `/indir <URL> [kalite] [format]` | Video/dosya indirir. |
  | `/indir_ses <URL> [format]` | Sadece ses dosyasını indirir. |
  | `/kes <dosya> <baş> <bitiş>` | Bir videonun belirtilen aralığını keser. |
  | `/gif_yap <dosya> <baş> <bitiş>` | Bir videodan GIF üretir. |
  | `/gorevler` | İnteraktif görev yöneticisini açar (Yönetici). |
  | `/calistir <yol> <süre>` | Sunucuda bir betik çalıştırır (Yönetici). |
  | `/kapat <PID>` | Belirtilen PID'ye sahip işlemi durdurur (Yönetici). |
  | `/duzenle` | Dosyaları otomatik olarak kategorilere ayırır. |
  | `/durum` | Temel sistem durumunu gösterir. |
  | `/sistem_bilgisi` | Ayrıntılı sistem bilgilerini raporlar (Yönetici). |
  | `/hiz_testi` | İnternet indirme/yükleme hızı ve ping ölçümü yapar. |
  | `/portlar` | İzlenen portların durumunu kontrol eder. |
  | `/ss` | Sunucunun ekran görüntüsünü alır (Yönetici). |
  | `/kayit_al`, `/kayit_durdur` | Ekran kaydını başlatır ve durdurur (Yönetici). |
  | `/izle` | İnternet kesinti izleyicisini açar/kapatır. |

</details>

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

## 🤝 Katkıda Bulunma

Bu proje kişisel kullanım için geliştirilmiştir, ancak her türlü fikir, öneri ve katkıya açıktır. Bir hata bulursanız veya yeni bir özellik önermek isterseniz, lütfen bir **[Issue](https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/issues)** açmaktan veya **[Pull Request](https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/pulls)** göndermekten çekinmeyin.

1.  Projeyi **Fork**'layın.
2.  Yeni bir özellik dalı oluşturun (`git checkout -b feature/YeniHarikaOzellik`).
3.  Değişikliklerinizi yapın ve **Commit**'leyin (`git commit -m 'Yeni bir harika özellik eklendi'`).
4.  Dalınızı **Push**'layın (`git push origin feature/YeniHarikaOzellik`).
5.  Bir **Pull Request** oluşturun.

---

## Lisans

Bu proje [MIT Lisansı](https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/blob/main/LICENSE) ile lisanslanmıştır.