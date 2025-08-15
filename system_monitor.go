// system_monitor.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// #############################################################################
// #                             SİSTEM İZLEME
// #############################################################################
// Bu dosya, sunucunun donanım ve ağ kaynaklarını izleyen fonksiyonları içerir.
// `gopsutil` kütüphanesi ile CPU, RAM, Disk gibi temel sistem bilgilerini
// toplar ve `speedtest.exe` gibi harici araçları çalıştırarak ağ performansını
// ölçer. Buradaki fonksiyonlar, `/durum`, `/sistem_bilgisi`, `/hiz_testi`
// gibi komutlara veri sağlar.

// ProcessInfo, bir portu dinleyen işlemin temel bilgilerini tutar.
type ProcessInfo struct {
	PID  int32
	Name string
}

// SpeedTestResult, `speedtest.exe --format json` komutunun çıktısını
// Go içinde kolayca işleyebilmek için tasarlanmış bir yapıdır (struct).
// `json:"..."` etiketleri, JSON alan adlarıyla struct alan adlarını eşleştirir.
type SpeedTestResult struct {
	Ping struct {
		Latency float64 `json:"latency"`
	} `json:"ping"`
	Download struct {
		Bandwidth int `json:"bandwidth"`
	} `json:"download"`
	Upload struct {
		Bandwidth int `json:"bandwidth"`
	} `json:"upload"`
	Server struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	} `json:"server"`
}

// checkListeningPorts, `gopsutil/net` kütüphanesini kullanarak sistemdeki
// tüm ağ bağlantılarını tarar ve yapılandırmada belirtilen portların
// "LISTEN" (dinleme) durumunda olup olmadığını kontrol eder.
func checkListeningPorts(portsToCheck []int) (map[int]ProcessInfo, error) {
	listeningPorts := make(map[int]ProcessInfo)
	portsSet := make(map[int]struct{})
	for _, p := range portsToCheck {
		portsSet[p] = struct{}{}
	}

	conns, err := net.Connections("inet")
	if err != nil {
		return nil, fmt.Errorf("bağlantılar alınamadı: %w", err)
	}

	for _, conn := range conns {
		if conn.Status == "LISTEN" {
			// Eğer bağlantı dinleme durumundaysa ve port izleme listemizdeyse,
			// sonucu haritaya ekle.
			if _, ok := portsSet[int(conn.Laddr.Port)]; ok {
				listeningPorts[int(conn.Laddr.Port)] = ProcessInfo{PID: conn.Pid, Name: "Bilinmiyor"}
			}
		}
	}
	return listeningPorts, nil
}

// getSystemInfoText, `/durum` ve `/sistem_bilgisi` komutları için formatlanmış
// metin raporları oluşturur. `detailed` parametresi, raporun özet mi yoksa
// detaylı mı olacağını belirler.
func getSystemInfoText(detailed bool) string {
	var builder strings.Builder
	// * `gopsutil` fonksiyonları, sistem kaynaklarına anlık erişim sağlar.
	// * `cpu.Percent` gibi bazı fonksiyonlar, doğru bir yüzde değeri
	// * hesaplayabilmek için kısa bir süre (örn: 1 saniye) bekler.
	cpuPercent, _ := cpu.Percent(time.Second, false)
	cpuCount, _ := cpu.Counts(true)
	vmStat, _ := mem.VirtualMemory()
	diskStat, _ := disk.Usage("/")
	fileCount, descriptionCount := getFileStats()

	if detailed {
		// Detaylı rapor formatı
		builder.WriteString(fmt.Sprintf("🖥️ *Detaylı Sistem Raporu*\n\n"))
		builder.WriteString(fmt.Sprintf("*💻 CPU Performansı:*\n- Çekirdek Sayısı: %d\n- Kullanım: %.1f%%\n\n", cpuCount, cpuPercent[0]))
		builder.WriteString(fmt.Sprintf("*🧠 Bellek (RAM):*\n- Toplam: %.2f GB\n- Kullanılan: %.2f GB (%.1f%%)\n\n", float64(vmStat.Total)/1e9, float64(vmStat.Used)/1e9, vmStat.UsedPercent))
		builder.WriteString(fmt.Sprintf("*💾 Disk Durumu:*\n- Toplam: %.2f GB\n- Kullanılan: %.2f GB (%.1f%%)\n\n", float64(diskStat.Total)/1e9, float64(diskStat.Used)/1e9, diskStat.UsedPercent))
		builder.WriteString(fmt.Sprintf("*📁 Bot Dosya Durumu:*\n- Toplam dosya: %d\n- Açıklama sayısı: %d", fileCount, descriptionCount))
	} else {
		// Özet rapor formatı
		builder.WriteString("📊 *Anlık Sistem ve Dosya Durumu:*\n\n")
		builder.WriteString(fmt.Sprintf("💻 CPU: %.1f%%\n", cpuPercent[0]))
		builder.WriteString(fmt.Sprintf("🧠 RAM: %.1f%%\n", vmStat.UsedPercent))
		builder.WriteString(fmt.Sprintf("💾 Disk: %.1f%%\n\n", diskStat.UsedPercent))
		builder.WriteString(fmt.Sprintf("📁 Toplam Dosya: %d\n", fileCount))
		builder.WriteString(fmt.Sprintf("📝 Açıklama Sayısı: %d", descriptionCount))
	}
	return builder.String()
}

// getFileStats, botun yönetimindeki toplam dosya sayısını ve
// açıklama eklenmiş dosya sayısını hesaplar.
func getFileStats() (int, int) {
	var fileCount int
	// * `filepath.Walk`, tüm alt dizinleri gezerek dosya sayısını sayar.
	filepath.Walk(config.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
		}
		return nil
	})
	metadataMutex.Lock()
	descriptionCount := len(fileMetadata)
	metadataMutex.Unlock()
	return fileCount, descriptionCount
}

// runSpeedTest, `speedtest.exe` komut satırı aracını çalıştırır,
// JSON çıktısını yakalar ve `SpeedTestResult` yapısına dönüştürür.
func runSpeedTest() (*SpeedTestResult, error) {
	// * `--format json` argümanı, çıktının makine tarafından kolayca
	// * okunabilir olmasını sağlar.
	cmd := exec.Command("speedtest", "--format", "json", "--accept-license", "--accept-gdpr")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("speedtest hatası: %s\n%w", stderr.String(), err)
	}
	var result SpeedTestResult
	// * Gelen JSON metnini `result` adlı struct'a "unmarshal" et (çözümle).
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("speedtest sonucu çözümlenemedi: %w\nÇıktı: %s", err, out.String())
	}
	return &result, nil
}

// getConnectionQuality, hız testi sonuçlarına göre internet bağlantısına
// basit bir "kalite notu" verir.
func getConnectionQuality(downloadSpeed, ping float64) string {
	if downloadSpeed >= 50 && ping <= 30 {
		return "🟢 Mükemmel"
	}
	if downloadSpeed >= 25 && ping <= 50 {
		return "🟡 İyi"
	}
	if downloadSpeed >= 10 && ping <= 100 {
		return "🟠 Orta"
	}
	return "🔴 Zayıf"
}