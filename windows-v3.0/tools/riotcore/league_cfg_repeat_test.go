package riotcore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigureLeagueConfigPreservesContentOnSecondRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.cfg")
	input := "[General]\r\nWindowMode=0\r\nWidth=1920\r\n\r\n[Performance]\r\nGraphicsSlider=4\r\n"
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatalf("Không thể tạo game.cfg: %v", err)
	}

	want := "[General]\r\nWindowMode=2\r\nWidth=1920\r\n\r\n" +
		"BorderlessWindow=1\r\nPreferDX9LegacyMode=0\r\nDX11Support=1\r\n" +
		"[Performance]\r\nGraphicsSlider=4\r\n"
	for run := 1; run <= 2; run++ {
		if err := ConfigureLeagueConfig(path); err != nil {
			t.Fatalf("Lần %d: không thể cấu hình game.cfg: %v", run, err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Lần %d: không thể đọc game.cfg: %v", run, err)
		}
		if string(got) != want {
			t.Fatalf("Lần %d: game.cfg = %q, muốn %q", run, got, want)
		}
	}
}
