<p align="right">
  <b>Ngôn ngữ:</b>
  <b>Tiếng Việt</b> |
  <a href="README_EN.md">English</a> |
  <a href="README_ZH.md">简体中文</a>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Mở Khoá NVIDIA CMP 30HX & 40HX v3.0.0 (PCIe Gen2 x16 & Riot Vanguard)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116%20%26%20TU106-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-T%E1%BA%AFt%20(An%20to%C3%A0n)-success)](#)
[![Web Control Center](https://img.shields.io/badge/Web%20UI-Localhost-blue?logo=html5&logoColor=white)](#-cach-nhanh-nhat-giao-dien-web-khuyen-dung)

Giải pháp mở khoá băng thông **PCIe Gen2 x16 (~6.4 GB/s)** và tối ưu hoá hiển thị đồ hoạ chuyên dụng cho dòng card **NVIDIA CMP 30HX (nhân TU116)** và **CMP 40HX (nhân TU106)** trên Windows 10/11 x64 và Linux.

- **Cài đặt 1-chạm**: Hỗ trợ Web Control Center trực quan hoặc script tự động hoàn toàn, chỉ cần nhấp đúp và chờ.
- **Tương thích mọi driver**: Hoạt động tốt với driver NVIDIA chính thức, driver desktop mod hoặc driver mới nhất (nhánh 5xx/6xx).
- **Tối ưu chơi game Riot & Anti-Cheat**: Tích hợp sẵn bộ điều hướng đồ hoạ DirectX Global Router, khắc phục lỗi khởi tạo màn hình trong LMHT, tương thích 100% với Riot Vanguard, EAC, BattlEye mà không cần bật Test Signing.
- **Không can thiệp BIOS**: Mở khoá an toàn qua BAR0 MMIO trong Windows/Linux, không cần flash VBIOS hay BIOS bo mạch chủ (đối với 30HX).
- **Mở khoá MRRS 512B**: Nâng Max Read Request Size lên 512 Bytes, tối đa hoá thông lượng DMA đạt trần ~6.4 GB/s.

> [!TIP]
> **Ủng hộ tác giả (Donate)**
>
> Dự án do tác giả phát triển từ thời sinh viên và chia sẻ hoàn toàn miễn phí cho cộng đồng. Nếu công cụ giúp ích cho bạn, hãy gửi một chút ủng hộ để tiếp thêm động lực phát triển nhé! Cảm ơn mọi người rất nhiều! ❤️
>
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
>
> - **Chủ tài khoản**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub Repository**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/cpu.svg?color=%2310b981" width="22" height="22" align="center" /> So sánh khả năng hỗ trợ phần cứng

| Tính năng | CMP 30HX (TU116) | CMP 40HX (TU106) |
|---|---|---|
| **Mục tiêu PCIe của công cụ** | Gen2 x16 | Gen2 x16 |
| **Cơ chế mở khoá PCIe** | BAR0 MMIO runtime (Windows / Linux) | EFI Loader (`40HXUNLK.EFI`) hoặc runtime MMIO |
| **Tensor Core Compute** | Không có trên kiến trúc TU116 | Có hỗ trợ (~50 TFLOPS khi mở qua EFI) |
| **Riot Vanguard / Secure Boot** | Mở khoá trực tiếp trong OS, Secure Boot bật nguyên bản | Ký số EFI với `UnlockRiotGame.exe` nạp vào Secure Boot `db` |
| **DirectX Global Router** | Tự động điều hướng sang card CMP | Tự động điều hướng sang card CMP |

Trên mẫu TU116 đã thử, tốc độ đích đọc lại vẫn là Gen2 sau yêu cầu Gen3. Nguyên nhân phần cứng của quan sát này chưa được xác định.

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. Chuẩn bị

**Yêu cầu phần cứng & hệ thống:**

- Card **CMP 30HX / 40HX** đã mod hàn trở nhận đủ lane PCIe x16, cắm vào khe PCIe x16 nối trực tiếp CPU.
- Cổng xuất hình cắm vào iGPU (CPU có card onboard) hoặc card đồ hoạ xuất hình phụ.
- Đã cài đặt driver NVIDIA (card hiển thị không bị dấu chấm than trong Device Manager).

**Tải bộ công cụ:**

Bấm **Code -> Download ZIP** trên GitHub hoặc tải bản phát hành trong mục Releases, sau đó giải nén ra thư mục cố định (ví dụ: `D:\CMP30HX-Unlock`).

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. Cài đặt trên Windows

### ⭐ Giao diện CMP Control Center

> **Chỉ 1 thao tác**: Nhấp đúp vào `Launch_WebUI.bat`.

Launcher ưu tiên mở ứng dụng desktop `UnlockAIO.exe` (Wails). Nếu thiếu tệp này, launcher dùng giao diện web cục bộ của `40HXInstaller.exe` và tự mở trình duyệt tại địa chỉ mà công cụ hiển thị.

### Cách 2: Script tự động trọn gói (All-In-One)

> Nhấp chuột phải vào `Setup_CMP30HX_WindowsAIO.bat` -> **Run as administrator**.

Script dành cho CMP 30HX (`10DE:2189`).
1. Kiểm tra card, các tệp cài đặt và driver cần thiết.
2. Sao chép tệp chạy, thử cấu hình PCIe Gen2 và đọc kết quả. Xác minh Gen2 x16 dưới tải.
3. Sau khi mở khoá đạt yêu cầu, đăng ký tác vụ khi khởi động Windows và khi đăng nhập. Bộ cài báo lỗi nếu driver bị chặn hoặc tác vụ không đăng ký được.

---

## <img src="https://api.iconify.design/lucide/gamepad-2.svg?color=%23ec4899" width="22" height="22" align="center" /> 3. Tối ưu chơi game Riot & Game 3D (UnlockRiotGame)

Nếu bạn chơi **Liên Minh Huyền Thoại (LoL), Đấu Trường Chân Lý (TFT), Valorant** hoặc các tựa game đồ hoạ khác:

Nút **[⚡ 1-CHẠM Tối Ưu Hóa]** trong `windows-v3.0\release\UnlockRiotGame.exe` có thể thay đổi cấu hình bảo mật Windows. Trên dòng lệnh, chế độ `-silent -optimize` yêu cầu xác nhận rõ ràng bằng `-confirm-riot` hoặc `-y`.

Các chức năng tối ưu bao gồm:
- **Tự nhận diện GPU xuất hình & GPU render**: Quét màn hình đang cắm vào card nào để gán làm Display Host (`EnableMsHybrid=2`), đồng thời gán CMP làm Render 3D dGPU (`EnableMsHybrid=1`, `CASO=1`, `FeatureScore=0xD1`).
- **DirectX Global Router**: Đăng ký Hardware ID của CMP vào `DirectXUserGlobalSettings` (`HighPerfAdapter`). Toàn bộ game 3D sẽ tự động chạy trên card CMP mà bạn không cần phải add thủ công từng file exe trong Windows Graphics Settings.
- **Khắc phục triệt để lỗi "Không thể khởi động thiết bị đồ họa"**: Quét thư mục cài đặt LoL (Riot, VNG) và tự động cấu hình `game.cfg` sang chuẩn `WindowMode=2`, `BorderlessWindow=1`, `PreferDX9LegacyMode=0`, `DX11Support=1`.
- **Thay đổi cấu hình bảo mật**: Tuỳ cấu hình hệ thống, quá trình tối ưu có thể thay đổi chính sách driver hoặc kho chứng chỉ Windows. Kết quả tương thích Vanguard phụ thuộc vào driver và môi trường đã kiểm tra.

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 4. Cài đặt trên Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

Script Linux dành cho CMP 30HX (`10DE:2189`). Công cụ cấu hình PCIe với mục tiêu Gen2 x16 và MRRS tối thiểu 512 byte, đồng thời có thể cài dịch vụ systemd và hook đánh thức. Xác minh tốc độ dưới tải sau khi khởi động lại và sau khi đánh thức máy.

| Lệnh | Chức năng |
|---|---|
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --status` | Xem trạng thái link hiện tại và thông số MRRS |
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` | Gỡ bỏ service tự kích hoạt và sleep hook |
| `./Setup_CMP30HX_LinuxAIO.sh -test --no-root` | Chạy thử nghiệm mô phỏng không cần card vật lý |

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 5. Kiểm tra kết quả

Sau khi cài đặt xong, bạn kiểm tra hiệu năng bằng 2 cách sau:

| Công cụ | Vị trí kiểm tra | Kết quả chuẩn |
|---|---|---|
| **GPU-Z** hoặc `40HXCheck.exe` | Mục **Bus Interface** | `PCIe x16 2.0 @ x16 2.0` |
| **AIDA64** -> Tools -> GPGPU Benchmark | Dòng **Memory Read / Memory Copy** | **6.3 - 6.4 GB/s** |

> [!NOTE]
> Nếu AIDA64 chỉ đạt ~2.5 GB/s dù GPU-Z báo Gen2 x16, điều đó có nghĩa là thông số MRRS đang bị kẹt ở mức mặc định 128 Bytes. Hãy chạy `windows-v3.0\release\40HXInstaller.exe -gen2-30hx` để kích hoạt lại MRRS 512B.

---

## <img src="https://api.iconify.design/lucide/wrench.svg?color=%23f59e0b" width="22" height="22" align="center" /> 6. Tiện ích bổ sung

Trong thư mục dự án có sẵn các công cụ tiện ích hỗ trợ người dùng:
- **`Fix_CMP30HX_CSM_Error43.bat`**: Sửa lỗi Code 43 và lỗi xung đột CSM khi bo mạch chủ không bật UEFI thuần.
- **`Setup_ReBar_CMP30HX.bat`**: Hỗ trợ cấu hình Resizable BAR cho card CMP, mở rộng kích thước bộ đệm VRAM giúp tăng khung hình trong game.

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 7. Xử lý sự cố thường gặp (Troubleshooting)

| Hiện tượng | Nguyên nhân | Hướng giải quyết |
|---|---|---|
| **Kẹt ở Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | Driver khởi tạo lại GPU, hệ điều hành chặn driver truy cập phần cứng hoặc khe cắm/riser có vấn đề | Giữ card ở trạng thái bật trong Device Manager. Dùng `Setup_CMP30HX_WindowsAIO.bat -preflight` để kiểm tra nhận diện card và xem thông tin chẩn đoán khi bộ cài báo lỗi driver. |
| **Mất kích hoạt Gen2 sau khi khởi động lại máy** | Tác vụ khởi động chạy lỗi, chưa đăng ký hoặc driver thay đổi trạng thái card | Kiểm tra `CMP30HX_Batch_Gen2_Unlock` trong Task Scheduler và đối chiếu kết quả chẩn đoán sau khi đăng nhập. |
| **Bị tụt về Gen1 x16 khi card ở chế độ rảnh (Idle)** | Cơ chế tiết kiệm điện có thể giảm tốc độ liên kết khi GPU nhàn rỗi | Đối chiếu tốc độ đích (TLS) và kiểm tra tốc độ, chiều rộng liên kết dưới tải. |
| **GPU-Z báo Gen2 x16 nhưng AIDA64 chỉ đạt ~2.5 GB/s** | Giá trị Max Read Request Size (MRRS) của card bị kẹp ở 128 Bytes mặc định | Chạy lệnh `40HXInstaller.exe -gen2-30hx` để nâng MRRS lên 512 Bytes và nạp lại hàng đợi DMA. |
| **Không nhận diện được GPU / Mã lỗi 43** | Mối hàn trở mod lane x16 chưa tiếp xúc tốt hoặc card chưa nhận driver | 1. Kiểm tra lại mối hàn trở trên card.<br>2. Chạy `Fix_CMP30HX_CSM_Error43.bat` nếu bo mạch chủ đang bật chế độ CSM.<br>3. Cài lại driver NVIDIA (dùng DDU gỡ sạch driver cũ rồi cài bản mới nhất). |
| **Đã thử mọi cách fix vẫn không được** | Driver NVIDIA bị xung đột cấu hình, profile registry lưu đè hoặc service driver lỗi | Gỡ sạch driver cũ bằng **DDU (Display Driver Uninstaller)** trong Safe Mode rồi tiến hành cài đặt lại driver. |
| **Lỗi “Không thể khởi động thiết bị đồ họa” / màn hình đen khi vào LMHT (LoL)** | Xung đột chế độ toàn màn hình độc quyền trên card không có cổng xuất hình | Dùng tính năng tối ưu ở phần 3 để cấu hình chế độ cửa sổ không viền và DirectX 11 trong `game.cfg`. |
| **Game 3D không nhận card CMP, chỉ chạy bằng card onboard (iGPU)** | Windows chưa cấu hình định tuyến đồ họa toàn cục cho ứng dụng DirectX | Chạy `UnlockRiotGame.exe` bấm **[⚡ 1-CHẠM Tối Ưu Hóa]**. Công cụ sẽ tự gán Hardware ID của CMP vào `DirectXUserGlobalSettings` (`HighPerfAdapter`), tự động ép tất cả game 3D chạy trên card CMP mà không cần thêm thủ công từng game trong Graphics Settings. |
| **Vanguard hoặc Windows chặn driver / báo lỗi chữ ký** | Chính sách bảo mật hoặc chữ ký driver chưa được Windows tin cậy | Kiểm tra trạng thái chữ ký driver và thông báo lỗi của Windows trước khi xác nhận thay đổi. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. Gỡ cài đặt

| Hệ điều hành | Phương thức gỡ |
|---|---|
| **Windows (CMP 30HX cài bằng batch)** | Chạy `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows (CMP 40HX cài bằng 40HXInstaller)** | Mở `windows-v3.0\release\40HXUninstaller.exe` với quyền Administrator |
| **Linux (CMP 30HX)** | Chạy `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

`-uninstall` của bộ cài batch dọn các tác vụ, mục tự khởi động cũ và tệp thuộc bộ cài. Dùng `-cleantasks` khi chỉ cần dọn tác vụ và mục tự khởi động do bộ cài nhận diện. Luồng Linux gỡ dịch vụ systemd, hook đánh thức và các tệp đã cài. Với bộ cài 40HX, xem [phạm vi gỡ và hoàn nguyên](windows-v3.0/release/README.md#8-gỡ-cài-đặt-và-hoàn-nguyên).

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời cảm ơn

Dự án lấy cảm hứng từ các công trình nghiên cứu ban đầu của cộng đồng **CMP40HX-Unlock**. Nếu công cụ này hữu ích cho bạn, hãy nhấn **Star** trên kho lưu trữ GitHub để ủng hộ tác giả nhé!
