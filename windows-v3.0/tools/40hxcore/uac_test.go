package hxcore

import "testing"

func TestIsAdmin(t *testing.T) {
	// Kiểm tra hàm IsAdmin thực thi trơn tru mà không panic
	_ = IsAdmin()
}

func TestMsgBoxYesNoCancel_Hook(t *testing.T) {
	origHook := MsgBoxPromptHook
	defer func() { MsgBoxPromptHook = origHook }()

	called := false
	MsgBoxPromptHook = func(title, text string) int {
		called = true
		if title != "Test Title" || text != "Test Message" {
			t.Errorf("unexpected args: title=%s text=%s", title, text)
		}
		return IdYes
	}

	res := MsgBoxYesNoCancel("Test Title", "Test Message")
	if !called {
		t.Fatalf("expected hook to be called")
	}
	if res != IdYes {
		t.Fatalf("expected IdYes, got %d", res)
	}
}
