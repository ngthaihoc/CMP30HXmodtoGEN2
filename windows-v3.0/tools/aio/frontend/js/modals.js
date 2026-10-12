// Modal & Safety Confirmation System
const Modal = {
  activeTimer: null,

  show({ title, subtitle, level = 'warning', description, rollback, requiresPhrase = false, phrase = '', onConfirm }) {
    const backdrop = document.getElementById('riskModal');
    const titleEl = document.getElementById('modalTitle');
    const subEl = document.getElementById('modalSubtitle');
    const iconBadge = document.getElementById('modalIconBadge');
    const descEl = document.getElementById('modalDescription');
    const rollbackEl = document.getElementById('modalRollback');
    const phraseBox = document.getElementById('modalPhraseBox');
    const phraseInput = document.getElementById('modalPhraseInput');
    const btnConfirm = document.getElementById('modalBtnConfirm');
    const btnCancel = document.getElementById('modalBtnCancel');

    if (!backdrop) return;

    if (this.activeTimer) {
      clearInterval(this.activeTimer);
      this.activeTimer = null;
    }

    titleEl.textContent = title || "Cảnh Báo Thao Tác An Toàn";
    subEl.textContent = subtitle || "Vui lòng xem xét các rủi ro hệ thống trước khi tiếp tục";
    descEl.innerHTML = description || "";
    rollbackEl.textContent = rollback ? `Khôi phục: ${rollback}` : "Có thể khôi phục từ cài đặt hoặc khởi động lại máy.";

    iconBadge.className = `modal-icon-badge ${level}`;
    iconBadge.textContent = level === 'crimson' ? '⚠️' : '⚡';

    btnConfirm.className = level === 'crimson' ? 'btn btn-crimson' : 'btn btn-primary';

    if (requiresPhrase) {
      phraseBox.style.display = 'block';
      phraseInput.value = '';
      btnConfirm.disabled = true;
      btnConfirm.textContent = "Xác nhận";

      phraseInput.oninput = () => {
        btnConfirm.disabled = phraseInput.value.trim() !== phrase;
      };
    } else {
      phraseBox.style.display = 'none';
      btnConfirm.disabled = true;
      let count = 2;
      btnConfirm.textContent = `Đợi (${count}s)...`;

      this.activeTimer = setInterval(() => {
        count--;
        if (count <= 0) {
          clearInterval(this.activeTimer);
          this.activeTimer = null;
          btnConfirm.disabled = false;
          btnConfirm.textContent = "Tôi hiểu và đồng ý";
        } else {
          btnConfirm.textContent = `Đợi (${count}s)...`;
        }
      }, 1000);
    }

    const close = () => {
      if (this.activeTimer) {
        clearInterval(this.activeTimer);
        this.activeTimer = null;
      }
      backdrop.classList.add('hidden');
    };

    btnCancel.onclick = close;

    btnConfirm.onclick = () => {
      close();
      if (typeof onConfirm === 'function') {
        onConfirm(requiresPhrase ? phraseInput.value.trim() : null);
      }
    };

    backdrop.classList.remove('hidden');
  }
};
