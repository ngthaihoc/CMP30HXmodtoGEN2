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
	hxcore.DriverFileProvider = func(filename string) ([]byte, error) {
		return embedded.ReadFile("embed/" + filename)
	}
	// GUI 无窗口版(v1.1): 输出全部镜像到日志(默认 %TEMP%\40HX_installer.log, 可 -log 指定)
	setupLog("40HX_installer.log")
	// v3.0: Mặc định khởi chạy Modern Web Control Center khi nhấp đúp hoặc UAC -elevated
	if hasArg("-gui-classic") {
		runGUI()
		return
	}
	if len(os.Args) <= 1 || (len(os.Args) == 2 && os.Args[1] == "-elevated") || hasArg("-ui") || hasArg("-web") || hasArg("-gui") {
		runWebGUI()
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
		case "-reset":
			resetPnpMain()
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
	hxcore.SelfElevate("Trình Cài Đặt 40HX / 30HX")
}

var (
	gen2Succeeded bool
)

const (
	mbIconInfo  = hxcore.MbIconInfo
	mbIconError = hxcore.MbIconError
	mbIconWarn  = hxcore.MbIconWarn // MB_ICONWARNING: v2.6.0: EFI 跳过/部分成功等"可继续但要注意"场景
	mbYesNo     = hxcore.MbYesNo    // MB_YESNO → 返回 IDYES=6 / IDNO=7
)

var (
	procCreateMutex = syscall.NewLazyDLL("kernel32.dll").NewProc("CreateMutexW")
)

func msgbox(title, text string, icon uint) {
	// -y / -silent(自动化/自启动) 时不弹框
	if hasArg("-y") || hasArg("-silent") {
		return
	}
	hxcore.MsgBox(title, text, icon)
}

// msgboxYesNo: 是/否询问。自动模式: -y→true(全自动继续), -silent→false(不打扰)。
func msgboxYesNo(title, text string) bool {
	if hasArg("-y") {
		return true
	}
	if hasArg("-silent") {
		return false
	}
	return hxcore.MsgBoxYesNo(title, text)
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
	return hxcore.IsAdmin()
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

func resetPnpMain() {
	if !isAdmin() {
		return
	}
	bus := &hxcore.ProductionBus{}
	bus.PnpResetDevice(0x1F0B) // 40HX only; 30HX is protected from PnP resets
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
	fmt.Println("    · Triển khai EFI mở khoá vào phân vùng EFI hệ thống (2 đường dẫn)...")
	esp := hxcore.MountESP()
	if esp == "" {
		if hxcore.FirmwareIsLegacy() {
			fmt.Println("[!] Hệ thống đang khởi động kiểu BIOS cũ (Legacy)+MBR — Không có phân vùng EFI, không thể triển khai EFI mở khoá.")
			fmt.Println("    Mở khoá hiệu năng cần UEFI+GPT: Vui lòng chuyển đổi ổ đĩa sang GPT bằng công cụ mbr2gpt (xem hướng dẫn trong hộp thoại),")
			fmt.Println("    Sau khi chuyển đổi và bật UEFI trong BIOS, hãy chạy lại trình cài đặt này.")
			fmt.Println("    [i] Tác vụ tự mở khoá Gen2 khi đăng nhập không bị ảnh hưởng, tiếp tục đăng ký (xem [7/8]).")
			msgbox("Trình Cài Đặt 40HX / 30HX (Cần chuyển đổi ổ đĩa sang GPT)",
				"Hệ thống của bạn đang chạy ở chế độ BIOS cũ (Legacy)+MBR, không có phân vùng EFI,\n"+
					"nên không thể cài đặt EFI mở khoá hiệu năng.\n\n"+
					"Vui lòng chuyển đổi sang UEFI+GPT (công cụ mbr2gpt chính thức của Microsoft, không mất dữ liệu):\n"+
					"  1. Sao lưu dữ liệu quan trọng; Đảm bảo BitLocker đã tắt hoặc tạm ngưng\n"+
					"  2. Mở CMD với quyền Admin và chạy:  mbr2gpt /validate /allowfullos\n"+
					"  3. Khi hiện 'Validation completed successfully', chạy tiếp:\n"+
					"        mbr2gpt /convert /allowfullos\n"+
					"  4. Khởi động lại vào BIOS, chuyển chế độ Boot từ Legacy sang UEFI (Tắt CSM)\n"+
					"  5. Vào Windows và chạy lại trình cài đặt này\n\n"+
					"Lưu ý: Quá trình chuyển đổi không thể đảo ngược; yêu cầu Windows 10 1703+ / Win 11 và bo mạch chủ hỗ trợ UEFI.\n"+
					"Trình cài đặt sẽ tiếp tục đăng ký phần tự mở khoá Gen2.",
				mbIconWarn)
		} else {
			fmt.Println("[!] Không thể gắn kết phân vùng EFI (mountvol /S thất bại)")
			fmt.Println("    Hệ thống là UEFI, nguyên nhân phổ biến: BitLocker đang bật hoặc phân vùng ESP có bất thường.")
			fmt.Println("    Có thể cài thủ công: mountvol S: /S, chép 40HXUNLK.EFI vào S:\\EFI\\40HX\\, mountvol S: /D")
			msgbox("Trình Cài Đặt 40HX / 30HX (Gắn kết phân vùng EFI thất bại)",
				"Không thể gắn kết phân vùng EFI (mountvol /S thất bại), EFI mở khoá chưa được triển khai lần này.\n"+
					"Khởi động hệ thống không bị ảnh hưởng.\n\n"+
					"Nguyên nhân phổ biến: BitLocker/mã hóa bên thứ ba đang bật, phân vùng ESP bất thường.\n"+
					"Bạn có thể sao chép thủ công (xem nhật ký và tài liệu Hướng Dẫn Sửa Lỗi EFI).\n\n"+
					"Trình cài đặt sẽ tiếp tục đăng ký phần tự mở khoá Gen2.",
				mbIconWarn)
		}
		return false
	}
	fmt.Printf("    ESP gắn kết tại %s: \\\n", esp)
	fb, err := deployEspEfi(esp)
	hxcore.UnmountESP(esp)
	if err != nil {
		fmt.Println("[!] Sao chép EFI thất bại:", err)
		msgbox("Trình Cài Đặt 40HX / 30HX (Ghi EFI thất bại)",
			"Sao chép EFI mở khoá vào ESP thất bại:\n"+err.Error()+
				"\n\nKhởi động hệ thống không bị ảnh hưởng, máy vẫn có thể khởi động vào Windows bình thường.\n\n"+
				"Nếu muốn triển khai thủ công, xem mục Cài đặt thủ công trong Hướng Dẫn Cứu Hộ EFI.\n\n"+
				"Trình cài đặt sẽ tiếp tục hoàn thành phần Gen2.", mbIconWarn)
		return false
	}
	if fb {
		fmt.Println("    [!] Phát hiện file bootx64.efi gốc, đã sao lưu thành bootx64.efi.40hx.bak")
	}
	efiOK = true

	// BootOrder (v2.4: 写回验证 + BIOS 指引弹框); 仅 EFI 部署成功才执行
	// BootOrder: Thiết lập mục khởi động ưu tiên số 1
	fmt.Println("    · Thiết lập mục khởi động firmware ('40HX Unlock' ưu tiên hàng đầu)...")
	bootOK := false
	if err := setupBootEntry(); err != nil {
		fmt.Println("[!] Tự động thiết lập mục khởi động thất bại:", err)
	} else {
		if ex, first, ord := verifyBootEntry(); ex {
			bootOK = first
			if first {
				fmt.Println("    Mục khởi động đã được đặt đầu tiên và xác minh thành công (displayorder hàng đầu)")
			} else {
				fmt.Println("    [!] Mục khởi động đã được tạo, nhưng chưa nằm đầu tiên trong displayorder:")
				fmt.Println("        Thứ tự khởi động hiện tại: " + ord)
				fmt.Println("        Vui lòng vào BIOS đặt '40HX Unlock' làm mục khởi động đầu tiên (xem hướng dẫn)")
			}
		} else {
			fmt.Println("    [!] Không tìm thấy mục '40HX Unlock' trong danh sách khởi động firmware")
			fmt.Println("        (Một số bo mạch chủ bỏ qua lệnh ghi BCD, vui lòng vào BIOS để thêm/đặt lên đầu)")
		}
	}
	if !bootOK {
		// Hướng dẫn BIOS
		msgbox("Trình Cài Đặt 40HX / 30HX (Lưu ý quan trọng: Vui lòng làm theo hướng dẫn)",
			"Mục khởi động tự động chưa được firmware bo mạch chủ chấp nhận hoàn toàn.\n"+
				"Vui lòng khởi động lại máy, bấm Del/F2 vào BIOS và thực hiện các thiết lập sau:\n\n"+
				"1. Tắt Secure Boot (nếu bật, file EFI chưa ký số sẽ bị chặn)\n"+
				"2. Tắt Fast Boot trong BIOS (nếu có)\n"+
				"3. Trong mục [Thứ tự khởi động / Boot Priority], đặt '40HX Unlock' lên vị trí đầu tiên\n"+
				"   hoặc chọn khởi động thủ công từ file \\EFI\\40HX\\40HXUNLK.EFI\n"+
				"4. Nếu danh sách chỉ có Windows Boot Manager:\n"+
				"   - Một số bo mạch chủ cần tắt CSM (chuyển sang thuần UEFI) mới hiện mục này\n"+
				"   - Hoặc chọn khởi động trực tiếp từ phân vùng UEFI (chạy qua bản dự phòng bootx64)\n\n"+
				"Trình cài đặt đã triển khai payload mở khoá vào cả 2 vị trí:\n"+
				"  \\EFI\\40HX\\40HXUNLK.EFI  (Đường dẫn BCD chính)\n"+
				"  \\EFI\\Boot\\bootx64.efi    (Đường dẫn dự phòng chuẩn)\n\n"+
				"Nhật ký chi tiết: "+filepath.Join(os.TempDir(), "40HX_installer.log"),
			mbIconError)
	}
	return efiOK
}

func install() {
	fmt.Println("==============================================")
	fmt.Println("  Trình Cài Đặt Mở Khoá CMP 40HX / 30HX Windows v3.0.0")
	fmt.Println("  Mở khoá Tensor (EFI V70 + Bật GSP) + PCIe Gen2 + Tự khởi động")
	fmt.Println("==============================================")

	if !isAdmin() {
		fmt.Println("[!] Cần quyền Quản trị viên (Administrator).")
		msgbox("Trình Cài Đặt 40HX / 30HX", "Cần quyền Quản trị viên.\nVui lòng nhấp chuột phải -> Chọn Run as administrator.", mbIconError)
		return
	}
	if lockOnce(`Local\40HXInstaller_v1`) == nil {
		msgbox("Trình Cài Đặt 40HX / 30HX", "Trình cài đặt đang chạy, vui lòng không nhấp trùng lặp.", mbIconInfo)
		return
	}

	// 0. Kiểm tra cài đặt trước đó
	if alreadyInstalled() {
		fmt.Println("[!] Phát hiện mở khoá 40HX đã được cài đặt trước đó (mục khởi động/khoá GSP đã tồn tại).")
		if !msgboxYesNo("Trình Cài Đặt 40HX / 30HX",
			"Phát hiện mở khoá 40HX đã được cài đặt trên hệ thống này.\n\n"+
				"Cài đặt lại sẽ ghi đè thiết lập hiện có (driver và mục khởi động sẽ được cập nhật, không ảnh hưởng khởi động Windows).\n"+
				"Nếu bạn muốn sửa lỗi hoặc nâng cấp, chọn \"Yes\" để tiếp tục;\n"+
				"Nếu chỉ vô tình mở, chọn \"No\" để giữ nguyên trạng thái.\n\n"+
				"Bạn có muốn tiếp tục cài đặt lại?") {
			fmt.Println("Đã huỷ — Giữ nguyên trạng thái cài đặt hiện tại.")
			return
		}
		fmt.Println("    Người dùng xác nhận, tiếp tục cài đặt ghi đè.")
	}

	// 1. Kiểm tra GPU
	fmt.Print("[1/8] Kiểm tra GPU ... ")
	if !hxcore.FindGPU() {
		fmt.Println("Không tìm thấy " + gpuVenDev)
		fmt.Println("[!] Không phát hiện card CMP 40HX / 30HX. Dừng quá trình.")
		msgbox("Trình Cài Đặt 40HX / 30HX", "Không tìm thấy card màn hình CMP 40HX / 30HX tương thích.\nQuá trình cài đặt đã dừng lại.", mbIconError)
		return
	}
	fmt.Println("Đã tìm thấy GPU tương thích!")

	// 2. Secure Boot
	fmt.Print("[2/8] Kiểm tra Secure Boot ... ")
	if hxcore.SecureBootOn() {
		fmt.Println("Đang BẬT!")
		fmt.Println("[!] Secure Boot đang bật, file EFI mở khoá chưa ký sẽ bị BIOS từ chối nạp.")
		msgbox("Trình Cài Đặt 40HX / 30HX (Cần tắt Secure Boot)",
			"Phát hiện Secure Boot đang BẬT, file EFI mở khoá sẽ bị BIOS từ chối nạp.\n\n"+
				"Vui lòng vào BIOS tắt Secure Boot trước khi chạy bộ cài:\n"+
				"  1. Khởi động lại máy, bấm Del / F2 (hoặc F1/F10/F12 tuỳ bo mạch chủ)\n"+
				"  2. Tìm mục Security / Boot\n"+
				"  3. Chuyển Secure Boot sang Disabled\n"+
				"  4. Bấm F10 lưu và khởi động lại vào Windows\n\n"+
				"Đây là bước bắt buộc vì EFI mở khoá không có chữ ký số của Microsoft.",
			mbIconError)
		return
	}
	fmt.Println("Đã tắt / Không khả dụng (OK)")

	// 3. Test Signing
	fmt.Print("[3/8] Kiểm tra Test Signing ... ")
	if hxcore.TestSigningOn() {
		fmt.Println("Đang bật — v2.5+ không cần, có thể tắt bằng: bcdedit /set testsigning off")
	} else {
		fmt.Println("Đã tắt (OK) — v2.5+ hoàn toàn không cần Test Signing")
	}

	// 3.5 GSP
	fmt.Print("[3.5/8] Bật GSP (EnableGpuFirmware) ... ")
	if hxcore.GspEnabled() {
		if sub, _, fw := hxcore.GspDiag(); sub != "" {
			fmt.Printf("Đã bật (OK) — Class\\%s EnableGpuFirmware=%d\n", sub, fw)
		} else {
			fmt.Println("Đã bật (OK)")
		}
	} else {
		if err := enableGsp(); err != nil {
			_, adapterDiag, _ := hxcore.GspDiag()
			fmt.Println("Cài đặt thất bại:", err)
			if adapterDiag != "" && !strings.Contains(adapterDiag, "Không khớp") {
				fmt.Println("    [!] AdapterString thực tế:", adapterDiag)
			}
			msgbox("Trình Cài Đặt 40HX / 30HX", "Thiết lập EnableGpuFirmware=1 thất bại (cần quyền Admin).\nSau khi mở khoá có thể bị lỗi Code 43.\nLỗi: "+err.Error(), mbIconError)
			return
		}
		fmt.Println("Đã đặt EnableGpuFirmware=1 (Có hiệu lực sau khi khởi động lại)")
		fmt.Println("    [!] GSP bắt buộc: Nếu không driver sẽ không nhận trạng thái mở khoá -> Lỗi Code 43")
	}

	// 3.6 Nguồn
	fmt.Print("[3.6/8] Thiết lập nguồn (Khởi động nhanh + Tiết kiệm điện PCIe) ... ")
	pwrNotes := applyPowerSettings()
	fmt.Println("Hoàn thành")
	for _, n := range pwrNotes {
		fmt.Println("    - " + n)
	}

	// 4. Driver
	fmt.Println("[4/8] Chuẩn bị driver Gen2 BYOVD (ThrottleStop + WinRing0)...")
	installDrivers()

	// 4.5 Defender
	fmt.Print("[4.5/8] Thêm loại trừ Defender (chống xoá nhầm driver) ... ")
	if err := hxcore.AddDefenderExclusions(); err != nil {
		fmt.Println("Chưa thực hiện (có thể bỏ qua):", err)
	} else {
		fmt.Println("Đã thêm loại trừ cho file driver ThrottleStop/WinRing0")
	}

	// 5+6. EFI & Boot Order
	fmt.Println("[5/8]+[6/8] Triển khai EFI mở khoá và mục khởi động firmware...")
	efiOK := installEFI()

	// 7. Gen2 Startup
	fmt.Println("[7/8] Đăng ký tự mở khoá Gen2 khi đăng nhập (Tác vụ SYSTEM + Khoá Run)...")
	setRunKey()
	if err := setupGen2Task(); err != nil {
		fmt.Println("[!]", err)
		msgbox("Trình Cài Đặt 40HX / 30HX (Đăng ký tự chạy thất bại)",
			"Đăng ký tác vụ tự mở khoá Gen2 khi đăng nhập thất bại.\n\n"+
				"Vui lòng chạy lại với quyền Admin bằng lệnh:\n"+
				"  40HXInstaller.exe -task\n\n"+
				"Các bước cài đặt khác đã hoàn thành.", mbIconWarn)
	}

	fmt.Println()
	fmt.Println("Cài đặt hoàn tất!")
	if efiOK {
		fmt.Println("  Lần khởi động tiếp theo: Firmware sẽ tự chạy '40HX Unlock' (Mở khoá Tensor) -> Vào Windows")
	} else {
		fmt.Println("  [!] Chưa triển khai EFI mở khoá hiệu năng — Hiệu năng tạm thời chưa mở khoá,")
		fmt.Println("      Làm theo hướng dẫn (mbr2gpt/cài thủ công) rồi chạy lại bộ cài.")
	}
	fmt.Println("  GSP đã bật: Driver hoạt động ở chế độ GSP-RM, sau mở khoá không lo lỗi Code 43")
	fmt.Println("  Sau khi đăng nhập: Gen2 tự động mở khoá (chạy ẩn, không hiện cửa sổ)")
	fmt.Println("  [!] Quá trình cài đặt không ép xung PCIe ngay, chỉ kích hoạt khi đăng nhập sau reboot (tránh xung đột driver)")
	fmt.Println("  Kiểm tra sau reboot: Chạy 40HXCheck.exe kiểm tra trạng thái mở khoá (SS0=0x88888888 là thành công)")

	efiNote := ""
	if efiOK {
		efiNote = "Lưu ý khi khởi động lại máy:\n" +
			"  · Nếu màn hình đen / hiển thị log chữ 40HX khoảng 10~30 giây là bình thường (đang mở khoá)\n" +
			"  · Sau khi mở khoá xong sẽ tự động vào Windows bình thường\n\n" +
			"Nếu khởi động lại vào thẳng Windows mà không qua màn hình mở khoá, hãy vào BIOS (Del/F2):\n" +
			"  1. Tắt Secure Boot\n" +
			"  2. Tắt Fast Boot\n" +
			"  3. Đặt '40HX Unlock' làm mục khởi động đầu tiên\n" +
			"     (Nếu danh sách chỉ có Windows Boot Manager, hãy tắt CSM)\n"
	} else {
		efiNote = "[!] Lần này chưa triển khai EFI mở khoá hiệu năng:\n" +
			"  · Hiệu năng tạm thời giữ nguyên\n" +
			"  · Tác vụ tự mở khoá Gen2 đã đăng ký thành công, không bị ảnh hưởng\n"
	}
	msgbox("Trình Cài Đặt 40HX / 30HX (Cài đặt hoàn tất)",
		"✅ Cài đặt hoàn tất! "+map[bool]string{true: "Sau khi khởi động lại sẽ tự động mở khoá.", false: "Phần Gen2 đã sẵn sàng."}[efiOK]+"\n\n"+
			efiNote+
			"\nSau khi vào lại Windows:\n"+
			"  · Chạy file 40HXCheck.exe để kiểm tra — Nếu báo\n"+
			"    'Mở khoá thành công: Tensor tối đa (SS0=0x88888888)' là hoàn tất\n"+
			"  · Nếu báo chưa mở khoá, công cụ sẽ hướng dẫn bước xử lý (bật Above 4G, v.v.)\n\n"+
			"· GSP đã bật (EnableGpuFirmware=1)\n"+
			"· Đã tắt Khởi động nhanh và Tiết kiệm điện PCIe (ASPM)\n"+
			"· Gen2 sẽ tự động mở khoá khi đăng nhập\n\n"+
			"Nhật ký chi tiết: "+filepath.Join(os.TempDir(), "40HX_installer.log"),
		mbIconInfo)
}

func installDrivers() {
	sysDir := os.Getenv("SystemRoot") + "\\System32\\drivers"
	svcRunning := func(name string) bool {
		out, _ := hxcore.RunOut("sc.exe", "query", name)
		return strings.Contains(out, "RUNNING")
	}
	tsApp := hxcore.ThrottleStopAppRunning()
	for _, d := range []struct{ name, file string }{
		{"ThrottleStop", "ThrottleStop.sys"},
		{"WinRing0_1_2_0", "WinRing0x64.sys"},
	} {
		dst := filepath.Join(sysDir, d.file)
		if tsApp {
			fmt.Printf("  Phát hiện phần mềm ThrottleStop đang chạy, tái sử dụng driver %s (không ghi đè/không xoá)\n", d.name)
			continue
		}
		if svcRunning(d.name) {
			fmt.Printf("  %s đang chạy, bỏ qua ghi đè (giữ nguyên trạng thái)\n", d.name)
			continue
		}
		hxcore.RunOut("sc.exe", "stop", d.name)
		pdDir := filepath.Join(os.Getenv("ProgramData"), "40HXUnlock", "drivers")
		os.MkdirAll(pdDir, 0o755)
		copyEmbedTo(filepath.Join(pdDir, d.file), d.file)
		if err := copyEmbedTo(dst, d.file); err != nil {
			if _, statErr := os.Stat(dst); statErr != nil {
				fmt.Printf("  [!] Sao chép %s thất bại: %v\n", d.file, err)
				continue
			}
		} else {
			fmt.Printf("  Đã sao chép %s\n", d.file)
		}
		ensureService(d.name, d.file)
	}
	fmt.Println("  File driver Gen2 đã sẵn sàng (demand), đăng nhập sẽ được tác vụ SYSTEM nạp và tự dọn dẹp")
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
			start = "demand (ban đầu bị đổi DISABLED, đã sửa lại)"
		case strings.Contains(qcu, "AUTO_START"):
			start = "auto (chú ý: nên là demand)"
		}
	}
	stateS := "?"
	switch {
	case strings.Contains(out, "RUNNING"):
		stateS = "RUNNING"
	case strings.Contains(out, "STOPPED"):
		stateS = "STOPPED"
	}
	fmt.Printf("  Dịch vụ %s đã đăng ký (%s, %s), sau khi đăng nhập sẽ do tác vụ SYSTEM nạp\n", name, start, stateS)
}

func setupBootEntry() error {
	// Bỏ qua nếu mục "40HX Unlock" đã tồn tại
	if out, _ := hxcore.RunOut("bcdedit.exe", "/enum", "firmware"); strings.Contains(out, bootDesc) {
		fmt.Println("    Mục khởi động đã tồn tại, bỏ qua")
		return nil
	}
	// 1. copy {bootmgr} làm mẫu
	out, err := hxcore.RunOut("bcdedit.exe", "/copy", "{bootmgr}", "/d", bootDesc)
	if err != nil {
		return fmt.Errorf("bcdedit copy: %v", err)
	}
	re := regexp.MustCompile(`\{([0-9a-fA-F-]{36})\}`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		return errors.New("Không thể phân tích đầu ra bcdedit: " + out)
	}
	guid := m[1]
	cleanup := func() { hxcore.RunOut("bcdedit.exe", "/delete", "{"+guid+"}", "/f") }

	// 2. Tìm ký tự ổ đĩa ESP (mountvol)
	esp := hxcore.MountESP()
	if esp == "" {
		cleanup()
		return errors.New("Không thể gắn kết phân vùng ESP")
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
	fmt.Printf("    Mục khởi động %s đã được đặt ưu tiên đầu tiên\n", guid)
	return nil
}

func setRunKey() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Println("  [!] Không thể lấy đường dẫn file exe:", err)
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
		fmt.Println("  [!] Ghi khóa Run thất bại:", err)
		return
	}
	defer k.Close()
	if err := k.SetStringValue("40HXGen2", val); err != nil {
		fmt.Println("  [!] Thiết lập khóa Run thất bại:", err)
		return
	}
	fmt.Println("  Đã ghi khóa Run Gen2 (HKCU, nạp tạm thời khi đăng nhập; tác vụ SYSTEM là kênh chính thức): " + abs)
}

func setupGen2Task() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Không thể lấy đường dẫn file exe: %v", err)
	}
	abs, _ := filepath.Abs(exe)
	tn := gen2TaskName
	var lastErr string
	for attempt := 1; attempt <= 3; attempt++ {
		out, cerr := hxcore.RunOut("schtasks.exe", "/create", "/tn", tn,
			"/tr", fmt.Sprintf("\"%s\" -gen2 -silent -guard", abs),
			"/sc", "onlogon", "/ru", "SYSTEM", "/delay", "0000:30", "/f")
		if cerr == nil {
			fmt.Println("  Tác vụ Gen2 đã đăng ký (SYSTEM, độ trễ đăng nhập 30s, chạy ngầm): " + abs)
			return nil
		}
		lastErr = strings.TrimSpace(out)
		if attempt < 3 {
			fmt.Printf("  [!] Đăng ký tác vụ thất bại (lần %d), đang thử lại... (%s)\n", attempt, lastErr)
			time.Sleep(800 * time.Millisecond)
		}
	}
	return fmt.Errorf("Tạo tác vụ tự động Gen2 thất bại (đã thử lại): %s\n      Có thể thủ công: Chạy 40HXInstaller.exe -task với quyền Administrator", lastErr)
}

// ===================== Gen2 解锁 (原生, 无 python) =====================

func gen2Main() {
	// 幂等; -silent(登录自启动调用)时全程无窗口静默
	// v2.5: BYOVD (ThrottleStop + WinRing0): Không cần test-signing, tự động dọn dẹp khi hoàn tất

	// v2.6.0: Đơn phiên duy nhất: Tránh xung đột khi tác vụ SYSTEM, Run key hoặc -gen2 thủ công chạy đồng thời,
	// tránh hai tiến trình cùng sc start driver hoặc tranh chấp BAR0 gây lỗi trạng thái.
	// Đặt ở đầu: Không lấy được mutex thì thoát ngay, tuyệt đối không tải driver.
	owned, release := gen2AcquireSingleInstance()
	if !owned {
		gen2Succeeded = true
		fmt.Println("[Gen2] Một tiến trình Gen2 khác đang chạy, bỏ qua (bảo vệ đơn phiên)")
		_ = hxcore.WriteStructuredGen2Status(hxcore.StatusContract{
			StatusCode: hxcore.StatusGen2Skipped,
			ErrorCode:  "ANOTHER_INSTANCE_RUNNING",
			Details: []string{
				"⏭️ Bỏ qua: Một tiến trình Gen2 khác đang chạy (bảo vệ đơn phiên, tránh xung đột driver)",
			},
		})
		return
	}
	defer release()

	// v2.6.0: Bảo vệ thời gian: Đợi nvlddmkm vào trạng thái RUNNING trước khi can thiệp GPU.
	// Tránh thao tác trước khi driver nv sẵn sàng vì khi driver khởi động có thể reset link PCIe hoặc ghi đè thanh ghi GPU.
	// Tránh xung đột với quá trình khởi động driver (thay thế delay cố định 30s).
	waitForNvDriver(60 * time.Second)

	// Chạy toàn bộ chu trình truy cập phần cứng và huấn luyện PCIe trong DriverSession khép kín (RAII)
	// Tự động giải phóng handle và dọn sạch driver BYOVD khi kết thúc
	err := hxcore.RunScopedBus(true, func(bus hxcore.HardwareBus) error {
		// Định vị GPU được hỗ trợ (40HX/30HX), không cố định BDF
		var gpuBDF uint32
		var gpuProfile hxcore.GPUProfile
		gpuFound := false
		for attempt := 1; attempt <= 3; attempt++ {
			gpuBDF, gpuProfile, gpuFound = hxcore.FindGPUPCIWithBus(bus)
			if gpuFound {
				break
			}
			if attempt < 3 {
				fmt.Printf("[Gen2] Chưa định vị được GPU được hỗ trợ, thử lại sau 2s (%d/3)...\n", attempt)
				time.Sleep(2 * time.Second)
			}
		}
		if !gpuFound {
			fmt.Println("[Gen2] Không thể định vị GPU được hỗ trợ (40HX/30HX). Vui lòng gửi file log.")
			gen2StatusFail("Không tìm thấy GPU được hỗ trợ trên bus PCI")
			gen2Notify("Không tìm thấy GPU được hỗ trợ trên bus PCI.\nVui lòng kiểm tra lại card và driver.")
			return nil
		}

		if gpuProfile.HasSafePL0 {
			ensureGspSilent()
		}

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
		cur := bus.LinkSpeed(gpuBDF)
		fmt.Printf("[Gen%d] Băng thông hiện tại: Gen%d\n", targetGen, cur)
		// v2.6.0: Ghi nhận thanh ghi PCIe gốc (LNKCAP/LNKCTL/LNKCTL2) để chẩn đoán
		if cap := bus.PcieCap(gpuBDF); cap != 0 {
			rd := func(off uint32) uint32 {
				v, _ := bus.ReadPCIConfig(gpuBDF, cap+off)
				return v
			}
			fmt.Printf("[Gen%d] LNKCAP=0x%08X LNKCTL=0x%08X LNKCTL2=0x%08X (Mục tiêu Gen%d)\n",
				targetGen, rd(0x0C), rd(0x10), rd(0x30), rd(0x30)&0xF)
		}
		if cur >= targetGen {
			gen2Succeeded = true
			fmt.Printf("[Gen%d] Đã đạt Gen%d, không cần thao tác thêm.\n", targetGen, cur)
			stContract := hxcore.StatusContract{
				StatusCode:   hxcore.StatusGen2Success,
				SpeedCurrent: cur,
				WidthCurrent: bus.LinkWidth(gpuBDF),
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
			return nil
		}

		// PCIe Capability gate (Phía GPU + Phía Root Port)
		root := bus.FindRootPort(gpuBus)
		if root == 0xFFFFFFFF {
			fmt.Printf("[Gen%d] Không tìm thấy root port, dùng GPU retrain dự phòng\n", targetGen)
		} else {
			fmt.Printf("[Gen%d] root port = 00:%02x.%x\n", targetGen, (root>>3)&0x1F, root&7)
		}
		gpuMax := bus.PcieMaxSpeed(gpuBDF)
		rootMax := uint32(0)
		if root != 0xFFFFFFFF {
			rootMax = bus.PcieMaxSpeed(root)
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
					return nil
				}
			} else {
				width := bus.LinkWidth(gpuBDF)
				tls := uint32(0)
				if cap := bus.PcieCap(gpuBDF); cap != 0 {
					if v, err := bus.ReadPCIConfig(gpuBDF, cap+0x30); err == nil {
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
				return nil
			}
		}

		negotiator := hxcore.NewLinkNegotiator(bus)
		allowStage2 := hasArg("-hard") || (gpuProfile.DeviceID != 0x2189 && gen2AutoHardEnabled())

		fmt.Printf("[Gen%d] Khởi chạy LinkNegotiator cho %s (DEV_%04X)...\n", targetGen, gpuProfile.Name, gpuProfile.DeviceID)
		res, err := negotiator.Negotiate(gpuBDF, gpuProfile, root, targetGen, allowStage2)
		if err != nil {
			fmt.Printf("[Gen%d][!] Lỗi thương lượng link: %v\n", targetGen, err)
			gen2StatusFail(fmt.Sprintf("Lỗi thương lượng link: %v", err))
			return nil
		}

		gen2Succeeded = res.Success
		if res.Success {
			deleteGen2Retry()
		} else if hxcore.DriverStrategy() != hxcore.DriverStrategyResident {
			scheduleGen2Retry(retryDepth())
		}

		fmt.Printf("[Gen%d] %s\n", res.TargetGen, res.Verdict)

		statusCode := hxcore.StatusGen1Stuck
		if res.Success {
			statusCode = hxcore.StatusGen2Success
		}
		details := []string{
			fmt.Sprintf("Kết luận: %s", res.Verdict),
			fmt.Sprintf("Quyền thực thi: %s", map[bool]string{true: "Quản trị viên (Admin)/SYSTEM", false: "Người dùng thường (Bị hạn chế)"}[isAdmin()]),
			fmt.Sprintf("%s Vị trí: %02x:%02x.%x", gpuProfile.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7),
			fmt.Sprintf("Root Port: %02x:%02x.%x", (root>>8)&0xFF, (root>>3)&0x1F, root&7),
			fmt.Sprintf("Băng thông: Hiện tại Gen%d x%d / GPU TLS=Gen%d / Root TLS=Gen%d", res.CurrentSpeed, res.CurrentWidth, res.TargetTLS, res.RootTLS),
		}
		if gpuProfile.DeviceID == 0x2189 {
			details = append(details, "MRRS: 512B [Đã tối ưu]")
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
			fmt.Printf("[Gen%d] Ghi file trạng thái thất bại: %v\n", res.TargetGen, err)
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
			msgbox(fmt.Sprintf("%s Gen%d", gpuProfile.Name, res.TargetGen), txt, icon)
		}
		return nil
	})

	if err != nil {
		if !isAdmin() {
			fmt.Println("[Gen2] Lỗi tải driver và hiện tại không có quyền Admin: Chuyển cho SYSTEM task, thoát im lặng:", err)
			gen2StatusFail("Driver chưa được tải, hiện tại quyền hạn bị giới hạn (do task SYSTEM xử lý)")
			return
		}
		fmt.Println("[Gen2] Lỗi khởi động driver:", err)
		gen2StatusFail("Driver không chạy được: " + err.Error())
		gen2Notify("Lỗi khởi động driver kernel.\nNguyên nhân: Antivirus chặn WinRing0/ThrottleStop hoặc xung đột phần mềm.\nVui lòng chạy với quyền Administrator.")
		return
	}
}

// probe30HX: Chẩn đoán chỉ đọc thanh ghi BAR0 MMIO link và PHY CMP 30HX (TU116)
func probe30HX() {
	fmt.Println("=== Chẩn đoán chỉ đọc thanh ghi BAR0 MMIO CMP 30HX (TU116) ===")
	err := hxcore.RunScopedBus(true, func(bus hxcore.HardwareBus) error {
		gpuBDF, gpuProfile, gpuFound := hxcore.FindGPUPCIWithBus(bus)
		if !gpuFound {
			fmt.Println("[!] Không định vị được card đồ hoạ hỗ trợ (30HX/40HX) trên bus PCI")
			return nil
		}
		gpuBus := (gpuBDF >> 8) & 0xFF
		fmt.Printf("[Probe] Card đồ hoạ: %s (DEV_%04X) tại %02x:%02x.%x\n", gpuProfile.Name, gpuProfile.DeviceID, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7)

		bar0raw, err := bus.ReadPCIConfig(gpuBDF, 0x10)
		if err != nil || bar0raw == 0 || bar0raw == 0xFFFFFFFF {
			fmt.Printf("[!] Đọc BAR0 bất thường: 0x%08X (err=%v)\n", bar0raw, err)
			return nil
		}
		bar0Phys := uint64(bar0raw & 0xFFFFFFF0)
		fmt.Printf("[Probe] PCI BAR0 (0x10) = 0x%08X (Địa chỉ vật lý gốc: 0x%08X)\n", bar0raw, bar0Phys)

		boot0, berr := bus.ReadMMIO(bar0Phys + 0x0)
		if berr != nil {
			fmt.Printf("[!] Đọc BOOT_0 thất bại: %v\n", berr)
			return nil
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
			val, rerr := bus.ReadMMIO(bar0Phys + r.off)
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
		return nil
	})
	if err != nil {
		fmt.Printf("[!] Lỗi khởi tạo phiên driver chẩn đoán: %v\n", err)
	}
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
