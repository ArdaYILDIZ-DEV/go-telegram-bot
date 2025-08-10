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
)

// #############################################################################
// #
// #                            KOMUT İŞLEYİCİLERİ
// #
// # Bu dosya, botun aldığı komutların ana iş mantığını içerir. Her bir "handle"
// # fonksiyonu, belirli bir komutun veya bir grup benzer komutun nasıl
// # çalışacağını tanımlar. `telegram_bot.go` dosyasındaki ana yönlendirici (router),
// # gelen komutları buradaki uygun fonksiyonlara dağıtır.
// #
// # Temel Sorumluluklar:
// #   - Kullanıcıdan gelen komut argümanlarını ayrıştırma.
// #   - İlgili arka plan fonksiyonlarını (dosya arama, sistem bilgisi alma vb.) çağırma.
// #   - Sonuçları kullanıcıya anlaşılır bir formatta sunma.
// #
// #############################################################################

// handleGeneralCommands, genellikle basit, metin tabanlı yanıtlar üreten ve
// karmaşık mantık gerektirmeyen temel komutları bir araya toplayan bir fonksiyondur.
func handleGeneralCommands(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")
	msg.ParseMode = "Markdown"
	command := message.Command()

	switch command {
	// * /start: Bota ilk kez başlandığında veya merhaba demek için kullanılır.
	case "start":
		msg.Text = "*Hoş geldin*\n\n" +
			"Bu sistem; dosya erişimi, medya yönetimi ve sistem denetimi gibi işlemleri Telegram üzerinden kontrol edebilmen için optimize edildi.\n\n" +
			"Tüm komutları listelemek için `/help` komutunu kullanabilirsin."

	// * /help: Botun tüm yeteneklerini listeleyen yardım menüsü.
	case "help":
		msg.Text = "*Komut Seti – Fonksiyonlara Göre Gruplandırılmıştır:*\n\n" +
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

	// * /duzenle: `Gelenler` klasöründeki dosyaları kategorilere ayırır.
	case "duzenle":
		count := organizeFiles()
		msg.Text = fmt.Sprintf("🗂️ Dosyalar kategorilere göre yeniden düzenlendi.\nTaşınan dosya sayısı: *%d*", count)

	// * /sistem_bilgisi: Detaylı sistem raporu sunar.
	case "sistem_bilgisi":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "📊 Sistem bilgileri getiriliyor..."))
		msg.Text = getSystemInfoText(true)

	// * /durum: Anlık ve özet sistem raporu sunar.
	case "durum":
		msg.Text = getSystemInfoText(false)

	// * /hiz_testi: İnternet bağlantı hızını ölçer.
	case "hiz_testi":
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "⏳ Bağlantı testi başlatılıyor..."))
		speedTestResult, err := runSpeedTest()
		if err != nil {
			msg.Text = fmt.Sprintf("❌ Test başarısız:\n`%v`", err)
		} else {
			// # ÖNEMLİ: Hız testi sonuçları genellikle bit/saniye cinsinden gelir.
			// # Mbps'ye (Megabit per second) çevirmek için 8 ile çarpıp (byte -> bit)
			// # 1 milyona (1e6) bölmek gerekir.
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

// handleToggleInternetMonitorCommand, /izle komutunu işleyerek internet kesinti izleyicisini açıp kapatır.
func handleToggleInternetMonitorCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	// # ÖNEMLİ: `monitorMutex`, `internetMonitorEnabled` global değişkenine
	// # aynı anda birden fazla yerden (örneğin iki farklı komutla) erişilmesini
	// # engelleyerek "race condition" oluşmasını önler.
	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	internetMonitorEnabled = !internetMonitorEnabled // Durumu tersine çevir (toggle)

	var statusText string
	if internetMonitorEnabled {
		statusText = "🟢 *Aktif*"
		log.Println("İnternet izleyici kullanıcı tarafından AKTİF edildi.")
		// * Kullanıcı deneyimini iyileştirmek için, monitör açılır açılmaz
		// * bir kontrol tetiklenir, böylece sonuç için bir sonraki periyodu beklemek gerekmez.
		go checkInternetConnection(bot)
	} else {
		statusText = "🔴 *Pasif*"
		log.Println("İnternet izleyici kullanıcı tarafından PASİF edildi.")
		internetDown = false // Monitör kapanırsa, kesinti durumunu sıfırla.
	}

	msgText := fmt.Sprintf("📡 *İnternet Kesinti Monitörü* durumu güncellendi:\n\nDurum: %s", statusText)
	bot.Send(tgbotapi.NewMessage(chatID, msgText))
}

// handleMoveFileCommand, bir dosyayı bir yerden başka bir yere taşır.
func handleMoveFileCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	chatID := message.Chat.ID
	// * `SplitN` fonksiyonu, string'i belirtilen ayırıcıya göre en fazla N parçaya böler.
	// * Burada 2 kullanmak, hedef klasör adında boşluk olsa bile onu tek bir parça olarak almamızı sağlar.
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
	// # GÜVENLİK: `filepath.Clean` ve `strings.HasPrefix` kontrolleri, kullanıcının
	// # `../` gibi ifadeler kullanarak ana çalışma dizininin dışına dosya taşımasını engeller.
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

// handleGetFileCommand, sunucudaki bir dosyayı Telegram üzerinden kullanıcıya gönderir.
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
	if fileInfo.Size() > 50*1024*1024 {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Dosya çok büyük! (%.1f MB). Limit 50 MB.", float64(fileInfo.Size())/1024/1024)))
		return
	}
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	var captionBuilder strings.Builder
	captionBuilder.WriteString(fmt.Sprintf("📄 *%s*", filepath.Base(filePath)))
	if description, ok := getDescription(args); ok {
		captionBuilder.WriteString(fmt.Sprintf("\n\n📝 *Açıklama:*\n%s", description))
	}
	doc.Caption = captionBuilder.String()
	doc.ParseMode = "Markdown"
	if _, err := bot.Send(doc); err != nil {
		log.Printf("Dosya gönderilemedi: %v", err)
	}
}

// handleAddDescriptionCommand, bir dosyaya açıklama ekler.
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

// handleListDescriptionsCommand, tüm dosya açıklamalarını listeler.
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
	builder.WriteString(fmt.Sprintf("```\n%s\n```", strings.Join(fileNames, "\n")))
	builder.WriteString("\n💡 *Dosya almak için:* `/getir <dosya_adı>`")
	msg := tgbotapi.NewMessage(chatID, builder.String())
	msg.ParseMode = "MarkdownV2"
	bot.Send(msg)
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
	// # ÖNEMLİ: `filepath.Walk` Go'nun dosya sisteminde gezinmek için güçlü bir aracıdır.
	// # Belirtilen bir kök dizinden başlayarak tüm alt dizinleri ve dosyaları
	// # tek tek ziyaret eder ve her biri için belirttiğiniz fonksiyonu çalıştırır.
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

// handleDeleteFileCommand, bir dosyayı silmek için kullanıcıdan onay ister.
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
	// # KRİTİK: Geri alınamaz bir işlem olan silme öncesinde kullanıcıya onay
	// # sordurmak, kullanıcı deneyimi açısından çok önemlidir.
	// # Bu, inline butonlar ve callback query'ler ile sağlanır.
	text := fmt.Sprintf("⚠️ *Emin misiniz?*\n\n`%s` dosyası kalıcı olarak silinecek. Bu işlem geri alınamaz.", filename)
	// * Callback verileri, hangi butona basıldığını ve hangi dosya için basıldığını
	// * ayırt etmemizi sağlayan özel bir formattır: `eylem_sonuç_veri`.
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
	// * Bir dosyanın açıklaması varsa, yeniden adlandırıldıktan sonra
	// * bu açıklamanın kaybolmaması için eski kaydı silip yenisini oluştururuz.
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
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🔍 Portlar kontrol ediliyor: %v", config.MonitoredPorts)))
	activePorts, err := checkListeningPorts(config.MonitoredPorts)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Hata: %v\n💡 Yönetici izni gerekebilir.", err)))
		return
	}
	var builder strings.Builder
	builder.WriteString("📊 *Port Durum Raporu:*\n\n")
	// # ÖNEMLİ: Portları sayısal olarak sıralamak, her seferinde aynı
	// # sırada ve daha okunaklı bir rapor sunulmasını sağlar.
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