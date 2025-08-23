// llm_handler.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// #############################################################################
// #                         YAPAY ZEKA İŞLEYİCİSİ (LLM)
// #############################################################################
// Bu dosya, botun Google Gemini API ile olan tüm etkileşimini yönetir.
// Kullanıcıdan gelen doğal dil isteklerini analiz eder, "Function Calling"
// yeteneğini kullanarak bu istekleri botun kendi fonksiyonlarına yönlendirir
// ve modelden gelen cevapları kullanıcıya sunar. Ayrıca, API kota hatalarına
// karşı dayanıklılık sağlayan "model fallback" mekanizmasını içerir.

const telegramMaxMessageLength = 4096

type UserLlmSession struct {
	Session   *genai.ChatSession
	ModelName string
}

var (
	llmActiveUsers   = make(map[int64]bool)
	userChatSessions = make(map[int64]*UserLlmSession)
	llmMutex         = &sync.Mutex{}
	systemPrompt     string
)

var modelFallbackList = []string{
	"gemini-2.5-flash",
	"gemini-2.0-flash",
	"gemini-2.5-flash-lite",
}

var botTools = []*genai.Tool{
	{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_system_status",
				Description: "Kullanıcı 'sistem durumu nasıl?', 'sunucu ne durumda?', 'kaynak kullanımı' gibi genel bir soru sorduğunda kullanılır. Anlık CPU, RAM ve Disk kullanımını özetleyen, formatlanmış bir metin döndürür.",
			},
			{
				Name:        "get_detailed_system_info",
				Description: "Kullanıcı 'detaylı sistem raporu', 'sunucu özellikleri neler?', 'donanım bilgisi ver' gibi daha teknik sorular sorduğunda kullanılır. CPU çekdek sayısı, toplam RAM/disk boyutu gibi ayrıntılı donanım bilgilerini içeren bir metin döndürür.",
			},
			{
				Name:        "run_speed_test",
				Description: "Kullanıcı 'internet hızı nasıl?', 'bağlantıyı test et' gibi ifadeler kullandığında çalıştırılır. Sunucunun internet bağlantı hızını ölçer ve sonucu formatlı bir rapor olarak döndürür.",
			},
			{
				Name:        "get_screenshot",
				Description: "Kullanıcı 'ekran görüntüsü al', 'masaüstünü göster' dediğinde kullanılır. Bu fonksiyon, ekran görüntüsü alma işlemini tetikler ve işlemin başlatıldığına dair bir onay metni döndürür. Asıl dosya, kullanıcıya ayrı bir işlem olarak gönderilir.",
			},
			{
				Name:        "list_files",
				Description: "Kullanıcı 'hangi dosyalar var?', 'gelenler klasörünü listele' gibi bir istekte bulunduğunda kullanılır. Ana 'Gelenler' klasöründeki dosyaların bir listesini metin olarak döndürür.",
			},
			{
				Name:        "list_files_in_category",
				Description: "Belirtilen bir kategori içindeki dosyaları listeler. Örneğin kullanıcı 'resimler klasöründe ne var?' diye sorduğunda, 'category' parametresi 'Resimler' olarak bu fonksiyon çağrılır.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"category": {
							Type:        genai.TypeString,
							Description: "İçeriği listelenecek kategori adı. Geçerli değerler: 'Resimler', 'Dokümanlar', 'Videolar', 'Sesler', 'Arşivler', 'Diğer'.",
						},
					},
					Required: []string{"category"},
				},
			},
			{
				Name:        "search_files",
				Description: "Kullanıcı 'içinde ... geçen dosyaları bul' veya '... dosyasını ara' dediğinde kullanılır. Verilen anahtar kelimeyi tüm dosya adlarında arar ve bulunan dosyaların yollarını içeren bir liste döndürür.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"keyword": {
							Type:        genai.TypeString,
							Description: "Dosya adlarında aranacak kelime veya kelime parçası.",
						},
					},
					Required: []string{"keyword"},
				},
			},
			{
				Name:        "send_file",
				Description: "Kullanıcı bir dosyanın kendisine gönderilmesini istediğinde kullanılır (Örn: 'rapor.pdf dosyasını gönder'). Bu araç, belirtilen dosyayı bulur ve doğrudan kullanıcıya gönderir. Bu bir 'ateşle ve unut' (fire-and-forget) komutudur; asıl dosya gönderim işlemi arka planda gerçekleşir. Bu fonksiyonun döndürdüğü metin, sadece işlemin başarıyla tetiklendiğini belirten bir onaydır.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"filename": {
							Type:        genai.TypeString,
							Description: "Kullanıcıya gönderilecek olan dosyanın tam adı ve uzantısı. Yapay zeka, kullanıcının isteğinden dosya adını doğru bir şekilde çıkarmalıdır.",
						},
					},
					Required: []string{"filename"},
				},
			},
			{
				Name:        "delete_file",
				Description: "DİKKAT: Sunucudan bir dosyayı KALICI olarak siler. Bu işlem geri alınamaz. Kullanıcı '... dosyasını sil' veya '... dosyasını kaldır' gibi çok net bir komut verdiğinde kullanılır. Başarılı veya hatalı olduğuna dair bir sonuç metni döndürür.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"filename": {
							Type:        genai.TypeString,
							Description: "Kalıcı olarak silinecek dosyanın uzantısı dahil tam adı.",
						},
					},
					Required: []string{"filename"},
				},
			},
			{
				Name:        "organize_files",
				Description: "Kullanıcı 'dosyaları düzenle', 'gelenleri temizle' veya 'klasörleri organize et' gibi bir istekte bulunduğunda çalıştırılır. 'Gelenler' klasöründeki dosyaları kategorilerine taşır ve taşınan dosya sayısını raporlar.",
			},
			{
				Name:        "download_video_or_audio",
				Description: "Kullanıcı bir URL verip 'bunu indir' veya 'bunun sesini indir' dediğinde kullanılır. Verilen URL'den medya indirme işlemini başlatır ve işlemin başladığına dair bir onay metni döndürür.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"url": {
							Type:        genai.TypeString,
							Description: "İndirilecek medyanın tam URL'si.",
						},
						"download_audio_only": {
							Type:        genai.TypeBoolean,
							Description: "Eğer kullanıcı açıkça 'sadece sesini', 'mp3 olarak' gibi bir ifade kullanırsa 'true' olmalıdır. Aksi halde 'false' olur.",
						},
					},
					Required: []string{"url"},
				},
			},
			{
				Name:        "start_application",
				Description: "Kullanıcı '... aç', '... başlat' gibi bir komut verdiğinde, önceden tanımlanmış bir uygulama kısayolunu kullanarak programı başlatır.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"shortcut_name": {
							Type:        genai.TypeString,
							Description: "Başlatılacak uygulamanın .env dosyasında tanımlanmış olan kısayol adı. Örneğin: 'chrome', 'oyun', 'hesapmakinesi'.",
						},
					},
					Required: []string{"shortcut_name"},
				},
			},
		},
	},
}

// #############################################################################
// #                            LLM Komut İşleyicileri
// #############################################################################

func handleLlmOnCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if config.GeminiAPIKey == "" {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ LLM özelliği yönetici tarafından yapılandırılmamış. (API Anahtarı eksik)"))
		return
	}
	userID := message.From.ID
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(config.GeminiAPIKey))
	if err != nil {
		log.Printf("Gemini istemcisi oluşturulamadı: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Yapay zeka servisine bağlanırken bir hata oluştu."))
		return
	}

	modelName := modelFallbackList[0]
	log.Printf("Yeni sohbet oturumu '%s' modeli ile başlatılıyor.", modelName)
	model := client.GenerativeModel(modelName)
	model.Tools = botTools
	if systemPrompt != "" {
		model.SystemInstruction = &genai.Content{
			Parts: []genai.Part{genai.Text(systemPrompt)},
		}
	}
	chatSession := model.StartChat()

	llmMutex.Lock()
	llmActiveUsers[userID] = true
	userChatSessions[userID] = &UserLlmSession{
		Session:   chatSession,
		ModelName: modelName,
	}
	llmMutex.Unlock()

	log.Printf("Kullanıcı %d için yeni LLM sohbet oturumu (Araçlarla) başlatıldı.", userID)
	msgText := "🤖 *Akıllı LLM Modu Aktif.*\n\nArtık komutları doğal dilde ifade edebilirsin. (Örn: \"Sistem durumu nasıl?\" veya \"'rapor' kelimesini içeren dosyaları ara\") Modu kapatmak için `/llm_kapat` yazman yeterli."
	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown

	if _, err := bot.Send(msg); err != nil {
		log.Printf("[HATA] LLM On mesajı gönderilemedi: %v", err)
	}
}

func handleLlmQuery(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userID := message.From.ID
	userQuery := message.Text

	// Her sorguyu kendi goroutine'i içinde işle, böylece loglar karışmaz.
	go func() {
		log.Printf("\n\n--- YENI AKILLI SORGU BAŞLADI ---\nKullanıcı: %d, Sorgu: %s", userID, userQuery)

		statusMsg, _ := bot.Send(tgbotapi.NewMessage(chatID, "🧠 Düşünüyorum..."))
		messageID := statusMsg.MessageID

		llmMutex.Lock()
		currentUserSession := userChatSessions[userID]
		llmMutex.Unlock()

		if currentUserSession == nil || currentUserSession.Session == nil {
			bot.Request(tgbotapi.NewEditMessageText(chatID, messageID, "❌ Aktif bir sohbet oturumu bulunamadı. Lütfen `/llm` komutuyla yeniden başlatın."))
			return
		}

		ctx := context.Background()
		
		var finalResponse *genai.GenerateContentResponse
		var finalErr error

		// Kullanıcının bu spesifik sorgusu için ilk prompt.
		// Bu değişken, döngüler boyunca değiştirilmeyecek.
		initialPrompt := []genai.Part{genai.Text(userQuery)}

		for _, modelName := range modelFallbackList {
			log.Printf("[DEBUG] Model denemesi başlıyor: %s", modelName)
			bot.Request(tgbotapi.NewEditMessageText(chatID, messageID, fmt.Sprintf("🧠 Düşünüyorum... (Model: %s)", modelName)))

			// Gerekirse sohbet oturumunu ve modelini değiştir.
			if currentUserSession.ModelName != modelName {
				log.Printf("[INFO] Model değiştiriliyor: %s -> %s", currentUserSession.ModelName, modelName)
				
				client, err := genai.NewClient(ctx, option.WithAPIKey(config.GeminiAPIKey))
				if err != nil { finalErr = err; break }
				
				newModel := client.GenerativeModel(modelName)
				newModel.Tools = botTools
				if systemPrompt != "" {
					newModel.SystemInstruction = &genai.Content{Parts: []genai.Part{genai.Text(systemPrompt)}}
				}

				newSession := newModel.StartChat()
				newSession.History = currentUserSession.Session.History
				
				currentUserSession = &UserLlmSession{
					Session:   newSession,
					ModelName: modelName,
				}
			}
			
			var resp *genai.GenerateContentResponse
			var attemptErr error
			
			// Bu model için Function Calling döngüsünü başlat.
			// Her zaman en baştaki, orijinal prompt ile başla.
			promptPartsForThisAttempt := initialPrompt
			const maxTurns = 5
			for i := 0; i < maxTurns; i++ {
				log.Printf("[DEBUG] -> Model '%s' ile API çağrısı yapılıyor (Tur %d)", modelName, i+1)
				resp, attemptErr = currentUserSession.Session.SendMessage(ctx, promptPartsForThisAttempt...)
				if attemptErr != nil {
					break 
				}

				if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
					attemptErr = fmt.Errorf("modelden boş veya geçersiz içerik alındı")
					break
				}
				
				candidate := resp.Candidates[0]
				hasFunctionCall := false
				for _, part := range candidate.Content.Parts {
					if _, ok := part.(genai.FunctionCall); ok {
						hasFunctionCall = true
						break
					}
				}

				if hasFunctionCall {
					log.Printf("[DEBUG] <- Model '%s' bir FunctionCall isteğiyle yanıt verdi.", modelName)
					var nextPromptParts []genai.Part
					for _, part := range candidate.Content.Parts {
						if call, ok := part.(genai.FunctionCall); ok {
							responsePart := executeTool(bot, message, &call)
							nextPromptParts = append(nextPromptParts, responsePart)
						}
					}
					promptPartsForThisAttempt = nextPromptParts
					continue
				} else {
					log.Printf("[DEBUG] <- Model '%s' nihai bir metin cevabıyla yanıt verdi.", modelName)
					break 
				}
			}

			if attemptErr != nil {
				errorText := attemptErr.Error()
				if strings.Contains(errorText, "429") || strings.Contains(strings.ToLower(errorText), "quota") {
					log.Printf("[UYARI] KOTA HATASI ALGILANDI: Model '%s' başarısız oldu. Bir sonraki modele geçiliyor.", modelName)
					finalErr = attemptErr
					continue
				}
				
				log.Printf("[HATA] KRİTİK HATA: Model '%s' başarısız oldu. Hata: %v", modelName, attemptErr)
				finalErr = attemptErr
				break
			}

			finalResponse = resp
			finalErr = nil
			log.Printf("[BAŞARI] Başarılı yanıt '%s' modelinden alındı.", modelName)
			break
		}

		if finalErr != nil {
			log.Printf("[HATA] Tüm modeller denendi ve hepsi başarısız oldu. Son hata: %v", finalErr)
			bot.Request(tgbotapi.NewEditMessageText(chatID, messageID, "❌ Servis şu anda çok yoğun veya bir hata oluştu. Lütfen daha sonra tekrar deneyin."))
			return
		}

		llmMutex.Lock()
		userChatSessions[userID] = currentUserSession
		llmMutex.Unlock()

		candidate := finalResponse.Candidates[0]
		if txt, ok := candidate.Content.Parts[0].(genai.Text); ok {
			sendFinalLlmResponse(bot, chatID, messageID, string(txt))
		} else {
			log.Printf("[HATA] Son yanıt metin değil, beklenmedik bir durum. Part: %T", candidate.Content.Parts[0])
			bot.Request(tgbotapi.NewEditMessageText(chatID, messageID, "✅ İşlem tamamlandı (ancak bir özet metni üretilemedi)."))
		}
	}() // Goroutine'i burada kapat.
}

func handleLlmOffCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	userID := message.From.ID
	llmMutex.Lock()
	delete(llmActiveUsers, userID)
	delete(userChatSessions, userID)
	llmMutex.Unlock()
	log.Printf("Kullanıcı %d LLM modundan çıktı ve sohbet geçmişi temizlendi.", userID)
	msgText := "👍 *LLM Modu Kapatıldı.*\n\nSohbet geçmişin temizlendi. Normal komut moduna geri dönüldü."
	msg := tgbotapi.NewMessage(message.Chat.ID, msgText)
	msg.ParseMode = tgbotapi.ModeMarkdown
	bot.Send(msg)
}

// #############################################################################
// #                         Yardımcı Fonksiyonlar
// #############################################################################

func executeTool(bot *tgbotapi.BotAPI, message *tgbotapi.Message, call *genai.FunctionCall) genai.Part {
	log.Printf("[DEBUG] executeTool çağrıldı. İstenen fonksiyon: %s, Parametreler: %v", call.Name, call.Args)
	
	var toolResult string
	var toolErr error

	switch call.Name {
	case "get_system_status":
		toolResult = getSystemInfoText(false)
	case "get_detailed_system_info":
		toolResult = getSystemInfoText(true)
	case "run_speed_test":
		bot.Send(tgbotapi.NewChatAction(message.Chat.ID, "typing"))
		toolResult, toolErr = runSpeedTestInternal()
	case "get_screenshot":
		go handleScreenshotCommand(bot, message)
		toolResult = "Ekran görüntüsü alma komutu başarıyla tetiklendi ve arka planda çalışıyor. Sonuç doğrudan kullanıcıya gönderilecek."
	case "list_files":
		toolResult, toolErr = getFileListText()
	case "list_files_in_category":
		if category, ok := call.Args["category"].(string); ok {
			toolResult, toolErr = listFilesInCategoryInternal(category)
		} else {
			toolErr = fmt.Errorf("kategori parametresi eksik")
		}
	case "search_files":
		if keyword, ok := call.Args["keyword"].(string); ok {
			toolResult = searchFilesText(keyword)
		} else {
			toolErr = fmt.Errorf("keyword parametresi eksik")
		}
	case "send_file":
		if filename, ok := call.Args["filename"].(string); ok {
			go func() {
				bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("🔍 `%s` dosyası aranıyor...", filename)))
				
				filePath, found := findFile(filename)
				if !found {
					bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Sentinel, `%s` dosyasını bulamadı.", filename)))
					return
				}

				fileInfo, err := os.Stat(filePath)
				if err != nil || fileInfo.Size() > 50*1024*1024 {
					bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ `%s` dosyası gönderilemiyor (çok büyük veya okunamıyor).", filename)))
					return
				}

				doc := tgbotapi.NewDocument(message.Chat.ID, tgbotapi.FilePath(filePath))
				doc.Caption = "Sentinel tarafından gönderildi."
				if _, err := bot.Send(doc); err != nil {
					log.Printf("[HATA] LLM aracılığıyla dosya gönderilemedi: %v", err)
					bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ `%s` dosyası gönderilirken bir hata oluştu.", filename)))
				}
			}()
			toolResult = fmt.Sprintf("`%s` adlı dosyayı gönderme işlemi başlatıldı. Dosya birazdan gelecek.", filename)
		} else {
			toolErr = fmt.Errorf("filename parametresi eksik")
		}
	case "delete_file":
		if filename, ok := call.Args["filename"].(string); ok {
			toolResult, toolErr = deleteFileInternal(filename)
		} else {
			toolErr = fmt.Errorf("filename parametresi eksik")
		}
	case "organize_files":
		toolResult = organizeFilesInternal()
	case "download_video_or_audio":
		if url, ok := call.Args["url"].(string); ok {
			audioOnly := false
			if ao, ok := call.Args["download_audio_only"].(bool); ok {
				audioOnly = ao
			}

			go func() {
				fakeMessage := *message
				if audioOnly {
					fakeMessage.Text = "/indir_ses " + url
					handleAudioDownloadCommand(bot, &fakeMessage)
				} else {
					fakeMessage.Text = "/indir " + url
					handleDownloadCommand(bot, &fakeMessage)
				}
			}()
			toolResult = fmt.Sprintf("`%s` adresinden indirme işlemi arka planda başlatıldı.", url)
		} else {
			toolErr = fmt.Errorf("url parametresi eksik")
		}
	case "start_application":
		if shortcut, ok := call.Args["shortcut_name"].(string); ok {
			toolResult, toolErr = startApplicationInternal(shortcut)
		} else {
			toolErr = fmt.Errorf("shortcut_name parametresi eksik")
		}

	default:
		toolErr = fmt.Errorf("'%s' adında bir araç bulunamadı", call.Name)
	}

	var finalResult map[string]any
	if toolErr != nil {
		log.Printf("[HATA] Araç çalıştırılırken hata oluştu: %v", toolErr)
		finalResult = map[string]any{"status": "error", "message": toolErr.Error()}
	} else {
		finalResult = map[string]any{"status": "success", "result": toolResult}
	}
	
	responseJSON, _ := json.Marshal(finalResult)
	log.Printf("[DEBUG] executeTool sonucu Gemini'ye gönderilmek üzere hazırlanıyor: %s", string(responseJSON))
	
	return &genai.FunctionResponse{
		Name:     call.Name,
		Response: map[string]any{"response": string(responseJSON)},
	}
}
func sendFinalLlmResponse(bot *tgbotapi.BotAPI, chatID int64, messageID int, responseText string) {
	log.Printf("[DEBUG] Orijinal LLM yanıtı: %s", responseText)
	if responseText == "" {
		responseText = "Anlaşılır bir yanıt üretemedim."
	}
	replacer := strings.NewReplacer(
		"**", "*",
	)
	cleanResponse := replacer.Replace(responseText)
	lines := strings.Split(cleanResponse, "\n")
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "* *") {
			lines[i] = strings.Replace(line, "* *", "* ", 1)
		}
	}
	finalResponse := strings.Join(lines, "\n")
	log.Printf("[DEBUG] Düzeltilmiş final yanıt: %s", finalResponse)

	if len(finalResponse) > telegramMaxMessageLength {
		bot.Request(tgbotapi.NewEditMessageText(chatID, messageID, "✅ Yanıtınız hazırlandı. Şimdi parçalar halinde gönderiliyor..."))
		chunks := splitMessageSmart(finalResponse)
		for _, chunk := range chunks {
			if strings.TrimSpace(chunk) == "" {
				continue
			}
			msg := tgbotapi.NewMessage(chatID, chunk)
			msg.ParseMode = tgbotapi.ModeMarkdown
			if _, errChunk := bot.Send(msg); errChunk != nil {
				msg.ParseMode = ""
				bot.Send(msg)
			}
			time.Sleep(500 * time.Millisecond)
		}
		return
	}
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, finalResponse)
	editMsg.ParseMode = tgbotapi.ModeMarkdown
	if _, err := bot.Request(editMsg); err != nil {
		log.Printf("[HATA] LLM yanıtı gönderilirken Markdown hatası oluştu: %v. Fallback deneniyor.", err)
		editMsg.ParseMode = ""
		bot.Request(editMsg)
	}
}
func isUserInLlmMode(userID int64) bool {
	llmMutex.Lock()
	defer llmMutex.Unlock()
	return llmActiveUsers[userID]
}
func loadSystemPrompt() error {
	data, err := os.ReadFile("system_prompt.txt")
	if err != nil {
		log.Printf("⚠️ Sistem prompt dosyası (system_prompt.txt) okunamadı. Hata: %v", err)
		return err
	}
	systemPrompt = string(data)
	log.Println("Sistem prompt'u başarıyla yüklendi.")
	return nil
}
func splitMessageSmart(text string) []string {
	var chunks []string
	parts := strings.Split(text, "```")
	for i, part := range parts {
		isCodeBlock := (i % 2 != 0)
		if isCodeBlock {
			codeContent := "```" + part + "```"
			if len(codeContent) > telegramMaxMessageLength {
				start := 0
				for start < len(codeContent) {
					end := start + (telegramMaxMessageLength - 6)
					if end > len(part) { end = len(part) }
					chunks = append(chunks, "```"+part[start:end]+"```")
					start = end
				}
			} else {
				chunks = append(chunks, codeContent)
			}
		} else {
			var currentChunk strings.Builder
			lines := strings.Split(part, "\n")
			for _, line := range lines {
				if currentChunk.Len()+len(line)+1 > telegramMaxMessageLength {
					chunks = append(chunks, currentChunk.String())
					currentChunk.Reset()
				}
				currentChunk.WriteString(line + "\n")
			}
			if currentChunk.Len() > 0 { chunks = append(chunks, currentChunk.String()) }
		}
	}
	return chunks
}
func runSpeedTestInternal() (string, error) {
	speedTestResult, err := runSpeedTest()
	if err != nil {
		return "", fmt.Errorf("hız testi başarısız oldu: %v", err)
	}
	downloadMbps := float64(speedTestResult.Download.Bandwidth*8) / 1e6
	uploadMbps := float64(speedTestResult.Upload.Bandwidth*8) / 1e6
	ping := speedTestResult.Ping.Latency
	quality := getConnectionQuality(downloadMbps, ping)
	return fmt.Sprintf(
		"İnternet Hız Raporu:\nDeğerlendirme: *%s*\nİndirme: *%.2f Mbps*\nYükleme: *%.2f Mbps*\nGecikme (ping): *%.2f ms*",
		quality, downloadMbps, uploadMbps, ping,
	), nil
}
func deleteFileInternal(filename string) (string, error) {
	filePath, found := findFile(filename)
	if !found {
		return "", fmt.Errorf("silinecek dosya bulunamadı: `%s`", filename)
	}
	if err := os.Remove(filePath); err != nil {
		return "", fmt.Errorf("`%s` dosyası silinirken bir hata oluştu: %v", filename, err)
	}
	removeDescription(filename)
	return fmt.Sprintf("`%s` dosyası başarıyla silindi.", filename), nil
}
func organizeFilesInternal() string {
	count := organizeFiles()
	if count == 0 {
		return "Taşınacak yeni dosya bulunamadığı için herhangi bir işlem yapılmadı."
	}
	return fmt.Sprintf("%d adet dosya başarıyla kategorilere ayrıldı.", count)
}
func listFilesInCategoryInternal(category string) (string, error) {
	categoryDir := filepath.Join(config.BaseDir, category)
	files, err := os.ReadDir(categoryDir)
	if err != nil {
		var cats []string
		for k := range kategoriler {
			cats = append(cats, k)
		}
		sort.Strings(cats)
		return "", fmt.Errorf("`%s` kategorisi bulunamadı. Geçerli kategoriler: `%s`", category, strings.Join(cats, "`, `"))
	}
	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}
	if len(fileNames) == 0 {
		return fmt.Sprintf("`%s` klasörü boş.", category), nil
	}
	return fmt.Sprintf("`%s` kategorisinde %d dosya bulundu:\n- %s", category, len(fileNames), strings.Join(fileNames, "\n- ")), nil
}
func getFileListText() (string, error) {
	files, err := os.ReadDir(config.BaseDir)
	if err != nil { return "", fmt.Errorf("ana klasördeki dosyalar okunurken bir hata oluştu") }
	var fileNames []string
	for _, file := range files { if !file.IsDir() { fileNames = append(fileNames, file.Name()) } }
	if len(fileNames) == 0 { return "Ana klasör boş.", nil }
	return fmt.Sprintf("Ana klasörde %d dosya bulundu:\n- %s", len(fileNames), strings.Join(fileNames, "\n- ")), nil
}
func searchFilesText(keyword string) string {
	var foundFiles []string
	filepath.Walk(config.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if !info.IsDir() && strings.Contains(strings.ToLower(info.Name()), strings.ToLower(keyword)) {
			relPath, _ := filepath.Rel(config.BaseDir, path)
			foundFiles = append(foundFiles, relPath)
		}
		return nil
	})
	if len(foundFiles) == 0 { return fmt.Sprintf("Adında '%s' geçen dosya bulunamadı.", keyword) }
	return fmt.Sprintf("'%s' araması için %d sonuç bulundu:\n- %s", keyword, len(foundFiles), strings.Join(foundFiles, "\n- "))
}
func startApplicationInternal(shortcutName string) (string, error) {
	appName := strings.ToLower(shortcutName)

	appPath, found := config.Uygulamalar[appName]
	if !found {
		var availableApps []string
		for name := range config.Uygulamalar {
			availableApps = append(availableApps, name)
		}
		sort.Strings(availableApps)
		return "", fmt.Errorf("`%s` adında bir uygulama kısayolu bulunamadı. Mevcut kısayollar: `%s`", appName, strings.Join(availableApps, "`, `"))
	}

	var cmd *exec.Cmd

	if strings.HasSuffix(strings.ToLower(appPath), ".lnk") {
		cmd = exec.Command("cmd", "/c", "start", "\"\"", appPath)
	} else {
		cmd = exec.Command(appPath)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Uygulama başlatılamadı: %v", err)
		return "", fmt.Errorf("`%s` uygulaması başlatılırken bir hata oluştu: %v", appName, err)
	}

	log.Printf("Uygulama başarıyla başlatıldı: %s (Yol: %s)", appName, appPath)
	return fmt.Sprintf("`%s` uygulaması başarıyla başlatıldı.", appName), nil
}