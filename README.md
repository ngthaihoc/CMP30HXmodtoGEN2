<p align="right">
  <b>Ngôn ngữ:</b>
  <b>Tiếng Việt</b> |
  <a href="README_EN.md">English</a> |
  <a href="README_ZH.md">简体中文</a>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Mở Khoá NVIDIA CMP 30HX v3.0.0 (PCIe Gen2 x16)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-Không%20cần%20thiết-success)](#)

Giải pháp mở khoá băng thông **PCIe Gen2 x16 (~6.4 GB/s)** cho card đồ hoạ **NVIDIA CMP 30HX (nhân TU116)** trên các hệ điều hành Windows 10/11 x64 và Linux (Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux).

- **Cài đặt 1-chạm tự động**: Tự thiết lập môi trường, tắt tiết kiệm điện PCIe ASPM, cấu hình Memory Integrity và tự động kích hoạt khi bật máy.
- **Tương thích mọi Driver**: Hỗ trợ toàn bộ driver NVIDIA (chính thức, desktop, mod, phiên bản mới nhất, không giới hạn bản 537.58).
- **Hệ thống nguyên bản & sạch sẽ**: Không chỉnh sửa BIOS/VBIOS, không cần nạp bootloader EFI, không cần bật Windows Test Signing (không ảnh hưởng game hay phần mềm Anti-cheat).

> [!TIP]
> **Ủng hộ tác giả (Donate)**:  
> Dự án này do em phát triển lúc còn là sinh viên. Nếu được hãy ủng hộ cho em một chút nhé! Cảm ơn mọi người rất nhiều! ❤️  
> 
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
> 
> - **Chủ tài khoản**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub Repository**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. Chuẩn Bị & Tải Về

### 1.1 Yêu cầu phần cứng & phần mềm
1. **Phần cứng**:
   - Card **CMP 30HX (TU116)** đã được mod hàn trở lane PCIe vật lý để nhận diện chế độ x16.
   - Cắm card vào khe PCIe x16 nối trực tiếp CPU (hạn chế dùng cáp riser kém chất lượng hoặc khe phụ qua chipset).
2. **Driver**:
   - Cài đặt bất kỳ bản driver NVIDIA nào bạn muốn (Driver Game Ready / Studio từ trang chủ NVIDIA, driver desktop gán INF, hoặc driver mod).
   - Card cần hiển thị bình thường trong **Device Manager** (mặc định ban đầu nhận Gen1 x16).

### 1.2 Tải bộ công cụ
- Tải toàn bộ kho lưu trữ bằng cách bấm **Code $\rightarrow$ Download ZIP** trên GitHub (hoặc dùng `git clone`).
- Giải nén ra một thư mục cố định trên ổ cứng (ví dụ: `C:\CMP30HX-Unlock` hoặc `D:\CMP30HX-Unlock`).

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. Hướng Dẫn Cài Đặt Chi Tiết

### Cách 1: Cài đặt tự động 1-chạm (Khuyến nghị cho mọi người dùng)

Trong thư mục vừa giải nén, nhấp chuột phải vào tệp **`Setup_CMP30HX_WindowsAIO.bat`** và chọn **Run as administrator** (hoặc nhấp đúp chuột, script sẽ tự động yêu cầu quyền Admin nếu cần).

Script sẽ tự động thực hiện tuần tự 6 bước tối ưu hệ thống:
1. **Tắt PCIe ASPM (Active State Power Management) & Hybrid Sleep**:
   - Ngăn Windows tự động hạ tốc độ link PCIe từ Gen2 về Gen1 khi GPU ở trạng thái nghỉ (idle).
2. **Tắt Fast Startup (Khởi động nhanh / Hiberboot)**:
   - Ngăn Windows lưu cache phiên kernel khi tắt máy, chống lỗi kẹt link Gen1 sau khi khởi động lại máy tính.
3. **Tắt Microsoft Vulnerable Driver Blocklist**:
   - Ngăn Windows Defender và CI chặn nạp driver WinRing0 / ThrottleStop sau khi khởi động lại.
4. **Kiểm tra và tắt Memory Integrity (Core Isolation / HVCI)**:
   - Vô hiệu hoá tính năng chặn driver kernel của Windows trong Registry để công cụ có thể ghi đè thanh ghi BAR0 MMIO.
5. **Triển khai cố định & đăng ký đa cơ chế duy trì Gen2 (`CMP30HX_Gen2_Unlock`)**:
   - Tự động sao chép bộ công cụ vào `%ProgramFiles%\40HXUnlock\` để tránh lỗi mất file sau khi reboot.
   - Tạo Scheduled Task chạy dưới quyền `NT AUTHORITY\SYSTEM` với 3 bộ kích hoạt (Multi-triggers):
     + **AtStartup** (Delay 15s sau khi nạp kernel).
     + **AtLogOn** (Delay 5s khi người dùng đăng nhập).
     + **Wake from Sleep** (Kích hoạt khi máy tỉnh dậy từ chế độ ngủ/Modern Standby qua Event ID 1 của Power-Troubleshooter).
   - Tích hợp thêm bảo hiểm kép qua Registry Run Key (`HKLM` và `HKCU`), đảm bảo vĩnh viễn không bị tụt về Gen1 sau mỗi lần khởi động lại máy hoặc thức dậy từ Sleep.
6. **Mở khoá Gen2 x16 & kích hoạt MRRS 512B ngay lập tức**:
   - Nâng băng thông link lên Gen2 x16 và tối ưu Max Read Request Size lên 512B, card sẵn sàng hoạt động ngay mà không bắt buộc khởi động lại.

> [!IMPORTANT]
> **LƯU Ý VỀ KHỞI ĐỘNG LẠI MÁY (REBOOT)**:  
> - Nếu máy tính của bạn trước đó đang **BẬT Memory Integrity (Core Isolation)**, script sẽ tắt tính năng này và hiện thông báo nhắc nhở.  
> - Trong trường hợp này, bạn **CẦN KHỞI ĐỘNG LẠI MÁY TÍNH (Reboot)** một lần để Windows dỡ bỏ hoàn toàn hypervisor bảo vệ, sau đó khi đăng nhập vào Windows hệ thống sẽ tự động chuyển sang Gen2 x16.

---

### Cách 2: Cài đặt thủ công bằng dòng lệnh (Dành cho người dùng nâng cao)

Nếu muốn tự kiểm soát từng bước qua cửa sổ dòng lệnh (Terminal / Command Prompt / PowerShell Admin):

1. **Tắt PCIe ASPM & Hybrid Sleep**:
   ```cmd
   powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
   powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
   powercfg -setacvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
   powercfg -setdcvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
   powercfg -setactive SCHEME_CURRENT
   ```

2. **Tắt Fast Startup & Microsoft Vulnerable Driver Blocklist** (chống lỗi kẹt Gen1 sau khi reboot):
   ```cmd
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f
   ```

3. **Tắt Memory Integrity (nếu đang bật)**:
   ```cmd
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f
   ```
   *(Khởi động lại máy nếu vừa thay đổi giá trị này từ 1 thành 0).*

4. **Kích hoạt mở khoá Gen2 x16 ngay**:
   ```cmd
   cd /d "Đường_dẫn_thư_mục_giải_nén"
   .\windows-v3.0\release\40HXInstaller.exe -gen2-30hx
   ```

5. **Thiết lập tự động duy trì Gen2 sau Reboot & Wake from Sleep**:
   - **Triển khai bộ cài vào thư mục hệ thống**:
     ```cmd
     if not exist "%ProgramFiles%\40HXUnlock" mkdir "%ProgramFiles%\40HXUnlock"
     copy /y ".\windows-v3.0\release\40HXInstaller.exe" "%ProgramFiles%\40HXUnlock\"
     copy /y ".\windows-v3.0\release\40HXCheck.exe" "%ProgramFiles%\40HXUnlock\"
     ```
   - **Đăng ký Scheduled Task SYSTEM với Multi-trigger (Startup + Logon + Wake) bằng PowerShell**:
     ```powershell
     $action = New-ScheduledTaskAction -Execute "$env:ProgramFiles\40HXUnlock\40HXInstaller.exe" -Argument '-gen2-30hx -silent' -WorkingDirectory "$env:ProgramFiles\40HXUnlock"
     $t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'
     $t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'
     $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew
     $principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
     Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force
     ```
   - **Thêm bảo hiểm kép qua Registry Run Key (HKLM & HKCU)**:
     ```cmd
     reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
     reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
     ```

---

### Cách 3: Cài đặt tự động trên Linux (Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux)

Dành cho các máy chạy Linux hoặc trâu cày/AI server dùng Linux. Tệp **`Setup_CMP30HX_LinuxAIO.sh`** là bộ công cụ All-In-One riêng biệt, chạy trực tiếp không cần cài thêm driver bên ngoài:

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

**Tính năng tự động của `Setup_CMP30HX_LinuxAIO.sh`:**
1. **Tắt PCIe ASPM & Runtime Power Management**: Đặt policy kernel sang `performance` và đặt `power/control=on` cho toàn bộ thiết bị PCI, chống tụt Gen1 khi idle.
2. **Quét & Xử lý toàn bộ card đồ họa**: Tự động phát hiện và áp dụng cho tất cả card CMP 30HX (`10de:2189`) và CMP 40HX (`10de:1f0b`) trên máy.
3. **BAR0 MMIO Direct Injection**: Can thiệp thanh ghi kernel (`XVE_OVR`, `PRIV_MISC_1`, `LINK_CONFIG_0`, `LNKCAP`, `LNKCTL2`).
4. **Tối ưu MRRS 512B & Retrain Link**: Cấu hình Target Link Speed = Gen2, nâng Max Read Request Size lên 512 Bytes (DEVCTL) và thực hiện chu trình retrain đạt trần ~6.4 GB/s.
5. **Cài đặt Systemd Service & Sleep Hook**: Tự động kích hoạt service `cmp30hx-gen2-unlock.service` và sleep hook `/lib/systemd/system-sleep/cmp30hx-unlock` để duy trì Gen2 sau khi reboot hoặc wake up.
6. **Đồng bộ chuẩn Seam 2 StatusContract**: Xuất tệp trạng thái chuẩn hoá (`gen2_status.txt` hoặc qua cờ `--status-file`) với định dạng máy đọc:
   ```text
   STATUS_CODE=GEN2_SUCCESS
   SPEED_CURRENT=2
   WIDTH_CURRENT=16
   TLS_TARGET=2
   ERROR_CODE=NONE
   TIMESTAMP=2026-09-26T05:30:00Z
   ```

**Các lệnh tiện ích trên Linux:**
- Kiểm tra trạng thái link & MRRS hiện tại:
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --status
  ```
- Gỡ bỏ hoàn toàn systemd service và hook:
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
  ```
- Kiểm thử giả lập (Mock Test) không cần GPU thật và không cần quyền root:
  ```bash
  ./Setup_CMP30HX_LinuxAIO.sh -test --no-root
  ```
- Chạy bộ kiểm thử tự động toàn diện trên Linux / WSL (8 suites, 42 assertions):
  ```bash
  chmod +x Test_Mock_CMP30HX_Linux.sh && ./Test_Mock_CMP30HX_Linux.sh
  ```

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 3. Kiểm Tra & Xác Nhận Băng Thông

Sau khi kích hoạt (hoặc sau khi đăng nhập lại Windows), kiểm tra bằng 2 công cụ sau:

1. **Kiểm tra tốc độ link PCIe**:
   - Chạy **`40HXCheck.exe`** (hoặc mở phần mềm **GPU-Z**):
   - Mục **Bus Interface / PCIe**: Hiển thị chính xác **`PCIe x16 2.0 @ x16 2.0`** (hoặc `Gen2 x16`).
2. **Kiểm tra băng thông truyền tải thực tế (Xác nhận MRRS 512B)**:
   - Mở **AIDA64** $\rightarrow$ chọn thanh menu **Tools** $\rightarrow$ chọn **GPGPU Benchmark**.
   - Chọn card đồ hoạ CMP 30HX và bấm **Run Benchmarks**:
   - Kiểm tra hai dòng **Memory Read** và **Memory Copy**:
     - **Thành công**: Đạt khoảng **6.3 – 6.4 GB/s** (đạt trần băng thông lý thuyết của Gen2 x16).
     - **Chưa tối ưu**: Nếu chỉ đạt ~2.5 GB/s là do MRRS đang ở mức mặc định 128B (chạy lại lệnh `-gen2-30hx` để kích hoạt tối ưu 512B).

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 4. Xử Lý Sự Cố Thường Gặp (Troubleshooting)

| Hiện tượng | Nguyên nhân | Hướng giải quyết |
|---|---|---|
| **Kẹt ở Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | - Hệ thống GPU kép (CMP 30HX + iGPU Intel UHD) hoặc driver mod (RainCandy) giữ hook/DMA context, ngăn GPU đàm phán lại tốc độ PCIe.<br>- Hoặc Windows bật Memory Integrity (HVCI), hoặc cáp Riser tiếp xúc kém | 1. **Tự động**: `Setup_CMP30HX_WindowsAIO.bat` bản mới đã tích hợp tự động chu trình **Soft Reset (Disable/Enable qua PnP)** để giải phóng driver và retrain lại Gen2.<br>2. **Thủ công**: Mở **Device Manager** $\rightarrow$ Chuột phải vào **NVIDIA CMP 30HX** $\rightarrow$ Chọn **Disable device** $\rightarrow$ Chọn **Enable device** lại ngay $\rightarrow$ Chạy lại `Setup_CMP30HX_WindowsAIO.bat`.<br>3. Nếu HVCI đang bật: Tắt HVCI và **Khởi động lại máy tính (Reboot)**. |
| **Mất kích hoạt Gen2 sau khi khởi động lại máy** | - Driver NVIDIA/RainCandy nạp trễ hoặc ghi đè trạng thái vBIOS sau khi khởi động.<br>- Xung đột cấu hình driver cũ hoặc Fast Startup còn bật | 1. `Setup_CMP30HX_WindowsAIO.bat` đã tích hợp Scheduled Task SYSTEM (delay 15s) kèm script polling tự động kiểm tra và retrain lại.<br>2. **Khuyến cáo DDU**: Dùng **DDU (Display Driver Uninstaller)** trong chế độ Safe Mode gỡ sạch toàn bộ driver hiển thị cũ trước khi cài lại driver mod để tránh lỗi xung đột registry. |
| **Bị tụt về Gen1 x16 khi card ở chế độ rảnh (Idle)** | Tính năng tiết kiệm điện PCIe ASPM của Windows đang bật | Chạy lại `Setup_CMP30HX_WindowsAIO.bat` (script tự động tắt ASPM) hoặc chỉnh trong Power Options $\rightarrow$ PCI Express $\rightarrow$ Link State Power Management: **Off**. |
| **GPU-Z báo Gen2 x16 nhưng AIDA64 chỉ đạt ~2.5 GB/s** | Giá trị Max Read Request Size (MRRS) của card bị kẹp ở 128 Bytes mặc định | Chạy lệnh `40HXInstaller.exe -gen2-30hx` để nâng MRRS lên 512 Bytes và nạp lại hàng đợi DMA. |
| **Không nhận diện được GPU / Mã lỗi 43** | Mối hàn trở mod lane x16 chưa tiếp xúc tốt hoặc card chưa nhận driver | 1. Kiểm tra lại mối hàn trở trên card.<br>2. Cài lại driver NVIDIA (dùng DDU gỡ sạch driver cũ rồi cài bản mới nhất). |
| **Đã thử mọi cách fix vẫn không được** | Driver NVIDIA bị xung đột cấu hình, profile registry lưu đè hoặc service driver lỗi | Gỡ sạch driver cũ bằng **DDU (Display Driver Uninstaller)** trong Safe Mode rồi tiến hành cài đặt lại driver. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 5. Gỡ Cài Đặt (Uninstall)

Khi muốn đưa hệ thống về trạng thái ban đầu:
- **Cách nhanh**: Chạy lệnh `Setup_CMP30HX_WindowsAIO.bat -uninstall`.
- **Cách qua giao diện**: Nhấp chuột phải vào **`40HXUninstaller.exe`** $\rightarrow$ chọn **Run as administrator**.

Hệ thống sẽ tự động dọn dẹp sạch sẽ:
- Xoá Scheduled Task `CMP30HX_Gen2_Unlock` khỏi Windows.
- Xoá các khoá khởi động ngầm Registry Run Key (`HKLM` và `HKCU`).
- Xoá thư mục chương trình `%ProgramFiles%\40HXUnlock`.
- Dọn dẹp các tệp driver và trạng thái tạm thời trong `%ProgramData%\40HXUnlock`.

---

## <img src="https://api.iconify.design/lucide/cpu.svg?color=%2310b981" width="22" height="22" align="center" /> 6. Kiến Trúc Deep Module & Độ Tin Cậy Phần Cứng (v3.0.0)

Phiên bản v3.0.0 được tái cấu trúc toàn diện theo kiến trúc **Deep Module** với 2 đường ranh giới kỹ thuật (Seams) độc lập:
1. **`LinkNegotiator`**:
   - Đóng gói toàn bộ máy trạng thái đàm phán link PCIe, kẹp cứng giới hạn phần cứng eFuse Gen2 cho TU116, tối ưu hóa DEVCTL MRRS 512B (`0x2000`) và chuỗi ghi shadow register MMIO (`PRIV_MISC_1`, `XVE_OVR`, `LINK_CONFIG_0`, `PL_LINK_RATE`, `CYA_0`).
   - Tự động phát hiện trạng thái ngủ tiết kiệm điện (ASPM) bằng cơ chế lấy mẫu nhanh (Fast-polling 75ms) để ghi nhận link speed tức thì, loại bỏ tình trạng nhận diện sai tốc độ link.
2. **`ComputeInspector` (Dành cho CMP 40HX)**:
   - Tích hợp lớp bảo vệ chống crash phần cứng: đọc và kiểm tra thanh ghi `BOOT_0` (`0x00`) để xác thực đúng họ chip TU106 (`0x16xxxxxx`) trước khi truy xuất vùng nhớ BAR0.
   - Giải mã định kiểu chuẩn hóa thanh ghi kép `SS0` (`0x409664` - cờ mở khoá chính `0x88888888`, ~50 TFLOPS FP16) và `SS1` (`0x40966C` - cờ phụ đồng bộ EFI loader).
3. **`HardwareBus` (Seam 1)**:
   - Tách rời hoàn toàn giao tiếp driver cấp kernel (`WinRing0`, `ThrottleStop`) khỏi logic nghiệp vụ của ứng dụng.
   - Cung cấp `MockHardwareBus` giả lập không gian PCI config và bộ nhớ vật lý MMIO, cho phép chạy trọn vẹn bộ test unit độc lập không cần phần cứng thật.
4. **`StatusContract` (Seam 2)**:
   - Chuẩn hoá định dạng trao đổi dữ liệu trạng thái có cấu trúc (`STATUS_CODE=GEN2_SUCCESS`, `SPEED_CURRENT`, `WIDTH_CURRENT`, `TLS_TARGET`, `ERROR_CODE`) giữa Go engine và các script Batch/PowerShell.
   - Ngăn chặn triệt để lỗi báo thành công giả khi script bị crash hoặc mất tệp trạng thái.

---

## <img src="https://api.iconify.design/lucide/shield-check.svg?color=%2306b6d4" width="22" height="22" align="center" /> 7. Tương Thích Hoàn Toàn Với Anti-Cheat (Riot Vanguard, EAC, BattlEye)

Người dùng CMP 40HX và CMP 30HX hoàn toàn có thể chơi các tựa game có bảo mật gắt gao như **Valorant, League of Legends (Riot Vanguard), Apex Legends, Fortnite (Easy Anti-Cheat / BattlEye)**:

- **Giải pháp Riot Vanguard trên Windows 11 qua `UnlockRiotGame.exe`**:
  - **Với CMP 40HX (TU106)**: Riot Vanguard trên Windows 11 yêu cầu bắt buộc Secure Boot = Enabled và TPM 2.0. Tuy nhiên firmware `40HXUNLK.EFI` chưa có chữ ký số của Microsoft. Công cụ `windows-v3.0/release/UnlockRiotGame.exe` tự động:
    1. Tạo chứng chỉ bảo mật X.509 (`CMP40HX_Key.cer`) với thuật toán SHA256.
    2. Ký số Authenticode cho `40HXUNLK.EFI` (trong cả thư mục phát hành lẫn phân vùng ESP).
    3. Xuất file chứng chỉ `CMP40HX_Key.cer` ra ổ C:\, Desktop và phân vùng ESP (`\EFI\40HX\`).
    4. Cung cấp hướng dẫn trực quan yêu cầu người dùng chụp lại màn hình, khởi động lại vào BIOS, chuyển sang **Custom Mode** và nạp chứng chỉ `CMP40HX_Key.cer` vào cơ sở dữ liệu chữ ký tin cậy **`db`** (Key Management -> Authorized Signatures -> Append Key).
    5. Kết quả: Hệ thống vừa giữ nguyên **Secure Boot BẬT** để đáp ứng kiểm tra của Riot Vanguard (`vgk.sys`), vừa thực thi firmware `40HXUNLK.EFI` để mở khoá toàn bộ Tensor Core (`SS0=0x88888888`, ~50 TFLOPS) và PCIe Gen2.
  - **Với CMP 30HX (TU116)**: Do silicon TU116 không có Tensor Core và mở khoá Gen2 hoàn toàn qua ghi đè thanh ghi BAR0 MMIO trong Windows ring-0 (không nạp EFI loader), người dùng **giữ nguyên Secure Boot BẬT trong BIOS** bình thường mà không cần nạp thêm key, tương thích 100% với Riot Games.
- **Cơ chế Dùng-Xong-Rút (Transient BYOVD on-demand)**:
  - Driver kernel (`WinRing0x64.sys`, `ThrottleStop.sys`) chỉ được nạp lên bộ nhớ trong vài mili-giây lúc hệ thống khởi động hoặc đăng nhập để cấu hình thanh ghi PCIe.
  - Ngay sau khi đàm phán link hoàn tất, công cụ tự động dừng dịch vụ (`sc stop`), xoá dịch vụ (`sc delete`) và xoá bỏ tệp `.sys` khỏi thư mục hệ thống.
  - Khi game hoặc Vanguard (`vgk.sys`) khởi chạy, hệ điều hành hoàn toàn sạch sẽ, không tồn tại bất kỳ driver danh sách đen hay tiến trình chạy ngầm nào.
- **Không yêu cầu Windows Test Signing**:
  - Không cần lệnh `bcdedit /set testsigning on` nguy hiểm (vốn bị Vanguard chặn 100%).
  - Môi trường Windows giữ nguyên chứng thực toàn vẹn mã gốc của Microsoft.
- **Pre-boot EFI cho CMP 40HX**:
  - Tensor Core được mở khoá ở giai đoạn UEFI trước khi Windows khởi động. Đến khi Windows và driver anti-cheat nạp, card đã ở trạng thái mở khoá tự nhiên ở mức phần cứng.

---

## <img src="https://api.iconify.design/lucide/info.svg?color=%238b5cf6" width="22" height="22" align="center" /> 8. Ghi Chú Kỹ Thuật Tóm Tắt

> [!NOTE]
> - **Tại sao trần là Gen2 x16 mà không thể lên Gen3?**  
>   Trên nhân TU116 của CMP 30HX, NVIDIA đã ngắt cầu chì phần cứng **Silicon eFuse (Bit 3 - 8.0 GT/s)** ngay tại nhà máy. Vì vậy, Gen2 x16 (5.0 GT/s) là giới hạn vật lý tối đa của phần cứng. Công cụ can thiệp qua BAR0 MMIO để mở khoá mức trần này an toàn 100%, tuyệt đối không cố ép Gen3 để tránh lỗi treo link huấn luyện lại.
> - **MRRS 512B**:  
>   Việc nâng Max Read Request Size từ 128B lên 512B giúp loại bỏ nghẽn phân mảnh gói tin TLP, giải phóng toàn bộ ~6.4 GB/s băng thông bộ nhớ DMA.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời Cảm Ơn (Acknowledgments)

> Dự án lấy cảm hứng từ công trình ban đầu của **CMP40HX-Unlock**. Nếu thấy công cụ hữu ích, xin hãy để lại 1 Star trên kho lưu trữ để ủng hộ tác giả nhé!
