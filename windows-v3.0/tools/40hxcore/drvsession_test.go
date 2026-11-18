package hxcore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

func TestThrottleStopAppRunning_DoesNotPanic(t *testing.T) {
	_ = ThrottleStopAppRunning()
}

func TestDriverFileProvider_HookCanBeRegistered(t *testing.T) {
	origProvider := DriverFileProvider
	defer func() { DriverFileProvider = origProvider }()

	called := false
	DriverFileProvider = func(filename string) ([]byte, error) {
		called = true
		return []byte("dummy driver data"), nil
	}

	data, err := DriverFileProvider("WinRing0x64.sys")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected provider to be called")
	}
	if string(data) != "dummy driver data" {
		t.Fatalf("expected dummy driver data, got %s", string(data))
	}
}

type fakeOwnedDriverService struct {
	events    *[]string
	startErr  error
	stopErr   error
	deleteErr error
}

func (s *fakeOwnedDriverService) Start(...string) error {
	*s.events = append(*s.events, "start")
	return s.startErr
}

func (s *fakeOwnedDriverService) Control(command svc.Cmd) (svc.Status, error) {
	*s.events = append(*s.events, "stop")
	return svc.Status{State: svc.Stopped}, s.stopErr
}

func (s *fakeOwnedDriverService) Query() (svc.Status, error) {
	return svc.Status{State: svc.Stopped}, nil
}

func (s *fakeOwnedDriverService) Delete() error {
	*s.events = append(*s.events, "delete")
	return s.deleteErr
}

func (s *fakeOwnedDriverService) Close() error {
	*s.events = append(*s.events, "close service")
	return nil
}

func TestDriverSessionCleansOnlyOwnedResources(t *testing.T) {
	for _, tc := range []struct {
		name            string
		preexistingFile bool
		resident        bool
		forceTransient  bool
		startErr        error
		stopErr         error
		wantErr         error
		wantEvents      string
		wantFile        bool
	}{
		{name: "transient", wantEvents: "start,close device,stop,delete,close service"},
		{name: "shared file", preexistingFile: true, wantEvents: "start,close device,stop,delete,close service", wantFile: true},
		{name: "resident", resident: true, wantEvents: "start,close device,close service", wantFile: true},
		{name: "transient overrides resident", resident: true, forceTransient: true, wantEvents: "start,close device,stop,delete,close service"},
		{name: "start failure", resident: true, startErr: windows.ERROR_ACCESS_DENIED, wantErr: windows.ERROR_ACCESS_DENIED, wantEvents: "start,stop,delete,close service"},
		{name: "stop failure", stopErr: windows.ERROR_DEPENDENT_SERVICES_RUNNING, wantErr: windows.ERROR_DEPENDENT_SERVICES_RUNNING, wantEvents: "start,close device,stop,close service", wantFile: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("SystemRoot", root)
			t.Setenv("ProgramData", t.TempDir())
			path := filepath.Join(root, "System32", "drivers", "WinRing0x64.sys")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.preexistingFile {
				if err := os.WriteFile(path, []byte("shared driver"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			originalRunner, originalCreate := driverCmdRunner, createDriverService
			originalOpen, originalClose := openWinRing0, driverCloseHandle
			originalStrategy, originalProvider := driverStrategyGetter, DriverFileProvider
			t.Cleanup(func() {
				driverCmdRunner, createDriverService = originalRunner, originalCreate
				openWinRing0, driverCloseHandle = originalOpen, originalClose
				driverStrategyGetter, DriverFileProvider = originalStrategy, originalProvider
			})
			driverCmdRunner = func(string, ...string) (string, error) {
				return "", windows.ERROR_SERVICE_DOES_NOT_EXIST
			}
			var events []string
			createDriverService = func(name, binaryPath string) (ownedDriverService, error) {
				if name != "WinRing0_1_2_0" || binaryPath != path {
					t.Errorf("unexpected service: %q %q", name, binaryPath)
				}
				return &fakeOwnedDriverService{events: &events, startErr: tc.startErr, stopErr: tc.stopErr}, nil
			}
			openWinRing0 = func() (syscall.Handle, error) { return 111, nil }
			driverCloseHandle = func(syscall.Handle) { events = append(events, "close device") }
			DriverFileProvider = func(string) ([]byte, error) { return []byte("session driver"), nil }
			driverStrategyGetter = func() int {
				if tc.resident {
					return DriverStrategyResident
				}
				return DriverStrategyTransient
			}

			run := RunScopedBus
			if tc.forceTransient {
				run = RunTransientDriverSession
			}
			err := run(false, func(HardwareBus) error { return nil })
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("session returned %v, want %v", err, tc.wantErr)
			}
			if got := strings.Join(events, ","); got != tc.wantEvents {
				t.Errorf("lifecycle = %q, want %q", got, tc.wantEvents)
			}
			data, readErr := os.ReadFile(path)
			if tc.wantFile {
				if readErr != nil {
					t.Fatalf("driver file should remain: %v", readErr)
				}
				if tc.preexistingFile && string(data) != "shared driver" {
					t.Errorf("shared driver was replaced: %q", data)
				}
			} else if !errors.Is(readErr, os.ErrNotExist) {
				t.Errorf("session-owned driver file remains: %q, %v", data, readErr)
			}
		})
	}
}

func TestTransientDriverSessionLeavesStoppedSharedServiceUntouched(t *testing.T) {
	originalRunner := driverCmdRunner
	t.Cleanup(func() { driverCmdRunner = originalRunner })
	driverCmdRunner = func(name string, args ...string) (string, error) {
		if args[0] != "query" {
			t.Errorf("shared service was modified: %q %v", name, args)
		}
		return "STATE: 1 STOPPED", nil
	}

	err := RunTransientDriverSession(false, func(HardwareBus) error {
		t.Fatal("hardware operation reached with a stopped driver")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "WinRing0") {
		t.Fatalf("expected stopped WinRing0 error, got %v", err)
	}
}

func TestTransientDriverUsesBundledSourceAndPreservesExistingFile(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	filename := "WinRing0x64.sys"
	if err := os.MkdirAll(PdDrvDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(PdDrvDir(), filename), []byte("stale backup"), 0o644); err != nil {
		t.Fatal(err)
	}
	origProvider := DriverFileProvider
	t.Cleanup(func() { DriverFileProvider = origProvider })
	DriverFileProvider = func(string) ([]byte, error) { return []byte("embedded driver"), nil }

	d := &loadedDriver{path: filepath.Join(t.TempDir(), filename)}
	if err := d.stage(filename, true); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(d.path); err != nil || string(data) != "embedded driver" {
		t.Fatalf("staged %q, err = %v", data, err)
	}
	if err := d.cleanup(false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(d.path, []byte("existing driver"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := &loadedDriver{path: d.path}
	if err := other.stage(filename, true); err == nil {
		t.Fatal("bundled-only session accepted a pre-existing System32 file")
	}
	if err := other.cleanup(false); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(d.path); err != nil || string(data) != "existing driver" {
		t.Fatalf("shared file changed: %q, err = %v", data, err)
	}
}

func TestDriverSessionExcludesAnotherProjectProcess(t *testing.T) {
	release, err := acquireDriverSession()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if another, err := acquireDriverSession(); err == nil {
		another()
		t.Fatal("a second driver session acquired the same lock")
	}
}

func TestDriverSessionPreservesPreexistingService(t *testing.T) {
	t.Setenv("SystemRoot", t.TempDir())
	origRunner := driverCmdRunner
	origOpenWR := openWinRing0
	origClose := driverCloseHandle
	origStrategy := driverStrategyGetter
	t.Cleanup(func() {
		driverCmdRunner = origRunner
		openWinRing0 = origOpenWR
		driverCloseHandle = origClose
		driverStrategyGetter = origStrategy
	})
	driverStrategyGetter = func() int { return DriverStrategyTransient }
	driverCmdRunner = func(name string, args ...string) (string, error) {
		if args[0] != "query" {
			t.Errorf("shared driver command %q %v", name, args)
		}
		return "STATE: 4  RUNNING", nil
	}
	openWinRing0 = func() (syscall.Handle, error) { return 111, nil }
	driverCloseHandle = func(syscall.Handle) {}

	err := RunScopedBus(false, func(HardwareBus) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestDriverSession_RunScopedBus_RAIICleanupOnClosureError(t *testing.T) {
	t.Setenv("SystemRoot", t.TempDir())
	origRunner := driverCmdRunner
	origOpenWR := openWinRing0
	origOpenTS := openThrottleStop
	origClose := driverCloseHandle
	origAppRunning := isThrottleStopAppRunning
	defer func() {
		driverCmdRunner = origRunner
		openWinRing0 = origOpenWR
		openThrottleStop = origOpenTS
		driverCloseHandle = origClose
		isThrottleStopAppRunning = origAppRunning
	}()

	isThrottleStopAppRunning = func() bool { return false }
	driverCmdRunner = func(name string, args ...string) (string, error) {
		if name == "sc.exe" && len(args) >= 2 && args[0] == "query" {
			return "STATE: 4  RUNNING", nil
		}
		return "", nil
	}

	closedHandles := []syscall.Handle{}
	openWinRing0 = func() (syscall.Handle, error) { return 111, nil }
	openThrottleStop = func() (syscall.Handle, error) { return 222, nil }
	driverCloseHandle = func(h syscall.Handle) {
		closedHandles = append(closedHandles, h)
	}

	expectedErr := errors.New("simulated failure inside closure")

	err := RunScopedBus(true, func(bus HardwareBus) error {
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	has111, has222 := false, false
	for _, h := range closedHandles {
		if h == 111 {
			has111 = true
		}
		if h == 222 {
			has222 = true
		}
	}
	if !has111 || !has222 {
		t.Fatalf("expected handles 111 and 222 to be closed, got %v", closedHandles)
	}
}

func TestDriverSession_RunScoped_EnablesMMIOForTU116(t *testing.T) {
	t.Setenv("SystemRoot", t.TempDir())
	origRunner := driverCmdRunner
	origOpenWR := openWinRing0
	origOpenTS := openThrottleStop
	origClose := driverCloseHandle
	defer func() {
		driverCmdRunner = origRunner
		openWinRing0 = origOpenWR
		openThrottleStop = origOpenTS
		driverCloseHandle = origClose
	}()

	driverCmdRunner = func(name string, args ...string) (string, error) {
		return "STATE: 4  RUNNING", nil
	}
	openWinRing0 = func() (syscall.Handle, error) { return 111, nil }
	tsOpened := false
	openThrottleStop = func() (syscall.Handle, error) {
		tsOpened = true
		return 222, nil
	}
	driverCloseHandle = func(h syscall.Handle) {}

	prof30 := GPUProfile{DeviceID: 0x2189, Family: "TU116", RequiresMMIO: true}

	err := RunScoped(prof30, func(bus HardwareBus) error {
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tsOpened {
		t.Fatalf("expected ThrottleStop to be opened for CMP 30HX profile requiring MMIO")
	}
}

func TestDriverSession_RunScopedBus_FallbackWhenThrottleStopFails(t *testing.T) {
	t.Setenv("SystemRoot", t.TempDir())
	origRunner := driverCmdRunner
	origOpenWR := openWinRing0
	origOpenTS := openThrottleStop
	origClose := driverCloseHandle
	defer func() {
		driverCmdRunner = origRunner
		openWinRing0 = origOpenWR
		openThrottleStop = origOpenTS
		driverCloseHandle = origClose
	}()

	driverCmdRunner = func(name string, args ...string) (string, error) {
		if len(args) >= 2 && args[0] == "query" && args[1] == "WinRing0_1_2_0" {
			return "STATE: 4  RUNNING", nil
		}
		if len(args) >= 2 && (args[0] == "start" || args[0] == "query") && args[1] == "ThrottleStop" {
			return "", errors.New("ERROR_GEN_FAILURE 31")
		}
		return "", nil
	}
	openWinRing0 = func() (syscall.Handle, error) { return 111, nil }
	openThrottleStop = func() (syscall.Handle, error) { return 0, errors.New("driver not started") }
	driverCloseHandle = func(h syscall.Handle) {}

	called := false
	err := RunScopedBus(true, func(bus HardwareBus) error {
		called = true
		prod, ok := bus.(*ProductionBus)
		if !ok {
			t.Fatalf("expected bus to be *ProductionBus")
		}
		if prod.wh != 111 {
			t.Errorf("expected WinRing0 handle 111, got %v", prod.wh)
		}
		if prod.th != 0 {
			t.Errorf("expected ThrottleStop handle 0, got %v", prod.th)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected RunScopedBus to succeed with fallback, got error: %v", err)
	}
	if !called {
		t.Fatalf("expected closure to be executed despite ThrottleStop failure")
	}
}

func TestProductionBus_MMIO_FallbackToWinRing0(t *testing.T) {
	// Khi cả hai handle = 0
	busNoHandles := NewProductionBus(0, 0)
	if _, err := busNoHandles.ReadMMIO(0x1000); err == nil {
		t.Errorf("expected error when both handles are zero")
	}
	if err := busNoHandles.WriteMMIO(0x1000, 0x123); err == nil {
		t.Errorf("expected error when both handles are zero")
	}

	// Khi th = 0 nhưng wh != 0, ProductionBus phải uỷ quyền sang WinRing0 (WRReadMem / WRWriteMem)
	busFallback := NewProductionBus(999, 0)
	if busFallback.th != 0 || busFallback.wh != 999 {
		t.Fatalf("unexpected handles: wh=%v, th=%v", busFallback.wh, busFallback.th)
	}
	// WRReadMem/WRWriteMem gọi IoCtl với handle 999 (sẽ trả lỗi ioctl thay vì 'ThrottleStop handle is zero')
	_, rErr := busFallback.ReadMMIO(0x1000)
	if rErr != nil && rErr.Error() == "ThrottleStop handle is zero" {
		t.Errorf("ReadMMIO should have fallen back to WinRing0 instead of returning ThrottleStop handle is zero")
	}
	wErr := busFallback.WriteMMIO(0x1000, 0x123)
	if wErr != nil && wErr.Error() == "ThrottleStop handle is zero" {
		t.Errorf("WriteMMIO should have fallen back to WinRing0 instead of returning ThrottleStop handle is zero")
	}
}

func TestDriverSession_RunScoped_Delegates(t *testing.T) {
	t.Setenv("SystemRoot", t.TempDir())
	origRunner := driverCmdRunner
	origOpenWR := openWinRing0
	origClose := driverCloseHandle
	defer func() {
		driverCmdRunner = origRunner
		openWinRing0 = origOpenWR
		driverCloseHandle = origClose
	}()

	driverCmdRunner = func(name string, args ...string) (string, error) {
		return "STATE: 4  RUNNING", nil
	}
	openWinRing0 = func() (syscall.Handle, error) { return 111, nil }
	driverCloseHandle = func(h syscall.Handle) {}

	session := NewDriverSession()
	called := false
	err := session.RunScoped(GPUProfile{DeviceID: 0x1F0B, Family: "TU106"}, func(bus HardwareBus) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected closure to be called")
	}
}

func TestClassifyDriverLoadError_KnownCodes(t *testing.T) {
	cases := []struct {
		errStr   string
		contains string
	}{
		{"Failed with error 1275", "1275"},
		{"System blocked error 577", "577"},
		{"Service disabled 1058", "1058"},
		{"Service marked for deletion 1072", "1072"},
		{"Access is denied", "HVCI"},
		{"Service error 1060 not found", "1060"},
		{"Random unknown error", "HIPS"},
	}

	for _, c := range cases {
		classified := ClassifyDriverLoadError(c.errStr)
		if !strings.Contains(classified, c.contains) {
			t.Errorf("expected %q to contain %q, got: %s", c.errStr, c.contains, classified)
		}
	}
}
