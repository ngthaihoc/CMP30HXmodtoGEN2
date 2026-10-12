package main

import (
	"embed"
	"fmt"
	"os"

	hxcore "40hxcore"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend
var assets embed.FS

func main() {
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

	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "CMP Unlock Control Center AIO — NVIDIA 30HX & 40HX",
		Width:             1120,
		Height:            780,
		MinWidth:          980,
		MinHeight:         680,
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
	app := NewApp()

	for _, a := range os.Args[1:] {
		switch a {
		case "-gen2", "-gen2-30hx", "-force-root-gen2":
			res := app.UnlockGen2Now()
			fmt.Printf("Gen2 Result: %v (Speed=%d, Width=%d): %s\n", res.Success, res.Speed, res.Width, res.Message)
			return
		case "-optimize":
			_ = app.RiotOptimize()
			fmt.Println("Riot Optimization completed.")
			return
		case "-uninstall":
			res := app.UninstallAll()
			fmt.Printf("Uninstall completed: %s\n", res.Message)
			return
		case "-status":
			rep := app.ScanEnvironment()
			fmt.Printf("GPU: %s (Detected: %v, Gen2: %v)\n", rep.GPUName, rep.GPUDetected, rep.IsGen2)
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
