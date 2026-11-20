# Nghiệm thu CMP 30HX và CMP 40HX

**Ngày 2026-11-20 · commit `ea173fb` · kết luận chưa đủ điều kiện xác nhận ổn định.**

Phiên này kiểm thử mã trong workspace và dựng gói từ cây đã commit theo cách CI lấy đầu vào. Máy kiểm thử là Windows build `26100.1457`, có RTX 2070 `10DE:1F07`, driver `621.01`. Các mock Linux chạy trong Git Bash. Chưa có máy thử gắn CMP trong phiên.

## Kết quả theo mốc

| Mốc | Kết quả | Bằng chứng |
| --- | --- | --- |
| Build và kiểm tra tĩnh | Đạt | `go vet ./...` qua 8 mô-đun. Build 5 công cụ Windows và build production `UnlockAIO.exe` bằng Wails 2.16.0, với cờ theo CI. `node --check` qua các tệp JS và `bash -n` qua 2 script. |
| Hồi quy phần mềm | Đạt | `go test -cover -count=1 ./...` qua 8 mô-đun. Mock AIO, persistence, status và ReBAR trên Windows qua. Mock cài/gỡ, lỗi persistence, lựa chọn cờ và bỏ qua MMIO trên Linux qua. Kiểm thử telemetry qua. |
| Coverage 80% | Chưa đạt | 40hxcore 40,1% · aio 33,9% · check40x 4,4% · fix30hxcsm 3,0% · inst40hx 2,6% · riotcore 73,0% · uninstall40x 0% · unlockriot 44,9%. |
| Gói cài từ commit | Đạt kiểm tra cấu trúc và luồng mock chuẩn | ZIP có 69 mục và đủ 25 tệp kiểm tra. Xác minh CRC, SHA-256, 6 tệp EXE mới build, định dạng PE ứng dụng EFI và quyền thực thi của Linux injector. Mock Windows và cú pháp cài/gỡ theo CI cũng qua trên bản giải nén. |
| Báo lỗi và an toàn | Chưa đạt | Hai kiểm thử bơm lỗi xác nhận build EFI và ghi PCIe Linux có thể trả mã thành công sai. Rà soát Web UI dự phòng thấy thiếu kiểm tra yêu cầu thay đổi trạng thái. |
| Phần cứng CMP | Chưa chạy | Còn thiếu thiết bị đích và log cài/gỡ, link dưới tải, reboot, sleep/wake, EFI, ReBAR và khôi phục. |

Gói thử chỉ nằm trong `.local/acceptance-2026-11-20/`. Đầu vào là `git archive HEAD` và sáu bản build mới. Dùng Python `zipfile` để tạo ZIP, giữ quyền Unix của các tệp. Job phát hành theo tag trên GitHub và bước chạy EFI trên hệ thống UEFI chưa được chạy trong phiên này.

```text
CMP-Unlock-acceptance-ea173fb.zip
SHA-256 74d9693602edda83b6bd9819b2da0895d178d74010aa69739818d491f5ff5066
```

## Lỗi chặn nghiệm thu

1. **Linux ghi PCIe thất bại nhưng trả thành công.** Fixture thay toàn bộ truy cập phần cứng bằng stub. Mọi ghi trả lỗi, kết quả đọc lại là Gen2 ×8 và TLS vẫn là Gen1. Hàm vẫn in `[OK]` cho ghi TLS/MRRS và trả `0`. Tiêu chí kiểm lại là trả lỗi khi ghi hoặc đọc xác nhận thất bại, đồng thời phân biệt Gen2 ×8 với mục tiêu Gen2 ×16. [Luồng ghi PCIe](../../Setup_CMP30HX_LinuxAIO.sh#L416) · [Điều kiện trả kết quả](../../Setup_CMP30HX_LinuxAIO.sh#L577)
2. **Build EFI báo mã thành công khi linker thất bại.** Fixture mô phỏng linker trả `22`. Script in `DONE`, trả `0` và thiếu tệp EFI. Tiêu chí kiểm lại là build trả mã lỗi khi linker hoặc chuyển đổi thất bại. [Script build EFI](../../windows-v3.0/tools/unlock40x/build_v70.sh#L74)
3. **Web UI dự phòng chạy quyền quản trị thiếu kiểm tra yêu cầu.** Rà soát thấy một số API thay đổi trạng thái chấp nhận yêu cầu mà chưa kiểm tra phương thức và nguồn. Cần có kiểm thử hồi quy cho lớp kiểm tra trước khi dùng luồng này để nghiệm thu. [Web UI của installer](../../windows-v3.0/tools/inst40hx/webserver.go#L284) · [Bộ kiểm thử lớp bảo vệ của AIO](../../windows-v3.0/tools/aio/fallback_web_security_test.go)

Gói từ commit `ea173fb` còn một lỗi cờ Linux. Khi đặt `--uninstall` trước `--status-file <path>` trong chế độ mock, gói đi vào nhánh gỡ mock nhưng bỏ qua tệp trạng thái chỉ định. Cú pháp theo CI đạt. Bài kiểm tra thứ tự cờ trong workspace đã đạt với parser đang sửa cục bộ.

## Bằng chứng cần cho card thật

Ghi commit hoặc SHA-256 của gói đã thử, model và PCI ID, bản OS/driver, bo mạch/BIOS, khe PCIe và log trước/sau. Các thay đổi driver, firmware, cấu hình bảo mật hoặc tự khởi động cần xác nhận riêng, bản sao khôi phục và đối chiếu trước/sau theo quy tắc dự án.

| Thiết bị và luồng | Điều kiện đạt |
| --- | --- |
| CMP 30HX `10DE:2189` trên Windows | Cài và nhận đúng card. Đo Gen2 ×16 dưới tải với TLS Gen2. Tác vụ tự khởi động chạy đúng sau reboot. Sau sleep/wake đo lại dưới tải. Gỡ hai lần, xác nhận tác vụ, tệp và thiết lập do công cụ quản lý được dọn hoặc khôi phục theo log. |
| CMP 30HX `10DE:2189` trên Linux gốc | Đo Gen2 ×16 và TLS Gen2 dưới tải. Xác minh dịch vụ sau boot và hook sau sleep/wake. Gỡ hai lần, giữ tệp ngoài phạm vi của công cụ và đối chiếu cấu hình trước/sau. |
| CMP 40HX `10DE:1F0B` trên Windows/UEFI | Khởi động được với EFI đúng SKU và xác minh luồng phục hồi boot. Đo Gen2 ×16 dưới tải, kiểm tra trạng thái mở khóa theo tài liệu 40HX, rồi lặp lại sau reboot và wake. Gỡ hai lần và xác minh đường boot gốc. |
| ReBAR trên cấu hình hỗ trợ | Xác minh trạng thái thực bằng chẩn đoán sau reboot. Gỡ và đối chiếu lại trạng thái với lần đo trước cài. |

Gen1 khi GPU nhàn rỗi cần đối chiếu với TLS và phép đo dưới tải. CMP 30HX dùng hướng Gen2, tách khỏi luồng EFI của CMP 40HX. [Hồ sơ GPU](../../windows-v3.0/tools/40hxcore/const.go#L38) · [Ghi chú PCIe Gen3](cmp30hx-gen3-observations.md) · [Hướng dẫn phục hồi EFI](../../windows-v3.0/release/README.md#L82)

Log CMP 30HX cũ của v3.0.0 ghi `GEN1_STUCK`. Log đó không xác nhận kết quả của commit đã nghiệm thu trong phiên này. [Log chẩn đoán cũ](../../UserLogIssue/40HXCheck.log)

## Kiểm tra lại sau bản sửa ECC

Các lỗi ghi nhận ở trên đã được sửa trong workspace. Kết quả phần cứng vẫn mở.

- Linux giới hạn cài đặt vào CMP 30HX, kiểm tra ghi và đọc lại cấu hình, phân biệt Gen2 ×16 với link hẹp hoặc idle, và bỏ qua đăng ký tự khởi động nếu bất kỳ card nào thất bại. Luồng cài đã bỏ reset bus, dừng tiến trình GPU, chỉnh GRUB và thiết lập ASPM/Runtime PM toàn hệ thống. Bộ test PCIe có 12 ca đạt trong Git Bash. Bộ test gỡ và trạng thái có 6 ca đạt.
- Build EFI trả đúng mã lỗi linker và chuyển đổi. Nếu lỗi hoặc đầu ra rỗng, tệp EFI tốt trước đó vẫn được giữ. Bài hồi quy chạy với toolchain giả lập.
- Web UI dự phòng xác minh Host, phương thức và nguồn yêu cầu trước khi gọi handler. Các thao tác tối ưu Riot qua CLI cần xác nhận trước bước nâng quyền. Chương trình tối ưu và hoàn tác giữ nguyên thiết lập Windows Vulnerable Driver Blocklist. Nút hoàn tác DirectX giữ lựa chọn GPU ngoài CMP.
- Tất cả 8 mô-đun chạy được `go test` và `go vet`. Mô-đun `uninstall40x` chưa có bài test. Coverage 0–73,9%, dưới mục tiêu 80%. Các mock PowerShell, telemetry, cú pháp JS/Bash, `git diff --check` và `actionlint` đều đạt.
- Build mới 4 EXE Windows và ELF x86-64 tĩnh cho Linux. Wails 2.17.1 qua build production. Tệp ELF đã qua kiểm tra định dạng sau build chéo. Bài test MMIO chạy trên Linux gốc và workflow tag vẫn chờ môi trường tương ứng. Các mốc trên CMP thật ở trên vẫn chưa chạy.
