// metadata_manager.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// #############################################################################
// #                             METADATA YÖNETİCİSİ
// #############################################################################
// Bu dosya, dosyalara eklenen açıklamaların (metadata) yönetilmesinden
// sorumludur. Tüm açıklamalar bellekte bir haritada (map) tutulur ve
// program her kapandığında veya bir değişiklik olduğunda `metadata.json`
// adlı dosyaya kalıcı olarak kaydedilir.

// FileMetadata, her bir dosya için saklanacak olan verileri tanımlayan yapıdır.
// `json:"..."` etiketleri, bu yapının JSON formatına nasıl çevrileceğini belirtir.
type FileMetadata struct {
	Description string `json:"description"`
	Updated     string `json:"updated"`
}

// * Bu global değişkenler, tüm metadata işlemlerinin merkezidir.
var (
	// fileMetadata, dosya adlarını (string) FileMetadata yapılarına eşleyen
	// ve tüm açıklamaları bellekte tutan ana haritadır.
	fileMetadata map[string]FileMetadata

	// metadataMutex, `fileMetadata` haritasına aynı anda birden fazla yerden
	// (örneğin iki farklı komutla) yazılmasını veya okunmasını engelleyen
	// bir kilittir. Bu, "race condition" hatalarını önler.
	metadataMutex = &sync.Mutex{}
)

// loadMetadata, program başlangıcında çalışarak `metadata.json` dosyasını
// okur ve içeriğini `fileMetadata` haritasına yükler.
func loadMetadata() error {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	fileMetadata = make(map[string]FileMetadata)
	// * Program ilk kez çalışıyorsa veya dosya silinmişse, `metadata.json`
	// * dosyası mevcut olmayabilir. Bu durumda, boş bir dosya oluşturulur.
	if _, err := os.Stat(config.MetadataFilePath); os.IsNotExist(err) {
		log.Println("metadata.json bulunamadı, boş olarak oluşturuluyor.")
		return saveMetadata() // Boş dosyayı kaydet ve çık.
	}

	data, err := os.ReadFile(config.MetadataFilePath)
	if err != nil {
		return fmt.Errorf("metadata dosyası okunamadı: %w", err)
	}
	if len(data) == 0 {
		return nil // Boş dosya bir hata değildir.
	}

	log.Println("Metadata başarıyla yüklendi.")
	// * Okunan JSON verisini `fileMetadata` haritasına "unmarshal" et (çözümle).
	return json.Unmarshal(data, &fileMetadata)
}

// saveMetadata, bellekteki `fileMetadata` haritasının güncel halini
// `metadata.json` dosyasına, okunabilir bir formatta yazar.
func saveMetadata() error {
	// * `json.MarshalIndent`, JSON verisini girintili (insan tarafından okunabilir)
	// * bir şekilde formatlar.
	data, err := json.MarshalIndent(fileMetadata, "", "  ")
	if err != nil {
		return fmt.Errorf("metadata JSON'a çevrilemedi: %w", err)
	}
	// * `os.WriteFile`, veriyi belirtilen dosyaya yazar. `0644` dosya izinleridir.
	return os.WriteFile(config.MetadataFilePath, data, 0644)
}

// addDescription, bir dosyaya yeni bir açıklama ekler veya mevcut olanı günceller.
func addDescription(filename, description string) error {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	fileMetadata[filename] = FileMetadata{
		Description: description,
		Updated:     time.Now().Format(time.RFC3339),
	}
	return saveMetadata()
}

// getDescription, bir dosyanın açıklamasını döndürür.
func getDescription(filename string) (string, bool) {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()
	meta, found := fileMetadata[filename]
	return meta.Description, found
}

// removeDescription, bir dosyanın açıklamasını siler.
func removeDescription(filename string) error {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	if _, found := fileMetadata[filename]; found {
		delete(fileMetadata, filename)
		return saveMetadata()
	}
	return fmt.Errorf("açıklama bulunamadı: %s", filename)
}

// searchDescriptions, verilen anahtar kelimeyi hem dosya adlarında hem de
// açıklamalarda (büyük/küçük harf duyarsız) arar.
func searchDescriptions(keyword string) map[string]string {
	metadataMutex.Lock()
	defer metadataMutex.Unlock()

	results := make(map[string]string)
	keywordLower := strings.ToLower(keyword)

	for filename, meta := range fileMetadata {
		if strings.Contains(strings.ToLower(filename), keywordLower) || strings.Contains(strings.ToLower(meta.Description), keywordLower) {
			results[filename] = meta.Description
		}
	}
	return results
}