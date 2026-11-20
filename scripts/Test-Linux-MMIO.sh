#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "$0")/.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -f -- "$scratch"/*; rmdir -- "$scratch"' EXIT
source_file="$project_root/linux-mmio/mmio_injector.c"
compiler="${CC:-cc}"
flags=(-std=c11 -O2 -Wall -Wextra -Werror)

"$compiler" "${flags[@]}" "$source_file" -o "$scratch/injector"
cat >"$scratch/io-fixture.c" <<'C'
#include <errno.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <unistd.h>

void *__wrap_mmap(void *address, size_t length, int prot, int flags, int fd, off_t offset)
{
    (void)address;
    (void)length;
    (void)prot;
    (void)flags;
    (void)fd;
    (void)offset;
    errno = ENODEV;
    return MAP_FAILED;
}

ssize_t __real_read(int fd, void *buffer, size_t count);
ssize_t __real_write(int fd, const void *buffer, size_t count);

ssize_t __wrap_read(int fd, void *buffer, size_t count)
{
    const char *mode = getenv("FIXTURE_IO");
    if (mode && strcmp(mode, "short-read") == 0 && lseek(fd, 0, SEEK_CUR) > 0)
        return __real_read(fd, buffer, count - 1);
    return __real_read(fd, buffer, count);
}

ssize_t __wrap_write(int fd, const void *buffer, size_t count)
{
    const char *mode = getenv("FIXTURE_IO");
    if (mode && strcmp(mode, "short-write") == 0)
        return __real_write(fd, buffer, count - 1);
    if (mode && strcmp(mode, "stale-tls") == 0 && lseek(fd, 0, SEEK_CUR) == 0x880a8)
        return (ssize_t)count;
    return __real_write(fd, buffer, count);
}
C
"$compiler" "${flags[@]}" "$source_file" "$scratch/io-fixture.c" \
    -Wl,--wrap=mmap,--wrap=read,--wrap=write -o "$scratch/fallback"

make_resource() {
    truncate -s 0 "$scratch/resource"
    truncate -s 1048576 "$scratch/resource"
    printf '\001\000\000\026' | dd of="$scratch/resource" conv=notrunc status=none
}

assert_rejected() {
    local result=0
    "$@" >"$scratch/output" 2>&1 || result=$?
    if [[ "$result" != 1 ]] || grep -Fq '[OK] Da inject MMIO' "$scratch/output"; then
        cat "$scratch/output" >&2
        echo "FAIL MMIO fixture exited $result instead of reporting an I/O error: $1" >&2
        exit 1
    fi
}

make_resource
"$scratch/injector" "$scratch/resource" >"$scratch/output"
grep -Fq '[OK] Da inject MMIO' "$scratch/output"
make_resource
"$scratch/fallback" "$scratch/resource" >"$scratch/output" 2>&1
grep -Fq '[OK] Da inject MMIO' "$scratch/output"
echo 'PASS MMIO performs and verifies writes via mmap and fd fallback'

for fixture in short-read short-write stale-tls; do
    make_resource
    assert_rejected env FIXTURE_IO="$fixture" "$scratch/fallback" "$scratch/resource"
    echo "PASS MMIO rejects $fixture"
done

make_resource
printf '\000\000\000\000' | dd of="$scratch/resource" conv=notrunc status=none
assert_rejected "$scratch/injector" "$scratch/resource"
make_resource
truncate -s 4096 "$scratch/resource"
assert_rejected "$scratch/injector" "$scratch/resource"
echo 'PASS MMIO rejects an unrelated or undersized BAR before writing'
