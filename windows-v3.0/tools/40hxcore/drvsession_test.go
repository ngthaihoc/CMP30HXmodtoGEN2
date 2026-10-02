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
	defer func() {
		driverCmdRunner = origRunner
		isThrottleStopAppRunning = origAppRunning
	}()

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
	defer func() {
		driverCmdRunner = origRunner
		isThrottleStopAppRunning = origAppRunning
	}()

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
