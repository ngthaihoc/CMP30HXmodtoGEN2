package main

// v2.6.0: 精简单界面 GUI (walk) — 无选项卡、无状态表格, 安装器只管安装。
// 打开时自动只读扫描一次(不展示明细): 结果只用于
//   ① 组件安装区的"环境提示"行(未检测到卡/SecureBoot/Legacy 等警告)
//   ② 缺失/未达标组件的自动预勾(已装不勾 = 不覆盖)
// 界面自上而下:
//   ① 组件安装 — [安装所选组件] / [一键完整安装(全流程)], 装完自动重扫更新提示
//   ② Gen2 策略 — 驱动运行策略 + 自动 Stage2 回退 + 失败重试次数/间隔,[保存策略]
//   ③ 操作日志 — AttachLogSink 实时输出(GUI 与 CLI 共用全部实现)
// 卸载与详细诊断不在本界面: 40HXUninstaller.exe / -uninstall / 40HXCheck.exe。
// 线程约定: OnClicked(UI 线程)只读控件 → goroutine 执行 → UI 变更一律经 sync()。

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"40hxcore"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows/registry"
)

// ---- 环境扫描(只读; 结果喂 tip 与预勾选, 不展示明细表) ----

type statusItem struct {
	name string
	ok   bool
	note string
}

func scanStatus() []statusItem {
	items := []statusItem{}
	legacy := hxcore.FirmwareIsLegacy()
	items = append(items, statusItem{"Chế độ Boot", !legacy,
		map[bool]string{true: "UEFI (OK)", false: "Legacy+MBR — Cần chuyển sang GPT bằng mbr2gpt"}[legacy]})
	sbOn := hxcore.SecureBootOn()
	items = append(items, statusItem{"Secure Boot", !sbOn,
		map[bool]string{true: "Đang Bật (Cần Tắt trong BIOS!)", false: "Đã Tắt (OK)"}[sbOn]})
	gpuOK := hxcore.FindGPU()
	items = append(items, statusItem{"Card đồ hoạ", gpuOK,
		map[bool]string{true: "Đã phát hiện GPU tương thích", false: "Chưa phát hiện — Kiểm tra khe cắm & driver"}[gpuOK]})
	gspOK := hxcore.GspEnabled()
	items = append(items, statusItem{"GSP (EnableGpuFirmware)", gspOK,
		map[bool]string{true: "Đã bật (OK)", false: "Chưa bật — Có thể bị Code 43 sau mở khoá"}[gspOK]})

	espEFI := false
	if esp := hxcore.MountESP(); esp != "" {
		if _, err := os.Stat(esp + ":\\EFI\\40HX\\40HXUNLK.EFI"); err == nil {
			espEFI = true
		}
		hxcore.UnmountESP(esp)
	}
	items = append(items, statusItem{"ESP EFI Mở khoá", espEFI,
		map[bool]string{true: "\\EFI\\40HX\\40HXUNLK.EFI đã nạp", false: "Chưa nạp (Legacy không hỗ trợ)"}[espEFI]})
	// 启动项三态: 首位 / 存在但不在首位 / 未创建
	bootOK := false
	bootNote := "Chưa tạo"
	if ex, first, ord := verifyBootEntry(); ex {
		if first {
			bootOK = true
			bootNote = "Tồn tại và nằm đầu tiên (displayorder)"
		} else {
			bootNote = "Tồn tại nhưng chưa đặt đầu tiên (thứ tự: " + ord + ") — Cần vào BIOS chỉnh"
		}
	}
	items = append(items, statusItem{"Mục khởi động BIOS", bootOK, bootNote})

	taskOK, taskStatus, taskResult := hxcore.TaskInfo(gen2TaskName)
	taskNote := "Chưa đăng ký — Sẽ không tự mở khoá khi bật máy"
	if taskOK {
		taskNote = "Trạng thái: " + taskStatus + ", Kết quả gần nhất: " + taskResult
	}
	items = append(items, statusItem{"Tác vụ tự khởi động", taskOK, taskNote})
	rkOK := runKeyPresent()
	items = append(items, statusItem{"Khoá Run dự phòng Registry", rkOK,
		map[bool]string{true: "Đã ghi (40HXGen2)", false: "Chưa ghi"}[rkOK]})

	deps := hxcore.InspectGen2Drivers()
	if !hxcore.Gen2DriversDeployedOnce() {
		items = append(items, statusItem{"Driver PCIe (Chưa cài)", false,
			"Chưa có driver — Hãy tích chọn [Cài đặt Driver PCIe] bên dưới"})
	} else {
		var notes []string
		curOK := true
		cleanEnd := true
		for _, d := range deps {
			svcS := "Dịch vụ chưa đăng ký"
			if d.SvcReg {
				svcS = "Dịch vụ: " + d.SvcStart
				if d.SvcStart == "DISABLED" {
					svcS += " ⚠Bị vô hiệu hoá (cài lại để sửa)"
				}
				if d.SvcRunning {
					svcS += "/Đang chạy"
				}
			}
			notes = append(notes, d.File+": System32="+d.SysState.String()+", "+svcS)
			if d.SysState != hxcore.DrvOk || !d.SvcReg || d.SvcStart == "DISABLED" {
				curOK = false
			}
			if d.SvcReg || d.SysState != hxcore.DrvAbsent {
				cleanEnd = false
			}
		}
		if cleanEnd && hxcore.DriverStrategy() != hxcore.DriverStrategyResident {
			items = append(items, statusItem{"Driver PCIe (Đã dọn dẹp sạch)", true,
				"Đã cài; Tự dọn dẹp sau khi chạy (bình thường, lần đăng nhập sau sẽ tự nạp lại)"})
		} else {
			items = append(items, statusItem{"Trạng thái Driver PCIe", curOK, strings.Join(notes, " | ")})
		}
	}
	if exOK, err := hxcore.DefenderExclusionsPresent(); err != nil {
		items = append(items, statusItem{"Loại trừ Windows Defender", false,
			"Kiểm tra thất bại (" + err.Error() + ") — Chạy lại với quyền Admin"})
	} else {
		items = append(items, statusItem{"Loại trừ Windows Defender", exOK,
			map[bool]string{true: "Đã thêm loại trừ cho 2 file .sys và thư mục ProgramData", false: "Chưa thêm loại trừ — Antivirus có thể chặn driver"}[exOK]})
	}

	fsOn := hxcore.FastStartupOn()
	items = append(items, statusItem{"Khởi động nhanh (Fast Startup)", !fsOn,
		map[bool]string{true: "Đang Bật (Khuyên tắt để nạp UEFI chuẩn)", false: "Đã Tắt (OK)"}[fsOn]})
	if ac, dc, aspmOK := hxcore.ASPMSavings(); aspmOK {
		off := ac == 0 && dc == 0
		items = append(items, statusItem{"Tiết kiệm điện PCIe (ASPM)", off,
			map[bool]string{true: "Đã Tắt (OK)", false: fmt.Sprintf("Đang Bật (AC=%d DC=%d) — Có thể bị tụt tốc độ khi nghỉ", ac, dc)}[off]})
	}
	return items
}

func runKeyPresent() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue("40HXGen2")
	return err == nil
}

// ---- 日志面板写入器 (AttachLogSink 目标) ----

type guiLog struct {
	mw *walk.MainWindow
	te *walk.TextEdit
}

func (g *guiLog) Write(p []byte) (int, error) {
	s := string(p)
	if g.mw != nil && g.te != nil {
		// EDIT 控件换行需要 CRLF: 统一把 \n 规范成 \r\n, 否则日志会挤成一段
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		s = strings.ReplaceAll(s, "\n", "\r\n")
		g.mw.Synchronize(func() { g.te.AppendText(s) })
	}
	return len(p), nil
}

// ---- GUI 状态 ----

type guiState struct {
	mw       *walk.MainWindow
	log      *guiLog
	teLog    *walk.TextEdit
	tip      *walk.Label
	busyBy   string // 当前占用互斥的操作名(""=空闲)。只在 UI 线程读写:
	// begin() 在 OnClicked(UI线程) 调用, end() 经 sync 回到 UI 线程 → 无线程竞争。
	lastScan  []statusItem
	lastGuide string // 上次打印的环境指引(变化才打印, 防重扫刷屏)
	lastAV      string // 上次识别到的第三方杀软(同上)
	lastDefWarn string // "无 Defender 模块"提示去重

	ckGsp, ckDrv, ckEfi, ckTask            *walk.CheckBox
	ckFast, ckAspm, ckPerf, ckDefOff       *walk.CheckBox
	pbInstall, pbFull                      *walk.PushButton

	rbStrategy     [3]*walk.RadioButton
	ckAutoHard     *walk.CheckBox
	neRetryCnt     *walk.NumberEdit
	neRetryMin     *walk.NumberEdit
	pbSave, pbGen2, pbGen2Install *walk.PushButton
}

func (st *guiState) sync(f func()) {
	if st.mw != nil {
		st.mw.Synchronize(f)
	} else {
		f()
	}
}

// begin: 在 UI 线程(OnClicked)同步抢全局互斥 — 一次只允许一个长操作在跑。
// 抢到即占住(busyBy)并同步禁用按钮, 杜绝"连点/快速双击"在 goroutine 里抢锁的竞态。
func (st *guiState) begin(what string) bool {
	if st.busyBy != "" {
		fmt.Println("[!] Đang thực hiện " + st.busyBy + " — " + what + " đã bỏ qua, vui lòng đợi hoàn tất rồi thử lại")
		return false
	}
	st.busyBy = what
	return true
}

func (st *guiState) end() {
	st.sync(func() { st.busyBy = "" })
}

// setActionsEnabled: 初始自动扫描期间禁用执行按钮, 防止与手动操作并发抢 IO。
func (st *guiState) setActionsEnabled(on bool) {
	st.sync(func() {
		for _, b := range []*walk.PushButton{st.pbInstall, st.pbFull, st.pbSave, st.pbGen2} {
			if b != nil {
				b.SetEnabled(on)
			}
		}
	})
}

// summaryText: 把"不处理就装不上/装了也不生效"的最关键状态合成安装区顶部提示。
func (st *guiState) summaryText(items []statusItem) string {
	m := map[string]statusItem{}
	for _, it := range items {
		m[it.name] = it
	}
	if it, ok := m["Card đồ hoạ"]; ok && !it.ok {
		return "⚠ Chưa phát hiện GPU hỗ trợ — Kiểm tra khe cắm & driver trước khi cài đặt"
	}
	var warns []string
	if it, ok := m["Secure Boot"]; ok && !it.ok {
		warns = append(warns, "Secure Boot đang bật (cần vào BIOS tắt)")
	}
	if it, ok := m["Chế độ Boot"]; ok && !it.ok {
		warns = append(warns, "Khởi động Legacy+MBR (cần dùng mbr2gpt chuyển sang GPT)")
	}
	if it, ok := m["Driver PCIe (Chưa cài)"]; ok && !it.ok {
		warns = append(warns, "Driver PCIe chưa được cài đặt")
	}
	if it, ok := m["Tác vụ tự khởi động"]; ok && !it.ok {
		warns = append(warns, "Tác vụ tự khởi động PCIe chưa đăng ký")
	}
	// 启动项: 仅 UEFI 下检查
	if it, ok := m["Mục khởi động BIOS"]; ok && !it.ok {
		if bl, ok2 := m["Chế độ Boot"]; !ok2 || bl.ok {
			if strings.Contains(it.note, "chưa đặt") {
				warns = append(warns, "Mục khởi động tồn tại nhưng chưa đặt đầu tiên")
			} else {
				warns = append(warns, "Chưa tạo mục khởi động UEFI")
			}
		}
	}
	if len(warns) == 0 {
		return "✓ Môi trường sẵn sàng — Đã tự chọn mục cần thiết, bấm [Cài đặt mục đã chọn] hoặc [Cài đặt toàn bộ]"
	}
	s := "⚠ " + strings.Join(warns, "; ")
	if r := []rune(s); len(r) > 90 {
		s = string(r[:90]) + "…"
	}
	return s
}

// envGuide: "Known issues -> Resolution steps"
func (st *guiState) envGuide(items []statusItem) string {
	m := map[string]statusItem{}
	for _, it := range items {
		m[it.name] = it
	}
	var g []string
	if it, ok := m["Card đồ hoạ"]; ok && !it.ok {
		g = append(g, "· Chưa phát hiện GPU: ① Kiểm tra nguồn phụ & cắm chắc khe PCIe; ② Xem Device Manager có Code 43 không; ③ Tắt CSM trong BIOS (chọn UEFI thuần)")
	}
	if it, ok := m["Secure Boot"]; ok && !it.ok {
		g = append(g, "· Secure Boot đang bật: Khởi động lại bấm Del/F2 vào BIOS → Security/Boot → Secure Boot=Disabled → F10 lưu và khởi động lại")
	}
	if it, ok := m["Chế độ Boot"]; ok && !it.ok {
		g = append(g, "· Khởi động Legacy+MBR: Không có phân vùng EFI → Mở CMD Admin chạy: mbr2gpt /validate /allowfullos → mbr2gpt /convert /allowfullos → Vào BIOS tắt CSM")
	}
	if it, ok := m["Mục khởi động BIOS"]; ok && !it.ok {
		if bl, ok2 := m["Chế độ Boot"]; !ok2 || bl.ok {
			if strings.Contains(it.note, "chưa đặt") {
				g = append(g, "· Mục khởi động chưa ưu tiên: Vào BIOS đặt '40HX Unlock' lên vị trí đầu tiên (Boot Option #1)")
			} else {
				g = append(g, "· Chưa tạo mục khởi động: Tích chọn [EFI Mở khoá + Mục khởi động BIOS] để tạo tự động")
			}
		}
	}
	if it, ok := m["Driver PCIe (Chưa cài)"]; ok && !it.ok {
		g = append(g, "· Driver PCIe chưa cài: Tích chọn [Cài đặt Driver PCIe + Thêm loại trừ Defender] để cài đặt")
	}
	if it, ok := m["Tác vụ tự khởi động"]; ok && !it.ok {
		g = append(g, "· Tác vụ tự mở khoá chưa đăng ký: Tích chọn [Tự khởi động mở khoá PCIe khi đăng nhập]")
	}
	if it, ok := m["ESP EFI Mở khoá"]; ok && !it.ok {
		if bl, ok2 := m["Chế độ Boot"]; !ok2 || bl.ok {
			g = append(g, "· EFI mở khoá chưa nạp: Tích chọn [EFI Mở khoá + Mục khởi động BIOS] (chỉ dành cho CMP 40HX)")
		}
	}
	return strings.Join(g, "\n")
}

// scanOnce: 只读扫描一次并刷新顶部提示(不展示明细)。
func (st *guiState) scanOnce() {
	items := scanStatus()
	st.lastScan = items
	tip := st.summaryText(items)
	st.sync(func() {
		if st.tip != nil {
			st.tip.SetText(tip)
		}
	})
	av := hxcore.DetectThirdPartyAV()
	avKey := strings.Join(av, ",")
	if avKey != st.lastAV {
		if len(av) > 0 {
			fmt.Println("[Antivirus] Phát hiện phần mềm bảo mật bên thứ ba: " + strings.Join(av, " / ") +
				" — Vui lòng thêm 4 đường dẫn driver vào danh sách tin cậy/loại trừ:")
			fmt.Println("        C:\\Windows\\System32\\drivers\\ThrottleStop.sys")
			fmt.Println("        C:\\Windows\\System32\\drivers\\WinRing0x64.sys")
			fmt.Println("        %ProgramData%\\40HXUnlock\\drivers\\ThrottleStop.sys và WinRing0x64.sys")
		}
		st.lastAV = avKey
	}
}

// applySmartDefaults: 按最近一次扫描预勾选 — 组件缺失/未达标才勾(已装不勾=不覆盖)。
func (st *guiState) applySmartDefaults() {
	items := st.lastScan
	if len(items) == 0 {
		items = scanStatus()
		st.lastScan = items
	}
	flags := map[string]bool{}
	for _, it := range items {
		flags[it.name] = it.ok
	}
	if !flags["Card đồ hoạ"] {
		fmt.Println("[i] Chưa phát hiện card đồ hoạ hỗ trợ — Giữ nguyên không chọn mục nào (vui lòng kiểm tra GPU/driver)")
		st.sync(func() {
			st.ckGsp.SetChecked(false)
			st.ckDrv.SetChecked(false)
			st.ckEfi.SetChecked(false)
			st.ckTask.SetChecked(false)
			st.ckFast.SetChecked(false)
			st.ckAspm.SetChecked(false)
			st.ckPerf.SetChecked(false)
			st.ckDefOff.SetChecked(false)
		})
		return
	}
	aspmOK := true
	if v, present := flags["Tiết kiệm điện PCIe (ASPM)"]; present {
		aspmOK = v
	}
	needGsp := !flags["GSP (EnableGpuFirmware)"]
	needDrv := hxcore.Gen2DriversNeedDeploy()
	needEfi := false
	if flags["Chế độ Boot"] {
		needEfi = !flags["ESP EFI Mở khoá"] || !flags["Mục khởi động BIOS"]
	} else {
		fmt.Println("[i] Khởi động Legacy+MBR: Không thể cài EFI mở khoá — Bỏ chọn (cần mbr2gpt trước)")
	}
	needTask := !flags["Tác vụ tự khởi động"]
	needFast := !flags["Khởi động nhanh (Fast Startup)"]
	needAspm := !aspmOK
	needPerf := !hxcore.HighPerfPlanActive()
	needDefOff, defKnown := false, false
	if on, err := hxcore.DefenderRealtimeProtectionOn(); err == nil {
		defKnown = true
		needDefOff = on
	} else {
		defWarn := "Máy tính không có module quản trị Defender (phần mềm diệt virus bên thứ 3 vui lòng tự thêm whitelist)"
		if !errors.Is(err, hxcore.ErrMpUnavailable) {
			defWarn = "Kiểm tra trạng thái Defender thất bại: " + err.Error()
		}
		if defWarn != st.lastDefWarn {
			fmt.Println("[i] " + defWarn)
			st.lastDefWarn = defWarn
		}
	}
	var pre []string
	st.sync(func() {
		st.ckGsp.SetChecked(needGsp)
		if needGsp {
			pre = append(pre, "GSP")
		}
		st.ckDrv.SetChecked(needDrv)
		if needDrv {
			pre = append(pre, "Driver PCIe")
		}
		st.ckEfi.SetChecked(needEfi)
		if needEfi {
			pre = append(pre, "EFI Mở khoá + Khởi động BIOS")
		}
		st.ckTask.SetChecked(needTask)
		if needTask {
			pre = append(pre, "Tự mở khoá khi đăng nhập")
		}
		st.ckFast.SetChecked(needFast)
		if needFast {
			pre = append(pre, "Tắt Fast Startup")
		}
		st.ckAspm.SetChecked(needAspm)
		if needAspm {
			pre = append(pre, "Tắt ASPM")
		}
		st.ckPerf.SetChecked(needPerf)
		if needPerf {
			pre = append(pre, "Hiệu năng cao High Performance")
		}
		wantDefOff := defKnown && needDefOff
		st.ckDefOff.SetChecked(wantDefOff)
		if wantDefOff {
			pre = append(pre, "Tắt Defender Realtime")
		}
	})
	if len(pre) > 0 {
		fmt.Println("[i] Tự động chọn: " + strings.Join(pre, " / ") + " → Bấm [Cài đặt mục đã chọn] để thực thi; Các mục đã sẵn sàng được bỏ qua")
		return
	}
	var ready []string
	if !needGsp {
		ready = append(ready, "GSP đã bật")
	}
	if !needDrv {
		ready = append(ready, "Driver PCIe đã sẵn sàng")
	}
	if !flags["Chế độ Boot"] {
		ready = append(ready, "EFI Mở khoá (cần mbr2gpt trước)")
	} else if !needEfi {
		ready = append(ready, "EFI + Khởi động BIOS đã sẵn sàng")
	}
	if !needTask {
		ready = append(ready, "Tự khởi động đã đăng ký")
	}
	if !needFast {
		ready = append(ready, "Khởi động nhanh đã tắt")
	}
	if !needAspm {
		ready = append(ready, "ASPM đã tắt")
	}
	if !needPerf {
		ready = append(ready, "Đã ở chế độ High Performance")
	}
	if defKnown && !needDefOff {
		ready = append(ready, "Defender Realtime đã tắt")
	}
	fmt.Println("[i] Tất cả đã sẵn sàng, không cần chọn thêm: " + strings.Join(ready, " | "))
}

// printDefErr: Hiển thị lỗi Defender ngắn gọn.
func (st *guiState) printDefErr(prefix string, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, hxcore.ErrMpUnavailable) {
		fmt.Println(prefix + "Hệ thống không có module quản trị Defender — Thêm loại trừ / tắt bảo vệ tự động không khả dụng")
		fmt.Println(prefix + "Nếu dùng phần mềm diệt virus bên thứ ba, vui lòng thêm ngoại lệ cho:")
		fmt.Println("      C:\\Windows\\System32\\drivers\\ThrottleStop.sys / WinRing0x64.sys")
		fmt.Println("      %ProgramData%\\40HXUnlock\\drivers\\ (2 file .sys cùng tên)")
		return
	}
	fmt.Println(prefix + err.Error())
}

// loadPolicyUI: Khởi động đọc lại chính sách (gọi trên luồng UI — sau Create, trước Run)
func (st *guiState) loadPolicyUI() {
	strat := hxcore.DriverStrategy()
	for i, rb := range st.rbStrategy {
		rb.SetChecked(i == strat)
	}
	st.ckAutoHard.SetChecked(hxcore.ConfigInt("Gen2AutoHard", 1) != 0)
	cnt, interval := hxcore.Gen2RetryPolicy()
	st.neRetryCnt.SetValue(float64(cnt))
	st.neRetryMin.SetValue(float64(interval))
}

// savePolicy: Lưu cấu hình PCIe (HKLM\SOFTWARE\40HXUnlock, đọc bởi tác vụ tự chạy và lệnh -gen2)
func (st *guiState) savePolicy() {
	defer st.end()
	strat := 0
	for i, rb := range st.rbStrategy {
		if rb.Checked() {
			strat = i
		}
	}
	if err := hxcore.SetConfigInt("DriverStrategy", strat); err != nil {
		fmt.Println("[Cấu hình] Lưu thất bại:", err)
		return
	}
	auto := 0
	if st.ckAutoHard.Checked() {
		auto = 1
	}
	cnt, interval := int(st.neRetryCnt.Value()), int(st.neRetryMin.Value())
	hxcore.SetConfigInt("Gen2AutoHard", auto)
	hxcore.SetConfigInt("Gen2RetryCount", cnt)
	hxcore.SetConfigInt("Gen2RetryIntervalMin", interval)
	fmt.Printf("[Cấu hình] Đã lưu: Chiến lược driver=%d Gen2AutoHard=%d Thử lại=%d lần / Giãn cách=%d phút\n", strat, auto, cnt, interval)
	if strat == hxcore.DriverStrategyResident {
		if ok, _, _ := hxcore.TaskInfo(gen2TaskName); !ok {
			fmt.Println("[Gợi ý] Chế độ Thường trú cần tác vụ tự chạy khi đăng nhập: Vui lòng tích chọn [Tự động mở khoá PCIe khi đăng nhập Windows] ở mục ① rồi bấm [Cài đặt mục đã chọn], hoặc bấm nút [Mở khoá ngay & Cài tự khởi động]")
		} else {
			fmt.Println("[Gợi ý] Đã chọn Thường trú: Vui lòng bấm [Mở khoá ngay & Cài tự khởi động] để cập nhật tham số canh giữ (-guard) cho tác vụ")
		}
	}
}

func runGUI() {
	if !isAdmin() {
		selfElevate()
		return
	}
	st := &guiState{log: &guiLog{}}

	createErr := MainWindow{
		AssignTo: &st.mw,
		Title:    "Trình Mở Khoá CMP 40HX / 30HX v3.0.0",
		MinSize:  Size{Width: 800, Height: 680},
		Size:     Size{Width: 880, Height: 800},
		Layout:   VBox{Spacing: 6},
		Children: []Widget{
			GroupBox{
				Title:  "① Cài đặt thành phần & Môi trường (Tự động chọn theo máy; Tích chọn = Cài đặt/Làm mới)",
				Layout: VBox{Spacing: 4},
				Children: []Widget{
					Label{AssignTo: &st.tip, Text: "Đang kiểm tra môi trường hệ thống…"},
					Composite{
						Layout: Grid{Columns: 2},
						Children: []Widget{
							CheckBox{AssignTo: &st.ckGsp, Text: "Bật GSP (EnableGpuFirmware=1)"},
							CheckBox{AssignTo: &st.ckEfi, Text: "EFI Mở khoá + Mục khởi động BIOS (Chỉ 40HX)"},
							CheckBox{AssignTo: &st.ckDrv, Text: "Cài đặt Driver PCIe + Thêm loại trừ Defender"},
							CheckBox{AssignTo: &st.ckTask, Text: "Tự động mở khoá PCIe khi đăng nhập Windows"},
							CheckBox{AssignTo: &st.ckFast, Text: "Nguồn: Tắt Fast Startup (Khởi động nhanh)"},
							CheckBox{AssignTo: &st.ckAspm, Text: "Nguồn: Tắt ASPM (Tiết kiệm điện PCIe)"},
							CheckBox{AssignTo: &st.ckPerf, Text: "Nguồn: Bật chế độ High Performance"},
							CheckBox{AssignTo: &st.ckDefOff, Text: "Tắt bảo vệ thời gian thực Defender"},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							PushButton{AssignTo: &st.pbInstall, Text: "Cài đặt mục đã chọn", OnClicked: func() {
								if !st.begin("Cài đặt thành phần") {
									return
								}
								sel := map[string]bool{
									"gsp":    st.ckGsp.Checked(),
									"drv":    st.ckDrv.Checked(),
									"efi":    st.ckEfi.Checked(),
									"task":   st.ckTask.Checked(),
									"fast":   st.ckFast.Checked(),
									"aspm":   st.ckAspm.Checked(),
									"perf":   st.ckPerf.Checked(),
									"defoff": st.ckDefOff.Checked(),
								}
								go st.installSelected(sel)
							}},
							PushButton{AssignTo: &st.pbFull, Text: "Cài đặt toàn bộ (Một chạm)", OnClicked: func() {
								if !st.begin("Cài đặt toàn bộ") {
									return
								}
								go func() {
									defer st.end()
									st.sync(func() { st.pbFull.SetEnabled(false) })
									defer st.sync(func() { st.pbFull.SetEnabled(true) })
									install()
									st.scanOnce()
									st.applySmartDefaults()
								}()
							}},
						},
					},
				},
			},
			GroupBox{
				Title:  "② Cấu hình PCIe (Lưu có hiệu lực ngay; Tác vụ tự chạy sẽ áp dụng cấu hình này)",
				Layout: VBox{Spacing: 4},
				Children: []Widget{
					Label{Text: "Chính sách Driver: Cách xử lý driver can thiệp sau khi mở khoá PCIe"},
					Label{Text: "① Dùng xong gỡ ngay (Mặc định, sạch sẽ, chuẩn game/anti-cheat)   ② Thử lại nếu lỗi   ③ Thường trú (Giữ driver, canh và giữ tốc độ)"},
					Composite{
						Layout: Grid{Columns: 3},
						Children: []Widget{
							RadioButton{AssignTo: &st.rbStrategy[0], Text: "Dùng xong gỡ ngay (Khuyên dùng)"},
							RadioButton{AssignTo: &st.rbStrategy[1], Text: "Tự động thử lại khi lỗi"},
							RadioButton{AssignTo: &st.rbStrategy[2], Text: "Thường trú (Canh giữ PCIe)"},
						},
					},
					CheckBox{AssignTo: &st.ckAutoHard, Text: "Tự động kích hoạt Stage 2 nếu chưa đạt (Link Disable + Khôi phục PnP)"},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							Label{Text: "Thử lại khi lỗi:"},
							NumberEdit{AssignTo: &st.neRetryCnt, MinValue: 0.0, MaxValue: 12.0, MinSize: Size{Width: 56}},
							Label{Text: "lần / Giãn cách:"},
							NumberEdit{AssignTo: &st.neRetryMin, MinValue: 1.0, MaxValue: 240.0, MinSize: Size{Width: 56}},
							Label{Text: "phút"},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							PushButton{AssignTo: &st.pbSave, Text: "Lưu cấu hình", OnClicked: func() {
								if !st.begin("Lưu cấu hình") {
									return
								}
								go st.savePolicy()
							}},
							PushButton{AssignTo: &st.pbGen2, Text: "Mở khoá PCIe ngay (Phiên này)", OnClicked: func() {
								if !st.begin("Mở khoá PCIe ngay") {
									return
								}
								go func() {
									defer st.end()
									st.sync(func() { st.pbGen2.SetEnabled(false) })
									defer st.sync(func() { st.pbGen2.SetEnabled(true) })
									fmt.Println("[PCIe] Mở khoá ngay một lần (tương đương lệnh -gen2)...")
									gen2Main()
									st.scanOnce()
								}()
							}},
							PushButton{AssignTo: &st.pbGen2Install, Text: "Mở khoá ngay & Cài tự khởi động", OnClicked: func() {
								if !st.begin("Mở khoá & Cài tự khởi động") {
									return
								}
								go func() {
									defer st.end()
									st.sync(func() { st.pbGen2Install.SetEnabled(false) })
									defer st.sync(func() { st.pbGen2Install.SetEnabled(true) })
									fmt.Println("== Mở khoá PCIe và cài đặt tự khởi động ==")
									fmt.Println("[PCIe] Bước 1/2: Mở khoá phiên hiện tại...")
									gen2Main()
									fmt.Println("[PCIe] Bước 2/2: Cài đặt tự khởi động khi đăng nhập Windows...")
									installDrivers()
									if err := hxcore.AddDefenderExclusions(); err != nil {
										st.printDefErr("  [Defender] ", err)
									} else {
										fmt.Println("  [Defender] Đã thêm loại trừ cho file driver và ProgramData")
									}
									setRunKey()
									if err := setupGen2Task(); err != nil {
										fmt.Println("  [!] Đăng ký tác vụ tự chạy thất bại:", err)
									} else {
										fmt.Println("  [Tự chạy] Đăng ký thành công — Máy tính sẽ tự động mở khoá PCIe khi đăng nhập Windows")
									}
									st.scanOnce()
								}()
							}},
						},
					},
				},
			},
			GroupBox{
				Title:  "③ Nhật ký hoạt động (Thời gian thực)",
				Layout: VBox{},
				Children: []Widget{
					TextEdit{AssignTo: &st.teLog, ReadOnly: true, VScroll: true,
						MinSize: Size{Height: 120}, StretchFactor: 2},
				},
			},
		},
	}.Create()
	if createErr != nil {
		msgbox("Trình Cài Đặt 40HX / 30HX", "Khởi tạo GUI thất bại: "+createErr.Error()+"\nVui lòng sử dụng chế độ dòng lệnh (40HXInstaller.exe -h).", mbIconError)
		return
	}
	st.log.mw = st.mw
	st.log.te = st.teLog
	AttachLogSink(st.log)

	st.loadPolicyUI()

	fmt.Println("Trình Mở Khoá CMP 40HX / 30HX v3.0.0 đã khởi động (Quyền Admin).")
	go func() {
		// 打开自动扫描一次: 预勾选 + 顶部提示; 期间禁用执行按钮防并发。
		// 结果由 applySmartDefaults 打印([i] 已预勾… / [i] 组件均已就绪…)。
		st.setActionsEnabled(false)
		defer st.setActionsEnabled(true)
		st.scanOnce()
		st.applySmartDefaults()
	}()
	st.mw.Run()
}

// installSelected: Cài đặt các thành phần và thiết lập đã chọn (thực hiện tuần tự, đưa vào bảng log).
func (st *guiState) installSelected(sel map[string]bool) {
	defer st.end()
	st.sync(func() { st.pbInstall.SetEnabled(false) })
	defer st.sync(func() { st.pbInstall.SetEnabled(true) })
	nameOf := map[string]string{
		"gsp": "Bật GSP", "drv": "Cài đặt Driver PCIe", "efi": "EFI Mở khoá + Khởi động",
		"task": "Tự mở khoá khi đăng nhập", "fast": "Tắt Khởi động nhanh", "aspm": "Tắt ASPM",
		"perf": "Hiệu năng cao (High Perf)", "defoff": "Tắt Defender thời gian thực",
	}
	var parts []string
	for _, k := range []string{"gsp", "drv", "efi", "task", "fast", "aspm", "perf", "defoff"} {
		if sel[k] {
			parts = append(parts, nameOf[k])
		}
	}
	if len(parts) == 0 {
		fmt.Println("[i] Chưa chọn thành phần nào — Vui lòng tích chọn trước khi bấm [Cài đặt mục đã chọn]")
		return
	}
	fmt.Println("== Thực hiện: " + strings.Join(parts, " / ") + " ==")
	if sel["gsp"] {
		fmt.Println("──── Bật GSP (EnableGpuFirmware=1, then chốt chống đen màn hình) ────")
		if err := enableGsp(); err != nil {
			fmt.Println("  [GSP] Thất bại:", err)
		} else {
			fmt.Println("  [GSP] EnableGpuFirmware=1 Đã thiết lập (Khởi động lại máy để GSP-RM có hiệu lực)")
		}
	}
	if sel["drv"] {
		fmt.Println("──── Cài đặt Driver PCIe + Thêm loại trừ Defender ────")
		installDrivers()
		if err := hxcore.AddDefenderExclusions(); err != nil {
			st.printDefErr("  [Defender] ", err)
		} else {
			fmt.Println("  [Defender] Đã thêm thư mục driver và ProgramData vào danh sách loại trừ")
		}
	}
	if sel["efi"] {
		fmt.Println("──── Triển khai EFI Mở khoá + Mục khởi động BIOS (Chỉ 40HX) ────")
		installEFI()
	}
	if sel["task"] {
		fmt.Println("──── Tự động mở khoá PCIe khi đăng nhập (Tác vụ SYSTEM + Run key) ────")
		setRunKey()
		if err := setupGen2Task(); err != nil {
			fmt.Println("  [!] Đăng ký tác vụ tự chạy thất bại:", err)
			fmt.Println("  [!] Có thể chạy bằng tay với quyền Admin: 40HXInstaller.exe -task")
		}
	}
	if sel["fast"] || sel["aspm"] || sel["perf"] {
		fmt.Println("──── Cài đặt Nguồn điện (Có thể bật lại trong Windows Power Options) ────")
	}
	if sel["fast"] {
		if hxcore.FastStartupOn() {
			if err := hxcore.SetFastStartupOff(); err != nil {
				fmt.Println("  [Nguồn] Tắt Khởi động nhanh (Fast Startup) thất bại:", err)
			} else {
				fmt.Println("  [Nguồn] Đã tắt Khởi động nhanh (HiberbootEnabled=0)")
			}
		} else {
			fmt.Println("  [Nguồn] Khởi động nhanh: Đã tắt từ trước (OK)")
		}
	}
	if sel["aspm"] {
		if ac, dc, ok := hxcore.ASPMSavings(); !ok {
			fmt.Println("  [Nguồn] PCIe ASPM: Máy tính không hỗ trợ cài đặt này, bỏ qua")
		} else if ac == 0 && dc == 0 {
			fmt.Println("  [Nguồn] PCIe ASPM: Đã tắt từ trước (OK)")
		} else {
			if err := hxcore.SetASPMOff(); err != nil {
				fmt.Println("  [Nguồn] Tắt ASPM thất bại:", err)
			} else {
				fmt.Printf("  [Nguồn] PCIe ASPM đã tắt (Gốc: AC=%d/DC=%d)\n", ac, dc)
			}
		}
	}
	if sel["perf"] {
		if hxcore.HighPerfPlanActive() {
			fmt.Println("  [Nguồn] Chế độ nguồn: Đã là High Performance (OK)")
		} else if err := hxcore.SetHighPerfPlan(); err != nil {
			fmt.Println("  [Nguồn] Chuyển sang High Performance thất bại:", err)
		} else {
			fmt.Println("  [Nguồn] Đã chuyển sang chế độ High Performance")
		}
	}
	if sel["defoff"] {
		fmt.Println("──── Bảo vệ thời gian thực Windows Defender ────")
		on, err := hxcore.DefenderRealtimeProtectionOn()
		if err != nil {
			st.printDefErr("  [!] ", err)
		} else if !on {
			fmt.Println("  [Defender] Bảo vệ thời gian thực hiện đang Tắt (Không cần thao tác)")
		} else if err := hxcore.SetDefenderRealtimeProtection(false); err != nil {
			st.printDefErr("  [!] ", err)
			if !errors.Is(err, hxcore.ErrMpUnavailable) {
				fmt.Println("  [!] Nguyên nhân thường gặp: Tính năng 'Tamper Protection' đang bật — Vui lòng tắt trong Windows Security rồi thử lại")
			}
		} else {
			fmt.Println("  [Defender] Đã tắt bảo vệ thời gian thực")
			fmt.Println("  [Defender] Bật lại: Chạy PowerShell Admin: Set-MpPreference -DisableRealtimeMonitoring $False")
		}
	}
	fmt.Println("== Hoàn tất thực hiện, tự động quét lại trạng thái ==")
	st.scanOnce()
	st.applySmartDefaults()
}
