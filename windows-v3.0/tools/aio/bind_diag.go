package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	hxcore "40hxcore"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

// TelemetryData chứa thông số trạng thái thời gian thực được đẩy qua event
type TelemetryData struct {
	GPUDetected    bool     `json:"gpuDetected"`
	GPUName        string   `json:"gpuName"`
	PCIBusID       string   `json:"pciBusId"`
	DeviceID       uint16   `json:"deviceId"`
	Is30HX         bool     `json:"is30HX"`
	Is40HX         bool     `json:"is40HX"`
	GSPActive      bool     `json:"gspActive"`
	IsGen2         bool     `json:"isGen2"`
	LinkSpeed      uint32   `json:"linkSpeed"`
	LinkWidth      uint32   `json:"linkWidth"`
	ThroughputMBs  uint32   `json:"throughputMBs"`
	ClockMHz       uint32   `json:"clockMHz"`
	TemperatureC   uint32   `json:"temperatureC"`
	Lanes          [16]bool `json:"lanes"`
	MRRS           uint32   `json:"mrrs"`
	DriverStrategy int      `json:"driverStrategy"`
	Timestamp      string   `json:"timestamp"`
}

// EnvCheckItem mô tả từng mục kiểm tra môi trường
type EnvCheckItem struct {
	Name string `json:"name"`
	Ok   bool   `json:"ok"`
	Note string `json:"note"`
}

// EnvironmentReport chứa toàn bộ báo cáo chẩn đoán môi trường
type EnvironmentReport struct {
	GPUDetected     bool           `json:"gpuDetected"`
	GPUName         string         `json:"gpuName"`
	PCIBusID        string         `json:"pciBusId"`
	DetectedModel   string         `json:"detectedModel"`
	DeviceID        uint16         `json:"deviceId"`
	Is30HX          bool           `json:"is30HX"`
	Is40HX          bool           `json:"is40HX"`
	GSPActive       bool           `json:"gspActive"`
	IsGen2          bool           `json:"isGen2"`
	DriverStrategy  int            `json:"driverStrategy"`
	AutoHard        bool           `json:"autoHard"`
	RetryCount      int            `json:"retryCount"`
	RetryInterval   int            `json:"retryInterval"`
	Items           []EnvCheckItem `json:"items"`
	Recommendations map[string]bool `json:"recommendations"`
}

// HVCIReport phản hồi trạng thái Memory Integrity (HVCI)
type HVCIReport struct {
	Enabled     bool   `json:"enabled"`
	StatusText  string `json:"statusText"`
	GuideText   string `json:"guideText"`
	CanAutoTurn bool   `json:"canAutoTurn"`
}

func (a *App) startTelemetryLoop() {
	ctx, cancel := context.WithCancel(context.Background())
	a.telemetryCancel = cancel

	go func() {
		ticker := time.NewTicker(1500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if a.ctx == nil {
					continue
				}
				data := a.gatherTelemetry()
				runtime.EventsEmit(a.ctx, "telemetry:tick", data)
			}
		}
	}()
}

func (a *App) gatherTelemetry() TelemetryData {
	prof, found := hxcore.FindGPUWithProfile()
	data := TelemetryData{
		GPUDetected:    found,
		GPUName:        "Chưa phát hiện GPU CMP",
		PCIBusID:       "Không khả dụng",
		DeviceID:       0,
		GSPActive:      hxcore.GspEnabled(),
		DriverStrategy: hxcore.DriverStrategy(),
		Timestamp:      time.Now().Format("15:04:05"),
		MRRS:           128,
	}

	if !found {
		return data
	}

	data.GPUName = fmt.Sprintf("%s (%s)", prof.Name, prof.Family)
	data.PCIBusID = prof.HardwareID
	data.DeviceID = prof.DeviceID
	data.Is30HX = prof.DeviceID == 0x2189 || prof.Family == "TU116"
	data.Is40HX = prof.DeviceID == 0x1F0B || prof.Family == "TU106"

	// Đọc trạng thái tốc độ link thực tế từ StatusContract hoặc Task
	taskOK, _, _ := hxcore.TaskInfo("CMP30HX_Gen2_Unlock")
	if !taskOK {
		taskOK, _, _ = hxcore.TaskInfo("40HX PCIe Gen2 Bring-up")
	}

	// Mặc định an toàn là Gen1 hoặc Gen2 theo cờ cấu hình
	speed := uint32(1)
	width := uint32(16)

	// Phân tích trạng thái gen2_status.txt nếu có
	statusPath := filepath.Join(os.Getenv("ProgramData"), "40HXUnlock", "gen2_status.txt")
	if content, err := os.ReadFile(statusPath); err == nil {
		text := string(content)
		if strings.Contains(text, "STATUS_CODE=GEN2_SUCCESS") || strings.Contains(text, "SPEED_CURRENT=2") {
			speed = 2
		}
		if strings.Contains(text, "DEVCTL MRRS=512B") || strings.Contains(text, "MRRS 512B") {
			data.MRRS = 512
		}
	} else if taskOK {
		speed = 2
	}

	data.LinkSpeed = speed
	data.LinkWidth = width
	data.IsGen2 = speed >= 2

	// Tính toán băng thông thực tế tương đối
	// Gen1: 250 MB/s mỗi lane; Gen2: 500 MB/s mỗi lane
	perLane := uint32(250)
	if speed >= 2 {
		perLane = 500
	}
	data.ThroughputMBs = perLane * width
	if speed >= 2 && width == 16 && data.MRRS == 512 {
		data.ThroughputMBs = 6400 // ~6.4 GB/s tối đa
	}

	// Ma trận 16 làn vật lý
	for i := uint32(0); i < 16; i++ {
		data.Lanes[i] = i < width
	}

	// Đọc xung nhịp và nhiệt độ GPU
	clock, temp := queryGPUSensors()
	data.ClockMHz = clock
	data.TemperatureC = temp

	return data
}

// ScanEnvironment quét toàn bộ môi trường và trả về danh sách chỉ số chi tiết
func (a *App) ScanEnvironment() EnvironmentReport {
	legacy := hxcore.FirmwareIsLegacy()
	sbOn := hxcore.SecureBootOn()
	gpuProf, gpuFound := hxcore.FindGPUWithProfile()
	gspOK := hxcore.GspEnabled()

	// Kiểm tra trạng thái Microsoft Vulnerable Driver Blocklist
	blocklistDisabled := false
	kCi, errCi := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\CI\Config`, registry.QUERY_VALUE)
	if errCi == nil {
		v, _, errVal := kCi.GetIntegerValue("VulnerableDriverBlocklistEnable")
		if errVal == nil && v == 0 {
			blocklistDisabled = true
		}
		kCi.Close()
	}

	items := []EnvCheckItem{
		{
			Name: "Chế độ Boot",
			Ok:   !legacy,
			Note: map[bool]string{true: "UEFI (OK)", false: "Legacy+MBR — Cần chuyển sang GPT bằng mbr2gpt"}[!legacy],
		},
		{
			Name: "Secure Boot",
			Ok:   !sbOn,
			Note: map[bool]string{true: "Đang Bật (Cần kiểm tra tương thích key)", false: "Đã Tắt (OK)"}[sbOn],
		},
		{
			Name: "Danh sách đen driver (Blocklist)",
			Ok:   blocklistDisabled,
			Note: map[bool]string{true: "Đã Tắt (VulnerableDriverBlocklistEnable=0 — nạp driver mod OK)", false: "Đang Bật (Có thể chặn driver mod và công cụ can thiệp)"}[blocklistDisabled],
		},
		{
			Name: "Card đồ hoạ",
			Ok:   gpuFound,
			Note: map[bool]string{true: fmt.Sprintf("Phát hiện %s [%s]", gpuProf.Name, gpuProf.Family), false: "Chưa phát hiện — Kiểm tra khe cắm & driver"}[gpuFound],
		},
		{
			Name: "GSP (EnableGpuFirmware)",
			Ok:   gspOK,
			Note: map[bool]string{true: "Đã bật (OK)", false: "Chưa bật — Nguy cơ lỗi Code 43"}[gspOK],
		},
	}

	// Kiểm tra ESP EFI
	espEFI := false
	if esp := hxcore.MountESP(); esp != "" {
		if _, err := os.Stat(filepath.Join(esp, "EFI", "40HX", "40HXUNLK.EFI")); err == nil {
			espEFI = true
		}
		hxcore.UnmountESP(esp)
	}
	items = append(items, EnvCheckItem{
		Name: "ESP EFI Mở khoá",
		Ok:   espEFI,
		Note: map[bool]string{true: "\\EFI\\40HX\\40HXUNLK.EFI đã nạp", false: "Chưa nạp vào ESP"}[espEFI],
	})

	// Tác vụ tự khởi động
	taskOK, taskStatus, _ := hxcore.TaskInfo("CMP30HX_Gen2_Unlock")
	if !taskOK {
		taskOK, taskStatus, _ = hxcore.TaskInfo("40HX PCIe Gen2 Bring-up")
	}
	taskNote := "Chưa đăng ký — Sẽ không tự mở khoá khi bật máy"
	if taskOK {
		taskNote = "Trạng thái: " + taskStatus + " (Đã đăng ký)"
	}
	items = append(items, EnvCheckItem{
		Name: "Tác vụ tự khởi động",
		Ok:   taskOK,
		Note: taskNote,
	})

	// ASPM
	aspmOK := true
	if ac, dc, ok := hxcore.ASPMSavings(); ok {
		aspmOK = (ac == 0 && dc == 0)
	}
	items = append(items, EnvCheckItem{
		Name: "Tiết kiệm điện PCIe (ASPM)",
		Ok:   aspmOK,
		Note: map[bool]string{true: "Đã tắt (Tối đa hiệu năng)", false: "Đang bật (Nguy cơ rớt link Gen1 khi rảnh)"}[aspmOK],
	})

	// Fast startup
	fastOK := !hxcore.FastStartupOn()
	items = append(items, EnvCheckItem{
		Name: "Khởi động nhanh (Fast Startup)",
		Ok:   fastOK,
		Note: map[bool]string{true: "Đã tắt (OK)", false: "Đang bật (Có thể làm kẹt link Gen1 sau reboot)"}[fastOK],
	})

	strat := hxcore.DriverStrategy()
	autoHard := hxcore.ConfigInt("Gen2AutoHard", 1) != 0
	cnt, interval := hxcore.Gen2RetryPolicy()

	gpuName := "Chưa phát hiện GPU CMP"
	pciBusId := "Không khả dụng"
	detectedModel := "Unknown"
	deviceID := uint16(0)
	if gpuFound {
		gpuName = fmt.Sprintf("%s (%s)", gpuProf.Name, gpuProf.Family)
		pciBusId = gpuProf.HardwareID
		detectedModel = gpuProf.Name
		deviceID = gpuProf.DeviceID
	}

	return EnvironmentReport{
		GPUDetected:    gpuFound,
		GPUName:        gpuName,
		PCIBusID:       pciBusId,
		DetectedModel:  detectedModel,
		DeviceID:       deviceID,
		Is30HX:         deviceID == 0x2189,
		Is40HX:         deviceID == 0x1F0B,
		GSPActive:      gspOK,
		IsGen2:         taskOK,
		DriverStrategy: strat,
		AutoHard:       autoHard,
		RetryCount:     cnt,
		RetryInterval:  interval,
		Items:          items,
		Recommendations: map[string]bool{
			"gsp":    !gspOK,
			"drv":    hxcore.Gen2DriversNeedDeploy(),
			"efi":    !espEFI,
			"task":   !taskOK,
			"fast":   !fastOK,
			"aspm":   !aspmOK,
			"perf":   !hxcore.HighPerfPlanActive(),
		},
	}
}

// CheckHVCIStatus kiểm tra tính năng Memory Integrity (Core Isolation)
func (a *App) CheckHVCIStatus() HVCIReport {
	enabled := false
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		val, _, err := k.GetIntegerValue("Enabled")
		if err == nil && val == 1 {
			enabled = true
		}
	}

	report := HVCIReport{
		Enabled:     enabled,
		CanAutoTurn: false, // TUYỆT ĐỐI KHÔNG TỰ TẮT theo yêu cầu an toàn
	}

	if enabled {
		report.StatusText = "Memory Integrity (HVCI) ĐANG BẬT"
		report.GuideText = "Windows đang bật Memory Integrity (HVCI). Tính năng này bảo vệ kernel nhưng sẽ chặn driver phần cứng ghi thanh ghi BAR0 MMIO.\n\n👉 HƯỚNG DẪN THỦ CÔNG:\n1. Bấm nút [Mở Windows Security] bên dưới\n2. Trong mục Core isolation -> Chuyển 'Memory integrity' sang TẮT (Off)\n3. Khởi động lại máy tính để thay đổi có hiệu lực."
	} else {
		report.StatusText = "Memory Integrity (HVCI) ĐÃ TẮT"
		report.GuideText = "Hệ thống sẵn sàng cho việc ghi cấu hình phần cứng qua kernel driver."
	}

	return report
}

// OpenCoreIsolationSettings mở trang cài đặt Core Isolation của Windows
func (a *App) OpenCoreIsolationSettings() error {
	cmd := exec.Command("cmd.exe", "/c", "start", "windowsdefender://coreisolation")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("không thể mở cài đặt Core Isolation: %w", err)
	}
	return nil
}

func queryGPUSensors() (uint32, uint32) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nvidia-smi.exe", "--query-gpu=clocks.current.graphics,temperature.gpu", "--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) >= 2 {
		var clock, temp uint32
		fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &clock)
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &temp)
		return clock, temp
	}

	return 0, 0
}
