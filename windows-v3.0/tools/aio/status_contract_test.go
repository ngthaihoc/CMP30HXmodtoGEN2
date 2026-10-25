package main

import (
	"os"
	"testing"

	hxcore "40hxcore"
)

// Kiểm tra giao diện mà không mount ESP, nạp hoặc gỡ driver thật.
func TestMain(m *testing.M) {
	cleanupKernelDrivers = func() {}
	findGPUProfile = func() (hxcore.GPUProfile, bool) { return hxcore.GPUProfile{}, false }
	mountESP = func() string { return "" }
	os.Exit(m.Run())
}

func TestAppTelemetryUsesStatusContractWithoutGuessedLanes(t *testing.T) {
	old := findGPUProfile
	findGPUProfile = func() (hxcore.GPUProfile, bool) {
		prof, _ := hxcore.LookupGPUProfile(0x10DE, 0x2189)
		return prof, true
	}
	t.Cleanup(func() { findGPUProfile = old })
	t.Setenv("ProgramData", t.TempDir())
	app := NewApp()
	data := app.gatherTelemetry()
	if data.LinkWidth != 0 || data.ThroughputMBs != 0 || data.MRRS != 0 || data.IsGen2 {
		t.Fatalf("Thiếu trạng thái phải giữ dữ liệu chưa biết: %+v", data)
	}
	if err := hxcore.WriteStructuredGen2Status(hxcore.StatusContract{
		StatusCode: hxcore.StatusGen2Success, DeviceID: 0x2189, TLSTarget: 2,
		SpeedCurrent: 1, WidthCurrent: 4, MRRS: 512,
	}); err != nil {
		t.Fatal(err)
	}
	data = app.gatherTelemetry()
	if !data.IsGen2 || data.LinkSpeed != 1 || data.LinkWidth != 4 || data.ThroughputMBs != 1000 || !data.Lanes[3] || data.Lanes[4] {
		t.Fatalf("Telemetry phải giữ cả cấu hình và quan sát: %+v", data)
	}
	if report := app.ScanEnvironment(); !report.IsGen2 {
		t.Fatalf("Quét môi trường phải cùng cách hiểu: %+v", report)
	}
}
