// command_handlers.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

// handleGeneralCommands, /start, /help, /durum gibi basit metin tabanlı komutları yönetir.
func handleGeneralCommands(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")
	msg.ParseMode = "Markdown"
	command := message.Command()

	switch command {
	case "start":
		msg.Text = "*Hoş geldin*\n\n" +
			"Bu sistem; dosya erişimi, medya yönetimi ve sistem denetimi gibi işlemleri Telegram üzerinden kontrol edebilmen için optimize edildi.\n\n" +
			"Tüm komutları listelemek için `/help` komutunu kullanabilirsin."

	case "help":
		msg.Text = "*-Komut Seti –*\n\n" +
			"^^ *Akıllı Asistan (Sentinel):*\n" +
			"`/llm` – Yapay zeka ile sohbet modunu başlatır\n" +
			"`/llm_kapat` – Aktif sohbet modunu sonlandırır\n\n" +
			"== *Dosya Yönetimi:*\n" +
			"`/getir <dosya>` – Dosyayı gönder\n" +
			"`/sil <dosya>` – Dosyayı sil (onaylı)\n" +
			"`/yenidenadlandir <eski> <yeni>` – Dosyayı yeniden adlandır\n" +
			"`/tasi <dosya> <klasör>` – Dosyayı taşı\n\n" +
			"oo *Arama ve Listeleme:*\n" +
			"`/ara <kelime>` – Dosya adlarında ara\n" +
			"`/liste` – Ana klasördeki dosyaları göster\n" +
			"`/klasor <kategori>` – Kategori klasörünü listele\n\n" +
			":: *Açıklama Yönetimi:*\n" +
			"`/aciklama_ekle <dosya> <açıklama>`\n" +
			"`/aciklama_sil <dosya>`\n" +
			"`/aciklamalar` – Tüm açıklamaları listele\n" +
			"`/aciklama_ara <kelime>` – Açıklamalarda ara\n\n" +
			"//  *İndirme ve Medya İşleme:*\n" +
			"`/indir <URL> [kalite] [format]` – Video/dosya indir\n" +
			"`/indir_ses <URL> [format]` – Sadece sesi indir\n" +
			"`/kes <dosya> <baş> <bitiş>` – Video kes\n" +
			"`/gif_yap <dosya> <bitiş>` – GIF üret\n\n" +
			"==| *Sistem ve İşlem Yönetimi:*\n" +
			"`/gorevler` – İnteraktif görev yöneticisi (Yönetici)\n" +
			"`/kapat <PID>` – Çalışan işlemi durdur (Yönetici)\n" +
			"`/durum` – Temel sistem durumu\n" +
			"`/sistem_bilgisi` – Ayrıntılı sistem bilgisi (Yönetici)\n" +
			"`/hiz_testi` – İndirme/yükleme hızı ve ping ölçümü\n" +
			"`/portlar` – İzlenen port durumları\n" +
			"`/ss` – Ekran görüntüsü al (Yönetici)\n" +
			"`/kayit_al`, `/kayit_durdur` – Ekran kaydı (Yönetici)\n" +
			"`/duzenle` – Dosyaları otomatik kategorilere ayır\n" +
			"`/izle` – Ağ bağlantısını izlemeye başla/durdur\n\n" +
			"++ *Uygulama & Betik Çalıştırma (Yönetici):*\n" +
			"`/calistir <yol> <süre>` – Betik çalıştır ve çıktısını al\n" +
			"`/uygulama_calistir <kısayol>` – Önceden tanımlı uygulamayı başlat\n" +
			"`/calistir_dosya <yol>` – Dosya yolu ile uygulama başlat"

	case "duzenle":
		count := organizeFiles()
		msg.Text = fmt.Sprintf("🗂️ Dosyalar kategorilere göre yeniden düzenlendi.\nTaşınan dosya sayısı: *%d*", count)

	case "sistem_bilgisi":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "📊 Sistem bilgileri getiriliyor..."))
		msg.Text = getSystemInfoText(true)

	case "durum":
		msg.Text = getSystemInfoText(false)
	}
	if msg.Text != "" {
		bot.Send(msg)
	}
}

// handleSpeedTestCommand, /hiz_testi komutunu asenkron olarak işler.
func handleSpeedTestCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	statusMsg, err := bot.Send(tgbotapi.NewMessage(chatID, "⏳ Bağlantı testi başlatılıyor... Bu işlem 30 saniye kadar sürebilir."))
	if err != nil {
		log.Printf("Hız testi başlangıç mesajı gönderilemedi: %v", err)
		return
	}

	// Hız testi uzun sürebileceği için botu bloklamamak adına ayrı bir goroutine'de çalıştırılır.
	go func() {
		var finalText string
		speedTestResult, err := runSpeedTest()

		if err != nil {
			finalText = fmt.Sprintf("❌ Test başarısız:\n`%v`", err)
		} else {
			downloadMbps := float64(speedTestResult.Download.Bandwidth*8) / 1e6
			uploadMbps := float64(speedTestResult.Upload.Bandwidth*8) / 1e6
			ping := speedTestResult.Ping.Latency
			quality := getConnectionQuality(downloadMbps, ping)
			finalText = fmt.Sprintf(
				"📡 *İnternet Hız Raporu*\n\n"+
					"🧠 Değerlendirme: *%s*\n"+
					"⬇️ İndirme: *%.2f Mbps*\n"+
					"⬆️ Yükleme: *%.2f Mbps*\n"+
					"📶 Gecikme (ping): *%.2f ms*",
				quality, downloadMbps, uploadMbps, ping,
			)
		}

		// Başlangıçta gönderilen mesaj düzenlenerek sonuç gösterilir.
		editMsg := tgbotapi.NewEditMessageText(chatID, statusMsg.MessageID, finalText)
		editMsg.ParseMode = "Markdown"
		bot.Request(editMsg)
	}()
}

// handleToggleInternetMonitorCommand, internet kesinti izleyicisini açar veya kapatır.
func handleToggleInternetMonitorCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	internetMonitorEnabled = !internetMonitorEnabled

	var statusText string
	if internetMonitorEnabled {
		statusText = "🟢 *Aktif*"
		log.Println("İnternet izleyici kullanıcı tarafından AKTİF edildi.")
		go checkInternetConnection(bot)
	} else {
		statusText = "🔴 *Pasif*"
		log.Println("İnternet izleyici kullanıcı tarafından PASİF edildi.")
		internetDown = false
	}

	msgText := fmt.Sprintf("📡 *İnternet Kesinti Monitörü* durumu güncellendi:\n\nDurum: %s", statusText)
	bot.Send(tgbotapi.NewMessage(chatID, msgText))
}

// handleMoveFileCommand, bir dosyayı belirtilen klasöre taşır.
func handleMoveFileCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	chatID := message.Chat.ID
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

	// Güvenlik: Üst dizinlere ("..") çıkışı engeller.
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

// handleGetFileCommand, sunucudaki bir dosyayı kullanıcıya gönderir.
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

	// Telegram'ın 50MB'lık dosya gönderme limitini uygular.
	if fileInfo.Size() > 50*1024*1024 {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Dosya çok büyük! (%.1f MB). Limit 50 MB.", float64(fileInfo.Size())/1024/1024)))
		return
	}

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	var captionBuilder strings.Builder
	captionBuilder.WriteString(fmt.Sprintf("📄 *%s*", filepath.Base(filePath)))

	// Varsa, dosyanın açıklamasını da gönderir.
	if description, ok := getDescription(args); ok {
		captionBuilder.WriteString(fmt.Sprintf("\n\n📝 *Açıklama:*\n%s", description))
	}

	doc.Caption = captionBuilder.String()
	doc.ParseMode = "Markdown"
	if _, err := bot.Send(doc); err != nil {
		log.Printf("Dosya gönderilemedi: %v", err)
	}
}

// handleAddDescriptionCommand, bir dosyaya açıklama metni ekler.
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

// handleRemoveDescriptionCommand, bir dosyanın açıklamasını siler.
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

// handleListDescriptionsCommand, kayıtlı tüm dosya açıklamalarını listeler.
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

// handleSearchDescriptionsCommand, açıklamalarda ve dosya adlarında arama yapar.
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

// handleListFilesCommand, ana dizindeki dosyaları listeler.
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
	builder.WriteString(fmt.Sprintf("`%s`", strings.Join(fileNames, "`\n`")))
	builder.WriteString("\n\n💡 *Dosya almak için:* `/getir <dosya_adı>`")

	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		// Mesaj Markdown formatında gönderilemezse, düz metin olarak göndermeyi dene.
		log.Printf("Liste gönderilirken Markdown hatası (fallback denenecek): %v", err)
		msg.ParseMode = ""
		bot.Send(msg)
	}
}

// handleListCategoryCommand, belirli bir kategori klasöründeki dosyaları listeler.
func handleListCategoryCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	category := message.CommandArguments()
	chatID := message.Chat.ID
	if category == "" {
		var cats []string
		for k := range kategoriler {
			cats = append(cats, k)
		}
		sort.Strings(cats)
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
	builder.WriteString(fmt.Sprintf("`%s`", strings.Join(fileNames, "`\n`")))
	builder.WriteString("\n\n💡 *Dosya almak için:* `/getir <dosya_adı>`")

	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ParseMode = "Markdown"
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Klasör listesi gönderilirken Markdown hatası (fallback denenecek): %v", err)
		msg.ParseMode = ""
		bot.Send(msg)
	}
}

// handleSearchFilesCommand, tüm klasörlerde dosya adı araması yapar.
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
	// `filepath.Walk` ile tüm alt dizinler taranır.
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

// handleDeleteFileCommand, bir dosyayı silmek için kullanıcıya onay butonları gönderir.
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
	// Silme işlemi tehlikeli olduğu için inline keyboard ile onay istenir.
	text := fmt.Sprintf("⚠️ *Emin misiniz?*\n\n`%s` dosyası kalıcı olarak silinecek. Bu işlem geri alınamaz.", filename)
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

// handleRenameFileCommand, bir dosyanın adını değiştirir.
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

	// Varsa, dosya açıklamasını da yeni dosyaya taşır.
	if desc, ok := getDescription(oldName); ok {
		removeDescription(oldName)
		addDescription(newName, desc)
	}
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Dosya yeniden adlandırıldı:\n`%s` -> `%s`", oldName, newName)))
}

// handlePortsCommand, yapılandırmada belirtilen portların durumunu kontrol eder.
func handlePortsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	if len(config.MonitoredPorts) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "ℹ️ İzlenecek port listesi .env dosyasında ayarlanmamış veya boş."))
		return
	}

	var portsToCheck []int
	for port := range config.MonitoredPorts {
		portsToCheck = append(portsToCheck, port)
	}

	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🔍 Portlar kontrol ediliyor...")))
	activePorts, err := checkListeningPorts(portsToCheck)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Hata: %v\n💡 Yönetici izni gerekebilir.", err)))
		return
	}
	var builder strings.Builder
	builder.WriteString("📊 *Port Durum Raporu:*\n\n")

	sortedPorts := portsToCheck
	sort.Ints(sortedPorts)

	for _, port := range sortedPorts {
		serviceName := config.MonitoredPorts[port]
		if info, ok := activePorts[port]; ok {
			builder.WriteString(fmt.Sprintf("🟢 *%s* (Port %d): KULLANIMDA\n   - `%s` (PID: %d)\n", serviceName, port, info.Name, info.PID))
		} else {
			builder.WriteString(fmt.Sprintf("🔴 *%s* (Port %d): BOŞ\n", serviceName, port))
		}
	}
	bot.Send(tgbotapi.NewMessage(chatID, builder.String()))
}

// ProcessDetail, görev yöneticisinde gösterilecek bir işlemin bilgilerini tutar.
type ProcessDetail struct {
	PID  int32
	Name string
	CPU  float64
	RAM  uint64
}

// Sayfa başına gösterilecek işlem sayısı.
const processesPerPage = 15

// handleListProcessesCommand, interaktif görev yöneticisini başlatır.
func handleListProcessesCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msg := createProcessListMessage(message.Chat.ID, "cpu", "desc", 1)
	bot.Send(msg)
}

// handleRunApplicationCommand, .env dosyasında tanımlı bir uygulamayı kısayol adıyla başlatır.
func handleRunApplicationCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	appName := strings.ToLower(strings.TrimSpace(message.CommandArguments()))
	chatID := message.Chat.ID

	if appName == "" {
		var availableApps []string
		for name := range config.Uygulamalar {
			availableApps = append(availableApps, name)
		}
		sort.Strings(availableApps)
		reply := fmt.Sprintf("❌ Kullanım: `/uygulama_calistir <kısayol>`\n\n"+
			"Mevcut Kısayollar:\n`%s`", strings.Join(availableApps, "`, `"))
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}

	appPath, found := config.Uygulamalar[appName]
	if !found {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `%s` adında bir uygulama kısayolu bulunamadı!", appName)))
		return
	}

	var cmd *exec.Cmd
	// Windows kısayol (.lnk) dosyaları için özel başlatma komutu kullanılır.
	if strings.HasSuffix(strings.ToLower(appPath), ".lnk") {
		cmd = exec.Command("cmd", "/c", "start", "\"\"", appPath)
	} else {
		cmd = exec.Command(appPath)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Uygulama başlatılamadı: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ `%s` uygulaması başlatılırken bir hata oluştu.", appName)))
		return
	}

	log.Printf("Uygulama başarıyla başlatıldı: %s (Yol: %s)", appName, appPath)
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ `%s` uygulaması başarıyla başlatıldı.", appName)))
}

// handleRunPathCommand, tam dosya yolu belirtilerek bir uygulama başlatır.
func handleRunPathCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	path := message.CommandArguments()
	chatID := message.Chat.ID

	if path == "" {
		reply := "❌ Kullanım: `/calistir_dosya <dosya_yolu>`\n\n" +
			"Örnek: `/calistir_dosya C:\\Program Files\\Notepad++\\notepad++.exe`"
		bot.Send(tgbotapi.NewMessage(chatID, reply))
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Belirtilen yolda bir dosya veya kısayol bulunamadı:\n`%s`", path)))
		return
	}

	var cmd *exec.Cmd
	if strings.HasSuffix(strings.ToLower(path), ".lnk") {
		cmd = exec.Command("cmd", "/c", "start", "\"\"", path)
	} else {
		cmd = exec.Command(path)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Dosya yoluyla uygulama başlatılamadı: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Uygulama başlatılırken bir hata oluştu:\n`%s`", path)))
		return
	}

	log.Printf("Uygulama (doğrudan yoldan) başarıyla başlatıldı: %s", path)
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Uygulama başarıyla başlatıldı:\n`%s`", path)))
}

// createProcessListMessage, görev yöneticisi mesajını ve butonlarını oluşturur.
func createProcessListMessage(chatID int64, sortKey, sortDir string, page int) tgbotapi.MessageConfig {
	// 1. Tüm aktif işlemler
	allProcesses, err := process.Processes()
	if err != nil {
		return tgbotapi.NewMessage(chatID, "❌ İşlem listesi alınırken bir hata oluştu.")
	}

	var processDetails []ProcessDetail
	for _, p := range allProcesses {
		var detail ProcessDetail
		detail.PID = p.Pid
		cpu, _ := p.CPUPercent()
		detail.CPU = cpu
		name, _ := p.Name()
		if name == "" {
			name = "[erişilemedi]"
		}
		detail.Name = name
		mem, err := p.MemoryInfo()
		if err == nil {
			detail.RAM = mem.RSS
		}
		processDetails = append(processDetails, detail)
	}

	// 2. gelen sıralama anahtarı ve yönüne göre işlemleri sıraya koy
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

	// 3. Sayfa için başlangıç ve bitiş indekslerini hesapla.
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

	// 4. Kullanıcıya gösterilecek metin mesajını yarat
	var builder strings.Builder
	builder.WriteString("==| *Aktif Görev Yöneticisi*\n\n")
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

	// 5. Sayfalar arası geçiş ve yeniden sıralama için interaktif butonlar
	nextCpuDir := "desc"
	if sortKey == "cpu" && sortDir == "desc" {
		nextCpuDir = "asc"
	}
	nextRamDir := "desc"
	if sortKey == "ram" && sortDir == "desc" {
		nextRamDir = "asc"
	}
	cpuButtonText := "📊 CPU"
	if sortKey == "cpu" {
		cpuButtonText = fmt.Sprintf("📊 CPU (%s)", sortDirText)
	}
	ramButtonText := "🧠 RAM"
	if sortKey == "ram" {
		ramButtonText = fmt.Sprintf("🧠 RAM (%s)", sortDirText)
	}

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