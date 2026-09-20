# BỐI CẢNH DỰ ÁN (PROJECT CONTEXT): CMP 40HX & CMP 30HX PCIE UNLOCK

Tài liệu này được tạo nhằm giúp các AI Agent khi bắt đầu phiên làm việc mới có thể nắm bắt ngay toàn bộ bối cảnh kỹ thuật, lịch sử phát triển, cấu trúc mã nguồn, các giới hạn phần cứng và công việc đang tiếp diễn mà không cần đọc lại hàng nghìn dòng nhật ký hội thoại.

---

## 1. TỔNG QUAN VÀ MỤC TIÊU DỰ ÁN

- **Mục tiêu cốt lõi**: Mở khoá băng thông PCIe cho các dòng card chuyên đào coin NVIDIA CMP (Crypto Mining Processor) chạy trên hệ điều hành Windows để sử dụng cho đồ hoạ, AI inference và chơi game.
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
Trong quá trình thử nghiệm mở khoá cho CMP 30HX (TU116):
- **Gen2 (5.0 GT/s)**:
  - Chỉ bị **khoá mềm (Software / VBIOS Shadow Lock)**.
  - Véc-tơ tốc độ link hỗ trợ (`LNKCAP2`, offset `0x0880A4`) có bit 2 (5.0 GT/s) vẫn nguyên vẹn (`0x06` = `0110b`).
  - Cho phép ghi đè thông qua thanh ghi BAR0 MMIO để nâng link lên Gen2 thành công 100%.
- **Gen3 (8.0 GT/s)**:
  - Bị **KHOÁ CỨNG TẠI SILICON DIE BẰNG EFUSE (Physical Silicon eFuse Blown)** bởi chính NVIDIA từ nhà máy.
  - **Chứng cứ thực nghiệm trên phần cứng sống**:
    - Khi ghi ép giá trị `0x0E` (Gen1+Gen2+Gen3) vào thanh ghi `LNKCAP2` (`0x0880A4`), đọc ngược lại (readback) chỉ ra `0x00000006` (Bit 3 bị triệt tiêu ở mức phần cứng).
    - Khi ghi `0x00010003` (Target Link Speed = Gen3) vào `LNKCTL2` (`0x0880A8`), đọc ngược lại chỉ trả về `0x00010002` (Target Link Speed bị kẹp ở Gen2).
    - Thanh ghi xung nhịp PHY Lane 0 (`0x08C4B0`) cố định ở 5.0 GHz (`0x50000000`), không thể nâng lên 8.0 GHz.
  - **Kết luận bắt buộc**: **CMP 30HX hoàn toàn KHÔNG THỂ lên PCIe Gen3 hay Gen4**. Giới hạn vật lý tối đa của CMP 30HX là **PCIe Gen2 x16**. Bất kỳ nỗ lực ép Gen3 nào đều thất bại và gây lỗi vòng lặp retrain.

### 2.2. Chuỗi thanh ghi BAR0 MMIO mở khoá Gen2 trên TU116 (CMP 30HX)
Quy trình mở khoá Gen2 cho CMP 30HX thông qua driver WinRing0 / ThrottleStop:
1. Ghi `0xE0B42D00` vào `PRIV_MISC_1` (`0x08841C`) để tắt cơ chế chống ghi (Write-Protect) của shadow registers.
2. Ghi `0x00000006` vào `XVE_OVR` (`0x08872C`) để khai báo hỗ trợ Gen1 + Gen2 (5.0 GT/s).
3. Ghi `0x80085800` vào `LINK_CONFIG_0` (`0x08C040`) và `0x068731B3` vào `CYA_0` (`0x08C2C0`).
4. Kiểm tra thanh ghi PHY Lane 0 (`0x08C4B0`): chuyển từ `0x25000000` (2.5 GHz - Gen1) sang `0x50000000` (5.0 GHz - Gen2).
5. Kích hoạt huấn luyện lại link (PCIe Retrain) thông qua Root Port (CPU / Chipset) bằng cách đảo bit 5 trong thanh ghi `LNKCTL` (`cap + 0x10`).

### 2.3. Tinh chỉnh DEVCTL MRRS (Nút thắt nghẽn băng thông DMA)
- **Vấn đề đã gặp**: Dù GPU-Z và HWiNFO64 báo link đã đạt `Gen2 x16`, nhưng khi chạy benchmark **AIDA64 GPGPU Memory Copy / Read**, tốc độ chỉ đạt ~2.5 GB/s (ngang Gen1).
- **Nguyên nhân cốt lõi**: Thanh ghi `DEVCTL` (`cap + 0x08`) có các bit [14:12] quy định Max Read Request Size (MRRS). Giá trị mặc định là 128 Bytes (`000b`), khiến các gói TLP của bộ nhớ phân mảnh cực kỳ nghiêm trọng khi đọc DMA.
- **Giải pháp**: Ghi cấu hình MRRS = 512 Bytes (`2 << 12 = 0x2000`) vào `DEVCTL`, sau đó khởi động lại service `NVDisplay.ContainerLocalSystem` để driver nạp lại hàng đợi DMA.
- **Kết quả**: Băng thông AIDA64 Memory Copy tăng vọt từ ~2.5 GB/s lên **~6.3 - 6.4 GB/s** (đạt ~98% hiệu suất lý thuyết của PCIe Gen2 x16).

---

## 3. LỖI ĐÃ XỬ LÝ VÀ TRẠNG THÁI HIỆN TẠI

### 3.1. Lỗi vòng lặp vô hạn tác vụ `40HXGen2Retry`
- **Hiện tượng**: Khi chạy thử nghiệm cờ `-gen3-30hx` hoặc `-force-root-gen3`, biến `targetGen` bị gán bằng 3. Vì phần cứng kẹp ở Gen2, điều kiện `cur >= targetGen` (`2 >= 3`) trả về `false`. Chương trình báo `❌ Gen3 未达成` và gọi `scheduleGen2Retry()`, tự tạo Scheduled Task chạy lại mỗi 1 phút vô tận.
- **Cách khắc phục**:
  1. Xoá tác vụ lặp trên máy: `schtasks /delete /tn "40HXGen2Retry" /f`.
  2. Trong mã nguồn (`40hxcore/const.go` và `inst40hx/main.go`), giới hạn `MaxSupportedGen` của CMP 30HX cố định là 2.
  3. Khi phát hiện `cur >= 2`, xác nhận thành công ngay (`ok = true`), gọi `deleteGen2Retry()` để huỷ lịch trình retry.

### 3.2. Tiến trình Việt hoá (Localization)
Dự án đang được chuyển đổi toàn bộ giao diện, thông báo, nhật ký sang tiếng Việt:
- `windows-v3.0/tools/check40x/main.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/uninstall40x/main.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/40hxcore/uninstall_ops.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/inst40hx/gui.go`: Đã Việt hoá 100%.
- `windows-v3.0/tools/inst40hx/main.go`: Đang hoàn thiện các chuỗi log và CLI sang tiếng Việt, khoá luồng 30HX ở Gen2.
- `README.md` (Gốc & `windows-v3.0/README.md`): Cần viết lại hoàn toàn bằng tiếng Việt với đầy đủ hướng dẫn mod CMP 30HX Gen2.

---

## 4. BẢN ĐỒ CẤU TRÚC DỰ ÁN

```
D:\ClodeGithub\CMP40HX-Unlock-main\
├── CLAUDE.md                                    # Hướng dẫn nhanh cho Claude Code
├── PROJECT_CONTEXT.md                           # Tệp bối cảnh chi tiết này
├── README.md                                    # Tài liệu hướng dẫn chính (Việt hoá)
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

### 6.1. Khởi động mở khoá CMP 30HX Gen2 khi đăng nhập Windows (Scheduled Task)
```cmd
schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f
```

### 6.2. Huỷ bỏ tác vụ lặp lỗi cũ
```cmd
schtasks /delete /tn "40HXGen2Retry" /f
```

### 6.3. Biên dịch lại toàn bộ công cụ (Go Build)
```cmd
cd D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\tools\inst40hx
go build -ldflags "-s -w -H=windowsgui" -o "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe" .

cd D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\tools\check40x
go build -ldflags "-s -w -H=windowsgui" -o "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXCheck.exe" .

cd D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\tools\uninstall40x
go build -ldflags "-s -w -H=windowsgui" -o "D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXUninstaller.exe" .
```

### 6.4. Chạy kiểm tra chẩn đoán
```cmd
D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXCheck.exe
```
hoặc chạy kiểm tra trạng thái từ bộ cài:
```cmd
D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe -status
```
