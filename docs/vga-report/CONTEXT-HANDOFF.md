# VGA Collector & Modding Research System — Tài liệu Bàn giao Bối cảnh (Context Handoff)

> **Cập nhật:** 2026-11-22  
> **Trạng thái:** Hoàn thành **P0** (Độ tin cậy collector) và **P1** (Schema v2 & Collector cơ bản mở rộng), đã triển khai Worker v2 lên Cloudflare Sandbox và xác minh E2E thành công.  
> **Mục đích:** Tài liệu kỹ thuật chuẩn theo nguyên lý *Context Engineering* nhằm cung cấp toàn bộ bối cảnh hệ thống, kiến trúc dữ liệu, các mốc đã hoàn thành, kết quả kiểm chứng và lộ trình triển khai các pha tiếp theo (P2+).

---

## 1. Tổng quan Dự án & Mục tiêu

Hệ thống **VGA Diagnostic Collector & Modding Research** được xây dựng nhằm thu thập dữ liệu phần cứng chuyên sâu, khách quan và có cấu trúc từ các dòng card đồ họa chuyên dụng và card đào coin (CMP 30HX, CMP 40HX, P106-100, P106-090, Tesla P100 PCIe/SXM2, v.v.) cũng như các dòng card đối chứng (RTX, GTX).

### Mục tiêu cốt lõi:
1. **Phục vụ nghiên cứu unlock / modding:** Phân biệt rõ ràng giữa giới hạn vật lý của GPU, giới hạn khe cắm/PCIe root port/riser, thiết lập BIOS/CSM/ReBAR, phiên bản driver, firmware VBIOS, và ảnh hưởng nhiệt độ/công suất.
2. **Thu thập chỉ đọc (Read-only) an toàn tuyệt đối:** Collector chạy ở chế độ người dùng (user-mode), không tự ý ghi registry, không ghi PCI config space hay MMIO, không ép xung/đổi power limit, không flash VBIOS và không xâm phạm cơ chế bảo mật hệ thống (HVCI/Secure Boot).
3. **Bảo mật & Không thu thập thông tin cá nhân (Zero-PII):** Tuyệt đối không thu thập hostname, username, đường dẫn thư mục người dùng, địa chỉ MAC, IPv4 công khai, IPv6, GPU UUID hay số serial phần cứng.
4. **Vận chuyển dữ liệu tự động & có kiểm soát:** Báo cáo được ký HMAC-SHA256 gửi qua Cloudflare Worker intake để tự động tạo GitHub Issue trên kho lưu trữ mục tiêu với cơ chế chống trùng lặp (dedup), giới hạn hạn mức (quota) và phân loại nhãn tự động.

---

## 2. Kiến trúc Hệ thống & Hợp đồng Dữ liệu

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           MÁY NGƯỜI DÙNG / TESTBED                          │
│                                                                             │
│  [Collect_VGA_Report.bat v1.1.0]                                            │
│   ├── CIM/PnP (Win32_VideoController, Win32_PnPEntity, PnP Problem Code)   │
│   ├── NVIDIA SMI & NVML (18 trường cơ bản + 18 trường extended)             │
│   ├── Security & Boot (UEFI/Legacy, Secure Boot, HVCI, Test Signing)        │
│   ├── Driver Details (Provider, Date, Inf, Problem Code)                    │
│   ├── Legacy Status Parser (gen2_status.txt, cmp30hx_batch_status.txt)      │
│   └── Local Output: JSON v2 + Summary Markdown + ZIP                        │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                      Ký HMAC-SHA256 (X-Report-Signature)
                      Payload JSON v2 (tối đa 2 MiB)
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CLOUDFLARE WORKER INTAKE (v2 Sandbox)                    │
│                 cmp-vga-reports.wellynguyen.workers.dev                     │
│                                                                             │
│   1. Route & Method Check (POST /report, GET /healthz)                      │
│   2. Content-Type (application/json) & Stream Size Limit (2 MiB)            │
│   3. Xác thực Chữ ký HMAC-SHA256 (lệch thời gian cho phép ±300s)            │
│   4. Kiểm tra Công tắc Intake (INTAKE_ENABLED)                              │
│   5. Kiểm định Schema (v1 & v2) & Quét vi phạm PII                          │
│   6. Chống trùng lặp (Dedup qua KV REPORTS: sha256(report_id))              │
│   7. Kiểm tra Quota (IP_LIMIT = 10/ngày, DAILY_CAP = 200/ngày qua KV)       │
│   8. Tạo GitHub Issue qua GitHub App Installation Token (octokit/app)       │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                         Tạo Issue REST API (Issues: Write)
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        GITHUB REPOSITORY (Issues)                           │
│  Hiện tại: ngthaihoc/vga-report-sandbox                                     │
│  Sản xuất: ngthaihoc/CMP30HXmodtoGEN2                                       │
│                                                                             │
│  - Issue #1: Báo cáo v1 (Baseline RTX 2070)                                │
│  - Issue #2: Báo cáo v2 (Đầy đủ trường extended & security)                │
│  - Nhãn tự động: vga-report, hardware-sample, unverified                    │
│  - Marker ẩn chống trùng: <!-- vga-report:<report_id> -->                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.1. Hợp đồng Dữ liệu: Schema v1 so với Schema v2

Tập tin schema gốc tại `docs/vga-report/schema.json` hỗ trợ song song hai phiên bản:
- **`schema_version: 1`:** Bản cơ bản ban đầu, kiểm tra nghiêm ngặt `additionalProperties: false`, lưu trữ thông tin cơ bản của hệ thống và 18 trường `nvidia-smi`.
- **`schema_version: 2`:** Bản mở rộng phục vụ modding và phân tích sâu, bao gồm:
  - `previous_report_id`: Chuỗi UUID liên kết báo cáo trước đó để tạo chuỗi lịch sử thử nghiệm (đọc tự động từ `last_report_id.txt`).
  - `target_bdf`: BDF của GPU được người dùng chọn làm đối tượng nghiên cứu chính (tránh nhầm lẫn trên máy chạy nhiều GPU).
  - `collection_started_utc` & `collection_finished_utc`: Khung thời gian thu thập chuẩn UTC.
  - `source_timestamps`: Thời điểm lấy dữ liệu của từng nguồn riêng biệt (`pnp_utc`, `cim_utc`, `nvidia_smi_utc`, `security_utc`) để giải thích hiện tượng lệch tốc độ PCIe giữa các thời điểm đọc.
  - `evidence_level`: Mức độ tin cậy của bằng chứng (`reported`, `observed`, `compared`, `owner_verified`).
  - `security`: Trạng thái `boot_mode` (`UEFI`/`Legacy`...), `secure_boot`, `hvci_running`, `test_signing`.
  - `completeness`: Đánh giá 7 nhóm dữ liệu (`inventory`, `topology`, `link_idle`, `link_load`, `compute`, `stability`, `comparison`) với danh sách thiếu sót và trạng thái (`ok`, `partial`, `skipped`, `unsupported`, `error`).
  - `legacy_status`: Cho phép đọc an toàn log cũ của dự án (`gen2_status.txt`, `cmp30hx_batch_status.txt`) qua allowlist không chứa PII (`present`, `source`, `status_code`, `error_code`, `speed_current`, `width_current`, `tls_target`, `device_id`, `mrrs_bytes`, `observed_utc`).
  - `context.mod_type`: Loại can thiệp modding (`none`, `driver`, `efi`, `vbios`, `strap`, `software_aio`, `multi`, `unknown`).
  - `gpus[].profile`: Hồ sơ nhận diện từng GPU (`id`, `name`, `family`, `category`, `target_pcie_gen`, `tensor_applicable`, `identity_note`).
  - `gpus[].driver`: Bổ sung `provider`, `date`, `inf_name` (chỉ giữ tên file, loại bỏ toàn bộ đường dẫn), `service`, `problem_code`, `location_path`.
  - `nvidia.extended` (và tùy chọn `gpus[].nvidia.extended`): 18 trường telemetry công suất, nhiệt độ, tải, bộ nhớ, ECC và lý do điều tiết xung nhịp:
    - `power.limit`, `enforced.power.limit`, `power.default_limit`, `power.min_limit`, `power.max_limit`
    - `temperature.gpu.tlimit`, `temperature.memory`
    - `fan.speed`
    - `utilization.gpu`, `utilization.memory`
    - `memory.used`, `memory.free`
    - `ecc.mode.current`
    - `clocks_throttle_reasons.active`, `clocks_throttle_reasons.sw_power_cap`, `clocks_throttle_reasons.hw_slowdown`, `clocks_throttle_reasons.hw_thermal_slowdown`, `clocks_throttle_reasons.hw_power_brake_slowdown` (collector tự ánh xạ fallback từ `clocks_event_reasons.*` sang tên khóa chuẩn này).

### 2.2. Cấu trúc GitHub Issue Chuẩn hóa (Sau P1 Issue Enhancement)

Mỗi GitHub Issue được tạo bởi `renderSummary()` (qua Worker) và bản tóm tắt cục bộ `Format-VietnameseSummary` hiển thị đầy đủ dữ liệu v1/v2 theo 4 lớp tách biệt của `modding-data-plan.md` §4:
1. **Siêu dữ liệu & Định danh:** `report_id`, `experiment_id`, `previous_report_id` (hoặc đánh dấu báo cáo đầu tiên), cửa sổ đo `collection_started_utc` → `collection_finished_utc`, mốc thời gian theo nguồn và thông tin đầy đủ GPU mục tiêu (`family`, `category`, `target_pcie_gen`, `tensor_applicable`, `identity_note`).
2. **Tóm tắt 4 lớp kết luận tách biệt:** (1) Nhận diện (Identity), (2) Cấu hình & Môi trường (Config & Context), (3) Trạng thái đo đa nguồn (Measurement), (4) Workload & Bằng chứng (Evidence). Giữ nguyên mâu thuẫn nguồn kèm cảnh báo, không tự suy diễn vượt quá phép đo.
3. **Chi tiết từng phần:** Nền tảng hệ thống, Bảo mật & Boot, Ngữ cảnh thu thập (kèm lưu ý tái lập nếu còn `unknown`), Danh sách GPU mở rộng (có PCIe PnP, kiến trúc/loại, mục tiêu Gen/Tensor của từng card), Đối chiếu PCIe PnP vs `nvidia-smi`, 18 trường `nvidia-smi` cơ bản + tóm tắt nhanh & 18 trường mở rộng (`M/18` + `extended_missing`), trạng thái `legacy_status` (kể cả khi `present=false`), bảng `completeness` 7 nhóm.
4. **Mẫu bổ sung hồ sơ phần cứng & thử nghiệm (Mục 7 & 9):** Khối `<details>` ở cuối Issue và trường tùy chọn `hardware_context` trong `.github/ISSUE_TEMPLATE/vga-report.yml` để chủ card bổ sung thông tin vật lý (PSU, Riser, PCB/VRAM, gói mod/SHA-256, kết quả sau reboot/sleep-wake/rollback).

---

## 3. Những công việc đã hoàn thành

### Giai đoạn P0 — Khắc phục độ tin cậy Collector
1. **Đồng bộ Device Profile với 40hxcore:** Sửa bảng ánh xạ thiết bị khớp chuẩn `windows-v3.0/tools/40hxcore/const.go`:
   - `10DE:1F0B` -> `CMP 40HX`
   - `10DE:2189` -> `CMP 30HX`
   - Xóa bỏ mã hardcode `1F06` (RTX 2060 SUPER, trước đây gán nhầm là CMP 40HX; nay coi là card lạ/baseline).
2. **Chọn GPU mục tiêu rõ ràng (`Select-TargetGpu`):**
   - Hỗ trợ tham số `-TargetBdf`, biến môi trường `$env:VGA_TARGET_BDF`, và menu tương tác hiển thị danh sách GPU khi hệ thống có từ 2 GPU trở lên.
   - Ngăn chặn việc GPU này mượn nhầm block `nvidia-smi` của GPU khác khi thiếu dòng.
3. **Chuẩn hóa hiển thị PCIe & Cảnh báo:**
   - Chuẩn hóa chuỗi PCIe thành `GenX ×Y` (ví dụ `Gen3 ×16`).
   - Thêm cảnh báo khi có sự mâu thuẫn giữa PnP (`Gen1 ×16`) và `nvidia-smi` (`Gen3 ×16`).
   - Đổi cách dùng từ trên giao diện: từ "18/18 hoàn thành" thành "18/18 trường NVIDIA cơ bản".

### Giai đoạn P1 — Schema v2 và Collector cơ bản mở rộng
1. **Xây dựng Schema & Fixture Dual-version:**
   - `docs/vga-report/schema.json` hỗ trợ song song v1 và v2 thông qua `allOf` + `if/then` có điều kiện.
   - Xuất khẩu các helper trong `scripts/vga-report/report.cjs`: `identifyGpuProfile`, `getGpuProfileDetails`, `NVIDIA_EXTENDED_FIELDS`.
   - Bổ sung hồ sơ nhận diện cho: `P106-100` (`10DE:1C07`), `P106-090` (`10DE:1C09`), `Tesla P100 PCIe 16GB` (`10DE:15F8`), `Tesla P100 PCIe 12GB` (`10DE:15F7`), `Tesla P100 SXM2` (`10DE:15F9`), kèm cảnh báo khi tên card chứa "P100" nhưng PCI ID không thuộc kiến trúc GP100.
2. **Nâng cấp `Collect_VGA_Report.bat` lên v1.1.0:**
   - Đóng gói cách ly lỗi `try/catch` cho từng GPU trong `Get-VgaInventory`: lỗi trên một card không làm mất dữ liệu các card còn lại.
   - Thêm hàm `Get-SecurityAndBootInfo` thu thập trạng thái bảo mật chỉ đọc.
   - Truy vấn 18 trường `nvidia.extended` với cơ chế fallback giữa `clocks_event_reasons` và `clocks_throttle_reasons`.
   - Parser allowlist cho `Get-LegacyUnlockStatus` đọc `gen2_status.txt` và `cmp30hx_batch_status.txt`.
   - Liên kết tự động báo cáo trước qua `last_report_id.txt` và thêm câu hỏi chọn `mod_type`.
   - Thêm bước kiểm tra PII preview hiển thị cho người dùng trước khi lưu hoặc gửi.
   - Đảm bảo định dạng UTF-8 không BOM và xuống dòng CRLF chuẩn Windows.
3. **Cập nhật Cloudflare Worker Intake:**
   - Hỗ trợ giải mã và kiểm định cả schema v1 lẫn v2.
   - Cập nhật `buildSafeIssueTitle` ưu tiên tên GPU khớp với `target_bdf`.
   - Hiển thị bảng Extended Power & Thermal trong Markdown body của GitHub Issue.

### Triển khai & Xác minh Thực tế trên Cloudflare Sandbox
1. **Triển khai Cloudflare Worker:**
   - Đã chạy `pnpm dlx wrangler deploy` trong `worker/vga-reports`.
   - **URL triển khai:** `https://cmp-vga-reports.wellynguyen.workers.dev`
   - **Version ID mới nhất:** `9dc489d9-4359-41ad-b6ab-8f57f94b9399` (`TARGET_REPO = "ngthaihoc/vga-report-sandbox"`, đã bao gồm giao diện Issue 4 lớp + sửa bảng GFM).
   - Endpoint `/healthz` trả về `{"ok":true,"time":...}` (HTTP 200).
2. **Kiểm thử đầu cuối trực tiếp (Live E2E Verification):**
   - **Lần gửi P1 mới (`Issue #4`):** Gửi từ `Collect_VGA_Report.bat` chế độ UX 1 bước (`69cbe3b4-...`), kiểm chứng lọc PII đệ quy và không tạo file `experiment_id.txt`.
   - **Lần gửi P2 mới (`Issue #5`):** Gửi từ `Collect_VGA_Report.bat -RunLoadTest` (`27f5116a-...`), xác minh [Issue #5](https://github.com/ngthaihoc/vga-report-sandbox/issues/5) nâng cấp mức bằng chứng `observed` (`4/7` nhóm đạt `ok`), đo được thực tế trên RTX 2070: `H2D=12.03 GB/s, D2H=12.59 GB/s (64 MiB, pinned=True)` và `CUDA PTX FP32 OK (41.93 GFLOPS, CC 7.5)`.

---

## 4. Ma trận Kiểm chứng & Kết quả Thử nghiệm

Tất cả các bộ kiểm thử cục bộ và đầu cuối đều đạt 100% PASS:

| Bộ kiểm tra | Lệnh thực thi | Kết quả | Ghi chú |
| :--- | :--- | :---: | :--- |
| **Shared Validator & Worker Unit Tests** | `node --test scripts/vga-report/report.test.cjs worker/vga-reports/test/worker.test.cjs` | **55/55 PASS** | Kiểm tra validator, allowlist, PII, schema v1/v2, cấu trúc Issue 4 lớp, bảng GFM 2 cột, HMAC, rate limit |
| **Collector PowerShell Test Suite** | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/Test-VGA-Collector.ps1` | **42/42 PASS** | Bổ sung test `-KeepHistory` opt-in, `Redact-ReportObject` giữ nguyên schema, và mock P2 `vgabench` |
| **Real CMD Smoke Test** | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/Test-VGA-Collector-Batch.ps1` | **4/4 PASS** | Chạy BAT thật chế độ 1 bước trong cmd.exe, xác nhận không tạo tệp theo dõi mặc định |
| **P2 Helper Go Tests & Vet (`vgabench`)** | `Push-Location windows-v3.0/tools/vgabench; go test -cover ./...; go vet ./...` | **5/5 PASS** | Pure Go + `nvcuda.dll` + embedded PTX (`sm_50`), mock `CudaDriver` (coverage 44.8%) |
| **Telemetry Test** | `node --test windows-v3.0/tools/aio/tests/telemetry.test.cjs` | **1/1 PASS** | Đảm bảo không xung đột với telemetry hiện hữu |
| **Git Diff Format Check** | `git diff --check` | **0 errors** | Đảm bảo không có lỗi trailing whitespace hoặc định dạng CRLF |

---

## 5. Bản đồ File & Các điểm lưu ý kỹ thuật (Gotchas)

### 5.1. Bản đồ File thay đổi
- `Collect_VGA_Report.bat`: Script giao diện dòng lệnh chính thu thập thông tin trên Windows.
- `docs/vga-report/schema.json`: JSON schema hợp nhất cho cả v1 và v2.
- `scripts/vga-report/report.cjs`: Module kiểm định dữ liệu, làm sạch PII, nhận diện hồ sơ GPU và render Markdown summary 4 lớp.
- `scripts/vga-report/report.test.cjs`: Bộ test cho validator và renderer.
- `.github/ISSUE_TEMPLATE/vga-report.yml`: Biểu mẫu Issue dự phòng kèm phần khai báo hồ sơ phần cứng & thử nghiệm bổ sung.
- `worker/vga-reports/src/index.js`: Mã nguồn Cloudflare Worker intake tiếp nhận báo cáo và đẩy lên GitHub Issues.
- `worker/vga-reports/test/worker.test.cjs`: Bộ test chuyên sâu cho Worker intake.
- `worker/vga-reports/wrangler.toml`: Cấu hình Cloudflare Worker, KV binding và biến môi trường.
- `scripts/Test-VGA-Collector.ps1`: Unit test cho các hàm PowerShell bên trong collector.
- `scripts/Test-VGA-Collector-Batch.ps1`: Integration test chạy `Collect_VGA_Report.bat` qua `cmd.exe`.
- `windows-v3.0/tools/vgabench/`: Công cụ Go thuần (`nvcuda.dll` + PTX nhúng sẵn, không cần cài CUDA Toolkit) đo băng thông PCIe H2D/D2H và kiểm tra tính đúng FP32 cho P2 (`-RunLoadTest`).
- `docs/vga-report/modding-data-plan.md`: Kế hoạch dữ liệu modding từ P0 đến P5.
- `docs/vga-report/runbook.md` & `docs/vga-report/policy.md`: Tài liệu vận hành và chính sách bảo mật/quyền riêng tư.

### 5.2. Các điểm lưu ý kỹ thuật sống còn
1. **Định dạng file BAT trên Windows:**
   - `Collect_VGA_Report.bat` **bắt buộc** phải là file UTF-8 **không có BOM** và kết thúc dòng bằng **CRLF**.
   - Nếu file bị chuyển sang LF, `cmd.exe` sẽ đọc sai vị trí byte và nhảy thẳng vào đoạn code PowerShell ở đuôi file, dẫn đến lỗi thoát exit code 255.
2. **Xử lý mảng trong PowerShell `ConvertTo-Json`:**
   - Trong PowerShell 5.1, biểu thức điều kiện gán mảng một phần tử (ví dụ `if ($c) { @('a') } else { @('b') }`) có thể bị ép kiểu ngầm thành một chuỗi đơn (scalar string) thay vì mảng.
   - Khi chuyển đổi sang JSON, JSON Schema sẽ từ chối vì trường yêu cầu `type: "array"`.
   - **Giải pháp đã áp dụng:** Sử dụng hàm helper đóng gói kiểu mảng tường minh `[string[]]@(...)` hoặc khởi tạo danh sách kiểu `System.Collections.Generic.List[string]`.
3. **Giới hạn kích thước GitHub Issue Body:**
   - GitHub Issue body có giới hạn tối đa 65.536 ký tự.
   - Renderer áp dụng ngưỡng an toàn cứng: 60.000 ký tự (bao gồm marker ẩn). Dữ liệu chi tiết dạng mảng lớn hoặc chuỗi log phải được cắt gọn hoặc lưu trong file đính kèm cục bộ, không đẩy vượt ngưỡng lên GitHub.

---

## 6. Ranh giới An toàn & Quy tắc Bất di bất dịch

Khi tiếp tục phát triển hệ thống hoặc chuyển sang các pha tiếp theo (P2+), các kỹ sư và agent hỗ trợ **bắt buộc** tuân thủ các nguyên tắc sau:

1. **Chuỗi giao diện tiếng Việt có dấu:** Mọi thông báo người dùng, menu lựa chọn và tóm tắt hiển thị phải bằng tiếng Việt chuẩn.
2. **Chế độ Chỉ đọc (Read-only Boundary):** Không bao giờ tự động ghi thanh ghi PCI/MMIO, không ghi registry, không tắt cơ chế bảo vệ hệ thống (HVCI/Secure Boot), và không can thiệp vào driver khi chưa có sự xác nhận của người dùng.
3. **Không xử lý Bí mật (No Secret Handling):** Tuyệt đối không yêu cầu người dùng dán private key `.pem`, GitHub tokens, hoặc `IP_SALT` vào hội thoại chat. Chỉ làm việc với các định danh không bí mật (App ID, Installation ID, Worker URL, Public Key ID).
4. **Không tự ý Release / Push:** Không tự ý push mã nguồn lên nhánh chính của remote hoặc tạo GitHub Release khi chưa có sự phê duyệt và nghiệm thu phần cứng thực tế từ chủ dự án.

---

## 7. Lộ trình Triển khai Tiếp theo: Pha P2 (Đo Dưới Tải & So sánh Diff)

Sau khi P0 và P1 đã hoàn thành và xác minh ổn định, giai đoạn tiếp theo cần triển khai là **P2 — Đo Idle/Load và So sánh Đối chứng**:

### 7.1. Các mục tiêu cụ thể trong P2
1. **Module đo tải nhẹ và an toàn (Controlled Load Benchmark):**
   - Đo trạng thái nhàn rỗi (Idle) trong 10 giây (lấy mẫu xung nhịp, công suất, nhiệt độ, PCIe link speed).
   - Chạy tải kiểm tra có kiểm soát (Load) tối đa 30 giây (copy bộ nhớ Host-to-Device / Device-to-Host, kernel tính toán cơ bản).
   - Đo thời gian phục hồi (Recovery) trong 10 giây.
   - **Cơ chế Watchdog:** Tự động ngắt tải ngay lập tức nếu nhiệt độ vượt ngưỡng an toàn hoặc GPU gặp sự cố TDR.
2. **Đo Băng thông PCIe thực tế (H2D / D2H Bandwidth):**
   - Đo tốc độ truyền dữ liệu qua bộ nhớ pinned (GB/s) theo cả 2 chiều H2D và D2H để phát hiện nghẽn ở khe cắm, riser hoặc cấu hình PCIe Gen.
3. **Kiểm tra Tính đúng đắn của Tính toán (Compute & Tensor Correctness):**
   - Kiểm tra khả năng thực thi kernel CUDA FP32 cơ bản.
   - Đối với dòng card hỗ trợ (hoặc cần kiểm chứng Tensor như CMP 40HX): chạy kernel matrix multiplication tương ứng để xác định khả năng tính toán thực tế thay vì chỉ nhìn vào thông số quảng cáo.
4. **Bộ phân tích So sánh Trước/Sau (Diff Analyzer):**
   - So sánh giữa 2 báo cáo (dựa vào `previous_report_id` hoặc chọn 2 báo cáo local) để chỉ ra thay đổi về xung nhịp, PCIe speed, nhiệt độ và công suất trước và sau khi áp dụng mod.

### 7.2. Các quyết định cần chủ dự án phê duyệt trước khi lập trình P2
- **Tên và vị trí đặt công cụ helper CUDA:** Dự kiến đặt mã nguồn helper dưới `windows-v3.0/tools/` (ví dụ `windows-v3.0/tools/vga-benchmark/` hoặc tích hợp vào `40hxcore`).
- **Hình thức phân phối binary helper:** Chỉ đóng gói sẵn file thực thi kèm hash kiểm tra SHA-256 hay cho phép tải về từ GitHub Release đã ký?
- **Ngưỡng nhiệt độ và công suất an toàn:** Thiết lập ngưỡng dừng khẩn cấp cho từng dòng card (ví dụ: ngắt benchmark nếu GPU temp > 85°C hoặc VRAM temp > 105°C).

---
*Tài liệu được khởi tạo và chuẩn hóa bởi hệ thống Antigravity Context Engineering.*
