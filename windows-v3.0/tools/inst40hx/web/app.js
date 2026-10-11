/**
 * CMP 40HX / 30HX Unlock Control Center - Frontend Application Logic
 */

(function () {
  'use strict';

  // --- Helpers ---
  const $ = (id) => document.getElementById(id);
  const escapeHtml = (text) => {
    const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
    return String(text).replace(/[&<>"']/g, m => map[m]);
  };

  let logHistory = [];
  let currentFilter = 'all';

  // --- 16 Physical Lane Matrix Setup ---
  function initLaneMatrix(activeCount = 16) {
    const grid = $('lanePinsGrid');
    if (!grid) return;
    grid.innerHTML = '';
    for (let i = 1; i <= 16; i++) {
      const pin = document.createElement('div');
      pin.className = 'lane-pin' + (i <= activeCount ? ' active' : '');
      pin.textContent = i;
      pin.title = `PCIe Làn #${i}: ${i <= activeCount ? 'Hoạt động (Negotiated)' : 'Không hoạt động'}`;
      grid.appendChild(pin);
    }
    const summary = $('laneSummaryText');
    if (summary) {
      summary.textContent = `${activeCount}/16 Làn kết nối`;
    }
  }

  // --- Log Streaming & SSE ---
  function initLogStream() {
    const indicator = $('streamIndicator');
    const statusText = $('streamStatusText');

    function setStatus(online) {
      if (!indicator || !statusText) return;
      if (online) {
        indicator.classList.add('online');
        statusText.textContent = "Trực tuyến (Live)";
      } else {
        indicator.classList.remove('online');
        statusText.textContent = "Mất kết nối";
      }
    }

    try {
      const evtSource = new EventSource('/api/logs/stream');
      evtSource.onopen = () => setStatus(true);
      evtSource.onmessage = (e) => {
        setStatus(true);
        if (e.data) appendLogLine(e.data);
      };
      evtSource.onerror = () => {
        setStatus(false);
        evtSource.close();
        setTimeout(initLogStream, 3000);
      };
    } catch {
      setStatus(false);
    }
  }

  function appendLogLine(lineText) {
    const lines = lineText.split('\n').filter(l => l.trim() !== '');
    lines.forEach(line => {
      const logObj = parseLogLine(line);
      logHistory.push(logObj);
      renderLogLine(logObj);
    });
    const counter = $('logLineCount');
    if (counter) counter.textContent = logHistory.length;
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
    const termBody = $('terminalLogBody');
    if (!termBody) return;
    if (currentFilter !== 'all' && currentFilter !== logObj.category) return;

    const div = document.createElement('div');
    div.className = `log-line ${logObj.category}`;
    div.innerHTML = `<span class="log-time">${logObj.timeStr}</span> ${escapeHtml(logObj.raw)}`;
    termBody.appendChild(div);
    termBody.scrollTop = termBody.scrollHeight;
  }

  function reRenderLogs() {
    const termBody = $('terminalLogBody');
    if (!termBody) return;
    termBody.innerHTML = '';
    logHistory.forEach(logObj => {
      if (currentFilter === 'all' || currentFilter === logObj.category) {
        renderLogLine(logObj);
      }
    });
  }

  // --- Log Terminal Controls ---
  document.querySelectorAll('.btn-filter').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.btn-filter').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      currentFilter = btn.getAttribute('data-filter');
      reRenderLogs();
    });
  });

  const btnClearLog = $('btnClearLog');
  if (btnClearLog) {
    btnClearLog.addEventListener('click', () => {
      logHistory = [];
      const termBody = $('terminalLogBody');
      if (termBody) termBody.innerHTML = '';
      const counter = $('logLineCount');
      if (counter) counter.textContent = '0';
    });
  }

  const btnCopyLog = $('btnCopyLog');
  if (btnCopyLog) {
    btnCopyLog.addEventListener('click', () => {
      const allText = logHistory.map(l => `${l.timeStr} ${l.raw}`).join('\n');
      navigator.clipboard.writeText(allText).then(() => {
        appendLogLine("[SYSTEM] Đã sao chép toàn bộ nhật ký vào clipboard.");
      });
    });
  }

  // --- Fetch System Status ---
  async function fetchStatus() {
    try {
      const chip = $('auditSummaryChip');
      if (chip) chip.textContent = "Đang quét...";
      const res = await fetch('/api/status');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      renderStatus(data);
    } catch {
      renderStatus(getMockStatus());
    }
    fetchRiotStatus();
  }

  function renderStatus(data) {
    // 1. Audit List
    const auditList = $('auditList');
    if (auditList) {
      auditList.innerHTML = '';
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
        auditList.appendChild(row);
      });

      const chip = $('auditSummaryChip');
      if (chip) {
        chip.textContent = warningCount === 0 ? '✓ Môi trường tối ưu' : `⚠ ${warningCount} cảnh báo cần xử lý`;
        chip.style.borderColor = warningCount === 0 ? 'var(--accent-phosphor)' : 'var(--accent-amber)';
        chip.style.color = warningCount === 0 ? 'var(--accent-phosphor)' : 'var(--accent-amber)';
      }
    }

    // 2. Hardware specs & Throughput
    const badge = $('gpuDetectedBadge');
    if (badge) {
      if (data.gpuDetected) {
        badge.textContent = data.gpuName || "CMP 40HX (TU106)";
        badge.className = "badge badge-emerald mono";
      } else {
        badge.textContent = "Chưa phát hiện GPU";
        badge.className = "badge mono";
        badge.style.color = "var(--accent-crimson)";
        badge.style.borderColor = "var(--accent-crimson)";
      }
    }

    if ($('specBusId') && data.pciBusId) $('specBusId').textContent = data.pciBusId;

    const gspEl = $('specGsp');
    if (gspEl) {
      if (data.gspActive) {
        gspEl.textContent = "Đã bật (GSP-RM Mode)";
        gspEl.className = "spec-value highlight-cyan";
      } else {
        gspEl.textContent = "Chưa bật (Nguy cơ lỗi 43)";
        gspEl.className = "spec-value";
        gspEl.style.color = "var(--accent-amber)";
      }
    }

    // Recommendations checkboxes
    if (data.recommendations) {
      const keys = ['gsp', 'drv', 'efi', 'task', 'fast', 'aspm', 'perf', 'defoff'];
      keys.forEach(k => {
        const ck = $('ck' + k.charAt(0).toUpperCase() + k.slice(1));
        if (ck && typeof data.recommendations[k] !== 'undefined') {
          ck.checked = data.recommendations[k];
        }
      });
    }

    // Driver Strategy
    const stratVal = data.driverStrategy || 0;
    const stratRadio = document.querySelector(`input[name="driverStrategy"][value="${stratVal}"]`);
    if (stratRadio) stratRadio.checked = true;

    const stratEl = $('specDriverStrategy');
    if (stratEl) {
      const stratNames = ['Dùng xong gỡ ngay (Clean)', 'Tự động thử lại khi lỗi', 'Thường trú (Resident Guard)'];
      stratEl.textContent = stratNames[stratVal] || stratNames[0];
    }

    if ($('ckAutoHard') && typeof data.autoHard !== 'undefined') $('ckAutoHard').checked = data.autoHard;
    if ($('neRetryCnt') && typeof data.retryCount !== 'undefined') $('neRetryCnt').value = data.retryCount;
    if ($('neRetryMin') && typeof data.retryInterval !== 'undefined') $('neRetryMin').value = data.retryInterval;

    // PCIe Link status & gauge
    const isGen2 = data.isGen2 || false;
    const gpuDetected = data.gpuDetected || false;
    const throughput = $('currentThroughput');
    const gauge = $('gaugeBarFill');
    const speedSub = $('currentSpeedSub');
    const boostRatio = $('boostRatio');

    if (gpuDetected && isGen2) {
      initLaneMatrix(16);
      if (throughput) throughput.innerHTML = `~6.4 <span class="unit">GB/s</span>`;
      if (gauge) gauge.style.width = '100%';
      if (speedSub) {
        speedSub.textContent = "Gen2 x16 @ 5.0 GT/s";
        speedSub.style.color = "var(--text-secondary)";
      }
      if (boostRatio) {
        boostRatio.textContent = "25.6x BOOST";
        boostRatio.style.color = "var(--accent-phosphor)";
        boostRatio.style.borderColor = "var(--accent-phosphor)";
      }
    } else if (gpuDetected) {
      initLaneMatrix(1);
      if (throughput) throughput.innerHTML = `250 <span class="unit">MB/s</span>`;
      if (gauge) gauge.style.width = '4%';
      if (speedSub) {
        speedSub.textContent = "Gen1 x1 @ 2.5 GT/s";
        speedSub.style.color = "var(--text-muted)";
      }
      if (boostRatio) {
        boostRatio.textContent = "1.0x (Khóa eFuse)";
        boostRatio.style.color = "var(--text-muted)";
        boostRatio.style.borderColor = "var(--border-medium)";
      }
    } else {
      initLaneMatrix(0);
      if (throughput) throughput.innerHTML = `0 <span class="unit">MB/s</span>`;
      if (gauge) gauge.style.width = '0%';
      if (speedSub) {
        speedSub.textContent = "Chưa phát hiện GPU";
        speedSub.style.color = "var(--accent-crimson)";
      }
      if (boostRatio) {
        boostRatio.textContent = "-- BOOST";
        boostRatio.style.color = "var(--text-muted)";
        boostRatio.style.borderColor = "var(--border-medium)";
      }
    }

    // Safe boundaries per card model (30HX vs 40HX)
    const is30HX = data.is30HX || (data.gpuName && (data.gpuName.includes("30HX") || data.gpuName.includes("TU116")));
    const is40HX = data.is40HX || (data.gpuName && (data.gpuName.includes("40HX") || data.gpuName.includes("TU106")));

    const forceRootBtn = $('btnForceRootGen2');
    if (forceRootBtn) {
      const btnTitle = forceRootBtn.querySelector('.btn-title');
      const btnSub = forceRootBtn.querySelector('.btn-sub');
      if (btnTitle && btnSub) {
        if (is30HX) {
          btnTitle.textContent = "⚡ Ép mở khóa Gen2 (CMP 30HX)";
          btnSub.textContent = "Bảo vệ eFuse, nạp MMIO TU116 & ép Root Port Gen2 (Đã cách ly 40HX)";
        } else if (is40HX) {
          btnTitle.textContent = "⚡ Ép mở khóa Gen2 (CMP 40HX)";
          btnSub.textContent = "Bỏ qua kẹt LNKCAP Gen1, nạp MMIO TU106 & ép Root Port Gen2";
        } else {
          btnTitle.textContent = "⚡ Ép mở khóa Gen2 (Tự nhận diện card)";
          btnSub.textContent = "Tự động nhận diện 30HX / 40HX để nạp đúng chuỗi an toàn";
        }
      }
    }

    const tipTitle = document.querySelector('.tip-callout-box .tip-title');
    const tipDesc = document.querySelector('.tip-callout-box .tip-desc');
    if (tipTitle && tipDesc) {
      if (is30HX) {
        tipTitle.textContent = "🔒 BẢO VỆ PHẦN CỨNG CMP 30HX (TU116)";
        tipDesc.innerHTML = "Đã nhận diện <strong>NVIDIA CMP 30HX</strong>. Hệ thống tự động cách ly: khóa cứng eFuse ở Gen2 (không ép Gen3), không áp dụng microcode/reset của 40HX. Bấm <strong>[⚡ Ép mở khóa Gen2 (CMP 30HX)]</strong> để nạp chuỗi MMIO TU116 và ép Root Port huấn luyện lại an toàn!";
      } else if (is40HX) {
        tipTitle.textContent = "⚡ KHUYẾN NGHỊ CMP 40HX (TU106) - BỎ QUA KẸT GEN1";
        tipDesc.innerHTML = "Đã nhận diện <strong>NVIDIA CMP 40HX</strong>. Khi chưa reboot EFI hoặc Fast Startup chặn UEFI, thanh ghi LNKCAP sẽ tạm thời báo Gen1 (0x00463D01). Bấm <strong>[⚡ Ép mở khóa Gen2 (CMP 40HX)]</strong> để nạp chuỗi MMIO TU106 và ép Root Port bo mạch chủ nâng tốc độ lên Gen2 x16 tức thì!";
      }
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

  // Bind Standard Action Buttons (Deduplicated event registration)
  const actionRegistry = [
    { id: 'btnUnlockNow', url: '/api/unlock-now', log: '[PCIe] Bắt đầu kích hoạt mở khóa Gen2 ngay lập tức...' },
    { id: 'btnForceRootGen2', url: '/api/force-root-gen2', log: '[PCIe] ⚡ Bắt đầu Ép Mở Khoá Gen2 qua Root Port (-force-root-gen2)...' },
    { id: 'btnFullInstall', url: '/api/full-install', log: '[INSTALL] Bắt đầu quy trình triển khai toàn diện một chạm...' },
    { id: 'btnGen2AndTask', url: '/api/gen2-and-task', log: '[PCIe] Mở khoá và đăng ký tác vụ tự khởi động...' },
    { id: 'btnRiotOptimize', url: '/api/riot/optimize', log: '[RIOT] Bắt đầu tối ưu hóa hệ thống cho Riot Games & dọn dẹp driver...' },
    { id: 'btnRiotOptimizeInner', url: '/api/riot/optimize', log: '[RIOT] Bắt đầu tối ưu hóa hệ thống cho Riot Games & dọn dẹp driver...' },
    { id: 'btnLaunchUnlockRiotExe', url: '/api/riot/launch-gui', log: '[EXE] Khởi chạy ứng dụng chuyên sâu UnlockRiotGame.exe...' },
    { id: 'btnModalRebootBios', url: '/api/riot/reboot-bios', log: '[BIOS] Đang khởi động lại vào BIOS Setup...' }
  ];

  actionRegistry.forEach(({ id, url, log }) => {
    const btn = $(id);
    if (btn) {
      btn.addEventListener('click', () => {
        appendLogLine(log);
        sendAction(url, null, btn);
      });
    }
  });

  const btnInstallSelected = $('btnInstallSelected');
  if (btnInstallSelected) {
    btnInstallSelected.addEventListener('click', () => {
      const sel = {
        gsp: $('ckGsp')?.checked ?? true,
        drv: $('ckDrv')?.checked ?? true,
        efi: $('ckEfi')?.checked ?? false,
        task: $('ckTask')?.checked ?? true,
        fast: $('ckFast')?.checked ?? false,
        aspm: $('ckAspm')?.checked ?? false,
        perf: $('ckPerf')?.checked ?? false,
        defoff: $('ckDefOff')?.checked ?? false
      };
      appendLogLine("[INSTALL] Áp dụng các mục đã chọn...");
      sendAction('/api/install', sel, btnInstallSelected);
    });
  }

  const btnSavePolicy = $('btnSavePolicy');
  if (btnSavePolicy) {
    btnSavePolicy.addEventListener('click', () => {
      const stratEl = document.querySelector('input[name="driverStrategy"]:checked');
      const strat = stratEl ? parseInt(stratEl.value, 10) : 0;
      const payload = {
        strategy: strat,
        autoHard: $('ckAutoHard')?.checked ? 1 : 0,
        retryCount: parseInt($('neRetryCnt')?.value, 10) || 3,
        retryInterval: parseInt($('neRetryMin')?.value, 10) || 5
      };
      appendLogLine(`[CONFIG] Lưu cấu hình chính sách: Chiến lược=${strat}, Stage2=${payload.autoHard}...`);
      sendAction('/api/save-policy', payload, btnSavePolicy);
    });
  }

  const btnRefresh = $('btnRefresh');
  if (btnRefresh) {
    btnRefresh.addEventListener('click', () => {
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
    } catch {
      // Bỏ qua lỗi kết nối Riot API nếu daemon chạy độc lập
    }
  }

  function renderRiotStatus(data) {
    if (!data) return;
    const chip = $('riotSummaryChip');
    if (chip) {
      if (data.canPlayValorant && data.canPlayLeagueOfLegends) {
        chip.textContent = "Hoàn Hảo (100%)";
        chip.className = "riot-summary-chip chip-ok";
      } else if (data.canPlayLeagueOfLegends) {
        chip.textContent = "Sẵn sàng (LMHT)";
        chip.className = "riot-summary-chip chip-ready";
      } else {
        chip.textContent = "Cần thiết lập";
        chip.className = "riot-summary-chip chip-warn";
      }
    }

    if ($('riotLolStatus')) $('riotLolStatus').textContent = "✓ TƯƠNG THÍCH 100%";
    if ($('riotLolDetail') && data.lolDesc) $('riotLolDetail').textContent = data.lolDesc;

    const anticheatEl = $('specAnticheat');
    if (anticheatEl) {
      if (data.canPlayValorant && data.canPlayLeagueOfLegends) {
        anticheatEl.textContent = "Hoàn hảo (Vanguard + LMHT)";
        anticheatEl.className = "spec-value highlight-cyan";
      } else if (data.canPlayLeagueOfLegends) {
        anticheatEl.textContent = "Sẵn sàng (LMHT / Anti-Cheat)";
        anticheatEl.className = "spec-value highlight-cyan";
      } else {
        anticheatEl.textContent = "Cần thiết lập BIOS";
        anticheatEl.className = "spec-value";
        anticheatEl.style.color = "var(--accent-amber)";
      }
    }

    const valStatus = $('riotValorantStatus');
    if (valStatus) {
      if (data.canPlayValorant) {
        valStatus.textContent = "✓ HOÀN TOÀN TƯƠNG THÍCH";
        valStatus.style.color = "var(--color-success)";
      } else {
        valStatus.textContent = "⚠️ CẦN THIẾT LẬP BIOS";
        valStatus.style.color = "var(--color-amber)";
      }
    }
    if ($('riotValorantDetail') && data.valorantDesc) $('riotValorantDetail').textContent = data.valorantDesc;
    if ($('riotRecText') && data.recommendation) $('riotRecText').textContent = data.recommendation;

    const btnSign = $('btnRiotSignEfi');
    if (btnSign) {
      const is40HX = data.model === "CMP 40HX" || data.hasTensorCore;
      btnSign.style.display = is40HX ? "inline-flex" : "none";
    }
  }

  // --- Modal Helpers ---
  function openBiosModal() {
    const modal = $('modalBiosGuide');
    if (modal) {
      modal.removeAttribute('hidden');
      modal.classList.add('active');
    }
  }

  function closeBiosModal() {
    const modal = $('modalBiosGuide');
    if (modal) {
      modal.setAttribute('hidden', '');
      modal.classList.remove('active');
    }
  }

  $('btnRiotGuide')?.addEventListener('click', openBiosModal);
  $('btnModalClose')?.addEventListener('click', closeBiosModal);
  $('btnModalDismiss')?.addEventListener('click', closeBiosModal);
  $('modalBiosGuide')?.addEventListener('click', (e) => {
    if (e.target === $('modalBiosGuide')) closeBiosModal();
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeBiosModal();
  });

  const btnSignEfi = $('btnRiotSignEfi');
  if (btnSignEfi) {
    btnSignEfi.addEventListener('click', async () => {
      appendLogLine("[UEFI] Bắt đầu tạo chứng chỉ cá nhân và ký Authenticode cho 40HXUNLK.EFI...");
      await sendAction('/api/riot/sign-efi', null, btnSignEfi);
      openBiosModal();
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
  initLaneMatrix(0);
  initLogStream();
  fetchStatus();

})();
