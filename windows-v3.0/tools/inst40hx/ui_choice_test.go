package main

import (
	"40hxcore"
	"testing"
)

func TestPromptUIMode(t *testing.T) {
	origHook := hxcore.MsgBoxPromptHook
	defer func() { hxcore.MsgBoxPromptHook = origHook }()

	tests := []struct {
		name     string
		mockRes  int
		expected int
	}{
		{"User picks Web UI (Yes)", hxcore.IdYes, uiModeWeb},
		{"User picks Classic Go UI (No)", hxcore.IdNo, uiModeClassic},
		{"User cancels (Cancel)", hxcore.IdCancel, uiModeCancel},
		{"User closes dialog", 0, uiModeCancel},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hxcore.MsgBoxPromptHook = func(title, text string) int {
				if title != "CMP 40HX & 30HX Control Center" {
					t.Errorf("unexpected title: %s", title)
				}
				return tc.mockRes
			}

			res := promptUIMode()
			if res != tc.expected {
				t.Fatalf("expected mode %d, got %d", tc.expected, res)
			}
		})
	}
}
