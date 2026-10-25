# BỐI CẢNH DỰ ÁN (PROJECT CONTEXT): CMP 40HX & CMP 30HX PCIE UNLOCK

> Trạng thái phiên hiện tại: [docs/agents/session-handoff.md](docs/agents/session-handoff.md), ghi nhận sau commit `31259cc` ngày 25/10/2026. Đọc `AGENTS.md` và đối chiếu Git trước khi tiếp tục.
>
> Phần dưới là ghi chú lịch sử. Các số đo, kết luận phần cứng và hướng dẫn cũ cần đối chiếu với mã hiện tại và bằng chứng kiểm chứng; chúng không mô tả kết quả của phiên triển khai mới nhất. Quyết định kiến trúc vừa triển khai nằm trong [spec dùng chung](docs/specs/shared-unlock-graphics-and-status.md).

Tài liệu này được tạo nhằm giúp các AI Agent khi bắt đầu phiên làm việc mới có thể nắm bắt ngay toàn bộ bối cảnh kỹ thuật, lịch sử phát triển, cấu trúc mã nguồn, các giới hạn phần cứng và công việc đang tiếp diễn mà không cần đọc lại hàng nghìn dòng nhật ký hội thoại.

---

## 1. TỔNG QUAN VÀ MỤC TIÊU DỰ ÁN

- **Mục tiêu cốt lõi**: Mở khoá băng thông PCIe cho các dòng card chuyên đào coin NVIDIA CMP (Crypto Mining Processor) chạy trên cả Windows (10/11 x64) và Linux (Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux) để sử dụng cho đồ hoạ, AI inference, tính toán và chơi game.
- **Phần cứng mục tiêu**:
  1. **NVIDIA CMP 40HX**:
     - GPU: `10DE:1F0B`, kiến trúc Turing TU106.
     - Giới hạn gốc: Bị khoá cứng VBIOS ở PCIe Gen1 x4 hoặc Gen2 x4; không hỗ trợ xuất hình trực tiếp.
     - Phương pháp mở khoá: Kết hợp EFI firmware bootloader (`40HXUNLK.EFI` / BOOTX64.EFI) can thiệp trước khi Windows nạp driver + Tool Windows kích hoạt Gen2 khi khởi động (`40HXInstaller.exe -gen2`).
  2. **NVIDIA CMP 30HX**:
     - GPU: `10DE:2189`, Subsystem `1458:408A` (Gigabyte `GV-N30HXD6-6G`), kiến trúc Turing TU116.
     - Tình trạng mod phần cứng: Đã câu trở/mod phần cứng thành công chạy ổn định PCIe x16 Gen1 (băng thông lý thuyết ~3.2 GB/s, FurMark và game AAA ổn định 1-2 tiếng).
     - Mục tiêu mod phần mềm: Mở khoá link PCIe lên **Gen2 x16** (~6.4 GB/s) và tinh chỉnh bộ điều khiển PCIe để đạt hiệu năng tối đa.

---

## 2. NHỮNG PHÁT HIỆN KỸ THUẬT QUAN TRỌNG

### 2.1. Phân biệt Silicon Die eFuse (Khoá cứng phần cứng) vs. Shadow Register (Khoá mềm VBIOS)
Trong quá trình thử nghiệm mở khoá cho CMP 30HX (TU116) và đối chiếu phân tích từ dự án mở rộng `cmpunlocker2` (trên Ampere GA100 / CMP 170HX và Turing TU106 / CMP 40HX):
- **Gen2 (5.0 GT/s)**:
  - Chỉ bị **khoá mềm (Software / VBIOS Shadow Lock)**.
  - Véc-tơ tốc độ link hỗ trợ (`LNKCAP2`, offset `0x0880A4`) có bit 2 (5.0 GT/s) vẫn nguyên vẹn (`0x06` = `0110b`).
  - Cho phép ghi đè thông qua thanh ghi BAR0 MMIO để nâng link lên Gen2 thành công 100%.
- **Gen3 (8.0 GT/s) và Gen4 (16.0 GT/s)**:
  - Bị **KHOÁ CỨNG TẠI SILICON DIE BẰNG EFUSE (Physical Silicon OTP eFuse Blown)** bởi chính NVIDIA từ nhà máy.
  - **Chứng cứ đối chiếu đa kiến trúc (Cross-Architecture Verification từ cmpunlocker2 & live dumps)**:
    * **Ampere GA100 (CMP 170HX)**: Tài liệu và mã nguồn kiểm thử BootROM (`GEN2_MECHANISM.md`, `pcie_gen3_unlock.sh`, `candidate_unlocks.py`) xác nhận bit eFuse `FUSE_PCIE_GEN23_DIS` đã bị bắn đứt vĩnh viễn trên đế silicon. Mọi lệnh ghi đè thanh ghi bóng (`XVE_OVR 0x8872C`, `LNKCTL2 0xA0`) dù đọc lại thấy giá trị Gen3/Gen4 nhưng tầng vật lý PHY link layer vẫn từ chối đàm phán trên 5.0 GT/s.
    * **Turing TU116 (CMP 30HX) & TU106 (CMP 40HX)**: Cùng chịu cơ chế bảo mật eFuse cấp fab tương tự.
  - **Chứng cứ thực nghiệm trên phần cứng sống CMP 30HX**:
    - Khi ghi ép giá trị `0x0E` (Gen1+Gen2+Gen3) vào thanh ghi `LNKCAP2` (`0x0880A4`), đọc ngược lại (readback) chỉ ra `0x00000006` (Bit 3 bị triệt tiêu ở mức phần cứng).
    - Khi ghi `0x00010003` (Target Link Speed = Gen3) vào `LNKCTL2` (`0x0880A8`), đọc ngược lại chỉ trả về `0x00010002` (Target Link Speed bị kẹp ở Gen2).
    - Thanh ghi xung nhịp PHY Lane 0 (`0x08C4B0`) cố định ở 5.0 GHz (`0x50000000`), không thể nâng lên 8.0 GHz.
  - **Kết luận bắt buộc**: **Các dòng CMP (30HX, 40HX, 170HX) hoàn toàn KHÔNG THỂ lên PCIe Gen3 hay Gen4 bằng phần mềm**. Giới hạn vật lý tối đa có thể đạt được là **PCIe Gen2 x16** (hoặc Gen2 x4/x8 tùy số lane mod). Bất kỳ nỗ lực ép Gen3 nào đều thất bại và gây lỗi vòng lặp retrain hoặc kẹt RmInit (`speed=15`).

### 2.2. Chuỗi thanh ghi BAR0 MMIO mở khoá Gen2 trên TU116 (CMP 30HX)
Quy trình mở khoá Gen2 cho CMP 30HX thông qua driver WinRing0 / ThrottleStop (Windows) hoặc Direct Sysfs (Linux):
1. Ghi `0xE0B42D00` vào `PRIV_MISC_1` (`0x08841C`) để tắt cơ chế chống ghi (Write-Protect) của shadow registers.
2. Ghi `0x00000006` vào `XVE_OVR` (`0x08872C`) để khai báo hỗ trợ Gen1 + Gen2 (5.0 GT/s).
3. Ghi `0x80085800` vào `LINK_CONFIG_0` (`0x08C040`).
4. Ghi `0x00240036` vào `PL_LINK_RATE` (`0x08C1C0`, xóa bit 20 để loại bỏ giới hạn công suất link rate của bộ điều khiển PCIe).
5. Ghi `0x068731B3` vào `CYA_0` (`0x08C2C0`).
6. Kiểm tra thanh ghi PHY Lane 0 (`0x08C4B0`): chuyển từ `0x25000000` (2.5 GHz - Gen1) sang `0x50000000` (5.0 GHz - Gen2).
7. Kích hoạt huấn luyện lại link (PCIe Retrain) thông qua Root Port (CPU / Chipset) bằng cách đảo bit 5 trong thanh ghi `LNKCTL` (`cap + 0x10`).

### 2.3. Tinh chỉnh DEVCTL MRRS (Nút thắt nghẽn băng thông DMA)
- **Vấn đề đã gặp**: Dù GPU-Z và HWiNFO64 báo link đã đạt `Gen2 x16`, nhưng khi chạy benchmark **AIDA64 GPGPU Memory Copy / Read**, tốc độ chỉ đạt ~2.5 GB/s (ngang Gen1).
- **Nguyên nhân cốt lõi**: Thanh ghi `DEVCTL` (`cap + 0x08`) có các bit [14:12] quy định Max Read Request Size (MRRS). Giá trị mặc định là 128 Bytes (`000b`), khiến các gói TLP của bộ nhớ phân mảnh cực kỳ nghiêm trọng khi đọc DMA.
- **Giải pháp**: Ghi cấu hình MRRS = 512 Bytes (`2 << 12 = 0x2000`) vào `DEVCTL`, sau đó khởi động lại service `NVDisplay.ContainerLocalSystem` để driver nạp lại hàng đợi DMA.
- **Kết quả**: Băng thông AIDA64 Memory Copy tăng vọt từ ~2.5 GB/s lên **~6.3 - 6.4 GB/s** (đạt ~98% hiệu suất lý thuyết của PCIe Gen2 x16).

### 2.4. Kiến trúc mở khoá trên Linux không cần nạp driver ngoài (Sysfs Direct MMIO)
- **Đặc tính Linux**: Khác với Windows cần driver kernel ký số (`WinRing0` / `ThrottleStop.sys`), Linux cho phép tiến trình có quyền `root` truy cập trực tiếp không gian BAR0 MMIO của GPU qua `/sys/bus/pci/devices/<BDF>/resource0`.
- **Cơ chế hoạt động của `Setup_CMP30HX_LinuxAIO.sh`**:
  1. Dùng Python 3 nhúng sẵn đọc kiểm tra `BOOT_0` (`0x00000000`) = `0x16xxxxxx` (xác thực chip TU116).
  2. Ghi trực tiếp chuỗi thanh ghi shadow: `PRIV_MISC_1` (`0x0008841C` = `0xE0B42D00`), `XVE_OVR` (`0x0008872C` = `0x00000006`), `LINK_CONFIG_0` (`0x0008C040` = `0x80085800`), `PL_LINK_RATE` (`0x0008C1C0` = `0x00240036`), `CYA_0` (`0x0008C2C0` = `0x068731B3`).
  3. Dùng `setpci` cấu hình `LNKCTL2` (`cap + 0x30`) đặt Target Link Speed = Gen2 trên cả GPU và Root Port.
  4. Tối ưu `DEVCTL` (`cap + 0x08`) nâng MRRS lên 512 Bytes (`0x2000`).
  5. Đảo bit 5 `LNKCTL` (`cap + 0x10`) luân phiên 6 lượt giữa Root Port và GPU để kích hoạt chu trình retrain.
  6. Tắt hoàn toàn PCIe ASPM (`/sys/module/pcie_aspm/parameters/policy` = `performance`) và Runtime PM (`power/control` = `on`).
  7. Tích hợp Systemd service (`cmp30hx-gen2-unlock.service`) và sleep hook (`/lib/systemd/system-sleep/cmp30hx-unlock`) đảm bảo Gen2 duy trì liên tục qua các lần khởi động lại hoặc sau khi máy sleep/resume.

### 2.5. Hiện tượng GPU LNKCTL2 chỉ đọc và cơ chế hạ tốc khi rảnh (Idle Power Saving)
- **Hành vi thanh ghi**: Trên TU116, thanh ghi `LNKCTL2` của GPU thường là chỉ đọc/không dính (`0x00000001`, TLS=1). Điều này là hoàn toàn bình thường trong thiết kế phần cứng của NVIDIA Turing; việc huấn luyện lại tốc độ link Gen2 phụ thuộc chủ yếu vào việc Root Port trên bo mạch chủ phát tín hiệu ép đàm phán (`Root LNKCTL2 TLS=2`).
- **Hiện tượng kẹt Gen1 khi rảnh (Idle)**:
  - Khi không có tải đồ hoạ/CUDA, GPU tự động kích hoạt tính năng tiết kiệm điện phần cứng để hạ xung nhịp PHY PCIe về Gen1 (2.5 GT/s).
  - Khi có tải 3D (Render test trong GPU-Z, FurMark, AIDA64), GPU sẽ tự động bung xung nhịp lên Gen2 x16 (5.0 GT/s, ~6.4 GB/s).
- **Các nguyên nhân thực tế khiến retrain Gen2 thất bại**:
  1. Cáp Riser PCIe chất lượng kém hoặc riser 1x không chịu được tần số cao 5.0 GHz (Gen2) làm suy hao tín hiệu vi sai, card tự động rớt về Gen1.
  2. Khe PCIe cắm qua Chipset bo mạch chủ có thiết lập BIOS bị gán cứng ở `Gen1`.
  3. Mối hàn trở mod x16 bị lem hoặc tiếp xúc không chuẩn các đường clock/data.
  4. Memory Integrity (HVCI) trên Windows đang bật chặn nạp driver can thiệp BAR0 MMIO.

### 2.6. Hướng dẫn đọc log chẩn đoán và cây quyết định xử lý sự cố (Log Diagnostics & Troubleshooting Tree)
Khi chạy `40HXInstaller.exe -gen2-30hx` hoặc `Setup_CMP30HX_WindowsAIO.bat`, file nhật ký sẽ ghi lại chi tiết:
- **Mẫu log điển hình**:
  ```text
  [Gen2-30HX] [MMIO] PRIV_MISC_1
  [Gen2-30HX] [MMIO] LINK_CONFIG_0 (0x08C040): Ghi 0x80085800 -> Đọc lại 0x80085800
  [Gen2-30HX] [MMIO] XVE_OVR=6_CONFIRM (0x08872C): Ghi 0x00000006 -> Đọc lại 0x00000006
  GPU LNKCTL2 là chỉ đọc/không dính (0x00000001, TLS=1; trong dự tính, phụ thuộc vào ROOT Port điều khiển huấn luyện lại)
  ROOT LNKCTL2 TLS=2 (0x0002 -> 0x0002)
  [Gen2-30HX] [DEVCTL] Hiện tại MPS=1 (256 bytes), MRRS=2 (512 bytes)
  [Gen2-30HX] Huan luyen lai link #1..#6 (phía ROOT) ...
  [Gen2-30HX] × Gen2 chưa đạt: Vẫn đang ở Gen1 x16 (GPU TLS=Gen1, Root TLS=Gen2)
  ```
- **Phân tích từng chỉ số**:
  1. `XVE_OVR=6_CONFIRM`: Ghi đè shadow register thành công, phần cứng đã mở khoá cho phép đàm phán Gen2.
  2. `GPU LNKCTL2 TLS=1` & `Root LNKCTL2 TLS=2`: Hoàn toàn bình thường trên TU116, CPU Root Port đóng vai trò chủ động kéo link lên 5.0 GT/s.
  3. `Vẫn đang ở Gen1 x16`: Tín hiệu vật lý (Signal Integrity) tại thời điểm retrain chưa đạt chuẩn 5.0 GHz hoặc GPU đang ở trạng thái ngủ (ASPM L0s/L1).
- **Cây quyết định xử lý (Triage Tree)**:
  - **Bước 1 (Kiểm tra tải đồ hoạ)**: Mở GPU-Z, bấm dấu `?` cạnh dòng Bus Interface và chọn **Start Render Test**. Nếu bus nhảy ngay sang `PCIe x16 2.0 @ x16 2.0`, card đã mở khoá thành công (chỉ hạ tốc khi rảnh để tiết kiệm điện).
  - **Bước 2 (Kiểm tra phần cứng & Riser)**: Nếu khi render test vẫn giữ `x16 1.1`, kiểm tra kết nối phần cứng:
    * Cắm trực tiếp card vào khe PCIe x16 số 1 (khe nối thẳng vào CPU), tháo bỏ toàn bộ cáp riser hoặc dây nối dài.
    * Vào BIOS bo mạch chủ, tìm mục **PCIe Link Speed** của khe cắm và đổi từ `Auto` hoặc `Gen1` sang `Gen2` / `Gen3`.
    * Dùng kính lúp hoặc kính hiển vi kiểm tra lại các mối hàn câu trở mod x16, vệ sinh sạch chân tiếp xúc PCIe bằng cồn Isopropyl (IPA).

### 2.7. Cơ chế xuất hình cho card không cổng xuất hình (Headless Display Routing / CASO)
- Do dòng CMP (30HX / 40HX) không có cổng xuất hình vật lý (DisplayPort/HDMI), việc xuất hình khi chơi game hoặc làm đồ hoạ được xử lý qua công nghệ **Cross-Adapter Scan-Out (CASO)** hoặc DWM copy engine của hệ điều hành:
  - **Trên Windows 10/11**:
    1. Cắm dây màn hình vào cổng xuất hình của bo mạch chủ (sử dụng iGPU Intel UHD / AMD Radeon) hoặc card phụ có cổng xuất.
    2. Cài đặt driver NVIDIA tiêu chuẩn (hoặc Studio Driver).
    3. Vào **Windows Settings** -> **System** -> **Display** -> **Graphics**: Chọn ứng dụng/game muốn chạy -> Nhấp **Options** -> Chọn **High performance** (gán cho NVIDIA CMP 30HX/40HX).
    4. GPU CMP sẽ chịu trách nhiệm render 3D và đẩy khung hình hoàn tất qua bus PCIe (DMA) về bộ nhớ iGPU để xuất ra màn hình. Nhờ đã mở khoá **Gen2 x16 + MRRS 512B** (~6.4 GB/s), độ trễ copy khung hình bị triệt tiêu hoàn toàn, không gây hiện tượng giật lag / drop FPS.
  - **Trên Linux**:
    - Dùng cơ chế **PRIME Render Offload**:
      ```bash
      __NV_PRIME_RENDER_OFFLOAD=1 __GLX_VENDOR_LIBRARY_NAME=nvidia %command%
      ```
    - Hoặc cấu hình Xorg server chạy màn hình bằng card chính / iGPU, còn tính toán CUDA/Vulkan chuyển giao cho card CMP.

### 2.8. Kiến trúc phục hồi đa tầng PCIe (2-Stage PCIe Recovery Architecture)
Đúc kết từ thực nghiệm trên `cmpunlocker2` và kiểm thử dòng Turing CMP:
- **Nguyên nhân kẹt link Gen1 hoặc lỗi đàm phán PHY**:
  1. Tiến trình đồ họa, CUDA runtime hoặc display driver holding open file descriptors (`/dev/nvidia*`, `/dev/dri/*`, hoặc Display Driver Container) sẽ giữ chặt DMA context, ngăn cản link retraining.
  2. Chu trình reset phần cứng hoặc tắt/bật PCIe quá gấp (< 500ms) không đủ thời gian xả tụ điện bên trong chip, khiến GPU state machine rơi vào trạng thái bất định (RmInit error, `speed=15`).
- **Mô hình phục hồi 2 tầng (2-Stage Recovery)**:
  - **Stage 1 (Direct MMIO + Retrain Pulses)**:
    * Ghi chuỗi MMIO BAR0 (`PRIV_MISC_1`, `XVE_OVR`, `LINK_CONFIG_0`, `PL_LINK_RATE`, `CYA_0`).
    * Thiết lập `LNKCTL2` và phát xung retrain qua Root Port / GPU.
  - **Stage 2 (Soft Reset Fallback)**:
    * Nếu Stage 1 chưa đạt Gen2 (do DMA context bị chiếm đóng), tự động kích hoạt Stage 2:
    * *Trên Linux (`Setup_CMP30HX_LinuxAIO.sh`)*: Giải phóng process chiếm GPU (`fuser -k /dev/nvidia* /dev/dri/*`), ngắt thiết bị khỏi bus PCI qua sysfs (`echo 1 > /sys/bus/pci/devices/<BDF>/remove`), chờ tối thiểu 2.0 giây xả tụ, rescan lại bus PCI (`echo 1 > /sys/bus/pci/rescan`), tái nạp MMIO và phát xung retrain 2.0s.
    * *Trên Windows (`Setup_CMP30HX_WindowsAIO.bat` & Go Core)*: PnP Restart Device / Disable-Enable PnP device, chờ tối thiểu 2.0 giây, khởi động lại service `NVDisplay.ContainerLocalSystem` để dọn sạch hàng đợi DMA, tái nạp MMIO và tối ưu lại MRRS 512B.

---

## 3. LỖI ĐÃ XỬ LÝ VÀ TRẠNG THÁI HIỆN TẠI

### 3.1. Lỗi vòng lặp vô hạn tác vụ `40HXGen2Retry`
- **Hiện tượng**: Khi chạy thử nghiệm cờ `-gen3-30hx` hoặc `-force-root-gen3`, biến `targetGen` bị gán bằng 3. Vì phần cứng kẹp ở Gen2, điều kiện `cur >= targetGen` (`2 >= 3`) trả về `false`. Chương trình báo `❌ Gen3 Chưa đạt (Not achieved)` và gọi `scheduleGen2Retry()`, tự tạo Scheduled Task chạy lại mỗi 1 phút vô tận.
- **Cách khắc phục**:
  1. Xoá tác vụ lặp trên máy: `schtasks /delete /tn "40HXGen2Retry" /f`.
  2. Trong mã nguồn (`40hxcore/const.go` và `inst40hx/main.go`), giới hạn `MaxSupportedGen` của CMP 30HX cố định là 2.
  3. Khi phát hiện `cur >= 2`, xác nhận thành công ngay (`ok = true`), gọi `deleteGen2Retry()` để huỷ lịch trình retry.

### 3.2. Tiến trình Việt hoá (Localization)
Dự án đã được chuyển đổi toàn bộ giao diện, thông báo, nhật ký sang tiếng Việt (100%):
- `windows-v3.0/tools/check40x/main.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/uninstall40x/main.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/40hxcore/uninstall_ops.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/inst40hx/gui.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/inst40hx/main.go`: Đã hoàn thiện toàn bộ chuỗi log và CLI sang tiếng Việt, khoá luồng 30HX cố định ở Gen2.
- `README.md` (Gốc & `windows-v3.0/README.md`): Đã hoàn thiện toàn diện bằng tiếng Việt kèm hình ảnh, sơ đồ và hướng dẫn cho cả Windows và Linux.

### 3.3. Tối ưu bộ cài All-In-One (AIO) & Cơ chế chạy ngầm Windows
- **Loại bỏ hoàn toàn tác vụ lặp `40HXGen2Retry`**: Tác vụ cũ lặp mỗi 1 phút đã bị khai tử vì dễ gây xung đột driver.
- **Thay thế bằng Scheduled Task SYSTEM với trễ 45 giây**:
  - Tạo tác vụ `CMP30HX_Gen2_Unlock` chạy dưới tài khoản `NT AUTHORITY\SYSTEM` lúc khởi động máy tính với độ trễ (delay) **45 giây**.
  - Độ trễ 45 giây giúp bảo đảm toàn bộ driver nền tảng của bo mạch chủ và card màn hình đã nạp hoàn tất trước khi công cụ can thiệp BAR0 MMIO.
  - Tự động tắt PCIe ASPM, Fast Startup (`HiberbootEnabled = 0`), Microsoft Vulnerable Driver Blocklist (`VulnerableDriverBlocklistEnable = 0`) và Memory Integrity (`HypervisorEnforcedCodeIntegrity\Enabled = 0`).
- **Chuẩn hoá định danh tệp AIO**:
  - Windows: `Setup_CMP30HX_WindowsAIO.bat`
  - Linux: `Setup_CMP30HX_LinuxAIO.sh`
- **Quản lý định dạng file Git (`.gitattributes`)**: Thiết lập `*.sh text eol=lf` và `*.bat text eol=crlf` để chống lỗi hỏng bash script khi checkout trên môi trường Windows.

### 3.4. Chuẩn mực thiết kế script phòng thủ (Defensive Scripting Standards)
Để đảm bảo script vận hành tin cậy trên hàng loạt hệ điều hành và môi trường người dùng:
- **Linux (`Setup_CMP30HX_LinuxAIO.sh`)**:
  - Áp dụng các nguyên tắc từ `wshobson/agents@bash-defensive-patterns`.
  - Nghiêm ngặt kiểm tra quyền root (`EUID == 0`), thiết lập `set -euo pipefail`.
  - Tự động phát hiện và kiểm tra các gói công cụ phụ thuộc (`lspci`, `setpci`, `python3`).
  - Dùng Python 3 với `struct.pack`/`unpack` xử lý nhị phân trực tiếp trên sysfs BAR0 MMIO, tránh dùng các tiện ích nhị phân bên ngoài thiếu tính di động.
  - Thiết lập bẫy lỗi (`trap`) dọn dẹp biến tạm và ghi log nhật ký tại `/var/log/cmp30hx_gen2.log`.
- **Windows (`Setup_CMP30HX_WindowsAIO.bat`)**:
  - Áp dụng các tiêu chuẩn từ `github/awesome-copilot@batch-files`.
  - Tự động phát hiện và leo thang đặc quyền Administrator thông qua PowerShell `Start-Process -Verb RunAs`.
  - Bao bọc toàn bộ biến đường dẫn và tham số bằng cặp dấu nháy kép `"..."` chống lỗi khoảng trắng.
  - Quản lý tác vụ ngầm an toàn bằng `schtasks` và PowerShell `New-ScheduledTask*` với tài khoản `NT AUTHORITY\SYSTEM` có độ trễ khởi động 45s.
  - Vô hiệu hoá hoàn toàn Fast Startup và driver blocklist để ngăn chặn Windows chặn nạp driver `WinRing0x64.sys`.

---

## 4. BẢN ĐỒ CẤU TRÚC DỰ ÁN

```
D:\ClodeGithub\CMP40HX-Unlock-main\
├── .gitattributes                               # Cố định EOL LF cho shell script và CRLF cho bat
├── CLAUDE.md                                    # Hướng dẫn nhanh cho Claude Code
├── PROJECT_CONTEXT.md                           # Tệp bối cảnh chi tiết này
├── README.md                                    # Tài liệu hướng dẫn chính (Việt hoá, Windows & Linux)
├── Setup_CMP30HX_WindowsAIO.bat                 # Bộ cài All-In-One tự động 1-chạm cho Windows
├── Setup_CMP30HX_LinuxAIO.sh                    # Bộ cài All-In-One tự động cho Linux (Ubuntu, HiveOS...)
├── windows-v3.0\
│   ├── 40HX_Gen2_Windows\                       # Script batch cũ dự phòng
│   │   ├── install_autostart.bat
│   │   ├── run_gen2.bat
│   │   └── uninstall.bat
│   ├── release\                                 # Thư mục chứa file nhị phân sau khi biên dịch
│   │   ├── 40HXInstaller.exe                    # Công cụ cài đặt / mở khoá Gen2 chính
│   │   ├── 40HXCheck.exe                        # Công cụ kiểm tra, chẩn đoán link & driver
│   │   ├── 40HXUninstaller.exe                  # Công cụ gỡ cài đặt sạch sẽ
│   │   └── files\
│   │       ├── 40HXUNLK.EFI                     # Payload EFI cho CMP 40HX (TU106)
│   │       └── WinRing0x64.sys                  # Driver đọc/ghi PCI config & MMIO
│   └── tools\                                   # Mã nguồn Go
│       ├── 40hxcore\                            # Thư viện dùng chung
│       │   ├── const.go                         # Khai báo cấu hình GPU profile (40HX, 30HX)
│       │   ├── state.go                         # Tìm BDF GPU, kiểm tra trạng thái
│       │   ├── regs.go                          # Đọc/ghi PCIe config space & BAR0 MMIO
│       │   ├── tsdrv.go                         # Giao tiếp driver ThrottleStop / WinRing0
│       │   ├── uninstall_ops.go                 # Thao tác gỡ cài đặt sạch sẽ
│       │   └── service.go / power.go / gsp.go   # Quản lý Windows Service, Power, GSP
│       ├── inst40hx\                            # Mã nguồn 40HXInstaller.exe
│       │   ├── main.go                          # CLI điều phối, logic Gen2/Gen3, scheduler
│       │   └── gui.go                           # Giao diện GUI người dùng (Walk-based)
│       ├── check40x\                            # Mã nguồn 40HXCheck.exe
│       │   └── main.go                          # Kiểm tra trạng thái GPU, PCIe, Driver
│       └── uninstall40x\                        # Mã nguồn 40HXUninstaller.exe
│           └── main.go                          # GUI/CLI gỡ cài đặt
```

---

## 5. CÁC NGUYÊN TẮC AN TOÀN BẮT BUỘC (CRITICAL BOUNDARIES)

1. **TUYỆT ĐỐI KHÔNG DÙNG FIRMWARE 40HX CHO 30HX**:
   - Không được nạp `40HXUNLK.EFI`, blob microcode GA102/TU106 lên CMP 30HX. VBIOS và chip bảo mật của TU116 khác biệt hoàn toàn, cố nạp sẽ gây brick card hoặc treo POST.
2. **KHÔNG ÉP GEN3 TRÊN CMP 30HX**:
   - Tuyệt đối không thêm logic ép Gen3 trên CMP 30HX do eFuse phần cứng đã bị đứt. Mọi cấu hình cho 30HX phải clamp cứng ở `targetGen = 2`.
3. **KHÔNG GỌI CÁC HÀM NGUY HIỂM TRÊN CMP 30HX**:
   - Không gọi `gen2RootLinkDisable`, `gen2HardFallback`, hoặc PnP disable/enable device reset trên CMP 30HX (chỉ dùng cho 40HX TU106 khi thật sự cần).
4. **FAIL-CLOSED TRƯỚC KHI GHI THANH GHI**:
   - Phải kiểm tra sự tồn tại của PCIe Capability và readback an toàn trước khi thực hiện bất kỳ lệnh ghi nào vào PCI config hoặc MMIO.

---

## 6. LỆNH VẬN HÀNH THƯỜNG DÙNG

### 6.1. Trên Windows (Setup_CMP30HX_WindowsAIO.bat)
- **Cài đặt tự động 1-chạm**:
  Nhấp chuột phải vào `Setup_CMP30HX_WindowsAIO.bat` chọn **Run as administrator**.
- **Kiểm tra trạng thái**:
  Chạy `windows-v3.0\release\40HXCheck.exe` hoặc mở **GPU-Z** bấm nút `?` (Render Test).
- **Gỡ bỏ cài đặt tự động**:
  ```cmd
  Setup_CMP30HX_WindowsAIO.bat -uninstall
  ```

### 6.2. Trên Linux (Setup_CMP30HX_LinuxAIO.sh)
- **Cài đặt tự động mở khoá & đăng ký Systemd**:
  ```bash
  chmod +x Setup_CMP30HX_LinuxAIO.sh
  sudo ./Setup_CMP30HX_LinuxAIO.sh
  ```
- **Kiểm tra trạng thái link & MRRS**:
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --status
  ```
- **Gỡ bỏ service & sleep hook**:
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
  ```

### 6.3. Biên dịch lại toàn bộ công cụ Go (Khi phát triển)
```cmd
cd D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\tools\inst40hx
go build -ldflags "-s -w -H=windowsgui" -o "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe" .

cd D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\tools\check40x
go build -ldflags "-s -w -H=windowsgui" -o "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXCheck.exe" .

cd D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\tools\uninstall40x
go build -ldflags "-s -w -H=windowsgui" -o "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXUninstaller.exe" .
```

---

## 7. TIÊU CHÍ ĐÁNH GIÁ THÀNH CÔNG & BENCHMARK THAM CHIẾU (VERIFICATION METRICS)

| Công cụ đo / Chỉ số | Trạng thái Gen1 x16 (Chưa mở khoá) | Trạng thái Gen2 x16 (Đã mở khoá thành công) | Ghi chú kỹ thuật |
| :--- | :--- | :--- | :--- |
| **GPU-Z Bus Interface** | `PCIe x16 1.1 @ x16 1.1` | `PCIe x16 2.0 @ x16 2.0` | Bấm Render Test `?` để kiểm tra khi có tải 3D |
| **AIDA64 GPGPU Memory Read** | ~2.5 - 2.8 GB/s | **~6.1 - 6.4 GB/s** | Cần MRRS = 512B (`0x2000`) |
| **AIDA64 GPGPU Memory Write**| ~2.6 - 2.9 GB/s | **~5.8 - 6.2 GB/s** | Tăng gấp đôi tốc độ nạp texture/model |
| **AIDA64 GPGPU Memory Copy** | ~2.4 - 2.6 GB/s | **~6.3 - 6.5 GB/s** | Chỉ số quan trọng nhất cho xuất hình CASO/PRIME |
| **3DMark TimeSpy PCIe Test** | ~2.5 GB/s | **~6.2 GB/s** | Đo băng thông thực tế khung hình truyền qua bus |
| **Trạng thái DEVCTL** | `MRRS = 128B` (`0x0000`) | `MRRS = 512B` (`0x2000`) | Đọc tại offset `cap + 0x08` |
| **Độ ổn định (Stress Test)** | FurMark 1-2 tiếng ổn định | FurMark / 3DMark ổn định, nhiệt độ GPU < 75°C | Đảm bảo mod phần cứng và nguồn cấp chịu tải tốt |
