# Đặc Tả Kỹ Thuật (Spec): Tái Cấu Trúc Hợp Nhất LinkNegotiator, DriverSession và Chuẩn Hóa StatusContract

## Problem Statement

Hiện tại, codebase gặp phải ba điểm ma sát kiến trúc (architectural friction) nghiêm trọng:
1. **Phân nhánh xử lý PCIe nông và trùng lặp (Forked & Shallow PCIe Training)**: Module sâu `LinkNegotiator` trong `40hxcore/link.go` mới chỉ được kích hoạt cho CMP 30HX (`gen2MainCMP30HX`), trong khi CMP 40HX vẫn chạy qua một quy trình thủ tục dài hơn 700 dòng (`gen2Main` trong `windows-v3.0/tools/inst40hx/main.go`, cc=101) chứa toàn bộ các mảng thanh ghi MMIO viết chay, vòng lặp polling 75ms thủ công, can thiệp trực tiếp vào thanh ghi root port và khôi phục PnP lỗi thời. Các cải tiến an toàn gần đây (chẳng hạn như tối ưu MRRS 512B, giới hạn eFuse, và logic retrain chuẩn) bị phân mảnh giữa 30HX và 40HX.
2. **Rò rỉ vòng đời driver BYOVD (Leaky BYOVD Driver Lifecycle)**: Các driver kernel `WinRing0x64.sys` và `ThrottleStop.sys` được nạp, khởi chạy và dừng dịch vụ một cách thủ công, rải rác ở nhiều công cụ (`inst40hx`, `check40x`, `state.go`). Nếu xảy ra lỗi giữa chừng hoặc chương trình bị tắt đột ngột, các driver này vẫn tồn tại trong kernel Windows, dẫn đến nguy cơ các hệ thống chống gian lận game như Riot Vanguard (Valorant) hoặc Easy Anti-Cheat (Apex Legends) phát hiện và xử phạt tài khoản người dùng.
3. **Seam trạng thái Go-Batch bị phá vỡ (Unstructured Status Leakage)**: Mặc dù đã có `StatusContract` cấu trúc hóa dạng cặp khóa-giá trị, hàm `WriteGen2Status(string)` vẫn tồn tại và được gọi tùy tiện với các chuỗi tự do (như `⏭️ 跳过: ...`), làm các kịch bản Batch như `Setup_CMP30HX_WindowsAIO.bat` không thể bóc tách mã trạng thái hoặc rơi vào nhánh lỗi sai lệch.

## Solution

1. **Hợp nhất toàn bộ luồng huấn luyện vào Deep Module `LinkNegotiator`**: Tuyến đường huấn luyện PCIe của cả CMP 30HX và CMP 40HX đều đi qua duy nhất một phương thức `LinkNegotiator.Negotiate(gpuBDF, prof, rootBDF, targetGen, allowStage2)`. Thu nhỏ hàm `gen2Main()` trong `inst40hx/main.go` từ hơn 700 dòng xuống còn khoảng 20 dòng điều phối cấp cao.
2. **Đóng gói vòng đời driver kernel vào Deep Module `DriverSession`**: Sử dụng mô hình Scoped Closure / RAII (`DriverSession.RunScoped(prof, fn func(bus HardwareBus) error) error`). Toàn bộ việc triển khai file `.sys`, tạo dịch vụ, khởi động driver, mở handle, và dọn dẹp sạch sẽ khi kết thúc đều được thực hiện tự động và đảm bảo dọn dẹp 100% trong khối `defer`, loại bỏ hoàn toàn nguy cơ sót driver kernel.
3. **Chuẩn hóa Seam trạng thái qua `StatusContract`**: Loại bỏ hàm `WriteGen2Status(string)`. Bắt buộc mọi giao tiếp trạng thái giữa tầng Go engine và các script Batch/PowerShell phải sử dụng `WriteStructuredGen2Status(StatusContract)`, đảm bảo tính tất định và tương thích lâu dài.

## User Stories

1. Là người dùng CMP 40HX, tôi muốn quá trình huấn luyện PCIe Gen2/Gen3 sử dụng cùng một logic an toàn thanh ghi MMIO như CMP 30HX, để liên kết PCIe đạt tốc độ tối đa ổn định và không bị lỗi lệch cấu hình thanh ghi.
2. Là người dùng CMP 40HX, tôi muốn công cụ tự động tối ưu thanh ghi `DEVCTL` lên MRRS 512B (`0x2000`), để đạt băng thông truyền dữ liệu tối đa ~6.4 GB/s.
3. Là người dùng CMP 40HX, tôi muốn kiểm tra chính xác họ vi kiến trúc phần cứng `BOOT_0` (`0x16xxxxxx` cho TU106) trước khi ghi thanh ghi PL0, để ngăn chặn rủi ro ghi nhầm vào vùng nhớ thiết bị khác khi có hiện tượng BAR remap.
4. Là người dùng CMP 30HX, tôi muốn phần cứng luôn được bảo vệ bởi rào an toàn eFuse bit 3 (tự động kẹp về Gen2 và cấm các thao tác reset nguy hiểm), để card không bị treo hoặc mất liên kết PCIe.
5. Là người chơi game có hệ thống chống gian lận (Riot Vanguard, Easy Anti-Cheat, BattlEye), tôi muốn sau khi huấn luyện PCIe xong hoặc nếu có lỗi xảy ra, toàn bộ driver BYOVD (`WinRing0`, `ThrottleStop`) được gỡ bỏ ngay lập tức khỏi kernel, để không bị phần mềm chống gian lận cấm chơi hoặc khóa tài khoản.
6. Là người chơi game, tôi muốn khi chạy chẩn đoán (`check40x`) hoặc cài đặt (`inst40hx`), các driver BYOVD chỉ tồn tại tạm thời trong bộ nhớ trong đúng thời gian đo đạc và tự giải phóng, không để lại bất kỳ tàn dư nào trên đĩa hoặc kernel.
7. Là quản trị viên hệ thống hoặc người dùng chạy tự động qua Scheduled Task (SYSTEM account), tôi muốn `gen2_status.txt` luôn chứa các trường định dạng chuẩn (`STATUS_CODE`, `SPEED_CURRENT`, `WIDTH_CURRENT`, `TLS_TARGET`, `ERROR_CODE`), để script Batch và PowerShell đọc kết quả một cách chính xác mà không bị lỗi phân tích chuỗi.
8. Là người bảo trì codebase, tôi muốn có thể kiểm thử toàn bộ quá trình huấn luyện PCIe cho cả 30HX và 40HX trên môi trường giả lập (CI/CD) mà không cần card đồ họa vật lý, thông qua adapter `MockHardwareBus`.
9. Là người bảo trì, tôi muốn bảng thanh ghi shadow MMIO (`PRIV_MISC_1`, `XVE_OVR`, `LINK_CONFIG_0`, `PL_LINK_RATE`, `CYA_0`) chỉ được khai báo ở một nơi duy nhất trong `40hxcore/link.go`, không bị sao chép trong `inst40hx/main.go`.
10. Là người phát triển, tôi muốn mã nguồn của `inst40hx/main.go` giảm độ phức tạp chu trình (cyclomatic complexity), loại bỏ hơn 700 dòng mã thủ tục lặp lại để dễ dàng mở rộng và bảo trì.

## Implementation Decisions

### 1. Phân Tách Seam & Module

Kiến trúc mới tổ chức hệ thống thành các module sâu rõ ràng:

```
┌────────────────────────────────────────────────────────┐
│                   inst40hx / check40x                  │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                      DriverSession                     │  (Deep Module)
│  • Tự động chọn driver cần thiết (WinRing0 / TS)       │
│  • Nạp service & mở handles an toàn                    │
│  • Đảm bảo dọn dẹp (RAII defer cleanup)                │
└───────────────────────────┬────────────────────────────┘
                            │
                 (Seam 1: HardwareBus)
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                     LinkNegotiator                     │  (Deep Module)
│  • Rào eFuse & Giới hạn Profile                        │
│  • Kiểm tra BOOT_0 & Tuần tự ghi MMIO Shadow Registers │
│  • Cấu hình MRRS 512B & Vô hiệu hóa ASPM               │
│  • Stage 1: Retrain Pulses với Fast Polling 75ms       │
│  • Stage 2: Phục hồi Soft PnP an toàn                  │
└───────────────────────────┬────────────────────────────┘
                            │
                 (Seam 2: StatusContract)
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│         Setup_CMP30HX_WindowsAIO.bat / Scripts         │
└────────────────────────────────────────────────────────┘
```

### 2. Module `DriverSession` (40hxcore/drvsession.go)
- Đóng gói toàn bộ vòng đời driver kernel:
  - Phương thức duy nhất: `RunScoped(prof GPUProfile, fn func(bus HardwareBus) error) error`.
  - Kiểm tra `prof.HasSafePL0`: Nếu cần mở khóa PL0 (40HX), nạp cả `WinRing0` và `ThrottleStop`. Nếu chỉ cần PCI Config (30HX), chỉ nạp `WinRing0`.
  - Khởi tạo `ProductionBus(wh, th)` và chuyển giao cho `fn`.
  - Trong `defer`: đóng handles, dừng services, và xóa file `.sys` để đảm bảo sạch sẽ tuyệt đối cho anti-cheat.

### 3. Nâng Cấp Module `LinkNegotiator` (40hxcore/link.go)
- Tiếp nhận đầy đủ logic của cả 30HX và 40HX:
  - Kiểm tra an toàn vi kiến trúc: Đọc `BOOT_0` qua `bus.ReadMMIO(bar0Phys)`. Nếu `prof.Family == "TU106"` nhưng `(boot0 & 0xFF000000) != 0x16000000`, lập tức hủy ghi MMIO để bảo vệ hệ thống.
  - Hợp nhất tuần tự ghi MMIO: `PRIV_MISC_1` (`0x8841C`) -> `XVE_OVR` (`0x8872C`) -> `LINK_CONFIG_0` (`0x8C040`) -> `PL_LINK_RATE` (`0x8C1C0`) -> `CYA_0` (`0x8C2C0`).
  - Hỗ trợ Stage 2 phục hồi PnP: Thực hiện qua `bus.PnpResetDevice(prof.DeviceID)` khi `allowStage2` được bật và phần cứng cho phép (`prof.DeviceID != 0x2189`).

### 4. Thu Nhỏ `inst40hx/main.go`
- Thay thế toàn bộ mã lặp trong `gen2Main()`:
  - Bỏ các hàm thủ tục cấp thấp không còn dùng: `gen2WritePL0`, `gen2SetTLS`, `gen2RetrainPulse`, `gen2RootLinkDisable`, `gen2HardFallback`, `gen2VerdictHard`.
  - Thu gọn `gen2Main()`: Gọi `DriverSession.RunScoped`, bên trong tạo `LinkNegotiator` và gọi `Negotiate()`.
  - Ghi nhận kết quả thông qua `hxcore.WriteStructuredGen2Status`.

### 5. Chuẩn Hóa Seam `StatusContract` (40hxcore/status.go & service.go)
- Xóa bỏ hoặc chuyển `WriteGen2Status(string)` thành nội bộ.
- Bắt buộc mọi ghi nhận trạng thái từ `inst40hx` đều chuyển thành cấu trúc `StatusContract` với `StatusCode` định kiểu (`GEN2_SUCCESS`, `GEN2_FAILED`, `GEN2_SKIPPED`, `GEN2_HARDWARE_LIMIT`).

## Testing Decisions

### 1. Tiêu Chí Kiểm Thử Chuẩn (Good Tests)
- Kiểm thử chỉ tương tác qua Seam bên ngoài (`HardwareBus`), không kiểm tra biến nội bộ hoặc cấu trúc bên trong.
- Toàn bộ suite kiểm thử chạy độc lập trên mọi môi trường phát triển (Windows/Linux CI) mà không cần card đồ họa vật lý.

### 2. Các Module Được Kiểm Thử
- **Unit Test `LinkNegotiator` (`link_test.go`)**:
  - Test kịch bản 40HX: Mô phỏng `BOOT_0 == 0x166000A1`, ghi shadow registers, retrain pulse và đạt Gen2 x16 thành công.
  - Test kịch bản 40HX BOOT_0 mismatch: Mô phỏng `BOOT_0 == 0xFFFFFFFF`, đảm bảo module an toàn dừng ghi MMIO và trả về lỗi.
  - Test kịch bản 30HX: Đảm bảo eFuse clamping luôn đưa mục tiêu về Gen2 và `allowStage2` bị vô hiệu hóa hoàn toàn.
  - Test kịch bản Stage 2 Soft Recovery: Mô phỏng Stage 1 thất bại, kích hoạt `PnpResetDevice` và retrain thành công ở Stage 2.
- **Unit Test `StatusContract` (`status_test.go`)**:
  - Xác thực việc sinh mã và phân tích cú pháp chuỗi trạng thái không có lỗi định dạng.

## Out of Scope

1. Không can thiệp nạp BIOS SPI ROM vật lý hoặc chỉnh sửa strap điện trở phần cứng.
2. Không viết lại toàn bộ kịch bản AIO sang Go; giữ nguyên giao diện dòng lệnh thân thiện của `Setup_CMP30HX_WindowsAIO.bat` và chỉ chuẩn hóa seam giao tiếp giữa hai tầng.
3. Không gỡ bỏ các công cụ độc lập khác (`unlockriot`, `uninstall40x`) trừ khi có yêu cầu riêng biệt.

## Further Notes

- Việc xóa bỏ hơn 700 dòng mã thừa trong `inst40hx/main.go` giúp giảm đáng kể kích thước mã nguồn và rủi ro hồi quy (regression) khi nâng cấp các phiên bản driver NVIDIA mới trong tương lai.
- Tính an toàn đối với hệ thống chống gian lận (Anti-Cheat Safety) là ưu tiên hàng đầu, do đó kiến trúc Scoped Closure của `DriverSession` loại trừ hoàn toàn khả năng sót driver trong kernel.
