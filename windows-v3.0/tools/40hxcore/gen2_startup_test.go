package hxcore

import (
	"errors"
	"reflect"
	"testing"
)

func TestRunGen2StartupOrderingAndRetry(t *testing.T) {
	var calls []string
	verdict := &NegotiationVerdict{Success: false, NeedsRetry: true, StatusContract: StatusContract{StatusCode: StatusGen1Stuck}}
	got := RunGen2Startup(Gen2StartupInput{RetryDepth: 2}, Gen2StartupHooks{
		AcquireSingleInstance: func() (bool, func()) {
			calls = append(calls, "acquire")
			return true, func() { calls = append(calls, "release") }
		},
		WaitForDriver: func() { calls = append(calls, "wait") },
		RunScoped: func(fn func(HardwareBus) error) error {
			calls = append(calls, "scoped")
			return fn(nil)
		},
		Negotiate: func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error) {
			calls = append(calls, "negotiate")
			return verdict, nil
		},
		WriteStatus: func(StatusContract) error { calls = append(calls, "status"); return nil },
		RetryDisposition: func(*NegotiationVerdict) Gen2RetryDisposition {
			calls = append(calls, "decide-retry")
			return Gen2RetrySchedule
		},
		ApplyRetry: func(disposition Gen2RetryDisposition, depth int) {
			if disposition != Gen2RetrySchedule || depth != 2 {
				t.Fatalf("retry args = %v, %d", disposition, depth)
			}
			calls = append(calls, "apply-retry")
		},
	})
	want := []string{"acquire", "wait", "scoped", "negotiate", "decide-retry", "apply-retry", "status", "release"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	if got.Verdict != verdict || got.RetryDisposition != Gen2RetrySchedule || got.Err != nil {
		t.Fatalf("result = %+v", got)
	}
}

func TestRunGen2StartupSkipWritesStatusWithoutWaiting(t *testing.T) {
	var calls []string
	got := RunGen2Startup(Gen2StartupInput{}, Gen2StartupHooks{
		AcquireSingleInstance: func() (bool, func()) { calls = append(calls, "acquire"); return false, nil },
		WaitForDriver:         func() { calls = append(calls, "wait") },
		RunScoped:             func(func(HardwareBus) error) error { calls = append(calls, "scoped"); return nil },
		Negotiate:             func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error) { return nil, nil },
		WriteStatus: func(status StatusContract) error {
			calls = append(calls, "status")
			if status.StatusCode != StatusGen2Skipped || status.ErrorCode != "ANOTHER_INSTANCE_RUNNING" {
				t.Fatalf("status = %+v", status)
			}
			return nil
		},
	})
	if !got.Skipped || !reflect.DeepEqual(calls, []string{"acquire", "status"}) {
		t.Fatalf("result = %+v; calls = %v", got, calls)
	}
}

func TestRunGen2StartupErrorAndCleanupPaths(t *testing.T) {
	statusErr := errors.New("status unavailable")
	t.Run("status failure", func(t *testing.T) {
		got := RunGen2Startup(Gen2StartupInput{}, startupTestHooks(&NegotiationVerdict{StatusContract: StatusContract{}}, nil, nil, statusErr, &startupTestHookState{}))
		if !errors.Is(got.StatusWriteError, statusErr) {
			t.Fatalf("status error = %v", got.StatusWriteError)
		}
	})
	t.Run("negotiation error", func(t *testing.T) {
		negErr := errors.New("negotiation failed")
		state := &startupTestHookState{}
		got := RunGen2Startup(Gen2StartupInput{}, startupTestHooks(&NegotiationVerdict{StatusContract: StatusContract{ErrorCode: "NEGOTIATION_ERROR"}}, negErr, nil, nil, state))
		if !errors.Is(got.Err, negErr) || got.Verdict == nil || !state.released || !state.wroteStatus {
			t.Fatalf("result = %+v; state = %+v", got, state)
		}
	})
	t.Run("scoped error releases and writes failure", func(t *testing.T) {
		scopedErr := errors.New("driver session failed")
		state := &startupTestHookState{}
		got := RunGen2Startup(Gen2StartupInput{}, startupTestHooks(nil, nil, scopedErr, statusErr, state))
		if !errors.Is(got.Err, scopedErr) || !errors.Is(got.StatusWriteError, statusErr) || !state.released || !state.wroteStatus {
			t.Fatalf("result = %+v; state = %+v", got, state)
		}
	})
}

type startupTestHookState struct {
	released    bool
	wroteStatus bool
}

func startupTestHooks(verdict *NegotiationVerdict, negotiateErr, scopedErr, statusErr error, state *startupTestHookState) Gen2StartupHooks {
	return Gen2StartupHooks{
		AcquireSingleInstance: func() (bool, func()) { return true, func() { state.released = true } },
		WaitForDriver:         func() {},
		RunScoped: func(fn func(HardwareBus) error) error {
			if scopedErr != nil {
				return scopedErr
			}
			return fn(nil)
		},
		Negotiate:   func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error) { return verdict, negotiateErr },
		WriteStatus: func(StatusContract) error { state.wroteStatus = true; return statusErr },
	}
}

func TestRunGen2StartupRequiresHooks(t *testing.T) {
	if got := RunGen2Startup(Gen2StartupInput{}, Gen2StartupHooks{}); !errors.Is(got.Err, ErrInvalidGen2StartupHooks) {
		t.Fatalf("error = %v", got.Err)
	}
}

func TestRunGen2StartupNegotiationErrorStillWritesVerdictStatus(t *testing.T) {
	negErr := errors.New("negotiation failed")
	var written StatusContract
	got := RunGen2Startup(Gen2StartupInput{}, Gen2StartupHooks{
		AcquireSingleInstance: func() (bool, func()) { return true, func() {} },
		WaitForDriver:         func() {},
		RunScoped:             func(fn func(HardwareBus) error) error { return fn(nil) },
		Negotiate: func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error) {
			return &NegotiationVerdict{StatusContract: StatusContract{ErrorCode: "NEGOTIATION_ERROR"}}, negErr
		},
		WriteStatus: func(status StatusContract) error { written = status; return nil },
	})
	if !errors.Is(got.Err, negErr) || written.ErrorCode != "NEGOTIATION_ERROR" {
		t.Fatalf("result = %+v; written = %+v", got, written)
	}
}

func TestRunGen2StartupScopedErrorWritesFailureStatus(t *testing.T) {
	scopedErr := errors.New("driver unavailable")
	var written StatusContract
	got := RunGen2Startup(Gen2StartupInput{}, Gen2StartupHooks{
		AcquireSingleInstance: func() (bool, func()) { return true, func() {} },
		WaitForDriver:         func() {},
		RunScoped:             func(func(HardwareBus) error) error { return scopedErr },
		Negotiate:             func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error) { return nil, nil },
		WriteStatus:           func(status StatusContract) error { written = status; return nil },
	})
	if !errors.Is(got.Err, scopedErr) || written.StatusCode != StatusDrvFail {
		t.Fatalf("result = %+v; written = %+v", got, written)
	}
}

func TestRunGen2StartupCleanupOnScopedError(t *testing.T) {
	state := &startupTestHookState{}
	errSession := errors.New("scoped")
	hooks := startupTestHooks(nil, nil, errSession, nil, state)
	got := RunGen2Startup(Gen2StartupInput{}, hooks)
	if !errors.Is(got.Err, errSession) || !state.released {
		t.Fatalf("result = %+v; state = %+v", got, state)
	}
}

func TestRunGen2StartupStatusErrorOnSkippedInstance(t *testing.T) {
	statusErr := errors.New("status")
	got := RunGen2Startup(Gen2StartupInput{}, Gen2StartupHooks{
		AcquireSingleInstance: func() (bool, func()) { return false, nil },
		WaitForDriver:         func() {},
		RunScoped:             func(func(HardwareBus) error) error { return nil },
		Negotiate:             func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error) { return nil, nil },
		WriteStatus:           func(StatusContract) error { return statusErr },
	})
	if !got.Skipped || !errors.Is(got.StatusWriteError, statusErr) {
		t.Fatalf("result = %+v", got)
	}
}

func TestRunGen2StartupRetryDispositionDoesNotRunWhenAbsent(t *testing.T) {
	called := false
	RunGen2Startup(Gen2StartupInput{}, Gen2StartupHooks{
		AcquireSingleInstance: func() (bool, func()) { return true, func() {} },
		WaitForDriver:         func() {},
		RunScoped:             func(fn func(HardwareBus) error) error { return fn(nil) },
		Negotiate:             func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error) { return &NegotiationVerdict{}, nil },
		WriteStatus:           func(StatusContract) error { return nil },
		RetryDisposition:      func(*NegotiationVerdict) Gen2RetryDisposition { called = true; return Gen2RetryNone },
		ApplyRetry:            func(Gen2RetryDisposition, int) { t.Fatal("unexpected retry") },
	})
	if !called {
		t.Fatal("retry disposition was not evaluated")
	}
}
