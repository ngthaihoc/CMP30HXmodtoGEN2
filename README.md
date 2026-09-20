# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Mở Khoá NVIDIA CMP 30HX v3.0.0 (PCIe Gen2 x16)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%2010%20%7C%2011%20x64-0078D6?logo=windows&logoColor=white)](https://microsoft.com)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-Không%20cần%20thiết-success)](#)

Giải pháp mở khoá băng thông **PCIe Gen2 x16 (~6.4 GB/s)** cho card đồ hoạ **NVIDIA CMP 30HX (nhân TU116)** trên hệ điều hành Windows 10/11 x64.

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

Trong thư mục vừa giải nén, nhấp chuột phải vào tệp **`Setup_CMP30HX.bat`** và chọn **Run as administrator** (hoặc nhấp đúp chuột, script sẽ tự động yêu cầu quyền Admin nếu cần).

Script sẽ tự động thực hiện tuần tự 6 bước tối ưu hệ thống:
1. **Tắt PCIe ASPM (Active State Power Management) & Hybrid Sleep**:
   - Ngăn Windows tự động hạ tốc độ link PCIe từ Gen2 về Gen1 khi GPU ở trạng thái nghỉ (idle).
2. **Tắt Fast Startup (Khởi động nhanh / Hiberboot)**:
   - Ngăn Windows lưu cache phiên kernel khi tắt máy, chống lỗi kẹt link Gen1 sau khi khởi động lại máy tính.
3. **Tắt Microsoft Vulnerable Driver Blocklist**:
   - Ngăn Windows Defender và CI chặn nạp driver WinRing0 / ThrottleStop sau khi khởi động lại.
4. **Kiểm tra và tắt Memory Integrity (Core Isolation / HVCI)**:
   - Vô hiệu hoá tính năng chặn driver kernel của Windows trong Registry để công cụ có thể ghi đè thanh ghi BAR0 MMIO.
5. **Đăng ký tác vụ khởi động ngầm (`CMP30HX_Gen2_Unlock`)**:
   - Tạo Scheduled Task với tài khoản `NT AUTHORITY\SYSTEM` tự động tắt ASPM và kích hoạt chế độ `-gen2-30hx -silent` với quyền cao nhất mỗi khi bạn đăng nhập Windows. Bạn không cần phải mở công cụ hay thao tác thủ công sau mỗi lần bật máy.
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

5. **Tạo Scheduled Task để tự động kích hoạt khi đăng nhập Windows**:
   - **Bằng PowerShell (Khuyến nghị - chạy ngầm SYSTEM)**:
     ```powershell
     $a1 = New-ScheduledTaskAction -Execute 'powercfg.exe' -Argument '-setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0'
     $a2 = New-ScheduledTaskAction -Execute 'powercfg.exe' -Argument '-setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0'
     $a3 = New-ScheduledTaskAction -Execute 'powercfg.exe' -Argument '-setactive SCHEME_CURRENT'
     $a4 = New-ScheduledTaskAction -Execute "$PWD\windows-v3.0\release\40HXInstaller.exe" -Argument '-gen2-30hx -silent'
     $trigger = New-ScheduledTaskTrigger -AtLogOn
     $principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
     Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action @($a1, $a2, $a3, $a4) -Trigger $trigger -Principal $principal -Force
     ```
   - **Bằng CMD**:
     ```cmd
     schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "cmd.exe /c powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0 & powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0 & powercfg -setactive SCHEME_CURRENT & \"%CD%\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /ru SYSTEM /rl highest /f
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
| **Kẹt ở Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | Windows bật Memory Integrity (HVCI) chặn nạp driver kernel can thiệp MMIO, hoặc cáp Riser lỏng / tiếp xúc kém | 1. Chạy `Setup_CMP30HX.bat` và **Khởi động lại máy tính (Reboot)**.<br>2. Cắm card trực tiếp vào khe PCIe x16 nối CPU, hạn chế dùng cáp riser. |
| **Bị tụt về Gen1 x16 khi card ở chế độ rảnh (Idle)** | Tính năng tiết kiệm điện PCIe ASPM của Windows đang bật | Chạy lại `Setup_CMP30HX.bat` (script tự động tắt ASPM) hoặc chỉnh trong Power Options $\rightarrow$ PCI Express $\rightarrow$ Link State Power Management: **Off**. |
| **GPU-Z báo Gen2 x16 nhưng AIDA64 chỉ đạt ~2.5 GB/s** | Giá trị Max Read Request Size (MRRS) của card bị kẹp ở 128 Bytes mặc định | Chạy lệnh `40HXInstaller.exe -gen2-30hx` để nâng MRRS lên 512 Bytes và nạp lại hàng đợi DMA. |
| **Không nhận diện được GPU / Mã lỗi 43** | Mối hàn trở mod lane x16 chưa tiếp xúc tốt hoặc card chưa nhận driver | 1. Kiểm tra lại mối hàn trở trên card.<br>2. Cài lại driver NVIDIA (có thể dùng DDU quét sạch driver cũ rồi cài bản mới nhất). |
| **Đã thử mọi cách fix vẫn không được** | Driver NVIDIA bị xung đột cấu hình, profile registry lưu đè hoặc service driver lỗi | Gỡ sạch driver cũ bằng **DDU (Display Driver Uninstaller)** rồi tiến hành cài đặt lại driver NVIDIA. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 5. Gỡ Cài Đặt (Uninstall)

Khi muốn đưa hệ thống về trạng thái ban đầu:
- **Cách nhanh**: Chạy lệnh `Setup_CMP30HX.bat -uninstall`.
- **Cách qua giao diện**: Nhấp chuột phải vào **`40HXUninstaller.exe`** $\rightarrow$ chọn **Run as administrator**.

Hệ thống sẽ tự động dọn dẹp sạch sẽ:
- Xoá Scheduled Task `CMP30HX_Gen2_Unlock` khỏi Windows.
- Dọn dẹp các tệp driver tạm thời trong `%ProgramData%\40HXUnlock`.

---

## <img src="https://api.iconify.design/lucide/info.svg?color=%238b5cf6" width="22" height="22" align="center" /> 6. Ghi Chú Kỹ Thuật Tóm Tắt

> [!NOTE]
> - **Tại sao trần là Gen2 x16 mà không thể lên Gen3?**  
>   Trên nhân TU116 của CMP 30HX, NVIDIA đã ngắt cầu chì phần cứng **Silicon eFuse (Bit 3 - 8.0 GT/s)** ngay tại nhà máy. Vì vậy, Gen2 x16 (5.0 GT/s) là giới hạn vật lý tối đa của phần cứng. Công cụ can thiệp qua BAR0 MMIO để mở khoá mức trần này an toàn 100%, tuyệt đối không cố ép Gen3 để tránh lỗi treo link huấn luyện lại.
> - **MRRS 512B**:  
>   Việc nâng Max Read Request Size từ 128B lên 512B giúp loại bỏ nghẽn phân mảnh gói tin TLP, giải phóng toàn bộ ~6.4 GB/s băng thông bộ nhớ DMA.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời Cảm Ơn (Acknowledgments)

> Dự án lấy cảm hứng từ công trình ban đầu của **CMP40HX-Unlock**. Nếu thấy công cụ hữu ích, xin hãy để lại 1 Star trên kho lưu trữ để ủng hộ tác giả nhé!
