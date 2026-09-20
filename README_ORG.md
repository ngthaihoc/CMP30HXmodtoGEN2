# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> NVIDIA CMP 40HX & CMP 30HX Windows Unlock

[![Platform](https://img.shields.io/badge/Platform-Windows%2010%20%7C%2011%20x64-0078D6?logo=windows&logoColor=white)](https://microsoft.com)
[![GPU](https://img.shields.io/badge/NVIDIA-TU106%20%7C%20TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![Go](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Status](https://img.shields.io/badge/Test%20Signing-Not%20Required-success)](#)

Bộ công cụ mã nguồn mở mở khoá toàn diện băng thông **PCIe Gen2 x16** và **Tensor Core** cho các dòng card chuyên dụng NVIDIA CMP (Crypto Mining Processor) trên hệ điều hành Windows.

- **Tự động hoá hoàn toàn**: Khởi chạy lúc đăng nhập hệ thống, tự động dọn dẹp driver sau khi hoàn thành.
- **Không cần Test Signing**: Giữ hệ thống nguyên bản, an toàn tuyệt đối cho phần mềm chống gian lận (Anti-Cheat) của các tựa game trực tuyến.
- **Tối ưu DMA**: Khắc phục nút thắt cổ chai TLP qua cấu hình Max Read Request Size (MRRS) 512 Bytes.

---

## <img src="https://api.iconify.design/lucide/gauge.svg?color=%230284c7" width="22" height="22" align="center" /> Kết Quả Đo Kiểm Thực Tế

| Thông số kiểm thử | CMP 40HX (TU106) | CMP 30HX (TU116 Mod x16) |
|---|---|---|
| **Hiệu năng Tensor / Tính toán** | `SS0=0x88888888` · FP16 HGEMM **~50 TFLOPS** (Gốc ~8T) | FP32 / CUDA Core theo VBIOS gốc |
| **Băng thông PCIe thực tế** | **Gen2 x16** (~6.4 GB/s lý thuyết) | **Gen2 x16** (**~6.3 – 6.4 GB/s** AIDA64 Memory Copy) |
| **Trạng thái Driver NVIDIA** | Hoạt động bình thường, không Code 43, GSP ON | Hoạt động bình thường, không cần GSP |
| **Cơ chế mở khoá** | EFI Firmware Bootloader + Windows Stage2 | WinRing0 MMIO Override + MRRS Tuning |

---

## <img src="https://api.iconify.design/lucide/microchip.svg?color=%238b5cf6" width="22" height="22" align="center" /> Cơ Chế Kỹ Thuật: CMP 30HX (TU116)

> [!IMPORTANT]
> **Giới hạn phần cứng eFuse tại Silicon Die (Physical Limit)**
> 
> CMP 30HX chỉ có thể mở khoá tối đa lên **PCIe Gen2 x16 (5.0 GT/s)**. Không thể đạt PCIe Gen3 (8.0 GT/s) hay Gen4 do NVIDIA đã thổi đứt cầu chì vật lý eFuse bit 3 (`LNKCAP2` bit 3) trong nhân TU116 khi xuất xưởng:
> - Ghi `0x0E` vào `LNKCAP2` (`0x0880A4`) $\rightarrow$ đọc ngược lại vẫn là `0x00000006` (Bit 3 bị ngắt vật lý).
> - Ghi TLS=Gen3 vào `LNKCTL2` (`0x0880A8`) $\rightarrow$ đọc ngược lại bị kẹp cứng ở `0x00010002` (Gen2).
> - Xung nhịp PHY Lane 0 (`0x08C4B0`) cố định ở 5.0 GHz (`0x50000000`).

### 1. Chuỗi can thiệp BAR0 MMIO mở khoá Gen2
Quy trình override thanh ghi bóng VBIOS trên TU116:
1. Ghi `0xE0B42D00` vào `PRIV_MISC_1` (`0x08841C`) để gỡ bảo vệ ghi (Write-Protect) của shadow register.
2. Ghi `0x00000006` vào `XVE_OVR` (`0x08872C`) nhằm công bố hỗ trợ Gen1 + Gen2.
3. Ghi `0x80085800` vào `LINK_CONFIG_0` (`0x08C040`) và `0x068731B3` vào `CYA_0` (`0x08C2C0`).
4. Kiểm tra xung PHY Lane 0 (`0x08C4B0`) chuyển từ 2.5 GHz sang 5.0 GHz.
5. Gửi tín hiệu retrain PCIe từ Root Port (CPU/Chipset) bằng cách đảo bit 5 trong `LNKCTL`.

### 2. Nút thắt cổ chai DEVCTL MRRS
Mặc định `DEVCTL` (`cap + 0x08`) đặt MRRS là 128 Bytes (`000b`), khiến các gói tin TLP của bộ nhớ bị phân mảnh nghiêm trọng, làm AIDA64 Memory Copy chỉ đạt ~2.5 GB/s (tương đương Gen1).
Công cụ tự động ghi đè MRRS lên **512 Bytes** (`0x2000`) và khởi động lại dịch vụ hiển thị `NVDisplay.ContainerLocalSystem`, giải phóng toàn bộ băng thông lên **~6.4 GB/s**.

---

## <img src="https://api.iconify.design/lucide/package.svg?color=%23f59e0b" width="22" height="22" align="center" /> Thành Phần Bản Dựng

Thư mục phát hành [`windows-v3.0/release/`](windows-v3.0/release/) bao gồm:

| Tệp thực thi | Chức năng |
|---|---|
| `40HXInstaller.exe` | **Bộ cài đặt & giao diện điều phối**: Hỗ trợ GUI trực quan hoặc chạy dòng lệnh nền qua Scheduled Task. |
| `40HXCheck.exe` | **Công cụ chẩn đoán độc lập**: Kiểm tra tốc độ link, thanh ghi SS0, trạng thái driver, tự dọn dẹp sau khi đo. |
| `40HXUninstaller.exe` | **Bộ gỡ cài đặt sạch sẽ**: Xoá bỏ hoàn toàn Scheduled Task, Run Key, driver service và mục khởi động EFI. |
| `OpenCL.exe` | **Kiểm thử hiệu năng tính toán**: Dành cho CMP 40HX để đo hashrate / FP16 TFLOPS. |
| `files/WinRing0x64.sys` | Kernel driver trung gian truy cập PCI Configuration Space và BAR0 MMIO. |
| `files/40HXUNLK.EFI` | Payload UEFI bootloader can thiệp SS0 cho CMP 40HX (TU106). |

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> Hướng Dẫn Sử Dụng

### Dành Cho CMP 30HX (Đã mod x16 Gen1)

> [!NOTE]
> CMP 30HX **không sử dụng** và **không được cài đặt** firmware EFI của 40HX. Chỉ cần cấu hình tác vụ Windows.

1. **Khởi chạy tức thì (Kiểm tra ngay)**:
   Mở Command Prompt (Administrator):
   ```cmd
   "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe" -gen2-30hx
   ```

2. **Cài đặt tự động kích hoạt khi đăng nhập Windows**:
   ```cmd
   schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f
   ```

3. **Huỷ bỏ vòng lặp thử lại cũ (nếu có)**:
   ```cmd
   schtasks /delete /tn "40HXGen2Retry" /f
   ```

---

### Dành Cho CMP 40HX (TU106)

> [!CAUTION]
> **Yêu cầu cấu hình BIOS (Bắt buộc)**:
> - **Above 4G Decoding**: `Enabled` (Bắt buộc bật, thiếu mục này EFI sẽ thất bại).
> - **Secure Boot**: `Disabled` (Tắt để nạp EFI không chứng thực).
> - **CSM**: `Disabled` (Chạy chế độ Pure UEFI).
> - **Fast Boot**: `Disabled`.
> - **Định dạng ổ đĩa**: Phải là chuẩn `GPT + UEFI` (dùng `mbr2gpt` nếu ổ đĩa còn ở chuẩn MBR).

1. Khởi chạy `40HXInstaller.exe` dưới quyền Administrator.
2. Tại khu vực cài đặt, chọn **Cài đặt toàn bộ** (GSP Firmware, EFI bootloader, Gen2 autostart, tối ưu nguồn điện).
3. Khởi động lại máy tính. Màn hình POST sẽ hiển thị thông báo nạp payload `40HX Unlock` trong 1-2 giây rồi vào Windows.

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> Kiểm Tra Và Xác Nhận

Khởi chạy `40HXCheck.exe`:
- **CMP 40HX**:
  - `SS0`: Báo `✓ Đầy đủ (SS0=0x88888888)` $\rightarrow$ Mở khoá tính toán thành công.
  - `PCIe`: Báo `Gen2 x16`.
- **CMP 30HX**:
  - `WinRing0`: Báo `✓`.
  - `PCIe`: Báo `Gen2 x16`.
  - Kiểm tra bằng **AIDA64 GPGPU Benchmark** mục **Memory Read / Memory Copy** đạt **~6300 – 6400 MB/s** chứng tỏ MRRS 512B đã hoạt động chuẩn xác.

---

## <img src="https://api.iconify.design/lucide/wrench.svg?color=%23ec4899" width="22" height="22" align="center" /> Biên Dịch Từ Mã Nguồn

Yêu cầu cài đặt Go 1.20 trở lên:

```powershell
# Biên dịch Installer
cd windows-v3.0/tools/inst40hx
go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXInstaller.exe" .

# Biên dịch Check Tool
cd ../check40x
go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXCheck.exe" .

# Biên dịch Uninstaller
cd ../uninstall40x
go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXUninstaller.exe" .
```

---

## <img src="https://api.iconify.design/lucide/shield-alert.svg?color=%23ef4444" width="22" height="22" align="center" /> Quy Tắc An Toàn Phần Cứng Cốt Lõi

1. **Không nạp firmware 40HX lên 30HX**: Tuyệt đối không sao chép `40HXUNLK.EFI` hay blob GA102/TU106 lên CMP 30HX. Cấu trúc VBIOS và bộ điều khiển hoàn toàn khác biệt.
2. **Không ép Gen3 trên CMP 30HX**: eFuse đã đứt vật lý, mọi thao tác ép Gen3 đều vô hiệu và kích hoạt vòng lặp lỗi retrain.
3. **Không gọi Reset cứng**: Không kích hoạt `gen2RootLinkDisable`, `gen2HardFallback` hoặc PnP device restart trên CMP 30HX.
4. **Nguyên tắc Fail-closed**: Luôn kiểm tra tính sẵn sàng của PCIe Capability và readback an toàn trước khi thực hiện bất kỳ lệnh ghi nào vào PCI Config hoặc MMIO.
