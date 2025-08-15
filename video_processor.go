// video_processor.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                             VİDEO İŞLEMCİSİ
// #############################################################################
// Bu dosya, sunucudaki yerel video dosyaları üzerinde işlem yapan
// komutları içerir. `FFmpeg` komut satırı aracını kullanarak video kesme (`/kes`)
// ve videodan GIF oluşturma (`/gif_yap`) gibi medya işleme görevlerini yürütür.
// Bu işlemler, karışıklığı önlemek için özel bir `KırpmaKlasörü` içinde çalışır.

// getClippingFolderPath, medya işleme komutlarının çalışacağı özel klasörün
// yolunu döndürür. Eğer klasör mevcut değilse oluşturur.
func getClippingFolderPath() string {
	path := filepath.Join(config.BaseDir, "KırpmaKlasörü")
	os.MkdirAll(path, os.ModePerm)
	return path
}

// findVideoToClip, sadece `KırpmaKlasörü` içinde bir video arar.
// Bu, kullanıcıların sistemdeki diğer videolara erişmesini kısıtlar.
func findVideoToClip(videoName string) (string, bool) {
	filePath := filepath.Join(getClippingFolderPath(), videoName)
	if _, err := os.Stat(filePath); err == nil {
		return filePath, true
	}
	return "", false
}

// handleClipCommand, /kes komutunu işler. Belirtilen videonun, belirtilen
// başlangıç ve bitiş zamanları arasındaki bölümünü keser.
func handleClipCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	args := message.CommandArguments()
	parts := strings.Fields(args)

	if len(parts) != 3 {
		reply := "❌ Kullanım: `/kes <dosya_adı> <başlangıç> <bitiş>`\n" +
			"Örnek: `/kes videom.mp4 00:01:25 00:01:32`"
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}

	videoName, startTime, endTime := parts[0], parts[1], parts[2]

	sourcePath, found := findVideoToClip(videoName)
	if !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `KırpmaKlasörü` içinde `%s` adında bir video bulunamadı!", videoName)))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✂️ `%s` videosu kesiliyor...", videoName)))

	// * İşlenen dosyayı, sistemin geçici dosyalar klasörüne kaydetmek,
	// * ana çalışma dizinini temiz tutar.
	outputFileName := fmt.Sprintf("kesilmis_%d.mp4", time.Now().Unix())
	outputPath := filepath.Join(os.TempDir(), outputFileName)

	// * ÖNEMLİ: `-c copy` argümanı, FFmpeg'e videoyu yeniden kodlamamasını,
	// * sadece belirtilen bölümü kopyalamasını söyler. Bu, işlemi
	// * inanılmaz derecede hızlandırır (saniyeler içinde tamamlanır).
	cmd := exec.Command("ffmpeg",
		"-i", sourcePath,
		"-ss", startTime,
		"-to", endTime,
		"-c", "copy",
		outputPath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg kesme hatası: %s\n%v", string(output), err)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Video kesilirken bir hata oluştu. Zaman formatını kontrol edin (sa:dk:sn).")))
		return
	}

	log.Printf("Video başarıyla kesildi: %s", outputFileName)
	bot.Send(tgbotapi.NewMessage(chatID, "✅ Video kesildi, şimdi gönderiliyor..."))

	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(outputPath))
	video.Caption = fmt.Sprintf("`%s` dosyasından kesilen klip.", videoName)
	video.ParseMode = "Markdown"
	if _, err := bot.Send(video); err != nil {
		log.Printf("Kesilen video gönderilemedi: %v", err)
	}

	// * Geçici dosya, gönderim sonrası silinir.
	os.Remove(outputPath)
}

// handleGifCommand, /gif_yap komutunu işler. Bir videonun belirli bir
// bölümünden yüksek kaliteli bir GIF oluşturur.
func handleGifCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	args := message.CommandArguments()
	parts := strings.Fields(args)

	if len(parts) != 3 {
		reply := "❌ Kullanım: `/gif_yap <dosya_adı> <başlangıç> <bitiş>`\n" +
			"Örnek: `/gif_yap videom.mp4 00:01:25 00:01:32`"
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}

	videoName, startTime, endTime := parts[0], parts[1], parts[2]

	sourcePath, found := findVideoToClip(videoName)
	if !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `KırpmaKlasörü` içinde `%s` adında bir video bulunamadı!", videoName)))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✨ `%s` videosundan GIF oluşturuluyor... Bu işlem biraz sürebilir.", videoName)))

	outputFileName := fmt.Sprintf("gif_%d.gif", time.Now().Unix())
	outputPath := filepath.Join(os.TempDir(), outputFileName)

	// * ÖNEMLİ: Bu karmaşık filtre zinciri, FFmpeg'in yüksek kaliteli bir GIF
	// * oluşturmasını sağlar. Önce videodan özel bir renk paleti çıkarır,
	// * sonra bu paleti kullanarak GIF'i oluşturur. Bu, standart GIF'lere
	// * göre çok daha iyi renk doğruluğu ve daha az "beneklenme" (dithering) sağlar.
	filter := "fps=15,scale=480:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse"

	cmd := exec.Command("ffmpeg",
		"-i", sourcePath,
		"-ss", startTime,
		"-to", endTime,
		"-vf", filter,
		outputPath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("FFmpeg GIF yapma hatası: %s\n%v", string(output), err)
		bot.Send(tgbotapi.NewMessage(chatID, "❌ GIF oluşturulurken bir hata oluştu."))
		return
	}

	log.Printf("GIF başarıyla oluşturuldu: %s", outputFileName)
	bot.Send(tgbotapi.NewMessage(chatID, "✅ GIF oluşturuldu, şimdi gönderiliyor..."))

	// * GIF'i bir `NewDocument` yerine `NewAnimation` olarak göndermek,
	// * Telegram'ın onu otomatik oynatılan bir animasyon olarak göstermesini sağlar.
	animation := tgbotapi.NewAnimation(chatID, tgbotapi.FilePath(outputPath))
	animation.Caption = fmt.Sprintf("`%s` dosyasından oluşturulan GIF.", videoName)
	animation.ParseMode = "Markdown"
	if _, err := bot.Send(animation); err != nil {
		log.Printf("GIF gönderilemedi: %v", err)
	}

	os.Remove(outputPath)
}