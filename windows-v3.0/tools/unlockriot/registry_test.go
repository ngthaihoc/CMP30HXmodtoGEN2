package main

import (
	"bytes"
	"testing"
)

func TestClassifyAdapter(t *testing.T) {
	tests := []struct {
		name     string
		matchID  string
		desc     string
		provider string
		expected AdapterRole
	}{
		{
			name:     "CMP40HX_Nvidia",
			matchID:  "PCI\\VEN_10DE&DEV_1F0B&SUBSYS_00000000",
			desc:     "NVIDIA CMP 40HX",
			provider: "NVIDIA",
			expected: RoleNvidiaRender,
		},
		{
			name:     "CMP30HX_Nvidia",
			matchID:  "PCI\\VEN_10DE&DEV_2189&SUBSYS_408A1458",
			desc:     "NVIDIA CMP 30HX",
			provider: "NVIDIA",
			expected: RoleNvidiaRender,
		},
		{
			name:     "Generic_NvidiaGraphicsDevice",
			matchID:  "pci\\ven_10de&dev_25ad&subsys_00000000",
			desc:     "NVIDIA Graphics Device",
			provider: "NVIDIA",
			expected: RoleNvidiaRender,
		},
		{
			name:     "AMD_Radeon_iGPU",
			matchID:  "PCI\\VEN_1002&DEV_1681&SUBSYS_1DC31043",
			desc:     "AMD Radeon(TM) Graphics",
			provider: "Advanced Micro Devices, Inc.",
			expected: RoleDisplayHost,
		},
		{
			name:     "Intel_UHD_Graphics",
			matchID:  "PCI\\VEN_8086&DEV_4692&SUBSYS_00000000",
			desc:     "Intel(R) UHD Graphics 730",
			provider: "Intel Corporation",
			expected: RoleDisplayHost,
		},
		{
			name:     "Microsoft_BasicDisplay",
			matchID:  "PCI\\VEN_10DE&DEV_1F0B",
			desc:     "Microsoft Basic Display Adapter",
			provider: "Microsoft",
			expected: RoleUnknown,
		},
		{
			name:     "Unknown_Device",
			matchID:  "PCI\\VEN_1234&DEV_5678",
			desc:     "Generic Controller",
			provider: "Generic",
			expected: RoleUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyAdapter(tt.matchID, tt.desc, tt.provider)
			if got != tt.expected {
				t.Errorf("ClassifyAdapter() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestConfigureDriverClassRegistry_Execution(t *testing.T) {
	buf := &bytes.Buffer{}
	res, err := ConfigureDriverClassRegistry(buf, GPUModelCMP40HX)
	if err != nil {
		t.Fatalf("ConfigureDriverClassRegistry returned error: %v", err)
	}

	t.Logf("Registry config output:\n%s", buf.String())
	t.Logf("NvidiaIndices: %v, DisplayIndices: %v, TDR: %v, ICD: %v",
		res.NvidiaIndices, res.DisplayIndices, res.TdrConfigured, res.IcdRegistered)

	if !res.TdrConfigured {
		t.Errorf("Expected TdrConfigured to be true")
	}
	if !res.IcdRegistered {
		t.Errorf("Expected IcdRegistered to be true")
	}
}
