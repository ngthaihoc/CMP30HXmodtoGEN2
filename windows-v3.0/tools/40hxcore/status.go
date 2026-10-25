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
	StatusGen2Skipped StatusCode = "GEN2_SKIPPED"
	StatusGen1Stuck   StatusCode = "GEN1_STUCK"
	StatusHwLimit     StatusCode = "GEN2_HARDWARE_LIMIT"
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
	DeviceID     uint16
	MRRS         uint32
	ObservedAt   time.Time
}

// LinkState là kết quả lần chạy gần nhất, không phải phép đo phần cứng trực tiếp.
type LinkState struct {
	Configured    bool
	Speed         uint32
	Width         uint32
	MRRS          uint32
	ThroughputMBs uint32
	ObservedAt    time.Time
}

// LinkState tập trung cách diễn giải bằng chứng, giữ số 0 khi chưa biết.
func (c StatusContract) LinkState(deviceID uint16) LinkState {
	if (c.DeviceID != 0 && c.DeviceID != deviceID) || c.StatusCode == StatusUnknown || c.StatusCode == StatusNoGPU || c.StatusCode == StatusGen2Skipped {
		return LinkState{}
	}
	s := LinkState{Speed: c.SpeedCurrent, Width: c.WidthCurrent, MRRS: c.MRRS, ObservedAt: c.ObservedAt}
	if s.Speed > 6 {
		s.Speed = 0
	}
	if s.Width > 16 {
		s.Width = 0
	}
	if s.MRRS < 128 || s.MRRS > 4096 || s.MRRS&(s.MRRS-1) != 0 {
		s.MRRS = 0
	}
	s.Configured = c.StatusCode == StatusGen2Success && ((c.TLSTarget >= 2 && c.TLSTarget <= 6) || s.Speed >= 2)
	// Băng thông lý thuyết một chiều, đã trừ mã hóa đường truyền.
	if s.Speed == 1 || s.Speed == 2 {
		s.ThroughputMBs = 250 * s.Speed * s.Width
	}
	return s
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
	if c.DeviceID != 0 {
		sb.WriteString(fmt.Sprintf("DEVICE_ID=%04X\n", c.DeviceID))
	}
	if c.MRRS != 0 {
		sb.WriteString(fmt.Sprintf("MRRS_BYTES=%d\n", c.MRRS))
	}
	if !c.ObservedAt.IsZero() {
		sb.WriteString("OBSERVED_AT=" + c.ObservedAt.UTC().Format(time.RFC3339Nano) + "\n")
	}

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
		} else if strings.HasPrefix(line, "DEVICE_ID=") {
			v, _ := strconv.ParseUint(strings.TrimPrefix(line, "DEVICE_ID="), 16, 16)
			res.DeviceID = uint16(v)
		} else if strings.HasPrefix(line, "MRRS_BYTES=") {
			v, _ := strconv.ParseUint(strings.TrimPrefix(line, "MRRS_BYTES="), 10, 32)
			res.MRRS = uint32(v)
		} else if strings.HasPrefix(line, "OBSERVED_AT=") {
			res.ObservedAt, _ = time.Parse(time.RFC3339Nano, strings.TrimPrefix(line, "OBSERVED_AT="))
		} else {
			res.Details = append(res.Details, line)
		}
	}

	// Chỉ đọc văn bản legacy khi không có header STATUS_CODE.
	if !hasHeader {
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
			strings.Contains(low, "đã đạt mục tiêu gen2 thành công"):
			res.StatusCode = StatusGen2Success
			res.SpeedCurrent = 2
		case strings.Contains(content, "TLS=Gen2"):
			res.StatusCode = StatusGen2Success
			res.TLSTarget = 2
		case strings.Contains(low, "gpu tls=gen1") || strings.Contains(low, "chua dat") ||
			strings.Contains(low, "chưa đạt"):
			res.StatusCode = StatusGen1Stuck
			res.SpeedCurrent = 1
		}
	}
	switch res.StatusCode {
	case StatusGen2Success, StatusGen2Skipped, StatusGen1Stuck, StatusHwLimit, StatusDrvFail, StatusNoGPU:
	default:
		res.StatusCode = StatusUnknown
	}

	return res
}

// WriteStructuredGen2Status ghi trạng thái có cấu trúc ra %ProgramData%\40HXUnlock\gen2_status.txt
func WriteStructuredGen2Status(c StatusContract) error {
	p := Gen2StatusPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return os.WriteFile(p, []byte(FormatStatus(c)), 0o644)
}
