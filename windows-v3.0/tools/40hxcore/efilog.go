package hxcore

import (
	"os"
	"strings"
)

// AnalyzeEfiLog: 读 ESP 根目录 40hx_log.txt (EFI 解锁链日志), 按失败特征
// 分类给出精确解决步骤。SS0 锁定时调用; 返回诊断建议字符串(可多行)。
//
// v2.4.4: 社区 #1 (X99 双卡 code43) 教训 — 相同症状可能来自不同根因:
//   A. "WPR2 NOT up" + "IMEM[0]=0xffffffff"  → GPU DMA 够不到 >4GB 载荷
//      = 主板 Above 4G Decoding 未开 (开发机 Z170 开了才成功)
//   B. "not-OK (m0=0x89 halted)" → booter 注入失败 (时序/槽位问题)
//   C. 日志显示已 UNLOCKED 但 SS0 锁 → 驱动层覆盖(重跑安装器)
//   D. 无日志 → EFI 没跑 (启动项未置顶/Secure Boot)
func AnalyzeEfiLog() string {
	esp := MountESP()
	if esp == "" {
		return "  [Nhật ký EFI] Không thể gắn phân vùng ESP (cần quyền Admin) — Không đọc được nhật ký mở khoá"
	}
	defer UnmountESP(esp)
	efiPath := esp + `:\EFI\40HX\40HXUNLK.EFI`
	_, efiErr := os.Stat(efiPath)
	p := esp + ":\\40hx_log.txt"
	data, err := os.ReadFile(p)
	if err != nil {
		if efiErr != nil {
			return "  [Nhật ký EFI] EFI mở khoá chưa cài đặt / đã gỡ bỏ (trên ESP không có \\EFI\\40HX\\40HXUNLK.EFI, cũng không có 40hx_log.txt)\n  Hiệu năng giữ khoá là bình thường; Để mở khoá hiệu năng: Mở 40HXInstaller.exe chọn [Cài đặt EFI hiệu năng + Mục khởi động firmware]"
		}
		return "  [Nhật ký EFI] Trên ESP không có 40hx_log.txt — EFI có thể chưa từng được thực thi\n  Vui lòng vào BIOS: Đặt '40HX Unlock' làm mục khởi động đầu tiên hoặc tắt Secure Boot"
	}
	if efiErr != nil {
		return "  [Nhật ký EFI] Lưu ý: 40hx_log.txt tại gốc ESP là dữ liệu cũ còn sót lại — EFI mở khoá không còn tồn tại\n  Nhật ký cũ không đại diện cho trạng thái hiện tại; Muốn khôi phục hãy cài lại [Cài đặt EFI hiệu năng + Mục khởi động firmware]"
	}
	low := strings.ToLower(string(data))
	hit := func(s string) bool { return strings.Contains(low, strings.ToLower(s)) }

	if hit("*** unlocked ***") || hit("already unlocked (ss0/ss1 exact)") {
		return "  [Nhật ký EFI] Quá trình mở khoá EFI thực tế ĐÃ THÀNH CÔNG — Trạng thái bị driver ghi đè\n  Vui lòng chạy lại bộ cài (thiết lập lại GSP) rồi khởi động lại, hoặc dùng phiên bản driver tương thích"
	}
	// Lỗi A: Không truy cập được DMA >4GB (Chưa bật Above 4G)
	if hit("wpr2 not up") || (hit("imem[0]=0xffffffff") && hit("fwsec40")) {
		r := "  [Nhật ký EFI] Không bật được WPR2 + Đọc DMA trả về toàn F\n"
		r += "  → GPU không truy cập được vùng nhớ >4GB chứa payload mở khoá. Đây là do cài đặt BIOS, vui lòng kiểm tra:\n"
		r += "  1. Above 4G Decoding / Giải mã trên 4G → Enabled ← Nguyên nhân phổ biến nhất!\n"
		r += "  2. Resizable BAR / Re-BAR → Auto/Enabled (nếu có tuỳ chọn)\n"
		r += "  3. Chuyển card sang khe PCIe x16 đầu tiên (kết nối trực tiếp CPU)\n"
		r += "  4. Fast Boot trong BIOS → Disabled\n"
		r += "  (Bo mạch X99: Tìm Above 4G trong mục Advanced/PCI Subsystem)"
		return r
	}
	// Lỗi B: booter HALT
	if hit("final]: plm=") && hit("ss0=0x00000000") && (hit("halted") || hit("not-ok")) {
		r := "  [Nhật ký EFI] Inject booter thất bại (nhiều lần HALT, SS0 cuối cùng vẫn là 0)\n"
		r += "  → Vấn đề thời gian khởi động hoặc khe cắm phụ/cắm qua chip cầu PLX, vui lòng kiểm tra:\n"
		r += "  1. Chuyển card sang khe PCIe x16 đầu tiên (tránh qua chip cầu PLX)\n"
		r += "  2. Above 4G Decoding → Enabled\n"
		r += "  3. Fast Boot → Disabled\n"
		r += "  4. Nếu chạy nhiều GPU: Tạm thời tháo các card khác, chỉ giữ lại card CMP để kiểm tra"
		return r
	}
	// Lỗi C: EFI không tìm thấy card
	if hit("gpu not found; abort") || hit("not found (both encodings)") {
		r := "  [Nhật ký EFI] EFI không tìm thấy card: Thường do số hiệu Bus PCI nằm ngoài phạm vi quét cũ\n"
		r += "  (Các dòng bo mạch AGESA / MSI B450 thường đánh số GPU rời vào bus ≥ 16, hoặc qua cầu PLX)\n"
		r += "  → Vui lòng dùng bộ cài v3.0 cài lại EFI mở khoá (đã hỗ trợ quét toàn bộ 256 bus CF8),\n"
		r += "    sau đó tắt nguồn hẳn rồi bật lại máy"
		return r
	}
	// Lỗi không xác định: Trích xuất dòng chính
	var key []string
	for _, ln := range strings.Split(string(data), "\n") {
		l := strings.ToLower(ln)
		if strings.Contains(l, "wpr2") || strings.Contains(l, "ss0") ||
			strings.Contains(l, "result") || strings.Contains(l, "not found") ||
			strings.Contains(l, "unlocked") || strings.Contains(l, "halt") {
			key = append(key, strings.TrimSpace(ln))
			if len(key) >= 5 {
				break
			}
		}
	}
	return "  [Nhật ký EFI] Chưa tự động phân loại, các dòng đáng chú ý:\n  " + strings.Join(key, "\n  ")
}
