package hxcore

import (
	"strings"
	"testing"
)

func TestStatusContract_FormatAndParse(t *testing.T) {
	// Arrange
	contract := StatusContract{
		StatusCode:   StatusGen2Success,
		SpeedCurrent: 2,
		WidthCurrent: 16,
		TLSTarget:    2,
		ErrorCode:    "NONE",
		Details: []string{
			"GPU: NVIDIA CMP 30HX [TU116] [DEV_2189]",
			"Ket qua: da dat muc tieu Gen2 thanh cong",
		},
	}

	// Act
	formatted := FormatStatus(contract)
	parsed := ParseStatus(formatted)

	// Assert
	if parsed.StatusCode != StatusGen2Success {
		t.Fatalf("expected StatusGen2Success, got %s", parsed.StatusCode)
	}
	if parsed.SpeedCurrent != 2 {
		t.Fatalf("expected SpeedCurrent=2, got %d", parsed.SpeedCurrent)
	}
	if parsed.WidthCurrent != 16 {
		t.Fatalf("expected WidthCurrent=16, got %d", parsed.WidthCurrent)
	}
	if parsed.TLSTarget != 2 {
		t.Fatalf("expected TLSTarget=2, got %d", parsed.TLSTarget)
	}
	if parsed.ErrorCode != "NONE" {
		t.Fatalf("expected ErrorCode=NONE, got %s", parsed.ErrorCode)
	}
	if !strings.Contains(formatted, "STATUS_CODE=GEN2_SUCCESS") {
		t.Fatalf("expected formatted string to contain STATUS_CODE=GEN2_SUCCESS")
	}
}

func TestStatusContract_ParseLegacyFallback(t *testing.T) {
	// Test fallback when STATUS_CODE header is absent
	legacySuccess := "==== 40HX Gen2 Ket qua ====\nGPU: CMP 30HX\nKet qua: da dat muc tieu Gen2 thanh cong\n"
	parsed := ParseStatus(legacySuccess)
	if parsed.StatusCode != StatusGen2Success {
		t.Fatalf("expected fallback to StatusGen2Success, got %s", parsed.StatusCode)
	}

	legacyGen1 := "==== 40HX Gen2 Ket qua ====\nGPU: CMP 30HX\nTrang thai: chua dat muc tieu Gen2\nGPU TLS=Gen1\n"
	parsedGen1 := ParseStatus(legacyGen1)
	if parsedGen1.StatusCode != StatusGen1Stuck {
		t.Fatalf("expected fallback to StatusGen1Stuck, got %s", parsedGen1.StatusCode)
	}

	legacyDrv := "==== 40HX Gen2 Ket qua ====\nWinRing0 driver bi chan [Loi 5 / ERROR_ACCESS_DENIED]\n"
	parsedDrv := ParseStatus(legacyDrv)
	if parsedDrv.StatusCode != StatusDrvFail {
		t.Fatalf("expected fallback to StatusDrvFail, got %s", parsedDrv.StatusCode)
	}

	legacyNoGpu := "==== 40HX Gen2 Ket qua ====\nKhong tim thay thiet bi tren bus PCI\n"
	parsedNoGpu := ParseStatus(legacyNoGpu)
	if parsedNoGpu.StatusCode != StatusNoGPU {
		t.Fatalf("expected fallback to StatusNoGPU, got %s", parsedNoGpu.StatusCode)
	}
}

func TestStatusContract_SkippedAndHwLimitCodes(t *testing.T) {
	skipped := StatusContract{
		StatusCode: StatusGen2Skipped,
		ErrorCode:  "ANOTHER_INSTANCE_RUNNING",
		Details:    []string{"Skipped instance"},
	}
	hwLimit := StatusContract{
		StatusCode: StatusHwLimit,
		ErrorCode:  "EFUSE_LOCKED_GEN2",
		Details:    []string{"Hardware limit reached"},
	}

	fmtSkipped := FormatStatus(skipped)
	parsedSkipped := ParseStatus(fmtSkipped)

	fmtHwLimit := FormatStatus(hwLimit)
	parsedHwLimit := ParseStatus(fmtHwLimit)

	if parsedSkipped.StatusCode != StatusGen2Skipped || parsedSkipped.ErrorCode != "ANOTHER_INSTANCE_RUNNING" {
		t.Fatalf("unexpected skipped parsing: %+v", parsedSkipped)
	}
	if parsedHwLimit.StatusCode != StatusHwLimit || parsedHwLimit.ErrorCode != "EFUSE_LOCKED_GEN2" {
		t.Fatalf("unexpected hwLimit parsing: %+v", parsedHwLimit)
	}
}
