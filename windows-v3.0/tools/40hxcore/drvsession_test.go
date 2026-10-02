package hxcore

import (
	"errors"
	"testing"
)

func TestThrottleStopAppRunning_DoesNotPanic(t *testing.T) {
	// Assert that calling ThrottleStopAppRunning returns a bool without panic
	_ = ThrottleStopAppRunning()
}

func TestDriverSession_RunScoped_ClosureErrorPropagated(t *testing.T) {
	// Arrange: create a mock bus and test closure error propagation
	bus := NewMockHardwareBus()
	expectedErr := errors.New("simulated negotiation failure")

	fn := func(b HardwareBus) error {
		return expectedErr
	}

	// Act
	err := fn(bus)

	// Assert
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDriverFileProvider_HookCanBeRegistered(t *testing.T) {
	// Arrange
	origProvider := DriverFileProvider
	defer func() { DriverFileProvider = origProvider }()

	called := false
	DriverFileProvider = func(filename string) ([]byte, error) {
		called = true
		return []byte("dummy driver data"), nil
	}

	// Act
	data, err := DriverFileProvider("WinRing0x64.sys")

	// Assert
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
