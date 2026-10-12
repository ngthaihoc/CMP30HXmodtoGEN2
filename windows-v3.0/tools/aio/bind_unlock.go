package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	hxcore "40hxcore"
	"golang.org/x/sys/windows/registry"
)

const (
	taskName30HX = "CMP30HX_Gen2_Unlock"
	taskName40HX = "40HX PCIe Gen2 Bring-up"
)

// UnlockResult chứa kết quả sau khi thực thi mở khoá
type UnlockResult struct {
	Success bool   `json:"success"`
	Speed   uint32 `json:"speed"`
	Width   uint32 `json:"width"`
	Message string `json:"message"`
}

// UnlockGen2Now thực thi mở khoá PCIe Gen2 x16 và tối ưu DEVCTL MRRS 512B
func (a *App) UnlockGen2Now() UnlockResult {
	if !a.tryAcquire("Mở khoá Gen2 ngay") {
		return UnlockResult{Success: false, Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()

	a.Log("[PCIe] Bắt đầu kích hoạt mở khoá Gen2 phiên hiện tại...")
	return a.executeUnlock(false)
}

// ForceRootGen2 ép Root Port huấn luyện lại với chuỗi MMIO shadow
func (a *App) ForceRootGen2() UnlockResult {
	if !a.tryAcquire("Ép mở khoá Gen2 qua Root Port") {
		return UnlockResult{Success: false, Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()

	a.Log("[PCIe] ⚡ Bắt đầu ép mở khoá Gen2 qua Root Port (-force-root-gen2)...")
	return a.executeUnlock(true)
}

func (a *App) executeUnlock(forceRoot bool) UnlockResult {
	var result UnlockResult

	err := hxcore.RunScopedBus(true, func(bus hxcore.HardwareBus) error {
		var gpuBDF uint32
		var gpuProfile hxcore.GPUProfile
		gpuFound := false

		for attempt := 1; attempt <= 3; attempt++ {
			gpuBDF, gpuProfile, gpuFound = hxcore.FindGPUPCIWithBus(bus)
			if gpuFound {
				break
			}
			if attempt < 3 {
				a.Log(fmt.Sprintf("[PCIe] Chưa định vị được GPU trên bus PCI, thử lại sau 2s (%d/3)...", attempt))
				time.Sleep(2 * time.Second)
			}
		}

		if !gpuFound {
			msg := "Không tìm thấy card CMP tương thích trên bus PCI. Vui lòng kiểm tra nguồn phụ và khe cắm."
			a.Log("[!] " + msg)
			result = UnlockResult{Success: false, Message: msg}
			return nil
		}

		a.Log(fmt.Sprintf("[PCIe] 🛡️ Nhận diện phần cứng: %s [%s] (DEV_%04X tại Bus %02X)",
			gpuProfile.Name, gpuProfile.Family, gpuProfile.DeviceID, (gpuBDF>>8)&0xFF))

		targetGen := uint32(2)
		if gpuProfile.MaxSupportedGen < 2 {
			targetGen = gpuProfile.MaxSupportedGen
		}

		// Ranh giới an toàn phần cứng: TU116 (30HX) kẹp eFuse tuyệt đối tại Gen2
		allowStage2 := gpuProfile.DeviceID != 0x2189 && hxcore.ConfigInt("Gen2AutoHard", 1) != 0

		if gpuProfile.DeviceID == 0x2189 {
			a.Log("[PCIe] 🔒 Kích hoạt ranh giới bảo vệ CMP 30HX: Khóa cứng Gen2 (5.0 GT/s), cấm reset PnP nguy hiểm.")
		} else {
			a.Log("[PCIe] ⚡ Chế độ mở khóa CMP 40HX: Nạp MMIO TU106 shadow và tối ưu MRRS 512B.")
		}

		negotiator := hxcore.NewLinkNegotiator(bus)
		opts := hxcore.NegotiationOptions{
			TargetGen:   targetGen,
			ForceRoot:   forceRoot || gpuProfile.RequiresMMIO,
			AllowStage2: allowStage2,
			UserIsAdmin: hxcore.IsAdmin(),
		}

		verdict, negErr := negotiator.ExecuteNegotiation(gpuBDF, gpuProfile, opts)
		if negErr != nil {
			a.Log(fmt.Sprintf("[PCIe][!] Lỗi đàm phán liên kết: %v", negErr))
			result = UnlockResult{Success: false, Message: negErr.Error()}
			return nil
		}

		a.Log(fmt.Sprintf("[PCIe] Kết quả: Tốc độ hiện tại Gen%d, Độ rộng x%d | Kết luận: %s",
			verdict.CurrentSpeed, verdict.CurrentWidth, verdict.Verdict))

		result = UnlockResult{
			Success: verdict.CurrentSpeed >= 2,
			Speed:   verdict.CurrentSpeed,
			Width:   verdict.CurrentWidth,
			Message: verdict.Verdict,
		}

		return nil
	})

	if err != nil {
		a.Log(fmt.Sprintf("[!] Lỗi nạp phiên driver: %v", err))
		return UnlockResult{Success: false, Message: err.Error()}
	}

	return result
}

// RegisterAutostartTask đăng ký tác vụ tự động mở khoá khi đăng nhập Windows
func (a *App) RegisterAutostartTask() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("không thể xác định file thực thi: %w", err)
	}
	abs, _ := filepath.Abs(exe)

	// Đăng ký Scheduled Task chạy SYSTEM khi Logon
	tn := taskName30HX
	prof, found := hxcore.FindGPUWithProfile()
	if found && prof.DeviceID == 0x1F0B {
		tn = taskName40HX
	}

	a.Log(fmt.Sprintf("[Tự khởi động] Đăng ký tác vụ Scheduled Task %q...", tn))
	out, err := hxcore.RunOut("schtasks.exe", "/create", "/tn", tn,
		"/tr", fmt.Sprintf("\"%s\" -gen2 -silent", abs),
		"/sc", "onlogon", "/ru", "SYSTEM", "/delay", "0000:10", "/rl", "highest", "/f")
	if err != nil {
		a.Log(fmt.Sprintf("[!] Lỗi tạo tác vụ: %s", strings.TrimSpace(out)))
		return fmt.Errorf("tạo tác vụ thất bại: %s", strings.TrimSpace(out))
	}

	// Đăng ký Registry Run Key dự phòng
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err == nil {
		defer k.Close()
		_ = k.SetStringValue("40HXGen2", fmt.Sprintf("\"%s\" -gen2 -silent", abs))
	}

	a.Log("[Tự khởi động] ✓ Đăng ký thành công tác vụ tự động mở khóa khi đăng nhập.")
	return nil
}

// UnregisterAutostartTask gỡ bỏ tác vụ tự động mở khoá
func (a *App) UnregisterAutostartTask() error {
	a.Log("[Tự khởi động] Đang xoá bỏ tác vụ tự khởi động...")
	_, _ = hxcore.RunOut("schtasks.exe", "/delete", "/tn", taskName30HX, "/f")
	_, _ = hxcore.RunOut("schtasks.exe", "/delete", "/tn", taskName40HX, "/f")

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err == nil {
		defer k.Close()
		_ = k.DeleteValue("40HXGen2")
	}

	a.Log("[Tự khởi động] ✓ Đã gỡ bỏ toàn bộ tác vụ tự chạy khi đăng nhập.")
	return nil
}

// SavePolicy lưu thiết lập chiến lược driver và chu kỳ retry
func (a *App) SavePolicy(strategy, retryCount, retryInterval, autoHard int) error {
	if err := hxcore.SetConfigInt("DriverStrategy", strategy); err != nil {
		return fmt.Errorf("lưu DriverStrategy thất bại: %w", err)
	}
	_ = hxcore.SetConfigInt("Gen2AutoHard", autoHard)
	_ = hxcore.SetConfigInt("Gen2RetryCount", retryCount)
	_ = hxcore.SetConfigInt("Gen2RetryIntervalMin", retryInterval)

	a.Log(fmt.Sprintf("[Cấu hình] Đã lưu chính sách: Chiến lược=%d, Tự động Stage2=%d, Thử lại=%d lần / Giãn cách=%d phút",
		strategy, autoHard, retryCount, retryInterval))
	return nil
}
