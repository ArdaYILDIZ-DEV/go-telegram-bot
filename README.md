# Sentinel Bot

![Go Version](https://img.shields.io/badge/Go-1.18+-blue.svg)![License](https://img.shields.io/badge/License-MIT-green.svg)![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)

Sentinel Bot, Go ile yazılmış, gelişmiş bir Telegram botudur.Bu bot, sunucunuza uzaktan tam kontrol sağlamak için tasarlanmıştır.

## Temel Yetenekler

^ **Akıllı Asistan (Google Gemini)**
*   Doğal dil komutlarını anlama ve işleme.
*   "Function Calling" yeteneği sayesinde "sistem durumu nasıl?" gibi cümleleri `/durum` komutuna çevirme.
*   API kota hatalarına karşı model değiştirerek (fallback) sorgu tekrarı yapabilme.

==> **Dosya Yönetimi**
*   Dosyaları sunucuya yükleme, sunucudan indirme (`/getir`).
*   Dosya listeleme (`/liste`), arama (`/ara`), silme (`/sil`), yeniden adlandırma (`/yenidenadlandir`) ve taşıma (`/tasi`).
*   Dosyalara kalıcı açıklamalar ekleme ve bu açıklamalarda arama yapma.
*   "Gelenler" klasöründeki dosyaları uzantılarına göre otomatik olarak kategorilere ayırma (`/duzenle`).

**--* **Sistem ve İşlem Yönetimi**
*   Anlık ve detaylı sistem kaynak (CPU, RAM, Disk) raporları alma (`/durum`, `/sistem_bilgisi`).
*   İnteraktif, sayfalara ayrılmış ve sıralanabilir görev yöneticisi (`/gorevler`).
*   PID ile işlem sonlandırma (`/kapat`).
*   Belirtilen betikleri/programları zaman aşımı kontrolü ile çalıştırma ve çıktısını alma (`/calistir`).
*   Önceden tanımlanmış uygulamaları kısayol ile başlatma (`/uygulama_calistir`).
*   Ekran görüntüsü alma (`/ss`) ve ekran kaydı yapma (`/kayit_al`, `/kayit_durdur`).

# **İndirme ve Medya İşlemleri**
*   `yt-dlp` entegrasyonu ile popüler video platformlarından video indirme.
*   Videolardan sadece ses dosyasını indirme (`/indir_ses`).
*   `FFmpeg` kullanarak video kesme (`/kes`) ve GIF oluşturma (`/gif_yap`).

-- **Otomasyon ve İzleme**
*   Belirli aralıklarla (saatlik) sistem raporu ve hız testi sonucunu yöneticiye gönderme.
*   İnternet bağlantısını sürekli izleme ve kesinti sonrası toplam kesinti süresini bildirme.
*   İzlenen servis portlarının durumunu (başladı/durdu) anlık olarak bildirme.
*   "Magic Folder": `TelegramaGonder` klasörüne atılan dosyaları otomatik olarak Telegram'a gönderip sunucudan silme.

## Teknoloji Mimarisi

*   **Dil:** Go
*   **Telegram API:** [go-telegram-bot-api/v5](https://github.com/go-telegram-bot-api/telegram-bot-api)
*   **Sistem Bilgileri:** [gopsutil](https://github.com/shirou/gopsutil)
*   **Yapay Zeka (LLM):** Google Gemini API (Function Calling ile)
*   **Harici Bağımlılıklar (PATH üzerinde olmalı):**
    *   `yt-dlp`: Video ve ses indirme işlemleri için.
    *   `FFmpeg`: Medya kesme, GIF yapma ve ekran kaydı için.
    *   `speedtest-cli`: İnternet hız testi için (Ookla'nın resmi CLI aracı).

## Kurulum ve Yapılandırma

1.  **Gereksinimleri Yükleyin:**
    Sunucunuzda Go'nun ve yukarıda belirtilen harici bağımlılıkların (yt-dlp, ffmpeg, speedtest) kurulu ve sistem PATH'ine eklenmiş olduğundan emin olun.

2.  **Projeyi Klonlayın:**
    ```bash
    git clone https://github.com/kullanici/sentinel-bot.git
    cd sentinel-bot
    ```

3.  **Yapılandırma Dosyasını Oluşturun:**
    Proje ana dizininde `.env` adında bir dosya oluşturun ve aşağıdaki şablonu kendi bilgilerinizle doldurun.

    ```dotenv
    # .env.example

    # Telegram Botfather'dan alacağınız API token'ı.
    BOT_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
    
    # Botun yönetici olarak tanıyacağı Telegram kullanıcısının Chat ID'si.
    # Tüm kritik bildirimler ve yönetici komutları bu ID'ye bağlıdır.
    ADMIN_CHAT_ID=123456789
    
    # Botu kullanmasına izin verilen diğer kullanıcıların Chat ID'leri (virgülle ayırın).
    ALLOWED_IDS=987654321,123123123
    
    # Google AI Studio'dan alacağınız Gemini API anahtarı.
    GEMINI_API_KEY=AIzaSyB...
    
    # Botun dosyaları yöneteceği ana klasörün adı.
    BASE_DIR=C:\Users\windowsİsminiz\Desktop\Gelenler
    
    # İzlenecek portlar ve servis isimleri (isim:port,isim2:port2 şeklinde).
    MONITORED_PORTS=SSH:22,HTTP:80,Postgres:5432
    
    # /uygulama_calistir komutu için tanımlanacak kısayollar (isim:yol,isim2:yol2).
    # Yollarda bosluk varsa tirnak icine almayin. Windows icin backslash'leri cift yazin (\\).
    UYGULAMALAR=chrome:C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe,vscode:C:\\Users\\Admin\\AppData\\Local\\Programs\\Microsoft VS Code\\Code.exe
    ```

4.  **Bağımlılıkları İndirin ve Derleyin:**
    ```bash
    go mod tidy
    go build -o sentinel_bot
    ```

5.  **Botu Çalıştırın:**
    ```bash
    ./sentinel_bot
    ```
    Bot, log tutabilecek şekilde tasarlandı.

## Kullanım

Bota komutları iki farklı modda verebilirsiniz:

1.  **Standart Mod:**
    Komutları doğrudan Telegram'a yazarak kullanırsınız. Örnek: `/getir rapor.pdf`

2.  **Akıllı Asistan Modu:**
    *   `/llm` komutu ile bu modu etkinleşerek daha iyi bir deneyim elde edebilirsiniz.
    *   Bu modda, komutları doğal bir dilde yazabilirsiniz. Bot, cümlenizi analiz ederek doğru komutu kendisi çalıştıracaktır.
    *   Örnek: "bana rapor.pdf dosyasını gönder", "internet hızı ne durumda?", "gelenler klasöründeki dosyaları düzenle".
    *   Moddan çıkmak için `/llm_kapat` komutunu kullanın.

## Komut Listesi

```
-Komut Seti –

^^ *Akıllı Asistan (Sentinel):*
/llm – Yapay zeka ile sohbet modunu başlatır
/llm_kapat – Aktif sohbet modunu sonlandırır

== *Dosya Yönetimi:*
/getir <dosya> – Dosyayı gönder
/sil <dosya> – Dosyayı sil (onaylı)
/yenidenadlandir <eski> <yeni> – Dosyayı yeniden adlandır
/tasi <dosya> <klasör> – Dosyayı taşı

oo *Arama ve Listeleme:*
/ara <kelime> – Dosya adlarında ara
/liste – Ana klasördeki dosyaları göster
/klasor <kategori> – Kategori klasörünü listele

:: *Açıklama Yönetimi:*
/aciklama_ekle <dosya> <açıklama>
/aciklama_sil <dosya>
/aciklamalar – Tüm açıklamaları listele
/aciklama_ara <kelime> – Açıklamalarda ara

//  *İndirme ve Medya İşleme:*
/indir <URL> [kalite] [format] – Video/dosya indir
/indir_ses <URL> [format] – Sadece sesi indir
/kes <dosya> <baş> <bitiş> – Video kes
/gif_yap <dosya> <bitiş> – GIF üret

==| *Sistem ve İşlem Yönetimi:*
/gorevler – İnteraktif görev yöneticisi (Yönetici)
/kapat <PID> – Çalışan işlemi durdur (Yönetici)
/durum – Temel sistem durumu
/sistem_bilgisi – Ayrıntılı sistem bilgisi (Yönetici)
/hiz_testi – İndirme/yükleme hızı ve ping ölçümü
/portlar – İzlenen port durumları
/ss – Ekran görüntüsü al (Yönetici)
/kayit_al, /kayit_durdur – Ekran kaydı (Yönetici)
/duzenle – Dosyaları otomatik kategorilere ayır
/izle – Ağ bağlantısını izlemeye başla/durdur

++ *Uygulama & Betik Çalıştırma (Yönetici):*
/calistir <yol> <süre> – Betik çalıştır ve çıktısını al
/uygulama_calistir <kısayol> – Önceden tanımlı uygulamayı başlat
/calistir_dosya <yol> – Dosya yolu ile uygulama başlat
```