package hxcore

import "testing"

func TestLookupGPUProfile(t *testing.T) {
	cases := []struct {
		name         string
		vendor       uint16
		device       uint16
		wantName     string
		wantFirmware bool
		wantMaxGen   uint32
		wantFound    bool
	}{
		{name: "cmp40", vendor: 0x10DE, device: 0x1F0B, wantName: "CMP 40HX", wantFirmware: true, wantMaxGen: 2, wantFound: true},
		{name: "cmp30", vendor: 0x10DE, device: 0x2189, wantName: "CMP 30HX", wantFirmware: false, wantMaxGen: 2, wantFound: true},
		{name: "unknown", vendor: 0x10DE, device: 0x1234, wantFound: false},
	}
	for _, tc := range cases {
		got, ok := LookupGPUProfile(tc.vendor, tc.device)
		if ok != tc.wantFound {
			t.Fatalf("%s: found=%v, want %v", tc.name, ok, tc.wantFound)
		}
		if ok && (got.Name != tc.wantName || got.FirmwareUnlock != tc.wantFirmware || got.MaxSupportedGen != tc.wantMaxGen) {
			t.Fatalf("%s: profile=%+v", tc.name, got)
		}
	}
}

func TestLinkTargetAllowed(t *testing.T) {
	cases := []struct {
		name                string
		endpoint, root, cap uint32
		target              uint32
		want                bool
	}{
		{name: "both gen2", endpoint: 2, root: 3, cap: 3, target: 2, want: true},
		{name: "endpoint gen1 for gen2", endpoint: 1, root: 3, cap: 3, target: 2, want: false},
		{name: "root gen1 for gen2", endpoint: 3, root: 1, cap: 3, target: 2, want: false},
		{name: "profile ceiling gen1 for gen2", endpoint: 3, root: 3, cap: 1, target: 2, want: false},
		{name: "all gen3 for gen3", endpoint: 3, root: 3, cap: 3, target: 3, want: true},
		{name: "root gen2 for gen3", endpoint: 3, root: 2, cap: 3, target: 3, want: false},
		{name: "cap gen2 for gen3", endpoint: 3, root: 3, cap: 2, target: 3, want: false},
	}
	for _, tc := range cases {
		if got := LinkTargetAllowed(tc.endpoint, tc.root, tc.cap, tc.target); got != tc.want {
			t.Fatalf("%s: allowed=%v, want %v", tc.name, got, tc.want)
		}
	}
}
