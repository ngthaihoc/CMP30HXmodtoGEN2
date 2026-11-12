package hxcore

import (
	"testing"
	"time"
)

type ignoredMRRSWriteBus struct{ *MockHardwareBus }

func (b ignoredMRRSWriteBus) WritePCIConfig(bdf, reg uint32, data []byte) error {
	if reg == 0x48 {
		return nil
	}
	return b.MockHardwareBus.WritePCIConfig(bdf, reg, data)
}

func TestNegotiationRejectsUnappliedMRRS(t *testing.T) {
	for _, speed := range []uint32{1, 2} {
		t.Run(string(rune('0'+speed)), func(t *testing.T) {
			bus := ignoredMRRSWriteBus{NewMockHardwareBus()}
			const bdf = 0x100
			bus.SetPCICap(bdf, 0x40)
			bus.SetPCIConfig(bdf, 0x4c, 2)
			bus.SetPCIConfig(bdf, 0x52, speed | 0x20)
			prof := GPUProfile{DeviceID: 0x2189, Family: "TU116", MaxSupportedGen: 2}
			n := newLinkNegotiator(bus, func(time.Duration) {})
			verdict, err := n.ExecuteNegotiation(bdf, prof, NegotiationOptions{TargetGen: 2})
			if err == nil || verdict == nil || verdict.Success {
				t.Fatalf("expected failure for ignored MRRS write, got verdict=%+v err=%v", verdict, err)
			}
		})
	}
}
