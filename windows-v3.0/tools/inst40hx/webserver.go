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
		fmt.Printf("[!] ─Éang bß║¡n thao t├íc '%s' ΓÇö Bß╗Å qua y├¬u cß║ºu '%s'\n", opBusyBy, what)
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
	fmt.Println("Khß╗ƒi chß║íy CMP 40HX / 30HX Modern Web Control Center...")

	// Extract sub-filesystem from webFS
	subWeb, err := fs.Sub(webFS, "web")
	if err != nil {
		fmt.Println("[!] Lß╗ùi nß║íp t├ái nguy├¬n web nh├║ng:", err)
		runGUI() // Fallback to classic walk GUI
		return
	}

	mux := http.NewServeMux()

	// 1. Static Web Files
	mux.Handle("/", http.FileServer(http.FS(subWeb)))

	// 2. Real-time Logs SSE Stream
	mux.HandleFunc("/api/logs/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
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
		w.Header().Set("Content-Type", "application/json")
		items := scanStatus()

		flags := make(map[string]bool)
		for _, it := range items {
			flags[it.Name] = it.Ok
		}

		strat := hxcore.DriverStrategy()
		autoHard := hxcore.ConfigInt("Gen2AutoHard", 1) != 0
		cnt, interval := hxcore.Gen2RetryPolicy()

		gpuFound := flags["Card ─æß╗ô hoß║í"]
		gspActive := flags["GSP (EnableGpuFirmware)"]

		gpuName := "Ch╞░a ph├ít hiß╗çn GPU CMP"
		pciBusId := "Kh├┤ng khß║ú dß╗Ñng"
		if prof, ok := hxcore.FindGPUWithProfile(); ok {
			gpuName = fmt.Sprintf("%s (%s)", prof.Name, prof.Family)
			pciBusId = prof.HardwareID
		}

		isGen2 := false
		if gpuFound {
			isGen2 = gen2Succeeded || flags["T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng"]
		}

		aspmOK := true
		if v, ok := flags["Tiß║┐t kiß╗çm ─æiß╗çn PCIe (ASPM)"]; ok {
			aspmOK = v
		}

		needGsp := !gspActive
		needDrv := hxcore.Gen2DriversNeedDeploy()
		needEfi := false
		if flags["Chß║┐ ─æß╗Ö Boot"] {
			needEfi = !flags["ESP EFI Mß╗ƒ kho├í"] || !flags["Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS"]
		}
		needTask := !flags["T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng"]
		needFast := !flags["Khß╗ƒi ─æß╗Öng nhanh (Fast Startup)"]
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
		w.Header().Set("Content-Type", "application/json")
		if !tryAcquireOp("Mß╗ƒ kho├í PCIe ngay") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hß╗ç thß╗æng ─æang bß║¡n thao t├íc kh├íc."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("[PCIe] Bß║»t ─æß║ºu k├¡ch hoß║ít mß╗ƒ kh├│a Gen2 ngay...")
			gen2Main()
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 5. Full Install 1-Click
	mux.HandleFunc("/api/full-install", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !tryAcquireOp("C├ái ─æß║╖t to├án bß╗Ö") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hß╗ç thß╗æng ─æang bß║¡n thao t├íc kh├íc."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("== C├ái ─æß║╖t to├án bß╗Ö tß╗▒ ─æß╗Öng (Mß╗Öt chß║ím) ==")
			install()
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 6. Gen2 and Autostart Task
	mux.HandleFunc("/api/gen2-and-task", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !tryAcquireOp("Mß╗ƒ kho├í & C├ái tß╗▒ khß╗ƒi ─æß╗Öng") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hß╗ç thß╗æng ─æang bß║¡n thao t├íc kh├íc."})
			return
		}
		go func() {
			defer releaseOp()
			fmt.Println("== Mß╗ƒ kho├í PCIe v├á c├ái ─æß║╖t tß╗▒ khß╗ƒi ─æß╗Öng ==")
			fmt.Println("[PCIe] B╞░ß╗¢c 1/2: Mß╗ƒ kho├í phi├¬n hiß╗çn tß║íi...")
			gen2Main()
			fmt.Println("[PCIe] B╞░ß╗¢c 2/2: C├ái ─æß║╖t tß╗▒ khß╗ƒi ─æß╗Öng khi ─æ─âng nhß║¡p Windows...")
			installDrivers()
			if err := hxcore.AddDefenderExclusions(); err != nil {
				fmt.Println("  [Defender] Lß╗ùi ngoß║íi lß╗ç:", err)
			} else {
				fmt.Println("  [Defender] ─É├ú th├¬m loß║íi trß╗½ cho file driver v├á ProgramData")
			}
			setRunKey()
			if err := setupGen2Task(); err != nil {
				fmt.Println("  [!] ─É─âng k├╜ t├íc vß╗Ñ tß╗▒ chß║íy thß║Ñt bß║íi:", err)
			} else {
				fmt.Println("  [Tß╗▒ chß║íy] ─É─âng k├╜ th├ánh c├┤ng ΓÇö Tß╗▒ ─æß╗Öng mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p")
			}
		}()
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// 7. Install Selected Components
	mux.HandleFunc("/api/install", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var sel map[string]bool
		if err := json.NewDecoder(r.Body).Decode(&sel); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if !tryAcquireOp("C├ái ─æß║╖t th├ánh phß║ºn ─æ├ú chß╗ìn") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hß╗ç thß╗æng ─æang bß║¡n thao t├íc kh├íc."})
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
		w.Header().Set("Content-Type", "application/json")
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
			http.Error(w, "L╞░u DriverStrategy thß║Ñt bß║íi: "+err.Error(), http.StatusInternalServerError)
			return
		}
		hxcore.SetConfigInt("Gen2AutoHard", req.AutoHard)
		hxcore.SetConfigInt("Gen2RetryCount", req.RetryCount)
		hxcore.SetConfigInt("Gen2RetryIntervalMin", req.RetryInterval)

		fmt.Printf("[Cß║Ñu h├¼nh] ─É├ú l╞░u: Chiß║┐n l╞░ß╗úc=%d Gen2AutoHard=%d Thß╗¡ lß║íi=%d lß║ºn / Gi├ún c├ích=%d ph├║t\n",
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
			fmt.Println("[!] Kh├┤ng thß╗â tß║ío m├íy chß╗º web nß╗Öi bß╗Ö:", err)
			runGUI() // Fallback to walk GUI
			return
		}
	}

	addr := listener.Addr().String()
	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("\n============================================================\n")
	fmt.Printf(" [Γ£ô] CMP Control Center Web UI ─æang chß║íy tß║íi: %s\n", url)
	fmt.Printf("     Tß╗▒ ─æß╗Öng mß╗ƒ tr├¼nh duyß╗çt... (─É├│ng cß╗¡a sß╗ò n├áy ─æß╗â tho├ít)\n")
	fmt.Printf("============================================================\n\n")

	// Auto launch browser
	go func() {
		time.Sleep(400 * time.Millisecond)
		openBrowser(url)
	}()

	server := &http.Server{Handler: mux}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Println("[!] M├íy chß╗º web dß╗½ng:", err)
	}
}

func openBrowser(url string) {
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}
