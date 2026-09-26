# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Công Cụ Mở Khoá Windows Cho NVIDIA CMP 40HX & CMP 30HX v3.0.0

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%2010%20%7C%2011%20x64-0078D6?logo=windows&logoColor=white)](https://microsoft.com)
[![GPU](https://img.shields.io/badge/NVIDIA-TU106%20%7C%20TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Status](https://img.shields.io/badge/Test%20Signing-Not%20Required-success)](#)

**CMP 40HX (TU106) / CMP 30HX (TU116) → Tối Đa Tốc Độ Tensor Core + PCIe Gen2 x16**  
Chạy trực tiếp trên Windows nguyên bản, cài đặt một chạm, tự động kích hoạt khi khởi động, không cần thao tác thủ công mỗi lần mở máy.  
Từ bản v2.5 trở đi **không cần bật chế độ Test Signing**, hệ thống luôn sạch sẽ, không ảnh hưởng đến phần mềm chống gian lận (Anti-cheat) khi chơi game.

> [!TIP]
> **Ủng hộ tác giả (Donate)**:  
> Dự án này do em làm lúc còn là sinh viên. Nếu các bác thấy hữu ích và thích công cụ này thì donate ủng hộ cho em tí nhé ạ! Cảm ơn mọi người rất nhiều! ❤️  
> 
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
> 
> - **Chủ tài khoản**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub Repository**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
> - *This project is inspired by **CMP40HX-Unlock**. If you find it useful, please consider giving a star to both the original author and this repository.*

**Kết quả đo thực tế trên phần cứng:**

| Chỉ số | CMP 40HX (TU106) | CMP 30HX (TU116 Mod x16) |
|---|---|---|
| **Hiệu năng Tính toán** | `SS0=0x88888888`, FP16 HGEMM **~50 TFLOPS** (Gốc ~8T) | FP32/Tensor mặc định theo VBIOS gốc |
| **Băng thông PCIe** | Gen2 ×16 (~6.4 GB/s lý thuyết) | Gen2 ×16 (**~6.3 – 6.4 GB/s** AIDA64 sau khi chỉnh MRRS 512B) |
| **Driver NVIDIA** | Hoạt động bình thường, không Code 43, bật GSP | Hoạt động bình thường, không cần GSP |

---

## <img src="https://api.iconify.design/lucide/folder-tree.svg?color=%230284c7" width="22" height="22" align="center" /> Cấu Trúc Kho Lưu Trữ

Kho lưu trữ này chứa toàn bộ mã nguồn công cụ mở khoá **CMP 40HX & 30HX Windows Unlock v3.0.0** (Go + C).

- **Sử dụng trực tiếp:** Tải tệp thực thi tại [`release/`](release/) bao gồm `40HXInstaller.exe`, `40HXUninstaller.exe`, `40HXCheck.exe`.
- **Mã nguồn Firmware EFI:** `tools/unlock40x/` (`unlock40x_v70.c` + script build `build_v70.sh`).
- **Tài liệu bối cảnh kỹ thuật chi tiết:** [../PROJECT_CONTEXT.md](../PROJECT_CONTEXT.md)

**Biên dịch từ mã nguồn** (Yêu cầu Go 1.20+, chạy trong thư mục `tools/`):

```bat
cd tools\inst40hx     && go build -a -trimpath -ldflags="-H=windowsgui -s -w" -o ..\release\40HXInstaller.exe .
cd ..\uninstall40x    && go build -a -trimpath -ldflags="-H=windowsgui -s -w" -o ..\release\40HXUninstaller.exe .
cd ..\check40x        && go build -a -trimpath -ldflags="-H=windowsgui -s -w" -o ..\release\40HXCheck.exe .
```

---

## <img src="https://api.iconify.design/lucide/microchip.svg?color=%238b5cf6" width="22" height="22" align="center" /> 0. Hướng Dẫn Nhanh Dành Cho CMP 30HX (TU116)

> [!NOTE]
> CMP 30HX (mã GPU `10DE:2189`, ví dụ Gigabyte GV-N30HXD6-6G) sau khi đã hàn trở mod vật lý các lane PCIe để nhận x16 Gen1, sử dụng công cụ này để nâng băng thông lên **PCIe Gen2 x16** (~6.4 GB/s).

### 0.1 Bản chất kỹ thuật: Tại sao chỉ lên được Gen2 mà không lên được Gen3?

> [!IMPORTANT]
> - **Gen2 (5.0 GT/s)**: Bị khoá mềm bởi thanh ghi bóng (VBIOS shadow registers). Công cụ can thiệp qua BAR0 MMIO để ghi đè vector tốc độ link và huấn luyện lại link thành công 100%.
> - **Gen3 (8.0 GT/s)**: Bị **đứt eFuse ở cấp độ chip silicon (Physical Silicon eFuse Blown)** do NVIDIA cấu hình khi xuất xưởng:
>   - Ghi `0x0E` vào `LNKCAP2` (`0x0880A4`) $\rightarrow$ đọc ngược lại chỉ trả về `0x00000006` (Bit 3 bị ngắt vật lý).
>   - Ghi `0x00010003` vào `LNKCTL2` (`0x0880A8`) $\rightarrow$ đọc ngược lại chỉ trả về `0x00010002` (Target Link Speed bị kẹp ở Gen2).
>   - Xung nhịp PHY Lane 0 (`0x08C4B0`) cố định ở 5.0 GHz (`0x50000000`), không thể nâng lên 8.0 GHz.
> - Do đó, **Gen2 x16 là giới hạn vật lý tối đa của CMP 30HX**. Không cố ép Gen3 để tránh lỗi vòng lặp retrain.

### 0.2 Tinh chỉnh DEVCTL MRRS (Mở khoá toàn bộ băng thông DMA)
Mặc định `DEVCTL` (`cap + 0x08`) có Max Read Request Size (MRRS) đặt là 128 Bytes, gây phân mảnh gói tin TLP khiến tốc độ AIDA64 Memory Copy bị nghẽn ở ~2.5 GB/s (như Gen1).  
Công cụ tự động nâng MRRS lên **512 Bytes** (`0x2000`) và khởi động lại `NVDisplay.ContainerLocalSystem`, giúp băng thông đạt tối đa **~6.3 – 6.4 GB/s** (đạt ~98% lý thuyết của Gen2 x16).

### 0.3 Lệnh khởi động tự động Gen2 cho CMP 30HX khi bật máy
Khuyến nghị chạy script **`Setup_CMP30HX_WindowsAIO.bat`** ở thư mục gốc để tự cấu hình 1-chạm.  
Nếu muốn cấu hình thủ công:
- **Trên PowerShell**:
```powershell
$a = New-ScheduledTaskAction -Execute "$PWD\windows-v3.0\release\40HXInstaller.exe" -Argument '-gen2-30hx -silent'
$t = New-ScheduledTaskTrigger -AtLogOn
$p = New-ScheduledTaskPrincipal -UserId $env:USERNAME -RunLevel Highest
Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $a -Trigger $t -Principal $p -Force
```
- **Trên Command Prompt (CMD)**:
```cmd
schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"%CD%\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f
```
*(Nếu từng bị lỗi vòng lặp thử lại 1 phút do bản cũ, xoá bằng: `schtasks /delete /tn "40HXGen2Retry" /f`)*.

---

## <img src="https://api.iconify.design/lucide/package.svg?color=%23f59e0b" width="22" height="22" align="center" /> 1. Thành Phần Trong Gói Phát Hành

| Tệp | Mục đích |
|---|---|
| `40HXInstaller.exe` | **Giao diện cài đặt và quản lý** (Mặc định mở GUI; hỗ trợ tham số dòng lệnh) |
| `40HXUninstaller.exe` | **Gỡ cài đặt tự động** (Nhấp đúp $\rightarrow$ Yêu cầu quyền Administrator) |
| `40HXCheck.exe` | **Chẩn đoán độc lập** (Kiểm tra Hashrate/Năng lực tính toán + Tốc độ link PCIe Gen2; tự thu hồi driver sau khi đo) |
| `UnlockRiotGame.exe` | **Mở khoá tương thích Riot Games / Vanguard**: Ký số Authenticode cho `40HXUNLK.EFI` và hướng dẫn nạp key vào BIOS `db` (Custom Mode) trên Windows 11 cho CMP 40HX; kiểm tra và giữ nguyên Secure Boot cho CMP 30HX |
| `OpenCL.exe` | **Kiểm tra năng lực tính toán (CMP 40HX)** (So sánh hiệu năng FP16/FP32 trước và sau khi mở khoá) |
| `files\40HXUNLK.EFI` | Firmware EFI mở khoá (Chỉ dành cho CMP 40HX TU106; không nạp cho 30HX) |
| `WinRing0x64.sys` | Driver truy cập PCI Configuration Space và MMIO |

---

## <img src="https://api.iconify.design/lucide/settings.svg?color=%2364748b" width="22" height="22" align="center" /> 2. Chuẩn Bị BIOS Trước Khi Cài Đặt (Áp dụng cho CMP 40HX)

> [!CAUTION]
> Đối với CMP 40HX (TU106), firmware mở khoá hoạt động ở giai đoạn khởi động UEFI, do đó cần cấu hình BIOS như sau:

### 2.1 Các mục bắt buộc thiết lập
| Ưu tiên | Cài đặt | Giá trị | Giải thích |
|---|---|---|---|
| <img src="https://api.iconify.design/lucide/star.svg?color=%23eab308" width="16" height="16" /> Bắt buộc | **Above 4G Decoding** | **Enabled** | Bắt buộc bật. Nếu tắt, EFI không thể nạp payload vào bộ nhớ trên 4GB |
| <img src="https://api.iconify.design/lucide/star.svg?color=%23eab308" width="16" height="16" /> Bắt buộc | **Secure Boot** | **Disabled** | Tắt Secure Boot để firmware EFI không chứng thực có thể khởi chạy |
| <img src="https://api.iconify.design/lucide/star.svg?color=%23eab308" width="16" height="16" /> Bắt buộc | **CSM / Compatibility Support Module** | **Disabled** (Pure UEFI) | Để mục khởi động "40HX Unlock" xuất hiện trong danh sách boot |
| Khuyến nghị | **Fast Boot / Khởi động nhanh** | **Disabled** | Tránh bỏ qua các dịch vụ EFI trong quá trình POST |
| Khuyến nghị | **Resizable BAR** | **Auto / Enabled** | Tối ưu truyền tải bộ nhớ với Above 4G |

### 2.2 Vị trí khe cắm
- **Cắm card vào khe PCIe x16 đầu tiên** kết nối trực tiếp với CPU.
- Với hệ thống chạy 2 GPU (xuất hình bằng card phụ), cắm card xuất hình ở khe phụ, giữ 40HX ở khe chính x16.
- Đặt **"40HX Unlock"** lên vị trí đầu tiên trong danh sách ưu tiên khởi động (Boot Priority).

### 2.3 Chế độ phân vùng và nguồn điện
- **Bắt buộc dùng chuẩn UEFI + GPT**: Ổ đĩa cài Windows phải theo định dạng GPT và có phân vùng EFI. Nếu ổ đĩa là Legacy/MBR, hãy chuyển đổi bằng công cụ chính thức của Microsoft:
  ```cmd
  mbr2gpt /validate /allowfullos
  mbr2gpt /convert /allowfullos
  ```
- **Tắt Khởi động nhanh (Fast Startup) trong Windows**: Tránh việc Windows sử dụng ngủ đông lai (Hybrid Sleep) bỏ qua quá trình khởi động UEFI.
- **Tắt tiết kiệm điện liên kết PCIe (ASPM)**: Tránh việc GPU tự hạ xung xuống Gen1 khi không có tải nặng.

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 3. Cài Đặt Và Sử Dụng

### 3.1 Cài đặt cho CMP 40HX qua giao diện GUI
1. Nhấp đúp vào `40HXInstaller.exe` (chọn Yes khi UAC hỏi quyền Admin).
2. Tại vùng **① Cài đặt thành phần**: Các mục thiếu sẽ được tự động tích chọn. Nhấn **[Cài đặt toàn bộ]** (Bao gồm GSP + Driver Gen2 + EFI + Tự khởi động + Tắt Fast Startup/ASPM).
3. Khởi động lại máy tính. Màn hình khởi động sẽ hiện thông báo mở khoá trong 1-2 giây rồi vào Windows.
4. Sau khi đăng nhập, tiến trình nền tự động hoàn thành mở khoá Gen2 và dọn dẹp driver khỏi bộ nhớ.

### 3.2 Cài đặt cho CMP 30HX (Chi tiết từng bước: Bỏ qua cài đặt 40HX)

> [!IMPORTANT]
> **Lưu ý đặc biệt (Bỏ qua toàn bộ các bước cấu hình của CMP 40HX):**
> - **KHÔNG chỉnh sửa BIOS**: Không cần tắt Secure Boot, không cần bật CSM, không bắt buộc Above 4G (khác với 40HX phải nạp EFI bootloader không chứng thực).
> - **KHÔNG nạp tệp EFI (`40HXUNLK.EFI`)**: CMP 30HX chạy hoàn toàn trên môi trường Windows thông qua MMIO override, không sử dụng firmware bootloader.
> - **KHÔNG nhấn "Cài đặt toàn bộ" trên GUI của `40HXInstaller.exe`**: Nút bấm này sẽ cấu hình EFI và GSP dành riêng cho nhân TU106 (40HX).
> - **KHÔNG kích hoạt GSP Firmware (`EnableGpuFirmware=1`)**: Kiến trúc TU116 của 30HX không hỗ trợ và không cần GSP.
> - **Hỗ trợ mọi phiên bản Driver**: Tương thích với bất kỳ driver NVIDIA nào (chính thức, desktop, mod, không giới hạn phiên bản).

#### Các bước cài đặt chi tiết:

1. **Chuẩn bị môi trường & Driver**:
   - Đảm bảo card CMP 30HX đã được mod hàn trở lane vật lý x16 và nhận diện ổn định trong Device Manager (thường ở tốc độ mặc định Gen1 x16).
   - Cài đặt driver NVIDIA: Hỗ trợ **mọi phiên bản driver** (driver chính thức NVIDIA, driver desktop, driver mod hoặc bản mới nhất đều được, không giới hạn phiên bản).

2. **Chạy mở khoá Gen2 ngay lần đầu (Không cần khởi động lại)**:
   - Nhấp chuột phải vào nút Start menu $\rightarrow$ Chọn **Terminal (Admin)** hoặc **Command Prompt (Administrator)**.
   - Di chuyển đến thư mục dự án vừa giải nén (hoặc clone):
     ```cmd
     cd /d "Đường_dẫn_thư_mục_dự_án"
     ```
   - Chạy lệnh kích hoạt trực tiếp:
     ```cmd
     .\windows-v3.0\release\40HXInstaller.exe -gen2-30hx
     ```
   - Công cụ sẽ mở driver `WinRing0x64.sys`, ghi đè các thanh ghi bóng BAR0 (`0x08841C`, `0x08872C`, `0x08C040`, `0x08C2C0`), nâng MRRS lên 512B và gửi tín hiệu retrain. Thông báo `[Gen2-30HX] ✅ Gen2 Thành công: Link hiện tại Gen2 x16` xuất hiện.

3. **Thiết lập tự động mở khoá khi đăng nhập Windows**:
   - Do phần cứng GPU trở về trạng thái gốc Gen1 sau mỗi lần khởi động lại máy tính (cold boot / reboot), bạn chỉ cần tạo một Scheduled Task để tự động kích hoạt khi đăng nhập tài khoản.
   - **Cách 1: Khuyến nghị cho PowerShell (Native, không lo lỗi escape ngoặc kép)**:
     ```powershell
     $a = New-ScheduledTaskAction -Execute "$PWD\windows-v3.0\release\40HXInstaller.exe" -Argument '-gen2-30hx -silent'
     $t = New-ScheduledTaskTrigger -AtLogOn
     $p = New-ScheduledTaskPrincipal -UserId $env:USERNAME -RunLevel Highest
     Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $a -Trigger $t -Principal $p -Force
     ```
   - **Cách 2: Dành cho Command Prompt (CMD)**:
     ```cmd
     schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"%CD%\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f
     ```
     *(Mẹo: Nếu dùng `schtasks` trên PowerShell, thêm `--%` để tránh PowerShell nuốt ngoặc kép: `schtasks --% /create /tn "CMP30HX_Gen2_Unlock" /tr "\"$PWD\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f`).*
   - **Giải thích tham số**:
     - `-gen2-30hx`: Kích hoạt chế độ mở khoá riêng biệt cho nhân TU116 (kẹp cứng Gen2, cấm ép Gen3, tối ưu MRRS 512B).
     - `-silent`: Chạy hoàn toàn ngầm không hiện cửa sổ, tự kết thúc sau ~1-2 giây và tự giải phóng driver `WinRing0x64.sys` khỏi RAM.

4. **Dọn dẹp tác vụ lặp cũ (Nếu từng dùng bản cũ trước đây)**:
   ```cmd
   schtasks /delete /tn "40HXGen2Retry" /f
   ```

5. **Xác nhận tốc độ và băng thông thực tế**:
   - Nhấp đúp vào `40HXCheck.exe`: Xác nhận mục `PCIe: Gen2 x16`.
   - Khởi chạy **AIDA64** $\rightarrow$ **Tools** $\rightarrow$ **GPGPU Benchmark** $\rightarrow$ Đo kiểm **Memory Read / Memory Copy**: Tốc độ đạt **~6.3 – 6.4 GB/s** (gấp 2.5 lần mức ~2.5 GB/s gốc).

### 3.3 Các tham số dòng lệnh hữu ích
```cmd
40HXInstaller.exe               # Khởi chạy giao diện đồ hoạ GUI
40HXInstaller.exe -status       # Kiểm tra toàn diện trạng thái phần cứng, driver và PCIe
40HXInstaller.exe -gen2         # Kích hoạt Gen2 cho CMP 40HX ngay lập tức
40HXInstaller.exe -gen2-30hx    # Kích hoạt Gen2 cho CMP 30HX (áp dụng chuẩn TU116 và tối ưu MRRS 512B)
40HXInstaller.exe -uninstall    # Gỡ bỏ sạch sẽ toàn bộ dịch vụ, tác vụ và EFI

UnlockRiotGame.exe              # Khởi chạy giao diện GUI hỗ trợ Riot Vanguard & Secure Boot
UnlockRiotGame.exe -status      # Kiểm tra GPU, phiên bản Windows và trạng thái Secure Boot
UnlockRiotGame.exe -sign        # Ký số tự động cho 40HXUNLK.EFI và xuất CMP40HX_Key.cer
```

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 4. Kiểm Tra Và Xác Nhận (Bằng 40HXCheck.exe)

Sau khi vào Windows, nhấp đúp vào **`40HXCheck.exe`**:
- **CMP 40HX**:
  - `SS0`: Báo `✓ Đầy đủ (SS0=0x88888888)` là đã mở khoá toàn bộ sức mạnh tính toán.
  - `PCIe`: Báo `Gen2 x16` là đạt chuẩn.
- **CMP 30HX**:
  - `WinRing0`: Báo `✓` (Driver cấu hình đã kích hoạt).
  - `PCIe`: Báo `Gen2 x16`.
  - Mở **AIDA64 GPGPU Benchmark** kiểm tra **Memory Read / Memory Copy**: Đạt khoảng **6300 – 6400 MB/s** chứng tỏ MRRS 512B đã kích hoạt thành công (nếu chỉ đạt ~2500 MB/s là đang chạy ở cấu hình 128B).

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 5. Xử Lý Sự Cố Thường Gặp

| Hiện tượng | Nguyên nhân | Hướng khắc phục |
|---|---|---|
| **Khởi động lại bị về Gen1** | Chưa đăng ký tác vụ tự khởi động khi đăng nhập | Tạo Scheduled Task theo mục 0.3 (đối với 30HX) hoặc mục 3.1 (đối với 40HX). |
| **GPU-Z báo Gen2 nhưng AIDA64 chỉ đạt 2.5 GB/s** | DEVCTL MRRS bị kẹp ở 128B mặc định | Chạy `40HXInstaller.exe -gen2-30hx` để kích hoạt tối ưu MRRS 512B và nạp lại hàng đợi DMA. |
| **Bị treo ở vòng lặp tác vụ 40HXGen2Retry** | Do phiên bản cũ ép cờ Gen3 trên CMP 30HX | Chạy `schtasks /delete /tn "40HXGen2Retry" /f` và cập nhật bản `40HXInstaller.exe` mới nhất đã kẹp cứng Gen2. |
| **Không nhận diện được GPU** | Chưa cắm chắc card hoặc thiếu driver NVIDIA | Cài đặt driver NVIDIA (hỗ trợ mọi phiên bản driver chính thức hoặc mod) và kiểm tra Device Manager. |
| **Màn hình đen sau khi mở khoá 40HX** | Chưa bật GSP Firmware (`EnableGpuFirmware=1`) | Chạy `40HXInstaller.exe -status` để kiểm tra cờ GSP và bật lại qua GUI. |

---

## <img src="https://api.iconify.design/lucide/cpu.svg?color=%2310b981" width="22" height="22" align="center" /> 6. Kiến Trúc Deep Module & Cơ Chế An Toàn (v3.0.0)

Phiên bản v3.0.0 áp dụng kiến trúc **Deep Module** với các tầng trừu tượng hoá mạnh mẽ:
- **`LinkNegotiator`**: Đóng gói toàn bộ máy trạng thái huấn luyện link PCIe, kẹp eFuse Gen2 cho TU116, tối ưu hóa DEVCTL MRRS 512B (`0x2000`) và chuỗi ghi shadow register MMIO (`PRIV_MISC_1`, `XVE_OVR`, `LINK_CONFIG_0`, `PL_LINK_RATE`, `CYA_0`).
- **`ComputeInspector`**: Kiểm tra an toàn `BOOT_0` (`0x16` cho TU106) chống crash hệ thống và giải mã định kiểu Tensor Core (`SS0 == 0x88888888`, `SS1 == 0x40966C`).
- **`HardwareBus` (Seam 1)**: Tách biệt hoàn toàn kernel driver (`WinRing0`, `ThrottleStop`) khỏi nghiệp vụ chính, kèm `MockHardwareBus` cho phép kiểm thử đơn vị độc lập.
- **`StatusContract` (Seam 2)**: Chuẩn hoá hợp đồng trạng thái định kiểu (`STATUS_CODE=GEN2_SUCCESS`, v.v.) giữa Go Engine và các script Batch.
- **Tương thích Anti-Cheat (Riot Vanguard / EAC / BattlEye) qua `UnlockRiotGame.exe`**:
  - **CMP 40HX**: Riot Vanguard trên Windows 11 yêu cầu Secure Boot = Enabled và TPM 2.0. `UnlockRiotGame.exe` tự động ký số Authenticode cho `40HXUNLK.EFI`, xuất chứng chỉ `CMP40HX_Key.cer` ra ổ C:, Desktop và ESP, hướng dẫn người dùng nạp key vào BIOS `db` (Custom Mode). Nhờ vậy, Secure Boot vẫn BẬT cho Vanguard trong khi `40HXUNLK.EFI` vẫn chạy được để mở khoá Tensor Core `SS0=0x88888888` và Gen2.
  - **CMP 30HX**: Hoàn toàn không dùng firmware EFI (mở khoá qua MMIO ring-0), vì vậy Secure Boot có thể giữ nguyên BẬT trong BIOS, tương thích 100% với Riot Vanguard mà không cần nạp key.
  - **Mô hình Dùng-Xong-Rút (Transient BYOVD)**: Driver kernel chỉ nạp trong mili-giây lúc khởi động rồi giải phóng ngay lập tức, không để lại driver trong danh sách đen khi game kiểm tra.
- **Bộ kiểm thử tự động toàn diện**: 13 Go unit tests (`40hxcore`), 18 Go unit tests (`unlockriot`), 10 Mock test suites Windows (`Test_Mock_CMP30HX.bat`), và 8 Mock test suites Linux (`Test_Mock_CMP30HX_Linux.sh`).

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 7. Gỡ Cài Đặt Hoàn Toàn

Để đưa hệ thống về trạng thái nguyên bản xuất xưởng:
1. Nhấp đúp vào **`40HXUninstaller.exe`** với quyền Administrator (hoặc chạy `40HXInstaller.exe -uninstall`).
2. Chương trình sẽ tự động xoá:
   - Các tác vụ lịch trình Scheduled Tasks (`CMP30HX_Gen2_Unlock`, `40HXGen2Retry`, v.v.).
   - Khoá tự khởi động Run trong Registry.
   - Mục khởi động EFI trong NVRAM và tệp mở khoá trong phân vùng ESP.
   - Driver dịch vụ và tệp tạm trong `%ProgramData%\40HXUnlock`.
3. Khởi động lại máy tính.

---

## <img src="https://api.iconify.design/lucide/shield-alert.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. Quy Tắc An Toàn Phần Cứng Cốt Lõi

1. **Không nạp firmware 40HX lên 30HX**: Tuyệt đối không sao chép `40HXUNLK.EFI` hay blob GA102/TU106 lên CMP 30HX. Cấu trúc VBIOS và bộ điều khiển hoàn toàn khác biệt.
2. **Không ép Gen3 trên CMP 30HX**: eFuse đã đứt vật lý, mọi thao tác ép Gen3 đều vô hiệu và kích hoạt vòng lặp lỗi retrain.
3. **Không gọi Reset cứng**: Không kích hoạt `gen2RootLinkDisable`, `gen2HardFallback` hoặc PnP device restart tự ý trên CMP 30HX (chỉ chạy khi có cờ `-hard`).
4. **Nguyên tắc Fail-closed**: Luôn kiểm tra tính sẵn sàng của PCIe Capability và readback an toàn trước khi thực hiện bất kỳ lệnh ghi nào vào PCI Config hoặc MMIO.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời Cảm Ơn (Acknowledgments)

> This project is inspired by **CMP40HX-Unlock**. If you find it useful, please consider giving a star to both the original author and this repository.
