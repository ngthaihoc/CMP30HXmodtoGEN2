// Telemetry & Hardware Matrix Visualizer
const Telemetry = {
  init() {
    this.initLaneGrid();

    if (window.runtime && window.runtime.EventsOn) {
      window.runtime.EventsOn('telemetry:tick', (data) => {
        this.render(data);
      });
    }
  },

  initLaneGrid() {
    const grid = document.getElementById('lanePinsGrid');
    if (!grid) return;
    grid.innerHTML = '';
    for (let i = 1; i <= 16; i++) {
      const pin = document.createElement('div');
      pin.id = `lanePin_${i}`;
      pin.className = 'lane-pin';
      pin.textContent = i;
      pin.title = `PCIe Làn #${i}: Chưa kích hoạt`;
      grid.appendChild(pin);
    }
  },

  render(data) {
    if (!data) return;

    // 1. Hardware Information
    const gpuBadge = document.getElementById('hudGpuBadge');
    if (gpuBadge) {
      if (data.gpuDetected) {
        gpuBadge.textContent = data.gpuName;
        gpuBadge.style.color = "var(--accent-emerald)";
        gpuBadge.style.borderColor = "var(--accent-emerald)";
      } else {
        gpuBadge.textContent = "Chưa phát hiện GPU CMP";
        gpuBadge.style.color = "var(--accent-crimson)";
        gpuBadge.style.borderColor = "var(--accent-crimson)";
      }
    }

    const busEl = document.getElementById('specBusId');
    if (busEl) busEl.textContent = data.pciBusId || "--";

    const gspEl = document.getElementById('specGspStatus');
    if (gspEl) {
      gspEl.textContent = data.gspActive ? "Đã bật (GSP-RM)" : "Tắt (CPU-RM)";
      gspEl.style.color = data.gspActive ? "var(--accent-emerald)" : "var(--accent-amber)";
    }

    const mrrsEl = document.getElementById('specMrrsVal');
    if (mrrsEl) {
      mrrsEl.textContent = `${data.mrrs || 128} Bytes`;
      mrrsEl.style.color = data.mrrs === 512 ? "var(--accent-emerald)" : "var(--accent-amber)";
    }

    const clockTempEl = document.getElementById('specClockTempVal');
    if (clockTempEl) {
      if (data.clockMHz > 0 || data.temperatureC > 0) {
        clockTempEl.textContent = `${data.clockMHz || '--'} MHz / ${data.temperatureC || '--'} °C`;
      } else {
        clockTempEl.textContent = `Chế độ chờ (Idle)`;
      }
    }

    const stratEl = document.getElementById('specStratVal');
    if (stratEl) {
      const stratNames = ["Dùng xong gỡ ngay", "Tự thử lại khi lỗi", "Thường trú (Resident Guard)"];
      stratEl.textContent = stratNames[data.driverStrategy] || "Mặc định";
    }

    // 2. 16-Lane PCIe Matrix
    const activeLanesCount = (data.lanes || []).filter(Boolean).length;
    const laneCountText = document.getElementById('laneCountText');
    if (laneCountText) {
      laneCountText.textContent = `${activeLanesCount}/16 Làn`;
    }

    for (let i = 1; i <= 16; i++) {
      const pin = document.getElementById(`lanePin_${i}`);
      if (pin) {
        const isActive = data.lanes && data.lanes[i - 1];
        if (isActive) {
          pin.classList.add('active');
          pin.title = `PCIe Làn #${i}: Hoạt động (Gen${data.linkSpeed || 1})`;
        } else {
          pin.classList.remove('active');
          pin.title = `PCIe Làn #${i}: Chưa kích hoạt`;
        }
      }
    }

    // 3. Bandwidth Throughput & Gauges
    const tpNumber = document.getElementById('throughputNumber');
    const tpSub = document.getElementById('throughputSubtext');
    const gaugeFill = document.getElementById('gaugeFill');
    const boostTag = document.getElementById('boostTag');

    const mb = data.throughputMBs || 250;
    if (tpNumber) {
      if (mb >= 1000) {
        tpNumber.innerHTML = `${(mb / 1000).toFixed(1)} <span class="throughput-unit">GB/s</span>`;
      } else {
        tpNumber.innerHTML = `${mb} <span class="throughput-unit">MB/s</span>`;
      }
    }

    if (tpSub) {
      tpSub.textContent = `PCIe Gen${data.linkSpeed || 1} x${data.linkWidth || 1} · MRRS ${data.mrrs || 128}B`;
    }

    // Tỉ lệ thanh đo (Max Gen2 x16 là 6400 MB/s = 100%)
    const pct = Math.min(100, Math.max(4, Math.round((mb / 6400) * 100)));
    if (gaugeFill) {
      gaugeFill.style.width = `${pct}%`;
    }

    if (boostTag) {
      if (data.isGen2 && data.linkWidth === 16) {
        boostTag.textContent = "✓ TỐI ĐA GEN2 x16";
        boostTag.className = "boost-tag max";
      } else {
        boostTag.textContent = "⚡ CHƯA MỞ KHOÁ";
        boostTag.className = "boost-tag";
      }
    }
  }
};
