package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type eventPayload struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

var (
	eventListenersMu sync.Mutex
	eventListeners   = make(map[chan eventPayload]bool)
)

func broadcastEvent(event string, data interface{}) {
	eventListenersMu.Lock()
	defer eventListenersMu.Unlock()
	p := eventPayload{Event: event, Data: data}
	for ch := range eventListeners {
		select {
		case ch <- p:
		default:
		}
	}
}

func runFallbackWebServer(app *App) {
	InitEmbeddedDriverHook()
	app.startTelemetryLoop()
	defer app.shutdown(context.Background())
	app.Log("[KHỞI ĐỘNG] CMP Unlock Control Center AIO v3.0.0 (Web Mode) sẵn sàng.")

	subFS, err := fs.Sub(assets, "frontend")
	if err != nil {
		fmt.Printf("[!] Lỗi nạp tài nguyên frontend: %v\n", err)
		return
	}

	mux := http.NewServeMux()

	// 1. Static Web Files
	fsServer := http.FileServer(http.FS(subFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasSuffix(p, ".html") || p == "/" || p == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else if strings.HasSuffix(p, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		} else if strings.HasSuffix(p, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		fsServer.ServeHTTP(w, r)
	})

	// 2. Real-time Events SSE Stream
	mux.HandleFunc("/api/events/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		ch := make(chan eventPayload, 100)
		eventListenersMu.Lock()
		eventListeners[ch] = true
		eventListenersMu.Unlock()

		defer func() {
			eventListenersMu.Lock()
			delete(eventListeners, ch)
			close(ch)
			eventListenersMu.Unlock()
		}()

		flusher.Flush()

		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case p, ok := <-ch:
				if !ok {
					return
				}
				b, _ := json.Marshal(p)
				fmt.Fprintf(w, "data: %s\n\n", string(b))
				flusher.Flush()
			}
		}
	})

	// 3. API RPC Endpoints
	jsonResp := func(w http.ResponseWriter, v interface{}) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(v)
	}

	mux.HandleFunc("/api/unlock-now", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.UnlockGen2Now())
	})
	mux.HandleFunc("/api/force-root-gen2", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.ForceRootGen2())
	})
	mux.HandleFunc("/api/register-task", func(w http.ResponseWriter, r *http.Request) {
		err := app.RegisterAutostartTask()
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/unregister-task", func(w http.ResponseWriter, r *http.Request) {
		err := app.UnregisterAutostartTask()
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/save-policy", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Strategy      int `json:"strategy"`
			RetryCount    int `json:"retryCount"`
			RetryInterval int `json:"retryInterval"`
			AutoHard      int `json:"autoHard"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		err := app.SavePolicy(req.Strategy, req.RetryCount, req.RetryInterval, req.AutoHard)
		jsonResp(w, map[string]interface{}{"success": err == nil})
	})
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.ScanEnvironment())
	})
	mux.HandleFunc("/api/hvci", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.CheckHVCIStatus())
	})
	mux.HandleFunc("/api/open-core-isolation", func(w http.ResponseWriter, r *http.Request) {
		_ = app.OpenCoreIsolationSettings()
		jsonResp(w, map[string]bool{"success": true})
	})
	mux.HandleFunc("/api/riot-status", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.GetRiotStatus())
	})
	mux.HandleFunc("/api/riot-optimize", func(w http.ResponseWriter, r *http.Request) {
		err := app.RiotOptimize()
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/riot-revert", func(w http.ResponseWriter, r *http.Request) {
		err := app.RiotRevertOptimization()
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/riot-sign-efi", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Phrase string `json:"phrase"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		err := app.RiotSignEFI(req.Phrase)
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/reboot-bios", func(w http.ResponseWriter, r *http.Request) {
		_ = app.RebootToBIOS()
		jsonResp(w, map[string]bool{"success": true})
	})
	mux.HandleFunc("/api/fix-csm", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.FixCSMError43())
	})
	mux.HandleFunc("/api/setup-rebar", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			BarSizeGB int `json:"barSizeGB"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		jsonResp(w, app.SetupReBar(req.BarSizeGB))
	})
	mux.HandleFunc("/api/revert-rebar", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.RevertReBar())
	})
	mux.HandleFunc("/api/uninstall", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.UninstallAll())
	})

	// 4. Khởi chạy HTTP Server trên localhost
	port := 40100
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fmt.Printf("[!] Không thể tạo máy chủ web nội bộ: %v\n", err)
			return
		}
	}

	addr := listener.Addr().String()
	url := fmt.Sprintf("http://%s", addr)

	fmt.Println("================================================================")
	fmt.Println("  [i] Phát hiện hệ thống chưa cài đặt WebView2 Runtime.")
	fmt.Println("  [✓] Tự động kích hoạt Hybrid Fallback Web UI (Zero Overhead)!")
	fmt.Printf("  [✓] Giao diện đang chạy tại: %s\n", url)
	fmt.Println("      Đang mở trình duyệt mặc định... (Đóng console để thoát)")
	fmt.Println("================================================================")

	go func() {
		time.Sleep(300 * time.Millisecond)
		openBrowser(url)
	}()

	server := &http.Server{Handler: mux}
	_ = server.Serve(listener)
}

func openBrowser(url string) {
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}
