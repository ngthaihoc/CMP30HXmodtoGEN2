# VGA Collector & Modding Research System — Tài liệu Bàn giao Bối cảnh (Context Handoff)

> **Cập nhật:** 2026-11-22
> **Trạng thái:** Đã hoàn thành triển khai **P0**, **P1**, và **Phase 2 & 3 (Topology/Stability/LoadTest/Comparison)** trên nhánh `feat/vga-report-phase2-3`, kiểm chứng với mock suite + RTX 2070 tham chiếu và redeploy Worker v2 lên Cloudflare Sandbox. Nghiệm thu trên phần cứng CMP/mining thật (`CMP 30HX`, `CMP 40HX`, `P106-100`, `P100 PCIe`) vẫn đang chờ phần cứng thực tế (`pending_hardware`).
> **Mục đích:** Tài liệu kỹ thuật chuẩn theo nguyên lý *Context Engineering* nhằm cung cấp toàn bộ bối cảnh hệ thống, kiến trúc dữ liệu, các mốc đã hoàn thành, giới hạn chỉ-đọc, kết quả kiểm chứng và ma trận nghiệm thu phần cứng.

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
  - Các khối mở rộng tùy chọn Phase 2 & 3 (chỉ cho phép trên `schema_version: 2`):
    - `topology`: `upstream_chain`, `root_port` (từ `DEVPKEY_Device_Parent`), `aspm` (chính sách Windows `powercfg`, phân tách với thanh ghi `LnkCtl` chưa đọc trực tiếp ở user-mode), `pcie_caps` (`MPS`/`MRRS`), `rebar` (`bar1_total_mib` từ XML `nvidia-smi`, WMI `Win32_DeviceMemoryAddress`, giữ `rebar_capability_status: "unknown"` nếu không đọc được thanh ghi HW ReBAR Cap).
    - `load_test`: chu kỳ `idle -> load -> recovery`, kết quả băng thông `H2D/D2H` (pinned/pageable, min/median/max), kiểm tra FP32 SAXPY (`math.IsNaN`/`IsInf` fail-closed), ngưỡng watchdog nhiệt/công suất/timeout, và tối đa 120 mẫu time-series (bản đầy đủ lưu cục bộ vào `*-timeseries.json` và `timeseries-full.json` trong ZIP).
    - `stability`: truy vấn Event Log `System` có giới hạn cửa sổ thời gian cho `Microsoft-Windows-WHEA-Logger` (ID 17/18/19) và `nvlddmkm` (ID 13/14/153) + `Display` (ID 4101), phân tách `permission_denied`, `out_of_window`, `other_gpu`, `non_pcie_whea`, `unattributed`.
    - `comparison`: ghép cặp với báo cáo trước trong cùng `experiment_id`, khớp danh tính GPU ngay cả khi BDF dịch chuyển (từ chối nếu có ≥ 2 GPU giống hệt nhau gây mơ hồ), kiểm tra thứ tự `stage` (`baseline -> before_unlock -> after_unlock -> troubleshooting`), phát hiện yếu tố gây nhiễu (`driver`, `vbios`, `bios`, `riser`, `slot`, `method`, `pinned_host`, `transfer_size_mb`, `iterations`), tính `deltas` tương thích và chỉ nâng `evidence_level = "compared"` khi mọi điều kiện đạt chuẩn.

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
   - **Version ID mới nhất:** `4952fa18-5e50-4661-b761-16177bbaf7bf` (`TARGET_REPO = "ngthaihoc/vga-report-sandbox"`, bao gồm `GET /v1/capabilities` và phần render `topology`/`load_test`/`stability`/`comparison`).
   - Endpoint `/healthz` trả về `{"ok":true,"time":...}` (HTTP 200) và `/v1/capabilities` trả `schema_versions:[1,2]`, `v2_extensions:["topology","load_test","stability","comparison"]`.
2. **Kiểm thử đầu cuối & Thử nghiệm ghép đôi tham chiếu RTX 2070:**
   - **Sandbox GitHub Issues:** `#4` (P1) và `#5` (`27f5116a-...`, P2 `observed`, `H2D=12.03 GB/s`, `D2H=12.59 GB/s`, `41.93 GFLOPS`).
   - **Thử nghiệm ghép đôi 2 lần (cục bộ offline trên RTX 2070, 2026-11-22):** chạy `Collect_VGA_Report.bat -RunLoadTest -KeepHistory -DetailedContext -SaveOnly -IdleSec 2 -LoadSec 4 -RecoverySec 2` cùng `vgabench.exe` cạnh BAT:
     - Lần 1 (`before_unlock`, `d32bab00-6446-44d4-80da-d1dd21acdb1e`): `evidence_level = "observed"`, `H2D = 13.23 GB/s`, `D2H = 13.34 GB/s`, `FP32 = 40.2 GFLOPS`, `9` mẫu time-series, `stability = ok`, `comparison = skipped`.
     - Lần 2 (`after_unlock`, `8d5a20a3-e4e4-46e0-857b-3c66a6df9ef8`, cùng `experiment_id = 2fd2a5a6-948f-4c73-accf-4577dddd4001`): `evidence_level = "compared"`, `previous_report_id = d32bab00-...`, `comparison.status = "ok"`, `deltas`: `pcie_gen_idle=0`, `h2d_median_gbps=-0.1`, `d2h_median_gbps=-0.26`, `fp32_gflops=0.88`, `bar1_total_mib=0`.
     - Cả 2 báo cáo đạt `validateReport ok=true`, `findPii=[]` trên Node (`report.cjs`). `topology` giữ `partial` (thiếu `tls`, `aspm_hw_state`, `rebar_capability` vì không đọc thanh ghi HW trực tiếp ở user-mode) và `compute` giữ `partial` trên RTX 2070 (`tensor_applicable=unknown`).

---

## 4. Ma trận Kiểm chứng & Nghiệm thu Phần cứng

| Đối tượng / Bộ kiểm tra | Phạm vi / Lệnh thực thi | Trạng thái | Ghi chú |
| :--- | :--- | :---: | :--- |
| **Shared Validator & Worker & Telemetry** | `pnpm exec node --test scripts/vga-report/report.test.cjs worker/vga-reports/test/worker.test.cjs windows-v3.0/tools/aio/tests/telemetry.test.cjs` | **58/58 PASS** | Validator v1/v2, Phase 2/3 sections, Worker `/v1/capabilities`, AIO telemetry |
| **Collector PowerShell Test Suite** | `powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/Test-VGA-Collector.ps1` | **52/52 PASS** | PII/offline, `System32` loader, timeout, topology, stability, load workflow, comparison |
| **Real CMD Smoke Test** | `powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/Test-VGA-Collector-Batch.ps1` | **6/6 PASS** | BOM/CRLF, save, placeholder offline, exit, `-SaveOnly`, 2-run `-KeepHistory -DetailedContext` |
| **P2 Helper Go (`vgabench`)** | `windows-v3.0/tools/vgabench`: `go test -cover ./...`, `go vet ./...`, `go build ./...` | **10/10 PASS (46.6%)** | Pure Go + `System32\nvcuda.dll`, `-duration-ms`, check non-finite FP32 |
| **RTX 2070 (`10DE:1F07`)** | Máy tham chiếu + Sandbox | **Verified Reference** | Đạt `observed` & `compared` 2 lần ghép đôi; không đại diện cho card CMP |
| **CMP 30HX (`10DE:2189`)** | `Collect_VGA_Report.bat` + `vgabench.exe` + Gen2/reboot/sleep | **`pending_hardware`** | Chưa gắn card thật để nghiệm thu |
| **CMP 40HX (`10DE:1F0B`)** | `Collect_VGA_Report.bat` + `vgabench.exe` + Tensor/BAR1 | **`pending_hardware`** | Chưa gắn card thật để nghiệm thu |
| **P106-100 / P106-090 (`10DE:1C07`/`10DE:1C09`)** | `Collect_VGA_Report.bat` + `vgabench.exe` | **`pending_hardware`** | Chưa gắn card thật để nghiệm thu |
| **Tesla P100 PCIe (`10DE:15F8`/`15F7`)** | `Collect_VGA_Report.bat` + `vgabench.exe` | **`pending_hardware`** | Chưa gắn card thật để nghiệm thu |

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

## 7. Hạng mục còn lại & Điều kiện chuyển Production

Phần lớn kiến trúc P2 & P3 đã được triển khai trên nhánh `feat/vga-report-phase2-3`. Các hạng mục còn chờ trước khi mở PR vào `main`, chuyển `TARGET_REPO` sang `ngthaihoc/CMP30HXmodtoGEN2`, hoặc phát hành release:

1. **Nghiệm thu phần cứng đích (`pending_hardware`):** Chạy `Collect_VGA_Report.bat -RunLoadTest -KeepHistory -DetailedContext` trên máy có `CMP 30HX`, `CMP 40HX`, `P106-100`/`090`, và `Tesla P100 PCIe` (bao gồm chu kỳ trước/sau unlock và sau reboot/sleep-wake).
2. **Đo thanh ghi phần cứng cấp thấp (chờ phần cứng & quyết định phân quyền):** Nếu cần đưa `topology.rebar_capability_status`, `tls`, hoặc `aspm.endpoint_link_control` từ `partial` sang `ok`, cần tích hợp đọc an toàn có xác nhận từ `40hxcore`/driver thay cho user-mode BAT thuần túy.
3. **Kiểm tra Tensor chuyên biệt (`tensor_check`):** Xem xét bổ sung kernel kiểm tra Tensor thực tế cho kiến trúc Turing/Volta nếu cần nâng `compute` lên `ok` trên card có `tensor_applicable != "no"`.
4. **Rà soát PR & Merge:** Chờ review liên-agent trên nhánh `feat/vga-report-phase2-3`, giữ nguyên 6 `.exe` sản xuất trên `main` và chỉ đóng gói `vgabench.exe` qua CI release hoặc phê duyệt rõ ràng.

---
*Tài liệu được khởi tạo và chuẩn hóa bởi hệ thống Antigravity Context Engineering.*
