package hxcore

// v2.6.0: 卸载操作上提 40hxcore — GUI 组件级卸载与 40HXUninstaller.exe 共用同一实现,
// 消除两份漂移副本(原 uninstall40x 私有函数)。
// 全部为破坏性操作, 调用方(GUI 卸载页/卸载器)负责确认与提权。
// 电源设置(快速启动/ASPM)属用户偏好, 刻意不提供回滚操作 — 恢复方法见 README。

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const bootDesc40 = "40HX Unlock"

// UninstallTaskNames: 本工具历史上用过的全部计划任务名(含 40HXGen2Retry 重试任务)
var UninstallTaskNames = []string{
	"CMP30HX_Gen2_Unlock",
	"CMP30HX_Gen2_Unlock_User",
	"40HXGen2",
	"40HX PCIe Gen2 Bring-up",
	"40HXGen2Retry",
	"40HXGspEnsure",
}

// UninstallTasks: 删除计划任务, 返回实际删掉的名字 (使用 COM go-ole)
func UninstallTasks() []string {
	var removed []string

	_ = withTaskScheduler(func(folder *ole.IDispatch) error {
		for _, tn := range UninstallTaskNames {
			if _, err := oleutil.CallMethod(folder, "DeleteTask", tn, 0); err == nil {
				fmt.Printf("  Đã xoá tác vụ lịch trình %s\n", tn)
				removed = append(removed, tn)
			}
		}
		return nil
	})
	return removed
}

// UninstallRunKey removes the startup value from both historical HKCU and AIO HKLM locations.
func UninstallRunKey() {
	for _, hive := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		k, err := registry.OpenKey(hive,
			`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
		if err == nil {
			_ = k.DeleteValue("40HXGen2")
			k.Close()
		}
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
			if _, err := RunOut("bcdedit.exe", "/delete", "{"+curGuid+"}", "/f"); err == nil {
				fmt.Printf("  Đã xoá mục khởi động %s\n", curGuid)
				removed = true
			} else {
				fmt.Printf("  [!] Xoá mục khởi động %s thất bại: %v\n", curGuid, err)
			}
			curGuid = ""
		}
	}
	return removed
}

// UninstallEspEfi removes the unlock image and restores the original fallback bootloader.
func UninstallEspEfi() bool {
	removed, err := UninstallEspEfiChecked()
	if err != nil {
		fmt.Printf("  [!] Khôi phục ESP thất bại: %v\n", err)
	}
	return removed
}

// UninstallEspEfiChecked reports restoration failures to callers that can surface them.
func UninstallEspEfiChecked() (bool, error) {
	esp := MountESP()
	if esp == "" {
		if FirmwareIsLegacy() {
			return false, nil
		}
		return false, fmt.Errorf("không thể gắn phân vùng EFI hệ thống để kiểm tra và gỡ cài đặt")
	}
	defer UnmountESP(esp)
	removed := false
	std := esp + ":\\EFI\\Boot\\bootx64.efi"
	bak := std + ".40hx.bak"
	if _, err := os.Stat(bak); err == nil {
		if err := restoreEfiBackup(std, bak); err != nil {
			return false, fmt.Errorf("khôi phục bootx64.efi: %w", err)
		}
		fmt.Println("    Đã khôi phục bootx64.efi gốc (từ .40hx.bak, kiểm tra OK)")
		removed = true
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("kiểm tra bản sao lưu bootx64.efi: %w", err)
	}
	target := esp + ":\\EFI\\40HX\\40HXUNLK.EFI"
	if err := os.Remove(target); err == nil {
		removed = true
	}
	if entries, err := os.ReadDir(esp + ":\\EFI\\40HX"); err == nil && len(entries) == 0 {
		_ = os.Remove(esp + ":\\EFI\\40HX")
	}
	if err := os.Remove(esp + ":\\40hx_log.txt"); err == nil {
		fmt.Println("    Đã xoá nhật ký lịch sử EFI 40hx_log.txt")
	}
	return removed, nil
}

var efiWriteFile = os.WriteFile

func restoreEfiBackup(target, backup string) error {
	original, err := os.ReadFile(backup)
	if err != nil {
		return err
	}
	if len(original) < 0x2000 || !bytes.HasPrefix(original, []byte("MZ")) {
		return fmt.Errorf("bản sao lưu bootx64.efi không phải ảnh EFI hợp lệ")
	}
	if err := efiWriteFile(target, original, 0o644); err != nil {
		return err
	}
	readback, err := os.ReadFile(target)
	if err != nil {
		return err
	}
	if !bytes.Equal(readback, original) {
		return fmt.Errorf("bootx64.efi readback differs from backup")
	}
	return os.Remove(backup)
}

// UninstallDriverServices removes only services with project-specific names.
// Generic ThrottleStop and WinRing0 services may belong to other applications.
func UninstallDriverServices() {
	for _, name := range []string{"40hx_bridge", "40hx_early", "40hx_early-d"} {
		err := withService(name, func(s *mgr.Service) error {
			_, _ = s.Control(svc.Stop)
			time.Sleep(300 * time.Millisecond)
			return s.Delete()
		})
		if err != nil {
			fmt.Printf("  Dịch vụ %s không tồn tại hoặc xoá thất bại: %v\n", name, err)
		}
	}
}

// UninstallDriverFiles removes only project-specific files.
func UninstallDriverFiles() {
	for _, name := range []string{"40hx_bridge.sys", "40hx_early-d.sys", "40hx_early.sys"} {
		path := filepath.Join(sysRoot(), "System32", "drivers", name)
		if err := os.Remove(path); err == nil {
			fmt.Printf("  Đã xoá %s\n", name)
		} else if _, statErr := os.Stat(path); statErr == nil {
			fmt.Printf("  %s xoá thất bại (có thể đang bị chiếm dụng)\n", name)
		}
	}
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
	_ = os.RemoveAll(PdDrvDir())
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		os.Remove(dir)
	}
	// The batch installer owns ProgramFiles\40HXUnlock and its runner.
	// Its uninstall command removes that directory after unregistering its startup task.
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
	for _, hive := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		if k, err := registry.OpenKey(hive,
			`Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE); err == nil {
			if _, _, e := k.GetStringValue("40HXGen2"); e == nil {
				rem = append(rem, "- Khoá Run 40HXGen2")
				fmt.Println("  [!] Khoá Run vẫn còn tàn dư")
			}
			k.Close()
		}
	}
	taskLeft := false
	for _, tn := range UninstallTaskNames {
		exist, _, _ := TaskInfo(tn)
		if exist {
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
	svcNames := []string{"40hx_bridge", "40hx_early", "40hx_early-d"}
	svcLeft := false
	for _, sn := range svcNames {
		if err := withService(sn, func(_ *mgr.Service) error { return nil }); err == nil {
			rem = append(rem, "- Dịch vụ driver "+sn)
			fmt.Println("  [!] Dịch vụ driver " + sn + " vẫn còn tàn dư")
			svcLeft = true
		}
	}
	if !svcLeft {
		fmt.Println("  Dịch vụ driver: Đã dọn sạch")
	}
	fileLeft := false
	for _, fn := range []string{"40hx_bridge.sys", "40hx_early-d.sys", "40hx_early.sys"} {
		if _, err := os.Stat(filepath.Join(sysRoot(), "System32", "drivers", fn)); err == nil {
			rem = append(rem, "- Tệp driver "+fn)
			fmt.Println("  [!] Tệp driver " + fn + " vẫn còn tàn dư")
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
		UnmountESP(esp)
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
