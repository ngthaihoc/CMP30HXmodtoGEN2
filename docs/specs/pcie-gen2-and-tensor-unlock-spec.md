# Đặc Tả Kỹ Thuật (Spec): Tái Cấu Trúc Kiến Trúc Mở Khoá PCIe Gen2 (30HX & 40HX) và Tensor Core (40HX)

## Problem Statement

Người dùng sở hữu card CMP 30HX (TU116) và CMP 40HX (TU106) đối mặt với các rào cản phần cứng và kiến trúc phần mềm phức tạp:
1. **Rủi ro phần cứng & Khóa vật lý**: CMP 30HX bị đứt eFuse bit 3 ở cấp độ die silicon (giới hạn vật lý tối đa 5.0 GT/s / Gen2), trong khi CMP 40HX cần kích hoạt Tensor Core qua thanh ghi SS0 (`0x409664 == 0x88888888`) và mở liên kết PCIe qua Root Port retrain.
2. **Kiến trúc nông và rò rỉ (Shallow & Leaky)**: Logic điều khiển phần cứng trong codebase hiện tại bị phân mảnh: mã điều phối luồng (`gen2Main` >1400 dòng) trộn lẫn trực tiếp với các lệnh gọi driver kernel cấp thấp (`WinRing0`, `ThrottleStop`), các bảng ghi thanh ghi MMIO lặp đi lặp lại 2 lần, các vòng lặp chờ polling 75ms và khôi phục PnP thiết bị.
3. **Không thể kiểm thử tự động (Untestable without hardware)**: Thiếu các seam kiến trúc khiến cho toàn bộ máy trạng thái huấn luyện lại PCIe và bộ giải mã tính toán Tensor Core không thể được kiểm thử unit-test nếu không có phần cứng vật lý và đặc quyền kernel tương ứng.
4. **Giao tiếp mong manh (Brittle Inter-process Contract)**: Kịch bản Batch (`Setup_CMP30HX_WindowsAIO.bat`) phân tích kết quả từ Go Engine qua tìm kiếm chuỗi văn bản tự do (`findstr /c:...`), dẫn đến lỗi nhận diện sai khi câu chữ thông báo tiếng Việt có sự thay đổi.

## Solution

Chuyển đổi toàn bộ hệ thống điều khiển phần cứng từ các kịch bản nông sang hai Deep Module cốt lõi:
1. **Module `LinkNegotiator`**: Đóng gói toàn bộ máy trạng thái huấn luyện lại PCIe, tự động áp dụng trần eFuse theo từng GPU Profile (bảo vệ tuyệt đối TU116 không bị ép vượt Gen2), tối ưu cấu hình DEVCTL MRRS 512B (`0x2000`), và thực thi quy trình khôi phục PnP 2 giai đoạn.
2. **Module `ComputeInspector`**: Đóng gói toàn bộ vòng đời driver BYOVD, xác thực an toàn họ vi kiến trúc phần cứng (`BOOT_0`), và giải mã trạng thái tính toán Tensor Core thành báo cáo định kiểu (typed report) hoàn chỉnh.
3. **Seam `HardwareBus` & `StatusContract`**:
   - `HardwareBus`: Điểm phân tách giữa logic phần cứng và trình điều khiển kernel (hỗ trợ adapter Production chạy BYOVD và adapter Mock phục vụ kiểm thử không cần GPU).
   - `StatusContract`: Định dạng giao tiếp trạng thái chuẩn hoá có cấu trúc giữa Go Engine và Batch/PowerShell.

---

## User Stories

1. Là người dùng CMP 30HX, tôi muốn công cụ tự động nhận diện chính xác GPU và giới hạn tốc độ huấn luyện ở Gen2, để card không bị treo hoặc mất liên kết PCIe do cố ép xung vượt quá eFuse bit 3.
2. Là người dùng CMP 30HX, tôi muốn thanh ghi `DEVCTL` được tự động tối ưu giá trị MRRS lên 512 bytes (`0x2000`), để thông lượng DMA đọc ghi không bị phân mảnh và đạt hiệu suất tối đa ~6.4 GB/s.
3. Là người dùng CMP 30HX, tôi muốn service `NVDisplay.ContainerLocalSystem` được khởi động lại sau khi huấn luyện PCIe thành công, để hàng đợi DMA của driver NVIDIA được tái tạo mà không cần khởi động lại toàn bộ máy tính.
4. Là người dùng CMP 40HX, tôi muốn kiểm tra chính xác trạng thái mở khoá Tensor Core (SS0 == `0x88888888` và SS1), để xác nhận năng lực tính toán FP16 HGEMM đạt ~50 TFLOPS cho các tác vụ AI/Deep Learning.
5. Là người dùng CMP 40HX, tôi muốn công cụ tự động ngăn chặn việc nạp các blob EFI hoặc vi mã không tương thích vào các GPU khác (như CMP 30HX), để tránh nguy cơ làm hỏng firmware hoặc biến card thành "cục gạch".
6. Là người dùng hệ thống có Secure Boot bật (chơi game có Riot Vanguard / Valorant hoặc Easy Anti-Cheat), tôi muốn sau khi huấn luyện PCIe xong thì toàn bộ driver BYOVD (`WinRing0`, `ThrottleStop`) được gỡ bỏ sạch sẽ khỏi kernel và ổ đĩa, để không bị phần mềm chống gian lận chặn hoặc xử phạt.
7. Là người dùng gặp lỗi Windows Security chặn driver (Error 5 / HVCI / Memory Integrity), tôi muốn nhận được thông báo chẩn đoán bằng tiếng Việt rõ ràng chỉ dẫn cần tắt Memory Integrity hoặc khởi động lại máy, để tôi biết chính xác cách khắc phục mà không hoang mang.
8. Là người dùng trên bo mạch chủ OEM/X99/server có Root Port khó tính, tôi muốn công cụ tự động kích hoạt Stage 2 (PnP Soft Reset và nạp lại thanh ghi MMIO), để liên kết PCIe tự động nhảy lên Gen2 mà không cần tôi phải vào Device Manager thao tác thủ công.
9. Là người dùng cắm card trên khe PCIe bị giới hạn tốc độ (chỉ hỗ trợ Gen1), tôi muốn công cụ cảnh báo rõ ràng khả năng phần cứng của Root Port và GPU, để tôi điều chỉnh BIOS hoặc đổi vị trí cắm phù hợp.
10. Là người bảo trì và phát triển codebase, tôi muốn có thể kiểm thử toàn bộ máy trạng thái huấn luyện PCIe và các kịch bản lỗi biên bằng unit-test tự động trong CI/CD mà không cần gắn card CMP vật lý vào máy.
11. Là người bảo trì, tôi muốn bảng ghi thanh ghi MMIO (`PRIV_MISC_1`, `XVE_OVR`, `LINK_CONFIG_0`, `PL_LINK_RATE`, `CYA_0`) chỉ được định nghĩa tại một nơi duy nhất, để tránh lỗi lệch cấu hình khi bảo trì giữa Stage 1 và Stage 2.
12. Là người bảo trì, tôi muốn kịch bản Batch phân tích kết quả thực thi của Go Engine thông qua các mã token trạng thái định kiểu thay vì tìm kiếm chuỗi văn bản tự do, để không bị lỗi nhận diện sai khi thay đổi nội dung ngôn ngữ giao diện.
13. Là người dùng đang ở chế độ tiết kiệm điện (ASPM), tôi muốn công cụ phân biệt được giữa việc "mở khoá thành công nhưng đang nghỉ nên tạm về Gen1" và việc "mở khoá thất bại", để không đưa ra chẩn đoán sai lệch.
14. Là quản trị viên hệ thống, tôi muốn tác vụ chạy nền định kỳ kiểm tra liên kết PCIe chỉ can thiệp khi phát hiện liên kết bị tụt tốc độ bất thường, để không lãng phí tài nguyên CPU của máy.
15. Là người dùng kích hoạt tính năng Resizable BAR (ReBAR), tôi muốn kịch bản tự động kiểm tra sự tương thích của nền tảng (tránh bật trên laptop hoặc bo mạch chủ không hỗ trợ) và lưu cấu hình driver bền vững, để tránh hiện tượng màn hình đen khi khởi động lại.

---

## Implementation Decisions

### 1. Phân Tách Seam & Module

Hệ thống được tổ chức thành 2 seam chính với các module sâu tương ứng:

```
[Orchestrator: CLI / Daemon / Diagnostic]
                  │
                  ▼
┌─────────────────────────────────────────────────────────┐
│                     LinkNegotiator                      │  (Deep Module)
│  • Profile & eFuse Enforcement                          │
│  • MMIO Shadow Register Sequencing                      │
│  • Adaptive Retrain & 75ms Fast Polling                 │
│  • DEVCTL MRRS 512B DMA Optimization                    │
│  • Stage 2 PnP Soft Recovery Orchestration              │
└─────────────────────────────────────────────────────────┘
                  │
        (Seam 1: HardwareBus)
                  │
      ┌───────────┴───────────┐
      ▼                       ▼
[ProductionBus]         [MockHardwareBus]
(WinRing0 + TS)         (In-Memory Simulation)
```

```
[Status Producer: LinkNegotiator / Diagnostic]
                  │
       (Seam 2: StatusContract)
                  │
      ┌───────────┴───────────┐
      ▼                       ▼
[Structured Text Engine]  [Batch Consumer Parser]
(STATUS=..., REASON=...)  (ParseStatusFile macro)
```

### 2. Seam 1: `HardwareBus` Interface
- Interface này bao bọc toàn bộ thao tác I/O phần cứng cấp thấp:
  - Đọc/ghi cấu hình PCI Config (thông qua BDF và thanh ghi offset).
  - Đọc/ghi bộ nhớ MMIO vật lý BAR0 (thông qua địa chỉ vật lý và offset).
  - Định vị khả năng PCIe Capability (PCIe Cap offset).
  - Thao tác đặt lại thiết bị PnP (PnP Soft Reset).
- **Production Adapter**: Mở thiết bị kernel `\\.\WinRing0_1_2_0` và `\\.\ThrottleStop`, thực hiện DeviceIoControl thực tế.
- **Mock Adapter**: Lưu trữ bản đồ thanh ghi trong bộ nhớ (in-memory map), cho phép kiểm thử các kịch bản:
  - GPU trả về tốc độ Gen1 và nhảy lên Gen2 sau N lần retrain.
  - GPU bị kẹt Gen1 buộc kích hoạt Stage 2 PnP reset.
  - Driver kernel bị chặn quyền (trả về lỗi truy cập mã 5).
  - Địa chỉ BAR0 không hợp lệ hoặc họ chip `BOOT_0` không khớp.

### 3. Deep Module: `LinkNegotiator`
- Ẩn toàn bộ chi tiết triển khai phía sau một phương thức duy nhất:
  - Đầu vào: `GPUProfile`, cấu hình mục tiêu (`TargetGen`), và cờ cho phép kích hoạt fallback (`AllowStage2`).
  - Đầu ra: Cấu trúc kết quả `NegotiationResult` chứa tốc độ đạt được (`CurrentSpeed`), độ rộng (`CurrentWidth`), trạng thái mục tiêu (`TargetTLS`), và kết luận (`Verdict`).
- **Quy tắc phần cứng bắt buộc**:
  - Đối với TU116 (CMP 30HX, DEV_2189): Tuyệt đối kẹp `TargetGen <= 2`. Bất kể tham số người dùng truyền vào `-gen3` hay `-gen3-30hx`, module sẽ hạ mục tiêu về Gen2 kèm ghi nhận eFuse lock.
  - Thứ tự ghi MMIO bất biến: `PRIV_MISC_1` (`0x8841C`) -> `XVE_OVR` (`0x8872C`) -> `LINK_CONFIG_0` (`0x8C040`) -> `PL_LINK_RATE` (`0x8C1C0`) -> `CYA_0` (`0x8C2C0`).
  - Tối ưu `DEVCTL` (offset cap+0x08): Kiểm tra bits[14:12] (MRRS). Nếu nhỏ hơn 2 (128B hoặc 256B), tự động ghi đè lên 2 (512B - `0x2000`).

### 4. Deep Module: `ComputeInspector`
- Ẩn toàn bộ quy trình kiểm tra Tensor Core:
  - Tự động nạp driver tạm thời nếu chưa sẵn sàng, đo đạc thanh ghi `SS0` (`0x409664`) và `SS1` (`0x40966C`), sau đó dọn dẹp sạch sẽ driver.
  - Trước khi đọc MMIO, bắt buộc xác thực byte họ vi kiến trúc tại `BAR0+0x00` (`BOOT_0`): với TU106 (CMP 40HX) phải khớp `0x16xxxxxx`. Nếu không khớp, từ chối đọc để chống truy cập sai vùng nhớ thiết bị khác.
  - Trả về đối tượng `ComputeReport` định kiểu: `Status` (Enum: `Unlocked`, `HardwareLocked`, `DriverBlocked`, `FamilyMismatch`).

### 5. Seam 2: `StatusContract` (Giao Thức Đồng Bộ Go-Batch)
- Tệp `gen2_status.txt` được cấu trúc thành 2 phần:
  - Header chuẩn hoá dùng cho máy phân tích (Machine-readable):
    ```
    STATUS_CODE=GEN2_SUCCESS
    SPEED_CURRENT=2
    WIDTH_CURRENT=16
    TLS_TARGET=2
    ERROR_CODE=NONE
    ```
  - Phần thân báo cáo tiếng Việt phục vụ người dùng đọc trực tiếp.
- Phía Batch (`Setup_CMP30HX_WindowsAIO.bat`) sử dụng subroutine `:ParseStatusFile` phân tích trực tiếp theo `STATUS_CODE` bằng các chuỗi cố định không dấu cách, loại bỏ hoàn toàn việc tìm kiếm cụm từ tiếng Việt dễ vỡ.

---

## Testing Decisions

### 1. Tiêu Chí Kiểm Thử Chuẩn (Good Tests)
- Kiểm thử chỉ tương tác qua Seam bên ngoài (External Interface), không kiểm tra biến nội bộ hoặc cấu trúc bên trong của hàm.
- Không phụ thuộc vào môi trường máy thật: các kịch bản kiểm thử lõi phải chạy được trên mọi máy tính phát triển và máy ảo CI/CD không có card đồ hoạ rời.

### 2. Các Module Được Kiểm Thử
- **Unit Test Go (`link_test.go`)**:
  - Test kịch bản eFuse Clamping: Ép `TargetGen = 3` trên CMP 30HX profile phải tự động điều chỉnh về `Gen2`.
  - Test kịch bản Retrain thành công ở lượt thứ 1.
  - Test kịch bản Retrain thất bại Stage 1 và kích hoạt thành công Stage 2 PnP recovery.
  - Test kịch bản tối ưu hoá `DEVCTL` MRRS từ 128B lên 512B.
  - Test kịch bản phát hiện BAR0 sai họ vi kiến trúc (BOOT_0 != 0x16).
- **Unit Test Go (`compute_test.go`)**:
  - Test giải mã SS0 == `0x88888888` trả về trạng thái `Unlocked`.
  - Test giải mã SS0 != `0x88888888` trả về trạng thái `HardwareLocked`.
- **E2E / Integration Test (`Test_Mock_CMP30HX.bat`)**:
  - Giữ vững 10 suite hiện có (39 assertions), bổ sung kiểm thử xác thực header `STATUS_CODE` mới.

### 3. Tiền Lệ Trong Codebase (Prior Art)
- `windows-v3.0/tools/40hxcore/profile_test.go`: Đã có sẵn mẫu unit-test bảng (table-driven test) của Go kiểm tra `LookupGPUProfile` và `LinkTargetAllowed`.
- `Test_Mock_CMP30HX.bat`: Mẫu kiểm thử tích hợp tự động cho toàn bộ kịch bản Batch từ chẩn đoán, nạp file, đến xử lý mã lỗi.

---

## Out of Scope

1. **Thay đổi VBIOS vật lý**: Không can thiệp nạp VBIOS SPI ROM hoặc chỉnh sửa strap điện trở phần cứng.
2. **Ép xung PCIe Gen3 cho CMP 30HX**: Nghiêm cấm mọi hành vi cố gắng phá vỡ eFuse bit 3 bằng phần mềm vì điều này bất khả thi về mặt vật lý và gây mất ổn định hệ thống.
3. **Thay thế toàn bộ Batch sang Go**: Không viết lại toàn bộ kịch bản AIO sang Go; giữ nguyên giao diện dòng lệnh thân thiện của `Setup_CMP30HX_WindowsAIO.bat` và chỉ chuẩn hoá seam giao tiếp giữa hai tầng.

---

## Further Notes

- **Tính tương thích chống gian lận (Anti-Cheat Compatibility)**: Cơ chế nạp và dọn dẹp driver tức thì (`BYOVD Cleanup`) là yêu cầu sống còn để người dùng có thể chơi các tựa game như Valorant (Riot Vanguard), Apex Legends, PUBG (Easy Anti-Cheat / BattlEye) mà không phải gỡ cài đặt bộ công cụ.
- **Tính toán băng thông thực tế**: Sau khi mở khoá Gen2 x16 và tối ưu MRRS 512B, băng thông đo được qua `nvidia-smi dmon` hoặc các bài test CUDA memcpy đạt xấp xỉ ~6.4 GB/s (tăng gấp đôi so với Gen1 x16 ~3.1 GB/s).
