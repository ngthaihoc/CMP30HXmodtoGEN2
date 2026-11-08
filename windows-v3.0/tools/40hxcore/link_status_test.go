package hxcore

import "testing"

func TestStatusContractLinkStateDistinguishesConfiguredAndObserved(t *testing.T) {
	tests := []struct {
		name, text                     string
		configured                     bool
		speed, width, mrrs, throughput uint32
	}{
		{name: "thiếu trạng thái"},
		{name: "văn bản hỏng", text: "STATUS_CODE=broken\nTLS=Gen2"},
		{name: "TLS nhàn rỗi", text: "STATUS_CODE=GEN2_SUCCESS\nSPEED_CURRENT=1\nWIDTH_CURRENT=16\nTLS_TARGET=2\nMRRS_BYTES=512", configured: true, speed: 1, width: 16, mrrs: 512, throughput: 4000},
		{name: "Gen2 đủ thông số", text: "STATUS_CODE=GEN2_SUCCESS\nSPEED_CURRENT=2\nWIDTH_CURRENT=16\nTLS_TARGET=2\nMRRS_BYTES=512", configured: true, speed: 2, width: 16, mrrs: 512, throughput: 8000},
		{name: "thiếu độ rộng", text: "STATUS_CODE=GEN2_SUCCESS\nSPEED_CURRENT=2\nTLS_TARGET=2", configured: true, speed: 2},
		{name: "legacy", text: "Ket qua: da dat muc tieu Gen2 thanh cong", configured: true, speed: 2},
		{name: "legacy chỉ TLS", text: "TLS=Gen2", configured: true},
		{name: "khác GPU", text: "STATUS_CODE=GEN2_SUCCESS\nDEVICE_ID=1F0B\nSPEED_CURRENT=2\nWIDTH_CURRENT=16\nTLS_TARGET=2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := ParseStatus(tt.text).LinkState(0x2189)
			if state.Configured != tt.configured || state.Speed != tt.speed || state.Width != tt.width || state.MRRS != tt.mrrs || state.ThroughputMBs != tt.throughput {
				t.Fatalf("Diễn giải trạng thái sai: %+v", state)
			}
		})
	}
}
