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
| **Giới hạn phần cứng (eFuse)** | Bị cắt eFuse Gen3 (chỉ đạt tối đa Gen2 x16) | Hỗ trợ Gen2 x16 / Gen3 x4 tuỳ cấu hình |
| **Cơ chế mở khoá PCIe** | BAR0 MMIO runtime (Windows / Linux) | EFI Loader (`40HXUNLK.EFI`) hoặc runtime MMIO |
| **Tensor Core Compute** | Không có trên kiến trúc TU116 | Có hỗ trợ (~50 TFLOPS khi mở qua EFI) |
| **Riot Vanguard / Secure Boot** | Mở khoá trực tiếp trong OS, Secure Boot bật nguyên bản | Ký số EFI với `UnlockRiotGame.exe` nạp vào Secure Boot `db` |
| **DirectX Global Router** | Tự động điều hướng sang card CMP | Tự động điều hướng sang card CMP |

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

### ⭐ Cách nhanh nhất: Giao diện Web Control Center (Khuyên dùng)

> **Chỉ 1 thao tác**: Nhấp đúp vào `Launch_WebUI.bat`.

Trình duyệt sẽ tự động mở trang điều khiển cục bộ (`http://127.0.0.1:8080`) với đầy đủ bảng ma trận 16-lane PCIe thời gian thực, thông số xung nhịp GPU, nhiệt độ và nút bấm **1-Chạm Mở Khoá Gen2**.

### Cách 2: Script tự động trọn gói (All-In-One)

> Nhấp chuột phải vào `Setup_CMP30HX_WindowsAIO.bat` -> **Run as administrator**.

Script sẽ tự động thực hiện toàn diện các bước:
1. **Tắt PCIe ASPM và Hybrid Sleep**: Ngăn hệ điều hành Windows hạ tốc độ link PCIe khi GPU ở trạng thái chờ (idle).
2. **Tắt Fast Startup (hiberboot)**: Tránh hiện tượng kẹt link Gen1 sau khi khởi động lại máy.
3. **Tắt Vulnerable Driver Blocklist**: Cho phép hệ điều hành nạp driver kernel an toàn trong tích tắc.
4. **Tắt Memory Integrity (HVCI / Core Isolation)**: Cho phép nạp thanh ghi BAR0 MMIO (nếu đang bật, script sẽ tắt và yêu cầu bạn khởi động lại máy 1 lần).
5. **Đăng ký cơ chế Resident Guard (Thường trú)**: Tạo Scheduled Task đa tầng (Startup delay 45s + Logon delay 10s + Wake from Sleep) và Registry Run Key, tự động bù tốc nếu card bị vBIOS reset về Gen1.
6. **Mở khoá Gen2 x16 + MRRS 512B**: Kích hoạt toàn bộ băng thông DMA ngay lập tức.

---

## <img src="https://api.iconify.design/lucide/gamepad-2.svg?color=%23ec4899" width="22" height="22" align="center" /> 3. Tối ưu chơi game Riot & Game 3D (UnlockRiotGame)

Nếu bạn chơi **Liên Minh Huyền Thoại (LoL), Đấu Trường Chân Lý (TFT), Valorant** hoặc các tựa game đồ hoạ khác:

Chạy ứng dụng `windows-v3.0\release\UnlockRiotGame.exe` và bấm **[⚡ 1-CHẠM Tối Ưu Hóa]** (hoặc chạy lệnh `UnlockRiotGame.exe -silent -optimize`).

Công cụ sẽ tự động xử lý trong 2 giây:
- **Tự nhận diện GPU xuất hình & GPU render**: Quét màn hình đang cắm vào card nào để gán làm Display Host (`EnableMsHybrid=2`), đồng thời gán CMP làm Render 3D dGPU (`EnableMsHybrid=1`, `CASO=1`, `FeatureScore=0xD1`).
- **DirectX Global Router**: Đăng ký Hardware ID của CMP vào `DirectXUserGlobalSettings` (`HighPerfAdapter`). Toàn bộ game 3D sẽ tự động chạy trên card CMP mà bạn không cần phải add thủ công từng file exe trong Windows Graphics Settings.
- **Khắc phục triệt để lỗi "Không thể khởi động thiết bị đồ họa"**: Quét thư mục cài đặt LoL (Riot, VNG) và tự động cấu hình `game.cfg` sang chuẩn `WindowMode=2`, `BorderlessWindow=1`, `PreferDX9LegacyMode=0`, `DX11Support=1`.
- **Vanguard Unblocker cho Driver 5xx/6xx**: Vô hiệu hoá VulnerableDriverBlocklist, tự import chứng chỉ driver mod vào Windows Root Certificate Store, và dọn dẹp sạch sẽ các driver tạm để giữ hệ thống sạch với Vanguard (`testsigning` luôn tắt).

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 4. Cài đặt trên Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

Script Linux tự động xử lý: tắt ASPM, quét toàn bộ card CMP 30HX/40HX trong máy, ghi thanh ghi BAR0 MMIO, nâng MRRS lên 512B, retrain link Gen2, và cài đặt systemd service + hook sleep để duy trì sau mỗi lần bật máy hoặc resume từ chế độ ngủ.

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
| **Kẹt ở Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | - Hệ thống GPU kép (CMP 30HX + iGPU AMD/Intel) hoặc driver mod (RainCandy) khởi tạo lại, tự động reset GPU TLS về Gen1.<br>- Hoặc Windows bật Memory Integrity (HVCI), hoặc cáp Riser tiếp xúc kém | 1. **Tự động**: `Setup_CMP30HX_WindowsAIO.bat` bản mới kích hoạt chế độ **Thường trú (Resident Guard)** và tác vụ Logon (delay 10s) tự bù tốc Gen2.<br>2. **Thủ công**: Giữ card ở trạng thái **Enable** trong Device Manager (tuyệt đối **KHÔNG** Disable vì sẽ ngắt kết nối BAR0 MMIO) -> Chạy `Setup_CMP30HX_WindowsAIO.bat` hoặc `40HXInstaller.exe -gen2-30hx` khi desktop đã ổn định.<br>3. Nếu HVCI đang bật: Tắt HVCI và **Khởi động lại máy tính (Reboot)**. |
| **Mất kích hoạt Gen2 sau khi khởi động lại máy** | - Driver NVIDIA/RainCandy nạp trễ trên hệ thống APU hoặc ghi đè trạng thái vBIOS sau khi khởi động.<br>- Xung đột cấu hình driver cũ hoặc Fast Startup còn bật | 1. `Setup_CMP30HX_WindowsAIO.bat` đã tích hợp Scheduled Task đa tầng (Logon delay 10s + Startup delay 45s) kèm chế độ **DriverStrategy=2 (Resident Guard)** tự động kiểm tra mỗi phút và retrain lại khi mất Gen2.<br>2. **Khuyến cáo DDU**: Dùng **DDU (Display Driver Uninstaller)** trong chế độ Safe Mode gỡ sạch toàn bộ driver hiển thị cũ trước khi cài lại driver mod để tránh lỗi xung đột registry. |
| **Bị tụt về Gen1 x16 khi card ở chế độ rảnh (Idle)** | Tính năng tiết kiệm điện PCIe ASPM của Windows đang bật | Chạy lại `Setup_CMP30HX_WindowsAIO.bat` (script tự động tắt ASPM) hoặc chỉnh trong Power Options -> PCI Express -> Link State Power Management: **Off**. |
| **GPU-Z báo Gen2 x16 nhưng AIDA64 chỉ đạt ~2.5 GB/s** | Giá trị Max Read Request Size (MRRS) của card bị kẹp ở 128 Bytes mặc định | Chạy lệnh `40HXInstaller.exe -gen2-30hx` để nâng MRRS lên 512 Bytes và nạp lại hàng đợi DMA. |
| **Không nhận diện được GPU / Mã lỗi 43** | Mối hàn trở mod lane x16 chưa tiếp xúc tốt hoặc card chưa nhận driver | 1. Kiểm tra lại mối hàn trở trên card.<br>2. Chạy `Fix_CMP30HX_CSM_Error43.bat` nếu bo mạch chủ đang bật chế độ CSM.<br>3. Cài lại driver NVIDIA (dùng DDU gỡ sạch driver cũ rồi cài bản mới nhất). |
| **Đã thử mọi cách fix vẫn không được** | Driver NVIDIA bị xung đột cấu hình, profile registry lưu đè hoặc service driver lỗi | Gỡ sạch driver cũ bằng **DDU (Display Driver Uninstaller)** trong Safe Mode rồi tiến hành cài đặt lại driver. |
| **Lỗi "Không thể khởi động thiết bị đồ họa" / màn hình đen khi vào LMHT (LoL)** | Chế độ Fullscreen độc quyền bị xung đột trên hệ thống card đồ họa không có cổng xuất hình | Mở `windows-v3.0\release\UnlockRiotGame.exe` bấm **[⚡ 1-CHẠM Tối Ưu Hóa]** (hoặc chạy lệnh `UnlockRiotGame.exe -silent -optimize`) để công cụ tự ép `WindowMode=2`, `BorderlessWindow=1`, `PreferDX9LegacyMode=0`, `DX11Support=1` vào file cấu hình `game.cfg` của LoL. |
| **Game 3D không nhận card CMP, chỉ chạy bằng card onboard (iGPU)** | Windows chưa cấu hình định tuyến đồ họa toàn cục cho ứng dụng DirectX | Chạy `UnlockRiotGame.exe` bấm **[⚡ 1-CHẠM Tối Ưu Hóa]**. Công cụ sẽ tự gán Hardware ID của CMP vào `DirectXUserGlobalSettings` (`HighPerfAdapter`), tự động ép tất cả game 3D chạy trên card CMP mà không cần thêm thủ công từng game trong Graphics Settings. |
| **Vanguard hoặc Windows chặn nạp driver mod 5xx/6xx / Báo lỗi chứng chỉ** | Tính năng Microsoft Vulnerable Driver Blocklist kích hoạt hoặc thiếu chứng chỉ mod driver | Chạy `UnlockRiotGame.exe` để tự động tắt `VulnerableDriverBlocklistEnable=0`, nạp chứng chỉ mod driver (.cer) vào Windows Root Certificate Store và dọn dẹp sạch sẽ các driver xung đột. Tuyệt đối không bật `testsigning` vì Vanguard sẽ chặn 100%. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. Gỡ cài đặt

| Hệ điều hành | Phương thức gỡ |
|---|---|
| **Windows** (script) | Chạy lệnh `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows** (giao diện) | Nhấp chuột phải vào `windows-v3.0\release\40HXUninstaller.exe` -> **Run as administrator** |
| **Linux** | Chạy lệnh `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

Quá trình gỡ cài đặt sẽ tự động xoá toàn bộ Scheduled Task, Registry Run Key, thư mục chương trình và khôi phục cài đặt hệ thống về trạng thái ban đầu.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Lời cảm ơn

Dự án lấy cảm hứng từ các công trình nghiên cứu ban đầu của cộng đồng **CMP40HX-Unlock**. Nếu công cụ này hữu ích cho bạn, hãy nhấn **Star** trên kho lưu trữ GitHub để ủng hộ tác giả nhé!
