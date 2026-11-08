package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	hxcore "40hxcore"
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

func validFallbackMutation(r *http.Request) bool {
	if r.Method != http.MethodPost || r.Header.Get("Origin") == "" || r.Host == "" {
		return false
	}
	origin, err := url.Parse(r.Header.Get("Origin"))
	if err != nil || origin.Scheme != "http" || origin.Host != r.Host || origin.User != nil || origin.Path != "" && origin.Path != "/" || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return false
	}
	return true
}

func fallbackMutationGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/events/stream" && !validFallbackMutation(r) {
			http.Error(w, "Yêu cầu API không hợp lệ hoặc khác nguồn", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/api/events/stream" {
			host, _, err := net.SplitHostPort(r.Host)
			ip := net.ParseIP(host)
			origin := r.Header.Get("Origin")
			if r.Method != http.MethodGet || err != nil || ip == nil || !ip.IsLoopback() || origin != "" && origin != "http://"+r.Host {
				http.Error(w, "Yêu cầu luồng sự kiện không hợp lệ hoặc khác nguồn", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func confirmFallbackOperation(title, warning string) bool {
	return hxcore.MsgBoxYesNo(title, warning+"\n\nBạn xác nhận tiếp tục?")
}

func requireFallbackConfirmation(w http.ResponseWriter, confirm func() bool) bool {
	if confirm() {
		return true
	}
	http.Error(w, "Thao tác đã bị hủy hoặc thiếu xác nhận riêng.", http.StatusForbidden)
	return false
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

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Trình duyệt không hỗ trợ luồng sự kiện SSE", http.StatusInternalServerError)
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
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Mở khóa Gen2", "Thao tác thay đổi trạng thái phần cứng; không vượt giới hạn Gen2.")
		}) {
			return
		}
		jsonResp(w, app.UnlockGen2Now())
	})
	mux.HandleFunc("/api/force-root-gen2", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Buộc root Gen2", "Thay đổi liên kết PCIe ở mức thấp; không thực hiện reset nguy hiểm trên CMP 30HX.")
		}) {
			return
		}
		jsonResp(w, app.ForceRootGen2())
	})
	mux.HandleFunc("/api/register-task", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Đăng ký tự khởi động", "Tạo tác vụ Windows chạy với quyền cao khi đăng nhập.")
		}) {
			return
		}
		err := app.RegisterAutostartTask()
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/unregister-task", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Gỡ tự khởi động", "Xóa tác vụ tự động mở khóa khi đăng nhập.")
		}) {
			return
		}
		err := app.UnregisterAutostartTask()
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/save-policy", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Lưu chính sách", "Ghi thay đổi chính sách mở khóa vào cấu hình.")
		}) {
			return
		}
		var req PolicyConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonResp(w, map[string]interface{}{"success": false, "error": "Dữ liệu chính sách không hợp lệ."})
			return
		}
		err := app.SavePolicy(req)
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.ScanEnvironment())
	})
	mux.HandleFunc("/api/hvci", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.CheckHVCIStatus())
	})
	mux.HandleFunc("/api/open-core-isolation", func(w http.ResponseWriter, r *http.Request) {
		if err := app.OpenCoreIsolationSettings(); err != nil {
			jsonResp(w, map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		jsonResp(w, map[string]bool{"success": true})
	})
	mux.HandleFunc("/api/riot-status", func(w http.ResponseWriter, r *http.Request) {
		jsonResp(w, app.GetRiotStatus())
	})
	mux.HandleFunc("/api/riot-optimize", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Tối ưu Riot", "Thay đổi cấu hình bảo mật, dịch vụ và driver hệ thống.")
		}) {
			return
		}
		err := app.RiotOptimize(true)
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/riot-revert", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Hoàn tác Riot", "Khôi phục cài đặt bảo mật và đồ họa Riot.")
		}) {
			return
		}
		err := app.RiotRevertOptimization()
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/riot-sign-efi", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Phrase string `json:"phrase"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonResp(w, map[string]interface{}{"success": false, "error": "Dữ liệu yêu cầu không hợp lệ."})
			return
		}
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Ký EFI Riot", "Thay đổi EFI/driver có ảnh hưởng đến khởi động và chống gian lận.")
		}) {
			return
		}
		err := app.RiotSignEFI(req.Phrase)
		jsonResp(w, map[string]interface{}{"success": err == nil, "error": fmt.Sprint(err)})
	})
	mux.HandleFunc("/api/reboot-bios", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Khởi động lại BIOS", "Máy sẽ khởi động lại ngay; hãy lưu công việc trước.")
		}) {
			return
		}
		if err := app.RebootToBIOS(); err != nil {
			jsonResp(w, map[string]interface{}{"success": false, "error": err.Error()})
			return
		}
		jsonResp(w, map[string]bool{"success": true})
	})
	mux.HandleFunc("/api/fix-csm", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Sửa CSM", "Thay đổi thiết lập nguồn PCIe và registry; tạo bản sao lưu hoặc ghi lại trạng thái trước.")
		}) {
			return
		}
		jsonResp(w, app.FixCSMError43())
	})
	mux.HandleFunc("/api/setup-rebar", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			BarSizeGB int  `json:"barSizeGB"`
			Confirmed bool `json:"confirmed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonResp(w, map[string]interface{}{"success": false, "error": "Cần xác nhận cảnh báo rủi ro ReBAR trước khi thay đổi cấu hình."})
			return
		}
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Thiết lập ReBAR", "Thay đổi cấu hình PCIe/firmware; cần reboot, xác minh bằng GPU-Z và giữ khả năng hoàn tác.")
		}) {
			return
		}
		jsonResp(w, app.SetupReBar(req.BarSizeGB))
	})
	mux.HandleFunc("/api/revert-rebar", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Hoàn tác ReBAR", "Khôi phục cấu hình trước đó; cần reboot rồi xác minh trạng thái bằng GPU-Z.")
		}) {
			return
		}
		jsonResp(w, app.RevertReBar())
	})
	mux.HandleFunc("/api/uninstall", func(w http.ResponseWriter, r *http.Request) {
		if !requireFallbackConfirmation(w, func() bool {
			return confirmFallbackOperation("Gỡ cài đặt", "Xóa driver, tác vụ và các cấu hình đã cài bởi ứng dụng.")
		}) {
			return
		}
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

	server := &http.Server{Handler: fallbackMutationGuard(mux)}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		_ = server.Close()
	}()

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Log(fmt.Sprintf("[LỖI] Máy chủ web nội bộ dừng bất thường: %v", err))
	}
}

func openBrowser(url string) {
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}
