package hxcore

import "testing"

func TestIsAdmin(t *testing.T) {
	// Kiểm tra hàm IsAdmin thực thi trơn tru mà không panic
	_ = IsAdmin()
}
