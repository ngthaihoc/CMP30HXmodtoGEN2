package riotcore

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	GpuClassPath             = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	GraphicsDriversKey       = `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`
	PciEnumKey               = `SYSTEM\CurrentControlSet\Enum\PCI`
	CiConfigPath             = `SYSTEM\CurrentControlSet\Control\CI\Config`
	DxGraphicsSettingsPath   = `Software\Microsoft\DirectX\GraphicsSettings`
	DxUserGpuPreferencesPath = `Software\Microsoft\DirectX\UserGpuPreferences`

	// FeatureScore 0xD1 (209 thập phân): Định danh card đồ họa Mobile / MSHybrid discrete GPU (Render dGPU)
	// Tránh việc DirectX / DWM coi card là Desktop có cổng xuất hình rời dẫn đến lỗi EnumOutputs rỗng
	FeatureScoreMSHybrid = uint32(0xD1)
)

// AdapterRole phân loại vai trò của card đồ họa trong hệ thống Microsoft Hybrid Graphics
type AdapterRole int

const (
	RoleUnknown AdapterRole = iota
	RoleNvidiaRender
	RoleDisplayHost
)

// ClassifyAdapter phân loại card dựa trên Hardware ID, Driver Description và Provider
func ClassifyAdapter(matchID, desc, provider string) AdapterRole {
	combined := strings.ToUpper(fmt.Sprintf("%s %s %s", matchID, desc, provider))

	// Loại bỏ Basic Display Adapter hoặc Microsoft generic
	if strings.Contains(combined, "BASIC DISPLAY") || strings.Contains(combined, "MICROSOFT") {
		return RoleUnknown
	}

	// Nhận diện NVIDIA Discrete GPU (Render dGPU)
	if strings.Contains(combined, "VEN_10DE") ||
		strings.Contains(combined, "NVIDIA") ||
		strings.Contains(combined, "CMP 40HX") ||
		strings.Contains(combined, "CMP 30HX") ||
		strings.Contains(combined, "GEFORCE") {
		return RoleNvidiaRender
	}

	// Nhận diện Host Display Adapter (iGPU Intel / AMD xuất hình qua cổng màn hình bo mạch chủ)
	if strings.Contains(combined, "VEN_1002") || // AMD
		strings.Contains(combined, "VEN_8086") || // Intel
		strings.Contains(combined, "RADEON") ||
		strings.Contains(combined, "INTEL") ||
		strings.Contains(combined, "UHD GRAPHICS") ||
		strings.Contains(combined, "HD GRAPHICS") ||
		strings.Contains(combined, "IRIS") {
		return RoleDisplayHost
	}

	return RoleUnknown
}

// RegistryConfigResult chứa kết quả cấu hình Registry cho driver
type RegistryConfigResult struct {
	NvidiaIndices           []string
	DisplayIndices          []string
	TdrConfigured           bool
	IcdRegistered           bool
	FriendlyNamed           bool
	BlocklistDisabled       bool
	DirectXGlobalConfigured bool
	HighPerfDID             string
	Details                 []string
}

// setDWords chỉ báo thành công khi mọi giá trị được ghi thành công.
func setDWords(root registry.Key, path string, values map[string]uint32) error {
	k, _, err := graphicsRegistry.CreateKey(root, path, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("mở Registry %s: %w", path, err)
	}
	defer k.Close()
	var errs []error
	for name, value := range values {
		if err := k.SetDWordValue(name, value); err != nil {
			errs = append(errs, fmt.Errorf("ghi %s\\%s: %w", path, name, err))
		}
	}
	return errors.Join(errs...)
}

// DisableVulnerableDriverBlocklist giữ chính sách cấu hình chung của dự án.
func DisableVulnerableDriverBlocklist(log io.Writer) error {
	return setDWords(registry.LOCAL_MACHINE, CiConfigPath, map[string]uint32{"VulnerableDriverBlocklistEnable": 0})
}

// routingSettings giữ nguyên tùy chọn DirectX ngoài HighPerfAdapter.
func routingSettings(current, did string) string {
	var parts []string
	hasSwap := false
	for _, part := range strings.Split(current, ";") {
		part = strings.TrimSpace(part)
		key, _, _ := strings.Cut(part, "=")
		if part == "" || strings.EqualFold(key, "HighPerfAdapter") {
			continue
		}
		if strings.EqualFold(key, "SwapEffectUpgradeEnable") {
			hasSwap = true
		}
		parts = append(parts, part)
	}
	if did != "" {
		parts = append(parts, "HighPerfAdapter="+did)
		if !hasSwap {
			parts = append(parts, "SwapEffectUpgradeEnable=1")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";") + ";"
}

func writeGlobalRouting(did string) error {
	k, _, err := graphicsRegistry.CreateKey(registry.CURRENT_USER, DxUserGpuPreferencesPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	current, _, readErr := k.GetStringValue("DirectXUserGlobalSettings")
	if readErr != nil && !errors.Is(readErr, registry.ErrNotExist) {
		return readErr
	}
	value := routingSettings(current, did)
	if value == "" {
		err = k.DeleteValue("DirectXUserGlobalSettings")
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	return k.SetStringValue("DirectXUserGlobalSettings", value)
}

// ConfigureDirectXGlobalRouting định tuyến CMP và phản ánh lỗi ghi thực tế.
func ConfigureDirectXGlobalRouting(log io.Writer, highPerfDID string) error {
	if highPerfDID != "10DE&2189" && highPerfDID != "10DE&1F0B" {
		return fmt.Errorf("không có GPU CMP hợp lệ để định tuyến DirectX")
	}
	return errors.Join(
		setDWords(registry.CURRENT_USER, DxGraphicsSettingsPath, map[string]uint32{"DefaultHighPerfGPUApplicable": 1, "SpecificGPUOptionApplicable": 1}),
		writeGlobalRouting(highPerfDID),
	)
}

// ConfigureDriverClassRegistry dùng cùng topology, chính sách và kết quả cho mọi caller.
func ConfigureDriverClassRegistry(log io.Writer, deviceID uint16) (*RegistryConfigResult, error) {
	if log == nil {
		log = io.Discard
	}
	res := &RegistryConfigResult{}
	topo, err := ScanGPUTopology()
	if err != nil {
		return res, err
	}
	for _, gpu := range topo.RenderGPUs {
		if (gpu.DID == "10DE&2189" || gpu.DID == "10DE&1F0B") && (deviceID == 0 || gpu.DID == fmt.Sprintf("10DE&%04X", deviceID)) {
			res.HighPerfDID = gpu.DID
			break
		}
	}
	if res.HighPerfDID == "" {
		return res, fmt.Errorf("không tìm thấy GPU CMP render phù hợp; chưa thay đổi Registry")
	}
	var errs []error
	record := func(label string, err error) bool {
		if err != nil {
			res.Details = append(res.Details, label+": "+err.Error())
			errs = append(errs, fmt.Errorf("%s: %w", label, err))
			fmt.Fprintf(log, "[!] %s: %v\n", label, err)
			return false
		}
		fmt.Fprintf(log, "[V] %s\n", label)
		res.Details = append(res.Details, label)
		return true
	}
	res.BlocklistDisabled = record("Cấu hình danh sách chặn driver", DisableVulnerableDriverBlocklist(log))
	for _, gpu := range topo.RenderGPUs {
		if gpu.DID != res.HighPerfDID {
			continue
		}
		path := GpuClassPath + `\` + gpu.ClassIndex
		ok := record("Cấu hình CMP render "+gpu.ClassIndex, setDWords(registry.LOCAL_MACHINE, path, map[string]uint32{
			"FeatureScore": FeatureScoreMSHybrid, "EnableMsHybrid": 1, "EnableCrossAdapterScanOut": 1, "EnableCoproc": 1, "D3ColdSupported": 0, "RmDisableGpuPowerMgmt": 1,
		}))
		if k, err := graphicsRegistry.OpenKey(registry.LOCAL_MACHINE, path, registry.SET_VALUE); err == nil {
			for _, name := range []string{"AdapterType", "LargePageMinimum"} {
				if err := k.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
					ok = record("Xóa "+name, err) && ok
				}
			}
			k.Close()
		} else {
			ok = record("Mở cấu hình CMP", err) && ok
		}
		if ok {
			res.NvidiaIndices = append(res.NvidiaIndices, gpu.ClassIndex)
		}
	}
	for _, gpu := range topo.DisplayHosts {
		if record("Cấu hình GPU xuất hình "+gpu.ClassIndex, setDWords(registry.LOCAL_MACHINE, GpuClassPath+`\`+gpu.ClassIndex, map[string]uint32{"EnableMsHybrid": 2})) {
			res.DisplayIndices = append(res.DisplayIndices, gpu.ClassIndex)
		}
	}
	res.TdrConfigured = record("Cấu hình thời gian chờ TDR", setDWords(registry.LOCAL_MACHINE, GraphicsDriversKey, map[string]uint32{"TdrDelay": 10, "TdrDdiDelay": 10, "TdrLevel": 3}))
	res.IcdRegistered = true
	for _, path := range []string{`SOFTWARE\Khronos\OpenCL\Vendors`, `SOFTWARE\WOW6432Node\Khronos\OpenCL\Vendors`} {
		if !record("Đăng ký OpenCL", setDWords(registry.LOCAL_MACHINE, path, map[string]uint32{"nvopencl.dll": 0, "nvopencl64.dll": 0})) {
			res.IcdRegistered = false
		}
	}
	for _, path := range []string{`SOFTWARE\Khronos\Vulkan\Drivers`, `SOFTWARE\WOW6432Node\Khronos\Vulkan\Drivers`} {
		if !record("Đăng ký Vulkan", setDWords(registry.LOCAL_MACHINE, path, map[string]uint32{"nv-vk64.json": 0, "nv-vk32.json": 0})) {
			res.IcdRegistered = false
		}
	}
	res.DirectXGlobalConfigured = record("Định tuyến DirectX "+res.HighPerfDID, ConfigureDirectXGlobalRouting(log, res.HighPerfDID))
	var friendlyErr error
	res.FriendlyNamed, friendlyErr = syncFriendlyNames(res.HighPerfDID)
	if friendlyErr != nil {
		record("Đồng bộ tên GPU", friendlyErr)
	}
	return res, errors.Join(errs...)
}

func syncFriendlyNames(did string) (bool, error) {
	base, err := graphicsRegistry.OpenKey(registry.LOCAL_MACHINE, PciEnumKey, registry.ENUMERATE_SUB_KEYS)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer base.Close()
	devices, err := base.ReadSubKeyNames(-1)
	if err != nil {
		return false, err
	}
	changed := false
	for _, device := range devices {
		if ExtractShortDID(device) != did {
			continue
		}
		path := PciEnumKey + `\` + device
		instances, err := graphicsRegistry.OpenKey(registry.LOCAL_MACHINE, path, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			return changed, err
		}
		names, err := instances.ReadSubKeyNames(-1)
		instances.Close()
		if err != nil {
			return changed, err
		}
		for _, name := range names {
			k, err := graphicsRegistry.OpenKey(registry.LOCAL_MACHINE, path+`\`+name, registry.QUERY_VALUE|registry.SET_VALUE)
			if err != nil {
				return changed, err
			}
			current, _, readErr := k.GetStringValue("FriendlyName")
			if readErr != nil && !errors.Is(readErr, registry.ErrNotExist) {
				k.Close()
				return changed, readErr
			}
			if current == "" || strings.Contains(strings.ToUpper(current), "UNKNOWN") {
				label := "NVIDIA CMP 30HX"
				if did == "10DE&1F0B" {
					label = "NVIDIA CMP 40HX"
				}
				err = k.SetStringValue("FriendlyName", label)
				if err != nil {
					k.Close()
					return changed, err
				}
				changed = true
			}
			k.Close()
		}
	}
	return changed, nil
}

// RevertGraphicsOptimization khôi phục blocklist và gỡ riêng lựa chọn HighPerfAdapter.
func RevertGraphicsOptimization(log io.Writer) error {
	return errors.Join(setDWords(registry.LOCAL_MACHINE, CiConfigPath, map[string]uint32{"VulnerableDriverBlocklistEnable": 1}), writeGlobalRouting(""))
}
