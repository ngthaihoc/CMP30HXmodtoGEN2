package main

import (
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
	NvidiaIndices          []string
	DisplayIndices         []string
	TdrConfigured          bool
	IcdRegistered          bool
	FriendlyNamed          bool
	BlocklistDisabled      bool
	DirectXGlobalConfigured bool
	HighPerfDID            string
	Details                []string
}

// DisableVulnerableDriverBlocklist tắt danh sách đen driver của Windows để ngăn Vanguard/Windows chặn driver mod
func DisableVulnerableDriverBlocklist(log io.Writer) error {
	if log == nil {
		log = io.Discard
	}
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, CiConfigPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("không thể mở Registry CI\\Config: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("VulnerableDriverBlocklistEnable", 0); err != nil {
		return fmt.Errorf("không thể ghi VulnerableDriverBlocklistEnable=0: %w", err)
	}
	fmt.Fprintln(log, "  [V] Đã tắt Microsoft Vulnerable Driver Blocklist (ngăn Vanguard/Windows chặn driver mod).")
	return nil
}

// ConfigureDirectXGlobalRouting thiết lập định tuyến GPU hiệu năng cao toàn cục cho DirectX
func ConfigureDirectXGlobalRouting(log io.Writer, highPerfDID string) error {
	if log == nil {
		log = io.Discard
	}

	// Cho phép hệ thống chọn GPU High-Performance trong cài đặt DirectX
	if gsKey, _, err := registry.CreateKey(registry.CURRENT_USER, DxGraphicsSettingsPath, registry.SET_VALUE); err == nil {
		_ = gsKey.SetDWordValue("DefaultHighPerfGPUApplicable", 1)
		_ = gsKey.SetDWordValue("SpecificGPUOptionApplicable", 1)
		gsKey.Close()
		fmt.Fprintln(log, "  [V] Đã kích hoạt tùy chọn High-Performance trong DirectX Graphics Settings.")
	}

	// Gán HighPerfAdapter toàn cục
	if highPerfDID != "" {
		ugpKey, _, err := registry.CreateKey(registry.CURRENT_USER, DxUserGpuPreferencesPath, registry.QUERY_VALUE|registry.SET_VALUE)
		if err == nil {
			defer ugpKey.Close()
			current, _, _ := ugpKey.GetStringValue("DirectXUserGlobalSettings")
			swapEffect := "SwapEffectUpgradeEnable=1;"
			if strings.Contains(current, "SwapEffectUpgradeEnable=0") {
				swapEffect = "SwapEffectUpgradeEnable=0;"
			}
			newSetting := fmt.Sprintf("HighPerfAdapter=%s;%s", highPerfDID, swapEffect)
			if err := ugpKey.SetStringValue("DirectXUserGlobalSettings", newSetting); err == nil {
				fmt.Fprintf(log, "  [V] Đã thiết lập HighPerfAdapter toàn cục cho DirectX: %s\n", highPerfDID)
			}
		}
	}
	return nil
}

// ConfigureDriverClassRegistry tự động cấu hình MSHybrid, CASO, TDR, ICD và DirectX toàn cục
func ConfigureDriverClassRegistry(log io.Writer, gpuModel GPUModel) (*RegistryConfigResult, error) {
	if log == nil {
		log = io.Discard
	}

	res := &RegistryConfigResult{}

	// Tắt Vulnerable Driver Blocklist
	if err := DisableVulnerableDriverBlocklist(log); err == nil {
		res.BlocklistDisabled = true
	} else {
		fmt.Fprintf(log, "  [!] Cảnh báo tắt Vulnerable Driver Blocklist: %v\n", err)
	}

	// Quét cấu trúc liên kết GPU (GPU xuất hình vs GPU render 3D)
	topo, _ := ScanGPUTopology()

	// 1. Quét và cấu hình Class Display Adapter
	baseKey, err := registry.OpenKey(registry.LOCAL_MACHINE, GpuClassPath, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return res, fmt.Errorf("không thể mở Driver Class Registry: %w", err)
	}
	defer baseKey.Close()

	subkeys, err := baseKey.ReadSubKeyNames(-1)
	if err != nil {
		return res, fmt.Errorf("không thể đọc danh sách subkeys trong Driver Class: %w", err)
	}

	for _, sub := range subkeys {
		if len(sub) != 4 {
			continue
		}

		subPath := GpuClassPath + `\` + sub
		sk, err := registry.OpenKey(registry.LOCAL_MACHINE, subPath, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			continue
		}

		matchID, _, _ := sk.GetStringValue("MatchingDeviceId")
		desc, _, _ := sk.GetStringValue("DriverDesc")
		provider, _, _ := sk.GetStringValue("ProviderName")

		// Kiểm tra vai trò dựa trên topo đã quét trước, nếu không có thì fallback sang ClassifyAdapter
		role := RoleUnknown
		if topo != nil {
			for _, h := range topo.DisplayHosts {
				if h.ClassIndex == sub {
					role = RoleDisplayHost
					break
				}
			}
			if role == RoleUnknown {
				for _, r := range topo.RenderGPUs {
					if r.ClassIndex == sub {
						role = RoleNvidiaRender
						break
					}
				}
			}
		}
		if role == RoleUnknown {
			role = ClassifyAdapter(matchID, desc, provider)
		}

		switch role {
		case RoleNvidiaRender:
			res.NvidiaIndices = append(res.NvidiaIndices, sub)
			fmt.Fprintf(log, "  -> Phát hiện Render dGPU (NVIDIA) tại Class Index [%s]: %s\n", sub, desc)

			_ = sk.SetDWordValue("FeatureScore", FeatureScoreMSHybrid)
			_ = sk.SetDWordValue("EnableMsHybrid", 1)
			_ = sk.SetDWordValue("EnableCrossAdapterScanOut", 1)
			_ = sk.SetDWordValue("EnableCoproc", 1)

			_ = sk.SetDWordValue("D3ColdSupported", 0)
			_ = sk.SetDWordValue("RmDisableGpuPowerMgmt", 1)

			_ = sk.DeleteValue("AdapterType")
			_ = sk.DeleteValue("LargePageMinimum")

			res.Details = append(res.Details, fmt.Sprintf("[%s] Cấu hình Render dGPU: FeatureScore=0xD1, EnableMsHybrid=1, CASO=1, D3Cold=0", sub))
			fmt.Fprintf(log, "     [V] Đã áp dụng: FeatureScore=0xD1, EnableMsHybrid=1, CASO=1, D3Cold=0, RmDisablePowerMgmt=1\n")

		case RoleDisplayHost:
			res.DisplayIndices = append(res.DisplayIndices, sub)
			fmt.Fprintf(log, "  -> Phát hiện Display Adapter (Xuất hình) tại Class Index [%s]: %s\n", sub, desc)

			_ = sk.SetDWordValue("EnableMsHybrid", 2)
			res.Details = append(res.Details, fmt.Sprintf("[%s] Cấu hình Display Host: EnableMsHybrid=2", sub))
			fmt.Fprintf(log, "     [V] Đã áp dụng: EnableMsHybrid=2 (Chỉ định làm Adapter xuất hình màn hình chính)\n")
		}

		sk.Close()
	}

	// 2. Cấu hình TDR Delay trong GraphicsDrivers
	gfxKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, GraphicsDriversKey, registry.SET_VALUE)
	if err == nil {
		_ = gfxKey.SetDWordValue("TdrDelay", 10)
		_ = gfxKey.SetDWordValue("TdrDdiDelay", 10)
		_ = gfxKey.SetDWordValue("TdrLevel", 3)
		gfxKey.Close()
		res.TdrConfigured = true
		fmt.Fprintln(log, "  [V] Đã tăng thời gian chờ TDR lên 10 giây (ngăn crash timeout khi chuyển frame qua PCIe).")
	}

	// 3. Đăng ký Khronos OpenCL & Vulkan ICD
	for _, oclPath := range []string{`SOFTWARE\Khronos\OpenCL\Vendors`, `SOFTWARE\WOW6432Node\Khronos\OpenCL\Vendors`} {
		vk, _, err := registry.CreateKey(registry.LOCAL_MACHINE, oclPath, registry.SET_VALUE)
		if err == nil {
			_ = vk.SetDWordValue("nvopencl.dll", 0)
			_ = vk.SetDWordValue("nvopencl64.dll", 0)
			vk.Close()
		}
	}
	for _, vkPath := range []string{`SOFTWARE\Khronos\Vulkan\Drivers`, `SOFTWARE\WOW6432Node\Khronos\Vulkan\Drivers`} {
		vk, _, err := registry.CreateKey(registry.LOCAL_MACHINE, vkPath, registry.SET_VALUE)
		if err == nil {
			_ = vk.SetDWordValue("nv-vk64.json", 0)
			_ = vk.SetDWordValue("nv-vk32.json", 0)
			vk.Close()
		}
	}
	res.IcdRegistered = true
	fmt.Fprintln(log, "  [V] Đã đăng ký OpenCL và Vulkan ICD cho NVIDIA driver.")

	// 4. Cấu hình DirectX Global Routing (HighPerfAdapter)
	targetDID := ""
	if topo != nil && topo.HighPerfDID != "" {
		targetDID = topo.HighPerfDID
	} else {
		switch gpuModel {
		case GPUModelCMP40HX:
			targetDID = "10DE&1F0B"
		case GPUModelCMP30HX:
			targetDID = "10DE&2189"
		}
	}

	if targetDID != "" {
		res.HighPerfDID = targetDID
		if err := ConfigureDirectXGlobalRouting(log, targetDID); err == nil {
			res.DirectXGlobalConfigured = true
		}
	}

	// 5. Đồng bộ FriendlyName trong Enum\PCI nếu cần
	targetFriendly := ""
	switch gpuModel {
	case GPUModelCMP40HX:
		targetFriendly = "NVIDIA CMP 40HX"
	case GPUModelCMP30HX:
		targetFriendly = "NVIDIA CMP 30HX"
	}

	if targetFriendly != "" {
		if pciKey, err := registry.OpenKey(registry.LOCAL_MACHINE, PciEnumKey, registry.ENUMERATE_SUB_KEYS); err == nil {
			pciSubkeys, _ := pciKey.ReadSubKeyNames(-1)
			pciKey.Close()

			for _, pSub := range pciSubkeys {
				upperSub := strings.ToUpper(pSub)
				if strings.Contains(upperSub, "VEN_10DE") &&
					(strings.Contains(upperSub, "DEV_1F0B") || strings.Contains(upperSub, "DEV_2189")) {
					instKeyPath := PciEnumKey + `\` + pSub
					if instBase, err := registry.OpenKey(registry.LOCAL_MACHINE, instKeyPath, registry.ENUMERATE_SUB_KEYS); err == nil {
						instList, _ := instBase.ReadSubKeyNames(-1)
						instBase.Close()
						for _, inst := range instList {
							finalPath := instKeyPath + `\` + inst
							if devKey, err := registry.OpenKey(registry.LOCAL_MACHINE, finalPath, registry.QUERY_VALUE|registry.SET_VALUE); err == nil {
								currFriendly, _, _ := devKey.GetStringValue("FriendlyName")
								if currFriendly == "" || strings.Contains(strings.ToUpper(currFriendly), "UNKNOWN") {
									_ = devKey.SetStringValue("FriendlyName", targetFriendly)
									res.FriendlyNamed = true
									fmt.Fprintf(log, "  [V] Đã thiết lập FriendlyName='%s' tại PCI Instance: %s\n", targetFriendly, inst)
								}
								devKey.Close()
							}
						}
					}
				}
			}
		}
	}

	return res, nil
}
