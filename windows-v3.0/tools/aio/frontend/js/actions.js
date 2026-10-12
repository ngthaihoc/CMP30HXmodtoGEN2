// User Actions, Tabs & Go Bindings Dispatcher
const Actions = {
  init() {
    this.bindTabs();
    this.bindButtons();
    this.refreshAudit();
  },

  bindTabs() {
    const tabs = document.querySelectorAll('.tab-btn');
    tabs.forEach(tab => {
      tab.addEventListener('click', () => {
        tabs.forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        const target = tab.getAttribute('data-tab');
        document.querySelectorAll('.tool-panel').forEach(p => p.classList.remove('active'));
        const panel = document.getElementById(`panel_${target}`);
        if (panel) panel.classList.add('active');
      });
    });
  },

  bindButtons() {
    // 1. Quét lại môi trường
    const btnScan = document.getElementById('btnRefreshScan');
    if (btnScan) {
      btnScan.addEventListener('click', () => this.refreshAudit());
    }

    // 2. Mở khoá Gen2 ngay
    const btnUnlock = document.getElementById('btnUnlockGen2');
    if (btnUnlock) {
      btnUnlock.addEventListener('click', () => {
        Modal.show({
          title: "Mở Khóa PCIe Gen2 x16",
          subtitle: "Thiết lập tốc độ Gen2 cho phiên làm việc hiện tại",
          level: "amber",
          description: "Thao tác sẽ nạp chuỗi thanh ghi MMIO và ép cổng Root Port huấn luyện lại liên kết lên Gen2 x16.<br>Màn hình có thể nhấp nháy nhẹ trong 0.5 giây.",
          rollback: "Khởi động lại máy tính sẽ khôi phục liên kết mặc định của phần cứng.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.UnlockGen2Now().then(res => {
                Logger.append(`[KẾT QUẢ] ${res.message}`);
                this.refreshAudit();
              });
            }
          }
        });
      });
    }

    // 2b. Ép mở khoá Root Gen2
    const btnForceRoot = document.getElementById('btnForceRootGen2');
    if (btnForceRoot) {
      btnForceRoot.addEventListener('click', () => {
        Modal.show({
          title: "Ép Mở Khóa Gen2 Qua Root Port",
          subtitle: "Bỏ qua trạng thái LNKCAP và ghi đè thanh ghi bóng MMIO",
          level: "amber",
          description: "Sử dụng khi card đang bị kẹt ở Gen1 sau khi khởi động lại hoặc khi driver nạp trễ.<br>Cơ chế sẽ ghi đè TU106/TU116 Shadow Sequence và phát xung retrain trực tiếp.",
          rollback: "Khởi động lại máy tính để hoàn tác.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.ForceRootGen2().then(res => {
                Logger.append(`[KẾT QUẢ] ${res.message}`);
                this.refreshAudit();
              });
            }
          }
        });
      });
    }

    // 3. Tác vụ tự khởi động khi đăng nhập
    const btnTaskReg = document.getElementById('btnRegisterTask');
    if (btnTaskReg) {
      btnTaskReg.addEventListener('click', () => {
        Modal.show({
          title: "Đăng Ký Tự Khởi Động Khi Đăng Nhập",
          subtitle: "Tạo tác vụ Task Scheduler SYSTEM và Registry Run Key",
          level: "amber",
          description: "Đăng ký tác vụ tự động mở khóa Gen2 mỗi khi đăng nhập Windows (chạy ẩn quyền SYSTEM, không hiện cửa sổ console).",
          rollback: "Có thể gỡ bỏ bất kỳ lúc nào bằng nút [Gỡ Tác Vụ] bên cạnh.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.RegisterAutostartTask().then(() => this.refreshAudit());
            }
          }
        });
      });
    }

    const btnTaskUnreg = document.getElementById('btnUnregisterTask');
    if (btnTaskUnreg) {
      btnTaskUnreg.addEventListener('click', () => {
        if (window.go && window.go.main && window.go.main.App) {
          window.go.main.App.UnregisterAutostartTask().then(() => this.refreshAudit());
        }
      });
    }

    // 4. Riot Games Optimize
    const btnRiotOpt = document.getElementById('btnRiotOptimize');
    if (btnRiotOpt) {
      btnRiotOpt.addEventListener('click', () => {
        Modal.show({
          title: "Tối Ưu Hoá Riot Games (1-Chạm)",
          subtitle: "Cấu hình MSHybrid CASO, DirectX Router & Dọn dẹp driver tạm",
          level: "amber",
          description: "<strong>CÁC THAY ĐỔI HỆ THỐNG:</strong><br>" +
            "1. Tắt <code>VulnerableDriverBlocklistEnable=0</code> để tránh chặn driver mod.<br>" +
            "2. Gán CMP làm <code>HighPerfAdapter</code> toàn cục cho DirectX.<br>" +
            "3. Cấu hình game.cfg Liên Minh Huyền Thoại sang <code>WindowMode=2</code> và <code>DX11Support=1</code>.<br>" +
            "4. Dọn dẹp triệt để các driver tạm thời trong kernel.",
          rollback: "Bấm nút [Hoàn Tác Riot] để bật lại Blocklist và xoá định tuyến DirectX.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.RiotOptimize().then(() => this.refreshRiot());
            }
          }
        });
      });
    }

    const btnRiotRevert = document.getElementById('btnRiotRevert');
    if (btnRiotRevert) {
      btnRiotRevert.addEventListener('click', () => {
        Modal.show({
          title: "Hoàn Tác Cấu Hình Riot Games",
          subtitle: "Khôi phục cài đặt bảo mật mặc định",
          level: "amber",
          description: "Thao tác sẽ bật lại <code>VulnerableDriverBlocklistEnable=1</code> và xoá bỏ cấu hình định tuyến DirectX toàn cục.",
          rollback: "Có thể bấm [Tối Ưu 1-Chạm] để cấu hình lại bất cứ lúc nào.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.RiotRevertOptimization().then(() => this.refreshRiot());
            }
          }
        });
      });
    }

    // 5. Ký Chữ Ký Số EFI (Yêu cầu gõ cụm từ xác nhận)
    const btnRiotSign = document.getElementById('btnRiotSignEFI');
    if (btnRiotSign) {
      btnRiotSign.addEventListener('click', () => {
        Modal.show({
          title: "Ký Chữ Ký Số UEFI Cho 40HXUNLK.EFI",
          subtitle: "Tạo chứng chỉ cá nhân và ký Authenticode PE",
          level: "crimson",
          description: "<strong>CẢNH BÁO NGUY HIỂM:</strong><br>" +
            "Thao tác này dành cho CMP 40HX trên Windows 11 khi bật Secure Boot.<br>" +
            "Nạp chứng chỉ vào BIOS db nếu cấu hình sai có thể gây từ chối khởi động.<br>" +
            "Yêu cầu gõ chính xác cụm từ xác nhận bên dưới để mở khóa thực hiện.",
          rollback: "Xoá file C:\\CMP40HX_Key.cer và xoá Key khỏi BIOS Key Management.",
          requiresPhrase: true,
          phrase: "XAC NHAN KY EFI",
          onConfirm: (typed) => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.RiotSignEFI(typed);
            }
          }
        });
      });
    }

    // 6. Khởi động lại BIOS
    const btnRebootBios = document.getElementById('btnRebootBIOS');
    if (btnRebootBios) {
      btnRebootBios.addEventListener('click', () => {
        Modal.show({
          title: "Khởi Động Lại Vào BIOS Setup",
          subtitle: "Chuyển thẳng vào màn hình cấu hình UEFI Firmware",
          level: "amber",
          description: "Máy tính sẽ tự động khởi động lại sau 3 giây và vào thẳng giao diện BIOS Setup.",
          rollback: "Không có rủi ro dữ liệu.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.RebootToBIOS();
            }
          }
        });
      });
    }

    // 7. Sửa lỗi CSM Code 43
    const btnFixCsm = document.getElementById('btnFixCSM');
    if (btnFixCsm) {
      btnFixCsm.addEventListener('click', () => {
        Modal.show({
          title: "Khắc Phục Lỗi CSM Code 43",
          subtitle: "Quét lại bus PCI & Khoá chống ngủ D3Cold",
          level: "amber",
          description: "Sửa lỗi card đồ họa biến mất hoặc báo Code 43 trên bo mạch chủ bật CSM Legacy.<br>Vô hiệu hoá D3Cold và kéo dài thời gian chờ TDR lên 10 giây.",
          rollback: "Có thể khôi phục từ cài đặt nguồn hoặc gỡ driver.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.FixCSMError43().then(res => {
                Logger.append(`[CSM] ${res.message}`);
                this.refreshAudit();
              });
            }
          }
        });
      });
    }

    // 8. Setup ReBAR (Turing Resizable BAR)
    const btnSetupRebar = document.getElementById('btnSetupReBar');
    if (btnSetupRebar) {
      btnSetupRebar.addEventListener('click', () => {
        Modal.show({
          title: "Kích Hoạt Resizable BAR Cho CMP 30HX",
          subtitle: "Tối ưu hóa VRAM qua NvStrapsReBar & Profile Inspector",
          level: "amber",
          description: "Kích hoạt cờ ReBAR Turing trong profile driver và nạp NvStrapsReBar.<br>" +
            "<strong>LƯU Ý:</strong> Công cụ có khóa an toàn tự động phát hiện Laptop và CHẶN 100% để chống brick BIOS.",
          rollback: "Bấm nút [Hoàn Tác ReBAR] để khôi phục cấu hình mặc định.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.SetupReBar(6).then(res => {
                Logger.append(`[ReBAR] ${res.message}`);
              });
            }
          }
        });
      });
    }

    const btnRevertRebar = document.getElementById('btnRevertReBar');
    if (btnRevertRebar) {
      btnRevertRebar.addEventListener('click', () => {
        if (window.go && window.go.main && window.go.main.App) {
          window.go.main.App.RevertReBar().then(res => {
            Logger.append(`[ReBAR] ${res.message}`);
          });
        }
      });
    }

    // 9. Gỡ cài đặt hoàn chỉnh
    const btnUninstall = document.getElementById('btnUninstallAll');
    if (btnUninstall) {
      btnUninstall.addEventListener('click', () => {
        Modal.show({
          title: "Gỡ Bỏ Toàn Bộ Cài Đặt (Uninstall All)",
          subtitle: "Xóa sạch tác vụ, khóa registry và khôi phục hệ thống",
          level: "amber",
          description: "Thao tác sẽ xoá bỏ toàn bộ:<br>" +
            "- Mọi tác vụ Task Scheduler đã đăng ký<br>" +
            "- Khóa tự chạy Registry Run và cấu hình HKLM\\SOFTWARE\\40HXUnlock<br>" +
            "- Mục khởi động UEFI '40HX Unlock' trong BIOS<br>" +
            "- Dừng và xóa driver kernel tạm thời khỏi System32\\drivers",
          rollback: "Có thể chạy lại cài đặt bất kỳ lúc nào.",
          onConfirm: () => {
            if (window.go && window.go.main && window.go.main.App) {
              window.go.main.App.UninstallAll().then(res => {
                Logger.append(`[GỠ CÀI ĐẶT] ${res.message}`);
                this.refreshAudit();
              });
            }
          }
        });
      });
    }

    // 10. HVCI Mở Core Isolation
    const btnOpenHvci = document.getElementById('btnOpenHvciSettings');
    if (btnOpenHvci) {
      btnOpenHvci.addEventListener('click', () => {
        if (window.go && window.go.main && window.go.main.App) {
          window.go.main.App.OpenCoreIsolationSettings();
        }
      });
    }
  },

  refreshAudit() {
    if (window.go && window.go.main && window.go.main.App) {
      window.go.main.App.ScanEnvironment().then(rep => this.renderAudit(rep));
      window.go.main.App.CheckHVCIStatus().then(hvci => this.renderHVCI(hvci));
      this.refreshRiot();
    }
  },

  renderAudit(rep) {
    if (!rep) return;
    const listEl = document.getElementById('auditItemsList');
    if (!listEl) return;
    listEl.innerHTML = '';

    (rep.items || []).forEach(item => {
      const row = document.createElement('div');
      row.className = 'audit-row';
      row.innerHTML = `
        <div class="audit-name-col">
          <div class="audit-icon ${item.ok ? 'ok' : 'warn'}">${item.ok ? '✓' : '!'}</div>
          <strong>${item.name}</strong>
        </div>
        <div class="audit-note mono">${item.note}</div>
      `;
      listEl.appendChild(row);
    });

    const guardStatus = document.getElementById('guardStatusText');
    if (guardStatus) {
      guardStatus.textContent = rep.isGen2 ? "Đang bật (Tự mở khóa khi Logon)" : "Chưa đăng ký";
      guardStatus.style.color = rep.isGen2 ? "var(--accent-emerald)" : "var(--accent-muted)";
    }
  },

  renderHVCI(hvci) {
    const box = document.getElementById('hvciAlertBox');
    const text = document.getElementById('hvciStatusText');
    const desc = document.getElementById('hvciGuideText');
    if (!box) return;

    if (hvci && hvci.enabled) {
      box.style.display = 'flex';
      if (text) text.textContent = hvci.statusText;
      if (desc) desc.textContent = hvci.guideText;
    } else {
      box.style.display = 'none';
    }
  },

  refreshRiot() {
    if (window.go && window.go.main && window.go.main.App) {
      window.go.main.App.GetRiotStatus().then(st => {
        const lolEl = document.getElementById('riotLolStatus');
        const valEl = document.getElementById('riotValStatus');
        const recEl = document.getElementById('riotRecText');

        if (lolEl) {
          lolEl.textContent = st.canPlayLeagueOfLegends ? "✓ Sẵn sàng" : "Chưa tương thích";
          lolEl.style.color = st.canPlayLeagueOfLegends ? "var(--accent-emerald)" : "var(--accent-amber)";
        }
        if (valEl) {
          valEl.textContent = st.canPlayValorant ? "✓ Sẵn sàng" : "⚠️ Cần kiểm tra";
          valEl.style.color = st.canPlayValorant ? "var(--accent-emerald)" : "var(--accent-amber)";
        }
        if (recEl) {
          recEl.textContent = st.recommendation || "--";
        }
      });
    }
  }
};
