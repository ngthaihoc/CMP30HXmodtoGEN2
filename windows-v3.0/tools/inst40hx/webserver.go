package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"40hxcore"

	"golang.org/x/sys/windows/registry"
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

	// 9. Riot Games & Vanguard Status Query
	mux.HandleFunc("/api/riot/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		exePath := findUnlockRiotExe()
		if exePath != "" {
			cmd := exec.Command(exePath, "-json-status")
			out, err := cmd.Output()
			if err == nil && len(out) > 0 {
				w.Write(out)
				return
			}
		}

		// Fallback: direct evaluation
		prof, _ := hxcore.FindGPUWithProfile()
		sbOn := hxcore.SecureBootOn()
		isWin11 := isWindows11()
		is40HX := (prof.DeviceID == 0x1F0B || strings.Contains(strings.ToUpper(prof.Name), "40HX"))
		is30HX := (prof.DeviceID == 0x2189 || strings.Contains(strings.ToUpper(prof.Name), "30HX"))

		model := "Unknown"
		if is40HX {
			model = "CMP 40HX"
		} else if is30HX {
			model = "CMP 30HX"
		}

		osName := "Windows 10"
		if isWin11 {
			osName = "Windows 11"
		}
		sbDesc := "ĐÃ TẮT (Disabled)"
		if sbOn {
			sbDesc = "ĐANG BẬT (Enabled)"
		}

		st := RiotStatusJSON{
			Model:                  model,
			IsWin11:                isWin11,
			SecureBootOn:           sbOn,
			CanPlayLeagueOfLegends: true,
			OsDesc:                 osName,
			SecureBootDesc:         sbDesc,
			LoLDesc:                "✓ Sẵn sàng 100% (MSHybrid CASO + Borderless Windowed)",
		}

		if is40HX {
			st.HasTensorCore = true
			st.GpuDesc = "NVIDIA CMP 40HX [TU106] (Gen 2 & Tensor Core)"
			if !isWin11 {
				st.CanPlayValorant = true
				st.ValorantDesc = "✓ Sẵn sàng (Secure Boot Tắt + Tensor Core 100%)"
				st.Recommendation = "👉 Khuyến nghị: Giữ Secure Boot TẮT (Disabled) trong BIOS. Bấm [⚡ 1-CHẠM] để tối ưu hệ thống!"
			} else {
				st.NeedsEFISigning = true
				if sbOn {
					st.CanPlayValorant = true
					st.ValorantDesc = "⚠️ Secure Boot BẬT: Cần ký Key vào BIOS db để nạp Tensor Core"
					st.Recommendation = "👉 Khuyến nghị: Bấm nút [🔐 Tự Động Ký Chữ Ký Số EFI] bên dưới để nạp Key cá nhân vào BIOS db!"
				} else {
					st.CanPlayValorant = false
					st.ValorantDesc = "ℹ️ Secure Boot TẮT: Chơi được LMHT. Cần nạp Key & Bật SB để chơi Valorant"
					st.Recommendation = "👉 Khuyến nghị: Bấm nút [🔐 Tự Động Ký Chữ Ký Số EFI] để chuẩn bị nạp Key và BẬT Secure Boot!"
				}
			}
		} else if is30HX {
			st.GpuDesc = "NVIDIA CMP 30HX [TU116] (Gen 2 Hardware Lock)"
			st.CanPlayValorant = (!isWin11 || sbOn)
			if !isWin11 || sbOn {
				st.ValorantDesc = "✓ Sẵn sàng (Secure Boot BẬT bình thường)"
				st.Recommendation = "👉 Khuyến nghị: CMP 30HX giữ Secure Boot BẬT bình thường. Bấm [⚡ 1-CHẠM] để tối ưu ngay!"
			} else {
				st.ValorantDesc = "ℹ️ Cần BẬT Secure Boot trong BIOS để chơi Valorant trên Win 11"
				st.Recommendation = "👉 Khuyến nghị: Vào BIOS BẬT Secure Boot để chơi Valorant. Bấm [⚡ 1-CHẠM] để tối ưu ngay!"
			}
		} else {
			st.GpuDesc = prof.Name
			st.CanPlayValorant = (!isWin11 || sbOn)
			st.ValorantDesc = "✓ Sẵn sàng"
			st.Recommendation = "👉 Khuyến nghị: Bấm [⚡ 1-CHẠM] để tối ưu Registry và MSHybrid CASO cho Riot Games."
		}

		json.NewEncoder(w).Encode(st)
	})

	// 10. Riot Games 1-Click Optimize
	mux.HandleFunc("/api/riot/optimize", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if !tryAcquireOp("Tối ưu hóa Riot Games (1-Chạm)") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hệ thống đang bận thao tác khác."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("== [Riot Vanguard] Bắt đầu tối ưu hóa hệ thống và dọn dẹp driver ==")
			exePath := findUnlockRiotExe()
			if exePath != "" {
				cmd := exec.Command(exePath, "-silent", "-optimize")
				cmd.Stdout = hub
				cmd.Stderr = hub
				if err := cmd.Run(); err != nil {
					fmt.Fprintf(hub, "[!] Lỗi khi chạy UnlockRiotGame: %v\n", err)
				}
			} else {
				fmt.Fprintf(hub, "[!] Không tìm thấy UnlockRiotGame.exe để thực thi tối ưu.\n")
			}
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 11. Riot EFI Signing (Authenticode)
	mux.HandleFunc("/api/riot/sign-efi", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if !tryAcquireOp("Ký chữ ký số EFI cá nhân") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hệ thống đang bận thao tác khác."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("== [UEFI Key] Bắt đầu tự động tạo chứng chỉ và ký Authenticode cho 40HXUNLK.EFI ==")
			exePath := findUnlockRiotExe()
			if exePath != "" {
				cmd := exec.Command(exePath, "-silent", "-auto-sign")
				cmd.Stdout = hub
				cmd.Stderr = hub
				if err := cmd.Run(); err != nil {
					fmt.Fprintf(hub, "[!] Lỗi khi chạy tự động ký EFI: %v\n", err)
				}
			} else {
				fmt.Fprintf(hub, "[!] Không tìm thấy UnlockRiotGame.exe để thực thi ký EFI.\n")
			}
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 12. Reboot to BIOS Firmware Setup
	mux.HandleFunc("/api/riot/reboot-bios", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Println("[BIOS] Nhận yêu cầu khởi động lại máy tính vào BIOS Setup...")
		cmd := exec.Command("shutdown", "/r", "/fw", "/t", "2")
		if err := cmd.Run(); err != nil {
			exec.Command("shutdown", "/r", "/t", "2").Run()
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "rebooting"})
	})

	// 13. Launch standalone UnlockRiotGame.exe GUI
	mux.HandleFunc("/api/riot/launch-gui", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		exePath := findUnlockRiotExe()
		if exePath == "" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"message": "Không tìm thấy UnlockRiotGame.exe"})
			return
		}
		cmd := exec.Command(exePath)
		if err := cmd.Start(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "launched"})
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

type RiotStatusJSON struct {
	Model                  string `json:"model"`
	IsWin11                bool   `json:"isWin11"`
	SecureBootOn           bool   `json:"secureBootOn"`
	HasTensorCore          bool   `json:"hasTensorCore"`
	CanPlayValorant        bool   `json:"canPlayValorant"`
	CanPlayLeagueOfLegends bool   `json:"canPlayLeagueOfLegends"`
	NeedsEFISigning        bool   `json:"needsEFISigning"`
	RecommendedSecureBoot  string `json:"recommendedSecureBoot"`
	GpuDesc                string `json:"gpuDesc"`
	OsDesc                 string `json:"osDesc"`
	SecureBootDesc         string `json:"secureBootDesc"`
	ValorantDesc           string `json:"valorantDesc"`
	LoLDesc                string `json:"lolDesc"`
	Recommendation         string `json:"recommendation"`
}

func findUnlockRiotExe() string {
	exe, err := os.Executable()
	var dir string
	if err == nil {
		dir = filepath.Dir(exe)
	}
	candidates := []string{
		filepath.Join(dir, "UnlockRiotGame.exe"),
		filepath.Join(dir, "..", "release", "UnlockRiotGame.exe"),
		filepath.Join(dir, "windows-v3.0", "release", "UnlockRiotGame.exe"),
		`D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\UnlockRiotGame.exe`,
		"UnlockRiotGame.exe",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}

func isWindows11() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	buildStr, _, err := k.GetStringValue("CurrentBuild")
	if err != nil {
		buildStr, _, err = k.GetStringValue("CurrentBuildNumber")
	}
	if err != nil {
		return false
	}
	var b int
	fmt.Sscanf(buildStr, "%d", &b)
	return b >= 22000
}
