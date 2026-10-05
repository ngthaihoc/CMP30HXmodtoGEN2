package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckDriverSignature_Unsigned(t *testing.T) {
	tmpDir := t.TempDir()
	dummy := filepath.Join(tmpDir, "dummy.dll")
	if err := os.WriteFile(dummy, []byte("MZdummycontent"), 0644); err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	valid, status, err := CheckDriverSignature(dummy)
	if err != nil {
		t.Fatalf("CheckDriverSignature returned unexpected error: %v", err)
	}
	if valid {
		t.Errorf("Expected dummy file to be invalid, got valid=true")
	}
	if status == "" {
		t.Errorf("Expected non-empty status description")
	}
}

func TestCheckDriverSignature_NonExistent(t *testing.T) {
	_, _, err := CheckDriverSignature(`C:\nonexistent_path\fake_driver.dll`)
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestCheckDriverSignature_SystemSigned(t *testing.T) {
	target := findDriverDLL()
	if _, err := os.Stat(target); err != nil {
		sysRoot := os.Getenv("SystemRoot")
		if sysRoot == "" {
			sysRoot = `C:\Windows`
		}
		target = filepath.Join(sysRoot, "explorer.exe")
	}

	valid, status, err := CheckDriverSignature(target)
	if err != nil {
		t.Fatalf("CheckDriverSignature failed for %s: %v", target, err)
	}
	if !valid {
		t.Errorf("Expected %s to be signed and valid, got valid=false, status=%s", target, status)
	}
}

func TestFindDriverDLL(t *testing.T) {
	dll := findDriverDLL()
	t.Logf("findDriverDLL() = %s", dll)
	if dll == "" {
		t.Errorf("Expected non-empty driver DLL path")
	}
}
