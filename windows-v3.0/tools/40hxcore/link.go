package hxcore

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
)

var (
	ErrFamilyMismatch   = errors.New("BAR0 BOOT_0 family mismatch")
	ErrPCIeCapMissing   = errors.New("PCIe capability missing")
	ErrInvalidBAR0      = errors.New("invalid BAR0 physical address")
	ErrRootPortIncapable = errors.New("root port does not support target speed")
)

// HardwareBus: Điểm phân tách (seam) giao tiếp I/O phần cứng cấp thấp
type HardwareBus interface {
	ReadPCIConfig(bdf uint32, reg uint32) (uint32, error)
	WritePCIConfig(bdf uint32, reg uint32, data []byte) error
	ReadMMIO(physAddr uint64) (uint32, error)
	WriteMMIO(physAddr uint64, val uint32) error
	PnpResetDevice(devID uint16) bool
	RestartNVDisplay() error
	Sleep(d time.Duration)
}

// MMIORegWrite: Bản ghi thiết lập thanh ghi MMIO chuẩn hoá
type MMIORegWrite struct {
	Offset uint64
	Value  uint32
	Name   string
}

// TU116ShadowSequence: Đơn vị duy nhất định nghĩa chuỗi thanh ghi mở khoá shadow cho TU116
var TU116ShadowSequence = []MMIORegWrite{
	{0x0008841C, 0xE0B42D00, "PRIV_MISC_1"},
	{0x0008872C, 6, "XVE_OVR"},
	{0x0008C040, 0x80085800, "LINK_CONFIG_0"},
	{0x0008C1C0, 0x00240036, "PL_LINK_RATE"},
	{0x0008C2C0, 0x068731B3, "CYA_0"},
	{0x0008872C, 6, "XVE_OVR_CONFIRM"},
}

// TU106PL0Sequence: Chuỗi thanh ghi mở khoá cho TU106 (CMP 40HX)
var TU106PL0Sequence = []MMIORegWrite{
	{0x8872C, 0x6, "XVE_OVR=6"},
	{0x8C040, 0x80085800, "LINK_CONFIG_0"},
	{0x8841C, 0xE0B42D00, "PRIV_MISC_1"},
	{0x8C1C0, 0x00240036, "PL_LINK_RATE"},
	{0x8C2C0, 0x068731B3, "CYA_0"},
}

// NegotiationResult: Kết quả thương lượng và huấn luyện lại PCIe
type NegotiationResult struct {
	CurrentSpeed     uint32
	CurrentWidth     uint32
	TargetGen        uint32
	TargetTLS        uint32
	RootTLS          uint32
	Success          bool
	Stage2Triggered  bool
	Verdict          string
	DiagnosticReport string
}

// LinkNegotiator: Deep Module điều khiển huấn luyện lại PCIe và tối ưu DMA
type LinkNegotiator struct {
	bus HardwareBus
}

// NewLinkNegotiator khởi tạo LinkNegotiator với HardwareBus adapter
func NewLinkNegotiator(bus HardwareBus) *LinkNegotiator {
	return &LinkNegotiator{bus: bus}
}

// Negotiate thực thi quy trình huấn luyện PCIe sâu và tối ưu MRRS 512B
func (n *LinkNegotiator) Negotiate(gpuBDF uint32, prof GPUProfile, rootBDF uint32, targetGen uint32, allowStage2 bool) (*NegotiationResult, error) {
	// 1. eFuse Hardware Boundary Enforcement
	if prof.DeviceID == 0x2189 && targetGen > 2 {
		targetGen = 2
	}
	if prof.MaxSupportedGen > 0 && targetGen > prof.MaxSupportedGen {
		targetGen = prof.MaxSupportedGen
	}

	res := &NegotiationResult{
		TargetGen: targetGen,
	}

	cap := n.findPcieCap(gpuBDF)
	if cap == 0 {
		return res, ErrPCIeCapMissing
	}

	// 2. Tối ưu Device Control MRRS -> 512B (cap+0x08)
	if dctl, err := n.bus.ReadPCIConfig(gpuBDF, cap+0x08); err == nil {
		mrrs := (dctl >> 12) & 0x7
		if mrrs < 2 {
			newDctl := uint16((dctl &^ 0x7000) | (2 << 12)) // 0x2000 = 512 bytes
			_ = n.bus.WritePCIConfig(gpuBDF, cap+0x08, []byte{byte(newDctl), byte(newDctl >> 8)})
		}
	}

	// 3. BAR0 MMIO Injection (Shadow Registers)
	bar0Phys, err := n.resolveBAR0(gpuBDF)
	if err == nil && bar0Phys != 0 {
		if err := n.injectMMIOShadowRegisters(bar0Phys, prof, targetGen); err != nil {
			return res, err
		}
	}

	// 4. Thiết lập LNKCTL2 TLS mục tiêu trên GPU và Root
	n.setTLS(gpuBDF, cap, uint16(targetGen))
	if rootBDF != 0xFFFFFFFF {
		if rcap := n.findPcieCap(rootBDF); rcap != 0 {
			n.setTLS(rootBDF, rcap, uint16(targetGen))
		}
	}

	// 5. Khôi phục Common Clock Configuration & Vô hiệu hoá ASPM
	n.restoreLnkctl(gpuBDF, cap)
	if rootBDF != 0xFFFFFFFF {
		if rcap := n.findPcieCap(rootBDF); rcap != 0 {
			n.disableRootASPM(rootBDF, rcap)
		}
	}

	// 6. Stage 1: Retrain Pulses với Fast Polling 75ms
	cur := n.linkSpeed(gpuBDF, cap)
	for attempt := 0; attempt < 6; attempt++ {
		targetBDF := gpuBDF
		if attempt%2 == 0 && rootBDF != 0xFFFFFFFF {
			targetBDF = rootBDF
		}
		_ = n.retrainPulse(targetBDF)
		for poll := 0; poll < 25; poll++ {
			n.bus.Sleep(75 * time.Millisecond)
			cur = n.linkSpeed(gpuBDF, cap)
			if cur >= targetGen {
				break
			}
		}
		if cur >= targetGen {
			break
		}
	}

	// 7. Stage 2: PnP Soft Reset nếu Stage 1 chưa đạt và được phép
	if cur < targetGen && allowStage2 {
		res.Stage2Triggered = true
		if n.bus.PnpResetDevice(prof.DeviceID) {
			n.bus.Sleep(2 * time.Second)
			if bar0Phys != 0 {
				_ = n.injectMMIOShadowRegisters(bar0Phys, prof, targetGen)
			}
			n.setTLS(gpuBDF, cap, uint16(targetGen))
			if rootBDF != 0xFFFFFFFF {
				if rcap := n.findPcieCap(rootBDF); rcap != 0 {
					n.setTLS(rootBDF, rcap, uint16(targetGen))
				}
			}
			n.restoreLnkctl(gpuBDF, cap)
			for attempt := 0; attempt < 4; attempt++ {
				targetBDF := gpuBDF
				if attempt%2 == 0 && rootBDF != 0xFFFFFFFF {
					targetBDF = rootBDF
				}
				_ = n.retrainPulse(targetBDF)
				for poll := 0; poll < 25; poll++ {
					n.bus.Sleep(75 * time.Millisecond)
					cur = n.linkSpeed(gpuBDF, cap)
					if cur >= targetGen {
						break
					}
				}
				if cur >= targetGen {
					break
				}
			}
		}
	}

	// 8. Đọc lại trạng thái cuối cùng
	res.CurrentSpeed = cur
	res.CurrentWidth = n.linkWidth(gpuBDF, cap)
	if v, err := n.bus.ReadPCIConfig(gpuBDF, cap+0x30); err == nil {
		res.TargetTLS = v & 0xF
	}
	if rootBDF != 0xFFFFFFFF {
		if rcap := n.findPcieCap(rootBDF); rcap != 0 {
			if v, err := n.bus.ReadPCIConfig(rootBDF, rcap+0x30); err == nil {
				res.RootTLS = v & 0xF
			}
		}
	}

	if res.CurrentSpeed >= targetGen {
		res.Success = true
		res.Verdict = fmt.Sprintf("✅ Mở khoá Gen%d thành công: Băng thông hiện tại Gen%d x%d", targetGen, res.CurrentSpeed, res.CurrentWidth)
		_ = n.bus.RestartNVDisplay()
	} else if res.TargetTLS >= targetGen {
		res.Success = true
		res.Verdict = fmt.Sprintf("🟢 Gen%d đã cấu hình (TLS=Gen%d): Hiện đang Gen%d x%d do trạng thái tiết kiệm điện PCIe rảnh", targetGen, res.TargetTLS, res.CurrentSpeed, res.CurrentWidth)
	} else {
		res.Success = false
		res.Verdict = fmt.Sprintf("❌ Gen%d chưa đạt: Vẫn đang ở Gen%d x%d (GPU TLS=Gen%d, Root TLS=Gen%d)", targetGen, res.CurrentSpeed, res.CurrentWidth, res.TargetTLS, res.RootTLS)
	}

	return res, nil
}

func (n *LinkNegotiator) resolveBAR0(gpuBDF uint32) (uint64, error) {
	bar0raw, err := n.bus.ReadPCIConfig(gpuBDF, 0x10)
	if err != nil || bar0raw == 0 || bar0raw == 0xFFFFFFFF {
		return 0, ErrInvalidBAR0
	}
	return uint64(bar0raw & 0xFFFFFFF0), nil
}

func (n *LinkNegotiator) injectMMIOShadowRegisters(bar0Phys uint64, prof GPUProfile, targetGen uint32) error {
	boot0, err := n.bus.ReadMMIO(bar0Phys + 0x00)
	if err != nil {
		return err
	}
	fam := (boot0 >> 24) & 0xFF
	if fam != 0x16 && fam != 0x17 && fam != 0x21 {
		return ErrFamilyMismatch
	}

	seq := TU116ShadowSequence
	if prof.DeviceID == 0x1F0B {
		seq = TU106PL0Sequence
	}
	for _, reg := range seq {
		_ = n.bus.WriteMMIO(bar0Phys+reg.Offset, reg.Value)
	}

	if prof.DeviceID == 0x2189 { // TU116
		// LNKCAP (0x088084)
		if origCap, err := n.bus.ReadMMIO(bar0Phys + 0x00088084); err == nil {
			_ = n.bus.WriteMMIO(bar0Phys+0x00088084, (origCap&0xFFFFFFF0)|targetGen)
		}
		// LNKCAP2 (0x0880A4) & XVE_F0 (0x0880F0) -> speedVectorMask = 0x6 (Gen2)
		_ = n.bus.WriteMMIO(bar0Phys+0x000880A4, 0x00000006)
		_ = n.bus.WriteMMIO(bar0Phys+0x000880F0, 0x00000006)
		// LNKCTL2 (0x0880A8)
		if origCtl2, err := n.bus.ReadMMIO(bar0Phys + 0x000880A8); err == nil {
			_ = n.bus.WriteMMIO(bar0Phys+0x000880A8, (origCtl2&0xFFFFFFF0)|targetGen)
		}
	}
	return nil
}

func (n *LinkNegotiator) findPcieCap(bdf uint32) uint32 {
	hdr, err := n.bus.ReadPCIConfig(bdf, 0x34)
	if err != nil {
		return 0
	}
	cur := hdr & 0xFF
	for i := 0; i < 20; i++ {
		if cur < 0x40 || cur > 0xFF {
			return 0
		}
		c, err := n.bus.ReadPCIConfig(bdf, cur)
		if err != nil {
			return 0
		}
		if (c & 0xFF) == 0x10 {
			return cur
		}
		cur = (c >> 8) & 0xFF
	}
	return 0
}

func (n *LinkNegotiator) linkSpeed(bdf uint32, cap uint32) uint32 {
	if cap == 0 {
		cap = n.findPcieCap(bdf)
	}
	if cap == 0 {
		return 0
	}
	v, err := n.bus.ReadPCIConfig(bdf, cap+0x12)
	if err != nil {
		return 0
	}
	return v & 0xF
}

func (n *LinkNegotiator) linkWidth(bdf uint32, cap uint32) uint32 {
	if cap == 0 {
		cap = n.findPcieCap(bdf)
	}
	if cap == 0 {
		return 0
	}
	v, err := n.bus.ReadPCIConfig(bdf, cap+0x12)
	if err != nil {
		return 0
	}
	return (v >> 4) & 0x3F
}

func (n *LinkNegotiator) setTLS(bdf uint32, cap uint32, tls uint16) {
	if curRaw, err := n.bus.ReadPCIConfig(bdf, cap+0x30); err == nil {
		nv := uint16(curRaw&0xFFF0) | (tls & 0xF)
		_ = n.bus.WritePCIConfig(bdf, cap+0x30, []byte{byte(nv), byte(nv >> 8)})
	}
}

func (n *LinkNegotiator) restoreLnkctl(gpuBDF uint32, cap uint32) {
	if cur, err := n.bus.ReadPCIConfig(gpuBDF, cap+0x10); err == nil {
		w := uint16(cur & 0xFFFF)
		w &^= 0x0003 // Tắt ASPM
		w |= 0x0140  // Common Clock + Extended Synch
		_ = n.bus.WritePCIConfig(gpuBDF, cap+0x10, []byte{byte(w), byte(w >> 8)})
	}
}

func (n *LinkNegotiator) disableRootASPM(rootBDF uint32, rcap uint32) {
	if rctl, err := n.bus.ReadPCIConfig(rootBDF, rcap+0x10); err == nil {
		curRootCtl := uint16(rctl & 0xFFFF)
		if (curRootCtl&0x0140) != 0x0140 || (curRootCtl&0x3) != 0 {
			wantRoot := (curRootCtl &^ 0x3) | 0x0140
			_ = n.bus.WritePCIConfig(rootBDF, rcap+0x10, []byte{byte(wantRoot), byte(wantRoot >> 8)})
		}
	}
}

func (n *LinkNegotiator) retrainPulse(bdf uint32) error {
	cap := n.findPcieCap(bdf)
	if cap == 0 {
		return ErrPCIeCapMissing
	}
	ctl, err := n.bus.ReadPCIConfig(bdf, cap+0x10)
	if err != nil {
		return err
	}
	lo := uint16(ctl & 0xFFFF)
	clear := []byte{byte(lo &^ 0x20), byte(lo >> 8)}
	if err := n.bus.WritePCIConfig(bdf, cap+0x10, clear); err != nil {
		return err
	}
	n.bus.Sleep(50 * time.Millisecond)
	ctl2, err := n.bus.ReadPCIConfig(bdf, cap+0x10)
	if err != nil {
		return err
	}
	set := uint16(ctl2&0xFFFF) | 0x20
	return n.bus.WritePCIConfig(bdf, cap+0x10, []byte{byte(set), byte(set >> 8)})
}

// -------------------------------------------------------------
// MockHardwareBus: Adapter phục vụ unit-test không cần GPU thật
// -------------------------------------------------------------

type MockHardwareBus struct {
	pciConfig              map[uint64]uint32
	mmio                   map[uint64]uint32
	PnpResetCalled         bool
	OnPnpReset             func(devID uint16) bool
	RestartNVDisplayCalled bool
	OnRestartNVDisplay     func() error
}

func NewMockHardwareBus() *MockHardwareBus {
	return &MockHardwareBus{
		pciConfig: make(map[uint64]uint32),
		mmio:      make(map[uint64]uint32),
	}
}

func (m *MockHardwareBus) SetPCIConfig(bdf uint32, reg uint32, val uint32) {
	key := (uint64(bdf) << 32) | uint64(reg&^3)
	shift := (reg & 3) * 8
	if shift > 0 {
		mask := uint32(0xFFFF) << shift
		m.pciConfig[key] = (m.pciConfig[key] &^ mask) | ((val & 0xFFFF) << shift)
	} else {
		m.pciConfig[key] = val
	}
}

func (m *MockHardwareBus) SetPCICap(bdf uint32, capOffset uint32) {
	// Set Cap pointer at 0x34
	m.SetPCIConfig(bdf, 0x34, capOffset)
	// Set PCIe Cap header (ID=0x10)
	m.SetPCIConfig(bdf, capOffset, 0x00000010)
}

func (m *MockHardwareBus) SetMMIO(physAddr uint64, val uint32) {
	m.mmio[physAddr] = val
}

func (m *MockHardwareBus) ReadPCIConfig(bdf uint32, reg uint32) (uint32, error) {
	key := (uint64(bdf) << 32) | uint64(reg&^3)
	v, ok := m.pciConfig[key]
	if !ok {
		return 0, nil
	}
	shift := (reg & 3) * 8
	return v >> shift, nil
}

func (m *MockHardwareBus) WritePCIConfig(bdf uint32, reg uint32, data []byte) error {
	key := (uint64(bdf) << 32) | uint64(reg&^3)
	cur := m.pciConfig[key]
	shift := (reg & 3) * 8
	mask := uint32(0)
	val := uint32(0)
	for i, b := range data {
		bitShift := shift + uint32(i*8)
		if bitShift < 32 {
			mask |= 0xFF << bitShift
			val |= uint32(b) << bitShift
		}
	}
	m.pciConfig[key] = (cur &^ mask) | val
	return nil
}

func (m *MockHardwareBus) ReadMMIO(physAddr uint64) (uint32, error) {
	v, ok := m.mmio[physAddr]
	if !ok {
		return 0, nil
	}
	return v, nil
}

func (m *MockHardwareBus) WriteMMIO(physAddr uint64, val uint32) error {
	m.mmio[physAddr] = val
	return nil
}

func (m *MockHardwareBus) PnpResetDevice(devID uint16) bool {
	m.PnpResetCalled = true
	if m.OnPnpReset != nil {
		return m.OnPnpReset(devID)
	}
	return true
}

func (m *MockHardwareBus) RestartNVDisplay() error {
	m.RestartNVDisplayCalled = true
	if m.OnRestartNVDisplay != nil {
		return m.OnRestartNVDisplay()
	}
	return nil
}

func (m *MockHardwareBus) Sleep(d time.Duration) {
	// Fast simulation: No actual sleep during unit tests
}

// -------------------------------------------------------------
// ProductionBus: Adapter thực tế qua WinRing0 và ThrottleStop
// -------------------------------------------------------------

type ProductionBus struct {
	wh syscall.Handle
	th syscall.Handle
}

func NewProductionBus(wh, th syscall.Handle) *ProductionBus {
	return &ProductionBus{wh: wh, th: th}
}

func (p *ProductionBus) ReadPCIConfig(bdf uint32, reg uint32) (uint32, error) {
	return PciRd(p.wh, bdf, reg)
}

func (p *ProductionBus) WritePCIConfig(bdf uint32, reg uint32, data []byte) error {
	return PciWr(p.wh, bdf, reg, data)
}

func (p *ProductionBus) ReadMMIO(physAddr uint64) (uint32, error) {
	if p.th == 0 {
		return 0, errors.New("ThrottleStop handle is zero")
	}
	return TSRead(p.th, physAddr)
}

func (p *ProductionBus) WriteMMIO(physAddr uint64, val uint32) error {
	if p.th == 0 {
		return errors.New("ThrottleStop handle is zero")
	}
	return TSWrite(p.th, physAddr, val)
}

func (p *ProductionBus) PnpResetDevice(devID uint16) bool {
	cmd := fmt.Sprintf("$devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'DEV_%04X' }; foreach ($d in $devs) { try { & pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}; try { Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 800; Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } catch {}; try { $st = (Get-PnpDevice -InstanceId $d.InstanceId -ErrorAction SilentlyContinue).Status; if ($st -ne 'OK') { Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } } catch {} }", devID)
	_, err := RunOut("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
	return err == nil
}

func (p *ProductionBus) RestartNVDisplay() error {
	// 1. Đảm bảo cấu hình service là auto để không bị vô hiệu hoá
	_, _ = RunOut("sc.exe", "config", "NVDisplay.ContainerLocalSystem", "start=", "auto")

	// 2. Yêu cầu dừng service
	out, _ := RunOut("sc.exe", "stop", "NVDisplay.ContainerLocalSystem")
	if strings.Contains(string(out), "1060") {
		// Service không tồn tại trên hệ thống (không có driver NVIDIA)
		return nil
	}

	// 3. Đợi service dừng hoàn toàn (chuyển sang trạng thái 1 STOPPED) thay vì sleep mù 500ms
	for i := 0; i < 25; i++ { // tối đa 5 giây
		time.Sleep(200 * time.Millisecond)
		qOut, err := RunOut("sc.exe", "query", "NVDisplay.ContainerLocalSystem")
		if err != nil || strings.Contains(string(qOut), "STOPPED") {
			break
		}
	}

	// 4. Khởi động lại service
	_, err := RunOut("sc.exe", "start", "NVDisplay.ContainerLocalSystem")

	// 5. Xác nhận service đã ở trạng thái 4 RUNNING; nếu gặp lỗi 1056 (đang chuyển trạng thái) thì thử lại
	for i := 0; i < 25; i++ { // tối đa 5 giây
		time.Sleep(200 * time.Millisecond)
		qOut, qErr := RunOut("sc.exe", "query", "NVDisplay.ContainerLocalSystem")
		if qErr == nil && strings.Contains(string(qOut), "RUNNING") {
			return nil
		}
		if strings.Contains(string(qOut), "STOPPED") {
			_, err = RunOut("sc.exe", "start", "NVDisplay.ContainerLocalSystem")
		}
	}
	return err
}

func (p *ProductionBus) Sleep(d time.Duration) {
	time.Sleep(d)
}
