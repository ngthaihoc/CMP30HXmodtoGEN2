package hxcore

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

var (
	ErrFamilyMismatch    = errors.New("BAR0 BOOT_0 family mismatch")
	ErrPCIeCapMissing    = errors.New("PCIe capability missing")
	ErrInvalidBAR0       = errors.New("invalid BAR0 physical address")
	ErrRootPortIncapable = errors.New("root port does not support target speed")
)

// HardwareBus: Điểm phân tách (seam) giao tiếp I/O phần cứng vật lý cấp thấp (6 phương thức cốt lõi)
type HardwareBus interface {
	ReadPCIConfig(bdf uint32, reg uint32) (uint32, error)
	WritePCIConfig(bdf uint32, reg uint32, data []byte) error
	ReadMMIO(physAddr uint64) (uint32, error)
	WriteMMIO(physAddr uint64, val uint32) error
	PnpResetDevice(devID uint16) bool
	RestartNVDisplay() error
}

// MMIORegWrite: Bản ghi thiết lập thanh ghi MMIO chuẩn hoá
type MMIORegWrite struct {
	Offset        uint64
	Value         uint32
	Name          string
	RMW           bool
	Mask          uint32
	FallbackValue uint32
}

// ComputeValue tính toán giá trị cuối cùng cần ghi vào thanh ghi MMIO.
// Nếu RMW=true, đọc giá trị hiện tại qua readFn rồi áp dụng bitmask.
// Nếu đọc thất bại, sử dụng FallbackValue (hoặc Value nếu không có fallback).
func (r MMIORegWrite) ComputeValue(readFn func(offset uint64) (uint32, error)) uint32 {
	if !r.RMW {
		return r.Value
	}
	old, err := readFn(r.Offset)
	if err != nil {
		if r.FallbackValue != 0 {
			return r.FallbackValue
		}
		return r.Value
	}
	return (old &^ r.Mask) | r.Value
}

// TU116ShadowSequence: Đơn vị duy nhất định nghĩa chuỗi thanh ghi mở khoá shadow cho TU116
var TU116ShadowSequence = []MMIORegWrite{
	{Offset: 0x0008841C, Value: 0xE0B42D00, Name: "PRIV_MISC_1"},
	{Offset: 0x0008872C, Value: 6, Name: "XVE_OVR"},
	{Offset: 0x0008C040, Value: 0x80085800, Name: "LINK_CONFIG_0"},
	{Offset: 0x0008C1C0, Value: 0x00240036, Name: "PL_LINK_RATE"},
	{Offset: 0x0008C2C0, Value: 0x068731B3, Name: "CYA_0"},
	{Offset: 0x0008872C, Value: 6, Name: "XVE_OVR_CONFIRM"},
}

// TU106PL0Sequence: Chuỗi thanh ghi mở khoá cho TU106 (CMP 40HX)
var TU106PL0Sequence = []MMIORegWrite{
	{Offset: 0x8872C, Value: 0x6, Name: "XVE_OVR=6"},
	{Offset: 0x8C040, Value: 0x00080000, Name: "LINK_CONFIG_0", RMW: true, Mask: 0x000C0000, FallbackValue: 0x80085800},
	{Offset: 0x8841C, Value: 0xE0B42D00, Name: "PRIV_MISC_1"},
	{Offset: 0x8C1C0, Value: 0x00040000, Name: "PL_LINK_RATE", RMW: true, Mask: 0x00060000, FallbackValue: 0x00240036},
	{Offset: 0x8C2C0, Value: 0x068731B3, Name: "CYA_0"},
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

// NegotiationOptions thiết lập tham số cho quy trình đàm phán hoàn chỉnh
type NegotiationOptions struct {
	TargetGen    uint32
	AllowGen3    bool
	ForceRoot    bool
	AllowStage2  bool
	UserIsAdmin  bool
}

// NegotiationVerdict trả về kết quả đàm phán hoàn chỉnh kèm StatusContract và chỉ dẫn Retry
type NegotiationVerdict struct {
	Success          bool
	CurrentSpeed     uint32
	CurrentWidth     uint32
	TargetGen        uint32
	Verdict          string
	StatusContract   StatusContract
	NeedsRetry       bool
	RetryAdvice      string
	DiagnosticReport string
}

// LinkNegotiator: Deep Module điều khiển huấn luyện lại PCIe và tối ưu DMA
type LinkNegotiator struct {
	bus     HardwareBus
	sleepFn func(time.Duration)
}

// NewLinkNegotiator khởi tạo LinkNegotiator với HardwareBus adapter
func NewLinkNegotiator(bus HardwareBus) *LinkNegotiator {
	n := &LinkNegotiator{
		bus:     bus,
		sleepFn: time.Sleep,
	}
	if _, isMock := bus.(*MockHardwareBus); isMock {
		n.sleepFn = func(time.Duration) {} // Fast simulation for unit tests
	}
	return n
}

// SetSleepFn cho phép tùy biến hàm delay (phục vụ mô phỏng test hoặc tốc độ cao)
func (n *LinkNegotiator) SetSleepFn(fn func(time.Duration)) {
	if fn != nil {
		n.sleepFn = fn
	}
}

func (n *LinkNegotiator) sleep(d time.Duration) {
	if n.sleepFn != nil {
		n.sleepFn(d)
	}
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
	// CMP 30HX (TU116) has laser-cut eFuse and cannot tolerate Link Disable or PnP resets.
	// Hardware Rule 3: No dangerous resets on 30HX. Force allowStage2 to false.
	if prof.DeviceID == 0x2189 || prof.Family == "TU116" {
		allowStage2 = false
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

	// 2.5. Đảm bảo PCI Command Register bật Memory Space Enable (bit 1) & Bus Master Enable (bit 2)
	// Tránh trường hợp GPU phụ đang ở D3/ngủ sâu làm vô hiệu hóa bộ giải mã BAR0 MMIO
	if cmd, err := n.bus.ReadPCIConfig(gpuBDF, 0x04); err == nil {
		if (cmd & 0x06) != 0x06 {
			newCmd := uint16(cmd | 0x06)
			_ = n.bus.WritePCIConfig(gpuBDF, 0x04, []byte{byte(newCmd), byte(newCmd >> 8)})
		}
	}

	// 3. BAR0 MMIO Injection (Shadow Registers)
	bar0Phys, err := n.resolveBAR0(gpuBDF)
	if err == nil && bar0Phys != 0 {
		if err := n.injectMMIOShadowRegisters(bar0Phys, prof, targetGen); err != nil {
			if errors.Is(err, ErrFamilyMismatch) {
				return res, err
			}
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
	cur := n.pollRetrain(gpuBDF, rootBDF, cap, targetGen)

	// 7. Stage 2: Root Link Disable + PnP Soft Reset nếu Stage 1 chưa đạt và được phép
	if cur < targetGen && allowStage2 {
		res.Stage2Triggered = true

		// Đối với các profile không phải TU116 (như CMP 40HX TU106), thực hiện Root Link Disable trước
		if prof.DeviceID != 0x2189 && prof.Family != "TU116" && rootBDF != 0xFFFFFFFF {
			n.rootLinkDisable(rootBDF, gpuBDF, bar0Phys, prof, targetGen)
			cur = n.pollRetrain(gpuBDF, rootBDF, cap, targetGen)
		}

		if cur < targetGen && n.bus.PnpResetDevice(prof.DeviceID) {
			n.sleep(2 * time.Second)
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
			cur = n.pollRetrain(gpuBDF, rootBDF, cap, targetGen)
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

func (n *LinkNegotiator) rootLinkDisable(rootBDF uint32, gpuBDF uint32, bar0Phys uint64, prof GPUProfile, targetGen uint32) {
	rcap := n.findPcieCap(rootBDF)
	if rcap == 0 {
		return
	}
	ctl, err := n.bus.ReadPCIConfig(rootBDF, rcap+0x10)
	if err != nil {
		return
	}
	lo := uint16(ctl & 0xFFFF)
	set := lo | 0x10 // bit 4 = Link Disable
	_ = n.bus.WritePCIConfig(rootBDF, rcap+0x10, []byte{byte(set), byte(set >> 8)})
	n.sleep(500 * time.Millisecond)

	// Cấu hình TLS phía Root Port trong khi link đang tạm ngắt
	n.setTLS(rootBDF, rcap, uint16(targetGen))

	// Bật lại link phía Root Port
	ctl2, err := n.bus.ReadPCIConfig(rootBDF, rcap+0x10)
	if err == nil {
		clr := uint16(ctl2&0xFFFF) &^ 0x10
		_ = n.bus.WritePCIConfig(rootBDF, rcap+0x10, []byte{byte(clr), byte(clr >> 8)})
	}
	n.sleep(2 * time.Second)

	// Sau khi link đã hoạt động trở lại, tiến hành nạp lại thanh ghi MMIO shadow và TLS cho GPU endpoint
	if bar0Phys != 0 {
		_ = n.injectMMIOShadowRegisters(bar0Phys, prof, targetGen)
	}
	if gcap := n.findPcieCap(gpuBDF); gcap != 0 {
		n.setTLS(gpuBDF, gcap, uint16(targetGen))
		n.restoreLnkctl(gpuBDF, gcap)
	}
	n.disableRootASPM(rootBDF, rcap)
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
	if prof.Family == "TU106" && fam != 0x16 {
		return ErrFamilyMismatch
	}
	if prof.Family == "TU116" && fam != 0x16 && fam != 0x17 && fam != 0x21 {
		return ErrFamilyMismatch
	}
	if fam != 0x16 && fam != 0x17 && fam != 0x21 {
		return ErrFamilyMismatch
	}

	seq := TU116ShadowSequence
	if prof.DeviceID == 0x1F0B || prof.Family == "TU106" {
		seq = TU106PL0Sequence
	}
	for _, reg := range seq {
		val := reg.ComputeValue(func(off uint64) (uint32, error) {
			return n.bus.ReadMMIO(bar0Phys + off)
		})
		_ = n.bus.WriteMMIO(bar0Phys+reg.Offset, val)
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

func (n *LinkNegotiator) pollRetrain(gpuBDF, rootBDF, cap, targetGen uint32) uint32 {
	cur := n.linkSpeed(gpuBDF, cap)
	for attempt := 0; attempt < 6; attempt++ {
		targetBDF := gpuBDF
		if attempt%2 == 0 && rootBDF != 0xFFFFFFFF {
			targetBDF = rootBDF
		}
		_ = n.retrainPulse(targetBDF)
		for poll := 0; poll < 25; poll++ {
			n.sleep(75 * time.Millisecond)
			cur = n.linkSpeed(gpuBDF, cap)
			if cur >= targetGen {
				break
			}
		}
		if cur >= targetGen {
			break
		}
	}
	return cur
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
	n.sleep(50 * time.Millisecond)
	ctl2, err := n.bus.ReadPCIConfig(bdf, cap+0x10)
	if err != nil {
		return err
	}
	set := uint16(ctl2&0xFFFF) | 0x20
	return n.bus.WritePCIConfig(bdf, cap+0x10, []byte{byte(set), byte(set >> 8)})
}

func (n *LinkNegotiator) pcieMaxSpeed(bdf uint32) uint32 {
	cap := n.findPcieCap(bdf)
	if cap == 0 {
		return 0
	}
	v, err := n.bus.ReadPCIConfig(bdf, cap+0x0C)
	if err != nil {
		return 0
	}
	return v & 0xF
}

func (n *LinkNegotiator) findRootPort(gpuBus uint32) uint32 {
	for d := uint32(0); d < 32; d++ {
		for f := uint32(0); f < 8; f++ {
			bdf := (0 << 8) | (d << 3) | f
			id, err := n.bus.ReadPCIConfig(bdf, 0x00)
			if err != nil || id == 0xFFFFFFFF || (id&0xFFFF) == 0 {
				continue
			}
			cls, _ := n.bus.ReadPCIConfig(bdf, 0x08)
			if ((cls >> 16) & 0xFFFF) != 0x0604 {
				continue
			}
			sec, _ := n.bus.ReadPCIConfig(bdf, 0x18)
			if ((sec >> 8) & 0xFF) == gpuBus {
				return bdf
			}
		}
	}
	return 0xFFFFFFFF
}

// ExecuteNegotiation: Deep API thực hiện toàn bộ quy trình từ kiểm tra năng lực,
// đàm phán link, khôi phục Stage 2, đến xuất bản StatusContract hoàn chỉnh và tư vấn retry.
func (n *LinkNegotiator) ExecuteNegotiation(gpuBDF uint32, prof GPUProfile, opts NegotiationOptions) (*NegotiationVerdict, error) {
	targetGen := opts.TargetGen
	if targetGen == 0 {
		targetGen = 2
	}
	if prof.DeviceID == 0x2189 && targetGen > 2 {
		targetGen = 2
	}
	if prof.MaxSupportedGen > 0 && targetGen > prof.MaxSupportedGen {
		targetGen = prof.MaxSupportedGen
	}

	gpuBus := (gpuBDF >> 8) & 0xFF
	cap := n.findPcieCap(gpuBDF)
	if cap == 0 {
		st := StatusContract{
			StatusCode: StatusDrvFail,
			ErrorCode:  "PCIE_CAP_MISSING",
			Details:    []string{"Không tìm thấy PCIe Capability trên GPU"},
		}
		return &NegotiationVerdict{
			Success:        false,
			TargetGen:      targetGen,
			Verdict:        "Không tìm thấy PCIe Capability",
			StatusContract: st,
			NeedsRetry:     false,
		}, ErrPCIeCapMissing
	}

	cur := n.linkSpeed(gpuBDF, cap)
	curWidth := n.linkWidth(gpuBDF, cap)

	// 1. Kiểm tra trạng thái đã đạt sẵn (Fast-path)
	if cur >= targetGen {
		st := StatusContract{
			StatusCode:   StatusGen2Success,
			SpeedCurrent: cur,
			WidthCurrent: curWidth,
			TLSTarget:    targetGen,
			ErrorCode:    "NONE",
			Details: []string{
				fmt.Sprintf("Kết luận: ✅ Gen%d không cần thao tác: Băng thông hiện tại đã là Gen%d", targetGen, cur),
				fmt.Sprintf("Vị trí %s: %02x:%02x.%x", prof.Name, gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7),
				fmt.Sprintf("đã đạt mục tiêu Gen%d thành công", targetGen),
			},
		}
		return &NegotiationVerdict{
			Success:        true,
			CurrentSpeed:   cur,
			CurrentWidth:   curWidth,
			TargetGen:      targetGen,
			Verdict:        fmt.Sprintf("PCIe đã đạt Gen%d, không cần thao tác thêm", cur),
			StatusContract: st,
			NeedsRetry:     false,
		}, nil
	}

	// 2. Tra cứu Root Port và giới hạn tốc độ phần cứng
	rootBDF := n.findRootPort(gpuBus)
	gpuMax := n.pcieMaxSpeed(gpuBDF)
	rootMax := gpuMax
	if rootBDF != 0xFFFFFFFF {
		rootMax = n.pcieMaxSpeed(rootBDF)
	}

	allowTarget := LinkTargetAllowed(gpuMax, rootMax, prof.MaxSupportedGen, targetGen)
	// Đối với card CMP (RequiresMMIO / HasSafePL0 như CMP 40HX & 30HX), LNKCAP ban đầu bị khoá ở Gen1 khi chưa nạp thanh ghi Shadow.
	// Miễn là Root Port và Profile hỗ trợ targetGen, cho phép nạp MMIO Shadow và đàm phán link thay vì dừng lại.
	canUnlockViaMMIO := (prof.RequiresMMIO || prof.HasSafePL0) && rootMax >= targetGen && prof.MaxSupportedGen >= targetGen
	if !allowTarget && !opts.ForceRoot && !canUnlockViaMMIO {
		diagMsg := fmt.Sprintf("Phần cứng hoặc Profile không hỗ trợ Gen%d (GPU Max=%d, Root Max=%d, Cap=%d)", targetGen, gpuMax, rootMax, prof.MaxSupportedGen)
		st := StatusContract{
			StatusCode:   StatusHwLimit,
			SpeedCurrent: cur,
			WidthCurrent: curWidth,
			TLSTarget:    targetGen,
			ErrorCode:    "GEN2_HARDWARE_LIMIT",
			Details: []string{
				diagMsg,
				fmt.Sprintf("GPU Max: Gen%d | Root Port Max: Gen%d", gpuMax, rootMax),
			},
		}
		return &NegotiationVerdict{
			Success:          false,
			CurrentSpeed:     cur,
			CurrentWidth:     curWidth,
			TargetGen:        targetGen,
			Verdict:          diagMsg,
			StatusContract:   st,
			NeedsRetry:       false,
			DiagnosticReport: diagMsg,
		}, nil
	}

	// 3. Thực hiện chuỗi đàm phán & huấn luyện lại link
	allowStage2 := opts.AllowStage2
	if prof.DeviceID == 0x2189 || prof.Family == "TU116" {
		allowStage2 = false
	}
	res, err := n.Negotiate(gpuBDF, prof, rootBDF, targetGen, allowStage2)
	if err != nil {
		st := StatusContract{
			StatusCode:   StatusDrvFail,
			SpeedCurrent: cur,
			WidthCurrent: curWidth,
			TLSTarget:    targetGen,
			ErrorCode:    "NEGOTIATION_ERROR",
			Details:      []string{fmt.Sprintf("Lỗi thương lượng link: %v", err)},
		}
		return &NegotiationVerdict{
			Success:        false,
			CurrentSpeed:   cur,
			CurrentWidth:   curWidth,
			TargetGen:      targetGen,
			Verdict:        fmt.Sprintf("Lỗi thương lượng link: %v", err),
			StatusContract: st,
			NeedsRetry:     true,
		}, err
	}

	statusCode := StatusGen1Stuck
	if res.Success {
		statusCode = StatusGen2Success
	}
	st := StatusContract{
		StatusCode:   statusCode,
		SpeedCurrent: res.CurrentSpeed,
		WidthCurrent: res.CurrentWidth,
		TLSTarget:    res.TargetTLS,
		ErrorCode:    "NONE",
		Details: []string{
			fmt.Sprintf("Kết luận: %s", res.Verdict),
			fmt.Sprintf("Vị trí: %02x:%02x.%x | TLS: Gen%d", gpuBus, (gpuBDF>>3)&0x1F, gpuBDF&7, res.TargetTLS),
		},
	}

	return &NegotiationVerdict{
		Success:          res.Success,
		CurrentSpeed:     res.CurrentSpeed,
		CurrentWidth:     res.CurrentWidth,
		TargetGen:        res.TargetGen,
		Verdict:          res.Verdict,
		StatusContract:   st,
		NeedsRetry:       !res.Success,
		DiagnosticReport: res.DiagnosticReport,
	}, nil
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
	OnWriteMMIO            func(physAddr uint64, val uint32)
	OnWritePCIConfig       func(bdf uint32, reg uint32, data []byte)
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
	if m.OnWritePCIConfig != nil {
		m.OnWritePCIConfig(bdf, reg, data)
	}
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
	if m.OnWriteMMIO != nil {
		m.OnWriteMMIO(physAddr, val)
	}
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

func (m *MockHardwareBus) LinkSpeed(bdf uint32) uint32 {
	cap := m.PcieCap(bdf)
	if cap == 0 {
		return 0
	}
	v, err := m.ReadPCIConfig(bdf, cap+0x12)
	if err != nil {
		return 0
	}
	return v & 0xF
}

func (m *MockHardwareBus) LinkWidth(bdf uint32) uint32 {
	cap := m.PcieCap(bdf)
	if cap == 0 {
		return 0
	}
	v, err := m.ReadPCIConfig(bdf, cap+0x12)
	if err != nil {
		return 0
	}
	return (v >> 4) & 0x3F
}

func (m *MockHardwareBus) PcieCap(bdf uint32) uint32 {
	hdr, err := m.ReadPCIConfig(bdf, 0x34)
	if err != nil {
		return 0
	}
	cur := hdr & 0xFF
	for i := 0; i < 20; i++ {
		if cur < 0x40 || cur > 0xFF {
			return 0
		}
		c, err := m.ReadPCIConfig(bdf, cur)
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

func (m *MockHardwareBus) PcieMaxSpeed(bdf uint32) uint32 {
	cap := m.PcieCap(bdf)
	if cap == 0 {
		return 0
	}
	v, err := m.ReadPCIConfig(bdf, cap+0x0C)
	if err != nil {
		return 0
	}
	return v & 0xF
}

func (m *MockHardwareBus) FindRootPort(gpuBus uint32) uint32 {
	for d := uint32(0); d < 32; d++ {
		for f := uint32(0); f < 8; f++ {
			bdf := (0 << 8) | (d << 3) | f
			id, err := m.ReadPCIConfig(bdf, 0x00)
			if err != nil || id == 0xFFFFFFFF || (id&0xFFFF) == 0 {
				continue
			}
			cls, _ := m.ReadPCIConfig(bdf, 0x08)
			if ((cls >> 16) & 0xFFFF) != 0x0604 {
				continue
			}
			sec, _ := m.ReadPCIConfig(bdf, 0x18)
			if ((sec >> 8) & 0xFF) == gpuBus {
				return bdf
			}
		}
	}
	return 0xFFFFFFFF
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
	if p.th != 0 {
		return TSRead(p.th, physAddr)
	}
	if p.wh != 0 {
		return WRReadMem(p.wh, physAddr)
	}
	return 0, errors.New("ThrottleStop handle is zero")
}

func (p *ProductionBus) WriteMMIO(physAddr uint64, val uint32) error {
	if p.th != 0 {
		return TSWrite(p.th, physAddr, val)
	}
	if p.wh != 0 {
		return WRWriteMem(p.wh, physAddr, val)
	}
	return errors.New("ThrottleStop handle is zero")
}

func (p *ProductionBus) PnpResetDevice(devID uint16) bool {
	devInfo, err := windows.SetupDiGetClassDevsEx(nil, "", 0, windows.DIGCF_ALLCLASSES|windows.DIGCF_PRESENT, 0, "")
	if err != nil {
		return false
	}
	defer windows.SetupDiDestroyDeviceInfoList(devInfo)

	targetDev := fmt.Sprintf("DEV_%04X", devID)
	success := false

	for i := 0; ; i++ {
		devInfoData, err := windows.SetupDiEnumDeviceInfo(devInfo, i)
		if err != nil {
			break
		}

		id, err := windows.SetupDiGetDeviceInstanceId(devInfo, devInfoData)
		if err != nil {
			continue
		}

		if strings.Contains(strings.ToUpper(id), targetDev) {
			propChange := windows.PropChangeParams{
				ClassInstallHeader: *windows.MakeClassInstallHeader(windows.DIF_PROPERTYCHANGE),
				StateChange:        windows.DICS_PROPCHANGE,
				Scope:              windows.DICS_FLAG_GLOBAL,
				HwProfile:          0,
			}

			err = windows.SetupDiSetClassInstallParams(devInfo, devInfoData, &propChange.ClassInstallHeader, uint32(unsafe.Sizeof(propChange)))
			if err == nil {
				err = windows.SetupDiCallClassInstaller(windows.DIF_PROPERTYCHANGE, devInfo, devInfoData)
				if err == nil {
					success = true
				} else {
					// Fallback to Disable then Enable
					propChange.StateChange = windows.DICS_DISABLE
					windows.SetupDiSetClassInstallParams(devInfo, devInfoData, &propChange.ClassInstallHeader, uint32(unsafe.Sizeof(propChange)))
					windows.SetupDiCallClassInstaller(windows.DIF_PROPERTYCHANGE, devInfo, devInfoData)

					time.Sleep(800 * time.Millisecond)

					propChange.StateChange = windows.DICS_ENABLE
					windows.SetupDiSetClassInstallParams(devInfo, devInfoData, &propChange.ClassInstallHeader, uint32(unsafe.Sizeof(propChange)))
					if err := windows.SetupDiCallClassInstaller(windows.DIF_PROPERTYCHANGE, devInfo, devInfoData); err == nil {
						success = true
					}
				}
			}
		}
	}
	return success
}

func (p *ProductionBus) RestartNVDisplay() error {
	err := withService("NVDisplay.ContainerLocalSystem", func(s *mgr.Service) error {
		// 1. Đảm bảo cấu hình service là auto để không bị vô hiệu hoá
		conf, err := s.Config()
		if err == nil && conf.StartType != mgr.StartAutomatic {
			conf.StartType = mgr.StartAutomatic
			s.UpdateConfig(conf)
		}

		// 2. Yêu cầu dừng service
		st, _ := s.Query()
		if st.State != svc.Stopped {
			s.Control(svc.Stop)
		}

		// 3. Đợi service dừng hoàn toàn
		for i := 0; i < 25; i++ { // tối đa 5 giây
			time.Sleep(200 * time.Millisecond)
			st, err := s.Query()
			if err != nil || st.State == svc.Stopped {
				break
			}
		}

		// 4. Khởi động lại service
		s.Start()

		// 5. Xác nhận service đã ở trạng thái 4 RUNNING
		for i := 0; i < 25; i++ { // tối đa 5 giây
			time.Sleep(200 * time.Millisecond)
			st, err := s.Query()
			if err == nil && st.State == svc.Running {
				_ = registerNvCplContextMenu()
				return nil
			}
			if st.State == svc.Stopped {
				s.Start()
			}
		}
		_ = registerNvCplContextMenu()
		return nil
	})
	if err != nil {
		// Service không tồn tại trên hệ thống
		return nil
	}
	return nil
}
