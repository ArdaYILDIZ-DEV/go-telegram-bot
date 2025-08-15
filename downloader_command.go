// downloader_command.go
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                             İNDİRME YÖNETİCİSİ
// #############################################################################
// Bu dosya, `/indir` ve `/indir_ses` komutlarının tüm mantığını içerir.
// Gelen URL'nin türünü (direkt dosya mı, video platformu mu) analiz eder
// ve işi uygun fonksiyona yönlendirir. Her iki indirme türü için de
// kullanıcıya anlık ilerleme durumu bildirme yeteneğine sahiptir.

// ProgressWriter, `io.Writer` arayüzünü uygulayarak yazılan byte miktarını
// sayan ve indirme ilerlemesini takip etmek için kullanılan bir yapıdır.
type ProgressWriter struct {
	Downloaded int64
	Total      int64
}

// Write, gelen veriyi sayar. `atomic.AddInt64` kullanmak, birden çok
// goroutine'in bu sayaca aynı anda güvenli bir şekilde yazmasını sağlar.
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	atomic.AddInt64(&pw.Downloaded, int64(n))
	return n, nil
}

// isVideoPlatformURL, verilen bir URL'nin yt-dlp ile işlenmesi gereken
// popüler video platformlarından birine ait olup olmadığını kontrol eder.
func isVideoPlatformURL(urlStr string) bool {
	platforms := []string{"youtube.com", "youtu.be", "vimeo.com", "twitter.com", "instagram.com"}
	for _, p := range platforms {
		if strings.Contains(strings.ToLower(urlStr), p) {
			return true
		}
	}
	return false
}

// handleDownloadCommand, `/indir` komutu için ana yönlendiricidir.
// Gelen URL'yi analiz eder ve işi `handleYtDlpDownload` veya `handleDirectDownload`'a devreder.
func handleDownloadCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := strings.Fields(message.CommandArguments())
	if len(args) == 0 {
		reply := "❌ Kullanım: `/indir <URL> [kalite] [format]`"
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, reply))
		return
	}
	urlStr := args[0]
	if _, err := url.ParseRequestURI(urlStr); err != nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Geçersiz URL: `%s`", urlStr)))
		return
	}

	if isVideoPlatformURL(urlStr) {
		handleYtDlpDownload(bot, message, args)
	} else {
		handleDirectDownload(bot, message, urlStr)
	}
}

// handleYtDlpDownload, `yt-dlp` CLI aracını kullanarak video indirir.
func handleYtDlpDownload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, args []string) {
	chatID := message.Chat.ID
	urlStr := args[0]
	var quality, format string
	if len(args) > 1 {
		quality = strings.ToLower(args[1])
	}
	if len(args) > 2 {
		format = strings.ToLower(args[2])
	}

	// * ÖNEMLİ: Buradaki mantık, kullanıcının kalite ve format tercihlerinden
	// * bir "öncelik zinciri" oluşturur. `yt-dlp`, bu zinciri (`/` ile ayrılmış)
	// * sırayla dener ve başarılı olduğu ilk seçenekte durur.
	// * Bu, "istenen format yok" hatalarını en aza indiren çok esnek bir yaklaşımdır.
	var preferences []string
	qualitySelector := ""
	if quality != "" && quality != "best" {
		qualitySelector = fmt.Sprintf("[height<=?%s]", strings.TrimSuffix(quality, "p"))
	}
	formatSelector := ""
	if format != "" {
		formatSelector = fmt.Sprintf("[ext=%s]", format)
	}
	if quality != "" && format != "" {
		preferences = append(preferences, fmt.Sprintf("bestvideo%s%s+bestaudio", qualitySelector, formatSelector))
	}
	if quality != "" {
		preferences = append(preferences, fmt.Sprintf("bestvideo%s+bestaudio", qualitySelector))
	}
	if format != "" {
		preferences = append(preferences, fmt.Sprintf("bestvideo%s+bestaudio", formatSelector))
	}
	preferences = append(preferences, "bestvideo+bestaudio")
	preferences = append(preferences, "best")
	formatStr := strings.Join(preferences, "/")

	statusMsg, _ := bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("▶️ Video hazırlanıyor (Tercihler: Kalite=%s, Format=%s)...", quality, format)))
	cmdArgs := []string{"--progress", "--newline", "--force-overwrites", "-f", formatStr, "-o", filepath.Join(config.BaseDir, "%(title)s.%(ext)s"), urlStr}
	cmd := exec.Command("yt-dlp", cmdArgs...)

	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	var progressPercentage float64
	var progressMutex sync.Mutex

	if err := cmd.Start(); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ yt-dlp başlatılamadı: %v", err)))
		return
	}

	// * `yt-dlp`'nin standart hata çıktısını (stderr) ayrı bir tamponda yakalarız.
	// * Bu, indirme başarısız olduğunda kullanıcıya detaylı hata mesajı göstermemizi sağlar.
	var stderrBuf bytes.Buffer
	go func() { io.Copy(&stderrBuf, stderrPipe) }()

	// * Bu goroutine, `yt-dlp`'nin ilerleme çıktısını anlık olarak okur,
	// * içinden yüzdelik değeri ayrıştırır ve bunu global bir değişkene yazar.
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "[download]") && strings.Contains(line, "%") {
				fields := strings.Fields(line)
				for _, field := range fields {
					if strings.HasSuffix(field, "%") {
						percentStr := strings.TrimSuffix(field, "%")
						if p, err := strconv.ParseFloat(percentStr, 64); err == nil {
							progressMutex.Lock()
							progressPercentage = p
							progressMutex.Unlock()
						}
					}
				}
			}
		}
	}()

	// * Bu ikinci goroutine, periyodik olarak (2 saniyede bir) ilerleme değişkenini
	// * okur ve eğer bir değişiklik varsa Telegram'daki durum mesajını günceller.
	doneChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		var lastSentPercent float64 = -1
		for {
			select {
			case <-doneChan:
				return
			case <-ticker.C:
				progressMutex.Lock()
				currentPercent := progressPercentage
				progressMutex.Unlock()
				if int(currentPercent) > int(lastSentPercent) {
					editMsg := tgbotapi.NewEditMessageText(chatID, statusMsg.MessageID, fmt.Sprintf("⏳ Video indiriliyor: *%.0f%%*", currentPercent))
					editMsg.ParseMode = "Markdown"
					bot.Request(editMsg)
					lastSentPercent = currentPercent
				}
			}
		}
	}()

	err := cmd.Wait()
	close(doneChan)
	bot.Request(tgbotapi.NewDeleteMessage(chatID, statusMsg.MessageID))
	if err != nil {
		errMsg := fmt.Sprintf("❌ Video indirilirken hata oluştu.\n`%v`\n\n**Hata Detayı:**\n`%s`", err, stderrBuf.String())
		if len(errMsg) > 4096 {
			errMsg = errMsg[:4000] + "..."
		}
		bot.Send(tgbotapi.NewMessage(chatID, errMsg))
		return
	}
	bot.Send(tgbotapi.NewMessage(chatID, "✅ Video başarıyla `Gelenler` klasörüne indirildi."))
}

// handleAudioDownloadCommand, `yt-dlp` kullanarak sadece ses dosyası indirir.
func handleAudioDownloadCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	args := strings.Fields(message.CommandArguments())
	if len(args) < 1 {
		reply := "❌ Kullanım: `/indir_ses <URL> [format] [kalite]`\nÖrnek: `/indir_ses <link> opus` veya `/indir_ses <link> mp3 0`"
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}
	urlStr := args[0]
	audioFormat := "opus"
	if len(args) > 1 {
		audioFormat = strings.ToLower(args[1])
	}
	audioQuality := "0"
	if len(args) > 2 {
		audioQuality = args[2]
	}
	statusMsg, _ := bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🎵 Ses hazırlanıyor (Format: %s, Kalite: %s)...", audioFormat, audioQuality)))
	
	// * ÖNEMLİ: `-x` ve `--audio-format` argümanları, `yt-dlp`'ye videoyu
	// * tamamen yoksayıp sadece en iyi ses akışını indirmesini ve ardından
	// * belirtilen formata (opus, mp3, flac vb.) dönüştürmesini söyler.
	cmdArgs := []string{"-x", "--audio-format", audioFormat, "--audio-quality", audioQuality, "-f", "bestaudio/best", "--progress", "--newline", "--force-overwrites", "-o", filepath.Join(config.BaseDir, "%(title)s.%(ext)s"), urlStr}
	
	cmd := exec.Command("yt-dlp", cmdArgs...)

	// ... (İlerleme takibi ve hata yönetimi mantığı handleYtDlpDownload ile aynıdır)
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()
	var progressPercentage float64
	var progressMutex sync.Mutex
	if err := cmd.Start(); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ yt-dlp başlatılamadı: %v", err)))
		return
	}
	var stderrBuf bytes.Buffer
	go func() { io.Copy(&stderrBuf, stderrPipe) }()
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "[download]") && strings.Contains(line, "%") {
				fields := strings.Fields(line)
				for _, field := range fields {
					if strings.HasSuffix(field, "%") {
						percentStr := strings.TrimSuffix(field, "%")
						if p, err := strconv.ParseFloat(percentStr, 64); err == nil {
							progressMutex.Lock()
							progressPercentage = p
							progressMutex.Unlock()
						}
					}
				}
			}
		}
	}()
	doneChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		var lastSentPercent float64 = -1
		for {
			select {
			case <-doneChan:
				return
			case <-ticker.C:
				progressMutex.Lock()
				currentPercent := progressPercentage
				progressMutex.Unlock()
				if int(currentPercent) > int(lastSentPercent) {
					editMsg := tgbotapi.NewEditMessageText(chatID, statusMsg.MessageID, fmt.Sprintf("⏳ Ses indiriliyor: *%.0f%%*", currentPercent))
					editMsg.ParseMode = "Markdown"
					bot.Request(editMsg)
					lastSentPercent = currentPercent
				}
			}
		}
	}()
	err := cmd.Wait()
	close(doneChan)
	bot.Request(tgbotapi.NewDeleteMessage(chatID, statusMsg.MessageID))
	if err != nil {
		errMsg := fmt.Sprintf("❌ Ses indirilirken hata oluştu.\n`%v`\n\n**Hata Detayı:**\n`%s`", err, stderrBuf.String())
		if len(errMsg) > 4096 {
			errMsg = errMsg[:4000] + "..."
		}
		bot.Send(tgbotapi.NewMessage(chatID, errMsg))
		return
	}
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Ses dosyası başarıyla `Gelenler` klasörüne indirildi.")))
}

// handleDirectDownload, standart HTTP GET isteği ile doğrudan dosya indirir.
func handleDirectDownload(bot *tgbotapi.BotAPI, message *tgbotapi.Message, urlStr string) {
	chatID := message.Chat.ID
	statusMsg, err := bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("⬇️ İndirme başlatılıyor...\nURL: `%s`", urlStr)))
	if err != nil {
		return
	}
	editMessage := func(newText string) {
		editMsg := tgbotapi.NewEditMessageText(chatID, statusMsg.MessageID, newText)
		editMsg.ParseMode = "Markdown"
		bot.Request(editMsg)
	}
	resp, err := http.Get(urlStr)
	if err != nil {
		editMessage(fmt.Sprintf("❌ İndirme başarısız: `%v`", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		editMessage(fmt.Sprintf("❌ Sunucu hatası: `%s`", resp.Status))
		return
	}
	fileName := filepath.Base(resp.Request.URL.Path)
	if fileName == "" || fileName == "." || len(fileName) > 200 {
		fileName = fmt.Sprintf("download_%d", time.Now().Unix())
	}
	destPath := filepath.Join(config.BaseDir, fileName)
	file, err := os.Create(destPath)
	if err != nil {
		editMessage(fmt.Sprintf("❌ Dosya oluşturulamadı: `%v`", err))
		return
	}
	defer file.Close()
	totalSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	progress := &ProgressWriter{Total: totalSize}
	
	// * `io.MultiWriter`, gelen veriyi aynı anda birden çok hedefe yazar.
	// * Burada, indirilen veriyi hem diske (`file`) hem de ilerlemeyi sayan
	// * `progress` nesnemize yazıyoruz.
	writer := io.MultiWriter(file, progress)

	doneChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		var lastSentText string
		for {
			select {
			case <-doneChan:
				return
			case <-ticker.C:
				downloaded := atomic.LoadInt64(&progress.Downloaded)
				var statusText string
				if totalSize > 0 {
					percent := int(float64(downloaded) * 100 / float64(totalSize))
					statusText = fmt.Sprintf("⏳ İndiriliyor: *%d%%*\n(%.1f MB / %.1f MB)", percent, float64(downloaded)/1e6, float64(totalSize)/1e6)
				} else {
					statusText = fmt.Sprintf("⏳ İndiriliyor...\n(%.1f MB indirildi)", float64(downloaded)/1e6)
				}
				if statusText != lastSentText {
					editMessage(statusText)
					lastSentText = statusText
				}
			}
		}
	}()
	_, err = io.Copy(writer, resp.Body)
	close(doneChan)
	if err != nil {
		editMessage(fmt.Sprintf("❌ İndirme sırasında hata: `%v`", err))
		os.Remove(destPath)
		return
	}
	bot.Request(tgbotapi.NewDeleteMessage(chatID, statusMsg.MessageID))
	finalDownloaded := atomic.LoadInt64(&progress.Downloaded)
	replyText := fmt.Sprintf("✅ *Dosya başarıyla indirildi!*\n\n📄 *Ad:* `%s`\n📏 *Boyut:* %.1f MB\n📁 *Konum:* Gelenler", fileName, float64(finalDownloaded)/1e6)
	bot.Send(tgbotapi.NewMessage(chatID, replyText))
}