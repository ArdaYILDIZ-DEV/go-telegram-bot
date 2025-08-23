// config.go
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken         string
	BaseDir          string
	MetadataFilePath string
	MonitoredPorts   map[int]string
	AdminChatID      int64
	AllowedIDs       []int64
	GeminiAPIKey     string
	WorkerIntervalInternet time.Duration
	WorkerIntervalPort     time.Duration
	Uygulamalar map[string]string
}

var config Config

func loadConfig() error {
	if err := godotenv.Load(); err != nil {
		log.Println("Uyarı: .env dosyası bulunamadı.")
	}
	
	config.BotToken = os.Getenv("BOT_TOKEN")
	if config.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN .env dosyasında bulunamadı")
	}

	config.GeminiAPIKey = os.Getenv("GEMINI_API_KEY")
	if config.GeminiAPIKey == "" {
		log.Println("Uyarı: GEMINI_API_KEY .env dosyasında bulunamadı. LLM özelliği çalışmayacak.")
	}

	adminIDStr := os.Getenv("ADMIN_CHAT_ID")
	if adminIDStr == "" {
		return fmt.Errorf("KRİTİK HATA: ADMIN_CHAT_ID .env dosyasında ayarlanmamış!")
	}
	id, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("ADMIN_CHAT_ID .env dosyasında geçersiz: %w", err)
	}
	config.AdminChatID = id
	log.Printf("Yönetici ID'si başarıyla yüklendi: %d", config.AdminChatID)

	allowedIDsStr := os.Getenv("ALLOWED_IDS")
	if allowedIDsStr != "" {
		parts := strings.Split(allowedIDsStr, ",")
		for _, part := range parts {
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err == nil {
				config.AllowedIDs = append(config.AllowedIDs, id)
			}
		}
		log.Printf("%d adet izinli kullanıcı yüklendi.", len(config.AllowedIDs))
	}
	
	baseDir := os.Getenv("BASE_DIR")
	if baseDir == "" {
		log.Println("Uyarı: BASE_DIR .env dosyasında belirtilmemiş. Varsayılan olarak 'Gelenler' klasörü kullanılacak.")
		baseDir = "Gelenler"
	}
	config.BaseDir = baseDir
	log.Printf("Ana çalışma klasörü ayarlandı: %s", config.BaseDir)
	
	config.MetadataFilePath = "metadata.json"

	config.MonitoredPorts = make(map[int]string)
	portsStr := os.Getenv("MONITORED_PORTS")
	if portsStr != "" {
		services := strings.Split(portsStr, ",")
		for _, service := range services {
			parts := strings.SplitN(strings.TrimSpace(service), ":", 2)
			if len(parts) != 2 { continue }
			name := parts[0]
			port, err := strconv.Atoi(parts[1])
			if err != nil { continue }
			config.MonitoredPorts[port] = name
		}
	}

	internetInterval, err := strconv.Atoi(os.Getenv("WORKER_INTERVAL_INTERNET"))
	if err != nil || internetInterval <= 0 { internetInterval = 30 }
	config.WorkerIntervalInternet = time.Duration(internetInterval) * time.Second

	portInterval, err := strconv.Atoi(os.Getenv("WORKER_INTERVAL_PORT"))
	if err != nil || portInterval <= 0 { portInterval = 5 }
	config.WorkerIntervalPort = time.Duration(portInterval) * time.Second

	config.Uygulamalar = make(map[string]string)
	uygulamalarStr := os.Getenv("UYGULAMALAR")
	if uygulamalarStr != "" {
		apps := strings.Split(uygulamalarStr, ",")
		for _, app := range apps {
			parts := strings.SplitN(strings.TrimSpace(app), ":", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				log.Printf("Uyarı: .env dosyasındaki uygulama tanımı geçersiz, atlanıyor: '%s'", app)
				continue
			}
			kısayol := strings.ToLower(parts[0])
			yol := parts[1]
			config.Uygulamalar[kısayol] = yol
		}
	}


	log.Println("Yapılandırma başarıyla yüklendi.")
	return nil
}