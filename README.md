🌟 Tanıtım
Go Telegram Asistan Botu, sunucu yönetiminizi Telegram üzerinden gerçekleştirmenizi sağlayan gelişmiş bir otomasyon çözümüdür. Basit bir bot olmanın ötesinde, Go'nun eşzamanlılık gücünü kullanarak:

🕒 Arka planda çalışan akıllı zamanlayıcılar

📁 Gerçek zamanlı dosya sistemi izleme

⚡ Uzun süren görevler için engellemesiz işlemler

🔄 Proaktif sunucu yönetimi

sağlayarak kişisel bir sunucu asistanı deneyimi sunar.

graph LR
A[Telegram Kullanıcısı] --> B[Bot Komutları]
B --> C[Dosya Yönetimi]
B --> D[Sistem Kontrolü]
B --> E[Medya İşleme]
B --> F[Otomatik Raporlama]
C --> G[Metadata Sistemi]
D --> H[Görev Yöneticisi]
E --> I[Akıllı İndirme]
F --> J[Proaktik İzleme]


🚀 Öne Çıkan Özellikler
📁 Gelişmiş Dosya Yönetimi
Akıllı Listeleme & Arama: /liste, /klasor, /ara komutlarıyla tüm alt klasörlerde arama

Güvenli Operasyonlar: Onaylı silme (/sil), metadata korumalı yeniden adlandırma (/yenidenadlandir)

Metadata Sistemi: Dosyalara kalıcı açıklamalar ekleme ve içeriklerde arama

⚙️ Sistem Kontrol & İzleme
İnteraktif Görev Yöneticisi: /gorevler ile canlı işlem takibi

Zaman Aşımlı Betik Çalıştırma: /calistir ile güvenli komut yürütme

Gerçek Zamanlı Ekran Yönetimi: /ss ve /kayit_al ile uzaktan kontrol

🎬 Medya İşleme & İndirme
Akıllı İndirme Motoru: /indir ve /indir_ses ile optimize edilmiş içerik indirme

Saniyeler İçinde Video Kesme: /kes ile yeniden kodlama olmadan hızlı kesim

Optimize GIF Üretimi: /gif_yap ile yüksek kaliteli, küçük boyutlu GIF'ler

🤖 Akıllı Otomasyonlar

sequenceDiagram
    Kullanıcı->>+Bot: Dosyayı TelegramaGonder klasörüne atar
    Bot->>+FSWatcher: Dosya değişikliği algılandı
    FSWatcher->>+Bot: Olay bildirimi
    Bot->>+Kullanıcı: Dosyayı Telegram'a gönder
    Bot->>+Sunucu: Orijinal dosyayı temizle


Magic Folder: Dosya bırakma ile otomatik gönderim

Saatlik Sistem Raporu: Otomatik performans ve ağ izleme

Proaktif Ağ İzleme: /izle ile kesinti bildirimleri

⚙️ Teknik Mimarî
Modüler Tasarım:

📦 go-telegram-asistan
├── 📂 core
│   ├── 📜 bot.go           # Ana bot yapılandırması
│   ├── 📜 commands.go      # Komut işleyiciler
│   └── 📜 scheduler.go     # Zamanlanmış görevler
├── 📂 modules
│   ├── 📜 file_manager.go  # Dosya yönetim sistemi
│   ├── 📜 media_processor.go # Medya işleme
│   └── 📜 system_monitor.go # Sistem izleme
├── 📜 metadata.json        # Dosya açıklamaları DB
└── 📜 .env                 # Yapılandırma ayarları


Performans Optimizasyonu:

Goroutine'ler ile eşzamanlı işlem yönetimi

Mutex kilitleri ile thread güvenliği

FFmpeg'in donanım hızlandırmasından yararlanma

🚀 Hızlı Kurulum
Ön Koşullar:

# Gerekli araçlar
choco install ffmpeg yt-dlp speedtest-cli -y  # Windows
brew install ffmpeg yt-dlp speedtest-cli      # macOS

Kurulum Adımları
1.Repoyu klonla:

git clone https://github.com/ArdaYILDIZ-DEV/go-telegram-asistan.git
cd go-telegram-asistan


2.Yapılandırma dosyası oluştur (.env):

BOT_TOKEN=123456:ABC-DEF12345ghijklmnopqrstuvwxyz
ADMIN_CHAT_ID=987654321
BASE_DIR=C:/SunucuDosyalari
MONITORED_PORTS=80,443,22

3.Bağımlılıkları yükle ve çalıştır:

go mod tidy
go build -o asistan && ./asistan


📊 Komut Referansı
Kategori	Komut	Açıklama
Dosya	/getir <dosya>	Dosyayı indir
/aciklama_ekle ...	Dosyaya açıklama ekle
Sistem	/gorevler	İşlemleri listele
/sistem_bilgisi	Detaylı sistem raporu
Medya	/indir <URL>	Video/audio indir
/gif_yap ...	Videodan GIF oluştur
Otomasyon	/izle	Ağ bağlantısını izle
/duzenle	Dosyaları otomatik kategorize et


🤝 Katkıda Bulunma
Katkılarınızı bekliyoruz! İşte katkı süreci:

graph TB
    A[Fork Repo] --> B[Özellik Dalı Oluştur]
    B --> C[Değişiklikleri Yap]
    C --> D[Testleri Çalıştır]
    D --> E[Pull Request Gönder]
    E --> F[Code Review]
    F --> G[Merge]

1.Repoyu fork'layın

2.Yeni bir özellik dalı oluşturun (feature/yeni-ozellik)

3.Değişikliklerinizi commit'leyin

4.Testleri çalıştırın: go test ./...

5.Pull Request oluşturun

📜 Lisans
Bu proje MIT Lisansı ile lisanslanmıştır.

Copyright (c) 2023 Arda YILDIZ

İzin verilen ücretsiz kullanım, kopyalama, değiştirme, birleştirme, yayımlama, dağıtma...