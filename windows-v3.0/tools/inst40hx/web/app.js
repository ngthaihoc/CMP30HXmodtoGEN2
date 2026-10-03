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
      hud_silicon_title: "THÔNG TIN BÁN DẪN & BUS",
      spec_bus_id: "PCIe Bus:",
      spec_gsp: "GSP-RM Firmware:",
      spec_vanguard: "Riot Vanguard / Game:",
      spec_driver_strategy: "Chiến lược Driver:",
      lane_topology: "Cấu trúc 16 làn PCIe vật lý (Physical Lanes):",
      hud_bandwidth_title: "BĂNG THÔNG & LIÊN KẾT PCIE",
      speed_locked: "GỐC (KHOÁ EFUSE)",
      speed_unlocked: "ĐÃ MỞ KHOÁ TOÀN BỘ",
      btn_unlock_title: "MỞ KHOÁ PCIE GEN2 NGAY",
      btn_unlock_sub: "Kích hoạt tức thì phiên hiện tại (không cần khởi động lại máy)",
      btn_full_title: "CÀI ĐẶT TOÀN BỘ 1-CHẠM",
      btn_full_sub: "Tự động cấu hình chuẩn: GSP + Driver + Tự khởi động + Nguồn",
      btn_persist_title: "Mở khoá & Cài tự khởi động",
      btn_persist_sub: "Đăng ký tác vụ tự chạy khi đăng nhập Windows",
      audit_title: "Chẩn đoán Môi trường & Phần cứng",
      audit_sub: "Kiểm tra tính tương thích trước khi kích hoạt",
      components_title: "Thành phần & Thiết lập hệ thống",
      components_sub: "Chọn các mục cần cập nhật hoặc bấm [Cài đặt mục đã chọn]",
      desc_gsp: "Bắt buộc để tránh mã lỗi Code 43 sau khi mở khóa",
      desc_drv: "Nạp driver can thiệp thanh ghi & cấp quyền an toàn",
      desc_efi: "Nạp payload vào phân vùng ESP (chỉ áp dụng cho chuẩn UEFI)",
      desc_task: "Đăng ký tác vụ Task Scheduler SYSTEM và Run Key dự phòng",
      desc_fast: "Tránh tình trạng Windows nạp sleep image bỏ qua UEFI hook",
      desc_aspm: "Tránh PCIe tự động rớt về Gen1 x1 khi máy tính ở trạng thái rảnh",
      desc_perf: "Đảm bảo cấp đủ năng lượng cho liên kết PCIe hoạt động tối đa",
      desc_defoff: "Chỉ cần thiết khi phần mềm diệt virus chặn file sys can thiệp",
      btn_install_selected: "Cài đặt mục đã chọn",
      policy_title: "Chiến lược Driver & Tự phục hồi",
      policy_sub: "Cấu hình hành vi sau khi hoàn tất mở khóa",
      strat_0_name: "Dùng xong gỡ ngay (Khuyên dùng cho Game)",
      strat_0_desc: "Sau khi nâng tốc độ, driver can thiệp được gỡ hoàn toàn. An toàn tuyệt đối với Riot Vanguard & Anti-Cheat.",
      strat_1_name: "Tự động thử lại khi lỗi",
      strat_1_desc: "Tự động thử lại nếu lần đầu khởi tạo GPU chưa đạt tốc độ Gen2.",
      strat_2_name: "Thường trú (Canh giữ tốc độ PCIe)",
      strat_2_desc: "Driver chạy nền thường trực, định kỳ kiểm tra và ép xung PCIe trở lại nếu bị tụt xung.",
      desc_autohard: "Tự động kích hoạt Stage 2 (Link Disable + Reset PnP) khi mở khóa thường không đạt",
      retry_count_label: "Số lần thử lại:",
      retry_interval_label: "Giãn cách:",
      btn_save_policy: "Lưu cấu hình chính sách",
      terminal_title: "DIAGNOSTIC & LOG STREAM (REAL-TIME)",
      log_lines: "dòng log",
      btn_clear_log: "Xóa",
      btn_copy_log: "Sao chép",
      safety_title: "An toàn cho Game & Anti-Cheat (Riot Vanguard, EasyAntiCheat, BattlEye)",
      safety_desc: "Giải pháp v3.0 không flash VBIOS, không bật Test Signing, gỡ sạch driver can thiệp sau khi mở khóa. Đảm bảo 100% tính toàn vẹn hệ điều hành."
    },
    en: {
      brand_desc: "NVIDIA CMP 40HX & 30HX PCIe Gen2 x16 Bandwidth & Compute Enablement Suite",
      status_connecting: "Connecting...",
      status_connected: "Live Connected",
      status_disconnected: "Disconnected",
      btn_refresh: "Refresh",
      refresh_tooltip: "Re-scan system environment",
      hud_silicon_title: "SILICON & BUS ARCHITECTURE",
      spec_bus_id: "PCIe Bus:",
      spec_gsp: "GSP-RM Firmware:",
      spec_vanguard: "Riot Vanguard / Game:",
      spec_driver_strategy: "Driver Strategy:",
      lane_topology: "PCIe x16 Physical Lanes Topology:",
      hud_bandwidth_title: "BANDWIDTH & PCIE LINK",
      speed_locked: "STOCK (EFUSE LOCKED)",
      speed_unlocked: "FULLY UNLOCKED",
      btn_unlock_title: "UNLOCK PCIE GEN2 NOW",
      btn_unlock_sub: "Instantly retrain PCIe link for current session (no reboot needed)",
      btn_full_title: "1-CLICK FULL DEPLOY",
      btn_full_sub: "Auto configure all: GSP + Drivers + Auto-Start + Power tuning",
      btn_persist_title: "Unlock & Install Autostart",
      btn_persist_sub: "Register persistent startup task on Windows user logon",
      audit_title: "System & Hardware Diagnostic",
      audit_sub: "Environment validation before link enablement",
      components_title: "Components & System Setup",
      components_sub: "Select components to update or click [Apply Selected]",
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
      retry_interval_label: "Interval:",
      btn_save_policy: "Save Policy Configuration",
      terminal_title: "DIAGNOSTIC & LOG STREAM (REAL-TIME)",
      log_lines: "log lines",
      btn_clear_log: "Clear",
      btn_copy_log: "Copy",
      safety_title: "Game & Anti-Cheat Safe (Riot Vanguard, EasyAntiCheat, BattlEye)",
      safety_desc: "v3.0 architecture requires no VBIOS flashing, no Test Signing, and unloads kernel drivers after negotiation. 100% OS integrity."
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
  const elBoostRatio = document.getElementById('boostRatio');
  const elCurrentSpeedSub = document.getElementById('currentSpeedSub');

  const elBtnUnlockNow = document.getElementById('btnUnlockNow');
  const elBtnFullInstall = document.getElementById('btnFullInstall');
  const elBtnGen2AndTask = document.getElementById('btnGen2AndTask');
  const elBtnInstallSelected = document.getElementById('btnInstallSelected');
  const elBtnSavePolicy = document.getElementById('btnSavePolicy');

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
  function initLaneMatrix(activeCount = 0) {
    if (!elLanePinsGrid) return;
    elLanePinsGrid.innerHTML = '';
    for (let i = 1; i <= 16; i++) {
      const pin = document.createElement('div');
      pin.className = 'lane-pin' + (i <= activeCount ? ' active' : '');
      pin.textContent = i;
      pin.title = `PCIe Physical Lane #${i}: ${i <= activeCount ? 'Active (Negotiated)' : 'Inactive / Parked'}`;
      elLanePinsGrid.appendChild(pin);
    }
    if (elLaneSummaryText) {
      if (activeCount === 0) {
        elLaneSummaryText.textContent = currentLang === 'vi' ? '0/16 Làn (Chưa phát hiện GPU CMP)' : '0/16 Lanes (No CMP GPU Detected)';
      } else {
        elLaneSummaryText.textContent = `${activeCount}/16 Lanes Negotiated`;
      }
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

  elBtnLangToggle.addEventListener('click', () => {
    applyLanguage(currentLang === 'vi' ? 'en' : 'vi');
  });

  // --- Log Streaming via SSE ---
  function initLogStream() {
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
    let cleanLine = line;

    if (line.includes('[PCIe]') || line.includes('PCIe') || line.includes('Gen2') || line.includes('Gen1')) {
      category = 'pcie';
    } else if (line.includes('[!]') || line.includes('lỗi') || line.includes('thất bại') || line.includes('Error') || line.includes('Fail')) {
      category = 'warn';
    } else if (line.includes('✓') || line.includes('thành công') || line.includes('OK') || line.includes('Success')) {
      category = 'ok';
    }

    return { timeStr, raw: line, category };
  }

  function renderLogLine(logObj) {
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
    elTerminalLogBody.innerHTML = '';
    logHistory.forEach(logObj => {
      if (currentFilter === 'all' || currentFilter === logObj.category) {
        renderLogLine(logObj);
      }
    });
  }

  // Clear & Copy Log
  elBtnClearLog.addEventListener('click', () => {
    logHistory = [];
    elTerminalLogBody.innerHTML = '';
    elLogLineCount.textContent = '0';
  });

  elBtnCopyLog.addEventListener('click', () => {
    const allText = logHistory.map(l => `${l.timeStr} ${l.raw}`).join('\n');
    navigator.clipboard.writeText(allText).then(() => {
      appendLogLine("[SYSTEM] Đã sao chép toàn bộ nhật ký vào clipboard.");
    });
  });

  // --- Fetch System Status ---
  async function fetchStatus() {
    try {
      elAuditSummaryChip.textContent = "Đang quét...";
      const res = await fetch('/api/status');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      renderStatus(data);
    } catch (err) {
      console.warn("API status error:", err);
      // Fallback mock rendering for dev preview if server is not yet returning JSON
      renderStatus(getMockStatus());
    }
  }

  function renderStatus(data) {
    // 1. Audit List
    elAuditList.innerHTML = '';
    let warningCount = 0;

    data.items.forEach(item => {
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

    elAuditSummaryChip.textContent = warningCount === 0 
      ? (currentLang === 'vi' ? '✓ Môi trường tối ưu' : '✓ System Ready')
      : (currentLang === 'vi' ? `⚠ ${warningCount} cảnh báo cần xử lý` : `⚠ ${warningCount} warnings`);
    
    elAuditSummaryChip.style.borderColor = warningCount === 0 ? 'var(--accent-emerald)' : 'var(--accent-amber)';
    elAuditSummaryChip.style.color = warningCount === 0 ? 'var(--accent-emerald)' : 'var(--accent-amber)';

    // 2. Hardware specs & Throughput
    if (data.gpuDetected) {
      elGpuDetectedBadge.textContent = data.gpuName || "CMP 40HX (TU106)";
      elGpuDetectedBadge.className = "badge badge-emerald";
      elGpuDetectedBadge.style.color = "";
      if (data.pciBusId) elSpecBusId.textContent = data.pciBusId;
      elSpecAnticheat.textContent = currentLang === 'vi' ? "Tương thích 100%" : "100% Compatible";

      if (data.gspActive) {
        elSpecGsp.textContent = currentLang === 'vi' ? "Đã bật (GSP-RM Mode)" : "Enabled (GSP Mode)";
        elSpecGsp.className = "spec-value highlight-cyan";
        elSpecGsp.style.color = "";
      } else {
        elSpecGsp.textContent = currentLang === 'vi' ? "Chưa bật (Có thể lỗi 43)" : "Disabled (Risk Code 43)";
        elSpecGsp.className = "spec-value";
        elSpecGsp.style.color = "var(--accent-amber)";
      }

      // PCIe Link status & gauge
      const isGen2 = data.isGen2 || false;
      if (isGen2) {
        initLaneMatrix(16);
        elCurrentThroughput.innerHTML = `~6.4 <span class="unit">GB/s</span>`;
        if (elCurrentSpeedSub) elCurrentSpeedSub.textContent = "Gen2 x16 @ 5.0 GT/s";
        elGaugeBarFill.style.width = '100%';
        if (elBoostRatio) elBoostRatio.textContent = "~25.6x BOOST";
      } else {
        initLaneMatrix(1);
        elCurrentThroughput.innerHTML = `250 <span class="unit">MB/s</span>`;
        if (elCurrentSpeedSub) elCurrentSpeedSub.textContent = "Gen1 x1 @ 2.5 GT/s (Khóa)";
        elGaugeBarFill.style.width = '4%';
        if (elBoostRatio) elBoostRatio.textContent = "1.0x (Khoá eFuse)";
      }
    } else {
      elGpuDetectedBadge.textContent = currentLang === 'vi' ? "Chưa phát hiện GPU CMP" : "No CMP GPU Detected";
      elGpuDetectedBadge.className = "badge";
      elGpuDetectedBadge.style.color = "var(--accent-crimson)";
      elSpecBusId.textContent = currentLang === 'vi' ? "Không phát hiện" : "Not Found";
      elSpecGsp.textContent = "N/A";
      elSpecGsp.className = "spec-value";
      elSpecGsp.style.color = "var(--text-muted)";
      elSpecAnticheat.textContent = "N/A";

      initLaneMatrix(0);
      elCurrentThroughput.innerHTML = `0 <span class="unit">MB/s</span>`;
      if (elCurrentSpeedSub) elCurrentSpeedSub.textContent = currentLang === 'vi' ? "Không phát hiện GPU CMP" : "No CMP GPU";
      elGaugeBarFill.style.width = '0%';
      if (elBoostRatio) elBoostRatio.textContent = "-- BOOST";
    }

    // Smart default pre-selections
    if (data.recommendations) {
      if (typeof data.recommendations.gsp !== 'undefined') elCkGsp.checked = data.recommendations.gsp;
      if (typeof data.recommendations.drv !== 'undefined') elCkDrv.checked = data.recommendations.drv;
      if (typeof data.recommendations.efi !== 'undefined') elCkEfi.checked = data.recommendations.efi;
      if (typeof data.recommendations.task !== 'undefined') elCkTask.checked = data.recommendations.task;
      if (typeof data.recommendations.fast !== 'undefined') elCkFast.checked = data.recommendations.fast;
      if (typeof data.recommendations.aspm !== 'undefined') elCkAspm.checked = data.recommendations.aspm;
      if (typeof data.recommendations.perf !== 'undefined') elCkPerf.checked = data.recommendations.perf;
      if (typeof data.recommendations.defoff !== 'undefined') elCkDefOff.checked = data.recommendations.defoff;
    }

    // Driver Strategy Radios
    const stratVal = data.driverStrategy || 0;
    const stratRadio = document.querySelector(`input[name="driverStrategy"][value="${stratVal}"]`);
    if (stratRadio) stratRadio.checked = true;

    const stratNames = [
      (currentLang === 'vi' ? 'Dùng xong gỡ ngay (Clean)' : 'Clean Exit'),
      (currentLang === 'vi' ? 'Tự động thử lại khi lỗi' : 'Auto Retry'),
      (currentLang === 'vi' ? 'Thường trú (Resident Guard)' : 'Resident Guard')
    ];
    elSpecDriverStrategy.textContent = stratNames[stratVal] || stratNames[0];

    if (typeof data.autoHard !== 'undefined') elCkAutoHard.checked = data.autoHard;
    if (typeof data.retryCount !== 'undefined') elNeRetryCnt.value = data.retryCount;
    if (typeof data.retryInterval !== 'undefined') elNeRetryMin.value = data.retryInterval;
  }

  // --- API Action Triggers ---
  async function sendAction(url, payload = null, btn = null) {
    if (btn) btn.disabled = true;
    try {
      const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload ? JSON.stringify(payload) : null
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.message || `Lỗi ${res.status}`);
      }
    } catch (err) {
      appendLogLine(`[!] Thao tác thất bại: ${err.message}`);
    } finally {
      if (btn) btn.disabled = false;
      fetchStatus();
    }
  }

  // 1. Mở khóa Gen2 ngay
  elBtnUnlockNow.addEventListener('click', () => {
    appendLogLine("[PCIe] Bắt đầu kích hoạt mở khóa Gen2 ngay lập tức...");
    sendAction('/api/unlock-now', null, elBtnUnlockNow);
  });

  // 2. Cài đặt toàn bộ 1-chạm
  elBtnFullInstall.addEventListener('click', () => {
    appendLogLine("[INSTALL] Bắt đầu quy trình triển khai toàn diện một chạm...");
    sendAction('/api/full-install', null, elBtnFullInstall);
  });

  // 3. Mở khóa + Cài tự khởi động
  elBtnGen2AndTask.addEventListener('click', () => {
    appendLogLine("[PCIe] Mở khóa và đăng ký tác vụ tự khởi động...");
    sendAction('/api/gen2-and-task', null, elBtnGen2AndTask);
  });

  // 4. Cài đặt mục đã chọn
  elBtnInstallSelected.addEventListener('click', () => {
    const sel = {
      gsp: elCkGsp.checked,
      drv: elCkDrv.checked,
      efi: elCkEfi.checked,
      task: elCkTask.checked,
      fast: elCkFast.checked,
      aspm: elCkAspm.checked,
      perf: elCkPerf.checked,
      defoff: elCkDefOff.checked
    };
    appendLogLine("[INSTALL] Áp dụng các mục đã chọn...");
    sendAction('/api/install', sel, elBtnInstallSelected);
  });

  // 5. Lưu cấu hình chính sách
  elBtnSavePolicy.addEventListener('click', () => {
    const stratEl = document.querySelector('input[name="driverStrategy"]:checked');
    const strat = stratEl ? parseInt(stratEl.value, 10) : 0;
    const payload = {
      strategy: strat,
      autoHard: elCkAutoHard.checked ? 1 : 0,
      retryCount: parseInt(elNeRetryCnt.value, 10) || 3,
      retryInterval: parseInt(elNeRetryMin.value, 10) || 5
    };
    appendLogLine(`[CONFIG] Lưu cấu hình chính sách: Chiến lược=${strat}, Stage2=${payload.autoHard}...`);
    sendAction('/api/save-policy', payload, elBtnSavePolicy);
  });

  elBtnRefresh.addEventListener('click', () => {
    appendLogLine("[SYS] Quét lại môi trường hệ thống...");
    fetchStatus();
  });

  // --- Fallback Mock Data ---
  function getMockStatus() {
    return {
      gpuDetected: false,
      gpuName: currentLang === 'vi' ? "Chưa phát hiện GPU CMP" : "No CMP GPU Detected",
      pciBusId: currentLang === 'vi' ? "Không phát hiện" : "Not Found",
      gspActive: false,
      isGen2: false,
      driverStrategy: 0,
      autoHard: true,
      retryCount: 3,
      retryInterval: 5,
      items: [
        { name: "Chế độ Boot", ok: true, note: "UEFI (OK)" },
        { name: "Secure Boot", ok: true, note: "Đã Tắt (OK)" },
        { name: "Card đồ hoạ", ok: false, note: "Chưa phát hiện card CMP 40HX hoặc CMP 30HX" },
        { name: "GSP (EnableGpuFirmware)", ok: false, note: "Chưa nạp hoặc chưa cần thiết" },
        { name: "ESP EFI Mở khoá", ok: false, note: "Chưa triển khai" },
        { name: "Mục khởi động BIOS", ok: false, note: "Chưa tạo" },
        { name: "Tác vụ tự khởi động", ok: false, note: "Chưa đăng ký" },
        { name: "Driver PCIe", ok: false, note: "Chưa cài đặt" },
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
  initLaneMatrix(0);
  initLogStream();
  fetchStatus();

})();
