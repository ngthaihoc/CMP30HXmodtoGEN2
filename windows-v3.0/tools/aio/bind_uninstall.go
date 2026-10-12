package main

import (
	hxcore "40hxcore"
)

// UninstallResult phản hồi kết quả gỡ cài đặt
type UninstallResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UninstallAll gỡ bỏ toàn bộ tác vụ lập lịch, khoá registry, driver và file đã cài đặt
func (a *App) UninstallAll() UninstallResult {
	if !a.tryAcquire("Gỡ cài đặt hoàn chỉnh") {
		return UninstallResult{Success: false, Message: "Hệ thống đang bận thao tác khác."}
	}
	defer a.releaseOp()

	a.Log("==============================================")
	a.Log("  [Gỡ Cài Đặt] Bắt đầu dọn dẹp toàn bộ hệ thống")
	a.Log("==============================================")

	// 1. Gỡ tác vụ Task Scheduler
	a.Log("[*] [1/7] Gỡ bỏ các tác vụ lập lịch (Task Scheduler)...")
	removedTasks := hxcore.UninstallTasks()
	a.Log("  [✓] Đã xoá các tác vụ:")
	for _, t := range removedTasks {
		a.Log("      - " + t)
	}

	// 2. Gỡ bỏ khoá Registry Run
	a.Log("[*] [2/7] Gỡ bỏ khoá tự khởi động Registry Run...")
	hxcore.UninstallRunKey()
	a.Log("  [✓] Đã xoá giá trị 40HXGen2 trong Run key.")

	// 3. Xoá mục khởi động firmware (UEFI Boot Entry)
	a.Log("[*] [3/7] Kiểm tra và xoá mục khởi động '40HX Unlock' trong BIOS...")
	if hxcore.UninstallBootEntry() {
		a.Log("  [✓] Đã xoá mục khởi động 40HX Unlock khỏi BCD.")
	} else {
		a.Log("  [i] Không có mục khởi động 40HX Unlock (hoặc đang chạy Legacy).")
	}

	// 4. Dọn dẹp phân vùng ESP và khôi phục bootx64.efi
	a.Log("[*] [4/7] Dọn dẹp phân vùng ESP EFI...")
	if hxcore.UninstallEspEfi() {
		a.Log("  [✓] Đã dọn dẹp file EFI mở khoá và khôi phục bootx64.efi gốc nếu có sao lưu.")
	} else {
		a.Log("  [i] Không tìm thấy file trong phân vùng ESP cần xử lý.")
	}

	// 5. Dừng và xoá dịch vụ driver kernel
	a.Log("[*] [5/7] Dừng và xoá các dịch vụ driver kernel tạm thời...")
	hxcore.UninstallDriverServices()
	hxcore.UninstallDriverFiles()
	a.Log("  [✓] Đã xoá dịch vụ và các file driver .sys trong System32\\drivers.")

	// 6. Khôi phục GSP (EnableGpuFirmware) và cấu hình HKLM
	a.Log("[*] [6/7] Khôi phục cài đặt GSP và xoá cấu hình HKLM\\SOFTWARE\\40HXUnlock...")
	if hxcore.UninstallGspKey() {
		a.Log("  [✓] Đã xoá cờ EnableGpuFirmware (khôi phục mặc định CPU-RM).")
	}
	hxcore.DeleteConfig()
	_ = hxcore.RemoveDefenderExclusions()

	// 7. Xoá bộ nhớ đệm và nhật ký ProgramData
	a.Log("[*] [7/7] Dọn dẹp ProgramData\\40HXUnlock...")
	hxcore.UninstallProgramData()
	a.Log("  [✓] Đã dọn dẹp toàn bộ dữ liệu ProgramData.")

	a.Log("==============================================")
	a.Log("  [✓] Gỡ cài đặt hoàn tất! Hệ thống đã được khôi phục về trạng thái sạch.")
	a.Log("==============================================")

	return UninstallResult{
		Success: true,
		Message: "Đã gỡ bỏ toàn bộ tác vụ lịch trình, khóa Registry và driver khỏi hệ thống.",
	}
}
