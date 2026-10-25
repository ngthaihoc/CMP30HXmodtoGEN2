# Bàn giao phiên: 25/10/2026

## Đọc khi tiếp tục

1. Đọc `AGENTS.md`, tài liệu này và kết quả mới của `git status --short`, `git log -1 --oneline`.
2. Đối chiếu HEAD với mốc bên dưới. Nếu mã hoặc baseline đã đổi, xác minh lại phần phụ thuộc vào thay đổi đó.
3. Chọn đúng phần trong [spec dùng chung](../specs/shared-unlock-graphics-and-status.md), rồi đọc source và test tương ứng. `CONTEXT.md` là từ vựng miền; `PROJECT_CONTEXT.md` là ghi chú lịch sử.

## Trạng thái và phạm vi đã hoàn tất

- Nhánh: `fix/aio-review-findings`.
- Mốc mã đã kiểm chứng: `31259ccfe7f985e0617c4fe8acba7f475a4a0715`, `refactor: share unlock, graphics and status decisions`. HEAD thực tế có thể mới hơn do commit tài liệu; đối chiếu Git khi tiếp tục.
- Người dùng đã yêu cầu triển khai cả ba đề xuất của báo cáo kiến trúc bằng skill `implement`, bao gồm review và commit trên nhánh hiện tại. Việc đó đã hoàn tất; không có bước triển khai còn nợ trong phạm vi đã nhận.
- Sau lượt `context-engineering`, người dùng yêu cầu “commit và push đi”. Phạm vi là tài liệu context và commit kiến trúc đã hoàn tất, trên nhánh hiện tại; phần mã nguồn chưa commit bên dưới được giữ riêng.
- Bộ context đưa vào Git gồm `AGENTS.md`, `PROJECT_CONTEXT.md`, `CONTEXT.md`, tài liệu bàn giao này và hai hướng dẫn được tham chiếu: `docs/agents/domain.md`, `docs/agents/issue-tracker.md`.
- Chưa có nhiệm vụ tiếp theo được chỉ định. Phần chưa commit bên dưới được giữ nguyên cho công việc riêng của nó; sự hiện diện trong working tree không có nghĩa là đã được review hoặc sẵn sàng commit.

## Điểm vào mã theo nhiệm vụ

Đường dẫn đầy đủ trong bảng tính từ root repo; tên tệp rút gọn tiếp theo nằm cùng thư mục. Xem danh sách thay đổi bằng `git show --stat 31259cc`.

| Nhiệm vụ | Source và test cần đọc |
| --- | --- |
| Dò GPU, chính sách mở khóa, verdict | `windows-v3.0/tools/40hxcore/link.go`, `unlock_test.go`, `link_test.go` |
| Vòng đời driver, adapter HardwareBus | `windows-v3.0/tools/40hxcore/drvsession.go`, `drvsession_test.go` |
| StatusContract và cách diễn giải dữ liệu | `windows-v3.0/tools/40hxcore/status.go`, `status_test.go`, `link_status_test.go` |
| GPUTopologyDetector, MSHybrid/CASO, DirectX Global Router | `windows-v3.0/tools/riotcore/gpu_detector.go`, `registry.go`, `registry_adapter.go`, `graphics_test.go` |
| Caller AIO | `windows-v3.0/tools/aio/bind_unlock.go`, `bind_diag.go`, `bind_riot.go`, `status_contract_test.go` |
| Caller Installer và Web Control Center | `windows-v3.0/tools/inst40hx/main.go`, `webserver.go`, `web/app.js` |
| Caller UnlockRiotGame | `windows-v3.0/tools/unlockriot/main.go`, `cli.go`, `policy.go` |
| Hiển thị trạng thái | `windows-v3.0/tools/aio/frontend/js/telemetry.js`, `windows-v3.0/tools/aio/tests/telemetry.test.cjs`; tài nguyên Installer phải khớp `windows-v3.0/release/web/` |

Giữ các quyết định đã chốt:

- LinkNegotiator sở hữu discovery, clamp GPU Profile, đàm phán và verdict. Caller dùng nguyên verdict; Installer giữ mutex, chờ driver, GSP và lịch retry riêng.
- `NegotiationOptions.Prepare` nhận profile thực sự dò được trước đàm phán. Không thay bằng một lần dò Registry trước retry.
- MRRS đúng 512B; restart NVDisplay một lần khi cần, kể cả nhánh thành công TLS khi tốc độ đang nhàn rỗi. CMP 30HX luôn giữ Gen2 và cấm Stage 2/PnP reset theo AGENTS.md.
- Registry cấu hình dùng chung trả kết quả/lỗi từng bước, chọn CMP render thay vì GPU đang xuất hình, giữ tùy chọn DirectX khác khi hoàn tác.
- `LinkState` là dữ liệu lần chạy trước; cấu hình TLS khác tốc độ quan sát. Giá trị thiếu không thành x16 hay MRRS giả định; băng thông hiển thị là lý thuyết. Telemetry nền không nạp BYOVD.
- Node test đặt ngoài `frontend/` để không bị nhúng vào executable AIO.

## Working tree được giữ lại

Index rỗng sau commit. Các thay đổi sau đã có trước lượt triển khai; một số dependency/file nay được commit làm nền cho module dùng chung, nhưng phần extraction game còn lại vẫn chưa commit:

```text
 M windows-v3.0/tools/40hxcore/status.go
 M windows-v3.0/tools/aio/app.go
 M windows-v3.0/tools/aio/app_test.go
 M windows-v3.0/tools/aio/bind_diag.go
 M windows-v3.0/tools/aio/bind_riot.go
 M windows-v3.0/tools/aio/fallback_web.go
 M windows-v3.0/tools/aio/go.mod
 M windows-v3.0/tools/riotcore/go.mod
 M windows-v3.0/tools/unlockriot/game_finder.go
 M windows-v3.0/tools/unlockriot/go.mod
 M windows-v3.0/tools/unlockriot/league_cfg.go
 M windows-v3.0/tools/unlockriot/main.go
?? windows-v3.0/tools/riotcore/game_finder.go
?? windows-v3.0/tools/riotcore/game_finder_test.go
?? windows-v3.0/tools/riotcore/league_cfg.go
?? windows-v3.0/tools/riotcore/league_cfg_test.go
```

Phần này gồm extraction tìm game/cấu hình LoL sang riotcore, thay đổi event/shutdown AIO và hàm đọc status tiện dụng. Đọc diff hiện tại trước khi xử lý; giữ các tệp đó khi làm nhiệm vụ khác. `git diff` sau `31259cc` không phải toàn bộ phần triển khai kiến trúc vừa hoàn tất.

Các tệp context được chọn đã được thêm tường minh vào Git bằng `git add -f`; `.gitignore` vẫn giữ nguyên. Dùng `git ls-files` để phân biệt tài liệu được theo dõi với các ghi chú cục bộ khác.

## Bằng chứng kiểm chứng

Mốc kiểm chứng là bản stage đã được xuất riêng trước commit, tương ứng source của `31259cc`; không phải working tree có thêm extraction game chưa commit. Snapshot: `C:/Users/Admin/AppData/Local/Temp/cmp-architecture-verify-20261025-193311`.

Lệnh đầy đủ đã chạy cho từng module, từ `windows-v3.0/tools/<module>` của snapshot:

```powershell
& 'C:\Users\Admin\AppData\Local\Temp\cmp-architecture-go\go\bin\go.exe' test -json -count=1 ./...
```

| Module | Test/subtest đạt | Kết quả |
| --- | ---: | --- |
| 40hxcore | 57 | Đạt |
| riotcore | 17 | Đạt |
| unlockriot | 65 | Đạt |
| aio | 10 | Đạt |
| inst40hx | 6 | Đạt |
| check40x | 0 | Biên dịch, không có test |
| uninstall40x | 0 | Biên dịch, không có test |
| fix30hxcsm | 2 | Đạt |

Tổng 157 test/subtest. `go vet ./...` đã đạt tại 40hxcore, riotcore, unlockriot, aio và inst40hx. Kiểm tra frontend đã đạt với `node windows-v3.0/tools/aio/tests/telemetry.test.cjs`, gồm dữ liệu thiếu, link nhàn rỗi và mất kết nối không hiển thị GPU giả.

Các executable phát hành đã build và được commit: `40HXInstaller.exe`, `40HXCheck.exe`, `40HXUninstaller.exe`, `UnlockRiotGame.exe`, `UnlockAIO.exe`. Bốn công cụ đầu dùng `go build -ldflags '-s -w -H=windowsgui' -o <release-exe> .`; AIO dùng lệnh Wails trong AGENTS.md, từ snapshot. Đã kiểm tra PE x64 GUI, binding Wails không khác, hai bản web giống nhau và test không nằm trong frontend được embed.

Review Standards và Spec đã hoàn tất. Hai phát hiện về restart sau MRRS khi nhàn rỗi và chuẩn bị GSP sau discovery đã được sửa, có test hồi quy và được reviewer xác nhận lại.

## Tooling và giới hạn

- Go không có sẵn trong PATH của phiên vừa chạy. SDK Go 1.26.10 đã được tải từ go.dev và kiểm tra SHA256, nằm tại `C:/Users/Admin/AppData/Local/Temp/cmp-architecture-go/go/bin/`; Wails CLI 2.16.0 ở `C:/Users/Admin/AppData/Local/Temp/cmp-architecture-go/bin/wails.exe`. Các đường dẫn Temp có thể bị xóa; xác minh trước khi dùng. Yêu cầu phiên bản thực tế nằm trong từng `go.mod`.
- Log kiểm chứng ở `%TEMP%/cmp-architecture-test-<module>.log`; tổng hợp ở `%TEMP%/cmp-architecture-test-results.json`. Bản sao các tệp dirty ban đầu ở `%TEMP%/cmp-architecture-baseline-20261025-191543/`. Đây là bằng chứng tạm, không thay thế Git/spec.
- Git chưa có `user.name`/`user.email`; commit vừa hoàn tất dùng cấu hình riêng cho lệnh, theo tác giả nhất quán của lịch sử repo. Không đổi cấu hình toàn cục. `gh` chưa đăng nhập trong phiên này.
- Kiểm tra đã dùng adapter mô phỏng và biên dịch; chưa chạy mở khóa trên phần cứng thật. Tự động kiểm tra hiển thị báo cáo HTML bị timeout; Node check không chứng minh toàn bộ hành vi trình duyệt.
- Báo cáo HTML ban đầu trong Temp bị lỗi dấu tiếng Việt qua pipeline PowerShell; dùng spec đã commit cho nội dung kỹ thuật. Khi truyền văn bản Unicode sang Python, đặt `$OutputEncoding = [System.Text.UTF8Encoding]::new($false)` hoặc ghi qua `apply_patch`.
- Người dùng đã cho phép commit và push lên `origin/fix/aio-review-findings`. Trạng thái đồng bộ cần đối chiếu với remote; chưa có bước tạo PR, đóng gói ZIP hay chạy tối ưu phần cứng trong phạm vi này.
