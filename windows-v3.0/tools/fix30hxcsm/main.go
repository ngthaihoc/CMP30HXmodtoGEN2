package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	hxcore "40hxcore"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type guiLog struct {
	mw *walk.MainWindow
	te *walk.TextEdit
}

func (g *guiLog) Write(p []byte) (int, error) {
	s := string(p)
	if g.mw != nil && g.te != nil {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		s = strings.ReplaceAll(s, "\n", "\r\n")
		g.mw.Synchronize(func() { g.te.AppendText(s) })
	}
	return len(p), nil
}

func main() {
	flagFix := flag.Bool("fix", false, "Tu dong sua loi 43 va khoi phuc card bien mat trong CSM")
	flagScan := flag.Bool("scan", false, "Quet lai bus PCIe de tim card bi mat")
	flagWatchdog := flag.Bool("watchdog", false, "Cai dat tac vu canh gac tu dong Keep-Alive")
	flagUnwatchdog := flag.Bool("unwatchdog", false, "Go bo tac vu canh gac tu dong Keep-Alive")
	flagStatus := flag.Bool("status", false, "Kiem tra va in bao cao trang thai chan doan")
	flagNoAdmin := flag.Bool("noadmin", false, "Bo qua kiem tra quyen Administrator")
	flag.Parse()

	// Kiểm tra quyền Administrator (trừ khi có cờ -noadmin hoặc chỉ xem -status)
	if !*flagNoAdmin && !*flagStatus && !hxcore.IsAdmin() {
		hxcore.SelfElevate("FixCMP30HX_CSM")
		return
	}

	// 1. Chế độ dòng lệnh (CLI Mode)
	if *flagFix || *flagScan || *flagWatchdog || *flagUnwatchdog || *flagStatus {
		attachConsoleIfCLI()
		runCLI(*flagFix, *flagScan, *flagWatchdog, *flagUnwatchdog, *flagStatus)
		return
	}

	// 2. Chế độ giao diện đồ họa (GUI Mode)
	runGUI()
}

func attachConsoleIfCLI() {
	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle := modkernel32.NewProc("GetStdHandle")
	procGetFileType := modkernel32.NewProc("GetFileType")
	const STD_OUTPUT_HANDLE = ^uint32(10) // -11
	const STD_ERROR_HANDLE = ^uint32(11)  // -12
	const FILE_TYPE_CHAR = 2

	hStdOut, _, _ := procGetStdHandle.Call(uintptr(STD_OUTPUT_HANDLE))
	ft, _, _ := procGetFileType.Call(hStdOut)

	// Nếu stdout đã được redirect vào pipe hoặc file (khác char/console hoặc hStdOut hợp lệ và khác 0)
	if hStdOut != 0 && hStdOut != uintptr(syscall.InvalidHandle) && ft != 0 && ft != FILE_TYPE_CHAR {
		os.Stdout = os.NewFile(hStdOut, "/dev/stdout")
		hStdErr, _, _ := procGetStdHandle.Call(uintptr(STD_ERROR_HANDLE))
		if hStdErr != 0 && hStdErr != uintptr(syscall.InvalidHandle) {
			os.Stderr = os.NewFile(hStdErr, "/dev/stderr")
		}
		return
	}

	procAttachConsole := modkernel32.NewProc("AttachConsole")
	const ATTACH_PARENT_PROCESS = ^uint32(0) // -1
	r, _, _ := procAttachConsole.Call(uintptr(ATTACH_PARENT_PROCESS))
	if r != 0 {
		h, err := syscall.CreateFile(
			syscall.StringToUTF16Ptr("CONOUT$"),
			syscall.GENERIC_READ|syscall.GENERIC_WRITE,
			syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
			nil,
			syscall.OPEN_EXISTING,
			0,
			0,
		)
		if err == nil {
			f := os.NewFile(uintptr(h), "/dev/stdout")
			os.Stdout = f
			os.Stderr = f
		}
	} else if hStdOut != 0 && hStdOut != uintptr(syscall.InvalidHandle) {
		os.Stdout = os.NewFile(hStdOut, "/dev/stdout")
	}
}

func runCLI(doFix, doScan, doWatchdog, doUnwatchdog, doStatus bool) {
	fw := DetectFirmwareMode()
	gpu := ScanCMP30HXDevice()
	watchdog := IsWatchdogInstalled()

	if doStatus {
		fmt.Println("================================================================")
		fmt.Println("  BÁO CÁO CHẨN ĐOÁN CMP 30HX TRÊN HỆ THỐNG UEFI / CSM")
		fmt.Println("================================================================")
		fmt.Printf("Chế độ Firmware: %s (Raw: %d)\n", fw.ModeName, fw.RawValue)
		if fw.DiskDetails != "" {
			fmt.Printf("Cấu trúc đĩa:    %s\n", fw.DiskDetails)
		}
		fmt.Printf("Secure Boot:     %v\n", fw.SecureBoot)
		fmt.Println("----------------------------------------------------------------")
		fmt.Printf("CMP 30HX Phát Hiện: %v\n", gpu.Detected)
		fmt.Printf("Trạng thái cắm:     %v (Hidden/Ghost: %v)\n", gpu.IsPresent, gpu.IsHiddenOrGhost)
		if gpu.DeviceInstanceID != "" {
			fmt.Printf("Device Instance ID: %s\n", gpu.DeviceInstanceID)
		}
		if gpu.DriverClassIndex != "" {
			fmt.Printf("Driver Class Index: %s\n", gpu.DriverClassIndex)
		}
		if gpu.IsBasicDisplay {
			fmt.Println("[!] CẢNH BÁO DRIVER: CMP 30HX đang chạy 'Microsoft Basic Display Adapter'!")
			fmt.Println("    -> Cần cài driver NVIDIA mod DEV_2189 để kích hoạt 3D và CUDA.")
		} else if gpu.DriverDesc != "" {
			fmt.Printf("Driver hiện tại:    %s (%s)\n", gpu.DriverDesc, gpu.DriverVersion)
		}
		if gpu.ProblemCode != 0 {
			fmt.Printf("Mã lỗi thiết bị:    Code %d (%s)\n", gpu.ProblemCode, gpu.ProblemDesc)
		} else if gpu.Detected && gpu.IsPresent && !gpu.IsBasicDisplay {
			fmt.Println("Mã lỗi thiết bị:    Không có lỗi (Hoạt động tốt)")
		}
		fmt.Printf("Chống ngủ D3cold:   %v\n", gpu.HasD3ColdBlocked)
		fmt.Printf("Kích hoạt CASO:     %v\n", gpu.HasCASOEnabled)
		fmt.Printf("Canh gác Watchdog:  %v\n", watchdog)
		fmt.Println("================================================================")
	}

	if doScan {
		_ = RescanPCIBus(os.Stdout)
		gpu = ScanCMP30HXDevice()
	}

	if doFix {
		fmt.Println("[*] Bắt đầu quy trình tự động sửa lỗi và cấu hình tối ưu...")
		if !gpu.IsPresent {
			_ = RescanPCIBus(os.Stdout)
			gpu = ScanCMP30HXDevice()
		}
		_ = FixCSMPowerSettings(os.Stdout, &gpu)
		_ = FixError43AndCASO(os.Stdout, &gpu)
		fmt.Println("[OK] Quy trình hoàn tất! Kiểm tra lại Device Manager.")
	}

	if doWatchdog {
		_ = InstallWatchdogTask(os.Stdout)
	}

	if doUnwatchdog {
		_ = RemoveWatchdogTask(os.Stdout)
	}
}

func runGUI() {
	// Ẩn console window nếu người dùng mở ở chế độ GUI
	moduser32 := syscall.NewLazyDLL("user32.dll")
	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow := modkernel32.NewProc("GetConsoleWindow")
	procShowWindow := moduser32.NewProc("ShowWindow")
	if hwnd, _, _ := procGetConsoleWindow.Call(); hwnd != 0 {
		procShowWindow.Call(hwnd, 0) // SW_HIDE
	}

	fw := DetectFirmwareMode()
	gpu := ScanCMP30HXDevice()
	watchdog := IsWatchdogInstalled()

	var mw *walk.MainWindow
	var teLog *walk.TextEdit
	var lblStatus *walk.Label
	var pbFix *walk.PushButton
	var pbScan *walk.PushButton
	var pbWatchdog *walk.PushButton
	var pbMBR2GPT *walk.PushButton
	var pbGuide *walk.PushButton

	logSink := &guiLog{}

	getStatusBanner := func() string {
		var sb strings.Builder
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString(fmt.Sprintf("🖥️  CHẾ ĐỘ MÁY: %s", fw.ModeName))
		if fw.IsCSM {
			sb.WriteString(" ⚠️ [Đang dùng CSM - Cần tối ưu chống rớt card & Code 43]\n")
		} else {
			sb.WriteString(" [UEFI Thuần]\n")
		}

		if !gpu.Detected || !gpu.IsPresent {
			sb.WriteString("❌ TRẠNG THÁI GPU: CMP 30HX ĐANG BỊ BIẾN MẤT KHỎI DEVICE MANAGER (Code 45)!\n")
			sb.WriteString("   (Do tụt link PCIe, Windows D3cold hoặc khe PCIe UEFI chưa khóa Gen2)\n")
		} else if gpu.IsBasicDisplay {
			sb.WriteString("⚠️ TRẠNG THÁI GPU: ĐANG CHẠY 'MICROSOFT BASIC DISPLAY ADAPTER'!\n")
			sb.WriteString("   (Cần cài Driver NVIDIA mod DEV_2189 qua Have Disk để nhận full 3D/CUDA)\n")
		} else if gpu.ProblemCode == 43 {
			sb.WriteString("⚠️ TRẠNG THÁI GPU: ĐANG DÍNH LỖI 43 (CODE 43)!\n")
			sb.WriteString("   (Do Above 4G bị tắt, thiếu CASO hoặc xung đột MMIO 32-bit)\n")
		} else if gpu.ProblemCode != 0 {
			sb.WriteString(fmt.Sprintf("⚠️ TRẠNG THÁI GPU: Mã sự cố Code %d (%s)\n", gpu.ProblemCode, gpu.ProblemDesc))
		} else {
			sb.WriteString("✅ TRẠNG THÁI GPU: Đang hoạt động bình thường trên Device Manager!\n")
		}

		if watchdog {
			sb.WriteString("🛡️  TÁC VỤ CANH GÁC: ĐÃ BẬT (Tự động phục hồi khi mở máy)\n")
		} else {
			sb.WriteString("🛡️  TÁC VỤ CANH GÁC: Chưa bật (Khuyến nghị bật để chống mất card sau sleep)\n")
		}
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return sb.String()
	}

	err := MainWindow{
		AssignTo: &mw,
		Title:    "Công Cụ Sửa Lỗi 43 & Biến Mất Device Manager Cho CMP 30HX (UEFI / CSM)",
		MinSize:  Size{Width: 720, Height: 680},
		Size:     Size{Width: 750, Height: 720},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				AssignTo: &lblStatus,
				Text:     getStatusBanner(),
				Font:     Font{Bold: true, PointSize: 9},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						AssignTo: &pbFix,
						Text:     "🛠️ TỰ ĐỘNG SỬA LỖI 43 & KHÔI PHỤC CARD (1-CHẠM KHUYẾN NGHỊ)",
						OnClicked: func() {
							pbFix.SetEnabled(false)
							pbScan.SetEnabled(false)
							teLog.SetText("")
							go func() {
								fmt.Fprintln(logSink, "================================================================")
								fmt.Fprintln(logSink, "  BẮT ĐẦU QUY TRÌNH TỰ ĐỘNG KHÔI PHỤC VÀ SỬA LỖI 43 CHO CMP 30HX")
								fmt.Fprintln(logSink, "================================================================")
								// 1. Quét bus nếu card chưa hiện
								if !gpu.IsPresent {
									_ = RescanPCIBus(logSink)
									gpu = ScanCMP30HXDevice()
								}
								// 2. Chống ngủ sâu trong CSM
								_ = FixCSMPowerSettings(logSink, &gpu)
								// 3. Sửa lỗi 43 & CASO
								_ = FixError43AndCASO(logSink, &gpu)
								// 4. Tự động bật watchdog
								_ = InstallWatchdogTask(logSink)
								watchdog = true

								// Quét lại trạng thái cuối
								gpu = ScanCMP30HXDevice()
								fw = DetectFirmwareMode()

								mw.Synchronize(func() {
									pbFix.SetEnabled(true)
									pbScan.SetEnabled(true)
									lblStatus.SetText(getStatusBanner())
									walk.MsgBox(mw, "Hoàn tất xử lý",
										"Đã áp dụng toàn bộ cấu hình sửa lỗi 43, khóa D3cold chống mất card,\n"+
											"và đăng ký tác vụ canh gác khởi động thành công!\n\n"+
											"Vui lòng kiểm tra lại Device Manager.",
										walk.MsgBoxIconInformation)
								})
							}()
						},
					},
					PushButton{
						AssignTo: &pbScan,
						Text:     "🔍 Quét Lại Bus PCIe",
						OnClicked: func() {
							pbScan.SetEnabled(false)
							go func() {
								_ = RescanPCIBus(logSink)
								gpu = ScanCMP30HXDevice()
								mw.Synchronize(func() {
									pbScan.SetEnabled(true)
									lblStatus.SetText(getStatusBanner())
								})
							}()
						},
					},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						AssignTo: &pbWatchdog,
						Text:     "🛡️ Bật/Tắt Canh Gác (Watchdog)",
						OnClicked: func() {
							if watchdog {
								_ = RemoveWatchdogTask(logSink)
								watchdog = false
								walk.MsgBox(mw, "Tác Vụ Canh Gác", "Đã gỡ bỏ tác vụ canh gác khởi động.", walk.MsgBoxIconInformation)
							} else {
								_ = InstallWatchdogTask(logSink)
								watchdog = true
								walk.MsgBox(mw, "Tác Vụ Canh Gác", "Đã bật tác vụ canh gác khởi động thành công!", walk.MsgBoxIconInformation)
							}
							lblStatus.SetText(getStatusBanner())
						},
					},
					PushButton{
						AssignTo: &pbMBR2GPT,
						Text:     "⚡ Kiểm Tra Chuyển Sang UEFI",
						OnClicked: func() {
							go func() {
								fmt.Fprintln(logSink, "[*] Đang kiểm tra tính tương thích chuyển đổi MBR sang GPT/UEFI...")
								can, msg := CheckMBR2GPT()
								mw.Synchronize(func() {
									if can {
										walk.MsgBox(mw, "Đủ Điều Kiện UEFI",
											"Máy tính của bạn đủ điều kiện nâng cấp lên UEFI thuần 100% không mất dữ liệu!\n\n"+
												"Lợi ích khi chuyển sang UEFI:\n"+
												"- Bật được Above 4G Decoding & Resizable BAR\n"+
												"- Triệt tiêu hoàn toàn lỗi 43 và biến mất card\n\n"+
												"Để chuyển đổi, bạn mở CMD (Admin) và chạy lệnh:\n"+
												"mbr2gpt /convert /allowfullos\n"+
												"Sau đó vào BIOS tắt CSM và bật UEFI Boot.",
											walk.MsgBoxIconInformation)
									} else {
										walk.MsgBox(mw, "Kiểm Tra MBR2GPT", msg, walk.MsgBoxIconWarning)
									}
								})
							}()
						},
					},
					PushButton{
						AssignTo: &pbGuide,
						Text:     "📖 Hướng Dẫn BIOS & CSM",
						OnClicked: func() {
							showBiosGuide(mw)
						},
					},
				},
			},
			Label{
				Text: "Nhật Ký Thực Thi (Execution Log):",
			},
			TextEdit{
				AssignTo: &teLog,
				ReadOnly: true,
				VScroll:  true,
			},
		},
	}.Create()

	if err != nil {
		return
	}

	logSink.mw = mw
	logSink.te = teLog

	// Ghi thông điệp chào mừng
	fmt.Fprintln(logSink, "Công cụ chuyên dụng sửa lỗi 43 và chống biến mất CMP 30HX trên hệ thống CSM / Legacy.")
	fmt.Fprintln(logSink, "Nhấn nút 'TỰ ĐỘNG SỬA LỖI 43 & KHÔI PHỤC CARD' để khắc phục 1-chạm.")

	mw.Run()
}

func showBiosGuide(owner walk.Form) {
	guide := `HƯỚNG DẪN CẤU HÌNH BIOS CHO CMP 30HX TRÊN MAIN CSM / LEGACY:

1. NGUYÊN NHÂN CHÍNH GÂY LỖI TRÊN CSM:
- CMP 30HX là card đào coin KHÔNG CÓ CỔNG XUẤT HÌNH (Headless).
- Khi chạy CSM, bo mạch chủ không hỗ trợ quản lý điện năng PCIe hiện đại, khiến card rơi vào trạng thái D3cold (ngủ sâu) và biến mất khỏi Bus.
- CSM thường tự động TẮT "Above 4G Decoding", làm VRAM 6GB của card không đủ chỗ trong vùng nhớ 32-bit (<4GB) dẫn đến LỖI 43.

2. CÁC THIẾT LẬP BIOS KHUYẾN NGHỊ:
- Primary Display / Initial Display Output: Chọn iGPU (card onboard) hoặc card phụ có cổng xuất hình. KHÔNG chọn PCIe slot của CMP 30HX.
- Above 4G Decoding: Nếu BIOS cho phép bật trong khi CSM bật -> Hãy chọn ENABLED.
- PCIe Link Speed: Đặt khe cắm CMP 30HX ở Gen2 hoặc Gen3 (hoặc Auto).
- Fast Boot / Fast Startup: Đặt DISABLED.

3. GIẢI PHÁP TỐI ƯU LÂU DÀI:
- Chuyển đổi ổ đĩa từ MBR sang GPT (dùng mbr2gpt) rồi TẮT HOÀN TOÀN CSM trong BIOS để chạy UEFI thuần. Khi đó bạn sẽ bật được Above 4G Decoding và Resizable BAR đạt hiệu năng tối đa!`

	walk.MsgBox(owner, "Hướng Dẫn BIOS & CSM", guide, walk.MsgBoxIconInformation)
}
