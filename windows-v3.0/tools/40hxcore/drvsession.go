package hxcore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// DriverFileProvider: Hàm hook tuỳ chọn cho phép ứng dụng cấp dữ liệu file driver nhúng (embed)
var DriverFileProvider func(filename string) ([]byte, error)

// Test seams for driver execution and lifecycle mocking
var (
	driverCmdRunner          = RunOut
	openWinRing0             = func() (syscall.Handle, error) { return OpenDevice(`\\.\WinRing0_1_2_0`) }
	openThrottleStop         = func() (syscall.Handle, error) { return OpenThrottleStop() }
	driverCloseHandle        = CloseHandle
	isThrottleStopAppRunning = ThrottleStopAppRunning
	driverStrategyGetter     = DriverStrategy
)

// ThrottleStopAppRunning kiểm tra xem phần mềm ThrottleStop.exe của người dùng có đang chạy không
func ThrottleStopAppRunning() bool {
	out, _ := RunOut("tasklist.exe", "/FI", "IMAGENAME eq ThrottleStop.exe")
	return strings.Contains(out, "ThrottleStop.exe")
}

// EnsureDriverLoaded triển khai file và khởi động service cho driver kernel BYOVD
func EnsureDriverLoaded(svcName, fileName string) error {
	out, _ := driverCmdRunner("sc.exe", "query", svcName)
	if strings.Contains(out, "RUNNING") {
		return nil
	}

	sysDir := filepath.Join(sysRoot(), "System32", "drivers")
	dst := filepath.Join(sysDir, fileName)

	// Kiểm tra tính hợp lệ của file trong System32
	needCopy := false
	if b, err := os.ReadFile(dst); err != nil || len(b) == 0 {
		needCopy = true
	}

	if needCopy {
		copied := false
		// 1. Thử lấy từ ProgramData\40HXUnlock\drivers
		src := filepath.Join(PdDrvDir(), fileName)
		if sb, err := os.ReadFile(src); err == nil && len(sb) > 0 {
			if err := os.WriteFile(dst, sb, 0o644); err == nil {
				copied = true
			}
		}
		// 2. Thử lấy từ DriverFileProvider (embed trong bộ cài đặt)
		if !copied && DriverFileProvider != nil {
			if eb, err := DriverFileProvider(fileName); err == nil && len(eb) > 0 {
				if err := os.WriteFile(dst, eb, 0o644); err == nil {
					copied = true
				}
			}
		}
		if copied {
			_ = AddDefenderExclusions()
		}
	}

	bin := `\SystemRoot\System32\drivers\` + fileName
	driverCmdRunner("sc.exe", "create", svcName, "type=", "kernel", "start=", "demand", "binPath=", bin)
	if _, err := driverCmdRunner("sc.exe", "start", svcName); err != nil {
		// Thử xoá và tạo lại (phòng trường hợp service bị đánh dấu delete hoặc config lỗi)
		driverCmdRunner("sc.exe", "delete", svcName)
		driverCmdRunner("sc.exe", "create", svcName, "type=", "kernel", "start=", "demand", "binPath=", bin)
		if out2, err2 := driverCmdRunner("sc.exe", "start", svcName); err2 != nil {
			return fmt.Errorf("không thể khởi động dịch vụ %s: %s (%v)", svcName, strings.TrimSpace(out2), err2)
		}
	}
	return nil
}

// CleanupByovd dừng và xoá dịch vụ driver BYOVD cùng file .sys trong System32
// nhằm đảm bảo an toàn tuyệt đối trước các hệ thống Anti-Cheat
func CleanupByovd() {
	if driverStrategyGetter() == DriverStrategyResident {
		return
	}
	appRunning := isThrottleStopAppRunning()

	for _, d := range []struct{ name, file string }{
		{"ThrottleStop", "ThrottleStop.sys"},
		{"WinRing0_1_2_0", "WinRing0x64.sys"},
	} {
		if d.name == "ThrottleStop" && appRunning {
			continue
		}
		driverCmdRunner("sc.exe", "stop", d.name)
		driverCmdRunner("sc.exe", "delete", d.name)
		_ = os.Remove(filepath.Join(sysRoot(), "System32", "drivers", d.file))
	}
}

// DriverSession: Deep Module quản lý vòng đời đóng kín (RAII) của driver kernel
type DriverSession struct{}

// NewDriverSession khởi tạo instance DriverSession
func NewDriverSession() *DriverSession {
	return &DriverSession{}
}

// RunScoped thực thi tác vụ với HardwareBus phù hợp với GPUProfile thông qua receiver
func (s *DriverSession) RunScoped(prof GPUProfile, fn func(bus HardwareBus) error) error {
	return RunScoped(prof, fn)
}

// RunScoped thực thi tác vụ với HardwareBus phù hợp với GPUProfile,
// đảm bảo tự động dọn dẹp driver ngay cả khi xảy ra lỗi hoặc panic
func RunScoped(prof GPUProfile, fn func(bus HardwareBus) error) error {
	needsMMIO := prof.RequiresMMIO || prof.HasSafePL0 || prof.DeviceID == 0x2189 || prof.Family == "TU116" || prof.Family == "TU106"
	return RunScopedBus(needsMMIO, fn)
}

// RunScopedBus thực thi fn với HardwareBus, tải ThrottleStop nếu needsPL0=true
func RunScopedBus(needsPL0 bool, fn func(bus HardwareBus) error) error {
	defer CleanupByovd()

	// 1. Tải và mở WinRing0
	if err := EnsureDriverLoaded("WinRing0_1_2_0", "WinRing0x64.sys"); err != nil {
		return fmt.Errorf("WinRing0 error: %w", err)
	}
	wh, err := openWinRing0()
	if err != nil {
		return fmt.Errorf("không thể mở handle WinRing0: %w", err)
	}
	defer driverCloseHandle(wh)

	// 2. Tải và mở ThrottleStop nếu cần truy cập MMIO vật lý (tuỳ chọn: trên CPU AMD không hỗ trợ MSR Intel hoặc khi bị chặn, tiếp tục với WinRing0)
	var th syscall.Handle = 0
	if needsPL0 {
		if err := EnsureDriverLoaded("ThrottleStop", "ThrottleStop.sys"); err == nil {
			if t, err := openThrottleStop(); err == nil {
				th = t
				defer driverCloseHandle(th)
			}
		}
	}

	// 3. Khởi tạo ProductionBus và chạy closure
	bus := NewProductionBus(wh, th)
	return fn(bus)
}
