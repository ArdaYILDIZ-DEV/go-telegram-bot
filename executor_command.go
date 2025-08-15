// executor_command.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                         İŞLEM YÜRÜTME VE YÖNETİMİ
// #############################################################################
// Bu dosya, sunucu üzerinde harici komutları ve betikleri çalıştırma (`/calistir`)
// ve bu işlemleri sonlandırma (`/kapat`) gibi kritik ve yöneticiye özel
// yetenekleri içerir.

// safeBuffer, birden çok goroutine'den gelen yazma işlemlerini
// bir "mutex" (kilit) kullanarak güvenli bir şekilde yönetir.
// Bu, "race condition" hatalarını önler.
type safeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (sb *safeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// handleRunCommand, /calistir komutunu işler. Belirtilen betiği,
// belirtilen bir zaman aşımı süresiyle çalıştırır ve çıktısını raporlar.
func handleRunCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	args := message.CommandArguments()
	parts := strings.Fields(args)

	if len(parts) < 2 {
		reply := "❌ Kullanım: `/calistir <dosya_yolu> <süre_saniye>`"
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}

	filePath := parts[0]
	timeoutStr := parts[1]

	timeoutSec, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutSec <= 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Geçersiz süre! Lütfen pozitif bir tam sayı girin."))
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Dosya bulunamadı: `%s`", filePath)))
		return
	}

	msgText := fmt.Sprintf("⏳ `%s` çalıştırılıyor... Azami %d saniye sonra çıktı gönderilecek.", filepath.Base(filePath), timeoutSec)
	bot.Send(tgbotapi.NewMessage(chatID, msgText))

	// * ÖNEMLİ: `context.WithTimeout`, belirtilen süre sonunda `ctx.Done()`
	// * kanalına bir sinyal gönderen bir "zamanlayıcı" oluşturur.
	// * Bu, uzun süren işlemlerin sonsuza dek çalışmasını engeller.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel() // Her durumda context kaynaklarını temizle.

	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".bat", ".cmd":
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", filePath)
	case ".ps1":
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", filePath)
	default:
		cmd = exec.CommandContext(ctx, filePath)
	}
	// * Betiğin, kendi bulunduğu dizinde çalışmasını sağlar. Bu, betik içindeki
	// * göreceli dosya yollarının (örn: "config.txt") doğru çalışması için kritiktir.
	cmd.Dir = filepath.Dir(filePath)

	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	var output safeBuffer
	if err := cmd.Start(); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ İşlem başlatılamadı: %v", err)))
		return
	}

	// * İki ayrı goroutine, işlem çalışırken standart çıktı (stdout) ve standart hata
	// * (stderr) akışlarını anlık olarak okuyup `output` tamponuna yazar.
	go func() { io.Copy(&output, stdoutPipe) }()
	go func() { io.Copy(&output, stderrPipe) }()

	doneChan := make(chan error, 1)
	go func() {
		doneChan <- cmd.Wait() // İşlem bittiğinde bu kanala sinyal gönder.
	}()

	var statusMessage string
	// * ÖNEMLİ: `select` bloğu, iki olaydan hangisinin "önce" gerçekleşeceğini
	// * bekler: ya zaman aşımı (`ctx.Done()`) ya da işlemin kendi kendine bitmesi (`doneChan`).
	// * Bu yapı, her iki senaryoyu da (erken bitme veya zaman aşımına uğrama)
	// * tek bir yerde zarifçe yönetmemizi sağlar.
	select {
	case <-ctx.Done(): // Zaman aşımı kazandı.
		cmd.Process.Kill()
		statusMessage = fmt.Sprintf("⚠️ *İşlem, %d saniyelik zaman aşımına uğradığı için sonlandırıldı.*", timeoutSec)
	case err := <-doneChan: // İşlem erken bitti.
		if err != nil {
			statusMessage = fmt.Sprintf("❌ *İşlem bir hata ile tamamlandı: %v*", err)
		} else {
			statusMessage = "✅ *İşlem başarıyla tamamlandı.*"
		}
	}

	var replyBuilder strings.Builder
	replyBuilder.WriteString(fmt.Sprintf("📄 *Sonuç: `%s`*\n\n", filepath.Base(filePath)))
	replyBuilder.WriteString(statusMessage)

	outputStr := output.String()
	if len(outputStr) > 0 {
		// * Telegram'ın mesaj başına karakter limitini (4096) aşmamak için
		// * çıktıyı belirli bir uzunlukta kesiyoruz.
		const maxLen = 3800
		if len(outputStr) > maxLen {
			outputStr = outputStr[:maxLen] + "\n\n... (Çıktı çok uzun olduğu için kesildi) ..."
		}
		replyBuilder.WriteString(fmt.Sprintf("\n\n```\n%s\n```", outputStr))
	} else {
		replyBuilder.WriteString("\n\n*_(İşlem herhangi bir çıktı üretmedi.)_*")
	}

	finalMsg := tgbotapi.NewMessage(chatID, replyBuilder.String())
	finalMsg.ParseMode = "Markdown"
	bot.Send(finalMsg)
}

// handleKillCommand, /kapat komutunu işler. Belirtilen Process ID'ye (PID)
// sahip olan işlemi sunucu üzerinde sonlandırır.
func handleKillCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	pidStr := message.CommandArguments()

	if pidStr == "" {
		reply := "❌ Kullanım: `/kapat <PID>`\n" +
			"Örnek: `/kapat 12345`"
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Geçersiz PID! Lütfen bir sayı girin: `%s`", pidStr)))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ PID %d ile işlem sonlandırılıyor...", pid)))

	// * `os.FindProcess`, verilen PID'ye sahip olan işlemi bulur.
	// * Bu fonksiyon, işlem gerçekten var olmasa bile hata vermez.
	// * Asıl hata, o işlem üzerinde bir eylem yapmaya çalıştığımızda ortaya çıkar.
	process, err := os.FindProcess(pid)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ PID %d ile işlem bulunurken hata oluştu: %v", pid, err)))
		return
	}

	// * `process.Kill()` işletim sistemine işlemi sonlandırması için
	// * bir sinyal gönderir. Windows'ta bu, `taskkill /f` komutuna benzer.
	err = process.Kill()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ PID %d ile işlem sonlandırılamadı: %v", pid, err)))
		return
	}

	successMsg := fmt.Sprintf("✅ PID %d ile işlem başarıyla sonlandırıldı.", pid)
	bot.Send(tgbotapi.NewMessage(chatID, successMsg))
}