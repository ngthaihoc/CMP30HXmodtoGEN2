package main

import (
	"os"
	"path/filepath"
	"testing"

	hxcore "40hxcore"
)

func TestReadEmbeddedAsset_DriversAndEFI(t *testing.T) {
	requiredFiles := []string{
		"ThrottleStop.sys",
		"WinRing0x64.sys",
		"40HXUNLK.EFI",
		"NvStrapsReBar.exe",
		"nvidiaProfileInspector.exe",
	}

	for _, name := range requiredFiles {
		data, err := ReadEmbeddedAsset(name)
		if err != nil {
			t.Fatalf("ReadEmbeddedAsset(%q) thất bại: %v", name, err)
		}
		if len(data) == 0 {
			t.Errorf("ReadEmbeddedAsset(%q) trả về dữ liệu rỗng", name)
		}
	}
}

func TestExtractAndCleanupToolAssets(t *testing.T) {
	dir, err := ExtractToolAssets()
	if err != nil {
		t.Fatalf("ExtractToolAssets() thất bại: %v", err)
	}

	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Fatalf("Thư mục trích xuất %q không tồn tại hoặc không phải thư mục", dir)
	}

	sampleFile := filepath.Join(dir, "WinRing0x64.sys")
	if fi, err := os.Stat(sampleFile); err != nil || fi.Size() == 0 {
		t.Errorf("Tệp trích xuất %q không hợp lệ", sampleFile)
	}

	// Dọn dẹp
	CleanupToolAssets()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("CleanupToolAssets() không xoá được thư mục tạm: %q", dir)
	}
}

func TestInitEmbeddedDriverHook(t *testing.T) {
	InitEmbeddedDriverHook()
	if hxcore.DriverFileProvider == nil {
		t.Fatal("hxcore.DriverFileProvider chưa được khởi tạo")
	}

	data, err := hxcore.DriverFileProvider("WinRing0x64.sys")
	if err != nil || len(data) == 0 {
		t.Fatalf("DriverFileProvider qua hook thất bại: %v (len=%d)", err, len(data))
	}
}
