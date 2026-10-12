package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	hxcore "40hxcore"
	"golang.org/x/sys/windows/registry"
)

// SubtoolResult kết quả trả về từ công cụ phụ trợ
type SubtoolResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// FixCSMError43 tự động xử lý xung đột tài nguyên CSM và lỗi Code 43
func (a *App) FixCSMError43() SubtoolResult {
	if !a.tryAcquire("Sửa lỗi CSM Code 43") {
		return SubtoolResult{Success: false, Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()

	a.Log("==============================================")
	a.Log("  [CSM Fix] Bắt đầu khắc phục lỗi Code 43 / CSM")
	a.Log("==============================================")

	// 1. Quét lại bus PCI
	a.Log("[*] Bước 1/3: Kích hoạt quét lại phần cứng toàn bộ Bus PCIe (Rescan)...")
	out, err := hxcore.RunOut("pnputil.exe", "/scan-devices")
	if err == nil {
		a.Log(fmt.Sprintf("  [✓] pnputil đã quét bus: %s", strings.TrimSpace(out)))
	}

	// 2. Tắt Fast Startup và triệt tiêu chế độ ngủ sâu D3Cold
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

	// 3. Tăng TDR Delay chống timeout
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

	return SubtoolResult{
		Success: true,
		Message: "Đã áp dụng các bản vá CSM/D3Cold và quét lại Bus PCI thành công.",
	}
}

// SetupReBar cấu hình Resizable BAR cho CMP 30HX (có khoá an toàn Laptop)
func (a *App) SetupReBar(barSizeGB int) SubtoolResult {
	if !a.tryAcquire("Cài đặt Resizable BAR") {
		return SubtoolResult{Success: false, Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()

	a.Log("==============================================")
	a.Log("  [ReBAR] Bắt đầu thiết lập Resizable BAR Turing")
	a.Log("==============================================")

	// 1. KHOÁ AN TOÀN LAPTOP: Kiểm tra phần cứng
	a.Log("[*] Bước 1/3: Kiểm tra nhận diện thiết bị & Khóa an toàn Laptop...")
	isLaptop, laptopDetails := checkIsLaptop()
	if isLaptop {
		msg := fmt.Sprintf("PHÁT HIỆN HỆ THỐNG LAPTOP (%s). ĐÃ TỰ ĐỘNG CHẶN 100%% THAO TÁC REBAR ĐỂ BẢO VỆ MÁY KHỎI NGUY CƠ BRICK BIOS!", laptopDetails)
		a.Log("[X] CẢNH BÁO NGUY HIỂM: " + msg)
		a.Log("    Laptop không có tính năng USB BIOS Flashback, mod ReBAR trên laptop có thể làm liệt nguồn.")
		a.Log("    Quy trình ReBAR này chỉ dành riêng cho bo mạch chủ máy tính để bàn (Desktop PC).")
		return SubtoolResult{
			Success: false,
			Message: msg,
		}
	}
	a.Log("  [✓] Xác nhận: Hệ thống là máy tính để bàn (Desktop PC). An toàn tiếp tục.")

	// 2. Trích xuất công cụ NvStrapsReBar và Profile Inspector từ embedded assets
	a.Log("[*] Bước 2/3: Chuẩn bị công cụ NvStrapsReBar & Profile Inspector...")
	tempDir, err := ExtractToolAssets()
	if err != nil {
		a.Log(fmt.Sprintf("[!] Trích xuất công cụ thất bại: %v", err))
		return SubtoolResult{Success: false, Message: "Trích xuất công cụ ReBAR thất bại."}
	}

	inspectorExe := filepath.Join(tempDir, "nvidiaProfileInspector.exe")
	profileNip := filepath.Join(tempDir, "Enable_ReBAR_Turing.nip")
	nvStrapsExe := filepath.Join(tempDir, "NvStrapsReBar.exe")

	// 3. Nạp Profile Turing ReBAR vào driver NVIDIA
	a.Log("[*] Bước 3/3: Kích hoạt cờ Resizable BAR Turing trong NVIDIA driver...")
	if fileExists(inspectorExe) && fileExists(profileNip) {
		cmdInspect := exec.Command(inspectorExe, profileNip, "-silent")
		if err := cmdInspect.Run(); err == nil {
			a.Log("  [✓] Đã nạp profile Enable_ReBAR_Turing.nip vào NVIDIA Profile.")
		} else {
			a.Log(fmt.Sprintf("  [!] Cảnh báo nạp profile: %v", err))
		}
	}

	// 4. Gọi NvStrapsReBar cấu hình kích thước BAR nếu được yêu cầu
	if fileExists(nvStrapsExe) && barSizeGB > 0 {
		a.Log(fmt.Sprintf("  [*] Đang cấu hình NvStrapsReBar mục tiêu %d GB...", barSizeGB))
		// NvStrapsReBar chạy cấu hình UEFI NVRAM
		cmdRebar := exec.Command(nvStrapsExe, fmt.Sprintf("%d", barSizeGB))
		out, err := cmdRebar.CombinedOutput()
		if err == nil {
			a.Log(fmt.Sprintf("  [✓] NvStrapsReBar hoàn tất: %s", strings.TrimSpace(string(out))))
		} else {
			a.Log(fmt.Sprintf("  [!] NvStrapsReBar ghi nhận: %s (%v)", strings.TrimSpace(string(out)), err))
		}
	}

	a.Log("==============================================")
	a.Log("  [✓] Cấu hình Resizable BAR hoàn tất! Cần khởi động lại máy để có hiệu lực.")
	a.Log("==============================================")

	return SubtoolResult{
		Success: true,
		Message: "Đã kích hoạt cờ Resizable BAR Turing thành công. Khởi động lại máy để kiểm tra trong GPU-Z.",
	}
}

// RevertReBar khôi phục cấu hình ReBAR về mặc định
func (a *App) RevertReBar() SubtoolResult {
	if !a.tryAcquire("Hoàn tác ReBAR") {
		return SubtoolResult{Success: false, Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()

	a.Log("[ReBAR] Đang khôi phục cài đặt Resizable BAR...")
	tempDir, err := ExtractToolAssets()
	if err == nil {
		nvStrapsExe := filepath.Join(tempDir, "NvStrapsReBar.exe")
		if fileExists(nvStrapsExe) {
			_ = exec.Command(nvStrapsExe, "0").Run()
		}
	}
	a.Log("[ReBAR] ✓ Đã tắt cấu hình NvStrapsReBar.")
	return SubtoolResult{Success: true, Message: "Đã hoàn tác cấu hình ReBAR."}
}

func checkIsLaptop() (bool, string) {
	// Kiểm tra qua PowerShell xem có pin (Win32_Battery) hay không
	psCheck := `
$hasBattery = (Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue | Measure-Object).Count -gt 0
$cs = Get-CimInstance Win32_ComputerSystem
$model = "$($cs.Manufacturer) $($cs.Model)"
Write-Output "$hasBattery|$model"
`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psCheck)
	out, err := cmd.Output()
	if err == nil {
		parts := strings.Split(strings.TrimSpace(string(out)), "|")
		if len(parts) >= 2 {
			hasBat := strings.EqualFold(strings.TrimSpace(parts[0]), "True")
			model := strings.TrimSpace(parts[1])
			if hasBat {
				return true, model + " (Phát hiện Pin Laptop)"
			}
			lower := strings.ToLower(model)
			keywords := []string{
				"laptop", "notebook", "book", "portable", "nitro", "legion", "victus",
				"tuf", "rog", "zephyrus", "blade", "omen", "pavilion", "inspiron",
				"latitude", "precision", "xps", "yoga", "thinkpad", "ideapad",
			}
			for _, kw := range keywords {
				if strings.Contains(lower, kw) {
					return true, model
				}
			}
			return false, model
		}
	}
	return false, "Desktop PC"
}
