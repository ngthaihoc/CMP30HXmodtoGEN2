package hxcore

import (
	"errors"
	"testing"
)

func TestLinkNegotiator_TU116_EFuseClamp(t *testing.T) {
	// Arrange: TU116 (CMP 30HX) has silicon eFuse bit 3 blown; must clamp target to Gen2
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		Name:            "CMP 30HX",
		Family:          "TU116",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x218910DE) // Ven/Dev
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x40+0x0C, 0x00000001) // LNKCAP: Gen1
	bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000011) // LNKSTA: Gen1 x1
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)      // BAR0

	bar0 := uint64(0xF6000000)
	bus.SetMMIO(bar0+0x00, 0x17000000) // BOOT_0: TU116

	negotiator := NewLinkNegotiator(bus)

	// Act: request Gen3 on CMP 30HX
	res, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 3, false)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TargetGen != 2 {
		t.Fatalf("expected TargetGen clamped to 2, got %d", res.TargetGen)
	}
}

func TestLinkNegotiator_DEVCTL_MRRS_512B_Optimization(t *testing.T) {
	// Arrange: DEVCTL MRRS at 128B (mrrs=0) must be upgraded to 512B (mrrs=2)
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		Name:            "CMP 30HX",
		Family:          "TU116",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x218910DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x40+0x08, 0x00000000) // DEVCTL: MRRS=128B
	bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000022) // LNKSTA: Gen2 x2
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x17000000)

	negotiator := NewLinkNegotiator(bus)

	// Act
	_, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert: DEVCTL bits [14:12] must be 2 (512B) -> 0x2000
	devctl, _ := bus.ReadPCIConfig(bdf, 0x40+0x08)
	mrrs := (devctl >> 12) & 0x7
	if mrrs != 2 {
		t.Fatalf("expected DEVCTL MRRS=2 (512B), got %d (raw=0x%04X)", mrrs, devctl)
	}
}

func TestLinkNegotiator_MMIO_ShadowRegisterSequence(t *testing.T) {
	// Arrange: verify single source of truth MMIO shadow sequence is executed
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		Name:            "CMP 30HX",
		Family:          "TU116",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x218910DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bar0 := uint64(0xF6000000)
	bus.SetMMIO(bar0+0x00, 0x17000000) // BOOT_0 TU116
	bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000021) // Gen2 attained

	negotiator := NewLinkNegotiator(bus)

	// Act
	res, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}

	// Assert MMIO writes
	privMisc, _ := bus.ReadMMIO(bar0 + 0x0008841C)
	if privMisc != 0xE0B42D00 {
		t.Errorf("PRIV_MISC_1 mismatch: got 0x%08X", privMisc)
	}
	xveOvr, _ := bus.ReadMMIO(bar0 + 0x0008872C)
	if xveOvr != 6 {
		t.Errorf("XVE_OVR mismatch: got 0x%08X", xveOvr)
	}
	linkCfg, _ := bus.ReadMMIO(bar0 + 0x0008C040)
	if linkCfg != 0x80085800 {
		t.Errorf("LINK_CONFIG_0 mismatch: got 0x%08X", linkCfg)
	}
	plLinkRate, _ := bus.ReadMMIO(bar0 + 0x0008C1C0)
	if plLinkRate != 0x00240036 {
		t.Errorf("PL_LINK_RATE mismatch: got 0x%08X", plLinkRate)
	}
	cya0, _ := bus.ReadMMIO(bar0 + 0x0008C2C0)
	if cya0 != 0x068731B3 {
		t.Errorf("CYA_0 mismatch: got 0x%08X", cya0)
	}
}

func TestLinkNegotiator_Stage2_PnPRecovery(t *testing.T) {
	// Arrange: Link starts at Gen1 and fails Stage 1 retrains;
	// When allowStage2=true, PnpResetDevice is called and link recovers to Gen2
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		Name:            "CMP 30HX",
		Family:          "TU116",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x218910DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x17000000)
	bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000011) // Stuck at Gen1

	// Hook PnP reset to flip link speed to Gen2
	bus.OnPnpReset = func(devID uint16) bool {
		bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000022) // Gen2 recovered
		return true
	}

	negotiator := NewLinkNegotiator(bus)

	// Act
	res, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	if !res.Success || res.CurrentSpeed < 2 {
		t.Fatalf("expected Stage 2 recovery to Gen2, got %+v", res)
	}
	if !bus.PnpResetCalled {
		t.Fatalf("expected PnpResetDevice to be invoked")
	}
}

func TestLinkNegotiator_BOOT0_FamilyMismatch_Guarded(t *testing.T) {
	// Arrange: If BAR0 BOOT_0 does not match expected NVIDIA family, refuse MMIO write
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		Name:            "CMP 30HX",
		Family:          "TU116",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x218910DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x99000000) // Foreign family byte 0x99

	negotiator := NewLinkNegotiator(bus)

	// Act
	_, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, false)

	// Assert
	if err == nil {
		t.Fatalf("expected error on BOOT_0 family mismatch, got nil")
	}
	if !errors.Is(err, ErrFamilyMismatch) {
		t.Fatalf("expected ErrFamilyMismatch, got %v", err)
	}
}

func TestLinkNegotiator_RestartNVDisplay_InvokedOnSuccess(t *testing.T) {
	// Arrange: When targetGen is achieved, RestartNVDisplay must be invoked to refresh driver container
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		Name:            "CMP 30HX",
		Family:          "TU116",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x218910DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x17000000)
	bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000022) // Gen2 already negotiated

	negotiator := NewLinkNegotiator(bus)

	// Act
	res, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if !bus.RestartNVDisplayCalled {
		t.Fatalf("expected RestartNVDisplay to be called on Gen2 success")
	}
}
