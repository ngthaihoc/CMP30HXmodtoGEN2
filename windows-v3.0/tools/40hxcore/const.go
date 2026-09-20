// 40hxcore — CMP 40HX/30HX 解锁工具链共享核心(只读探测 + 寄存器访问)
//
// 由 tools/inst40hx(main) 与 tools/check40x(40HXCheck.exe) 共同引用,
// 保证"诊断逻辑只有一份实现, 不会两处漂移"。常量与源码与
// inst40hx/main.go 同步维护 — 改动任一侧需同步另一侧。
package hxcore

const (
	// 40HX PCI 硬件 ID (兼容旧常量)
	GpuVenDev = "VEN_10DE&DEV_1F0B"

	GpuVenDev30HX = "VEN_10DE&DEV_2189"

	// 显示适配器 Class 注册表路径与子键值
	GpuClassPath  = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	GpuClassGUID  = `{4d36e968-e325-11ce-bfc1-08002be10318}` // Enum Driver 值反查用
	GpuEnableFw   = "EnableGpuFirmware"                      // GSP 启用开关 (=1)
	GpuAdapterStr = "HardwareInformation.AdapterString"      // 驱动真实卡名(伪装也改不了此路径)
	GpuAdapter40  = "CMP 40HX"
	GpuAdapter30  = "CMP 30HX"

	// PCI 设备枚举根
	GpuEnumBase = `SYSTEM\CurrentControlSet\Enum\PCI`
)

type GPUProfile struct {
	VendorID        uint16
	DeviceID        uint16
	HardwareID      string
	Name            string
	Family          string
	MaxSupportedGen uint32
	FirmwareUnlock  bool
	HasSafePL0      bool
}

var SupportedGPUProfiles = []GPUProfile{
	{
		VendorID:        0x10DE,
		DeviceID:        0x1F0B,
		HardwareID:      GpuVenDev,
		Name:            GpuAdapter40,
		Family:          "TU106",
		MaxSupportedGen: 2,
		FirmwareUnlock:  true,
		HasSafePL0:      true,
	},
	{
		VendorID:        0x10DE,
		DeviceID:        0x2189,
		HardwareID:      GpuVenDev30HX,
		Name:            GpuAdapter30,
		Family:          "TU116",
		MaxSupportedGen: 2, // ponytail: CMP 30HX (TU116) bi dut eFuse bit 3 (8.0 GT/s), gioi han phan cung la Gen2 (5.0 GT/s)
		FirmwareUnlock:  false,
		HasSafePL0:      false,
	},
}

func LookupGPUProfile(ven, dev uint16) (GPUProfile, bool) {
	for _, p := range SupportedGPUProfiles {
		if p.VendorID == ven && p.DeviceID == dev {
			return p, true
		}
	}
	return GPUProfile{}, false
}
