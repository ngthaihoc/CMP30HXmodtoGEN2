package main

import (
	"context"
	"fmt"
	"sync"

	hxcore "40hxcore"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct chính điều phối toàn bộ binding giữa Wails và Go Core
type App struct {
	ctx             context.Context
	opMu            sync.Mutex
	busyBy          string
	telemetryCancel context.CancelFunc
}

// NewApp khởi tạo thể hiện App
func NewApp() *App {
	return &App{}
}

// startup được Wails gọi khi ứng dụng khởi chạy
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	InitEmbeddedDriverHook()
	a.Log("[KHỞI ĐỘNG] CMP Unlock Control Center AIO v3.0.0 sẵn sàng.")
	a.startTelemetryLoop()
}

// shutdown được Wails gọi khi ứng dụng đóng
func (a *App) shutdown(ctx context.Context) {
	if a.telemetryCancel != nil {
		a.telemetryCancel()
	}
	CleanupToolAssets()
	hxcore.CleanupByovd()
}

// tryAcquire chiếm quyền thực thi thao tác phần cứng (chống xung đột song song)
func (a *App) tryAcquire(what string) bool {
	a.opMu.Lock()
	defer a.opMu.Unlock()
	if a.busyBy != "" {
		a.Log(fmt.Sprintf("[!] Hệ thống đang bận thực hiện %q — Bỏ qua yêu cầu %q", a.busyBy, what))
		return false
	}
	a.busyBy = what
	return true
}

// releaseOp giải phóng quyền thực thi
func (a *App) releaseOp() {
	a.opMu.Lock()
	defer a.opMu.Unlock()
	a.busyBy = ""
}

// Log đẩy thông điệp nhật ký thời gian thực sang giao diện Wails
func (a *App) Log(line string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "log:entry", line)
	}
}

// GetAppVersion trả về phiên bản hiện tại
func (a *App) GetAppVersion() string {
	return "v3.0.0"
}
