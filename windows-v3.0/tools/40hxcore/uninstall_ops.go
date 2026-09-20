package hxcore

// v2.6.0: 卸载操作上提 40hxcore — GUI 组件级卸载与 40HXUninstaller.exe 共用同一实现,
// 消除两份漂移副本(原 uninstall40x 私有函数)。
// 全部为破坏性操作, 调用方(GUI 卸载页/卸载器)负责确认与提权。
// 电源设置(快速启动/ASPM)属用户偏好, 刻意不提供回滚操作 — 恢复方法见 README。

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

const bootDesc40 = "40HX Unlock"

// UninstallTaskNames: 本工具历史上用过的全部计划任务名(含 40HXGen2Retry 重试任务)
var UninstallTaskNames = []string{"40HXGen2", "40HX PCIe Gen2 Bring-up", "40HXGen2Retry", "40HXGspEnsure"}

// UninstallTasks: 删除计划任务, 返回实际删掉的名字
func UninstallTasks() []string {
	var removed []string
	for _, tn := range UninstallTaskNames {
		out, err := RunOut("schtasks.exe", "/delete", "/tn", tn, "/f")
		if err == nil || strings.Contains(out, "成功") || strings.Contains(strings.ToLower(out), "success") {
			fmt.Printf("  Đã xoá tác vụ lịch trình %s\n", tn)
			removed = append(removed, tn)
		}
	}
	return removed
}

// UninstallRunKey: 删 HKCU Run 值 40HXGen2
func UninstallRunKey() {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err == nil {
		k.DeleteValue("40HXGen2")
		k.Close()
	}
}

// UninstallBootEntry: 删固件启动项 '40HX Unlock', 返回是否删除过
func UninstallBootEntry() bool {
	out, err := RunOut("bcdedit.exe", "/enum", "firmware")
	if err != nil {
		return false
	}
	curGuid := ""
	re := regexp.MustCompile(`\{([0-9a-fA-F-]{36})\}`)
	removed := false
	for _, ln := range strings.Split(out, "\n") {
		if m := re.FindStringSubmatch(ln); len(m) > 1 {
			if strings.Contains(ln, "{") && !strings.Contains(ln, "displayorder") &&
				!strings.Contains(ln, "bootsequence") {
				curGuid = m[1]
			}
		}
		if strings.Contains(ln, bootDesc40) && curGuid != "" {
			RunOut("bcdedit.exe", "/delete", "{"+curGuid+"}", "/f")
			fmt.Printf("  Đã xoá mục khởi động %s\n", curGuid)
			removed = true
			curGuid = ""
		}
	}
	return removed
}

// UninstallEspEfi: 删 \EFI\40HX\40HXUNLK.EFI; bootx64.efi 用"覆盖写+校验"从
// .40hx.bak 还原(不用 删→rename — 中间窗口会让机器起不来)。返回是否动过 ESP。
func UninstallEspEfi() bool {
	esp := MountESP()
	if esp == "" {
		return false
	}
	defer RunOut("mountvol.exe", esp+":", "/D")
	removed := false
	target := esp + ":\\EFI\\40HX\\40HXUNLK.EFI"
	if err := os.Remove(target); err == nil {
		removed = true
	}
	if entries, err := os.ReadDir(esp + ":\\EFI\\40HX"); err == nil && len(entries) == 0 {
		os.Remove(esp + ":\\EFI\\40HX")
	}
	// v3.0.0: EFI 运行时写的历史日志一并清除 — 否则卸载后 40HXCheck 会把它
	// 当本次日志分析, 给没装 EFI 的用户派无关引导。
	if err := os.Remove(esp + ":\\40hx_log.txt"); err == nil {
		fmt.Println("    Đã xoá nhật ký lịch sử EFI 40hx_log.txt")
	}
	std := esp + ":\\EFI\\Boot\\bootx64.efi"
	bak := esp + ":\\EFI\\Boot\\bootx64.efi.40hx.bak"
	if data, berr := os.ReadFile(bak); berr == nil {
		if werr := os.WriteFile(std, data, 0o644); werr != nil {
			fmt.Println("  [!] Khôi phục bootx64.efi ghi thất bại:", werr)
			fmt.Println("      Bản sao lưu gốc vẫn giữ tại bootx64.efi.40hx.bak, có thể khôi phục thủ công")
			return removed
		}
		if rb, rerr := os.ReadFile(std); rerr == nil && len(rb) == len(data) {
			os.Remove(bak)
			fmt.Println("    Đã khôi phục bootx64.efi gốc (từ .40hx.bak, kiểm tra OK)")
		} else {
			fmt.Println("  [!] bootx64.efi sau khi khôi phục kiểm tra không khớp — Giữ lại .bak để xử lý thủ công")
		}
		removed = true
	}
	return removed
}

// UninstallDriverServices: 停止并删除历史驱动服务(v2.5 BYOVD + 旧版 bridge/early)
func UninstallDriverServices() {
	for _, name := range []string{"ThrottleStop", "40hx_bridge", "40hx_early", "40hx_early-d", "WinRing0_1_2_0", "WinRing0x64", "WinRing0"} {
		RunOut("sc.exe", "stop", name)
		time.Sleep(300 * time.Millisecond)
		out, err := RunOut("sc.exe", "delete", name)
		switch {
		case err == nil || strings.Contains(strings.ToLower(out), "success") || strings.Contains(out, "成功"):
			fmt.Printf("  Dịch vụ %s đã được xoá\n", name)
		case strings.Contains(out, "不存在") || strings.Contains(strings.ToLower(out), "not") || strings.Contains(out, "1060"):
			fmt.Printf("  Dịch vụ %s không tồn tại (bỏ qua)\n", name)
		default:
			fmt.Printf("  Xoá dịch vụ %s thất bại: %s\n", name, strings.TrimSpace(out))
		}
	}
}

// UninstallDriverFiles: 删 System32\drivers 下历史 .sys 与 System32\WinRing0x64.dll
func UninstallDriverFiles() {
	for _, name := range []string{"ThrottleStop.sys", "40hx_bridge.sys", "40hx_early-d.sys", "40hx_early.sys", "WinRing0x64.sys"} {
		p := os.Getenv("SystemRoot") + "\\System32\\drivers\\" + name
		if err := os.Remove(p); err != nil {
			if _, statErr := os.Stat(p); statErr == nil {
				fmt.Printf("  %s xoá thất bại (có thể đang bị chiếm dụng, khởi động lại sẽ tự xoá được)\n", name)
			}
		} else {
			fmt.Printf("  Đã xoá %s\n", name)
		}
	}
	os.Remove(os.Getenv("SystemRoot") + "\\System32\\WinRing0x64.dll")
}

// UninstallGspKey: 删 EnableGpuFirmware(恢复 GSP 默认关), 返回是否删除过
func UninstallGspKey() bool {
	key := FindGpuClassKey()
	if key == "" {
		return false
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.SET_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	if err := k.DeleteValue("EnableGpuFirmware"); err != nil {
		return false
	}
	return true
}

// UninstallProgramData: 清 ProgramData\40HXUnlock (gen2_status 历史缓存 + 驱动备份)。
// gen2_status.txt 必须删 — 诊断工具会把它当"上次结果"显示, 残留 ✅ 会误导用户。
func UninstallProgramData() {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	dir := base + "\\40HXUnlock"
	_ = os.Remove(dir + "\\gen2_status.txt")
	_ = os.RemoveAll(dir + "\\drivers")
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		os.Remove(dir)
	}
	// v2.6.0 修复: 策略键一并删除 — 否则卸载后 DriverStrategy/Gen2AutoHard 等
	// 残留, 重装会继承旧策略而非默认(README §2.5 承诺"卸载器会一并删除")。
	DeleteConfig()
	fmt.Println("  Khoá chính sách HKLM\\SOFTWARE\\40HXUnlock đã được xoá (cài lại sẽ về mặc định)")
}

// CheckLeftover: 卸载收尾的残留清单(供 GUI/卸载器展示)
func CheckLeftover() []string {
	var rem []string
	if out, _ := RunOut("bcdedit.exe", "/enum", "firmware"); strings.Contains(out, bootDesc40) {
		rem = append(rem, "- Mục khởi động firmware '40HX Unlock' (xoá thủ công trong BIOS)")
		fmt.Println("  [!] Mục khởi động vẫn còn tàn dư: bcdedit /delete {guid} /f (xem menu BIOS)")
	} else {
		fmt.Println("  Mục khởi động: Đã dọn sạch")
	}
	if k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE); err == nil {
		if _, _, e := k.GetStringValue("40HXGen2"); e == nil {
			rem = append(rem, "- Khoá Run 40HXGen2")
			fmt.Println("  [!] Khoá Run vẫn còn tàn dư")
		}
		k.Close()
	}
	taskLeft := false
	for _, tn := range UninstallTaskNames {
		if _, err := RunOut("schtasks.exe", "/query", "/tn", tn); err == nil {
			rem = append(rem, "- Tác vụ lịch trình "+tn)
			fmt.Println("  [!] Tác vụ lịch trình " + tn + " vẫn còn tàn dư")
			taskLeft = true
		}
	}
	if !taskLeft {
		fmt.Println("  Tác vụ lịch trình: Đã dọn sạch")
	}
	// v3.0.0: 补查驱动服务与 System32 驱动文件 — 常驻策略/文件被占用时
	// 卸载可能只删了服务注册、文件要重启后才能删, 不能假装干净。
	svcNames := []string{"ThrottleStop", "40hx_bridge", "40hx_early", "40hx_early-d", "WinRing0_1_2_0", "WinRing0x64", "WinRing0"}
	svcLeft := false
	for _, sn := range svcNames {
		if _, err := RunOut("sc.exe", "query", sn); err == nil {
			rem = append(rem, "- Dịch vụ driver "+sn)
			fmt.Println("  [!] Dịch vụ driver " + sn + " vẫn còn tàn dư (có thể vẫn đang chạy, hãy chạy lại gỡ cài đặt sau khi khởi động lại)")
			svcLeft = true
		}
	}
	if !svcLeft {
		fmt.Println("  Dịch vụ driver: Đã dọn sạch")
	}
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	fileLeft := false
	for _, fn := range []string{"ThrottleStop.sys", "40hx_bridge.sys", "40hx_early-d.sys", "40hx_early.sys", "WinRing0x64.sys"} {
		if _, err := os.Stat(sysRoot + "\\System32\\drivers\\" + fn); err == nil {
			rem = append(rem, "- Tệp driver " + fn)
			fmt.Println("  [!] Tệp driver " + fn + " vẫn còn tàn dư (có thể đang bị chiếm dụng, hãy chạy lại gỡ cài đặt sau khi khởi động lại)")
			fileLeft = true
		}
	}
	if !fileLeft {
		fmt.Println("  Tệp driver: Đã dọn sạch")
	}
	if esp := MountESP(); esp != "" {
		if _, err := os.Stat(esp + ":\\EFI\\40HX\\40HXUNLK.EFI"); err == nil {
			rem = append(rem, "- Tệp EFI mở khoá trong ESP")
			fmt.Println("  [!] EFI mở khoá trong ESP vẫn còn tàn dư")
		} else {
			fmt.Println("  EFI mở khoá trong ESP: Đã dọn sạch")
		}
		if _, err := os.Stat(esp + ":\\EFI\\Boot\\bootx64.efi.40hx.bak"); err == nil {
			rem = append(rem, "- Bản sao lưu bootx64.efi.40hx.bak chưa được khôi phục")
			fmt.Println("  [!] Bản sao lưu bootx64.efi.40hx.bak vẫn tồn tại")
		}
		if _, err := os.Stat(esp + ":\\40hx_log.txt"); err == nil {
			rem = append(rem, "- Gốc ESP 40hx_log.txt (nhật ký lịch sử EFI)")
			fmt.Println("  [!] Nhật ký lịch sử 40hx_log.txt vẫn tồn tại (chạy lại gỡ cài đặt một lần nữa sẽ sạch)")
		}
		RunOut("mountvol.exe", esp+":", "/D")
	}
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	pdDir := base + "\\40HXUnlock"
	if _, err := os.Stat(pdDir + "\\gen2_status.txt"); err == nil {
		rem = append(rem, "- ProgramData\\40HXUnlock\\gen2_status.txt (bộ nhớ đệm chẩn đoán)")
		fmt.Println("  [!] gen2_status.txt vẫn còn tàn dư")
	}
	if _, err := os.Stat(pdDir + "\\drivers"); err == nil {
		rem = append(rem, "- ProgramData\\40HXUnlock\\drivers (sao lưu driver)")
		fmt.Println("  [!] Bản sao lưu drivers vẫn còn tàn dư")
	}
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, ConfigKeyPath, registry.QUERY_VALUE); err == nil {
		k.Close()
		rem = append(rem, "- Khoá chính sách HKLM\\SOFTWARE\\40HXUnlock")
		fmt.Println("  [!] Khoá cấu hình chính sách HKLM\\SOFTWARE\\40HXUnlock vẫn còn tàn dư")
	}
	return rem
}
