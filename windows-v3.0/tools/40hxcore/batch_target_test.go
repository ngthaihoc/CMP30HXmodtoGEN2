package hxcore

import "testing"

func TestUnlockSelectsRequestedDeviceWithoutRestartingDisplay(t *testing.T) {
	bus := NewMockHardwareBus()
	const cmp40 = 0x100
	const cmp30 = 0x200
	bus.SetPCIConfig(cmp40, 0, 0x1F0B10DE)
	bus.SetPCIConfig(cmp30, 0, 0x218910DE)
	bus.SetPCICap(cmp30, 0x40)
	bus.SetPCIConfig(cmp30, 0x50, (2|16<<4)<<16)
	bus.SetPCIConfig(cmp30, 0x48, 0)

	verdict, err := NewLinkNegotiator(bus).Unlock(NegotiationOptions{
		TargetGen: 2, DeviceID: 0x2189, SkipDisplayRestart: true,
	})
	if err != nil || !verdict.Success || verdict.GPUBDF != cmp30 || verdict.StatusContract.DeviceID != 0x2189 {
		t.Fatalf("expected only CMP 30HX to be selected, got %+v, %v", verdict, err)
	}
	if bus.RestartNVDisplayCalled {
		t.Fatal("batch unlock changed NVIDIA display service")
	}
}
