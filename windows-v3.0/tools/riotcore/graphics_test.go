package riotcore

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

type memoryRegistry struct {
	keys      map[string]map[string]interface{}
	failValue string
	writes    int
}
type memoryKey struct {
	store *memoryRegistry
	path  string
}

func registryPath(root registry.Key, path string) string { return fmt.Sprintf("%d\\%s", root, path) }
func (m *memoryRegistry) OpenKey(root registry.Key, path string, access uint32) (registryKey, error) {
	p := registryPath(root, path)
	if _, ok := m.keys[p]; !ok {
		return nil, registry.ErrNotExist
	}
	return memoryKey{m, p}, nil
}
func (m *memoryRegistry) CreateKey(root registry.Key, path string, access uint32) (registryKey, bool, error) {
	p := registryPath(root, path)
	if m.keys[p] == nil {
		m.keys[p] = map[string]interface{}{}
	}
	return memoryKey{m, p}, false, nil
}
func (k memoryKey) Close() error { return nil }
func (k memoryKey) ReadSubKeyNames(n int) ([]string, error) {
	var names []string
	for path := range k.store.keys {
		if tail, ok := strings.CutPrefix(path, k.path+`\`); ok && !strings.Contains(tail, `\`) {
			names = append(names, tail)
		}
	}
	sort.Strings(names)
	return names, nil
}
func (k memoryKey) GetStringValue(name string) (string, uint32, error) {
	if value, ok := k.store.keys[k.path][name].(string); ok {
		return value, registry.SZ, nil
	}
	return "", 0, registry.ErrNotExist
}
func (k memoryKey) set(name string, value interface{}) error {
	if name == k.store.failValue {
		return errors.New("ghi Registry bị từ chối")
	}
	k.store.writes++
	k.store.keys[k.path][name] = value
	return nil
}
func (k memoryKey) SetDWordValue(name string, value uint32) error { return k.set(name, value) }
func (k memoryKey) SetStringValue(name, value string) error       { return k.set(name, value) }
func (k memoryKey) DeleteValue(name string) error                 { delete(k.store.keys[k.path], name); return nil }

func graphicsFixture(t *testing.T) *memoryRegistry {
	t.Helper()
	m := &memoryRegistry{keys: map[string]map[string]interface{}{
		registryPath(registry.LOCAL_MACHINE, GpuClassPath):         {},
		registryPath(registry.LOCAL_MACHINE, GpuClassPath+`\0000`): {"MatchingDeviceId": `PCI\VEN_10DE&DEV_2504`, "DriverDesc": "NVIDIA GeForce"},
		registryPath(registry.LOCAL_MACHINE, GpuClassPath+`\0001`): {"MatchingDeviceId": `PCI\VEN_10DE&DEV_2189`, "DriverDesc": "NVIDIA CMP 30HX"},
	}}
	previous, previousEnum := graphicsRegistry, enumDisplayDevicesHook
	graphicsRegistry = m
	enumDisplayDevicesHook = func() []ActiveDisplayTarget {
		return []ActiveDisplayTarget{{DeviceID: `PCI\VEN_10DE&DEV_2504&SUBSYS_1234`, IsAttached: true}}
	}
	t.Cleanup(func() { graphicsRegistry = previous; enumDisplayDevicesHook = previousEnum })
	return m
}

func TestConfigureDriverClassRegistryRoutesCMPAndPreservesDisplayHost(t *testing.T) {
	m := graphicsFixture(t)
	res, err := ConfigureDriverClassRegistry(nil, 0x2189)
	if err != nil {
		t.Fatal(err)
	}
	host := m.keys[registryPath(registry.LOCAL_MACHINE, GpuClassPath+`\0000`)]
	cmp := m.keys[registryPath(registry.LOCAL_MACHINE, GpuClassPath+`\0001`)]
	if host["EnableMsHybrid"] != uint32(2) || host["FeatureScore"] != nil || cmp["EnableMsHybrid"] != uint32(1) || cmp["EnableCrossAdapterScanOut"] != uint32(1) {
		t.Fatalf("Phân vai GPU sai: host=%v, CMP=%v", host, cmp)
	}
	if !res.DirectXGlobalConfigured || res.HighPerfDID != "10DE&2189" || !res.TdrConfigured || !res.BlocklistDisabled {
		t.Fatalf("Kết quả thiếu bước: %+v", res)
	}
}

func TestConfigureDriverClassRegistryNoCMPDoesNotWrite(t *testing.T) {
	m := graphicsFixture(t)
	delete(m.keys, registryPath(registry.LOCAL_MACHINE, GpuClassPath+`\0001`))
	res, err := ConfigureDriverClassRegistry(nil, 0)
	if err == nil || m.writes != 0 || res.DirectXGlobalConfigured {
		t.Fatalf("Không có CMP không được ghi cấu hình: %+v, %v, ghi=%d", res, err, m.writes)
	}
}

func TestConfigureDriverClassRegistryReportsPartialFailure(t *testing.T) {
	m := graphicsFixture(t)
	m.failValue = "TdrDelay"
	res, err := ConfigureDriverClassRegistry(nil, 0x2189)
	if err == nil || res.TdrConfigured || !res.DirectXGlobalConfigured {
		t.Fatalf("Phải phản ánh lỗi từng bước: %+v, %v", res, err)
	}
	m.failValue = "DirectXUserGlobalSettings"
	res, err = ConfigureDriverClassRegistry(nil, 0x2189)
	if err == nil || res.DirectXGlobalConfigured {
		t.Fatalf("Không được báo định tuyến thành công khi ghi lỗi: %+v, %v", res, err)
	}
}

func TestDirectXGlobalRouterRevertKeepsOtherPreferences(t *testing.T) {
	m := graphicsFixture(t)
	path := registryPath(registry.CURRENT_USER, DxUserGpuPreferencesPath)
	m.keys[path] = map[string]interface{}{"DirectXUserGlobalSettings": "OtherOption=7;SwapEffectUpgradeEnable=0;HighPerfAdapter=OLD;"}
	if err := ConfigureDirectXGlobalRouting(nil, "10DE&2189"); err != nil {
		t.Fatal(err)
	}
	value := m.keys[path]["DirectXUserGlobalSettings"].(string)
	if !strings.Contains(value, "OtherOption=7;") || !strings.Contains(value, "SwapEffectUpgradeEnable=0;") || !strings.Contains(value, "HighPerfAdapter=10DE&2189;") {
		t.Fatal(value)
	}
	if err := RevertGraphicsOptimization(nil); err != nil {
		t.Fatal(err)
	}
	value = m.keys[path]["DirectXUserGlobalSettings"].(string)
	if value != "OtherOption=7;SwapEffectUpgradeEnable=0;" {
		t.Fatal(value)
	}
}
