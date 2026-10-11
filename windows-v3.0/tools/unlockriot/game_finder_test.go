package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseProductSettingsYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "product_settings.yaml")
	content := `product_id: "valorant"
product_install_full_path: "D:/MyCustomGames/VALORANT"
product_patchline: "live"
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write yaml: %v", err)
	}

	p := parseInstallPathFromYAML(yamlPath)
	expected := filepath.Clean("D:/MyCustomGames/VALORANT")
	if filepath.Clean(p) != expected {
		t.Fatalf("expected %s, got %s", expected, p)
	}
}

func TestDiscoverGamesFromMetadata(t *testing.T) {
	tmpRoot := t.TempDir()
	programData := filepath.Join(tmpRoot, "ProgramData")
	metaVal := filepath.Join(programData, "Riot Games", "Metadata", "valorant.live")
	metaLol := filepath.Join(programData, "Riot Games", "Metadata", "league_of_legends.live")

	if err := os.MkdirAll(metaVal, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(metaLol, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake install targets in a completely custom path
	customGameDir := filepath.Join(tmpRoot, "CustomDrive", "GameOnline")
	valDir := filepath.Join(customGameDir, "Shooter_Valo")
	lolDir := filepath.Join(customGameDir, "Moba_LoL")

	valExe := filepath.Join(valDir, "live", "ShooterGame", "Binaries", "Win64", "VALORANT-Win64-Shipping.exe")
	lolExe := filepath.Join(lolDir, "Game", "League of Legends.exe")
	lolCfg := filepath.Join(lolDir, "Game", "Config", "game.cfg")

	for _, p := range []string{valExe, lolExe, lolCfg} {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Write metadata YAMLs pointing to custom paths
	_ = os.WriteFile(filepath.Join(metaVal, "valorant.live.product_settings.yaml"),
		[]byte("product_install_full_path: \""+filepath.ToSlash(valDir)+"\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(metaLol, "league_of_legends.live.product_settings.yaml"),
		[]byte("product_install_full_path: \""+filepath.ToSlash(lolDir)+"\"\n"), 0644)

	res := DiscoverGamesWithMetadata(programData)

	foundVal := false
	for _, exe := range res.ValorantExes {
		if strings.EqualFold(filepath.Clean(exe), filepath.Clean(valExe)) {
			foundVal = true
		}
	}
	if !foundVal {
		t.Fatalf("failed to discover custom Valorant exe: %v", res.ValorantExes)
	}

	foundLol := false
	for _, exe := range res.LeagueExes {
		if strings.EqualFold(filepath.Clean(exe), filepath.Clean(lolExe)) {
			foundLol = true
		}
	}
	if !foundLol {
		t.Fatalf("failed to discover custom LoL exe: %v", res.LeagueExes)
	}

	foundCfg := false
	for _, cfg := range res.LeagueConfigs {
		if strings.EqualFold(filepath.Clean(cfg), filepath.Clean(lolCfg)) {
			foundCfg = true
		}
	}
	if !foundCfg {
		t.Fatalf("failed to discover custom LoL config: %v", res.LeagueConfigs)
	}
}

func TestSmartDirectoryScan_Fallback(t *testing.T) {
	tmpRoot := t.TempDir()
	customDir := filepath.Join(tmpRoot, "DeepCustom", "Games")
	valExe := filepath.Join(customDir, "MyValo", "live", "ShooterGame", "Binaries", "Win64", "VALORANT-Win64-Shipping.exe")
	_ = os.MkdirAll(filepath.Dir(valExe), 0755)
	_ = os.WriteFile(valExe, []byte("dummy"), 0644)

	found := smartScanGameExes([]string{tmpRoot}, 5)
	valFound := false
	for _, e := range found.ValorantExes {
		if strings.EqualFold(filepath.Clean(e), filepath.Clean(valExe)) {
			valFound = true
		}
	}
	if !valFound {
		t.Fatalf("smart directory scan failed to find deeply nested Valorant exe: %v", found.ValorantExes)
	}
}
