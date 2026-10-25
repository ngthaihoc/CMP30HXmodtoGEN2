package riotcore

import "golang.org/x/sys/windows/registry"

// registryStore là seam nội bộ giữa cấu hình đồ họa và Registry Windows.
type registryStore interface {
	OpenKey(registry.Key, string, uint32) (registryKey, error)
	CreateKey(registry.Key, string, uint32) (registryKey, bool, error)
}

type registryKey interface {
	GetStringValue(string) (string, uint32, error)
	ReadSubKeyNames(int) ([]string, error)
	SetDWordValue(string, uint32) error
	SetStringValue(string, string) error
	DeleteValue(string) error
	Close() error
}

type windowsRegistry struct{}

func (windowsRegistry) OpenKey(root registry.Key, path string, access uint32) (registryKey, error) {
	return registry.OpenKey(root, path, access)
}

func (windowsRegistry) CreateKey(root registry.Key, path string, access uint32) (registryKey, bool, error) {
	return registry.CreateKey(root, path, access)
}

var graphicsRegistry registryStore = windowsRegistry{}
