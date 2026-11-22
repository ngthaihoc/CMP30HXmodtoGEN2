package main

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

type mockCudaDriver struct {
	initErr       error
	lookupErr     error
	registerErr   error
	driverVer     int
	deviceName    string
	ccMajor       int
	ccMinor       int
	corruptOut    bool
	nonFiniteOut  bool
	copyDelay     time.Duration
	nextDevPtr   uint64
	devMem       map[uint64][]byte
	lastDA       uint64
	lastDB       uint64
	lastDOut     uint64
	lastAlpha    float32
	lastN        uint32
	destroyedCtx bool
	freedCount   int
}

func newMockCudaDriver() *mockCudaDriver {
	return &mockCudaDriver{
		driverVer:  12040,
		deviceName: "NVIDIA CMP 40HX",
		ccMajor:    7,
		ccMinor:    5,
		nextDevPtr: 0x1000,
		devMem:     make(map[uint64][]byte),
	}
}

func (m *mockCudaDriver) Init() error {
	return m.initErr
}

func (m *mockCudaDriver) DriverVersion() (int, error) {
	return m.driverVer, nil
}

func (m *mockCudaDriver) DeviceByPCIBusID(bdf string) (int32, error) {
	if m.lookupErr != nil {
		return 0, m.lookupErr
	}
	return 0, nil
}

func (m *mockCudaDriver) DeviceName(dev int32) (string, error) {
	return m.deviceName, nil
}

func (m *mockCudaDriver) ComputeCapability(dev int32) (int, int, error) {
	return m.ccMajor, m.ccMinor, nil
}

func (m *mockCudaDriver) CreateContext(dev int32) (uintptr, error) {
	return 0xBEEF, nil
}

func (m *mockCudaDriver) DestroyContext(ctx uintptr) error {
	m.destroyedCtx = true
	return nil
}

func (m *mockCudaDriver) RegisterHost(buf []byte) error {
	return m.registerErr
}

func (m *mockCudaDriver) UnregisterHost(buf []byte) error {
	return nil
}

func (m *mockCudaDriver) AllocDevice(bytes uintptr) (uint64, error) {
	ptr := m.nextDevPtr
	m.nextDevPtr += uint64(bytes) + 4096
	m.devMem[ptr] = make([]byte, bytes)
	return ptr, nil
}

func (m *mockCudaDriver) FreeDevice(dptr uint64) error {
	delete(m.devMem, dptr)
	m.freedCount++
	return nil
}

func (m *mockCudaDriver) MemcpyHtoD(dst uint64, src []byte) error {
	if m.copyDelay > 0 {
		time.Sleep(m.copyDelay)
	}
	buf, ok := m.devMem[dst]
	if !ok {
		return errors.New("invalid device pointer")
	}
	copy(buf, src)
	return nil
}

func (m *mockCudaDriver) MemcpyDtoH(dst []byte, src uint64) error {
	if m.copyDelay > 0 {
		time.Sleep(m.copyDelay)
	}
	buf, ok := m.devMem[src]
	if !ok {
		return errors.New("invalid device pointer")
	}
	copy(dst, buf)
	return nil
}

func (m *mockCudaDriver) Synchronize() error {
	return nil
}

func (m *mockCudaDriver) LoadModule(ptx string) (uintptr, error) {
	if !strings.Contains(ptx, "fp32_saxpy") {
		return 0, errors.New("missing kernel symbol")
	}
	return 0xCAFE, nil
}

func (m *mockCudaDriver) UnloadModule(mod uintptr) error {
	return nil
}

func (m *mockCudaDriver) GetFunction(mod uintptr, name string) (uintptr, error) {
	return 0xF00D, nil
}

func (m *mockCudaDriver) LaunchSAXPY(fn uintptr, n uint32, dA, dB, dOut uint64, alpha float32) error {
	m.lastN = n
	m.lastDA = dA
	m.lastDB = dB
	m.lastDOut = dOut
	m.lastAlpha = alpha

	aBuf := m.devMem[dA]
	bBuf := m.devMem[dB]
	outVals := make([]float32, n)
	for i := 0; i < int(n); i++ {
		a := decodeFloat32At(aBuf, i)
		b := decodeFloat32At(bBuf, i)
		if m.nonFiniteOut && i == 50 {
			outVals[i] = float32(math.NaN())
		} else if m.corruptOut && i == 100 {
			outVals[i] = 99999.0
		} else {
			outVals[i] = alpha*a + b
		}
	}
	copy(m.devMem[dOut], encodeFloat32Slice(outVals))
	return nil
}

func TestRunBenchmark_SuccessPinnedAndFP32Correctness(t *testing.T) {
	drv := newMockCudaDriver()
	drv.copyDelay = 2 * time.Millisecond

	res := RunBenchmark(drv, BenchConfig{
		BDF:        "0000:02:00.0",
		SizeMB:     4,
		Iterations: 3,
		TimeoutMs:  10000,
	})

	if res.Status != "ok" {
		t.Fatalf("expected status=ok, got %s (reason=%s)", res.Status, res.Reason)
	}
	if !res.PinnedHost || !res.FP32KernelOK {
		t.Fatalf("expected PinnedHost=true and FP32KernelOK=true, got %+v", res)
	}
	if res.DeviceName != "NVIDIA CMP 40HX" || res.ComputeCapability != "7.5" {
		t.Fatalf("unexpected device info: %+v", res)
	}
	if res.H2DBandwidthGBps <= 0 || res.D2HBandwidthGBps <= 0 {
		t.Fatalf("expected positive H2D/D2H bandwidth, got %+v", res)
	}
	if !drv.destroyedCtx || drv.freedCount != 4 {
		t.Fatalf("expected context destroyed and 4 buffers freed, got ctx=%v freed=%d", drv.destroyedCtx, drv.freedCount)
	}
}

func TestRunBenchmark_CudaUnavailableReturnsUnsupported(t *testing.T) {
	drv := newMockCudaDriver()
	drv.initErr = ErrCudaUnavailable

	res := RunBenchmark(drv, BenchConfig{BDF: "0000:01:00.0"})
	if res.Status != "unsupported" {
		t.Fatalf("expected unsupported, got %s", res.Status)
	}
}

func TestRunBenchmark_MissingBDFOrLookupError(t *testing.T) {
	drv := newMockCudaDriver()
	resEmpty := RunBenchmark(drv, BenchConfig{BDF: ""})
	if resEmpty.Status != "error" || resEmpty.Reason != "missing_target_bdf" {
		t.Fatalf("unexpected result for empty BDF: %+v", resEmpty)
	}

	drv.lookupErr = errors.New("no device at BDF")
	resLookup := RunBenchmark(drv, BenchConfig{BDF: "0000:99:00.0"})
	if resLookup.Status != "error" || !strings.Contains(resLookup.Reason, "device_lookup_failed") {
		t.Fatalf("unexpected result for lookup failure: %+v", resLookup)
	}
}

func TestRunBenchmark_FP32MismatchFailsClosed(t *testing.T) {
	drv := newMockCudaDriver()
	drv.corruptOut = true

	res := RunBenchmark(drv, BenchConfig{BDF: "0000:01:00.0", SizeMB: 4, Iterations: 1})
	if res.Status != "error" || res.FP32KernelOK {
		t.Fatalf("expected error with FP32KernelOK=false, got %+v", res)
	}
	if !strings.Contains(res.Reason, "fp32_mismatch") {
		t.Fatalf("expected fp32_mismatch reason, got %s", res.Reason)
	}
}

func TestRunBenchmark_WatchdogTimeoutStopsLoad(t *testing.T) {
	drv := newMockCudaDriver()
	drv.copyDelay = 300 * time.Millisecond

	res := RunBenchmark(drv, BenchConfig{
		BDF:        "0000:01:00.0",
		SizeMB:     4,
		Iterations: 10,
		TimeoutMs:  1000,
	})
	if res.Status != "timeout" {
		t.Fatalf("expected timeout, got %s (reason=%s)", res.Status, res.Reason)
	}
}

func TestRunBenchmark_FP32NonFiniteFailsClosed(t *testing.T) {
	drv := newMockCudaDriver()
	drv.nonFiniteOut = true

	res := RunBenchmark(drv, BenchConfig{BDF: "0000:01:00.0", SizeMB: 4, Iterations: 1})
	if res.Status != "error" || res.FP32KernelOK {
		t.Fatalf("expected error with FP32KernelOK=false on NaN, got %+v", res)
	}
	if !strings.Contains(res.Reason, "fp32_non_finite") {
		t.Fatalf("expected fp32_non_finite reason, got %s", res.Reason)
	}
}

func TestRunBenchmark_PinnedAllocationFailureFallsBackExplicitly(t *testing.T) {
	drv := newMockCudaDriver()
	drv.registerErr = errors.New("page lock failed")

	res := RunBenchmark(drv, BenchConfig{BDF: "01:00.0", SizeMB: 4, Iterations: 2})
	if res.Status != "ok" {
		t.Fatalf("expected ok fallback with pageable buffer, got %+v", res)
	}
	if res.PinnedHost {
		t.Fatalf("expected PinnedHost=false on RegisterHost error")
	}
	if res.BDF != "0000:01:00.0" {
		t.Fatalf("expected normalized BDF 0000:01:00.0, got %s", res.BDF)
	}
	if !strings.Contains(res.Reason, "pinned_host_allocation_failed") {
		t.Fatalf("expected explicit pinned fallback reason, got %q", res.Reason)
	}
}

func TestRunBenchmark_InvalidBDFRejected(t *testing.T) {
	drv := newMockCudaDriver()
	res := RunBenchmark(drv, BenchConfig{BDF: "not-a-bdf", SizeMB: 4, Iterations: 1})
	if res.Status != "error" || !strings.Contains(res.Reason, "invalid_target_bdf") {
		t.Fatalf("expected invalid_target_bdf error, got %+v", res)
	}
}

func TestSystemCudaDLLPath_RestrictedToSystem32(t *testing.T) {
	p, err := systemCudaDLLPath(`C:\Windows\System32`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.EqualFold(p, `C:\Windows\System32\nvcuda.dll`) {
		t.Fatalf("unexpected system path: %s", p)
	}
	if _, err := systemCudaDLLPath(""); err == nil {
		t.Fatalf("expected error for empty system dir")
	}
}
