# LinkNegotiator, MSHybrid và StatusContract dùng chung

Phạm vi triển khai ba đề xuất trong báo cáo kiến trúc ngày 25/10/2026.

## Quyền sở hữu hành vi

- `LinkNegotiator.Unlock` dò GPU với ba lần thử, áp dụng GPU Profile và trả nguyên verdict cùng StatusContract. HardwareBus vẫn là seam phần cứng. AIO và Installer không phân loại lại thành công hoặc restart NVDisplay lần nữa.
- Installer giữ mutex liên tiến trình, chờ driver, chuẩn bị GSP, thông báo và lịch retry riêng. CMP 30HX luôn giới hạn Gen2 và cấm Stage 2 Recovery/PnP reset.
- `riotcore.ConfigureDriverClassRegistry` dùng GPUTopologyDetector để phân biệt CMP render và GPU xuất hình, rồi cấu hình MSHybrid/CASO, blocklist, TDR, ICD và DirectX Global Router. Không có CMP phù hợp thì không ghi cấu hình. Kết quả phản ánh lỗi từng bước; caller không thông báo hoàn tất khi có lỗi.
- Registry Windows và Registry trong bộ nhớ là hai adapter tại seam nội bộ của riotcore. Test không ghi cấu hình đồ họa thật. Test vòng đời driver sử dụng SystemRoot tạm.
- Thao tác testsigning và ưu tiên từng tệp game riêng của UnlockRiotGame vẫn nằm tại caller; bước xác nhận của từng caller được giữ nguyên. Hoàn tác chỉ bỏ HighPerfAdapter, giữ các tùy chọn DirectX khác.

## Ý nghĩa StatusContract

`STATUS_CODE`, `SPEED_CURRENT`, `WIDTH_CURRENT`, `TLS_TARGET`, `ERROR_CODE` giữ tương thích. `DEVICE_ID` là số hex, `MRRS_BYTES` là số byte thập phân, `OBSERVED_AT` là thời điểm UTC theo RFC3339. Dữ liệu legacy vẫn được đọc.

`LinkState` phân biệt đã cấu hình TLS Gen2 với tốc độ quan sát trong lần chạy trước. Thiếu độ rộng hoặc MRRS giữ giá trị chưa biết; tác vụ tự khởi động không chứng minh Gen2. Trạng thái của GPU khác bị bỏ qua khi có định danh. Tệp legacy thiếu định danh/thời điểm không được coi là phép đo mới.

Băng thông hiển thị là giá trị lý thuyết suy từ tốc độ và độ rộng của lần chạy trước, không phải phép đo DMA. Gen2 x16 lý thuyết là 8 GB/s; thông lượng thực tế phụ thuộc tải và hệ thống. Telemetry nền không nạp BYOVD. Hai bản tài nguyên Web Control Center phải giống nhau.

## Kiểm chứng

Chạy `go test ./...` tại `40hxcore`, `riotcore`, `unlockriot`, `aio` và `inst40hx`; chạy `go vet ./...` và build các công cụ Windows. Kiểm tra giao diện: `node windows-v3.0/tools/aio/tests/telemetry.test.cjs`.

Các trường hợp chính: lỗi capability giữ nguyên, GPU nạp trễ, TLS đạt khi link nhàn rỗi, restart NVDisplay một lần, MRRS 512B ngay cả khi đã Gen2, không có CMP, host NVIDIA đang xuất hình, lỗi Registry từng bước, hoàn tác giữ tùy chọn khác, trạng thái thiếu/hỏng/legacy và GPU không khớp.
