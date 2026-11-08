package hxcore

import (
	"errors"
	"testing"
)

type diagnosticsBus struct {
	HardwareBus
	pciErrReg uint32
	mmioErr   bool
	writes    int
}

func (b *diagnosticsBus) ReadPCIConfig(bdf, reg uint32) (uint32, error) {
	if reg == b.pciErrReg && b.pciErrReg != 0 {
		return 0, errors.New("injected PCI read failure")
	}
	return b.HardwareBus.ReadPCIConfig(bdf, reg)
}

func (b *diagnosticsBus) ReadMMIO(addr uint64) (uint32, error) {
	if b.mmioErr {
		return 0, errors.New("injected MMIO read failure")
	}
	return b.HardwareBus.ReadMMIO(addr)
}

func (b *diagnosticsBus) WritePCIConfig(bdf, reg uint32, data []byte) error {
	b.writes++
	return b.HardwareBus.WritePCIConfig(bdf, reg, data)
}

func (b *diagnosticsBus) WriteMMIO(addr uint64, value uint32) error {
	b.writes++
	return b.HardwareBus.WriteMMIO(addr, value)
}

func (b *diagnosticsBus) PnpResetDevice(uint16) bool {
	b.writes++
	return false
}

func (b *diagnosticsBus) RestartNVDisplay() error {
	b.writes++
	return nil
}

func TestCollectPlatformDiagnosticsReadOnlySnapshot(t *testing.T) {
	mock := NewMockHardwareBus()
	gpu, root := uint32(0x0100), uint32(0x0008)
	mock.SetPCIConfig(gpu, 0, 0x218910DE)
	mock.SetPCICap(gpu, 0x40)
	mock.SetPCIConfig(gpu, 0x52, 0x22)
	mock.SetPCIConfig(gpu, 0x70, 2)
	mock.SetPCIConfig(gpu, 0x10, 0xF6000000)
	mock.SetMMIO(0xF6000000, 0x17000000)
	mock.SetPCICap(root, 0x50)
	mock.SetPCIConfig(root, 0x50+0x30, 2)
	mock.SetPCIConfig(root, 0, 0x123410DE)
	mock.SetPCIConfig(root, 0x08, 0x06040000)
	mock.SetPCIConfig(root, 0x18, 0x00010100)
	bus := &diagnosticsBus{HardwareBus: mock}

	got, err := CollectPlatformDiagnostics(bus)
	if err != nil {
		t.Fatal(err)
	}
	if got.GPUBDF != gpu || got.VendorID.Value != 0x10de || got.DeviceID.Value != 0x2189 ||
		got.RootBDF.Value != root || got.RootVendorID.Value != 0x10de || got.RootDeviceID.Value != 0x1234 || got.RootTLS.Value != 2 ||
		got.RootTLS.Error != "" ||
		got.BAR0.Value != 0xf6000000 || got.BOOT0.Value != 0x17000000 || got.TLS.Value != 2 ||
		got.LinkSpeed.Value != 2 || got.LinkWidth.Value != 2 {
		t.Fatalf("unexpected diagnostics snapshot: %+v", got)
	}
	if bus.writes != 0 || mock.PnpResetCalled || mock.RestartNVDisplayCalled {
		t.Fatalf("diagnostics caused side effects: writes=%d reset=%v restart=%v", bus.writes, mock.PnpResetCalled, mock.RestartNVDisplayCalled)
	}
}

func TestCollectPlatformDiagnosticsPreservesFieldReadErrors(t *testing.T) {
	mock := NewMockHardwareBus()
	gpu := uint32(0x0100)
	mock.SetPCIConfig(gpu, 0, 0x218910DE)
	mock.SetPCICap(gpu, 0x40)
	mock.SetPCIConfig(gpu, 0x52, 0x22)
	mock.SetPCIConfig(gpu, 0x70, 2)
	mock.SetPCIConfig(gpu, 0x10, 0xF6000000)
	bus := &diagnosticsBus{HardwareBus: mock, mmioErr: true}

	got, err := CollectPlatformDiagnostics(bus)
	if err != nil {
		t.Fatal(err)
	}
	if got.BOOT0.Error != "injected MMIO read failure" || got.GPUBDF != gpu || got.TLS.Value != 2 {
		t.Fatalf("partial read error was not preserved: %+v", got)
	}
	if bus.writes != 0 || mock.PnpResetCalled || mock.RestartNVDisplayCalled {
		t.Fatalf("diagnostics caused side effects: writes=%d", bus.writes)
	}
}

func TestCollectPlatformDiagnosticsPreservesPCIFieldReadErrors(t *testing.T) {
	mock := NewMockHardwareBus()
	gpu := uint32(0x0100)
	mock.SetPCIConfig(gpu, 0, 0x218910DE)
	mock.SetPCICap(gpu, 0x40)
	mock.SetPCIConfig(gpu, 0x52, 0x22)
	mock.SetPCIConfig(gpu, 0x70, 2)
	bus := &diagnosticsBus{HardwareBus: mock, pciErrReg: 0x10}

	got, err := CollectPlatformDiagnostics(bus)
	if err != nil {
		t.Fatal(err)
	}
	if got.BAR0.Error != "injected PCI read failure" || got.BOOT0.Error != "" || got.TLS.Value != 2 {
		t.Fatalf("PCI field failure was hidden or unrelated fields were lost: %+v", got)
	}
	if bus.writes != 0 || mock.PnpResetCalled || mock.RestartNVDisplayCalled {
		t.Fatalf("diagnostics caused side effects: writes=%d", bus.writes)
	}
}

func TestCollectPlatformDiagnosticsPreservesTLSReadError(t *testing.T) {
	mock := NewMockHardwareBus()
	gpu := uint32(0x0100)
	mock.SetPCIConfig(gpu, 0, 0x218910DE)
	mock.SetPCICap(gpu, 0x40)
	mock.SetPCIConfig(gpu, 0x70, 2)
	mock.SetPCIConfig(gpu, 0x10, 0xF6000000)
	mock.SetMMIO(0xF6000000, 0x17000000)
	bus := &diagnosticsBus{HardwareBus: mock, pciErrReg: 0x70}

	got, err := CollectPlatformDiagnostics(bus)
	if err != nil {
		t.Fatal(err)
	}
	if got.TLS.Error != "injected PCI read failure" || got.BOOT0.Value != 0x17000000 {
		t.Fatalf("TLS failure was hidden or unrelated reads were lost: %+v", got)
	}
	if bus.writes != 0 || mock.PnpResetCalled || mock.RestartNVDisplayCalled {
		t.Fatalf("diagnostics caused side effects: writes=%d", bus.writes)
	}
}

func TestCollectPlatformDiagnosticsPreservesRootLinkReadError(t *testing.T) {
	mock := NewMockHardwareBus()
	gpu, root := uint32(0x0100), uint32(0x0008)
	mock.SetPCIConfig(gpu, 0, 0x218910DE)
	mock.SetPCICap(gpu, 0x40)
	mock.SetPCIConfig(gpu, 0x70, 2)
	mock.SetPCIConfig(gpu, 0x10, 0xF6000000)
	mock.SetMMIO(0xF6000000, 0x17000000)
	mock.SetPCICap(root, 0x50)
	mock.SetPCIConfig(root, 0, 0x123410DE)
	mock.SetPCIConfig(root, 0x08, 0x06040000)
	mock.SetPCIConfig(root, 0x18, 0x00010100)
	bus := &diagnosticsBus{HardwareBus: mock, pciErrReg: 0x50 + 0x30}

	got, err := CollectPlatformDiagnostics(bus)
	if err != nil {
		t.Fatal(err)
	}
	if got.RootTLS.Error != "injected PCI read failure" || got.RootBDF.Value != root || got.TLS.Value != 2 {
		t.Fatalf("root TLS failure was hidden or unrelated reads were lost: %+v", got)
	}
	if bus.writes != 0 || mock.PnpResetCalled || mock.RestartNVDisplayCalled {
		t.Fatalf("diagnostics caused side effects: writes=%d", bus.writes)
	}
}
