// 40HXCheck — CMP 40HX 解锁独立诊断工具 v3.0.0
//
// 双击即诊, 只读为主; v2.5 起若发现"算力/Gen2 无法实测(驱动未运行)"且驱动文件
// 在包内, 会临时拉起 ThrottleStop + WinRing0 实测后自清理(用完即卸, 保持无痕):
//
//	① 最优先显示: 算力解锁状态 + PCIe Gen2 状态
//	② 其次: GPU/Secure Boot/GSP/测试签名
//	③ 明细与建议
//
// 安装器(40HXInstaller)负责"装", 本工具负责"查"。
//
// 实现共享 tools/40hxcore (与安装器同一份探测/诊断代码, 不会漂移)。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	hxcore "40hxcore"
	"golang.org/x/sys/windows"
)

const (
	appTitle            = "CMP 40HX / 30HX Chẩn Đoán Mở Khoá"
	logsDirName         = "40HXUnlock"              // %LOCALAPPDATA%\40HXUnlock\logs
	gen2TaskName        = "40HX PCIe Gen2 Bring-up" // 与安装器 setupGen2Task 同名
	cmp30HXTaskName     = "CMP30HX_Gen2_Unlock"
	cmp30HXUserTaskName = "CMP30HX_Gen2_Unlock_User"
)

var (
	procMsgBoxW = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
)

// ---- v2.5: 临时驱动管理 (ThrottleStop + WinRing0, 用完即卸) ----
const (
	fileTS = "ThrottleStop.sys"
	fileWR = "WinRing0x64.sys"
	svcTS  = "ThrottleStop"
	svcWR  = "WinRing0_1_2_0"
)

func sysDrvDir() string {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return filepath.Join(root, "System32", "drivers")
}

// driverSrcDir: 在包结构中定位 drivers/ (v2.5: exe 旁 gen2/drivers / ProgramData 备份)。
func driverSrcDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	pd := filepath.Join(os.Getenv("ProgramData"), "40HXUnlock", "drivers") // 安装器留下的备份源
	for _, c := range []string{
		pd,
		filepath.Join(dir, "gen2", "drivers"),
		filepath.Join(dir, "drivers"),
	} {
		if _, e1 := os.Stat(filepath.Join(c, fileTS)); e1 == nil {
			if _, e2 := os.Stat(filepath.Join(c, fileWR)); e2 == nil {
				return c
			}
		}
	}
	return ""
}

func svcState(name string) string {
	_, _, st := hxcore.ServiceInfo(name)
	return st
}

// throttleStopAppRunning: 本机是否正在运行 ThrottleStop 软件(同名驱动共存, 不删它的)。
func throttleStopAppRunning() bool {
	out, _ := hxcore.RunOut("tasklist.exe", "/fi", "imagename eq ThrottleStop.exe")
	return strings.Contains(out, "ThrottleStop.exe")
}

// ensureDrivers: 确保 TS/WinRing0 服务 RUNNING。已运行→不管(外部管理);
// 否则从包 drivers 部署+启动。
// 返回 (deployed 本工具是否部署/尝试拉起过, ok 是否两个都在运行, fail 拉起失败线索)。
// 杀软隔离常把 .sys 替换成 0 字节占位(文件仍在)→ 仅判"不存在"会漏, 故按
// 缺失/0字节自愈重部署; 重部署后补 Defender 排除防再删。
// 说明: "拉起"本身就是一次驱动加载测试 — 失败大多能归因(见 classifyLoadErr)。
func ensureDrivers() (deployed bool, ok bool, fail string) {
	src := driverSrcDir()
	allRunning := svcState(svcTS) == "RUNNING" && svcState(svcWR) == "RUNNING"
	if allRunning {
		return false, true, ""
	}
	if src == "" {
		return false, false, "Trong gói phát hành không tìm thấy thư mục driver tạm thời (gen2\\drivers)"
	}
	var fails []string
	for _, d := range []struct{ svc, file string }{
		{svcTS, fileTS}, {svcWR, fileWR},
	} {
		dst := filepath.Join(sysDrvDir(), d.file)
		if b, e := os.ReadFile(dst); e != nil || len(b) == 0 {
			if sb, e2 := os.ReadFile(filepath.Join(src, d.file)); e2 == nil {
				os.WriteFile(dst, sb, 0o644)
				_ = hxcore.AddDefenderExclusions() // best-effort 防再删
			}
		}
		if svcState(d.svc) == "RUNNING" {
			continue
		}
		deployed = true
		hxcore.RunOut("sc.exe", "create", d.svc, "type=", "kernel",
			"start=", "demand", "binPath=", `\SystemRoot\System32\drivers\`+d.file)
		if _, err := hxcore.RunOut("sc.exe", "start", d.svc); err != nil {
			// 服务可能被标记为删除(1072)/禁用(1058)→ 清标记后重建+启动一次
			// (对齐安装器 ensureSvcLoaded; 卸载残留态下诊断也能自愈加载)
			hxcore.RunOut("sc.exe", "delete", d.svc)
			hxcore.RunOut("sc.exe", "create", d.svc, "type=", "kernel",
				"start=", "demand", "binPath=", `\SystemRoot\System32\drivers\`+d.file)
			if out2, err2 := hxcore.RunOut("sc.exe", "start", d.svc); err2 != nil {
				fails = append(fails, d.file+": "+strings.TrimSpace(out2))
			}
		}
	}
	time.Sleep(400 * time.Millisecond)
	ok = svcState(svcTS) == "RUNNING" && svcState(svcWR) == "RUNNING"
	if !ok && len(fails) > 0 {
		fail = strings.Join(fails, " || ")
	}
	return deployed, ok, fail
}

// classifyLoadErr: Chuyển đổi mã lỗi nạp driver thô sang nguyên nhân và cách xử lý dễ hiểu.
func classifyLoadErr(raw string) string {
	r := strings.ToLower(raw)
	switch {
	case strings.Contains(r, "1275"):
		return "Cài đặt bảo mật Windows chặn tải driver (Lỗi 1275) — Thường do Defender bật 'Cách ly lõi / Tính toàn vẹn bộ nhớ', 'Danh sách chặn driver dễ bị tấn công' hoặc Smart App Control; vui lòng tạm tắt các bảo vệ này trong Windows Security rồi thử lại (sau khi nạp xong có thể bật lại)"
	case strings.Contains(r, "577"):
		return "Hình ảnh driver bị hệ thống từ chối (Lỗi 577) — Tệp đã bị chỉnh sửa hoặc bị chính sách bảo mật chặn; vui lòng chạy lại 40HXInstaller để triển khai lại driver gốc và kiểm tra cài đặt 'Driver không tin cậy' trong Windows Security"
	case strings.Contains(r, "1058"):
		return "Dịch vụ bị vô hiệu hóa (Lỗi 1058) — Công cụ đã cố gắng kích hoạt lại và tạo lại dịch vụ"
	case strings.Contains(r, "1072"):
		return "Dịch vụ đang ở trạng thái tồn đọng 'Đánh dấu để xóa' (Lỗi 1072) — Công cụ đã tạo lại và thử lại"
	case strings.Contains(r, "拒绝访问"), strings.Contains(r, "access is denied"), strings.Contains(r, "error 5"), strings.Contains(r, " 5:"):
		return "Không đủ quyền hạn hoặc bị chặn driver (Lỗi 5 / Access Denied) — Nếu đã chạy Administrator: Lỗi do Tính toàn vẹn bộ nhớ (HVCI) hoặc Danh sách chặn driver (Vulnerable Driver Blocklist). CẦN KHỞI ĐỘNG LẠI MÁY (REBOOT) để Windows áp dụng tắt HVCI."
	case strings.Contains(r, "1060"), strings.Contains(r, "不存在"):
		return "Không tìm thấy dịch vụ (Lỗi 1060) — Tệp driver chưa được triển khai thành công, hãy chạy lại trình cài đặt rồi thử lại"
	}
	return "Khởi động driver thất bại — Thường do tính năng HIPS / chặn driver của phần mềm diệt virus bên thứ ba, vui lòng thêm hai tệp .sys vào danh sách tin cậy / loại trừ rồi thử lại; nếu vẫn không được hãy gửi nhật ký cho tác giả"
}

// cleanupDrivers: Tự dọn dẹp — Dừng dịch vụ, xoá dịch vụ, xoá tệp driver (giữ sạch sẽ hệ thống).
func cleanupDrivers() {
	if throttleStopAppRunning() {
		return
	}
	// Tôn trọng chiến lược driver: "Thường trú" thì không gỡ; còn lại tự động dọn dẹp bình thường.
	switch hxcore.DriverStrategy() {
	case hxcore.DriverStrategyResident:
		fmt.Println("  Chiến lược thường trú: Giữ lại dịch vụ driver và tệp (chẩn đoán không dọn dẹp)")
		return
	}
	for _, d := range []struct{ svc, file string }{
		{svcTS, fileTS}, {svcWR, fileWR},
	} {
		hxcore.RunOut("sc.exe", "stop", d.svc)
		hxcore.RunOut("sc.exe", "delete", d.svc)
		os.Remove(filepath.Join(sysDrvDir(), d.file))
	}
}

func msgbox(text string, icon uint) {
	t, _ := syscall.UTF16PtrFromString(appTitle)
	b, _ := syscall.UTF16PtrFromString(text)
	procMsgBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), uintptr(icon))
}

// isAdmin: 与安装器同款实现 (TokenElevation 在受限环境可能误报 0, 再试 SCM 全权)
func isAdmin() bool {
	var t windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &t)
	if err == nil {
		defer t.Close()
		var e uint32
		var n uint32
		if err = windows.GetTokenInformation(t, windows.TokenElevation,
			(*byte)(unsafe.Pointer(&e)), uint32(unsafe.Sizeof(e)), &n); err == nil && e != 0 {
			return true
		}
	}
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_ALL_ACCESS)
	if err == nil {
		windows.CloseServiceHandle(scm)
		return true
	}
	return false
}

// selfElevate: 非管理员时 ShellExecute runas 提权重启(诊断要挂 ESP 读 40hx_log)
func selfElevate() {
	exe, _ := os.Executable()
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	args := append([]string{}, os.Args[1:]...)
	args = append(args, "-elevated")
	params, _ := syscall.UTF16PtrFromString(strings.Join(args, " "))
	proc := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	r, _, _ := proc.Call(0,
		uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)), 0, 1)
	if r <= 32 {
		msgbox("Cần quyền Quản trị viên để đọc nhật ký mở khoá EFI (40hx_log.txt).\nVui lòng nhấp chuột phải vào chương trình -> Chọn Run as administrator.", 0x30)
	}
	os.Exit(0)
}

// logsDir: %LOCALAPPDATA%\40HXUnlock\logs (统一日志收集目录, 用户好找)
func logsDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	d := filepath.Join(base, logsDirName, "logs")
	os.MkdirAll(d, 0o755)
	return d
}

// collectLogs: 把相关日志汇集到固定目录, 返回目录路径
func collectLogs(diagSnapshot string) string {
	dir := logsDir()
	// 1. 安装器/Gen2 日志 (%TEMP%\40HX_installer.log)
	if b, err := os.ReadFile(filepath.Join(os.TempDir(), "40HX_installer.log")); err == nil {
		os.WriteFile(filepath.Join(dir, "installer.log"), b, 0o644)
	}
	// 2. EFI 解锁链日志 (ESP 根 40hx_log.txt) — 管理员下可读;
	//    v3.0.0: 仅当解锁 EFI 本体还在时才收集 — 卸载 EFI 后该文件是历史残留,
	//    拷进 logs 会在回溯时被误当"本次 EFI 运行日志"。
	if esp := hxcore.MountESP(); esp != "" {
		if _, efiErr := os.Stat(esp + `:\EFI\40HX\40HXUNLK.EFI`); efiErr == nil {
			if b, err := os.ReadFile(esp + ":\\40hx_log.txt"); err == nil {
				os.WriteFile(filepath.Join(dir, "40hx_log.txt"), b, 0o644)
			}
		}
		hxcore.UnmountESP(esp)
	}
	// 3. 本次诊断快照(最新) + 时间戳归档(保留历史便于对比)
	os.WriteFile(filepath.Join(dir, "diagnose.txt"), []byte(diagSnapshot), 0o644)
	ts := time.Now().Format("20060102_150405")
	os.WriteFile(filepath.Join(dir, "diagnose_"+ts+".txt"), []byte(diagSnapshot), 0o644)
	return dir
}

// indentLines: 多行文本统一加 4 空格缩进(状态文件内容展示用)
func indentLines(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = "    " + ln
	}
	return strings.Join(lines, "\n")
}

// copyToClipboard: PowerShell Set-Clipboard(失败静默 — 仅增强, 不阻塞)
func copyToClipboard(s string) bool {
	f, err := os.CreateTemp("", "40hx_clip_*.txt")
	if err != nil {
		return false
	}
	p := f.Name()
	f.WriteString(s)
	f.Close()
	defer os.Remove(p)
	out, err := hxcore.RunOut("powershell.exe", "-NoProfile", "-Command",
		"Get-Content -LiteralPath '"+p+"' -Raw -Encoding UTF8 | Set-Clipboard")
	return err == nil && !strings.Contains(out, "denied")
}

// ---- v2.6.0: 加强诊断快照, 便于社区反馈定位 ----
// 环境/驱动/原始 PCIe 寄存器/已知限制 四段富文本, 写入 diagnose.txt 与剪贴板。

// osVersion: Phiên bản/bản dựng Windows (cmd /c ver)
func osVersion() string {
	out, _ := hxcore.RunOut("cmd.exe", "/c", "ver")
	out = strings.TrimSpace(out)
	if out == "" {
		out = "Không rõ"
	}
	arch := os.Getenv("PROCESSOR_ARCHITECTURE")
	if arch == "" {
		arch = "?"
	}
	return fmt.Sprintf("%s [%s]", out, arch)
}

// driverDetail: Chi tiết triển khai từng driver — Nguồn dữ liệu = hxcore.InspectGen2Drivers()
func driverDetail(svc, file string) string {
	for _, d := range hxcore.InspectGen2Drivers() {
		if d.Service != svc || d.File != file {
			continue
		}
		sysS := "System32 thiếu"
		switch d.SysState {
		case hxcore.DrvZero:
			sysS = "System32 0 byte ⚠ Bị phần mềm diệt virus cách ly"
		case hxcore.DrvSizeMismatch:
			sysS = fmt.Sprintf("Kích thước System32=%d byte ⚠ Không khớp bản sao lưu (bị thay thế?)", d.SysSize)
		case hxcore.DrvOk:
			sysS = fmt.Sprintf("Kích thước System32=%d byte", d.SysSize)
		}
		svcS := "Dịch vụ chưa đăng ký"
		if d.SvcReg {
			svcS = "Dịch vụ " + d.SvcStart
			if d.SvcStart == "DISABLED" {
				svcS += " ⚠ Bị vô hiệu hóa (Gen2 không thể khởi động, chạy lại bộ cài để sửa)"
			} else if d.SvcRunning {
				svcS += "/Đang chạy"
			} else {
				svcS += "(demand, chờ tác vụ đăng nhập khởi động)"
			}
		}
		backS := "Không có nguồn sao lưu (chưa từng cài đặt)"
		if d.BackupOK {
			backS = "Nguồn sao lưu OK (đã từng triển khai)"
			if d.SysState == hxcore.DrvAbsent && !d.SvcReg {
				backS += "; Dùng xong gỡ bỏ tự dọn dẹp là bình thường, lần đăng nhập sau sẽ tự triển khai lại"
			}
		}
		return fmt.Sprintf("  %-16s %s | %s | %s\n", d.File, sysS, svcS, backS)
	}
	return fmt.Sprintf("  %-16s (Trạng thái chưa được kiểm tra)\n", file)
}

// spdName: PCIe 链路速率编码 → 名称
func spdName(s uint32) string {
	names := []string{"?", "Gen1(2.5GT/s)", "Gen2(5GT/s)", "Gen3(8GT/s)", "Gen4(16GT/s)", "Gen5"}
	if s >= uint32(len(names)) {
		return "?"
	}
	return names[s]
}

// rawPcieDump: Thanh ghi liên kết PCIe gốc + BAR0 BOOT_0 (Bằng chứng chẩn đoán cốt lõi)
func rawPcieDump() string {
	wh, err := hxcore.OpenDevice(`\\.\WinRing0_1_2_0`)
	if err != nil {
		return "  (WinRing0 không khả dụng, bỏ qua đọc thanh ghi gốc)\n"
	}
	defer hxcore.CloseHandle(wh)
	bdf, ok := hxcore.FindGPUPCI(wh)
	if !ok {
		return "  (FindGPUPCI không tìm thấy GPU, bỏ qua)\n"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  BDF=0x%05X (bus=%d dev=%d fn=%d)\n", bdf,
		(bdf>>8)&0xFF, (bdf>>3)&0x1F, bdf&0x7))
	cap := hxcore.PcieCap(wh, bdf)
	if cap == 0 {
		return sb.String() + "  (Không có PCIe Capability)\n"
	}
	rd := func(off uint32) uint32 {
		v, e := hxcore.PciRd(wh, bdf, off)
		if e != nil {
			return 0xFFFFFFFF
		}
		return v
	}
	lnkcap, lnkctl, lnksta := rd(cap+0x0C), rd(cap+0x10), rd(cap+0x12)
	lnkctl2, lnksta2 := rd(cap+0x30), rd(cap+0x32)
	sb.WriteString(fmt.Sprintf("  LNKCAP =0x%08X  Tốc độ tối đa=%s\n", lnkcap, spdName(lnkcap&0xF)))
	sb.WriteString(fmt.Sprintf("  LNKCTL =0x%08X (ASPM=%d RetrainLink=%d)\n", lnkctl, (lnkctl>>0)&3, (lnkctl>>5)&1))
	sb.WriteString(fmt.Sprintf("  LNKSTA =0x%08X Hiện tại=%s Độ rộng=x%d\n", lnksta, spdName(lnksta&0xF), (lnksta>>4)&0x3F))
	sb.WriteString(fmt.Sprintf("  LNKCTL2=0x%08X Mục tiêu=%s\n", lnkctl2, spdName(lnkctl2&0xF)))
	sb.WriteString(fmt.Sprintf("  LNKSTA2=0x%08X\n", lnksta2))
	// BAR0 BOOT_0 — Xác nhận BAR0 thực sự trỏ tới MMIO GPU
	if bar0raw, e := hxcore.PciRd(wh, bdf, 0x10); e == nil {
		bar0 := uint64(bar0raw & 0xFFFFFFF0)
		th, e2 := hxcore.OpenThrottleStop()
		if e2 == nil {
			defer hxcore.CloseHandle(th)
			if v, e3 := hxcore.TSRead(th, bar0+0x0); e3 == nil {
				fam := (v >> 24) & 0xFF
				tag := "Không rõ"
				if fam == 0x16 {
					tag = "TU10x (40HX OK)"
				} else if fam == 0x21 || fam == 0x17 {
					tag = "TU116 (30HX OK)"
				}
				sb.WriteString(fmt.Sprintf("  BAR0+0x00 BOOT_0=0x%08X (Họ chip=0x%02X %s)\n", v, fam, tag))
			} else {
				sb.WriteString(fmt.Sprintf("  BAR0+0x00 Đọc BOOT_0 thất bại: %v\n", e3))
			}
		} else {
			sb.WriteString("  (ThrottleStop không khả dụng, bỏ qua BOOT_0)\n")
		}
	}
	return sb.String()
}

// knownIssuesBlock: Giới hạn đã biết / sự cố tiềm ẩn
func knownIssuesBlock() string {
	var sb strings.Builder
	sb.WriteString("\n--- Giới hạn đã biết / Sự cố tiềm ẩn (v3.0.0) ---\n")
	sb.WriteString("· Trình điều khiển/GSP quản lý liên kết (GSP-RM): Khi nghỉ/tải thấp nvlddmkm ghi đè TLS phía GPU về Gen1; Stage2 tự động hoặc LD(-hard) có thể giải quyết. Đã xác nhận **không phải bảo vệ ghi firmware**, **KHÔNG CẦN FLASH VBIOS** — Nếu vẫn lỗi hãy gửi chẩn đoán và nhật ký cho tác giả.\n")
	sb.WriteString("· Mainboard OEM/Đa card: retrain-only trên một số bo mạch chủ không huấn luyện được Gen2, cần Root Link Disable (ngắt liên kết chớp nhoáng). Tác vụ đăng nhập v2.6+ mặc định tự thử Stage2 (Gen2AutoHard); nếu vẫn lỗi có thể chạy thủ công `40HXInstaller.exe -gen2 -hard`.\n")
	sb.WriteString("· EFI không tìm thấy card: v3.0 đã mở rộng quét bus 0-16 + CF8 0-255 bao quát toàn bộ; nếu sau khi cài EFI vẫn không tìm thấy, thường do firmware mainboard chưa khởi tạo khe không cổng xuất hình — chỉnh BIOS: Above 4G + Re-Size BAR / Init Display First=PEG / cắm khe PCIe trên cùng nối CPU.\n")
	sb.WriteString("· Nghỉ tiết kiệm điện: Khi tải thấp liên kết hạ về Gen1 là bình thường, có tải sẽ tự về Gen2/Gen3; Chẩn đoán TLS=Gen2/Gen3 nghĩa là cấu hình đã thành công.\n")
	sb.WriteString("· Chung sống với ThrottleStop: Driver của công cụ cùng tên với ThrottleStop, tự động tái sử dụng driver của nhau mà không xoá; nếu xung đột hãy tắt ThrottleStop rồi chạy lại.\n")
	sb.WriteString("· Phần mềm diệt virus cách ly: Có thể chặn các tệp driver .sys; vui lòng thêm tệp dự án vào danh sách loại trừ trong Windows Security.\n")
	sb.WriteString("· Báo cáo sự cố (issue) vui lòng đính kèm: diagnose.txt + %TEMP%\\40HX_installer.log + Cấu hình Mainboard/CPU/GPU/Hệ điều hành.\n")
	return sb.String()
}

// drvLogEvidence: Kiểm tra lịch sử khi driver chưa tải
func drvLogEvidence() (bool, string) {
	var parts []string
	// 1. gen2_status.txt
	if gs := hxcore.ReadGen2Status(); gs != "" {
		for _, ln := range strings.Split(gs, "\n") {
			t := strings.TrimSpace(ln)
			if strings.Contains(t, "Gen2") || strings.Contains(t, "Gen3") || strings.Contains(t, "无需操作") || strings.Contains(t, "ACHIEVED") {
				parts = append(parts, "Lịch sử tác vụ: "+t)
				break
			}
		}
	}
	// 2. installer.log
	logPath := filepath.Join(os.TempDir(), "40HX_installer.log")
	if b, err := os.ReadFile(logPath); err == nil {
		s := string(b)
		if strings.Contains(s, "GEN2 ACHIEVED") || strings.Contains(s, "GEN3 ACHIEVED") || strings.Contains(s, "BAR0 校验通过") {
			parts = append(parts, "installer.log: Driver từng nạp thành công và đạt tốc độ cao")
		}
		if strings.Contains(s, "启动服务") && strings.Contains(s, "失败") {
			parts = append(parts, "installer.log: Từng xuất hiện lỗi khởi động driver (có thể do diệt virus/tồn đọng 1072)")
		}
	}
	if len(parts) == 0 {
		return false, ""
	}
	return true, strings.Join(parts, "; ")
}

func check() {
	var sb strings.Builder
	w := func(format string, a ...interface{}) { sb.WriteString(fmt.Sprintf(format, a...)) }
	var tips []string
	// v2.6.0: 版本标题写入 sb → 弹窗/diagnose.txt 都可见 (此前 fmt.Println 只进 log)
	w("==============================================\n")
	w("  Chẩn đoán mở khoá CMP 40HX / 30HX  v3.0.0   %s\n", time.Now().Format("2006-01-02 15:04:05"))
	w("==============================================\n")
	gpuOK := hxcore.FindGPU()
	sbOn := hxcore.SecureBootOn()
	tsOn := hxcore.TestSigningOn()
	gsOn := hxcore.GspEnabled()
	sub, _, _ := hxcore.GspDiag()

	// --- A. Phán đoán chính: Hiệu năng + Gen2/Gen3 ưu tiên cao nhất ---
	selfM, drvOK, drvFail := ensureDrivers()
	st := hxcore.ReadUnlockStateV2(6, 800)
	prof, profOK := hxcore.FindGPUWithProfile()
	cmp30HX := profOK && prof.DeviceID == 0x2189
	bar := strings.Repeat("=", 46)
	w("\n%s\n", bar)
	state := "Không thể kiểm tra thực tế (Driver chưa sẵn sàng)"
	switch {
	case cmp30HX && st.Speed >= 2:
		state = fmt.Sprintf("CMP 30HX Gen2 thành công (Liên kết hiện tại Gen%d)", st.Speed)
	case cmp30HX && st.TLS >= 2:
		state = fmt.Sprintf("CMP 30HX Gen2 đã cấu hình (TLS=Gen%d; hiện tại Gen%d do tiết kiệm điện/rảnh)", st.TLS, st.Speed)
	case cmp30HX && st.Speed >= 1:
		state = fmt.Sprintf("CMP 30HX chưa đạt Gen2 (hiện tại Gen%d, TLS=Gen%d)", st.Speed, st.TLS)
	case st.SS0OK && st.Unlocked && st.Speed >= 2:
		state = fmt.Sprintf("Hiệu năng tối đa + Đã đạt Gen%d", st.Speed)
	case st.SS0OK && st.Unlocked && st.TLS >= 2:
		state = fmt.Sprintf("Hiệu năng tối đa + Mục tiêu Gen%d đã cấu hình (Liên kết hiện tại chưa đạt)", st.TLS)
	case st.SS0OK && st.Unlocked:
		state = "Hiệu năng tối đa, Gen2/Gen3 chưa đạt"
	case st.SS0OK:
		state = "Chưa mở khoá (SS0 bị khoá)"
	}
	w("  Trạng thái mở khoá : %s\n", state)
	comp := "Không thể đọc"
	if st.ComputeReport != nil {
		comp = st.ComputeReport.Verdict
	} else if st.SS0OK {
		comp = fmt.Sprintf("%s (SS0=0x%08X SS1=0x%08X)",
			map[bool]string{true: "✓ Tối đa", false: "✗ Bị khoá"}[st.Unlocked], st.SS0, st.SS1)
	}
	spd := "Không thể đọc"
	if st.Speed >= 1 {
		names := map[uint32]string{1: "Gen1 (2.5 GT/s)", 2: "Gen2 (5.0 GT/s)", 3: "Gen3 (8.0 GT/s)", 4: "Gen4 (16 GT/s)"}
		spd = names[st.Speed]
		if spd == "" {
			spd = fmt.Sprintf("Gen%d", st.Speed)
		}
		if st.Width >= 1 {
			spd += fmt.Sprintf(" ×%d", st.Width)
		}
		if st.Speed < 2 && st.TLS >= 2 {
			spd += fmt.Sprintf(" (Mục tiêu Gen%d — Nghỉ tiết kiệm điện hạ tốc độ là bình thường; nếu tải liên tục vẫn Gen1 xem kết luận)", st.TLS)
		}
	}
	w("  Hiệu năng: %s\n", comp)
	w("  PCIe     : %s\n", spd)
	w("%s\n", bar)

	// --- B. Trạng thái cơ bản ---
	gspTxt := "✗ Chưa bật (Sau khi cài driver NVIDIA sẽ do bộ cài tự thiết lập)"
	if cmp30HX {
		gspTxt = "Không hỗ trợ (TU116 không có phần cứng GSP - Bỏ qua)"
	} else if gsOn {
		gspTxt = "✓ Đã bật"
	} else if sub == "" {
		gspTxt = "— Không tìm thấy khoá GSP (Cần cài driver NVIDIA trước mới có thể bật GSP)"
	}
	sbTxt := map[bool]string{true: "Bật (Cần tắt!)", false: "Tắt (OK)"}[sbOn]
	if cmp30HX {
		sbTxt = map[bool]string{true: "Bật (OK - Tương thích Riot Vanguard)", false: "Tắt (OK)"}[sbOn]
	}
	w("GPU CMP : %s   Secure Boot: %s   GSP: %s\n",
		map[bool]string{true: "✓", false: "✗"}[gpuOK],
		sbTxt, gspTxt)
	w("Ký thử nghiệm: %s  (v2.5 trở lên không cần, khuyến nghị tắt)\n",
		map[bool]string{true: "Đã bật", false: "Tắt"}[tsOn])

	bootMode := "UEFI (OK)"
	if hxcore.FirmwareIsLegacy() {
		bootMode = "Legacy BIOS+MBR (Không có phân vùng EFI, không thể mở khoá hiệu năng!)"
	}
	fsOn := hxcore.FastStartupOn()
	ac, dc, aspmOK := hxcore.ASPMSavings()
	aspmStr := "Không thể kiểm tra (bỏ qua)"
	if aspmOK {
		if ac == 0 && dc == 0 {
			aspmStr = "Tắt (OK)"
		} else {
			aspmStr = fmt.Sprintf("Bật (AC=%d DC=%d) — Lúc nghỉ có thể hạ về Gen1, khuyến nghị tắt", ac, dc)
		}
	}
	w("Chế độ khởi động: %s\n", bootMode)
	w("Khởi động nhanh : %s   PCIe ASPM: %s\n",
		map[bool]string{true: "Bật (Khuyến nghị tắt)", false: "Tắt (OK)"}[fsOn],
		aspmStr)

	// --- C. Chi tiết driver & tác vụ ---
	tsTxt := "✗ Không chạy"
	if st.TSOK {
		tsTxt = "✓ Khả dụng"
	} else if svcState(svcTS) == "RUNNING" {
		tsTxt = "⚠ Dịch vụ tồn tại nhưng không mở được thiết bị"
	}
	wrTxt := "✗ Không chạy"
	if st.WinRingOK {
		wrTxt = "✓ Khả dụng"
	} else if svcState(svcWR) == "RUNNING" {
		wrTxt = "⚠ Dịch vụ tồn tại nhưng không mở được thiết bị"
	}
	w("Driver PCIe : ThrottleStop %s   WinRing0 %s\n", tsTxt, wrTxt)
	if selfM {
		w("  ↑ Do chẩn đoán tạm thời kích hoạt để kiểm tra, đo xong tự gỡ — Không có nghĩa là đã cài đặt\n")
	} else if st.TSOK || st.WinRingOK {
		w("  ↑ Driver đã được triển khai / nạp sẵn từ bên ngoài\n")
	}
	if !drvOK && drvFail != "" {
		w("  └ Khởi động thất bại: %s\n", classifyLoadErr(drvFail))
	}
	w("\n")
	taskName := gen2TaskName
	if cmp30HX {
		taskName = cmp30HXTaskName
	}
	taskOK, taskStatus, taskResult := hxcore.TaskInfo(taskName)
	if !taskOK && cmp30HX {
		taskOK, taskStatus, taskResult = hxcore.TaskInfo(cmp30HXUserTaskName)
	}
	w("Tác vụ Gen2 : %s\n",
		map[bool]string{true: "Đã đăng ký (" + taskStatus + ", Kết quả lần trước: " + taskResult + ")",
			false: "Chưa đăng ký (Cách sửa: Nhấp chuột phải Run as administrator Setup_CMP30HX_WindowsAIO.bat)"}[taskOK])

	if !st.SS0OK || st.Speed < 2 {
		if gs := hxcore.ReadGen2Status(); gs != "" {
			w("  Lưu ý: Tồn tại bản ghi lịch sử tác vụ lần trước (không phải kiểm tra thực tế lần này):\n%s\n", indentLines(gs))
			if !drvOK {
				w("       ↑ Driver hiện không chạy — Đây là dữ liệu cũ còn sót lại, không đại diện cho trạng thái hiện tại\n")
			}
		}
	}
	if drvOK && !st.SS0OK {
		w("(Driver đã chạy nhưng không đọc được thanh ghi hiệu năng — Bất thường)\n")
	}

	// --- D. Kết luận & Đề xuất ---
	verdict := ""
	switch {
	case cmp30HX && st.Speed >= 2:
		verdict = fmt.Sprintf(">>> CMP 30HX mở khoá thành công: PCIe Gen%d", st.Speed)
		if st.Width >= 1 {
			verdict += fmt.Sprintf(" ×%d", st.Width)
		} else {
			verdict += " (Độ rộng liên kết chưa đo được)"
		}
	case cmp30HX && st.TLS >= 2:
		if st.Speed == 1 {
			verdict = fmt.Sprintf(">>> CMP 30HX: Mục tiêu Gen%d đã cấu hình (Hiện tại Gen1 do tiết kiệm điện rảnh; khi có tải 3D/CUDA/AIDA64 sẽ tự lên Gen%d)", st.TLS, st.TLS)
		} else {
			verdict = fmt.Sprintf(">>> CMP 30HX: Mục tiêu Gen%d đã cấu hình (Tốc độ liên kết hiện tại chưa đo được)", st.TLS)
		}
	case cmp30HX && st.Speed == 1 && st.TLS < 2:
		verdict = ">>> CMP 30HX chưa đạt Gen2: Cả liên kết và mục tiêu đều ở Gen1. Chạy lại Setup_CMP30HX_WindowsAIO.bat với quyền Admin, kiểm tra riser/khe PCIe và khởi động lại"
	case st.Unlocked && st.Speed >= 2:
		verdict = fmt.Sprintf(">>> Mở khoá thành công: Tensor hiệu năng tối đa + Gen%d", st.Speed)
		if st.Width >= 1 {
			verdict += fmt.Sprintf(" ×%d", st.Width)
		} else {
			verdict += " (Độ rộng liên kết chưa đo được)"
		}
	case st.Unlocked && st.TLS >= 2:
		if st.Speed == 1 {
			verdict = fmt.Sprintf(">>> Hiệu năng tối đa + Mục tiêu Gen%d đã cấu hình (Hiện tại Gen1: Tiết kiệm điện hoặc chưa huấn luyện xong; tải nặng/huấn luyện lại sẽ lên Gen%d)", st.TLS, st.TLS)
		} else {
			verdict = fmt.Sprintf(">>> Hiệu năng tối đa + Mục tiêu Gen%d đã cấu hình (Tốc độ liên kết hiện tại chưa đo được)", st.TLS)
		}
	case st.Unlocked:
		verdict = ">>> Hiệu năng tối đa; Gen2/Gen3 chưa đạt — Tác vụ đăng nhập đã tự động thực thi mỗi lần đăng nhập; có thể bấm [Thực thi Gen2 ngay] trong GUI ② để thử lại"
	case st.SS0OK:
		verdict = ">>> Lần khởi động này chưa mở khoá (SS0 bị khoá)"
	default:
		if !drvOK {
			if !isAdmin() {
				verdict = ">>> Driver chưa nạp: Công cụ cần quyền Quản trị viên để nạp driver kernel đo đạc trạng thái. Vui lòng nhấp chuột phải -> Chọn Run as administrator"
			} else if gpuOK && gsOn {
				if ever, ev := drvLogEvidence(); ever {
					verdict = ">>> Driver hiện không chạy, nhưng nhật ký cho thấy trước đó đã đạt tốc độ cao thành công (" + ev + "). Mở khoá đã có hiệu lực, nhấp chuột phải Run as administrator công cụ này để đọc trạng thái thời gian thực"
				} else {
					verdict = ">>> Driver không khả dụng — Vui lòng chạy từ thư mục phát hành 40HXUnlock (chứa gen2\\drivers), hoặc chạy 40HXInstaller.exe với quyền Quản trị viên để cài đặt driver"
				}
			} else {
				verdict = ">>> Không thể hoàn thành phán đoán mở khoá (xem chi tiết bên trên)"
			}
		} else {
			verdict = ">>> Không thể hoàn thành phán đoán mở khoá (xem chi tiết bên trên)"
		}
	}
	w("\n%s\n", verdict)
	if st.SS0OK && !st.Unlocked {
		if reason := hxcore.AnalyzeEfiLog(); reason != "" {
			w("%s\n", reason)
		}
	}

	// v2.6.0: 弹窗只显示"简洁结论 + 基础状态"; 下方详细建议/原始寄存器只进 diagnose.txt(日志)与剪贴板,
	// 不放 GUI —— 用户要求提示放 txt 不放弹窗, 社区看完整诊断去 logs 目录即可。
	guiHead := sb.String()

	if !gpuOK {
		tips = append(tips, "· Không phát hiện card CMP: Xác nhận card đã cắm và driver đã cài đặt")
	}
	if !cmp30HX && sbOn {
		tips = append(tips, "· Secure Boot đang bật: Vào BIOS để tắt (nếu không EFI mở khoá sẽ bị từ chối)")
	}
	if tsOn {
		tips = append(tips, "· Chế độ Test Signing đang bật (v2.5 trở lên không cần): bcdedit /set testsigning off để tắt")
	}
	if !cmp30HX && !gsOn {
		tips = append(tips, "· GSP chưa bật: Mở 40HXInstaller.exe -> ① Chọn [Bật GSP] -> Bấm [Cài đặt các mục đã chọn]")
	}
	if gpuOK && st.SS0OK && !st.Unlocked {
		tips = append(tips, "· EFI mở khoá hiệu năng chưa có hiệu lực (SS0 bị khoá): Nếu máy chưa triển khai / đã gỡ bỏ '40HX Unlock' EFI, hiệu năng sẽ giữ khoá — Muốn khôi phục hãy chạy 40HXInstaller.exe chọn [Cài đặt EFI hiệu năng + Mục khởi động firmware]; Nếu EFI đã cài, hãy kiểm tra máy đã khởi động qua mục '40HX Unlock' chưa / Above 4G đã bật / Secure Boot đã tắt")
	}
	if hxcore.FirmwareIsLegacy() {
		tips = append(tips, "· Chế độ khởi động là Legacy BIOS+MBR: Không có phân vùng EFI, không thể cài mở khoá hiệu năng —\n  Theo dõi README §2.4 dùng mbr2gpt chuyển sang GPT rồi chạy lại bộ cài (Gen2 không bị ảnh hưởng)")
	}
	if fsOn {
		tips = append(tips, "· Khởi động nhanh (Fast Startup) đang bật: Tắt máy mở lại có thể không qua trình khởi động UEFI đầy đủ -> EFI không thực thi;\n  Bộ cài sẽ tự động tắt; thủ công: Vào Control Panel -> Power Options bỏ chọn 'Turn on fast startup'")
	}
	if aspmOK && (ac > 0 || dc > 0) {
		tips = append(tips, "· Tính năng tiết kiệm điện PCIe (ASPM) đang bật: Lúc nghỉ hạ về Gen1 là bình thường, có tải sẽ tự tăng lại;\n  Nếu muốn cố định tốc độ cao có thể tắt: powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0\n  (thêm -setdcvalueindex cùng tham số, sau đó -setactive SCHEME_CURRENT để áp dụng)")
	}
	if !taskOK {
		tips = append(tips, "· Tác vụ lịch trình Gen2 chưa đăng ký: Sau khi đăng nhập sẽ không tự động mở khoá —\n  Mở 40HXInstaller.exe -> ② Bấm [Thực thi Gen2 và cài đặt tự khởi động]; hoặc ① Chọn [Tự khởi động Gen2 khi đăng nhập] bấm [Cài đặt các mục đã chọn]")
	}
	if st.SS0OK && st.TLS < 2 && st.TLS >= 1 && st.Unlocked {
		tips = append(tips, "· Tốc độ mục tiêu (TLS) vẫn là Gen1: Ghi mở khoá chưa có hiệu lực — Nếu nhật ký cho thấy thanh ghi PL0 đều OK nhưng đọc lại LNKCTL2 vẫn Gen1,\n  thường do driver ghi đè lại trong tích tắc; Tác vụ đăng nhập sẽ tự động chạy Stage2; nếu vẫn thất bại hãy bấm [Thực thi Gen2 ngay] trong GUI ②. Đã xác nhận không phải bảo vệ ghi firmware, **KHÔNG CẦN FLASH VBIOS** — nếu vẫn lỗi gửi chẩn đoán và installer.log phản hồi tác giả")
	}
	if st.SS0OK && st.Unlocked && st.Speed < 2 && st.TLS >= 2 {
		tips = append(tips, "· Tốc độ mục tiêu đã cấu hình (TLS=Gen2/Gen3) nhưng liên kết hiện tại Gen1: Thường do nghỉ tiết kiệm điện (bình thường, có tải tự tăng); nếu liên tục tải vẫn Gen1: Bấm [Thực thi Gen2 ngay] trong GUI ② để huấn luyện lại; hoặc dùng dòng lệnh `40HXInstaller.exe -gen2 -hard`")
	}
	if !taskOK && st.SS0OK && st.Unlocked && (st.Speed >= 2 || st.TLS >= 2) {
		tips = append(tips, "· Lưu ý: Trạng thái tốc độ cao hiện tại đến từ đo đạc thanh ghi mục tiêu (TLS); Nếu lần khởi động này chưa có tác vụ tự động nào chạy, giá trị này có thể là tàn dư lần trước — tắt máy hẳn hoặc reset card sẽ bị khoá lại. Vui lòng dùng GUI ② [Thực thi Gen2 và cài đặt tự khởi động] để hoàn tất một bước")
	}
	if throttleStopAppRunning() && !drvOK {
		tips = append(tips, "· Phát hiện phần mềm ThrottleStop đang chạy và driver công cụ chưa sẵn sàng: Công cụ tự động tái sử dụng driver của ThrottleStop; nếu vẫn lỗi, hãy đóng ThrottleStop rồi chạy lại bộ cài")
	}
	if !drvOK {
		if drvFail != "" {
			tips = append(tips, "· Driver PCIe không thể khởi động: "+classifyLoadErr(drvFail))
		}
		if ever, ev := drvLogEvidence(); ever {
			tips = append(tips, "· Driver hiện không chạy, nhưng nhật ký cho thấy trước đó đã đạt tốc độ cao thành công ("+ev+") — Mở khoá đã có hiệu lực, nhấp chuột phải chọn Run as administrator công cụ này để đọc trạng thái thời gian thực")
		} else {
			tips = append(tips, "· Driver chưa từng được triển khai thành công (không có ghi nhận thành công): Mở 40HXInstaller.exe -> ① Bấm [Cài đặt toàn bộ một nhấp] để hoàn tất triển khai và đăng ký tự khởi động")
		}
	}
	if len(tips) > 0 {
		w("\nKhuyến nghị:\n%s\n", strings.Join(tips, "\n"))
	}
	popupTip := ""
	if len(tips) > 0 {
		popupTip = "\n\n▶ Bước tiếp theo: " + strings.SplitN(tips[0], "\n", 2)[0]
	}

	w("\n========== MÔI TRƯỜNG ==========\n")
	w("  Hệ điều hành : %s\n", osVersion())
	w("  Quản trị viên: %s   Phiên bản: v3.0.0\n", map[bool]string{true: "Có", false: "Không"}[isAdmin()])
	w("\n========== DRIVER ==========\n")
	w("%s", driverDetail(svcTS, fileTS))
	w("%s", driverDetail(svcWR, fileWR))
	if selfM {
		w("  ↑ Chẩn đoán 【tạm thời nạp】 để đọc phần cứng, đo xong tự dọn — Không đại diện đã cài đặt\n")
	} else if st.TSOK || st.WinRingOK {
		w("  ↑ Driver đã được triển khai / nạp sẵn, công cụ không dọn dẹp\n")
	}
	w("  ThrottleStop đang chạy: %s\n", map[bool]string{true: "Có (chung sống, không xoá driver)", false: "Không"}[throttleStopAppRunning()])
	w("\n========== THANH GHI PCIE GỐC ==========\n")
	w("%s", rawPcieDump())
	w("%s", knownIssuesBlock())

	out := sb.String()
	fmt.Println(out)

	if selfM {
		cleanupDrivers()
	}

	dir := collectLogs(out)
	copied := copyToClipboard(out)
	note := ""
	if copied {
		note = "\n\nChẩn đoán đầy đủ (kèm khuyến nghị và thanh ghi gốc) đã sao chép vào bộ nhớ tạm (Clipboard) — Bạn có thể dán trực tiếp khi báo lỗi."
	}
	ok := st.Unlocked && (st.Speed >= 2 || st.TLS >= 2)
	msgbox(guiHead+popupTip+"\n\nChẩn đoán đầy đủ (kèm khuyến nghị & thanh ghi PCIe gốc) đã ghi vào:\n"+dir+"\\diagnose.txt"+note,
		map[bool]uint{true: 0x40, false: 0x30}[ok])
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "-elevated" {
		if !isAdmin() {
			selfElevate()
			return
		}
	}
	// 输出镜像到统一日志目录
	dir := logsDir()
	if f, err := os.Create(filepath.Join(dir, "40HXCheck.log")); err == nil {
		os.Stdout = f
		os.Stderr = f
		fmt.Fprintf(f, "==== 40HXCheck %s ====\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	check()
}
