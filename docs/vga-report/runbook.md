# Runbook vận hành — Collector VGA (`worker/vga-reports`)

Tài liệu hướng dẫn cài đặt, giới hạn gói miễn phí đã kiểm chứng, xoay khóa, tắt khẩn cấp và xử lý sự cố cho hệ thống tiếp nhận báo cáo VGA.

## 1. Giới hạn gói miễn phí Cloudflare (đã kiểm chứng 2026-11-21)

Nguồn tham chiếu: tài liệu chính thức Cloudflare Workers & Workers KV (`developers.cloudflare.com`):

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

- Yêu cầu không có chữ ký hoặc sai chữ ký bị `verifySignature` từ chối `401` **trước khi đọc/ghi KV** (0 lượt ghi KV).
- Mỗi báo cáo đi qua quota và tạo issue thực hiện tối đa **3 lượt ghi KV**: 1 lần tăng bộ đếm ngày `d:<day>`, 1 lần tăng bộ đếm IP `q:<day>:<hash>`, 1 lần lưu chống trùng `r:<report_id>`.
- Với `DAILY_CAP = 200`, tổng ghi KV tối đa khoảng **600 writes/ngày** (còn dư ~400 writes/ngày cho yêu cầu lỗi schema/PII hợp lệ chữ ký hoặc thử nghiệm), an toàn dưới trần **1.000 KV writes/ngày** của gói Free.

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
Nếu lưu lượng cộng đồng vượt **1.000 KV writes/ngày** (tương đương > ~330 báo cáo/ngày) hoặc cần giới hạn cao hơn, nâng cấp lên gói **Workers Paid** trong bảng điều khiển Cloudflare, sau đó nâng `DAILY_CAP` trong `wrangler.toml` và triển khai lại.
