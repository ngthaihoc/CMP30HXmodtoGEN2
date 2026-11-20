#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "$0")/.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -f -- "$scratch/build_v70.sh" "$scratch/build.log" "$scratch"/unlock40x_v70.*; rmdir -- "$scratch"' EXIT
cp "$project_root/windows-v3.0/tools/unlock40x/build_v70.sh" "$scratch/"

gcc() { return 0; }
ld() {
    if [[ "$EFI_TEST_CASE" == link ]]; then
        return 22
    fi
    while [[ "$1" != -o ]]; do
        shift
    done
    printf 'linked fixture\n' > "$2"
}
objcopy() {
    if [[ "$1" == --input-target ]]; then
        return 0
    fi
    case "$EFI_TEST_CASE" in
        convert) printf 'partial\n' > "${!#}"; return 23 ;;
        empty) : > "${!#}" ;;
        *) printf 'new EFI fixture\n' > "${!#}" ;;
    esac
}
export -f gcc ld objcopy

check_build() {
    local scenario="$1" expected="$2" result=0
    EFI_TEST_CASE="$scenario" bash "$scratch/build_v70.sh" > "$scratch/build.log" 2>&1 || result="$?"
    if [[ "$result" -ne "$expected" ]]; then
        echo "FAIL EFI $scenario returned $result, expected $expected" >&2
        cat "$scratch/build.log" >&2
        exit 1
    fi
}

efi="$scratch/unlock40x_v70.efi"
check_build link 22
[[ ! -e "$efi" ]]
! grep -Fxq v70_EFI_OK "$scratch/build.log"

printf 'previous EFI fixture\n' > "$efi"
check_build convert 23
[[ "$(cat "$efi")" == 'previous EFI fixture' ]]
! grep -Fxq v70_EFI_OK "$scratch/build.log"

check_build empty 1
[[ "$(cat "$efi")" == 'previous EFI fixture' ]]
! grep -Fxq v70_EFI_OK "$scratch/build.log"

check_build success 0
[[ "$(cat "$efi")" == 'new EFI fixture' ]]
grep -Fxq v70_EFI_OK "$scratch/build.log"
echo 'PASS EFI build propagates tool failures and publishes only a complete new output'
