#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "$0")/.." && pwd)"
script="$project_root/Setup_CMP30HX_LinuxAIO.sh"
scratch="$(mktemp -d)"
trap 'rm -rf -- "$scratch"' EXIT

make_fixture() {
    mkdir -p "$scratch/linux-mmio"
    touch "$scratch/service" "$scratch/sleep" "$scratch/bin"
    touch "$scratch/linux-mmio/mmio_injector" "$scratch/linux-mmio/unrelated"
}

run_uninstall() {
    local fail_injector="$1"
    bash -c '
        source "$1"
        SYSTEMD_SVC_NAME="cmp30hx-test.service"
        SYSTEMD_SVC_PATH="$2/service"
        SUSPEND_HOOK_PATH="$2/sleep"
        BIN_INSTALL_PATH="$2/bin"
        MMIO_INSTALL_PATH="$2/linux-mmio/mmio_injector"
        CUSTOM_STATUS_FILE="$2/status"
        IS_MOCK=0
        systemctl() {
            case "$1" in
                is-active|is-enabled) return 1 ;;
                *) return 0 ;;
            esac
        }
        if [[ "$3" == "1" ]]; then
            rm() {
                case "$*" in
                    *mmio_injector*) return 1 ;;
                esac
                command rm "$@"
            }
        fi
        uninstall_service
    ' _ "$script" "$scratch" "$fail_injector"
}

make_fixture
run_uninstall 0 >/dev/null
grep -Fxq 'STATUS_CODE=UNINSTALLED' "$scratch/status"
[[ ! -e "$scratch/service" && ! -e "$scratch/sleep" && ! -e "$scratch/bin" ]]
[[ ! -e "$scratch/linux-mmio/mmio_injector" && -e "$scratch/linux-mmio/unrelated" ]]
run_uninstall 0 >/dev/null
echo 'PASS Linux uninstall preserves unrelated files and is repeatable'

make_fixture
if run_uninstall 1 >/dev/null 2>&1; then
    echo 'FAIL Linux uninstall reported success after a delete error' >&2
    exit 1
fi
grep -Fxq 'STATUS_CODE=UNINSTALL_FAILED' "$scratch/status"
[[ -e "$scratch/linux-mmio/mmio_injector" ]]
run_uninstall 0 >/dev/null
echo 'PASS Linux uninstall reports deletion failure and recovers on retry'

if bash -c '
    source "$1"
    install_systemd_service() { return 1; }
    main -test --no-root --status-file="$2"
' _ "$script" "$scratch/status" >/dev/null 2>&1; then
    echo 'FAIL Linux setup reported success after persistence failed' >&2
    exit 1
fi
grep -Fxq 'STATUS_CODE=INSTALL_FAILED' "$scratch/status"
echo 'PASS Linux setup reports persistence failure'

if bash -c '
    source "$1"
    install_systemd_service() { exit 99; }
    main -mock-fail --no-root --status-file="$2"
' _ "$script" "$scratch/status" >/dev/null 2>&1; then
    echo 'FAIL Linux setup reported success after PCIe failure' >&2
    exit 1
fi
grep -Fxq 'STATUS_CODE=GEN2_FAILED' "$scratch/status"
echo 'PASS Linux setup skips persistence after PCIe failure'

# Status must report the PCIe signaling rate for the negotiated generation.
status_output=$(bash "$script" --status -test)
grep -Fq 'Gen2 (5.0 GT/s)' <<< "$status_output"
status_output=$(bash "$script" --status -mock-idle)
grep -Fq 'Gen1 (2.5 GT/s)' <<< "$status_output"
echo 'PASS Linux status reports the correct PCIe signaling rates'

# Parse and validate the entire command line before executing any action.
for args in '--unknown' '--status --unknown' '--uninstall --status' '--status-file' '--status-file='; do
    read -ra flags <<< "$args"
    if bash "$script" -test "${flags[@]}" >/dev/null 2>&1; then
        echo "FAIL Linux accepted invalid arguments: $args" >&2
        exit 1
    fi
done
bash "$script" --uninstall -test --status-file "$scratch/ordered-status" >/dev/null
grep -Fxq 'STATUS_CODE=UNINSTALLED' "$scratch/ordered-status"
echo 'PASS Linux validates options before running the selected action'
