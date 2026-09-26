# Hướng Dẫn Cứu Hộ Khẩn Cấp 40HX Unlock (Khôi Phục Khi Lỗi Boot EFI)

> **Tài liệu này CHỈ dùng khi "sau khi cài đặt công cụ mở khoá bị lỗi khởi động EFI"**, khi máy hoạt động bình thường xin đừng can thiệp.
> Phạm vi áp dụng: Khởi động máy hiện màn hình xanh báo lỗi `0xc000000f` / `0xc000007b` / `0xc0000098`,
> hoặc báo không tìm thấy `\EFI\40HX\40HXUNLK.EFI`, hoặc bị treo ở màn hình 40HX Unlock không vào được Windows.

---

## 0. Nguyên Nhân & Hướng Xử Lý

Công cụ mở khoá sẽ ghi firmware mở khoá vào **phân vùng hệ thống EFI (ESP)** và đăng ký một mục khởi động firmware.
Trong một số rất ít trường hợp (file firmware không trọn vẹn, timing mainboard, mất điện đột ngột), chuỗi khởi động có thể bị gián đoạn:

```
Khởi động máy → Mainboard theo mục boot NVRAM tìm \EFI\40HX\40HXUNLK.EFI
             → File bị thiếu / hỏng → Windows Boot Manager báo lỗi 0xc000000f
```

**Nguyên tắc cốt lõi: Hệ điều hành Windows không hề bị hỏng, chỉ có "cổng vào khởi động" bị nghẽn.**
Cách khắc phục = ① Xoá mục boot 40HX bị kẹt cùng các file tạm → ② Khôi phục / Tạo lại cơ sở dữ liệu boot BCD của chính Windows.

**Bạn cần chuẩn bị một USB cài đặt Windows** (tạo bằng công cụ Media Creation Tool chính thức từ Microsoft là tốt nhất),
hoặc một USB khởi động Linux như Ubuntu (chỉ dùng để xoá file, không thể sửa BCD).

---

## 1. Phương Án Tối Ưu: Dùng USB Cài Đặt Windows (Khuyến Nghị, 1 Quy Trình Hoàn Chỉnh)

### 1.1 Khởi động vào môi trường phục hồi Windows (Windows Recovery Environment)

1. Cắm **USB cài đặt Windows**, bật máy và chọn boot từ USB (chọn mục có tiền tố `UEFI:`).
2. Khi thấy màn hình cài đặt Windows, bấm vào dòng **Repair your computer (Sửa chữa máy tính)** ở góc dưới bên trái → **Troubleshoot (Khắc phục sự cố)** → **Advanced options (Tuỳ chọn nâng cao)** → **Command Prompt (Dấu nhắc lệnh)**.

### 1.2 Gắn ký tự ổ đĩa cho phân vùng EFI

Trong cửa sổ Command Prompt, thực thi:

```bat
diskpart
list disk
select disk 0        ← Ổ cứng cài Windows của bạn, nếu có nhiều ổ hãy dùng list disk để xác nhận
list volume
```

Tìm phân vùng có **loại là "System" (ESP, định dạng FAT32, thường có dung lượng 100~500MB)**, ghi nhớ số Volume (ví dụ: Volume 2), sau đó:

```bat
select volume 2      ← Thay bằng số Volume thực tế bạn vừa nhìn thấy
assign letter=Z:     ← Gắn phân vùng EFI thành ổ Z:
exit
```

### 1.3 Xoá tệp tin tàn dư của 40HX Unlock (Quan trọng)

```bat
Z:
dir \EFI
```

Nếu thấy thư mục `\EFI\40HX`, tiến hành xoá:

```bat
rmdir /s /q Z:\EFI\40HX
```

Đồng thời dọn dẹp các file nhật ký và bản sao lưu tại thư mục gốc EFI (nếu có thì xoá, không có thì bỏ qua):

```bat
del Z:\40hx_log.txt
del Z:\40hx_vbios.bin
del Z:\EFI\Boot\bootx64.efi.40hx.bak
```

### 1.4 Tạo lại cơ sở dữ liệu khởi động Windows BCD (Quan trọng nhất)

```bat
cd /d Z:\EFI\Microsoft\Boot
ren BCD BCD.old              ← Đổi tên BCD cũ để sao lưu
bootrec /rebuildbcd
```

Làm theo hướng dẫn trên màn hình, nhập `Y` để thêm bản cài đặt Windows vừa quét được vào danh sách boot. Sau khi hoàn tất có thể chạy thêm các lệnh sửa bản ghi boot (để bảo đảm):

```bat
bootrec /fixmbr
bootrec /fixboot
```

> Nếu lệnh `bootrec /rebuildbcd` không quét thấy hệ điều hành, bạn có thể dùng lệnh thay thế sau (thay Z: bằng ổ ESP bạn đã gán):
> ```bat
> bcdboot C:\Windows /s Z: /f UEFI
> ```

### 1.5 Hoàn tất & Khởi động lại

```bat
exit
```

Khởi động lại máy tính (rút USB ra). Lúc này hệ thống sẽ vào thẳng Windows bình thường.

### 1.6 Dọn dẹp sau khi vào lại Windows

Nhấp chuột phải chọn Run as administrator tệp **`40HXUninstaller.exe`** trong bộ công cụ phát hành:
Công cụ sẽ tự động dọn sạch các Scheduled Task còn lại / mục boot firmware / driver và dịch vụ / cấu hình GSP, đưa hệ thống hoàn toàn về trạng thái sạch sẽ ban đầu.

> Kiểm tra lại bằng **EasyUEFI hoặc Boot Menu BIOS**: Xác nhận Windows Boot Manager đang đứng ở vị trí đầu tiên và không còn mục 40HX Unlock tàn dư.

---

## 2. Phương Án Dự Phòng: Dùng USB Ubuntu / Linux (Chỉ Xoá Được File, Không Sửa Được BCD)

> Chỉ áp dụng khi không có sẵn USB cài đặt Windows để chữa cháy. **Linux không sửa được BCD của Windows**,
> sau khi xoá file bạn vẫn cần dùng lệnh `bootrec`/`bcdboot` ở Phương án 1 để tạo lại boot (hoặc mượn USB Windows sau).

1. Khi khởi động từ USB Ubuntu, **bắt buộc chọn mục có tiền tố `UEFI:`** (nếu không sẽ không truy cập được biến NVRAM EFI).
2. Mở Terminal (Cửa sổ dòng lệnh):

```bash
# Tìm phân vùng EFI (thường vài trăm MB, định dạng FAT32)
sudo lsblk -f

# Gắn phân vùng (thay nvme0n1p2 bằng tên phân vùng EFI thực tế của bạn)
sudo mkdir -p /mnt/efi
sudo mount /dev/nvme0n1p2 /mnt/efi

# Xoá tàn dư 40HX (thư mục + file rác ở gốc + backup)
sudo rm -rf /mnt/efi/EFI/40HX
sudo rm -f  /mnt/efi/40hx_log.txt /mnt/efi/40hx_vbios.bin
sudo rm -f  /mnt/efi/EFI/Boot/bootx64.efi.40hx.bak

# Xác nhận không còn tệp tin liên quan
sudo find /mnt/efi -iname "*40hx*" -o -iname "*unlk*"
```

3. Dùng efibootmgr để xoá mục boot 40HX trong NVRAM:

```bash
sudo efibootmgr -v          # Tìm mã BootXXXX tương ứng với "40HX Unlock"
sudo efibootmgr -b 0001 -B  # Xoá mục đó (thay 0001 bằng số thực tế)
```

4. Chạy `sudo umount /mnt/efi` và khởi động lại máy.
5. **Nếu vẫn chưa vào được hệ thống** → Chuyển sang Phương án 1, dùng USB Windows để tạo lại BCD.

---

## 3. Xử Lý Trực Tiếp Trong BIOS

Truy cập BIOS (bật máy và nhấn liên tục phím Del hoặc F2):

1. **Menu Boot** → Tìm danh sách mục khởi động, chuyển mục `40HX Unlock` sang **Disabled** hoặc bấm Delete để xoá;
2. Xác nhận **Windows Boot Manager** được đặt ở vị trí khởi động số 1;
3. Nếu mục 40HX Unlock không xoá được trong menu boot, hãy thử **Reset CMOS bo mạch chủ**
   (tháo pin CMOS 30 giây hoặc chọn Load Optimized Defaults trong BIOS), thao tác này sẽ đặt lại danh sách mục boot NVRAM.

---

## 4. Dấu Hiệu Phục Hồi Thành Công

- Bật máy vào thẳng Windows, không xuất hiện menu trung gian hay màn hình xanh báo lỗi;
- Trong Boot Menu của BIOS không còn mục 40HX Unlock;
- Windows Event Log không còn ghi nhận lỗi thiếu file `\EFI\40HX\40HXUNLK.EFI`.

---

## 5. Cách Phòng Tránh Sự Cố Lặp Lại

1. **Tuyệt đối không tắt nguồn / ép khởi động lại khi firmware mở khoá đang chạy**: Mất điện giữa chừng khi ghi firmware vào ESP là nguyên nhân chính gây lỗi;
2. Trước khi cập nhật Windows lớn hoặc nâng cấp driver đồ hoạ, nên gỡ công cụ mở khoá trước (`40HXUninstaller.exe`), sau khi cập nhật xong mới cài đặt lại;
3. Nếu có điều kiện, định kỳ dùng DiskGenius / Macrium Reflect sao lưu phân vùng ESP (rất nhỏ, chỉ vài trăm KB);
4. Để tìm hiểu chi tiết các bước xử lý lỗi khác, xem mục 5 "Xử lý sự cố" và mục 8 trong tệp README.md cùng thư mục.

---

*Tài liệu dành cho nghiên cứu phần cứng và học tập cá nhân. Vui lòng tuân thủ các quy định pháp luật và điều khoản bảo hành của nhà sản xuất phần cứng.*
