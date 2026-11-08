package hxcore

import (
	"errors"
	"golang.org/x/sys/windows/svc/mgr"
	"testing"
)

func TestServiceInfo_NonExistentService(t *testing.T) {
	exists, stype, state := ServiceInfo("Service_NonExistent_Mock_Test_12345")
	if exists {
		t.Fatalf("expected non-existent service to return false, got exists=true, stype=%s, state=%s", stype, state)
	}
}

func TestEnsureNvidiaControlPanelHealthy_RunsWithoutPanic(t *testing.T) {
	previous := nvidiaControlPanelService
	t.Cleanup(func() { nvidiaControlPanelService = previous })
	denied := errors.New("không có quyền mở dịch vụ NVIDIA")
	nvidiaControlPanelService = func(name string, fn func(*mgr.Service) error) error { return denied }
	err := EnsureNvidiaControlPanelHealthy()
	if !errors.Is(err, denied) {
		t.Fatalf("Phải trả lại lỗi dịch vụ: %v", err)
	}
}
