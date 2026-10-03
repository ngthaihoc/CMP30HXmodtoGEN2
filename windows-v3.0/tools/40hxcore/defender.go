package hxcore

// Defender 驱动排除与实时防护管理。
//
// 背景: WinRing0x64.sys / ThrottleStop.sys 这类底层驱动可能被安全软件当作
// HackTool/漏洞驱动隔离(常见表现: 文件变 0 字节占位) → Gen2 自启失败。
// 本模块提供:
//   1. 加白(默认, 精确到本项目文件, 不关闭任何系统防护; 卸载时同名移除)
//   2. 可选关闭实时防护(高风险, 仅当驱动反复被隔离时用, 默认不勾)
//   3. 状态查询(排除列表 / 实时防护是否开启) — 供扫描与界面提示
//
// 编码: PowerShell 输出经管道回读是系统 ANSI(GBK), 直接按 UTF-8 解读会乱码
// (曾把报错显示成一串乱码)。统一在命令前设 [Console]::OutputEncoding=UTF8。
//
// 兼容: 第三方杀软接管或精简系统会把 Defender 管理模块整个拿掉,
// 此时 Add/Remove/Get-MpPreference 报"不是 cmdlet" — mpErr 识别后给中文提示。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// psUtf8: 让 powershell 以 UTF-8 向管道输出(避免中文报错在日志里乱码)
const psUtf8 = "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8;"

// Gen2DrvFiles: v2.5 BYOVD 用到的两个驱动文件名。
func Gen2DrvFiles() []string {
	return []string{"ThrottleStop.sys", "WinRing0x64.sys"}
}

// ExclusionPaths: 需要加白的具体路径(精确到文件/目录)。
//   1-2. System32\drivers 下的两个驱动文件
//   3.   %ProgramData%\40HXUnlock\drivers 备份源
//   4.   当前 exe 所在目录(随入口变化, 不作查询判定项)
func ExclusionPaths() []string {
	var ps []string
	sys := os.Getenv("SystemRoot")
	if sys == "" {
		sys = `C:\Windows`
	}
	for _, f := range Gen2DrvFiles() {
		ps = append(ps, filepath.Join(sys, "System32", "drivers", f))
	}
	ps = append(ps, PdDrvDir())
	if exe, err := os.Executable(); err == nil {
		ps = append(ps, filepath.Dir(exe))
	}
	return ps
}

// psArray: 拼 PowerShell 数组字面量 @('a','b'), 单引号包裹路径。
func psArray(ps []string) string {
	q := make([]string, 0, len(ps))
	for _, p := range ps {
		q = append(q, "'"+strings.ReplaceAll(p, "'", "''")+"'")
	}
	return "@(" + strings.Join(q, ",") + ")"
}

// ErrMpUnavailable: Defender 管理模块缺失(第三方杀软接管/精简系统把模块拿掉)。
// 供调用方 errors.Is 判断后显示简短提示, 避免把整段报错嵌套进界面文案。
var ErrMpUnavailable = errors.New("Mô-đun quản lý Defender không khả dụng")

// mpErr: Chuyển đổi lỗi liên quan đến Defender thành thông điệp rõ ràng
func mpErr(action, out string, err error) error {
	low := strings.ToLower(out)
	switch {
	case strings.Contains(low, "not recognized"),
		strings.Contains(low, "commandnotfoundexception"),
		strings.Contains(out, "不是内部"),
		strings.Contains(out, "无法将"):
		return fmt.Errorf("%w: Máy tính chưa cài mô-đun quản lý Defender (%s)", ErrMpUnavailable, action)
	}
	msg := strings.TrimSpace(out)
	if len(msg) > 200 {
		msg = msg[:200]
	}
	if msg != "" {
		return fmt.Errorf("%s thất bại: %v %s", action, err, msg)
	}
	return fmt.Errorf("%s thất bại: %v", action, err)
}

func runMp(cmd string) (string, error) {
	return RunOut("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psUtf8+cmd)
}

// AddDefenderExclusions: Thêm file driver / thư mục sao lưu vào danh sách loại trừ Defender.
func AddDefenderExclusions() error {
	ps := ExclusionPaths()
	if len(ps) == 0 {
		return fmt.Errorf("Không có đường dẫn loại trừ")
	}
	out, err := runMp("Add-MpPreference -ExclusionPath " + psArray(ps))
	if err != nil {
		return mpErr("Thêm loại trừ Defender", out, err)
	}
	return nil
}

// RemoveDefenderExclusions: Khi gỡ cài đặt — dọn dẹp các mục loại trừ đã thêm.
func RemoveDefenderExclusions() error {
	ps := ExclusionPaths()
	if len(ps) == 0 {
		return nil
	}
	out, err := runMp("Remove-MpPreference -ExclusionPath " + psArray(ps))
	if err != nil {
		return mpErr("Gỡ bỏ loại trừ Defender", out, err)
	}
	return nil
}

// DefenderExclusionsPresent: Kiểm tra danh sách loại trừ đã chứa file driver hay chưa.
func DefenderExclusionsPresent() (bool, error) {
	out, err := runMp("@((Get-MpPreference).ExclusionPath) | ConvertTo-Json -Compress")
	if err != nil {
		return false, mpErr("Truy vấn danh sách loại trừ Defender", out, err)
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "null" {
		return false, nil
	}
	var paths []string
	if strings.HasPrefix(out, "[") {
		if err := json.Unmarshal([]byte(out), &paths); err != nil {
			return false, fmt.Errorf("Phân tích danh sách loại trừ Defender thất bại: %v", err)
		}
	} else {
		var s string
		if err := json.Unmarshal([]byte(out), &s); err != nil {
			return false, fmt.Errorf("Phân tích danh sách loại trừ Defender thất bại: %v", err)
		}
		paths = []string{s}
	}
	var want []string
	sys := os.Getenv("SystemRoot")
	if sys == "" {
		sys = `C:\Windows`
	}
	for _, f := range Gen2DrvFiles() {
		want = append(want, filepath.Join(sys, "System32", "drivers", f))
	}
	want = append(want, PdDrvDir())
	trim := func(p string) string { return strings.TrimRight(strings.TrimSpace(p), `\`) }
	hit := 0
	for _, w := range want {
		for _, p := range paths {
			if strings.EqualFold(trim(p), trim(w)) {
				hit++
				break
			}
		}
	}
	return hit == len(want), nil
}

// DefenderRealtimeProtectionOn: Kiểm tra bảo vệ thời gian thực của Defender.
func DefenderRealtimeProtectionOn() (bool, error) {
	out, err := runMp("(Get-MpComputerStatus).RealTimeProtectionEnabled | ConvertTo-Json -Compress")
	if err != nil {
		return false, mpErr("Truy vấn bảo vệ thời gian thực Defender", out, err)
	}
	switch strings.ToLower(strings.TrimSpace(out)) {
	case "true", "1":
		return true, nil
	case "false", "0", "":
		return false, nil
	}
	return false, fmt.Errorf("Truy vấn bảo vệ thời gian thực Defender trả về bất thường: %s", strings.TrimSpace(out))
}

// SetDefenderRealtimeProtection: Tắt hoặc bật lại bảo vệ thời gian thực Defender.
func SetDefenderRealtimeProtection(on bool) error {
	v := "False"
	act := "Tắt"
	if on {
		v = "True"
		act = "Bật lại"
	}
	out, err := runMp("Set-MpPreference -DisableRealtimeMonitoring $" + v)
	if err != nil {
		return mpErr(act+" bảo vệ thời gian thực Defender (nếu Windows Security đang bật 'Tamper Protection' thao tác sẽ bị từ chối, hãy tắt nó trước)", out, err)
	}
	return nil
}
