package hxcore

import (
	"testing"
)

func TestComputeInspector_Unlocked(t *testing.T) {
	// Arrange: CMP 40HX (TU106) with SS0 == 0x88888888 (Tensor Core Unlocked)
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:       0x10DE,
		DeviceID:       0x1F0B,
		Name:           "CMP 40HX",
		Family:         "TU106",
		FirmwareUnlock: true,
	}
	bar0 := uint64(0xF6000000)
	bus.SetPCIConfig(bdf, 0x10, uint32(bar0))
	bus.SetMMIO(bar0+0x00, 0x166000A1)     // BOOT_0: TU106 (0x16)
	bus.SetMMIO(bar0+SS0Offset, 0x88888888) // SS0: Unlocked
	bus.SetMMIO(bar0+SS1Offset, 0x12345678) // SS1

	inspector := NewComputeInspector(bus)

	// Act
	report, err := inspector.InspectCompute(bdf, prof)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Status != ComputeStatusUnlocked {
		t.Fatalf("expected ComputeStatusUnlocked, got %v", report.Status)
	}
	if !report.Unlocked || report.SS0 != 0x88888888 {
		t.Fatalf("expected Unlocked=true, SS0=0x88888888, got %+v", report)
	}
}

func TestComputeInspector_Locked(t *testing.T) {
	// Arrange: CMP 40HX (TU106) with SS0 != 0x88888888
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:       0x10DE,
		DeviceID:       0x1F0B,
		Name:           "CMP 40HX",
		Family:         "TU106",
		FirmwareUnlock: true,
	}
	bar0 := uint64(0xF6000000)
	bus.SetPCIConfig(bdf, 0x10, uint32(bar0))
	bus.SetMMIO(bar0+0x00, 0x166000A1)     // BOOT_0: TU106 (0x16)
	bus.SetMMIO(bar0+SS0Offset, 0x00000000) // SS0: Locked

	inspector := NewComputeInspector(bus)

	// Act
	report, err := inspector.InspectCompute(bdf, prof)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Status != ComputeStatusLocked || report.Unlocked {
		t.Fatalf("expected ComputeStatusLocked and Unlocked=false, got %+v", report)
	}
}

func TestComputeInspector_FamilyMismatch(t *testing.T) {
	// Arrange: BOOT_0 byte does not match TU10x family (0x16)
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:       0x10DE,
		DeviceID:       0x1F0B,
		Name:           "CMP 40HX",
		Family:         "TU106",
		FirmwareUnlock: true,
	}
	bar0 := uint64(0xF6000000)
	bus.SetPCIConfig(bdf, 0x10, uint32(bar0))
	bus.SetMMIO(bar0+0x00, 0x99000000) // Invalid family

	inspector := NewComputeInspector(bus)

	// Act
	report, err := inspector.InspectCompute(bdf, prof)

	// Assert
	if err == nil {
		t.Fatalf("expected error on family mismatch, got nil (report: %+v)", report)
	}
	if report.Status != ComputeStatusFamilyMismatch {
		t.Fatalf("expected ComputeStatusFamilyMismatch, got %v", report.Status)
	}
}

func TestComputeInspector_TU116_NotApplicable(t *testing.T) {
	// Arrange: CMP 30HX (TU116) does not have firmware/tensor unlock
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:       0x10DE,
		DeviceID:       0x2189,
		Name:           "CMP 30HX",
		Family:         "TU116",
		FirmwareUnlock: false,
	}

	inspector := NewComputeInspector(bus)

	// Act
	report, err := inspector.InspectCompute(bdf, prof)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Status != ComputeStatusNotApplicable {
		t.Fatalf("expected ComputeStatusNotApplicable, got %v", report.Status)
	}
}
