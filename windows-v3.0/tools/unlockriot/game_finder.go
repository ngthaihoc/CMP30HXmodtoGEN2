package main

import "riotcore"

// DiscoveredGames chứa danh sách đường dẫn các file thực thi và config game đã tìm thấy
type DiscoveredGames = riotcore.DiscoveredGames

func uniquePaths(paths []string) []string {
	return riotcore.UniquePaths(paths)
}

func parseInstallPathFromYAML(yamlPath string) string {
	return riotcore.ParseInstallPathFromYAML(yamlPath)
}

// DiscoverGamesWithMetadata quét các file product_settings.yaml của Riot Client
func DiscoverGamesWithMetadata(programData string) DiscoveredGames {
	return riotcore.DiscoverGamesWithMetadata(programData)
}

func smartScanGameExes(roots []string, maxDepth int) DiscoveredGames {
	return riotcore.SmartScanGameExes(roots, maxDepth)
}

// FindAllRiotGameTargets tổng hợp phát hiện từ Metadata, Registry, đường dẫn quen thuộc và Quét đĩa
func FindAllRiotGameTargets(drives []string) DiscoveredGames {
	return riotcore.FindAllRiotGameTargets(drives)
}
