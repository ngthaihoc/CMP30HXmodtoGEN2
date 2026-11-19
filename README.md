<p align="right">
  <b>Ngôn ngữ:</b>
  <b>Tiếng Việt</b> |
  <a href="README_EN.md">English</a> |
  <a href="README_ZH.md">简体中文</a>
</p>

# NVIDIA CMP 30HX / 40HX PCIe Unlock

Công cụ nghiên cứu mở liên kết PCIe Gen2 cho NVIDIA CMP 30HX (TU116, PCI ID `10DE:2189`) và CMP 40HX (TU106, PCI ID `10DE:1F0B`). Phần cứng được hỗ trợ trong mã Windows giới hạn ở hai ID này. Kết quả phụ thuộc GPU, bo mạch chủ, khe cắm, riser, firmware và driver. Gen2 x16 là mục tiêu kỹ thuật, không phải cam kết hiệu năng.

[![GitHub Releases](https://img.shields.io/badge/GitHub-Releases-181717?logo=github)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2/releases)

## Chọn công cụ

- **Windows CMP 30HX:** chạy `Setup_CMP30HX_WindowsAIO.bat` từ thư mục dự án với quyền Administrator.
- **Windows CMP 40HX:** xem [hướng dẫn Windows](windows-v3.0/README.md) và hướng dẫn đi kèm gói phát hành. Tệp EFI CMP 40HX chỉ dùng cho CMP 40HX.
- **Linux:** dùng `Setup_CMP30HX_LinuxAIO.sh` để cài, xem trạng thái hoặc gỡ service. Đọc trợ giúp của script trước khi chạy.
- **Resizable BAR / CSM:** `Setup_ReBar_CMP30HX.bat` và `Fix_CMP30HX_CSM_Error43.bat` là các tiện ích riêng, có thể thay đổi cấu hình hệ thống hoặc thiết bị. Xem prompt và hướng dẫn của chúng trước khi xác nhận.

### Chẩn đoán Resizable BAR trên CMP 30HX

Chạy `Setup_ReBar_CMP30HX.bat -status` để xem BAR1 hiện tại. BAR0 16 MB là một BAR khác và riêng con số này không xác nhận BAR1 đã được mở rộng. Tệp `rebar_status.txt` chỉ ghi nhận lần gửi cấu hình; sau khi khởi động lại cần xem GPU-Z (Advanced → PCIe Resizable BAR) và `nvidia-smi -q -d memory`.

Nếu NvStrapsReBar báo `Bridge Found`, DXE đã thấy PCI bridge nhưng vẫn chưa xác nhận BAR1 của GPU. Script sẽ thử gửi `E/S`, rồi yêu cầu kiểm tra lại sau reboot. Nếu vẫn không thay đổi, kiểm tra UEFI boot, CSM đã tắt và Above 4G Decoding đã bật. Mainboard cũ có thể cần cấu hình PCI-side BAR bằng mục `P` theo [hướng dẫn NvStrapsReBar](https://github.com/terminatorul/NvStrapsReBar/blob/master/README.md); script không tự thay đổi tùy chọn này vì phụ thuộc bo mạch. Gửi model mainboard, phiên bản BIOS, ảnh GPU-Z sau reboot và toàn bộ đầu ra NvStrapsReBar để chẩn đoán tiếp. Không thể bảo đảm BAR1 8 GiB trên mọi hệ thống.

`Setup_ReBar_CMP30HX.bat -mock -nowait` chỉ mô phỏng luồng, không ghi EFI, task, Registry hay tệp cài đặt.

## Windows CMP 30HX

Tải và giải nén đầy đủ ZIP của dự án hoặc gói phát hành. Giữ nguyên thư mục đã giải nén để có thể gỡ cài đặt. Mở Terminal hoặc Command Prompt với quyền Administrator tại thư mục này.

```cmd
Setup_CMP30HX_WindowsAIO.bat
Setup_CMP30HX_WindowsAIO.bat -preflight
Setup_CMP30HX_WindowsAIO.bat -notask
Setup_CMP30HX_WindowsAIO.bat -cleantasks
Setup_CMP30HX_WindowsAIO.bat -uninstall
```

Menu tương tác có năm lựa chọn: [1] cài đặt và mở khóa, [2] gỡ bộ cài, [3] sửa NVIDIA Control Panel, [4] tùy chọn cấu hình Riot/Unity, [5] thoát. Lựa chọn [3] và [4] là thao tác riêng, có thể thay đổi cấu hình NVIDIA hoặc game.

- `-preflight` chỉ kiểm tra phần cứng. Yêu cầu đúng **một** GPU đang kết nối có PCI ID CMP 30HX `10DE:2189`. Không có hoặc có nhiều hơn một GPU khớp sẽ bị từ chối.
- Sau khi mở khóa thành công, script đăng ký tác vụ SYSTEM `CMP30HX_Batch_Gen2_Unlock`: chạy sau khởi động 45 giây hoặc sau đăng nhập 10 giây. Tệp cài được đặt trong `%ProgramFiles%\40HXUnlock`, trạng thái batch trong `%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt`.
- `-notask` bỏ qua đăng ký persistence ở lần cài mới. Mở khóa chỉ có hiệu lực trong phiên hiện tại.
- `-cleantasks` gỡ tác vụ batch và tên tác vụ cũ chỉ khi chúng chạy `RunUnlock.bat` hoặc `UnlockAIO.exe` trong `%ProgramFiles%\40HXUnlock`; khóa Run trỏ tới hai tệp này cũng được dọn. Tác vụ khác và tệp công cụ được giữ lại.
- `-uninstall` gỡ tác vụ persistence và các tệp thuộc bộ cài, đồng thời giữ lại driver, service và cấu hình dùng chung. Khởi động lại để xóa trạng thái mở khóa runtime của phiên hiện tại.

Để xác minh, kiểm tra kết quả cài đặt và trạng thái PCIe dưới tải GPU, sau đó khởi động lại và kiểm tra lại. Tác vụ có thể xem bằng:

```powershell
Get-ScheduledTask -TaskName 'CMP30HX_Batch_Gen2_Unlock'
```

Chọn [2] trong menu hoặc chạy `-cleantasks` để gỡ persistence. Xác nhận tác vụ đã được gỡ bằng:

```cmd
schtasks /query /tn "CMP30HX_Batch_Gen2_Unlock"
```

Lệnh sẽ báo không tìm thấy tác vụ sau khi gỡ thành công. Khởi động lại rồi kiểm tra link không còn được mở khóa tự động. Giữ bản ZIP/thư mục dự án cho đến khi hoàn tất gỡ bỏ.

## Control Center và tiện ích

`Launch_WebUI.bat` khởi chạy CMP Control Center nếu có trong gói. Giao diện AIO cung cấp các tab mở khóa Gen2, chẩn đoán, Riot & 3D, công cụ phụ và gỡ cài đặt. Phần chẩn đoán hiển thị thông tin PCIe, MRRS, GSP, xung/nhiệt độ và ma trận lane. Tác vụ Riot/Unity và sửa NVIDIA Control Panel là thao tác tùy chọn, không chạy tự động trong luồng mở khóa mặc định.

## Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh --status
sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
./Setup_CMP30HX_LinuxAIO.sh -test --no-root
```

Gỡ Linux xóa service, sleep hook và các tệp được cài cho công cụ. Việc này không hoàn nguyên các thay đổi thủ công hoặc mọi trạng thái cấu hình phần cứng. Khởi động lại nếu cần làm mới trạng thái PCIe runtime.

## Kiểm tra, khôi phục và giới hạn

Dùng GPU-Z hoặc công cụ chẩn đoán khi GPU đang chịu tải. GPU nhàn rỗi có thể báo Gen1 do quản lý điện năng. Benchmark thay đổi theo nền tảng và workload, vì vậy không xem một con số băng thông là kết quả được bảo đảm.

Kiểm thử mô phỏng xác nhận luồng phần mềm, không xác nhận mở khóa trên CMP thật, ổn định qua sleep/reboot hoặc tương thích anti-cheat. Chưa có cam kết tương thích Vanguard, EAC hay BattlEye. Cần kiểm chứng trên đúng phần cứng trước khi sử dụng hàng ngày.

Nếu hệ thống CMP 40HX không khởi động sau khi dùng luồng EFI, chỉ áp dụng [hướng dẫn cứu hộ EFI](windows-v3.0/release/EFI应急修复指南.md) khi gặp lỗi boot EFI được nêu trong tài liệu. Không chỉnh phân vùng EFI khi máy vẫn khởi động bình thường.

## Tải xuống, đóng góp và ghi nhận

Tải gói phát hành từ [GitHub Releases](https://github.com/ngthaihoc/CMP30HXmodtoGEN2/releases). Dùng hướng dẫn đi kèm đúng gói vì nội dung có thể khác theo phiên bản.

Dự án được chia sẻ miễn phí. Nếu công cụ hữu ích, bạn có thể ủng hộ tác giả qua MoMo/VietQR trong [ảnh QR](assets/donate_momo.jpg). Dự án lấy cảm hứng từ các công trình nghiên cứu ban đầu của cộng đồng CMP40HX-Unlock.

Tuân thủ luật địa phương, điều khoản nhà sản xuất và quy định của phần mềm bạn sử dụng.

## CI/CD

GitHub Actions chạy kiểm thử Go, batch/PowerShell, frontend và Linux mock trên pull request, push lên `main` và tag `v*`. Khi các kiểm tra và bản build Windows thành công, tag `v*` tạo ZIP giữ cấu trúc repository, tệp SHA-256 và một GitHub Release **bản nháp**. Sáu công cụ Windows được build lại từ mã của tag; các thành phần binary/EFI khác lấy từ repository. Maintainer kiểm tra nội dung, checksum, release notes và kết quả thử trên phần cứng phù hợp trước khi xuất bản bản nháp. Kiểm thử CI không xác nhận hoạt động trên GPU thật. Cần cấu hình branch protection yêu cầu các job CI trong GitHub; workflow không tự bật thiết lập này.
