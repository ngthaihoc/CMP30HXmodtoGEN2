package main

import (
	"fmt"

	hxcore "40hxcore"
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
		verdict, negErr := hxcore.NewLinkNegotiator(bus).Unlock(hxcore.NegotiationOptions{
			TargetGen: 2, ForceRoot: forceRoot,
			AllowStage2: hxcore.ConfigInt("Gen2AutoHard", 1) != 0,
			UserIsAdmin: hxcore.IsAdmin(),
		})
		if err := hxcore.WriteStructuredGen2Status(verdict.StatusContract); err != nil {
			a.Log(fmt.Sprintf("[!] Không thể lưu trạng thái mở khóa: %v", err))
		}
		if negErr != nil {
			a.Log(fmt.Sprintf("[PCIe][!] Lỗi đàm phán liên kết: %v", negErr))
		}
		a.Log("[PCIe] " + verdict.Verdict)
		result = UnlockResult{Success: verdict.Success, Speed: verdict.CurrentSpeed,
			Width: verdict.CurrentWidth, Message: verdict.Verdict}
		return nil
	})
	if err != nil {
		a.Log(fmt.Sprintf("[!] Lỗi nạp phiên driver: %v", err))
		if writeErr := hxcore.WriteStructuredGen2Status(hxcore.StatusContract{
			StatusCode: hxcore.StatusDrvFail, ErrorCode: "DRIVER_SESSION_FAILED", Details: []string{err.Error()},
		}); writeErr != nil {
			a.Log(fmt.Sprintf("[!] Không thể lưu trạng thái lỗi driver: %v", writeErr))
		}
		return UnlockResult{Message: err.Error()}
	}
	return result
}

// RegisterAutostartTask registers one SYSTEM task for the current AIO executable.
func (a *App) RegisterAutostartTask() error {
	a.Log("[Tự khởi động] Đang đăng ký tác vụ khi đăng nhập...")
	if err := registerAutostart(defaultStartupDependencies()); err != nil {
		a.Log("[!] " + err.Error())
		return err
	}
	a.Log("[Tự khởi động] Đã đăng ký tác vụ SYSTEM. Run key cũ thuộc ứng dụng đã được dọn.")
	return nil
}

// UnregisterAutostartTask removes only tasks and the old Run value owned by this AIO executable.
func (a *App) UnregisterAutostartTask() error {
	a.Log("[Tự khởi động] Đang gỡ tác vụ khi đăng nhập...")
	if err := unregisterAutostart(defaultStartupDependencies()); err != nil {
		a.Log("[!] " + err.Error())
		return err
	}
	a.Log("[Tự khởi động] Đã gỡ tác vụ và Run key cũ thuộc ứng dụng.")
	return nil
}

// PolicyConfig gom các thiết lập chiến lược driver và chu kỳ retry.
type PolicyConfig struct {
	Strategy      int `json:"strategy"`
	RetryCount    int `json:"retryCount"`
	RetryInterval int `json:"retryInterval"`
	AutoHard      int `json:"autoHard"`
}

// SavePolicy lưu thiết lập chiến lược driver và chu kỳ retry.
func (a *App) SavePolicy(cfg PolicyConfig) error {
	if err := hxcore.SetConfigInt("DriverStrategy", cfg.Strategy); err != nil {
		return fmt.Errorf("lưu DriverStrategy thất bại: %w", err)
	}
	_ = hxcore.SetConfigInt("Gen2AutoHard", cfg.AutoHard)
	_ = hxcore.SetConfigInt("Gen2RetryCount", cfg.RetryCount)
	_ = hxcore.SetConfigInt("Gen2RetryIntervalMin", cfg.RetryInterval)

	a.Log(fmt.Sprintf("[Cấu hình] Đã lưu chính sách: Chiến lược=%d, Tự động Stage2=%d, Thử lại=%d lần / Giãn cách=%d phút",
		cfg.Strategy, cfg.AutoHard, cfg.RetryCount, cfg.RetryInterval))
	return nil
}
