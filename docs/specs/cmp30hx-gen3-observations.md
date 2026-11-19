# Quan sát PCIe Gen3 trên CMP 30HX (TU116)

Nhánh `research-gen3` (commit `83c64a7`) ghi nhận hai quan sát trên CMP 30HX. Giá trị đọc lại của `LNKCAP2` vẫn chỉ quảng bá Gen1 và Gen2. Giá trị tốc độ đích đọc lại từ `LNKCTL2` vẫn là Gen2 sau khi thử đặt Gen3.

Các quan sát này cho thấy các thanh ghi đã kiểm tra không giữ cấu hình Gen3 trong điều kiện thử nghiệm. Chúng chưa đủ để xác định nguyên nhân là eFuse, strap hay cấu hình khởi tạo. Kết quả chỉ áp dụng cho mẫu TU116 được thử nghiệm. CMP 40HX (TU106) cần bằng chứng riêng.

Mục tiêu hỗ trợ của CMP 30HX vẫn là **PCIe Gen2 x16**. Khi đánh giá log, phân biệt tốc độ tối đa được quảng bá, tốc độ đích và tốc độ liên kết thực tế. Gen1 lúc nhàn rỗi rồi trở lại Gen2 dưới tải phù hợp với hiện tượng tiết kiệm điện, nhưng không cho thấy card hỗ trợ Gen3.

Xem [đặc tả PCIe Gen2](pcie-gen2-and-tensor-unlock-spec.md) để đối chiếu giới hạn của profile và [bảng profile GPU](../../windows-v3.0/tools/40hxcore/const.go) để kiểm tra cấu hình hiện hành.
