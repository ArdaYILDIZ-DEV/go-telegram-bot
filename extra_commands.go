// extra_commands.go
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                         EKSTRA VE HARİCİ KOMUTLAR
// #############################################################################
// Bu dosya, harici CLI araçlarına (PowerShell, FFmpeg) bağımlı olan ve
// genellikle daha karmaşık arka plan işlemleri gerektiren komutları içerir.
// Ekran görüntüsü alma ve ekran kaydı gibi özellikler burada yönetilir.

// * Bu global değişkenler, `/kayit_al` ile başlatılan ve `/kayit_durdur`
// * ile sonlandırılan ekran kaydı işleminin durumunu (state) tutmak için kullanılır.
// * `recordingMutex`, bu değişkenlere aynı anda birden fazla yerden erişilmesini
// * engelleyerek veri bütünlüğünü korur.
var (
	recordingCmd      *exec.Cmd
	recordingStdin    io.WriteCloser
	recordingFileName string
	recordingMutex    sync.Mutex
)

// handleScreenshotCommand, /ss komutunu işleyerek sunucunun anlık ekran görüntüsünü alır.
func handleScreenshotCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	msg := tgbotapi.NewMessage(chatID, "🖥️ Ekran görüntüsü alınıyor, lütfen bekleyin...")
	bot.Send(msg)

	// * Ekran görüntüsü, geçici olarak Go programının çalıştığı dizine kaydedilir,
	// * gönderildikten sonra silinir.
	tempPath, err := os.Getwd()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Ekran görüntüsü alınırken bir hata oluştu (dizin hatası)."))
		return
	}
	fileName := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
	filePath := filepath.Join(tempPath, fileName)

	// * Bu komut sadece Windows işletim sisteminde çalışır.
	// * `runtime.GOOS` kontrolü, programın farklı sistemlerde çökmesini engeller.
	if runtime.GOOS == "windows" {
		// * .NET kütüphanelerini kullanan tek satırlık bir PowerShell komutu ile ekran görüntüsü alınır.
		psCommand := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $Screen = [System.Windows.Forms.Screen]::PrimaryScreen; $Bounds = $Screen.Bounds; $Bitmap = New-Object System.Drawing.Bitmap($Bounds.Width, $Bounds.Height); $Graphics = [System.Drawing.Graphics]::FromImage($Bitmap); $Graphics.CopyFromScreen($Bounds.Location, [System.Drawing.Point]::Empty, $Bounds.Size); $Bitmap.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png); $Graphics.Dispose(); $Bitmap.Dispose()`, filePath)

		cmd := exec.Command("powershell", "-Command", psCommand)
		// # ÖNEMLİ: `syscall.SysProcAttr{HideWindow: true}` ayarı, komut çalıştırıldığında
		// # arka planda bir PowerShell penceresinin anlık olarak belirip kaybolmasını engeller.
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		if err := cmd.Run(); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Ekran görüntüsü alınamadı."))
			return
		}
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Bu komut sadece Windows işletim sisteminde çalışır."))
		return
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(filePath))
	photo.Caption = fmt.Sprintf("Ekran Görüntüsü - %s", time.Now().Format("02-01-2006 15:04:05"))

	if _, err := bot.Send(photo); err != nil {
		log.Printf("Ekran görüntüsü gönderilemedi: %v", err)
	}
	// * Geçici ekran görüntüsü dosyası, gönderim sonrası silinerek diskte yer kaplaması önlenir.
	os.Remove(filePath)
}

// handleStartRecordingCommand, FFmpeg kullanarak bir ekran kaydı işlemi başlatır.
func handleStartRecordingCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	recordingMutex.Lock()
	defer recordingMutex.Unlock()

	if recordingCmd != nil && recordingCmd.Process != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Zaten devam eden bir kayıt var."))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, "📹 Ekran kaydı başlatılıyor..."))

	fileName := fmt.Sprintf("kayit_%d.mp4", time.Now().Unix())
	filePath := filepath.Join(config.BaseDir, fileName)

	// * FFmpeg komutu, Windows'a özel `gdigrab` ile masaüstünü yakalar ve
	// * düşük gecikmeli, hızlı kodlama ayarlarıyla bir MP4 dosyasına yazar.
	cmd := exec.Command("ffmpeg",
		"-f", "gdigrab", "-framerate", "15", "-i", "desktop",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "28",
		"-pix_fmt", "yuv420p", "-profile:v", "baseline", "-level", "3.0",
		"-movflags", "+faststart",
		filePath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// * ÖNEMLİ: `cmd.StdinPipe()` oluşturmak, çalışan FFmpeg işlemine
	// * daha sonra komut (bizim durumumuzda 'q' harfi) gönderebilmemizi sağlar.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kayıt altyapısı oluşturulamadı."))
		return
	}
	recordingStdin = stdin

	if err := cmd.Start(); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kayıt başlatılamadı. FFmpeg'in kurulu olduğundan emin olun."))
		return
	}

	// * Başlatılan kayıt işleminin bilgilerini global değişkenlere kaydet.
	recordingCmd = cmd
	recordingFileName = fileName
	log.Printf("Ekran kaydı başlatıldı. PID: %d, Dosya: %s", recordingCmd.Process.Pid, recordingFileName)
	bot.Send(tgbotapi.NewMessage(chatID, "✅ Ekran kaydı başlatıldı.\nKaydı durdurmak için `/kayit_durdur` komutunu kullanın."))
}

// handleStopRecordingCommand, daha önce başlatılmış olan ekran kaydını sonlandırır.
func handleStopRecordingCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	recordingMutex.Lock()
	defer recordingMutex.Unlock()

	if recordingCmd == nil || recordingCmd.Process == nil || recordingStdin == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "ℹ️ Devam eden bir kayıt bulunmuyor."))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, "📹 Kayıt durduruluyor ve video işleniyor, lütfen bekleyin..."))
	log.Printf("Ekran kaydı durduruluyor. PID: %d", recordingCmd.Process.Pid)

	// * ÖNEMLİ: FFmpeg'i düzgün bir şekilde kapatmanın en iyi yolu,
	// * standart girdisine (stdin) 'q' karakterini yazmaktır. Bu, videonun
	// * düzgün bir şekilde tamamlanıp kaydedilmesini sağlar.
	_, err := recordingStdin.Write([]byte("q\n"))
	if err != nil {
		// * Eğer 'q' göndermek başarısız olursa, son çare olarak işlemi zorla sonlandır.
		log.Printf("FFmpeg'e 'q' komutu gönderilemedi: %v. İşlem zorla sonlandırılacak.", err)
		recordingCmd.Process.Kill()
	}

	// * `cmd.Wait()`, FFmpeg işleminin tamamen sonlanmasını bekler.
	err = recordingCmd.Wait()
	if err != nil {
		log.Printf("FFmpeg Wait() hatası (genellikle normal): %v", err)
	}

	recordingStdin.Close()

	// * Kayıt durumuyla ilgili global değişkenleri sıfırla, böylece yeni bir kayıt başlatılabilir.
	fileName := recordingFileName
	recordingCmd = nil
	recordingStdin = nil
	recordingFileName = ""
	log.Println("Kayıt işlemi sonlandırıldı.")

	filePath := filepath.Join(config.BaseDir, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Kayıt dosyası (`%s`) bulunamadı!", fileName)))
		return
	}

	bot.Send(tgbotapi.NewMessage(chatID, "📤 Video gönderiliyor..."))
	video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	video.Caption = fmt.Sprintf("Ekran Kaydı - %s", time.Now().Format("02-01-2006 15:04:05"))

	if _, err := bot.Send(video); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Video gönderilirken bir hata oluştu."))
	}

	// * Video gönderildikten sonra sunucudaki kopyasını sil.
	os.Remove(filePath)
}