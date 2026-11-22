# Runbook vận hành — Collector VGA (`worker/vga-reports`)

Tài liệu hướng dẫn cài đặt, giới hạn gói miễn phí đã kiểm chứng, xoay khóa, tắt khẩn cấp và xử lý sự cố cho hệ thống tiếp nhận báo cáo VGA.

## 1. Giới hạn gói miễn phí Cloudflare (đã kiểm chứng 2026-11-21)

Nguồn tham chiếu: tài liệu chính thức Cloudflare [giới hạn KV](https://developers.cloudflare.com/kv/platform/limits/), [ghi KV và tính nhất quán](https://developers.cloudflare.com/kv/api/write-key-value-pairs/), [định giá KV](https://developers.cloudflare.com/kv/platform/pricing/).

| Tài nguyên | Giới hạn gói Free | Ghi chú thiết kế |
| --- | --- | --- |
| **Workers Requests** | **100.000 requests/ngày** (reset 00:00 UTC) | Dùng chung cho các Worker trong tài khoản |
| **Workers CPU time** | **10 ms/request** | Thời gian chờ `fetch()` tới GitHub không tính vào CPU time |
| **Workers Memory / Subrequests** | **128 MB RAM**, **50 subrequests/request** | Mỗi báo cáo dùng tối đa 2 subrequests tới GitHub |
| **Workers KV Reads** | **100.000 reads/ngày** | Mỗi yêu cầu hợp lệ đọc ~3 khóa (`d:`, `q:`, `r:`) |
| **Workers KV Writes** | **1.000 writes/ngày** (khác khóa), **1 write/giây** trên cùng khóa | Nút thắt chính của gói miễn phí |
| **Workers KV Storage / Value** | **1 GB/tài khoản**, **25 MiB/value** | Khóa `q:` và `d:` tự hết hạn sau 2 ngày (`expirationTtl: 172800`) |
| **Workers Logs (nếu bật)** | **3 ngày** (cũ, 200k events/ngày) → **7 ngày** (từ 01/12/2026, 0,5 GB/ngày) | Khuyến nghị tắt trong vận hành chuẩn |

### Lý do chọn `DAILY_CAP = 200`

- Thứ tự xử lý: route → loại nội dung/kích thước (đọc theo luồng, tối đa 2 MiB) → chữ ký → công tắc tiếp nhận → JSON/schema/PII → cấu hình → chống trùng → quota → GitHub.
- Không có/sai chữ ký trả `401`; JSON/schema sai trả `400`; PII (gồm IPv4/IPv6) trả `422`. Các trường hợp này không đọc/ghi KV và không gọi GitHub. Khi intake tắt, yêu cầu đã ký nhận `503` trước kiểm tra nội dung và KV.
- Gửi lại tuần tự cùng ID và cùng nội dung trả `200` cùng URL, khác nội dung trả `409`: đọc 1 khóa chống trùng, không tăng quota hay gọi GitHub, kể cả khi IP đã hết hạn mức.
- Mỗi yêu cầu mới được quota chấp nhận ghi 2 khóa: `d:<day>` và `q:<day>:<hash>` (TTL `172800` giây). Tạo Issue và lưu chống trùng thành công ghi thêm `r:<report_id>` (không TTL). Lỗi GitHub vẫn tiêu thụ quota đã nhận.
- Gọi `A` là số yêu cầu được quota chấp nhận, `S` là số bản ghi Issue lưu thành công: số ghi hoàn tất theo luồng bình thường là `2A + S`. Trong mô hình một isolate, không lỗi KV và bộ đếm nhất quán, `A ≤ 200`, `S ≤ A` nên tối đa 600 lượt ghi. Đây **không phải trần bảo đảm trên Cloudflare thật**; các ghi từng phần khi lỗi, thao tác quản trị và ghi đè giữa isolate cần đo riêng.
- KV không có phép tăng nguyên tử và có tính nhất quán cuối cùng. Khóa Promise chỉ tuần tự hóa trong một isolate; `DAILY_CAP`/`IP_LIMIT` là hạn mức **gần đúng** toàn hệ thống. Chống trùng cũng không nguyên tử: hai yêu cầu đồng thời cùng ID có thể tạo hai Issue. Không tự thử lại hàng loạt để bù lỗi.
- KV chỉ cho **1 lượt ghi/giây trên cùng khóa**, kể cả gói Paid. Mọi yêu cầu mới cùng ngày đều cập nhật `d:<day>`, nên tải đồng thời có thể gây lỗi KV trước khi hết quota ngày. Worker trả `503` với `reason: kv_error` thay vì lỗi không được bắt; lỗi KV trước khi tạo Issue không gọi GitHub. BAT giữ ZIP và `.pending.json`.
- Thiếu/trống `IP_SALT` hoặc thiếu binding KV trả `503` với `reason: configuration_error`, không dùng muối mặc định công khai và không truy cập KV.
- Nếu Issue đã tạo nhưng ghi `r:` thất bại, Worker trả `503`; gửi lại có thể tạo Issue trùng. Đối chiếu marker `vga-report:<report_id>` trên repo trước khi gửi lại nhiều lần. Muốn quota và chống trùng nghiêm ngặt phải thiết kế cơ chế điều phối riêng (ví dụ Durable Object); không triển khai trong v1.

### Phương án bổ sung: Cloudflare Rate Limiting binding trên `workers.dev`

Cloudflare hỗ trợ `[[ratelimits]]` binding trực tiếp trong `wrangler.toml` (kể cả trên `*.workers.dev`) để chặn bot ở edge mà không tốn lượt ghi KV. Nếu lưu lượng tăng, có thể bổ sung vào `worker/vga-reports/wrangler.toml` và gọi `env.MY_RATE_LIMITER.limit({ key })` trước `checkQuota`:

```toml
# Tùy chọn bổ sung (không bắt buộc cho v1):
# [[ratelimits]]
# name = "REPORT_RATE_LIMITER"
# namespace_id = "1001"
# simple = { limit = 20, period = 60 }
```

---

## 2. Hướng dẫn cài đặt (Phần A — Chủ dự án thực hiện)

> **Nguyên tắc bảo mật:** Không bao giờ dán khóa riêng `.pem`, token GitHub, `IP_SALT` hoặc secret vào hội thoại AI hay commit vào Git.

### A1. Chuẩn bị môi trường
1. Tạo repo thử trước khi chạy thật (ví dụ `ngthaihoc/vga-report-sandbox`, bật **Issues**).
2. Tạo tài khoản Cloudflare (Free tier). Không cần mua domain riêng; dùng subdomain `*.workers.dev`.
3. Đảm bảo máy cài Node.js và `pnpm`, sau đó đăng nhập Wrangler:
   ```powershell
   pnpm dlx wrangler login
   ```

### A2. Tạo GitHub App
1. Vào **GitHub → Settings → Developer settings → GitHub Apps → New GitHub App**.
2. Đặt tên App, Homepage URL là URL repo. **Tắt Webhook** (bỏ chọn Active).
3. Quyền **Repository permissions**: chỉ đặt **Issues = Read & write** (Metadata read-only tự có). Không cấp quyền nào khác.
4. **Where can this app be installed?** chọn **Only on this account**.
5. Tạo App, ghi lại **App ID**. Chọn **Generate a private key** để tải file `.pem` (giữ cục bộ, **không đưa vào repo**).
6. Vào **Install App** → **Only select repositories** → ban đầu chỉ chọn repo thử. Ghi lại **Installation ID** (con số ở cuối URL trang cài đặt).
7. Chuyển khóa từ PKCS#1 sang **PKCS#8** (Web Crypto của Cloudflare Worker yêu cầu PKCS#8):
   ```bash
   openssl pkcs8 -topk8 -inform PEM -outform PEM -nocrypt -in app.pem -out app-pkcs8.pem
   ```

### A3. Tạo KV, nạp secret và triển khai Worker
Chạy trong thư mục `worker/vga-reports/`:
1. Tạo KV namespace:
   ```powershell
   pnpm dlx wrangler kv namespace create REPORTS
   ```
   Chép giá trị `id` nhận được vào `worker/vga-reports/wrangler.toml` thay cho `REPLACE_WITH_KV_NAMESPACE_ID`.
2. Sinh khóa băm IP và khóa ký yêu cầu cho BAT:
   ```bash
   openssl rand -hex 32        # dùng làm IP_SALT
   openssl rand -base64 32     # dùng làm giá trị khóa k1 trong SIGNING_KEYS
   ```
3. Nạp các secret vào Worker:
   ```powershell
   pnpm dlx wrangler secret put GITHUB_APP_ID
   pnpm dlx wrangler secret put GITHUB_APP_INSTALLATION_ID
   pnpm dlx wrangler secret put GITHUB_APP_PRIVATE_KEY    # dán toàn bộ nội dung app-pkcs8.pem
   pnpm dlx wrangler secret put IP_SALT
   pnpm dlx wrangler secret put SIGNING_KEYS              # ví dụ: {"k1":"<chuoi_base64_32_byte>"}
   ```
   Sau khi nạp xong, xóa `app.pem` và `app-pkcs8.pem` khỏi máy nếu không cần lưu dự phòng offline.
4. Triển khai lên Cloudflare:
   ```powershell
   pnpm dlx wrangler deploy
   ```
   Ghi lại URL `https://cmp-vga-reports.<tai-khoan>.workers.dev`.
5. Kiểm tra nhanh:
   ```bash
   curl https://cmp-vga-reports.<tai-khoan>.workers.dev/healthz
   # Kỳ vọng: {"ok":true,"time":...}
   ```
   Để theo dõi lỗi runtime (Worker không log IP hay nội dung báo cáo): `pnpm dlx wrangler tail`.

### A4. Chuyển sang repository thật
1. Đổi `TARGET_REPO = "ngthaihoc/CMP30HXmodtoGEN2"` trong `worker/vga-reports/wrangler.toml`.
2. Vào phần cài đặt của GitHub App, thêm `ngthaihoc/CMP30HXmodtoGEN2` vào danh sách **Only select repositories**.
3. Chạy lại `pnpm dlx wrangler deploy`.
4. Cập nhật 3 hằng trong `Collect_VGA_Report.bat`:
   - `$script:AllowedEndpointHost` (hostname `*.workers.dev` thật)
   - `$script:SigningKeyId` (`'k1'`)
   - `$script:SigningKey` (chuỗi base64 công khai chống spam tương ứng trong `SIGNING_KEYS`)
   *(Lưu ý: khóa ký nằm trong BAT để chống curl/bot quét thông thường, không phải bí mật thật; không bao giờ đưa khóa GitHub App vào BAT).*

---

## 3. Quy trình vận hành và ứng phó sự cố

### Công tắc tắt khẩn cấp (Kill Switch)
1. Đặt `INTAKE_ENABLED = "false"` trong `worker/vga-reports/wrangler.toml`.
2. Chạy `pnpm dlx wrangler deploy` trong `worker/vga-reports/`.
   - Mọi yêu cầu tới `POST /v1/reports` đã ký hợp lệ sẽ nhận `503` (0 lượt ghi KV, 0 gọi GitHub), BAT tự động giữ file `.zip` và `.pending.json` cho người dùng.

### Xoay khóa ký chống spam khi bị lạm dụng (`SIGNING_KEYS`)
1. Sinh khóa mới `k2` (`openssl rand -base64 32`).
2. Tạm thời giữ cả hai khóa để không làm gián đoạn người dùng đang chuyển tiếp:
   `SIGNING_KEYS = {"k1":"<cu>","k2":"<moi>"}` → `pnpm dlx wrangler secret put SIGNING_KEYS`.
3. Cập nhật `$script:SigningKeyId = 'k2'` và `$script:SigningKey` trong `Collect_VGA_Report.bat`, phát hành bản BAT mới.
4. Sau khi hết thời gian chuyển tiếp (hoặc ngay lập tức nếu `k1` bị spam nặng), xóa `k1` khỏi `SIGNING_KEYS` (`{"k2":"<moi>"}`) và chạy `wrangler secret put SIGNING_KEYS`. Bản BAT cũ dùng `k1` sẽ bị từ chối `401 unknown_key` trước khi chạm KV/GitHub.

### Xoay hoặc thu hồi khóa GitHub App (khi nghi ngờ lộ secret)
1. Vào trang quản lý GitHub App → **Private keys** → **Generate a private key** mới → chuyển PKCS#8 bằng `openssl pkcs8`.
2. Chạy `pnpm dlx wrangler secret put GITHUB_APP_PRIVATE_KEY` để nạp khóa mới.
3. Quay lại trang GitHub App → **Delete** khóa cũ ngay lập tức.
4. Để thu hồi hoàn toàn hệ thống: **Uninstall / Suspend** GitHub App khỏi repository và xóa Worker (`pnpm dlx wrangler delete`).

### Xử lý spam trên GitHub Issues và Quota
1. Tạm khóa tiếp nhận bằng `INTAKE_ENABLED = "false"` hoặc hạ `DAILY_CAP` / `IP_LIMIT` trong `wrangler.toml` rồi `wrangler deploy`.
2. Thu hồi `key_id` bị lạm dụng như hướng dẫn xoay khóa ở trên.
3. Đóng/xóa các issue spam trên GitHub (hoặc chặn tài khoản nếu có qua Issue Form thủ công).

### Nâng cấp gói Cloudflare
Nếu lưu lượng cộng đồng tiến gần **1.000 KV writes/ngày**, đo số ghi thực tế (gồm lỗi và thao tác quản trị) trong Cloudflare trước khi nâng gói hoặc tăng `DAILY_CAP`. Workers Paid không loại bỏ giới hạn ghi cùng khóa hay biến KV thành bộ đếm nguyên tử; cần đánh giá lại kiến trúc khi tải tăng.
