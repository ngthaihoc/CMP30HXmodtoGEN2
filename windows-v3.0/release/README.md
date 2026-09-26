# Bộ Công Cụ Mở Khoá CMP 40HX Cho Windows v3.0.0

**Card Đồ Hoạ Trâu Cày CMP 40HX (TU106) → Mở Khoá Tối Đa Tốc Độ Tensor Core + PCIe Gen2**  
Chạy trực tiếp trên Windows nguyên bản, cài đặt một chạm, tự động chạy khi khởi động, không cần thao tác thủ công mỗi lần mở máy.  
Từ bản v2.5 trở đi **không cần bật chế độ Test Signing**, hệ thống luôn sạch sẽ, không ảnh hưởng đến phần mềm chống gian lận (Anti-cheat) khi chơi game.

**Kết quả kiểm tra thực tế** (xác minh trên máy thử nghiệm):
| Chỉ số | Kết quả |
|---|---|
| Năng lực tính toán (Compute/Hashrate) Tensor | `SS0=0x88888888`, FP16 HGEMM **~50 TFLOPS** (Trạng thái bị khoá ~8T) |
| Băng thông PCIe | Gen2 ×16 (Băng thông ~6.4 GB/s, gấp đôi Gen1) |
| Driver NVIDIA | Hoạt động bình thường không lỗi Code 43, bật GSP |

---

## 0. Gặp sự cố? Hãy copy toàn bộ nội dung file 《AI辅助安装提示词.txt》 gửi cho AI

> Khi gặp khó khăn trong quá trình cài đặt / mở khoá, hãy mở file **`AI辅助安装提示词.txt`** cùng thư mục, copy toàn bộ nội dung phía dưới đường phân cách gửi cho bất kỳ
> trợ lý AI nào (ChatGPT / Claude / DeepSeek...), sau đó gửi kèm kết quả chẩn đoán từ 40HXCheck.exe khi AI yêu cầu.
> Dữ liệu chẩn đoán sẽ tự động được gom vào `%LOCALAPPDATA%\40HXUnlock\logs\` và tự động sao chép vào Clipboard.

---

## Mục lục

0. [Prompt AI hỗ trợ cài đặt](#0-gặp-sự-cố-hãy-copy-toàn-bộ-nội-dung-file-ai辅助安装提示词txt-gửi-cho-ai)
1. [Các thành phần trong bộ cài](#1-các-thành-phần-trong-bộ-cài)
2. [Chuẩn bị BIOS trước khi cài đặt (Bắt buộc đọc)](#2-chuẩn-bị-bios-trước-khi-cài-đặt-bắt-buộc-đọc)
3. [Cài đặt và cấu hình](#3-cài-đặt-và-cấu-hình)
4. [Kiểm tra sau khi khởi động lại (Nhấp đúp 40HXCheck.exe)](#4-kiểm-tra-sau-khi-khởi-động-lại-nhấp-đúp-40hxcheckexe)
5. [Xử lý sự cố](#5-xử-lý-sự-cố)
6. [Tương thích Riot Games (Valorant / Vanguard) & Secure Boot](#6-tương-thích-riot-games-valorant--vanguard--secure-boot)
7. [Lưu ý khi sử dụng hàng ngày](#7-lưu-ý-khi-sử-dụng-hàng-ngày)
8. [Gỡ cài đặt và hoàn nguyên](#8-gỡ-cài-đặt-và-hoàn-nguyên)
9. [Lỗi boot EFI? Hướng dẫn cứu hộ khẩn cấp](#9-lỗi-boot-efi-hướng-dẫn-cứu-hộ-khẩn-cấp)
10. [Lịch sử phiên bản](#10-lịch-sử-phiên-bản)

---

## 1. Các thành phần trong bộ cài

| Tệp tin | Mục đích |
|---|---|
| `40HXInstaller.exe` | **Giao diện cài đặt + Quản lý** (Mặc định mở GUI; hỗ trợ đầy đủ tham số dòng lệnh) |
| `40HXUninstaller.exe` | **Gỡ cài đặt tự động** (Nhấp đúp chuột → Yêu cầu quyền Administrator) |
| `40HXCheck.exe` | **Chẩn đoán độc lập** (Nhấp đúp kiểm tra ngay: Trạng thái Năng lực tính toán + Gen2; chủ yếu ở chế độ đọc, tự dọn dẹp sau khi đo) |
| `UnlockRiotGame.exe` | **Mở khoá tương thích Riot Games / Vanguard**: Tự động tạo chứng chỉ, ký Authenticode cho `40HXUNLK.EFI` và hướng dẫn nạp key vào BIOS `db` (Custom Mode) trên Windows 11 cho CMP 40HX; kiểm tra và giữ nguyên Secure Boot cho CMP 30HX |
| `OpenCL.exe` | **Kiểm tra năng lực tính toán** (Nhấp đúp để chạy, so sánh hiệu năng số thực OpenCL trước và sau khi mở khoá; đối chiếu bảng thông số ở đầu trang) |
| `files\40HXUNLK.EFI` | Firmware mở khoá (Bộ cài dùng để triển khai; tệp dùng để tạo USB boot cứu hộ cũng là file này) |
| `make_usb_efi.bat` | **Phương án dự phòng boot**: Tự động nhận diện USB và chép EFI mở khoá (Dùng khi boot USB thủ công, xem mục 2.2; tham số `/restore` để hoàn nguyên USB) |
| `EFI应急修复指南.md` | **Chỉ dùng khi máy bị lỗi boot khởi động** (Cách dùng USB cài Windows để sửa lại BCD) |
| `README.md` | Tài liệu hướng dẫn này |

> Ba file exe đảm nhận nhiệm vụ riêng biệt: Installer = **Cài đặt / Cấu hình**; Uninstaller = **Gỡ cài đặt**; Check = **Kiểm tra / Chẩn đoán**.
> Khi gặp sự cố chỉ cần nhấp đúp vào Check là xong, không cần gõ lệnh.

---

## 2. Chuẩn bị BIOS trước khi cài đặt (Bắt buộc đọc)

Firmware mở khoá không có chữ ký của Microsoft và nạp dữ liệu vào vùng nhớ trên 4GB, do đó các thiết lập BIOS dưới đây **bắt buộc phải tuân thủ đầy đủ**.  
Phím tắt vào BIOS: `Del` / `F2` (một số bo mạch chủ dùng F1/F10/F12).

### 2.1 Các mục bắt buộc phải bật / tắt

| Ưu tiên | Mục cài đặt | Giá trị | Giải thích |
|---|---|---|---|
| ⭐ | **Above 4G Decoding** | **Enabled** | Nếu tắt mục này thì mở khoá chắc chắn thất bại trong im lặng, đây là nguyên nhân số 1 khiến "màn hình mở khoá xuất hiện nhưng card vẫn bị khoá tốc độ" |
| ⭐ | **Secure Boot** | **Disabled** | Khi bật mục này, firmware không có chứng thực số sẽ bị chặn (nếu mục này bị mờ thì cần tắt CSM trước) |
| ⭐ | **CSM / Chế độ tương thích** | Tắt (Pure UEFI) | Để mục khởi động "40HX Unlock" xuất hiện trong danh sách boot của BIOS |
| | **Fast Boot / Khởi động nhanh** | Disabled | Tránh việc BIOS bỏ qua các dịch vụ nạp firmware mở khoá |
| | **Resizable BAR** | Auto / Enabled | Nếu bo mạch chủ có hỗ trợ, hãy bật cùng với Above 4G |

### 2.2 Vị trí khe cắm và thứ tự khởi động

- **Cắm CMP 40HX vào khe PCIe x16 đầu tiên** (khe kết nối trực tiếp với CPU)
- Người dùng chạy 2 card: Cắm card xuất hình ở khe phụ, giữ 40HX ở khe chính x16
- **Đặt mục "40HX Unlock" lên vị trí ưu tiên số 1 trong Boot Priority**

**BIOS không thấy '40HX Unlock' / Đã đặt ưu tiên 1 nhưng không chạy? → Dùng USB boot mở khoá thủ công (Phương án dự phòng, không phụ thuộc vào mục boot BIOS):**

1. Chuẩn bị một **USB định dạng FAT32**, sao chép file `files\40HXUNLK.EFI` vào 2 đường dẫn sau trên USB:
   `\EFI\40HX\40HXUNLK.EFI` và `\EFI\Boot\bootx64.efi` (bootx64 là tên chuẩn fallback của UEFI, nếu USB có sẵn file này thì sao lưu trước rồi ghi đè).
2. Khi bật máy, bấm liên tục **phím mở Boot Menu** (Asus/Gigabyte: F8, MSI: F11, Lenovo: F12; hoặc vào BIOS chọn mục boot) → Chọn **USB có tên bắt đầu bằng chữ UEFI:**.
3. Xuất hiện dòng chữ mở khoá 40HX khoảng 10~30 giây = Dữ liệu mở khoá đã được nạp thành công; nếu sau đó máy không tự vào Windows, hãy khởi động lại và mở Boot Menu chọn **Windows (ổ cứng)** để vào hệ thống.
4. Vào Windows mở 40HXCheck.exe để kiểm tra (báo SS0=0x88888888 là đã thành công).

> Tệp `make_usb_efi.bat` trong thư mục này có thể tự động thực hiện toàn bộ quy trình: "Quét USB → Sao chép vào cả 2 đường dẫn → Kiểm tra đối soát dữ liệu": Nhấp đúp chuột để chạy, script sẽ liệt kê các ổ USB tìm thấy,
> nếu chỉ có 1 USB thì nhấn Enter để chọn, nếu có nhiều USB thì nhập chữ cái ổ đĩa (hoặc gõ lệnh `make_usb_efi.bat E` trực tiếp).
> Sau khi dùng xong, chạy `make_usb_efi.bat /restore` để hoàn nguyên USB về trạng thái ban đầu.
> Phương án này áp dụng cho các trường hợp đặc biệt như "BIOS không nhận mục boot / firmware không thực thi / chuỗi boot bị gián đoạn"; quy trình chuẩn khi hệ thống ổn định vẫn nên dùng bộ cài Installer.

### 2.3 Không tìm thấy các mục cài đặt trong BIOS?

Vị trí tham khảo trên một số dòng bo mạch chủ:
- Asus: Advanced → PCI Subsystem Settings → Above 4G Decoding
- MSI: Settings → Advanced → PCI Subsystem Settings → Above 4G
- Gigabyte: Peripherals → Above 4G Decoding
- Một số bo mạch chủ cần bật tuỳ chọn "Windows 8/10 Features / UEFI Boot" trước thì các mục trên mới hiển thị.

### 2.4 Chế độ boot và cấu hình nguồn điện

**Chế độ boot bắt buộc phải là UEFI + GPT.** Việc mở khoá năng lực tính toán dựa vào firmware nạp trong môi trường UEFI. Nếu ổ đĩa dùng chuẩn cũ BIOS Legacy + MBR
sẽ không có phân vùng EFI, dẫn đến việc không thể cài đặt EFI mở khoá: đây là nguyên nhân gốc rễ của lỗi "không cài được EFI / tốc độ tính toán luôn bị khoá".

Cách kiểm tra: Nhấn `Win + R` → Nhập `msinfo32` → Enter, nhìn vào dòng "BIOS Mode":
- Hiển thị **UEFI** → Chuẩn xác, bình thường;
- Hiển thị **Legacy (Truyền thống)** → Cần chuyển đổi ổ đĩa sang GPT bằng công cụ chính thức của Microsoft (`mbr2gpt`):
  1. **Sao lưu dữ liệu quan trọng**; nếu có bật BitLocker thì tạm dừng trước.
  2. Mở Command Prompt với quyền Administrator và chạy: `mbr2gpt /validate /allowfullos`
  3. Sau khi thấy thông báo Validation completed successfully, chạy tiếp: `mbr2gpt /convert /allowfullos`
  4. Khởi động lại máy vào BIOS, chuyển chế độ Boot Mode từ Legacy sang **UEFI** (và tắt CSM).
  5. Vào lại Windows và chạy lại `40HXInstaller.exe`.

  Lưu ý: Quá trình chuyển đổi là **một chiều** (không thể chuyển ngược lại MBR không mất dữ liệu); yêu cầu Windows 10 bản 1703 trở lên hoặc Windows 11 và bo mạch chủ hỗ trợ UEFI.
  Bộ cài Installer sẽ tự động phát hiện ở bước [5/8] và hiển thị hướng dẫn này, đồng thời **không** bỏ qua phần cài đặt mở khoá Gen2.

**Hai thiết lập nguồn điện quan trọng (Bộ cài đã tự động cấu hình, hướng dẫn thủ công để tra cứu khi cần):**

| Cài đặt | Lý do cần tắt | Cách tắt thủ công | Cách khôi phục |
|---|---|---|---|
| Khởi động nhanh Windows (Fast Startup / Hiberboot) | Khi bật tính năng này, "Tắt máy → Bật lại" thực chất là nạp lại trạng thái ngủ đông (không boot UEFI hoàn chỉnh), firmware mở khoá EFI có thể bị bỏ qua | Control Panel → Power Options → Choose what the power buttons do → Bỏ dấu tick "Turn on fast startup" | Tick chọn lại mục này |
| Power Plan → PCI Express → Link State Power Management (ASPM) | Khi bật, GPU khi ở trạng thái nghỉ (idle) sẽ tự hạ xuống Gen1 để tiết kiệm điện, dễ gây hiểu nhầm là "Mở khoá Gen2 thất bại" (khi có tải nặng sẽ tự bung lên Gen2, không gây hại) | Mở CMD Admin: `powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0`, chạy tiếp lệnh tương tự với `-setdcvalueindex`, cuối cùng chạy `powercfg -setactive SCHEME_CURRENT` để áp dụng | Thay số `0` ở cuối thành `1` (Tiết kiệm vừa) hoặc `2` (Tiết kiệm tối đa) rồi chạy lại 3 lệnh |

> Lưu ý bổ sung: Vùng ① trên GUI còn cung cấp tuỳ chọn "High Performance Power Plan" (không bắt buộc). Bộ cài tự động xử lý
> việc tắt Fast Startup + tắt ASPM; tuỳ chọn High Performance có thể tick chọn trên GUI hoặc cài thủ công (xem mục 3.0).

> Lưu ý: Dù đã tắt ASPM, một số phiên bản driver vẫn có thể tự hạ xuống Gen1 khi card rảnh rỗi: đây là hành vi tiết kiệm điện bình thường của phần cứng,
> khi đào coin hoặc chạy AI tải liên tục, liên kết sẽ tự động chuyển về Gen2 sau vài giây. Tiêu chuẩn đánh giá mở khoá Gen2 thành công chuẩn xác nhất là
> **Tốc độ mục tiêu (Target Link Speed - TLS)** (40HXCheck.exe sẽ hiển thị "Mục tiêu Gen2 / TLS=Gen2"), chứ không phụ thuộc vào tốc độ tức thời lúc không tải.
> Nếu muốn hạn chế tối đa việc hạ tốc độ, bạn có thể vào NVIDIA Control Panel chỉnh chế độ Power management mode của 40HX thành "Prefer maximum performance" (tuỳ chọn).

### 2.5 Chính sách Gen2 (Tuỳ chọn nâng cao)

Khoá Registry `HKLM\SOFTWARE\40HXUnlock` quản lý chính sách Gen2 (có thể chỉnh trực tiếp trên giao diện GUI tại vùng ② Chính sách Gen2; hoặc qua dòng lệnh bên dưới):

> **Tóm tắt dễ hiểu**: Quá trình mở khoá Gen2 cần nạp tạm thời 2 driver kernel (ThrottleStop/WinRing0), sau khi hoàn tất việc xử lý 2 driver này ra sao do mục số 1 quyết định:
> ① "Dùng xong gỡ ngay" = Sau khi mở khoá xong sẽ tự động dọn sạch driver khỏi bộ nhớ (Mặc định, sạch sẽ nhất cho chơi game và phần mềm chống gian lận);
> ② "Thất bại tự thử lại" (Trước đây gọi là Watchdog) = Nếu mở khoá lỗi sẽ tự động thử lại theo số lần và khoảng cách bên dưới, mở khoá thành công vẫn tự dọn sạch driver;
> ③ "Dịch vụ thường trú" = Giữ driver trong hệ thống + Mỗi 1 phút tự động kiểm tra lại Gen2 (Nếu mất cấu hình TLS sẽ tự động ghi đè và huấn luyện lại), cần đăng ký tác vụ tự khởi động khi đăng nhập (Phần mềm chống gian lận có thể cảnh báo, hãy cân nhắc).

| Tên khoá | Mặc định | Ý nghĩa |
|---|---|---|
| `DriverStrategy` | `0` | Chính sách driver: `0`=Dùng xong gỡ ngay (Mặc định) · `1`=Thất bại tự thử lại (Theo cấu hình 2 khoá bên dưới) · `2`=Dịch vụ thường trú (Giữ driver, tác vụ tự kiểm tra mỗi phút, tự huấn luyện lại nếu mất TLS; phần mềm chống gian lận có thể cảnh báo) |
| `Gen2AutoHard` | `1` (Bật) | Khi Gen2 chưa đạt sẽ tự động thực thi Stage2 rollback (Link Disable + PnP phục hồi, chớp ngắt liên kết khoảng 5~10 giây). Đặt thành `0` để tắt, thích hợp cho máy dùng 40HX làm card xuất hình duy nhất để tránh chớp màn hình sau khi đăng nhập |
| `Gen2RetryCount` | `3` | Số lần tự động thử lại sau khi thất bại (0=Không thử lại) |
| `Gen2RetryIntervalMin` | `1` (Phút) | Khoảng thời gian giữa các lần thử lại |

> Cách thay đổi (quyền Administrator): `reg add HKLM\SOFTWARE\40HXUnlock /v Gen2AutoHard /t REG_DWORD /d 0 /f`  
> Khi gỡ cài đặt (40HXUninstaller.exe / -uninstall), toàn bộ khoá Registry chính sách này sẽ được tự động xoá sạch.

---

## 3. Cài đặt và cấu hình

### 3.0 Giao diện đồ hoạ GUI (Mặc định)

**Nhấp đúp chuột vào `40HXInstaller.exe`** để mở giao diện quản lý cửa sổ đơn (không chia tab, các tham số dòng lệnh CLI vẫn hỗ trợ đầy đủ, xem mục 3.3).  
Khi mở lên, chương trình sẽ **tự động quét trạng thái hệ thống một lần** (ở chế độ chỉ đọc): tự động tick chọn các mục "còn thiếu / chưa đạt chuẩn" (mục đã hoàn tất sẽ không tick để tránh ghi đè). Giao diện chia làm 3 khu vực từ trên xuống dưới:

| Khu vực | Nội dung |
|---|---|
| ① Cài đặt thành phần & Môi trường | Các tuỳ chọn: Bật GSP / EFI Compute + Mục boot firmware / Cài driver Gen2 + Loại trừ Defender / Tự khởi động Gen2 khi đăng nhập / Tắt Fast Startup / Tắt ASPM / Power Plan hiệu năng cao / Tắt bảo vệ thời gian thực của Defender. Dòng trạng thái trên cùng tóm tắt môi trường (nếu đạt chuẩn sẽ báo "✓ Môi trường sẵn sàng"). Bấm [Cài đặt các thành phần đã chọn] để thực thi các mục đã tick; hoặc bấm [Cài đặt toàn bộ] (Toàn bộ quy trình: GSP + Driver Gen2 + EFI/Mục boot + Tự khởi động + Tắt Fast Startup/ASPM) |
| ② Chính sách Gen2 | 3 chế độ chạy driver (Dùng xong gỡ ngay = Mặc định / Thất bại tự thử lại / Dịch vụ thường trú), công tắc tự động Stage2 rollback, số lần và khoảng cách thử lại (Mặc định 3 lần / 1 phút); Nút [Lưu chính sách] ghi vào Registry có hiệu lực ngay, nút [Kích hoạt Gen2 ngay] chỉ mở khoá phiên này (không tạo tự khởi động); nút [Kích hoạt Gen2 & Cài đặt tự khởi động] vừa mở khoá ngay vừa đăng ký tự chạy khi bật máy (Khuyến nghị: một chạm dùng luôn, không cần reboot) |
| ③ Nhật ký thao tác | Báo cáo kiểm tra môi trường và dữ liệu xuất thời gian thực của mọi tác vụ (Trạng thái phần mềm diệt virus, Defender, kết quả ghi thanh ghi đều hiển thị tại đây) |

- Mục đã sẵn sàng sẽ không được tick = Tránh ghi đè thiết lập hiện có; toàn bộ kết quả quét đều hiển thị minh bạch trong nhật ký.
- Chức năng gỡ cài đặt không nằm trong giao diện này: Hãy dùng `40HXUninstaller.exe` (hoặc chạy lệnh `40HXInstaller.exe -uninstall`).

### 3.1 Cài đặt một chạm (Khuyến nghị cho người mới)

1. Nhấp đúp chuột vào `40HXInstaller.exe` (chọn Yes khi hộp thoại UAC hỏi quyền Administrator).
2. Tại khu vực ①: Các thành phần còn thiếu đã được tự động tick chọn (mục đã có sẽ không tick), bấm [Cài đặt các thành phần đã chọn]; hoặc bấm thẳng vào [Cài đặt toàn bộ] (có thể tick thêm High Performance Plan / Tắt bảo vệ thời gian thực Defender nếu muốn).
3. Bộ cài sẽ tự động thực hiện: Quét môi trường → Bật GSP → Triển khai firmware mở khoá (Ghi đúp vào phân vùng ESP + Đặt mục boot lên ưu tiên 1) → Đăng ký tác vụ tự khởi động Gen2 khi đăng nhập.
4. **Khởi động lại máy tính** (Bộ cài đã tự tắt Fast Startup nên bạn chọn "Khởi động lại" hay "Tắt máy rồi bật lại" đều được; nếu nhật ký báo tắt Fast Startup thất bại thì hãy chọn Khởi động lại hoặc làm theo mục 2.4 để tắt thủ công).
5. Khi mở máy, màn hình sẽ hiện dòng chữ mở khoá trong 1–2 giây → sau đó tự động vào Windows.
6. Sau khi đăng nhập vào Windows, chờ khoảng vài giây, Gen2 sẽ tự động được kích hoạt và driver tự động dọn dẹp khỏi RAM, bạn không cần thao tác thêm gì cả.

Tệp nhật ký cài đặt: `%TEMP%\40HX_installer.log`

### 3.2 Bộ cài không chạy được?

`40HXInstaller.exe` đã tích hợp đầy đủ mọi bước cài đặt (GSP / Firmware mở khoá / Mục boot / Tự khởi động Gen2 / Nguồn điện), **không cần dùng script cài đặt thủ công riêng lẻ nữa**. Nếu không mở được hoặc báo lỗi giữa chừng, hãy xử lý theo thứ tự sau:

1. **Nhấp chuột phải vào `40HXInstaller.exe` → Chọn Run as administrator** (Bắt buộc chọn Yes tại UAC; quyền người dùng thường không thể ghi vào phân vùng ESP và Registry).
2. **Bị phần mềm diệt virus bên thứ 3 chặn**: Hãy thêm toàn bộ thư mục bộ cài vào danh sách tin cậy / loại trừ của phần mềm rồi thử lại (Windows Defender đã được bộ cài tự động thêm loại trừ).
3. **Xem file nhật ký**: `%TEMP%\40HX_installer.log`, chi tiết từng bước lỗi đều được ghi lại tại đây.
4. **Không muốn mở giao diện đồ hoạ**: Sử dụng các tham số dòng lệnh ở mục 3.3 để chạy từng bước (chạy `40HXInstaller.exe -status` trước để xem trạng thái).

Nếu vẫn không được, hãy gửi file `%TEMP%\40HX_installer.log` cùng tóm tắt chẩn đoán từ `40HXCheck.exe` cho tác giả để được hỗ trợ.

### 3.3 Các tham số dòng lệnh CLI (Dành cho người dùng nâng cao / Gọi qua script)

```cmd
40HXInstaller.exe              # Mở giao diện đồ hoạ GUI
40HXInstaller.exe -task        # Chỉ đăng ký tác vụ tự khởi động Gen2 khi đăng nhập (quyền Admin)
40HXInstaller.exe -gen2        # Kích hoạt mở khoá Gen2 ngay một lần (quyền Admin)
40HXInstaller.exe -gen2 -hard  # Ép buộc chạy Stage2 Link Disable rollback khi Gen2 thất bại
40HXInstaller.exe -uninstall   # Gỡ cài đặt toàn bộ mức thành phần (tương đương 40HXUninstaller.exe)
40HXInstaller.exe -status      # Kiểm tra toàn diện trạng thái phần cứng, driver và PCIe
```

---

## 4. Kiểm tra sau khi khởi động lại (Nhấp đúp 40HXCheck.exe)

**Sau khi vào Windows, nhấp đúp vào `40HXCheck.exe`** (chủ yếu ở chế độ chỉ đọc; khi cần đo kiểm thực tế sẽ tạm thời nạp driver và tự dọn dẹp ngay sau khi đo xong):
- Hai dòng đầu tiên trên cửa sổ bật lên là kết quả bạn cần chú ý:
  - **Năng lực tính toán (Compute)**: Báo "✓ Tối đa (Full) (SS0=0x88888888)" = Mở khoá năng lực tính toán thành công.
  - **PCIe**: Báo Gen2 = Mở khoá băng thông liên kết thành công.
- Nếu chưa mở khoá / chưa đạt chuẩn, hướng dẫn khắc phục chi tiết sẽ được ghi vào file `diagnose.txt` và tự động copy vào Clipboard (cửa sổ sẽ hiển thị gợi ý "Bước tiếp theo"); khi gặp lỗi bạn chỉ việc paste nội dung này để hỏi hỗ trợ.
- Nhật ký chẩn đoán tự động gom về thư mục `%LOCALAPPDATA%\40HXUnlock\logs\`.
- Nếu lần đầu mở lên có thông báo "Yêu cầu quyền Administrator", hãy **nhấp chuột phải → Run as administrator** (để công cụ có thể đọc nhật ký mở khoá từ phân vùng ESP).

Các cách xác nhận khác: Kiểm tra bằng GPU-Z thấy Bus Interface hiển thị 2.0 / 5 GT/s; chạy thử suy luận AI FP16 đạt ~50 TFLOPS (trước khi mở khoá chỉ đạt khoảng 8T); hoặc nhấp đúp vào **`OpenCL.exe`** trong thư mục này để đo điểm số OpenCL so sánh trước và sau khi mở khoá.

---

## 5. Xử lý sự cố

### 5.1 Sử dụng công cụ chẩn đoán tự động trước tiên

**Nhấp đúp vào `40HXCheck.exe`**: Công cụ sẽ đánh dấu ✓✗ từng mục kèm kết luận cuối cùng, bạn không cần phải tự đọc và phân tích file log phức tạp.

### 5.2 Bảng tra cứu lỗi thường gặp

| Hiện tượng | Nguyên nhân | Hướng khắc phục |
|---|---|---|
| Bật máy vào thẳng Windows, không hiện màn hình mở khoá | Firmware EFI chưa được thực thi | Kiểm tra xem mục boot 40HX đã đặt lên vị trí số 1 chưa / Secure Boot / Fast Boot đã tắt chưa (xem mục 2) |
| Màn hình mở khoá hiện chữ "not found" rồi vào thẳng Windows (firmware cũ) | EFI chỉ quét bus 0–7, không tìm thấy card ở các bus cao (do AGESA / nhiều chip cầu) | Nâng cấp lên v3.0 dùng bộ cài nạp lại EFI (bản mới quét ban đầu 0–16 + quét trực tiếp cổng CF8 toàn bộ 0–255 dự phòng); nếu vẫn not found hãy gửi file 40hx_log.txt (chứa đoạn "diag: CF8 visible devices") cho tác giả |
| 40HXCheck báo Gen2 nhưng [Tác vụ Gen2] báo Chưa đăng ký | Giá trị tốc độ mục tiêu TLS của lần mở khoá trước còn sót lại (phiên bật máy này chưa từng chạy mở khoá) | Cần đăng ký tác vụ để tự động mở khoá mỗi khi mở máy: Vào GUI ② bấm [Kích hoạt Gen2 & Cài đặt tự khởi động], hoặc vào vùng ① tick chọn cài đặt; tắt máy hoàn toàn rồi bật lại và dùng 40HXCheck kiểm tra lại (giá trị cũ sẽ mất khi tắt nguồn card) |
| Màn hình mở khoá xuất hiện nhưng năng lực tính toán vẫn bị khoá | **Above 4G Decoding chưa bật** | Vào BIOS bật tính năng này, sau đó tắt hẳn máy rồi bật lại (xem mục 2.1) |
| Bộ cài / Chẩn đoán báo chuẩn boot là Legacy+MBR | Ổ đĩa dùng định dạng MBR, không có phân vùng EFI | Làm theo mục 2.4 dùng lệnh `mbr2gpt` chuyển đổi không mất dữ liệu sang GPT rồi chạy lại bộ cài |
| Báo lỗi Code 43 / Mất driver | Mở khoá chưa trọn vẹn, GSP chưa bật hoặc trạng thái bị lưu đè | Bật Above 4G trước; chạy lại bộ cài; nếu vẫn Code 43 thì dùng Uninstaller dọn sạch một lần rồi cài lại từ đầu |
| 40HXCheck báo "Năng lực tính toán bị khoá" | Phiên khởi động này chưa nạp firmware mở khoá, hoặc GPU vừa bị reset | **Tắt hẳn nguồn máy tính rồi bật lại** (không dùng nút "Khởi động lại / Restart") |
| 40HXCheck báo "Tác vụ Gen2: Chưa đăng ký" | Tác vụ tự khởi động khi đăng nhập chưa được tạo (nguyên nhân số 1 khiến Gen2 không tự kích hoạt) | Mở 40HXInstaller.exe → GUI ② bấm [Kích hoạt Gen2 & Cài đặt tự khởi động]; hoặc vùng ① tick [Tự khởi động Gen2 khi đăng nhập] rồi bấm cài đặt; đăng xuất rồi đăng nhập lại máy sẽ tự mở khoá |
| Báo Gen2 chưa đạt | Sau khi đăng nhập, tác vụ tự kích hoạt chạy không thành công | Chưa đăng ký tác vụ → GUI ② bấm [Kích hoạt Gen2 & Cài đặt tự khởi động]; Đã có tác vụ nhưng phiên này chưa đạt → Bấm [Kích hoạt Gen2 ngay] (Gen2AutoHard mặc định bật sẽ tự chạy Stage2 rollback) |
| Khi rảnh rỗi GPU-Z / Chẩn đoán báo Gen1 | **Cơ chế tiết kiệm điện hạ xung, KHÔNG PHẢI lỗi** (có tải sẽ tự lên Gen2) | Không cần can thiệp; xem công cụ chẩn đoán báo "Mục tiêu TLS=Gen2" tức là đã thành công; nếu muốn cố định Gen2 xem mục 2.4 để tắt ASPM |
| 4 thanh ghi PL0 đều ghi thành công nhưng TLS vẫn ở Gen1 | Driver/GSP can thiệp ghi đè chính sách liên kết trong vòng mili-giây (**không phải lỗi khoá firmware BIOS, không lấy số lô .06/.04 làm thước đo**; **tuyệt đối KHÔNG flash VBIOS**) | Tác vụ đăng nhập (nếu đã tạo) mặc định sẽ tự động chạy Stage2 rollback (Gen2AutoHard); **nếu chưa tạo tác vụ thì hệ thống không tự chạy, hãy đăng ký tác vụ trước**; sau đó vào GUI ② bấm [Kích hoạt Gen2 & Cài đặt tự khởi động] hoặc [Kích hoạt Gen2 ngay] để thử lại; nếu vẫn không được hãy gửi file chẩn đoán và nhật ký cho tác giả |
| Phần mềm diệt virus chặn file driver | Một số ít phần mềm diệt virus nhận nhầm driver kích hoạt Gen2 | Khi cài đặt đã **tự động thêm loại trừ vào Windows Defender** (chỉ loại trừ file của dự án, không tắt bảo vệ máy); nếu vẫn bị xoá, hãy khôi phục file trong Security Center rồi chạy lại bộ cài |
| Hệ thống đa card / Cắm khe phụ bị lỗi | Lệch pha thời gian tín hiệu bus sau chip cầu | Chuyển card 40HX sang cắm ở khe PCIe x16 đầu tiên, hoặc tạm thời tháo bớt card phụ để kiểm tra |
| GPU-Z báo PCIe x8 | Không liên quan đến việc mở khoá, do phân chia số lane giữa khe M.2 và PCIe trên bo mạch chủ | Tra cứu sách hướng dẫn bo mạch chủ về chia sẻ băng thông khe M.2/PCIe, chuyển card sang khe x16 độc lập |
| Phần mềm diệt virus bên thứ ba (Huorong, 360...) xoá driver | Phần mềm bên thứ ba không đọc danh sách loại trừ của Defender | Thêm thủ công 4 đường dẫn driver vào danh sách tin cậy: file ThrottleStop.sys, WinRing0x64.sys trong thư mục `%SystemRoot%\System32\drivers\` và cùng 2 file .sys đó trong thư mục `%ProgramData%\40HXUnlock\drivers\` |
| Driver bị cách ly liên tục (dấu hiệu sắp bị Code 43) | Phần mềm diệt virus liên tục xoá file .sys | Nếu dùng Defender: GUI vùng ① tick "Tắt bảo vệ thời gian thực của Defender" rồi bấm cài đặt; nếu dùng phần mềm bên thứ 3: thêm vào danh sách tin cậy theo dòng trên |
| Sau khi mở khoá bị khởi động lại liên tục / Không vào được Windows | Lỗi cổng vào khởi động hoặc cơ sở dữ liệu BCD bị lỗi | Tắt nguồn rút điện rồi bật lại; nếu vẫn lỗi, làm theo mục 8 dùng USB cài Windows (xoá `\EFI\40HX` + chạy lệnh `bootrec /rebuildbcd`) |
| BIOS không hiện mục boot '40HX Unlock' | Một số bo mạch chủ (Maxsun...) không hiện danh sách / bỏ qua việc ghi BCD | Dùng Windows PE (firPE) hoặc DiskGenius để thêm mục boot thủ công; hoặc chọn ổ đĩa hệ thống UEFI làm ưu tiên 1 (sẽ tự động chạy qua file cứu hộ `\EFI\Boot\bootx64.efi`, xem mục 8) |

### 5.3 Vị trí các file nhật ký

Chạy `40HXCheck.exe` một lần sẽ tự động thu thập đầy đủ các file nhật ký vào thư mục **`%LOCALAPPDATA%\40HXUnlock\logs\`** (installer.log / 40hx_log.txt / diagnose.txt), bạn chỉ cần nén cả thư mục này gửi đi nhờ hỗ trợ. Vị trí các file gốc:
- `%TEMP%\40HX_installer.log` (Nhật ký bộ cài)
- `%TEMP%\40HX_uninstaller.log` (Nhật ký gỡ cài đặt)
- File `40hx_log.txt` tại thư mục gốc phân vùng ESP (Nhật ký thực thi của firmware mở khoá EFI)

---

## 6. Tương thích Riot Games (Valorant / Vanguard) & Secure Boot

Khi chơi các tựa game của Riot Games (Valorant, League of Legends, TFT) được bảo vệ bởi **Riot Vanguard** trên Windows 11:

### 6.1 Sự khác biệt kiến trúc giữa CMP 40HX và CMP 30HX

| Đặc tính | CMP 40HX (TU106) | CMP 30HX (TU116) |
|---|---|---|
| **Cơ chế mở khoá Tensor Core** | Bắt buộc nạp payload EFI (`40HXUNLK.EFI`) trước khi boot Windows để ghi `SS0 = 0x88888888` | Không có Tensor Core trong silicon TU116 (không dùng EFI bootloader) |
| **Cơ chế mở khoá PCIe Gen2** | Ghi thanh ghi bóng MMIO + Retrain link | Ghi thanh ghi bóng BAR0 MMIO + DEVCTL MRRS 512B (`0x2000`) |
| **Yêu cầu Secure Boot trên Win 11** | Bắt buộc bật Secure Boot (`SecureBoot == 1`) cho Vanguard, nhưng EFI loader chưa có chữ ký Microsoft | **Secure Boot BẬT (Enabled) bình thường**, không cần can thiệp BIOS |
| **Giải pháp tương thích Riot Vanguard** | **Chạy `UnlockRiotGame.exe`**: Ký số Authenticode cho `40HXUNLK.EFI` và nạp chứng chỉ `CMP40HX_Key.cer` vào BIOS `db` (Custom Mode) | Hệ thống tự nhiên tương thích 100% khi bật Secure Boot và nạp driver transient |

### 6.2 Hướng dẫn chi tiết cho CMP 40HX trên Windows 11 (Qua `UnlockRiotGame.exe`)

1. **Khởi chạy công cụ**:
   - Nhấp đúp vào **`UnlockRiotGame.exe`** (yêu cầu quyền Administrator).
   - Công cụ sẽ tự động phát hiện GPU (`TU106` hay `TU116`), phiên bản Windows và trạng thái Secure Boot.
2. **Ký số tự động (Automated Authenticode Signing)**:
   - Nhấn nút **[1. Bắt đầu Ký số EFI & Xuất Key BIOS]**.
   - Công cụ sẽ tạo chứng chỉ số X.509 tự ký (`CMP40HX_Key.cer`) với thuật toán SHA256.
   - Tự động ký số bảo mật cho tệp `40HXUNLK.EFI` (cả trong thư mục phát hành lẫn trong phân vùng ESP `\EFI\40HX\`).
   - Tự động sao chép chứng chỉ `CMP40HX_Key.cer` vào 3 vị trí:
     - `C:\CMP40HX_Key.cer` (gốc ổ C để BIOS dễ duyệt tệp)
     - Desktop của người dùng
     - Phân vùng ESP (`\EFI\40HX\CMP40HX_Key.cer`)
3. **Quy trình nạp Key vào BIOS `db` (Quan trọng)**:
   - Nhấn nút **[2. Xem hướng dẫn nạp Key vào BIOS]** và **chụp lại ảnh màn hình bằng điện thoại** trước khi khởi động lại:
     + Khởi động lại máy tính, bấm `Del` hoặc `F2` để vào BIOS.
     + Chuyển chế độ Secure Boot Mode từ **Standard** sang **Custom**.
     + Vào mục **Key Management** (hoặc Secure Boot Policy).
     + Chọn mục **Authorized Signatures (db)** -> chọn **Append Key** (hoặc Enroll Signature / Add Signature).
     + Duyệt đến ổ đĩa C: hoặc phân vùng ESP, chọn file **`CMP40HX_Key.cer`**.
     + Lưu thay đổi và khởi động lại (`F10` -> Save & Exit).
4. **Kết quả đạt được**:
   - Hệ điều hành Windows 11 báo `SecureBoot == 1` và TPM 2.0 hợp lệ.
   - Riot Vanguard (`vgk.sys`) xác nhận hệ thống an toàn và cho phép chơi Valorant, LoL mượt mà.
   - Đồng thời `40HXUNLK.EFI` được firmware tin cậy và thực thi, mở khoá trọn vẹn Tensor Core (`SS0=0x88888888`, ~50 TFLOPS) và PCIe Gen2!

### 6.3 Hướng dẫn cho CMP 30HX (Không cần nạp EFI)

- CMP 30HX hoàn toàn **không sử dụng firmware EFI**, quá trình mở khoá Gen2 diễn ra hoàn toàn trong không gian ring-0 của Windows sau khi hệ điều hành khởi động.
- Do đó, bạn có thể **bật Secure Boot bình thường trong BIOS**. Riot Vanguard sẽ hoạt động trơn tru mà không cần chuyển BIOS sang Custom Mode hay nạp thêm key.

---

## 7. Lưu ý khi sử dụng hàng ngày

1. **Việc mở khoá chỉ có hiệu lực tạm thời cho mỗi phiên khởi động, không ghi đè vĩnh viễn vào phần cứng.** Mỗi lần bật máy, firmware mở khoá sẽ nạp lại mã can thiệp. Nếu một lần bật máy nào đó thấy bị mất mở khoá? **Chỉ cần tắt hẳn máy rồi bật lại** là sẽ bình thường.
2. **Khuyến nghị giữ phiên bản driver NVIDIA ổn định.** Nếu sau này cập nhật driver NVIDIA mới mà card bị khoá lại thì đây là hiện tượng bình thường, chỉ cần chạy lại bộ cài Installer một lần là xong.
3. **Bộ công cụ này chỉ tác động lên duy nhất card CMP 40HX**, hoàn toàn không ảnh hưởng đến các card đồ hoạ khác trên cùng máy tính.
4. Chơi game, chạy mô hình AI hay render video đều hoạt động bình thường; mặc định chế độ "Dùng xong gỡ ngay" sẽ không để lại bất kỳ driver chạy ngầm nào trong hệ thống (trừ khi bạn chọn chế độ ③ Dịch vụ thường trú).
5. **Tương thích với phần mềm diệt virus**: Khi cài đặt, chương trình sẽ tự động thêm driver vào danh sách loại trừ của Windows Defender (chỉ loại trừ file của dự án, không tắt bảo vệ của hệ thống). **Phần mềm diệt virus bên thứ ba không đọc danh sách của Defender**, nếu bị chặn, vui lòng thêm thủ công 4 đường dẫn driver được liệt kê ở mục 5.2 vào danh sách trắng.
6. **Tắt bảo vệ thời gian thực của Defender (Tuỳ chọn)**: Tại GUI vùng ① có tuỳ chọn "Tắt bảo vệ thời gian thực của Defender", nếu tính năng này đang bật thì bộ cài sẽ tự động tick chọn, bạn bấm [Cài đặt các thành phần đã chọn] thì mới thực thi. Sau khi tắt sẽ duy trì liên tục, muốn bật lại: Mở PowerShell với quyền Administrator chạy: `Set-MpPreference -DisableRealtimeMonitoring $False`. Nếu máy dùng bản Windows rút gọn không có mô-đun Defender hoặc bị tính năng "Tamper Protection" chặn, giao diện và nhật ký sẽ thông báo rõ ràng để bạn nắm được.

---

## 8. Gỡ cài đặt và hoàn nguyên

- Tự động: Nhấp đúp chuột vào `40HXUninstaller.exe` (Dùng chung bộ lõi gỡ cài đặt sạch sẽ với `40HXInstaller.exe -uninstall`).
- Quá trình này sẽ xoá sạch: Các Scheduled Task (kể cả tác vụ thử lại khi lỗi) cùng khoá tự khởi động / Mục boot firmware / Firmware mở khoá trong phân vùng ESP (khôi phục lại file bootx64.efi gốc) / Dịch vụ và file driver / Cấu hình GSP / Khoá chính sách trong Registry / Thư mục đệm ProgramData / Danh sách loại trừ Defender.
- Các thiết lập nguồn điện (Fast Startup / ASPM / High Performance Plan) **sẽ được giữ nguyên**, nếu muốn khôi phục thủ công xem tại mục 2.4.
- Sau khi khởi động lại máy, card đồ hoạ sẽ trở về trạng thái xuất xưởng ban đầu. Nếu trong BIOS vẫn còn sót tên mục boot, bạn có thể vào BIOS xoá thủ công.

---

## 9. Lỗi boot EFI? Hướng dẫn cứu hộ khẩn cấp

**Nếu sau khi cài đặt hoặc sau một lần khởi động nào đó gặp phải các trường hợp dưới đây, đừng hoang mang, hệ điều hành của bạn không hề bị hỏng, chỉ có cổng vào khởi động bị kẹt:**

- Màn hình xanh báo mã lỗi `0xc000000f` / `0xc000007b` / `0xc0000098`
- Thông báo không tìm thấy tệp `\EFI\40HX\40HXUNLK.EFI`
- Bị treo ở màn hình chữ 40HX Unlock không vào được Windows

**Hãy mở ngay tệp 《EFI应急修复指南.md》 trong thư mục này**, làm theo hướng dẫn tại Mục 1:  
Cắm một chiếc **USB cài đặt Windows** vào máy → Chọn Repair your computer → Command Prompt →  
Gắn ổ đĩa cho phân vùng EFI → Xoá thư mục `\EFI\40HX` → Chạy `bootrec /rebuildbcd` để tạo lại cơ sở dữ liệu khởi động → Khởi động lại là máy sẽ vào Windows bình thường.

> Tóm tắt quy trình cứu hộ trong một câu: **Xoá mục boot 40HX bị kẹt cùng các file tạm, sau đó để Windows tạo lại cơ sở dữ liệu boot BCD của chính nó**.  
> Toàn bộ các câu lệnh chi tiết, phương án dự phòng bằng USB Linux hay xử lý trong BIOS đều có sẵn trong cẩm nang cứu hộ, bạn chỉ việc gõ theo.  
> Sau khi cứu hộ thành công, nếu bạn vẫn muốn mở khoá card thì có thể dùng phương án "USB boot mở khoá thủ công" ở mục 2.2, hoạt động độc lập không phụ thuộc vào mục boot của bo mạch chủ, bảo đảm an toàn tuyệt đối.

---

## 10. Lịch sử phiên bản

- **v3.0.0 (Deep Module Architecture & Vanguard Clean)**: Tái cấu trúc toàn diện kiến trúc phần mềm theo nguyên lý Deep Module với 2 ranh giới Seams rõ ràng:
  - **`LinkNegotiator`**: Đóng gói toàn bộ máy trạng thái huấn luyện PCIe, kẹp cứng giới hạn phần cứng eFuse Gen2 trên TU116, tối ưu DEVCTL MRRS 512B (`0x2000`) nâng băng thông thực tế lên ~6.4 GB/s, chuỗi nạp shadow register MMIO (`PRIV_MISC_1`, `XVE_OVR`, `LINK_CONFIG_0`, `PL_LINK_RATE`, `CYA_0`) và phục hồi an toàn PnP (thời gian xả tụ 2.0s).
  - **`ComputeInspector`**: Tích hợp chốt an toàn phần cứng kiểm tra `BOOT_0` (`0x16xxxxxx` họ TU106) chống crash hệ thống và giải mã định kiểu thanh ghi kép `SS0` (`0x409664` == `0x88888888`) và `SS1` (`0x40966C`).
  - **`HardwareBus` (Seam 1)**: Tách biệt hoàn toàn tầng driver I/O (`WinRing0`, `ThrottleStop`) khỏi logic nghiệp vụ, hỗ trợ `MockHardwareBus` cho kiểm thử đơn vị.
  - **`StatusContract` (Seam 2)**: Chuẩn hóa hợp đồng trạng thái có cấu trúc (`STATUS_CODE=GEN2_SUCCESS`, v.v.) giữa Go engine và các script Batch.
  - **Tương thích tuyệt đối Anti-Cheat (Riot Vanguard / EAC / BattlEye)**: Cơ chế nạp driver tạm thời (Dùng-Xong-Rút) dọn dẹp sạch sẽ dịch vụ và file `.sys` ngay sau khi ghi thanh ghi, không cần bật Test Signing, bảo đảm an toàn khi chơi game Valorant / LoL.
  - **Bộ kiểm thử tự động toàn diện**: 13 Go unit tests và 10 Mock test suites (43 assertions) xác thực tự động mọi kịch bản và ranh giới an toàn.
  - Đồng thời kế thừa toàn bộ các cải tiến: Sửa lỗi bo mạch chủ AGESA / bus cao không tìm thấy card trong môi trường EFI; quét ban đầu mở rộng bus 0–16 và fallback CF8/CFC bus 0–255; sửa hiển thị SS1 sang 0x40966C; tự động phân loại lỗi trong nhật ký.
- **v2.6.0**: Giao diện đồ hoạ GUI cửa sổ đơn hoàn toàn mới (Mặc định mở khi nhấp đúp; không chia tab với 3 khu vực: ① Cài đặt thành phần & Môi trường: Tự động đánh dấu mục thiếu, thực thi độc lập từng mục GSP / EFI Compute+Mục boot / Driver Gen2+Loại trừ Defender / Tự khởi động khi đăng nhập / 3 mục nguồn điện(Tắt Fast Startup·Tắt ASPM·Power Plan hiệu năng cao) / Tuỳ chọn "Tắt bảo vệ thời gian thực Defender", sau khi chạy tự động quét lại; ② Chính sách Gen2: 3 chế độ chạy driver (Dùng xong gỡ ngay·Thất bại tự thử lại·Dịch vụ thường trú) + Tự động Stage2 rollback + Thử lại khi lỗi, **mặc định 3 lần / 3 phút**; ③ Nhật ký thời gian thực); Tự động quét kiểm tra môi trường ngay khi mở; bổ sung nhận diện phần mềm diệt virus bên thứ ba, báo cáo trung thực trạng thái Defender (Danh sách loại trừ/Bảo vệ thời gian thực/Thiếu mô-đun), kiểm tra đa tầng "trạng thái driver" (Nguồn sao lưu ProgramData / 4 trạng thái file trong System32 bao gồm cả file 0 byte / Sửa lỗi dịch vụ bị DISABLED), không còn báo nhầm trạng thái "dùng xong gỡ ngay" là chưa cài driver; kiểm tra mục boot 3 trạng thái (Đứng đầu/Có tồn tại nhưng không đứng đầu/Chưa tạo); Khoá nút bấm chống click liên tục trong GUI; sửa lỗi xuống dòng nhật ký; tham số `-uninstall` nâng cấp thành gỡ cài đặt toàn bộ mức thành phần đồng bộ với 40HXUninstaller.exe; gỡ cài đặt dọn dẹp khoá Registry chính sách; giải mã UTF-8 khi báo lỗi loại trừ Defender và nhận diện lỗi "thiếu mô-đun"; kế thừa toàn bộ từ v2.5.1: Tự động Stage2 rollback cho Gen2 (Link Disable + PnP, Gen2AutoHard mặc định bật), đọc-sửa-ghi LNKCTL2, chu trình retrain xen kẽ root/GPU tối đa 4 vòng, đánh giá theo tốc độ mục tiêu TLS, lỗi EFI không làm dừng cài đặt + hướng dẫn mbr2gpt, tạo tác vụ kiểm tra mã thoát 0, chẩn đoán dựa trên file XML tác vụ.
- **v2.5.1**: Bản sửa lỗi độ tin cậy từ phản hồi cộng đồng: ① Lỗi triển khai EFI (Legacy+MBR không có phân vùng EFI...) không làm gián đoạn cài đặt, tác vụ tự khởi động Gen2 vẫn được tạo bình thường, máy chạy Legacy tự động hiển thị hướng dẫn chuyển đổi `mbr2gpt` không mất dữ liệu; ② Tác vụ lịch trình Gen2 sau khi tạo có bước kiểm tra lại + tự động thử lại, tham số `-task` nếu lỗi sẽ trả về mã thoát khác 0; ③ Sửa lỗi chẩn đoán trên Windows tiếng Trung nhận nhầm tác vụ đã tạo thành "chưa tạo"; ④ Cải tiến lõi Gen2: Đọc-sửa-ghi LNKCTL2, retrain xen kẽ root/GPU tối đa 4 vòng (khắc phục cho mainboard không chính hãng/đa card), đánh giá thành công theo tốc độ mục tiêu TLS (tránh báo lỗi nhầm khi card hạ Gen1 lúc rảnh rỗi); ⑤ Tự động tắt Khởi động nhanh và PCIe ASPM, cung cấp lệnh khôi phục; ⑥ manual_install.bat nhấp đúp tự xin quyền Admin, lỗi EFI không dừng script, sửa lỗi đường dẫn ProgramData trong manual_uninstall.bat; ⑦ Kiểm tra tàn dư gỡ cài đặt bổ sung kiểm tra tên tác vụ hiện tại; ⑧ Bổ sung prompt AI hỗ trợ cài đặt ở đầu file README.
- **v2.5**: Mở khoá Gen2 không cần bật chế độ "Test Signing" (Hệ thống luôn sạch sẽ, thân thiện với game và phần mềm chống gian lận); Driver Gen2 chỉ nạp trong tích tắc khi mở khoá rồi tự động giải phóng; Công cụ chẩn đoán ưu tiên hiển thị ngay 2 kết quả "Năng lực tính toán + Gen2"; Viết lại toàn bộ bộ cài đặt.

---

*Tài liệu chỉ phục vụ mục đích nghiên cứu phần cứng và học tập cá nhân, vui lòng tuân thủ pháp luật địa phương và các điều khoản của nhà sản xuất phần cứng.*
