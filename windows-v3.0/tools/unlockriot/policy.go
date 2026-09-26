package main

import (
	"fmt"
	"strings"
)

// GPUModel biểu thị dòng GPU NVIDIA được nhận diện
type GPUModel string

const (
	GPUModelCMP40HX GPUModel = "CMP 40HX"
	GPUModelCMP30HX GPUModel = "CMP 30HX"
	GPUModelUnknown GPUModel = "Unknown"
)

// RiotStatus chứa kết quả đánh giá tính tương thích và khuyến nghị bảo mật
type RiotStatus struct {
	Model                  GPUModel
	IsWin11                bool
	SecureBootOn           bool
	HasTensorCore          bool
	CanPlayValorant        bool
	CanPlayLeagueOfLegends bool
	NeedsEFISigning        bool
	RecommendedSecureBoot  string
	BannerText             string
	CompletionTitle        string
	CompletionDetail       string
}

// DetectGPUModel nhận diện dòng GPU dựa trên Device ID và tên thiết bị
func DetectGPUModel(deviceID uint16, name string) GPUModel {
	upperName := strings.ToUpper(name)
	if deviceID == DeviceIDCMP40HX || strings.Contains(upperName, "40HX") {
		return GPUModelCMP40HX
	}
	if deviceID == DeviceIDCMP30HX || strings.Contains(upperName, "30HX") {
		return GPUModelCMP30HX
	}
	return GPUModelUnknown
}

// EvaluateRiotStatus đánh giá khả năng chơi game Riot và trạng thái Tensor Core / Gen 2
func EvaluateRiotStatus(deviceID uint16, gpuName string, sbOn, isWin11 bool) RiotStatus {
	model := DetectGPUModel(deviceID, gpuName)
	osName := "Windows 10"
	if isWin11 {
		osName = "Windows 11"
	}

	sbText := "ĐÃ TẮT (Disabled)"
	if sbOn {
		sbText = "ĐANG BẬT (Enabled)"
	}

	banner := fmt.Sprintf("HỆ ĐIỀU HÀNH: %s | SECURE BOOT: %s\n", osName, sbText)

	status := RiotStatus{
		Model:                  model,
		IsWin11:                isWin11,
		SecureBootOn:           sbOn,
		CanPlayLeagueOfLegends: true,
	}

	switch model {
	case GPUModelCMP40HX:
		status.HasTensorCore = true
		banner += "🔴 PHÁT HIỆN: NVIDIA CMP 40HX [TU106] (Gen 2 & Tensor Core)\n\n"
		if !isWin11 {
			status.CanPlayValorant = true
			status.NeedsEFISigning = false
			status.RecommendedSecureBoot = "DISABLED"
			status.CompletionTitle = "Hoàn tất Tối Ưu (CMP 40HX - Windows 10)"
			status.CompletionDetail = "Đã dọn dẹp driver mở khóa và tối ưu Registry thành công!\n\n" +
				"✅ VỚI WINDOWS 10 (LÝ TƯỞNG NHẤT):\n" +
				"- Hãy giữ TẮT SECURE BOOT (Disabled) trong BIOS.\n" +
				"- 40HXUNLK.EFI sẽ nạp mở khóa Tensor Core & Gen 2 thành công.\n" +
				"- Bạn có thể chơi cả Valorant và LMHT mà không mất Tensor Core!\n"

			banner += "✅ Cấu hình tối ưu trên Windows 10:\n" +
				"👉 Giữ TẮT SECURE BOOT (Disabled) trong BIOS để nạp EFI Tensor Core.\n" +
				"👉 Vanguard trên Win 10 KHÔNG ép bật Secure Boot -> Chơi mượt mà cả Valorant & LMHT!\n"
		} else {
			status.NeedsEFISigning = true
			status.RecommendedSecureBoot = "CUSTOM_ENROLL_ENABLED"
			status.CompletionTitle = "Hoàn tất Tối Ưu (CMP 40HX - Windows 11)"
			status.CompletionDetail = "Đã dọn dẹp driver mở khóa và tối ưu Registry thành công!\n\n" +
				"🔴 ĐỐI VỚI CMP 40HX TRÊN WINDOWS 11:\n\n" +
				"1. Với Liên Minh Huyền Thoại / TFT:\n" +
				"   - Tắt Secure Boot trong BIOS (Disabled) để giữ cả Gen 2 và Tensor Core.\n\n" +
				"2. Với VALORANT (Yêu cầu Secure Boot):\n" +
				"   - Để vừa BẬT Secure Boot cho Valorant vừa nạp được 40HXUNLK.EFI:\n" +
				"     👉 Xem chi tiết ở nút [📖 Hướng Dẫn Giữ Cả Gen 2 & Tensor Core]!\n"

			if sbOn {
				status.CanPlayValorant = true
				banner += "⚠️ Bạn đang ở Windows 11 và Secure Boot ĐANG BẬT:\n" +
					"👉 Valorant chạy được, NHƯNG file 40HXUNLK.EFI bị BIOS chặn (mất Tensor Core).\n" +
					"👉 Hãy dùng nút [🔐 Tự Động Ký Chữ Ký Số EFI & Chuẩn Bị Key BIOS] bên dưới để nạp Key cá nhân!\n"
			} else {
				status.CanPlayValorant = false
				banner += "ℹ️ Bạn đang ở Windows 11 và Secure Boot ĐÃ TẮT:\n" +
					"👉 Gen 2 và Tensor Core (40HXUNLK.EFI) hoạt động 100%!\n" +
					"👉 Chơi tốt Liên Minh Huyền Thoại (LMHT / TFT).\n" +
					"👉 Để chơi cả Valorant: Nhấn [🔐 Tự Động Ký Chữ Ký Số EFI...] để chuẩn bị nạp Key và BẬT Secure Boot!\n"
			}
		}

	case GPUModelCMP30HX:
		status.HasTensorCore = false
		status.NeedsEFISigning = false
		status.RecommendedSecureBoot = "ENABLED"
		status.CanPlayLeagueOfLegends = true
		status.CanPlayValorant = (!isWin11 || sbOn)
		status.CompletionTitle = "Hoàn tất Tối Ưu (CMP 30HX)"
		status.CompletionDetail = "Đã dọn dẹp driver mở khóa và tối ưu Registry cho Riot Games thành công!\n\n" +
			"✅ Với CMP 30HX:\n" +
			"Bạn KHÔNG cần tắt Secure Boot. Hãy giữ Secure Boot BẬT bình thường trong BIOS để chơi tốt cả Valorant và LMHT trên Windows 10 & 11."

		banner += "🟢 PHÁT HIỆN: NVIDIA CMP 30HX [TU116] (Gen 2 Hardware Lock)\n" +
			"✅ CMP 30HX không cần nạp EFI Tensor Core, Secure Boot có thể giữ BẬT bình thường để chơi Valorant & LMHT trên cả Win 10 và 11.\n"

	default:
		status.HasTensorCore = false
		status.NeedsEFISigning = false
		status.RecommendedSecureBoot = "ENABLED"
		status.CanPlayValorant = (!isWin11 || sbOn)
		status.CompletionTitle = "Hoàn tất Tối Ưu"
		status.CompletionDetail = "Đã dọn dẹp driver mở khóa và tối ưu Registry thành công!\n\n" +
			"⚠️ LƯU Ý CHO CMP 40HX:\n" +
			"Nếu bạn dùng CMP 40HX, hãy chú ý cấu hình Secure Boot phù hợp để giữ cả Gen 2 và Tensor Core!"

		banner += "⚠️ Lưu ý: Hãy đảm bảo bạn đã mở khóa Gen 2 trước khi chạy tối ưu hóa Riot.\n"
	}

	banner += "\nCông cụ này sẽ thực hiện:\n" +
		"1. Dọn dẹp driver mở khóa (WinRing0x64.sys, ThrottleStop.sys) và tắt Testsigning để Riot Vanguard không chặn.\n" +
		"2. Tự động sinh cert & ký PE Authenticode cho 40HXUNLK.EFI để vừa bật Secure Boot vừa giữ Tensor Core.\n" +
		"3. Ép Windows ưu tiên GPU hiệu năng cao (CMP) cho Valorant và Liên Minh Huyền Thoại."

	status.BannerText = banner
	return status
}

// GetBiosRebootGuide xuất nội dung hướng dẫn thao tác trong BIOS Setup
func GetBiosRebootGuide(res *SigningResult) string {
	espLine := ""
	if res.CertPathESP != "" {
		espLine = fmt.Sprintf("  3. Phân vùng ESP: %s\n", res.CertPathESP)
	}

	return fmt.Sprintf(`⚠️ BƯỚC BẮT BUỘC: HÃY DÙNG ĐIỆN THOẠI CHỤP LẠI MÀN HÌNH NÀY TRƯỚC KHI REBOOT!

Đã tạo chứng chỉ và ký chữ ký số UEFI cho 40HXUNLK.EFI thành công!
File chứng chỉ (.cer) đã được lưu sẵn tại:
  1. Ổ hệ thống: %s
  2. Màn hình Desktop: %s
%s
Sau khi bạn tích xác nhận và nhấn [Khởi Động Lại Vào BIOS Ngay], máy tính sẽ TỰ ĐỘNG khởi động lại và vào THẲNG màn hình cài đặt BIOS Setup.

HÃY THỰC HIỆN ĐÚNG CÁC BƯỚC SAU TRONG BIOS:
================================================================================
BƯỚC 1: Vào tab [Security] hoặc [Boot] -> Tìm mục [Key Management]
         (Nếu đang ở Standard Mode, hãy chuyển sang Custom Mode).

BƯỚC 2: Tìm mục [Authorized Signatures (db)] hoặc [Append db / Enroll Key from File]:
         -> Chọn ổ đĩa hệ thống (hoặc phân vùng ESP).
         -> Chọn file: "CMP40HX_Key.cer".
         -> Chọn [Append / Add] để nạp chứng chỉ cá nhân vào cơ sở dữ liệu db.
         (LƯU Ý: Giữ nguyên các key mặc định của Microsoft, KHÔNG xóa).

BƯỚC 3: Chuyển mục [Secure Boot] sang: [ENABLED] (BẬT).

BƯỚC 4: Nhấn phím F10 để Lưu cấu hình và Khởi động lại vào Windows.
================================================================================

KẾT QUẢ ĐẠT ĐƯỢC SAU KHI VÀO LẠI WINDOWS:
  [V] 40HXUNLK.EFI được BIOS tin tưởng và nạp thành công -> TENSOR CORE & GEN 2 MỞ KHÓA!
  [V] Windows 11 nhận diện Secure Boot ĐANG BẬT -> RIOT VANGUARD KHÔNG CHẶN, CHƠI ĐƯỢC TẤT CẢ GAME (VALORANT, LMHT, TFT)!`,
		res.CertPathC,
		res.CertPathDesktop,
		espLine,
	)
}

// GetTensorGuide tạo nội dung hướng dẫn bảo lưu Gen 2 và Tensor Core
func GetTensorGuide(isWin11, is40HX bool) string {
	msg := "=== HƯỚNG DẪN DÀNH CHO BẠN ĐÃ MỞ KHÓA GEN 2 & TENSOR CORE ===\n\n"
	msg += "Mục tiêu: Giữ 100% PCIe Gen 2 và Tensor Core (40HXUNLK.EFI) mà không bị Riot Vanguard chặn.\n\n"

	msg += "1. NẾU DÙNG WINDOWS 10 (KHUYÊN DÙNG NHẤT CHO VALORANT + TENSOR CORE):\n" +
		"  - Cấu hình BIOS: TẮT SECURE BOOT (Disabled).\n" +
		"  - Lý do: Riot Vanguard trên Win 10 KHÔNG yêu cầu bật Secure Boot.\n" +
		"  - File 40HXUNLK.EFI sẽ nạp mượt mà lúc khởi động (Tensor Core mở khóa SS0=0x88888888).\n" +
		"  - Sau khi chạy UnlockRiotGame dọn dẹp driver, bạn chơi được cả Valorant và LMHT mà không mất Tensor Core!\n\n"

	msg += "2. NẾU DÙNG WINDOWS 11:\n" +
		"  A. Với Liên Minh Huyền Thoại (LMHT / TFT):\n" +
		"     - Giữ Secure Boot: TẮT (Disabled).\n" +
		"     - Vanguard trên LMHT không bắt buộc Secure Boot, Tensor Core và Gen 2 hoạt động 100%.\n\n" +
		"  B. Với VALORANT (Bắt buộc bật Secure Boot + TPM):\n" +
		"     Nếu muốn VỪA BẬT SECURE BOOT (để chơi Valorant) VỪA CHẠY ĐƯỢC 40HXUNLK.EFI (Tensor Core):\n" +
		"     👉 Cách 1 (Tự ký chữ ký số UEFI cho file EFI):\n" +
		"        1. Tạo chứng chỉ cá nhân (.cer) và private key (.key).\n" +
		"        2. Ký chứng chỉ vào file 40HXUNLK.EFI bằng sbsigntool hoặc SignTool.\n" +
		"        3. Vào BIOS -> Security / Boot -> Key Management (Custom Mode).\n" +
		"        4. Chọn 'Append db' (Authorized Signatures) -> Nạp file .cer của bạn vào (giữ nguyên key Microsoft).\n" +
		"        5. Đặt Secure Boot: ENABLED (BẬT).\n" +
		"        => BIOS sẽ cho phép chạy 40HXUNLK.EFI vì cert đã có trong db, đồng thời Vanguard Win 11 nhận diện Secure Boot đang BẬT!\n\n" +
		"     👉 Cách 2 (Khuyên dùng nếu không thạo BIOS Key):\n" +
		"        Cài thêm một phân vùng Windows 10 (Dual Boot) để chơi Valorant và dùng Tensor Core thoải mái mà không bị ép bật Secure Boot!"

	return msg
}
