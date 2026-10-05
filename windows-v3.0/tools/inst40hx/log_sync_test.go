package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStartLogSync_LifecycleAndSync(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	f, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer f.Close()

	// Khởi chạy log sync với chu kỳ 10ms
	stopSync := startLogSync(f, 10*time.Millisecond)

	// Ghi dữ liệu vào file
	_, err = f.WriteString("hello log sync\n")
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Đợi ticker thực hiện sync
	time.Sleep(30 * time.Millisecond)

	// Dừng đồng bộ an toàn, kiểm soát vòng đời sạch sẽ
	stopSync()

	// Kiểm tra tính lũy đẳng (idempotent): dừng lần 2 không panic
	stopSync()

	// Truyền file nil phải trả về hàm no-op an toàn
	noOp := startLogSync(nil, 10*time.Millisecond)
	noOp()
}
