package main

import "riotcore"

// ConfigureLeagueConfig đảm bảo thiết lập Borderless Windowed và DirectX 11 trong game.cfg
func ConfigureLeagueConfig(cfgPath string) error {
	return riotcore.ConfigureLeagueConfig(cfgPath)
}

// FindAndConfigureLeagueConfigs tìm và cấu hình tất cả các file game.cfg trên các ổ đĩa
func FindAndConfigureLeagueConfigs(drives []string) ([]string, error) {
	return riotcore.FindAndConfigureLeagueConfigs(drives)
}
