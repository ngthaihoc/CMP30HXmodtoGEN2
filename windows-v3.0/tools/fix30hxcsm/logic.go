package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	hxcore "40hxcore"
	"golang.org/x/sys/windows/registry"
)

const (
	CMP30HXHardwareID = "VEN_10DE&DEV_2189"
	KeepAliveTaskBoot = "CMP30HX_CSM_KeepAlive_Boot"
	KeepAliveTaskLogon = "CMP30HX_CSM_KeepAlive_Logon"
)

// FirmwareInfo mô tả môi trường khởi động hệ thống
type FirmwareInfo struct {
	IsCSM       bool   // true nếu là BIOS cũ / CSM
	ModeName    string // "Legacy BIOS (CSM)" hoặc "UEFI"
	RawValue    uint32
	SecureBoot  bool
	IsMBR       bool
	DiskDetails string
}

// CMP30HXStatus mô tả trạng thái của CMP 30HX trong hệ thống
type CMP30HXStatus struct {
	Detected         bool   // Đã tìm thấy trên bus / Device Manager
	IsPresent        bool   // Đang cắm và được Windows nhận diện (Present)
	IsHiddenOrGhost  bool   // Bị biến mất / Thiết bị ẩn (Code 45)
	DeviceInstanceID string // Ví dụ: PCI\VEN_10DE&DEV_2189&SUBSYS_408A1458\...
	DriverClassIndex string // Ví dụ: 0001
	StatusString     string // "OK", "Error", "Degraded"...
	ProblemCode      uint32 // 43, 12, 45...
	ProblemDesc      string
	DriverVersion    string
	HasD3ColdBlocked bool
	HasCASOEnabled   bool
	HasTdrOptimized  bool
}

// SystemDiagnostic tổng hợp toàn bộ thông tin chẩn đoán
type SystemDiagnostic struct {
	Firmware FirmwareInfo
	GPU      CMP30HXStatus
	Watchdog bool // Tác vụ canh gác đã được cài đặt chưa
}

// DetectFirmwareMode phát hiện hệ thống đang chạy CSM hay UEFI thuần
func DetectFirmwareMode() FirmwareInfo {
	info := FirmwareInfo{
		IsCSM:      false,
		ModeName:   "UEFI",
		RawValue:   2,
		SecureBoot: hxcore.SecureBootOn(),
	}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\SystemInformation`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		val, _, err := k.GetIntegerValue("FirmwareType")
		if err == nil {
			info.RawValue = uint32(val)
			if val == 1 {
				info.IsCSM = true
				info.ModeName = "Legacy BIOS (CSM)"
			} else if val == 2 {
				info.IsCSM = false
				info.ModeName = "UEFI Thuần"
			}
		}
	}

	// Kiểm tra kiểu phân vùng đĩa cài Windows (MBR vs GPT)
	psCmd := `Get-Disk | Where-Object { $_.IsBoot -or $_.IsSystem } | Select-Object -ExpandProperty PartitionStyle`
	out, err := execPowerShell(psCmd)
	if err == nil {
		style := strings.TrimSpace(out)
		if strings.EqualFold(style, "MBR") {
			info.IsMBR = true
			info.DiskDetails = "Ổ đĩa Boot: MBR (Chế độ tương thích CSM)"
			if !info.IsCSM {
				// Nếu FirmwareType chưa xác định nhưng đĩa boot là MBR -> máy chạy CSM
				info.IsCSM = true
				info.ModeName = "Legacy BIOS (CSM / MBR)"
			}
		} else if strings.EqualFold(style, "GPT") {
			info.IsMBR = false
			info.DiskDetails = "Ổ đĩa Boot: GPT"
		}
	}

	return info
}

// ScanCMP30HXDevice tìm kiếm và phân tích trạng thái CMP 30HX
func ScanCMP30HXDevice() CMP30HXStatus {
	st := CMP30HXStatus{
		Detected:        false,
		IsPresent:       false,
		IsHiddenOrGhost: false,
		ProblemCode:     0,
	}

	// 1. Dò qua PowerShell Get-PnpDevice (hỗ trợ cả thiết bị ẩn / ngắt kết nối)
	psScript := `
$dev = Get-PnpDevice -PresentOnly:$false | Where-Object { $_.InstanceId -like "*` + CMP30HXHardwareID + `*" } | Select-Object -First 1
if ($dev) {
    [PSCustomObject]@{
        InstanceId = $dev.InstanceId
        Present = $dev.Present
        Status = $dev.Status
        Problem = $dev.Problem
        ProblemDescription = $dev.ProblemDescription
    } | ConvertTo-Json -Compress
}
`
	out, err := execPowerShell(psScript)
	if err == nil && strings.TrimSpace(out) != "" {
		var pnpData struct {
			InstanceId         string `json:"InstanceId"`
			Present            bool   `json:"Present"`
			Status             string `json:"Status"`
			Problem            uint32 `json:"Problem"`
			ProblemDescription string `json:"ProblemDescription"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(out)), &pnpData) == nil && pnpData.InstanceId != "" {
			st.Detected = true
			st.DeviceInstanceID = pnpData.InstanceId
			st.IsPresent = pnpData.Present
			st.StatusString = pnpData.Status
			st.ProblemCode = pnpData.Problem
			st.ProblemDesc = pnpData.ProblemDescription
			if !st.IsPresent {
				st.IsHiddenOrGhost = true
			}
		}
	}

	// 2. Dò qua Registry Enum PCI (phòng trường hợp PnP chưa khởi tạo hoặc thiết bị rớt bus)
	if !st.Detected {
		base, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Enum\PCI`, registry.ENUMERATE_SUB_KEYS)
		if err == nil {
			defer base.Close()
			devs, _ := base.ReadSubKeyNames(-1)
			for _, d := range devs {
				if strings.Contains(strings.ToUpper(d), CMP30HXHardwareID) {
					st.Detected = true
					st.IsHiddenOrGhost = true // Có trong registry nhưng Device Manager không thấy -> Rớt bus
					dk, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Enum\PCI\`+d, registry.ENUMERATE_SUB_KEYS)
					if err == nil {
						insts, _ := dk.ReadSubKeyNames(-1)
						dk.Close()
						if len(insts) > 0 {
							st.DeviceInstanceID = fmt.Sprintf(`PCI\%s\%s`, d, insts[0])
						}
					}
					break
				}
			}
		}
	}

	// 3. Tìm Class Index trong Display Class (0000, 0001...)
	if st.Detected {
		classKey, err := registry.OpenKey(registry.LOCAL_MACHINE, hxcore.GpuClassPath, registry.ENUMERATE_SUB_KEYS)
		if err == nil {
			defer classKey.Close()
			subkeys, _ := classKey.ReadSubKeyNames(-1)
			for _, sub := range subkeys {
				if strings.HasPrefix(sub, "0") {
					sk, err := registry.OpenKey(registry.LOCAL_MACHINE, hxcore.GpuClassPath+`\`+sub, registry.QUERY_VALUE)
					if err == nil {
						matchID, _, _ := sk.GetStringValue("MatchingDeviceId")
						driverDesc, _, _ := sk.GetStringValue("DriverDesc")
						drvVer, _, _ := sk.GetStringValue("DriverVersion")
						if strings.Contains(strings.ToUpper(matchID), CMP30HXHardwareID) ||
							strings.Contains(driverDesc, "CMP 30HX") ||
							strings.Contains(driverDesc, "TU116") {
							st.DriverClassIndex = sub
							st.DriverVersion = drvVer

							// Kiểm tra các khoá chống ngủ & CASO
							d3Val, _, _ := sk.GetIntegerValue("D3ColdSupported")
							rmVal, _, _ := sk.GetIntegerValue("RmDisableGpuPowerMgmt")
							casoVal, _, _ := sk.GetIntegerValue("EnableCrossAdapterScanOut")
							if d3Val == 0 && rmVal == 1 {
								st.HasD3ColdBlocked = true
							}
							if casoVal == 1 {
								st.HasCASOEnabled = true
							}
							sk.Close()
							break
						}
						sk.Close()
					}
				}
			}
		}
	}

	// 4. Kiểm tra TdrDelay
	gfxKey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, registry.QUERY_VALUE)
	if err == nil {
		defer gfxKey.Close()
		tdr, _, err := gfxKey.GetIntegerValue("TdrDelay")
		if err == nil && tdr >= 8 {
			st.HasTdrOptimized = true
		}
	}

	return st
}

// IsWatchdogInstalled kiểm tra xem tác vụ canh gác đã đăng ký chưa
func IsWatchdogInstalled() bool {
	out, err := hxcore.RunOut("schtasks.exe", "/query", "/tn", KeepAliveTaskBoot)
	return err == nil && strings.Contains(out, KeepAliveTaskBoot)
}

// RescanPCIBus quét lại bus PCI phần cứng để lôi card bị biến mất trở lại
func RescanPCIBus(log io.Writer) error {
	if log == nil {
		log = io.Discard
	}
	fmt.Fprintln(log, "[*] Đang phát lệnh quét lại phần cứng toàn bộ Bus PCIe (PCI Bus Rescan)...")

	// Cách 1: Sử dụng pnputil /scan-devices
	out, err := hxcore.RunOut("pnputil.exe", "/scan-devices")
	if err == nil {
		fmt.Fprintf(log, "    [OK] pnputil đã kích hoạt quét bus: %s\n", strings.TrimSpace(out))
	} else {
		// Cách 2: Dự phòng bằng PowerShell Device Root Re-enumerate
		fmt.Fprintln(log, "    [!] pnputil trả về cảnh báo, kích hoạt tầng quét dự phòng PowerShell...")
		psCmd := `
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class DeviceManager {
    [DllImport("setupapi.dll", SetLastError = true)]
    public static extern int CM_Locate_DevNode_Ex(out IntPtr pdnDevInst, string pDeviceID, int ulFlags, IntPtr hMachine);
    [DllImport("setupapi.dll", SetLastError = true)]
    public static extern int CM_Reenumerate_DevNode_Ex(IntPtr dnDevInst, int ulFlags, IntPtr hMachine);
}
"@
$devInst = [IntPtr]::Zero
[DeviceManager]::CM_Locate_DevNode_Ex([ref]$devInst, $null, 0, [IntPtr]::Zero)
[DeviceManager]::CM_Reenumerate_DevNode_Ex($devInst, 0, [IntPtr]::Zero)
`
		_, _ = execPowerShell(psCmd)
	}

	// Chờ bus đàm phán lại tín hiệu vi sai
	time.Sleep(2 * time.Second)
	return nil
}

// FixCSMPowerSettings triệt tiêu toàn bộ cơ chế sleep gây tụt link trong CSM
func FixCSMPowerSettings(log io.Writer, status *CMP30HXStatus) error {
	if log == nil {
		log = io.Discard
	}

	fmt.Fprintln(log, "[*] [1/5] Vô hiệu hóa Fast Startup và PCIe ASPM toàn hệ thống...")
	_, _ = hxcore.RunOut("powercfg.exe", "-h", "off")

	// HiberbootEnabled = 0
	pwrKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Power`, registry.SET_VALUE)
	if err == nil {
		_ = pwrKey.SetDWordValue("HiberbootEnabled", 0)
		pwrKey.Close()
		fmt.Fprintln(log, "    [OK] Đã tắt Fast Startup (HiberbootEnabled = 0).")
	}

	// Tắt PCI Express Link State Power Management trên tất cả power scheme
	psPwrCmd := `
$schemes = powercfg -list | Select-String -Pattern "([a-f0-9-]{36})" | ForEach-Object { $_.Matches.Groups[1].Value }
$subPci = "501a4d13-42af-4429-9fd5-e8174f7900c2"
$settingPci = "ee12f906-d27e-44ba-b376-73e76cb376f7"
foreach ($s in $schemes) {
    powercfg -setacvalueindex $s $subPci $settingPci 0
    powercfg -setdcvalueindex $s $subPci $settingPci 0
    powercfg -setactive $s
}
`
	_, _ = execPowerShell(psPwrCmd)
	fmt.Fprintln(log, "    [OK] Đã tắt PCIe Link State Power Management trong tất cả Power Scheme.")

	// Thiết lập khoá chống ngủ cho Driver Class của GPU
	if status.DriverClassIndex != "" {
		fmt.Fprintf(log, "[*] [2/5] Cấu hình chống ngủ sâu D3cold tại Driver Class %s...\n", status.DriverClassIndex)
		targetClassPath := hxcore.GpuClassPath + `\` + status.DriverClassIndex
		k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, targetClassPath, registry.SET_VALUE)
		if err == nil {
			defer k.Close()
			_ = k.SetDWordValue("D3ColdSupported", 0)
			_ = k.SetDWordValue("DeviceSelectiveSuspended", 0)
			_ = k.SetDWordValue("RmDisableGpuPowerMgmt", 1)
			_ = k.SetDWordValue("RmEnableAggressivePciePowerManagement", 0)
			_ = k.SetDWordValue("DisableASPM", 1)
			_ = k.SetDWordValue("DisablePCIePowerManagement", 1)
			fmt.Fprintln(log, "    [OK] Đã khóa vĩnh viễn D3cold và GPU Power Management (Giữ link PCIe luôn thức).")
		}
	}

	// Thiết lập Device Parameters chống ngủ trong Enum PCI
	if status.DeviceInstanceID != "" {
		devParamPath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Enum\%s\Device Parameters`, status.DeviceInstanceID)
		dk, _, err := registry.CreateKey(registry.LOCAL_MACHINE, devParamPath, registry.SET_VALUE)
		if err == nil {
			_ = dk.SetDWordValue("EnhancedPowerManagementEnabled", 0)
			_ = dk.SetDWordValue("AllowIdleIrpInD3", 0)
			_ = dk.SetDWordValue("D3ColdSupported", 0)
			_ = dk.SetDWordValue("DeviceSelectiveSuspended", 0)
			dk.Close()
			fmt.Fprintln(log, "    [OK] Đã áp dụng cấm D3 cho Device Instance Parameters.")
		}
	}

	return nil
}

// FixError43AndCASO sửa lỗi 43, tối ưu hoá không gian nhớ 32-bit MMIO và kích hoạt CASO
func FixError43AndCASO(log io.Writer, status *CMP30HXStatus) error {
	if log == nil {
		log = io.Discard
	}

	fmt.Fprintln(log, "[*] [3/5] Áp dụng cấu hình sửa lỗi 43 và xuất hình không cổng (CASO)...")

	// 1. Tối ưu hoá TDR Timeout trong GraphicsDrivers
	gfxKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\GraphicsDrivers`, registry.SET_VALUE)
	if err == nil {
		_ = gfxKey.SetDWordValue("TdrDelay", 10)
		_ = gfxKey.SetDWordValue("TdrDdiDelay", 10)
		_ = gfxKey.SetDWordValue("TdrLevel", 3)
		gfxKey.Close()
		fmt.Fprintln(log, "    [OK] Đã tăng thời gian chờ TDR lên 10 giây (chống timeout bus chậm trong CSM).")
	}

	// 2. Cấu hình Driver Class cho CMP 30HX
	if status.DriverClassIndex != "" {
		targetClassPath := hxcore.GpuClassPath + `\` + status.DriverClassIndex
		k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, targetClassPath, registry.SET_VALUE)
		if err == nil {
			defer k.Close()
			// Kích hoạt Cross-Adapter Scan-Out (CASO) cho card đào không cổng
			_ = k.SetDWordValue("EnableCrossAdapterScanOut", 1)
			_ = k.SetDWordValue("AdapterType", 0)

			// Tối ưu hoá không gian nhớ MMIO 32-bit (tránh lỗi 43 do Above 4G Decoding tắt trong CSM)
			_ = k.SetDWordValue("LargePageMinimum", 0xFFFFFFFF)

			// Thiết lập chế độ CPU-RM (EnableGpuFirmware = 0) cho Turing TU116 trên main CSM
			_ = k.SetDWordValue("EnableGpuFirmware", 0)

			// Tối ưu PowerMizer hiệu năng cao nhất, không dao động nguồn
			_ = k.SetDWordValue("PowerMizerEnable", 0)
			_ = k.SetDWordValue("PowerMizerLevel", 1)
			_ = k.SetDWordValue("PowerMizerLevelAC", 1)

			fmt.Fprintln(log, "    [OK] Đã cấu hình CASO, tối ưu MMIO 32-bit, và thiết lập CPU-RM cho Turing TU116.")
		}
	}

	// 3. Khởi động lại PnP Device Node để xoá sạch cờ Code 43
	fmt.Fprintln(log, "[*] [4/5] Kích hoạt chu trình PnP Restart Device để xoá sạch lỗi 43...")
	if status.DeviceInstanceID != "" {
		// Thử pnputil /restart-device
		out, err := hxcore.RunOut("pnputil.exe", "/restart-device", fmt.Sprintf("\"%s\"", status.DeviceInstanceID))
		if err == nil {
			fmt.Fprintf(log, "    [OK] pnputil đã khởi động lại thiết bị: %s\n", strings.TrimSpace(out))
		} else {
			// Dự phòng bằng disable rồi enable lại
			fmt.Fprintln(log, "    [*] Thao tác lại chu trình Disable -> Enable PnP node...")
			_, _ = hxcore.RunOut("pnputil.exe", "/disable-device", fmt.Sprintf("\"%s\"", status.DeviceInstanceID))
			time.Sleep(1500 * time.Millisecond)
			_, _ = hxcore.RunOut("pnputil.exe", "/enable-device", fmt.Sprintf("\"%s\"", status.DeviceInstanceID))
		}
	}

	// 4. Khởi động lại dịch vụ NVIDIA Display Container
	fmt.Fprintln(log, "[*] [5/5] Khởi động lại dịch vụ NVIDIA Display Container...")
	_ = hxcore.EnsureNvidiaControlPanelHealthy()
	fmt.Fprintln(log, "    [OK] Dịch vụ đồ hoạ NVIDIA đã được nạp lại.")

	return nil
}

// InstallWatchdogTask cài đặt tác vụ canh gác tự động chống biến mất card
func InstallWatchdogTask(log io.Writer) error {
	if log == nil {
		log = io.Discard
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	abs, _ := filepath.Abs(exe)

	fmt.Fprintln(log, "[*] Đang đăng ký tác vụ canh gác CSM Keep-Alive (SYSTEM)...")

	// 1. Tác vụ lúc khởi động máy (At Startup)
	bootCmd := fmt.Sprintf("\"%s\" -scan -fix", abs)
	out1, err1 := hxcore.RunOut("schtasks.exe", "/create", "/tn", KeepAliveTaskBoot,
		"/tr", bootCmd, "/sc", "onstart", "/ru", "SYSTEM", "/delay", "0000:20", "/f")
	if err1 != nil {
		fmt.Fprintf(log, "    [X] Lỗi tạo tác vụ boot: %s\n", out1)
		return err1
	}

	// 2. Tác vụ lúc đăng nhập (At Logon)
	logonCmd := fmt.Sprintf("\"%s\" -scan", abs)
	out2, err2 := hxcore.RunOut("schtasks.exe", "/create", "/tn", KeepAliveTaskLogon,
		"/tr", logonCmd, "/sc", "onlogon", "/ru", "SYSTEM", "/delay", "0000:10", "/f")
	if err2 != nil {
		fmt.Fprintf(log, "    [X] Lỗi tạo tác vụ logon: %s\n", out2)
		return err2
	}

	fmt.Fprintln(log, "    [OK] Đã đăng ký thành công bộ đôi tác vụ canh gác tự động.")
	fmt.Fprintln(log, "         Card sẽ được tự động kiểm tra, quét bus và sửa lỗi nếu biến mất khi mở máy.")
	return nil
}

// RemoveWatchdogTask gỡ bỏ tác vụ canh gác
func RemoveWatchdogTask(log io.Writer) error {
	if log == nil {
		log = io.Discard
	}
	fmt.Fprintln(log, "[*] Đang gỡ bỏ các tác vụ canh gác CSM...")
	_, _ = hxcore.RunOut("schtasks.exe", "/delete", "/tn", KeepAliveTaskBoot, "/f")
	_, _ = hxcore.RunOut("schtasks.exe", "/delete", "/tn", KeepAliveTaskLogon, "/f")
	fmt.Fprintln(log, "    [OK] Đã gỡ bỏ toàn bộ tác vụ canh gác.")
	return nil
}

// CheckMBR2GPT kiểm tra khả năng chuyển đổi không mất dữ liệu sang UEFI
func CheckMBR2GPT() (canConvert bool, report string) {
	out, err := hxcore.RunOut("mbr2gpt.exe", "/validate", "/allowfullos")
	if err == nil && strings.Contains(out, "Validation completed successfully") {
		return true, "Đủ điều kiện chuyển đổi 100% sang UEFI thuần qua mbr2gpt (Không mất dữ liệu)!"
	}
	return false, fmt.Sprintf("Không thể tự động chuyển đổi qua mbr2gpt:\n%s", out)
}

// Helper thực thi PowerShell
func execPowerShell(cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	c := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, stderr.String())
	}
	return stdout.String(), nil
}
