package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigureLeagueConfig đảm bảo thiết lập Borderless Windowed trong game.cfg
func ConfigureLeagueConfig(cfgPath string) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("không thể đọc file config: %w", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var newLines []string

	inGeneral := false
	generalFound := false
	windowModeFound := false
	borderlessFound := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if inGeneral {
				// Kết thúc section [General], chèn các thuộc tính còn thiếu
				if !windowModeFound {
					newLines = append(newLines, "WindowMode=2")
					windowModeFound = true
				}
				if !borderlessFound {
					newLines = append(newLines, "BorderlessWindow=1")
					borderlessFound = true
				}
				inGeneral = false
			}

			if strings.EqualFold(trimmed, "[General]") {
				inGeneral = true
				generalFound = true
			}
			newLines = append(newLines, line)
			continue
		}

		if inGeneral {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				if strings.EqualFold(key, "WindowMode") {
					newLines = append(newLines, "WindowMode=2")
					windowModeFound = true
					continue
				}
				if strings.EqualFold(key, "BorderlessWindow") {
					newLines = append(newLines, "BorderlessWindow=1")
					borderlessFound = true
					continue
				}
			}
		}

		// Giữ nguyên dòng cuối cùng nếu là rỗng khi split
		if i == len(lines)-1 && trimmed == "" {
			continue
		}
		newLines = append(newLines, line)
	}

	// Nếu file kết thúc khi đang trong [General]
	if inGeneral {
		if !windowModeFound {
			newLines = append(newLines, "WindowMode=2")
		}
		if !borderlessFound {
			newLines = append(newLines, "BorderlessWindow=1")
		}
	}

	// Nếu hoàn toàn không có [General]
	if !generalFound {
		if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
			newLines = append(newLines, "")
		}
		newLines = append(newLines, "[General]", "WindowMode=2", "BorderlessWindow=1")
	}

	var buf bytes.Buffer
	for _, l := range newLines {
		buf.WriteString(l)
		buf.WriteString("\r\n")
	}

	return os.WriteFile(cfgPath, buf.Bytes(), 0644)
}

// FindAndConfigureLeagueConfigs tìm và cấu hình tất cả các file game.cfg trên các ổ đĩa
func FindAndConfigureLeagueConfigs(drives []string) ([]string, error) {
	var modified []string

	relativePaths := []string{
		`Riot Games\League of Legends\Config\game.cfg`,
		`League of Legends\Config\game.cfg`,
		`Garena\Games\32787\Game\Config\game.cfg`,
	}

	for _, drive := range drives {
		for _, rel := range relativePaths {
			fullPath := filepath.Join(drive, rel)
			if fi, err := os.Stat(fullPath); err == nil && !fi.IsDir() {
				if err := ConfigureLeagueConfig(fullPath); err == nil {
					modified = append(modified, fullPath)
				}
			}
		}
	}

	return modified, nil
}
