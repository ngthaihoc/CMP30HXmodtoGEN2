// Real-time Logger & Terminal System
const Logger = {
  history: [],
  currentFilter: 'all',

  init() {
    const termBody = document.getElementById('terminalBody');
    const filterBtns = document.querySelectorAll('.term-filter-btn');
    const btnClear = document.getElementById('btnClearLog');
    const btnCopy = document.getElementById('btnCopyLog');

    filterBtns.forEach(btn => {
      btn.addEventListener('click', () => {
        filterBtns.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        this.currentFilter = btn.getAttribute('data-filter') || 'all';
        this.renderAll();
      });
    });

    if (btnClear) {
      btnClear.addEventListener('click', () => {
        this.history = [];
        if (termBody) termBody.innerHTML = '';
        this.updateCount();
      });
    }

    if (btnCopy) {
      btnCopy.addEventListener('click', () => {
        const text = this.history.map(l => `[${l.time}] ${l.raw}`).join('\n');
        navigator.clipboard.writeText(text).then(() => {
          this.append("[HỆ THỐNG] Đã sao chép toàn bộ nhật ký vào clipboard.");
        });
      });
    }

    // Lắng nghe sự kiện Wails runtime
    if (window.runtime && window.runtime.EventsOn) {
      window.runtime.EventsOn('log:entry', (line) => {
        this.append(line);
      });
    }
  },

  append(line) {
    if (!line) return;
    const now = new Date();
    const time = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`;
    let category = 'sys';

    const lower = line.toLowerCase();
    if (lower.includes('[pcie]') || lower.includes('gen2') || lower.includes('gen1') || lower.includes('lane') || lower.includes('mrrs')) {
      category = 'pcie';
    } else if (lower.includes('[!]') || lower.includes('lỗi') || lower.includes('thất bại') || lower.includes('error') || lower.includes('fail')) {
      category = 'err';
    } else if (lower.includes('cảnh báo') || lower.includes('warn') || lower.includes('chú ý')) {
      category = 'warn';
    } else if (lower.includes('✓') || lower.includes('[v]') || lower.includes('thành công') || lower.includes('ok')) {
      category = 'ok';
    }

    const item = { time, raw: line, category };
    this.history.push(item);
    if (this.history.length > 500) {
      this.history.shift();
    }

    if (this.currentFilter === 'all' || this.currentFilter === category) {
      this.renderLine(item);
    }
    this.updateCount();
  },

  renderLine(item) {
    const termBody = document.getElementById('terminalBody');
    if (!termBody) return;

    const div = document.createElement('div');
    div.className = `log-line ${item.category}`;
    div.textContent = `[${item.time}] ${item.raw}`;
    termBody.appendChild(div);
    termBody.scrollTop = termBody.scrollHeight;
  },

  renderAll() {
    const termBody = document.getElementById('terminalBody');
    if (!termBody) return;
    termBody.innerHTML = '';
    this.history.forEach(item => {
      if (this.currentFilter === 'all' || this.currentFilter === item.category) {
        this.renderLine(item);
      }
    });
  },

  updateCount() {
    const countEl = document.getElementById('logCountText');
    if (countEl) {
      countEl.textContent = `${this.history.length} dòng`;
    }
  }
};
