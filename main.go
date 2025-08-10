// main.go
package main

import (
	"io"
	"log"
	"os"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                             UYGULAMA GİRİŞ NOKTASI
// #############################################################################
// Bu dosya, `go run .` komutu çalıştırıldığında ilk harekete geçen kısımdır.
// Programın ana `main` fonksiyonunu içerir ve temel başlatma işlemlerini
// sırasıyla gerçekleştirir: loglama ayarları, yapılandırma yükleme,
// botu Telegram API'sine bağlama ve gelen güncellemeleri dinlemeye başlama.

func main() {
	// * ÖNEMLİ: Loglama ayarları, programın hem çalışırken terminalde (stdout)
	// * hem de kalıcı olarak bir dosyada (`telegram_bot.log`) kayıt tutmasını sağlar.
	// * Bu, bot çalışırken oluşan hataları veya olayları daha sonra incelemek için kritiktir.
	logFile, err := os.OpenFile("telegram_bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// Log dosyası oluşturulamazsa, programın devam etmesinin bir anlamı yoktur.
		log.Fatalf("Log dosyası açılamadı: %v", err)
	}
	// * `io.MultiWriter`, log çıktısının aynı anda birden çok hedefe yazılmasına olanak tanır.
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	// * Log satırlarının başına tarih, saat ve hangi kod dosyasının hangi satırında
	// * loglandığı bilgisini ekler. Hata ayıklama için çok faydalıdır.
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("Uygulama başlatılıyor...")

	// * Yapılandırma dosyasını (`config.go`) yükle. Hata varsa programı durdur.
	if err := loadConfig(); err != nil {
		log.Fatalf("Yapılandırma yüklenemedi: %v", err)
	}
	// * Metadata dosyasını (`metadata_manager.go`) yükle. Hata varsa programı durdur.
	if err := loadMetadata(); err != nil {
		log.Fatalf("Metadata yüklenemedi: %v", err)
	}

	// * Botun ihtiyaç duyduğu tüm klasörlerin var olduğundan emin ol (`file_manager.go`).
	ensureDirectories()

	// * Yapılandırmadan alınan BOT_TOKEN ile Telegram API'sine bağlan.
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		// Token geçersizse veya ağ bağlantısı yoksa burada program çöker.
		log.Panic(err)
	}
	bot.Debug = false // Geliştirme sırasında daha fazla bilgi için 'true' yapılabilir.
	log.Printf("%s botu olarak yetkilendirildi.", bot.Self.UserName)

	// * Telegram'dan güncellemeleri (mesajlar, buton tıklamaları vb.) almak için
	// * bir "updates channel" oluştur. `Timeout` değeri, yeni bir güncelleme
	// * yoksa ne kadar süre bekleneceğini belirtir (long polling).
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// * Programın ana döngüsü burada başlar. `handleUpdates`, gelen güncellemeleri
	// * sonsuza dek dinleyecek ve ilgili fonksiyonlara yönlendirecektir.
	log.Println("Bot güncellemeleri dinlemeye başladı...")
	handleUpdates(bot, updates)
}