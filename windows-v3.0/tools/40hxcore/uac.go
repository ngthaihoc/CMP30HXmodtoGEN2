package hxcore

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	MbIconError = 0x10
	MbIconWarn  = 0x30
	MbIconInfo  = 0x40
	MbYesNo     = 0x04
)

var (
	procShellExecuteW = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	procMsgBoxW       = syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
)

// IsAdmin checks if the current process is running with elevated administrator privileges.
func IsAdmin() bool {
	t, err := windows.OpenCurrentProcessToken()
	if err == nil {
		defer t.Close()
		var elevated uint32
		var returnedLen uint32
		if err = windows.GetTokenInformation(t, windows.TokenElevation,
			(*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &returnedLen); err == nil && elevated != 0 {
			return true
		}
	}
	// Fallback: check if we can open SCM with SC_MANAGER_ALL_ACCESS
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_ALL_ACCESS)
	if err == nil {
		windows.CloseServiceHandle(scm)
		return true
	}
	return false
}

// SelfElevate relaunches the current executable with administrative privileges via UAC "runas".
func SelfElevate(appTitle string) {
	// Phòng chống vòng lặp nâng quyền UAC
	for _, a := range os.Args[1:] {
		if a == "-elevated" {
			MsgBox(appTitle, "Nâng quyền thất bại: Tài khoản không thể nhận quyền Quản trị viên (Administrator).\nVui lòng nhấp chuột phải -> Chọn 'Run as administrator'.", MbIconError)
			os.Exit(1)
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)

	// Escape các tham số có chứa dấu cách/kí tự đặc biệt
	var escapedArgs []string
	for _, arg := range os.Args[1:] {
		if strings.ContainsAny(arg, " \t\n\v\"") {
			escapedArgs = append(escapedArgs, `"`+strings.ReplaceAll(arg, `"`, `\"`)+`"`)
		} else {
			escapedArgs = append(escapedArgs, arg)
		}
	}
	escapedArgs = append(escapedArgs, "-elevated")

	params, _ := syscall.UTF16PtrFromString(strings.Join(escapedArgs, " "))
	r, _, _ := procShellExecuteW.Call(0,
		uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)), 0, 1)
	if r <= 32 {
		MsgBox(appTitle, fmt.Sprintf("Nâng quyền thất bại (Mã lỗi %d).\nVui lòng nhấp chuột phải -> Chọn 'Run as administrator'.", r), MbIconError)
	}
	os.Exit(0)
}

// MsgBox displays a native Windows MessageBox modal.
func MsgBox(title, text string, icon uint) {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	procMsgBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), uintptr(icon))
}

// MsgBoxYesNo displays a native Windows MessageBox with Yes/No options.
func MsgBoxYesNo(title, text string) bool {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := procMsgBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), uintptr(MbYesNo|MbIconInfo))
	return r == 6 // IDYES
}
