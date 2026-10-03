package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchLeagueGameConfigContent_Normal(t *testing.T) {
	input := `[General]
WindowMode=0
PreferDX9Legacy=1
Width=1920
Height=1080

[Performance]
ShadowsEnabled=1
`
	patched, modified := PatchLeagueGameConfigContent(input)
	if !modified {
		t.Fatalf("Mong đợi modified=true nhưng nhận về false")
	}

	if !strings.Contains(patched, "WindowMode=2") {
		t.Errorf("Mong đợi WindowMode=2 nhưng không tìm thấy trong:\n%s", patched)
	}
	if !strings.Contains(patched, "BorderlessWindow=1") {
		t.Errorf("Mong đợi BorderlessWindow=1 nhưng không tìm thấy")
	}
	if !strings.Contains(patched, "PreferDX9Legacy=0") {
		t.Errorf("Mong đợi PreferDX9Legacy=0 nhưng không tìm thấy")
	}
	if !strings.Contains(patched, "ShadowsEnabled=1") {
		t.Errorf("Mong đợi giữ nguyên các section khác như ShadowsEnabled=1")
	}
}

func TestPatchLeagueGameConfigContent_Empty(t *testing.T) {
	patched, modified := PatchLeagueGameConfigContent("")
	if !modified {
		t.Fatalf("Mong đợi modified=true khi input rỗng")
	}

	if !strings.Contains(patched, "[General]") || !strings.Contains(patched, "WindowMode=2") {
		t.Errorf("Nội dung tạo mới không đầy đủ:\n%s", patched)
	}
}

func TestPatchLeagueGameConfigContent_AlreadyOptimal(t *testing.T) {
	input := `[General]
WindowMode=2
BorderlessWindow=1
PreferDX9Legacy=0
Width=1920
Height=1080
`
	_, modified := PatchLeagueGameConfigContent(input)
	if modified {
		t.Errorf("Mong đợi modified=false khi cấu hình đã tối ưu")
	}
}

func TestFormatHighPerfAdapterString(t *testing.T) {
	str := FormatHighPerfAdapterString(0x10DE, 0x1F0B)
	expected := "HighPerfAdapter=10DE&1F0B&00000000;SwapEffectUpgradeEnable=0;"
	if str != expected {
		t.Errorf("FormatHighPerfAdapterString mong đợi '%s', nhận về '%s'", expected, str)
	}

	str30 := FormatHighPerfAdapterString(0x10DE, 0x2189)
	expected30 := "HighPerfAdapter=10DE&2189&00000000;SwapEffectUpgradeEnable=0;"
	if str30 != expected30 {
		t.Errorf("FormatHighPerfAdapterString mong đợi '%s', nhận về '%s'", expected30, str30)
	}
}

func TestRiotInstallsJSON_Parsing(t *testing.T) {
	sampleJSON := `{
		"rc_default": "C:\\Riot Games\\Riot Client\\RiotClientServices.exe",
		"rc_live": "C:\\Riot Games\\Riot Client\\RiotClientServices.exe",
		"league_of_legends.live": "D:\\Riot Games\\League of Legends",
		"valorant.live": "E:\\Games\\VALORANT\\live"
	}`

	var installs RiotInstallsJSON
	err := json.Unmarshal([]byte(sampleJSON), &installs)
	if err != nil {
		t.Fatalf("Không thể parse RiotInstallsJSON: %v", err)
	}

	if installs.RCLive != `C:\Riot Games\Riot Client\RiotClientServices.exe` {
		t.Errorf("RCLive sai: %s", installs.RCLive)
	}
	if installs.LeagueLive != `D:\Riot Games\League of Legends` {
		t.Errorf("LeagueLive sai: %s", installs.LeagueLive)
	}
	if installs.ValorantLive != `E:\Games\VALORANT\live` {
		t.Errorf("ValorantLive sai: %s", installs.ValorantLive)
	}
}

func TestFixLeagueGameConfigFile_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "Config", "game.cfg")

	// Lần 1: Tạo mới từ chưa tồn tại
	patched, err := FixLeagueGameConfigFile(cfgPath)
	if err != nil {
		t.Fatalf("FixLeagueGameConfigFile thất bại: %v", err)
	}
	if !patched {
		t.Errorf("Mong đợi patched=true khi file chưa có")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("Không thể đọc file vừa ghi: %v", err)
	}
	if !strings.Contains(string(data), "WindowMode=2") {
		t.Errorf("File vừa ghi không có WindowMode=2:\n%s", string(data))
	}

	// Lần 2: Chạy lại khi đã tối ưu -> patched phải là false
	patchedAgain, err := FixLeagueGameConfigFile(cfgPath)
	if err != nil {
		t.Fatalf("FixLeagueGameConfigFile lần 2 thất bại: %v", err)
	}
	if patchedAgain {
		t.Errorf("Mong đợi patchedAgain=false khi file đã tối ưu")
	}
}

func TestCheckHAGSStatus(t *testing.T) {
	status, val := CheckHAGSStatus()
	t.Logf("HAGS Status: %s (code: %d)", status, val)
}
