package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

func TestIsAdmin(t *testing.T) {
	// Kiểm tra hàm isAdmin thực thi trơn tru mà không panic
	_ = isAdmin()
}

func TestIsWindows11(t *testing.T) {
	// Kiểm tra hàm isWindows11 đọc thông tin từ Registry hệ thống
	isW11 := isWindows11()
	t.Logf("isWindows11() = %v", isW11)
}

func TestParseFlags(t *testing.T) {
	args := []string{"-silent", "-optimize", "-auto-sign"}
	opts, err := ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	if !opts.Silent || !opts.Optimize || !opts.AutoSign {
		t.Errorf("Unexpected opts: %+v", opts)
	}
}

type mockUEFIManager struct {
	signCalled   bool
	rebootCalled bool
}

func (m *mockUEFIManager) PrepareAndSignEFI(ctx context.Context, log io.Writer) (*SigningResult, error) {
	m.signCalled = true
	return &SigningResult{
		CertPathC:       `C:\test.cer`,
		CertPathDesktop: `C:\Users\test\Desktop\test.cer`,
	}, nil
}

func (m *mockUEFIManager) RebootToFirmware(ctx context.Context) error {
	m.rebootCalled = true
	return nil
}

func TestRunCLI_AutoSign(t *testing.T) {
	var buf bytes.Buffer
	mock := &mockUEFIManager{}
	opts := CLIOptions{
		Silent:   true,
		AutoSign: true,
	}
	err := RunCLI(opts, &buf, mock)
	if err != nil {
		t.Fatalf("RunCLI failed: %v", err)
	}
	if !mock.signCalled {
		t.Errorf("Expected PrepareAndSignEFI to be called")
	}
	if !strings.Contains(buf.String(), "Tự Động Ký") && !strings.Contains(buf.String(), "chứng chỉ") {
		t.Errorf("Log missing expected sign output: %s", buf.String())
	}
}

func TestRunCLI_Gen2Warning(t *testing.T) {
	var buf bytes.Buffer
	opts := CLIOptions{
		Silent:      true,
		Gen2Warning: true,
	}
	err := RunCLI(opts, &buf, nil)
	if err != nil {
		t.Fatalf("RunCLI failed: %v", err)
	}
	if !strings.Contains(buf.String(), "ASPM") || !strings.Contains(buf.String(), "Render Test") {
		t.Errorf("Log missing expected ASPM explanation: %s", buf.String())
	}
}

func TestRunCLI_JSONStatus(t *testing.T) {
	var buf bytes.Buffer
	opts := CLIOptions{
		Silent:     true,
		JSONStatus: true,
	}
	err := RunCLI(opts, &buf, nil)
	if err != nil {
		t.Fatalf("RunCLI failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"model"`) || !strings.Contains(out, `"gpuDesc"`) || !strings.Contains(out, `"valorantDesc"`) {
		t.Errorf("Log missing expected JSON status fields: %s", out)
	}
}


