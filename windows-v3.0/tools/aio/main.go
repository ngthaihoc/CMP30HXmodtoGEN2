package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"

	hxcore "40hxcore"

	"github.com/wailsapp/go-webview2/webviewloader"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend
var assets embed.FS

func main() {
	app := NewApp()

	// 0. Hỗ trợ wails build sinh bindings tự động
	if os.Getenv("tsoutputtype") != "" {
		_ = wails.Run(&options.App{
			Bind: []interface{}{app},
		})
		return
	}

	// 1. Xử lý các cờ dòng lệnh (CLI mode cho tác vụ tự chạy khi Logon và kịch bản)
	if len(os.Args) > 1 {
		handleCLIArgs()
		return
	}

	// 2. Chế độ GUI: Bắt buộc quyền Administrator
	if !hxcore.IsAdmin() {
		hxcore.SelfElevate("CMP Unlock Control Center AIO")
		return
	}

	// 3. Kiểm tra tính khả dụng của WebView2 Runtime
	wv2Ver, errVer := webviewloader.GetAvailableCoreWebView2BrowserVersionString("")
	if errVer != nil || wv2Ver == "" {
		// Thiếu WebView2 Runtime -> Kích hoạt Hybrid Fallback Web UI (mở trình duyệt mặc định, không popup)
		runFallbackWebServer(app)
		return
	}

	err := wails.Run(&options.App{
		Title:     "CMP Unlock Control Center AIO — NVIDIA 30HX & 40HX",
		Width:     1120,
		Height:    780,
		MinWidth:  980,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 15, B: 20, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi chạy Wails AIO: %v\n", err)
	}
}

func handleCLIArgs() {
	InitEmbeddedDriverHook()
	defer CleanupToolAssets()
	defer hxcore.CleanupByovd()

	app := NewApp()

	for i, a := range os.Args[1:] {
		switch a {
		case "-gen2", "-gen2-30hx":
			res := app.UnlockGen2Now()
			fmt.Printf("Kết quả mở khoá Gen2: %v (Tốc độ=%d, Độ rộng=%d): %s\n", res.Success, res.Speed, res.Width, res.Message)
			return
		case "-force-root-gen2":
			res := app.ForceRootGen2()
			fmt.Printf("Kết quả ép Root Port Gen2: %v (Tốc độ=%d, Độ rộng=%d): %s\n", res.Success, res.Speed, res.Width, res.Message)
			return
		case "-optimize":
			confirmed := false
			for _, flag := range os.Args[i+2:] {
				if flag == "-y" || flag == "-confirm-riot" {
					confirmed = true
					break
				}
			}
			if !confirmed {
				fmt.Fprintln(os.Stderr, "CẢNH BÁO: Tối ưu Riot sẽ tắt VulnerableDriverBlocklistEnable và thay đổi cấu hình hệ thống. Thêm -y để xác nhận.")
				return
			}
			if err := app.RiotOptimize(true); err != nil {
				fmt.Fprintf(os.Stderr, "Tối ưu Riot thất bại: %v\n", err)
			}
			return
		case "-json-status":
			st := app.GetRiotStatus()
			_ = json.NewEncoder(os.Stdout).Encode(st)
			return
		case "-uninstall":
			res := app.UninstallAll()
			fmt.Printf("Gỡ cài đặt hoàn tất: %s\n", res.Message)
			return
		case "-status":
			rep := app.ScanEnvironment()
			fmt.Printf("GPU: %s (Phát hiện: %v, Gen2: %v)\n", rep.GPUName, rep.GPUDetected, rep.IsGen2)
			for _, it := range rep.Items {
				statusStr := "✗"
				if it.Ok {
					statusStr = "✓"
				}
				fmt.Printf("  [%s] %s: %s\n", statusStr, it.Name, it.Note)
			}
			return
		}
	}
}
