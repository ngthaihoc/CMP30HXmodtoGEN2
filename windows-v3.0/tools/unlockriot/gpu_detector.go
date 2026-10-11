package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	DisplayDeviceAttachedToDesktop = 0x00000001
	DisplayDevicePrimaryDevice     = 0x00000004
)

type displayDeviceW struct {
	cb           uint32
	deviceName   [32]uint16
	deviceString [128]uint16
	stateFlags   uint32
	deviceID     [128]uint16
	deviceKey    [128]uint16
}

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumDisplayDevicesW = user32.NewProc("EnumDisplayDevicesW")

	// enumDisplayDevicesHook cho phép mock dữ liệu trong unit test
	enumDisplayDevicesHook func() []ActiveDisplayTarget
)

// ActiveDisplayTarget chứa thông tin thiết bị xuất hình màn hình desktop
type ActiveDisplayTarget struct {
	DeviceID   string
	StateFlags uint32
	IsAttached bool
}

// GetActiveDisplayTargets quét danh sách các cổng/thiết bị hiển thị đang kết nối màn hình
func GetActiveDisplayTargets() []ActiveDisplayTarget {
	if enumDisplayDevicesHook != nil {
		return enumDisplayDevicesHook()
	}

	var targets []ActiveDisplayTarget
	var dev displayDeviceW
	dev.cb = uint32(unsafe.Sizeof(dev))

	for i := uintptr(0); ; i++ {
		r, _, _ := procEnumDisplayDevicesW.Call(
			0,
			i,
			uintptr(unsafe.Pointer(&dev)),
			0,
		)
		if r == 0 {
			break
		}

		id := syscall.UTF16ToString(dev.deviceID[:])
		attached := (dev.stateFlags & DisplayDeviceAttachedToDesktop) != 0

		targets = append(targets, ActiveDisplayTarget{
			DeviceID:   id,
			StateFlags: dev.stateFlags,
			IsAttached: attached,
		})
	}

	return targets
}

// ExtractShortDID trích xuất chuỗi định danh Vendor & Device cho DirectX (vd: "10DE&1F0B")
func ExtractShortDID(hardwareID string) string {
	upper := strings.ToUpper(hardwareID)
	venIdx := strings.Index(upper, "VEN_")
	devIdx := strings.Index(upper, "DEV_")

	if venIdx != -1 && devIdx != -1 {
		ven := ""
		if len(upper) >= venIdx+8 {
			ven = upper[venIdx+4 : venIdx+8]
		}
		dev := ""
		if len(upper) >= devIdx+8 {
			dev = upper[devIdx+4 : devIdx+8]
		}
		if ven != "" && dev != "" {
			return fmt.Sprintf("%s&%s", ven, dev)
		}
	}

	// Fallback phân tách theo dấu &
	tokens := strings.Split(upper, "&")
	var v, d string
	for _, tok := range tokens {
		if strings.HasPrefix(tok, "VEN_") {
			v = strings.TrimPrefix(tok, "VEN_")
		} else if strings.HasPrefix(tok, "DEV_") {
			d = strings.TrimPrefix(tok, "DEV_")
		}
	}
	if v != "" && d != "" {
		return fmt.Sprintf("%s&%s", v, d)
	}

	return ""
}

// GPUAdapterInfo chi tiết một card đồ họa tìm thấy trong hệ thống
type GPUAdapterInfo struct {
	ClassIndex string
	MatchingID string
	Desc       string
	Provider   string
	Role       AdapterRole
	IsAttached bool
	DID        string
}

// GPUTopology cấu trúc phân định GPU xuất hình vs GPU render 3D
type GPUTopology struct {
	DisplayHosts []GPUAdapterInfo
	RenderGPUs   []GPUAdapterInfo
	HighPerfDID  string
}

// ScanGPUTopology quét và phân loại GPU dựa trên Registry và trạng thái màn hình
func ScanGPUTopology() (*GPUTopology, error) {
	topo := &GPUTopology{}
	targets := GetActiveDisplayTargets()

	baseKey, err := registry.OpenKey(registry.LOCAL_MACHINE, GpuClassPath, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return topo, fmt.Errorf("không thể mở Driver Class Registry: %w", err)
	}
	defer baseKey.Close()

	subkeys, err := baseKey.ReadSubKeyNames(-1)
	if err != nil {
		return topo, fmt.Errorf("không thể đọc subkeys: %w", err)
	}

	for _, sub := range subkeys {
		if len(sub) != 4 {
			continue
		}

		subPath := GpuClassPath + `\` + sub
		sk, err := registry.OpenKey(registry.LOCAL_MACHINE, subPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		matchID, _, _ := sk.GetStringValue("MatchingDeviceId")
		desc, _, _ := sk.GetStringValue("DriverDesc")
		provider, _, _ := sk.GetStringValue("ProviderName")
		sk.Close()

		role := ClassifyAdapter(matchID, desc, provider)
		if role == RoleUnknown {
			continue
		}

		shortDID := ExtractShortDID(matchID)

		// Kiểm tra xem GPU này có cổng màn hình đang cắm hay không
		isAttached := false
		upperMatch := strings.ToUpper(matchID)
		for _, tgt := range targets {
			if tgt.IsAttached && tgt.DeviceID != "" {
				upperTgt := strings.ToUpper(tgt.DeviceID)
				if shortDID != "" && strings.Contains(upperTgt, shortDID) {
					isAttached = true
					break
				}
				if strings.Contains(upperTgt, upperMatch) || strings.Contains(upperMatch, upperTgt) {
					isAttached = true
					break
				}
			}
		}

		adapter := GPUAdapterInfo{
			ClassIndex: sub,
			MatchingID: matchID,
			Desc:       desc,
			Provider:   provider,
			Role:       role,
			IsAttached: isAttached,
			DID:        shortDID,
		}

		// Nếu GPU xuất hình (iGPU hoặc card có màn hình gắn)
		if isAttached || role == RoleDisplayHost {
			adapter.Role = RoleDisplayHost
			topo.DisplayHosts = append(topo.DisplayHosts, adapter)
		} else if role == RoleNvidiaRender {
			// Card CMP / NVIDIA render không cắm màn hình
			topo.RenderGPUs = append(topo.RenderGPUs, adapter)
			if topo.HighPerfDID == "" && shortDID != "" {
				topo.HighPerfDID = shortDID
			}
		}
	}

	// Nếu tìm thấy Render dGPU nhưng HighPerfDID chưa có
	if topo.HighPerfDID == "" && len(topo.RenderGPUs) > 0 {
		topo.HighPerfDID = topo.RenderGPUs[0].DID
	}

	return topo, nil
}
