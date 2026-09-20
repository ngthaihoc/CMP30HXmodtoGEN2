# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Công Cụ Mở Khoá Windows Cho NVIDIA CMP 30HX v3.0.0

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%2010%20%7C%2011%20x64-0078D6?logo=windows&logoColor=white)](https://microsoft.com)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Status](https://img.shields.io/badge/Test%20Signing-Not%20Required-success)](#)

**CMP 30HX (TU116) → Mở Khoá Băng Thông PCIe Gen2 x16 (~6.4 GB/s)**  
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

| Chỉ số | CMP 30HX (TU116 Mod x16) |
|---|---|
| **Hiệu năng Tính toán** | FP32/Tensor mặc định theo VBIOS gốc |
| **Băng thông PCIe** | Gen2 ×16 (**~6.3 – 6.4 GB/s** AIDA64 sau khi chỉnh MRRS 512B) |
| **Driver NVIDIA** | Hoạt động bình thường, không cần GSP |

---

## <img src="https://api.iconify.design/lucide/folder-tree.svg?color=%230284c7" width="22" height="22" align="center" /> Cấu Trúc Kho Lưu Trữ

Kho lưu trữ này chứa toàn bộ mã nguồn công cụ mở khoá **CMP 30HX Windows Unlock v3.0.0** (Go).

- **Sử dụng trực tiếp:** Tải tệp thực thi tại [`release/`](release/) bao gồm `40HXInstaller.exe`, `40HXUninstaller.exe`, `40HXCheck.exe`.

**Biên dịch từ mã nguồn** (Yêu cầu Go 1.20+, chạy trong thư mục `tools/`):

```bat
cd tools\inst40hx     && go build -a -trimpath -ldflags="-H=windowsgui -s -w" -o ..\release\40HXInstaller.exe .
cd ..\uninstall40x    && go build -a -trimpath -ldflags="-H=windowsgui -s -w" -o ..\release\40HXUninstaller.exe .
cd ..\check40x        && go build -a -trimpath -ldflags="-H=windowsgui -s -w" -o ..\release\40HXCheck.exe .
```

---

## <img src="https://api.iconify.design/lucide/microchip.svg?color=%238b5cf6" width="22" height="22" align="center" /> 1. Bản Chất Kỹ Thuật CMP 30HX (TU116)

> [!NOTE]
> CMP 30HX (mã GPU `10DE:2189`, ví dụ Gigabyte GV-N30HXD6-6G) sau khi đã hàn trở mod vật lý các lane PCIe để nhận x16 Gen1, sử dụng công cụ này để nâng băng thông lên **PCIe Gen2 x16** (~6.4 GB/s).

### 1.1 Bản chất kỹ thuật: Tại sao chỉ lên được Gen2 mà không lên được Gen3?

> [!IMPORTANT]
> - **Gen2 (5.0 GT/s)**: Bị khoá mềm bởi thanh ghi bóng (VBIOS shadow registers). Công cụ can thiệp qua BAR0 MMIO để ghi đè vector tốc độ link và huấn luyện lại link thành công 100%.
> - **Gen3 (8.0 GT/s)**: Bị **đứt eFuse ở cấp độ chip silicon (Physical Silicon eFuse Blown)** do NVIDIA cấu hình khi xuất xưởng:
>   - Ghi `0x0E` vào `LNKCAP2` (`0x0880A4`) $\rightarrow$ đọc ngược lại chỉ trả về `0x00000006` (Bit 3 bị ngắt vật lý).
>   - Ghi `0x00010003` vào `LNKCTL2` (`0x0880A8`) $\rightarrow$ đọc ngược lại chỉ trả về `0x00010002` (Target Link Speed bị kẹp ở Gen2).
>   - Xung nhịp PHY Lane 0 (`0x08C4B0`) cố định ở 5.0 GHz (`0x50000000`), không thể nâng lên 8.0 GHz.
> - Do đó, **Gen2 x16 là giới hạn vật lý tối đa của CMP 30HX**. Không cố ép Gen3 để tránh lỗi vòng lặp retrain.

### 1.2 Tinh chỉnh DEVCTL MRRS (Mở khoá toàn bộ băng thông DMA)
Mặc định `DEVCTL` (`cap + 0x08`) có Max Read Request Size (MRRS) đặt là 128 Bytes, gây phân mảnh gói tin TLP khiến tốc độ AIDA64 Memory Copy bị nghẽn ở ~2.5 GB/s (như Gen1).  
Công cụ tự động nâng MRRS lên **512 Bytes** (`0x2000`) và khởi động lại `NVDisplay.ContainerLocalSystem`, giúp băng thông đạt tối đa **~6.3 – 6.4 GB/s** (đạt ~98% lý thuyết của Gen2 x16).

---

## <img src="https://api.iconify.design/lucide/package.svg?color=%23f59e0b" width="22" height="22" align="center" /> 2. Thành Phần Trong Gói Phát Hành

| Tệp | Mục đích |
|---|---|
| `Setup_CMP30HX.bat` | **Script AIO tự động 1-chạm** (Mở khoá Gen2 ngay + Đăng ký Scheduled Task khi Logon) |
| `40HXInstaller.exe` | **Giao diện cài đặt và quản lý** (Mặc định mở GUI; hỗ trợ tham số dòng lệnh `-gen2-30hx`) |
| `40HXUninstaller.exe` | **Gỡ cài đặt tự động** (Nhấp đúp $\rightarrow$ Yêu cầu quyền Administrator) |
| `40HXCheck.exe` | **Chẩn đoán độc lập** (Kiểm tra tốc độ link PCIe Gen2; tự thu hồi driver sau khi đo) |
| `WinRing0x64.sys` | Driver truy cập PCI Configuration Space và MMIO |

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 3. Cài Đặt Và Sử Dụng

CMP 30HX chạy hoàn toàn trên môi trường Windows thông qua ghi đè thanh ghi BAR0 MMIO:
- **KHÔNG cần chỉnh sửa BIOS**: Không cần tắt Secure Boot, không cần bật CSM, không bắt buộc Above 4G.
- **KHÔNG cần nạp firmware EFI**: Không can thiệp bootloader hay firmware.
- **KHÔNG cần GSP Firmware**: Kiến trúc TU116 không hỗ trợ và không cần GSP.

### 3.1 Cài đặt tự động 1-chạm (Khuyến nghị)

Nhấp đúp chuột vào tệp **`Setup_CMP30HX.bat`** (script tự động xin quyền Administrator nếu cần):
1. Tự động kiểm tra và dọn dẹp các Scheduled Task cũ.
2. Tự động đăng ký tác vụ **`CMP30HX_Gen2_Unlock`** chạy ngầm khi đăng nhập Windows (`-silent`).
3. Kích hoạt mở khoá **PCIe Gen2 x16** và tối ưu **MRRS 512B** ngay lập tức.

> *Muốn gỡ bỏ tự động khởi động: Chạy `Setup_CMP30HX.bat -uninstall`.*

---

### 3.2 Cài đặt thủ công bằng dòng lệnh

Nếu muốn tự cấu hình từng bước:

1. **Chuẩn bị môi trường & Driver**:
   - Đảm bảo card CMP 30HX đã được mod hàn trở lane vật lý x16 và nhận diện ổn định trong Device Manager (thường ở tốc độ mặc định Gen1 x16).
   - Cài đặt driver NVIDIA tương thích (khuyến nghị bản 537.58 hoặc driver desktop mod).

2. **Chạy mở khoá Gen2 ngay lần đầu (Không cần khởi động lại)**:
   - Nhấp chuột phải vào nút Start menu $\rightarrow$ Chọn **Terminal (Admin)** hoặc **Command Prompt (Administrator)**.
   - Chạy lệnh kích hoạt trực tiếp:
     ```cmd
     "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe" -gen2-30hx
     ```
   - Công cụ sẽ mở driver `WinRing0x64.sys`, ghi đè các thanh ghi bóng BAR0 (`0x08841C`, `0x08872C`, `0x08C040`, `0x08C2C0`), nâng MRRS lên 512B và gửi tín hiệu retrain. Thông báo `[Gen2-30HX] ✅ Gen2 Thành công: Link hiện tại Gen2 x16` xuất hiện.

3. **Thiết lập tự động mở khoá khi đăng nhập Windows**:
   - Do phần cứng GPU trở về trạng thái gốc Gen1 sau mỗi lần khởi động lại máy tính (cold boot / reboot), bạn chỉ cần tạo một Scheduled Task để tự động kích hoạt khi đăng nhập tài khoản:
     ```cmd
     schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f
     ```
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

### Các tham số dòng lệnh hữu ích
```cmd
40HXInstaller.exe -gen2-30hx            # Kích hoạt Gen2 cho CMP 30HX (chuẩn TU116 và tối ưu MRRS 512B)
40HXInstaller.exe -gen2-30hx -silent    # Chạy ngầm tự động (phù hợp cho Task Scheduler)
40HXInstaller.exe -status               # Kiểm tra toàn diện trạng thái phần cứng, driver và PCIe
40HXInstaller.exe -uninstall            # Gỡ bỏ sạch sẽ toàn bộ dịch vụ và tác vụ
```

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 4. Kiểm Tra Và Xác Nhận (Bằng 40HXCheck.exe)

Sau khi vào Windows, nhấp đúp vào **`40HXCheck.exe`**:
- `WinRing0`: Báo `✓` (Driver cấu hình đã kích hoạt).
- `PCIe`: Báo `Gen2 x16`.
- Mở **AIDA64 GPGPU Benchmark** kiểm tra **Memory Read / Memory Copy**: Đạt khoảng **6300 – 6400 MB/s** chứng tỏ MRRS 512B đã kích hoạt thành công (nếu chỉ đạt ~2500 MB/s là đang chạy ở cấu hình 128B).

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 5. Xử Lý Sự Cố Thường Gặp

| Hiện tượng | Nguyên nhân | Hướng khắc phục |
|---|---|---|
| **Khởi động lại bị về Gen1** | Chưa đăng ký tác vụ tự khởi động khi đăng nhập | Tạo Scheduled Task tự động kích hoạt `40HXInstaller.exe -gen2-30hx -silent` khi logon theo mục 3. |
| **GPU-Z báo Gen2 nhưng AIDA64 chỉ đạt 2.5 GB/s** | DEVCTL MRRS bị kẹp ở 128B mặc định | Chạy `40HXInstaller.exe -gen2-30hx` để kích hoạt tối ưu MRRS 512B và nạp lại hàng đợi DMA. |
| **Bị treo ở vòng lặp tác vụ 40HXGen2Retry** | Do phiên bản cũ ép cờ Gen3 trên CMP 30HX | Chạy `schtasks /delete /tn "40HXGen2Retry" /f` và cập nhật bản `40HXInstaller.exe` mới nhất đã kẹp cứng Gen2. |
| **Không nhận diện được GPU** | Chưa cắm chắc card hoặc thiếu driver NVIDIA | Cài đặt bản driver tương thích (khuyến nghị dòng 537.58 hoặc 55x/616.x) và kiểm tra Device Manager. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 6. Gỡ Cài Đặt Hoàn Toàn

Để đưa hệ thống về trạng thái nguyên bản:
1. Nhấp đúp vào **`40HXUninstaller.exe`** với quyền Administrator (hoặc chạy `40HXInstaller.exe -uninstall`).
2. Chương trình sẽ tự động xoá:
   - Các tác vụ lịch trình Scheduled Tasks (`CMP30HX_Gen2_Unlock`, `40HXGen2Retry`, v.v.).
   - Khoá tự khởi động Run trong Registry.
   - Driver dịch vụ và tệp tạm trong `%ProgramData%\40HXUnlock`.
3. Khởi động lại máy tính.

---

## <img src="https://api.iconify.design/lucide/shield-alert.svg?color=%23ef4444" width="22" height="22" align="center" /> 7. Quy Tắc An Toàn Phần Cứng Cốt Lõi

1. **Không nạp firmware 40HX lên 30HX**: Tuyệt đối không sao chép `40HXUNLK.EFI` hay blob GA102/TU106 lên CMP 30HX. Cấu trúc VBIOS và bộ điều khiển hoàn toàn khác biệt.
2. **Không ép Gen3 trên CMP 30HX**: eFuse đã đứt vật lý, mọi thao tác ép Gen3 đều vô hiệu và kích hoạt vòng lặp lỗi retrain.
3. **Không gọi Reset cứng**: Không kích hoạt `gen2RootLinkDisable`, `gen2HardFallback` hoặc PnP device restart trên CMP 30HX.
4. **Nguyên tắc Fail-closed**: Luôn kiểm tra tính sẵn sàng của PCIe Capability và readback an toàn trước khi thực hiện bất kỳ lệnh ghi nào vào PCI Config hoặc MMIO.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời Cảm Ơn (Acknowledgments)

> This project is inspired by **CMP40HX-Unlock**. If you find it useful, please consider giving a star to both the original author and this repository.
