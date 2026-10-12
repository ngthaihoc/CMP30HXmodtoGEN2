package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	hxcore "40hxcore"
)

//go:embed embed/*
var embeddedAssets embed.FS

var (
	tempDirMu      sync.Mutex
	currentTempDir string
)

// ReadEmbeddedAsset đọc nội dung tệp nhúng trong binary
func ReadEmbeddedAsset(name string) ([]byte, error) {
	data, err := embeddedAssets.ReadFile(filepath.ToSlash(filepath.Join("embed", filepath.Base(name))))
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy tài nguyên nhúng %q: %w", name, err)
	}
	return data, nil
}

// InitEmbeddedDriverHook đăng ký hook cho 40hxcore nạp driver từ binary
func InitEmbeddedDriverHook() {
	hxcore.DriverFileProvider = func(fileName string) ([]byte, error) {
		return ReadEmbeddedAsset(fileName)
	}
}

// ExtractToolAssets trích xuất toàn bộ công cụ nhúng vào thư mục tạm với ACL an toàn
func ExtractToolAssets() (string, error) {
	tempDirMu.Lock()
	defer tempDirMu.Unlock()

	if currentTempDir != "" {
		if fi, err := os.Stat(currentTempDir); err == nil && fi.IsDir() {
			return currentTempDir, nil
		}
	}

	dir := filepath.Join(os.TempDir(), fmt.Sprintf("UnlockAIO_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("tạo thư mục tạm thất bại: %w", err)
	}

	// Thiết lập ACL an toàn bằng SID chuẩn: S-1-5-32-544 (Administrators), S-1-5-18 (SYSTEM)
	_ = applySecureACL(dir)

	entries, err := embeddedAssets.ReadDir("embed")
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("đọc danh mục nhúng thất bại: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, readErr := embeddedAssets.ReadFile("embed/" + entry.Name())
		if readErr != nil {
			_ = os.RemoveAll(dir)
			return "", fmt.Errorf("đọc tệp nhúng %s thất bại: %w", entry.Name(), readErr)
		}
		dstPath := filepath.Join(dir, entry.Name())
		if writeErr := os.WriteFile(dstPath, data, 0o700); writeErr != nil {
			_ = os.RemoveAll(dir)
			return "", fmt.Errorf("trích xuất tệp %s thất bại: %w", entry.Name(), writeErr)
		}
	}

	currentTempDir = dir
	return dir, nil
}

// CleanupToolAssets dọn dẹp sạch sẽ thư mục tạm khi ứng dụng thoát
func CleanupToolAssets() {
	tempDirMu.Lock()
	defer tempDirMu.Unlock()

	if currentTempDir != "" {
		_ = os.RemoveAll(currentTempDir)
		currentTempDir = ""
	}
}

// applySecureACL khoá quyền truy cập thư mục tạm, chỉ cho phép Administrators và SYSTEM
func applySecureACL(path string) error {
	// Dùng SID chuẩn độc lập với ngôn ngữ hệ điều hành
	cmd := exec.Command(
		"icacls.exe",
		path,
		"/inheritance:r",
		"/grant:r",
		"*S-1-5-32-544:(OI)(CI)F",
		"/grant:r",
		"*S-1-5-18:(OI)(CI)F",
	)
	return cmd.Run()
}
