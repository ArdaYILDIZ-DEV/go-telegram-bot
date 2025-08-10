

# Go Telegram Asistan Botu

**Go Telegram Asistan Botu**, kişisel bir sunucuyu veya bilgisayarı uzaktan yönetmek için Go diliyle geliştirilmiş, çok fonksiyonlu bir Telegram botudur. Bu proje, dosya yönetiminden sistem izlemeye, medya indirmeden komut çalıştırmaya kadar geniş bir yelpazede otomasyon yetenekleri sunarak, bir sunucu üzerindeki kontrolü doğrudan Telegram arayüzüne taşır.

<p align="center">
  <!-- Sadece Teknoloji Rozetleri -->
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/FFmpeg-00780B?style=for-the-badge&logo=ffmpeg&logoColor=white" alt="FFmpeg">
  <img src="https://img.shields.io/badge/yt--dlp-838383?style=for-the-badge&logo=youtube&logoColor=white" alt="yt-dlp">
</p>

---

## ✨ Ana Özellikler

| Kategori | Yetenek | Açıklama |
| :--- | :--- | :--- |
| 🌐 **Medya İndirme** | Akıllı Video/Ses İndirme | YouTube gibi platformlardan, istenen kalite ve formatta video (`/indir`) veya sadece ses (`/indir_ses`) indirir. |
| 💻 **Uzaktan Kontrol** | Betik Çalıştırma & İşlem Yönetimi | Sunucudaki `.bat`/`.ps1` dosyalarını çalıştırır (`/calistir`) ve herhangi bir işlemi PID ile sonlandırır (`/kapat`). |
| 📁 **Dosya Yönetimi** | Tam Dosya Kontrolü | Dosyaları sunucuya yükleyin, sunucudan indirin (`/getir`), silin (`/sil`), yeniden adlandırın ve taşıyın (`/tasi`). |
| ⚙️ **Otomasyon** | Otomatik Organizasyon & İzleme | Dosyaları kategorilere ayırır (`/duzenle`), internet kesintilerini takip eder (`/izle`) ve periyodik raporlar sunar. |
| 📊 **Sistem Tanılama** | Anlık Sistem Raporları | CPU, RAM, Disk durumunu (`/durum`), internet hızını (`/hiz_testi`) ve port durumunu (`/portlar`) anında raporlar. |
| 🎥 **Medya İşleme** | Video Düzenleme | `FFmpeg` entegrasyonu ile videoları kesin (`/kes`), GIF oluşturun (`/gif_yap`) veya ekran kaydı (`/kayit_al`) alın. |

---

## 🚀 Hızlı Başlangıç

Projeyi yerel makinenizde çalıştırmak için aşağıdaki adımları izleyin.

### Gereksinimler

- **Go:** `1.18` veya üstü
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

2.  **Bağımlılıkları Yükleyin:**
    ```bash
    go mod tidy
    ```

3.  **Yapılandırma Dosyasını (`.env`) Oluşturun:**
    Proje ana dizininde `.env` adında bir dosya oluşturun ve aşağıdaki içeriği kendi bilgilerinizle doldurun:
    ```env
    # Telegram Botunuzun Token'ı (BotFather'dan alınır)
    BOT_TOKEN=12345:ABCDEFG...

    # Botun tüm kritik komutları kullanacak olan yöneticinin Telegram ID'si (ZORUNLU)
    ADMIN_CHAT_ID=987654321

    # (İsteğe bağlı) Botu kullanmasına izin verilen diğer kullanıcıların ID'leri (virgülle ayrılmış)
    ALLOWED_IDS=11223344,55667788

    # (İsteğe bağlı) /portlar komutunun izleyeceği portlar
    MONITORED_PORTS=80,443,8080
    # Gelenler klasörünün konumunu kendi bilgisayarınıza göre değiştirin.
    Örnek:
    BASE_DIR=C:/Users/kullanici/Desktop/Gelenler
    ```
    > **İpucu:** Telegram ID'nizi öğrenmek için `@userinfobot` gibi botları kullanabilirsiniz.

4.  **Botu Çalıştırın:**
    ```bash
    go run .
    ```
    Veya daha performanslı bir şekilde derleyip çalıştırmak için:
    ```bash
    go build -o asistan.exe
    ./asistan.exe
    ```

---

## 🛠️ Kullanılan Teknolojiler

<p align="left">
  <!-- skillicons.dev ile gelenler (yönlendirmesiz) -->
  <!-- Not: skillicons servisi varsayılan olarak link eklemez, bu yüzden sadece img etiketi yeterlidir -->
  <img src="https://skillicons.dev/icons?i=go,git,github,powershell,vscode" />
  
  <!-- Manuel olarak eklenen ve yönlendirmesi kaldırılan logolar -->
  <!-- Sadece <img> etiketini bırakarak tıklanabilirliği kaldırıyoruz -->
  <img src="https://repository-images.githubusercontent.com/947861912/79d2548e-a5dc-420e-8fda-3e9368a7b668" alt="FFmpeg" height="48">
  <img src="https://repository-images.githubusercontent.com/307260205/b6a8d716-9c7b-40ec-bc44-6422d8b741a0" alt="yt-dlp" height="48">
</p>

## 🤝 Katkıda Bulunma

Bu proje kişisel kullanım için geliştirilmiştir, ancak her türlü fikir, öneri ve katkıya açıktır. Bir hata bulursanız veya yeni bir özellik önermek isterseniz, lütfen bir **[Issue](https://github.com/ArdaYILDIZ-DEV/go-telegram-bot/issues)** açmaktan veya **[Pull Request](https://github.com/ArdaYILDIZ-DEV/go-telegram-bot/pulls)** göndermekten çekinmeyin.

1.  Projeyi Fork'layın.
2.  Yeni bir özellik dalı oluşturun (`git checkout -b feature/YeniHarikaOzellik`).
3.  Değişikliklerinizi yapın ve kaydedin (`git commit -m 'Yeni bir harika özellik eklendi'`).
4.  Dalınızı push'layın (`git push origin feature/YeniHarikaOzellik`).
5.  Bir Pull Request oluşturun.

---

## Lisans

Bu proje [MIT Lisansı](https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan/blob/main/LICENSE) ile lisanslanmıştır.
