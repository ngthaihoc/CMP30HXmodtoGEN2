package main

import (
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
