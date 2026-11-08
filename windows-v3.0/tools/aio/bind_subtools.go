package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	hxcore "40hxcore"
	"golang.org/x/sys/windows/registry"
)

// SubtoolResult kết quả trả về từ công cụ phụ trợ.
type SubtoolResult struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	Attempted         bool   `json:"attempted,omitempty"`
	Error             string `json:"error,omitempty"`
	PendingReboot     bool   `json:"pendingReboot,omitempty"`
	Verified          bool   `json:"verified,omitempty"`
	RollbackAttempted bool   `json:"rollbackAttempted,omitempty"`
	RollbackSucceeded bool   `json:"rollbackSucceeded,omitempty"`
	RollbackError     string `json:"rollbackError,omitempty"`
}

type rebarDependencies struct {
	checkLaptop   func() (bool, string)
	extractAssets func() (string, error)
	run           func(context.Context, string, ...string) (string, error)
	log           func(string)
}

func defaultRebarDependencies(log func(string)) rebarDependencies {
	return rebarDependencies{
		checkLaptop:   checkIsLaptop,
		extractAssets: ExtractToolAssets,
		run: func(ctx context.Context, executable string, args ...string) (string, error) {
			out, err := exec.CommandContext(ctx, executable, args...).CombinedOutput()
			return string(out), err
		},
		log: log,
	}
}

func (a *App) rebarLogResult(operation string, result SubtoolResult) SubtoolResult {
	a.Log(fmt.Sprintf("[ReBAR] %s kết quả: success=%t attempted=%t pendingReboot=%t verified=%t rollbackAttempted=%t rollbackSucceeded=%t error=%q rollbackError=%q", operation, result.Success, result.Attempted, result.PendingReboot, result.Verified, result.RollbackAttempted, result.RollbackSucceeded, result.Error, result.RollbackError))
	return result
}

// FixCSMError43 tự động xử lý xung đột tài nguyên CSM và lỗi Code 43.
func (a *App) FixCSMError43() SubtoolResult {
	if !a.tryAcquire("Sửa lỗi CSM Code 43") {
		return SubtoolResult{Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()

	a.Log("==============================================")
	a.Log("  [CSM Fix] Bắt đầu khắc phục lỗi Code 43 / CSM")
	a.Log("==============================================")
	a.Log("[*] Bước 1/3: Kích hoạt quét lại phần cứng toàn bộ Bus PCIe (Rescan)...")
	out, err := hxcore.RunOut("pnputil.exe", "/scan-devices")
	if err == nil {
		a.Log(fmt.Sprintf("  [✓] pnputil đã quét bus: %s", strings.TrimSpace(out)))
	}
	a.Log("[*] Bước 2/3: Cấu hình chống ngủ sâu D3Cold và khoá ASPM...")
	_ = hxcore.SetFastStartupOff()
	_ = hxcore.SetASPMOff()
	key := hxcore.FindGpuClassKey()
	if key != "" {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.SET_VALUE)
		if err == nil {
			_ = k.SetDWordValue("D3ColdSupported", 0)
			_ = k.SetDWordValue("DeviceSelectiveSuspended", 0)
			_ = k.SetDWordValue("RmDisableGpuPowerMgmt", 1)
			_ = k.SetDWordValue("DisableASPM", 1)
			_ = k.SetDWordValue("DisablePCIePowerManagement", 1)
			k.Close()
			a.Log("  [✓] Đã khoá D3Cold và thiết lập giữ kết nối PCIe luôn ở trạng thái thức.")
		}
	}
	a.Log("[*] Bước 3/3: Cấu hình TDR Delay 10s...")
	if tdrKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, registry.SET_VALUE); err == nil {
		_ = tdrKey.SetDWordValue("TdrDelay", 10)
		_ = tdrKey.SetDWordValue("TdrDdiDelay", 10)
		tdrKey.Close()
		a.Log("  [✓] Đã thiết lập TdrDelay=10s.")
	}
	a.Log("==============================================")
	a.Log("  [✓] Khắc phục lỗi CSM hoàn tất! Vui lòng khởi động lại máy nếu cần.")
	a.Log("==============================================")
	return SubtoolResult{Success: true, Message: "Đã áp dụng các bản vá CSM/D3Cold và quét lại Bus PCI thành công."}
}

// SetupReBar cấu hình Resizable BAR cho CMP 30HX (có khoá an toàn Laptop).
func (a *App) SetupReBar(barSizeGB int) SubtoolResult {
	if !a.tryAcquire("Cài đặt Resizable BAR") {
		return SubtoolResult{Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()
	a.Log("[ReBAR] Bắt đầu thiết lập; chưa xác minh trạng thái phần cứng.")
	return a.rebarLogResult("Thiết lập ReBAR", executeSetupReBar(barSizeGB, defaultRebarDependencies(a.Log)))
}

func executeSetupReBar(barSizeGB int, deps rebarDependencies) SubtoolResult {
	isLaptop, details := deps.checkLaptop()
	if !isLaptop && strings.TrimSpace(details) == "" {
		return SubtoolResult{Message: "Không thể xác minh loại thiết bị; đã chặn ReBAR để an toàn.", Error: "Không xác định được hệ thống Desktop/Laptop; thao tác bị chặn an toàn."}
	}
	if isLaptop {
		return SubtoolResult{Message: fmt.Sprintf("PHÁT HIỆN HỆ THỐNG LAPTOP (%s). ĐÃ CHẶN THAO TÁC REBAR ĐỂ BẢO VỆ MÁY.", details), Error: "Thao tác bị khóa an toàn Laptop."}
	}
	if barSizeGB <= 0 {
		return SubtoolResult{Message: "Kích thước BAR không hợp lệ.", Error: "Kích thước BAR phải lớn hơn 0 GB."}
	}
	tempDir, err := deps.extractAssets()
	if err != nil {
		return SubtoolResult{Message: "Trích xuất công cụ ReBAR thất bại.", Error: fmt.Sprintf("Trích xuất công cụ ReBAR thất bại: %v", err)}
	}
	inspectorExe := filepath.Join(tempDir, "nvidiaProfileInspector.exe")
	profileNip := filepath.Join(tempDir, "Enable_ReBAR_Turing.nip")
	nvStrapsExe := filepath.Join(tempDir, "NvStrapsReBar.exe")
	missing := make([]string, 0, 3)
	for _, path := range []string{inspectorExe, profileNip, nvStrapsExe} {
		if !fileExists(path) {
			missing = append(missing, filepath.Base(path))
		}
	}
	if len(missing) > 0 {
		return SubtoolResult{Message: "Thiếu tài nguyên ReBAR; không chạy công cụ.", Error: "Thiếu tài nguyên: " + strings.Join(missing, ", ")}
	}
	if deps.log != nil {
		deps.log("[ReBAR] Trước Profile Inspector; chưa xác minh trạng thái phần cứng.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	out, err := deps.run(ctx, inspectorExe, profileNip, "-silent")
	cancel()
	result := SubtoolResult{}
	if err != nil {
		result.Attempted = true
		result.Error = fmt.Sprintf("Profile Inspector thất bại: %s (%v)", strings.TrimSpace(out), err)
		result.Message = "Nạp profile ReBAR thất bại; không chạy NvStrapsReBar. Trạng thái phần cứng chưa được xác minh."
		if deps.log != nil {
			deps.log("[ReBAR] Sau Profile Inspector: thất bại; NvStrapsReBar không chạy.")
		}
		return result
	}
	if deps.log != nil {
		deps.log(fmt.Sprintf("[ReBAR] Sau Profile Inspector: lệnh kết thúc thành công; output=%q. Phần cứng chưa xác minh.", strings.TrimSpace(out)))
	}

	result.Attempted = true
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	out, err = deps.run(ctx, nvStrapsExe, fmt.Sprintf("%d", barSizeGB))
	cancel()
	if err != nil {
		result.Error = fmt.Sprintf("NvStrapsReBar thất bại: %s (%v)", strings.TrimSpace(out), err)
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		rollbackOut, rollbackErr := deps.run(ctx, nvStrapsExe, "0")
		cancel()
		result.RollbackAttempted = true
		result.RollbackSucceeded = rollbackErr == nil
		if rollbackErr != nil {
			result.RollbackError = fmt.Sprintf("NvStrapsReBar rollback thất bại: %s (%v)", strings.TrimSpace(rollbackOut), rollbackErr)
		}
		if deps.log != nil {
			deps.log(fmt.Sprintf("[ReBAR] Rollback: attempted=true succeeded=%t output=%q error=%v", result.RollbackSucceeded, strings.TrimSpace(rollbackOut), rollbackErr))
		}
		result.Message = "Thiết lập ReBAR không hoàn tất; xem lỗi và bằng chứng hoàn tác. Trạng thái phần cứng chưa được xác minh."
		return result
	}
	result.Success = true
	result.PendingReboot = true
	result.Message = "Lệnh thiết lập đã hoàn tất; cần khởi động lại rồi xác minh bằng GPU-Z. Trạng thái phần cứng hiện chưa được xác minh."
	if deps.log != nil {
		deps.log(fmt.Sprintf("[ReBAR] Sau NvStrapsReBar: lệnh kết thúc thành công; output=%q. Pending reboot; phần cứng chưa xác minh.", strings.TrimSpace(out)))
	}
	return result
}

// RevertReBar khôi phục cấu hình ReBAR về mặc định.
func (a *App) RevertReBar() SubtoolResult {
	if !a.tryAcquire("Hoàn tác ReBAR") {
		return SubtoolResult{Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()
	a.Log("[ReBAR] Bắt đầu hoàn tác; chưa xác minh trạng thái phần cứng.")
	return a.rebarLogResult("Hoàn tác ReBAR", executeRevertReBar(defaultRebarDependencies(a.Log)))
}

func executeRevertReBar(deps rebarDependencies) SubtoolResult {
	tempDir, err := deps.extractAssets()
	if err != nil {
		return SubtoolResult{Message: "Trích xuất công cụ ReBAR thất bại.", Error: fmt.Sprintf("Trích xuất công cụ ReBAR thất bại: %v", err)}
	}
	nvStrapsExe := filepath.Join(tempDir, "NvStrapsReBar.exe")
	if !fileExists(nvStrapsExe) {
		return SubtoolResult{Message: "Không thể hoàn tác ReBAR.", Error: "Thiếu tài nguyên NvStrapsReBar.exe."}
	}
	if deps.log != nil {
		deps.log("[ReBAR] Trước hoàn tác NvStrapsReBar; chưa xác minh trạng thái phần cứng.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	out, err := deps.run(ctx, nvStrapsExe, "0")
	cancel()
	result := SubtoolResult{Attempted: true}
	if err != nil {
		result.Error = fmt.Sprintf("NvStrapsReBar hoàn tác thất bại: %s (%v)", strings.TrimSpace(out), err)
		result.Message = "Hoàn tác ReBAR thất bại; trạng thái phần cứng chưa được xác minh."
		if deps.log != nil {
			deps.log(fmt.Sprintf("[ReBAR] Sau hoàn tác: thất bại; output=%q error=%v.", strings.TrimSpace(out), err))
		}
		return result
	}
	result.Success = true
	result.PendingReboot = true
	result.Message = "Lệnh hoàn tác đã hoàn tất; cần khởi động lại rồi xác minh bằng GPU-Z. Trạng thái phần cứng hiện chưa được xác minh."
	if deps.log != nil {
		deps.log(fmt.Sprintf("[ReBAR] Sau hoàn tác: lệnh kết thúc thành công; output=%q. Pending reboot; phần cứng chưa xác minh.", strings.TrimSpace(out)))
	}
	return result
}

func checkIsLaptop() (bool, string) {
	psCheck := `
$hasBattery = (Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue | Measure-Object).Count -gt 0
$cs = Get-CimInstance Win32_ComputerSystem
$model = "$($cs.Manufacturer) $($cs.Model)"
Write-Output "$hasBattery|$model"
`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", psCheck)
	out, err := cmd.Output()
	if err != nil {
		return true, "Không thể kiểm tra thiết bị: " + err.Error()
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return true, "Không thể xác định loại thiết bị."
	}
	hasBattery := strings.EqualFold(strings.TrimSpace(parts[0]), "True")
	model := strings.TrimSpace(parts[1])
	if hasBattery {
		return true, model + " (Phát hiện Pin Laptop)"
	}
	lower := strings.ToLower(model)
	for _, keyword := range []string{"laptop", "notebook", "book", "portable", "nitro", "legion", "victus", "tuf", "rog", "zephyrus", "blade", "omen", "pavilion", "inspiron", "latitude", "precision", "xps", "yoga", "thinkpad", "ideapad"} {
		if strings.Contains(lower, keyword) {
			return true, model
		}
	}
	return false, model
}
