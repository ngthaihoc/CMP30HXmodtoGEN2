package main

import (
	"strings"
	"testing"
)

func TestApp_ConcurrencyLock(t *testing.T) {
	app := NewApp()

	if !app.tryAcquire("Tác vụ 1") {
		t.Fatal("tryAcquire lần đầu phải thành công")
	}

	if app.tryAcquire("Tác vụ 2") {
		t.Fatal("tryAcquire khi đang bận phải trả về false")
	}

	app.releaseOp()

	if !app.tryAcquire("Tác vụ 2") {
		t.Fatal("tryAcquire sau khi giải phóng phải thành công")
	}
	app.releaseOp()
}

func TestApp_RiotSignEFI_PhraseGuard(t *testing.T) {
	app := NewApp()

	// 1. Phải từ chối cụm từ sai
	errWrong := app.RiotSignEFI("xac nhan")
	if errWrong == nil {
		t.Fatal("RiotSignEFI với cụm từ sai không được phép thành công")
	}
	if !strings.Contains(errWrong.Error(), "XAC NHAN KY EFI") {
		t.Errorf("Thông báo lỗi thiếu yêu cầu cụm từ chuẩn: %v", errWrong)
	}

	// 2. Phải từ chối chuỗi rỗng
	errEmpty := app.RiotSignEFI("")
	if errEmpty == nil {
		t.Fatal("RiotSignEFI với chuỗi rỗng không được phép thành công")
	}
}

func TestApp_HVCI_NeverAutoTurnsOff(t *testing.T) {
	app := NewApp()
	rep := app.CheckHVCIStatus()

	// Ràng buộc an toàn: Tool TUYỆT ĐỐI không tự động tắt HVCI
	if rep.CanAutoTurn {
		t.Errorf("Yêu cầu an toàn vi phạm: CanAutoTurn phải là false")
	}

	if rep.Enabled {
		if !strings.Contains(rep.GuideText, "HƯỚNG DẪN THỦ CÔNG") {
			t.Errorf("Khi HVCI bật, bắt buộc phải có hướng dẫn thủ công: %s", rep.GuideText)
		}
	}
}

func TestApp_TelemetryGathering(t *testing.T) {
	app := NewApp()
	telem := app.gatherTelemetry()

	// Kiểm tra độ dài mảng ma trận 16 làn
	if len(telem.Lanes) != 16 {
		t.Fatalf("Độ dài mảng làn PCIe phải là 16, thực tế: %d", len(telem.Lanes))
	}

	if telem.Timestamp == "" {
		t.Errorf("Timestamp không được để trống")
	}
}

func TestApp_ScanEnvironment(t *testing.T) {
	app := NewApp()
	rep := app.ScanEnvironment()

	if len(rep.Items) == 0 {
		t.Fatal("Danh sách kiểm tra môi trường không được rỗng")
	}

	foundBoot := false
	foundCard := false
	for _, it := range rep.Items {
		if it.Name == "Chế độ Boot" {
			foundBoot = true
		}
		if it.Name == "Card đồ hoạ" {
			foundCard = true
		}
	}

	if !foundBoot || !foundCard {
		t.Errorf("Thiếu các mục kiểm tra quan trọng trong môi trường: foundBoot=%v, foundCard=%v", foundBoot, foundCard)
	}
}

func TestApp_RiotStatus_NoUnverifiedClaims(t *testing.T) {
	app := NewApp()
	st := app.GetRiotStatus()

	// Ràng buộc 10: Tuyệt đối không xuất hiện claim chưa kiểm chứng "100% tương thích Riot Vanguard/EAC/BattlEye"
	unverified1 := "100% tương thích Riot Vanguard"
	unverified2 := "~50 TFLOPS"

	allTexts := []string{st.LoLDesc, st.ValorantDesc, st.Recommendation, st.GpuDesc}
	for _, txt := range allTexts {
		if strings.Contains(txt, unverified1) {
			t.Errorf("Phát hiện claim chưa kiểm chứng trong giao diện: %q", unverified1)
		}
		if strings.Contains(txt, unverified2) {
			t.Errorf("Phát hiện claim chưa kiểm chứng trong giao diện: %q", unverified2)
		}
	}
}
