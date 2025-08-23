package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Loglamayı hem konsola hem de bir dosyaya yapacak şekilde ayarla.
	logFile, err := os.OpenFile("telegram_bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Log dosyası açılamadı: %v", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// ASCII Art banner
	fmt.Println(`
██████ ███████ ███    ██ ████████ ██ ███    ██ ███████ ██     
██      ██      ████   ██    ██    ██ ████   ██ ██      ██     
███████ █████   ██ ██  ██    ██    ██ ██ ██  ██ █████   ██     
    ██ ██      ██  ██ ██    ██    ██ ██  ██ ██ ██      ██     
██████  ███████ ██   ████    ██    ██ ██  █████ ███████ ███████
▒▓██▒   ▒▓▓██▓▓ ▒█   ▒▓██    ▓█    ▒█ ▒█   ▒▓█▓ ▒▓▓▓█▓▓ ▒▓▓█▓▓
░▒▓▓░   ░▒▓▓▓░  ░▓   ░▒▓▓    ░▓    ░▓ ░▓   ░▒▓▓ ░▒▓▓▓▒  ░▒▓▓▓ 
░▒▓░    ░▒▒▓   ░▓    ░▒▓    ░▓    ░▓ ░▓    ░▒▓  ░▒▒▓   ░▒▒▓  
 ░▒      ░▒     ░     ░▒     ░     ░  ░     ░▒   ░▒     ░▒   
  ░       ░             ░                    ░    ░      ░
`)

	log.Println("Uygulama başlatılıyor...")

	// .env dosyasından yapılandırmayı yükle.
	if err := loadConfig(); err != nil {
		log.Fatalf("Yapılandırma yüklenemedi: %v", err)
	}

	// metadata.json dosyasından dosya açıklamalarını yükle.
	if err := loadMetadata(); err != nil {
		log.Fatalf("Metadata yüklenemedi: %v", err)
	}

	// system_prompt.txt dosyasından Gemini AI için sistem talimatını yükle.
	if err := loadSystemPrompt(); err != nil {
		// Fonksiyon zaten kendi içinde bir uyarı logu basıyor.
		// Botun çalışmaya devam etmesine izin veriyoruz.
	}

	// Botun çalışması için gerekli tüm klasörlerin var olduğundan emin ol.
	ensureDirectories()

	// Telegram Bot API ile bağlantı kur.
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("%s botu olarak yetkilendirildi.", bot.Self.UserName)

	// Arka planda çalışacak görevleri ayrı goroutine'lerde başlat.
	go runScheduler(bot) // Saatlik görevler ve dosya izleyiciyi başlatır.
	go startWorkers(bot) // Port ve internet izleyici worker'larını başlatır.

	// Telegram'dan güncellemeleri (mesajlar, vb.) almaya başla.
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("Bot güncellemeleri dinlemeye başladı...")

	// Gelen güncellemeleri sonsuz bir döngüde işle.
	handleUpdates(bot, updates)
}
