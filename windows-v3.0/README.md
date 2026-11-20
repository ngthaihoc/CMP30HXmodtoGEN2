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

### 0.1 Mục tiêu PCIe của CMP 30HX

Công cụ đặt mục tiêu **Gen2 x16 (5,0 GT/s)** cho CMP 30HX. Trên mẫu TU116 đã thử, `LNKCAP2` chỉ quảng bá Gen1 và Gen2, còn `LNKCTL2` đọc lại vẫn báo mục tiêu Gen2 sau yêu cầu Gen3. Kết quả trên mẫu này chưa xác định được nguyên nhân phần cứng. Đối chiếu tốc độ đích, tốc độ hiện tại và số lane khi GPU có tải.

### 0.2 Tối ưu DEVCTL MRRS
Công cụ nâng kích thước yêu cầu đọc tối đa (MRRS) lên 512 byte khi giá trị hiện tại thấp hơn. Kiểm tra giá trị MRRS trong chẩn đoán và đo băng thông dưới tải trên hệ thống của bạn.

### 0.3 Đăng ký tác vụ khởi động cho CMP 30HX

Dùng `Setup_CMP30HX_WindowsAIO.bat` ở thư mục gốc để cài đặt. Sau khi cấu hình Gen2 thành công, bộ cài tạo tác vụ `CMP30HX_Batch_Gen2_Unlock` chạy khi khởi động Windows và khi đăng nhập. Dùng tùy chọn `-cleantasks` của bộ cài để dọn các tác vụ và mục tự khởi động cũ mà bộ cài nhận diện.

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
> - **Driver**: Xác minh Windows nhận đúng CMP 30HX và xem nhật ký nếu driver truy cập phần cứng bị Windows chặn.

#### Các bước cài đặt chi tiết:

1. **Chuẩn bị môi trường & Driver**:
   - Đảm bảo card CMP 30HX đã được mod hàn trở lane vật lý x16 và nhận diện ổn định trong Device Manager (thường ở tốc độ mặc định Gen1 x16).
   - Kiểm tra trạng thái driver NVIDIA trong Device Manager.

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
   - Công cụ thử cấu hình PCIe Gen2 và MRRS 512 byte. Kiểm tra mã trạng thái và xác minh Gen2 x16 dưới tải.

3. **Thiết lập tự động mở khoá khi khởi động hoặc đăng nhập Windows**:
   - Dùng bộ cài batch ở mục 0.3 để đăng ký tác vụ. Sau khi khởi động lại, kiểm tra kết quả lần chạy gần nhất trong Task Scheduler.

4. **Dọn dẹp tác vụ cũ**:
   - Dùng `Setup_CMP30HX_WindowsAIO.bat -cleantasks` để dọn mục tự khởi động thuộc bộ cài.

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
UnlockRiotGame.exe -json-status # Xuất trạng thái GPU, Windows và Secure Boot dưới dạng JSON
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
| **Khởi động lại bị về Gen1** | Tác vụ tự khởi động chưa đăng ký hoặc chạy lỗi | Đối chiếu kết quả chạy tác vụ và đo lại tốc độ link dưới tải. Xem mục 0.3 cho 30HX và mục 3.1 cho 40HX. |
| **GPU-Z báo Gen2 nhưng AIDA64 chỉ đạt 2.5 GB/s** | DEVCTL MRRS bị kẹp ở 128B mặc định | Chạy `40HXInstaller.exe -gen2-30hx` để kích hoạt tối ưu MRRS 512B và nạp lại hàng đợi DMA. |
| **Tác vụ 40HXGen2Retry từ bản cũ lặp lại** | Tác vụ khởi động cũ vẫn còn đăng ký | Dùng tùy chọn `-cleantasks` của bộ cài batch nếu tác vụ thuộc bộ cài. |
| **Không nhận diện được GPU** | Chưa cắm chắc card hoặc thiếu driver NVIDIA | Cài đặt driver NVIDIA (hỗ trợ mọi phiên bản driver chính thức hoặc mod) và kiểm tra Device Manager. |
| **Màn hình đen sau khi mở khoá 40HX** | Chưa bật GSP Firmware (`EnableGpuFirmware=1`) | Chạy `40HXInstaller.exe -status` để kiểm tra cờ GSP và bật lại qua GUI. |

---

## <img src="https://api.iconify.design/lucide/cpu.svg?color=%2310b981" width="22" height="22" align="center" /> 6. Kiến Trúc Deep Module & Cơ Chế An Toàn (v3.0.0)

Phiên bản v3.0.0 áp dụng kiến trúc **Deep Module** với các tầng trừu tượng hoá mạnh mẽ:
- **`LinkNegotiator`** điều phối huấn luyện link PCIe với mục tiêu tối đa Gen2 theo profile TU116, tối ưu DEVCTL MRRS và cấu hình thanh ghi bóng MMIO.
- **`ComputeInspector`**: Kiểm tra an toàn `BOOT_0` (`0x16` cho TU106) chống crash hệ thống và giải mã định kiểu Tensor Core (`SS0 == 0x88888888`, `SS1 == 0x40966C`).
- **`HardwareBus` (Seam 1)**: Tách biệt hoàn toàn kernel driver (`WinRing0`, `ThrottleStop`) khỏi nghiệp vụ chính, kèm `MockHardwareBus` cho phép kiểm thử đơn vị độc lập.
- **`StatusContract` (Seam 2)**: Chuẩn hoá hợp đồng trạng thái định kiểu (`STATUS_CODE=GEN2_SUCCESS`, v.v.) giữa Go Engine và các script Batch.
- **`UnlockRiotGame.exe`** báo trạng thái GPU, driver và Secure Boot, đồng thời hỗ trợ cấu hình game. CMP 30HX dùng luồng PCIe trong hệ điều hành. Tính tương thích với Vanguard phụ thuộc vào driver và cấu hình Windows đã kiểm tra.
- **Bộ kiểm thử tự động toàn diện**: 13 Go unit tests (`40hxcore`), 18 Go unit tests (`unlockriot`), và 8 Mock test suites Linux (`Test_Mock_CMP30HX_Linux.sh`).

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 7. Gỡ Cài Đặt

- **CMP 30HX cài bằng batch**: Dùng `Setup_CMP30HX_WindowsAIO.bat -uninstall` để dọn tác vụ, mục tự khởi động cũ và các tệp của bộ cài.
- **CMP 40HX cài bằng 40HXInstaller**: Chạy `40HXUninstaller.exe` với quyền Administrator. Xem [phạm vi gỡ và hoàn nguyên](release/README.md#8-gỡ-cài-đặt-và-hoàn-nguyên).

Kiểm tra nhật ký gỡ và xác minh lại trạng thái sau khi khởi động lại.

---

## <img src="https://api.iconify.design/lucide/shield-alert.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. Quy Tắc An Toàn Phần Cứng Cốt Lõi

1. **Không nạp firmware 40HX lên 30HX**: Tuyệt đối không sao chép `40HXUNLK.EFI` hay blob GA102/TU106 lên CMP 30HX. Cấu trúc VBIOS và bộ điều khiển hoàn toàn khác biệt.
2. **CMP 30HX dùng mục tiêu Gen2**. Đối chiếu TLS và link dưới tải khi chẩn đoán.
3. **Giữ thiết bị hoạt động trong Device Manager**. Luồng 30HX tránh Link Disable, hard fallback và khởi động lại thiết bị qua PnP.
4. **Nguyên tắc Fail-closed**: Luôn kiểm tra tính sẵn sàng của PCIe Capability và readback an toàn trước khi thực hiện bất kỳ lệnh ghi nào vào PCI Config hoặc MMIO.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời Cảm Ơn (Acknowledgments)

> This project is inspired by **CMP40HX-Unlock**. If you find it useful, please consider giving a star to both the original author and this repository.
