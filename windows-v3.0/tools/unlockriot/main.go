package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	hxcore "40hxcore"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows/registry"
)

const (
	DeviceIDCMP40HX = 0x1F0B // NVIDIA TU106 (CMP 40HX)
	DeviceIDCMP30HX = 0x2189 // NVIDIA TU116 (CMP 30HX)
)

func isAdmin() bool {
	return hxcore.IsAdmin()
}

func selfElevate() {
	hxcore.SelfElevate("UnlockRiotGame")
}

func msgbox(title, msg string, flags uint32) {
	hxcore.MsgBox(title, msg, uint(flags))
}

func isWindows11() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	buildStr, _, err := k.GetStringValue("CurrentBuild")
	if err != nil {
		buildStr, _, err = k.GetStringValue("CurrentBuildNumber")
	}
	if err != nil {
		return false
	}
	var b int
	fmt.Sscanf(buildStr, "%d", &b)
	return b >= 22000
}

func showBiosRebootDialog(owner walk.Form, res *SigningResult, uefiMgr UEFIManager) {
	var dlg *walk.Dialog
	var cbAgree *walk.CheckBox
	var pbReboot *walk.PushButton
	var accepted bool

	guideText := GetBiosRebootGuide(res)

	err := Dialog{
		AssignTo: &dlg,
		Title:    "HƯỚNG DẪN BẮT BUỘC TRONG BIOS SETUP (CHỤP ẢNH MÀN HÌNH NÀY)",
		MinSize:  Size{Width: 650, Height: 550},
		Size:     Size{Width: 700, Height: 600},
		Layout:   VBox{},
		Children: []Widget{
			TextEdit{
				Text:     guideText,
				ReadOnly: true,
				VScroll:  true,
			},
			CheckBox{
				AssignTo: &cbAgree,
				Text:     "Tôi đã dùng điện thoại chụp lại hướng dẫn này và cam kết thực hiện đúng các bước trong BIOS.",
				OnCheckedChanged: func() {
					pbReboot.SetEnabled(cbAgree.Checked())
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						AssignTo: &pbReboot,
						Text:     "🔄 Tôi Đồng Ý - Khởi Động Lại Vào BIOS Ngay",
						Enabled:  false,
						OnClicked: func() {
							accepted = true
							dlg.Accept()
						},
					},
					PushButton{
						Text: "Để Sau / Đóng",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Create(owner)

	if err != nil {
		walk.MsgBox(owner, "Lỗi hiển thị", fmt.Sprintf("Không thể mở hộp thoại hướng dẫn: %v", err), walk.MsgBoxIconError)
		return
	}

	dlg.Run()

	if accepted {
		if err := uefiMgr.RebootToFirmware(context.Background()); err != nil {
			walk.MsgBox(owner, "Khởi động lại",
				"Không thể tự động chuyển vào BIOS, máy tính sẽ khởi động lại bình thường.\n"+
					"Vui lòng nhấn liên tục phím Del hoặc F2 khi máy khởi động để vào BIOS!",
				walk.MsgBoxIconWarning)
		}
	}
}

func showTensorGuide(owner walk.Form, isWin11, is40HX bool) {
	title := "HƯỚNG DẪN GIỮ CẢ GEN 2 VÀ TENSOR CORE (RIOT GAMES)"
	msg := GetTensorGuide(isWin11, is40HX)
	walk.MsgBox(owner, title, msg, walk.MsgBoxIconInformation)
}

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
	if !isAdmin() {
		selfElevate()
		return
	}

	// Xử lý chế độ dòng lệnh (CLI flags) nếu có truyền tham số
	if len(os.Args) > 1 {
		opts, err := ParseFlags(os.Args[1:])
		if err == nil && (opts.Silent || opts.Optimize || opts.AutoSign || opts.CheckSig || opts.Gen2Warning) {
			if err := RunCLI(opts, os.Stdout, &DefaultUEFIManager{}); err != nil {
				os.Exit(1)
			}
			return
		}
	}

	prof, _ := hxcore.FindGPUWithProfile()
	sbOn := hxcore.SecureBootOn()
	isWin11 := isWindows11()

	status := EvaluateRiotStatus(prof.DeviceID, prof.Name, sbOn, isWin11)
	is40HX := (status.Model == GPUModelCMP40HX)
	is30HX := (status.Model == GPUModelCMP30HX)
	bannerText := status.BannerText

	var mw *walk.MainWindow
	var teLog *walk.TextEdit
	var pbRun *walk.PushButton
	var pbSign *walk.PushButton

	logSink := &guiLog{}

	err := MainWindow{
		AssignTo: &mw,
		Title:    "UnlockRiotGame (Hỗ trợ Riot Vanguard & Giữ Gen 2 + Tensor Core)",
		MinSize:  Size{Width: 650, Height: 600},
		Size:     Size{Width: 700, Height: 660},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: bannerText,
			},
			PushButton{
				AssignTo: &pbSign,
				Text:     "🔐 Tự Động Ký Chữ Ký Số EFI & Chuẩn Bị Key BIOS (Valorant Win 11)",
				OnClicked: func() {
					pbSign.SetEnabled(false)
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
						defer cancel()
						uefiMgr := &DefaultUEFIManager{}
						res, err := uefiMgr.PrepareAndSignEFI(ctx, logSink)
						mw.Synchronize(func() {
							pbSign.SetEnabled(true)
							if err != nil {
								walk.MsgBox(mw, "Lỗi Ký EFI", fmt.Sprintf("Không thể chuẩn bị và ký EFI: %v", err), walk.MsgBoxIconError)
								return
							}
							showBiosRebootDialog(mw, res, uefiMgr)
						})
					}()
				},
			},
			PushButton{
				AssignTo: &pbRun,
				Text:     "⚡ Chạy Tối Ưu Hóa & Dọn Dẹp Driver Riot Vanguard",
				OnClicked: func() {
					pbRun.SetEnabled(false)
					mw.Synchronize(func() { teLog.SetText("") })
					go func() {
						runOptimization(logSink, is40HX, is30HX, sbOn, isWin11)
						mw.Synchronize(func() {
							pbRun.SetEnabled(true)
							icon := walk.MsgBoxIconInformation
							if is40HX && isWin11 {
								icon = walk.MsgBoxIconWarning
							}
							walk.MsgBox(mw, status.CompletionTitle, status.CompletionDetail, icon)
						})
					}()
				},
			},
			PushButton{
				Text: "📖 Hướng Dẫn Giữ Cả Gen 2 & Tensor Core (Win 10 & Win 11)",
				OnClicked: func() {
					showTensorGuide(mw, isWin11, is40HX)
				},
			},
			PushButton{
				Text: "⚡ Kiểm Tra PCIe Gen 2.0 & Render Test (ASPM)",
				OnClicked: func() {
					walk.MsgBox(mw, "GIẢI THÍCH PCIE GEN 2.0 / ASPM", GetGen2AspmExplanation(), walk.MsgBoxIconInformation)
				},
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

	mw.Run()
}

func getLogicalDrives() []string {
	var drives []string
	for _, l := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		d := string(l) + `:\`
		if _, err := os.Stat(d); err == nil {
			drives = append(drives, d)
		}
	}
	return drives
}

func runOptimization(log io.Writer, is40HX, is30HX, sbOn, isWin11 bool) {
	fmt.Fprintln(log, "==============================================")
	fmt.Fprintln(log, "  UnlockRiotGame: Dọn Dẹp Driver & Tối Ưu Riot")
	fmt.Fprintln(log, "==============================================")

	osName := "Windows 10"
	if isWin11 {
		osName = "Windows 11"
	}
	fmt.Fprintf(log, "[*] Môi trường hệ điều hành: %s\n", osName)

	if is40HX {
		fmt.Fprintln(log, "[*] Nhận diện phần cứng: NVIDIA CMP 40HX [TU106]")
		if sbOn {
			fmt.Fprintln(log, "[!] CẢNH BÁO: Secure Boot hiện ĐANG BẬT!")
			if isWin11 {
				fmt.Fprintln(log, "    => Windows 11: Đủ điều kiện chơi Valorant, nhưng file 40HXUNLK.EFI sẽ bị BIOS chặn (mất Tensor Core)")
				fmt.Fprintln(log, "       trừ khi bạn tự ký chứng chỉ cá nhân và nạp vào BIOS db (Key Management)!")
			} else {
				fmt.Fprintln(log, "    => Windows 10: Nên vào BIOS TẮT Secure Boot để nạp EFI Tensor Core mà vẫn chơi được Valorant.")
			}
		} else {
			fmt.Fprintln(log, "[V] Trạng thái Secure Boot: ĐÃ TẮT.")
			fmt.Fprintln(log, "    => 40HXUNLK.EFI sẽ nạp trọn vẹn Tensor Core (SS0=0x88888888) & PCIe Gen 2.")
			if isWin11 {
				fmt.Fprintln(log, "    => Chơi tốt LMHT/TFT. Riêng Valorant trên Win 11 cần Secure Boot (xem hướng dẫn Custom Key hoặc Win 10 Dual Boot).")
			} else {
				fmt.Fprintln(log, "    => Tuyệt vời: Win 10 cho phép chơi mọi game Riot (Valorant + LMHT) giữ 100% Tensor Core & Gen 2!")
			}
		}
	} else if is30HX {
		fmt.Fprintln(log, "[*] Nhận diện phần cứng: NVIDIA CMP 30HX [TU116]")
		fmt.Fprintln(log, "[V] CMP 30HX không dùng EFI Tensor Core. Có thể giữ Secure Boot BẬT bình thường.")
	} else {
		fmt.Fprintln(log, "[*] Kiểm tra phần cứng: Không phát hiện GPU cụ thể hoặc dùng card khác.")
	}

	fmt.Fprintln(log, "\n[*] BƯỚC 1: Dọn dẹp driver mở khóa (WinRing0 / ThrottleStop)...")
	exec.Command("sc", "stop", "WinRing0_1_2_0").Run()
	exec.Command("sc", "delete", "WinRing0_1_2_0").Run()
	exec.Command("sc", "stop", "ThrottleStop").Run()
	exec.Command("sc", "delete", "ThrottleStop").Run()

	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}

	p1 := filepath.Join(sysRoot, "System32", "drivers", "WinRing0x64.sys")
	p2 := filepath.Join(sysRoot, "System32", "drivers", "ThrottleStop.sys")

	if err := os.Remove(p1); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(log, "[!] Cảnh báo: Không thể xóa %s: %v\n", p1, err)
	}
	if err := os.Remove(p2); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(log, "[!] Cảnh báo: Không thể xóa %s: %v\n", p2, err)
	}
	fmt.Fprintln(log, "[V] Đã dọn dẹp sạch sẽ service và file driver trong System32\\drivers.")
	fmt.Fprintln(log, "    => Riot Vanguard (vgk.sys), Easy Anti-Cheat sẽ không thể phát hiện hay chặn driver.")

	fmt.Fprintln(log, "\n[*] BƯỚC 2: Tắt chế độ Windows Testsigning (Code Integrity)...")
	if out, err := exec.Command("bcdedit", "/set", "testsigning", "off").CombinedOutput(); err != nil {
		fmt.Fprintf(log, "[!] Cảnh báo bcdedit (%v): %s\n", err, strings.TrimSpace(string(out)))
	} else {
		fmt.Fprintln(log, "[V] Đã đảm bảo Testsigning = OFF (đáp ứng tiêu chuẩn Riot Vanguard).")
	}

	fmt.Fprintln(log, "\n[*] BƯỚC 3: Cấu hình Registry Driver Class MSHybrid & CASO (Zero-Driver-Reinstall)...")
	var model GPUModel = GPUModelUnknown
	if is40HX {
		model = GPUModelCMP40HX
	} else if is30HX {
		model = GPUModelCMP30HX
	}
	if _, err := ConfigureDriverClassRegistry(log, model); err != nil {
		fmt.Fprintf(log, "[!] Cảnh báo cấu hình Driver Class Registry: %v\n", err)
	}

	fmt.Fprintln(log, "\n[*] BƯỚC 4: Cấu hình Registry DirectX ưu tiên GPU hiệu năng cao cho Riot Games...")
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\DirectX\UserGpuPreferences`, registry.SET_VALUE)
	if err != nil {
		fmt.Fprintf(log, "[!] Không thể mở Registry: %v\n", err)
	} else {
		defer k.Close()

		drives := getLogicalDrives()
		found := 0

		targets := []string{
			`Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`,
			`Riot Games\League of Legends\Game\League of Legends.exe`,
			`Riot Games\League of Legends\LeagueClient.exe`,
			`Riot Games\Riot Client\RiotClientServices.exe`,
		}

		for _, drive := range drives {
			for _, t := range targets {
				fullPath := filepath.Join(drive, t)
				if _, err := os.Stat(fullPath); err == nil {
					k.SetStringValue(fullPath, "GpuPreference=2;")
					fmt.Fprintf(log, "  -> Đã gán GpuPreference=2 (High Performance) cho: %s\n", fullPath)
					found++
				}
			}
		}

		if found == 0 {
			fmt.Fprintln(log, "  -> Không tìm thấy thư mục cài game tự động, gán giá trị mặc định cho ổ C:...")
			k.SetStringValue(filepath.Join(sysRoot[:3], targets[0]), "GpuPreference=2;")
			k.SetStringValue(filepath.Join(sysRoot[:3], targets[1]), "GpuPreference=2;")
		}

		fmt.Fprintln(log, "[V] Hoàn tất cấu hình Registry DirectX.")
	}

	fmt.Fprintln(log, "\n[*] BƯỚC 5: Cấu hình Borderless Windowed cho League of Legends...")
	modifiedCfgs, err := FindAndConfigureLeagueConfigs(getLogicalDrives())
	if err != nil || len(modifiedCfgs) == 0 {
		fmt.Fprintln(log, "  -> Không tìm thấy file game.cfg nào trên các ổ đĩa để sửa tự động.")
	} else {
		for _, cfg := range modifiedCfgs {
			fmt.Fprintf(log, "  -> Đã thiết lập WindowMode=2 (Borderless) cho: %s\n", cfg)
		}
		fmt.Fprintln(log, "  [V] Đã tối ưu swapchain DWM/CASO cho League of Legends không cổng xuất hình.")
	}

	fmt.Fprintln(log, "\n==============================================")
	fmt.Fprintln(log, "HOÀN TẤT TỐI ƯU HÓA!")
	if is40HX {
		if !isWin11 {
			fmt.Fprintln(log, "👉 Windows 10: Giữ Secure Boot TẮT -> Thưởng thức game và Tensor Core trọn vẹn!")
		} else {
			fmt.Fprintln(log, "👉 Windows 11: Nhấn nút [📖 Hướng Dẫn Giữ Cả Gen 2 & Tensor Core] để xem cách chơi Valorant.")
		}
	}
	fmt.Fprintln(log, "==============================================")
}
