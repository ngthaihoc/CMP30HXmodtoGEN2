package hxcore

import (
	"errors"
	"testing"
	"time"
)

func TestLinkNegotiatorUnlockPreservesCapabilityFailure(t *testing.T) {
	bus := NewMockHardwareBus()
	bus.SetPCIConfig(0x0100, 0, 0x218910DE)
	verdict, err := NewLinkNegotiator(bus).Unlock(NegotiationOptions{TargetGen: 3, AllowStage2: true})
	if !errors.Is(err, ErrPCIeCapMissing) || verdict.StatusContract.ErrorCode != "PCIE_CAP_MISSING" {
		t.Fatalf("Phải giữ nguyên lỗi capability: verdict=%+v, err=%v", verdict, err)
	}
	if verdict.TargetGen != 2 || verdict.Profile.DeviceID != 0x2189 || bus.PnpResetCalled {
		t.Fatalf("Phải nhận diện và bảo vệ CMP 30HX: %+v", verdict)
	}
}

func TestLinkNegotiatorUnlockIdleLinkKeepsConfiguredVerdict(t *testing.T) {
	bus := NewMockHardwareBus()
	bus.SetPCIConfig(0x0100, 0, 0x218910DE)
	bus.SetPCICap(0x0100, 0x40)
	bus.SetPCIConfig(0x0100, 0x4C, 2)
	bus.SetPCIConfig(0x0100, 0x52, 0x101)
	verdict, err := NewLinkNegotiator(bus).Unlock(NegotiationOptions{TargetGen: 3, AllowStage2: true})
	if err != nil || !verdict.Success || verdict.CurrentSpeed != 1 || verdict.StatusContract.TLSTarget != 2 || verdict.NeedsRetry {
		t.Fatalf("TLS Gen2 khi nhàn rỗi phải giữ verdict thành công: %+v, %v", verdict, err)
	}
	if bus.PnpResetCalled || !bus.RestartNVDisplayCalled {
		t.Fatal("Không reset CMP 30HX; vẫn phải restart NVDisplay sau khi tối ưu MRRS")
	}
}

func TestLinkNegotiatorUnlockRetriesDiscovery(t *testing.T) {
	bus := NewMockHardwareBus()
	n := NewLinkNegotiator(bus)
	n.SetSleepFn(func(d time.Duration) {
		bus.SetPCIConfig(0x0100, 0, 0x218910DE)
		bus.SetPCICap(0x0100, 0x40)
		bus.SetPCIConfig(0x0100, 0x52, 0x102)
	})
	verdict, err := n.Unlock(NegotiationOptions{})
	if err != nil || !verdict.Success || verdict.GPUBDF != 0x0100 {
		t.Fatalf("Phải nhận GPU nạp trễ: %+v, %v", verdict, err)
	}
}

func TestLinkNegotiatorUnlockNoGPU(t *testing.T) {
	verdict, err := NewLinkNegotiator(NewMockHardwareBus()).Unlock(NegotiationOptions{})
	if err != nil || verdict.Success || verdict.StatusContract.StatusCode != StatusNoGPU {
		t.Fatalf("Không có GPU phải trả NO_GPU: %+v, %v", verdict, err)
	}
}

func TestLinkNegotiatorUnlockRestartsDisplayOnce(t *testing.T) {
	bus := NewMockHardwareBus()
	bus.SetPCIConfig(0x0100, 0, 0x218910DE)
	bus.SetPCICap(0x0100, 0x40)
	bus.SetPCIConfig(0x0100, 0x4C, 2)
	bus.SetPCIConfig(0x0100, 0x52, 0x101)
	bus.OnWritePCIConfig = func(bdf, reg uint32, data []byte) {
		if reg == 0x50 && len(data) > 0 && data[0]&0x20 != 0 {
			bus.SetPCIConfig(0x0100, 0x52, 0x102)
		}
	}
	restarts := 0
	bus.OnRestartNVDisplay = func() error { restarts++; return nil }
	verdict, err := NewLinkNegotiator(bus).Unlock(NegotiationOptions{})
	if err != nil || !verdict.Success || restarts != 1 {
		t.Fatalf("Mở khóa phải restart NVDisplay đúng một lần: %+v, %v, restart=%d", verdict, err, restarts)
	}
}

func TestLinkNegotiatorUnlockAlreadyGen2StillAppliesMRRS512(t *testing.T) {
	bus := NewMockHardwareBus()
	bus.SetPCIConfig(0x0100, 0, 0x218910DE)
	bus.SetPCICap(0x0100, 0x40)
	bus.SetPCIConfig(0x0100, 0x52, 0x102)
	verdict, err := NewLinkNegotiator(bus).Unlock(NegotiationOptions{})
	if err != nil || !verdict.Success || verdict.StatusContract.MRRS != 512 || !bus.RestartNVDisplayCalled {
		t.Fatalf("Link đã Gen2 vẫn phải tối ưu MRRS 512B: %+v, %v", verdict, err)
	}
}

func TestLinkNegotiatorUnlockPreparesActuallyDiscoveredProfile(t *testing.T) {
	bus := NewMockHardwareBus()
	n := NewLinkNegotiator(bus)
	n.SetSleepFn(func(time.Duration) {
		bus.SetPCIConfig(0x0100, 0, 0x1F0B10DE)
		bus.SetPCICap(0x0100, 0x40)
		bus.SetPCIConfig(0x0100, 0x52, 0x102)
	})
	prepared := false
	verdict, err := n.Unlock(NegotiationOptions{Prepare: func(profile GPUProfile) {
		prepared = profile.DeviceID == 0x1F0B && profile.HasSafePL0
	}})
	if err != nil || !verdict.Success || !prepared {
		t.Fatalf("Phải chuẩn bị GSP cho đúng GPU tìm thấy sau retry: %+v, %v", verdict, err)
	}
}
