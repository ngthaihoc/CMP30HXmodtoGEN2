#!/bin/bash
# build_v70.sh — 40HX (TU106) v90 诊断构建
# =============================================================================
# v90 = v88（当前部署, SEC2 falcon_dma_transfer 装载, 已证明不黑屏）+ 只读前置诊断
#   - 不改装载引擎 / 不改 SEC2 状态操作 → 黑屏行为与 v88 完全一致
#   - 仅新增 SEC2 CPUCTL/DMATRFCMD/BCR/HWCFG2/RESET_PLM/WPR2 只读打印（v90 pre-flight）
# 工具链：MSYS2 便携版 D:\Code\40HX\.scratch\msys64（PATH=/mingw64/bin:/usr/bin）
# =============================================================================
set -e
cd "$(dirname "$0")"

export PATH=/mingw64/bin:/usr/bin:$PATH
# 定位便携 MSYS2 工具链：兼容“项目根 tools/unlock40x”与“发布目录
# 40HXUnlock_Source_vX.X/tools/unlock40x”两种相对深度。
# 注意 set -e 下不能用裸命令替换探测（cd 失败会让脚本直接退出）。
SRCDIR="$(cd "$(dirname "$0")" && pwd)"
MROOT=""
for cand in "$SRCDIR/../../.scratch/msys64" "$SRCDIR/../../../.scratch/msys64"; do
  if [ -d "$cand" ]; then
    MROOT="$(cd "$cand" && pwd)"
    break
  fi
done
if [ -n "$MROOT" ]; then
  export PATH="$MROOT/mingw64/bin:$MROOT/usr/bin:$PATH"
fi

EFI_INC="${EFI_INC:-/usr/include/efi}"
EFI_LIB="${EFI_LIB:-/usr/lib}"

SRC=unlock40x_v70.c
OBJ=unlock40x_v70.o
OUT=unlock40x_v70.so
EFIOUT=unlock40x_v70.efi

echo "=== 1. compile $SRC ==="
gcc -c -O2 -fno-stack-protector -fno-pie -ffreestanding \
    -fno-asynchronous-unwind-tables -fno-unwind-tables \
    -fshort-wchar -mno-red-zone -maccumulate-outgoing-args \
    -fno-builtin -fno-strict-aliasing -Wno-unused-function \
    -I "$EFI_INC" -I "$EFI_INC/x86_64" \
    -DDIRECT_SEC2 -DRELEASE_BUILD -DVBIOS_DUMP \
    -o "$OBJ" "$SRC"

echo "=== 2. embed blobs ==="
embed() {
  local f="$1" tgt="$2"
  local base="${f//./_}"
  objcopy --input-target binary --output-target pe-x86-64 \
    --binary-architecture i386:x86-64 \
    --redefine-sym "_binary_${base}_start=${tgt}" \
    --redefine-sym "_binary_${base}_end=${tgt}_end" \
    --redefine-sym "_binary_${base}_size=${tgt}_size" \
    "$f" "${tgt}.o"
  echo "embedded $f -> $tgt"
}
embed v67_payload.bin         v67_payload_bin
embed booter_ucode_dbg.bin    booter_ucode_dbg
embed booter_ucode_prod.bin   booter_ucode_prod
embed gsp_rm_boot_dbg.bin     gsp_rm_boot_dbg
embed fwsec_ga102.bin         fwsec_ga102_bin
embed fwsec_ga102_sig.bin     fwsec_ga102_sig
embed fwsec_40hx_prod.bin     fwsec_40hx_prod_bin
embed fwsec_40hx_dbg.bin      fwsec_40hx_dbg_bin
embed sec2_ucode_vbios_49.bin sec2_ucode_vbios_49
embed sec2_ucode_vbios_89.bin sec2_ucode_vbios_89
embed bl_gsp_tu102.bin        gsp_bl_tu102

OBJS="$OBJ v67_payload_bin.o booter_ucode_dbg.o booter_ucode_prod.o \
gsp_rm_boot_dbg.o fwsec_ga102_bin.o fwsec_ga102_sig.o \
fwsec_40hx_prod_bin.o fwsec_40hx_dbg_bin.o \
sec2_ucode_vbios_49.o sec2_ucode_vbios_89.o gsp_bl_tu102.o"

echo "=== 3. link ==="
ld -mi386pep --subsystem 10 -e u40x_entry -o "$OUT" $OBJS "$EFI_LIB/libefi.a" 2>&1 | tail -12 || true

echo "=== 4. convert ==="
if [ -f "$OUT" ]; then
  objcopy -j .text -j .data -j .rdata -j .reloc \
    --target=efi-app-x86_64 "$OUT" "$EFIOUT" 2>&1 | tail -4 || true
  ls -la "$EFIOUT" 2>/dev/null && echo v70_EFI_OK || echo CONVERT_FAIL
fi
echo DONE


