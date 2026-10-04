package hxcore

import (
	"errors"
	"syscall"
	"testing"
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

func TestCleanupByovd_AlwaysPurgesWinRing0EvenIfThrottleStopRunning(t *testing.T) {
	origRunner := driverCmdRunner
	origAppRunning := isThrottleStopAppRunning
	origStrategy := driverStrategyGetter
	defer func() {
		driverCmdRunner = origRunner
		isThrottleStopAppRunning = origAppRunning
		driverStrategyGetter = origStrategy
	}()

	driverStrategyGetter = func() int { return DriverStrategyTransient }
	isThrottleStopAppRunning = func() bool { return true }
	var stoppedServices []string
	var deletedServices []string

	driverCmdRunner = func(name string, args ...string) (string, error) {
		if name == "sc.exe" && len(args) >= 2 {
			if args[0] == "stop" {
				stoppedServices = append(stoppedServices, args[1])
			} else if args[0] == "delete" {
				deletedServices = append(deletedServices, args[1])
			}
		}
		return "", nil
	}

	CleanupByovd()

	hasWinRingStop := false
	hasWinRingDelete := false
	for _, s := range stoppedServices {
		if s == "WinRing0_1_2_0" {
			hasWinRingStop = true
		}
		if s == "ThrottleStop" {
			t.Errorf("ThrottleStop was stopped while app was running, expected it to be skipped")
		}
	}
	for _, s := range deletedServices {
		if s == "WinRing0_1_2_0" {
			hasWinRingDelete = true
		}
		if s == "ThrottleStop" {
			t.Errorf("ThrottleStop was deleted while app was running, expected it to be skipped")
		}
	}
	if !hasWinRingStop || !hasWinRingDelete {
		t.Fatalf("WinRing0 was not cleaned up when ThrottleStop was running (stop=%v, delete=%v)", hasWinRingStop, hasWinRingDelete)
	}
}

func TestCleanupByovd_PurgesBothWhenThrottleStopNotRunning(t *testing.T) {
	origRunner := driverCmdRunner
	origAppRunning := isThrottleStopAppRunning
	origStrategy := driverStrategyGetter
	defer func() {
		driverCmdRunner = origRunner
		isThrottleStopAppRunning = origAppRunning
		driverStrategyGetter = origStrategy
	}()

	driverStrategyGetter = func() int { return DriverStrategyTransient }
	isThrottleStopAppRunning = func() bool { return false }
	var stoppedServices []string
	var deletedServices []string
	driverCmdRunner = func(name string, args ...string) (string, error) {
		if name == "sc.exe" && len(args) >= 2 {
			if args[0] == "stop" {
				stoppedServices = append(stoppedServices, args[1])
			} else if args[0] == "delete" {
				deletedServices = append(deletedServices, args[1])
			}
		}
		return "", nil
	}

	CleanupByovd()

	hasTSStop, hasWRStop := false, false
	hasTSDelete, hasWRDelete := false, false
	for _, s := range stoppedServices {
		if s == "ThrottleStop" {
			hasTSStop = true
		}
		if s == "WinRing0_1_2_0" {
			hasWRStop = true
		}
	}
	for _, s := range deletedServices {
		if s == "ThrottleStop" {
			hasTSDelete = true
		}
		if s == "WinRing0_1_2_0" {
			hasWRDelete = true
		}
	}
	if !hasTSStop || !hasWRStop || !hasTSDelete || !hasWRDelete {
		t.Fatalf("expected both drivers purged, got TS(stop=%v, del=%v), WR(stop=%v, del=%v)", hasTSStop, hasTSDelete, hasWRStop, hasWRDelete)
	}
}

func TestCleanupByovd_PreservesDriversWhenResidentStrategy(t *testing.T) {
	origRunner := driverCmdRunner
	origStrategy := driverStrategyGetter
	defer func() {
		driverCmdRunner = origRunner
		driverStrategyGetter = origStrategy
	}()

	driverStrategyGetter = func() int { return DriverStrategyResident }
	called := false
	driverCmdRunner = func(name string, args ...string) (string, error) {
		called = true
		return "", nil
	}

	CleanupByovd()

	if called {
		t.Fatalf("expected CleanupByovd to return immediately when DriverStrategy is Resident, but driverCmdRunner was called")
	}
}

func TestDriverSession_RunScopedBus_RAIICleanupOnClosureError(t *testing.T) {
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
