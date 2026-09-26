package hxcore

import (
	"testing"
)

func TestServiceInfo_NonExistentService(t *testing.T) {
	exists, stype, state := ServiceInfo("Service_NonExistent_Mock_Test_12345")
	if exists {
		t.Fatalf("expected non-existent service to return false, got exists=true, stype=%s, state=%s", stype, state)
	}
}

func TestEnsureNvidiaControlPanelHealthy_RunsWithoutPanic(t *testing.T) {
	// Should complete gracefully even in mock/test environments
	err := EnsureNvidiaControlPanelHealthy()
	_ = err
}
