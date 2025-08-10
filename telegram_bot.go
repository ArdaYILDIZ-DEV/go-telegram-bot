// telegram_bot.go
package main

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                           ANA BOT MANTIĞI VE YÖNLENDİRİCİ
// #############################################################################
// Bu dosya, botun beyni olarak işlev görür. Telegram'dan gelen tüm
// güncellemeleri (mesajlar, komutlar, buton tıklamaları) dinleyen ana döngüyü
// içerir. Gelen her isteği ilk olarak bir yetkilendirme filtresinden geçirir
// ve ardından isteğin türüne göre ilgili "handle" fonksiyonuna yönlendirir.

// handleUpdates, Telegram'dan gelen güncellemeleri sonsuz bir döngüde dinler.
// Programın ana işlevini bu fonksiyon yürütür.
func handleUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		var userID int64
		var chatID int64
		var fromUserName string

		// * Güncellemenin türüne (mesaj mı, buton tıklaması mı) göre
		// * ilgili kullanıcı bilgilerini tek bir yerden al.
		if update.Message != nil {
			userID = update.Message.From.ID
			chatID = update.Message.Chat.ID
			fromUserName = update.Message.From.UserName
			log.Printf("[%s] %s", fromUserName, update.Message.Text)
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
			chatID = update.CallbackQuery.Message.Chat.ID
			fromUserName = update.CallbackQuery.From.UserName
			log.Printf("[%s] Callback: %s", fromUserName, update.CallbackQuery.Data)
		} else {
			continue // Desteklenmeyen güncelleme türlerini (örn: kanal postası) atla.
		}

		// * ÖNEMLİ: BOTUN ANA GÜVENLİK KİLİDİ
		// * Herhangi bir işlem yapmadan önce, isteği yapan kullanıcının
		// * botu kullanma izni (`isUserAllowed`) olup olmadığını kontrol et.
		// * Yetkisiz kullanıcılar burada engellenir ve döngünün başına dönülür.
		if !isUserAllowed(userID) {
			log.Printf("⚠️ YETKİSİZ ERİŞİM DENEMESİ! Kullanıcı: %s (%d)", fromUserName, userID)
			bot.Send(tgbotapi.NewMessage(chatID, "🚫 Bu botu kullanma yetkiniz bulunmuyor."))
			continue
		}

		// * Yetkilendirme başarılı, şimdi gelen isteği işle.
		if update.Message != nil {
			if update.Message.IsCommand() {
				handleCommand(bot, update.Message) // Komut ise komut işleyiciye
			} else {
				handleFile(bot, update.Message) // Komut değilse (dosya, resim vb.), dosya işleyiciye
			}
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery) // Buton tıklaması ise callback işleyiciye
		}
	}
}

// handleCommand, gelen komutları alır, yönetici yetkisi gerekip gerekmediğini
// kontrol eder ve ardından `switch` bloğu ile ilgili alt fonksiyona yönlendirir.
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	command := message.Command()

	// * Yönetici yetkisi gerektiren "tehlikeli" komutların listesi.
	// * Yeni bir yönetici komutu eklendiğinde bu haritaya (map) eklenmesi yeterlidir.
	adminOnlyCommands := map[string]bool{
		"calistir":       true,
		"kapat":          true,
		"sistem_bilgisi": true,
		"kayit_al":       true,
		"kayit_durdur":   true,
		"ss":             true,
	}

	// * İkinci güvenlik katmanı: Komut, yönetici listesindeyse,
	// * gönderenin gerçekten yönetici (`isUserAdmin`) olup olmadığını kontrol et.
	if _, isProtected := adminOnlyCommands[command]; isProtected {
		if !isUserAdmin(message.From.ID) {
			log.Printf("⚠️ YETKİSİZ KOMUT DENEMESİ! Kullanıcı: %s (%d), Komut: /%s", message.From.UserName, message.From.ID, command)
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🚫 Bu komutu sadece yönetici kullanabilir."))
			return // Fonksiyonu burada sonlandırarak komutun işlenmesini engelle.
		}
	}

	// * Tüm yetki kontrollerinden geçen komut, `switch` bloğu ile ilgili
	// * `handle...` fonksiyonuna gönderilir. Burası ana yönlendiricidir (router).
	switch command {
	case "start", "help", "duzenle", "sistem_bilgisi", "durum", "hiz_testi":
		handleGeneralCommands(bot, message)
	case "calistir":
		handleRunCommand(bot, message)
	case "kapat":
		handleKillCommand(bot, message)
	case "ss":
		handleScreenshotCommand(bot, message)
	case "kayit_al":
		handleStartRecordingCommand(bot, message)
	case "kayit_durdur":
		handleStopRecordingCommand(bot, message)
	case "kes":
		handleClipCommand(bot, message)
	case "gif_yap":
		handleGifCommand(bot, message)
	case "portlar":
		handlePortsCommand(bot, message)
	case "getir":
		handleGetFileCommand(bot, message)
	case "aciklama_ekle":
		handleAddDescriptionCommand(bot, message)
	case "aciklama_sil":
		handleRemoveDescriptionCommand(bot, message)
	case "aciklamalar":
		handleListDescriptionsCommand(bot, message)
	case "aciklama_ara":
		handleSearchDescriptionsCommand(bot, message)
	case "liste":
		handleListFilesCommand(bot, message)
	case "klasor":
		handleListCategoryCommand(bot, message)
	case "ara":
		handleSearchFilesCommand(bot, message)
	case "sil":
		handleDeleteFileCommand(bot, message)
	case "yenidenadlandir":
		handleRenameFileCommand(bot, message)
	case "tasi":
		handleMoveFileCommand(bot, message)
	case "izle":
		handleToggleInternetMonitorCommand(bot, message)
	case "indir":
		handleDownloadCommand(bot, message)
	case "indir_ses":
		handleAudioDownloadCommand(bot, message)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Anlaşılmayan komut. Yardım için `/help` yazabilirsiniz.")
		bot.Send(msg)
	}
}

// handleFile, kullanıcı tarafından komut olmadan gönderilen içerikleri (dosya, resim,
// sesli not vb.) işler. Bu içerikleri sunucuya indirir ve kaydeder.
func handleFile(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	var fileID, fileName, mimeType string
	var fileSize int64

	// * Gelen mesajın türüne göre (döküman, fotoğraf, video vb.)
	// * dosya bilgilerini ilgili alanlardan çıkarır.
	switch {
	case message.Document != nil:
		fileID, fileName, fileSize, mimeType = message.Document.FileID, message.Document.FileName, int64(message.Document.FileSize), message.Document.MimeType
	case message.Photo != nil:
		photo := message.Photo[len(message.Photo) - 1] // En yüksek çözünürlüklü fotoğrafı seç.
		fileID, fileName, fileSize, mimeType = photo.FileID, fmt.Sprintf("photo_%d.jpg", time.Now().Unix()), int64(photo.FileSize), "image/jpeg"
	case message.Video != nil:
		fileID, fileName, fileSize, mimeType = message.Video.FileID, message.Video.FileName, int64(message.Video.FileSize), message.Video.MimeType
	case message.Audio != nil:
		fileID, fileName, fileSize, mimeType = message.Audio.FileID, message.Audio.FileName, int64(message.Audio.FileSize), message.Audio.MimeType
	case message.Voice != nil:
		fileID, fileName, fileSize, mimeType = message.Voice.FileID, fmt.Sprintf("voice_%d.ogg", time.Now().Unix()), int64(message.Voice.FileSize), message.Voice.MimeType
	case message.VideoNote != nil:
		videoNote := message.VideoNote
		fileID, fileName, fileSize, mimeType = videoNote.FileID, fmt.Sprintf("video_note_%d.mp4", time.Now().Unix()), int64(videoNote.FileSize), "video/mp4"
	default:
		return
	}

	// * Bazen Telegram dosya adı göndermez. Bu durumda, MIME türünden
	// * yola çıkarak bir dosya uzantısı bulmaya çalışırız.
	if fileName == "" {
		exts, _ := mime.ExtensionsByType(mimeType)
		if len(exts) > 0 {
			fileName = fmt.Sprintf("file_%d%s", time.Now().Unix(), exts[0])
		} else {
			fileName = fmt.Sprintf("file_%d", time.Now().Unix())
		}
	}

	// * `GetFileDirectURL` ile dosyayı doğrudan indirebileceğimiz bir URL alırız.
	fileURL, err := bot.GetFileDirectURL(fileID)
	if err != nil {
		log.Printf("Dosya URL'si alınamadı: %v", err)
		return
	}
	// * Standart Go `net/http` paketi ile dosyayı indir.
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Dosya indirilemedi: %v", err)
		return
	}
	defer resp.Body.Close()

	// * İndirilen veriyi `Gelenler` klasöründeki yeni bir dosyaya yaz.
	savePath := filepath.Join(config.BaseDir, fileName)
	file, err := os.Create(savePath)
	if err != nil {
		log.Printf("Dosya oluşturulamadı: %v", err)
		return
	}
	defer file.Close()
	io.Copy(file, resp.Body)

	log.Printf("Dosya kaydedildi: %s", fileName)
	replyText := fmt.Sprintf(
		"✅ *Dosya kaydedildi!*\n\n"+
			"📄 *Ad:* `%s`\n"+
			"📏 *Boyut:* %.1f KB\n"+
			"📁 *Kategori:* %s\n\n"+
			"💡 `/aciklama_ekle \"%s\" Açıklama...` ile not ekleyebilirsiniz.",
		fileName, float64(fileSize)/1024, getFileCategory(fileName), fileName,
	)
	reply := tgbotapi.NewMessage(message.Chat.ID, replyText)
	reply.ParseMode = "Markdown"
	bot.Send(reply)
}

// handleCallbackQuery, kullanıcıların inline butonlara (örn: silme onayı)
// tıkladığında gönderilen geri çağrı isteklerini işler.
func handleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	// * `NewCallback`, butona basıldıktan sonraki "yükleniyor" animasyonunu durdurur.
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	bot.Request(callback)

	data := callbackQuery.Data
	// * Gelen veriyi (`sil_evet_dosya.txt`) `_` karakterine göre parçalara ayır.
	parts := strings.SplitN(data, "_", 3)
	if len(parts) < 3 {
		return
	}

	command, action, payload := parts[0], parts[1], parts[2]
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	if command == "sil" {
		var newText string
		if action == "evet" {
			// * Kullanıcı "Evet" butonuna bastı, asıl silme işlemi burada gerçekleşir.
			filename := payload
			filePath, found := findFile(filename)
			if !found {
				newText = fmt.Sprintf("❌ Hata: `%s` dosyası zaten silinmiş veya taşınmış.", filename)
			} else if err := os.Remove(filePath); err != nil {
				newText = fmt.Sprintf("❌ `%s` dosyası silinirken bir hata oluştu.", filename)
			} else {
				removeDescription(filename) // Dosyayı siliyorsak, açıklamasını da silelim.
				newText = fmt.Sprintf("🗑️ `%s` dosyası başarıyla silindi.", filename)
			}
		} else {
			newText = "👍 Silme işlemi iptal edildi."
		}
		// * Butonların bulunduğu orijinal mesajı, sonuç metniyle düzenle.
		// * Bu, sohbet ekranının temiz kalmasını sağlar.
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
		editMsg.ParseMode = "Markdown"
		bot.Send(editMsg)
	}
}