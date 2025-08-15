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
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Burası botun beyni. Telegram'dan gelen tüm güncellemeler (mesajlar, komutlar,
// buton tıklamaları) burada karşılanır, yetki kontrolünden geçer ve ilgili
// fonksiyona yönlendirilir.

// Ana güncelleme döngüsü. Gelen her şey (mesaj, callback) bu `for` döngüsü
// tarafından yakalanır ve işlenir. Programın kalbi burasıdır.
func handleUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		var userID int64
		var chatID int64
		var fromUserName string

		// Gelen güncellemenin türüne göre temel bilgileri (kim, nereye) alalım.
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
			// Desteklemediğimiz bir güncelleme türü ise atla.
			continue
		}

		// Her şeyden önce: Bu kullanıcı botu kullanabilir mi?
		if !isUserAllowed(userID) {
			log.Printf("⚠️ YETKİSİZ ERİŞİM DENEMESİ! Kullanıcı: %s (%d)", fromUserName, userID)
			bot.Send(tgbotapi.NewMessage(chatID, "🚫 Bu botu kullanma yetkiniz bulunmuyor."))
			continue
		}

		// Gelen güncellemenin içeriğini doğru fonksiyona yönlendir.
		if update.Message != nil {
			if update.Message.IsCommand() {
				handleCommand(bot, update.Message)
			} else {
				// Komut değilse, muhtemelen bir dosyadır.
				handleFile(bot, update.Message)
			}
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery)
		}
	}
}

// Komut yönlendirici. Gelen komutun yönetici yetkisi gerektirip gerektirmediğini
// kontrol eder ve `switch` bloğu ile doğru fonksiyona paslar.
func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	command := message.Command()

	// Bu komutlar sadece botun sahibi tarafından kullanılabilir.
	adminOnlyCommands := map[string]bool{
		"calistir":       true,
		"kapat":          true,
		"sistem_bilgisi": true,
		"kayit_al":       true,
		"kayit_durdur":   true,
		"ss":             true,
		"gorevler":       true,
	}

	// Komut korumalı mı diye kontrol et.
	if _, isProtected := adminOnlyCommands[command]; isProtected {
		if !isUserAdmin(message.From.ID) {
			log.Printf("⚠️ YETKİSİZ KOMUT DENEMESİ! Kullanıcı: %s (%d), Komut: /%s", message.From.UserName, message.From.ID, command)
			bot.Send(tgbotapi.NewMessage(message.Chat.ID, "🚫 Bu komutu sadece yönetici kullanabilir."))
			return
		}
	}

	// Komutları ilgili işleyicilere dağıt.
	switch command {
	case "start", "help", "duzenle", "sistem_bilgisi", "durum", "hiz_testi":
		handleGeneralCommands(bot, message)
	case "gorevler":
		handleListProcessesCommand(bot, message)
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

// Komut olmayan mesajları (dosya, fotoğraf, ses kaydı vb.) işler.
// Gelen dosyayı sunucuya indirir ve kullanıcıya bilgi verir.
func handleFile(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	var fileID, fileName, mimeType string
	var fileSize int64

	// Gelen mesajın içeriği ne? Fotoğraf mı, belge mi, video mu?
	switch {
	case message.Document != nil:
		fileID, fileName, fileSize, mimeType = message.Document.FileID, message.Document.FileName, int64(message.Document.FileSize), message.Document.MimeType
	case message.Photo != nil:
		// En yüksek çözünürlüklü fotoğrafı al.
		photo := message.Photo[len(message.Photo)-1]
		fileID, fileName, fileSize, mimeType = photo.FileID, fmt.Sprintf("photo_%d.jpg", time.Now().Unix()), int64(photo.FileSize), "image/jpeg"
	case message.Video != nil:
		fileID, fileName, fileSize, mimeType = message.Video.FileID, message.Video.FileName, int64(message.Video.FileSize), message.Video.MimeType
	case message.Audio != nil:
		fileID, fileName, fileSize, mimeType = message.Audio.FileID, message.Audio.FileName, int64(message.Audio.FileSize), message.Audio.MimeType
	case message.Voice != nil:
		fileID, fileName, fileSize, mimeType = message.Voice.FileID, fmt.Sprintf("voice_%d.ogg", time.Now().Unix()), int64(message.Voice.FileSize), message.Voice.MimeType
	case message.VideoNote != nil:
		// Yuvarlak video mesajları.
		videoNote := message.VideoNote
		fileID, fileName, fileSize, mimeType = videoNote.FileID, fmt.Sprintf("video_note_%d.mp4", time.Now().Unix()), int64(videoNote.FileSize), "video/mp4"
	default:
		// Sadece metin mesajıysa veya desteklenmeyen bir türse, bir şey yapma.
		return
	}

	// Bazen dosya adı gelmeyebilir, mime türünden bir uzantı uyduralım.
	if fileName == "" {
		exts, _ := mime.ExtensionsByType(mimeType)
		if len(exts) > 0 {
			fileName = fmt.Sprintf("file_%d%s", time.Now().Unix(), exts[0])
		} else {
			fileName = fmt.Sprintf("file_%d", time.Now().Unix())
		}
	}

	// Dosyayı Telegram sunucularından indir.
	fileURL, err := bot.GetFileDirectURL(fileID)
	if err != nil {
		log.Printf("Dosya URL'si alınamadı: %v", err)
		return
	}
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("Dosya indirilemedi: %v", err)
		return
	}
	defer resp.Body.Close()

	// İndirilen dosyayı ana klasöre kaydet.
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

// Inline butonlardan gelen geri aramaları (callback) yönetir.
// Örneğin `/sil` komutundaki "Evet/İptal" butonları veya görev yöneticisindeki
// "Sonraki Sayfa" butonu buraya düşer.
func handleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	// Telegram'a "isteği aldım, bekleme simgesini kaldır" mesajı gönder.
	callback := tgbotapi.NewCallback(callbackQuery.ID, "")
	bot.Request(callback)

	data := callbackQuery.Data
	parts := strings.SplitN(data, "_", 2)
	if len(parts) < 1 {
		return
	}

	command := parts[0]
	chatID := callbackQuery.Message.Chat.ID
	messageID := callbackQuery.Message.MessageID

	if command == "sil" {
		// Callback verisi formatı: sil_evet_dosyaadi.txt veya sil_iptal_dosyaadi.txt
		silParts := strings.SplitN(data, "_", 3)
		if len(silParts) < 3 {
			return
		}

		action, payload := silParts[1], silParts[2]
		var newText string
		if action == "evet" {
			filename := payload
			filePath, found := findFile(filename)
			if !found {
				newText = fmt.Sprintf("❌ Hata: `%s` dosyası zaten silinmiş veya taşınmış.", filename)
			} else if err := os.Remove(filePath); err != nil {
				newText = fmt.Sprintf("❌ `%s` dosyası silinirken bir hata oluştu.", filename)
			} else {
				// Dosya silindiyse, açıklaması da gitsin.
				removeDescription(filename)
				newText = fmt.Sprintf("🗑️ `%s` dosyası başarıyla silindi.", filename)
			}
		} else { // "iptal" durumu
			newText = "👍 Silme işlemi iptal edildi."
		}

		// Butonları kaldırıp yerine sonuç mesajını yaz.
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
		editMsg.ParseMode = "Markdown"
		bot.Send(editMsg)

	} else if command == "gorevler" {
		// Görev yöneticisi callback'leri (sayfalama, sıralama).
		// Format: gorevler_sayfa_cpu_desc_2 veya gorevler_sirala_ram_asc_1
		gorevParts := strings.Split(data, "_")
		if len(gorevParts) != 5 {
			return // Hatalı formatı görmezden gel.
		}

		action := gorevParts[1]
		sortKey := gorevParts[2]
		sortDir := gorevParts[3]
		page, _ := strconv.Atoi(gorevParts[4])

		// Sıralama kriteri değiştiyse her zaman ilk sayfaya dön.
		if action == "sirala" {
			page = 1
		}

		// Yeni liste mesajını oluştur.
		updatedMessage := createProcessListMessage(chatID, sortKey, sortDir, page)

		// ÖNEMLİ: Telegram'da bir mesajın sadece metnini düzenlerseniz,
		// altındaki butonlar kaybolur. Butonları korumak için, hem yeni metni
		// hem de yeni butonları içeren tam bir `EditMessageText` objesi oluşturup
		// tek seferde göndermek gerekir.
		editMsg := tgbotapi.NewEditMessageText(
			chatID,
			messageID,
			updatedMessage.Text,
		)
		editMsg.ParseMode = "Markdown"
		// Oluşturulan yeni butonları da isteğe ekle.
		if markup, ok := updatedMessage.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup); ok {
			editMsg.ReplyMarkup = &markup
		}

		bot.Request(editMsg)
	}
}