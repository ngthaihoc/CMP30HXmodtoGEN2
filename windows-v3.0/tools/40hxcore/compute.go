package hxcore

import (
	"fmt"
)

type ComputeStatus string

const (
	ComputeStatusUnlocked       ComputeStatus = "UNLOCKED"
	ComputeStatusLocked         ComputeStatus = "LOCKED"
	ComputeStatusNotApplicable  ComputeStatus = "NOT_APPLICABLE"
	ComputeStatusFamilyMismatch ComputeStatus = "FAMILY_MISMATCH"
	ComputeStatusDriverBlocked  ComputeStatus = "DRIVER_BLOCKED"
)

// ComputeReport: Báo cáo định kiểu năng lực tính toán & Tensor Core
type ComputeReport struct {
	Status   ComputeStatus
	Unlocked bool
	SS0      uint32
	SS1      uint32
	BAR0     uint64
	Boot0    uint32
	Verdict  string
}

// ComputeInspector: Deep Module giải mã Tensor Core và năng lực tính toán
type ComputeInspector struct {
	bus HardwareBus
}

func NewComputeInspector(bus HardwareBus) *ComputeInspector {
	return &ComputeInspector{bus: bus}
}

// InspectCompute thực thi kiểm tra an toàn BAR0 BOOT_0 và giải mã SS0/SS1
func (c *ComputeInspector) InspectCompute(gpuBDF uint32, prof GPUProfile) (*ComputeReport, error) {
	if !prof.FirmwareUnlock {
		return &ComputeReport{
			Status:  ComputeStatusNotApplicable,
			Verdict: fmt.Sprintf("%s không có phần cứng Tensor Core / Firmware Unlock (bỏ qua)", prof.Name),
		}, nil
	}

	bar0raw, err := c.bus.ReadPCIConfig(gpuBDF, 0x10)
	if err != nil || bar0raw == 0 || bar0raw == 0xFFFFFFFF {
		return &ComputeReport{
			Status:  ComputeStatusDriverBlocked,
			Verdict: "Không thể đọc địa chỉ vật lý BAR0",
		}, ErrInvalidBAR0
	}
	bar0 := uint64(bar0raw & 0xFFFFFFF0)

	boot0, err := c.bus.ReadMMIO(bar0 + 0x00)
	if err != nil {
		return &ComputeReport{
			Status:  ComputeStatusDriverBlocked,
			BAR0:    bar0,
			Verdict: fmt.Sprintf("Không thể đọc MMIO BOOT_0: %v", err),
		}, err
	}

	fam := (boot0 >> 24) & 0xFF
	if fam != 0x16 {
		return &ComputeReport{
			Status:  ComputeStatusFamilyMismatch,
			BAR0:    bar0,
			Boot0:   boot0,
			Verdict: fmt.Sprintf("Họ vi kiến trúc BOOT_0 không khớp (0x%02X != 0x16 TU106)", fam),
		}, ErrFamilyMismatch
	}

	ss0, err := c.bus.ReadMMIO(bar0 + SS0Offset)
	if err != nil {
		return &ComputeReport{
			Status:  ComputeStatusDriverBlocked,
			BAR0:    bar0,
			Boot0:   boot0,
			Verdict: fmt.Sprintf("Lỗi đọc thanh ghi SS0: %v", err),
		}, err
	}

	ss1, _ := c.bus.ReadMMIO(bar0 + SS1Offset)

	unlocked := ss0 == 0x88888888
	status := ComputeStatusLocked
	verdict := fmt.Sprintf("❌ Bị khoá: SS0=0x%08X (kỳ vọng 0x88888888), SS1=0x%08X", ss0, ss1)
	if unlocked {
		status = ComputeStatusUnlocked
		verdict = fmt.Sprintf("✅ Tối đa: Mở khoá Tensor Core thành công (SS0=0x%08X, SS1=0x%08X, ~50 TFLOPS FP16)", ss0, ss1)
	}

	return &ComputeReport{
		Status:   status,
		Unlocked: unlocked,
		SS0:      ss0,
		SS1:      ss1,
		BAR0:     bar0,
		Boot0:    boot0,
		Verdict:  verdict,
	}, nil
}
