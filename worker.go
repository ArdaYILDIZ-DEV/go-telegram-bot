// worker.go
package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                             ARKA PLAN ÇALIŞANLARI (WORKERS)
// #############################################################################

var (
	lastPortStatus         = make(map[int]bool)
	portStatusMutex        = &sync.Mutex{}
	internetDown           = false
	downtimeStartTime      time.Time
	monitorMutex           sync.Mutex

	// Dinamik kontrol anahtarları
	internetMonitorEnabled = true
	portMonitorEnabled     = true

	// Gönderilemeyen bildirimler için bir kuyruk
	notificationQueue []tgbotapi.Chattable
	queueMutex        = &sync.Mutex{}
)

// startWorkers, tüm arka plan izleyicilerini başlatan ana fonksiyondur.
func startWorkers(bot *tgbotapi.BotAPI) {
	log.Printf("Worker'lar başlatılıyor: İnternet Kontrolü (%s), Port Kontrolü (%s)", config.WorkerIntervalInternet, config.WorkerIntervalPort)

	internetTicker := time.NewTicker(config.WorkerIntervalInternet)
	portTicker := time.NewTicker(config.WorkerIntervalPort)

	go runPortWorker(bot, portTicker)
	go runInternetWorker(bot, internetTicker)
}

// runPortWorker, port durumunu periyodik olarak kontrol eder.
func runPortWorker(bot *tgbotapi.BotAPI, ticker *time.Ticker) {
	checkAndNotifyPortStatus(bot)
	for range ticker.C {
		if portMonitorEnabled {
			checkAndNotifyPortStatus(bot)
		}
	}
}

// runInternetWorker, internet bağlantısını periyodik olarak kontrol eder.
func runInternetWorker(bot *tgbotapi.BotAPI, ticker *time.Ticker) {
	checkInternetConnection(bot)
	for range ticker.C {
		checkInternetConnection(bot)
	}
}

// checkAndNotifyPortStatus, port durumunu kontrol eder ve değişiklikleri bildirir.
func checkAndNotifyPortStatus(bot *tgbotapi.BotAPI) {
	if len(config.MonitoredPorts) == 0 {
		return
	}
	
	var portsToCheck []int
	for port := range config.MonitoredPorts {
		portsToCheck = append(portsToCheck, port)
	}

	activePorts, err := checkListeningPorts(portsToCheck)
	if err != nil {
		log.Printf("Port kontrolü sırasında hata: %v", err)
		return
	}
	portStatusMutex.Lock()
	defer portStatusMutex.Unlock()

	monitorMutex.Lock()
	isInternetDownNow := internetDown
	monitorMutex.Unlock()

	for port, serviceName := range config.MonitoredPorts {
		_, isCurrentlyActive := activePorts[port]
		wasActive, known := lastPortStatus[port]

		if !known || isCurrentlyActive != wasActive {
			var messageText string
			if isCurrentlyActive && !wasActive {
				messageText = fmt.Sprintf("✅ *Servis Başlatıldı:* %s (Port %d)", serviceName, port)
			} else if !isCurrentlyActive && wasActive {
				messageText = fmt.Sprintf("❌ *Servis Durdu:* %s (Port %d)", serviceName, port)
			}
			if messageText != "" && config.AdminChatID != 0 {
				msg := tgbotapi.NewMessage(config.AdminChatID, messageText)
				msg.ParseMode = "Markdown"
				go sendMessageOrQueue(bot, msg, isInternetDownNow)
			}
		}
		lastPortStatus[port] = isCurrentlyActive
	}
}

// sendMessageOrQueue, bir mesajı internet varsa gönderir, yoksa kuyruğa alır.
func sendMessageOrQueue(bot *tgbotapi.BotAPI, msg tgbotapi.Chattable, isInternetDown bool) {
	if isInternetDown {
		queueMutex.Lock()
		notificationQueue = append(notificationQueue, msg)
		queueMutex.Unlock()
		log.Printf("İnternet kesik. Mesaj kuyruğa eklendi. (Kuyruk boyutu: %d)", len(notificationQueue))
		return
	}

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Mesaj gönderilemedi, kuyruğa ekleniyor: %v", err)
		queueMutex.Lock()
		notificationQueue = append(notificationQueue, msg)
		queueMutex.Unlock()
	}
}

// checkInternetConnection, internet bağlantısını kontrol eder ve kuyruğu yönetir.
func checkInternetConnection(bot *tgbotapi.BotAPI) {
	cmd := exec.Command("ping", "-n", "1", "8.8.8.8")
	if runtime.GOOS != "windows" {
		cmd = exec.Command("ping", "-c", "1", "8.8.8.8")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	currentlyUp := err == nil

	monitorMutex.Lock()
	
	if !internetMonitorEnabled {
		monitorMutex.Unlock()
		return
	}

	var messagesToSend []tgbotapi.Chattable
	var processQueue bool

	if currentlyUp && internetDown {
		duration := time.Since(downtimeStartTime).Round(time.Second)
		log.Printf("İnternet geri geldi. Kesinti süresi: %s", duration)
		
		msgText := fmt.Sprintf("✅ *İnternet Bağlantısı Geri Geldi!*\n\n🕒 Toplam kesinti süresi: *%s*", duration)
		messagesToSend = append(messagesToSend, tgbotapi.NewMessage(config.AdminChatID, msgText))
		
		internetDown = false
		processQueue = true

	} else if !currentlyUp && !internetDown {
		log.Println("İnternet bağlantısı kesildi.")
		downtimeStartTime = time.Now()
		internetDown = true
	}

	isDownForQueue := internetDown
	
	monitorMutex.Unlock()

	for _, msg := range messagesToSend {
		go sendMessageOrQueue(bot, msg, isDownForQueue)
	}

	if processQueue {
		queueMutex.Lock()
		if len(notificationQueue) > 0 {
			log.Printf("%d adet bekleyen bildirim gönderiliyor...", len(notificationQueue))
			
			for _, queuedMsg := range notificationQueue {
				go sendMessageOrQueue(bot, queuedMsg, false)
				time.Sleep(500 * time.Millisecond)
			}
			notificationQueue = nil
		}
		queueMutex.Unlock()
	}
}