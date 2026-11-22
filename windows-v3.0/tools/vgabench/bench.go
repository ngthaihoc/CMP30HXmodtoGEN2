package main

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ErrCudaUnavailable is returned when nvcuda.dll cannot be loaded on the host.
var ErrCudaUnavailable = errors.New("nvcuda.dll unavailable")

var validBdfRe = regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]$`)
var shortBdfRe = regexp.MustCompile(`^[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]$`)

// BenchConfig configures a user-mode CUDA PCIe bandwidth and FP32 correctness check.
type BenchConfig struct {
	BDF        string
	SizeMB     int
	Iterations int
	TimeoutMs  int
}

// BenchResult holds the allowlisted, zero-PII benchmark metrics for reporting.
type BenchResult struct {
	Status            string  `json:"status"`
	BDF               string  `json:"bdf"`
	DeviceName        string  `json:"device_name,omitempty"`
	ComputeCapability string  `json:"compute_capability,omitempty"`
	DriverVersion     int     `json:"driver_version,omitempty"`
	BenchmarkMethod   string  `json:"benchmark_method,omitempty"`
	WarmupIncluded    bool    `json:"warmup_included"`
	TransferSizeMB    int     `json:"transfer_size_mb"`
	Iterations        int     `json:"iterations"`
	PinnedHost        bool    `json:"pinned_host"`
	H2DBandwidthGBps  float64 `json:"h2d_bandwidth_gbps"`
	H2DMinGBps        float64 `json:"h2d_min_gbps"`
	H2DMaxGBps        float64 `json:"h2d_max_gbps"`
	D2HBandwidthGBps  float64 `json:"d2h_bandwidth_gbps"`
	D2HMinGBps        float64 `json:"d2h_min_gbps"`
	D2HMaxGBps        float64 `json:"d2h_max_gbps"`
	FP32KernelOK      bool    `json:"fp32_kernel_ok"`
	FP32GFLOPS        float64 `json:"fp32_gflops"`
	DurationMs        int64   `json:"duration_ms"`
	Reason            string  `json:"reason,omitempty"`
}

// CudaDriver abstracts CUDA Driver API calls so tests can mock GPU execution without hardware.
type CudaDriver interface {
	Init() error
	DriverVersion() (int, error)
	DeviceByPCIBusID(bdf string) (int32, error)
	DeviceName(dev int32) (string, error)
	ComputeCapability(dev int32) (int, int, error)
	CreateContext(dev int32) (uintptr, error)
	DestroyContext(ctx uintptr) error
	RegisterHost(buf []byte) error
	UnregisterHost(buf []byte) error
	AllocDevice(bytes uintptr) (uint64, error)
	FreeDevice(dptr uint64) error
	MemcpyHtoD(dst uint64, src []byte) error
	MemcpyDtoH(dst []byte, src uint64) error
	Synchronize() error
	LoadModule(ptx string) (uintptr, error)
	UnloadModule(mod uintptr) error
	GetFunction(mod uintptr, name string) (uintptr, error)
	LaunchSAXPY(fn uintptr, n uint32, dA, dB, dOut uint64, alpha float32) error
}

const fp32SaxpyPTX = `.version 6.0
.target sm_50
.address_size 64

.visible .entry fp32_saxpy(
    .param .u64 param_a,
    .param .u64 param_b,
    .param .u64 param_out,
    .param .f32 param_alpha,
    .param .u32 param_n
)
{
    .reg .pred %p<2>;
    .reg .b32 %r<5>;
    .reg .f32 %f<5>;
    .reg .b64 %rd<7>;

    ld.param.u64 %rd1, [param_a];
    ld.param.u64 %rd2, [param_b];
    ld.param.u64 %rd3, [param_out];
    ld.param.f32 %f1, [param_alpha];
    ld.param.u32 %r1, [param_n];

    mov.u32 %r2, %ctaid.x;
    mov.u32 %r3, %ntid.x;
    mov.u32 %r4, %tid.x;
    mad.lo.u32 %r2, %r2, %r3, %r4;
    setp.ge.u32 %p1, %r2, %r1;
    @%p1 bra DONE;

    cvta.to.global.u64 %rd1, %rd1;
    cvta.to.global.u64 %rd2, %rd2;
    cvta.to.global.u64 %rd3, %rd3;
    mul.wide.u32 %rd4, %r2, 4;
    add.s64 %rd5, %rd1, %rd4;
    add.s64 %rd6, %rd2, %rd4;
    ld.global.f32 %f2, [%rd5];
    ld.global.f32 %f3, [%rd6];
    fma.rn.f32 %f4, %f1, %f2, %f3;
    add.s64 %rd5, %rd3, %rd4;
    st.global.f32 [%rd5], %f4;

DONE:
    ret;
}
`

func normalizeConfig(cfg BenchConfig) BenchConfig {
	cfg.BDF = strings.TrimSpace(cfg.BDF)
	if cfg.SizeMB <= 0 {
		cfg.SizeMB = 64
	} else if cfg.SizeMB < 4 {
		cfg.SizeMB = 4
	} else if cfg.SizeMB > 512 {
		cfg.SizeMB = 512
	}
	if cfg.Iterations <= 0 {
		cfg.Iterations = 5
	} else if cfg.Iterations > 50 {
		cfg.Iterations = 50
	}
	if cfg.TimeoutMs <= 0 {
		cfg.TimeoutMs = 15000
	} else if cfg.TimeoutMs < 1000 {
		cfg.TimeoutMs = 1000
	} else if cfg.TimeoutMs > 60000 {
		cfg.TimeoutMs = 60000
	}
	return cfg
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

func normalizeAndValidateBDF(raw string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(raw))
	if t == "" {
		return "", errors.New("missing_target_bdf")
	}
	if strings.HasPrefix(t, "0000") && len(t) == 16 && validBdfRe.MatchString(t[4:]) {
		return t[4:], nil
	}
	if shortBdfRe.MatchString(t) {
		return "0000:" + t, nil
	}
	if validBdfRe.MatchString(t) {
		return t, nil
	}
	return "", fmt.Errorf("invalid_target_bdf: %s", strings.TrimSpace(raw))
}

func minMaxFloat64(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return lo, hi
}

func medianFloat64(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]float64(nil), vals...)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2.0
}

func encodeFloat32Slice(vals []float32) []byte {
	out := make([]byte, len(vals)*4)
	for i, v := range vals {
		bits := math.Float32bits(v)
		off := i * 4
		out[off] = byte(bits)
		out[off+1] = byte(bits >> 8)
		out[off+2] = byte(bits >> 16)
		out[off+3] = byte(bits >> 24)
	}
	return out
}

func decodeFloat32At(buf []byte, idx int) float32 {
	off := idx * 4
	bits := uint32(buf[off]) | uint32(buf[off+1])<<8 | uint32(buf[off+2])<<16 | uint32(buf[off+3])<<24
	return math.Float32frombits(bits)
}

// RunBenchmark executes an isolated user-mode PCIe H2D/D2H copy test and FP32 PTX kernel check.
func RunBenchmark(drv CudaDriver, cfg BenchConfig) BenchResult {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cfg = normalizeConfig(cfg)
	start := time.Now()
	deadline := start.Add(time.Duration(cfg.TimeoutMs) * time.Millisecond)

	res := BenchResult{
		Status:         "error",
		BDF:            cfg.BDF,
		TransferSizeMB: cfg.SizeMB,
		Iterations:     cfg.Iterations,
	}

	normBDF, bdfErr := normalizeAndValidateBDF(cfg.BDF)
	if bdfErr != nil {
		res.Reason = bdfErr.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	cfg.BDF = normBDF
	res.BDF = normBDF

	if err := drv.Init(); err != nil {
		if errors.Is(err, ErrCudaUnavailable) {
			res.Status = "unsupported"
		}
		res.Reason = err.Error()
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	drvVer, err := drv.DriverVersion()
	if err == nil {
		res.DriverVersion = drvVer
	}

	dev, err := drv.DeviceByPCIBusID(cfg.BDF)
	if err != nil {
		res.Reason = fmt.Sprintf("device_lookup_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	if name, err := drv.DeviceName(dev); err == nil {
		res.DeviceName = strings.TrimSpace(name)
	}
	if maj, min, err := drv.ComputeCapability(dev); err == nil {
		res.ComputeCapability = fmt.Sprintf("%d.%d", maj, min)
	}

	ctx, err := drv.CreateContext(dev)
	if err != nil {
		res.Reason = fmt.Sprintf("context_create_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = drv.DestroyContext(ctx) }()

	transferBytes := uintptr(cfg.SizeMB) * 1024 * 1024
	hostBuf := make([]byte, transferBytes)
	for i := 0; i < len(hostBuf); i += 4096 {
		hostBuf[i] = byte(i)
	}
	pinnedNote := ""
	if err := drv.RegisterHost(hostBuf); err == nil {
		res.PinnedHost = true
		res.BenchmarkMethod = "nvcuda_pinned_host"
		defer func() { _ = drv.UnregisterHost(hostBuf) }()
	} else {
		res.PinnedHost = false
		res.BenchmarkMethod = "nvcuda_pageable_host"
		pinnedNote = fmt.Sprintf("pinned_host_allocation_failed_fallback_pageable: %v", err)
	}

	devBuf, err := drv.AllocDevice(transferBytes)
	if err != nil {
		res.Reason = fmt.Sprintf("device_alloc_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = drv.FreeDevice(devBuf) }()

	// Warmup copy before timing.
	if err := drv.MemcpyHtoD(devBuf, hostBuf); err != nil {
		res.Reason = fmt.Sprintf("warmup_htod_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	if err := drv.MemcpyDtoH(hostBuf, devBuf); err != nil {
		res.Reason = fmt.Sprintf("warmup_dtoh_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	if err := drv.Synchronize(); err != nil {
		res.Reason = fmt.Sprintf("warmup_sync_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	h2dRates := make([]float64, 0, cfg.Iterations)
	d2hRates := make([]float64, 0, cfg.Iterations)
	for iter := 0; iter < cfg.Iterations; iter++ {
		if time.Now().After(deadline) {
			res.Status = "timeout"
			res.Reason = "watchdog_timeout_during_pcie_copy"
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}

		t0 := time.Now()
		if err := drv.MemcpyHtoD(devBuf, hostBuf); err != nil {
			res.Reason = fmt.Sprintf("htod_failed: %v", err)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		if err := drv.Synchronize(); err != nil {
			res.Reason = fmt.Sprintf("htod_sync_failed: %v", err)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		h2dElapsed := time.Since(t0).Seconds()
		if h2dElapsed > 0 {
			h2dRates = append(h2dRates, float64(transferBytes)/h2dElapsed/1e9)
		}

		t1 := time.Now()
		if err := drv.MemcpyDtoH(hostBuf, devBuf); err != nil {
			res.Reason = fmt.Sprintf("dtoh_failed: %v", err)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		if err := drv.Synchronize(); err != nil {
			res.Reason = fmt.Sprintf("dtoh_sync_failed: %v", err)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		d2hElapsed := time.Since(t1).Seconds()
		if d2hElapsed > 0 {
			d2hRates = append(d2hRates, float64(transferBytes)/d2hElapsed/1e9)
		}
	}

	res.WarmupIncluded = true
	res.H2DBandwidthGBps = round2(medianFloat64(h2dRates))
	h2dMin, h2dMax := minMaxFloat64(h2dRates)
	res.H2DMinGBps = round2(h2dMin)
	res.H2DMaxGBps = round2(h2dMax)
	res.D2HBandwidthGBps = round2(medianFloat64(d2hRates))
	d2hMin, d2hMax := minMaxFloat64(d2hRates)
	res.D2HMinGBps = round2(d2hMin)
	res.D2HMaxGBps = round2(d2hMax)

	// FP32 correctness and lightweight compute check using embedded PTX.
	const elemCount = 1024 * 1024
	const alpha float32 = 2.5
	aVals := make([]float32, elemCount)
	bVals := make([]float32, elemCount)
	for i := 0; i < elemCount; i++ {
		aVals[i] = float32((i%17)+1) * 0.5
		bVals[i] = float32((i%13)+1) * 0.25
	}
	aBytes := encodeFloat32Slice(aVals)
	bBytes := encodeFloat32Slice(bVals)
	outBytes := make([]byte, len(aBytes))
	vecBytes := uintptr(len(aBytes))

	dA, err := drv.AllocDevice(vecBytes)
	if err != nil {
		res.Reason = fmt.Sprintf("compute_alloc_a_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = drv.FreeDevice(dA) }()

	dB, err := drv.AllocDevice(vecBytes)
	if err != nil {
		res.Reason = fmt.Sprintf("compute_alloc_b_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = drv.FreeDevice(dB) }()

	dOut, err := drv.AllocDevice(vecBytes)
	if err != nil {
		res.Reason = fmt.Sprintf("compute_alloc_out_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = drv.FreeDevice(dOut) }()

	if err := drv.MemcpyHtoD(dA, aBytes); err != nil {
		res.Reason = fmt.Sprintf("compute_htod_a_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	if err := drv.MemcpyHtoD(dB, bBytes); err != nil {
		res.Reason = fmt.Sprintf("compute_htod_b_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	mod, err := drv.LoadModule(fp32SaxpyPTX)
	if err != nil {
		res.Reason = fmt.Sprintf("ptx_load_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	defer func() { _ = drv.UnloadModule(mod) }()

	fn, err := drv.GetFunction(mod, "fp32_saxpy")
	if err != nil {
		res.Reason = fmt.Sprintf("ptx_func_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	// Warmup launch.
	if err := drv.LaunchSAXPY(fn, elemCount, dA, dB, dOut, alpha); err != nil {
		res.Reason = fmt.Sprintf("kernel_warmup_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	if err := drv.Synchronize(); err != nil {
		res.Reason = fmt.Sprintf("kernel_warmup_sync_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	const kernelLaunches = 20
	tK := time.Now()
	for k := 0; k < kernelLaunches; k++ {
		if time.Now().After(deadline) {
			res.Status = "timeout"
			res.Reason = "watchdog_timeout_during_kernel"
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		if err := drv.LaunchSAXPY(fn, elemCount, dA, dB, dOut, alpha); err != nil {
			res.Reason = fmt.Sprintf("kernel_launch_failed: %v", err)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
	}
	if err := drv.Synchronize(); err != nil {
		res.Reason = fmt.Sprintf("kernel_sync_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}
	kernelSec := time.Since(tK).Seconds()
	if kernelSec > 0 {
		// 2 FLOPs per FMA element per launch.
		flops := float64(elemCount) * 2.0 * float64(kernelLaunches)
		res.FP32GFLOPS = round2(flops / kernelSec / 1e9)
	}

	if err := drv.MemcpyDtoH(outBytes, dOut); err != nil {
		res.Reason = fmt.Sprintf("compute_dtoh_out_failed: %v", err)
		res.DurationMs = time.Since(start).Milliseconds()
		return res
	}

	for i := 0; i < elemCount; i++ {
		got := decodeFloat32At(outBytes, i)
		got64 := float64(got)
		if math.IsNaN(got64) || math.IsInf(got64, 0) {
			res.FP32KernelOK = false
			res.Reason = fmt.Sprintf("fp32_non_finite_at_%d_got_%v", i, got)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
		want := alpha*aVals[i] + bVals[i]
		if math.Abs(got64-float64(want)) > 1e-4 {
			res.FP32KernelOK = false
			res.Reason = fmt.Sprintf("fp32_mismatch_at_%d_got_%v_want_%v", i, got, want)
			res.DurationMs = time.Since(start).Milliseconds()
			return res
		}
	}

	res.FP32KernelOK = true
	res.Status = "ok"
	if pinnedNote != "" {
		res.Reason = pinnedNote
	}
	res.DurationMs = time.Since(start).Milliseconds()
	return res
}
