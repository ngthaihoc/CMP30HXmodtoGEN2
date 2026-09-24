package hxcore

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type StatusCode string

const (
	StatusGen2Success StatusCode = "GEN2_SUCCESS"
	StatusGen1Stuck   StatusCode = "GEN1_STUCK"
	StatusDrvFail     StatusCode = "DRV_FAIL"
	StatusNoGPU       StatusCode = "NO_GPU"
	StatusUnknown     StatusCode = "UNKNOWN"
)

// StatusContract: Seam 2 - Hợp đồng dữ liệu trạng thái định kiểu giữa Go Engine và Batch/CLI
type StatusContract struct {
	StatusCode   StatusCode
	SpeedCurrent uint32
	WidthCurrent uint32
	TLSTarget    uint32
	ErrorCode    string
	Details      []string
}

// FormatStatus định dạng cấu trúc trạng thái thành tệp văn bản có Header định kiểu
func FormatStatus(c StatusContract) string {
	var sb strings.Builder
	// 1. Structured Machine-readable Header
	sb.WriteString(fmt.Sprintf("STATUS_CODE=%s\n", c.StatusCode))
	sb.WriteString(fmt.Sprintf("SPEED_CURRENT=%d\n", c.SpeedCurrent))
	sb.WriteString(fmt.Sprintf("WIDTH_CURRENT=%d\n", c.WidthCurrent))
	sb.WriteString(fmt.Sprintf("TLS_TARGET=%d\n", c.TLSTarget))
	errCode := c.ErrorCode
	if errCode == "" {
		errCode = "NONE"
	}
	sb.WriteString(fmt.Sprintf("ERROR_CODE=%s\n", errCode))

	// 2. Human-readable Body
	sb.WriteString("==== 40HX Gen2 Ket qua " + time.Now().Format("2006-01-02 15:04:05") + " ====\n")
	for _, d := range c.Details {
		sb.WriteString(d + "\n")
	}
	return sb.String()
}

// ParseStatus phân tích chuỗi trạng thái thành StatusContract (hỗ trợ cả header mới và fallback legacy)
func ParseStatus(content string) StatusContract {
	res := StatusContract{
		StatusCode: StatusUnknown,
		ErrorCode:  "NONE",
	}

	lines := strings.Split(content, "\n")
	hasHeader := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "STATUS_CODE=") {
			hasHeader = true
			res.StatusCode = StatusCode(strings.TrimPrefix(line, "STATUS_CODE="))
		} else if strings.HasPrefix(line, "SPEED_CURRENT=") {
			if v, err := strconv.ParseUint(strings.TrimPrefix(line, "SPEED_CURRENT="), 10, 32); err == nil {
				res.SpeedCurrent = uint32(v)
			}
		} else if strings.HasPrefix(line, "WIDTH_CURRENT=") {
			if v, err := strconv.ParseUint(strings.TrimPrefix(line, "WIDTH_CURRENT="), 10, 32); err == nil {
				res.WidthCurrent = uint32(v)
			}
		} else if strings.HasPrefix(line, "TLS_TARGET=") {
			if v, err := strconv.ParseUint(strings.TrimPrefix(line, "TLS_TARGET="), 10, 32); err == nil {
				res.TLSTarget = uint32(v)
			}
		} else if strings.HasPrefix(line, "ERROR_CODE=") {
			res.ErrorCode = strings.TrimPrefix(line, "ERROR_CODE=")
		} else {
			res.Details = append(res.Details, line)
		}
	}

	// Legacy pattern fallback nếu không có header STATUS_CODE hoặc header rỗng
	if !hasHeader || res.StatusCode == StatusUnknown || res.StatusCode == "" {
		low := strings.ToLower(content)
		switch {
		case strings.Contains(low, "winring0") || strings.Contains(low, "throttlestop") ||
			strings.Contains(low, "loi 5") || strings.Contains(low, "error 5") ||
			strings.Contains(low, "error_access_denied"):
			res.StatusCode = StatusDrvFail
			res.ErrorCode = "ACCESS_DENIED_OR_DRIVER_BLOCKED"
		case strings.Contains(low, "khong tim thay") || strings.Contains(low, "not found") ||
			strings.Contains(low, "bus pci"):
			res.StatusCode = StatusNoGPU
			res.ErrorCode = "GPU_NOT_FOUND"
		case strings.Contains(low, "da dat muc tieu gen2 thanh cong") ||
			strings.Contains(low, "đã đạt mục tiêu gen2 thành công") ||
			strings.Contains(content, "TLS=Gen2"):
			res.StatusCode = StatusGen2Success
			res.SpeedCurrent = 2
		case strings.Contains(low, "gpu tls=gen1") || strings.Contains(low, "chua dat") ||
			strings.Contains(low, "chưa đạt"):
			res.StatusCode = StatusGen1Stuck
			res.SpeedCurrent = 1
		}
	}

	return res
}

// WriteStructuredGen2Status ghi trạng thái có cấu trúc ra %ProgramData%\40HXUnlock\gen2_status.txt
func WriteStructuredGen2Status(c StatusContract) error {
	p := Gen2StatusPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return os.WriteFile(p, []byte(FormatStatus(c)), 0o644)
}
