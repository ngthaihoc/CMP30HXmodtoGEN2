<p align="right">
  <b>Ngôn ngữ:</b>
  <b>Tiếng Việt</b> |
  <a href="README_EN.md">English</a> |
  <a href="README_ZH.md">简体中文</a>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Mở khoá NVIDIA CMP 30HX v3.0.0 (PCIe Gen2 x16)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-Không%20cần%20thiết-success)](#)

Giải pháp mở khoá băng thông **PCIe Gen2 x16 (~6.4 GB/s)** cho card đồ hoạ **NVIDIA CMP 30HX (nhân TU116)** trên Windows 10/11 x64 và Linux.

- **Cài đặt 1-chạm** — script tự xử lý mọi thứ, bạn chỉ cần nhấp đúp và chờ.
- **Tương thích mọi driver** — NVIDIA chính thức, desktop, mod, phiên bản mới nhất.
- **An toàn với anti-cheat** — không cần Test Signing, tương thích Riot Vanguard, EAC, BattlEye.
- **Không chỉnh sửa BIOS** — can thiệp qua phần mềm, không sửa BIOS/VBIOS.

> [!TIP]
> **Ủng hộ tác giả (Donate)**
>
> Dự án này do em phát triển lúc còn là sinh viên. Nếu được hãy ủng hộ cho em một chút nhé! Cảm ơn mọi người rất nhiều! ❤️
>
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
>
> - **Chủ tài khoản**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub Repository**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. Chuẩn bị

**Bạn cần có:**

- Card **CMP 30HX (TU116)** đã mod hàn trở lane PCIe x16, cắm vào khe PCIe x16 nối trực tiếp CPU.
- Driver NVIDIA bất kỳ đã cài sẵn (card hiển thị bình thường trong Device Manager).

**Tải bộ công cụ:**

Bấm **Code → Download ZIP** trên GitHub, giải nén ra thư mục cố định (ví dụ: `D:\CMP30HX-Unlock`).

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. Cài đặt trên Windows

### ⭐ Cách nhanh nhất: giao diện web (khuyên dùng)

> **Chỉ 1 bước** — nhấp đúp `Launch_WebUI.bat` rồi làm theo giao diện.

Trình duyệt sẽ mở trang điều khiển với đầy đủ thông tin GPU, trạng thái PCIe link và nút bấm 1-chạm để mở khoá Gen2.

### Cách 2: Script tự động 1-chạm

> **Chỉ 1 bước** — nhấp chuột phải `Setup_CMP30HX_WindowsAIO.bat` → **Run as administrator**.

Script sẽ tự động xử lý mọi thứ: tắt ASPM, tắt Fast Startup, cấu hình registry, mở khoá Gen2 x16, nâng MRRS lên 512B và đăng ký tự kích hoạt mỗi lần bật máy.

> [!IMPORTANT]
> Nếu máy bạn đang **bật Memory Integrity (Core Isolation)**, script sẽ tắt nó và yêu cầu bạn **reboot 1 lần**. Sau khi đăng nhập lại, hệ thống sẽ tự động chuyển sang Gen2 x16.

<details>
<summary>📋 Chi tiết: script làm những gì?</summary>

1. **Tắt PCIe ASPM và Hybrid Sleep** — ngăn Windows hạ tốc độ link PCIe khi GPU idle.
2. **Tắt Fast Startup (hiberboot)** — chống kẹt link Gen1 sau khi reboot.
3. **Tắt Microsoft Vulnerable Driver Blocklist** — cho phép nạp driver WinRing0 / ThrottleStop.
4. **Tắt Memory Integrity (HVCI)** — cho phép ghi đè thanh ghi BAR0 MMIO.
5. **Đăng ký tự kích hoạt** — tạo Scheduled Task (startup + logon + wake from sleep) và Registry Run Key, đảm bảo Gen2 luôn được duy trì.
6. **Mở khoá Gen2 x16 + MRRS 512B** — card sẵn sàng ngay, không cần reboot (trừ trường hợp HVCI).

</details>

<details>
<summary>🔧 Cách 3: Cài đặt thủ công bằng dòng lệnh (dành cho chuyên gia)</summary>

Mở **PowerShell** hoặc **Command Prompt** với quyền administrator:

**Bước 1 — Tắt PCIe ASPM và Hybrid Sleep:**
```cmd
powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
powercfg -setacvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
powercfg -setdcvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
powercfg -setactive SCHEME_CURRENT
```

**Bước 2 — Tắt Fast Startup và Vulnerable Driver Blocklist:**
```cmd
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f
```

**Bước 3 — Tắt Memory Integrity (nếu đang bật):**
```cmd
reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f
```
*(Reboot nếu vừa thay đổi giá trị này từ 1 thành 0.)*

**Bước 4 — Mở khoá Gen2 x16:**
```cmd
cd /d "Đường_dẫn_thư_mục_giải_nén"
.\windows-v3.0\release\40HXInstaller.exe -gen2-30hx
```

**Bước 5 — Đăng ký tự kích hoạt khi bật máy:**
```cmd
if not exist "%ProgramFiles%\40HXUnlock" mkdir "%ProgramFiles%\40HXUnlock"
copy /y ".\windows-v3.0\release\40HXInstaller.exe" "%ProgramFiles%\40HXUnlock\"
copy /y ".\windows-v3.0\release\40HXCheck.exe" "%ProgramFiles%\40HXUnlock\"
```

```powershell
$action = New-ScheduledTaskAction -Execute "$env:ProgramFiles\40HXUnlock\40HXInstaller.exe" -Argument '-gen2-30hx -silent' -WorkingDirectory "$env:ProgramFiles\40HXUnlock"
$t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'
$t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew
$principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force
```

```cmd
reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
```

</details>

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 3. Cài đặt trên Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

Script tự động xử lý: tắt ASPM, quét tất cả card CMP 30HX/40HX, ghi thanh ghi BAR0 MMIO, nâng MRRS 512B, retrain link Gen2, và cài systemd service để duy trì sau mỗi lần reboot.

| Lệnh | Chức năng |
|---|---|
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --status` | Kiểm tra trạng thái link và MRRS |
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` | Gỡ bỏ service và hook |
| `./Setup_CMP30HX_LinuxAIO.sh -test --no-root` | Kiểm thử giả lập không cần GPU |

<details>
<summary>📋 Chi tiết kỹ thuật script Linux</summary>

1. **Tắt PCIe ASPM và Runtime Power Management** — đặt policy kernel sang `performance`, `power/control=on` cho toàn bộ thiết bị PCI.
2. **Quét toàn bộ card** — tự phát hiện CMP 30HX (`10de:2189`) và CMP 40HX (`10de:1f0b`).
3. **BAR0 MMIO direct injection** — can thiệp thanh ghi `XVE_OVR`, `PRIV_MISC_1`, `LINK_CONFIG_0`, `LNKCAP`, `LNKCTL2`.
4. **MRRS 512B + retrain link** — Target Link Speed = Gen2, MRRS 512 Bytes, retrain đạt ~6.4 GB/s.
5. **Systemd service + sleep hook** — `cmp30hx-gen2-unlock.service` và `/lib/systemd/system-sleep/cmp30hx-unlock`.
6. **StatusContract** — xuất `gen2_status.txt` (hoặc qua cờ `--status-file`):
   ```text
   STATUS_CODE=GEN2_SUCCESS
   SPEED_CURRENT=2
   WIDTH_CURRENT=16
   TLS_TARGET=2
   ERROR_CODE=NONE
   TIMESTAMP=2026-09-26T05:30:00Z
   ```

</details>

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 4. Kiểm tra kết quả

Sau khi cài đặt xong, bạn kiểm tra bằng 1 trong 2 cách:

| Công cụ | Cách kiểm tra | Kết quả đúng |
|---|---|---|
| **GPU-Z** hoặc `40HXCheck.exe` | Mục **Bus Interface** | `PCIe x16 2.0 @ x16 2.0` |
| **AIDA64** → Tools → GPGPU Benchmark | Dòng **Memory Read / Memory Copy** | **6.3 – 6.4 GB/s** |

> Nếu AIDA64 chỉ đạt ~2.5 GB/s → MRRS đang ở 128B mặc định → chạy lại `40HXInstaller.exe -gen2-30hx`.

---

## <img src="https://api.iconify.design/lucide/shield-check.svg?color=%2306b6d4" width="22" height="22" align="center" /> 5. Tương thích anti-cheat (Riot Vanguard, EAC, BattlEye)

Bạn hoàn toàn có thể chơi **Valorant, League of Legends, Apex Legends, Fortnite** sau khi mở khoá:

- **CMP 30HX** — mở khoá Gen2 qua BAR0 MMIO trong Windows (không nạp EFI), giữ nguyên Secure Boot bật → tương thích 100% với Riot Vanguard.
- **CMP 40HX** — cần chạy thêm `UnlockRiotGame.exe` để ký số EFI firmware và nạp chứng chỉ vào Secure Boot db.
- **Cơ chế dùng-xong-rút** — driver kernel chỉ nạp vài mili-giây rồi tự xoá sạch. Khi game chạy, hệ thống hoàn toàn sạch.
- **Không cần `bcdedit /set testsigning on`** — lệnh này bị Vanguard chặn 100% và công cụ không yêu cầu nó.

<details>
<summary>📋 Chi tiết: giải pháp Riot Vanguard cho CMP 40HX (TU106)</summary>

Riot Vanguard trên Windows 11 yêu cầu Secure Boot = Enabled và TPM 2.0. Firmware `40HXUNLK.EFI` chưa có chữ ký Microsoft, nên `UnlockRiotGame.exe` sẽ tự động:

1. Tạo chứng chỉ X.509 (`CMP40HX_Key.cer`) với SHA256.
2. Ký số Authenticode cho `40HXUNLK.EFI` (thư mục phát hành và phân vùng ESP).
3. Xuất chứng chỉ ra `C:\`, Desktop và ESP (`\EFI\40HX\`).
4. Hướng dẫn bạn reboot vào BIOS → **Custom Mode** → nạp `CMP40HX_Key.cer` vào **db** (Key Management → Authorized Signatures → Append Key).
5. Kết quả: Secure Boot vẫn bật (Vanguard hài lòng), đồng thời firmware EFI mở khoá Tensor Core (~50 TFLOPS) và PCIe Gen2.

</details>

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 6. Xử Lý Sự Cố Thường Gặp (Troubleshooting)

| Hiện tượng | Nguyên nhân | Hướng giải quyết |
|---|---|---|
| **Kẹt ở Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | - Hệ thống GPU kép (CMP 30HX + iGPU AMD/Intel) hoặc driver mod (RainCandy) khởi tạo lại, tự động reset GPU TLS về Gen1.<br>- Hoặc Windows bật Memory Integrity (HVCI), hoặc cáp Riser tiếp xúc kém | 1. **Tự động**: `Setup_CMP30HX_WindowsAIO.bat` bản mới kích hoạt chế độ **Thường trú (Resident Guard)** và tác vụ Logon (delay 10s) tự bù tốc Gen2.<br>2. **Thủ công**: Giữ card ở trạng thái **Enable** trong Device Manager (tuyệt đối **KHÔNG** Disable vì sẽ ngắt kết nối BAR0 MMIO) → Chạy `Setup_CMP30HX_WindowsAIO.bat` hoặc `40HXInstaller.exe -gen2-30hx` khi desktop đã ổn định.<br>3. Nếu HVCI đang bật: Tắt HVCI và **Khởi động lại máy tính (Reboot)**. |
| **Mất kích hoạt Gen2 sau khi khởi động lại máy** | - Driver NVIDIA/RainCandy nạp trễ trên hệ thống APU hoặc ghi đè trạng thái vBIOS sau khi khởi động.<br>- Xung đột cấu hình driver cũ hoặc Fast Startup còn bật | 1. `Setup_CMP30HX_WindowsAIO.bat` đã tích hợp Scheduled Task đa tầng (Logon delay 10s + Startup delay 45s) kèm chế độ **DriverStrategy=2 (Resident Guard)** tự động kiểm tra mỗi phút và retrain lại khi mất Gen2.<br>2. **Khuyến cáo DDU**: Dùng **DDU (Display Driver Uninstaller)** trong chế độ Safe Mode gỡ sạch toàn bộ driver hiển thị cũ trước khi cài lại driver mod để tránh lỗi xung đột registry. |
| **Bị tụt về Gen1 x16 khi card ở chế độ rảnh (Idle)** | Tính năng tiết kiệm điện PCIe ASPM của Windows đang bật | Chạy lại `Setup_CMP30HX_WindowsAIO.bat` (script tự động tắt ASPM) hoặc chỉnh trong Power Options → PCI Express → Link State Power Management: **Off**. |
| **GPU-Z báo Gen2 x16 nhưng AIDA64 chỉ đạt ~2.5 GB/s** | Giá trị Max Read Request Size (MRRS) của card bị kẹp ở 128 Bytes mặc định | Chạy lệnh `40HXInstaller.exe -gen2-30hx` để nâng MRRS lên 512 Bytes và nạp lại hàng đợi DMA. |
| **Không nhận diện được GPU / Mã lỗi 43** | Mối hàn trở mod lane x16 chưa tiếp xúc tốt hoặc card chưa nhận driver | 1. Kiểm tra lại mối hàn trở trên card.<br>2. Cài lại driver NVIDIA (dùng DDU gỡ sạch driver cũ rồi cài bản mới nhất). |
| **Đã thử mọi cách fix vẫn không được** | Driver NVIDIA bị xung đột cấu hình, profile registry lưu đè hoặc service driver lỗi | Gỡ sạch driver cũ bằng **DDU (Display Driver Uninstaller)** trong Safe Mode rồi tiến hành cài đặt lại driver. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 7. Gỡ cài đặt

| Hệ điều hành | Lệnh |
|---|---|
| **Windows** (script) | `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows** (giao diện) | Nhấp chuột phải `40HXUninstaller.exe` → Run as administrator |
| **Linux** | `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

Công cụ sẽ tự xoá Scheduled Task, Registry Run Key, thư mục chương trình và các tệp tạm.

---

## <img src="https://api.iconify.design/lucide/info.svg?color=%238b5cf6" width="22" height="22" align="center" /> 8. Ghi chú kỹ thuật

> [!NOTE]
> - **Tại sao chỉ đạt Gen2 mà không lên được Gen3?**
>   NVIDIA đã ngắt cầu chì phần cứng (Silicon eFuse Bit 3 - 8.0 GT/s) trên nhân TU116 ngay tại nhà máy. Gen2 x16 (5.0 GT/s) là giới hạn vật lý tối đa — công cụ mở khoá an toàn mức trần này qua BAR0 MMIO, không cố ép Gen3 để tránh treo link.
> - **MRRS 512B**: nâng Max Read Request Size từ 128B lên 512B giúp loại bỏ nghẽn phân mảnh gói tin TLP, đạt trần ~6.4 GB/s băng thông DMA.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời cảm ơn

> Dự án lấy cảm hứng từ công trình ban đầu của **CMP40HX-Unlock**. Nếu bạn thấy công cụ hữu ích, hãy để lại 1 Star trên kho lưu trữ để ủng hộ tác giả nhé!
