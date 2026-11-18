package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	hxcore "40hxcore"
)

func TestInstallDriversStagesBundledBackupsAndPreservesExistingFiles(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	systemRoot := t.TempDir()
	t.Setenv("SystemRoot", systemRoot)
	sharedPath := filepath.Join(systemRoot, "System32", "drivers", "WinRing0x64.sys")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedPath, []byte("shared driver"), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := installDrivers(); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"ThrottleStop.sys", "WinRing0x64.sys"} {
		expected, err := embedded.ReadFile("embed/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(hxcore.PdDrvDir(), name)); err != nil || !bytes.Equal(got, expected) {
			t.Fatalf("cache mismatch for %s: %v", name, err)
		}
	}
	if got, err := os.ReadFile(sharedPath); err != nil || string(got) != "shared driver" {
		t.Fatalf("shared driver changed: %q, err = %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(sharedPath), "ThrottleStop.sys")); !os.IsNotExist(err) {
		t.Fatalf("installation staged ThrottleStop into System32: %v", err)
	}

	cachePath := filepath.Join(hxcore.PdDrvDir(), "WinRing0x64.sys")
	if err := os.WriteFile(cachePath, []byte("existing backup"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installDrivers(); err == nil {
		t.Fatal("mismatched existing driver backup was accepted")
	}
	if got, err := os.ReadFile(cachePath); err != nil || string(got) != "existing backup" {
		t.Fatalf("existing backup changed: %q, err = %v", got, err)
	}
}
