package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// DiscoveredGames chứa danh sách đường dẫn các file thực thi và config game đã tìm thấy
type DiscoveredGames struct {
	ValorantExes   []string
	LeagueExes     []string
	RiotClientExes []string
	LeagueConfigs  []string
}

func (d *DiscoveredGames) merge(other DiscoveredGames) {
	d.ValorantExes = append(d.ValorantExes, other.ValorantExes...)
	d.LeagueExes = append(d.LeagueExes, other.LeagueExes...)
	d.RiotClientExes = append(d.RiotClientExes, other.RiotClientExes...)
	d.LeagueConfigs = append(d.LeagueConfigs, other.LeagueConfigs...)
}

func (d *DiscoveredGames) deduplicate() {
	d.ValorantExes = uniquePaths(d.ValorantExes)
	d.LeagueExes = uniquePaths(d.LeagueExes)
	d.RiotClientExes = uniquePaths(d.RiotClientExes)
	d.LeagueConfigs = uniquePaths(d.LeagueConfigs)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]bool)
	var res []string
	for _, p := range paths {
		if p == "" {
			continue
		}
		clean := filepath.Clean(p)
		lower := strings.ToLower(clean)
		if !seen[lower] {
			seen[lower] = true
			res = append(res, clean)
		}
	}
	return res
}

// parseInstallPathFromYAML đọc product_install_full_path từ file YAML của Riot Client
func parseInstallPathFromYAML(yamlPath string) string {
	f, err := os.Open(yamlPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "product_install_full_path:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "product_install_full_path:"))
			val = strings.Trim(val, `"'`)
			return filepath.FromSlash(val)
		}
	}
	return ""
}

// DiscoverGamesWithMetadata quét các file product_settings.yaml của Riot Client
func DiscoverGamesWithMetadata(programData string) DiscoveredGames {
	var res DiscoveredGames
	if programData == "" {
		programData = os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
	}

	// 1. Quét RiotClientInstalls.json để tìm RiotClientServices.exe
	instJson := filepath.Join(programData, "Riot Games", "RiotClientInstalls.json")
	if data, err := os.ReadFile(instJson); err == nil {
		var clientMap map[string]interface{}
		if err := json.Unmarshal(data, &clientMap); err == nil {
			for _, k := range []string{"rc_default", "rc_live"} {
				if v, ok := clientMap[k].(string); ok && v != "" {
					cleanV := filepath.Clean(filepath.FromSlash(v))
					if fi, err := os.Stat(cleanV); err == nil && !fi.IsDir() {
						res.RiotClientExes = append(res.RiotClientExes, cleanV)
					}
				}
			}
		}
	}

	// 2. Quét thư mục Metadata
	metaDir := filepath.Join(programData, "Riot Games", "Metadata")
	entries, err := os.ReadDir(metaDir)
	if err != nil {
		return res
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(metaDir, entry.Name())
		files, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if strings.HasSuffix(strings.ToLower(f.Name()), ".product_settings.yaml") {
				yamlPath := filepath.Join(subDir, f.Name())
				installPath := parseInstallPathFromYAML(yamlPath)
				if installPath == "" {
					continue
				}

				nameLower := strings.ToLower(entry.Name())
				if strings.Contains(nameLower, "valorant") {
					valCandidates := []string{
						filepath.Join(installPath, `live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`),
						filepath.Join(installPath, `ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`),
					}
					for _, c := range valCandidates {
						if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
							res.ValorantExes = append(res.ValorantExes, c)
						}
					}
				}

				if strings.Contains(nameLower, "league_of_legends") {
					lolCandidates := []string{
						filepath.Join(installPath, `Game\League of Legends.exe`),
						filepath.Join(installPath, `League of Legends.exe`),
					}
					for _, c := range lolCandidates {
						if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
							res.LeagueExes = append(res.LeagueExes, c)
						}
					}

					cfgCandidates := []string{
						filepath.Join(installPath, `Game\Config\game.cfg`),
						filepath.Join(installPath, `Config\game.cfg`),
					}
					for _, c := range cfgCandidates {
						if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
							res.LeagueConfigs = append(res.LeagueConfigs, c)
						}
					}
				}
			}
		}
	}

	return res
}

// discoverGamesFromRegistry quét Registry Uninstall để tìm đường dẫn cài đặt
func discoverGamesFromRegistry() DiscoveredGames {
	var res DiscoveredGames
	regPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	for _, regRoot := range regPaths {
		baseKey, err := registry.OpenKey(registry.LOCAL_MACHINE, regRoot, registry.READ)
		if err != nil {
			continue
		}
		subkeys, err := baseKey.ReadSubKeyNames(-1)
		baseKey.Close()
		if err != nil {
			continue
		}

		for _, sub := range subkeys {
			subLower := strings.ToLower(sub)
			isVal := strings.Contains(subLower, "valorant")
			isLol := strings.Contains(subLower, "league of legends") || strings.Contains(subLower, "league_of_legends")

			k, err := registry.OpenKey(registry.LOCAL_MACHINE, regRoot+`\`+sub, registry.QUERY_VALUE)
			if err != nil {
				continue
			}

			dispName, _, _ := k.GetStringValue("DisplayName")
			nameLower := strings.ToLower(dispName)
			if !isVal && strings.Contains(nameLower, "valorant") {
				isVal = true
			}
			if !isLol && strings.Contains(nameLower, "league of legends") {
				isLol = true
			}

			loc, _, _ := k.GetStringValue("InstallLocation")
			k.Close()

			if loc == "" || (!isVal && !isLol) {
				continue
			}

			if isVal {
				valCandidates := []string{
					filepath.Join(loc, `live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`),
					filepath.Join(loc, `ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`),
				}
				for _, c := range valCandidates {
					if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
						res.ValorantExes = append(res.ValorantExes, c)
					}
				}
			}

			if isLol {
				lolCandidates := []string{
					filepath.Join(loc, `Game\League of Legends.exe`),
					filepath.Join(loc, `League of Legends.exe`),
				}
				for _, c := range lolCandidates {
					if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
						res.LeagueExes = append(res.LeagueExes, c)
					}
				}
				cfgCandidates := []string{
					filepath.Join(loc, `Game\Config\game.cfg`),
					filepath.Join(loc, `Config\game.cfg`),
				}
				for _, c := range cfgCandidates {
					if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
						res.LeagueConfigs = append(res.LeagueConfigs, c)
					}
				}
			}
		}
	}

	return res
}

func isDriveRoot(p string) bool {
	clean := filepath.Clean(p)
	return len(clean) <= 3 && strings.Contains(clean, ":")
}

// smartScanGameExes quét đĩa nông để tìm nhị phân nếu các nguồn metadata/registry chưa thấy
func smartScanGameExes(roots []string, maxDepth int) DiscoveredGames {
	var res DiscoveredGames

	// Các thư mục hệ thống cần bỏ qua hoàn toàn để quét siêu nhanh
	skipDirs := map[string]bool{
		"windows":                   true,
		"$recycle.bin":              true,
		"system volume information": true,
		"node_modules":              true,
		"recovery":                  true,
		"msocache":                  true,
		"appdata":                   true,
		"programdata":               true,
		"users":                     true,
		"perflogs":                  true,
		".git":                      true,
		"documents and settings":   true,
		"$windows.~bt":              true,
		"$windows.~ws":              true,
	}

	gameKeywords := []string{
		"riot", "game", "valorant", "league", "shooter", "live", "binaries", "win64", "vng", "garena",
	}

	for _, root := range roots {
		isDrive := isDriveRoot(root)
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err == nil && rel != "." {
				parts := strings.Split(rel, string(os.PathSeparator))
				depth := len(parts)

				if info.IsDir() {
					dirName := strings.ToLower(info.Name())
					if skipDirs[dirName] {
						return filepath.SkipDir
					}

					// Nếu duyệt từ gốc ổ đĩa (C:\, D:\): ở tầng 1 chỉ duyệt Program Files hoặc thư mục có tên liên quan game
					if isDrive && depth == 1 {
						if strings.EqualFold(dirName, "program files") || strings.EqualFold(dirName, "program files (x86)") {
							return nil
						}
						isGameDir := false
						for _, kw := range gameKeywords {
							if strings.Contains(dirName, kw) {
								isGameDir = true
								break
							}
						}
						if !isGameDir {
							return filepath.SkipDir
						}
					}

					// Tại Program Files / Program Files (x86), chỉ đi sâu vào các thư mục liên quan đến game
					if depth == 2 && len(parts) >= 2 {
						parent := strings.ToLower(parts[0])
						if parent == "program files" || parent == "program files (x86)" {
							isGameDir := false
							for _, kw := range gameKeywords {
								if strings.Contains(dirName, kw) {
									isGameDir = true
									break
								}
							}
							if !isGameDir {
								return filepath.SkipDir
							}
						}
					}

					// Nếu vượt quá độ sâu, chỉ tiếp tục nếu thư mục liên quan đến game
					if depth > maxDepth {
						isGameRelated := false
						for _, kw := range gameKeywords {
							if strings.Contains(dirName, kw) {
								isGameRelated = true
								break
							}
						}
						if !isGameRelated {
							return filepath.SkipDir
						}
					}
					// Giới hạn tuyệt đối để không duyệt quá sâu
					if depth > 10 {
						return filepath.SkipDir
					}
				}
			}

			if !info.IsDir() {
				nameLower := strings.ToLower(info.Name())
				if nameLower == "valorant-win64-shipping.exe" {
					res.ValorantExes = append(res.ValorantExes, path)
				} else if nameLower == "league of legends.exe" {
					res.LeagueExes = append(res.LeagueExes, path)
					cfgPath := filepath.Join(filepath.Dir(path), "Config", "game.cfg")
					if fi, err := os.Stat(cfgPath); err == nil && !fi.IsDir() {
						res.LeagueConfigs = append(res.LeagueConfigs, cfgPath)
					}
				}
			}
			return nil
		})
	}

	return res
}

// FindAllRiotGameTargets tổng hợp phát hiện từ Metadata, Registry, đường dẫn quen thuộc và Quét đĩa
func FindAllRiotGameTargets(drives []string) DiscoveredGames {
	var all DiscoveredGames

	// 1. Quét Metadata từ ProgramData
	all.merge(DiscoverGamesWithMetadata(""))

	// 2. Quét Registry Uninstall
	all.merge(discoverGamesFromRegistry())

	// 3. Quét danh sách đường dẫn quen thuộc
	knownVal := []string{
		`Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`,
		`VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`,
		`Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`,
		`Games\Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe`,
	}
	knownLol := []string{
		`Riot Games\League of Legends\Game\League of Legends.exe`,
		`League of Legends\Game\League of Legends.exe`,
		`VNGGames\League of Legends\Game\League of Legends.exe`,
		`Garena\Games\32787\Game\League of Legends.exe`,
		`Games\League of Legends\Game\League of Legends.exe`,
		`Games\Riot Games\League of Legends\Game\League of Legends.exe`,
	}
	knownCfg := []string{
		`Riot Games\League of Legends\Game\Config\game.cfg`,
		`Riot Games\League of Legends\Config\game.cfg`,
		`League of Legends\Game\Config\game.cfg`,
		`League of Legends\Config\game.cfg`,
		`VNGGames\League of Legends\Game\Config\game.cfg`,
		`VNGGames\League of Legends\Config\game.cfg`,
		`Garena\Games\32787\Game\Config\game.cfg`,
		`Games\League of Legends\Game\Config\game.cfg`,
	}
	knownClients := []string{
		`Riot Games\Riot Client\RiotClientServices.exe`,
		`Riot Client\RiotClientServices.exe`,
	}

	for _, d := range drives {
		for _, v := range knownVal {
			p := filepath.Join(d, v)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				all.ValorantExes = append(all.ValorantExes, p)
			}
		}
		for _, l := range knownLol {
			p := filepath.Join(d, l)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				all.LeagueExes = append(all.LeagueExes, p)
			}
		}
		for _, c := range knownCfg {
			p := filepath.Join(d, c)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				all.LeagueConfigs = append(all.LeagueConfigs, p)
			}
		}
		for _, rc := range knownClients {
			p := filepath.Join(d, rc)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				all.RiotClientExes = append(all.RiotClientExes, p)
			}
		}
	}

	// 4. Nếu vẫn chưa tìm thấy Valorant hoặc LoL, chạy smart scan trên các ổ đĩa với độ sâu 4
	if len(all.ValorantExes) == 0 || len(all.LeagueExes) == 0 {
		all.merge(smartScanGameExes(drives, 4))
	}

	all.deduplicate()
	return all
}
