package hxcore

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// DriverFileProvider: Hàm hook tuỳ chọn cho phép ứng dụng cấp dữ liệu file driver nhúng (embed)
var DriverFileProvider func(filename string) ([]byte, error)

// Test seams for driver execution and lifecycle mocking
var (
	driverCmdRunner          = RunOut
	createDriverService      = createRuntimeDriverService
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

// driverCommandError recognizes sc.exe's Win32 exit status.
func driverCommandError(err error, status windows.Errno) bool {
	var exitErr *exec.ExitError
	return errors.Is(err, status) || (errors.As(err, &exitErr) && exitErr.ExitCode() == int(status))
}

type ownedDriverService interface {
	Start(...string) error
	Control(svc.Cmd) (svc.Status, error)
	Query() (svc.Status, error)
	Delete() error
	Close() error
}

type loadedDriver struct {
	name    string
	service ownedDriverService
	path    string
	file    os.FileInfo
	started bool
}

func createRuntimeDriverService(name, path string) (ownedDriverService, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	defer m.Disconnect()
	service, err := m.CreateService(name, `\SystemRoot\System32\drivers\`+filepath.Base(path), mgr.Config{
		ServiceType:  windows.SERVICE_KERNEL_DRIVER,
		StartType:    mgr.StartManual,
		ErrorControl: mgr.ErrorNormal,
	})
	if err != nil {
		return nil, err
	}
	return service, nil
}

func acquireDriverSession() (func(), error) {
	// The lifetime of this named handle excludes other project processes.
	// We do not take thread ownership of the mutex.
	name, _ := windows.UTF16PtrFromString("Global\\CMPUnlockDriverSession")
	handle, err := windows.CreateMutex(nil, false, name)
	if handle != 0 && err != nil {
		_ = windows.CloseHandle(handle)
	}
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return nil, errors.New("một phiên driver CMP khác đang chạy")
	}
	if err != nil {
		return nil, fmt.Errorf("không thể khóa phiên driver CMP: %w", err)
	}
	return func() { _ = windows.CloseHandle(handle) }, nil
}

func loadDriver(name, filename string, bundledOnly bool) (*loadedDriver, error) {
	d := &loadedDriver{name: name, path: filepath.Join(sysRoot(), "System32", "drivers", filename)}
	out, err := driverCmdRunner("sc.exe", "query", name)
	if err == nil {
		if strings.Contains(out, "RUNNING") {
			return d, nil
		}
		if bundledOnly {
			return d, fmt.Errorf("dịch vụ %s đã tồn tại và chưa chạy", name)
		}
		out, err = driverCmdRunner("sc.exe", "start", name)
		if err != nil && !driverCommandError(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
			return d, fmt.Errorf("không thể khởi động dịch vụ có sẵn %s: %s (%w)", name, strings.TrimSpace(out), err)
		}
		return d, nil
	}
	if !driverCommandError(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return d, fmt.Errorf("không thể kiểm tra dịch vụ %s: %s (%w)", name, strings.TrimSpace(out), err)
	}

	if err := d.stage(filename, bundledOnly); err != nil {
		return d, err
	}
	d.service, err = createDriverService(name, d.path)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_EXISTS) || errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
			// Another process registered this name. Its service may use the file.
			d.file = nil
		}
		return d, fmt.Errorf("không thể tạo dịch vụ %s: %w", name, err)
	}
	if err := d.service.Start(); err != nil {
		return d, fmt.Errorf("không thể khởi động dịch vụ %s: %w", name, err)
	}
	d.started = true
	return d, nil
}

func (d *loadedDriver) stage(filename string, bundledOnly bool) error {
	info, err := os.Lstat(d.path)
	if err == nil {
		if bundledOnly {
			return fmt.Errorf("giữ nguyên tệp driver có sẵn: %s", d.path)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("tệp driver có sẵn không hợp lệ: %s", d.path)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	var data []byte
	if DriverFileProvider != nil {
		data, err = DriverFileProvider(filename)
	}
	if !bundledOnly && (err != nil || len(data) == 0) {
		data, err = os.ReadFile(filepath.Join(PdDrvDir(), filename))
	}
	if err != nil || len(data) == 0 {
		return fmt.Errorf("không tìm thấy nguồn driver hợp lệ cho %s", filename)
	}
	f, err := os.OpenFile(d.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	d.file, err = f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	_, writeErr := f.Write(data)
	return errors.Join(writeErr, f.Close())
}

func (d *loadedDriver) cleanup(keep bool) error {
	if d.service != nil {
		defer d.service.Close()
		if keep && d.started {
			return nil
		}
		if d.name == "ThrottleStop" && isThrottleStopAppRunning() {
			return errors.New("ThrottleStop đang chạy; giữ nguyên driver của phiên này")
		}
		if err := stopOwnedDriver(d.service); err != nil {
			return fmt.Errorf("không thể dừng dịch vụ %s: %w", d.name, err)
		}
		if err := d.service.Delete(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
			return fmt.Errorf("không thể xóa dịch vụ %s: %w", d.name, err)
		}
	}
	if d.file == nil {
		return nil
	}
	current, err := os.Lstat(d.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !os.SameFile(d.file, current) {
		return fmt.Errorf("tệp driver đã bị thay thế; giữ nguyên %s", d.path)
	}
	return os.Remove(d.path)
}

func stopOwnedDriver(service ownedDriverService) error {
	state, err := service.Control(svc.Stop)
	if errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return nil
	}
	if err != nil {
		return err
	}
	deadline := time.Now().Add(2 * time.Second)
	for state.State == svc.StopPending && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
		state, err = service.Query()
		if err != nil {
			return err
		}
	}
	if state.State != svc.Stopped {
		return fmt.Errorf("dịch vụ chưa dừng (trạng thái %d)", state.State)
	}
	return nil
}

var diagnosticDrivers struct {
	sync.Mutex
	owned   []*loadedDriver
	release func()
}

// EnsureDriverLoaded loads a driver for an explicit diagnostic session.
func EnsureDriverLoaded(name, filename string) error {
	diagnosticDrivers.Lock()
	defer diagnosticDrivers.Unlock()
	if diagnosticDrivers.release == nil {
		release, err := acquireDriverSession()
		if err != nil {
			return err
		}
		diagnosticDrivers.release = release
	}
	d, err := loadDriver(name, filename, false)
	diagnosticDrivers.owned = append(diagnosticDrivers.owned, d)
	return err
}

// CleanupByovd releases only resources created by this process for diagnostics.
func CleanupByovd() {
	diagnosticDrivers.Lock()
	defer diagnosticDrivers.Unlock()
	if diagnosticDrivers.release == nil {
		return
	}
	defer diagnosticDrivers.release()
	keep := driverStrategyGetter() == DriverStrategyResident
	for _, driver := range diagnosticDrivers.owned {
		_ = driver.cleanup(keep)
	}
	diagnosticDrivers.owned = nil
	diagnosticDrivers.release = nil
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

// RunScopedBus uses the configured driver strategy for this hardware operation.
func RunScopedBus(needsPL0 bool, fn func(bus HardwareBus) error) error {
	return runDriverSession(needsPL0, driverStrategyGetter() == DriverStrategyResident, false, fn)
}

// RunTransientDriverSession uses the bundled drivers, leaves existing driver
// files and stopped services alone, and does not apply registry policy.
func RunTransientDriverSession(needsPL0 bool, fn func(bus HardwareBus) error) error {
	return runDriverSession(needsPL0, false, true, fn)
}

func runDriverSession(needsPL0, keep, bundledOnly bool, fn func(HardwareBus) error) (result error) {
	release, err := acquireDriverSession()
	if err != nil {
		return err
	}
	defer release()

	wr, err := loadDriver("WinRing0_1_2_0", "WinRing0x64.sys", bundledOnly)
	defer func() { result = errors.Join(result, wr.cleanup(keep)) }()
	if err != nil {
		return fmt.Errorf("WinRing0 error: %w", err)
	}
	wh, err := openWinRing0()
	if err != nil {
		return fmt.Errorf("không thể mở handle WinRing0: %w", err)
	}
	defer driverCloseHandle(wh)

	var th syscall.Handle
	if needsPL0 {
		ts, err := loadDriver("ThrottleStop", "ThrottleStop.sys", bundledOnly)
		defer func() { result = errors.Join(result, ts.cleanup(keep)) }()
		if err == nil {
			if handle, err := openThrottleStop(); err == nil {
				th = handle
				defer driverCloseHandle(th)
			}
		}
	}
	return fn(NewProductionBus(wh, th))
}

// ClassifyDriverLoadError chuyển đổi mã lỗi nạp driver thô sang nguyên nhân và cách xử lý dễ hiểu (HVCI, Memory Integrity, v.v.)
func ClassifyDriverLoadError(raw string) string {
	r := strings.ToLower(raw)
	switch {
	case strings.Contains(r, "1275"):
		return "Chính sách bảo mật Windows chặn tải driver (Lỗi 1275). Xem nhật ký Code Integrity để xác định chính sách đang áp dụng."
	case strings.Contains(r, "577"):
		return "Windows từ chối chữ ký driver (Lỗi 577). Kiểm tra tính toàn vẹn của gói phát hành và nhật ký Code Integrity."
	case strings.Contains(r, "1058"):
		return "Dịch vụ có sẵn bị vô hiệu hóa (Lỗi 1058). Công cụ giữ nguyên cấu hình dịch vụ."
	case strings.Contains(r, "1072"):
		return "Dịch vụ đang được đánh dấu để xóa (Lỗi 1072). Đợi Windows hoàn thành hoặc khởi động lại rồi thử lại."
	case strings.Contains(r, "拒绝访问"), strings.Contains(r, "access is denied"), strings.Contains(r, "error 5"), strings.Contains(r, " 5:"):
		return "Không đủ quyền hạn hoặc bị chặn driver (Lỗi 5 / Access Denied) — Nếu đã chạy Administrator: Lỗi do Tính toàn vẹn bộ nhớ (HVCI) hoặc Danh sách chặn driver (Vulnerable Driver Blocklist). CẦN KHỞI ĐỘNG LẠI MÁY (REBOOT) để Windows áp dụng tắt HVCI."
	case strings.Contains(r, "1060"), strings.Contains(r, "不存在"):
		return "Không tìm thấy dịch vụ (Lỗi 1060) — Tệp driver chưa được triển khai thành công, hãy chạy lại trình cài đặt rồi thử lại"
	}
	return "Khởi động driver thất bại — Thường do tính năng HIPS / chặn driver của phần mềm diệt virus bên thứ ba, vui lòng thêm hai tệp .sys vào danh sách tin cậy / loại trừ rồi thử lại; nếu vẫn không được hãy gửi nhật ký cho tác giả"
}

// EnsureDiagnosticDrivers triển khai và khởi động ThrottleStop + WinRing0 cho công cụ chẩn đoán,
// trả về thông tin triển khai, tính sẵn sàng và nguyên nhân lỗi được phân loại chi tiết
func EnsureDiagnosticDrivers() (deployed bool, ok bool, fail string) {
	svcTS := "ThrottleStop"
	svcWR := "WinRing0_1_2_0"
	fileTS := "ThrottleStop.sys"
	fileWR := "WinRing0x64.sys"

	_, _, stTS := ServiceInfo(svcTS)
	_, _, stWR := ServiceInfo(svcWR)
	if stTS == "RUNNING" && stWR == "RUNNING" {
		return false, true, ""
	}

	deployed = true
	var fails []string
	if err := EnsureDriverLoaded(svcTS, fileTS); err != nil {
		fails = append(fails, fileTS+": "+ClassifyDriverLoadError(err.Error()))
	}
	if err := EnsureDriverLoaded(svcWR, fileWR); err != nil {
		fails = append(fails, fileWR+": "+ClassifyDriverLoadError(err.Error()))
	}

	_, _, stTS = ServiceInfo(svcTS)
	_, _, stWR = ServiceInfo(svcWR)
	ok = stTS == "RUNNING" && stWR == "RUNNING"
	if !ok && len(fails) > 0 {
		fail = strings.Join(fails, " || ")
	}
	return deployed, ok, fail
}
