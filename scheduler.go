// scheduler.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                         ZAMANLAYICI VE DOSYA İZLEYİCİ
// #############################################################################
// Bu dosya, botun proaktif ve otonom yeteneklerinin merkezidir.
// Belirli aralıklarla tekrarlanan görevleri (saatlik rapor, port kontrolü)
// ve dosya sistemindeki değişiklikleri (yeni dosya eklenmesi) anlık olarak
// takip eden mekanizmaları içerir. `runScheduler` fonksiyonu, bir "goroutine"
// olarak arka planda sürekli çalışır.

// * Bu global değişkenler, zamanlayıcının farklı görevleri arasındaki
// * durumu (state) takip etmek için kullanılır. Her bir grup, ilgili
// * mutex (kilit) ile "thread-safe" hale getirilmiştir.
var (
	lastPortStatus         = make(map[int]bool)
	portStatusMutex        = &sync.Mutex{}
	magicFilesProcessed    = make(map[string]bool)
	magicFilesMutex        = &sync.Mutex{}
	internetMonitorEnabled = true // Monitör başlangıçta aktif olsun.
	internetDown           = false
	downtimeStartTime      time.Time
	monitorMutex           sync.Mutex
)

// runScheduler, botun ana döngüsünü başlatır. Bu döngü, zamanlanmış olayları
// (ticker'lar) ve dosya sistemi olaylarını (watcher) dinler.
func runScheduler(bot *tgbotapi.BotAPI) {
	if config.AdminChatID == 0 {
		log.Println("Uyarı: Yönetici Chat ID'si ayarlanmadığı için zamanlayıcı başlatılamıyor.")
		return
	}
	log.Println("Zamanlayıcı ve Dosya İzleyici başlatıldı.")

	// * `time.NewTicker`, belirtilen aralıklarla bir kanala sinyal gönderen
	// * bir zamanlayıcı oluşturur. Bu, periyodik görevler için idealdir.
	hourlyTicker := time.NewTicker(1 * time.Hour)
	portTicker := time.NewTicker(5 * time.Minute)
	internetTicker := time.NewTicker(1 * time.Minute)
	defer hourlyTicker.Stop()
	defer portTicker.Stop()
	defer internetTicker.Stop()

	// * `fsnotify.NewWatcher`, dosya sistemindeki olayları (oluşturma, yazma, silme)
	// * dinleyen bir izleyici oluşturur. Bu, "Magic Folder" gibi özellikler için kullanılır.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Dosya İzleyici oluşturulamadı: %v", err)
	}
	defer watcher.Close()
	magicFolderPath := filepath.Join(config.BaseDir, "TelegramaGonder")
	os.MkdirAll(magicFolderPath, os.ModePerm)
	watcher.Add(config.BaseDir)
	watcher.Add(magicFolderPath)

	// * Başlangıçta bazı kontrolleri hemen yap.
	go organizeFiles()
	go checkAndNotifyPortStatus(bot)

	// * ÖNEMLİ: Bu `for` ve `select` yapısı, Go'nun eşzamanlılık (concurrency)
	// * modelinin kalbidir. Program, birden çok kanaldan (saatlik, port, dosya olayı vb.)
	// * hangisinden "ilk önce" bir sinyal gelirse, o `case` bloğunu çalıştırır ve
	// * sonra tekrar dinlemeye döner. Bu, programın takılmadan birden çok olayı
	// * aynı anda beklemesini sağlar.
	for {
		select {
		case <-hourlyTicker.C:
			log.Println("Saatlik görevler çalışıyor...")
			organizeFiles()
			sendAutomaticSystemInfo(bot)
		case <-portTicker.C:
			log.Println("Periyodik port kontrolü çalışıyor...")
			checkAndNotifyPortStatus(bot)
		case <-internetTicker.C:
			checkInternetConnection(bot)
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				info, err := os.Stat(event.Name)
				if err != nil || info.IsDir() {
					continue
				}
				if filepath.Dir(event.Name) == magicFolderPath {
					// * "Magic Folder" (`TelegramaGonder`) klasörüne yeni bir dosya atıldığında
					// * bu blok tetiklenir. Dosyanın tekrar tekrar işlenmesini önlemek için
					// * bir kontrol mekanizması kullanılır.
					magicFilesMutex.Lock()
					if _, processing := magicFilesProcessed[event.Name]; processing {
						magicFilesMutex.Unlock()
						continue
					}
					magicFilesProcessed[event.Name] = true
					magicFilesMutex.Unlock()
					log.Printf("[Magic Folder] Yeni dosya işlem kuyruğuna alındı: %s", event.Name)
					go sendAndDeleteFile(bot, event.Name)
				} else if filepath.Dir(event.Name) == config.BaseDir {
					log.Printf("[Gelenler] Yeni dosya algılandı: %s", event.Name)
					go organizeFiles()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("İzleyici Hatası: %v", err)
		}
	}
}

// checkInternetConnection, internet bağlantısının durumunu `ping` komutu ile
// kontrol eder ve durum değişikliği (kesilme/geri gelme) olduğunda bildirim gönderir.
func checkInternetConnection(bot *tgbotapi.BotAPI) {
	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	if !internetMonitorEnabled {
		return
	}

	cmd := exec.Command("ping", "-n", "1", "8.8.8.8")
	if runtime.GOOS != "windows" {
		cmd = exec.Command("ping", "-c", "1", "8.8.8.8")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	err := cmd.Run()
	currentlyUp := err == nil // Hata yoksa, internet var demektir.

	// * Durum makinesi: Önceki durum (internetDown) ile şimdiki durumu
	// * (currentlyUp) karşılaştırarak sadece değişiklik olduğunda eyleme geç.
	if currentlyUp && internetDown {
		duration := time.Since(downtimeStartTime).Round(time.Second)
		log.Printf("İnternet geri geldi. Kesinti süresi: %s", duration)
		msgText := fmt.Sprintf("✅ *İnternet Bağlantısı Geri Geldi!*\n\n🕒 Toplam kesinti süresi: *%s*", duration)
		bot.Send(tgbotapi.NewMessage(config.AdminChatID, msgText))
		internetDown = false
	} else if !currentlyUp && !internetDown {
		log.Println("İnternet bağlantısı kesildi.")
		downtimeStartTime = time.Now()
		internetDown = true
		msgText := "❌ *İnternet Bağlantısı Kesildi!*\n\nZamanlayıcı başlatıldı. Bağlantı geri geldiğinde bilgilendirileceksiniz."
		bot.Send(tgbotapi.NewMessage(config.AdminChatID, msgText))
	}
}

// sendAndDeleteFile, "Magic Folder" (`TelegramaGonder`) içine atılan bir dosyayı
// otomatik olarak yöneticiye gönderir ve ardından sunucudan siler.
func sendAndDeleteFile(bot *tgbotapi.BotAPI, filePath string) {
	defer func() {
		magicFilesMutex.Lock()
		delete(magicFilesProcessed, filePath)
		magicFilesMutex.Unlock()
	}()
	// * Dosya oluşturulduktan hemen sonra işleme almamak için kısa bir bekleme süresi.
	// * Bu, dosyanın tamamen yazılmasını beklemek için basit bir yöntemdir.
	time.Sleep(2 * time.Second)
	log.Printf("Gönderiliyor: %s", filePath)
	doc := tgbotapi.NewDocument(config.AdminChatID, tgbotapi.FilePath(filePath))
	doc.Caption = "Magic Folder'dan otomatik gönderim."
	_, err := bot.Send(doc)
	if err != nil {
		log.Printf("Sihirli Klasör'den dosya gönderilemedi: %v", err)
		return
	}
	err = os.Remove(filePath)
	if err != nil {
		log.Printf("Sihirli Klasör'deki dosya silinemedi: %v", err)
	} else {
		log.Printf("Gönderildi ve silindi: %s", filePath)
	}
}

// checkAndNotifyPortStatus, izlenen portların durumunu kontrol eder ve bir önceki
// duruma göre bir değişiklik (açılma/kapanma) varsa yöneticiye bildirim gönderir.
func checkAndNotifyPortStatus(bot *tgbotapi.BotAPI) {
	if len(config.MonitoredPorts) == 0 {
		return
	}
	activePorts, err := checkListeningPorts(config.MonitoredPorts)
	if err != nil {
		log.Printf("Port kontrolü sırasında hata: %v", err)
		return
	}
	portStatusMutex.Lock()
	defer portStatusMutex.Unlock()
	for _, port := range config.MonitoredPorts {
		_, isCurrentlyActive := activePorts[port]
		wasActive, known := lastPortStatus[port]

		// * Sadece durum değiştiyse (veya port ilk kez kontrol ediliyorsa) bildirim gönder.
		// * Bu, her 5 dakikada bir gereksiz mesaj gönderilmesini önler.
		if !known || isCurrentlyActive != wasActive {
			var messageText string
			if isCurrentlyActive && !wasActive {
				messageText = fmt.Sprintf("✅ Port Açıldı: *%d*", port)
			} else if !isCurrentlyActive && wasActive {
				messageText = fmt.Sprintf("❌ Port Kapandı: *%d*", port)
			}
			if messageText != "" && config.AdminChatID != 0 {
				msg := tgbotapi.NewMessage(config.AdminChatID, messageText)
				msg.ParseMode = "Markdown"
				bot.Send(msg)
			}
		}
		// * Portun son bilinen durumunu güncelle.
		lastPortStatus[port] = isCurrentlyActive
	}
}

// sendAutomaticSystemInfo, saatlik olarak tetiklenir ve yöneticiye
// detaylı sistem durumu ile birlikte bir hız testi sonucunu raporlar.
func sendAutomaticSystemInfo(bot *tgbotapi.BotAPI) {
	if config.AdminChatID == 0 {
		return
	}
	systemInfo := getSystemInfoText(true)
	speedTestResult, err := runSpeedTest()
	var speedTestText string
	if err != nil {
		speedTestText = "Hız testi yapılamadı."
	} else {
		downloadMbps := float64(speedTestResult.Download.Bandwidth*8) / 1e6
		uploadMbps := float64(speedTestResult.Upload.Bandwidth*8) / 1e6
		ping := speedTestResult.Ping.Latency
		speedTestText = fmt.Sprintf("*💨 Hız Testi:* %.1f↓ / %.1f↑ Mbps (%.1fms ping)",
			downloadMbps, uploadMbps, ping)
	}
	finalReport := fmt.Sprintf("⏰ *Saatlik Sistem Raporu*\n\n%s\n\n%s", systemInfo, speedTestText)
	msg := tgbotapi.NewMessage(config.AdminChatID, finalReport)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}