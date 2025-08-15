// file_manager.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// #############################################################################
// #                             DOSYA YÖNETİCİSİ
// #############################################################################
// Bu dosya, botun dosya sistemiyle ilgili temel işlemlerini yönetir.
// Dosyaları kategorilere ayırma, belirli bir dosyayı tüm klasörlerde arama
// ve program için gerekli klasör yapısını oluşturma gibi görevleri içerir.

// kategoriler, dosya uzantılarını ait oldukları kategoriyle eşleştiren
// bir haritadır (map). Yeni dosya türleri eklemek için bu liste genişletilebilir.
var kategoriler = map[string][]string{
	"Resimler":   {".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg", ".webp", ".tiff"},
	"Videolar":   {".mp4", ".mkv", ".mov", ".avi", ".webm", ".flv", ".wmv", ".m4v"},
	"Dokümanlar": {".pdf", ".docx", ".doc", ".xlsx", ".xls", ".pptx", ".ppt", ".txt", ".rtf", ".odt", ".ods"},
	"Sesler":     {".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a", ".opus"},
	"Arşivler":   {".zip", ".rar", ".7z", ".tar", ".gz", ".bz2"},
}

// ensureDirectories, program ilk başladığında çalışarak botun ihtiyaç duyduğu
// tüm klasörlerin (ana dizin, kategori klasörleri) var olduğundan emin olur.
// Eğer klasörler mevcut değilse, onları oluşturur.
func ensureDirectories() {
	os.MkdirAll(config.BaseDir, os.ModePerm)
	for category := range kategoriler {
		os.MkdirAll(filepath.Join(config.BaseDir, category), os.ModePerm)
	}
	os.MkdirAll(filepath.Join(config.BaseDir, "Diğer"), os.ModePerm)
}

// getFileCategory, bir dosya adını alır, uzantısını kontrol eder ve
// `kategoriler` haritasına göre hangi kategoriye ait olduğunu döndürür.
// Eşleşme bulunamazsa "Diğer" kategorisini döndürür.
func getFileCategory(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	for category, extensions := range kategoriler {
		for _, e := range extensions {
			if e == ext {
				return category
			}
		}
	}
	return "Diğer"
}

// findFile, verilen dosya adını tüm olası konumlarda (ana dizin, tüm kategori
// klasörleri ve medya işleme klasörleri) arar.
// Dosyayı bulursa, tam yolunu ve `true` değerini döndürür. Bulamazsa, boş string ve `false` döner.
func findFile(filename string) (string, bool) {
	// * Aranacak tüm klasörlerin bir listesi dinamik olarak oluşturulur.
	searchDirs := []string{config.BaseDir}
	for category := range kategoriler {
		searchDirs = append(searchDirs, filepath.Join(config.BaseDir, category))
	}
	searchDirs = append(searchDirs, filepath.Join(config.BaseDir, "Diğer"))
	searchDirs = append(searchDirs, getClippingFolderPath()) // Video kırpma klasörü de dahil edilir.

	for _, dir := range searchDirs {
		filePath := filepath.Join(dir, filename)
		// * `os.Stat`, bir dosya veya dizin hakkında bilgi döndürür.
		// * Hata `nil` ise (yani hata yoksa), bu dosyanın o konumda var olduğu anlamına gelir.
		if _, err := os.Stat(filePath); err == nil {
			return filePath, true
		}
	}
	return "", false
}

// organizeFiles, ana `Gelenler` klasöründeki tüm dosyaları tarar ve her birini
// `getFileCategory` fonksiyonunu kullanarak doğru kategori klasörüne taşır.
func organizeFiles() int {
	files, err := os.ReadDir(config.BaseDir)
	if err != nil {
		log.Printf("Ana klasör okunamadı: %v", err)
		return 0
	}

	organizedCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue // Sadece dosyalarla ilgilen, klasörleri atla.
		}

		sourcePath := filepath.Join(config.BaseDir, file.Name())
		category := getFileCategory(file.Name())
		targetDir := filepath.Join(config.BaseDir, category)
		targetPath := filepath.Join(targetDir, file.Name())

		// * ÖNEMLİ: Bu döngü, hedef klasörde aynı adda bir dosya varsa,
		// * üzerine yazmak yerine dosya adının sonuna `_1`, `_2` gibi
		// * bir sayaç ekleyerek isimlendirme çakışmalarını önler.
		counter := 1
		originalTargetPath := targetPath
		for {
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				// Dosya bu isimle mevcut değil, döngüden çık.
				break
			}
			// Dosya mevcut, yeni bir isim oluştur ve tekrar kontrol et.
			ext := filepath.Ext(originalTargetPath)
			base := strings.TrimSuffix(filepath.Base(originalTargetPath), ext)
			targetPath = filepath.Join(targetDir, fmt.Sprintf("%s_%d%s", base, counter, ext))
			counter++
		}

		// * `os.Rename`, Go'da hem dosyaları yeniden adlandırmak hem de
		// * aynı disk bölümü (volume) içinde taşımak için kullanılır.
		if err := os.Rename(sourcePath, targetPath); err == nil {
			organizedCount++
			log.Printf("Düzenlendi: %s -> %s", file.Name(), category)
		} else {
			log.Printf("Taşıma hatası: %s - %v", file.Name(), err)
		}
	}
	return organizedCount
}