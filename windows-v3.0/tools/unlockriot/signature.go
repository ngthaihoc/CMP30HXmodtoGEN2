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

// SignatureCategory phân loại nguồn gốc chữ ký số của driver
type SignatureCategory string

const (
	SigCategoryGenuineWHQL SignatureCategory = "WHQL_GENUINE"
	SigCategoryDartraiden  SignatureCategory = "DARTRAIDEN_MOD"
	SigCategorySelfSigned  SignatureCategory = "SELF_SIGNED"
	SigCategoryUnsigned    SignatureCategory = "UNSIGNED"
	SigCategoryUnknown     SignatureCategory = "UNKNOWN"
)

// ClassifySignature phân loại chữ ký dựa trên subject và tính hợp lệ
func ClassifySignature(isValid bool, subject string) SignatureCategory {
	upper := strings.ToUpper(subject)
	if strings.Contains(upper, "DARTRAIDEN") || strings.Contains(upper, "ATHEROS") {
		return SigCategoryDartraiden
	}
	if strings.Contains(upper, "MICROSOFT WINDOWS HARDWARE COMPATIBILITY") ||
		(strings.Contains(upper, "NVIDIA CORPORATION") && isValid) {
		return SigCategoryGenuineWHQL
	}
	if strings.Contains(upper, "NVCLEANSTALL") {
		return SigCategorySelfSigned
	}
	if !isValid && subject == "" {
		return SigCategoryUnsigned
	}
	if isValid {
		return SigCategoryGenuineWHQL
	}
	return SigCategoryUnknown
}

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

// CheckDriverSignatureEx kiểm tra chữ ký số và phân loại chi tiết nguồn gốc chứng chỉ
func CheckDriverSignatureEx(dllPath string) (bool, string, SignatureCategory, error) {
	if dllPath == "" {
		dllPath = findDriverDLL()
	}

	if _, err := os.Stat(dllPath); err != nil {
		return false, "", SigCategoryUnsigned, fmt.Errorf("không tìm thấy file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	escapedPath := strings.ReplaceAll(dllPath, `'`, `''`)
	cmdStr := fmt.Sprintf(`$st = 'NotSigned'; $sub = ''; try { $cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2 '%s'; if ($cert.Verify()) { $st = 'Valid' } else { $st = 'Invalid' }; $sub = ($cert.Subject -replace '[\r\n]',' ') } catch {}; [PSCustomObject]@{Status=$st; Subject=$sub} | ConvertTo-Json -Compress`, escapedPath)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", cmdStr)
	out, err := cmd.Output()
	if err != nil {
		return false, "", SigCategoryUnknown, fmt.Errorf("lỗi thực thi kiểm tra chữ ký PowerShell: %w", err)
	}

	var res SignatureResult
	if err := json.Unmarshal(out, &res); err != nil {
		raw := strings.TrimSpace(string(out))
		isValid := strings.Contains(raw, "Valid")
		cat := ClassifySignature(isValid, raw)
		return isValid, raw, cat, nil
	}

	isValid := strings.EqualFold(res.Status, "Valid")
	cat := ClassifySignature(isValid, res.Subject)
	desc := fmt.Sprintf("Trạng thái: %s | Chủ thể: %s | Phân loại: %s", res.Status, res.Subject, cat)
	return isValid, desc, cat, nil
}

// CheckDriverSignature kiểm tra trạng thái chữ ký số Authenticode của file driver DLL
func CheckDriverSignature(dllPath string) (bool, string, error) {
	valid, desc, _, err := CheckDriverSignatureEx(dllPath)
	return valid, desc, err
}
