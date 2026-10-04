package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	hxcore "40hxcore"
)

// CLIOptions chứa các cờ dòng lệnh
type CLIOptions struct {
	Silent      bool
	Optimize    bool
	AutoSign    bool
	CheckSig    bool
	Gen2Warning bool
	JSONStatus  bool
}

// ParseFlags phân tích cờ dòng lệnh từ mảng chuỗi
func ParseFlags(args []string) (CLIOptions, error) {
	fs := flag.NewFlagSet("UnlockRiotGame", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CLIOptions
	fs.BoolVar(&opts.Silent, "silent", false, "Chạy chế độ im lặng không hiển thị GUI")
	fs.BoolVar(&opts.Optimize, "optimize", false, "Tối ưu hóa Registry và dọn dẹp driver mở khóa")
	fs.BoolVar(&opts.AutoSign, "auto-sign", false, "Tự động tạo cert và ký Authenticode cho 40HXUNLK.EFI")
	fs.BoolVar(&opts.CheckSig, "check-sig", false, "Kiểm tra chữ ký số driver user-mode nvwgf2umx.dll")
	fs.BoolVar(&opts.Gen2Warning, "gen2-warning", false, "Hiển thị thông tin kiểm tra PCIe Gen 2.0 ASPM")
	fs.BoolVar(&opts.JSONStatus, "json-status", false, "Xuất thông tin trạng thái Riot dưới dạng JSON")

	err := fs.Parse(args)
	return opts, err
}

// RunCLI điều phối thực thi các tác vụ theo cờ dòng lệnh
func RunCLI(opts CLIOptions, log io.Writer, uefiMgr UEFIManager) error {
	if log == nil {
		log = os.Stdout
	}
	if uefiMgr == nil {
		uefiMgr = &DefaultUEFIManager{}
	}

	prof, _ := hxcore.FindGPUWithProfile()
	sbOn := hxcore.SecureBootOn()
	isWin11 := isWindows11()
	status := EvaluateRiotStatus(prof.DeviceID, prof.Name, sbOn, isWin11)
	is40HX := (status.Model == GPUModelCMP40HX)
	is30HX := (status.Model == GPUModelCMP30HX)

	if opts.JSONStatus {
		return json.NewEncoder(log).Encode(status)
	}

	if opts.AutoSign {
		fmt.Fprintln(log, "[*] Bắt đầu tự động tạo chứng chỉ và ký UEFI 40HXUNLK.EFI...")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		res, err := uefiMgr.PrepareAndSignEFI(ctx, log)
		if err != nil {
			fmt.Fprintf(log, "[!] Lỗi ký EFI: %v\n", err)
			return err
		}
		fmt.Fprintf(log, "[V] Đã tạo chứng chỉ và ký EFI thành công:\n  - %s\n  - %s\n", res.CertPathC, res.CertPathDesktop)
	}

	if opts.Optimize {
		runOptimization(log, is40HX, is30HX, sbOn, isWin11)
	}

	if opts.CheckSig {
		fmt.Fprintln(log, "[*] Kiểm tra chữ ký số driver nvwgf2umx.dll (User-Mode Driver)...")
		valid, info, err := CheckDriverSignature("")
		if err != nil {
			fmt.Fprintf(log, "[!] Cảnh báo kiểm tra chữ ký: %v\n", err)
		} else {
			if valid {
				fmt.Fprintf(log, "[V] Chữ ký số driver HỢP LỆ: %s\n", info)
			} else {
				fmt.Fprintf(log, "[!] CẢNH BÁO CHỮ KÝ: Driver nvwgf2umx.dll không hợp lệ hoặc bị sửa đổi!\n    %s\n", info)
				fmt.Fprintln(log, "    => Riot Vanguard có thể chặn DLL này khi khởi động LoL / Valorant.")
				fmt.Fprintln(log, "    => Khắc phục: Sử dụng driver Studio hoặc bản mod driver có kèm chứng chỉ được tin cậy.")
			}
		}
	}

	if opts.Gen2Warning {
		fmt.Fprintln(log, GetGen2AspmExplanation())
	}

	return nil
}

// GetGen2AspmExplanation giải thích trạng thái PCIe 1.1 khi card nghỉ
func GetGen2AspmExplanation() string {
	return `=== GIẢI THÍCH TRẠNG THÁI PCIE GEN 2.0 / ASPM TIẾT KIỆM ĐIỆN ===
- Hiện tượng: GPU-Z hiển thị "PCIe x16 1.1 @ x16 1.1" khi máy tính đang ở màn hình Desktop (IDLE).
- Nguyên nhân: Tính năng Active State Power Management (ASPM / Link State Power Management) của Windows
  tự động hạ tốc độ link xuống Gen 1 (2.5 GT/s) để giảm nhiệt độ và tiết kiệm điện năng khi GPU không tải.
- Cách kiểm chứng tốc độ thực:
  1. Mở GPU-Z cạnh mục "Bus Interface".
  2. Bấm vào biểu tượng dấu chấm hỏi [?] kế bên để mở cửa sổ "Render Test".
  3. Bấm "Start Render Test" để tạo tải đồ họa 3D.
  4. Quan sát: Bus Interface sẽ lập tức nhảy lên "PCIe x16 2.0 @ x16 2.0 [5.0 GT/s]"!`
}
