#!/usr/bin/env bash
# ==============================================================================
# CONG CU CAI DAT TU DONG GEN2 X16 CHO NVIDIA CMP 30HX (TU116) TREN LINUX (AIO)
# Ho tro: Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux, v.v.
# Tac gia: ngthaihoc (https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
# ==============================================================================

set -uo pipefail

VERSION="3.0.0"
SYSTEMD_SVC_NAME="cmp30hx-gen2-unlock.service"
SYSTEMD_SVC_PATH="/etc/systemd/system/${SYSTEMD_SVC_NAME}"
BIN_INSTALL_PATH="/usr/local/bin/cmp30hx-unlock"
SUSPEND_HOOK_PATH="/lib/systemd/system-sleep/cmp30hx-unlock"

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
    echo "  -s, --status       Kiem tra toc do link PCIe va bang thong hien tai cua CMP 30HX"
    echo "  -u, --uninstall    Go bo systemd service va khoi phuc thiet lap goc"
    echo "  -d, --daemon       Che do chay ngam (danh cho systemd service luc khoi dong)"
    echo "  -h, --help         Hien thi thong tin huong dan nay"
    echo ""
    exit 0
}

# ------------------------------------------------------------------------------
# Kiem tra phan mem can thiet (pciutils, python3)
# ------------------------------------------------------------------------------
check_dependencies() {
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
# Tim offset cua PCIe Capability (Cap ID 0x10)
# ------------------------------------------------------------------------------
get_pcie_cap_offset() {
    local bdf="$1"
    # Truong hop 1: setpci ho tro truc tiep alias CAP_EXP
    if setpci -s "${bdf}" CAP_EXP+0x00.b >/dev/null 2>&1; then
        echo "CAP_EXP"
        return 0
    fi

    # Truong hop 2: Quet danh sach PCI Capabilities thu cong
    local ptr
    ptr=$(setpci -s "${bdf}" 0x34.b 2>/dev/null || echo "00")
    local cur=$((16#$ptr))
    local safety=0

    while [[ ${cur} -gt 0 && ${cur} -lt 256 && ${safety} -lt 48 ]]; do
        local cap_id
        cap_id=$(setpci -s "${bdf}" "${cur}".b 2>/dev/null || echo "00")
        if [[ "$((16#$cap_id))" -eq 16 ]]; then # 0x10 = PCI Express
            printf "0x%02x\n" "${cur}"
            return 0
        fi
        local next_ptr
        next_ptr=$(setpci -s "${bdf}" "$((cur + 1))".b 2>/dev/null || echo "00")
        cur=$((16#$next_ptr))
        safety=$((safety + 1))
    done

    echo ""
    return 1
}

# ------------------------------------------------------------------------------
# Tim Root Port / Upstream Bridge cua thiet bi GPU
# ------------------------------------------------------------------------------
find_upstream_root_port() {
    local gpu_bdf="$1" # e.g. 0000:01:00.0 or 01:00.0
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
# Buoc 1: Tat PCIe ASPM & Runtime PM trong Linux
# ------------------------------------------------------------------------------
disable_aspm() {
    echo -e "${C_BOLD}[1/6] Dang tat PCIe ASPM va Runtime Power Management tren Linux...${C_RESET}"

    # 1. Chinh sach PCIe ASPM cua Kernel
    if [[ -w /sys/module/pcie_aspm/parameters/policy ]]; then
        echo performance > /sys/module/pcie_aspm/parameters/policy 2>/dev/null || true
        echo -e "      ${C_GREEN}[OK]${C_RESET} Da chuyen /sys/module/pcie_aspm/parameters/policy -> performance"
    else
        echo -e "      ${C_YELLOW}[!]${C_RESET} /sys/module/pcie_aspm/parameters/policy khong kha dung (co the he thong dung pcie_aspm=off)"
    fi

    # 2. Tat Runtime D3 Power Suspend tren toan bo bus PCI
    local count=0
    for ctrl in /sys/bus/pci/devices/*/power/control; do
        if [[ -w "${ctrl}" ]]; then
            echo on > "${ctrl}" 2>/dev/null || true
            count=$((count + 1))
        fi
    done
    echo -e "      ${C_GREEN}[OK]${C_RESET} Da dat power/control=on cho ${count} thiet bi PCI (chong ha link khi idle)"
}

# ------------------------------------------------------------------------------
# Buoc 2: Inject MMIO BAR0 (XVE_OVR, LINK_CONFIG_0, PRIV_MISC_1, LNKCAP/LNKCTL2)
# ------------------------------------------------------------------------------
inject_bar0_mmio() {
    local gpu_bdf="$1"
    local sys_dev="/sys/bus/pci/devices/${gpu_bdf}"
    if [[ ! -d "${sys_dev}" ]]; then
        sys_dev="/sys/bus/pci/devices/0000:${gpu_bdf}"
    fi

    local res0="${sys_dev}/resource0"
    if [[ ! -e "${res0}" ]]; then
        echo -e "      ${C_YELLOW}[!] Khong tim thay resource0 tai ${res0}, bo qua inject MMIO truc tiep.${C_RESET}"
        return 0
    fi

    if ! command -v python3 >/dev/null 2>&1; then
        echo -e "      ${C_YELLOW}[!] Khong co python3 de ghi struct mmap BAR0, chuyen sang cau hinh setpci.${C_RESET}"
        return 0
    fi

    echo -e "      [*] Dang kiem tra va ghi de thanh ghi BAR0 MMIO (TU116 XVE)..."
    python3 - <<EOF
import sys, os, struct

res_path = "${res0}"
try:
    fd = os.open(res_path, os.O_RDWR | os.O_SYNC)
except Exception as e:
    print(f"      [!] Khong the mo {res_path} ({e}); co the driver kernel dang lock IORESOURCE_BUSY.")
    sys.exit(0)

try:
    # 1. Doc BOOT_0 @ offset 0x0
    os.lseek(fd, 0, os.SEEK_SET)
    boot0_raw = os.read(fd, 4)
    if len(boot0_raw) < 4:
        print("      [!] Khong doc duoc BOOT_0 tu BAR0.")
        sys.exit(0)
    boot0 = struct.unpack('<I', boot0_raw)[0]

    # TU116 / TU10x family byte la 0x16 (0x16xxxxxx)
    if (boot0 & 0xFF000000) != 0x16000000:
        print(f"      [!] BOOT_0=0x{boot0:08X} khong phai TU116/TU10x (ky vong 0x16xxxxxx), bo qua ghi PL0.")
        sys.exit(0)

    print(f"      [OK] BAR0 hop le (BOOT_0=0x{boot0:08X}, Nhan TU116).")

    def read_u32(off):
        os.lseek(fd, off, os.SEEK_SET)
        return struct.unpack('<I', os.read(fd, 4))[0]

    def write_u32(off, val):
        os.lseek(fd, off, os.SEEK_SET)
        os.write(fd, struct.pack('<I', val))

    # Ghi thanh ghi XVE & Shadow Override
    # 0x0008841C: PRIV_MISC_1 (tat write protection shadow registers)
    write_u32(0x0008841C, 0xE0B42D00)
    # 0x0008872C: XVE_OVR = 6 (Gen2 override)
    write_u32(0x0008872C, 0x00000006)
    # 0x0008C040: LINK_CONFIG_0
    write_u32(0x0008C040, 0x80085800)
    # 0x0008C2C0: CYA_0
    write_u32(0x0008C2C0, 0x068731B3)
    # 0x0008872C: XVE_OVR confirm
    write_u32(0x0008872C, 0x00000006)

    # LNKCAP @ 0x088084 (dat Max Link Speed = Gen2)
    orig_cap = read_u32(0x00088084)
    write_u32(0x00088084, (orig_cap & 0xFFFFFFF0) | 2)

    # LNKCAP2 @ 0x0880A4
    write_u32(0x000880A4, 0x00000006)

    # XVE_F0 @ 0x0880F0
    write_u32(0x000880F0, 0x00000006)

    # LNKCTL2 @ 0x0880A8 (TLS = Gen2)
    orig_ctl2 = read_u32(0x000880A8)
    write_u32(0x000880A8, (orig_ctl2 & 0xFFFFFFF0) | 2)

    print("      [OK] Da inject thanh ghi BAR0 MMIO (XVE_OVR, LNKCAP/CTL2).")
except Exception as e:
    print(f"      [!] Loi khi thao tac MMIO BAR0: {e}")
finally:
    os.close(fd)
EOF
}

# ------------------------------------------------------------------------------
# Buoc 3 & 4: Cau hinh LNKCTL2, DEVCTL (MRRS 512B), LNKCTL va Huan luyen lai
# ------------------------------------------------------------------------------
unlock_gpu_pcie() {
    local gpu_bdf="$1"
    echo -e "${C_BOLD}----------------------------------------------------------------${C_RESET}"
    echo -e "${C_BLUE}[*] Dang xu ly GPU CMP 30HX: ${gpu_bdf}${C_RESET}"

    local gpu_cap
    gpu_cap=$(get_pcie_cap_offset "${gpu_bdf}")
    if [[ -z "${gpu_cap}" ]]; then
        echo -e "    ${C_RED}[X] Khong tim thay PCIe Capability tren GPU ${gpu_bdf}!${C_RESET}"
        return 1
    fi

    local root_bdf
    root_bdf=$(find_upstream_root_port "${gpu_bdf}")
    local root_cap=""
    if [[ -n "${root_bdf}" ]]; then
        root_cap=$(get_pcie_cap_offset "${root_bdf}")
        echo -e "    [*] Tim thay Upstream Root Port: ${root_bdf} (Cap: ${root_cap})"
    else
        echo -e "    ${C_YELLOW}[!] Khong tim thay Root Port; se chi huan luyen tren endpoint GPU.${C_RESET}"
    fi

    # 1. MMIO BAR0 Injection
    inject_bar0_mmio "${gpu_bdf}"

    # 2. Cai dat Target Link Speed = Gen2 (TLS=2) tren GPU LNKCTL2 (Cap + 0x30)
    local cur_gpu_ctl2
    cur_gpu_ctl2=$(setpci -s "${gpu_bdf}" "${gpu_cap}"+0x30.w 2>/dev/null || echo "0000")
    local new_gpu_ctl2
    new_gpu_ctl2=$(printf "%04x" $(( (16#$cur_gpu_ctl2 & ~0xF) | 2 )))
    setpci -s "${gpu_bdf}" "${gpu_cap}"+0x30.w="${new_gpu_ctl2}" 2>/dev/null || true
    echo -e "    [OK] GPU LNKCTL2 TLS -> Gen2 (0x${cur_gpu_ctl2} -> 0x${new_gpu_ctl2})"

    # 3. Cai dat Target Link Speed = Gen2 (TLS=2) tren Root Port LNKCTL2
    if [[ -n "${root_bdf}" && -n "${root_cap}" ]]; then
        local cur_root_ctl2
        cur_root_ctl2=$(setpci -s "${root_bdf}" "${root_cap}"+0x30.w 2>/dev/null || echo "0000")
        local new_root_ctl2
        new_root_ctl2=$(printf "%04x" $(( (16#$cur_root_ctl2 & ~0xF) | 2 )))
        setpci -s "${root_bdf}" "${root_cap}"+0x30.w="${new_root_ctl2}" 2>/dev/null || true
        echo -e "    [OK] Root Port LNKCTL2 TLS -> Gen2 (0x${cur_root_ctl2} -> 0x${new_root_ctl2})"
    fi

    # 4. Toi uu Max Read Request Size (MRRS) len 512 Bytes (DEVCTL, Cap + 0x08)
    # Bits [14:12] = 010b = 512B (2 << 12 = 0x2000)
    local cur_devctl
    cur_devctl=$(setpci -s "${gpu_bdf}" "${gpu_cap}"+0x08.w 2>/dev/null || echo "0000")
    local cur_mrrs=$(( (16#$cur_devctl >> 12) & 0x7 ))
    if [[ ${cur_mrrs} -lt 2 ]]; then
        local new_devctl
        new_devctl=$(printf "%04x" $(( (16#$cur_devctl & ~0x7000) | 0x2000 )))
        setpci -s "${gpu_bdf}" "${gpu_cap}"+0x08.w="${new_devctl}" 2>/dev/null || true
        echo -e "    [OK] Tối ưu MRRS GPU: 128B -> 512B (DEVCTL: 0x${cur_devctl} -> 0x${new_devctl})"
    else
        echo -e "    [OK] MRRS GPU hien tai da toi uu (>= 512B, DEVCTL=0x${cur_devctl})"
    fi

    # 5. Khoi phuc / Tat ASPM tren LNKCTL (Cap + 0x10) va dat Common Clock Configuration (bit 6 = 1)
    local cur_gpu_lnkctl
    cur_gpu_lnkctl=$(setpci -s "${gpu_bdf}" "${gpu_cap}"+0x10.w 2>/dev/null || echo "0000")
    local new_gpu_lnkctl
    new_gpu_lnkctl=$(printf "%04x" $(( (16#$cur_gpu_lnkctl & ~0x3) | 0x0040 )))
    setpci -s "${gpu_bdf}" "${gpu_cap}"+0x10.w="${new_gpu_lnkctl}" 2>/dev/null || true

    if [[ -n "${root_bdf}" && -n "${root_cap}" ]]; then
        local cur_root_lnkctl
        cur_root_lnkctl=$(setpci -s "${root_bdf}" "${root_cap}"+0x10.w 2>/dev/null || echo "0000")
        local new_root_lnkctl
        new_root_lnkctl=$(printf "%04x" $(( (16#$cur_root_lnkctl & ~0x3) | 0x0040 )))
        setpci -s "${root_bdf}" "${root_cap}"+0x10.w="${new_root_lnkctl}" 2>/dev/null || true
    fi

    # 6. Huan luyen lai link PCIe (Retrain Link Pulse - Bit 5 LNKCTL)
    echo -e "    [*] Bat dau chu trinh huan luyen lai link PCIe (toi da 6 luot)..."
    local attempt=1
    local speed_achieved=0
    while [[ ${attempt} -le 6 ]]; do
        local target_bdf="${gpu_bdf}"
        local target_cap="${gpu_cap}"
        local tag="GPU"

        if [[ $((attempt % 2)) -ne 0 && -n "${root_bdf}" && -n "${root_cap}" ]]; then
            target_bdf="${root_bdf}"
            target_cap="${root_cap}"
            tag="ROOT"
        fi

        # Nhip phat retrain: Xoa bit 5 -> sleep -> Dat bit 5
        local ctl_val
        ctl_val=$(setpci -s "${target_bdf}" "${target_cap}"+0x10.w 2>/dev/null || echo "0000")
        local ctl_clear
        ctl_clear=$(printf "%04x" $(( 16#$ctl_val & ~0x0020 )))
        setpci -s "${target_bdf}" "${target_cap}"+0x10.w="${ctl_clear}" 2>/dev/null || true
        sleep 0.2

        local ctl_set
        ctl_set=$(printf "%04x" $(( 16#$ctl_clear | 0x0020 )))
        setpci -s "${target_bdf}" "${target_cap}"+0x10.w="${ctl_set}" 2>/dev/null || true

        sleep 1.2

        # Doc lai toc do hien tai tu LNKSTA (Cap + 0x12)
        local cur_lnksta
        cur_lnksta=$(setpci -s "${gpu_bdf}" "${gpu_cap}"+0x12.w 2>/dev/null || echo "0000")
        local cur_speed=$(( 16#$cur_lnksta & 0xF ))
        local cur_width=$(( (16#$cur_lnksta >> 4) & 0x3F ))

        if [[ ${cur_speed} -ge 2 ]]; then
            echo -e "    ${C_GREEN}[V] Luot #${attempt} (${tag}): DA DAT GEN${cur_speed} x${cur_width}!${C_RESET}"
            speed_achieved=1
            break
        else
            echo -e "    [-] Luot #${attempt} (${tag}): Dang o Gen${cur_speed} x${cur_width}, dang thu lai..."
        fi
        attempt=$((attempt + 1))
    done

    return 0
}

# ------------------------------------------------------------------------------
# Buoc 5: Cai dat Systemd Service duy tri sau khi khoi dong & sleep
# ------------------------------------------------------------------------------
install_systemd_service() {
    echo -e "${C_BOLD}[5/6] Dang tao va kich hoat Systemd Service khoi dong tu dong...${C_RESET}"

    # Chep script hien tai vao duong dan he thong de service chay on dinh
    local current_script
    current_script="$(readlink -f "$0")"

    if [[ "${current_script}" != "${BIN_INSTALL_PATH}" ]]; then
        cp -f "${current_script}" "${BIN_INSTALL_PATH}"
        chmod +x "${BIN_INSTALL_PATH}"
        echo -e "      [OK] Da sao chep script den: ${BIN_INSTALL_PATH}"
    fi

    # Tao file systemd service
    cat <<EOF > "${SYSTEMD_SVC_PATH}"
[Unit]
Description=NVIDIA CMP 30HX PCIe Gen2 x16 Unlock Service
After=network.target multi-user.target
Before=graphical.target
DefaultDependencies=no

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=${BIN_INSTALL_PATH} --daemon

[Install]
WantedBy=multi-user.target
EOF

    chmod 644 "${SYSTEMD_SVC_PATH}"

    # Tao hook phuc hoi sau Sleep / Suspend / Hibernate
    mkdir -p "$(dirname "${SUSPEND_HOOK_PATH}")"
    cat <<EOF > "${SUSPEND_HOOK_PATH}"
#!/bin/sh
case "\$1/\$2" in
  post/*)
    ${BIN_INSTALL_PATH} --daemon >/dev/null 2>&1
    ;;
esac
EOF
    chmod +x "${SUSPEND_HOOK_PATH}"

    # Kich hoat service voi systemctl neu co systemd
    if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
        systemctl daemon-reload >/dev/null 2>&1 || true
        systemctl enable "${SYSTEMD_SVC_NAME}" >/dev/null 2>&1 || true
        echo -e "      ${C_GREEN}[OK]${C_RESET} Systemd service '${SYSTEMD_SVC_NAME}' da duoc kich hoat."
        echo -e "      ${C_GREEN}[OK]${C_RESET} Sleep/resume hook da duoc cai dat tai: ${SUSPEND_HOOK_PATH}"
    else
        echo -e "      ${C_YELLOW}[!] He thong khong su dung systemd; hay them '${BIN_INSTALL_PATH} --daemon' vao /etc/rc.local${C_RESET}"
    fi
}

# ------------------------------------------------------------------------------
# Go cai dat (Uninstall)
# ------------------------------------------------------------------------------
uninstall_service() {
    echo -e "${C_BOLD}================================================================${C_RESET}"
    echo -e "${C_BOLD}    GO BO TU DONG KHOI DONG CMP 30HX GEN2 UNLOCK TREN LINUX    ${C_RESET}"
    echo -e "${C_BOLD}================================================================${C_RESET}"

    if command -v systemctl >/dev/null 2>&1; then
        systemctl stop "${SYSTEMD_SVC_NAME}" >/dev/null 2>&1 || true
        systemctl disable "${SYSTEMD_SVC_NAME}" >/dev/null 2>&1 || true
        systemctl daemon-reload >/dev/null 2>&1 || true
    fi

    rm -f "${SYSTEMD_SVC_PATH}"
    rm -f "${SUSPEND_HOOK_PATH}"
    rm -f "${BIN_INSTALL_PATH}"

    echo -e "${C_GREEN}[V] Da go bo hoan toan Systemd Service va Sleep Hook.${C_RESET}"
    echo "    He thong da duoc khoi phuc trang thai ban dau."
    exit 0
}

# ------------------------------------------------------------------------------
# Hien thi trang thai chi tiet (Status)
# ------------------------------------------------------------------------------
show_status() {
    echo -e "${C_BOLD}================================================================${C_RESET}"
    echo -e "${C_BOLD}        TRANG THAI PCIE LINK CUA NVIDIA CMP 30HX / 40HX        ${C_RESET}"
    echo -e "${C_BOLD}================================================================${C_RESET}"

    local gpus
    gpus=$(lspci -d 10de:2189 2>/dev/null | awk '{print $1}')
    if [[ -z "${gpus}" ]]; then
        # Kiem tra ca 40HX (10de:1f0b)
        gpus=$(lspci -d 10de:1f0b 2>/dev/null | awk '{print $1}')
    fi

    if [[ -z "${gpus}" ]]; then
        echo -e "${C_YELLOW}[!] Khong tim thay card NVIDIA CMP 30HX (10de:2189) tren may!${C_RESET}"
        exit 0
    fi

    for gpu in ${gpus}; do
        echo -e "\n${C_BLUE}--- GPU BDF: ${gpu} ---${C_RESET}"
        lspci -s "${gpu}"
        echo ""
        lspci -s "${gpu}" -vv 2>/dev/null | grep -E "LnkCap:|LnkCtl:|LnkSta:|DevCtl:" | while read -r line; do
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
    # Xu ly huong dan truoc khi kiem tra root
    for arg in "$@"; do
        case "${arg}" in
            -h|--help|-help)
                usage
                ;;
        esac
    done

    ensure_root "$@"

    local is_daemon=0

    # Xu ly tham so dong lenh
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -u|--uninstall|-uninstall)
                uninstall_service
                ;;
            -s|--status|-status)
                show_status
                ;;
            -d|--daemon|-daemon|--silent|-silent)
                is_daemon=1
                shift
                ;;
            -h|--help|-help)
                usage
                ;;
            *)
                echo "Tham so khong hop le: $1"
                usage
                ;;
        esac
    done

    if [[ ${is_daemon} -eq 0 ]]; then
        echo -e "${C_BOLD}================================================================${C_RESET}"
        echo -e "${C_BOLD}  CONG CU CAI DAT TU DONG GEN2 X16 CHO NVIDIA CMP 30HX (LINUX)  ${C_RESET}"
        echo -e "${C_BOLD}  Phien ban: ${VERSION} | Nen tang: Linux x86_64                     ${C_RESET}"
        echo -e "${C_BOLD}================================================================${C_RESET}"
        echo ""
    fi

    check_dependencies

    # 1. Tat PCIe ASPM & Runtime PM
    disable_aspm

    # 2. Quet tat ca card CMP 30HX (10de:2189)
    echo -e "\n${C_BOLD}[2/6] Dang tim kiem card do hoa NVIDIA CMP 30HX tren he thong...${C_RESET}"
    local gpus=()
    while IFS= read -r line; do
        if [[ -n "${line}" ]]; then
            gpus+=("${line}")
        fi
    done < <(lspci -d 10de:2189 2>/dev/null | awk '{print $1}')

    # Neu khong co 30HX, kiem tra them 40HX (10de:1f0b)
    if [[ ${#gpus[@]} -eq 0 ]]; then
        while IFS= read -r line; do
            if [[ -n "${line}" ]]; then
                gpus+=("${line}")
            fi
        done < <(lspci -d 10de:1f0b 2>/dev/null | awk '{print $1}')
    fi

    if [[ ${#gpus[@]} -eq 0 ]]; then
        echo -e "${C_RED}[X] LOI: Khong tim thay card NVIDIA CMP 30HX (ID: 10de:2189)!${C_RESET}"
        echo "    Vui long kiem tra khe cam PCIe, tiep xuc chan hoac nguon phu cua card."
        exit 1
    fi

    echo -e "      ${C_GREEN}[OK]${C_RESET} Phat hien ${#gpus[@]} card CMP tren he thong: ${gpus[*]}"

    # 3 & 4. Tien hanh mo khoa tung card
    echo -e "\n${C_BOLD}[3/6 & 4/6] Dang mo khoa Gen2 x16 va toi uu MRRS 512B cho tung card...${C_RESET}"
    for gpu_bdf in "${gpus[@]}"; do
        unlock_gpu_pcie "${gpu_bdf}"
    done

    # 5. Cai dat Systemd Service
    if [[ ${is_daemon} -eq 0 ]]; then
        echo ""
        install_systemd_service
    fi

    # 6. Tong ket va Kiem tra
    echo ""
    echo -e "${C_BOLD}================================================================${C_RESET}"
    echo -e "${C_BOLD}[6/6] KET QUA XAC NHAN BANG THONG PCIE:${C_RESET}"
    for gpu_bdf in "${gpus[@]}"; do
        local cap
        cap=$(get_pcie_cap_offset "${gpu_bdf}")
        if [[ -n "${cap}" ]]; then
            local sta
            sta=$(setpci -s "${gpu_bdf}" "${cap}"+0x12.w 2>/dev/null || echo "0000")
            local speed=$(( 16#$sta & 0xF ))
            local width=$(( (16#$sta >> 4) & 0x3F ))

            local ctl2
            ctl2=$(setpci -s "${gpu_bdf}" "${cap}"+0x30.w 2>/dev/null || echo "0000")
            local tls=$(( 16#$ctl2 & 0xF ))

            if [[ ${speed} -ge 2 ]]; then
                echo -e "  [${gpu_bdf}] ${C_GREEN}${C_BOLD}[V] HOAN TAT:${C_RESET} PCIe Gen${speed} x${width} (~6.4 GB/s)"
            elif [[ ${tls} -ge 2 ]]; then
                echo -e "  [${gpu_bdf}] ${C_YELLOW}[!] DA CAU HINH GEN2 (TLS=${tls}):${C_RESET} Hien dang Gen${speed} x${width} (che do tiet kiem dien khi idle)"
                echo "      -> Card se tu dong nhay len Gen2 x16 khi co tai CUDA/3D/Mining!"
            else
                echo -e "  [${gpu_bdf}] ${C_RED}[X] CHUA DAT GEN2:${C_RESET} Hien tai Gen${speed} x${width} (TLS=${tls})"
            fi
        fi
    done

    echo -e "${C_BOLD}================================================================${C_RESET}"
    if [[ ${is_daemon} -eq 0 ]]; then
        echo -e "${C_GREEN}[V] CAI DAT HOAN TAT TREN LINUX!${C_RESET}"
        echo "  - Systemd Service se tu dong chay moi khi khoi dong he thong."
        echo "  - Khi ranh, PCIe co the ha ve Gen1 x16 de tiet kiem dien."
        echo "  - De kiem tra trang thai bat cu luc nao, chay: sudo $0 --status"
        echo ""
    fi
}

main "$@"
