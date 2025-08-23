// scheduler.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// #############################################################################
// #                     ZAMANLAYICI VE DOSYA İZLEYİCİ
// #############################################################################
// Bu dosya, belirli aralıklarla tekrarlanan (saatlik rapor gibi) ve anlık
// olaylara tepki veren (dosya sistemindeki değişiklikler gibi) görevleri yönetir.

var (
	magicFilesProcessed = make(map[string]bool)
	magicFilesMutex     = &sync.Mutex{}
)

// runScheduler, botun uzun vadeli ve olay bazlı görev döngüsünü başlatır.
func runScheduler(bot *tgbotapi.BotAPI) {
	if config.AdminChatID == 0 {
		log.Println("Uyarı: Yönetici Chat ID'si ayarlanmadığı için zamanlayıcı başlatılamıyor.")
		return
	}
	log.Println("Zamanlayıcı ve Dosya İzleyici başlatıldı.")

	hourlyTicker := time.NewTicker(1 * time.Hour)
	defer hourlyTicker.Stop()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Dosya İzleyici oluşturulamadı: %v", err)
	}
	defer watcher.Close()
	
	magicFolderPath := filepath.Join(config.BaseDir, "TelegramaGonder")
	os.MkdirAll(magicFolderPath, os.ModePerm)
	watcher.Add(config.BaseDir)
	watcher.Add(magicFolderPath)

	go organizeFiles()

	for {
		select {
		case <-hourlyTicker.C:
			log.Println("Saatlik görevler çalışıyor...")
			organizeFiles()
			sendAutomaticSystemInfo(bot)

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

// sendAndDeleteFile, "Magic Folder" içine atılan dosyayı gönderir ve siler.(silmesini istemiyorsanız fonksiyonu değiştirebilirsiniz.)
func sendAndDeleteFile(bot *tgbotapi.BotAPI, filePath string) {
	defer func() {
		magicFilesMutex.Lock()
		delete(magicFilesProcessed, filePath)
		magicFilesMutex.Unlock()
	}()
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

// sendAutomaticSystemInfo, saatlik olarak sistem durumu ve hız testi raporu gönderir.
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