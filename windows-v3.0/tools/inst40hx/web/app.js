/**
 * CMP 40HX / 30HX Unlock Control Center - Frontend Application Logic
 */

(function () {
  'use strict';

  // --- Bilingual Dictionary ---
  const i18n = {
    vi: {
      brand_desc: "NVIDIA CMP 40HX & 30HX PCIe Gen2 x16 / Kích hoạt Băng thông & Tính toán",
      status_connecting: "Đang kết nối...",
      status_connected: "Trực tuyến (Live)",
      status_disconnected: "Mất kết nối",
      btn_refresh: "Quét lại",
      refresh_tooltip: "Quét lại môi trường hệ thống",
      hud_silicon_title: "Thông tin Bán dẫn & PCIe Bus",
      spec_bus_id: "PCIe Bus Device:",
      spec_gsp: "GSP-RM Firmware:",
      spec_vanguard: "Riot Vanguard / Game:",
      spec_driver_strategy: "Chiến lược Driver:",
      lane_topology: "Cấu trúc 16 làn PCIe vật lý (Physical Lanes):",
      hud_bandwidth_title: "Băng thông & Liên kết PCIe",
      speed_locked: "Mặc định (Khóa eFuse)",
      speed_unlocked: "Đã mở khóa tối đa",
      btn_unlock_title: "Mở khóa PCIe Gen2 ngay",
      btn_unlock_sub: "Kích hoạt tức thì phiên hiện tại (không cần khởi động lại máy)",
      btn_full_title: "Cài đặt toàn bộ 1-chạm",
      btn_full_sub: "Tự động cấu hình chuẩn: GSP + Driver + Tự khởi động + Nguồn",
      btn_persist_title: "Mở khoá & Cài tự khởi động",
      btn_persist_sub: "Đăng ký tác vụ tự chạy khi đăng nhập Windows",
      audit_title: "Chẩn đoán Môi trường & Phần cứng",
      audit_sub: "Kiểm tra tính tương thích trước khi kích hoạt",
      audit_loading: "Đang nạp dữ liệu kiểm tra hệ thống...",
      components_title: "Thành phần & Thiết lập hệ thống",
      components_sub: "Chọn các mục cần cập nhật hoặc bấm [Cài đặt mục đã chọn]",
      name_gsp: "Bật GSP (EnableGpuFirmware=1)",
      name_drv: "Cài đặt Driver PCIe & Ngoại lệ Defender",
      name_efi: "EFI Mở khoá + Khởi động BIOS (Chỉ 40HX)",
      name_task: "Tự động mở khoá PCIe khi đăng nhập",
      name_fast: "Nguồn: Tắt Fast Startup (Khởi động nhanh)",
      name_aspm: "Nguồn: Tắt ASPM (Tiết kiệm điện PCIe)",
      name_perf: "Nguồn: Bật chế độ High Performance",
      name_defoff: "Tắt Defender Realtime Protection",
      desc_gsp: "Bắt buộc để tránh mã lỗi Code 43 sau khi mở khoá",
      desc_drv: "Nạp driver can thiệp thanh ghi & cấp quyền an toàn",
      desc_efi: "Nạp payload vào phân vùng ESP (chỉ áp dụng cho chuẩn UEFI)",
      desc_task: "Đăng ký tác vụ Task Scheduler SYSTEM và Run Key dự phòng",
      desc_fast: "Tránh tình trạng Windows nạp sleep image bỏ qua UEFI hook",
      desc_aspm: "Tránh PCIe tự động rớt về Gen1 x1 khi máy tính ở trạng thái rảnh",
      desc_perf: "Đảm bảo cấp đủ năng lượng cho liên kết PCIe hoạt động tối đa",
      desc_defoff: "Chỉ cần thiết khi phần mềm diệt virus chặn file sys can thiệp",
      btn_install_selected: "Cài đặt mục đã chọn",
      policy_title: "Chiến lược Driver & Tự phục hồi",
      policy_sub: "Cấu hình hành vi sau khi hoàn tất mở khoá",
      strat_0_name: "Dùng xong gỡ ngay (Khuyên dùng cho Game)",
      strat_0_desc: "Sau khi nâng tốc độ, driver can thiệp được gỡ hoàn toàn. An toàn tuyệt đối với Riot Vanguard & Anti-Cheat.",
      strat_1_name: "Tự động thử lại khi lỗi",
      strat_1_desc: "Tự động thử lại nếu lần đầu khởi tạo GPU chưa đạt tốc độ Gen2.",
      strat_2_name: "Thường trú (Canh giữ tốc độ PCIe)",
      strat_2_desc: "Driver chạy nền thường trực, định kỳ kiểm tra và ép xung PCIe trở lại nếu bị tụt xung.",
      desc_autohard: "Tự động kích hoạt Stage 2 (Link Disable + Reset PnP) khi mở khoá thường không đạt",
      retry_count_label: "Số lần thử lại:",
      retry_count_unit: "lần",
      retry_interval_label: "Giãn cách:",
      retry_interval_unit: "phút",
      btn_save_policy: "Lưu cấu hình chính sách",
      terminal_title: "Nhật ký Chẩn đoán thời gian thực",
      log_lines: "dòng log",
      btn_clear_log: "Xóa",
      btn_copy_log: "Sao chép",
      safety_title: "An toàn cho Game & Anti-Cheat (Riot Vanguard, EasyAntiCheat, BattlEye)",
      safety_desc: "Giải pháp v3.0 không flash VBIOS, không bật Test Signing, gỡ sạch driver can thiệp sau khi mở khoá. Đảm bảo 100% tính toàn vẹn hệ điều hành.",
      btn_riot_title: "Tối ưu hóa Riot Games (1-Chạm)",
      btn_riot_sub: "Dọn sạch driver can thiệp, kích hoạt MSHybrid CASO cho LoL / Valorant",
      riot_panel_title: "Tương Thích Riot Games (MSHybrid & Vanguard)",
      riot_panel_sub: "Hỗ trợ không cần cài lại driver, tối ưu LoL/TFT & Valorant",
      riot_lol_detail: "MSHybrid CASO + Tự động cấu hình Borderless Windowed (WindowMode=2) mượt mà không cổng xuất hình.",
      riot_valorant_detail: "Dọn dẹp driver can thiệp, tắt Testsigning đáp ứng tiêu chuẩn Riot Vanguard.",
      riot_rec_title: "KHUYẾN NGHỊ TỐI ƯU:",
      btn_riot_opt_inner: "⚡ Tối Ưu Hóa 1-Chạm",
      btn_riot_sign_inner: "🔐 Ký EFI (Valorant Win 11)",
      btn_riot_guide_inner: "📖 Hướng Dẫn BIOS",
      btn_launch_app: "Mở App Riêng",
      modal_bios_title: "Hướng Dẫn Nạp Key BIOS & Giữ Tensor Core",
      modal_bios_warning: "Hãy dùng điện thoại chụp lại hướng dẫn này trước khi khởi động lại máy tính!",
      modal_cert_locs: "Vị trí file chứng chỉ (CMP40HX_Key.cer):",
      modal_btn_close: "Đóng",
      modal_btn_reboot: "Khởi Động Lại Vào BIOS Ngay"
    },
    en: {
      brand_desc: "NVIDIA CMP 40HX & 30HX PCIe Gen2 x16 Bandwidth & Compute Enablement Suite",
      status_connecting: "Connecting...",
      status_connected: "Live Connected",
      status_disconnected: "Disconnected",
      btn_refresh: "Refresh",
      refresh_tooltip: "Re-scan system environment",
      hud_silicon_title: "Silicon & PCIe Bus Architecture",
      spec_bus_id: "PCIe Bus Device:",
      spec_gsp: "GSP-RM Firmware:",
      spec_vanguard: "Riot Vanguard / Game:",
      spec_driver_strategy: "Driver Strategy:",
      lane_topology: "PCIe x16 Physical Lanes Topology:",
      hud_bandwidth_title: "PCIe Link & Bandwidth",
      speed_locked: "Stock (eFuse Locked)",
      speed_unlocked: "Fully Unlocked",
      btn_unlock_title: "Unlock PCIe Gen2 Now",
      btn_unlock_sub: "Instantly retrain PCIe link for current session (no reboot needed)",
      btn_full_title: "1-Click Complete Setup",
      btn_full_sub: "Auto configure all: GSP + Drivers + Auto-Start + Power tuning",
      btn_persist_title: "Unlock & Install Autostart",
      btn_persist_sub: "Register persistent startup task on Windows user logon",
      audit_title: "System & Hardware Diagnostic",
      audit_sub: "Environment validation before link enablement",
      audit_loading: "Loading system diagnostic data...",
      components_title: "Components & System Setup",
      components_sub: "Select components to update or click [Apply Selected]",
      name_gsp: "Enable GSP (EnableGpuFirmware=1)",
      name_drv: "Install PCIe Drivers & Defender Exclusion",
      name_efi: "EFI Unlock + BIOS Boot Entry (40HX Only)",
      name_task: "Auto-Unlock PCIe at Windows Logon",
      name_fast: "Power: Disable Fast Startup",
      name_aspm: "Power: Disable PCIe ASPM",
      name_perf: "Power: Enable High Performance Profile",
      name_defoff: "Disable Defender Real-time Protection",
      desc_gsp: "Mandatory to prevent Code 43 error after PCIe unlock",
      desc_drv: "Deploy kernel MMIO driver & configure Defender security exclusions",
      desc_efi: "Deploy EFI payload to ESP partition (UEFI boot mode required)",
      desc_task: "Register SYSTEM scheduled task & registry fallback Run key",
      desc_fast: "Prevent hybrid sleep from bypassing UEFI boot sequence",
      desc_aspm: "Prevent PCIe bus from dropping to Gen1 x1 power-saving states",
      desc_perf: "Maintain sustained PCIe bus throughput with High Performance profile",
      desc_defoff: "Required only if third-party AV blocks temporary driver injection",
      btn_install_selected: "Apply Selected Components",
      policy_title: "Driver Policy & Stage 2 Recovery",
      policy_sub: "Configure post-unlock driver behavior & fallback retrain",
      strat_0_name: "Clean Exit (Recommended for Gaming)",
      strat_0_desc: "Unloads BYOVD kernel driver immediately after retrain. Fully clean & Vanguard safe.",
      strat_1_name: "Auto Retry on Failure",
      strat_1_desc: "Automatically attempts retrain sequence if link speed falls short.",
      strat_2_name: "Resident Guard (Continuous Monitoring)",
      strat_2_desc: "Keeps driver loaded; polls link speed every 60s and re-injects if downgraded.",
      desc_autohard: "Auto-trigger Stage 2 (Link Disable + PnP Reset) if standard retrain fails",
      retry_count_label: "Retry Count:",
      retry_count_unit: "times",
      retry_interval_label: "Interval:",
      retry_interval_unit: "minutes",
      btn_save_policy: "Save Policy Configuration",
      terminal_title: "Real-Time Diagnostic & Event Log",
      log_lines: "log lines",
      btn_clear_log: "Clear",
      btn_copy_log: "Copy",
      safety_title: "Game & Anti-Cheat Safe (Riot Vanguard, EasyAntiCheat, BattlEye)",
      safety_desc: "v3.0 architecture requires no VBIOS flashing, no Test Signing, and unloads kernel drivers after negotiation. 100% OS integrity.",
      btn_riot_title: "1-Click Riot Games Optimization",
      btn_riot_sub: "Purge BYOVD drivers, enable MSHybrid CASO for LoL & Valorant",
      riot_panel_title: "Riot Games & Vanguard Compatibility",
      riot_panel_sub: "Zero driver reinstall, high-performance LoL/TFT & Valorant tuning",
      riot_lol_detail: "MSHybrid CASO + Auto Borderless Windowed (WindowMode=2) for headless display.",
      riot_valorant_detail: "Purge unlock drivers, enforce Testsigning OFF compliant with Riot Vanguard.",
      riot_rec_title: "RECOMMENDED ACTION:",
      btn_riot_opt_inner: "⚡ 1-Click Optimize",
      btn_riot_sign_inner: "🔐 Sign EFI (Valorant Win 11)",
      btn_riot_guide_inner: "📖 BIOS Setup Guide",
      btn_launch_app: "Launch App",
      modal_bios_title: "BIOS Key Enrollment & Tensor Core Guide",
      modal_bios_warning: "Take a photo of this screen with your phone before restarting your computer!",
      modal_cert_locs: "Certificate File Locations (CMP40HX_Key.cer):",
      modal_btn_close: "Close",
      modal_btn_reboot: "Reboot into BIOS Setup Now"
    }
  };

  let currentLang = 'vi';
  let logHistory = [];
  let currentFilter = 'all';

  // --- DOM Elements ---
  const elStreamIndicator = document.getElementById('streamIndicator');
  const elStreamStatusText = document.getElementById('streamStatusText');
  const elBtnLangToggle = document.getElementById('btnLangToggle');
  const elBtnRefresh = document.getElementById('btnRefresh');
  const elAuditList = document.getElementById('auditList');
  const elAuditSummaryChip = document.getElementById('auditSummaryChip');
  const elLanePinsGrid = document.getElementById('lanePinsGrid');
  const elLaneSummaryText = document.getElementById('laneSummaryText');
  const elCurrentThroughput = document.getElementById('currentThroughput');
  const elGaugeBarFill = document.getElementById('gaugeBarFill');
  const elGpuDetectedBadge = document.getElementById('gpuDetectedBadge');
  const elSpecBusId = document.getElementById('specBusId');
  const elSpecGsp = document.getElementById('specGsp');
  const elSpecAnticheat = document.getElementById('specAnticheat');
  const elSpecDriverStrategy = document.getElementById('specDriverStrategy');

  const elBtnUnlockNow = document.getElementById('btnUnlockNow');
  const elBtnFullInstall = document.getElementById('btnFullInstall');
  const elBtnGen2AndTask = document.getElementById('btnGen2AndTask');
  const elBtnRiotOptimize = document.getElementById('btnRiotOptimize');
  const elBtnRiotOptimizeInner = document.getElementById('btnRiotOptimizeInner');
  const elBtnRiotSignEfi = document.getElementById('btnRiotSignEfi');
  const elBtnRiotGuide = document.getElementById('btnRiotGuide');
  const elBtnLaunchUnlockRiotExe = document.getElementById('btnLaunchUnlockRiotExe');
  const elBtnInstallSelected = document.getElementById('btnInstallSelected');
  const elBtnSavePolicy = document.getElementById('btnSavePolicy');

  const elRiotSummaryChip = document.getElementById('riotSummaryChip');
  const elRiotLolStatus = document.getElementById('riotLolStatus');
  const elRiotLolDetail = document.getElementById('riotLolDetail');
  const elRiotValorantStatus = document.getElementById('riotValorantStatus');
  const elRiotValorantDetail = document.getElementById('riotValorantDetail');
  const elRiotRecBanner = document.getElementById('riotRecBanner');
  const elRiotRecText = document.getElementById('riotRecText');

  const elModalBiosGuide = document.getElementById('modalBiosGuide');
  const elBtnModalClose = document.getElementById('btnModalClose');
  const elBtnModalDismiss = document.getElementById('btnModalDismiss');
  const elBtnModalRebootBios = document.getElementById('btnModalRebootBios');

  const elCkGsp = document.getElementById('ckGsp');
  const elCkDrv = document.getElementById('ckDrv');
  const elCkEfi = document.getElementById('ckEfi');
  const elCkTask = document.getElementById('ckTask');
  const elCkFast = document.getElementById('ckFast');
  const elCkAspm = document.getElementById('ckAspm');
  const elCkPerf = document.getElementById('ckPerf');
  const elCkDefOff = document.getElementById('ckDefOff');

  const elCkAutoHard = document.getElementById('ckAutoHard');
  const elNeRetryCnt = document.getElementById('neRetryCnt');
  const elNeRetryMin = document.getElementById('neRetryMin');

  const elTerminalLogBody = document.getElementById('terminalLogBody');
  const elLogLineCount = document.getElementById('logLineCount');
  const elBtnClearLog = document.getElementById('btnClearLog');
  const elBtnCopyLog = document.getElementById('btnCopyLog');

  // --- 16 Physical Lane Matrix Setup ---
  function initLaneMatrix(activeCount = 16) {
    if (!elLanePinsGrid) return;
    elLanePinsGrid.innerHTML = '';
    for (let i = 1; i <= 16; i++) {
      const pin = document.createElement('div');
      pin.className = 'lane-pin' + (i <= activeCount ? ' active' : '');
      pin.textContent = i;
      pin.title = `PCIe Physical Lane #${i}: ${i <= activeCount ? 'Active (Negotiated)' : 'Inactive'}`;
      elLanePinsGrid.appendChild(pin);
    }
    if (elLaneSummaryText) {
      elLaneSummaryText.textContent = `${activeCount}/16 Lanes Negotiated`;
    }
  }

  // --- Translation Engine ---
  function applyLanguage(lang) {
    currentLang = lang;
    document.querySelectorAll('[data-i18n]').forEach(el => {
      const key = el.getAttribute('data-i18n');
      if (i18n[lang] && i18n[lang][key]) {
        el.textContent = i18n[lang][key];
      }
    });
    document.querySelectorAll('[data-i18n-title]').forEach(el => {
      const key = el.getAttribute('data-i18n-title');
      if (i18n[lang] && i18n[lang][key]) {
        el.setAttribute('title', i18n[lang][key]);
      }
    });
  }

  if (elBtnLangToggle) {
    elBtnLangToggle.addEventListener('click', () => {
      applyLanguage(currentLang === 'vi' ? 'en' : 'vi');
      fetchStatus();
    });
  }

  // --- Log Streaming via SSE ---
  function initLogStream() {
    if (!elStreamIndicator || !elStreamStatusText) return;
    elStreamIndicator.className = 'status-indicator';
    elStreamStatusText.textContent = i18n[currentLang].status_connecting;

    const evtSource = new EventSource('/api/logs/stream');

    evtSource.onopen = () => {
      elStreamIndicator.className = 'status-indicator connected';
      elStreamStatusText.textContent = i18n[currentLang].status_connected;
    };

    evtSource.onmessage = (e) => {
      if (e.data) {
        appendLogLine(e.data);
      }
    };

    evtSource.onerror = () => {
      elStreamIndicator.className = 'status-indicator disconnected';
      elStreamStatusText.textContent = i18n[currentLang].status_disconnected;
      // EventSource will automatically retry in background
    };
  }

  function appendLogLine(rawText) {
    const lines = rawText.split(/\r?\n/).filter(l => l.trim().length > 0);
    lines.forEach(line => {
      const logObj = parseLogLine(line);
      logHistory.push(logObj);
      renderLogLine(logObj);
    });
    if (elLogLineCount) {
      elLogLineCount.textContent = logHistory.length;
    }
  }

  function parseLogLine(line) {
    const now = new Date();
    const timeStr = `[${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}]`;
    
    let category = 'sys';

    if (line.includes('[PCIe]') || line.includes('PCIe') || line.includes('Gen2') || line.includes('Gen1') || line.includes('Lanes')) {
      category = 'pcie';
    } else if (line.includes('[!]') || line.includes('Lỗi') || line.includes('thất bại') || line.includes('Error') || line.includes('Fail') || line.includes('cảnh báo')) {
      category = 'warn';
    } else if (line.includes('✓') || line.includes('thành công') || line.includes('OK') || line.includes('Success')) {
      category = 'ok';
    }

    return { timeStr, raw: line, category };
  }

  function renderLogLine(logObj) {
    if (!elTerminalLogBody) return;
    if (currentFilter !== 'all' && currentFilter !== logObj.category) {
      return;
    }
    const div = document.createElement('div');
    div.className = `log-line ${logObj.category}`;
    div.innerHTML = `<span class="log-time">${logObj.timeStr}</span> ${escapeHtml(logObj.raw)}`;
    elTerminalLogBody.appendChild(div);

    // Auto-scroll to bottom
    elTerminalLogBody.scrollTop = elTerminalLogBody.scrollHeight;
  }

  function escapeHtml(text) {
    const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
    return text.replace(/[&<>"']/g, m => map[m]);
  }

  // Filter Buttons
  document.querySelectorAll('.btn-filter').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.btn-filter').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      currentFilter = btn.getAttribute('data-filter');
      reRenderLogs();
    });
  });

  function reRenderLogs() {
    if (!elTerminalLogBody) return;
    elTerminalLogBody.innerHTML = '';
    logHistory.forEach(logObj => {
      if (currentFilter === 'all' || currentFilter === logObj.category) {
        renderLogLine(logObj);
      }
    });
  }

  // Clear & Copy Log
  if (elBtnClearLog) {
    elBtnClearLog.addEventListener('click', () => {
      logHistory = [];
      if (elTerminalLogBody) elTerminalLogBody.innerHTML = '';
      if (elLogLineCount) elLogLineCount.textContent = '0';
    });
  }

  if (elBtnCopyLog) {
    elBtnCopyLog.addEventListener('click', () => {
      const allText = logHistory.map(l => `${l.timeStr} ${l.raw}`).join('\n');
      navigator.clipboard.writeText(allText).then(() => {
        appendLogLine(currentLang === 'vi' ? "[SYSTEM] Đã sao chép toàn bộ nhật ký vào clipboard." : "[SYSTEM] Copied all logs to clipboard.");
      });
    });
  }

  // --- Fetch System Status ---
  async function fetchStatus() {
    try {
      if (elAuditSummaryChip) elAuditSummaryChip.textContent = currentLang === 'vi' ? "Đang quét..." : "Scanning...";
      const res = await fetch('/api/status');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      renderStatus(data);
    } catch (err) {
      console.warn("API status fallback:", err);
      // Fallback mock rendering for dev preview if server is not yet returning JSON
      renderStatus(getMockStatus());
    }
    fetchRiotStatus();
  }

  function renderStatus(data) {
    // 1. Audit List
    if (elAuditList) {
      elAuditList.innerHTML = '';
      let warningCount = 0;

      (data.items || []).forEach(item => {
        const row = document.createElement('div');
        row.className = 'audit-item';
        if (!item.ok) warningCount++;

        row.innerHTML = `
          <div class="audit-left">
            <div class="audit-icon ${item.ok ? 'ok' : 'warn'}">${item.ok ? '✓' : '!'}</div>
            <span class="audit-name">${escapeHtml(item.name)}</span>
          </div>
          <div class="audit-note">${escapeHtml(item.note)}</div>
        `;
        elAuditList.appendChild(row);
      });

      if (elAuditSummaryChip) {
        elAuditSummaryChip.textContent = warningCount === 0 
          ? (currentLang === 'vi' ? '✓ Môi trường tối ưu' : '✓ System Ready')
          : (currentLang === 'vi' ? `⚠ ${warningCount} cảnh báo cần xử lý` : `⚠ ${warningCount} warnings`);
        
        elAuditSummaryChip.style.borderColor = warningCount === 0 ? 'var(--accent-phosphor)' : 'var(--accent-amber)';
        elAuditSummaryChip.style.color = warningCount === 0 ? 'var(--accent-phosphor)' : 'var(--accent-amber)';
      }
    }

    // 2. Hardware specs & Throughput
    if (elGpuDetectedBadge) {
      if (data.gpuDetected) {
        elGpuDetectedBadge.textContent = data.gpuName || "CMP 40HX (TU106)";
        elGpuDetectedBadge.className = "badge badge-emerald mono";
      } else {
        elGpuDetectedBadge.textContent = currentLang === 'vi' ? "Chưa phát hiện GPU" : "No GPU Detected";
        elGpuDetectedBadge.className = "badge mono";
        elGpuDetectedBadge.style.color = "var(--accent-crimson)";
        elGpuDetectedBadge.style.borderColor = "var(--accent-crimson)";
      }
    }

    if (elSpecBusId && data.pciBusId) {
      elSpecBusId.textContent = data.pciBusId;
    }

    if (elSpecGsp) {
      if (data.gspActive) {
        elSpecGsp.textContent = currentLang === 'vi' ? "Đã bật (GSP-RM Mode)" : "Enabled (GSP Mode)";
        elSpecGsp.className = "spec-value highlight-cyan";
      } else {
        elSpecGsp.textContent = currentLang === 'vi' ? "Chưa bật (Nguy cơ lỗi 43)" : "Disabled (Risk Code 43)";
        elSpecGsp.className = "spec-value";
        elSpecGsp.style.color = "var(--accent-amber)";
      }
    }

    // Smart default pre-selections
    if (data.recommendations) {
      if (elCkGsp && typeof data.recommendations.gsp !== 'undefined') elCkGsp.checked = data.recommendations.gsp;
      if (elCkDrv && typeof data.recommendations.drv !== 'undefined') elCkDrv.checked = data.recommendations.drv;
      if (elCkEfi && typeof data.recommendations.efi !== 'undefined') elCkEfi.checked = data.recommendations.efi;
      if (elCkTask && typeof data.recommendations.task !== 'undefined') elCkTask.checked = data.recommendations.task;
      if (elCkFast && typeof data.recommendations.fast !== 'undefined') elCkFast.checked = data.recommendations.fast;
      if (elCkAspm && typeof data.recommendations.aspm !== 'undefined') elCkAspm.checked = data.recommendations.aspm;
      if (elCkPerf && typeof data.recommendations.perf !== 'undefined') elCkPerf.checked = data.recommendations.perf;
      if (elCkDefOff && typeof data.recommendations.defoff !== 'undefined') elCkDefOff.checked = data.recommendations.defoff;
    }

    // Driver Strategy Radios
    const stratVal = data.driverStrategy || 0;
    const stratRadio = document.querySelector(`input[name="driverStrategy"][value="${stratVal}"]`);
    if (stratRadio) stratRadio.checked = true;

    if (elSpecDriverStrategy) {
      const stratNames = [
        (currentLang === 'vi' ? 'Dùng xong gỡ ngay (Clean)' : 'Clean Exit'),
        (currentLang === 'vi' ? 'Tự động thử lại khi lỗi' : 'Auto Retry'),
        (currentLang === 'vi' ? 'Thường trú (Resident Guard)' : 'Resident Guard')
      ];
      elSpecDriverStrategy.textContent = stratNames[stratVal] || stratNames[0];
    }

    if (elCkAutoHard && typeof data.autoHard !== 'undefined') elCkAutoHard.checked = data.autoHard;
    if (elNeRetryCnt && typeof data.retryCount !== 'undefined') elNeRetryCnt.value = data.retryCount;
    if (elNeRetryMin && typeof data.retryInterval !== 'undefined') elNeRetryMin.value = data.retryInterval;

    // PCIe Link status & gauge
    const isGen2 = data.isGen2 || false;
    if (isGen2) {
      initLaneMatrix(16);
      if (elCurrentThroughput) elCurrentThroughput.innerHTML = `~6.4 <span class="unit">GB/s</span>`;
      if (elGaugeBarFill) elGaugeBarFill.style.width = '100%';
    } else {
      initLaneMatrix(1);
      if (elCurrentThroughput) elCurrentThroughput.innerHTML = `250 <span class="unit">MB/s</span>`;
      if (elGaugeBarFill) elGaugeBarFill.style.width = '4%';
    }
  }

  // --- API Action Triggers ---
  async function sendAction(url, payload = null, btn = null) {
    if (btn) {
      btn.disabled = true;
      btn.classList.add('loading');
    }
    try {
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload ? JSON.stringify(payload) : null
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.message || `HTTP ${res.status}`);
      }
    } catch (err) {
      appendLogLine(`[!] Thao tác thất bại: ${err.message}`);
    } finally {
      if (btn) {
        btn.disabled = false;
        btn.classList.remove('loading');
      }
      fetchStatus();
    }
  }

  // 1. Mở khoá Gen2 ngay
  if (elBtnUnlockNow) {
    elBtnUnlockNow.addEventListener('click', () => {
      appendLogLine("[PCIe] Bắt đầu kích hoạt mở khóa Gen2 ngay lập tức...");
      sendAction('/api/unlock-now', null, elBtnUnlockNow);
    });
  }

  // 2. Cài đặt toàn bộ 1-chạm
  if (elBtnFullInstall) {
    elBtnFullInstall.addEventListener('click', () => {
      appendLogLine("[INSTALL] Bắt đầu quy trình triển khai toàn diện một chạm...");
      sendAction('/api/full-install', null, elBtnFullInstall);
    });
  }

  // 3. Mở khoá + Cài tự khởi động
  if (elBtnGen2AndTask) {
    elBtnGen2AndTask.addEventListener('click', () => {
      appendLogLine("[PCIe] Mở khoá và đăng ký tác vụ tự khởi động...");
      sendAction('/api/gen2-and-task', null, elBtnGen2AndTask);
    });
  }

  // 4. Cài đặt mục đã chọn
  if (elBtnInstallSelected) {
    elBtnInstallSelected.addEventListener('click', () => {
      const sel = {
        gsp: elCkGsp ? elCkGsp.checked : true,
        drv: elCkDrv ? elCkDrv.checked : true,
        efi: elCkEfi ? elCkEfi.checked : false,
        task: elCkTask ? elCkTask.checked : true,
        fast: elCkFast ? elCkFast.checked : false,
        aspm: elCkAspm ? elCkAspm.checked : false,
        perf: elCkPerf ? elCkPerf.checked : false,
        defoff: elCkDefOff ? elCkDefOff.checked : false
      };
      appendLogLine("[INSTALL] Áp dụng các mục đã chọn...");
      sendAction('/api/install', sel, elBtnInstallSelected);
    });
  }

  // 5. Lưu cấu hình chính sách
  if (elBtnSavePolicy) {
    elBtnSavePolicy.addEventListener('click', () => {
      const stratEl = document.querySelector('input[name="driverStrategy"]:checked');
      const strat = stratEl ? parseInt(stratEl.value, 10) : 0;
      const payload = {
        strategy: strat,
        autoHard: elCkAutoHard && elCkAutoHard.checked ? 1 : 0,
        retryCount: elNeRetryCnt ? (parseInt(elNeRetryCnt.value, 10) || 3) : 3,
        retryInterval: elNeRetryMin ? (parseInt(elNeRetryMin.value, 10) || 5) : 5
      };
      appendLogLine(`[CONFIG] Lưu cấu hình chính sách: Chiến lược=${strat}, Stage2=${payload.autoHard}...`);
      sendAction('/api/save-policy', payload, elBtnSavePolicy);
    });
  }

  if (elBtnRefresh) {
    elBtnRefresh.addEventListener('click', () => {
      appendLogLine("[SYS] Quét lại môi trường hệ thống...");
      fetchStatus();
    });
  }

  // --- Fetch Riot Vanguard & MSHybrid Status ---
  async function fetchRiotStatus() {
    try {
      const res = await fetch('/api/riot/status');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      renderRiotStatus(data);
    } catch (err) {
      console.warn("Riot status API fallback:", err);
    }
  }

  function renderRiotStatus(data) {
    if (!data) return;
    if (elRiotSummaryChip) {
      if (data.canPlayValorant && data.canPlayLeagueOfLegends) {
        elRiotSummaryChip.textContent = currentLang === 'vi' ? "Hoàn Hảo (100%)" : "Perfect (100%)";
        elRiotSummaryChip.className = "riot-summary-chip chip-ok";
      } else if (data.canPlayLeagueOfLegends) {
        elRiotSummaryChip.textContent = currentLang === 'vi' ? "Sẵn sàng (LMHT)" : "Ready (LoL)";
        elRiotSummaryChip.className = "riot-summary-chip chip-ready";
      } else {
        elRiotSummaryChip.textContent = currentLang === 'vi' ? "Cần thiết lập" : "Setup Needed";
        elRiotSummaryChip.className = "riot-summary-chip chip-warn";
      }
    }

    if (elRiotLolStatus) {
      elRiotLolStatus.textContent = currentLang === 'vi' ? "✓ TƯƠNG THÍCH 100%" : "✓ 100% COMPATIBLE";
    }
    if (elRiotLolDetail && data.lolDesc) {
      elRiotLolDetail.textContent = data.lolDesc;
    }

    if (elRiotValorantStatus) {
      if (data.canPlayValorant) {
        elRiotValorantStatus.textContent = currentLang === 'vi' ? "✓ HOÀN TOÀN TƯƠNG THÍCH" : "✓ FULLY COMPATIBLE";
        elRiotValorantStatus.style.color = "var(--color-success)";
      } else {
        elRiotValorantStatus.textContent = currentLang === 'vi' ? "⚠️ CẦN THIẾT LẬP BIOS" : "⚠️ BIOS SETUP NEEDED";
        elRiotValorantStatus.style.color = "var(--color-amber)";
      }
    }
    if (elRiotValorantDetail && data.valorantDesc) {
      elRiotValorantDetail.textContent = data.valorantDesc;
    }

    if (elRiotRecText && data.recommendation) {
      elRiotRecText.textContent = data.recommendation;
    }

    if (elBtnRiotSignEfi) {
      const is40HX = data.model === "CMP 40HX" || data.hasTensorCore;
      elBtnRiotSignEfi.style.display = is40HX ? "inline-flex" : "none";
    }
  }

  // --- Modal Helpers ---
  function openBiosModal() {
    if (elModalBiosGuide) {
      elModalBiosGuide.removeAttribute('hidden');
      elModalBiosGuide.classList.add('active');
    }
  }

  function closeBiosModal() {
    if (elModalBiosGuide) {
      elModalBiosGuide.setAttribute('hidden', '');
      elModalBiosGuide.classList.remove('active');
    }
  }

  if (elBtnRiotGuide) {
    elBtnRiotGuide.addEventListener('click', openBiosModal);
  }
  if (elBtnModalClose) {
    elBtnModalClose.addEventListener('click', closeBiosModal);
  }
  if (elBtnModalDismiss) {
    elBtnModalDismiss.addEventListener('click', closeBiosModal);
  }
  if (elModalBiosGuide) {
    elModalBiosGuide.addEventListener('click', (e) => {
      if (e.target === elModalBiosGuide) closeBiosModal();
    });
  }
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && elModalBiosGuide && !elModalBiosGuide.hidden) {
      closeBiosModal();
    }
  });

  if (elBtnModalRebootBios) {
    elBtnModalRebootBios.addEventListener('click', () => {
      appendLogLine(currentLang === 'vi' ? "[BIOS] Đang khởi động lại vào BIOS Setup..." : "[BIOS] Rebooting into BIOS Firmware Setup...");
      sendAction('/api/riot/reboot-bios', null, elBtnModalRebootBios);
    });
  }

  // 6. Riot Games 1-Click Optimize
  if (elBtnRiotOptimize) {
    elBtnRiotOptimize.addEventListener('click', () => {
      appendLogLine("[RIOT] Bắt đầu tối ưu hóa hệ thống cho Riot Games & dọn dẹp driver...");
      sendAction('/api/riot/optimize', null, elBtnRiotOptimize);
    });
  }
  if (elBtnRiotOptimizeInner) {
    elBtnRiotOptimizeInner.addEventListener('click', () => {
      appendLogLine("[RIOT] Bắt đầu tối ưu hóa hệ thống cho Riot Games & dọn dẹp driver...");
      sendAction('/api/riot/optimize', null, elBtnRiotOptimizeInner);
    });
  }

  // 7. Riot EFI Signing
  if (elBtnRiotSignEfi) {
    elBtnRiotSignEfi.addEventListener('click', async () => {
      appendLogLine("[UEFI] Bắt đầu tạo chứng chỉ cá nhân và ký Authenticode cho 40HXUNLK.EFI...");
      await sendAction('/api/riot/sign-efi', null, elBtnRiotSignEfi);
      openBiosModal();
    });
  }

  // 8. Launch Native GUI
  if (elBtnLaunchUnlockRiotExe) {
    elBtnLaunchUnlockRiotExe.addEventListener('click', () => {
      appendLogLine("[EXE] Khởi chạy ứng dụng chuyên sâu UnlockRiotGame.exe...");
      sendAction('/api/riot/launch-gui', null, elBtnLaunchUnlockRiotExe);
    });
  }

  // --- Fallback Mock Data ---
  function getMockStatus() {
    return {
      gpuDetected: true,
      gpuName: "CMP 40HX (Turing TU106)",
      pciBusId: "VEN_10DE & DEV_1F0B",
      gspActive: true,
      isGen2: true,
      driverStrategy: 0,
      autoHard: true,
      retryCount: 3,
      retryInterval: 5,
      items: [
        { name: "Chế độ Boot", ok: true, note: "UEFI (OK)" },
        { name: "Secure Boot", ok: true, note: "Đã Tắt (OK)" },
        { name: "Card đồ hoạ", ok: true, note: "Đã phát hiện NVIDIA CMP 40HX" },
        { name: "GSP (EnableGpuFirmware)", ok: true, note: "Đã bật (OK)" },
        { name: "ESP EFI Mở khoá", ok: true, note: "\\EFI\\40HX\\40HXUNLK.EFI đã nạp" },
        { name: "Mục khởi động BIOS", ok: true, note: "Tồn tại và nằm đầu tiên (displayorder)" },
        { name: "Tác vụ tự khởi động", ok: true, note: "Trạng thái: Ready" },
        { name: "Driver PCIe", ok: true, note: "Đã cài; Tự dọn dẹp sau khi chạy (Game safe)" },
        { name: "Loại trừ Windows Defender", ok: true, note: "Đã thêm loại trừ cho file .sys" },
        { name: "Khởi động nhanh (Fast Startup)", ok: true, note: "Đã Tắt (OK)" },
        { name: "Tiết kiệm điện PCIe (ASPM)", ok: true, note: "Đã Tắt (OK)" }
      ],
      recommendations: {
        gsp: false,
        drv: false,
        efi: false,
        task: false,
        fast: false,
        aspm: false,
        perf: false,
        defoff: false
      }
    };
  }

  // --- Initialization ---
  initLaneMatrix(16);
  initLogStream();
  fetchStatus();

})();
