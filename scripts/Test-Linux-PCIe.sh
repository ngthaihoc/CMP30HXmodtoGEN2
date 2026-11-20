#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "$0")/.." && pwd)"
script="$project_root/Setup_CMP30HX_LinuxAIO.sh"
scratch="$(mktemp -d)"
trap 'rm -f -- "$scratch"/*; rmdir -- "$scratch"' EXIT

run_fixture() {
    local scenario="$1"
    bash -c '
        source "$1"
        work="$2"
        scenario="$3"
        gpu=fixture-gpu
        port=fixture-port
        get_pcie_cap_offset() { echo CAP_EXP; }
        find_upstream_root_port() { echo "$port"; }
        inject_bar0_mmio() {
            [[ "$scenario" != mmio_error ]]
        }
        sleep() { :; }
        fuser() { touch "$work/hard-fallback"; }
        check_dependencies() { :; }
        disable_aspm() { touch "$work/aspm"; }
        install_systemd_service() { touch "$work/persistence"; }
        lspci() {
            if [[ "$scenario" == only_40hx ]]; then
                if [[ "$*" == "-d 10de:1f0b" ]]; then printf "%s CMP 40HX\n" "$gpu"; fi
                return 0
            fi
            [[ "$*" == "-d 10de:2189" ]] || return 1
            printf "%s Mock GPU\n" "$gpu"
            if [[ "$scenario" == mixed ]]; then
                printf "fixture-valid Mock GPU\n"
            fi
        }
        for device in "$gpu" "$port" fixture-valid; do
            printf "00a1\n" > "$work/$device-0x30"
            printf "0102\n" > "$work/$device-0x12"
            printf "0080\n" > "$work/$device-0x08"
            printf "0003\n" > "$work/$device-0x10"
        done
        case "$scenario" in
            gen2_x8|mixed) printf "0082\n" > "$work/$gpu-0x12" ;;
            gen1_idle) printf "0101\n" > "$work/$gpu-0x12" ;;
        esac
        setpci() {
            [[ "$#" == 3 && "$1" == -s && "$3" == CAP_EXP+*.w* ]] || return 1
            local register=${3#CAP_EXP+}
            local path="$work/$2-${register%%.w*}"
            local value
            case "$register" in
                *=*)
                    if [[ "$scenario" == write_error ]]; then return 1; fi
                    if [[ "$scenario" == write_warning ]]; then echo "PCI write failed" >&2; return 0; fi
                    if [[ "$scenario" == stale_readback && "$register" == 0x30.w=* ]]; then return 0; fi
                    value=${register#*=}
                    if [[ "$value" == *:* ]]; then
                        local previous
                        previous=$(cat "$path")
                        printf -v value "%04x" "$(( (16#$previous & ~16#${value#*:}) | 16#${value%:*} ))"
                    fi
                    printf "%s\n" "$value" > "$path"
                    ;;
                0x30.w)
                    if [[ "$scenario" == malformed_read ]]; then echo "no device"; return 0; fi
                    if [[ "$scenario" == missing_device ]]; then echo ffff; return 0; fi
                    cat "$path"
                    ;;
                0x12.w)
                    if [[ "$scenario" == status_read_error ]]; then return 1; fi
                    cat "$path"
                    ;;
                *) cat "$path" ;;
            esac
        }
        main --no-root --status-file="$work/status"
    ' _ "$script" "$scratch" "$scenario"
}

for scenario in write_error write_warning stale_readback malformed_read missing_device status_read_error mmio_error gen2_x8 mixed; do
    rm -f -- "$scratch/persistence" "$scratch/status" "$scratch/hard-fallback"
    if run_fixture "$scenario" >"$scratch/output" 2>&1; then
        cat "$scratch/output" >&2
        echo "FAIL Linux PCIe setup accepted $scenario" >&2
        exit 1
    fi
    grep -Fxq 'STATUS_CODE=GEN2_FAILED' "$scratch/status"
    [[ ! -e "$scratch/persistence" && ! -e "$scratch/hard-fallback" && ! -e "$scratch/aspm" ]]
    echo "PASS Linux PCIe setup rejects $scenario without enabling persistence"
done

rm -f -- "$scratch/aspm" "$scratch/status"
if run_fixture only_40hx >"$scratch/output" 2>&1; then
    echo 'FAIL Linux setup accepted CMP 40HX' >&2
    exit 1
fi
grep -Fxq 'STATUS_CODE=NO_GPU_FOUND' "$scratch/status"
[[ ! -e "$scratch/aspm" && ! -e "$scratch/persistence" ]]
echo 'PASS Linux setup rejects CMP 40HX before touching power or persistence'
run_fixture gen1_idle >"$scratch/output" 2>&1
grep -Fxq 'STATUS_CODE=GEN2_IDLE' "$scratch/status"
grep -Fxq 'SPEED_CURRENT=1' "$scratch/status"
grep -Fxq 'WIDTH_CURRENT=16' "$scratch/status"
grep -Fxq 'TLS_TARGET=2' "$scratch/status"
[[ -e "$scratch/persistence" && ! -e "$scratch/hard-fallback" && ! -e "$scratch/aspm" ]]
echo 'PASS Linux PCIe setup accepts verified idle without changing global power policy'

run_fixture gen2_x16 >"$scratch/output" 2>&1
grep -Fxq 'STATUS_CODE=GEN2_SUCCESS' "$scratch/status"
grep -Fxq 'SPEED_CURRENT=2' "$scratch/status"
grep -Fxq 'WIDTH_CURRENT=16' "$scratch/status"
grep -Fxq 'TLS_TARGET=2' "$scratch/status"
[[ $(cat "$scratch/fixture-gpu-0x30") == 00a2 ]]
[[ $(cat "$scratch/fixture-port-0x30") == 00a2 ]]
[[ $(cat "$scratch/fixture-gpu-0x08") == 2080 ]]
echo 'PASS Linux PCIe setup verifies Gen2 x16 and preserves unrelated register bits'
