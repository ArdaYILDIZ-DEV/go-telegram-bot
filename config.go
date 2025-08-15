// config.go
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// #############################################################################
// #                           YAPILANDIRMA YÖNETİMİ
// #############################################################################
// Bu dosya, botun çalışması için gerekli olan tüm ayarları `.env` dosyasından
// veya sistem ortam değişkenlerinden okuyarak global bir yapıya (struct) yükler.
// Hassas bilgilerin koddan ayrılmasını sağlar.

// Config, program genelinde kullanılacak tüm yapılandırma ayarlarını tutar.
type Config struct {
	BotToken         string
	BaseDir          string
	MetadataFilePath string
	MonitoredPorts   []int
	AdminChatID      int64
	AllowedIDs       []int64
}

// config, tüm program tarafından erişilebilen global yapılandırma nesnesidir.
var config Config

// loadConfig, program başlangıcında çalışarak ayarları yükler.
func loadConfig() error {
	if err := godotenv.Load(); err != nil {
		log.Println("Uyarı: .env dosyası bulunamadı.")
	}

	config.BotToken = os.Getenv("BOT_TOKEN")
	if config.BotToken == "" {
		return fmt.Errorf("BOT_TOKEN .env dosyasında bulunamadı")
	}

	// === YÖNETİCİ VE İZİNLİ KULLANICI AYARLARI ===
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

	// ******************** DEĞİŞİKLİK BURADA BAŞLIYOR ********************
	// # BaseDir (Ana Klasör) yolunu .env dosyasından oku.
	baseDir := os.Getenv("BASE_DIR")
	if baseDir == "" {
		// # Eğer .env dosyasında BASE_DIR belirtilmemişse, programın çalıştığı
		// # dizinde "Gelenler" adında bir klasör kullanılmasını varsay.
		// # Bu, hızlı kurulum için bir kolaylık sağlar.
		log.Println("Uyarı: BASE_DIR .env dosyasında belirtilmemiş. Varsayılan olarak 'Gelenler' klasörü kullanılacak.")
		baseDir = "Gelenler"
	}
	config.BaseDir = baseDir
	log.Printf("Ana çalışma klasörü ayarlandı: %s", config.BaseDir)

	// # Metadata dosyasının yolu. Bu, ana klasöre göreceli olarak kalabilir.
	// # Daha da esnek olması istenirse bu da .env'ye taşınabilir.
	config.MetadataFilePath = "metadata.json"
	// ******************** DEĞİŞİKLİK BURADA BİTİYOR ********************

	portsStr := os.Getenv("MONITORED_PORTS")
	if portsStr != "" {
		parts := strings.Split(portsStr, ",")
		for _, part := range parts {
			port, err := strconv.Atoi(strings.TrimSpace(part))
			if err == nil {
				config.MonitoredPorts = append(config.MonitoredPorts, port)
			}
		}
	}

	log.Println("Yapılandırma başarıyla yüklendi.")
	return nil
}