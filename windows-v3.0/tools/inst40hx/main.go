// 40HX 一键安装工具 v3.0.0 (CMP 40HX Windows Unlock Installer)
// 功能:
//
//	(默认) 安装: GSP 启用(EnableGpuFirmware=1) + ESP 双路部署 40HXUNLK.EFI (V70)
//	      + BootOrder 置顶 + 驱动 + Gen2 自启动
//	-gen2        立即执行 Gen2 解锁(供登录自启动调用, 幂等)
//	-uninstall   卸载(移除启动项/Run键/驱动服务/EnableGpuFirmware)
//	-status      状态检查
//
// 资源 embed (v2.5): 40HXUNLK.EFI (V70 解锁版) / ThrottleStop.sys / WinRing0x64.sys
// v2.6.0 关键修复(社区 #2/#5/#6/#7 + v2.4.5 时代排障结论):
//  1. EFI 部署失败不再中止安装 — Legacy/MBR(无 ESP)只跳过 EFI 两步, Gen2 任务
//     照常注册(此前 [5/8] 直接 return, 是"装了驱动开机却不跑 Gen2"的统一根因)
//  2. 引导模式检测(GetFirmwareType): Legacy → 弹窗给 mbr2gpt 无损转换完整指引
//  3. 计划任务创建后 schtasks query 二次校验 + 重试; -task 失败以非零码退出
//     (命令行调用时的 errorlevel 检查从死代码变为有效)
//  4. 自动关闭快速启动(混合休眠)与 PCIe 链路省电(ASPM) — 前者避免"关机再开
//     不走完整 UEFI 引导", 后者减少空闲降到 Gen1 被误读为解锁失败
//  5. Gen2 核心增强: LNKCTL2 读改写(不清高位) + root/GPU 交替重训最多 4 轮 +
//     以 TLS 目标速率判成败(空闲省电降速 Gen1 不再误报失败)
//
// v2.6.0 关键加固(自启动通道设计与并发安全, 回应"多自启动路径怕出问题"):
//  1. Gen2 单实例内核互斥体(Global\40HXGen2SingleInstance): SYSTEM 任务 / Run 键 /
//     手动 -gen2 即使并发触发, 也仅一个进程进入"加载-卸载 BYOVD 驱动 + 抢 BAR0"
//     临界区, 杜绝双进程争用驱动服务名与链路寄存器导致的状态错乱
//  2. 自启动通道收敛为"两路互斥串行": Run 键登录瞬间先试(可能 GPU 未就绪而失败,
//     静默交权), SYSTEM 任务延迟 30s 再确认; 其余 13 类路径(HKCU/HKLM Run 之外)
//     均运行于用户态、无法 sc start 内核驱动, 故不采用(详见设计文档)
//  3. 定位 40HX 失败重试最多 3 次(间隔 2s), 容忍慢速 GPU 初始化导致的假失败
//
// v2.4 关键变更(社区兼容):
//  1. embed EFI 回到 V70 原版 (793d765e, 用户实测解锁成功) — v2.1/v2.2 精简版失败教训
//  2. ESP 双路部署: \EFI\40HX\40HXUNLK.EFI (BCD 主路径)
//     + \EFI\Boot\bootx64.efi (UEFI 标准 fallback, 原文件备份 .40hx.bak)
//     解决部分主板不认非标准 EFI 路径/忽略 BCD displayorder 导致"装完重启没反应"
//  3. BootOrder 写入后从固件读回验证, 不在首位时明确弹窗提示 BIOS 手动置顶
//  4. 关键 BIOS 操作全部进消息框 (社区用户不看 README/日志)
//
// v2.3 关键: EnableGpuFirmware=1 启用 GSP — 40HX 默认 GSP 关(CPU-RM 模式)时,
//
//	EFI 解锁后 nvlddmkm 拒绝 SEC2 状态 -> Code43 黑屏; GSP-RM 模式能接受解锁.
package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"40hxcore"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed embed/*
var embedded embed.FS

const (
	gpuVenDev = "VEN_10DE&DEV_1F0B"
	efiDir    = "\\EFI\\40HX"
	efiFile   = "40HXUNLK.EFI"
	bootDesc  = "40HX Unlock"
	// v2.4: UEFI 标准回退路径 (固件 BootOrder 全部无效/未签名时自动尝试此路径;
	// 解决部分主板忽略 BCD displayorder / 不认非标准 \EFI\40HX 目录)
	efiStdDir = "\\EFI\\Boot"
	efiStdF   = "bootx64.efi"
	efiBakExt = ".40hx.bak" // bootx64.efi.40hx.bak 原文件备份
	// v2.3: GSP 启用注册表 (EnableGpuFirmware=1) — 解锁不黑屏的关键!
	// 40HX 的显示适配器 Class 子键 (0001 = 40HX; 多卡时需按 AdapterString 找)
	gpuClassPath  = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	gpuClassGUID  = `{4d36e968-e325-11ce-bfc1-08002be10318}` // Driver 值反查用
	gpuEnableFw   = "EnableGpuFirmware"
	gpuAdapterStr = "HardwareInformation.AdapterString"
	gpuAdapter40  = "CMP 40HX"
	// v2.4.6: Gen2 的 SYSTEM 计划任务名(卸载时按名字删除)
	gen2TaskName = "40HX PCIe Gen2 Bring-up"
	// v2.6.0: Gen2 失败后的自动重试任务(一次性, 成功即删, 卸载链按名清理)
	gen2RetryTask = "40HXGen2Retry"
)

func main() {
	// GUI 无窗口版(v1.1): 输出全部镜像到日志(默认 %TEMP%\40HX_installer.log, 可 -log 指定)
	setupLog("40HX_installer.log")
	// v2.6.0: 双击(无参数)或 UAC 提权重启(-elevated)默认进入 GUI 管理界面;
	// 命令行参数(-gen2/-task/-uninstall/-status/-silent/-hard)语义保持不变。
	if len(os.Args) <= 1 || (len(os.Args) == 2 && os.Args[1] == "-elevated") {
		runGUI()
		return
	}
	// install/-uninstall 需管理员: 非提升时自动 ShellExecute runas 弹 UAC 重启
	needAdmin := true
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-gen2", "-gen3", "-force-root-gen2", "-force-root-gen3", "-gen2-30hx", "-gen3-30hx", "-probe-30hx", "-gspensure", "-status", "-h", "-help", "--help":
			needAdmin = false
		}
		// -task 需管理员(GUI 双击自动 UAC; gen2/status 等只读或 SYSTEM 任务调用无需)
		if os.Args[1] == "-task" || os.Args[1] == "-probe-30hx" || os.Args[1] == "-gen2-30hx" || os.Args[1] == "-gen3-30hx" {
			needAdmin = true
		}
	}
	if needAdmin && !isAdmin() {
		if hasArg("-elevated") {
			// 已提权过一次仍失败(如静默提权策略下受限token) -> 禁止再循环, 直接报错
			msgbox("Trình Cài Đặt 40HX / 30HX", "Nâng quyền thất bại: Tài khoản hiện tại không có quyền Quản trị viên (Administrator).\nVui lòng nhấp chuột phải vào ứng dụng -> Chọn 'Run as administrator'.", mbIconError)
			return
		}
		selfElevate()
		return
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-gen2", "-gen3", "-force-root-gen2", "-force-root-gen3", "-gen2-30hx", "-gen3-30hx":
			gen2Succeeded = false
			gen2Main()
			if !gen2Succeeded {
				os.Exit(1)
			}
			// v3.0.1: 常驻守护 — 由登录任务带 -guard 启动; 驱动保留并每分钟自查 Gen2
			if hasArg("-guard") && hxcore.DriverStrategy() == hxcore.DriverStrategyResident {
				residentGuard()
			}
			return
		case "-probe-30hx":
			probe30HX()
			return
		case "-gspensure":
			gspEnsureMain()
			return
		case "-uninstall":
			uninstall()
			return
		case "-status":
			status()
			return
		case "-task":
			// 仅注册 Gen2 登录自启任务(供 -task 模式调用;
			// 由 Go 构造 /TR 引号, 避免 bat 内嵌引号解析出错/闪退)
			regTaskOnly()
			return
		case "-h", "-help", "--help":
			printHelp()
			return
		}
	}
	install()
}

// regTaskOnly: 只注册 Gen2 SYSTEM 任务(不安装驱动/EFI/GSP)。
// -task 模式的最后一步调用本模式 — Go 处理引号。
// v2.6.0: 失败以非零码退出 — bat 的 errorlevel 检查依赖它(此前恒为 0, 检查是死代码)。
func regTaskOnly() {
	if !isAdmin() {
		fmt.Println("[!] Đăng ký tác vụ tự chạy cần quyền Quản trị viên (Administrator).")
		msgbox("Trình Cài Đặt 40HX / 30HX", "Đăng ký tác vụ tự chạy cần quyền Quản trị viên.\nVui lòng chạy với quyền Administrator.", mbIconError)
		os.Exit(1)
	}
	if err := setupGen2Task(); err != nil {
		fmt.Println("[!]", err)
		msgbox("Trình Cài Đặt 40HX / 30HX", "Đăng ký tác vụ mở khoá PCIe khi đăng nhập thất bại:\n"+err.Error()+
			"\n\nVui lòng đảm bảo chạy bằng quyền Administrator rồi thử lại.", mbIconError)
		os.Exit(1)
	}
	setRunKey()
	msgbox("Trình Cài Đặt 40HX / 30HX", "Tác vụ tự động mở khoá PCIe khi đăng nhập đã được đăng ký thành công.\nSau khi đăng nhập Windows, hệ thống sẽ tự động kích hoạt PCIe (chạy ẩn, dùng xong gỡ driver).", mbIconInfo)
}

// selfElevate: Khởi động lại với quyền Admin qua UAC ShellExecute "runas"
func selfElevate() {
	exe, _ := os.Executable()
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	args := append([]string{}, os.Args[1:]...)
	args = append(args, "-elevated")
	params, _ := syscall.UTF16PtrFromString(strings.Join(args, " "))
	r, _, _ := procShellExecuteW.Call(0,
		uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)), 0, 1)
	if r <= 32 {
		msgbox("Trình Cài Đặt 40HX / 30HX", fmt.Sprintf("Nâng quyền thất bại (Mã lỗi %d).\nVui lòng nhấp chuột phải -> Chọn 'Run as administrator'.", r), mbIconError)
	}
	os.Exit(0)
}

var (
	procShellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	gen2Succeeded     bool
)

const (
	mbIconInfo  = 0x40
	mbIconError = 0x10
	mbIconWarn  = 0x30 // MB_ICONWARNING — v2.6.0: EFI 跳过/部分成功等"可继续但要注意"场景
	mbYesNo     = 0x04 // MB_YESNO → 返回 IDYES=6 / IDNO=7
)

var (
	procMsgBoxW     = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	procCreateMutex = syscall.NewLazyDLL("kernel32.dll").NewProc("CreateMutexW")
)

func msgbox(title, text string, icon uint) {
	// -y / -silent(自动化/自启动) 时不弹框
	if hasArg("-y") || hasArg("-silent") {
		return
	}
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	procMsgBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), uintptr(icon))
}

// msgboxYesNo: 是/否询问。自动模式: -y→true(全自动继续), -silent→false(不打扰)。
func msgboxYesNo(title, text string) bool {
	if hasArg("-y") {
		return true
	}
	if hasArg("-silent") {
		return false
	}
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := procMsgBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), uintptr(mbYesNo|mbIconInfo))
	return r == 6 // IDYES
}

// setupLog: 输出镜像到日志文件(默认 %TEMP%/<name>, 命令行 -log <file> 优先)
func setupLog(defName string) {
	p := filepath.Join(os.TempDir(), defName)
	if i := argIndex("-log"); i >= 0 && i+1 < len(os.Args) {
		p = os.Args[i+1]
	}
	if f, err := os.Create(p); err == nil {
		os.Stdout = f
		os.Stderr = f
		fmt.Fprintf(f, "==== 40HX tool %s ====\n", time.Now().Format("2006-01-02 15:04:05"))
	}
}

// AttachLogSink: v2.6.0 GUI 用 — 用 os.Pipe 把后续 fmt.* 输出分流到 日志文件+UI。
// fmt.* 每次调用读 os.Stdout 变量; 但 os.Stdout 本身是 *os.File 具体类型,
// 不能赋 io.Writer, 故替换为管道写端, 由读协程同时写原文件与 GUI 日志面板。
func AttachLogSink(w io.Writer) {
	r, pw, err := os.Pipe()
	if err != nil {
		return
	}
	orig := os.Stdout // setupLog 建立的日志文件(或 GUI 下的无效控制台句柄)
	os.Stdout = pw
	os.Stderr = pw
	go func() {
		defer r.Close()
		buf := make([]byte, 4096)
		for {
			n, rerr := r.Read(buf)
			if n > 0 {
				orig.Write(buf[:n]) // 落日志文件(GUI 模式下失败可忽略)
				w.Write(buf[:n])    // 喂 GUI 日志面板
			}
			if rerr != nil {
				return
			}
		}
	}()
}

// lockOnce: 单实例互斥; 返回 nil 表示已有实例在跑
func lockOnce(name string) func() {
	n, _ := syscall.UTF16PtrFromString(name)
	h, _, e := procCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(n)))
	if h == 0 {
		return nil
	}
	if e == syscall.ERROR_ALREADY_EXISTS {
		syscall.CloseHandle(syscall.Handle(h))
		return nil
	}
	return func() { syscall.CloseHandle(syscall.Handle(h)) }
}

func hasArg(name string) bool {
	for _, a := range os.Args {
		if a == name {
			return true
		}
	}
	return false
}

func argIndex(name string) int {
	for i, a := range os.Args {
		if a == name {
			return i
		}
	}
	return -1
}

func printHelp() {
	fmt.Println("Trình Mở Khoá & Kích Hoạt PCIe CMP 40HX / 30HX trên Windows")
	fmt.Println("  Cách dùng: 40HXInstaller.exe                  # Cài đặt giao diện / toàn bộ (cần Admin)")
	fmt.Println("             40HXInstaller.exe -gen2            # Mở khoá Gen2 ngay lập tức")
	fmt.Println("             40HXInstaller.exe -gen3            # (CMP 30HX) Mở khoá Gen3 ngay lập tức")
	fmt.Println("             40HXInstaller.exe -force-root-gen2 # (CMP 30HX) Ép Root Port huấn luyện lại Gen2")
	fmt.Println("             40HXInstaller.exe -force-root-gen3 # (CMP 30HX) Ép Root Port huấn luyện lại Gen3")
	fmt.Println("             40HXInstaller.exe -gen2-30hx       # (CMP 30HX) MMIO ghi đè + Huấn luyện lại Gen2")
	fmt.Println("             40HXInstaller.exe -gen3-30hx       # (CMP 30HX) MMIO ghi đè + Huấn luyện lại Gen3")
	fmt.Println("             40HXInstaller.exe -probe-30hx      # (CMP 30HX) Đọc thanh ghi BAR0 MMIO chẩn đoán")
	fmt.Println("             40HXInstaller.exe -uninstall       # Gỡ cài đặt / Khôi phục hệ thống")
	fmt.Println("             40HXInstaller.exe -status          # Kiểm tra trạng thái hiện tại")
}

// ===================== 底层 =====================

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
		// TokenElevation 可能因受限环境(沙箱/服务)误报 0, 再试 SCM 全权
	}
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_ALL_ACCESS)
	if err == nil {
		windows.CloseServiceHandle(scm)
		return true
	}
	return false
}

// enableGsp: 设 EnableGpuFirmware=1 (需管理员)
func enableGsp() error {
	key := hxcore.FindGpuClassKey()
	if key == "" {
		return errors.New("找不到 40HX 的设备注册表键 (Class 子键)")
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetDWordValue(gpuEnableFw, 1)
}

// disableGsp: 删 EnableGpuFirmware (卸载用, 恢复默认关)
func disableGsp() {
	key := hxcore.FindGpuClassKey()
	if key == "" {
		return
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, key, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	k.DeleteValue(gpuEnableFw)
}

// ensureGspSilent: 确保 GSP 启用 (EnableGpuFirmware=1)。
// 供 -gen2(登录自启动)调用: 若 GSP 被改回(≠1)则重新启用。
// 写 HKLM 需管理员: 当前是管理员直接写; 否则注册一次性 SYSTEM 计划任务
// (SYSTEM 权限写 HKLM 无需 UAC, 无窗口)。
// 返回 true = GSP 已启用或已安排重设。
func ensureGspSilent() bool {
	if hxcore.GspEnabled() {
		return true // 已启用
	}
	fmt.Println("[GSP] EnableGpuFirmware 被改回, 重新启用...")
	if isAdmin() {
		if err := enableGsp(); err != nil {
			fmt.Println("[GSP] 重设失败:", err)
			return false
		}
		fmt.Println("[GSP] 已重设 EnableGpuFirmware=1 (重启后 GSP-RM 生效)")
		return true
	}
	// 非管理员: 用 SYSTEM 计划任务一次性重设 (无 UAC 弹窗)
	exe, _ := os.Executable()
	abs, _ := filepath.Abs(exe)
	tn := "40HXGspEnsure"
	if out, err := hxcore.RunOut("schtasks.exe", "/create", "/tn", tn,
		"/tr", fmt.Sprintf("\"%s\" -gspensure -silent", abs),
		"/sc", "once", "/st", "00:00", "/ru", "SYSTEM", "/f"); err != nil {
		fmt.Printf("[GSP] 计划任务创建失败: %s\n", strings.TrimSpace(out))
		return false
	}
	hxcore.RunOut("schtasks.exe", "/run", "/tn", tn)
	hxcore.RunOut("schtasks.exe", "/delete", "/tn", tn, "/f")
	fmt.Println("[GSP] 已通过 SYSTEM 任务重设 EnableGpuFirmware=1")
	return true
}

// gspEnsureMain: -gspensure 模式 (SYSTEM 计划任务调用, 只重设 GSP 后退出)
func gspEnsureMain() {
	if isAdmin() {
		if err := enableGsp(); err != nil {
			fmt.Println("[GSP] gspensure 重设失败:", err)
			return
		}
		fmt.Println("[GSP] gspensure: EnableGpuFirmware=1 已设置")
	}
}

func copyEmbedTo(target string, src string) error {
	data, err := embedded.ReadFile("embed/" + src)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

// deployEspEfi: 双路部署 40HXUNLK.EFI 到已挂载的 ESP <esp>。
//
//	A. \EFI\40HX\40HXUNLK.EFI   — BCD 启动项引用路径
//	B. \EFI\Boot\bootx64.efi    — UEFI 标准回退路径 (固件无条件尝试的最后手段;
//	   解决社区大量"装完重启直接进 Windows 没跑解锁"——主板忽略非标准目录)
//
// 备份规则: 若目标 bootx64.efi 存在且不是本工具部署过的副本, 先备份为
//
//	bootx64.efi.40hx.bak (卸载时恢复)。已部署过(.bak 已存在)则直接覆盖。
//
// 返回 fallback 是否新备份了原文件。
func deployEspEfi(esp string) (backedUp bool, err error) {
	// 读取 embed 一次, 两个路径共用
	data, rerr := embedded.ReadFile("embed/40HXUNLK.EFI")
	if rerr != nil {
		return false, rerr
	}
	// 写盘前校验 embed 数据本身完整 (PE 头 + 长度合理, 防 embed 损坏)
	if len(data) < 0x2000 { // < 8KB 的 EFI 文件必为损坏
		return false, fmt.Errorf("内嵌 40HXUNLK.EFI 数据异常 (%d bytes)", len(data))
	}
	if !bytes.HasPrefix(data, []byte("MZ")) {
		return false, errors.New("内嵌 40HXUNLK.EFI 不是有效 PE 镜像(缺 MZ 头)")
	}

	// A. 主路径
	dirA := esp + ":" + efiDir // Y:\EFI\40HX
	if merr := os.MkdirAll(dirA, 0o644); merr != nil {
		return false, merr
	}
	pA := filepath.Join(dirA, efiFile)
	if werr := writeVerified(pA, data); werr != nil {
		// 写失败或校验不一致 → 删掉可能半截的文件, 避免被 BCD 引用成坏引导
		os.Remove(pA)
		return false, werr
	}
	fmt.Printf("    [A] %s  (%d bytes, 校验 OK)\n", "\\EFI\\40HX\\"+efiFile, len(data))

	// B. 标准回退路径
	dirB := esp + ":" + efiStdDir // Y:\EFI\Boot
	if merr := os.MkdirAll(dirB, 0o644); merr != nil {
		return false, merr
	}
	pB := filepath.Join(dirB, efiStdF) // bootx64.efi
	pBak := pB + efiBakExt             // bootx64.efi.40hx.bak
	if _, berr := os.Stat(pBak); berr != nil {
		// 无备份记录 → 若目标存在且不是我们已部署的副本, 先备份
		if old, oerr := os.ReadFile(pB); oerr == nil && !bytes.Equal(old, data) {
			if cerr := os.Rename(pB, pBak); cerr != nil {
				return false, fmt.Errorf("备份原 %s 失败: %v", pB, cerr)
			}
			fmt.Printf("    [B] 原 %s 已备份为 %s\n", efiStdF, efiStdF+efiBakExt)
			backedUp = true
		} else if oerr != nil {
			// 目标不存在: 无备份(本来就是空位)
		}
	}
	if werr := writeVerified(pB, data); werr != nil {
		os.Remove(pB)
		return backedUp, werr
	}
	fmt.Printf("    [B] %s  (%d bytes, 校验 OK)\n", "\\EFI\\Boot\\"+efiStdF, len(data))
	return backedUp, nil
}

// writeVerified: 写文件后立即读回比对 — 防止写入中断/半截导致引导损坏。
// 不一致则删除并返回错误(调用方据此中止, 不让坏文件留在引导路径)。
func writeVerified(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	rb, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("写后校验读取失败 %s: %v", path, err)
	}
	if !bytes.Equal(rb, data) {
		return fmt.Errorf("写后校验不一致 %s (%d ≠ %d bytes)", path, len(rb), len(data))
	}
	return nil
}

// alreadyInstalled: 检测是否已安装过(避免无意义/重复的覆盖安装)。
// 判据: ① 固件启动项 "40HX Unlock" 存在; ② ESP 上已有 \EFI\40HX\40HXUNLK.EFI。
// 任一命中即认为装过 — 用于重入提示(不会因此阻止用户, 仅弹确认)。
func alreadyInstalled() bool {
	// ① bcdedit 固件枚举(不挂 ESP, 快速)
	if out, _ := hxcore.RunOut("bcdedit.exe", "/enum", "firmware"); strings.Contains(out, bootDesc) {
		return true
	}
	// ② ESP 文件
	esp := hxcore.MountESP()
	if esp == "" {
		return false // 挂不上 ESP 时保守视为未装(后面 [5/8] 会报错引导)
	}
	defer hxcore.UnmountESP(esp)
	if _, err := os.Stat(esp + ":" + efiDir + "\\" + efiFile); err == nil {
		return true
	}
	return false
}

// verifyBootEntry: 读回 {fwbootmgr} displayorder, 确认 40HX Unlock 是否在首位。
// 返回 (exists, isFirst, displayOrder描述)。
// 用 bcdedit /enum firmware 读固件 NVRAM — 若固件忽略 bcdedit 的写入,
// 这里会如实反映(不在列表/不在首位), 从而让安装器给出 BIOS 手动指引。
// 注意: bcdedit 输出为 GBK, 中文系统"标识符/说明"是乱码; 但字段值
// (guid / displayorder / 40HX Unlock / path) 均为 ASCII, 按块解析可靠。
func verifyBootEntry() (bool, bool, string) {
	out, err := hxcore.RunOut("bcdedit.exe", "/enum", "firmware")
	if err != nil {
		return false, false, "(bcdedit 读取失败: " + err.Error() + ")"
	}
	lines := strings.Split(out, "\r\n")
	if len(lines) < 2 {
		lines = strings.Split(out, "\n")
	}

	// 1. 收集 displayorder 下的 GUID 序列(固件实际启动顺序)
	var order []string
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "displayorder") {
			// 首个 GUID 可能同行: "displayorder {guid}"
			if m := guidRe().FindString(t); m != "" {
				order = append(order, strings.Trim(m, "{}"))
			}
			// 后续缩进行 {guid}
			for j := i + 1; j < len(lines); j++ {
				s := strings.TrimSpace(lines[j])
				if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
					order = append(order, strings.Trim(s, "{}"))
				} else if s != "" {
					break
				}
			}
			break // displayorder 只在 {fwbootmgr} 段, 取首个即可
		}
	}

	// 2. 找 description 为 "40HX Unlock" 的块的 GUID
	target := ""
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "description") &&
			strings.Contains(lines[i], bootDesc) {
			// 往上找最近的 {guid} 行 = 该块 identifier
			for j := i - 1; j >= 0 && j > i-6; j-- {
				if m := guidRe().FindString(lines[j]); m != "" {
					target = strings.Trim(m, "{}")
					break
				}
			}
			break
		}
	}
	if target == "" {
		joined := strings.Join(order, " > ")
		if joined == "" {
			joined = "(固件无 displayorder 条目)"
		}
		return false, false, joined
	}
	if len(order) == 0 {
		return true, false, "(displayorder 为空)"
	}
	isFirst := order[0] == target
	return true, isFirst, strings.Join(order, " > ")
}

var _guidRe = regexp.MustCompile(`\{([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\}`)

func guidRe() *regexp.Regexp { return _guidRe }

// ===================== 安装 =====================

// applyPowerSettings: 快速启动 + PCIe ASPM 两项电源优化(v2.6.0 [3.6/8] 段抽取,
// v2.6.0 GUI 策略页复用)。幂等: 原本已关则不动; 返回逐项说明行。
func applyPowerSettings() []string {
	notes := []string{}
	if hxcore.FastStartupOn() {
		if err := hxcore.SetFastStartupOff(); err != nil {
			notes = append(notes, fmt.Sprintf("快速启动关闭失败: %v (不影响安装, 建议电源选项手动关)", err))
		} else {
			notes = append(notes, "快速启动已关闭(原为开): 关机将走完整 UEFI 引导; 电源选项可恢复")
		}
	} else {
		notes = append(notes, "快速启动: 原本已关(OK)")
	}
	if ac, dc, ok := hxcore.ASPMSavings(); !ok {
		notes = append(notes, "PCIe ASPM: 本机未公开该设置, 跳过")
	} else if ac == 0 && dc == 0 {
		notes = append(notes, "PCIe ASPM: 原本已关(OK)")
	} else {
		if err := hxcore.SetASPMOff(); err != nil {
			notes = append(notes, fmt.Sprintf("ASPM 关闭失败: %v", err))
		} else {
			notes = append(notes, fmt.Sprintf("PCIe ASPM 已关闭(原 AC=%d/DC=%d): 减少空闲降到 Gen1; 恢复: powercfg 命令见 README", ac, dc))
		}
	}
	return notes
}

// installEFI: ESP 双路部署 40HXUNLK.EFI + 固件启动项(v2.6.0 [5/8]+[6/8] 段抽取,
// v2.6.0 GUI 组件安装页复用)。返回 EFI 是否部署成功;
// [7/8] Gen2 任务注册不依赖此结果(EFI 失败只跳过 EFI 两步 — 社区 #2/#5/#6/#7 统一根因修复)。
func installEFI() bool {
	//    主路径  \EFI\40HX\40HXUNLK.EFI  — BCD 启动项引用
	//    fallback \EFI\Boot\bootx64.efi   — UEFI 标准回退路径, 解决部分主板
	//    忽略 BCD displayorder / 不认非标准目录(社区"装完重启没反应"主因)。
	//    原 bootx64.efi 备份为 bootx64.efi.40hx.bak, 卸载时恢复。
	efiOK := false
	fmt.Println("    · 部署解锁 EFI 到系统 EFI 分区(双路)...")
	esp := hxcore.MountESP()
	if esp == "" {
		if hxcore.FirmwareIsLegacy() {
			fmt.Println("[!] 本系统为传统 BIOS(Legacy)+MBR 引导 — 没有 EFI 分区, 解锁 EFI 无法部署。")
			fmt.Println("    算力解锁需要 UEFI+GPT: 请先用微软 mbr2gpt 无损转换(完整步骤见弹窗),")
			fmt.Println("    转换完成并改 UEFI 引导后重跑本安装器。")
			fmt.Println("    [i] Gen2 登录自启不受影响, 继续注册(见 [7/8])。")
			msgbox("40HX 安装器 (需要先转换硬盘为 GPT)",
				"本系统是传统 BIOS(Legacy)+MBR 引导, 没有 EFI 分区,\n"+
					"算力解锁 EFI 无法部署 — 这就是\"EFI 装不上\"的原因。\n\n"+
					"请先转成 UEFI+GPT(微软官方无损转换, 不动数据):\n"+
					"  1. 备份重要数据; 确认未启用 BitLocker(有则先暂停)\n"+
					"  2. 管理员命令提示符运行:  mbr2gpt /validate /allowfullos\n"+
					"  3. 显示 Validation completed successfully 后运行:\n"+
					"        mbr2gpt /convert /allowfullos\n"+
					"  4. 重启进 BIOS, 把启动模式从 Legacy 改为 UEFI(关 CSM)\n"+
					"  5. 进 Windows 后重新运行本安装器\n\n"+
					"注意: 转换不可逆; 需 Win10 1703+ / Win11 且主板支持 UEFI。\n"+
					"本次安装将继续完成 Gen2 部分(算力解锁等转换后重跑安装器)。",
				mbIconWarn)
		} else {
			fmt.Println("[!] 无法挂载 EFI 分区(mountvol /S 失败)")
			fmt.Println("    系统是 UEFI, 常见原因: BitLocker/第三方加密未暂停、ESP 分区异常。")
			fmt.Println("    可手动: mountvol S: /S, 复制 40HXUNLK.EFI 到 S:\\EFI\\40HX\\, mountvol S: /D")
			msgbox("40HX 安装器 (EFI 分区挂载失败)",
				"无法挂载 EFI 分区 (mountvol /S 失败), 解锁 EFI 本次未部署。\n"+
					"系统引导不受影响。\n\n"+
					"常见原因: BitLocker/第三方加密未暂停、ESP 分区异常。\n"+
					"可手动部署(见日志与《EFI应急修复指南.md》)。\n\n"+
					"本次安装将继续完成 Gen2 部分, 算力解锁待 EFI 部署成功后生效。",
				mbIconWarn)
		}
		return false
	}
	fmt.Printf("    ESP 挂载于 %s: \\\n", esp)
	fb, err := deployEspEfi(esp)
	hxcore.UnmountESP(esp)
	if err != nil {
		fmt.Println("[!] 复制 EFI 失败:", err)
		msgbox("40HX 安装器 (EFI 写入失败)",
			"复制解锁 EFI 到 ESP 失败(已做写后校验, 坏文件不会残留):\n"+err.Error()+
				"\n\n系统引导未受影响, 重启应能正常进 Windows。\n\n"+
				"如需手动部署, 见同目录《EFI应急修复指南.md》中\n"+
				"“手动部署”一节。\n\n"+
				"本次安装将继续完成 Gen2 部分。", mbIconWarn)
		return false
	}
	if fb {
		fmt.Println("    [!] 检测到原 bootx64.efi, 已备份为 bootx64.efi.40hx.bak")
	}
	efiOK = true

	// BootOrder (v2.4: 写回验证 + BIOS 指引弹框); 仅 EFI 部署成功才执行
	fmt.Println("    · 设置固件启动项(40HX Unlock 置顶)...")
	bootOK := false
	if err := setupBootEntry(); err != nil {
		fmt.Println("[!] 自动设置启动项失败:", err)
	} else {
		if ex, first, ord := verifyBootEntry(); ex {
			bootOK = first
			if first {
				fmt.Println("    启动项已置顶并验证通过 (固件 displayorder 首位)")
			} else {
				fmt.Println("    [!] 启动项已创建, 但不在 displayorder 首位:")
				fmt.Println("        当前固件顺序: " + ord)
				fmt.Println("        请进 BIOS 手动将 '40HX Unlock' 设为第一启动项(见弹窗)")
			}
		} else {
			fmt.Println("    [!] 未能在固件启动列表中找到 '40HX Unlock' 项")
			fmt.Println("        (部分主板忽略 BCD 写入, 请进 BIOS 手动添加/置顶)")
		}
	}
	if !bootOK {
		// BIOS 指引弹窗 (社区用户不看日志/README 的关键一步)
		msgbox("40HX 安装器 (重要: 请按提示操作)",
			"自动启动项未被固件接受。\n"+
				"请重启并按 Del/F2 进 BIOS, 完成以下设置(否则不解锁):\n\n"+
				"1. 关闭 Secure Boot(已开则未签名 EFI 会被拒)\n"+
				"2. 关闭 Fast Boot / 快速启动(若有)\n"+
				"3. 在 [启动顺序/Boot Priority] 中把 '40HX Unlock' 设为第一项\n"+
				"   或手动从启动设备选择 \\EFI\\40HX\\40HXUNLK.EFI\n"+
				"4. 若列表只有 Windows Boot Manager:\n"+
				"   - 部分主板需关闭 CSM(纯 UEFI)后才会出现该启动项\n"+
				"   - 或直接选 UEFI 盘符启动(走 bootx64 回退)\n\n"+
				"安装器已把解锁 EFI 同时部署到:\n"+
				"  \\EFI\\40HX\\40HXUNLK.EFI  (BCD 路径)\n"+
				"  \\EFI\\Boot\\bootx64.efi    (标准回退路径)\n\n"+
				"详细日志: "+filepath.Join(os.TempDir(), "40HX_installer.log"),
			mbIconError)
	}
	return efiOK
}

func install() {
	fmt.Println("==============================================")
	fmt.Println("  CMP 40HX Windows Unlock Installer v3.0.0")
	fmt.Println("  Tensor 解锁(EFI V70 + GSP 启用) + PCIe Gen2 + 自启动")
	fmt.Println("==============================================")

	if !isAdmin() {
		fmt.Println("[!] 需要管理员权限。")
		msgbox("40HX 安装器", "需要管理员权限。\n请右键本程序 -> 以管理员身份运行。", mbIconError)
		return
	}
	if lockOnce(`Local\40HXInstaller_v1`) == nil {
		msgbox("40HX 安装器", "安装器已在运行, 请勿重复点击。", mbIconInfo)
		return
	}

	// 0. 重入检测: 已装过(固件启动项/GSP 键已存在) → 确认后再覆盖,
	//    避免用户误以为需要反复安装、或在不知情下覆盖现有部署。
	if alreadyInstalled() {
		fmt.Println("[!] 检测到 40HX 解锁已安装过(启动项/GSP 键存在)。")
		if !msgboxYesNo("40HX 安装器",
			"检测到 40HX 解锁已安装过。\n\n"+
				"再次安装会覆盖现有部署(驱动与启动项会更新, 不会损坏系统引导)。\n"+
				"如果是想修复异常/升级, 选\"是\"继续;\n"+
				"如果只是误打开, 选\"否\"保持现状即可。\n\n"+
				"继续重新安装?") {
			fmt.Println("已取消 — 保持现有安装不变。")
			return
		}
		fmt.Println("    用户确认, 继续覆盖安装。")
	}

	// 1. GPU 检测
	fmt.Print("[1/8] 检测 GPU ... ")
	if !hxcore.FindGPU() {
		fmt.Println("未找到 " + gpuVenDev)
		fmt.Println("[!] 未检测到 CMP 40HX。中止。")
		msgbox("40HX 安装器", "未检测到 CMP 40HX 显卡 (VEN_10DE&DEV_1F0B)。\n安装中止。", mbIconError)
		return
	}
	fmt.Println("CMP 40HX 已找到")

	// 2. Secure Boot
	fmt.Print("[2/8] Secure Boot 检查 ... ")
	if hxcore.SecureBootOn() {
		fmt.Println("开启!")
		fmt.Println("[!] Secure Boot 开启时, 未签名 EFI(40HXUNLK) 会被固件拒绝。")
		msgbox("40HX 安装器 (需要关闭 Secure Boot)",
			"检测到 Secure Boot 开启, 未签名的解锁 EFI 会被固件拒绝。\n\n"+
				"请重启进 BIOS 关闭后再运行本安装器:\n"+
				"  1. 重启, 开机按 Del / F2(部分主板 F1/F10/F12)进 BIOS\n"+
				"  2. 找 Security / Boot / 启动 选项卡\n"+
				"  3. 将 Secure Boot 设为 Disabled\n"+
				"     (若灰显, 先设 CSM/兼容模式 或恢复默认安全设置)\n"+
				"  4. 保存退出(F10)后重新运行本程序\n\n"+
				"这是解锁必需的: 40HX 解锁 EFI 无微软签名。",
			mbIconError)
		return
	}
	fmt.Println("关闭/不可用(OK)")

	// 3. 测试签名 (v2.5 不需要 — BYOVD 预签名驱动普通模式即可加载)
	fmt.Print("[3/8] 测试签名 ... ")
	if hxcore.TestSigningOn() {
		fmt.Println("已开启 — v2.5 不需要, 装完可 bcdedit /set testsigning off 关闭")
	} else {
		fmt.Println("关闭(OK) — v2.5 全程免测试签名")
	}

	// 3.5 GSP 启用 (v2.3: 解锁不黑屏的关键!)
	// 40HX 默认 GSP 关(CPU-RM 模式) -> EFI 解锁后 nvlddmkm 拒绝 -> Code43 黑屏
	// EnableGpuFirmware=1 -> GSP-RM 管理 SEC2/booter -> 接受解锁状态
	fmt.Print("[3.5/8] 启用 GSP (EnableGpuFirmware) ... ")
	if hxcore.GspEnabled() {
		if sub, _, fw := hxcore.GspDiag(); sub != "" {
			fmt.Printf("已启用(OK) — Class\\%s EnableGpuFirmware=%d\n", sub, fw)
		} else {
			fmt.Println("已启用(OK)")
		}
	} else {
		if err := enableGsp(); err != nil {
			// v2.4.1: 附带 AdapterString 诊断 — 伪装驱动(雨糖识别成2070等)会命中此分支
			_, adapterDiag, _ := hxcore.GspDiag()
			fmt.Println("设置失败:", err)
			if adapterDiag != "" && !strings.Contains(adapterDiag, "无 CMP 40HX") {
				fmt.Println("    [!] 实际 AdapterString:", adapterDiag)
			} else if adapterDiag != "" {
				fmt.Println("    [!]", adapterDiag)
			}
			fmt.Println("    [!] 若驱动是伪装版(识别成 2070 等): 换未伪装版驱动或手动设 GSP")
			msgbox("40HX 安装器", "设置 EnableGpuFirmware=1 失败(需管理员)。\n解锁后可能黑屏/掉驱动。\n错误: "+err.Error()+"\n若驱动是伪装版(识别成2070等),请换未伪装驱动或用 -status 查 AdapterString。", mbIconError)
			return
		}
		fmt.Println("已设 EnableGpuFirmware=1 (重启生效)")
		fmt.Println("    [!] GSP 必需: 否则 EFI 解锁后驱动不认 -> Code43 黑屏")
	}

	// 3.6 系统电源设置 (v2.6.0: 社区 v2.4.5 排障结论)
	//     快速启动: "关机→再开"走休眠恢复, 不做完整 UEFI 引导, EFI 可能不跑
	//     PCIe ASPM: 开启时空闲会降到 Gen1, 登录后实测容易被误读成"Gen2 失败"
	//     两项幂等设置, 只在当前为开时改; 均可在电源选项恢复, 不碰其他电源策略
	fmt.Print("[3.6/8] 电源设置(快速启动 + PCIe 链路省电) ... ")
	pwrNotes := applyPowerSettings()
	fmt.Println("完成")
	for _, n := range pwrNotes {
		fmt.Println("    - " + n)
	}

	// 4. 驱动安装
	fmt.Println("[4/8] 准备 Gen2 BYOVD 驱动(ThrottleStop + WinRing0)...")
	installDrivers()

	// 4.5 Defender 精确排除(防杀软误删驱动文件导致 Gen2 自启失败)
	//     只加我们自己的驱动/备份/发布目录, 不关任何系统防护。
	fmt.Print("[4.5/8] Defender 排除(防误删) ... ")
	if err := hxcore.AddDefenderExclusions(); err != nil {
		fmt.Println("未执行(可忽略):", err)
	} else {
		fmt.Println("已加白 ThrottleStop/WinRing0 驱动文件与备份目录")
	}

	// 5+6. EFI 部署与启动项 (v2.6.0: 抽取为 installEFI, GUI 按组件复用)
	fmt.Println("[5/8]+[6/8] 部署解锁 EFI 与固件启动项(双路写入 + displayorder 置顶)...")
	efiOK := installEFI()

	// 7. Gen2 自启动(安装时不 retrain!)
	// 重要: 安装过程中绝不执行 Gen2 PCIe 重训。此时 nvlddmkm 正占用 GPU,
	// 强行 retrain 会让 GPU/链路进入异常状态, 导致下次开机 EFI 接力或
	// nvlddmkm 初始化失败(实测: 设备报 code19 / Windows 启动异常进安全模式)。
	// 正确时机 = 重启后登录时执行(与手动方案一致, 已验证稳定)。
	// v2.6.0: 两路互斥串行设计 — Run 键登录瞬间先试 + SYSTEM 任务延迟30s确认;
	// 单实例互斥体(gen2AcquireSingleInstance)保证二者不会同时进入驱动加载临界区。
	// Run 键在普通权限下无法 sc start 驱动 → 自动交权给 SYSTEM 任务(静默)。
	fmt.Println("[7/8] 注册 Gen2 登录自启动(SYSTEM 任务 + Run 键, 互斥串行)...")
	setRunKey()
	if err := setupGen2Task(); err != nil {
		// v2.6.0: 任务是 Gen2 链的命脉, 注册失败必须让用户看见并可一键修复
		fmt.Println("[!]", err)
		msgbox("40HX 安装器 (Gen2 自启注册失败)",
			"Gen2 登录自启任务注册失败 — 登录后不会自动解锁 Gen2。\n\n"+
				"请稍后右键以管理员身份运行一次:\n"+
				"  40HXInstaller.exe -task\n\n"+
				"其余安装步骤已完成。", mbIconWarn)
	}

	fmt.Println()
	fmt.Println("安装完成!")
	if efiOK {
		fmt.Println("  下次重启: 固件将自动运行 40HX Unlock (Tensor 解锁) -> 自动进 Windows")
	} else {
		fmt.Println("  [!] EFI 算力解锁本次未部署(见 [5/8] 说明) — 算力暂不会解锁,")
		fmt.Println("      按 [5/8] 弹窗指引(mbr2gpt/手动部署)处理后重跑本安装器即可。")
	}
	fmt.Println("  GSP 已启用: 驱动以 GSP-RM 模式接管 GPU, 解锁后不再黑屏/掉驱动")
	fmt.Println("  登录后: Gen2 自动解锁 (已注册自启动, 无窗口静默)")
	fmt.Println("  [!] 安装时不重训 PCIe, 重启后登录时才执行(避免与显卡驱动冲突)")
	fmt.Println("  重启后验证: 双击 40HXCheck.exe 查看解锁状态(SS0=0x88888888 即成功)")
	fmt.Println("  若 testsigning 刚开启: 请先重启一次使驱动可加载")
	// v2.4: 完成弹框含关键 BIOS/重启指引(社区用户不依赖 README 也能操作)
	// v2.6.0: EFI 成败给出不同指引; 告知电源设置已自动调整及恢复方式
	efiNote := ""
	if efiOK {
		efiNote = "重启时请注意:\n" +
			"  · 若黑屏/显示 40HX 文字日志约 10~30 秒, 属正常(正在解锁)\n" +
			"  · 解锁完成后会自动进入 Windows\n\n" +
			"若重启后直接进了 Windows(没跑解锁), 请进 BIOS(Del/F2):\n" +
			"  1. 关闭 Secure Boot(未签名 EFI 需要)\n" +
			"  2. 关闭 Fast Boot\n" +
			"  3. 把 '40HX Unlock' 设为第一启动项\n" +
			"     (若列表只有 Windows Boot Manager, 关 CSM 后再看)\n"
	} else {
		efiNote = "[!] 本次 EFI 算力解锁未部署(原因见上方弹窗/日志):\n" +
			"  · 算力暂不会解锁, 按指引处理后重跑安装器即可\n" +
			"  · Gen2 自启已注册, 不受影响\n"
	}
	msgbox("40HX 安装器 (安装完成)",
		"✅ 安装完成! "+map[bool]string{true: "重启后将自动执行解锁。", false: "Gen2 部分已就绪。"}[efiOK]+"\n\n"+
			efiNote+
			"\n重启进系统后:\n"+
			"  · 双击同目录的 40HXCheck.exe 验证 — 显示\n"+
			"    '解锁成功: Tensor 满血(SS0=0x88888888)' 即完成\n"+
			"  · 若提示未解锁, 它会给下一步(如开 Above 4G)\n\n"+
			"· 测试签名若刚开启: 先重启一次驱动才可加载\n"+
			"· GSP 已启用(EnableGpuFirmware=1): 解锁不黑屏的关键\n"+
			"· 已自动关闭快速启动与 PCIe 链路省电(ASPM):\n"+
			"  前者保证关机再开也走完整 UEFI 引导, 后者减少空闲降到 Gen1;\n"+
			"  恢复方式见 README §2.4\n"+
			"· 登录后 Gen2 自动解锁(静默)\n\n"+
			"详细日志: "+filepath.Join(os.TempDir(), "40HX_installer.log"),
		mbIconInfo)
}

func installDrivers() {
	sysDir := os.Getenv("SystemRoot") + "\\System32\\drivers"
	svcRunning := func(name string) bool {
		out, _ := hxcore.RunOut("sc.exe", "query", name)
		return strings.Contains(out, "RUNNING")
	}
	// v2.5: 不再常驻 40hx_bridge(需测试签名)。Gen2 改 BYOVD:
	//   ThrottleStop(任意物理内存写, EV 预签名) + WinRing0(PCI config) —
	//   两者普通模式(testsigning off)即可加载。安装阶段仅放好驱动文件 +
	//   注册 demand 服务; 真正的加载与自清理由登录后的 -gen2(SYSTEM 任务)
	//   完成 → 用完即卸, 游戏时系统无第三方驱动。
	tsApp := throttleStopAppRunning()
	for _, d := range []struct{ name, file string }{
		{"ThrottleStop", "ThrottleStop.sys"},
		{"WinRing0_1_2_0", "WinRing0x64.sys"},
	} {
		dst := filepath.Join(sysDir, d.file)
		// 本机装了 ThrottleStop 软件 → 复用其同名驱动, 绝不覆盖/删除(避免冲突+写保护)
		if tsApp {
			fmt.Printf("  检测到 ThrottleStop 软件, 复用其 %s 驱动(不覆盖/不删)\n", d.name)
			continue
		}
		if svcRunning(d.name) {
			fmt.Printf("  %s 已在运行, 跳过覆盖(保持当前状态)\n", d.name)
			continue
		}
		hxcore.RunOut("sc.exe", "stop", d.name)
		// 留一份到 %ProgramData%\40HXUnlock\drivers 作为持久备份源
		// (40HXCheck 实测/Gen2 临时部署都从这里取; System32 的会被用完即卸删除)
		pdDir := filepath.Join(os.Getenv("ProgramData"), "40HXUnlock", "drivers")
		os.MkdirAll(pdDir, 0o755)
		copyEmbedTo(filepath.Join(pdDir, d.file), d.file)
		if err := copyEmbedTo(dst, d.file); err != nil {
			if _, statErr := os.Stat(dst); statErr != nil {
				fmt.Printf("  [!] 复制 %s 失败: %v\n", d.file, err)
				continue
			}
		} else {
			fmt.Printf("  已复制 %s\n", d.file)
		}
		ensureService(d.name, d.file)
	}
	fmt.Println("  Gen2 驱动文件已就绪(demand), 登录后由 SYSTEM 任务临时加载并自清理")
}

// ensureService: 仅注册(或更新)驱动服务, 不在此处加载。
// 安装阶段加载 40hx_bridge(映射 GPU BAR0)会与正在运行的 nvlddmkm 争用硬件,
// 实测导致 40HX 设备报 code19 / 后续启动异常。加载推迟到重启后登录时的 -gen2。
// v2.4.6 关键修复(社区 #1/#2 根因):
//
//	驱动服务注册为 start=demand(手动), 需在登录后由 -gen2 拉起。
//	而 -gen2 走 Run 键以普通用户权限运行 → sc start 需要管理员 →
//	"[SC] StartService: OpenService 失败 5: 拒绝访问" → 驱动永远起不来
//	→ Gen2 永远失败(用户现象: 算力解锁 OK 但 Gen2 ✗)。
//	正解 = 保持 demand(不改成 auto! 详见下), 并把 -gen2 的执行权限升到
//	SYSTEM: 注册 SYSTEM 计划任务(登录时触发 + 延迟 30s)跑 -gen2 -silent,
//	既不需要 UAC 弹窗, 又保留"登录后才加载驱动"的安全时序。
//
// 为什么不改成 start=auto: type=kernel auto 驱动在开机早期由 SCM 加载,
// 会与随后初始化的 nvlddmkm 争用 GPU BAR0 — 历史上实测导致 40HX 报
// code19 / Windows 启动异常进安全模式。demand + 登录后加载是经过验证的时序。
func ensureService(name string, sysFile string) {
	bin := fmt.Sprintf("\\SystemRoot\\System32\\drivers\\%s", sysFile)
	// 创建(已存在会失败, 忽略); 启动类型 demand — 由 SYSTEM 任务登录后拉起
	hxcore.RunOut("sc.exe", "create", name, "type=", "kernel", "start=", "demand", "binPath=", bin)
	out, err := hxcore.RunOut("sc.exe", "query", name)
	if err != nil || !strings.Contains(out, "STATE") {
		fmt.Printf("  [!] 注册服务 %s 失败: %s\n", name, strings.TrimSpace(out))
		return
	}
	// 纠正被安全软件/策略改错的启动类型(Disabled 会导致 Gen2 永远拉不起)。
	// 启动类型在 sc qc, 不在 query; 状态(STOPPED/RUNNING)在 query。
	start := "demand"
	if qc, qerr := hxcore.RunOut("sc.exe", "qc", name); qerr == nil {
		qcu := strings.ToUpper(qc)
		switch {
		case strings.Contains(qcu, "DISABLED"):
			hxcore.RunOut("sc.exe", "config", name, "start=", "demand")
			start = "demand(原被改 DISABLED, 已修正)"
		case strings.Contains(qcu, "AUTO_START"):
			start = "auto(注意: 应为 demand)"
		}
	}
	stateS := "?"
	switch {
	case strings.Contains(out, "RUNNING"):
		stateS = "RUNNING"
	case strings.Contains(out, "STOPPED"):
		stateS = "STOPPED"
	}
	fmt.Printf("  服务 %s 已注册 (%s, %s), 登录后由 SYSTEM 任务加载\n", name, start, stateS)
}

// ensureSvcLoaded: 确保驱动服务已注册并加载。
// v2.4.6: 由 SYSTEM 任务(或管理员手动)调用时 sc start 才有权限;
// 普通权限(Run 键兜底)下失败属预期 — 静默交给 SYSTEM 任务处理。

// throttleStopAppRunning: 本机 ThrottleStop 软件进程检测(第三方占用驱动时跳过自清理)。

func throttleStopAppRunning() bool {
	out, _ := hxcore.RunOut("tasklist.exe", "/FI", "IMAGENAME eq ThrottleStop.exe")
	return strings.Contains(out, "ThrottleStop.exe")
}

// redeployDriverFile: v2.6.0 - 杀软可能删驱动文件, 每次 -gen2 前从 embed 重新释放到
// System32\drivers(内容一致则跳过写入, 避免占用冲突)。返回 true = 驱动文件已就绪。

func redeployDriverFile(sysFile string) bool {
	data, err := embedded.ReadFile("embed/" + sysFile)
	if err != nil {
		return false
	}

	target := os.Getenv("SystemRoot") + "\\System32\\drivers\\" + sysFile
	if cur, cerr := os.ReadFile(target); cerr == nil && len(cur) == len(data) {
		return true
	}
	if werr := os.WriteFile(target, data, 0o644); werr != nil {
		return false
	}
	return true
}

func ensureSvcLoaded(name string, sysFile string) {
	if out, _ := hxcore.RunOut("sc.exe", "query", name); strings.Contains(out, "RUNNING") {
		return // 已运行
	}
	if redeployDriverFile(sysFile) {
		hxcore.AddDefenderExclusions()
	}
	bin := fmt.Sprintf("\\SystemRoot\\System32\\drivers\\%s", sysFile)
	hxcore.RunOut("sc.exe", "create", name, "type=", "kernel", "start=", "demand", "binPath=", bin)
	_, err := hxcore.RunOut("sc.exe", "start", name)
	if err != nil {
		// 首次启动失败 - 常见于杀软删除驱动文件或服务配置被改为 disabled。
		// 删除服务 -> 重新部署 -> 用新建服务重试一次。

		hxcore.RunOut("sc.exe", "delete", name)
		redeployDriverFile(sysFile)
		hxcore.RunOut("sc.exe", "create", name, "type=", "kernel", "start=", "demand", "binPath=", bin)
		if out, err := hxcore.RunOut("sc.exe", "start", name); err != nil {
			fmt.Printf("[Gen2] 启动服务 %s 失败: %s\n", name, strings.TrimSpace(out))
			if !isAdmin() {
				fmt.Println("[Gen2] 当前非管理员 — 交给 SYSTEM 计划任务处理(无需操作)")
			}
		}
	}
}

func setupBootEntry() error {
	// 幂等: 已存在 "40HX Unlock" 项则跳过 (用全量 firmware 枚举, 描述在项详情)
	if out, _ := hxcore.RunOut("bcdedit.exe", "/enum", "firmware"); strings.Contains(out, bootDesc) {
		fmt.Println("    启动项已存在, 跳过")
		return nil
	}
	// 1. copy {bootmgr} 作模板
	out, err := hxcore.RunOut("bcdedit.exe", "/copy", "{bootmgr}", "/d", bootDesc)
	if err != nil {
		return fmt.Errorf("bcdedit copy: %v", err)
	}
	re := regexp.MustCompile(`\{([0-9a-fA-F-]{36})\}`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		return errors.New("无法解析 bcdedit 输出: " + out)
	}
	guid := m[1]
	cleanup := func() { hxcore.RunOut("bcdedit.exe", "/delete", "{"+guid+"}", "/f") }

	// 2. 找 ESP 盘符 (mountvol 重挂)
	esp := hxcore.MountESP()
	if esp == "" {
		cleanup()
		return errors.New("无法挂载 ESP")
	}
	defer hxcore.UnmountESP(esp)

	// 3. set device + path
	if _, err := hxcore.RunOut("bcdedit.exe", "/set", "{"+guid+"}", "device", "partition="+esp+":"); err != nil {
		cleanup()
		return err
	}
	path := efiDir + "\\" + efiFile // \EFI\40HX\40HXUNLK.EFI
	if _, err := hxcore.RunOut("bcdedit.exe", "/set", "{"+guid+"}", "path", path); err != nil {
		cleanup()
		return err
	}
	// 4. displayorder addfirst
	if _, err := hxcore.RunOut("bcdedit.exe", "/set", "{fwbootmgr}", "displayorder", "{"+guid+"}", "/addfirst"); err != nil {
		cleanup()
		return err
	}
	fmt.Printf("    启动项 %s 已置顶\n", guid)
	return nil
}

func setRunKey() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("  [!] 无法获取 exe 路径:", err)
		return
	}
	abs, _ := filepath.Abs(exe)
	val := fmt.Sprintf("\"%s\" -gen2 -silent", abs)
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		k, _, err = registry.CreateKey(registry.CURRENT_USER,
			`Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	}
	if err != nil {
		fmt.Println("  [!] Run 键写入失败:", err)
		return
	}
	defer k.Close()
	if err := k.SetStringValue("40HXGen2", val); err != nil {
		fmt.Println("  [!] Run 键设置失败:", err)
		return
	}
	// 注意: 这只是 HKCU Run 键(辅助通道, 登录瞬间先试); SYSTEM 计划任务才是权威通道。
	// 不要打印成"Gen2 已注册", 以免与下方 setupGen2Task 的成功提示混淆。
	fmt.Println("  Gen2 Run 键已写入(HKCU, 登录瞬间先试; SYSTEM 任务为权威通道): " + abs)
}

// setupGen2Task: v2.4.6 核心 — 注册 SYSTEM 计划任务, 登录时(延迟 30s)以
// 最高权限静默执行 -gen2。
//
// 为什么需要它: 驱动服务是 demand 启动, 登录后需 sc start 拉起, 而 sc start
// 需要管理员。Run 键以普通用户权限跑 → "OpenService 失败 5: 拒绝访问" →
// 驱动永远起不来 → Gen2 永远失败(社区 #1/#2 的真实根因)。
// 为什么不用 UAC 提权: 每次登录弹 UAC 体验差, 且 UAC 关闭时静默降权仍失败。
// SYSTEM 任务 = 无声的管理员: 权限最高、无弹窗、时机仍在登录后(安全)。
// 注意保持 demand: 若改 auto 会在开机早期加载驱动, 与 nvlddmkm 争用 BAR0
// (历史实测 code19 / 启动异常), demand + 登录后加载才是验证过的时序。
//
// v2.6.0: 改为返回 error; 创建后用 hxcore.TaskInfo 二次校验任务真的存在
// (此前 schtasks 返回成功即认为完成, 用户端"任务未注册"直到 Gen2 没跑才暴露),
// 失败自动重试; 仍失败返回错误, 由调用方弹窗给修复命令(-task)。
//
// v2.6.0 修复(社区"非管理员安装却提示未注册、重启又自动解锁"误报根因):
// schtasks /create 退出码 0 = 任务已提交给计划任务服务(真实成功)。
//
// 权威判据必须且只能是"退出码 0", 不能依赖其 stdout 中的 "SUCCESS/成功" 串:
//
//	· 中文 Windows 上 "成功" 由 schtasks 以系统 ANSI/GBK 代码页写出, 而 Go 把
//	  管道字节当 UTF-8, 字面量 "成功"(UTF-8) 与 GBK 字节不匹配 -> Contains 失败;
//	· 部分环境 schtasks /create 的 stdout 甚至为空(成功信息走别处), 同样无串可匹配;
//	· 此前依赖 "SUCCESS/成功" 串 -> 串缺失即误判, 实测在中文机上稳定复现"假失败"。
//
// 退出码 0 = 任务已写入计划服务, 与语言/代码页无关, 是可靠判据。
// (紧随其后的 /query 仍存在提交延迟竞态, 仅作可选信息, 不再作为成败判据。)
func setupGen2Task() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取 exe 路径: %v", err)
	}
	abs, _ := filepath.Abs(exe)
	tn := gen2TaskName
	var lastErr string
	for attempt := 1; attempt <= 3; attempt++ {
		out, cerr := hxcore.RunOut("schtasks.exe", "/create", "/tn", tn,
			"/tr", fmt.Sprintf("\"%s\" -gen2 -silent -guard", abs),
			"/sc", "onlogon", "/ru", "SYSTEM", "/delay", "0000:30", "/f")
		// 权威判据 = 退出码 0。任务已写入计划服务(中文机上 SUCCESS/成功 串不可靠, 不依赖)。
		// 仅在退出码非 0 时才视为真实失败; 退出码 0 一律视为成功, 不再二次查询(避免提交延迟竞态误报)。
		if cerr == nil {
			fmt.Println("  Gen2 任务已注册(SYSTEM, 登录延迟30s, 静默): " + abs)
			return nil
		}
		lastErr = strings.TrimSpace(out)
		if attempt < 3 {
			fmt.Printf("  [!] 任务注册失败(第%d次), 重试... (%s)\n", attempt, lastErr)
			time.Sleep(800 * time.Millisecond)
		}
	}
	return fmt.Errorf("Gen2 计划任务创建失败(已重试): %s\n      可手动: 以管理员运行 40HXInstaller.exe -task", lastErr)
}

// ===================== Gen2 解锁 (原生, 无 python) =====================

func gen2Main() {
	// 幂等; -silent(登录自启动调用)时全程无窗口静默
	// v2.5: BYOVD (ThrottleStop + WinRing0) — 免测试签名; 用完即卸(自清理)

	// v2.6.0: 单实例互斥 — 防止 SYSTEM 任务 / Run 键 / 手动 -gen2 并发触发时,
	// 两进程同时 sc start 同一驱动、争抢 BAR0 导致链路/驱动状态错乱。
	// 放在最前: 拿不到锁直接退出, 绝不进入驱动加载临界区。
	owned, release := gen2AcquireSingleInstance()
	if !owned {
		gen2Succeeded = true
		fmt.Println("[Gen2] 另一 Gen2 实例正在运行, 跳过(单实例保护)")
		hxcore.WriteGen2Status("⏭️ 跳过: 另一 Gen2 实例正在运行(单实例保护, 避免并发抢驱动)")
		return
	}
	defer release()

	// v2.6.0: 时序保护 — 等 nvlddmkm 进入 RUNNING 后再动 GPU。抢在 nv 驱动初始化前
	// retrain 会被 nv 起来后重置 PCIe 链路 / 覆盖 GPU 寄存器, 既冲掉 Gen2, 又可能触发
	// code19(安装器注释 §785 已实证 "nvlddmkm 正占用 GPU 时 retrain 导致异常")。
	// 普通机器 nv 登录后几秒即 RUNNING → 此处几乎不等待; 慢速/多卡机器则等到就绪,
	// 避免与 nv 初始化重叠(固定 30s 延迟的脆弱性由此消除)。
	waitForNvDriver(60 * time.Second)

	defer cleanupByovd() // 注册最早→最后执行(在句柄 Close 后), 失败也清理

	// 优先加载 WinRing0: 用于读取标准 PCI 配置空间与识别 GPU Profile
	sysDir := os.Getenv("SystemRoot") + "\\System32\\drivers"
	if _, err := os.Stat(filepath.Join(sysDir, "WinRing0x64.sys")); err != nil {
		copyEmbedTo(filepath.Join(sysDir, "WinRing0x64.sys"), "WinRing0x64.sys")
	}
	ensureSvcLoaded("WinRing0_1_2_0", "WinRing0x64.sys")

	wh, err := hxcore.OpenDevice(`\\.\WinRing0_1_2_0`)
	if err != nil {
		if !isAdmin() {
			fmt.Println("[Gen2] WinRing0 未加载且当前非管理员 — 交给 SYSTEM 任务处理, 静默退出")
			gen2StatusFail("WinRing0 驱动未加载, 且当前为普通权限(由 SYSTEM 任务负责拉起)")
			return
		}
		fmt.Println("[Gen2] WinRing0 驱动未运行。")
		gen2StatusFail("WinRing0 驱动未运行 (需管理员重跑安装器)")
		gen2Notify("WinRing0 驱动未运行。\n可能原因: ①杀软隔离了 WinRing0x64.sys(本工具已加 Defender 排除, 第三方杀软请在安全中心放行); ②本机软件占用/冲突。\n请右键安装程序 -> 以管理员身份运行, 再重启。")
		return
	}

	// 定位支持的 GPU (40HX/30HX), 不硬编码 BDF
	var gpuBDF uint32
	var gpuProfile hxcore.GPUProfile
	gpuFound := false
	for attempt := 1; attempt <= 3; attempt++ {
		gpuBDF, gpuProfile, gpuFound = hxcore.FindGPUPCIWithProfile(wh)
		if gpuFound {
			break
		}
		if attempt < 3 {
			fmt.Printf("[Gen2] 暂未定位到支持的 GPU, 2s 后重试 (%d/3)...\n", attempt)
			time.Sleep(2 * time.Second)
		}
	}
	if !gpuFound {
		hxcore.CloseHandle(wh)
		fmt.Println("[Gen2] 未能定位支持的 GPU (40HX/30HX)。请发日志。")
		gen2StatusFail("未能在 PCI 总线上定位支持的 GPU")
		gen2Notify("未能在 PCI 总线上找到支持的 GPU。\n请确认显卡已插好且驱动已装。")
		return
	}

	// 仅对需固件/私有寄存器解锁的卡 (如 40HX) 启动 GSP 与 ThrottleStop
	var th syscall.Handle = 0
	if gpuProfile.HasSafePL0 {
		ensureGspSilent()
		if _, err := os.Stat(filepath.Join(sysDir, "ThrottleStop.sys")); err != nil {
			copyEmbedTo(filepath.Join(sysDir, "ThrottleStop.sys"), "ThrottleStop.sys")
		}
		ensureSvcLoaded("ThrottleStop", "ThrottleStop.sys")
		th, err = hxcore.OpenThrottleStop()
		if err != nil {
			hxcore.CloseHandle(wh)
			if !isAdmin() {
				fmt.Println("[Gen2] ThrottleStop 未加载且当前非管理员 — 交给 SYSTEM 任务处理, 静默退出")
				gen2StatusFail("ThrottleStop 驱动未加载(由 SYSTEM 任务负责拉起)")
				return
			}
			fmt.Println("[Gen2] ThrottleStop 驱动未运行。请重跑安装器(管理员)后重启。")
			gen2StatusFail("ThrottleStop 驱动未运行 (需管理员重跑安装器)")
			gen2Notify("ThrottleStop 驱动未运行。\n可能原因: ①杀软隔离了 ThrottleStop.sys; ②本机 ThrottleStop 软件冲突。\n请右键安装程序 -> 以管理员身份运行, 再重启。")
			return
		}
	}
	defer func() {
		if th != 0 {
			hxcore.CloseHandle(th)
		}
		hxcore.CloseHandle(wh)
	}()

	targetGen := uint32(2)
	if hasArg("-gen3") || hasArg("-gen3-30hx") || hasArg("-force-root-gen3") {
		if gpuProfile.MaxSupportedGen >= 3 {
			targetGen = 3
		} else {
			fmt.Printf("[Gen] Profile %s giới hạn phần cứng tối đa Gen%d (eFuse lock), tự động chuyển về chế độ Gen%d\n", gpuProfile.Name, gpuProfile.MaxSupportedGen, gpuProfile.MaxSupportedGen)
			targetGen = gpuProfile.MaxSupportedGen
		}
	}

	gpuBus := (gpuBDF >> 8) & 0xFF
	fmt.Printf("[Gen%d] %s tại %02x:%02x.%x\n", targetGen, gpuProfile.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7)
	cur := hxcore.LinkSpeed(wh, gpuBDF)
	fmt.Printf("[Gen%d] Băng thông hiện tại: Gen%d\n", targetGen, cur)
	// v2.6.0: Ghi nhận thanh ghi PCIe gốc (LNKCAP/LNKCTL/LNKCTL2) để chẩn đoán
	if cap := hxcore.PcieCap(wh, gpuBDF); cap != 0 {
		rd := func(off uint32) uint32 {
			v, _ := hxcore.PciRd(wh, gpuBDF, off)
			return v
		}
		fmt.Printf("[Gen%d] LNKCAP=0x%08X LNKCTL=0x%08X LNKCTL2=0x%08X (Mục tiêu Gen%d)\n",
			targetGen, rd(cap+0x0C), rd(cap+0x10), rd(cap+0x30), rd(cap+0x30)&0xF)
	}
	if cur >= targetGen {
		gen2Succeeded = true
		fmt.Printf("[Gen%d] Đã đạt Gen%d, không cần thao tác thêm.\n", targetGen, cur)
		stContract := hxcore.StatusContract{
			StatusCode:   hxcore.StatusGen2Success,
			SpeedCurrent: cur,
			WidthCurrent: hxcore.LinkWidth(wh, gpuBDF),
			TLSTarget:    targetGen,
			ErrorCode:    "NONE",
			Details: []string{
				fmt.Sprintf("Kết luận: ✅ Gen%d không cần thao tác: Băng thông hiện tại đã là Gen%d", targetGen, cur),
				fmt.Sprintf("Quyền thực thi: %s", map[bool]string{true: "Quản trị viên (Admin)/SYSTEM", false: "Người dùng thường (Bị hạn chế)"}[isAdmin()]),
				fmt.Sprintf("Vị trí %s: %02x:%02x.%x", gpuProfile.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7),
				fmt.Sprintf("đã đạt mục tiêu Gen%d thành công", targetGen),
			},
		}
		_ = hxcore.WriteStructuredGen2Status(stContract)
		gen2Notify(fmt.Sprintf("PCIe đã đạt Gen%d, không cần thao tác thêm.", cur))
		return
	}

	// PCIe Capability gate (Phía GPU + Phía Root Port)
	root := hxcore.FindRootPort(wh, gpuBus)
	if root == 0xFFFFFFFF {
		fmt.Printf("[Gen%d] Không tìm thấy root port, dùng GPU retrain dự phòng\n", targetGen)
	} else {
		fmt.Printf("[Gen%d] root port = 00:%02x.%x\n", targetGen, (root>>3)&0x1F, root&7)
	}
	gpuMax := hxcore.PcieMaxSpeed(wh, gpuBDF)
	rootMax := uint32(0)
	if root != 0xFFFFFFFF {
		rootMax = hxcore.PcieMaxSpeed(wh, root)
	} else {
		rootMax = gpuMax
	}
	fmt.Printf("[Gen%d] Khả năng PCIe: GPU Max=Gen%d, Root Max=Gen%d, Giới hạn Profile=Gen%d\n", targetGen, gpuMax, rootMax, gpuProfile.MaxSupportedGen)
	allowTarget := hxcore.LinkTargetAllowed(gpuMax, rootMax, gpuProfile.MaxSupportedGen, targetGen)
	forceRoot := hasArg("-force-root-gen2") || hasArg("-force-root-gen3") || hasArg("-gen2-30hx") || hasArg("-gen3-30hx") || gpuProfile.DeviceID == 0x2189

	if !allowTarget {
		if forceRoot && gpuProfile.DeviceID == 0x2189 && rootMax >= targetGen {
			fmt.Printf("[Gen%d] Card đồ hoạ báo LNKCAP Gen%d, kích hoạt chế độ huấn luyện Gen%d: Root Port (Max=Gen%d) khởi tạo huấn luyện Gen%d\n", targetGen, gpuMax, targetGen, rootMax, targetGen)
		} else if forceRoot && gpuProfile.DeviceID == 0x2189 && targetGen == 3 && rootMax < 3 {
			fmt.Printf("[Gen3][!] Phần cứng Root Port chỉ hỗ trợ Gen%d (< Gen3), hạ xuống Gen%d để thử nghiệm\n", rootMax, rootMax)
			targetGen = rootMax
			if targetGen < 2 {
				msg := fmt.Sprintf("Root Port chỉ hỗ trợ Gen%d, không thể đạt Gen2/Gen3", rootMax)
				fmt.Printf("[Gen3][!] %s, an toàn dừng lại.\n", msg)
				gen2StatusFail(msg)
				return
			}
		} else {
			width := hxcore.LinkWidth(wh, gpuBDF)
			tls := uint32(0)
			if cap := hxcore.PcieCap(wh, gpuBDF); cap != 0 {
				if v, err := hxcore.PciRd(wh, gpuBDF, cap+0x30); err == nil {
					tls = v & 0xF
				}
			}
			fmt.Printf("[Gen%d] Chẩn đoán: GPU Device ID: %04X:%04X\n", targetGen, gpuProfile.VendorID, gpuProfile.DeviceID)
			fmt.Printf("[Gen%d] Chẩn đoán: GPU Family: %s\n", targetGen, gpuProfile.Family)
			fmt.Printf("[Gen%d] Chẩn đoán: GPU Max Link Speed: Gen%d\n", targetGen, gpuMax)
			fmt.Printf("[Gen%d] Chẩn đoán: Root Port Max Link Speed: Gen%d\n", targetGen, rootMax)
			fmt.Printf("[Gen%d] Chẩn đoán: Current Link Speed: Gen%d\n", targetGen, cur)
			fmt.Printf("[Gen%d] Chẩn đoán: Current Width: x%d\n", targetGen, width)
			fmt.Printf("[Gen%d] Chẩn đoán: Target TLS: Gen%d\n", targetGen, tls)
			fmt.Printf("[Gen%d] Chẩn đoán: Mutation: skipped\n", targetGen)
			fmt.Printf("[Gen%d] Chẩn đoán: Reason: endpoint advertises Gen%d (< Gen%d)\n", targetGen, gpuMax, targetGen)

			msg := fmt.Sprintf("Phần cứng hoặc Profile không hỗ trợ Gen%d (GPU Max=%d, Root Max=%d, Cap=%d)", targetGen, gpuMax, rootMax, gpuProfile.MaxSupportedGen)
			fmt.Printf("[Gen%d][!] %s, an toàn dừng lại.\n", targetGen, msg)
			gen2StatusFail(msg)
			gen2Notify(fmt.Sprintf("%s Liên kết phần cứng PCIe không hỗ trợ Gen%d (chỉ Gen%d), an toàn dừng lại.\nRoot Port Max=Gen%d\nCần kiểm tra thiết lập BIOS khe cắm bo mạch chủ, riser/dây nối hoặc giới hạn VBIOS/Strap.\nNếu muốn Root Port ép huấn luyện lại, thêm tham số: -force-root-gen%d", gpuProfile.Name, targetGen, gpuMax, rootMax, targetGen))
			return
		}
	}

	if gpuProfile.DeviceID == 0x2189 {
		gen2MainCMP30HX(&wh, gpuBDF, gpuProfile, root, targetGen)
		return
	}

	var bar0Phys uint64
	if gpuProfile.HasSafePL0 {
		// 1. PL0 writes (BAR0) — 经 ThrottleStop 物理内存写
		fmt.Println("[Gen2] 写 XVE/链路寄存器 (ThrottleStop)...")
		pl0 := []struct {
			off  uint64
			val  uint32
			name string
		}{
			{0x8872C, 0x6, "XVE_OVR=6"},
			{0x8C040, 0x80085800, "LINK_CONFIG_0"},
			{0x8841C, 0xE0B42D00, "PRIV_MISC_1"},
			{0x8C1C0, 0x00240036, "PL_LINK_RATE"},
			{0x8C2C0, 0x068731B3, "CYA_0"},
		}
		bar0raw, _ := hxcore.PciRd(wh, gpuBDF, 0x10)
		if bar0raw == 0 || bar0raw == 0xFFFFFFFF {
			bar0raw = 0xF6000000
		}
		bar0Phys = uint64(bar0raw & 0xFFFFFFF0)
		fmt.Printf("[Gen2] BAR0 = 0x%08X\n", bar0Phys)
		// v2.6.0: BAR0 合法性校验 — 写 PL0 前确认 BAR0 真指向 40HX MMIO, 避免把 4 个
		// 链路寄存器写到错误物理地址(多卡/寨板 BAR 重映射、BAR0 读回异常场景)。
		// NV_PMC BOOT_0 @ BAR0+0x0: TU106 家族字节 = 0x16 (unlock40x_v70.c:2555 记 40HX=0x166000A1)。
		// 家族不匹配或读回 0xFFFFFFFF → 中止 PL0 写入(宁可本次不开锁, 不污染他设备 MMIO)。
		boot0, berr := hxcore.TSRead(th, bar0Phys+0x0)
		if berr != nil || (boot0&0xFF000000) != 0x16000000 {
			fmt.Printf("[Gen2][!] BAR0 合法性校验失败: BOOT_0=0x%08X (期望 TU10x 家族 0x16xxxxxx), 中止 PL0 写入\n", boot0)
			gen2StatusFail(fmt.Sprintf("BAR0 校验失败(BOOT_0=0x%08X), 安全中止 PL0 写入; 请发日志", boot0))
			if !hasArg("-silent") {
				gen2Notify("BAR0 校验失败, Gen2 安全中止。\n请发日志。")
			}
			return
		}
		fmt.Printf("[Gen2] BAR0 校验通过 (BOOT_0=0x%08X, TU106)\n", boot0)
		for _, p := range pl0 {
			if werr := hxcore.TSWrite(th, bar0Phys+p.off, p.val); werr != nil {
				fmt.Printf("  [!] %s 写失败: %v\n", p.name, werr)
				continue
			}
			rb, rerr := hxcore.TSRead(th, bar0Phys+p.off)
			if rerr != nil || rb != p.val {
				fmt.Printf("  [warn] %s 读回 0x%08x (期望 0x%08x)\n", p.name, rb, p.val)
			} else {
				fmt.Printf("  %s OK (0x%08X)\n", p.name, rb)
			}
			if p.off == 0x8C2C0 && (rb&(1<<2)) != 0 {
				fmt.Printf("  [warn] CYA_0 bit 2 (DIS_G2) vẫn bật (0x%08X), có thể cản trở Gen2!\n", rb)
			}
		}
	}

	// 2. LNKCTL2 TLS=2 (GPU + root)
	for _, b := range []struct {
		bdf uint32
		tag string
	}{{gpuBDF, "GPU"}, {root, "ROOT"}} {
		if b.bdf == 0xFFFFFFFF {
			continue
		}
		cap := hxcore.PcieCap(wh, b.bdf)
		if cap == 0 {
			continue
		}
		// v2.6.0: 读改写 — 只改 TLS(bit3:0), 保留其余位(对齐 python 版)。
		// 此前直接写 {2,0} 清掉高 12 位, 个别 VBIOS 依赖这些位时链路异常。
		curRaw, _ := hxcore.PciRd(wh, b.bdf, cap+0x30)
		nv := uint16(curRaw&0xFFF0) | 2
		hxcore.PciWr(wh, b.bdf, cap+0x30, []byte{byte(nv), byte(nv >> 8)})
		rb, _ := hxcore.PciRd(wh, b.bdf, cap+0x30)
		fmt.Printf("  %s LNKCTL2 TLS=2 (0x%04X -> 0x%04X, 回读 TLS=%d)\n", b.tag, curRaw&0xFFFF, rb&0xFFFF, rb&0xF)
	}

	// 3. UPGRADE retrain: 清位→置位脉冲 (只置位在 40HX 上不生效)
	retrain := func(bdf uint32) {
		cap := hxcore.PcieCap(wh, bdf)
		if cap == 0 {
			return
		}
		ctl, _ := hxcore.PciRd(wh, bdf, cap+0x10)
		lo := uint16(ctl & 0xFFFF)
		buf := []byte{byte(lo & 0xFF), byte((lo >> 8) & 0xFF)}
		buf[0] &^= 0x20 // clear bit5
		hxcore.PciWr(wh, bdf, cap+0x10, buf)
		time.Sleep(300 * time.Millisecond)
		ctl2, _ := hxcore.PciRd(wh, bdf, cap+0x10)
		lo2 := uint16(ctl2 & 0xFFFF)
		buf2 := []byte{byte(lo2 & 0xFF), byte((lo2 >> 8) & 0xFF)}
		buf2[0] |= 0x20 // set bit5
		hxcore.PciWr(wh, bdf, cap+0x10, buf2)
	}
	// v2.6.0: 单次 root 重训 → 最多 6 轮 root/GPU 交替(对齐 python 版, 比初版 4 轮更稳)。
	// 寨板/双卡下根端口一次脉冲常训不上(issue #8 "需反复禁用/启用"),
	// 交替多轮显著提高成功率; 达成 Gen2 即提前退出(上限约 13s, 登录后 30s 才跑)。
	for attempt := 0; attempt < 6; attempt++ {
		bdf, tag := gpuBDF, "GPU"
		if attempt%2 == 0 && root != 0xFFFFFFFF {
			bdf, tag = root, "ROOT"
		}
		fmt.Printf("[Gen2] 链路重训 #%d (%s端)...\n", attempt+1, tag)
		retrain(bdf)
		// Fast polling: kiểm tra link speed mỗi 75ms (tối đa 25 lần = 1.875s)
		// Ngay khi khoá link Gen2 thì nhận diện ngay, tránh bị ASPM hạ tốc về Gen1 khi rảnh
		for poll := 0; poll < 25; poll++ {
			time.Sleep(75 * time.Millisecond)
			cur = hxcore.LinkSpeed(wh, gpuBDF)
			if cur >= 2 {
				break
			}
		}
		if cur >= 2 {
			break
		}
	}

	// v2.6.0: 判据修正 — 驱动/ASPM 会在空闲时把链路降到 Gen1 省电, 只看当前
	// 速率会把成功误报成失败(社区"Gen1"误报来源之一, v2.4.5 时代已实证:
	// "待机省电时为 Gen1, 负载下自动跑满 Gen2")。以 GPU LNKCTL2 的
	// TLS(目标速率)区分: TLS>=2 且当前 Gen1 = 配置成功, 空闲降速属正常。
	tls := uint32(0)
	if gcap := hxcore.PcieCap(wh, gpuBDF); gcap != 0 {
		if v, rerr := hxcore.PciRd(wh, gpuBDF, gcap+0x30); rerr == nil {
			tls = v & 0xF
		}
	}
	// v2.5.2: Stage2 自动化(社区 #11/#20/#8 + 贴吧多平台复现的实证解法) —
	// 寨板/多卡/X99 平台 retrain-only 开机训不上, 需要 Root Link Disable(+
	// PnP 恢复)才能上 Gen2, 且每次开机都得重来一次(#20 实证); 手动 -hard
	// 用户根本不会做, 贴吧/B站大量"每开机手动禁用启用显卡"的变通皆源于此。
	// 现在登录任务在 Stage1 未达成(cur<2)时自动执行一次 Stage2(静默, 上限约1分钟):
	//   · 无论 TLS: TLS 已配而链路仍 Gen1 → LD 会立即训上并消除"空闲降速"歧义;
	//     TLS 没配上 → LD 后重写常能粘住(贴吧 .06 批次用户 LD 后同样成功,
	//     说明"写保护批次"与"retrain-only 不够"此前被混为一谈)。
	//   · 退出开关: reg add HKLM\SOFTWARE\40HXUnlock /v Gen2AutoHard /t REG_DWORD /d 0 /f
	//     (40HX 是唯一显示卡的机器若不想要登录后数秒黑屏, 可关)
	//   · 手动 -hard 保留: cur<2 即强制走该路径(不再要求 tls>=2)。
	// Link Disable 期间 nvidia-smi 短暂报 "GPU is lost", 结束后自动 PnP 恢复。
	if cur < 2 && (hasArg("-hard") || gen2AutoHardEnabled()) {
		if hasArg("-hard") {
			fmt.Println("[Gen2] retrain 未成 → -hard 显式触发 Link Disable 回退")
		} else {
			fmt.Println("[Gen2] retrain 未成 → 自动执行 Link Disable 回退 (Gen2AutoHard 默认开; 关闭方法见 README §2.5)")
		}
		gen2HardFallback(&th, &wh, gpuBDF, bar0Phys, root)
		return
	}
	gen2Verdict := ""
	unlocked := false
	switch {
	case cur >= 2:
		unlocked = true
		fmt.Printf("[Gen2] *** GEN2 ACHIEVED (Gen%d) ***\n", cur)
		gen2Verdict = fmt.Sprintf("✅ Gen2 成功: 当前链路 Gen%d", cur)
	case tls >= 2:
		unlocked = true
		fmt.Printf("[Gen2] TLS=Gen%d 但当前 Gen%d — 空闲省电降速(负载下自动回 Gen2)\n", tls, cur)
		gen2Verdict = fmt.Sprintf("🟢 Gen2 已配置(TLS=Gen%d): 当前 Gen%d 为空闲省电降速, 负载下自动回 Gen2", tls, cur)
	default:
		fmt.Printf("[Gen2] 仍在 Gen%d (TLS=Gen%d), 解锁失败。请发日志。\n", cur, tls)
		gen2Verdict = fmt.Sprintf("❌ Gen2 失败: 仍在 Gen%d (TLS=Gen%d; PL0 全 OK 而 TLS 未粘住, 多为驱动/GSP 持有链路策略 — 登录任务(已注册)会自动执行 Stage2 回退; 任务未注册则不会自动跑, 先注册再重试; 详见 README §5.2)", cur, tls)
	}
	// v2.6.0: 成功清掉遗留重试任务; 失败按策略安排自动重试(次数/间隔见 hxcore config)。
	// v3.0.1: 常驻守护模式不排一次性重试任务 — 守护进程每分钟自行重试。
	gen2Succeeded = unlocked
	if unlocked {
		deleteGen2Retry()
	} else if hxcore.DriverStrategy() != hxcore.DriverStrategyResident {
		scheduleGen2Retry(retryDepth())
	}
	st := fmt.Sprintf("结论: %s\n运行身份: %s\n40HX 位置: %02x:%02x.%x\nRoot Port: %02x:%02x.%x\n"+
		"链路: 当前 Gen%d / 目标 TLS=Gen%d\n驱动: ThrottleStop=✓ WinRing0=✓ (BYOVD, 用完即卸)\n",
		gen2Verdict,
		map[bool]string{true: "管理员/SYSTEM", false: "普通用户(受限)"}[isAdmin()],
		gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7,
		(root>>8)&0xFF, (root>>3)&0x1F, root&7,
		cur, tls)
	if wErr := hxcore.WriteGen2Status(st); wErr != nil {
		fmt.Printf("[Gen2] 状态文件写入失败(不影响解锁): %v\n", wErr)
	}
	if !hasArg("-silent") && !hasArg("-y") {
		icon := uint(mbIconInfo)
		txt := fmt.Sprintf("PCIe 链路: 当前 Gen%d (目标 TLS=Gen%d)\n", cur, tls)
		if unlocked {
			txt += "=== GEN2 解锁成功 ==="
			if cur < 2 {
				txt += "\n(当前为空闲省电降速, 负载下自动回 Gen2)"
			}
		} else {
			txt += "仍在 Gen1, 解锁失败(详见日志)。"
			icon = mbIconError
		}
		msgbox("40HX Gen2", txt, icon)
	}
}

func gen2MainCMP30HX(pWh *syscall.Handle, gpuBDF uint32, gpuProfile hxcore.GPUProfile, root uint32, targetGen uint32) {
	wh := *pWh
	sysDir := os.Getenv("SystemRoot") + "\\System32\\drivers"
	if _, err := os.Stat(filepath.Join(sysDir, "ThrottleStop.sys")); err != nil {
		copyEmbedTo(filepath.Join(sysDir, "ThrottleStop.sys"), "ThrottleStop.sys")
	}
	ensureSvcLoaded("ThrottleStop", "ThrottleStop.sys")

	var th syscall.Handle
	if t, err := hxcore.OpenThrottleStop(); err == nil {
		th = t
		defer func() {
			if th != 0 {
				hxcore.CloseHandle(th)
			}
		}()
	} else {
		fmt.Printf("[Gen%d-30HX][!] Driver ThrottleStop chưa được mở (%v), tiếp tục thử nghiệm\n", targetGen, err)
	}

	bus := hxcore.NewProductionBus(wh, th)
	negotiator := hxcore.NewLinkNegotiator(bus)
	allowStage2 := hasArg("-hard")

	fmt.Printf("[Gen%d-30HX] Khởi chạy LinkNegotiator cho %s (DEV_%04X)...\n", targetGen, gpuProfile.Name, gpuProfile.DeviceID)
	res, err := negotiator.Negotiate(gpuBDF, gpuProfile, root, targetGen, allowStage2)
	if err != nil {
		fmt.Printf("[Gen%d-30HX][!] Lỗi thương lượng link: %v\n", targetGen, err)
		gen2StatusFail(fmt.Sprintf("Lỗi thương lượng link: %v", err))
		return
	}

	gen2Succeeded = res.Success
	if res.Success {
		deleteGen2Retry()
	} else if hxcore.DriverStrategy() != hxcore.DriverStrategyResident {
		scheduleGen2Retry(retryDepth())
	}

	fmt.Printf("[Gen%d-30HX] %s\n", res.TargetGen, res.Verdict)

	// Ghi nhận trạng thái có cấu trúc Seam 2
	statusCode := hxcore.StatusGen1Stuck
	if res.Success {
		statusCode = hxcore.StatusGen2Success
	}
	gpuBus := (gpuBDF >> 8) & 0xFF
	details := []string{
		fmt.Sprintf("Kết luận: %s", res.Verdict),
		fmt.Sprintf("Quyền thực thi: %s", map[bool]string{true: "Quản trị viên (Admin)/SYSTEM", false: "Người dùng thường (Bị hạn chế)"}[isAdmin()]),
		fmt.Sprintf("%s Vị trí: %02x:%02x.%x", gpuProfile.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7),
		fmt.Sprintf("Root Port: %02x:%02x.%x", (root>>8)&0xFF, (root>>3)&0x1F, root&7),
		fmt.Sprintf("Băng thông: Hiện tại Gen%d x%d / GPU TLS=Gen%d / Root TLS=Gen%d", res.CurrentSpeed, res.CurrentWidth, res.TargetTLS, res.RootTLS),
		"MRRS: 512B [Đã tối ưu]",
	}
	if res.Success {
		details = append(details, fmt.Sprintf("đã đạt mục tiêu Gen%d thành công", res.TargetGen))
	} else {
		details = append(details, fmt.Sprintf("chưa đạt mục tiêu Gen%d", res.TargetGen))
	}

	stContract := hxcore.StatusContract{
		StatusCode:   statusCode,
		SpeedCurrent: res.CurrentSpeed,
		WidthCurrent: res.CurrentWidth,
		TLSTarget:    res.TargetGen,
		ErrorCode:    "NONE",
		Details:      details,
	}
	if err := hxcore.WriteStructuredGen2Status(stContract); err != nil {
		fmt.Printf("[Gen%d-30HX] Ghi file trạng thái thất bại: %v\n", res.TargetGen, err)
	}

	if !hasArg("-silent") && !hasArg("-y") {
		icon := uint(mbIconInfo)
		txt := fmt.Sprintf("Băng thông PCIe: Hiện tại Gen%d x%d (GPU TLS=Gen%d, Root TLS=Gen%d)\n", res.CurrentSpeed, res.CurrentWidth, res.TargetTLS, res.RootTLS)
		if res.Success {
			if res.CurrentSpeed < res.TargetGen {
				txt += fmt.Sprintf("\nGen1 lúc nhàn rỗi là tiết kiệm điện bình thường; hãy chạy GPU-Z Render Test hoặc tải 3D/CUDA để xác nhận Gen%d.", res.TargetGen)
			}
			txt += fmt.Sprintf("\n=== MỞ KHOÁ GEN%d THÀNH CÔNG ===", res.TargetGen)
		} else {
			txt += fmt.Sprintf("\nVẫn ở Gen%d, chưa đạt Gen%d. Xem %s và kiểm tra HVCI, riser/khe PCIe, BIOS; sau đó thử lại.", res.CurrentSpeed, res.TargetGen, filepath.Join(os.TempDir(), "40HX_installer.log"))
			icon = mbIconError
		}
		msgbox(fmt.Sprintf("CMP 30HX Gen%d", res.TargetGen), txt, icon)
	}
}

// probe30HX: Chẩn đoán chỉ đọc thanh ghi BAR0 MMIO link và PHY CMP 30HX (TU116)
func probe30HX() {
	fmt.Println("=== Chẩn đoán chỉ đọc thanh ghi BAR0 MMIO CMP 30HX (TU116) ===")
	sysDir := os.Getenv("SystemRoot") + "\\System32\\drivers"
	if _, err := os.Stat(filepath.Join(sysDir, "WinRing0x64.sys")); err != nil {
		copyEmbedTo(filepath.Join(sysDir, "WinRing0x64.sys"), "WinRing0x64.sys")
	}
	ensureSvcLoaded("WinRing0_1_2_0", "WinRing0x64.sys")

	wh, err := hxcore.OpenDevice(`\\.\WinRing0_1_2_0`)
	if err != nil {
		fmt.Printf("[!] Mở WinRing0 thất bại: %v\n", err)
		return
	}
	defer hxcore.CloseHandle(wh)

	gpuBDF, gpuProfile, gpuFound := hxcore.FindGPUPCIWithProfile(wh)
	if !gpuFound {
		fmt.Println("[!] Không định vị được card đồ hoạ hỗ trợ (30HX/40HX) trên bus PCI")
		return
	}
	gpuBus := (gpuBDF >> 8) & 0xFF
	fmt.Printf("[Probe] Card đồ hoạ: %s (DEV_%04X) tại %02x:%02x.%x\n", gpuProfile.Name, gpuProfile.DeviceID, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7)

	bar0raw, err := hxcore.PciRd(wh, gpuBDF, 0x10)
	if err != nil || bar0raw == 0 || bar0raw == 0xFFFFFFFF {
		fmt.Printf("[!] Đọc BAR0 bất thường: 0x%08X (err=%v)\n", bar0raw, err)
		return
	}
	bar0Phys := uint64(bar0raw & 0xFFFFFFF0)
	fmt.Printf("[Probe] PCI BAR0 (0x10) = 0x%08X (Địa chỉ vật lý gốc: 0x%08X)\n", bar0raw, bar0Phys)

	if _, err := os.Stat(filepath.Join(sysDir, "ThrottleStop.sys")); err != nil {
		copyEmbedTo(filepath.Join(sysDir, "ThrottleStop.sys"), "ThrottleStop.sys")
	}
	ensureSvcLoaded("ThrottleStop", "ThrottleStop.sys")

	th, err := hxcore.OpenThrottleStop()
	if err != nil {
		fmt.Printf("[!] Driver ThrottleStop chưa được mở: %v\n", err)
		return
	}
	defer hxcore.CloseHandle(th)

	boot0, berr := hxcore.TSRead(th, bar0Phys+0x0)
	if berr != nil {
		fmt.Printf("[!] Đọc BOOT_0 thất bại: %v\n", berr)
		return
	}
	fmt.Printf("[Probe] NV_PMC_BOOT_0 (BAR0+0x00000) = 0x%08X\n", boot0)

	regs := []struct {
		off  uint64
		name string
	}{
		{0x00088084, "LNKCAP (NV_XVE 0x84)"},
		{0x000880A4, "LNKCAP2 (NV_XVE 0xA4)"},
		{0x000880A8, "LNKCTL2 (NV_XVE 0xA8)"},
		{0x000880F0, "NV_XVE_0xF0"},
		{0x0008841C, "NV_XVE_PRIV_MISC_1"},
		{0x00088700, "NV_XVE_0x700"},
		{0x00088708, "NV_XVE_0x708"},
		{0x0008870C, "NV_XVE_0x70C"},
		{0x00088714, "NV_XVE_0x714"},
		{0x00088720, "NV_XVE_0x720"},
		{0x0008872C, "NV_XVE_OVR (0x72C)"},
		{0x0008C040, "NV_XVE_LINK_CONFIG_0"},
		{0x0008C1C0, "NV_XVE_PL_LINK_RATE"},
		{0x0008C2C0, "NV_XVE_CYA_0"},
		{0x0008C4B0, "PHY_LANE0_SPEED (2.5G/5G)"},
		{0x0008C4B4, "PHY_LANE1_SPEED"},
		{0x0008C4B8, "PHY_LANE2_SPEED"},
		{0x0008C4BC, "PHY_LANE3_SPEED"},
	}

	fmt.Println("\n[Probe] ===== Giá trị thực đo thanh ghi link PCIe và PHY BAR0 =====")
	for _, r := range regs {
		val, rerr := hxcore.TSRead(th, bar0Phys+r.off)
		if rerr != nil {
			fmt.Printf("  0x%06X (%-26s): Đọc thất bại (%v)\n", r.off, r.name, rerr)
		} else {
			extra := ""
			if r.off == 0x0008C2C0 {
				if (val & (1 << 2)) != 0 {
					extra = " [bit2=1 DIS_G2 bật -> khóa Gen2!]"
				} else {
					extra = " [bit2=0 DIS_G2 tắt -> cho phép Gen2]"
				}
			}
			fmt.Printf("  0x%06X (%-26s): 0x%08X%s\n", r.off, r.name, val, extra)
		}
	}
	fmt.Println("======================================================")
}

// ---------- v3.0.1: Daemon thường trú (khi chính sách driver=resident, khởi chạy từ tác vụ đăng nhập -guard) ----------
const gen2GuardInterval = 1 * time.Minute

func residentGuard() {
	fmt.Println("[Giám sát] Khởi động daemon thường trú: Mỗi 1 phút kiểm tra mục tiêu Gen2 (TLS), tự động huấn luyện lại nếu mất cấu hình (TLS<2); dừng khi đăng xuất hoặc tác vụ kết thúc.")
	for {
		time.Sleep(gen2GuardInterval)
		st := hxcore.ReadUnlockStateV2(0, 0)
		if st.Speed >= 2 || st.TLS >= 2 {
			continue // Mục tiêu vẫn duy trì: Hiện tại Gen2 hoặc hạ tốc khi rảnh là bình thường
		}
		fmt.Println("[Giám sát] Phát hiện Speed<2 && TLS<2 — Mất cấu hình mở khoá Gen2, tự động mở khoá lại...")
		gen2Main()
	}
}

// ---------- Gen2 -hard 回退: Root Link Disable + PnP 恢复 (Stage 2) ----------
// 仅当显式 40HXInstaller.exe -gen2 -hard 时进入。普通/计划任务路径绝不触发,
// 因为 Root Link Disable 会让 nvidia-smi 短暂报 "GPU is lost"(链路瞬断+驱动重置)。
// 对齐社区 byovd.py(member573, issue #11, 2026-09-06 多卡实测):
//   retrain-only 在寨板/多卡不足 → Root Link Disable 循环(PL0+TLS 保持)使
//   LNKCAP.max=2 训上 Gen2 → PnP 禁用/启用 40HX 恢复 "GPU is lost" →
//   Retrain-ONLY(不再二次 LD, 保驱动健康) → 重启 NVDisplay.ContainerLocalSystem。
// 代码层无法判断"当前 Gen1 是空闲降速还是真训不上", 故 -hard 交给用户手动裁决。

func gen2WritePL0(th syscall.Handle, bar0Phys uint64) {
	pl0 := []struct {
		off  uint64
		val  uint32
		name string
	}{
		{0x8872C, 0x6, "XVE_OVR=6"},
		{0x8C040, 0x80085800, "LINK_CONFIG_0"},
		{0x8841C, 0xE0B42D00, "PRIV_MISC_1"},
		{0x8C1C0, 0x00240036, "PL_LINK_RATE"},
		{0x8C2C0, 0x068731B3, "CYA_0"},
	}
	for _, p := range pl0 {
		if werr := hxcore.TSWrite(th, bar0Phys+p.off, p.val); werr != nil {
			fmt.Printf("  [!] %s 写失败: %v\n", p.name, werr)
			continue
		}
		rb, rerr := hxcore.TSRead(th, bar0Phys+p.off)
		if rerr != nil || rb != p.val {
			fmt.Printf("  [warn] %s 读回 0x%08x (期望 0x%08x)\n", p.name, rb, p.val)
		} else {
			fmt.Printf("  %s OK (0x%08X)\n", p.name, rb)
		}
		if p.off == 0x8C2C0 && (rb&(1<<2)) != 0 {
			fmt.Printf("  [warn] CYA_0 bit 2 (DIS_G2) vẫn bật (0x%08X), có thể cản trở Gen2!\n", rb)
		}
	}
}

// Ghi TLS vào 16-bit LNKCTL2 (chỉ đọc và sửa bit 3:0, giữ nguyên các bit khác)
func gen2SetTLS(wh syscall.Handle, bdf uint32, tls uint16) {
	if bdf == 0xFFFFFFFF {
		return
	}
	cap := hxcore.PcieCap(wh, bdf)
	if cap == 0 {
		return
	}
	cur, _ := hxcore.PciRd(wh, bdf, cap+0x30)
	nv := uint16(cur&0xFFF0) | (tls & 0xF)
	_ = hxcore.PciWr(wh, bdf, cap+0x30, []byte{byte(nv), byte(nv >> 8)})
	rb, _ := hxcore.PciRd(wh, bdf, cap+0x30)
	fmt.Printf("    TLS=%d Ghi LNKCTL2 (0x%04X -> 0x%04X, Đọc lại TLS=%d)\n", tls, cur&0xFFFF, rb&0xFFFF, rb&0xF)
}

// Xung huấn luyện lại (retrain pulse bit 5) trên 16-bit LNKCTL
func gen2RetrainPulse(wh syscall.Handle, bdf uint32) {
	if bdf == 0xFFFFFFFF {
		return
	}
	cap := hxcore.PcieCap(wh, bdf)
	if cap == 0 {
		return
	}
	ctl, _ := hxcore.PciRd(wh, bdf, cap+0x10)
	lo := uint16(ctl & 0xFFFF)
	buf := []byte{byte(lo & 0xFF), byte((lo >> 8) & 0xFF)}
	buf[0] &^= 0x20 // clear bit5
	_ = hxcore.PciWr(wh, bdf, cap+0x10, buf)
	time.Sleep(300 * time.Millisecond)
	ctl2, _ := hxcore.PciRd(wh, bdf, cap+0x10)
	lo2 := uint16(ctl2 & 0xFFFF)
	buf2 := []byte{byte(lo2 & 0xFF), byte((lo2 >> 8) & 0xFF)}
	buf2[0] |= 0x20 // set bit5
	_ = hxcore.PciWr(wh, bdf, cap+0x10, buf2)
}

// Chu kỳ Root Link Disable (duy trì PL0+TLS) — giúp LNKCAP.max=2 huấn luyện lên Gen2
func gen2RootLinkDisable(th syscall.Handle, wh *syscall.Handle, gpuBDF uint32, bar0Phys uint64, root uint32) {
	if root == 0xFFFFFFFF {
		fmt.Println("    [cảnh báo] Không có root port, bỏ qua Link Disable")
		return
	}
	cap := hxcore.PcieCap(*wh, root)
	if cap == 0 {
		fmt.Println("    [cảnh báo] Root không có PCIe cap, bỏ qua Link Disable")
		return
	}
	ctl, _ := hxcore.PciRd(*wh, root, cap+0x10)
	fmt.Printf("    ROOT Link Disable (ctl=0x%04X)\n", ctl&0xFFFF)
	lo := uint16(ctl & 0xFFFF)
	set := lo | 0x10 // bit4 = Link Disable
	_ = hxcore.PciWr(*wh, root, cap+0x10, []byte{byte(set), byte(set >> 8)})
	time.Sleep(500 * time.Millisecond)
	// PL0 + TLS duy trì trong thời gian link down
	gen2WritePL0(th, bar0Phys)
	gen2SetTLS(*wh, root, 2)
	gen2SetTLS(*wh, gpuBDF, 2)
	// clear bit4 → huấn luyện lại
	ctl2, _ := hxcore.PciRd(*wh, root, cap+0x10)
	clr := uint16(ctl2&0xFFFF) &^ 0x10
	_ = hxcore.PciWr(*wh, root, cap+0x10, []byte{byte(clr), byte(clr >> 8)})
	time.Sleep(2000 * time.Millisecond)
}

// PnP Vô hiệu hoá/Bật lại GPU — Khôi phục "GPU is lost" sau Link Disable hoặc giải phóng DMA locks
func gen2PnpRecoverGPU(devID uint16) bool {
	devPattern := "DEV_1F0B"
	if devID == 0x2189 {
		devPattern = "DEV_2189"
	} else if devID == 0 {
		devPattern = "(DEV_1F0B|DEV_2189)"
	} else {
		devPattern = fmt.Sprintf("DEV_%04X", devID)
	}
	ps := fmt.Sprintf(`$devs = Get-PnpDevice -Class Display -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.InstanceId -match '%s' }; `+
		`if ($devs) { foreach ($d in $devs) { try { pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}; try { Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 800; Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } catch {} }; Start-Sleep -Seconds 2; Write-Output "PnP-OK" } `+
		`else { Write-Output 'PnP-NONE' }`, devPattern)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).CombinedOutput()
	fmt.Printf("    PnP Khôi phục: %s (err=%v)\n", strings.TrimSpace(string(out)), err)
	return err == nil && strings.Contains(string(out), "PnP-OK")
}

func gen2PnpRecover40HX() bool {
	return gen2PnpRecoverGPU(0x1F0B)
}

// Khởi động lại NVDisplay Container (phục hồi hiển thị GPU-Z / Task Manager / NVIDIA Control Panel)
func gen2RestartNVDisplay() {
	ps := `sc.exe config NVDisplay.ContainerLocalSystem start= auto | Out-Null; Restart-Service NVDisplay.ContainerLocalSystem -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1; $s = Get-Service -Name NVDisplay.ContainerLocalSystem -ErrorAction SilentlyContinue; if ($s -and $s.Status -ne 'Running') { Start-Service NVDisplay.ContainerLocalSystem -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1 }`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", ps).CombinedOutput()
	fmt.Printf("    NVDisplay Container khởi động lại: %s (err=%v)\n", strings.TrimSpace(string(out)), err)
}

// Trong lúc -hard fallback, PnP reset GPU khiến handle cũ \\.\ThrottleStop / WinRing0 có thể bị hỏng -> mở lại
func gen2ReopenDrivers(th, wh *syscall.Handle) bool {
	if th != nil && *th != 0 {
		hxcore.CloseHandle(*th)
		*th = 0
		if nt, e := hxcore.OpenThrottleStop(); e != nil {
			fmt.Printf("    [!] Mở lại ThrottleStop thất bại: %v\n", e)
		} else {
			*th = nt
		}
	}
	if wh != nil && *wh != 0 {
		hxcore.CloseHandle(*wh)
		*wh = 0
		if nw, e := hxcore.OpenDevice(`\\.\WinRing0_1_2_0`); e != nil {
			fmt.Printf("    [!] Mở lại WinRing0 thất bại: %v\n", e)
			return false
		} else {
			*wh = nw
		}
	}
	return wh == nil || *wh != 0
}

// Khôi phục GPU LNKCTL CCC (0x0140, Common Clock + Extended Synch) và vô hiệu hoá ASPM [1:0]
func gen2RestoreGPULnkctl(wh syscall.Handle, gpuBDF uint32) {
	cap := hxcore.PcieCap(wh, gpuBDF)
	if cap == 0 {
		return
	}
	ctl, _ := hxcore.PciRd(wh, gpuBDF, cap+0x10)
	cur := uint16(ctl & 0xFFFF)
	if (cur&0x0140) != 0x0140 || (cur&0x3) != 0 {
		want := (cur &^ 0x3) | 0x0140
		_ = hxcore.PciWr(wh, gpuBDF, cap+0x10, []byte{byte(want), byte(want >> 8)})
		rb, _ := hxcore.PciRd(wh, gpuBDF, cap+0x10)
		fmt.Printf("    GPU LNKCTL Khôi phục/Vô hiệu hoá ASPM 0x%04X -> 0x%04X\n", cur, rb&0xFFFF)
	}
}

// Stage2 Điều phối: LD → retrain → PnP khôi phục → retrain-only → NVDisplay khởi động lại
func gen2HardFallback(th, wh *syscall.Handle, gpuBDF uint32, bar0Phys uint64, root uint32) {
	fmt.Println("\n[Gen2 -hard] === Quy trình Link Disable Fallback (Stage 2) ===")
	fmt.Println("[Gen2 -hard] Cảnh báo: Quy trình này khiến nvidia-smi tạm thời báo 'GPU is lost',")
	fmt.Println("[Gen2 -hard] khoảng vài giây sau sẽ khôi phục qua PnP. Chỉ dùng khi chắc chắn retrain thông thường không đạt.")
	if bar0raw, _ := hxcore.PciRd(*wh, gpuBDF, 0x10); bar0raw != 0 && bar0raw != 0xFFFFFFFF {
		bar0Phys = uint64(bar0raw & 0xFFFFFFF0)
	}
	fmt.Printf("[Gen2 -hard] BAR0 = 0x%08X\n", bar0Phys)
	didLD := false
	gen2WritePL0(*th, bar0Phys)
	if root != 0xFFFFFFFF {
		gen2RootLinkDisable(*th, wh, gpuBDF, bar0Phys, root)
		didLD = true
	}
	cur := hxcore.LinkSpeed(*wh, gpuBDF)
	fmt.Printf("[Gen2 -hard] Sau Link Disable: Gen%d\n", cur)
	if cur < 2 {
		for i := 0; i < 6; i++ {
			if i == 2 && cur < 2 && root != 0xFFFFFFFF {
				fmt.Println("[Gen2 -hard] Huấn luyện lại vẫn thất bại → Chạy lại chu kỳ Link Disable lần 2")
				gen2RootLinkDisable(*th, wh, gpuBDF, bar0Phys, root)
			}
			gen2WritePL0(*th, bar0Phys)
			gen2SetTLS(*wh, root, 2)
			gen2SetTLS(*wh, gpuBDF, 2)
			bdf := gpuBDF
			tag := "GPU"
			if root != 0xFFFFFFFF && i%2 == 0 {
				bdf, tag = root, "ROOT"
			}
			fmt.Printf("[Gen2 -hard] Huấn luyện lại #%d (%s)...\n", i+1, tag)
			gen2RetrainPulse(*wh, bdf)
			for poll := 0; poll < 25; poll++ {
				time.Sleep(75 * time.Millisecond)
				cur = hxcore.LinkSpeed(*wh, gpuBDF)
				if cur >= 2 {
					break
				}
			}
			if cur >= 2 {
				break
			}
		}
	}
	if didLD {
		if cur >= 2 {
			fmt.Println("[Gen2 -hard] Đã huấn luyện lên Gen2, thực thi khôi phục PnP + khởi động lại NVDisplay")
		} else {
			fmt.Println("[Gen2 -hard] Huấn luyện chưa đạt, vẫn thực thi khôi phục PnP để đảm bảo GPU về trạng thái bình thường")
		}
		gen2PnpRecover40HX()
		if !gen2ReopenDrivers(th, wh) {
			fmt.Println("[Gen2 -hard][!] Mở lại driver thất bại, huỷ bỏ các bước khôi phục tiếp theo")
			gen2VerdictHard(gpuBDF, cur, false)
			return
		}
		time.Sleep(3000 * time.Millisecond)
		if bar0raw, _ := hxcore.PciRd(*wh, gpuBDF, 0x10); bar0raw != 0 && bar0raw != 0xFFFFFFFF {
			bar0Phys = uint64(bar0raw & 0xFFFFFFF0)
		}
		gen2WritePL0(*th, bar0Phys)
		gen2SetTLS(*wh, root, 2)
		gen2SetTLS(*wh, gpuBDF, 2)
		for i := 0; i < 6; i++ {
			bdf := gpuBDF
			if root != 0xFFFFFFFF && i%2 == 0 {
				bdf = root
			}
			gen2RetrainPulse(*wh, bdf)
			for poll := 0; poll < 25; poll++ {
				time.Sleep(75 * time.Millisecond)
				cur = hxcore.LinkSpeed(*wh, gpuBDF)
				if cur >= 2 {
					break
				}
			}
			if cur >= 2 {
				break
			}
		}
		gen2RestoreGPULnkctl(*wh, gpuBDF)
		gen2RestartNVDisplay()
		cur = hxcore.LinkSpeed(*wh, gpuBDF)
		fmt.Printf("[Gen2 -hard] Sau khôi phục PnP: Gen%d\n", cur)
	}
	gen2VerdictHard(gpuBDF, cur, cur >= 2)
}

func gen2VerdictHard(gpuBDF uint32, cur uint32, success bool) {
	gen2Succeeded = success
	if success {
		deleteGen2Retry()
	} else {
		scheduleGen2Retry(retryDepth())
	}
	gpuBus := (gpuBDF >> 8) & 0xFF
	st := fmt.Sprintf("Kết luận (Link Disable Fallback): %s\nVị trí 40HX: %02x:%02x.%x\nBăng thông: Hiện tại Gen%d\nDriver: ThrottleStop=✓ WinRing0=✓ (BYOVD, thu hồi theo chính sách)\n",
		map[bool]string{true: "✅ Gen2 Thành công", false: "❌ Gen2 Thất bại (xem log/gửi cộng đồng)"}[success],
		gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7, cur)
	if wErr := hxcore.WriteGen2Status(st); wErr != nil {
		fmt.Printf("[Gen2 -hard] Ghi trạng thái thất bại: %v\n", wErr)
	}
	if !hasArg("-silent") && !hasArg("-y") {
		icon := uint(mbIconInfo)
		txt := fmt.Sprintf("Gen2 Fallback: Hiện tại Gen%d\n", cur)
		if success {
			txt += "=== MỞ KHOÁ GEN2 THÀNH CÔNG ==="
		} else {
			txt += "Vẫn ở Gen1, fallback chưa đạt (xem chi tiết trong log)."
			icon = mbIconError
		}
		msgbox("40HX Gen2", txt, icon)
	}
}

// ---------- v2.6.0: Gen2 自动重试 + Stage2 自动回退开关 + 策略配置 ----------

// retryDepth: 当前自动重试深度(-retrydepth=N, 0=登录任务首次执行)
func retryDepth() int {
	for _, a := range os.Args {
		if strings.HasPrefix(a, "-retrydepth=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(a, "-retrydepth=")); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

// gen2AutoHardEnabled: Stage2(Link Disable + PnP 恢复)自动执行开关, 默认开。
// 关闭: reg add HKLM\SOFTWARE\40HXUnlock /v Gen2AutoHard /t REG_DWORD /d 0 /f
// (40HX 是唯一显示卡、不希望登录后链路瞬断数秒黑屏的用户可关)
func gen2AutoHardEnabled() bool {
	return hxcore.ConfigInt("Gen2AutoHard", 1) != 0
}

// scheduleGen2Retry: 失败后安排一次性自动重试(SYSTEM, 静默, 默认 15 分钟后)。
// 覆盖"开机后驱动/GSP 就绪慢""链路状态恰好卡住"等时序类失败(社区 #12);
// depth 为已重试次数, 超出策略预算(Gen2RetryCount)即不再排; 成功路径 deleteGen2Retry。
func scheduleGen2Retry(depth int) {
	count, interval := hxcore.Gen2RetryPolicy()
	if depth >= count {
		fmt.Printf("[Gen2] 自动重试预算已用完(%d/%d), 等下次登录再试\n", depth, count)
		return
	}
	t := time.Now().Add(time.Duration(interval) * time.Minute)
	if t.Day() != time.Now().Day() {
		fmt.Println("[Gen2] Gần nửa đêm, bỏ qua lịch trình thử lại lần này (tác vụ once qua ngày không đáng tin cậy)")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	abs, _ := filepath.Abs(exe)
	out, err := hxcore.RunOut("schtasks.exe", "/create", "/tn", gen2RetryTask,
		"/tr", fmt.Sprintf("\"%s\" -gen2 -silent -retrydepth=%d", abs, depth+1),
		"/sc", "once", "/st", t.Format("15:04"), "/ru", "SYSTEM", "/f")
	if err != nil {
		fmt.Printf("[Gen2] Tạo tác vụ thử lại thất bại (không ảnh hưởng mở khoá): %s\n", strings.TrimSpace(out))
		return
	}
	fmt.Printf("[Gen2] Đã lên lịch tự động thử lại sau %d phút (%d/%d, tác vụ %s)\n", interval, depth+1, count, gen2RetryTask)
}

// deleteGen2Retry: Xoá tác vụ thử lại nếu có sau khi đạt Gen2 thành công
func deleteGen2Retry() {
	hxcore.RunOut("schtasks.exe", "/delete", "/tn", gen2RetryTask, "/f")
}

// gen2AcquireSingleInstance: v2.6.0 Bảo vệ đơn phiên bản
func gen2AcquireSingleInstance() (bool, func()) {
	name, _ := windows.UTF16PtrFromString("Global\\40HXGen2SingleInstance")
	h, err := windows.CreateMutex(nil, true, name)
	if err != nil {
		fmt.Println("[Gen2] Tạo mutex đơn phiên bản thất bại, bỏ qua:", err)
		return true, func() {}
	}
	if windows.GetLastError() == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(windows.Handle(h))
		return false, nil
	}
	return true, func() {
		_ = windows.ReleaseMutex(windows.Handle(h))
		_ = windows.CloseHandle(windows.Handle(h))
	}
}

// waitForNvDriver: Đợi driver nvlddmkm chuyển sang trạng thái RUNNING
func waitForNvDriver(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		out, _ := hxcore.RunOut("sc.exe", "query", "nvlddmkm")
		if strings.Contains(out, "does not exist") || strings.Contains(out, "chưa cài đặt") ||
			strings.Contains(out, "1060") {
			fmt.Println("[Gen2] Không phát hiện dịch vụ nvlddmkm, bỏ qua chờ đợi và tiếp tục mở khoá")
			return true
		}
		if strings.Contains(out, "RUNNING") {
			return true
		}
		if time.Now().After(deadline) {
			fmt.Printf("[Gen2] nvlddmkm không chuyển sang RUNNING trong vòng %s (xem log), tiếp tục mở khoá\n", timeout)
			return false
		}
		fmt.Println("[Gen2] Đang chờ nvlddmkm sẵn sàng...")
		time.Sleep(2 * time.Second)
	}
}

// cleanupByovd: Dừng và xoá dịch vụ driver ThrottleStop/WinRing0 sau khi dùng xong
func cleanupByovd() {
	if hxcore.DriverStrategy() == hxcore.DriverStrategyResident {
		fmt.Println("[Gen2] Chiến lược thường trú: Giữ lại dịch vụ driver và file (có thể gỡ bằng GUI/Uninstaller)")
		return
	}
	appRunning := throttleStopAppRunning()
	for _, d := range []struct{ name, file string }{
		{"ThrottleStop", "ThrottleStop.sys"},
		{"WinRing0_1_2_0", "WinRing0x64.sys"},
	} {
		if appRunning {
			continue
		}
		hxcore.RunOut("sc.exe", "stop", d.name)
		hxcore.RunOut("sc.exe", "delete", d.name)
		os.Remove(filepath.Join(os.Getenv("SystemRoot")+"\\System32\\drivers", d.file))
	}
}

// gen2Notify: Thông báo lỗi; chế độ im lặng không hiện hộp thoại
func gen2Notify(txt string) {
	if !hasArg("-silent") && !hasArg("-y") {
		msgbox("40HX Gen2", txt, mbIconError)
	}
}

// gen2StatusFail: Ghi lý do Gen2 không thực thi/thất bại vào file trạng thái
func gen2StatusFail(reason string) {
	ident := map[bool]string{true: "Quản trị viên/SYSTEM", false: "Người dùng thường (Bị hạn chế)"}[isAdmin()]
	code := hxcore.StatusDrvFail
	errCode := "DRV_BLOCKED"
	low := strings.ToLower(reason)
	if strings.Contains(low, "pci") || strings.Contains(low, "không tìm thấy") || strings.Contains(low, "chưa định vị") {
		code = hxcore.StatusNoGPU
		errCode = "GPU_NOT_FOUND"
	}
	_ = hxcore.WriteStructuredGen2Status(hxcore.StatusContract{
		StatusCode: code,
		ErrorCode:  errCode,
		Details: []string{
			"❌ Gen2 Chưa thực thi: " + reason,
			"Quyền thực thi: " + ident,
		},
	})
}

// ===================== Gỡ cài đặt / Trạng thái =====================

func uninstall() {
	if !isAdmin() {
		fmt.Println("[!] Cần quyền quản trị viên.")
		msgbox("Bộ cài đặt 40HX", "Cần quyền quản trị viên.\nVui lòng nhấp chuột phải vào chương trình -> Chọn Run as administrator.", mbIconError)
		return
	}
	if lockOnce(`Local\40HXUninstaller_v1`) == nil {
		msgbox("Bộ cài đặt 40HX", "Chương trình gỡ cài đặt đang chạy, vui lòng không nhấp trùng lặp.", mbIconInfo)
		return
	}
	fmt.Println("=== Gỡ cài đặt mở khoá 40HX (v3.0.0 Cấp thành phần) ===")
	fmt.Print("[1/8] Xoá tác vụ lịch trình ... ")
	if rem := hxcore.UninstallTasks(); len(rem) > 0 {
		fmt.Println("Hoàn thành")
	} else {
		fmt.Println("Không tìm thấy (bỏ qua)")
	}
	fmt.Print("[2/8] Xoá khoá Run tự khởi động Gen2 ... ")
	hxcore.UninstallRunKey()
	fmt.Println("Hoàn thành")
	fmt.Print("[3/8] Xoá mục khởi động firmware '40HX Unlock' ... ")
	if hxcore.UninstallBootEntry() {
		fmt.Println("Hoàn thành")
	} else {
		fmt.Println("Không tìm thấy (có thể đã được gỡ bỏ)")
	}
	fmt.Print("[4/8] Xoá EFI mở khoá trong ESP ... ")
	if hxcore.UninstallEspEfi() {
		fmt.Println("Hoàn thành")
	} else {
		fmt.Println("Không tìm thấy/bỏ qua")
	}
	fmt.Println("[5/8] Dừng và xoá dịch vụ driver...")
	hxcore.UninstallDriverServices()
	fmt.Println("[6/8] Xoá file driver...")
	hxcore.UninstallDriverFiles()
	fmt.Print("[6.5/8] Xoá EnableGpuFirmware (khôi phục GSP về tắt mặc định) ... ")
	if hxcore.UninstallGspKey() {
		fmt.Println("Hoàn thành")
	} else {
		fmt.Println("Không tìm thấy (bỏ qua)")
	}
	fmt.Print("[6.6/8] Dọn dẹp ProgramData + khoá chính sách ... ")
	hxcore.UninstallProgramData()
	fmt.Println("Hoàn thành")
	fmt.Print("[6.7/8] Dọn dẹp mục loại trừ Defender ... ")
	if err := hxcore.RemoveDefenderExclusions(); err != nil {
		fmt.Println("Chưa thực thi (có thể bỏ qua):", err)
	} else {
		fmt.Println("Hoàn thành")
	}
	fmt.Println("[7/8] Kiểm tra tàn dư...")
	left := hxcore.CheckLeftover()
	fmt.Println()
	fmt.Println("Gỡ cài đặt hoàn tất. Khuyến nghị khởi động lại máy tính.")
	fmt.Println("  Lưu ý: Thiết lập nguồn khi cài đặt (Fast Startup/ASPM) được giữ nguyên — cách khôi phục xem README §2.4.")
	icon := uint(mbIconInfo)
	txt := "Gỡ cài đặt hoàn tất.\nKhuyến nghị khởi động lại máy tính.\n\nLưu ý: Thiết lập nguồn khi cài đặt (Fast Startup/ASPM)\nđược giữ nguyên theo sở thích nguồn — cách khôi phục xem README §2.4.\n"
	if len(left) > 0 {
		icon = mbIconError
		txt += "\nVẫn còn tàn dư:\n" + strings.Join(left, "\n")
	}
	txt += "\nNhật ký chi tiết: " + filepath.Join(os.TempDir(), "40HX_installer.log")
	msgbox("Bộ cài đặt 40HX", txt, icon)
}

func status() {
	prof, gpuOK := hxcore.FindGPUWithProfile()
	cardName := "40HX"
	if gpuOK {
		cardName = prof.Name
	}
	fmt.Printf("=== Trạng thái mở khoá %s ===\n", cardName)
	sb := hxcore.SecureBootOn()
	ts := hxcore.TestSigningOn()
	fmt.Printf("Phát hiện GPU %s: %v\n", cardName, gpuOK)
	fmt.Printf("Secure Boot: %v\n", sb)
	fmt.Printf("Testsigning: %v\n", ts)

	gs := false
	if gpuOK && !prof.FirmwareUnlock {
		fmt.Printf("Phần cứng GPU: %s (%s, DEV_%04X)\n", prof.Name, prof.Family, prof.DeviceID)
		fmt.Println("Firmware GSP: Không cần (Kiến trúc TU116 không phụ thuộc GSP-RM)")
	} else {
		gs = hxcore.GspEnabled()
		fmt.Printf("Bật GSP (EnableGpuFirmware=1): %v\n", gs)
		if sub, adapter, fw := hxcore.GspDiag(); sub != "" {
			fmt.Printf("  Khoá GSP: Class\\%s (fw=%d)\n", sub, fw)
			fmt.Printf("  AdapterString: %s\n", adapter)
		} else {
			fmt.Println("  [!] " + adapter)
		}
	}
	dep := hxcore.InspectGen2Drivers()
	if !hxcore.Gen2DriversDeployedOnce() {
		fmt.Println("Driver Gen2: Chưa từng triển khai — chạy bộ cài đặt và khởi động lại để có hiệu lực")
	} else {
		for _, d := range dep {
			svcS := "Chưa đăng ký"
			if d.SvcReg {
				svcS = d.SvcStart
				if d.SvcRunning {
					svcS += "/Đang chạy"
				}
			}
			fmt.Printf("Driver Gen2 %-16s Nguồn sao lưu=%v  System32=%s  Dịch vụ=%s\n",
				d.File, map[bool]string{true: "OK", false: "Không"}[d.BackupOK], d.SysState.String(), svcS)
		}
	}
	if ex, err := hxcore.DefenderExclusionsPresent(); err != nil {
		fmt.Println("Loại trừ Defender: Truy vấn thất bại (" + err.Error() + ")")
	} else if ex {
		fmt.Println("Loại trừ Defender: Đã thêm danh sách trắng (OK)")
	} else {
		fmt.Println("Loại trừ Defender: Còn thiếu — phần mềm diệt virus có thể xoá nhầm driver")
	}
	st := hxcore.ReadUnlockStateV2(5, 800)
	tsRun := st.TSOK
	winringRun := st.WinRingOK
	speed := st.Speed
	ss0 := st.SS0
	ss0ok := st.SS0OK

	if gpuOK && !prof.FirmwareUnlock {
		fmt.Printf("WinRing0 (Truy cập PCI Config): %v\n", winringRun)
		fmt.Printf("ThrottleStop: %v (30HX không cần driver này)\n", tsRun)
		if winringRun {
			fmt.Printf("Băng thông PCIe: Gen%d\n", speed)
		} else {
			fmt.Println("Driver chưa chạy (cần WinRing0 để kiểm tra và huấn luyện lại PCIe)")
		}
	} else {
		fmt.Printf("ThrottleStop: %v\n", tsRun)
		fmt.Printf("WinRing0: %v\n", winringRun)
		if tsRun && winringRun {
			fmt.Printf("Băng thông PCIe: Gen%d\n", speed)
			if ss0ok {
				fmt.Printf("SS0 (Hashrate): 0x%08x %s\n", ss0, map[bool]string{true: "(Đã mở khoá)", false: "(Khoá)"}[st.Unlocked])
			}
		} else {
			fmt.Println("Driver chưa chạy (sẵn sàng kiểm tra Gen2/trạng thái sau khi cài đặt)")
		}
	}

	diag := []string{}
	if !gpuOK {
		diag = append(diag, fmt.Sprintf("· Không phát hiện %s —— Vui lòng kiểm tra cắm card và cài driver", cardName))
	}
	if gpuOK && !prof.FirmwareUnlock {
		if !winringRun {
			diag = append(diag, "· Driver WinRing0 chưa chạy: Chạy thủ công 40HXInstaller.exe -gen2 hoặc khởi động lại hệ thống")
		} else {
			diag = append(diag, fmt.Sprintf("· Băng thông PCIe hiện tại: Gen%d", speed))
			if speed < 2 {
				diag = append(diag, "· Link vẫn là Gen1: Cần kiểm tra tốc độ khe cắm BIOS mainboard, dây cáp riser hoặc giới hạn VBIOS/Strap")
			}
		}
	} else {
		if sb {
			diag = append(diag, "· Secure Boot đang bật: Cần vào BIOS tắt đi, nếu không EFI mở khoá sẽ bị từ chối")
		}
		if ts {
			diag = append(diag, "· Testsigning đang bật — v2.5 không cần thiết, có thể chạy bcdedit /set testsigning off để tắt")
		}
		if !gs {
			diag = append(diag, "· GSP chưa bật: Có thể đen màn hình sau khi mở khoá. Chạy bộ cài đặt (tự động bật EnableGpuFirmware=1)")
		}
		if !tsRun || !winringRun {
			diag = append(diag, "· Driver chưa chạy: Sẽ tự khởi động sau khi đăng nhập; hoặc chạy thủ công 40HXInstaller.exe -gen2")
		}
		if tsRun && winringRun {
			if !ss0ok {
				diag = append(diag, "· Driver đã chạy nhưng không đọc được thanh ghi hashrate (bất thường)")
			} else if ss0 == 0x88888888 {
				diag = append(diag, fmt.Sprintf("· SS0=0x%08x: Hashrate đã mở khoá! PCIe Gen%d", ss0, speed))
			} else {
				diag = append(diag, fmt.Sprintf("· SS0=0x%08x: Hashrate vẫn khoá —— Khi khởi động lại 40HX Unlock EFI chưa thực thi thành công", ss0))
				efiDiag := hxcore.AnalyzeEfiLog()
				if efiDiag != "" {
					diag = append(diag, efiDiag)
				}
			}
		}
	}

	msg := fmt.Sprintf("Trạng thái mở khoá %s\n========================\n", cardName)
	msg += fmt.Sprintf("GPU %s: %v    Secure Boot: %v\n", cardName, map[bool]string{true: "✓", false: "✗"}[gpuOK], map[bool]string{true: "Bật!", false: "Tắt (OK)"}[sb])
	if gpuOK && !prof.FirmwareUnlock {
		msg += fmt.Sprintf("WinRing0: %v    PCIe: Gen%d\n", map[bool]string{true: "✓", false: "✗"}[winringRun], speed)
	} else {
		msg += fmt.Sprintf("Testsigning: %v    GSP: %v\n", map[bool]string{true: "✓", false: "✗"}[ts], map[bool]string{true: "✓", false: "✗"}[gs])
		msg += fmt.Sprintf("ThrottleStop: %v  WinRing0: %v\n", map[bool]string{true: "✓", false: "✗"}[tsRun], map[bool]string{true: "✓", false: "✗"}[winringRun])
		if tsRun && winringRun {
			msg += fmt.Sprintf("PCIe: Gen%d    SS0: 0x%08x\n", speed, ss0)
		}
	}
	msg += "\nChẩn đoán:\n" + strings.Join(diag, "\n")
	if len(diag) == 0 {
		msg += "· Mọi thứ hoạt động bình thường"
	}
	msg += "\n\nNhật ký chi tiết: " + filepath.Join(os.TempDir(), "40HX_installer.log")
	msgbox(fmt.Sprintf("Trạng thái %s", cardName), msg, mbIconInfo)
	fmt.Println("=== Kết thúc trạng thái ===")
}

func pause() {
	// GUI 版: 无需按 Enter; 输出已入日志, 交互收尾用消息框
}
