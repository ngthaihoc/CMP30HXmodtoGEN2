package hxcore

import "fmt"

// DiagnosticField records a read result without hiding partial failures.
type DiagnosticField struct {
	Value uint32 `json:"value"`
	Error string `json:"error,omitempty"`
}

// PlatformDiagnostics is a read-only PCI/MMIO snapshot. A field error does not
// prevent the remaining independent reads from being collected.
type PlatformDiagnostics struct {
	GPUBDF       uint32          `json:"gpu_bdf"`
	GPUName      string          `json:"gpu_name"`
	VendorID     DiagnosticField `json:"vendor_id"`
	DeviceID     DiagnosticField `json:"device_id"`
	RootBDF      DiagnosticField `json:"root_bdf"`
	RootVendorID DiagnosticField `json:"root_vendor_id"`
	RootDeviceID DiagnosticField `json:"root_device_id"`
	RootTLS      DiagnosticField `json:"root_tls"`
	BAR0         DiagnosticField `json:"bar0"`
	BOOT0        DiagnosticField `json:"boot0"`
	TLS          DiagnosticField `json:"tls"`
	LinkSpeed    DiagnosticField `json:"link_speed"`
	LinkWidth    DiagnosticField `json:"link_width"`
}

// CollectPlatformDiagnostics reads GPU/root PCI state and BAR0 registers only.
func CollectPlatformDiagnostics(bus HardwareBus) (PlatformDiagnostics, error) {
	var snapshot PlatformDiagnostics
	gpuBDF, profile, found := FindGPUPCIWithBus(bus)
	if !found {
		return snapshot, fmt.Errorf("supported GPU not found")
	}
	snapshot.GPUBDF, snapshot.GPUName = gpuBDF, profile.Name
	readPCI := func(bdf, reg uint32) DiagnosticField {
		value, err := bus.ReadPCIConfig(bdf, reg)
		if err != nil {
			return DiagnosticField{Error: err.Error()}
		}
		return DiagnosticField{Value: value}
	}
	gpuID := readPCI(gpuBDF, 0x00)
	snapshot.VendorID = DiagnosticField{Value: gpuID.Value & 0xffff, Error: gpuID.Error}
	snapshot.DeviceID = DiagnosticField{Value: gpuID.Value >> 16, Error: gpuID.Error}

	root := NewLinkNegotiator(bus).findRootPort((gpuBDF >> 8) & 0xff)
	if root != 0xffffffff {
		snapshot.RootBDF = DiagnosticField{Value: root}
		rootID := readPCI(root, 0x00)
		snapshot.RootVendorID = DiagnosticField{Value: rootID.Value & 0xffff, Error: rootID.Error}
		snapshot.RootDeviceID = DiagnosticField{Value: rootID.Value >> 16, Error: rootID.Error}
		rootCap := NewLinkNegotiator(bus).findPcieCap(root)
		if rootCap == 0 {
			snapshot.RootTLS = DiagnosticField{Error: ErrPCIeCapMissing.Error()}
		} else {
			snapshot.RootTLS = readPCI(root, rootCap+0x30)
			snapshot.RootTLS.Value &= 0xf
		}
	} else {
		snapshot.RootBDF.Error = "root port not found"
	}
	cap := NewLinkNegotiator(bus).findPcieCap(gpuBDF)
	if cap == 0 {
		snapshot.TLS = DiagnosticField{Error: ErrPCIeCapMissing.Error()}
	} else {
		snapshot.TLS = readPCI(gpuBDF, cap+0x30)
		snapshot.TLS.Value &= 0xf
	}
	if value := bus.LinkSpeed(gpuBDF); value != 0 {
		snapshot.LinkSpeed = DiagnosticField{Value: value}
	} else {
		snapshot.LinkSpeed = DiagnosticField{Error: "link speed unavailable"}
	}
	if value := bus.LinkWidth(gpuBDF); value != 0 {
		snapshot.LinkWidth = DiagnosticField{Value: value}
	} else {
		snapshot.LinkWidth = DiagnosticField{Error: "link width unavailable"}
	}

	snapshot.BAR0 = readPCI(gpuBDF, 0x10)
	bar0 := snapshot.BAR0.Value & 0xfffffff0
	if snapshot.BAR0.Error != "" || bar0 == 0 || bar0 == 0xfffffff0 {
		if snapshot.BAR0.Error == "" {
			snapshot.BAR0.Error = ErrInvalidBAR0.Error()
		}
		return snapshot, nil
	}
	boot0, err := bus.ReadMMIO(uint64(bar0))
	if err != nil {
		snapshot.BOOT0.Error = err.Error()
	} else {
		snapshot.BOOT0.Value = boot0
	}
	return snapshot, nil
}
