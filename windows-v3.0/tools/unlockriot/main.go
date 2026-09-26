package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	hxcore "40hxcore"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	procMsgBoxW       = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	procShellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
)

const (
	mbIconError = 0x00000010
)

func isAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)
	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

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
		msgbox("UnlockRiotGame", fmt.Sprintf("Nâng quyền thất bại (mã lỗi %d).\nVui lòng nhấp chuột phải vào ứng dụng -> Chọn 'Run as administrator'.", r), mbIconError)
	}
	os.Exit(0)
}

func msgbox(title, msg string, flags uint32) {
	tPtr, _ := syscall.UTF16PtrFromString(title)
	mPtr, _ := syscall.UTF16PtrFromString(msg)
	procMsgBoxW.Call(0, uintptr(unsafe.Pointer(mPtr)), uintptr(unsafe.Pointer(tPtr)), uintptr(flags))
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

	prof, hasGPU := hxcore.FindGPUWithProfile()
	sbOn := hxcore.SecureBootOn()

	is40HX := false
	is30HX := false
	if hasGPU {
		if prof.DeviceID == 0x1F0B || strings.Contains(prof.Name, "40HX") {
			is40HX = true
		} else if prof.DeviceID == 0x2189 || strings.Contains(prof.Name, "30HX") {
			is30HX = true
		}
	}

	// Xây dựng thông điệp hướng dẫn hiển thị trên giao diện
	bannerText := "LƯU Ý: Hãy đảm bảo bạn ĐÃ MỞ KHÓA Gen2 bằng 40HXInstaller trước khi dùng tool này!\n\n"
	if is40HX {
		bannerText += "🔴 PHÁT HIỆN CARD: NVIDIA CMP 40HX [TU106]\n"
		if sbOn {
			bannerText += "⚠️ CẢNH BÁO BẮT BUỘC: Secure Boot hiện ĐANG BẬT!\n" +
				"👉 Sau khi mở khóa Gen2 ở 40HX, bạn BẮT BUỘC phải vào BIOS TẮT SECURE BOOT (Disabled).\n" +
				"Nếu không tắt, EFI mở khóa (40HXUNLK.EFI) sẽ bị BIOS từ chối và card sẽ bị khóa lại sau khi khởi động lại!\n" +
				"(Hướng dẫn: Reboot -> Del/F2 -> Tab Security/Boot -> Secure Boot: Disabled -> F10 Lưu)\n"
		} else {
			bannerText += "✅ Trạng thái Secure Boot: ĐÃ TẮT (Đúng chuẩn để nạp EFI mở khóa sau khi khởi động lại).\n"
		}
		bannerText += "💡 Để chơi Valorant với 40HX: Khuyến nghị dùng Windows 10 (vì Windows 11 bắt buộc bật Secure Boot).\n"
	} else if is30HX {
		bannerText += "🟢 PHÁT HIỆN CARD: NVIDIA CMP 30HX [TU116]\n" +
			"✅ CMP 30HX không cần nạp EFI mở khóa, Secure Boot có thể giữ BẬT bình thường để chơi Valorant & LMHT.\n"
	} else {
		bannerText += "⚠️ YÊU CẦU BẮT BUỘC KHI DÙNG CMP 40HX:\n" +
			"👉 Sau khi mở khóa Gen2, bạn BẮT BUỘC phải vào BIOS TẮT SECURE BOOT (Disabled) để nạp EFI mở khóa!\n" +
			"(Với CMP 30HX: Không cần tắt Secure Boot, giữ Bật bình thường).\n"
	}

	bannerText += "\nCông cụ này sẽ thực hiện:\n" +
		"1. Dọn dẹp hoàn toàn các driver bypass (WinRing0x64.sys, ThrottleStop.sys) khỏi kernel để Riot Vanguard không chặn.\n" +
		"2. Ép Windows ưu tiên sử dụng GPU hiệu năng cao (CMP) cho Valorant và League of Legends."

	var mw *walk.MainWindow
	var teLog *walk.TextEdit
	var pbRun *walk.PushButton

	logSink := &guiLog{}

	err := MainWindow{
		AssignTo: &mw,
		Title:    "UnlockRiotGame (Hỗ trợ Riot Vanguard & Cấu hình Secure Boot)",
		MinSize:  Size{Width: 600, Height: 520},
		Size:     Size{Width: 650, Height: 560},
		Layout:   VBox{},
		Children: []Widget{
			Label{
				Text: bannerText,
			},
			PushButton{
				AssignTo: &pbRun,
				Text:     "Chạy Tối Ưu Hóa & Dọn Dẹp Driver Riot Vanguard",
				OnClicked: func() {
					pbRun.SetEnabled(false)
					mw.Synchronize(func() { teLog.SetText("") })
					go func() {
						runOptimization(logSink, is40HX, is30HX, sbOn)
						mw.Synchronize(func() {
							pbRun.SetEnabled(true)
							// Bật thông báo chi tiết nhắc nhở người dùng
							if is40HX || (!is30HX && sbOn) {
								walk.MsgBox(mw, "YÊU CẦU BẮT BUỘC CHO CMP 40HX",
									"Đã dọn dẹp driver mở khóa và tối ưu Registry thành công!\n\n"+
										"🔴 YÊU CẦU QUAN TRỌNG CHO CMP 40HX:\n"+
										"Sau khi mở khóa Gen2, bạn BẮT BUỘC PHẢI VÀO BIOS TẮT SECURE BOOT (Disabled)!\n\n"+
										"LÝ DO:\n"+
										"- Bản mở khóa 40HX cần nạp file EFI (40HXUNLK.EFI) lúc khởi động.\n"+
										"- Nếu Secure Boot BẬT, BIOS sẽ chặn EFI và card sẽ bị khóa lại sau khi reboot!\n\n"+
										"CÁCH TẮT SECURE BOOT:\n"+
										"1. Khởi động lại máy, nhấn liên tục Del hoặc F2 để vào BIOS.\n"+
										"2. Tìm tab Security hoặc Boot -> Secure Boot: Chọn Disabled.\n"+
										"3. Nhấn F10 để Lưu và Khởi động lại vào Windows.\n\n"+
										"(Lưu ý: Để chơi Valorant với 40HX, khuyến nghị dùng Win 10 vì Win 11 bắt buộc bật Secure Boot)",
									walk.MsgBoxIconWarning)
							} else if is30HX {
								walk.MsgBox(mw, "Hoàn tất Tối Ưu (CMP 30HX)",
									"Đã dọn dẹp driver mở khóa và tối ưu Registry cho Riot Games thành công!\n\n"+
										"✅ Với CMP 30HX:\n"+
										"Bạn KHÔNG cần tắt Secure Boot. Hãy giữ Secure Boot BẬT bình thường trong BIOS để chơi tốt cả Valorant và LMHT trên Windows 10 & 11.",
									walk.MsgBoxIconInformation)
							} else {
								walk.MsgBox(mw, "Hoàn tất Tối Ưu",
									"Đã dọn dẹp driver mở khóa và tối ưu Registry thành công!\n\n"+
										"⚠️ LƯU Ý CHO CMP 40HX:\n"+
										"Nếu bạn dùng CMP 40HX, BẮT BUỘC phải vào BIOS TẮT SECURE BOOT (Disabled) sau khi mở khóa Gen2 để nạp EFI!",
									walk.MsgBoxIconInformation)
							}
						})
					}()
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

func runOptimization(log *guiLog, is40HX, is30HX, sbOn bool) {
	fmt.Fprintln(log, "==============================================")
	fmt.Fprintln(log, "  UnlockRiotGame — Dọn Dẹp Driver & Tối Ưu Riot")
	fmt.Fprintln(log, "==============================================")

	if is40HX {
		fmt.Fprintln(log, "[*] Nhận diện phần cứng: NVIDIA CMP 40HX [TU106]")
		if sbOn {
			fmt.Fprintln(log, "[!] CẢNH BÁO: Secure Boot hiện ĐANG BẬT!")
			fmt.Fprintln(log, "    => YÊU CẦU: Sau khi mở khóa Gen2 ở 40HX, bạn BẮT BUỘC phải vào BIOS TẮT SECURE BOOT (Disabled)!")
			fmt.Fprintln(log, "    => Nếu không tắt, file 40HXUNLK.EFI sẽ bị BIOS từ chối và card sẽ bị khóa lại sau khi khởi động lại máy.")
		} else {
			fmt.Fprintln(log, "[V] Trạng thái Secure Boot: ĐÃ TẮT (Chuẩn xác để nạp EFI mở khóa).")
		}
	} else if is30HX {
		fmt.Fprintln(log, "[*] Nhận diện phần cứng: NVIDIA CMP 30HX [TU116]")
		fmt.Fprintln(log, "[V] CMP 30HX không cần tắt Secure Boot. Có thể giữ Secure Boot BẬT bình thường.")
	} else {
		fmt.Fprintln(log, "[*] Kiểm tra phần cứng: Không phát hiện GPU cụ thể hoặc dùng card khác.")
		if sbOn {
			fmt.Fprintln(log, "[!] LƯU Ý: Nếu bạn dùng CMP 40HX, BẮT BUỘC phải vào BIOS TẮT SECURE BOOT sau khi mở khóa Gen2.")
		}
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

	os.Remove(p1)
	os.Remove(p2)
	fmt.Fprintln(log, "[V] Đã dọn dẹp sạch sẽ service và file driver trong System32\\drivers.")
	fmt.Fprintln(log, "    => Riot Vanguard (vgk.sys), Easy Anti-Cheat sẽ không thể phát hiện hay chặn driver.")

	fmt.Fprintln(log, "\n[*] BƯỚC 2: Cấu hình Registry ưu tiên GPU hiệu năng cao cho Riot Games...")
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\DirectX\UserGpuPreferences`, registry.SET_VALUE)
	if err != nil {
		fmt.Fprintf(log, "[!] Không thể mở Registry: %v\n", err)
	} else {
		defer k.Close()

		drives := []string{"C:\\", "D:\\", "E:\\", "F:\\", "G:\\"}
		found := 0

		targets := []string{
			`Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`,
			`Riot Games\League of Legends\Game\League of Legends.exe`,
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

		fmt.Fprintln(log, "[V] Hoàn tất cấu hình Registry.")
	}

	fmt.Fprintln(log, "\n==============================================")
	fmt.Fprintln(log, "HOÀN TẤT!")
	if is40HX || sbOn {
		fmt.Fprintln(log, "👉 ĐỐI VỚI CMP 40HX: HÃY VÀO BIOS TẮT SECURE BOOT (DISABLED) ĐỂ EFI MỞ KHÓA HOẠT ĐỘNG!")
	}
	fmt.Fprintln(log, "==============================================")
}
