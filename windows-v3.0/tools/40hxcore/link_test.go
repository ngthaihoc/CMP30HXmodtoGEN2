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
	bus.SetMMIO(bar0+0x00, 0x17000000)           // BOOT_0 TU116
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
	// On CMP 40HX (TU106), when allowStage2=true, PnpResetDevice is called and link recovers to Gen2
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x1F0B,
		Name:            "CMP 40HX",
		Family:          "TU106",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x1F0B10DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x16000000)     // BOOT_0: TU106 (0x16)
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

func TestLinkNegotiator_Stage2_40HX_RootLinkDisableAndPnP(t *testing.T) {
	// Arrange: CMP 40HX (TU106) with Root Port present.
	// When Stage 1 fails and allowStage2=true, rootLinkDisable must be executed on Root Port,
	// followed by PnpResetDevice and retrain.
	bus := NewMockHardwareBus()
	gpuBDF := uint32(0x0100)
	rootBDF := uint32(0x0008) // Root port at 00:01.0
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x1F0B,
		Name:            "CMP 40HX",
		Family:          "TU106",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(gpuBDF, 0x00, 0x1F0B10DE)
	bus.SetPCICap(gpuBDF, 0x40)
	bus.SetPCIConfig(gpuBDF, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x16000000)        // BOOT_0: TU106
	bus.SetPCIConfig(gpuBDF, 0x40+0x12, 0x00000011) // Stuck at Gen1

	// Root Port config
	bus.SetPCICap(rootBDF, 0x50)
	bus.SetPCIConfig(rootBDF, 0x50+0x10, 0x00000000) // LNKCTL

	// After Root Link Disable and PnP reset, simulate recovery
	bus.OnPnpReset = func(devID uint16) bool {
		bus.SetPCIConfig(gpuBDF, 0x40+0x12, 0x00000022) // Gen2 recovered
		return true
	}

	negotiator := NewLinkNegotiator(bus)

	// Act
	res, err := negotiator.Negotiate(gpuBDF, prof, rootBDF, 2, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	if !res.Success || res.CurrentSpeed < 2 {
		t.Fatalf("expected Stage 2 recovery to Gen2, got %+v", res)
	}
	if !res.Stage2Triggered {
		t.Fatalf("expected Stage2Triggered to be true")
	}
	if !bus.PnpResetCalled {
		t.Fatalf("expected PnpResetDevice to be invoked")
	}
}

func TestLinkNegotiator_TU116_DisallowsStage2(t *testing.T) {
	// Arrange: TU116 (CMP 30HX) must never trigger Stage 2 PnP reset even if allowStage2 is true
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

	negotiator := NewLinkNegotiator(bus)

	// Act: pass allowStage2 = true
	res, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert: Stage 2 must NOT be triggered, PnpResetDevice must NOT be called
	if res.Stage2Triggered {
		t.Fatalf("expected Stage2Triggered to be false on TU116, got true")
	}
	if bus.PnpResetCalled {
		t.Fatalf("PnpResetDevice was called on TU116, violates Hardware Rule 3")
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

func TestLinkNegotiator_BOOT0_TU106_Mismatch_Guarded(t *testing.T) {
	// Arrange: TU106 (CMP 40HX) must strictly match family 0x16.
	// If BAR0 remaps to a TU116 (0x21), it must safely abort MMIO writes.
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x1F0B,
		Name:            "CMP 40HX",
		Family:          "TU106",
		MaxSupportedGen: 2,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x1F0B10DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x21000000) // TU116 family byte 0x21, mismatch for TU106

	negotiator := NewLinkNegotiator(bus)

	// Act
	_, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, false)

	// Assert
	if err == nil {
		t.Fatalf("expected error on BOOT_0 TU106 mismatch, got nil")
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

func TestLinkNegotiator_RootLinkDisable_Sequencing(t *testing.T) {
	// Arrange
	bus := NewMockHardwareBus()
	gpuBDF := uint32(0x0100)
	rootBDF := uint32(0x0008)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x1F0B,
		Name:            "CMP 40HX",
		Family:          "TU106",
		MaxSupportedGen: 2,
		RequiresMMIO:    true,
	}
	bus.SetPCIConfig(gpuBDF, 0x00, 0x1F0B10DE)
	bus.SetPCICap(gpuBDF, 0x40)
	bus.SetPCIConfig(gpuBDF, 0x10, 0xF6000000)
	bus.SetMMIO(0xF6000000+0x00, 0x16000000)        // BOOT_0
	bus.SetPCIConfig(gpuBDF, 0x40+0x12, 0x00000011) // Gen1

	bus.SetPCICap(rootBDF, 0x50)
	bus.SetPCIConfig(rootBDF, 0x50+0x10, 0x00000000)

	rootLinkDisabled := false
	mmioWrittenWhileDisabled := false
	mmioWrittenAfterEnabled := false

	bus.OnWritePCIConfig = func(bdf uint32, reg uint32, data []byte) {
		if bdf == rootBDF && reg == 0x50+0x10 && len(data) >= 1 {
			if data[0]&0x10 != 0 {
				rootLinkDisabled = true
			} else {
				rootLinkDisabled = false
			}
		}
	}

	bus.OnWriteMMIO = func(physAddr uint64, val uint32) {
		if rootLinkDisabled {
			mmioWrittenWhileDisabled = true
		} else {
			mmioWrittenAfterEnabled = true
		}
	}

	negotiator := NewLinkNegotiator(bus)

	// Act
	negotiator.rootLinkDisable(rootBDF, gpuBDF, 0xF6000000, prof, 2)

	// Assert
	if mmioWrittenWhileDisabled {
		t.Fatalf("violation: MMIO shadow registers written while root link was disabled (bit 4=1)")
	}
	if !mmioWrittenAfterEnabled {
		t.Fatalf("violation: MMIO shadow registers NOT re-injected after root link was re-enabled (bit 4=0)")
	}
}

func TestLinkNegotiator_EnsuresMemorySpaceEnable(t *testing.T) {
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		Name:            "CMP 30HX",
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		Family:          "TU116",
		RequiresMMIO:    true,
		MaxSupportedGen: 2,
	}

	bus.SetPCIConfig(bdf, 0x00, 0x218910DE)
	// Đặt Command register (0x04) = 0x0000 (MSE và BME đều tắt)
	bus.SetPCIConfig(bdf, 0x04, 0x00000000)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xDE000000) // BAR0 = 0xDE000000
	bus.SetPCIConfig(bdf, 0x40+0x0C, 0x00000001) // LNKCAP Gen1
	bus.SetPCIConfig(bdf, 0x40+0x10, 0x00000000) // LNKCTL
	bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000011) // LNKSTA Gen1 x16
	bus.SetPCIConfig(bdf, 0x40+0x30, 0x00000001) // LNKCTL2 TLS=1

	bus.SetMMIO(0xDE000000, 0x16800000) // BOOT_0 TU116

	negotiator := NewLinkNegotiator(bus)
	res, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, false)
	if err != nil {
		t.Fatalf("Negotiate failed: %v", err)
	}

	// Xác nhận thanh ghi PCI Command 0x04 đã được bật bit 1 (MSE 0x02) và bit 2 (BME 0x04)
	cmdVal, _ := bus.ReadPCIConfig(bdf, 0x04)
	if (cmdVal & 0x06) != 0x06 {
		t.Errorf("expected PCI Command register (0x04) to have MSE and BME set (0x06), got 0x%04X", cmdVal)
	}

	if res.TargetTLS < 2 {
		t.Errorf("expected TargetTLS to be at least Gen2, got Gen%d", res.TargetTLS)
	}
}

func TestMMIORegWrite_ComputeValue(t *testing.T) {
	// Case 1: RMW is false -> returns Value directly without calling readFn
	regNonRMW := MMIORegWrite{Offset: 0x1000, Value: 0x12345678, RMW: false}
	val := regNonRMW.ComputeValue(func(offset uint64) (uint32, error) {
		t.Fatal("readFn should not be called when RMW is false")
		return 0, nil
	})
	if val != 0x12345678 {
		t.Errorf("expected 0x12345678, got 0x%08X", val)
	}

	// Case 2: RMW is true and read succeeds -> applies bitmask
	// old has 0x80005800, mask 0x000C0000, value 0x00080000 -> (0x80005800 &^ 0x000C0000) | 0x00080000 = 0x80085800
	regRMW := MMIORegWrite{
		Offset: 0x8C040, Value: 0x00080000, RMW: true, Mask: 0x000C0000, FallbackValue: 0x80085800,
	}
	val = regRMW.ComputeValue(func(offset uint64) (uint32, error) {
		return 0x80005800, nil
	})
	if val != 0x80085800 {
		t.Errorf("expected 0x80085800 after RMW bitmask, got 0x%08X", val)
	}

	// Case 3: RMW is true and read fails -> returns FallbackValue
	val = regRMW.ComputeValue(func(offset uint64) (uint32, error) {
		return 0, errors.New("read error")
	})
	if val != 0x80085800 {
		t.Errorf("expected fallback 0x80085800 on read error, got 0x%08X", val)
	}
}

func TestLinkNegotiator_TU106_MMIO_RMWSequence(t *testing.T) {
	// Arrange: CMP 40HX (TU106) with initial pre-existing MMIO register values
	bus := NewMockHardwareBus()
	bdf := uint32(0x0100)
	prof := GPUProfile{
		VendorID:        0x10DE,
		DeviceID:        0x1F0B,
		Name:            "CMP 40HX",
		Family:          "TU106",
		MaxSupportedGen: 2,
		RequiresMMIO:    true,
	}
	bus.SetPCIConfig(bdf, 0x00, 0x1F0B10DE)
	bus.SetPCICap(bdf, 0x40)
	bus.SetPCIConfig(bdf, 0x10, 0xF6000000)
	bar0 := uint64(0xF6000000)
	bus.SetMMIO(bar0+0x00, 0x16000000) // BOOT_0 TU106 (0x16)

	// Pre-seed LINK_CONFIG_0 with existing bits (e.g. 0x80005800, with rate bits 0)
	bus.SetMMIO(bar0+0x8C040, 0x80005800)
	// Pre-seed PL_LINK_RATE with existing bits (e.g. 0x00200036, with rate bits 0)
	bus.SetMMIO(bar0+0x8C1C0, 0x00200036)

	bus.SetPCIConfig(bdf, 0x40+0x12, 0x00000021) // Gen2 attained

	negotiator := NewLinkNegotiator(bus)
	res, err := negotiator.Negotiate(bdf, prof, 0xFFFFFFFF, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}

	// Assert that RMW preserved unmasked bits and inserted the new rate bits
	linkCfg, _ := bus.ReadMMIO(bar0 + 0x8C040)
	if linkCfg != 0x80085800 {
		t.Errorf("LINK_CONFIG_0 RMW mismatch: expected 0x80085800, got 0x%08X", linkCfg)
	}
	plLinkRate, _ := bus.ReadMMIO(bar0 + 0x8C1C0)
	if plLinkRate != 0x00240036 {
		t.Errorf("PL_LINK_RATE RMW mismatch: expected 0x00240036, got 0x%08X", plLinkRate)
	}
}

