// command_handlers.go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shirou/gopsutil/v3/process"
)

// Burası komutların asıl işi yaptığı yer. `telegram_bot.go` dosyasındaki
// yönlendirici, komutları buradaki uygun fonksiyonlara dağıtır.
//
// Temel amaçları:
//   - Komutla gelen argümanları (örn: dosya adı, URL) almak.
//   - Gerekli diğer fonksiyonları (dosya arama, sistem bilgisi alma vb.) çağırmak.
//   - Sonucu kullanıcıya düzgün bir şekilde formatlayıp göndermek.

// Basit, genellikle sadece metin döndüren komutları bir arada toplayan fonksiyon.
func handleGeneralCommands(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")
	msg.ParseMode = "Markdown"
	command := message.Command()

	switch command {
	// `/start`: Hoş geldin mesajı.
	case "start":
		msg.Text = "*Hoş geldin*\n\n" +
			"Bu sistem; dosya erişimi, medya yönetimi ve sistem denetimi gibi işlemleri Telegram üzerinden kontrol edebilmen için optimize edildi.\n\n" +
			"Tüm komutları listelemek için `/help` komutunu kullanabilirsin."

	// `/help`: Tüm komutları listeleyen yardım menüsü.
	case "help":
		msg.Text = "*-Komut Seti –*\n\n" +
			"📁 *Dosya Yönetimi:*\n" +
			"`/getir <dosya>` – Dosyayı gönder\n" +
			"`/sil <dosya>` – Dosyayı sil (onaylı)\n" +
			"`/yenidenadlandir <eski> <yeni>` – Dosyayı yeniden adlandır\n" +
			"`/tasi <dosya> <klasör>` – Dosyayı taşı\n\n" +
			"🔍 *Arama ve Listeleme:*\n" +
			"`/ara <kelime>` – Dosya adlarında ara\n" +
			"`/liste` – Ana klasördeki dosyaları göster\n" +
			"`/klasor <kategori>` – Kategori klasörünü listele\n\n" +
			"📝 *Açıklama Yönetimi:*\n" +
			"`/aciklama_ekle <dosya> <açıklama>`\n" +
			"`/aciklama_sil <dosya>`\n" +
			"`/aciklamalar` – Tüm açıklamaları listele\n" +
			"`/aciklama_ara <kelime>` – Açıklamalarda ara\n\n" +
			"🌐 *İndirme ve Medya İşleme:*\n" +
			"`/indir <URL> [kalite] [format]` – Video/dosya indir\n" +
			"`/indir_ses <URL> [format]` – Sadece sesi indir\n" +
			"`/kes <dosya> <baş> <bitiş>` – Video kes\n" +
			"`/gif_yap <dosya> <bitiş>` – GIF üret\n\n" +
			"🖥️ *Sistem ve İşlem Yönetimi:*\n" +
			"`/gorevler` – İnteraktif görev yöneticisi (Yönetici)\n" +
			"`/calistir <yol> <süre>` – Betik çalıştır (Yönetici)\n" +
			"`/kapat <PID>` – Çalışan işlemi durdur (Yönetici)\n" +
			"`/duzenle` – Dosyaları otomatik kategorilere ayır\n" +
			"`/durum` – Temel sistem durumu\n" +
			"`/sistem_bilgisi` – Ayrıntılı sistem bilgisi (Yönetici)\n" +
			"`/hiz_testi` – İndirme/yükleme hızı ve ping ölçümü\n" +
			"`/portlar` – İzlenen port durumları\n" +
			"`/ss` – Ekran görüntüsü al (Yönetici)\n" +
			"`/kayit_al`, `/kayit_durdur` – Ekran kaydı (Yönetici)\n" +
			"`/izle` – Ağ bağlantısını izlemeye başla/durdur"

	// `/duzenle`: "Gelenler" klasöründeki dosyaları kategorilere ayırır.
	case "duzenle":
		count := organizeFiles()
		msg.Text = fmt.Sprintf("🗂️ Dosyalar kategorilere göre yeniden düzenlendi.\nTaşınan dosya sayısı: *%d*", count)

	// `/sistem_bilgisi`: Detaylı sistem raporu sunar.
	case "sistem_bilgisi":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "📊 Sistem bilgileri getiriliyor..."))
		msg.Text = getSystemInfoText(true)

	// `/durum`: Anlık ve özet sistem raporu.
	case "durum":
		msg.Text = getSystemInfoText(false)

	// `/hiz_testi`: İnternet bağlantı hızını ölçer.
	case "hiz_testi":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "⏳ Bağlantı testi başlatılıyor..."))
		speedTestResult, err := runSpeedTest()
		if err != nil {
			msg.Text = fmt.Sprintf("❌ Test başarısız:\n`%v`", err)
		} else {
			// Hız testi sonuçları genellikle byte/saniye cinsinden gelir.
			// Mbps'ye (Megabit per second) çevirmek için 8 ile çarpıp (byte -> bit)
			// 1 milyona (1e6) bölmek gerekir.
			downloadMbps := float64(speedTestResult.Download.Bandwidth*8) / 1e6
			uploadMbps := float64(speedTestResult.Upload.Bandwidth*8) / 1e6
			ping := speedTestResult.Ping.Latency
			quality := getConnectionQuality(downloadMbps, ping)
			msg.Text = fmt.Sprintf(
				"📡 *İnternet Hız Raporu*\n\n"+
					"🧠 Değerlendirme: *%s*\n"+
					"⬇️ İndirme: *%.2f Mbps*\n"+
					"⬆️ Yükleme: *%.2f Mbps*\n"+
					"📶 Gecikme (ping): *%.2f ms*",
				quality, downloadMbps, uploadMbps, ping,
			)
		}
	}
	if msg.Text != "" {
		bot.Send(msg)
	}
}

// `/izle` komutu, internet kesinti izleyicisini açıp kapatır.
func handleToggleInternetMonitorCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	// `monitorMutex` kilidi, `internetMonitorEnabled` global değişkenine
	// aynı anda iki farklı yerden erişilmesini engelleyerek race condition'ı önler.
	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	internetMonitorEnabled = !internetMonitorEnabled // Durumu tersine çevir.

	var statusText string
	if internetMonitorEnabled {
		statusText = "🟢 *Aktif*"
		log.Println("İnternet izleyici kullanıcı tarafından AKTİF edildi.")
		// Monitör açılır açılmaz bir kontrol tetikleyelim ki kullanıcı
		// bir sonraki periyodu beklemek zorunda kalmasın.
		go checkInternetConnection(bot)
	} else {
		statusText = "🔴 *Pasif*"
		log.Println("İnternet izleyici kullanıcı tarafından PASİF edildi.")
		internetDown = false // Monitör kapanırsa kesinti durumunu sıfırla.
	}

	msgText := fmt.Sprintf("📡 *İnternet Kesinti Monitörü* durumu güncellendi:\n\nDurum: %s", statusText)
	bot.Send(tgbotapi.NewMessage(chatID, msgText))
}

// Bir dosyayı bir yerden başka bir yere taşır.
func handleMoveFileCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	chatID := message.Chat.ID
	// `SplitN` ile 2'ye bölmek, hedef klasör adında boşluk olsa bile
	// onu tek bir parça olarak almamızı sağlar.
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		reply := "❌ Kullanım: `/tasi <dosya_adı> <hedef_klasör>`\n" + "Örnek: `/tasi rapor.pdf Dokümanlar`"
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}
	sourceFileName, targetFolderPath := parts[0], parts[1]
	sourcePath, found := findFile(sourceFileName)
	if !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Taşınacak dosya bulunamadı: `%s`", sourceFileName)))
		return
	}
	// GÜVENLİK: Kullanıcının `../` gibi ifadelerle ana çalışma dizininin
	// dışına çıkmasını engelle.
	cleanTarget := filepath.Clean(targetFolderPath)
	if strings.HasPrefix(cleanTarget, "..") {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Geçersiz hedef klasör! Üst dizinlere çıkılamaz."))
		return
	}
	absoluteTargetDir := filepath.Join(config.BaseDir, cleanTarget)
	if err := os.MkdirAll(absoluteTargetDir, os.ModePerm); err != nil {
		log.Printf("Hedef klasör oluşturulamadı: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Hedef klasör oluşturulurken bir hata oluştu."))
		return
	}
	targetPath := filepath.Join(absoluteTargetDir, sourceFileName)
	err := os.Rename(sourcePath, targetPath)
	if err != nil {
		log.Printf("Dosya taşınamadı: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Dosya taşınırken bir hata oluştu."))
		return
	}
	log.Printf("Dosya taşındı: %s -> %s", sourcePath, targetPath)
	reply := fmt.Sprintf("✅ Dosya başarıyla taşındı.\n\n📄 `%s`\n⬇️\n📁 `%s`", sourceFileName, cleanTarget)
	bot.Send(tgbotapi.NewMessage(chatID, reply))
}

// Sunucudaki bir dosyayı Telegram üzerinden kullanıcıya gönderir.
func handleGetFileCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	chatID := message.Chat.ID
	if args == "" {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kullanım: `/getir dosya_adı.uzantı`"))
		return
	}
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🔍 `%s` dosyası aranıyor...", args)))
	filePath, found := findFile(args)
	if !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `%s` dosyası bulunamadı!", args)))
		return
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Dosya bilgileri okunamadı: `%s`", args)))
		return
	}
	// Telegram'ın botlar için dosya gönderme limiti 50MB.
	if fileInfo.Size() > 50*1024*1024 {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Dosya çok büyük! (%.1f MB). Limit 50 MB.", float64(fileInfo.Size())/1024/1024)))
		return
	}
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	var captionBuilder strings.Builder
	captionBuilder.WriteString(fmt.Sprintf("📄 *%s*", filepath.Base(filePath)))
	// Dosyanın açıklaması varsa onu da ekleyelim.
	if description, ok := getDescription(args); ok {
		captionBuilder.WriteString(fmt.Sprintf("\n\n📝 *Açıklama:*\n%s", description))
	}
	doc.Caption = captionBuilder.String()
	doc.ParseMode = "Markdown"
	if _, err := bot.Send(doc); err != nil {
		log.Printf("Dosya gönderilemedi: %v", err)
	}
}

// Bir dosyaya açıklama ekler.
func handleAddDescriptionCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	chatID := message.Chat.ID
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kullanım: `/aciklama_ekle <dosya_adı> <açıklama>`"))
		return
	}
	filename := parts[0]
	description := parts[1]
	if _, found := findFile(filename); !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `%s` dosyası bulunamadı!", filename)))
		return
	}
	if err := addDescription(filename, description); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Açıklama eklenirken bir hata oluştu."))
	} else {
		reply := fmt.Sprintf("✅ *Açıklama eklendi!*\n\n📄 *Dosya:* `%s`\n📝 *Açıklama:* %s", filename, description)
		bot.Send(tgbotapi.NewMessage(chatID, reply))
	}
}

// Bir dosyanın açıklamasını siler.
func handleRemoveDescriptionCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	filename := message.CommandArguments()
	chatID := message.Chat.ID
	if filename == "" {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kullanım: `/aciklama_sil <dosya_adı>`"))
		return
	}
	if err := removeDescription(filename); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `%s` için açıklama bulunamadı veya silinemedi.", filename)))
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ `%s` dosyasının açıklaması silindi.", filename)))
	}
}

// Kayıtlı tüm dosya açıklamalarını listeler.
func handleListDescriptionsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	metadataMutex.Lock()
	defer metadataMutex.Unlock()
	if len(fileMetadata) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "📝 Henüz hiçbir dosyaya açıklama eklenmemiş!"))
		return
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("📝 *Tüm Dosya Açıklamaları (%d adet):*\n\n", len(fileMetadata)))
	for filename, meta := range fileMetadata {
		builder.WriteString(fmt.Sprintf("📄 `%s`\n   💬 _%s_\n\n", filename, meta.Description))
	}
	builder.WriteString("💡 *Dosya almak için:* `/getir dosya_adı.uzantı`")
	bot.Send(tgbotapi.NewMessage(chatID, builder.String()))
}

// Açıklamalarda ve dosya adlarında arama yapar.
func handleSearchDescriptionsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	keyword := message.CommandArguments()
	chatID := message.Chat.ID
	if keyword == "" {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kullanım: `/aciklama_ara <anahtar_kelime>`"))
		return
	}
	results := searchDescriptions(keyword)
	if len(results) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Açıklamalarda `%s` ile eşleşen sonuç bulunamadı.", keyword)))
		return
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🔍 *Açıklama Arama Sonuçları: `%s` (%d adet)*\n\n", keyword, len(results)))
	for filename, desc := range results {
		builder.WriteString(fmt.Sprintf("📄 `%s`\n   💬 _%s_\n\n", filename, desc))
	}
	builder.WriteString("💡 *Dosya almak için:* `/getir dosya_adı.uzantı`")
	bot.Send(tgbotapi.NewMessage(chatID, builder.String()))
}

// Ana dizindeki dosyaları listeler.
func handleListFilesCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	files, err := os.ReadDir(config.BaseDir)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Ana klasördeki dosyalar okunurken bir hata oluştu."))
		return
	}
	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}
	if len(fileNames) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "📁 Ana klasör boş!"))
		return
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("📁 *Ana Klasör Dosyaları (%d adet):*\n", len(fileNames)))
	builder.WriteString(fmt.Sprintf("```\n%s\n```", strings.Join(fileNames, "\n")))
	builder.WriteString("\n💡 *Dosya almak için:* `/getir <dosya_adı>`")
	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
}

// Belirli bir kategori klasöründeki dosyaları listeler.
func handleListCategoryCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	category := message.CommandArguments()
	chatID := message.Chat.ID
	if category == "" {
		var cats []string
		for k := range kategoriler {
			cats = append(cats, k)
		}
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Kategori belirtmediniz.\nMevcut Kategoriler:\n`%s`", strings.Join(cats, "`, `"))))
		return
	}
	categoryDir := filepath.Join(config.BaseDir, category)
	files, err := os.ReadDir(categoryDir)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `%s` kategorisi bulunamadı veya okunamadı.", category)))
		return
	}
	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}
	if len(fileNames) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("📁 `%s` klasörü boş.", category)))
		return
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("📁 *%s Klasöründeki Dosyalar (%d adet):*\n", category, len(fileNames)))
	builder.WriteString(fmt.Sprintf("```\n%s\n```", strings.Join(fileNames, "\n")))
	builder.WriteString("\n💡 *Dosya almak için:* `/getir <dosya_adı>`")
	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
}

// Tüm klasörlerde dosya adı araması yapar.
func handleSearchFilesCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	keyword := message.CommandArguments()
	chatID := message.Chat.ID
	if keyword == "" {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kullanım: `/ara <anahtar_kelime>`"))
		return
	}
	type FoundFile struct {
		Name string
		Path string
	}
	var foundFiles []FoundFile
	// `filepath.Walk` tüm dosya sistemini (belirtilen kökten başlayarak)
	// bizim için gezer. Her bulduğu dosya/klasör için aşağıdaki fonksiyonu çalıştırır.
	filepath.Walk(config.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.Contains(strings.ToLower(info.Name()), strings.ToLower(keyword)) {
			relPath, _ := filepath.Rel(config.BaseDir, path)
			foundFiles = append(foundFiles, FoundFile{Name: info.Name(), Path: relPath})
		}
		return nil
	})
	if len(foundFiles) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Adında `%s` geçen dosya bulunamadı.", keyword)))
		return
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("🔍 *Arama Sonuçları: `%s` (%d adet)*\n\n", keyword, len(foundFiles)))
	for _, file := range foundFiles {
		builder.WriteString(fmt.Sprintf("📄 `%s`\n   _Konum: %s_\n\n", file.Name, filepath.Dir(file.Path)))
	}
	builder.WriteString("💡 *Dosya almak için:* `/getir <dosya_adı>`")
	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// Bir dosyayı silmek için kullanıcıdan onay ister. Asıl silme işi callback'te.
func handleDeleteFileCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	filename := message.CommandArguments()
	chatID := message.Chat.ID
	if filename == "" {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kullanım: `/sil <dosya_adı>`"))
		return
	}
	_, found := findFile(filename)
	if !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Silinecek dosya bulunamadı: `%s`", filename)))
		return
	}
	// Silme gibi geri alınamaz bir işlemden önce mutlaka onay iste.
	// Bu, inline butonlar ve callback'ler ile sağlanır.
	text := fmt.Sprintf("⚠️ *Emin misiniz?*\n\n`%s` dosyası kalıcı olarak silinecek. Bu işlem geri alınamaz.", filename)
	// Callback verisi, hangi butona ve hangi dosya için basıldığını anlamamızı sağlar.
	// Format: `eylem_sonuç_veri`
	yesButtonData := fmt.Sprintf("sil_evet_%s", filename)
	noButtonData := fmt.Sprintf("sil_iptal_%s", filename)

	var keyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Evet, Sil", yesButtonData),
			tgbotapi.NewInlineKeyboardButtonData("❌ İptal", noButtonData),
		),
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// Bir dosyanın adını değiştirir.
func handleRenameFileCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	chatID := message.Chat.ID
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Kullanım: `/yenidenadlandir <eski_ad> <yeni_ad>`"))
		return
	}
	oldName := parts[0]
	newName := parts[1]
	oldPath, found := findFile(oldName)
	if !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Dosya bulunamadı: `%s`", oldName)))
		return
	}
	newPath := filepath.Join(filepath.Dir(oldPath), newName)
	err := os.Rename(oldPath, newPath)
	if err != nil {
		log.Printf("Dosya yeniden adlandırılamadı: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Dosya yeniden adlandırılırken bir hata oluştu."))
		return
	}
	// Eğer dosyanın bir açıklaması varsa, bu açıklamanın kaybolmaması için
	// eski kaydı silip yenisini oluşturalım.
	if desc, ok := getDescription(oldName); ok {
		removeDescription(oldName)
		addDescription(newName, desc)
	}
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Dosya yeniden adlandırıldı:\n`%s` -> `%s`", oldName, newName)))
}

// Yapılandırmada belirtilen portların durumunu kontrol eder.
func handlePortsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	if len(config.MonitoredPorts) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "ℹ️ İzlenecek port listesi .env dosyasında ayarlanmamış veya boş."))
		return
	}
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🔍 Portlar kontrol ediliyor: %v", config.MonitoredPorts)))
	activePorts, err := checkListeningPorts(config.MonitoredPorts)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Hata: %v\n💡 Yönetici izni gerekebilir.", err)))
		return
	}
	var builder strings.Builder
	builder.WriteString("📊 *Port Durum Raporu:*\n\n")
	// Portları sayısal olarak sıralamak, her seferinde aynı sırada ve daha
	// okunaklı bir rapor sunulmasını sağlar.
	sortedPorts := config.MonitoredPorts
	sort.Ints(sortedPorts)
	for _, port := range sortedPorts {
		if info, ok := activePorts[port]; ok {
			builder.WriteString(fmt.Sprintf("🟢 *Port %d:* KULLANIMDA\n   - `%s` (PID: %d)\n", port, info.Name, info.PID))
		} else {
			builder.WriteString(fmt.Sprintf("🔴 *Port %d:* BOŞ\n", port))
		}
	}
	bot.Send(tgbotapi.NewMessage(chatID, builder.String()))
}

// ##############################
// #      GÖREV YÖNETİCİSİ      #
// ##############################

// Tek bir işlem hakkındaki temel bilgileri tutar.
type ProcessDetail struct {
	PID  int32
	Name string
	CPU  float64
	RAM  uint64 // Karşılaştırma için byte olarak kalsın.
}

const processesPerPage = 15 // Her sayfada kaç işlem gösterilecek.

// `/gorevler` komutunu ilk kez karşılar ve varsayılan olarak CPU'ya
// göre sıralanmış ilk sayfayı gönderir.
func handleListProcessesCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msg := createProcessListMessage(message.Chat.ID, "cpu", "desc", 1)
	bot.Send(msg)
}

// İşlem listesini oluşturan, sıralayan, sayfalayan ve butonları hazırlayan ana fonksiyon.
func createProcessListMessage(chatID int64, sortKey, sortDir string, page int) tgbotapi.MessageConfig {
	allProcesses, err := process.Processes()
	if err != nil {
		return tgbotapi.NewMessage(chatID, "❌ İşlem listesi alınırken bir hata oluştu.")
	}

	var processDetails []ProcessDetail
	for _, p := range allProcesses {
		// Hata kontrolleri önemli, bazı sistem işlemlerine erişim izni olmayabilir.
		var detail ProcessDetail
		detail.PID = p.Pid

		cpu, err := p.CPUPercent()
		if err != nil {
			cpu = 0.0 // Hata varsa 0 kabul et
		}
		detail.CPU = cpu

		name, err := p.Name()
		if err != nil {
			name = "[erişilemedi]" // Hata varsa belirt
		}
		detail.Name = name

		mem, err := p.MemoryInfo()
		if err != nil {
			detail.RAM = 0 // Hata varsa 0 kabul et
		} else {
			detail.RAM = mem.RSS // Sadece hata YOKSA .RSS'e eriş
		}

		processDetails = append(processDetails, detail)
	}

	// Gelen isteğe göre dilimi sırala.
	sort.Slice(processDetails, func(i, j int) bool {
		switch sortKey {
		case "cpu":
			if sortDir == "desc" {
				return processDetails[i].CPU > processDetails[j].CPU
			}
			return processDetails[i].CPU < processDetails[j].CPU
		case "ram":
			if sortDir == "desc" {
				return processDetails[i].RAM > processDetails[j].RAM
			}
			return processDetails[i].RAM < processDetails[j].RAM
		default:
			return false
		}
	})

	// Sayfalama mantığı.
	totalProcesses := len(processDetails)
	totalPages := (totalProcesses + processesPerPage - 1) / processesPerPage
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * processesPerPage
	end := start + processesPerPage
	if end > totalProcesses {
		end = totalProcesses
	}
	pageProcesses := processDetails[start:end]

	// Mesaj içeriğini oluştur.
	var builder strings.Builder
	builder.WriteString("🖥️ *Aktif Görev Yöneticisi*\n\n")
	builder.WriteString("```\n")
	builder.WriteString("PID     CPU     RAM      İsim\n")
	builder.WriteString("------- ------- -------- -----------------\n")
	for _, p := range pageProcesses {
		ramMB := float64(p.RAM) / 1024 / 1024
		builder.WriteString(fmt.Sprintf("%-7d %-6.1f%% %-7.1fMB %-17.17s\n", p.PID, p.CPU, ramMB, p.Name))
	}
	builder.WriteString("```\n")

	sortDirText := "Azalan"
	if sortDir == "asc" {
		sortDirText = "Artan"
	}
	sortKeyText := "CPU"
	if sortKey == "ram" {
		sortKeyText = "RAM"
	}

	builder.WriteString(fmt.Sprintf("`Sayfa: %d / %d  |  Sıralama: %s (%s)`\n\n", page, totalPages, sortKeyText, sortDirText))
	builder.WriteString("💡 Bir işlemi sonlandırmak için: `/kapat <PID>`")

	// Butonların bir sonraki durumunu belirle (örn: CPU desc ise bir sonraki asc olacak).
	nextCpuDir := "desc"
	if sortKey == "cpu" && sortDir == "desc" {
		nextCpuDir = "asc"
	}
	nextRamDir := "desc"
	if sortKey == "ram" && sortDir == "desc" {
		nextRamDir = "asc"
	}

	// Buton metinlerini dinamik olarak ayarla.
	cpuButtonText := "📊 CPU"
	if sortKey == "cpu" {
		cpuButtonText = fmt.Sprintf("📊 CPU (%s)", sortDirText)
	}
	ramButtonText := "🧠 RAM"
	if sortKey == "ram" {
		ramButtonText = fmt.Sprintf("🧠 RAM (%s)", sortDirText)
	}

	// Butonları oluştur.
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Önceki", fmt.Sprintf("gorevler_sayfa_%s_%s_%d", sortKey, sortDir, page-1)),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Yenile", fmt.Sprintf("gorevler_sayfa_%s_%s_%d", sortKey, sortDir, page)),
			tgbotapi.NewInlineKeyboardButtonData("▶️ Sonraki", fmt.Sprintf("gorevler_sayfa_%s_%s_%d", sortKey, sortDir, page+1)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(cpuButtonText, fmt.Sprintf("gorevler_sirala_cpu_%s_1", nextCpuDir)),
			tgbotapi.NewInlineKeyboardButtonData(ramButtonText, fmt.Sprintf("gorevler_sirala_ram_%s_1", nextRamDir)),
		),
	)

	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	return msg
}