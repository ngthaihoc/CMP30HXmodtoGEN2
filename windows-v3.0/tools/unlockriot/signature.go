package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SignatureResult cấu trúc kết quả kiểm tra chữ ký số
type SignatureResult struct {
	Status  string `json:"Status"`
	Subject string `json:"Subject"`
}

// findDriverDLL tìm đường dẫn file driver DLL nvwgf2umx.dll trong System32 hoặc DriverStore
func findDriverDLL() string {
	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	p := filepath.Join(sysRoot, "System32", "nvwgf2umx.dll")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	// Với Windows DCH Driver, DLL nằm trong DriverStore\FileRepository
	pattern := filepath.Join(sysRoot, "System32", "DriverStore", "FileRepository", "nv*", "nvwgf2umx.dll")
	matches, err := filepath.Glob(pattern)
	if err == nil && len(matches) > 0 {
		return matches[0]
	}

	return p
}

// CheckDriverSignature kiểm tra trạng thái chữ ký số Authenticode của file driver DLL
func CheckDriverSignature(dllPath string) (bool, string, error) {
	if dllPath == "" {
		dllPath = findDriverDLL()
	}

	if _, err := os.Stat(dllPath); err != nil {
		return false, "", fmt.Errorf("không tìm thấy file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	escapedPath := strings.ReplaceAll(dllPath, `'`, `''`)
	cmdStr := fmt.Sprintf(`$sig = Get-AuthenticodeSignature -FilePath '%s'; [PSCustomObject]@{Status=$sig.Status.ToString(); Subject=($sig.SignerCertificate.Subject -replace '[\r\n]',' ')} | ConvertTo-Json -Compress`, escapedPath)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", cmdStr)
	out, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("lỗi thực thi kiểm tra chữ ký PowerShell: %w", err)
	}

	var res SignatureResult
	if err := json.Unmarshal(out, &res); err != nil {
		// Fallback nếu JSON parse lỗi
		raw := strings.TrimSpace(string(out))
		return strings.Contains(raw, "Valid"), raw, nil
	}

	isValid := strings.EqualFold(res.Status, "Valid")
	desc := fmt.Sprintf("Trạng thái: %s | Chủ thể: %s", res.Status, res.Subject)
	return isValid, desc, nil
}
