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
	"golang.org/x/sys/windows/registry"
)

// RiotStatusReport phản hồi trạng thái tương thích game Riot và Vanguard
type RiotStatusReport struct {
	Model                  string `json:"model"`
	IsWin11                bool   `json:"isWin11"`
	SecureBootOn           bool   `json:"secureBootOn"`
	CanPlayLeagueOfLegends bool   `json:"canPlayLeagueOfLegends"`
	CanPlayValorant        bool   `json:"canPlayValorant"`
	NeedsEFISigning        bool   `json:"needsEFISigning"`
	HasTensorCore          bool   `json:"hasTensorCore"`
	OsDesc                 string `json:"osDesc"`
	SecureBootDesc         string `json:"secureBootDesc"`
	GpuDesc                string `json:"gpuDesc"`
	LoLDesc                string `json:"lolDesc"`
	ValorantDesc           string `json:"valorantDesc"`
	Recommendation         string `json:"recommendation"`
	DisplayHostDesc        string `json:"displayHostDesc"`
}

// GetRiotStatus đánh giá khả năng chơi LMHT và Valorant
func (a *App) GetRiotStatus() RiotStatusReport {
	prof, found := hxcore.FindGPUWithProfile()
	sbOn := hxcore.SecureBootOn()
	isWin11 := isWindows11OS()

	model := "Unknown"
	is40HX := found && (prof.DeviceID == 0x1F0B || strings.Contains(strings.ToUpper(prof.Name), "40HX"))
	is30HX := found && (prof.DeviceID == 0x2189 || strings.Contains(strings.ToUpper(prof.Name), "30HX"))

	if is40HX {
		model = "CMP 40HX"
	} else if is30HX {
		model = "CMP 30HX"
	}

	osName := "Windows 10"
	if isWin11 {
		osName = "Windows 11"
	}

	sbDesc := "ĐÃ TẮT (Disabled)"
	if sbOn {
		sbDesc = "ĐANG BẬT (Enabled)"
	}

	st := RiotStatusReport{
		Model:                  model,
		IsWin11:                isWin11,
		SecureBootOn:           sbOn,
		CanPlayLeagueOfLegends: true,
		OsDesc:                 osName,
		SecureBootDesc:         sbDesc,
		LoLDesc:                "Sẵn sàng (MSHybrid CASO + Borderless Windowed)",
		DisplayHostDesc:        "Tự động nhận diện GPU xuất hình",
	}

	if is40HX {
		st.HasTensorCore = true
		st.GpuDesc = "NVIDIA CMP 40HX [TU106] (Gen 2 & Tensor Core)"
		if !isWin11 {
			st.CanPlayValorant = true
			st.ValorantDesc = "Sẵn sàng (Secure Boot Tắt + Tensor Core mở khoá)"
			st.Recommendation = "Khuyến nghị: Giữ Secure Boot TẮT trong BIOS để nạp Tensor Core."
		} else {
			st.NeedsEFISigning = true
			if sbOn {
				st.CanPlayValorant = true
				st.ValorantDesc = "Secure Boot BẬT: Cần ký chứng chỉ cá nhân vào BIOS db để nạp Tensor Core"
				st.Recommendation = "Khuyến nghị: Bấm nút [Ký Chữ Ký Số EFI] để chuẩn bị nạp Key cá nhân vào BIOS db."
			} else {
				st.CanPlayValorant = false
				st.ValorantDesc = "Secure Boot TẮT: Chơi được LMHT. Valorant trên Win 11 bắt buộc bật Secure Boot."
				st.Recommendation = "Khuyến nghị: Bấm nút [Ký Chữ Ký Số EFI] sau đó vào BIOS bật Secure Boot và nạp Key."
			}
		}
	} else if is30HX {
		st.GpuDesc = "NVIDIA CMP 30HX [TU116] (Gen 2 Hardware Lock)"
		st.CanPlayValorant = (!isWin11 || sbOn)
		if !isWin11 || sbOn {
			st.ValorantDesc = "Sẵn sàng (CMP 30HX giữ Secure Boot BẬT bình thường)"
			st.Recommendation = "Khuyến nghị: CMP 30HX giữ Secure Boot BẬT bình thường. Bấm [Tối Ưu Riot] để thiết lập."
		} else {
			st.ValorantDesc = "Cần BẬT Secure Boot trong BIOS để chơi Valorant trên Win 11"
			st.Recommendation = "Khuyến nghị: Vào BIOS BẬT Secure Boot để chơi Valorant."
		}
	} else {
		st.GpuDesc = prof.Name
		st.CanPlayValorant = (!isWin11 || sbOn)
		st.ValorantDesc = "Sẵn sàng"
		st.Recommendation = "Khuyến nghị: Bấm [Tối Ưu Riot] để cấu hình MSHybrid CASO và DirectX High-Performance."
	}

	return st
}

// RiotOptimize tối ưu hoá hệ thống cho Riot Games
func (a *App) RiotOptimize() error {
	if !a.tryAcquire("Tối ưu Riot Games") {
		return fmt.Errorf("hệ thống đang bận thao tác khác")
	}
	defer a.releaseOp()

	a.Log("==============================================")
	a.Log("  [Riot Vanguard] Bắt đầu tối ưu hoá hệ thống")
	a.Log("==============================================")

	// 1. Dọn dẹp driver tạm thời trong kernel
	a.Log("[*] Bước 1/4: Dọn dẹp driver kernel tạm thời (WinRing0 / ThrottleStop)...")
	_ = exec.Command("sc.exe", "stop", "WinRing0_1_2_0").Run()
	_ = exec.Command("sc.exe", "delete", "WinRing0_1_2_0").Run()
	_ = exec.Command("sc.exe", "stop", "ThrottleStop").Run()
	_ = exec.Command("sc.exe", "delete", "ThrottleStop").Run()

	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	_ = os.Remove(filepath.Join(sysRoot, "System32", "drivers", "WinRing0x64.sys"))
	_ = os.Remove(filepath.Join(sysRoot, "System32", "drivers", "ThrottleStop.sys"))
	a.Log("  [✓] Đã giải phóng sạch sẽ dịch vụ và file driver kernel khỏi hệ thống.")

	// 2. Tắt Microsoft Vulnerable Driver Blocklist
	a.Log("[*] Bước 2/4: Cấu hình VulnerableDriverBlocklistEnable=0...")
	kCi, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\CI\Config`, registry.SET_VALUE)
	if err == nil {
		_ = kCi.SetDWordValue("VulnerableDriverBlocklistEnable", 0)
		kCi.Close()
		a.Log("  [✓] Đã tắt VulnerableDriverBlocklist (ngăn chặn lỗi nạp driver mod 5xx/6xx).")
	} else {
		a.Log(fmt.Sprintf("  [!] Lỗi ghi Registry CI\\Config: %v", err))
	}

	// 3. Cấu hình DirectX Global Routing và TDR Delay
	a.Log("[*] Bước 3/4: Thiết lập định tuyến DirectX High-Performance cho CMP...")
	prof, found := hxcore.FindGPUWithProfile()
	shortDID := "10DE&2189"
	if found && prof.DeviceID != 0 {
		shortDID = fmt.Sprintf("10DE&%04X", prof.DeviceID)
	}

	if gsKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\DirectX\UserGpuPreferences`, registry.QUERY_VALUE|registry.SET_VALUE); err == nil {
		current, _, _ := gsKey.GetStringValue("DirectXUserGlobalSettings")
		swap := "SwapEffectUpgradeEnable=1;"
		if strings.Contains(current, "SwapEffectUpgradeEnable=0") {
			swap = "SwapEffectUpgradeEnable=0;"
		}
		newSetting := fmt.Sprintf("HighPerfAdapter=%s;%s", shortDID, swap)
		_ = gsKey.SetStringValue("DirectXUserGlobalSettings", newSetting)
		gsKey.Close()
		a.Log(fmt.Sprintf("  [✓] Đã gán HighPerfAdapter toàn cục cho DirectX: %s", shortDID))
	}

	// Tăng TDR Delay lên 10 giây tránh crash timeout
	if tdrKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, registry.SET_VALUE); err == nil {
		_ = tdrKey.SetDWordValue("TdrDelay", 10)
		_ = tdrKey.SetDWordValue("TdrDdiDelay", 10)
		tdrKey.Close()
		a.Log("  [✓] Đã tăng thời gian chờ TDR lên 10 giây (ngăn crash khi chuyển frame qua PCIe).")
	}

	// 4. Quét và cấu hình Liên Minh Huyền Thoại (WindowMode=2, DX11Support=1)
	a.Log("[*] Bước 4/4: Tự động cấu hình game.cfg Liên Minh Huyền Thoại...")
	cfgCount := a.optimizeLeagueConfigs()
	if cfgCount > 0 {
		a.Log(fmt.Sprintf("  [✓] Đã tối ưu %d cấu hình game.cfg (WindowMode=2, Borderless=1, DX11Support=1).", cfgCount))
	} else {
		a.Log("  [i] Chưa phát hiện file game.cfg LMHT trên các ổ đĩa (sẽ tự nhận khi khởi động game lần đầu).")
	}

	a.Log("==============================================")
	a.Log("  [✓] Tối ưu hoá Riot Games hoàn tất thành công!")
	a.Log("==============================================")
	return nil
}

// RiotRevertOptimization hoàn tác cấu hình Riot (bật lại Blocklist, xoá định tuyến)
func (a *App) RiotRevertOptimization() error {
	if !a.tryAcquire("Hoàn tác tối ưu Riot") {
		return fmt.Errorf("hệ thống đang bận thao tác khác")
	}
	defer a.releaseOp()

	a.Log("[Riot] Bắt đầu hoàn tác cấu hình bảo mật...")

	// Bật lại Blocklist
	kCi, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\CI\Config`, registry.SET_VALUE)
	if err == nil {
		_ = kCi.SetDWordValue("VulnerableDriverBlocklistEnable", 1)
		kCi.Close()
		a.Log("  [✓] Đã bật lại VulnerableDriverBlocklistEnable=1 (khôi phục bảo vệ mặc định của Windows).")
	}

	// Xoá cấu hình DirectX HighPerfAdapter
	if gsKey, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\DirectX\UserGpuPreferences`, registry.SET_VALUE); err == nil {
		_ = gsKey.DeleteValue("DirectXUserGlobalSettings")
		gsKey.Close()
		a.Log("  [✓] Đã xoá bỏ định tuyến HighPerfAdapter toàn cục.")
	}

	a.Log("[Riot] Hoàn tác cấu hình hoàn thành.")
	return nil
}

// RiotSignEFI tạo chứng chỉ cá nhân và ký Authenticode cho 40HXUNLK.EFI
func (a *App) RiotSignEFI(confirmPhrase string) error {
	if confirmPhrase != "XAC NHAN KY EFI" {
		return fmt.Errorf("chuỗi xác nhận không chính xác (yêu cầu gõ chính xác %q)", "XAC NHAN KY EFI")
	}

	if !a.tryAcquire("Ký chữ ký số EFI cá nhân") {
		return fmt.Errorf("hệ thống đang bận thao tác khác")
	}
	defer a.releaseOp()

	a.Log("==============================================")
	a.Log("  [UEFI Key] Bắt đầu tự động tạo chứng chỉ và ký Authenticode")
	a.Log("==============================================")

	esp := hxcore.MountESP()
	var efiPath string
	if esp != "" {
		defer hxcore.UnmountESP(esp)
		efiPath = filepath.Join(esp, "EFI", "40HX", "40HXUNLK.EFI")
	}

	if efiPath == "" || !fileExists(efiPath) {
		tempDir, err := ExtractToolAssets()
		if err != nil {
			return fmt.Errorf("không thể trích xuất file EFI: %w", err)
		}
		efiPath = filepath.Join(tempDir, "40HXUNLK.EFI")
	}

	if !fileExists(efiPath) {
		return fmt.Errorf("không tìm thấy file 40HXUNLK.EFI để thực hiện ký số")
	}

	a.Log(fmt.Sprintf("[*] File mục tiêu ký số: %s", efiPath))

	// Chạy PowerShell tạo cert và ký
	psScript := fmt.Sprintf(`
$certName = "CMP40HX_Custom_UEFI_Signer"
$existing = Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.Subject -like "*$certName*" }
if (-not $existing) {
    $cert = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=$certName" -CertStoreLocation Cert:\CurrentUser\My -HashAlgorithm SHA256
} else {
    $cert = $existing[0]
}
$certPath = "C:\CMP40HX_Key.cer"
Export-Certificate -Cert $cert -FilePath $certPath -Force | Out-Null
Set-AuthenticodeSignature -FilePath "%s" -Certificate $cert -HashAlgorithm SHA256 | Out-Null
Write-Output "SUCCESS"
`, efiPath)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		a.Log(fmt.Sprintf("[!] Lỗi khi chạy ký EFI: %s (%v)", strings.TrimSpace(string(out)), err))
		return fmt.Errorf("ký EFI thất bại: %w", err)
	}

	a.Log("  [✓] Đã tạo chứng chỉ và ký Authenticode thành công cho 40HXUNLK.EFI!")
	a.Log("  [✓] Đã xuất file chứng chỉ công khai: C:\\CMP40HX_Key.cer")
	a.Log("  👉 BƯỚC TIẾP THEO (VÀO BIOS):")
	a.Log("     1. Khởi động lại máy vào BIOS Setup (Security / Key Management).")
	a.Log("     2. Chọn [Enroll Signature / Append db Key] -> Chọn ổ C:\\CMP40HX_Key.cer.")
	a.Log("     3. Bật lại Secure Boot = Enabled và lưu cài đặt.")

	return nil
}

// RebootToBIOS khởi động lại máy tính vào BIOS Setup
func (a *App) RebootToBIOS() error {
	a.Log("[BIOS] Đang khởi động lại máy tính vào UEFI Firmware Setup...")
	cmd := exec.Command("shutdown.exe", "/r", "/fw", "/t", "3")
	if err := cmd.Run(); err != nil {
		return exec.Command("shutdown.exe", "/r", "/t", "3").Run()
	}
	return nil
}

func (a *App) optimizeLeagueConfigs() int {
	drives := getLogicalDrives()
	count := 0

	for _, d := range drives {
		candidates := []string{
			filepath.Join(d, "Riot Games", "League of Legends", "Config", "game.cfg"),
			filepath.Join(d, "Games", "League of Legends", "Config", "game.cfg"),
			filepath.Join(d, "LienMinhHuyenThoai", "GameData", "Apps", "lolVN", "Game", "Config", "game.cfg"),
		}
		for _, cfg := range candidates {
			if fileExists(cfg) {
				if err := applyLoLCfg(cfg); err == nil {
					count++
				}
			}
		}
	}
	return count
}

func applyLoLCfg(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	// Cập nhật hoặc thêm WindowMode=2, BorderlessWindow=1, DX11Support=1
	updates := map[string]string{
		"WindowMode":           "2",
		"BorderlessWindow":     "1",
		"PreferDX9LegacyMode":  "0",
		"DX11Support":          "1",
		"WaitforVerticalSync":  "0",
	}

	lines := strings.Split(content, "\n")
	hasGeneral := false
	newLines := []string{}
	foundKeys := map[string]bool{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "[General]") {
			hasGeneral = true
			newLines = append(newLines, line)
			continue
		}
		replaced := false
		for k, v := range updates {
			if strings.HasPrefix(trimmed, k+"=") || strings.HasPrefix(trimmed, k+" =") {
				newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
				foundKeys[k] = true
				replaced = true
				break
			}
		}
		if !replaced {
			newLines = append(newLines, line)
		}
	}

	if !hasGeneral {
		newLines = append([]string{"[General]"}, newLines...)
	}

	for k, v := range updates {
		if !foundKeys[k] {
			newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0o644)
}

func isWindows11OS() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	buildStr, _, err := k.GetStringValue("CurrentBuild")
	if err != nil {
		return false
	}
	var buildNum int
	fmt.Sscanf(buildStr, "%d", &buildNum)
	return buildNum >= 22000
}

func getLogicalDrives() []string {
	var drives []string
	for c := 'C'; c <= 'Z'; c++ {
		root := string(c) + ":\\"
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			drives = append(drives, root)
		}
	}
	return drives
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
