package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	hxcore "40hxcore"
)

// SigningResult lưu thông tin kết quả ký file EFI và xuất chứng chỉ
type SigningResult struct {
	CertPathC       string
	CertPathESP     string
	CertPathDesktop string
	SignedEFIPath   string
}

// UEFIManager định nghĩa giao diện quản lý chữ ký EFI và khởi động BIOS
type UEFIManager interface {
	PrepareAndSignEFI(ctx context.Context, log io.Writer) (*SigningResult, error)
	RebootToFirmware(ctx context.Context) error
}

// DefaultUEFIManager thực thi giao diện UEFIManager chuẩn
type DefaultUEFIManager struct{}

// PrepareAndSignEFI tự động sinh chứng chỉ cá nhân, ký file 40HXUNLK.EFI và xuất file .cer
func (m *DefaultUEFIManager) PrepareAndSignEFI(ctx context.Context, log io.Writer) (*SigningResult, error) {
	if log == nil {
		log = io.Discard
	}

	fmt.Fprintln(log, "[*] Bắt đầu tự động tạo chứng chỉ và ký file EFI mở khóa...")

	// 1. Gắn phân vùng ESP để tìm/ghi file EFI
	esp := hxcore.MountESP()
	var espCertPath string
	var efiTarget string

	if esp != "" {
		defer hxcore.UnmountESP(esp)
		fmt.Fprintf(log, "[V] Đã gắn phân vùng ESP tại ổ đĩa %s:\n", esp)
		espDir := esp + `:\EFI\40HX`
		os.MkdirAll(espDir, 0755)
		efiTarget = filepath.Join(espDir, "40HXUNLK.EFI")
		espCertPath = esp + `:\CMP40HX_Key.cer`
	} else {
		fmt.Fprintln(log, "[!] Cảnh báo: Không thể gắn phân vùng ESP trực tiếp, sẽ tìm file cục bộ.")
	}

	// 2. Định vị file nguồn 40HXUNLK.EFI nếu trên ESP chưa có
	if efiTarget == "" || !fileExists(efiTarget) {
		srcEFI := findLocalSourceEFI()
		if srcEFI != "" && efiTarget != "" {
			if err := copyFile(srcEFI, efiTarget); err != nil {
				fmt.Fprintf(log, "[!] Sao chép EFI sang ESP thất bại: %v\n", err)
			} else {
				fmt.Fprintf(log, "[V] Đã nạp 40HXUNLK.EFI từ %s vào ESP.\n", srcEFI)
			}
		} else if srcEFI != "" {
			efiTarget = srcEFI
		}
	}

	if efiTarget == "" || !fileExists(efiTarget) {
		return nil, fmt.Errorf("không tìm thấy file 40HXUNLK.EFI trên hệ thống để ký")
	}

	fmt.Fprintf(log, "[*] Mục tiêu ký chữ ký số: %s\n", efiTarget)

	// 3. Thực thi PowerShell để sinh cert, export .cer và ký PE Authenticode
	res, err := executeSigningScript(ctx, efiTarget, esp)
	if err != nil {
		return nil, fmt.Errorf("thực thi ký chữ ký số thất bại: %w", err)
	}

	res.CertPathESP = espCertPath
	res.SignedEFIPath = efiTarget

	fmt.Fprintln(log, "[V] Tự động tạo chứng chỉ và ký chữ ký số PE hoàn tất!")
	fmt.Fprintf(log, "  -> Chứng chỉ ổ C: %s\n", res.CertPathC)
	fmt.Fprintf(log, "  -> Chứng chỉ Desktop: %s\n", res.CertPathDesktop)
	if res.CertPathESP != "" {
		fmt.Fprintf(log, "  -> Chứng chỉ ESP: %s\n", res.CertPathESP)
	}

	return res, nil
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// executeSigningScript sinh lệnh PowerShell tạo cert và ký EFI an toàn với context timeout
func executeSigningScript(ctx context.Context, efiPath, espDrive string) (*SigningResult, error) {
	sysDrive := os.Getenv("SystemDrive")
	if sysDrive == "" {
		sysDrive = "C:"
	}
	certC := filepath.Join(sysDrive, "CMP40HX_Key.cer")

	homeDir, _ := os.UserHomeDir()
	certDesktop := filepath.Join(homeDir, "Desktop", "CMP40HX_Key.cer")

	psCmd := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$certSubject = 'CN=CMP40HX Custom UEFI Key'
$cert = Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.Subject -like "*$certSubject*" } | Select-Object -First 1
if (-not $cert) {
    $cert = New-SelfSignedCertificate -Type CodeSigningCert -Subject $certSubject -CertStoreLocation "Cert:\CurrentUser\My" -NotAfter (Get-Date).AddYears(10)
}

$cPath = '%s'
Export-Certificate -Cert $cert -FilePath $cPath -Type CERT -Force | Out-Null

$deskPath = '%s'
Copy-Item -Path $cPath -Destination $deskPath -Force -ErrorAction SilentlyContinue

if ('%s' -ne '') {
    $espPath = '%s:\CMP40HX_Key.cer'
    Copy-Item -Path $cPath -Destination $espPath -Force -ErrorAction SilentlyContinue
}

$efiTarget = '%s'
if (Test-Path $efiTarget) {
    Set-AuthenticodeSignature -FilePath $efiTarget -Certificate $cert | Out-Null
}
`, escapePS(certC), escapePS(certDesktop), escapePS(espDrive), escapePS(espDrive), escapePS(efiPath))

	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("powershell lỗi (%v): %s", err, string(out))
	}

	if !fileExists(certC) {
		return nil, fmt.Errorf("không tìm thấy file chứng chỉ được tạo tại %s", certC)
	}

	return &SigningResult{
		CertPathC:       certC,
		CertPathDesktop: certDesktop,
	}, nil
}

// RebootToFirmware yêu cầu hệ thống khởi động lại thẳng vào giao diện BIOS Setup
func (m *DefaultUEFIManager) RebootToFirmware(ctx context.Context) error {
	execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Thử lệnh shutdown /r /fw (vào thẳng BIOS Setup)
	cmd := exec.CommandContext(execCtx, "shutdown.exe", "/r", "/fw", "/t", "3")
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Fallback sang shutdown /r thông thường nếu firmware không hỗ trợ /fw
	cmdFallback := exec.CommandContext(execCtx, "shutdown.exe", "/r", "/t", "3")
	if err := cmdFallback.Run(); err != nil {
		return fmt.Errorf("lệnh khởi động lại thất bại: %w", err)
	}

	return nil
}

// findLocalSourceEFI tìm file 40HXUNLK.EFI trong các thư mục phát hành
func findLocalSourceEFI() string {
	candidates := []string{
		`files\40HXUNLK.EFI`,
		`..\files\40HXUNLK.EFI`,
		`..\..\release\files\40HXUNLK.EFI`,
		`windows-v3.0\release\files\40HXUNLK.EFI`,
		`40HXUNLK.EFI`,
	}
	for _, c := range candidates {
		if fileExists(c) {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return filepath.Clean(c)
		}
	}
	return ""
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("mở file nguồn: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("tạo file đích: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("sao chép dữ liệu: %w", err)
	}
	return out.Sync()
}
