package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupReBarReportsMissingAssetsWithoutCommands(t *testing.T) {
	calls := 0
	result := executeSetupReBar(6, rebarDependencies{
		checkLaptop:   func() (bool, string) { return false, "desktop" },
		extractAssets: func() (string, error) { return t.TempDir(), nil },
		run:           func(context.Context, string, ...string) (string, error) { calls++; return "", nil },
	})
	if result.Success || !strings.Contains(result.Error, "nvidiaProfileInspector.exe") || !strings.Contains(result.Error, "NvStrapsReBar.exe") || calls != 0 || result.Attempted {
		t.Fatalf("expected missing assets and no command calls: calls=%d result=%+v", calls, result)
	}
}

func TestSetupReBarStopsAfterProfileFailure(t *testing.T) {
	dir := t.TempDir()
	createRebarAssets(t, dir)
	calls := 0
	result := executeSetupReBar(6, rebarDependencies{
		checkLaptop:   func() (bool, string) { return false, "desktop" },
		extractAssets: func() (string, error) { return dir, nil },
		run: func(context.Context, string, ...string) (string, error) {
			calls++
			return "profile rejected", errors.New("exit status 1")
		},
	})
	if result.Success || !result.Attempted || result.PendingReboot || !strings.Contains(result.Error, "Profile Inspector") || calls != 1 {
		t.Fatalf("expected profile failure without NvStraps attempt: calls=%d result=%+v", calls, result)
	}
}

func TestSetupReBarReportsNvStrapsAndRollbackFailure(t *testing.T) {
	dir := t.TempDir()
	createRebarAssets(t, dir)
	calls := 0
	result := executeSetupReBar(6, rebarDependencies{
		checkLaptop:   func() (bool, string) { return false, "desktop" },
		extractAssets: func() (string, error) { return dir, nil },
		run: func(_ context.Context, executable string, args ...string) (string, error) {
			calls++
			if filepath.Base(executable) == "NvStrapsReBar.exe" {
				return "failed", errors.New("exit status 2")
			}
			return "loaded", nil
		},
	})
	if result.Success || !result.Attempted || result.PendingReboot || !strings.Contains(result.Error, "NvStrapsReBar") || !strings.Contains(result.RollbackError, "NvStrapsReBar") || calls != 3 {
		t.Fatalf("expected setup and rollback failures: calls=%d result=%+v", calls, result)
	}
}

func TestSetupReBarRecordsSuccessfulRollback(t *testing.T) {
	dir := t.TempDir()
	createRebarAssets(t, dir)
	calls := 0
	result := executeSetupReBar(6, rebarDependencies{
		checkLaptop:   func() (bool, string) { return false, "desktop" },
		extractAssets: func() (string, error) { return dir, nil },
		run: func(_ context.Context, executable string, args ...string) (string, error) {
			calls++
			if filepath.Base(executable) == "NvStrapsReBar.exe" && len(args) == 1 && args[0] == "6" {
				return "setup failed", errors.New("exit status 2")
			}
			return "rollback command complete", nil
		},
	})
	if result.Success || !result.RollbackAttempted || !result.RollbackSucceeded || result.RollbackError != "" || calls != 3 {
		t.Fatalf("expected successful rollback evidence: calls=%d result=%+v", calls, result)
	}
}

func TestSetupReBarSuccessIsPendingRebootNotVerified(t *testing.T) {
	dir := t.TempDir()
	createRebarAssets(t, dir)
	result := executeSetupReBar(6, rebarDependencies{
		checkLaptop:   func() (bool, string) { return false, "desktop" },
		extractAssets: func() (string, error) { return dir, nil },
		run:           func(context.Context, string, ...string) (string, error) { return "ok", nil },
	})
	if !result.Success || !result.Attempted || !result.PendingReboot || result.Verified || result.Error != "" {
		t.Fatalf("expected pending post-reboot verification: %+v", result)
	}
}

func TestSetupReBarFailsClosedWhenLaptopDetectionUnavailable(t *testing.T) {
	called := false
	result := executeSetupReBar(6, rebarDependencies{
		checkLaptop:   func() (bool, string) { return false, "" },
		extractAssets: func() (string, error) { called = true; return "", nil },
	})
	if result.Success || called || !strings.Contains(strings.ToLower(result.Error), "laptop") {
		t.Fatalf("detection failure must block setup: assetsCalled=%t result=%+v", called, result)
	}
}

func TestRevertReBarReportsMissingAssetAndCommandFailure(t *testing.T) {
	deps := rebarDependencies{
		extractAssets: func() (string, error) { return t.TempDir(), nil },
		run: func(context.Context, string, ...string) (string, error) {
			t.Fatal("runner called without NvStraps asset")
			return "", nil
		},
	}
	missing := executeRevertReBar(deps)
	if missing.Success || missing.Attempted || !strings.Contains(missing.Error, "NvStrapsReBar.exe") {
		t.Fatalf("expected missing asset failure: %+v", missing)
	}
	dir := t.TempDir()
	createRebarAssets(t, dir)
	deps.extractAssets = func() (string, error) { return dir, nil }
	deps.run = func(context.Context, string, ...string) (string, error) { return "failed", errors.New("exit status 3") }
	failed := executeRevertReBar(deps)
	if failed.Success || !failed.Attempted || !strings.Contains(failed.Error, "NvStrapsReBar") || failed.PendingReboot {
		t.Fatalf("expected revert failure: %+v", failed)
	}
}

func TestRevertReBarSuccessIsPendingRebootNotVerified(t *testing.T) {
	dir := t.TempDir()
	createRebarAssets(t, dir)
	result := executeRevertReBar(rebarDependencies{
		extractAssets: func() (string, error) { return dir, nil },
		run:           func(context.Context, string, ...string) (string, error) { return "done", nil },
	})
	if !result.Success || !result.Attempted || !result.PendingReboot || result.Verified {
		t.Fatalf("revert success must remain pending verification: %+v", result)
	}
}

func createRebarAssets(t *testing.T, dir string) {
	t.Helper()
	for _, name := range []string{"nvidiaProfileInspector.exe", "Enable_ReBAR_Turing.nip", "NvStrapsReBar.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("stub"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
