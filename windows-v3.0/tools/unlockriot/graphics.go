package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// GraphicsFixResult chứa tổng hợp kết quả khắc phục lỗi thiết bị đồ họa
type GraphicsFixResult struct {
	ConfigsPatched   []string
	GpuPrefsAssigned []string
	HighPerfAdapter  string
	FriendlyNameSet  bool
	HAGSStatus       string
	Errors           []string
}

// RiotInstallsJSON biểu diễn cấu trúc của RiotClientInstalls.json
type RiotInstallsJSON struct {
	AssociatedClient map[string]interface{} `json:"associated_client"`
	RCDefault        string                 `json:"rc_default"`
	RCLive           string                 `json:"rc_live"`
	RCBeta           string                 `json:"rc_beta"`
	LeagueLive       string                 `json:"league_of_legends.live"`
	LeagueBeta       string                 `json:"league_of_legends.beta"`
	ValorantLive     string                 `json:"valorant.live"`
	ValorantBeta     string                 `json:"valorant.beta"`
}

// FindRiotGamePaths tìm kiếm tất cả các file thực thi và thư mục config của Riot Games
func FindRiotGamePaths() (executables []string, configPaths []string) {
	exeMap := make(map[string]bool)
	cfgMap := make(map[string]bool)

	addExe := func(p string) {
		p = filepath.Clean(p)
		if _, err := os.Stat(p); err == nil && !exeMap[strings.ToLower(p)] {
			exeMap[strings.ToLower(p)] = true
			executables = append(executables, p)
		}
	}

	addCfg := func(p string) {
		p = filepath.Clean(p)
		if !cfgMap[strings.ToLower(p)] {
			cfgMap[strings.ToLower(p)] = true
			configPaths = append(configPaths, p)
		}
	}

	// 1. Quét từ RiotClientInstalls.json trong ProgramData / AllUsersProfile
	jsonCandidates := []string{
		filepath.Join(os.Getenv("ALLUSERSPROFILE"), `Riot Games\RiotClientInstalls.json`),
		filepath.Join(os.Getenv("ProgramData"), `Riot Games\RiotClientInstalls.json`),
		`C:\ProgramData\Riot Games\RiotClientInstalls.json`,
	}

	for _, jPath := range jsonCandidates {
		if jPath == "" {
			continue
		}
		data, err := os.ReadFile(jPath)
		if err == nil {
			var installs RiotInstallsJSON
			if err := json.Unmarshal(data, &installs); err == nil {
				if installs.RCLive != "" {
					addExe(installs.RCLive)
				}
				if installs.LeagueLive != "" {
					addExe(filepath.Join(installs.LeagueLive, `Game\League of Legends.exe`))
					addExe(filepath.Join(installs.LeagueLive, `LeagueClient.exe`))
					addExe(filepath.Join(installs.LeagueLive, `LeagueClientUx.exe`))
					addCfg(filepath.Join(installs.LeagueLive, `Config\game.cfg`))
				}
				if installs.ValorantLive != "" {
					addExe(filepath.Join(installs.ValorantLive, `ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`))
					addExe(filepath.Join(installs.ValorantLive, `VALORANT.exe`))
				}
			}
		}
	}

	// 2. Quét Registry Uninstall keys cho Riot / League / VNG
	regRoots := []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER}
	regSubPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\Riot Games, Inc\League of Legends`,
	}

	for _, root := range regRoots {
		for _, sub := range regSubPaths {
			k, err := registry.OpenKey(root, sub, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			names, _ := k.ReadSubKeyNames(-1)
			for _, name := range names {
				sk, err := registry.OpenKey(k, name, registry.QUERY_VALUE)
				if err != nil {
					continue
				}
				loc, _, _ := sk.GetStringValue("InstallLocation")
				disp, _, _ := sk.GetStringValue("DisplayName")
				ico, _, _ := sk.GetStringValue("DisplayIcon")
				sk.Close()

				if strings.Contains(strings.ToLower(disp), "league of legends") ||
					strings.Contains(strings.ToLower(disp), "liên minh huyền thoại") ||
					strings.Contains(strings.ToLower(disp), "valorant") ||
					strings.Contains(strings.ToLower(disp), "riot") {
					if loc != "" {
						addExe(filepath.Join(loc, `Game\League of Legends.exe`))
						addExe(filepath.Join(loc, `LeagueClient.exe`))
						addExe(filepath.Join(loc, `ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`))
						addExe(filepath.Join(loc, `VALORANT.exe`))
						addCfg(filepath.Join(loc, `Config\game.cfg`))
					}
					if ico != "" && strings.HasSuffix(strings.ToLower(ico), ".exe") {
						addExe(ico)
					}
				}
			}
			k.Close()
		}
	}

	// 3. Quét các đường dẫn thông dụng trên toàn bộ ổ đĩa logic
	drives := getLogicalDrives()
	subDirs := []string{
		`Riot Games\League of Legends`,
		`Riot Games\VALORANT\live`,
		`Riot Games\Riot Client`,
		`Games\League of Legends`,
		`Games\Lien Minh Huyen Thoai`,
		`LienMinhHuyenThoai`,
		`VNGGames\League of Legends`,
		`Program Files\Riot Games\League of Legends`,
		`Program Files (x86)\Riot Games\League of Legends`,
	}

	for _, d := range drives {
		for _, s := range subDirs {
			baseDir := filepath.Join(d, s)
			addExe(filepath.Join(baseDir, `Game\League of Legends.exe`))
			addExe(filepath.Join(baseDir, `LeagueClient.exe`))
			addExe(filepath.Join(baseDir, `LeagueClientUx.exe`))
			addExe(filepath.Join(baseDir, `ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`))
			addExe(filepath.Join(baseDir, `VALORANT.exe`))
			addExe(filepath.Join(baseDir, `RiotClientServices.exe`))

			cfgFile := filepath.Join(baseDir, `Config\game.cfg`)
			if _, err := os.Stat(cfgFile); err == nil {
				addCfg(cfgFile)
			}
		}
	}

	return executables, configPaths
}

// PatchLeagueGameConfigContent cập nhật nội dung game.cfg: WindowMode=2 (Borderless), PreferDX9Legacy=0
func PatchLeagueGameConfigContent(content string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	inGeneral := false
	generalFound := false
	windowModeFound := false
	borderlessFound := false
	dx9Found := false
	modified := false

	var newLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			secName := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			if strings.EqualFold(secName, "General") {
				inGeneral = true
				generalFound = true
			} else {
				if inGeneral {
					// Đóng General section, chèn các key còn thiếu
					if !windowModeFound {
						newLines = append(newLines, "WindowMode=2")
						modified = true
						windowModeFound = true
					}
					if !borderlessFound {
						newLines = append(newLines, "BorderlessWindow=1")
						modified = true
						borderlessFound = true
					}
					if !dx9Found {
						newLines = append(newLines, "PreferDX9Legacy=0")
						modified = true
						dx9Found = true
					}
				}
				inGeneral = false
			}
			newLines = append(newLines, line)
			continue
		}

		if inGeneral {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])

				if strings.EqualFold(k, "WindowMode") {
					windowModeFound = true
					if v != "2" {
						newLines = append(newLines, "WindowMode=2")
						modified = true
						continue
					}
				} else if strings.EqualFold(k, "BorderlessWindow") {
					borderlessFound = true
					if v != "1" {
						newLines = append(newLines, "BorderlessWindow=1")
						modified = true
						continue
					}
				} else if strings.EqualFold(k, "PreferDX9Legacy") {
					dx9Found = true
					if v != "0" {
						newLines = append(newLines, "PreferDX9Legacy=0")
						modified = true
						continue
					}
				}
			}
		}

		newLines = append(newLines, line)
	}

	if !generalFound {
		newLines = append(newLines, "[General]", "WindowMode=2", "BorderlessWindow=1", "PreferDX9Legacy=0", "Width=1920", "Height=1080")
		modified = true
	} else if inGeneral {
		if !windowModeFound {
			newLines = append(newLines, "WindowMode=2")
			modified = true
		}
		if !borderlessFound {
			newLines = append(newLines, "BorderlessWindow=1")
			modified = true
		}
		if !dx9Found {
			newLines = append(newLines, "PreferDX9Legacy=0")
			modified = true
		}
	}

	return strings.Join(newLines, "\r\n"), modified
}

// FixLeagueGameConfigFile đọc, áp dụng bản vá và lưu lại game.cfg
func FixLeagueGameConfigFile(cfgPath string) (bool, error) {
	var content string
	if _, err := os.Stat(cfgPath); err == nil {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false, err
		}
		content = string(data)
	}

	patched, modified := PatchLeagueGameConfigContent(content)
	if !modified && content != "" {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return false, err
	}

	if err := os.WriteFile(cfgPath, []byte(patched), 0644); err != nil {
		return false, err
	}

	return true, nil
}

// FormatHighPerfAdapterString trả về chuỗi định danh Adapter cho DirectXUserGlobalSettings
func FormatHighPerfAdapterString(vendorID, deviceID uint16) string {
	return fmt.Sprintf("HighPerfAdapter=%04X&%04X&00000000;SwapEffectUpgradeEnable=0;", vendorID, deviceID)
}

// ConfigureDirectXUserGpuPreferences gán GpuPreference=2 và HighPerfAdapter vào Registry
func ConfigureDirectXUserGpuPreferences(log io.Writer, executables []string, targetDeviceID uint16) (int, string, error) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\DirectX\UserGpuPreferences`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return 0, "", fmt.Errorf("không thể mở Registry UserGpuPreferences: %w", err)
	}
	defer k.Close()

	count := 0
	for _, exe := range executables {
		if err := k.SetStringValue(exe, "GpuPreference=2;"); err == nil {
			count++
			if log != nil {
				fmt.Fprintf(log, "  [+] Đã cấu hình GpuPreference=2 (High Performance): %s\n", exe)
			}
		}
	}

	var highPerfStr string
	if targetDeviceID != 0 {
		highPerfStr = FormatHighPerfAdapterString(0x10DE, targetDeviceID)
		if err := k.SetStringValue("DirectXUserGlobalSettings", highPerfStr); err == nil {
			if log != nil {
				fmt.Fprintf(log, "  [+] Đã thiết lập DirectXUserGlobalSettings: %s\n", highPerfStr)
			}
		}
	}

	return count, highPerfStr, nil
}

// SetGpuFriendlyName cấu hình FriendlyName cho CMP GPU trong HKLM\SYSTEM\CurrentControlSet\Enum\PCI
func SetGpuFriendlyName(log io.Writer, deviceID uint16, friendlyName string) (bool, error) {
	if deviceID == 0 {
		deviceID = DeviceIDCMP40HX
	}
	if friendlyName == "" {
		if deviceID == DeviceIDCMP30HX {
			friendlyName = "NVIDIA GeForce GTX 1660 SUPER"
		} else {
			friendlyName = "NVIDIA GeForce RTX 2060 SUPER"
		}
	}

	baseKey := `SYSTEM\CurrentControlSet\Enum\PCI`
	base, err := registry.OpenKey(registry.LOCAL_MACHINE, baseKey, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return false, fmt.Errorf("không thể mở Enum\\PCI: %w", err)
	}
	defer base.Close()

	devs, err := base.ReadSubKeyNames(-1)
	if err != nil {
		return false, err
	}

	venDevPattern := fmt.Sprintf("VEN_10DE&DEV_%04X", deviceID)
	found := false

	for _, dev := range devs {
		if strings.Contains(strings.ToUpper(dev), venDevPattern) {
			subPath := baseKey + `\` + dev
			dk, err := registry.OpenKey(registry.LOCAL_MACHINE, subPath, registry.ENUMERATE_SUB_KEYS)
			if err != nil {
				continue
			}
			insts, _ := dk.ReadSubKeyNames(-1)
			dk.Close()

			for _, inst := range insts {
				instPath := subPath + `\` + inst
				ik, err := registry.OpenKey(registry.LOCAL_MACHINE, instPath, registry.SET_VALUE|registry.QUERY_VALUE)
				if err != nil {
					continue
				}
				_ = ik.SetStringValue("FriendlyName", friendlyName)
				ik.Close()
				found = true
				if log != nil {
					fmt.Fprintf(log, "  [+] Đã gán FriendlyName = '%s' tại: %s\n", friendlyName, instPath)
				}
			}
		}
	}

	return found, nil
}

// CheckHAGSStatus kiểm tra chế độ Lập lịch GPU tăng tốc phần cứng (HAGS)
func CheckHAGSStatus() (string, int) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, registry.QUERY_VALUE)
	if err != nil {
		return "Không rõ", 0
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("HwSchMode")
	if err != nil {
		return "Chưa cấu hình (Mặc định)", 0
	}
	if val == 2 {
		return "Đang BẬT (Enabled)", 2
	}
	if val == 1 {
		return "Đã TẮT (Disabled)", 1
	}
	return fmt.Sprintf("Giá trị: %d", val), int(val)
}

// FixRiotGraphicsDeviceError chạy toàn diện chu trình sửa lỗi không khởi động được thiết bị đồ họa
func FixRiotGraphicsDeviceError(log io.Writer, is40HX, is30HX bool) *GraphicsFixResult {
	res := &GraphicsFixResult{}

	fmt.Fprintln(log, "==========================================================")
	fmt.Fprintln(log, "  KHẮC PHỤC LỖI 'KHÔNG KHỞI ĐỘNG ĐƯỢC THIẾT BỊ ĐỒ HOẠ'")
	fmt.Fprintln(log, "  (Dành cho GPU CMP 40HX / 30HX & Game Riot: LMHT, Valorant)")
	fmt.Fprintln(log, "==========================================================")

	targetDeviceID := uint16(DeviceIDCMP40HX)
	if is30HX {
		targetDeviceID = uint16(DeviceIDCMP30HX)
	}

	// BƯỚC 1: Quét tìm tất cả đường dẫn cài đặt
	fmt.Fprintln(log, "\n[*] BƯỚC 1: Tự động quét vị trí cài đặt Riot Games & Liên Minh Huyền Thoại...")
	executables, configs := FindRiotGamePaths()

	if len(executables) == 0 {
		fmt.Fprintln(log, "  [!] Chưa tìm thấy game qua quét tự động, bổ sung đường dẫn mặc định...")
		sysDrive := os.Getenv("SystemDrive")
		if sysDrive == "" {
			sysDrive = "C:"
		}
		executables = append(executables,
			filepath.Join(sysDrive, `Riot Games\League of Legends\Game\League of Legends.exe`),
			filepath.Join(sysDrive, `Riot Games\League of Legends\LeagueClient.exe`),
			filepath.Join(sysDrive, `Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`),
		)
		configs = append(configs, filepath.Join(sysDrive, `Riot Games\League of Legends\Config\game.cfg`))
	} else {
		fmt.Fprintf(log, "  [V] Đã phát hiện %d file thực thi và %d file cấu hình Riot.\n", len(executables), len(configs))
	}

	// BƯỚC 2: Sửa file cấu hình game.cfg sang Borderless Windowed (Không viền)
	fmt.Fprintln(log, "\n[*] BƯỚC 2: Tối ưu hóa file cấu hình game.cfg (Tránh lỗi Fullscreen không cổng xuất hình)...")
	for _, cfg := range configs {
		patched, err := FixLeagueGameConfigFile(cfg)
		if err != nil {
			errStr := fmt.Sprintf("Không thể ghi file %s: %v", cfg, err)
			res.Errors = append(res.Errors, errStr)
			fmt.Fprintf(log, "  [!] %s\n", errStr)
		} else if patched {
			res.ConfigsPatched = append(res.ConfigsPatched, cfg)
			fmt.Fprintf(log, "  [V] Đã chuyển thành công sang BORDERLESS WINDOWED (WindowMode=2): %s\n", cfg)
		} else {
			fmt.Fprintf(log, "  [i] File cấu hình đã ở chế độ Borderless tối ưu: %s\n", cfg)
		}
	}

	// BƯỚC 3: Cấu hình Registry DirectX UserGpuPreferences
	fmt.Fprintln(log, "\n[*] BƯỚC 3: Cấu hình Registry Windows High Performance GPU...")
	count, highPerfStr, err := ConfigureDirectXUserGpuPreferences(log, executables, targetDeviceID)
	if err != nil {
		res.Errors = append(res.Errors, err.Error())
		fmt.Fprintf(log, "  [!] Lỗi Registry DirectX: %v\n", err)
	} else {
		res.GpuPrefsAssigned = executables
		res.HighPerfAdapter = highPerfStr
		fmt.Fprintf(log, "  [V] Đã gán thành công GPU hiệu năng cao cho %d file thực thi.\n", count)
	}

	// BƯỚC 4: Gán FriendlyName GeForce RTX cho CMP
	fmt.Fprintln(log, "\n[*] BƯỚC 4: Kiểm tra và định danh FriendlyName GeForce RTX...")
	fnSuccess, _ := SetGpuFriendlyName(log, targetDeviceID, "")
	res.FriendlyNameSet = fnSuccess
	if fnSuccess {
		fmt.Fprintln(log, "  [V] Đã định danh card như một GPU GeForce RTX tiêu chuẩn.")
	} else {
		fmt.Fprintln(log, "  [i] Không phát hiện card CMP cắm trực tiếp trên máy này (hoặc đã có sẵn tên).")
	}

	// BƯỚC 5: Chẩn đoán HAGS và hướng dẫn iGPU
	fmt.Fprintln(log, "\n[*] BƯỚC 5: Chẩn đoán cấu hình đồ họa hệ thống...")
	hagsText, _ := CheckHAGSStatus()
	res.HAGSStatus = hagsText
	fmt.Fprintf(log, "  -> Trạng thái HAGS (Hardware-Accelerated GPU Scheduling): %s\n", hagsText)
	fmt.Fprintln(log, "  💡 LƯU Ý QUAN TRỌNG ĐỂ KHÔNG BỊ LỖI THIẾT BỊ ĐỒ HỌA:")
	fmt.Fprintln(log, "     1. Trong BIOS: Đảm bảo bộ nhớ iGPU (DVMT Pre-Allocated) đặt từ 128MB - 512MB.")
	fmt.Fprintln(log, "     2. Trong Game: Giữ nguyên chế độ hiển thị 'Không viền' (Borderless), KHÔNG chọn 'Toàn màn hình'.")
	fmt.Fprintln(log, "     3. Dây màn hình: Bắt buộc cắm vào cổng xuất hình của iGPU (Mainboard) hoặc card phụ.")

	fmt.Fprintln(log, "\n==========================================================")
	fmt.Fprintln(log, "HOÀN TẤT XỬ LÝ LỖI ĐỒ HỌA!")
	fmt.Fprintln(log, "Hãy khởi động lại Riot Client và vào trận thử nghiệm (Phòng Tập).")
	fmt.Fprintln(log, "==========================================================")

	return res
}
