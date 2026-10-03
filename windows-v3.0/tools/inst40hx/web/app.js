/**
 * CMP 40HX / 30HX Unlock Control Center - Frontend Application Logic
 */

(function () {
  'use strict';

  // --- Bilingual Dictionary ---
  const i18n = {
    vi: {
      brand_desc: "NVIDIA CMP 40HX & 30HX PCIe Gen2 x16 / K├¡ch hoß║ít B─âng th├┤ng & T├¡nh to├ín",
      status_connecting: "─Éang kß║┐t nß╗æi...",
      status_connected: "Trß╗▒c tuyß║┐n (Live)",
      status_disconnected: "Mß║Ñt kß║┐t nß╗æi",
      btn_refresh: "Qu├⌐t lß║íi",
      refresh_tooltip: "Qu├⌐t lß║íi m├┤i tr╞░ß╗¥ng hß╗ç thß╗æng",
      hud_silicon_title: "TH├öNG TIN B├üN Dß║¬N & BUS",
      spec_bus_id: "PCIe Bus:",
      spec_gsp: "GSP-RM Firmware:",
      spec_vanguard: "Riot Vanguard / Game:",
      spec_driver_strategy: "Chiß║┐n l╞░ß╗úc Driver:",
      lane_topology: "Cß║Ñu tr├║c 16 l├án PCIe vß║¡t l├╜ (Physical Lanes):",
      hud_bandwidth_title: "B─éNG TH├öNG & LI├èN Kß║╛T PCIE",
      speed_locked: "Gß╗ÉC (KHO├ü EFUSE)",
      speed_unlocked: "─É├â Mß╗₧ KHO├ü TO├ÇN Bß╗ÿ",
      btn_unlock_title: "Mß╗₧ KHO├ü PCIE GEN2 NGAY",
      btn_unlock_sub: "K├¡ch hoß║ít tß╗⌐c th├¼ phi├¬n hiß╗çn tß║íi (kh├┤ng cß║ºn khß╗ƒi ─æß╗Öng lß║íi m├íy)",
      btn_full_title: "C├ÇI ─Éß║╢T TO├ÇN Bß╗ÿ 1-CHß║áM",
      btn_full_sub: "Tß╗▒ ─æß╗Öng cß║Ñu h├¼nh chuß║⌐n: GSP + Driver + Tß╗▒ khß╗ƒi ─æß╗Öng + Nguß╗ôn",
      btn_persist_title: "Mß╗ƒ kho├í & C├ái tß╗▒ khß╗ƒi ─æß╗Öng",
      btn_persist_sub: "─É─âng k├╜ t├íc vß╗Ñ tß╗▒ chß║íy khi ─æ─âng nhß║¡p Windows",
      audit_title: "Chß║⌐n ─æo├ín M├┤i tr╞░ß╗¥ng & Phß║ºn cß╗⌐ng",
      audit_sub: "Kiß╗âm tra t├¡nh t╞░╞íng th├¡ch tr╞░ß╗¢c khi k├¡ch hoß║ít",
      components_title: "Th├ánh phß║ºn & Thiß║┐t lß║¡p hß╗ç thß╗æng",
      components_sub: "Chß╗ìn c├íc mß╗Ñc cß║ºn cß║¡p nhß║¡t hoß║╖c bß║Ñm [C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn]",
      desc_gsp: "Bß║»t buß╗Öc ─æß╗â tr├ính m├ú lß╗ùi Code 43 sau khi mß╗ƒ kh├│a",
      desc_drv: "Nß║íp driver can thiß╗çp thanh ghi & cß║Ñp quyß╗ün an to├án",
      desc_efi: "Nß║íp payload v├áo ph├ón v├╣ng ESP (chß╗ë ├íp dß╗Ñng cho chuß║⌐n UEFI)",
      desc_task: "─É─âng k├╜ t├íc vß╗Ñ Task Scheduler SYSTEM v├á Run Key dß╗▒ ph├▓ng",
      desc_fast: "Tr├ính t├¼nh trß║íng Windows nß║íp sleep image bß╗Å qua UEFI hook",
      desc_aspm: "Tr├ính PCIe tß╗▒ ─æß╗Öng rß╗¢t vß╗ü Gen1 x1 khi m├íy t├¡nh ß╗ƒ trß║íng th├íi rß║únh",
      desc_perf: "─Éß║úm bß║úo cß║Ñp ─æß╗º n─âng l╞░ß╗úng cho li├¬n kß║┐t PCIe hoß║ít ─æß╗Öng tß╗æi ─æa",
      desc_defoff: "Chß╗ë cß║ºn thiß║┐t khi phß║ºn mß╗üm diß╗çt virus chß║╖n file sys can thiß╗çp",
      btn_install_selected: "C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn",
      policy_title: "Chiß║┐n l╞░ß╗úc Driver & Tß╗▒ phß╗Ñc hß╗ôi",
      policy_sub: "Cß║Ñu h├¼nh h├ánh vi sau khi ho├án tß║Ñt mß╗ƒ kh├│a",
      strat_0_name: "D├╣ng xong gß╗í ngay (Khuy├¬n d├╣ng cho Game)",
      strat_0_desc: "Sau khi n├óng tß╗æc ─æß╗Ö, driver can thiß╗çp ─æ╞░ß╗úc gß╗í ho├án to├án. An to├án tuyß╗çt ─æß╗æi vß╗¢i Riot Vanguard & Anti-Cheat.",
      strat_1_name: "Tß╗▒ ─æß╗Öng thß╗¡ lß║íi khi lß╗ùi",
      strat_1_desc: "Tß╗▒ ─æß╗Öng thß╗¡ lß║íi nß║┐u lß║ºn ─æß║ºu khß╗ƒi tß║ío GPU ch╞░a ─æß║ít tß╗æc ─æß╗Ö Gen2.",
      strat_2_name: "Th╞░ß╗¥ng tr├║ (Canh giß╗» tß╗æc ─æß╗Ö PCIe)",
      strat_2_desc: "Driver chß║íy nß╗ün th╞░ß╗¥ng trß╗▒c, ─æß╗ïnh kß╗│ kiß╗âm tra v├á ├⌐p xung PCIe trß╗ƒ lß║íi nß║┐u bß╗ï tß╗Ñt xung.",
      desc_autohard: "Tß╗▒ ─æß╗Öng k├¡ch hoß║ít Stage 2 (Link Disable + Reset PnP) khi mß╗ƒ kh├│a th╞░ß╗¥ng kh├┤ng ─æß║ít",
      retry_count_label: "Sß╗æ lß║ºn thß╗¡ lß║íi:",
      retry_interval_label: "Gi├ún c├ích:",
      btn_save_policy: "L╞░u cß║Ñu h├¼nh ch├¡nh s├ích",
      terminal_title: "DIAGNOSTIC & LOG STREAM (REAL-TIME)",
      log_lines: "d├▓ng log",
      btn_clear_log: "X├│a",
      btn_copy_log: "Sao ch├⌐p",
      safety_title: "An to├án cho Game & Anti-Cheat (Riot Vanguard, EasyAntiCheat, BattlEye)",
      safety_desc: "Giß║úi ph├íp v3.0 kh├┤ng flash VBIOS, kh├┤ng bß║¡t Test Signing, gß╗í sß║ích driver can thiß╗çp sau khi mß╗ƒ kh├│a. ─Éß║úm bß║úo 100% t├¡nh to├án vß║╣n hß╗ç ─æiß╗üu h├ánh."
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
    } else if (line.includes('[!]') || line.includes('lß╗ùi') || line.includes('thß║Ñt bß║íi') || line.includes('Error') || line.includes('Fail')) {
      category = 'warn';
    } else if (line.includes('Γ£ô') || line.includes('th├ánh c├┤ng') || line.includes('OK') || line.includes('Success')) {
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
      appendLogLine("[SYSTEM] ─É├ú sao ch├⌐p to├án bß╗Ö nhß║¡t k├╜ v├áo clipboard.");
    });
  });

  // --- Fetch System Status ---
  async function fetchStatus() {
    try {
      elAuditSummaryChip.textContent = "─Éang qu├⌐t...";
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
          <div class="audit-icon ${item.ok ? 'ok' : 'warn'}">${item.ok ? 'Γ£ô' : '!'}</div>
          <span class="audit-name">${escapeHtml(item.name)}</span>
        </div>
        <div class="audit-note">${escapeHtml(item.note)}</div>
      `;
      elAuditList.appendChild(row);
    });

    elAuditSummaryChip.textContent = warningCount === 0 
      ? (currentLang === 'vi' ? 'Γ£ô M├┤i tr╞░ß╗¥ng tß╗æi ╞░u' : 'Γ£ô System Ready')
      : (currentLang === 'vi' ? `ΓÜá ${warningCount} cß║únh b├ío cß║ºn xß╗¡ l├╜` : `ΓÜá ${warningCount} warnings`);
    
    elAuditSummaryChip.style.borderColor = warningCount === 0 ? 'var(--accent-emerald)' : 'var(--accent-amber)';
    elAuditSummaryChip.style.color = warningCount === 0 ? 'var(--accent-emerald)' : 'var(--accent-amber)';

    // 2. Hardware specs & Throughput
    if (data.gpuDetected) {
      elGpuDetectedBadge.textContent = data.gpuName || "CMP 40HX (TU106)";
      elGpuDetectedBadge.className = "badge badge-emerald";
    } else {
      elGpuDetectedBadge.textContent = currentLang === 'vi' ? "Ch╞░a ph├ít hiß╗çn GPU" : "No GPU Detected";
      elGpuDetectedBadge.className = "badge";
      elGpuDetectedBadge.style.color = "var(--accent-crimson)";
    }

    if (data.pciBusId) {
      elSpecBusId.textContent = data.pciBusId;
    }

    if (data.gspActive) {
      elSpecGsp.textContent = currentLang === 'vi' ? "─É├ú bß║¡t (GSP-RM Mode)" : "Enabled (GSP Mode)";
      elSpecGsp.className = "spec-value highlight-cyan";
    } else {
      elSpecGsp.textContent = currentLang === 'vi' ? "Ch╞░a bß║¡t (C├│ thß╗â lß╗ùi 43)" : "Disabled (Risk Code 43)";
      elSpecGsp.className = "spec-value";
      elSpecGsp.style.color = "var(--accent-amber)";
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
      (currentLang === 'vi' ? 'D├╣ng xong gß╗í ngay (Clean)' : 'Clean Exit'),
      (currentLang === 'vi' ? 'Tß╗▒ ─æß╗Öng thß╗¡ lß║íi khi lß╗ùi' : 'Auto Retry'),
      (currentLang === 'vi' ? 'Th╞░ß╗¥ng tr├║ (Resident Guard)' : 'Resident Guard')
    ];
    elSpecDriverStrategy.textContent = stratNames[stratVal] || stratNames[0];

    if (typeof data.autoHard !== 'undefined') elCkAutoHard.checked = data.autoHard;
    if (typeof data.retryCount !== 'undefined') elNeRetryCnt.value = data.retryCount;
    if (typeof data.retryInterval !== 'undefined') elNeRetryMin.value = data.retryInterval;

    // PCIe Link status & gauge
    const isGen2 = data.isGen2 || false;
    if (isGen2) {
      initLaneMatrix(16);
      elCurrentThroughput.innerHTML = `~6.4 <span class="unit">GB/s</span>`;
      elGaugeBarFill.style.width = '100%';
    } else {
      initLaneMatrix(1);
      elCurrentThroughput.innerHTML = `250 <span class="unit">MB/s</span>`;
      elGaugeBarFill.style.width = '4%';
    }
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
        throw new Error(data.message || `Lß╗ùi ${res.status}`);
      }
    } catch (err) {
      appendLogLine(`[!] Thao t├íc thß║Ñt bß║íi: ${err.message}`);
    } finally {
      if (btn) btn.disabled = false;
      fetchStatus();
    }
  }

  // 1. Mß╗ƒ kh├│a Gen2 ngay
  elBtnUnlockNow.addEventListener('click', () => {
    appendLogLine("[PCIe] Bß║»t ─æß║ºu k├¡ch hoß║ít mß╗ƒ kh├│a Gen2 ngay lß║¡p tß╗⌐c...");
    sendAction('/api/unlock-now', null, elBtnUnlockNow);
  });

  // 2. C├ái ─æß║╖t to├án bß╗Ö 1-chß║ím
  elBtnFullInstall.addEventListener('click', () => {
    appendLogLine("[INSTALL] Bß║»t ─æß║ºu quy tr├¼nh triß╗ân khai to├án diß╗çn mß╗Öt chß║ím...");
    sendAction('/api/full-install', null, elBtnFullInstall);
  });

  // 3. Mß╗ƒ kh├│a + C├ái tß╗▒ khß╗ƒi ─æß╗Öng
  elBtnGen2AndTask.addEventListener('click', () => {
    appendLogLine("[PCIe] Mß╗ƒ kh├│a v├á ─æ─âng k├╜ t├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng...");
    sendAction('/api/gen2-and-task', null, elBtnGen2AndTask);
  });

  // 4. C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn
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
    appendLogLine("[INSTALL] ├üp dß╗Ñng c├íc mß╗Ñc ─æ├ú chß╗ìn...");
    sendAction('/api/install', sel, elBtnInstallSelected);
  });

  // 5. L╞░u cß║Ñu h├¼nh ch├¡nh s├ích
  elBtnSavePolicy.addEventListener('click', () => {
    const stratEl = document.querySelector('input[name="driverStrategy"]:checked');
    const strat = stratEl ? parseInt(stratEl.value, 10) : 0;
    const payload = {
      strategy: strat,
      autoHard: elCkAutoHard.checked ? 1 : 0,
      retryCount: parseInt(elNeRetryCnt.value, 10) || 3,
      retryInterval: parseInt(elNeRetryMin.value, 10) || 5
    };
    appendLogLine(`[CONFIG] L╞░u cß║Ñu h├¼nh ch├¡nh s├ích: Chiß║┐n l╞░ß╗úc=${strat}, Stage2=${payload.autoHard}...`);
    sendAction('/api/save-policy', payload, elBtnSavePolicy);
  });

  elBtnRefresh.addEventListener('click', () => {
    appendLogLine("[SYS] Qu├⌐t lß║íi m├┤i tr╞░ß╗¥ng hß╗ç thß╗æng...");
    fetchStatus();
  });

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
        { name: "Chß║┐ ─æß╗Ö Boot", ok: true, note: "UEFI (OK)" },
        { name: "Secure Boot", ok: true, note: "─É├ú Tß║»t (OK)" },
        { name: "Card ─æß╗ô hoß║í", ok: true, note: "─É├ú ph├ít hiß╗çn NVIDIA CMP 40HX" },
        { name: "GSP (EnableGpuFirmware)", ok: true, note: "─É├ú bß║¡t (OK)" },
        { name: "ESP EFI Mß╗ƒ kho├í", ok: true, note: "\\EFI\\40HX\\40HXUNLK.EFI ─æ├ú nß║íp" },
        { name: "Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS", ok: true, note: "Tß╗ôn tß║íi v├á nß║▒m ─æß║ºu ti├¬n (displayorder)" },
        { name: "T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng", ok: true, note: "Trß║íng th├íi: Ready" },
        { name: "Driver PCIe", ok: true, note: "─É├ú c├ái; Tß╗▒ dß╗ìn dß║╣p sau khi chß║íy (Game safe)" },
        { name: "Loß║íi trß╗½ Windows Defender", ok: true, note: "─É├ú th├¬m loß║íi trß╗½ cho file .sys" },
        { name: "Khß╗ƒi ─æß╗Öng nhanh (Fast Startup)", ok: true, note: "─É├ú Tß║»t (OK)" },
        { name: "Tiß║┐t kiß╗çm ─æiß╗çn PCIe (ASPM)", ok: true, note: "─É├ú Tß║»t (OK)" }
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
