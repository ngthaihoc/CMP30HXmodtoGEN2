package main

import (
	"testing"
)

func TestExtractShortDID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "PCI\\VEN_10DE&DEV_1F0B&SUBSYS_00000000",
			expected: "10DE&1F0B",
		},
		{
			input:    "pci\\ven_10de&dev_2189&subsys_408a1458",
			expected: "10DE&2189",
		},
		{
			input:    "PCI\\VEN_8086&DEV_4692&SUBSYS_12345678",
			expected: "8086&4692",
		},
		{
			input:    "PCI\\VEN_1002&DEV_1681&SUBSYS_1DC31043",
			expected: "1002&1681",
		},
		{
			input:    "INVALID_STRING",
			expected: "",
		},
	}

	for _, tt := range tests {
		got := ExtractShortDID(tt.input)
		if got != tt.expected {
			t.Errorf("ExtractShortDID(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}
}

func TestGetActiveDisplayTargets_Hook(t *testing.T) {
	defer func() { enumDisplayDevicesHook = nil }()

	enumDisplayDevicesHook = func() []ActiveDisplayTarget {
		return []ActiveDisplayTarget{
			{
				DeviceID:   "PCI\\VEN_8086&DEV_4692&SUBSYS_00000000",
				StateFlags: DisplayDeviceAttachedToDesktop | DisplayDevicePrimaryDevice,
				IsAttached: true,
			},
			{
				DeviceID:   "PCI\\VEN_10DE&DEV_1F0B&SUBSYS_00000000",
				StateFlags: 0,
				IsAttached: false,
			},
		}
	}

	targets := GetActiveDisplayTargets()
	if len(targets) != 2 {
		t.Fatalf("Expected 2 targets, got %d", len(targets))
	}
	if !targets[0].IsAttached {
		t.Errorf("Expected target 0 to be attached")
	}
	if targets[1].IsAttached {
		t.Errorf("Expected target 1 not to be attached")
	}
}
