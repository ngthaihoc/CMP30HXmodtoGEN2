#!/usr/bin/env bash
# ==============================================================================
# CONG CU CAI DAT TU DONG GEN2 X16 CHO NVIDIA CMP 30HX (TU116) TREN LINUX (AIO)
# Ho tro: Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux, v.v.
# Tac gia: ngthaihoc (https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
# Phien ban: 3.1.4
# ==============================================================================

set -uo pipefail

VERSION="3.1.4"
SYSTEMD_SVC_NAME="cmp30hx-gen2-unlock.service"
SYSTEMD_SVC_PATH="/etc/systemd/system/${SYSTEMD_SVC_NAME}"
BIN_INSTALL_PATH="/usr/local/bin/cmp30hx-unlock"
SUSPEND_HOOK_PATH="/lib/systemd/system-sleep/cmp30hx-unlock"
MMIO_INSTALL_PATH="/usr/local/bin/linux-mmio/mmio_injector"

# Bien danh cho kiem thu va mo phong (Mock Testing)
IS_MOCK=0
IS_MOCK_FAIL=0
IS_MOCK_IDLE=0
IS_MOCK_NOGPU=0
MOCK_FAIL=0
MOCK_IDLE=0
MOCK_NOGPU=0
NO_ROOT=0
CUSTOM_STATUS_FILE=""

# Xac dinh duong dan file Status Contract (Seam 2)
get_status_file() {
    if [[ -n "${CUSTOM_STATUS_FILE}" ]]; then
        echo "${CUSTOM_STATUS_FILE}"
        return 0
    fi
    if [[ -w "/var/run" || "${EUID:-$(id -u)}" -eq 0 ]]; then
        echo "/var/run/40hxunlock/gen2_status.txt"
    else
        echo "/tmp/40hxunlock/gen2_status.txt"
    fi
}

# Ghi file trang thai gen2_status.txt theo chuan StatusContract
write_status_file() {
    local code="$1"
    local speed="$2"
    local width="$3"
    local tls="$4"
    local err="${5:-NONE}"
    local detail="${6:-}"
    local target_file
    target_file="$(get_status_file)"
    local target_dir
    target_dir="$(dirname "${target_file}")"

    mkdir -p "${target_dir}" 2>/dev/null || true
    cat <<EOF > "${target_file}" 2>/dev/null || true
STATUS_CODE=${code}
SPEED_CURRENT=${speed}
WIDTH_CURRENT=${width}
TLS_TARGET=${tls}
ERROR_CODE=${err}
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

[Chi Tiet Trang Thai]
${detail}
EOF
}

# Mau sac hien thi terminal (tu dong tat neu khong phai terminal tuong tac)
if [[ -t 1 ]]; then
    C_RESET="\033[0m"
    C_BOLD="\033[1m"
    C_GREEN="\033[32m"
    C_YELLOW="\033[33m"
    C_BLUE="\033[36m"
    C_RED="\033[31m"
else
    C_RESET=""
    C_BOLD=""
    C_GREEN=""
    C_YELLOW=""
    C_BLUE=""
    C_RED=""
fi

# ------------------------------------------------------------------------------
# Kiem tra quyen root / sudo
# ------------------------------------------------------------------------------
ensure_root() {
    if [[ "${NO_ROOT}" -eq 1 ]]; then
        return 0
    fi
    if [[ "${EUID}" -ne 0 ]]; then
        echo -e "${C_YELLOW}[!] Yeu cau quyen root / sudo de can thiep PCI va thanh ghi kernel.${C_RESET}"
        if command -v sudo >/dev/null 2>&1; then
            exec sudo -- "$0" "$@"
        else
            echo -e "${C_RED}[X] Khong tim thay 'sudo'. Vui long dang nhap root va chay lai.${C_RESET}" >&2
            exit 1
        fi
    fi
}

# ------------------------------------------------------------------------------
# Hien thi Huong dan su dung
# ------------------------------------------------------------------------------
usage() {
    echo "Su dung: $0 [TUY_CHON]"
    echo ""
    echo "Tuy chon:"
    echo "  (khong tham so)    Cai dat toan dien: Mo khoa Gen2 x16, MRRS 512B va tao systemd service"
    echo "  -s, --status       Đọc trạng thái liên kết PCIe của CMP 30HX hoặc 40HX"
    echo "  -u, --uninstall    Go bo service, sleep hook va tep cua cong cu"
    echo "  -d, --daemon       Che do chay ngam (danh cho systemd service luc khoi dong)"
    echo "  -test, --test      Kiem thu mo phong thanh cong (Mock Happy Path: Gen1 -> Gen2)"
    echo "  -mock-fail         Kiem thu mo phong that bai (Mock Failure: Retrain Timeout)"
    echo "  -mock-idle         Kiem thu mo phong che do idle (Mock Idle: TLS=Gen2, current=Gen1)"
    echo "  -mock-nogpu        Kiem thu mo phong khong tim thay GPU (Mock No GPU)"
    echo "  --no-root          Bo qua kiem tra quyen root (danh cho moi truong test CI/WSL)"
    echo "  -h, --help         Hien thi thong tin huong dan nay"
    echo ""
    exit "${1:-0}"
}

# ------------------------------------------------------------------------------
# Kiểm tra các công cụ của pciutils.
# ------------------------------------------------------------------------------
check_dependencies() {
    if [[ "${IS_MOCK}" -eq 1 ]]; then
        return 0
    fi
    local missing=()
    if ! command -v lspci >/dev/null 2>&1; then missing+=("pciutils (lspci)"); fi
    if ! command -v setpci >/dev/null 2>&1; then missing+=("pciutils (setpci)"); fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        echo -e "${C_RED}[X] Thieu cac goi can thiet: ${missing[*]}${C_RESET}"
        echo "    Vui long cai dat bang trinh quan ly goi cua ban:"
        echo "      - Debian/Ubuntu/HiveOS : apt-get update && apt-get install -y pciutils"
        echo "      - RHEL/CentOS/Fedora    : dnf install -y pciutils"
        echo "      - Arch Linux            : pacman -S pciutils"
        exit 1
    fi
}

# ------------------------------------------------------------------------------
# Tim offset cua PCIe Capability (Cap ID 0x10) - Kiem tra an toan gia tri tra ve
# ------------------------------------------------------------------------------
get_pcie_cap_offset() {
    local bdf="$1"
    if [[ "${IS_MOCK}" -eq 1 ]]; then
        echo "CAP_EXP"
        return 0
    fi
    local cap_id ptr cur tries=0
    if cap_id=$(setpci -s "$bdf" CAP_EXP+0x00.b 2>&1) && [[ "$cap_id" == 10 ]]; then
        echo "CAP_EXP"
        return 0
    fi

    # Duyệt danh sách capability khi pciutils không hỗ trợ bí danh CAP_EXP.
    ptr=$(setpci -s "$bdf" 0x34.b 2>&1) || return 1
    [[ "$ptr" =~ ^[0-9a-fA-F]{2}$ ]] || return 1
    cur=$((16#$ptr))
    while (( cur >= 0x40 && cur <= 0xfc && cur % 4 == 0 && tries < 48 )); do
        cap_id=$(setpci -s "$bdf" "$(printf '%02x.b' "$cur")" 2>&1) || return 1
        [[ "$cap_id" =~ ^[0-9a-fA-F]{2}$ && "${cap_id,,}" != ff ]] || return 1
        if [[ "$cap_id" == 10 ]]; then
            printf '0x%02x\n' "$cur"
            return 0
        fi
        ptr=$(setpci -s "$bdf" "$(printf '%02x.b' "$((cur + 1))")" 2>&1) || return 1
        [[ "$ptr" =~ ^[0-9a-fA-F]{2}$ ]] || return 1
        cur=$((16#$ptr))
        tries=$((tries + 1))
    done
    return 1
}

# ------------------------------------------------------------------------------
# Tim Root Port / Upstream Bridge cua thiet bi GPU
# ------------------------------------------------------------------------------
find_upstream_root_port() {
    local gpu_bdf="$1" # e.g. 0000:01:00.0 or 01:00.0
    if [[ "${IS_MOCK}" -eq 1 ]]; then
        echo "0000:00:01.0"
        return 0
    fi
    local sys_dev="/sys/bus/pci/devices/${gpu_bdf}"
    if [[ ! -d "${sys_dev}" ]]; then
        sys_dev="/sys/bus/pci/devices/0000:${gpu_bdf}"
    fi

    if [[ -d "${sys_dev}" ]]; then
        local parent_dir
        parent_dir="$(dirname "$(readlink -f "${sys_dev}")")"
        local parent_bdf
        parent_bdf="$(basename "${parent_dir}")"
        # BDF chuan dang [0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]
        if [[ "${parent_bdf}" =~ ^[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]$ ]]; then
            echo "${parent_bdf}"
            return 0
        fi
    fi

    # Du phong: dung lspci -s "$gpu_bdf" -PP de lay duong dan phan cap
    local pp
    pp=$(lspci -s "${gpu_bdf}" -PP 2>/dev/null | awk '{print $1}')
    if [[ -n "${pp}" && "${pp}" == *"/"* ]]; then
        local upstream
        upstream=$(echo "${pp}" | rev | cut -d'/' -f2 | rev)
        if [[ -n "${upstream}" ]]; then
            echo "${upstream}"
            return 0
        fi
    fi

    echo ""
    return 1
}

# ------------------------------------------------------------------------------
# Cấu hình các thanh ghi MMIO BAR0 của CMP 30HX.
# Trình ghi MMIO kiểm tra BAR0 trước khi ghi hoặc đọc lại thanh ghi.
# ------------------------------------------------------------------------------
inject_bar0_mmio() {
    local gpu_bdf="$1"
    local sys_dev="/sys/bus/pci/devices/${gpu_bdf}"
    if [[ ! -d "${sys_dev}" ]]; then
        sys_dev="/sys/bus/pci/devices/0000:${gpu_bdf}"
    fi

    local res0="${sys_dev}/resource0"
    if [[ ! -e "${res0}" ]]; then
        echo "[X] Không tìm thấy BAR0 tại ${res0}." >&2
        return 1
    fi

    local injector_bin
    injector_bin="$(dirname -- "$(readlink -f -- "${BASH_SOURCE[0]}")")/linux-mmio/mmio_injector"
    if [[ ! -x "${injector_bin}" ]]; then
        echo "[X] Không tìm thấy tệp thực thi mmio_injector tại ${injector_bin}." >&2
        return 1
    fi

    echo "    [*] Kiểm tra và cấu hình BAR0 của CMP 30HX."
    "${injector_bin}" "${res0}"
}

read_pcie_word() {
    local bdf="$1" register="$2" raw
    if ! raw=$(setpci -s "$bdf" "$register" 2>&1) ||
        [[ ! "$raw" =~ ^[0-9a-fA-F]{4}$ || "${raw,,}" == ffff ]]; then
        echo "[X] Không đọc được thanh ghi PCIe $register trên $bdf." >&2
        return 1
    fi
    printf '%d\n' "$((16#$raw))"
}

write_pcie_bits() {
    local bdf="$1" register="$2" value="$3" mask="$4" output
    # setpci có thể chỉ báo lỗi I/O trên stderr dù mã thoát bằng 0.
    if ! output=$(setpci -s "$bdf" "$register=$(printf '%04x:%04x' "$value" "$mask")" 2>&1) ||
        [[ -n "$output" ]]; then
        echo "[X] Không ghi được thanh ghi PCIe $register trên $bdf." >&2
        return 1
    fi
}

set_pcie_bits() {
    local bdf="$1" register="$2" value="$3" mask="$4" current
    current=$(read_pcie_word "$bdf" "$register") || return 1
    if (( (current & mask) != value )); then
        write_pcie_bits "$bdf" "$register" "$value" "$mask" || return 1
        current=$(read_pcie_word "$bdf" "$register") || return 1
        if (( (current & mask) != value )); then
            echo "[X] Giá trị đọc lại của $register trên $bdf khác giá trị yêu cầu." >&2
            return 1
        fi
    fi
}

read_pcie_link() {
    local bdf="$1" cap="$2" status control
    status=$(read_pcie_word "$bdf" "$cap+0x12.w") || return 1
    control=$(read_pcie_word "$bdf" "$cap+0x30.w") || return 1
    printf '%d %d %d %d\n' "$((status & 0xf))" "$(((status >> 4) & 0x3f))" "$((control & 0xf))" "$((status & 0x0800))"
}

pcie_link_result() {
    local speed="$1" width="$2" tls="$3" training="$4"
    (( tls == 2 && width == 16 && training == 0 )) || return 1
    if (( speed == 2 )); then
        return 0
    elif (( speed == 1 )); then
        return 2
    fi
    return 1
}

# ------------------------------------------------------------------------------
# Cấu hình LNKCTL2, DEVCTL, LNKCTL và huấn luyện lại liên kết.
# ------------------------------------------------------------------------------
unlock_gpu_pcie() {
    local gpu_bdf="$1"
    echo -e "${C_BOLD}----------------------------------------------------------------${C_RESET}"
    echo -e "${C_BLUE}[*] Dang xu ly GPU CMP 30HX: ${gpu_bdf}${C_RESET}"

    if [[ "${IS_MOCK}" -eq 1 ]]; then
        echo -e "    [*] [MOCK] Tim thay Upstream Root Port: 0000:00:01.0 (Cap: CAP_EXP)"
        echo -e "    [OK] GPU LNKCTL2 TLS -> Gen2 (0x0001 -> 0x0002)"
        echo -e "    [OK] Root Port LNKCTL2 TLS -> Gen2 (0x0001 -> 0x0002)"
        echo -e "    [OK] Tối ưu MRRS GPU: 128B -> 512B (DEVCTL: 0x0000 -> 0x2000)"
        echo -e "    [*] Bat dau chu trinh huan luyen lai link PCIe (toi da 6 luot)..."

        if [[ "${MOCK_FAIL}" -eq 1 ]]; then
            echo -e "    [-] Luot #1 (ROOT): Dang o Gen1 x16, dang thu lai..."
            echo -e "    [-] Luot #2 (GPU): Dang o Gen1 x16, dang thu lai..."
            echo -e "    ${C_RED}[X] Không đạt Gen2 sau các lần huấn luyện lại.${C_RESET}"
            return 1
        elif [[ "${MOCK_IDLE}" -eq 1 ]]; then
            echo -e "    [-] Luot #1 (ROOT): Dang o Gen1 x16, dang thu lai..."
            echo -e "    [-] Luot #2 (GPU): Dang o Gen1 x16, dang thu lai..."
            echo -e "    ${C_YELLOW}[!] TLS da thiet lap Gen2 nhung Link o Gen1 do tiet kiem nang luong khi idle.${C_RESET}"
            return 2
        else
            echo -e "    ${C_GREEN}[V] Luot #1 (ROOT): DA DAT GEN2 x16!${C_RESET}"
            return 0
        fi
    fi

    local gpu_cap root_bdf root_cap=""
    if ! gpu_cap=$(get_pcie_cap_offset "$gpu_bdf"); then
        echo "[X] Không tìm thấy PCIe Capability trên GPU $gpu_bdf." >&2
        return 1
    fi
    if root_bdf=$(find_upstream_root_port "$gpu_bdf"); then
        if ! root_cap=$(get_pcie_cap_offset "$root_bdf"); then
            echo "[X] Không xác minh được PCIe Capability của cổng upstream $root_bdf." >&2
            return 1
        fi
        echo "    [*] Cổng upstream: $root_bdf ($root_cap)."
    else
        echo "    [!] Không tìm thấy cổng upstream. Chỉ cấu hình GPU."
    fi

    inject_bar0_mmio "$gpu_bdf" || return 1

    # LNKCTL2 TLS = 2, giữ các bit khác.
    set_pcie_bits "$gpu_bdf" "$gpu_cap+0x30.w" 2 0xf || return 1
    if [[ -n "$root_cap" ]]; then
        set_pcie_bits "$root_bdf" "$root_cap+0x30.w" 2 0xf || return 1
    fi

    # Giữ MRRS hiện tại nếu đã từ 512 byte trở lên.
    local devctl
    devctl=$(read_pcie_word "$gpu_bdf" "$gpu_cap+0x08.w") || return 1
    if (( (devctl & 0x7000) < 0x2000 )); then
        set_pcie_bits "$gpu_bdf" "$gpu_cap+0x08.w" 0x2000 0x7000 || return 1
    fi

    # Giữ cấu hình Common Clock/CLKREQ hiện hành của công cụ khi tắt ASPM.
    set_pcie_bits "$gpu_bdf" "$gpu_cap+0x10.w" 0x0140 0x0143 || return 1
    if [[ -n "$root_cap" ]]; then
        set_pcie_bits "$root_bdf" "$root_cap+0x10.w" 0x0140 0x0143 || return 1
    fi

    echo "    [*] Huấn luyện lại liên kết PCIe (tối đa 6 lần)."
    local attempt target_bdf target_cap link speed width tls training result
    for ((attempt = 1; attempt <= 6; attempt++)); do
        target_bdf="$gpu_bdf"
        target_cap="$gpu_cap"
        if [[ -n "$root_cap" ]] && (( attempt % 2 == 1 )); then
            target_bdf="$root_bdf"
            target_cap="$root_cap"
        fi

        # Retrain là bit điều khiển tự xóa. Xác nhận bằng Link Status sau khi ghi.
        write_pcie_bits "$target_bdf" "$target_cap+0x10.w" 0 0x0020 || return 1
        sleep 0.3
        write_pcie_bits "$target_bdf" "$target_cap+0x10.w" 0x0020 0x0020 || return 1
        sleep 2.2

        link=$(read_pcie_link "$gpu_bdf" "$gpu_cap") || return 1
        read -r speed width tls training <<< "$link"
        if pcie_link_result "$speed" "$width" "$tls" "$training"; then
            return 0
        fi
        echo "    [-] Lần $attempt: Gen$speed x$width (TLS=$tls)."
    done

    # Không gỡ thiết bị khỏi bus hay dừng tiến trình GPU khi retrain không đạt.
    result=0
    pcie_link_result "$speed" "$width" "$tls" "$training" || result=$?
    return "$result"
}

# ------------------------------------------------------------------------------
# Đăng ký dịch vụ systemd và sleep hook.
# ------------------------------------------------------------------------------
install_systemd_service() {
    echo -e "${C_BOLD}Đang đăng ký dịch vụ systemd và sleep hook...${C_RESET}"
    if [[ "${IS_MOCK}" -eq 1 ]]; then
        echo -e "      ${C_GREEN}[OK]${C_RESET} [MOCK] Da tao Systemd Service: ${SYSTEMD_SVC_NAME}"
        echo -e "      ${C_GREEN}[OK]${C_RESET} [MOCK] Da tao Sleep/Resume Hook: ${SUSPEND_HOOK_PATH}"
        return 0
    fi

    # Chep script hien tai vao duong dan he thong de service chay on dinh
    local current_script
    current_script="$(readlink -f "$0")"

    if [[ "${current_script}" != "${BIN_INSTALL_PATH}" ]]; then
        cp -f "${current_script}" "${BIN_INSTALL_PATH}" || return 1
        chmod +x "${BIN_INSTALL_PATH}" || return 1
        echo -e "      [OK] Da sao chep script den: ${BIN_INSTALL_PATH}"

        local injector_dir="$(dirname "${current_script}")/linux-mmio"
        if [[ ! -f "${injector_dir}/mmio_injector" ]]; then
            echo -e "      ${C_RED}[X] Khong tim thay mmio_injector trong goi cai dat.${C_RESET}"
            return 1
        fi
        mkdir -p "$(dirname "${MMIO_INSTALL_PATH}")" || return 1
        cp -f "${injector_dir}/mmio_injector" "${MMIO_INSTALL_PATH}" || return 1
        chmod +x "${MMIO_INSTALL_PATH}" || return 1
    elif [[ ! -x "${MMIO_INSTALL_PATH}" ]]; then
        echo -e "      ${C_RED}[X] mmio_injector da cai khong ton tai hoac khong the chay.${C_RESET}"
        return 1
    fi

    # Tao file systemd service (chay truoc display-manager de tranh xung dot driver)
    cat <<EOF > "${SYSTEMD_SVC_PATH}" || return 1
[Unit]
Description=NVIDIA CMP 30HX PCIe Gen2 x16 Unlock Service
DefaultDependencies=no
After=local-fs.target systemd-modules-load.service
Before=display-manager.service graphical.target multi-user.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=${BIN_INSTALL_PATH} --daemon

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "${SYSTEMD_SVC_PATH}" || return 1

    # Tao hook phuc hoi sau Sleep / Suspend / Hibernate
    mkdir -p "$(dirname "${SUSPEND_HOOK_PATH}")" || return 1
    cat <<EOF > "${SUSPEND_HOOK_PATH}" || return 1
#!/bin/sh
case "\$1/\$2" in
  post/*)
    ${BIN_INSTALL_PATH} --daemon >/dev/null 2>&1
    ;;
esac
EOF
    chmod +x "${SUSPEND_HOOK_PATH}" || return 1

    if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
        systemctl daemon-reload >/dev/null 2>&1 || return 1
        systemctl enable "${SYSTEMD_SVC_NAME}" >/dev/null 2>&1 || return 1
        echo -e "      ${C_GREEN}[OK]${C_RESET} Systemd service '${SYSTEMD_SVC_NAME}' da duoc kich hoat."
        echo -e "      ${C_GREEN}[OK]${C_RESET} Sleep/resume hook da duoc cai dat tai: ${SUSPEND_HOOK_PATH}"
    else
        echo -e "      ${C_RED}[X] He thong khong chay systemd; khong the dang ky tu dong khoi dong.${C_RESET}"
        return 1
    fi
}

# ------------------------------------------------------------------------------
# Go cai dat (Uninstall)
# ------------------------------------------------------------------------------
uninstall_service() {
    echo -e "${C_BOLD}================================================================${C_RESET}"
    echo -e "${C_BOLD}    GO BO TU DONG KHOI DONG CMP 30HX GEN2 UNLOCK TREN LINUX    ${C_RESET}"
    echo -e "${C_BOLD}================================================================${C_RESET}"

    if [[ "${IS_MOCK}" -eq 1 ]]; then
        echo -e "${C_GREEN}[V] [MOCK] Da go bo hoan toan Systemd Service va Sleep Hook.${C_RESET}"
        write_status_file "UNINSTALLED" "1" "16" "1" "NONE" "Da go bo thanh cong service tren Linux."
        exit 0
    fi

    local failed=0
    if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
        if systemctl is-active --quiet "${SYSTEMD_SVC_NAME}"; then
            systemctl stop "${SYSTEMD_SVC_NAME}" >/dev/null 2>&1 || failed=1
        fi
        if systemctl is-enabled --quiet "${SYSTEMD_SVC_NAME}"; then
            systemctl disable "${SYSTEMD_SVC_NAME}" >/dev/null 2>&1 || failed=1
        fi
    fi

    local path
    for path in "${SYSTEMD_SVC_PATH}" "${SUSPEND_HOOK_PATH}" "${BIN_INSTALL_PATH}" "${MMIO_INSTALL_PATH}"; do
        if [[ -e "${path}" || -L "${path}" ]]; then
            if ! rm -f -- "${path}" || [[ -e "${path}" || -L "${path}" ]]; then
                echo -e "${C_RED}[X] Khong the xoa: ${path}${C_RESET}" >&2
                failed=1
            fi
        fi
    done
    rmdir -- "$(dirname "${MMIO_INSTALL_PATH}")" 2>/dev/null || true
    if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
        systemctl daemon-reload >/dev/null 2>&1 || failed=1
    fi
    if [[ "${failed}" -ne 0 ]]; then
        write_status_file "UNINSTALL_FAILED" "0" "0" "0" "ERR_UNINSTALL" "Khong the go het service, hook hoac tep cua cong cu."
        echo -e "${C_RED}[X] Go cai dat chua hoan tat. Kiem tra loi va chay lai.${C_RESET}" >&2
        exit 1
    fi

    echo -e "${C_GREEN}[V] Da go bo Systemd Service va Sleep Hook.${C_RESET}"
    echo "    Khoi dong lai neu can lam moi trang thai PCIe runtime."
    write_status_file "UNINSTALLED" "1" "16" "1" "NONE" "Da go bo thanh cong service tren Linux."
    exit 0
}

# ------------------------------------------------------------------------------
# Hien thi trang thai chi tiet (Status)
# ------------------------------------------------------------------------------
pcie_rate() {
    case "$1" in
        1) echo "2.5 GT/s" ;;
        2) echo "5.0 GT/s" ;;
        3) echo "8.0 GT/s" ;;
        4) echo "16.0 GT/s" ;;
        5) echo "32.0 GT/s" ;;
        6) echo "64.0 GT/s" ;;
        7) echo "128.0 GT/s" ;;
        *) echo "khong ro" ;;
    esac
}

show_status() {
    if [[ "${IS_MOCK}" -ne 1 ]] && { ! command -v lspci >/dev/null || ! command -v setpci >/dev/null; }; then
        echo "[X] Can cai pciutils (lspci va setpci) de xem trang thai PCIe." >&2
        return 1
    fi
    echo -e "${C_BOLD}================================================================${C_RESET}"
    echo -e "${C_BOLD}        TRANG THAI PCIE LINK CUA NVIDIA CMP 30HX / 40HX        ${C_RESET}"
    echo -e "${C_BOLD}================================================================${C_RESET}"

    local gpus=()
    if [[ "${IS_MOCK}" -eq 1 ]]; then
        if [[ "${MOCK_NOGPU}" -eq 1 ]]; then
            gpus=()
        else
            gpus=("0000:01:00.0")
        fi
    else
        while IFS= read -r line; do
            if [[ -n "${line}" ]]; then
                gpus+=("${line}")
            fi
        done < <(lspci -d 10de:2189 2>/dev/null | awk '{print $1}')

        if [[ ${#gpus[@]} -eq 0 ]]; then
            while IFS= read -r line; do
                if [[ -n "${line}" ]]; then
                    gpus+=("${line}")
                fi
            done < <(lspci -d 10de:1f0b 2>/dev/null | awk '{print $1}')
        fi
    fi

    if [[ ${#gpus[@]} -eq 0 ]]; then
        echo -e "${C_YELLOW}[!] Khong tim thay card NVIDIA CMP 30HX (10de:2189) hoac 40HX (10de:1f0b)!${C_RESET}"
        exit 0
    fi

    for gpu in "${gpus[@]}"; do
        echo -e "\n${C_BLUE}--- GPU BDF: ${gpu} ---${C_RESET}"
        if [[ "${IS_MOCK}" -eq 1 ]]; then
            echo "01:00.0 VGA compatible controller: NVIDIA Corporation TU116 [GeForce GTX 1660 SUPER / CMP 30HX] (rev a1)"
            echo ""
            local mock_spd=2
            local mock_tls=2
            if [[ "${MOCK_FAIL}" -eq 1 ]]; then
                mock_spd=1
                mock_tls=1
            elif [[ "${MOCK_IDLE}" -eq 1 ]]; then
                mock_spd=1
                mock_tls=2
            fi
            echo -e "  - Link Speed hien tai    : Gen${mock_spd} ($(pcie_rate "${mock_spd}"))"
            echo -e "  - Link Width hien tai    : x16"
            echo -e "  - Target Link Speed (TLS): Gen${mock_tls}"
            echo -e "  - Max Read Request Size  : 512 Bytes (DEVCTL: 0x2000)"
            echo -e "  - PCIe ASPM trang thai   : Tat (Disabled)"
            echo ""
            echo "  Chi tiet thanh ghi PCIe (lspci):"
            echo "    LnkCap: Port #0, Speed 8GT/s, Width x16, ASPM L0s L1, Exit Latency L0s <512ns, L1 <16us"
            echo "    LnkCtl: ASPM Disabled; Disabled- CommClk+ ExtSynch- ClockPM- AutWidDis- BWInt- AutBWInt-"
            echo "    LnkSta: Speed $(pcie_rate "${mock_spd}"), Width x16, TrErr- Train- SlotClk+ DLActive- BWMgmt- ABWMgmt-"
            echo "    DevCtl: CorrErr- NonFatalErr- FatalErr- UnsupReq- MaxPayload 256 bytes, MaxReadReq 512 bytes"
            echo "    LnkCtl2: Target Link Speed: $(pcie_rate "${mock_tls}"), EnterCompliance- SpeedDis-"
            continue
        fi

        lspci -s "${gpu}"
        echo ""

        local cap link speed width tls training devctl lnkctl
        if cap=$(get_pcie_cap_offset "$gpu") &&
            link=$(read_pcie_link "$gpu" "$cap") &&
            devctl=$(read_pcie_word "$gpu" "$cap+0x08.w") &&
            lnkctl=$(read_pcie_word "$gpu" "$cap+0x10.w"); then
            read -r speed width tls training <<< "$link"
            local mrrs_bytes=$((128 << ((devctl >> 12) & 0x7)))
            local aspm_bits=$((lnkctl & 0x3))
            local aspm_str="Tat (Disabled)"
            if [[ ${aspm_bits} -eq 1 ]]; then aspm_str="L0s Bat"; elif [[ ${aspm_bits} -eq 2 ]]; then aspm_str="L1 Bat"; elif [[ ${aspm_bits} -eq 3 ]]; then aspm_str="L0s & L1 Bat"; fi

            echo "  - Link Speed hien tai    : Gen$speed ($(pcie_rate "$speed"))"
            echo "  - Link Width hien tai    : x$width"
            echo "  - Target Link Speed (TLS): Gen$tls"
            printf '  - Max Read Request Size  : %d Bytes (DEVCTL: 0x%04x)\n' "$mrrs_bytes" "$devctl"
            echo "  - PCIe ASPM trang thai   : $aspm_str"
        else
            echo "[!] Không đọc đủ thông tin PCIe cho $gpu." >&2
        fi

        echo ""
        echo "  Chi tiet thanh ghi PCIe (lspci):"
        lspci -s "${gpu}" -vv 2>/dev/null | grep -E "LnkCap:|LnkCtl:|LnkSta:|DevCtl:|LnkCtl2:" | while read -r line; do
            echo "    ${line}"
        done
    done
    echo ""
    exit 0
}

# ------------------------------------------------------------------------------
# HAM CHINH (MAIN)
# ------------------------------------------------------------------------------
main() {
    local original_args=("$@")
    local action=
    local is_daemon=0
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -u|--uninstall|-uninstall|-s|--status|-status|-d|--daemon|-daemon|--silent|-silent)
                if [[ -n "${action}" ]]; then
                    echo "[X] Chi chon mot che do: status, uninstall hoac daemon." >&2
                    return 2
                fi
                case "$1" in
                    -u|--uninstall|-uninstall) action=uninstall ;;
                    -s|--status|-status) action=status ;;
                    *) action=daemon; is_daemon=1 ;;
                esac
                ;;
            -test|--test) IS_MOCK=1 ;;
            -mock-fail|--mock-fail) IS_MOCK=1; MOCK_FAIL=1 ;;
            -mock-idle|--mock-idle) IS_MOCK=1; MOCK_IDLE=1 ;;
            -mock-nogpu|--mock-nogpu) IS_MOCK=1; MOCK_NOGPU=1 ;;
            --no-root) NO_ROOT=1 ;;
            --status-file)
                if [[ -z "${2:-}" || "$2" == -* ]]; then
                    echo "[X] --status-file can duong dan tep." >&2
                    return 2
                fi
                CUSTOM_STATUS_FILE="$2"
                shift
                ;;
            --status-file=)
                echo "[X] --status-file can duong dan tep." >&2
                return 2
                ;;
            --status-file=*) CUSTOM_STATUS_FILE="${1#*=}" ;;
            -h|--help|-help) usage ;;
            *)
                echo "Tham so khong hop le: $1" >&2
                usage 2 >&2
                ;;
        esac
        shift
    done

    if [[ "${IS_MOCK}" -eq 1 ]]; then
        NO_ROOT=1
    fi
    ensure_root "${original_args[@]}"
    case "${action}" in
        uninstall) uninstall_service; return $? ;;
        status) show_status; return $? ;;
    esac

    if [[ ${is_daemon} -eq 0 ]]; then
        echo -e "${C_BOLD}================================================================${C_RESET}"
        echo -e "${C_BOLD}  CONG CU CAI DAT TU DONG GEN2 X16 CHO NVIDIA CMP 30HX (LINUX)  ${C_RESET}"
        echo -e "${C_BOLD}  Phien ban: ${VERSION} | Nen tang: Linux x86_64                     ${C_RESET}"
        echo -e "${C_BOLD}================================================================${C_RESET}"
        echo ""
    fi

    check_dependencies

    # Chỉ cấu hình thanh ghi của CMP 30HX và cổng upstream tương ứng.
    echo -e "\n${C_BOLD}Đang tìm card NVIDIA CMP 30HX (10de:2189)...${C_RESET}"
    local gpus=()
    if [[ "${IS_MOCK}" -eq 1 ]]; then
        if [[ "${MOCK_NOGPU}" -eq 1 ]]; then
            gpus=()
        else
            gpus=("0000:01:00.0")
        fi
    else
        while IFS= read -r line; do
            if [[ -n "${line}" ]]; then
                gpus+=("${line}")
            fi
        done < <(lspci -d 10de:2189 2>/dev/null | awk '{print $1}')
    fi

    if [[ ${#gpus[@]} -eq 0 ]]; then
        echo -e "${C_RED}[X] LOI: Khong tim thay card NVIDIA CMP 30HX (ID: 10de:2189)!${C_RESET}"
        echo "    Vui long kiem tra khe cam PCIe, tiep xuc chan hoac nguon phu cua card."
        write_status_file "NO_GPU_FOUND" "0" "0" "0" "ERR_NO_GPU" "Khong tim thay card CMP 30HX tren he thong."
        exit 1
    fi

    echo -e "      ${C_GREEN}[OK]${C_RESET} Phát hiện ${#gpus[@]} card CMP 30HX: ${gpus[*]}"

    echo -e "\n${C_BOLD}Đang kiểm tra và cấu hình Gen2 x16, MRRS từ 512 byte.${C_RESET}"
    local count_fail=0 final_priority=-1 final_status_code final_speed final_width final_tls final_err final_detail
    local res speed width tls training cap link priority status_code err detail

    for gpu_bdf in "${gpus[@]}"; do
        res=0
        speed=0
        width=0
        tls=0
        unlock_gpu_pcie "$gpu_bdf" || res=$?

        if [[ "$IS_MOCK" -eq 1 ]]; then
            speed=1
            width=16
            tls=2
            if (( res == 0 )); then speed=2; fi
            if (( res == 1 )); then tls=1; fi
        elif cap=$(get_pcie_cap_offset "$gpu_bdf") && link=$(read_pcie_link "$gpu_bdf" "$cap"); then
            read -r speed width tls training <<< "$link"
            if (( res != 1 )); then
                res=0
                pcie_link_result "$speed" "$width" "$tls" "$training" || res=$?
            fi
        else
            res=1
        fi

        err="NONE"
        case "$res" in
            0)
                priority=0
                status_code="GEN2_SUCCESS"
                detail="Đã xác nhận PCIe Gen2 x16."
                ;;
            2)
                priority=1
                status_code="GEN2_IDLE"
                detail="TLS Gen2, liên kết Gen1 x16. Kiểm tra lại khi GPU có tải."
                ;;
            *)
                priority=2
                count_fail=$((count_fail + 1))
                status_code="GEN2_FAILED"
                err="ERR_RETRAIN_FAILED"
                if [[ "$IS_MOCK" -eq 1 ]]; then err="ERR_RETRAIN_TIMEOUT"; fi
                detail="Không xác minh được cấu hình và liên kết PCIe Gen2 x16."
                ;;
        esac
        echo "    [$gpu_bdf] $detail Gen$speed x$width (TLS=$tls)."

        # Tệp trạng thái đại diện cho kết quả xấu nhất, không phụ thuộc thứ tự card.
        if (( priority > final_priority )); then
            final_priority="$priority"
            final_status_code="$status_code"
            final_speed="$speed"
            final_width="$width"
            final_tls="$tls"
            final_err="$err"
            final_detail="$gpu_bdf: $detail"
        fi
    done

    if (( count_fail > 0 )); then
        echo "[!] Có $count_fail/${#gpus[@]} card chưa đạt yêu cầu. Bỏ qua đăng ký dịch vụ." >&2
    elif (( is_daemon == 0 )) && ! install_systemd_service; then
        write_status_file "INSTALL_FAILED" "0" "0" "0" "ERR_PERSISTENCE" "Không thể cài dịch vụ tự động khởi động."
        echo "[X] Không thể cài dịch vụ tự động khởi động." >&2
        return 1
    fi

    write_status_file "$final_status_code" "$final_speed" "$final_width" "$final_tls" "$final_err" "$final_detail"
    if (( count_fail > 0 )); then
        return 1
    fi
    if (( is_daemon == 0 )); then
        echo "[$final_status_code] Đã cài dịch vụ tự động khởi động."
        echo "    Kiểm tra trạng thái bằng: sudo $0 --status"
    fi
    return 0
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
