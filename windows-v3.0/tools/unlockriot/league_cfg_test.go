package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureLeagueConfig_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "game.cfg")

	initialContent := `[General]
Width=1920
Height=1080
WindowMode=0
Colors=32

[Performance]
GraphicsSlider=4
`
	if err := os.WriteFile(cfgPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial cfg: %v", err)
	}

	err := ConfigureLeagueConfig(cfgPath)
	if err != nil {
		t.Fatalf("ConfigureLeagueConfig failed: %v", err)
	}

	updated, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("Failed to read updated cfg: %v", err)
	}
	content := string(updated)

	if !strings.Contains(content, "WindowMode=2") {
		t.Errorf("Expected WindowMode=2, got content:\n%s", content)
	}
	if !strings.Contains(content, "BorderlessWindow=1") {
		t.Errorf("Expected BorderlessWindow=1, got content:\n%s", content)
	}
	if !strings.Contains(content, "Width=1920") || !strings.Contains(content, "GraphicsSlider=4") {
		t.Errorf("Other settings were not preserved:\n%s", content)
	}
	if strings.Contains(content, "WindowMode=0") {
		t.Errorf("Old WindowMode=0 still present:\n%s", content)
	}
}

func TestConfigureLeagueConfig_NoGeneralSection(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "game.cfg")

	initialContent := `[Performance]
GraphicsSlider=4
`
	if err := os.WriteFile(cfgPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial cfg: %v", err)
	}

	err := ConfigureLeagueConfig(cfgPath)
	if err != nil {
		t.Fatalf("ConfigureLeagueConfig failed: %v", err)
	}

	updated, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("Failed to read updated cfg: %v", err)
	}
	content := string(updated)

	if !strings.Contains(content, "[General]") || !strings.Contains(content, "WindowMode=2") || !strings.Contains(content, "BorderlessWindow=1") {
		t.Errorf("Missing [General] or settings in content:\n%s", content)
	}
}

func TestConfigureLeagueConfig_NonExistent(t *testing.T) {
	err := ConfigureLeagueConfig(`C:\nonexistent\game.cfg`)
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestFindAndConfigureLeagueConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	relDir := filepath.Join(tmpDir, "Riot Games", "League of Legends", "Config")
	if err := os.MkdirAll(relDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	cfgFile := filepath.Join(relDir, "game.cfg")
	if err := os.WriteFile(cfgFile, []byte("[General]\nWindowMode=0\n"), 0644); err != nil {
		t.Fatalf("Failed to write cfg: %v", err)
	}

	modified, err := FindAndConfigureLeagueConfigs([]string{tmpDir})
	if err != nil {
		t.Fatalf("FindAndConfigureLeagueConfigs failed: %v", err)
	}
	if len(modified) != 1 {
		t.Fatalf("Expected 1 modified file, got %d", len(modified))
	}
	if modified[0] != cfgFile {
		t.Errorf("Expected modified file %s, got %s", cfgFile, modified[0])
	}

	content, _ := os.ReadFile(cfgFile)
	if !strings.Contains(string(content), "WindowMode=2") {
		t.Errorf("Expected WindowMode=2 in file content:\n%s", string(content))
	}
}

