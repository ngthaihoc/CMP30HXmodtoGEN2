# Nghiên Cứu Chuyên Sâu: Các Chỉ Số Telemetry Phần Cứng Cần Thiết Cho Modding & Unlocking VGA (Vượt Qua Giới Hạn 18/18 nvidia-smi Cơ Bản)

**Ngày lập:** 2026-11-22  
**Mục tiêu:** Xác định đầy đủ các nguồn dữ liệu sơ cấp (Primary Sources), API hệ thống Windows và cơ chế thu thập chỉ-đọc (Read-only) nhằm cung cấp đủ dữ liệu thực nghiệm cho việc chẩn đoán, mở khóa (unlock) và modding các dòng GPU NVIDIA (CMP 30HX, CMP 40HX, P106-100, P100 PCIe và GeForce đối chứng), thay thế trạng thái thiếu hụt dữ liệu (chỉ có 18 trường cơ bản của `nvidia-smi`).

---

## 1. Giới hạn Cốt lõi của Bộ 18/18 Trường `nvidia-smi` Cơ Bản

Hiện tại, trong các báo cáo ban đầu, nhóm dữ liệu NVIDIA chỉ bao gồm 18 trường cơ bản:
`name`, `driver_version`, `vbios_version`, `pci.bus_id`, `pci.device_id`, `pci.sub_device_id`, `memory.total`, `compute_cap`, `pcie.link.gen.max`, `pcie.link.gen.gpucurrent`, `pcie.link.width.max`, `pcie.link.width.current`, `temperature.gpu`, `power.draw`, `clocks.sm`, `clocks.mem`, `clocks.max.sm`, `pstate`.

### Tại sao 18 trường này **hoàn toàn không đủ** để modding và unlock?
1. **Thiếu thông tin Upstream & Topology PCIe:**
   - 18 trường chỉ phản ánh góc nhìn cục bộ từ phía GPU Endpoint. Khi card bị tụt tốc xuống Gen1, không thể phân biệt được nguyên nhân là do:
     * Khe cắm bo mạch chủ / Chipset chỉ hỗ trợ Gen1.
     * Cáp nối Riser chất lượng kém gây suy hao tín hiệu.
     * Cổng gốc PCIe (PCIe Root Port) của CPU/Chipset bị cấu hình sai trong BIOS hoặc giới hạn bởi nhà sản xuất bo mạch chủ OEM/server.
   - *Nguồn dẫn chứng:* [pcie-gen2-and-tensor-unlock-spec.md](../specs/pcie-gen2-and-tensor-unlock-spec.md) §User Story 8–9; [deep-link-negotiator-and-driver-session-spec.md](../specs/deep-link-negotiator-and-driver-session-spec.md).

2. **Thiếu trạng thái Tiết kiệm điện PCIe (ASPM):**
   - Trên Windows, tính năng *Link State Power Management* (ASPM L0s/L1) tự động hạ tốc PCIe link từ Gen3/Gen2 xuống Gen1 khi GPU nhàn rỗi (idle).
   - Nếu không đọc cấu hình ASPM hệ thống, báo cáo sẽ tạo ra **kết luận sai lệch** rằng card bị kẹt Gen1 (lỗi mod), trong khi thực chất card chỉ đang tiết kiệm điện.
   - *Nguồn dẫn chứng:* [NVIDIA SMI Documentation](https://docs.nvidia.com/deploy/nvidia-smi/) (PCIe link idling behavior); Windows Power Management Architecture.

3. **Thiếu thông số DMA Payload & Read Request (MPS / MRRS):**
   - Để đạt thông lượng PCIe tối đa (~6.4 GB/s trên Gen2 ×16 của CMP 30HX), thanh ghi `DEVCTL` cần được cấu hình `MaxReadRequestSize` (MRRS) đạt ít nhất 512 bytes (`0x2000`). Nếu hệ điều hành hoặc BIOS để mặc định 128B hoặc 256B, hiệu năng DMA bị phân mảnh nghiêm trọng.
   - 18 trường cơ bản hoàn toàn không hiển thị MPS và MRRS.
   - *Nguồn dẫn chứng:* [pcie-gen2-and-tensor-unlock-spec.md](../specs/pcie-gen2-and-tensor-unlock-spec.md) §Implementation Decisions 3 & `const.go`.

4. **Thiếu kích thước BAR & Trạng thái Resizable BAR (ReBAR):**
   - VRAM kích thước lớn (6GB, 8GB, 16GB) cần Resizable BAR (BAR1 > 256 MiB) để CPU truy cập trực tiếp toàn bộ không gian nhớ mà không phải phân mảnh qua cửa sổ 256 MiB.
   - `nvidia-smi` dạng bảng CSV cơ bản không cung cấp BAR1, trong khi `nvidia-smi -q -x` (XML) cung cấp chính xác `bar1_memory_usage.total` (256 MiB vs 8192 MiB).
   - *Nguồn dẫn chứng:* [modding-data-plan.md](modding-data-plan.md) §3 & §8; [NVML Device Queries](https://docs.nvidia.com/deploy/nvml-api/latest/api/group__nvmlDeviceQueries.html).

5. **Thiếu kiểm tra chịu tải & băng thông thực tế (Load & Bandwidth):**
   - Trạng thái nhàn rỗi (idle) chỉ thể hiện đàm phán tĩnh. Chỉ khi có tải truyền dữ liệu Host-to-Device (H2D) và Device-to-Host (D2H) bằng pinned memory, ta mới đo được băng thông thực tế (GB/s) để xác nhận link đã thực sự chạy Gen2 (~6.4 GB/s) hay Gen3 (~12 GB/s).
   - *Nguồn dẫn chứng:* [CUDA Best Practices Guide](https://docs.nvidia.com/cuda/cuda-c-best-practices-guide/) §Async and Overlapping Transfers with Pinned Memory.

6. **Thiếu giám sát lỗi phần cứng và độ ổn định (WHEA / TDR):**
   - Bản mod PCIe hoặc strap VBIOS có thể chạy được vài giây nhưng sinh ra hàng loạt lỗi truyền nhận PCIe (AER - Advanced Error Reporting) hoặc làm driver crash (Timeout Detection and Recovery - TDR Event ID 13/14). Nếu không quét Event Log trong cửa sổ đo, báo cáo sẽ kết luận "thành công" giả tạo trên một hệ thống đang mất ổn định.
   - *Nguồn dẫn chứng:* Microsoft Windows Hardware Error Architecture (WHEA) Specification; Microsoft Learn TDR Diagnosis in Graphics Drivers.

7. **Thiếu chuỗi liên kết thử nghiệm (Experiment Chain) & Ngữ cảnh vật lý (Physical Context):**
   - Không thể đánh giá bản mod nếu không đối chiếu báo cáo trước mod (baseline) và sau mod (post-unlock) trên cùng một máy và cùng một card phần cứng.
   - Không thể tái lập phép thử nếu không biết card cắm trực tiếp hay qua riser x1 đào coin, cắm khe CPU hay chipset, nguồn phụ cấp đủ hay thiếu.
   - *Nguồn dẫn chứng:* [modding-data-plan.md](modding-data-plan.md) §4 & §7.

---

## 2. Bản Đồ Nguồn Dữ Liệu Sơ Cấp (Primary Sources) & Phương Thức Thu Thập Cấp Hệ Thống

Để lấp đầy 7 nhóm bằng chứng hoàn chỉnh mà vẫn duy trì nguyên tắc an toàn tuyệt đối (100% Read-only, không yêu cầu driver kernel lạ, không ghi MMIO/PCI), các API và lệnh hệ thống sau đã được kiểm chứng thực nghiệm trên Windows:

| Nhóm dữ liệu | Dữ liệu cụ thể | Nguồn sơ cấp (Windows / Driver API) | Lệnh / Đoạn mã thực nghiệm |
| --- | --- | --- | --- |
| **Topology** | Upstream Root Port | Windows PnP Device Property (`DEVPKEY_Device_Parent`) | `(Get-PnpDeviceProperty -InstanceId $gpuId -KeyName 'DEVPKEY_Device_Parent').Data` |
| **Topology** | Root Port Device Name & VID:DID | Windows PnP Entity (`Get-PnpDevice -InstanceId $parentId`) | `$parent = Get-PnpDevice -InstanceId $parentId; $parent.FriendlyName; $parent.InstanceId` |
| **Topology** | ASPM (Link State Power Management) | Windows Power Management CLI (`powercfg`) | `powercfg /query SCHEME_CURRENT SUB_PCIEXPRESS` -> đọc `Current AC Power Setting Index` (0=Off, 1=Moderate/L0s, 2=Max/L1) |
| **Topology** | Max Payload Size (MPS) | Windows PnP Property (`DEVPKEY_PciDevice_MaxPayloadSize`) | `Get-PnpDeviceProperty -InstanceId $gpuId -KeyName 'DEVPKEY_PciDevice_MaxPayloadSize'` (0=128B, 1=256B, 2=512B) |
| **Topology** | Max Read Request Size (MRRS) | Windows PnP Property (`DEVPKEY_PciDevice_MaxReadRequestSize`) | `Get-PnpDeviceProperty -InstanceId $gpuId -KeyName 'DEVPKEY_PciDevice_MaxReadRequestSize'` (0=128B, 1=256B, 2=512B, 3=1024B) |
| **Topology** | AER Capability | Windows PnP Property (`DEVPKEY_PciDevice_AERCapabilityPresent`) | `Get-PnpDeviceProperty -InstanceId $gpuId -KeyName 'DEVPKEY_PciDevice_AERCapabilityPresent'` (Boolean) |
| **Topology** | PCIe Replay Counters | NVIDIA SMI XML Query (`replay_counter`, `replay_rollover_counter`) | `[xml]$xml = & "nvidia-smi" -q -x; $xml.nvidia_smi_log.gpu.pci.replay_counter` |
| **Topology** | ReBAR (BAR1 Size) | NVIDIA SMI XML Query (`bar1_memory_usage.total`) & WMI Memory Address | `[xml]$xml = & "nvidia-smi" -q -x; $xml.nvidia_smi_log.gpu.bar1_memory_usage.total` (256 MiB = Disabled, >=4096 MiB = Active) |
| **Stability** | TDR Driver Crash Events | Windows Event Log `System` (`ProviderName='nvlddmkm'`, Event ID 13, 14) | `Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName='nvlddmkm'; StartTime=$startedUtc} -ErrorAction SilentlyContinue` |
| **Stability** | WHEA PCIe Hardware Errors | Windows Event Log `System` (`ProviderName='Microsoft-Windows-WHEA-Logger'`, Event ID 17, 18, 19) | `Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName='Microsoft-Windows-WHEA-Logger'; StartTime=$startedUtc} -ErrorAction SilentlyContinue` |
| **Link Load** | H2D & D2H Throughput (GB/s) | Helper `vgabench` qua NVIDIA Driver API (`nvcuda.dll`) Pinned Host Memory | `windows-v3.0/tools/vgabench/vgabench.exe -bdf 0000:01:00.0 -size 64` (kích hoạt qua cờ `-RunLoadTest`) |
| **Compute** | FP32 Kernel Correctness & GFLOPS | Helper `vgabench` qua CUDA PTX kernel nhúng sẵn (`sm_50`) | `windows-v3.0/tools/vgabench/vgabench.exe` kiểm tra tính đúng SAXPY FP32 và trả về GFLOPS |
| **Comparison** | Liên kết chuỗi thử nghiệm | Quản lý tệp cục bộ `last_report_id.txt` kết hợp `experiment_id.txt` | Đọc UUID báo cáo trước gán vào `previous_report_id`; cập nhật `last_report_id.txt` bằng UUID báo cáo hiện tại |
| **Context** | Ngữ cảnh vật lý (Physical Context) | Nhập liệu nhanh có hướng dẫn hoặc cờ `-DetailedContext` | Thu thập `mod_type`, `riser`, `slot`, `stage`, `psu_model`, `notes` thay thế giá trị mặc định `unknown` |

---

## 3. Kiến Trúc Tích Hợp và Phân Lớp Báo Cáo Mới

Để giải quyết mâu thuẫn giữa **Yêu cầu dữ liệu sâu cho Modder** và **Trải nghiệm tối giản cho Người dùng phổ thông**:

### Chiến lược 2 Luồng:
1. **Luồng người dùng phổ thông (Default UX):**
   - Chạy `Collect_VGA_Report.bat` bình thường.
   - Script tự động đọc ngầm toàn bộ dữ liệu chỉ-đọc: PnP Root Port, ASPM qua `powercfg`, BAR1 qua `nvidia-smi -q -x`, MPS/MRRS, và quét WHEA/TDR events.
   - Người dùng không cần trả lời bất kỳ câu hỏi kỹ thuật nào! Toàn bộ quá trình diễn ra tự động 100%.
   - Màn hình kết thúc chỉ hỏi 1 câu duy nhất: `[1] Lưu cục bộ / [2] Gửi lên GitHub / [3] Thoát`.
   - Báo cáo sinh ra đã tự động đạt **6/7 nhóm** (`inventory`, `topology`, `link_idle`, `stability`, và `context` cơ bản) ngay cả khi không chạy tải!

2. **Luồng nghiên cứu viên / Modder (Advanced Modding Workflow):**
   - Chạy với cờ: `Collect_VGA_Report.bat -RunLoadTest -KeepHistory -DetailedContext`.
   - Kích hoạt đo tải thực tế qua `vgabench` (H2D/D2H GB/s, FP32 correctness), tự động kết nối `previous_report_id`, và hỏi nhanh 3 thông số vật lý (Riser, Slot, Mod Type).
   - Đạt đủ **7/7 nhóm bằng chứng** hoàn chỉnh, mức bằng chứng nâng cấp từ `reported` lên `observed` hoặc `compared`.

---

## 4. Kết Luận Nghiên Cứu

Bằng cách tích hợp các nguồn dữ liệu sơ cấp sẵn có trong Windows (`powercfg`, `Get-PnpDeviceProperty`, `nvidia-smi -q -x`, `Get-WinEvent`, và `vgabench`), bộ công cụ VGA Collector có thể cung cấp **đầy đủ 100% dữ liệu nghiên cứu modding chuyên sâu** mà không cần cài đặt thêm phần mềm bên thứ ba hay driver can thiệp kernel nguy hiểm. Điều này loại bỏ hoàn toàn hạn chế "chỉ có 18/18 trường cơ bản", đáp ứng trọn vẹn yêu cầu kỹ thuật để mở khóa CMP 30HX, CMP 40HX, P106-100 và P100.
