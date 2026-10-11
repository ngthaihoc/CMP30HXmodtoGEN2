package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifySignature(t *testing.T) {
	tests := []struct {
		name     string
		isValid  bool
		subject  string
		expected SignatureCategory
	}{
		{
			name:     "Dartraiden_PKI",
			isValid:  false,
			subject:  "CN=NVIDIA Corporation, OU=sign.pki.dartraiden.dn, O=dartraiden PKI Service",
			expected: SigCategoryDartraiden,
		},
		{
			name:     "Atheros_Expired",
			isValid:  false,
			subject:  "CN=Atheros Communications Inc.",
			expected: SigCategoryDartraiden,
		},
		{
			name:     "Microsoft_WHQL",
			isValid:  true,
			subject:  "CN=Microsoft Windows Hardware Compatibility Publisher, O=Microsoft Corporation",
			expected: SigCategoryGenuineWHQL,
		},
		{
			name:     "NVCleanstall_Cert",
			isValid:  true,
			subject:  "CN=NVCleanstall",
			expected: SigCategorySelfSigned,
		},
		{
			name:     "Unsigned_Empty",
			isValid:  false,
			subject:  "",
			expected: SigCategoryUnsigned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySignature(tt.isValid, tt.subject)
			if got != tt.expected {
				t.Errorf("ClassifySignature(%v, %q) = %v, expected %v", tt.isValid, tt.subject, got, tt.expected)
			}
		})
	}
}

func TestCheckDriverSignature_Unsigned(t *testing.T) {
	tmpDir := t.TempDir()
	dummy := filepath.Join(tmpDir, "dummy.dll")
	if err := os.WriteFile(dummy, []byte("MZdummycontent"), 0644); err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}

	valid, status, cat, err := CheckDriverSignatureEx(dummy)
	if err != nil {
		t.Fatalf("CheckDriverSignatureEx returned unexpected error: %v", err)
	}
	if valid {
		t.Errorf("Expected dummy file to be invalid, got valid=true")
	}
	if status == "" {
		t.Errorf("Expected non-empty status description")
	}
	if cat != SigCategoryUnsigned && cat != SigCategoryUnknown {
		t.Logf("Categorized as %v", cat)
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
