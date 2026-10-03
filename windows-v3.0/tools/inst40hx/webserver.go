package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"40hxcore"
)

//go:embed web/*
var webFS embed.FS

// Global lock to prevent overlapping hardware operations
var (
	opMutex  sync.Mutex
	opBusyBy string
)

func tryAcquireOp(what string) bool {
	opMutex.Lock()
	defer opMutex.Unlock()
	if opBusyBy != "" {
		fmt.Printf("[!] Đang bận thao tác '%s' — Bỏ qua yêu cầu '%s'\n", opBusyBy, what)
		return false
	}
	opBusyBy = what
	return true
}

func releaseOp() {
	opMutex.Lock()
	defer opMutex.Unlock()
	opBusyBy = ""
}

// Log Hub: broadcasts real-time stdout/stderr to all connected Web UI SSE clients
type sseLogHub struct {
	mu      sync.Mutex
	clients map[chan string]bool
	history []string
}

var hub = &sseLogHub{
	clients: make(map[chan string]bool),
	history: make([]string, 0, 300),
}

func (h *sseLogHub) Write(p []byte) (int, error) {
	s := string(p)
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if len(h.history) >= 300 {
			h.history = h.history[1:]
		}
		h.history = append(h.history, l)

		// Broadcast to all active SSE client channels
		for ch := range h.clients {
			select {
			case ch <- l:
			default:
			}
		}
	}
	return len(p), nil
}

func (h *sseLogHub) addClient() (chan string, []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 100)
	h.clients[ch] = true
	histCopy := make([]string, len(h.history))
	copy(histCopy, h.history)
	return ch, histCopy
}

func (h *sseLogHub) removeClient(ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}

func runWebGUI() {
	if !isAdmin() {
		selfElevate()
		return
	}

	AttachLogSink(hub)
	fmt.Println("Khởi chạy CMP 40HX / 30HX Modern Web Control Center...")

	// Extract sub-filesystem from webFS
	subWeb, err := fs.Sub(webFS, "web")
	if err != nil {
		fmt.Println("[!] Lỗi nạp tài nguyên web nhúng:", err)
		runGUI() // Fallback to classic walk GUI
		return
	}

	mux := http.NewServeMux()

	// 1. Static Web Files
	fsServer := http.FileServer(http.FS(subWeb))
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

	// 2. Real-time Logs SSE Stream
	mux.HandleFunc("/api/logs/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		ch, history := hub.addClient()
		defer hub.removeClient(ch)

		// Send initial history
		for _, line := range history {
			fmt.Fprintf(w, "data: %s\n\n", line)
		}
		flusher.Flush()

		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case line, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "data: %s\n\n", line)
				flusher.Flush()
			}
		}
	})

	// 3. Status API
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		items := scanStatus()

		flags := make(map[string]bool)
		for _, it := range items {
			flags[it.Name] = it.Ok
		}

		strat := hxcore.DriverStrategy()
		autoHard := hxcore.ConfigInt("Gen2AutoHard", 1) != 0
		cnt, interval := hxcore.Gen2RetryPolicy()

		gpuFound := flags["Card đồ hoạ"]
		gspActive := flags["GSP (EnableGpuFirmware)"]

		gpuName := "Chưa phát hiện GPU CMP"
		pciBusId := "Không khả dụng"
		if prof, ok := hxcore.FindGPUWithProfile(); ok {
			gpuName = fmt.Sprintf("%s (%s)", prof.Name, prof.Family)
			pciBusId = prof.HardwareID
		}

		isGen2 := false
		if gpuFound {
			isGen2 = gen2Succeeded || flags["Tác vụ tự khởi động"]
		}

		aspmOK := true
		if v, ok := flags["Tiết kiệm điện PCIe (ASPM)"]; ok {
			aspmOK = v
		}

		needGsp := !gspActive
		needDrv := hxcore.Gen2DriversNeedDeploy()
		needEfi := false
		if flags["Chế độ Boot"] {
			needEfi = !flags["ESP EFI Mở khoá"] || !flags["Mục khởi động BIOS"]
		}
		needTask := !flags["Tác vụ tự khởi động"]
		needFast := !flags["Khởi động nhanh (Fast Startup)"]
		needAspm := !aspmOK
		needPerf := !hxcore.HighPerfPlanActive()

		needDefOff := false
		if on, err := hxcore.DefenderRealtimeProtectionOn(); err == nil {
			needDefOff = on
		}

		resp := map[string]interface{}{
			"gpuDetected":    gpuFound,
			"gpuName":        gpuName,
			"pciBusId":       pciBusId,
			"gspActive":      gspActive,
			"isGen2":         isGen2,
			"driverStrategy": strat,
			"autoHard":       autoHard,
			"retryCount":     cnt,
			"retryInterval":  interval,
			"items":          items,
			"recommendations": map[string]bool{
				"gsp":    needGsp,
				"drv":    needDrv,
				"efi":    needEfi,
				"task":   needTask,
				"fast":   needFast,
				"aspm":   needAspm,
				"perf":   needPerf,
				"defoff": needDefOff,
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	// 4. Unlock Gen2 Now
	mux.HandleFunc("/api/unlock-now", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if !tryAcquireOp("Mở khoá PCIe ngay") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hệ thống đang bận thao tác khác."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("[PCIe] Bắt đầu kích hoạt mở khóa Gen2 ngay...")
			gen2Main()
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 5. Full Install 1-Click
	mux.HandleFunc("/api/full-install", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if !tryAcquireOp("Cài đặt toàn bộ") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hệ thống đang bận thao tác khác."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("== Cài đặt toàn bộ tự động (Một chạm) ==")
			install()
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 6. Gen2 and Autostart Task
	mux.HandleFunc("/api/gen2-and-task", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if !tryAcquireOp("Mở khoá & Cài tự khởi động") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hệ thống đang bận thao tác khác."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("== Mở khoá PCIe và cài đặt tự khởi động ==")
			fmt.Println("[PCIe] Bước 1/2: Mở khoá phiên hiện tại...")
			gen2Main()
			fmt.Println("[PCIe] Bước 2/2: Cài đặt tự khởi động khi đăng nhập Windows...")
			installDrivers()
			if err := hxcore.AddDefenderExclusions(); err != nil {
				fmt.Println("  [Defender] Lỗi ngoại lệ:", err)
			} else {
				fmt.Println("  [Defender] Đã thêm loại trừ cho file driver và ProgramData")
			}
			setRunKey()
			if err := setupGen2Task(); err != nil {
				fmt.Println("  [!] Đăng ký tác vụ tự chạy thất bại:", err)
			} else {
				fmt.Println("  [Tự chạy] Đăng ký thành công — Tự động mở khoá PCIe khi đăng nhập")
			}
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 7. Install Selected Components
	mux.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		var sel map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&sel); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !tryAcquireOp("Cài đặt thành phần đã chọn") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hệ thống đang bận thao tác khác."})
			return
		}
		go func() {
			defer releaseOp()
			executeInstallSelected(sel)
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 8. Save Policy
	mux.HandleFunc("/api/save-policy", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		var req struct {
			Strategy      int `json:"strategy"`
			AutoHard      int `json:"autoHard"`
			RetryCount    int `json:"retryCount"`
			RetryInterval int `json:"retryInterval"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := hxcore.SetConfigInt("DriverStrategy", req.Strategy); err != nil {
			http.Error(w, "Lưu DriverStrategy thất bại: "+err.Error(), http.StatusInternalServerError)
			return
		}
		hxcore.SetConfigInt("Gen2AutoHard", req.AutoHard)
		hxcore.SetConfigInt("Gen2RetryCount", req.RetryCount)
		hxcore.SetConfigInt("Gen2RetryIntervalMin", req.RetryInterval)

		fmt.Printf("[Cấu hình] Đã lưu: Chiến lược=%d Gen2AutoHard=%d Thử lại=%d lần / Giãn cách=%d phút\n",
			req.Strategy, req.AutoHard, req.RetryCount, req.RetryInterval)
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
	})

	// Bind to localhost port
	port := 40100
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		// Try dynamic port if 40100 is occupied
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			fmt.Println("[!] Không thể tạo máy chủ web nội bộ:", err)
			runGUI() // Fallback to walk GUI
			return
		}
	}

	addr := listener.Addr().String()
	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("\n============================================================\n")
	fmt.Printf(" [✓] CMP Control Center Web UI đang chạy tại: %s\n", url)
	fmt.Printf("     Tự động mở trình duyệt... (Đóng cửa sổ này để thoát)\n")
	fmt.Printf("============================================================\n\n")

	// Auto launch browser
	go func() {
		time.Sleep(400 * time.Millisecond)
		openBrowser(url)
	}()

	server := &http.Server{Handler: mux}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Println("[!] Máy chủ web dừng:", err)
	}
}

func openBrowser(url string) {
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}
