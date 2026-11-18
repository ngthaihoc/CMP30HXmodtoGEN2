package hxcore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreEfiBackupPreservesBackupWhenReadbackIsCorrupted(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bootx64.efi")
	backup := target + ".40hx.bak"
	original := append([]byte("MZ"), bytes.Repeat([]byte{0}, 0x2000-2)...)
	if err := os.WriteFile(backup, original, 0o644); err != nil {
		t.Fatal(err)
	}
	prior := efiWriteFile
	efiWriteFile = func(path string, data []byte, mode os.FileMode) error {
		return os.WriteFile(path, bytes.Repeat([]byte("x"), len(data)), mode)
	}
	t.Cleanup(func() { efiWriteFile = prior })
	if err := restoreEfiBackup(target, backup); err == nil {
		t.Fatal("same-length corrupted readback must fail")
	}
	if got, err := os.ReadFile(backup); err != nil || !bytes.Equal(got, original) {
		t.Fatalf("backup must remain intact, got %q, err %v", got, err)
	}
}

func TestRestoreEfiBackupRemovesBackupAfterVerifiedRestore(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bootx64.efi")
	backup := target + ".40hx.bak"
	original := append([]byte("MZ"), bytes.Repeat([]byte{0}, 0x2000-2)...)
	if err := os.WriteFile(backup, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := restoreEfiBackup(target, backup); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, original) {
		t.Fatalf("target readback differs, got %q, err %v", got, err)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup should be removed only after verification: %v", err)
	}
}

func TestUninstallDriverFilesPreservesSharedFilesWithoutBackup(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ProgramData", filepath.Join(root, "ProgramData"))
	t.Setenv("SystemRoot", filepath.Join(root, "Windows"))
	drivers := filepath.Join(sysRoot(), "System32", "drivers")
	if err := os.MkdirAll(drivers, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ThrottleStop.sys", "WinRing0x64.sys"} {
		path := filepath.Join(drivers, name)
		if err := os.WriteFile(path, []byte("third-party"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	legacy := filepath.Join(drivers, "40hx_bridge.sys")
	if err := os.WriteFile(legacy, []byte("project-owned"), 0o644); err != nil {
		t.Fatal(err)
	}
	UninstallDriverFiles()
	for _, name := range []string{"ThrottleStop.sys", "WinRing0x64.sys"} {
		if _, err := os.Stat(filepath.Join(drivers, name)); err != nil {
			t.Fatalf("shared %s must remain: %v", name, err)
		}
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("project-owned bridge driver should be removed: %v", err)
	}
}

func TestRestoreEfiBackupRejectsInvalidImage(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bootx64.efi")
	backup := target + ".40hx.bak"
	original := []byte("corrupt")
	if err := os.WriteFile(backup, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := restoreEfiBackup(target, backup); err == nil {
		t.Fatal("invalid backup should not overwrite the bootloader")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target should be unchanged: %v", err)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup should remain for inspection: %v", err)
	}
}
