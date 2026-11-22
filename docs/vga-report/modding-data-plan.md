# Kế hoạch collector phục vụ nghiên cứu unlock/mod P106, P100, CMP và GPU đào coin

Ngày lập: 2026-11-21. Trạng thái: P0 và P1 đã hoàn thành cục bộ (chưa commit/deploy; schema v1+v2 tương thích ngược); chờ duyệt báo cáo mẫu P1 và chốt quyết định mở trước khi triển khai P2. Các giai đoạn P2–P5 giữ trạng thái đề xuất.

## 1. Mục tiêu và bằng chứng hiện có

Mục tiêu là tạo bộ bằng chứng đủ để phân biệt: giới hạn GPU, giới hạn khe/root port/riser, cấu hình BIOS, driver, firmware, nhiệt/nguồn và hiệu quả của một thay đổi mod cụ thể. Không cam kết một lần chạy tự động có thể tìm ra mọi khóa phần cứng hoặc bảo đảm unlock thành công.

Luồng gửi báo cáo v1 đã hoạt động trên sandbox: [Issue #1](https://github.com/ngthaihoc/vga-report-sandbox/issues/1) được tạo với HTTP 201; gửi lại cùng report_id trả HTTP 200 duplicate và cùng URL. Đây là bằng chứng về vận chuyển dữ liệu, chưa phải bằng chứng unlock CMP.

Báo cáo RTX 2070 hiện có:
- Nhận diện platform, PCI ID/BDF, driver, VBIOS và 18 trường nvidia-smi.
- Chưa có so sánh trước/sau, PCIe root port, đo dưới tải, throughput, ReBAR thực tế hay thử tính toán.
- Ngữ cảnh mod/riser/khe đều unknown, không đủ để tái lập phép thử.
- PCIe tóm tắt Gen1 nhưng bảng nvidia-smi Gen3. Hai nguồn có thể đo khác thời điểm; chưa có timestamp/source chi tiết nên chưa kết luận nguồn nào sai.
- Đã đối chiếu code (2026-11-21): [Collect_VGA_Report.bat](../../Collect_VGA_Report.bat) dòng 463–465 và 637 ưu tiên `2189` (30HX) và `1F06` (gán là 40HX), trong khi [const.go](../../windows-v3.0/tools/40hxcore/const.go) dòng 10–12, 41, 52 dùng `1F0B` (40HX) và `2189` (30HX). `1F06` có khả năng là SKU GeForce khác, cần đối chiếu pci.ids trước khi kết luận; với collector hiện tại, một RTX mang ID đó có thể bị chọn làm “GPU chính”. Phải sửa trước khi mở rộng.
- Giả thuyết cho mâu thuẫn Gen1/Gen3: `nvidia-smi` báo link hiện tại và có thể hạ khi nhàn rỗi (xem NVIDIA SMI ở mục 8). Cần timestamp, trạng thái tải và cả `max` lẫn `current` ở mỗi nguồn mới kết luận; chưa coi là lỗi nguồn nào.
- Giới hạn đã có trong [report.cjs](../../scripts/vga-report/report.cjs) dòng 4–7: 2 MiB, 16 GPU, chuỗi 500 ký tự, summary 60000 ký tự. Kế hoạch v2 giữ đúng các hằng này.
- Chuỗi “Gen 3 gen x16 lanes” cần chuẩn hóa thành “Gen3 ×16”; dữ liệu gốc và đơn vị vẫn lưu riêng.

18/18 chỉ là mức đầy đủ của nhóm nvidia-smi v1, không biểu thị “đủ dữ liệu modding”.

## 2. Các chế độ thu thập

| Chế độ | Công việc | Điều kiện chạy |
| --- | --- | --- |
| Cơ bản | Đọc Windows/PnP/CIM, truy vấn NVIDIA, kiểm tra cấu hình hiện tại | Mặc định, không cần tự nâng quyền |
| Đo dưới tải | Copy host/device, kernel/CUDA, chuỗi mẫu khi chạy tải | Người dùng chủ động chọn GPU, bài thử và thời lượng; xem điều kiện dừng trước khi chạy |
| Chẩn đoán chuyên sâu | Đọc PCI capability/root port, BAR/ReBAR, thanh ghi được cho phép | Giai đoạn riêng; chỉ chạy khi có helper đã kiểm chứng, người dùng xác nhận nếu cần driver/quyền quản trị |
| Hồ sơ phần cứng | Ảnh PCB/VRAM, nhãn chip, lịch sử strap/flash, file ROM tùy chọn | Chủ card cung cấp và duyệt nội dung trước khi chia sẻ |

Các chế độ không tự flash VBIOS, sửa strap, ghi PCI/MMIO, reset PnP, retrain link, đổi power limit/clock/BIOS/security hoặc cài persistence. Kế hoạch thu thập dữ liệu không tự cho phép các thao tác mod đó. Theo quy tắc dự án, thay đổi bảo mật/persistence/driver cần cảnh báo tiếng Việt, xác nhận riêng, log trước/sau và rollback.

Không tự gọi công cụ cài đặt/unlock hiện tại để “lấy log”, vì có thể kích hoạt đường ghi phần cứng. Trước khi tái sử dụng 40hxcore phải kiểm tra đường gọi chỉ đọc từ đầu đến cuối.

## 3. Dữ liệu cần bổ sung và câu hỏi mà nó giải quyết

| Nhóm | Dữ liệu | Mục đích / nguồn dự kiến |
| --- | --- | --- |
| Nhận diện từng GPU | BDF, VEN/DEV/SUBSYS/REV, tên từ PCI và driver, SKU/family được nhận diện, GPU đích do người dùng chọn | Tránh nhầm card hiển thị với CMP; PnP/CIM + bảng profile của dự án |
| Platform | CPU/chipset, board/BIOS version/date, OS build/architecture, boot UEFI/Legacy | Phân biệt giới hạn platform; nguồn Windows có trạng thái hỗ trợ riêng |
| Thiết lập firmware | CSM, Above 4G, ReBAR, tốc độ khe đặt trong BIOS | Tách giá trị đọc được, người dùng khai báo và chưa biết; không suy ra từ model board |
| Bảo mật/driver | Secure Boot, HVCI/Memory Integrity đang chạy, test signing, driver version/provider/date/INF/catalog/signature nếu truy vấn được, PnP problem code | Chẩn đoán driver không tải/Code 43; chỉ đọc, không tắt bảo mật |
| Đường PCIe | Endpoint, từng bridge upstream/root port, BDF, capability max/current speed/width, TLS, ASPM, MPS/MRRS | Xác định mắt xích nghẽn; nguồn OS trước, helper PCI chỉ đọc ở giai đoạn sâu |
| Liên kết thực | Chuỗi mẫu idle/load/recovery: Gen, width, utilization, P-state, clocks, nhiệt/nguồn, replay/error counters nếu hỗ trợ | Phân biệt hạ tốc tiết kiệm điện với kẹt link |
| BAR/ReBAR | BAR index/type/size, resource allocation; ReBAR capability/supported sizes/current size ở endpoint và trạng thái sau reboot | Xác minh cấu hình thực; BAR1 memory từ NVML chỉ là dữ liệu bổ sung |
| VRAM/VBIOS | Dung lượng/loại/bus width khi nguồn hỗ trợ, VBIOS version, ROM hash nếu có bản dump, board revision/VRAM marking do chủ card cung cấp | Phân biệt biến thể PCB/VRAM và lịch sử flash; không đoán model chip nhớ từ tên GPU |
| Năng lực tính toán | CUDA driver/runtime, compute capability, SM count, memory bus/clock khi hỗ trợ, kernel correctness, lỗi runtime | Xác nhận driver có thực thi được workload; thuộc tính quảng cáo không chứng minh throughput |
| Tensor trên 40HX | Dữ liệu chẩn đoán thanh ghi có nguồn, CUDA tensor workload đúng kiến trúc/precision, kiểm tra kết quả và thời gian | Đánh giá nhiều lớp bằng chứng; không lấy tên GPU/compute_cap hoặc một magic value làm chứng minh hiệu năng |
| Hiệu năng | H2D/D2H, device copy, compute FP32 và Tensor phù hợp, VRAM correctness giới hạn dung lượng | Phân biệt PCIe bottleneck, memory và compute; ghi phương pháp/đơn vị |
| Ổn định | WHEA/display/TDR được lọc theo cửa sổ thử, lỗi CUDA, link/error counters trước/sau, crash/timeout | Phát hiện lỗi chỉ xuất hiện dưới tải hoặc sau resume |
| Lịch sử thử | Baseline / sau mod / sau reboot / sau sleep-wake / sau rollback | Xác minh thay đổi bền vững hay chỉ có hiệu lực một phiên |
| Ngữ cảnh vật lý | Khe, riser/model/chiều dài, cấp nguồn/PSU model do người dùng khai báo, mod phần mềm/EFI/strap/flash và phiên bản | Giúp tái lập thử nghiệm, giữ unknown khi không có thông tin |

Lưu ý nguồn dữ liệu dễ sai:
- `Win32_VideoController.AdapterRAM` là giá trị 32-bit, bị chặn ở 4 GiB; không dùng cho VRAM. Dùng `nvidia-smi memory.total`/NVML, ghi nguồn.
- Windows không có API chính thức cho trạng thái ReBAR; Above 4G/BIOS và BAR1 NVML chỉ là gián tiếp. Kết luận ReBAR cần capability PCI hoặc resource thực (P3), nếu không ghi `unknown`.
- Secure Boot/HVCI/test signing đọc bằng nguồn riêng, mỗi nguồn có trạng thái `unsupported`/`permission_denied` nếu thiếu quyền (ví dụ Secure Boot trên Legacy BIOS).
- Sau reboot BDF có thể đổi; không dùng BDF làm khóa dài hạn.

API và driver không hỗ trợ trường nào thì ghi rõ lý do. Không ép mọi card phải trả đủ cùng bộ trường. Không xuất serial, GPU UUID, MAC, hostname, username hay đường dẫn cá nhân để nhận diện card; dùng mã ngẫu nhiên trong một experiment.

## 4. Hợp đồng dữ liệu v2

- Giữ report_id cho một báo cáo, experiment_id để liên kết các giai đoạn, thêm previous_report_id, stage và target GPU.
- Mỗi phép đo có BDF, nguồn, thời điểm UTC, thời gian tương đối monotonic, đơn vị, giá trị gốc và giá trị chuẩn hóa.
- Ánh xạ trước/sau reboot bằng mã GPU trong experiment + PCI IDs/ngữ cảnh khe; BDF có thể đổi. Khi hai card giống nhau không phân biệt được, yêu cầu xác nhận thay vì ghép nhầm.
- Status: ok, unsupported, permission_denied, skipped, timeout, error. Mã lỗi và lý do có cấu trúc; không thay dữ liệu thiếu bằng 0.
- Ghi collector version, commit/build hash, helper version/hash, thời gian bắt đầu/kết thúc và manifest SHA-256 của artifact.
- Đủ dữ liệu theo từng nhóm: inventory, topology, link_idle, link_load, compute, stability, comparison. Mỗi nhóm có missing và reason.
- Mức bằng chứng: reported (nguồn tự báo), observed (đã đo), compared (đã đối chiếu cùng quy trình), owner_verified (chủ phần cứng xác nhận). Không tự nâng mức từ 18/18.
- GPU không phải NVIDIA vẫn có inventory; dữ liệu NVIDIA unsupported/skipped không được tính thành “hỏng”.
- Dữ liệu chuyên sâu chưa có phải xuất skipped với lý do; không cản gửi bản cơ bản.
- Tách kết luận: nhận diện, trạng thái cấu hình, trạng thái đo và kết quả workload. Giữ nguyên mâu thuẫn nguồn kèm cảnh báo, không tự chọn giá trị đẹp hơn.
- Worker chấp nhận v1 và v2 trong giai đoạn chuyển đổi. Shared validator, schema, renderer, test fixture và collector phải cùng hợp đồng trước khi bật client v2.
- Giữ giới hạn request 2 MiB, 16 GPU, notes 500 ký tự và Issue tối đa 60000 ký tự. Time series phải giới hạn số mẫu; dữ liệu vượt giới hạn lưu local, không cắt âm thầm.
- Summary GitHub hiển thị GPU đích, lỗi/cảnh báo, khoảng đo, kết quả idle/load và diff trước/sau; chi tiết còn lại có manifest và trạng thái “chỉ lưu cục bộ”.
- JSON/CSV/summary/manifest trong ZIP local. Trước gửi, preview đúng phần sẽ gửi và kết quả lọc thông tin cá nhân. Không tự upload ROM hoặc XML/event log thô.

## 5. Trình tự triển khai

### P0 — Sửa độ tin cậy trước khi tăng dữ liệu

- [x] Thống nhất DEV_1F0B/2189 theo hồ sơ hiện hành; nhận diện ID lạ là unknown, không tự suy ra 40HX.
- [x] Cho chọn GPU đích rõ ràng trên máy nhiều GPU; top-level summary và nvidia block cùng BDF.
- [x] Timestamp từng nguồn; chuẩn hóa Gen/width/đơn vị và cảnh báo dữ liệu khác nhau *(chuẩn hóa Gen/width + cảnh báo: xong; timestamp từng nguồn chuyển sang P1 vì schema v1 strict)*.
- [x] Sửa ý nghĩa completeness trên UI: “18/18 trường NVIDIA cơ bản”, không gọi là đầy đủ dữ liệu nghiên cứu.
- [x] Test hồi quy: CMP 40HX đứng sau RTX, CMP 30HX, ID lạ, BDF không khớp, hai nguồn khác Gen và driver không trả đủ trường.

Đầu ra: một báo cáo không nhầm GPU, không che mất mâu thuẫn nguồn.

### P1 — Schema v2 và collector cơ bản

- [x] Định nghĩa schema/fixture v2 theo mục 3–4; chốt danh sách trường tự động so với khai báo tay.
- [x] Bổ sung platform/boot/security/driver/PnP và truy vấn NVIDIA theo khả năng có trên máy *(kèm timestamp từng nguồn từ P0 và power/throttle/thermal/memory/utilization mở rộng)*.
- [x] Hỏi ngữ cảnh bằng lựa chọn ngắn: baseline/sau mod, loại mod, khe/riser, experiment; cho lưu bản nháp khi chưa biết *(bổ sung `mod_type`, `previous_report_id` và `last_report_id.txt`)*.
- [x] Tạo renderer v2 và preview thiếu dữ liệu; bảo đảm một GPU lỗi không làm mất báo cáo các GPU còn lại.
- [x] Bổ sung validator + allowlist + lọc PII cho mọi trường mới, đặc biệt log lỗi và đường dẫn *(kèm kiểm tra preview PII trước khi gửi)*.
- [x] Kiểm tra đọc log legacy của dự án qua parser allowlist; không đưa toàn bộ log vào Issue *(chỉ trích các khóa allowlist từ `gen2_status.txt`/`cmp30hx_batch_status.txt`)*.

Đầu ra: báo cáo cơ bản hữu ích cho driver/platform và đủ cấu trúc để ghép phép thử.

### P2 — Đo idle/load và phép so sánh

- [ ] Gói helper user-mode tối thiểu cho truy vấn CUDA, memory copy và kernel correctness; tải helper theo hash được chốt, không tự cài CUDA Toolkit.
- [ ] Trước khi thêm helper mới, đối chiếu phần có thể dùng lại từ mã dự án/CUDA Samples; không phân phối lại công cụ khi chưa kiểm tra điều kiện cấp phép.
- [ ] Chu kỳ khởi điểm: idle 10s, tải tối đa 30s, recovery 10s, lấy mẫu khoảng 1s; thời lượng thực tế và warmup ghi trong báo cáo. Chế độ nhanh vẫn giữ riêng.
- [ ] Ghi H2D và D2H bằng pinned memory, kích thước buffer, số lần lặp, đồng bộ, đơn vị GB/s, median và khoảng dao động. Không so sánh trực tiếp hai phương pháp đo khác nhau.
- [ ] Tải compute và tải truyền PCIe là hai bài thử riêng; không dùng compute-only để kết luận băng thông PCIe.
- [ ] VRAM correctness chỉ cấp phát phần giới hạn từ bộ nhớ đang trống; ghi dung lượng đã kiểm tra và phần chưa kiểm tra.
- [ ] Tensor test chỉ trên profile phù hợp; xác minh kết quả, kiểu dữ liệu, kích thước ma trận, math mode, warmup/lặp và đường Tensor được chọn. Đối chứng đường compute thông thường khi khả thi.
- [ ] Watchdog, hủy chạy, nhiệt/nguồn giới hạn theo thiết bị và lỗi GPU: dừng tải, lưu dữ liệu đã có, không tự tăng công suất/clock.
- [ ] Bộ so sánh trước/sau cảnh báo khác GPU, driver, firmware, BIOS, bài thử hoặc điều kiện tải. Không kết luận tăng do unlock khi nhiều biến đổi cùng lúc.

Đầu ra: bằng chứng dưới tải và diff có thể tái lập. Chế độ này được chọn chủ động vì tạo tải thực lên GPU.

### P3 — PCIe/BAR và chẩn đoán chuyên sâu

- [ ] Kiểm tra API OS có thể cung cấp topology/resources trước; ghi chính xác trường nào vẫn cần helper đặc quyền.
- [ ] Thiết kế helper chỉ đọc với allowlist register/capability, BDF/profile validation, bounds check và timeout; không tái sử dụng đường LinkNegotiator có chức năng ghi.
- [ ] Decode endpoint + upstream PCIe: capability max/current, TLS, ASPM, MPS/MRRS, trạng thái link và lỗi hỗ trợ.
- [ ] Decode ReBAR supported/current sizes và đối chiếu resource size; không dùng riêng Above 4G hay BAR1 total để tuyên bố ReBAR enabled.
- [ ] Với 40HX đúng profile và BOOT_0 hợp lệ, có thể nhập chẩn đoán SS0/SS1 đã kiểm chứng; ghi đó là heuristic của dự án và đối chiếu workload.
- [ ] Không đọc MMIO trên profile lạ. 30HX theo mục tiêu Gen2 của dự án, không chạy đường firmware/Tensor hoặc phục hồi PnP của 40HX.
- [ ] Nếu cần driver/quyền admin, trình bày thay đổi tạm, cách dọn/rollback và xin xác nhận riêng trước triển khai/chạy; không tự tắt cơ chế bảo vệ hệ thống.
- [ ] Test mock xác nhận không có lời gọi ghi, reset/retrain, và xử lý BAR lỗi, family mismatch, driver blocked.

Đầu ra: xác định giới hạn ở card hay đường upstream, và trạng thái ReBAR/thanh ghi có provenance.

### P4 — Hồ sơ phần cứng và quy trình thử mod

- [ ] Mẫu hồ sơ: mã card do chủ sở hữu đặt, PCB revision, marking GPU/VRAM, ảnh hai mặt đã che serial, trạng thái strap và lịch sử ROM/mod.
- [ ] VBIOS: ban đầu chỉ version + hash của file do chủ card cung cấp. Dump tự động là công việc riêng sau khi kiểm chứng đường chỉ đọc và được chủ card chọn.
- [ ] ROM nhị phân/ảnh gốc giữ local mặc định; ghi công cụ/version/hash và cho chủ card chọn tài liệu nào chia sẻ.
- [ ] Mỗi experiment chỉ thay một yếu tố: driver hoặc mod hoặc khe/riser; ghi phiên bản/hash của gói được dùng.
- [ ] Ghép baseline → sau thay đổi → sau reboot → sau sleep/wake → sau rollback, người dùng thực hiện reboot/mod theo hướng dẫn riêng.
- [ ] Không dùng screenshot đơn lẻ hay trạng thái “P0” làm kết luận unlock.
- [ ] Không suy diễn hiệu năng Tensor hoặc tuyên bố TFLOPS từ một giá trị thanh ghi.

Đầu ra: hồ sơ đối chiếu theo từng biến thể phần cứng, có khả năng truy nguồn và hoàn tác.

### P5 — Nghiệm thu, vận hành và phát hành

- [ ] Unit/fixture: parse nhiều driver, unsupported, timeout, PII, nhiều GPU, giới hạn body/summary, schema v1/v2.
- [ ] Collector Windows PowerShell 5.1/BAT thật: encoding/CRLF, đường dẫn khó, không có NVIDIA/CUDA, offline, hủy tải, pending/resend.
- [ ] Kiểm tra v1 tiếp tục gửi được trong giai đoạn chuyển đổi; v2 bị Worker từ chối có kiểm soát nếu server chưa hỗ trợ.
- [ ] E2E sandbox: Issue v2 đúng GPU/nhãn/summary, duplicate giữ URL, invalid/PII không tạo Issue; tiếp tục các kiểm tra vận hành E1/E2 cũ còn thiếu.
- [ ] Phần cứng: RTX đối chứng, CMP 30HX, CMP 40HX, máy nhiều GPU, trực tiếp/riser; ghi rõ trường nào API hỗ trợ trên mỗi driver.
- [ ] CMP 30HX nghiệm thu Gen2 ×16/TLS Gen2 khi đường vật lý hỗ trợ, đo dưới tải sau reboot/wake; CMP 40HX theo mục tiêu profile hiện hành và kiểm tra workload độc lập.
- [ ] Không đặt ngưỡng GB/s/TFLOPS dùng chung cho mọi máy. Chốt phương pháp, baseline, điều kiện nhiệt/nguồn và ngưỡng chấp nhận trước khi thử card đích.
- [ ] Giữ unverified cho đến khi chủ card xác nhận dữ liệu; chia trạng thái “collector chạy đúng” và “mod đạt mục tiêu”.
- [ ] Deploy sandbox trước, giữ khả năng dùng client v1/rollback Worker; chuyển repository thật và phát hành sau quyết định của chủ dự án.

## 6. Phạm vi file và các bước có thể giao thực hiện

| Bước | File/module dự kiến | Kiểm chứng chính |
| --- | --- | --- |
| P0 | Collect_VGA_Report.bat; scripts/Test-VGA-Collector.ps1; scripts/vga-report/report.cjs + test | Fixture chọn CMP, timestamp/source, render đơn vị |
| P1 | docs/vga-report/schema.json; collector; shared validator/renderer; Worker + test | Chấp nhận v1/v2, chặn PII/unknown fields, thiếu API vẫn gửi bản cơ bản |
| P2 | Helper user-mode mới dưới windows-v3.0/tools/ (chốt tên sau khảo sát build); collector; tests | Đúng BDF, correctness, thời gian/timeout/hủy tải, dữ liệu tải có giới hạn |
| P3 | Tái sử dụng types/decoder từ 40hxcore nếu thực sự chỉ đọc; helper chẩn đoán tách đường ghi | Mock không ghi/reset, topology/BAR fixtures; xác nhận phần cứng riêng |
| P4 | docs/vga-report/; mẫu context/experiment và renderer diff | Không ghép nhầm card, không chia sẻ ROM/PII ngoài lựa chọn |
| P5 | CI; runbook/policy; sandbox fixture/E2E | Kết quả thật theo từng card và phương án rollback |

Triển khai từng P theo thứ tự, mỗi bước có bản báo cáo mẫu xem được trước khi triển khai bước kế. Chưa cần thêm cơ sở dữ liệu, dashboard, dịch vụ lưu ROM hoặc hệ thống tự thử mod.

## 6b. Tiêu chí hoàn thành, rủi ro và quyết định còn mở

| Bước | Hoàn thành khi |
| --- | --- |
| P0 | Test hồi quy mục P0 đạt; RTX đứng trước CMP vẫn chọn đúng CMP; mâu thuẫn Gen hiển thị cảnh báo |
| P1 | Fixture v1 và v2 đều qua validator; thiếu API không chặn gửi bản cơ bản; PII bị loại trong mọi trường mới |
| P2 | Chạy lặp 3 lần cùng điều kiện có dao động ghi rõ; hủy tải/timeout lưu dữ liệu đã có |
| P3 | Mock xác nhận 0 lời gọi ghi/reset/retrain; BAR/ReBAR có provenance |
| P4 | Ghép được 5 giai đoạn của một experiment mà không nhầm card |
| P5 | Sandbox E2E đạt; phần cứng thật ghi `unverified` đến khi chủ card xác nhận |

| Rủi ro | Giảm thiểu |
| --- | --- |
| Tải GPU gây nhiệt/treo máy | Chế độ chọn chủ động, giới hạn thời lượng, watchdog, cảnh báo tiếng Việt |
| Lộ dữ liệu cá nhân qua log/đường dẫn | Allowlist + preview + lọc PII; ROM/event log thô giữ local |
| Kết luận sai do nhiều biến đổi | Mỗi experiment đổi một yếu tố; so sánh chỉ khi cùng quy trình |
| Helper đặc quyền ghi nhầm phần cứng | Helper chỉ đọc riêng, allowlist, mock test; không dùng lại đường ghi của LinkNegotiator ([link.go](../../windows-v3.0/tools/40hxcore/link.go)) |
| Worker/collector lệch hợp đồng | Chung schema/validator; Worker nhận v1 và v2 trước khi bật client v2 |

Quyết định còn mở (cần chủ dự án): tên và vị trí helper P2; có cho phép tải helper bên ngoài hay chỉ bundle; ngưỡng chấp nhận từng card; thời điểm chuyển từ sandbox sang repository thật.

## 7. Mẫu thông tin chủ card cần bổ sung

- Card cần thử: ________
- PCI ID/SUBSYS (collector sẽ tự điền nếu đọc được): ________
- Khe trên mainboard: ________
- Riser / cắm trực tiếp: ________
- PSU và cách cấp nguồn cho card: ________
- Đã thay strap/flash VBIOS/chạy EFI/mod driver chưa: ________
- Gói mod/driver và phiên bản hoặc SHA-256: ________
- Mục tiêu thử: công suất/TDP / PCIe Gen Speed / ReBAR / CUDA/Tensor / VRAM / ổn định / khác: ________
- Có bản baseline trước mod không: ________
- Có thể thử sau reboot/sleep-wake và khôi phục cấu hình cũ không: ________

Không cần gửi serial, khóa riêng, tài khoản hoặc thông tin cá nhân.

## 8. Căn cứ và giới hạn diễn giải

- Hồ sơ hỗ trợ thực tế: [40hxcore/const.go](../../windows-v3.0/tools/40hxcore/const.go).
- Heuristic thanh ghi hiện tại: [40hxcore/compute.go](../../windows-v3.0/tools/40hxcore/compute.go). Trạng thái từ code này cần được xác minh bằng workload, không tự chứng minh con số hiệu năng.
- Hợp đồng phần cứng: [PCIe/Tensor spec](../specs/pcie-gen2-and-tensor-unlock-spec.md) và [nghiệm thu CMP](../specs/cmp-release-acceptance-2026-11-20.md).
- [NVIDIA SMI](https://docs.nvidia.com/deploy/nvidia-smi/): link hiện tại có thể giảm khi nhàn rỗi; max còn phụ thuộc cấu hình GPU và hệ thống.
- [NVML device queries](https://docs.nvidia.com/deploy/nvml-api/latest/api/group__nvmlDeviceQueries.html): có truy vấn link/BAR1 và mã unsupported; BAR1 total/used/free không thay thế kiểm tra PCI ReBAR capability.
- [CUDA Best Practices](https://docs.nvidia.com/cuda/cuda-c-best-practices-guide/): cơ sở cho thuộc tính thiết bị và đo truyền dữ liệu có phương pháp.

## 9. Phạm vi nhiều dòng GPU và bằng chứng cho từng mục tiêu unlock

Mỗi hồ sơ ghi PCI vendor/device/subsystem/revision, dạng card, PCB revision, bộ nhớ và phiên bản VBIOS khi có. Mỗi khả năng ghi nguồn tham chiếu theo đúng SKU. Card chưa nhận diện vẫn có báo cáo cơ bản và được đánh dấu chờ xác minh. Phép đọc chuyên sâu chỉ áp dụng cho profile đã xác minh.

| Dòng card | Phép đo và điều kiện cần phân biệt |
| --- | --- |
| P106-XXX, ưu tiên P106-100 và P106-090 | Tách từng SKU, hãng và revision. Thu PCI ID, VBIOS, VRAM, cấu hình cấp nguồn, connector có thực trên PCB, driver nhận card, chế độ headless/render qua iGPU, Gen/width và băng thông dưới tải. GIGABYTE công bố PCIe 1.1 cho đúng GV-NP106D5-6G rev. 1.0. Các biến thể khác cần nguồn và baseline riêng. Với chip Pascal đã xác minh, đánh dấu Tensor không áp dụng. [GIGABYTE P106-100 rev. 1.0](https://www.gigabyte.com/sg/Graphics-Card/GV-NP106D5-6G-rev-10/sp) · [NVIDIA về Tensor Core](https://developer.nvidia.com/blog/?p=8496) |
| Tesla P100 PCIe 12 GB và 16 GB | Hồ sơ riêng theo SKU. Đối chiếu PCIe Gen3 ×16 và công suất bo mạch 250 W theo thông số hãng. Đo HBM2/ECC, CUDA FP16/FP32/FP64 và ghi luồng gió làm mát. Tensor không áp dụng cho GP100 Pascal. [Thông số P100 PCIe](https://www.nvidia.com/content/dam/en-zz/Solutions/Data-Center/tesla-product-literature/NV-tesla-p100-pcie-PB-08248-001-v01.pdf) · [Pascal Tuning Guide](https://docs.nvidia.com/cuda/pascal-tuning-guide/index.html) |
| Tesla P100 SXM2 | Ghi carrier/baseboard, đường kết nối host, NVLink nếu đo được, HBM2/ECC, nguồn và luồng gió. Dùng công suất tham chiếu và baseline riêng cho SXM2. [Thông số P100 theo dạng card](https://images.nvidia.com/content/grid/pdf/DA-09018-001_v05.pdf) |
| CMP 30HX và 40HX | 30HX giữ mục tiêu Gen2 và đánh dấu Tensor không áp dụng theo hồ sơ dự án. 40HX kiểm tra PCIe theo profile và xác nhận Tensor bằng workload. |
| CMP 50HX, 90HX, 170HX và card đào khác | Lập hồ sơ riêng theo SKU, PCB và driver. Xác minh công suất, PCIe, VRAM và compute trước. Thêm Tensor, ECC, NVLink, encode/decode hoặc xuất hình theo bằng chứng về khả năng của từng mẫu. |
| RTX hoặc Tesla đối chứng | Dùng cùng platform và phương pháp đo khi khả thi để phát hiện sai lệch từ thiết bị đo, driver và điều kiện tải. |

Tên rút gọn “P100” chỉ là tên khai báo ban đầu. Xác nhận PCI IDs, kiến trúc và dạng card trước khi áp thông số Tesla P100. Ghi riêng SKU vật lý và tên driver báo để nhận diện trường hợp driver hoặc VBIOS đã đổi tên card. Các dòng P104/P102 có thể thêm qua hồ sơ riêng khi có mẫu thật.

### Công suất, nhiệt và xung

- [ ] Tách công suất định mức bo mạch theo tài liệu đúng SKU, giới hạn mặc định, dải min/max driver báo, giới hạn được yêu cầu, giới hạn thực thi và công suất đo. Ghi phạm vi GPU/board/module và phương pháp lấy mẫu. Công suất toàn máy tại ổ điện là phép đo riêng.
- [ ] Lấy mẫu cùng thời điểm cho công suất, tải SM/bộ nhớ, P-state, xung core/memory, điện áp nếu hỗ trợ, nhiệt GPU/hotspot/VRAM, quạt hoặc luồng gió ngoài, lý do giảm xung và bộ đếm. Giữ tên trường gốc theo driver và trạng thái khi API thiếu dữ liệu. [NVIDIA SMI](https://docs.nvidia.com/deploy/nvidia-smi/index.html)
- [ ] So idle → tải compute → tải bộ nhớ hoặc PCIe → hồi phục. Đối chiếu thông lượng với cờ giới hạn công suất, nhiệt và power brake. Ghi PSU, đầu cấp nguồn, carrier/riser, PCB/VRM và cách làm mát do chủ card xác nhận. Chỉ chạy tải trên card tản thụ động sau khi xác nhận luồng gió phù hợp.
- [ ] Đánh giá kết quả theo thông lượng, ổn định, công suất và hiệu suất trên watt. Khi có nhiều nguyên nhân khả dĩ, lưu dữ liệu còn thiếu để phân biệt chúng.

### PCIe Gen Speed, Tensor và bộ nhớ

- [ ] Ghi GT/s, độ rộng link và Gen hỗ trợ, Gen mục tiêu TLS, Gen đang đàm phán ở endpoint và các cổng upstream nếu đọc được. Đo H2D/D2H lúc tải, lưu PCIe replay/AER/WHEA theo cửa sổ thử. Đối chiếu cắm trực tiếp và qua riser nếu khả thi.
- [ ] Ghi khả năng CUDA/OpenCL được driver báo, số SM, FP32/FP64/FP16, VRAM có bằng chứng vật lý so với dung lượng driver báo, ECC và lỗi bộ nhớ nếu hỗ trợ. Chạy kiểm tra tính đúng và thông lượng, ghi số byte thực sự đã kiểm tra.
- [ ] Với SKU có Tensor, ghi kiểu dữ liệu, math mode, kernel hoặc thông tin profiler nếu hỗ trợ, sai số, thông lượng và lý do fallback. So với bài đối chứng cùng dữ liệu và độ chính xác phù hợp. Phân biệt `không áp dụng` do kiến trúc, `không hỗ trợ` theo API, `chưa xác minh` và `đã xác nhận sử dụng`. [Turing Tuning Guide](https://docs.nvidia.com/cuda/turing-tuning-guide/)
- [ ] Với chức năng quan trọng của SKU, bổ sung phép đo ReBAR, NVLink/P2P, encode/decode, xuất hình hoặc hoạt động headless. Ghi riêng khả năng do API báo và kết quả workload.

### Ma trận phân biệt nguyên nhân và tiêu chí kết luận

| Mục tiêu | Bằng chứng tối thiểu | Kết luận được phép ghi |
| --- | --- | --- |
| Giới hạn công suất/TDP | Nguồn định mức theo đúng PCB, công suất và mức trần thực thi dưới cùng tải, cờ power/thermal/power brake, xung, nhiệt, nguồn và heatsink. Nếu có tài liệu phần cứng, ghi model VRM/controller và khả năng cấp nguồn theo bo mạch. | Xác nhận thay đổi mức trần khi số đo và cờ giới hạn thay đổi lặp lại qua tải đối chứng. Ghi giới hạn an toàn của bo mạch là chưa xác minh nếu chỉ có telemetry. |
| PCIe Gen/width | Đọc khả năng và trạng thái hai đầu link, TLS nếu có, đối chứng card/khe/riser, băng thông H2D/D2H cùng bài tải và lỗi link trước/sau. | Phân loại đang hạ tốc khi nhàn, nghẽn bởi riser/khe/upstream hoặc giới hạn phía card quan sát được. Nếu chưa tách được driver, firmware và phần cứng thì giữ giả thuyết mở. |
| Tensor/compute | Xác minh kiến trúc, driver/runtime và binary có hỗ trợ kiến trúc đích, workload kiểm tra tính đúng, tín hiệu profiler hoặc thông tin kernel cho đường Tensor, đối chứng fallback. | Ghi riêng khả năng kiến trúc, mức driver cung cấp và việc workload đã sử dụng Tensor. Lỗi thiếu binary CUDA phù hợp được ghi là lỗi tương thích công cụ. |
| VRAM/ECC và xuất hình | Đối chiếu chip/PCB với dung lượng driver, kiểm tra tính đúng trong vùng đã thử, lỗi ECC nếu có. Với card headless, xác định rõ GPU render, GPU xuất hình và connector vật lý. | Báo phần VRAM đã kiểm tra và chế độ render thực sự chạy. Kết quả render qua iGPU được ghi riêng với xuất hình qua cổng trên card. |

- [ ] Lập bảng nguồn theo driver và OS cho từng trường. Trên Linux lấy bản tóm tắt PCI từ `lspci`/sysfs nếu có. Trên Windows lấy PnP/CIM và NVIDIA SMI/NVML rồi ghi phần topology còn thiếu. Lưu mã lỗi phân biệt API thiếu, bị từ chối quyền và tool lỗi.
- [ ] Ghi phiên bản driver, runtime, thư viện và kiến trúc mà helper đã biên dịch. Chọn bản helper tương thích Pascal khi cần. [Hỗ trợ kiến trúc CUDA](https://developer.nvidia.com/blog/navigating-gpu-architecture-support-a-guide-for-nvidia-cuda-developers)
- [ ] Mỗi kết luận kèm nguồn, điều kiện đo, kết quả đối chứng và mức chắc chắn. Nhãn nguyên nhân gồm phần cứng, firmware, driver, đường kết nối, nhiệt/nguồn, hoặc chưa xác định. Chỉ xác nhận nguyên nhân khi có phép thử phân lập yếu tố đó. Lưu kết quả lỗi và thử nghiệm âm tính để tránh lặp giả thuyết đã loại.

### Hồ sơ thử nghiệm theo từng giới hạn

- Card/SKU/PCB và driver/VBIOS: ________
- Mục tiêu, triệu chứng, giá trị tham chiếu đúng SKU và nguồn: ________
- Baseline, bài tải, nhiệt/nguồn và topology: ________
- Giả thuyết về nguồn giới hạn, dữ liệu ủng hộ và dữ liệu còn thiếu: ________
- Một thay đổi được thử và phiên bản/hash: ________
- Kết quả sau thay đổi, sau reboot/sleep nếu áp dụng và sau khôi phục: ________
- Trạng thái đạt / chưa đạt / chưa xác minh / không áp dụng: ________

### Đưa vào các bước hiện có

- [ ] P0/P1 thêm hồ sơ P106-100/P106-090, P100 PCIe/SXM2, CMP và GPU chưa nhận diện. Bổ sung công suất, nhiệt, khả năng tính toán và ma trận nguồn theo driver vào schema v2, giữ tương thích báo cáo v1.
- [ ] P2 bổ sung bộ tải đối chứng và kiểm tra tính đúng. Giữ bài tải, driver và môi trường nhất quán giữa các lần đo, ngoại trừ biến chủ động thay đổi.
- [ ] P3/P4 áp dụng đọc chuyên sâu theo profile và giữ dữ liệu ROM/PCB gốc cục bộ theo chính sách. Lưu tên công cụ, phiên bản, hash và phương án khôi phục khi thử mod.
- [ ] P5 ưu tiên một P106-100, một P100 PCIe, CMP 30HX, CMP 40HX và RTX đối chứng. Mở rộng P106-090, SXM2, CMP khác và máy nhiều GPU khi có mẫu. Chạy fixture theo từng SKU và nghiệm thu card thật theo từng mục tiêu bằng phép đo lặp và hồ sơ trước/sau.
