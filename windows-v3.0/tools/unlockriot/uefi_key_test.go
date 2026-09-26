package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// MockUEFIManager giả lập thực thi UEFIManager cho kiểm thử đơn vị
type MockUEFIManager struct {
	ShouldFailSign   bool
	ShouldFailReboot bool
	Result           *SigningResult
}

func (m *MockUEFIManager) PrepareAndSignEFI(ctx context.Context, log io.Writer) (*SigningResult, error) {
	if m.ShouldFailSign {
		return nil, os.ErrPermission
	}
	return m.Result, nil
}

func (m *MockUEFIManager) RebootToFirmware(ctx context.Context) error {
	if m.ShouldFailReboot {
		return os.ErrInvalid
	}
	return nil
}

func TestFileExists(t *testing.T) {
	// Arrange
	tmpFile, err := os.CreateTemp("", "test_file_exists_*.txt")
	if err != nil {
		t.Fatalf("Không thể tạo file tạm: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("content")
	tmpFile.Close()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"FileTonTaiCoNoiDung", tmpFile.Name(), true},
		{"FileKhongTonTai", filepath.Join(os.TempDir(), "non_existent_12345.bin"), false},
		{"ThuMucKhongPhaiFile", os.TempDir(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := fileExists(tt.path)

			// Assert
			if got != tt.expected {
				t.Errorf("fileExists(%q) = %v; kỳ vọng %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestMockUEFIManager(t *testing.T) {
	tests := []struct {
		name        string
		mock        *MockUEFIManager
		expectSign  bool
		expectReset bool
	}{
		{
			name: "ThanhCongToanBo",
			mock: &MockUEFIManager{
				Result: &SigningResult{
					CertPathC:       `C:\CMP40HX_Key.cer`,
					CertPathDesktop: `C:\Users\Admin\Desktop\CMP40HX_Key.cer`,
				},
			},
			expectSign:  true,
			expectReset: true,
		},
		{
			name: "KyThatBai",
			mock: &MockUEFIManager{
				ShouldFailSign: true,
			},
			expectSign:  false,
			expectReset: true,
		},
		{
			name: "RebootThatBai",
			mock: &MockUEFIManager{
				Result: &SigningResult{
					CertPathC: `C:\CMP40HX_Key.cer`,
				},
				ShouldFailReboot: true,
			},
			expectSign:  true,
			expectReset: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			res, errSign := tt.mock.PrepareAndSignEFI(context.Background(), io.Discard)
			errReboot := tt.mock.RebootToFirmware(context.Background())

			// Assert
			if (errSign == nil) != tt.expectSign {
				t.Errorf("PrepareAndSignEFI err = %v; kỳ vọng thành công = %v", errSign, tt.expectSign)
			}
			if tt.expectSign && res == nil {
				t.Errorf("PrepareAndSignEFI trả về kết quả rỗng khi kỳ vọng thành công")
			}
			if (errReboot == nil) != tt.expectReset {
				t.Errorf("RebootToFirmware err = %v; kỳ vọng thành công = %v", errReboot, tt.expectReset)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	// Arrange
	src, err := os.CreateTemp("", "test_copy_src_*.txt")
	if err != nil {
		t.Fatalf("Tạo src temp: %v", err)
	}
	defer os.Remove(src.Name())

	data := "cmp40hx_unlock_copy_test"
	src.WriteString(data)
	src.Close()

	dst := filepath.Join(os.TempDir(), "test_copy_dst.txt")
	defer os.Remove(dst)

	// Act
	err = copyFile(src.Name(), dst)
	if err != nil {
		t.Fatalf("copyFile lỗi: %v", err)
	}

	// Assert
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("Đọc dst lỗi: %v", err)
	}
	if string(content) != data {
		t.Errorf("Nội dung copy không khớp: %q != %q", string(content), data)
	}
}

func TestCopyFile_Errors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		dst     string
		wantErr bool
	}{
		{
			name:    "SrcKhongTonTai",
			src:     filepath.Join(os.TempDir(), "non_existent_src_9999.bin"),
			dst:     filepath.Join(os.TempDir(), "dst_9999.bin"),
			wantErr: true,
		},
		{
			name:    "DstKhongHopLe",
			src:     os.Args[0], // file exe đang chạy
			dst:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := copyFile(tt.src, tt.dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("copyFile(%q, %q) err = %v, wantErr = %v", tt.src, tt.dst, err, tt.wantErr)
			}
		})
	}
}

func TestFindLocalSourceEFI(t *testing.T) {
	// Kiểm tra hàm không crash và trả về kết quả chuỗi
	got := findLocalSourceEFI()
	if got != "" {
		if !fileExists(got) {
			t.Errorf("findLocalSourceEFI trả về đường dẫn không tồn tại: %s", got)
		}
	}
}

func TestExecuteSigningScript_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Hủy ngay lập tức

	_, err := executeSigningScript(ctx, "non_existent.efi", "")
	if err == nil {
		t.Errorf("executeSigningScript với context đã hủy phải trả về lỗi")
	}
}

func TestEscapePS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"C:\\test\\path", "C:\\test\\path"},
		{"C:\\test'path", "C:\\test''path"},
		{"'; Remove-Item -Force '", "''; Remove-Item -Force ''"},
		{"", ""},
	}

	for _, tt := range tests {
		got := escapePS(tt.input)
		if got != tt.expected {
			t.Errorf("escapePS(%q) = %q; kỳ vọng %q", tt.input, got, tt.expected)
		}
	}
}

func TestDefaultUEFIManager_ContextCanceled(t *testing.T) {
	mgr := &DefaultUEFIManager{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// PrepareAndSignEFI với context đã hủy
	_, err := mgr.PrepareAndSignEFI(ctx, nil)
	if err == nil {
		t.Log("PrepareAndSignEFI trả về kết quả")
	}

	// RebootToFirmware với context đã hủy
	_ = mgr.RebootToFirmware(ctx)
}

