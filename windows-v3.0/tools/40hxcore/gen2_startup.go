package hxcore

import "errors"

var ErrInvalidGen2StartupHooks = errors.New("incomplete Gen2 startup hooks")

// Gen2StartupInput carries CLI/AIO policy into the shared startup pipeline.
type Gen2StartupInput struct {
	TargetGen   uint32
	ForceRoot   bool
	AllowStage2 bool
	UserIsAdmin bool
	PrepareGSP  bool
	RetryDepth  int
}

type Gen2RetryDisposition uint8

const (
	Gen2RetryNone Gen2RetryDisposition = iota
	Gen2RetryClear
	Gen2RetrySchedule
)

type Gen2StartupHooks struct {
	AcquireSingleInstance func() (bool, func())
	WaitForDriver         func()
	RunScoped             func(func(HardwareBus) error) error
	Negotiate             func(HardwareBus, Gen2StartupInput) (*NegotiationVerdict, error)
	WriteStatus           func(StatusContract) error
	RetryDisposition      func(*NegotiationVerdict) Gen2RetryDisposition
	ApplyRetry            func(Gen2RetryDisposition, int)
}

type Gen2StartupResult struct {
	Verdict          *NegotiationVerdict
	NegotiationError error
	StatusWriteError error
	RetryDisposition Gen2RetryDisposition
	Skipped          bool
	Err              error
}

// RunGen2Startup owns startup ordering while platform policy remains injected.
func RunGen2Startup(input Gen2StartupInput, hooks Gen2StartupHooks) Gen2StartupResult {
	if hooks.AcquireSingleInstance == nil || hooks.WaitForDriver == nil || hooks.RunScoped == nil ||
		hooks.Negotiate == nil || hooks.WriteStatus == nil {
		return Gen2StartupResult{Err: ErrInvalidGen2StartupHooks}
	}

	owned, release := hooks.AcquireSingleInstance()
	if !owned {
		result := Gen2StartupResult{Skipped: true}
		result.StatusWriteError = hooks.WriteStatus(StatusContract{
			StatusCode: StatusGen2Skipped,
			ErrorCode:  "ANOTHER_INSTANCE_RUNNING",
			Details:    []string{"Bỏ qua: Một tiến trình Gen2 khác đang chạy (bảo vệ đơn phiên)"},
		})
		return result
	}
	if release != nil {
		defer release()
	}

	hooks.WaitForDriver()
	var verdict *NegotiationVerdict
	var negotiationErr error
	scopedErr := hooks.RunScoped(func(bus HardwareBus) error {
		verdict, negotiationErr = hooks.Negotiate(bus, input)
		return nil
	})

	result := Gen2StartupResult{Verdict: verdict, NegotiationError: negotiationErr, Err: negotiationErr}
	if scopedErr != nil {
		result.Err = scopedErr
	}
	if scopedErr != nil {
		if verdict == nil {
			result.StatusWriteError = hooks.WriteStatus(StatusContract{
				StatusCode: StatusDrvFail,
				ErrorCode:  "DRIVER_SESSION_FAILED",
				Details:    []string{"Không thể khởi tạo phiên driver: " + scopedErr.Error()},
			})
			return result
		}
	}
	if verdict == nil {
		return result
	}
	if hooks.RetryDisposition != nil {
		result.RetryDisposition = hooks.RetryDisposition(verdict)
		if hooks.ApplyRetry != nil && result.RetryDisposition != Gen2RetryNone {
			hooks.ApplyRetry(result.RetryDisposition, input.RetryDepth)
		}
	}
	result.StatusWriteError = hooks.WriteStatus(verdict.StatusContract)
	return result
}
