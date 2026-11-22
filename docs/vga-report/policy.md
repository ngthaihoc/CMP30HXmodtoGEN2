# Chính sách dữ liệu và quyền riêng tư — Thu thập báo cáo VGA

Tài liệu này mô tả phạm vi dữ liệu do `Collect_VGA_Report.bat` thu thập và cách Cloudflare Worker (`worker/vga-reports`) xử lý trước khi tạo GitHub Issue trên repository `ngthaihoc/CMP30HXmodtoGEN2`.

## 1. Dữ liệu ĐƯỢC thu thập (Allowlist cố định)

Collector chỉ đọc dữ liệu kỹ thuật theo hợp đồng [schema.json](schema.json):

- **Siêu dữ liệu báo cáo:** `schema_version`, `collector_version`, `report_id` (UUID ngẫu nhiên mỗi báo cáo), `created_utc`, `experiment_id` (UUID ngẫu nhiên lưu cục bộ trong `experiment_id.txt` cạnh nơi xuất báo cáo), `consent`.
- **Danh sách GPU (CIM `Win32_VideoController` và PnP):** địa chỉ PCIe (`bdf`), `vendor_id`, `device_id`, `subsystem_id`, `revision`, tên thiết bị (`name`), trạng thái PnP (`pnp_status`), phiên bản driver (`driver_version`), tốc độ/băng thông PCIe (`max_gen`, `current_gen`, `max_width`, `current_width`).
- **Thông số `nvidia-smi` (18 trường cố định):** `name`, `driver_version`, `vbios_version`, `pci.bus_id`, `pci.device_id`, `pci.sub_device_id`, `memory.total`, `compute_cap`, `pcie.link.gen.max`, `pcie.link.gen.gpucurrent`, `pcie.link.width.max`, `pcie.link.width.current`, `temperature.gpu`, `power.draw`, `clocks.sm`, `clocks.mem`, `clocks.max.sm`, `pstate`, kèm `complete` và `missing[]`. Giá trị không hỗ trợ được ghi `unsupported` (giá trị `null`), không thay bằng `0`.
- **Nền tảng hệ thống:** tên CPU, hãng/mẫu bo mạch chủ, phiên bản BIOS, bản build Windows.
- **Ngữ cảnh do người dùng nhập:** trạng thái mod (`stock|modded|unknown`), dùng riser hay cắm trực tiếp, loại khe PCIe, giai đoạn (`before_unlock|after_unlock|troubleshooting|baseline|unknown`), ghi chú tùy chọn (tối đa 500 ký tự).

## 2. Dữ liệu KHÔNG thu thập

Công cụ tuyệt đối không thu thập và có bộ lọc PII ở cả hai phía (client `Remove-Pii` và Worker `findPii`):

- Tên người dùng Windows, tên máy (`hostname`), biến môi trường cá nhân.
- Đường dẫn thư mục cá nhân (`C:\Users\...`, `/home/...`, đường dẫn mạng UNC).
- Địa chỉ IP (IPv4 và IPv6), địa chỉ MAC, danh sách tiến trình đang chạy.
- Số serial bo mạch chủ/GPU hoặc UUID phần cứng GPU (`GPU-...`).
- Không băm hay gửi bất kỳ định danh cá nhân/phần cứng nêu trên trong JSON báo cáo.

## 3. Tính công khai của GitHub Issue

- Mọi báo cáo gửi lên đều tạo thành **GitHub Issue công khai** kèm nhãn `vga-report`, `hardware-sample`, `unverified`.
- Trước khi gửi, `Collect_VGA_Report.bat` hiển thị bản tóm tắt, tên máy chủ đích và yêu cầu xác nhận đồng ý.
- Báo cáo chỉ phản ánh thông số đọc được tại thời điểm chạy, **không phải bằng chứng mở khóa phần cứng thành công** và không thay thế kiểm chứng trên tải thực tế.

## 4. Địa chỉ IP, hạ tầng Cloudflare và nhật ký (Logs)

- Khi gửi qua HTTPS tới `*.workers.dev`, hạ tầng mạng của Cloudflare nhìn thấy địa chỉ IP kết nối theo cơ chế mạng chuẩn.
- **Trong mã nguồn Worker:** Worker **không ghi log** nội dung báo cáo và **không ghi log** địa chỉ IP gốc. Địa chỉ IP kết nối chỉ được băm theo ngày bằng khóa muối bí mật (`HMAC-SHA256(IP_SALT, "<ngày-UTC>:<ip>")`) để kiểm soát hạn mức trong Cloudflare KV với TTL tự hết hạn sau **2 ngày (172.800 giây)**. Thiếu/trống muối hoặc thiếu binding KV thì từ chối `503` trước khi truy cập KV, không dùng muối mặc định công khai.
- **Thời hạn giữ log của Cloudflare:** Theo tài liệu chính thức của Cloudflare Workers Logs / Observability (kiểm chứng ngày 2026-11-21), gói miễn phí lưu nhật ký sự kiện Worker tối đa **3 ngày** (tối đa 200.000 log events/ngày) theo cấu hình cũ và **7 ngày** (hạn mức nạp 0,5 GB/ngày) theo cơ chế Observability mới áp dụng từ 01/12/2026, **khi bật Workers Logs**. Dự án khuyến nghị giữ Workers Observability/Logs ở trạng thái tắt trong vận hành bình thường, chỉ bật tạm thời khi chẩn đoán lỗi.

## 5. Giới hạn hệ thống và bảo vệ chống lạm dụng

- Kích thước payload JSON tối đa: **2 MiB**.
- Số GPU tối đa mỗi báo cáo: **16**; chuỗi tối đa **500** ký tự; nội dung Issue tối đa **60.000** ký tự (kèm marker).
- Hạn mức mặc định: **10 yêu cầu mới hợp lệ/IP/ngày** (`IP_LIMIT`), **200 yêu cầu mới hợp lệ/ngày** (`DAILY_CAP`); tính trước khi gọi GitHub nên lỗi GitHub vẫn tiêu thụ quota. Chữ ký sai, JSON/schema sai và PII không tiêu thụ quota. Báo cáo trùng đã lưu được trả cùng URL mà không tăng quota.
- Các hạn mức và chống trùng trên KV là **gần đúng** giữa nhiều isolate, không phải bảo đảm nguyên tử toàn hệ thống. Hai yêu cầu đồng thời cùng ID hoặc gửi lại sau lỗi lưu chống trùng có thể tạo Issue trùng. Giới hạn ghi cùng khóa của KV có thể khiến Worker trả `503`; BAT giữ ZIP và `.pending.json`. Xem [runbook](runbook.md) để biết giới hạn và cách xử lý.
- Khi phát hiện spam hoặc lạm dụng khóa ký công khai trong BAT, chủ dự án thu hồi `key_id` cũ (xem [runbook](runbook.md)); bản BAT cũ nhận `401` tới khi cập nhật. Khóa này không xác thực danh tính người gửi và không ngăn được người chủ động tạo yêu cầu hợp lệ. Khi bị chặn nhầm, người dùng có thể gửi qua Issue Form thủ công.

## 6. Quy trình yêu cầu xóa báo cáo

- Worker không lưu bản sao nội dung báo cáo; khóa chống trùng `r:<report_id>` chứa mã băm SHA-256 của body, URL/số Issue và thời gian tạo. Khóa này không có TTL; dữ liệu quota hết hạn sau 2 ngày.
- Nếu bạn muốn xóa báo cáo đã đăng, hãy mở yêu cầu (hoặc bình luận trực tiếp trên issue) nêu rõ **số Issue** hoặc **`report_id`** để chủ repository đóng/xóa Issue tương ứng.
- **Lưu ý trung thực:** GitHub Issue là công khai; sau khi xuất bản, chủ dự án không thể bảo đảm xóa được mọi bản sao do bộ máy tìm kiếm, fork hoặc bên thứ ba đã lưu trữ trước thời điểm xóa.
