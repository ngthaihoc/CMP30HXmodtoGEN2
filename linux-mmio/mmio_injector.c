#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <unistd.h>
#include <string.h>
#include <errno.h>
#include <sys/stat.h>

/* Write a 32-bit value via mmap pointer */
static void mmap_write_u32(void *map_base, off_t offset, uint32_t value) {
    *((volatile uint32_t *)((char *)map_base + offset)) = value;
}

/* Read a 32-bit value via mmap pointer */
static uint32_t mmap_read_u32(void *map_base, off_t offset) {
    return *((volatile uint32_t *)((char *)map_base + offset));
}

/* Write a 32-bit value via lseek+write fallback (no mmap) */
static int fd_write_u32(int fd, off_t offset, uint32_t value) {
    if (lseek(fd, offset, SEEK_SET) == (off_t)-1) {
        return -1;
    }
    ssize_t written = write(fd, &value, sizeof(value));
    if (written == (ssize_t)sizeof(value)) {
        return 0;
    }
    if (written >= 0) {
        errno = EIO;
    }
    return -1;
}

/* Read a 32-bit value via lseek+read fallback (no mmap) */
static int fd_read_u32(int fd, off_t offset, uint32_t *out) {
    if (lseek(fd, offset, SEEK_SET) == (off_t)-1) {
        return -1;
    }
    ssize_t n = read(fd, out, sizeof(*out));
    if (n == (ssize_t)sizeof(*out)) {
        return 0;
    }
    if (n >= 0) {
        errno = EIO;
    }
    return -1;
}

int main(int argc, char *argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <sysfs_resource_path>\n", argv[0]);
        return 1;
    }

    const char *res_path = argv[1];
    int fd = open(res_path, O_RDWR | O_SYNC);
    if (fd < 0) {
        fprintf(stderr, "      [!] Failed to open %s: %s\n", res_path, strerror(errno));
        return 1;
    }

    enum { phy_lane0 = 0x0008C4B0 };
    size_t map_size = phy_lane0 + sizeof(uint32_t);
    struct stat resource;
    if (fstat(fd, &resource) < 0 || resource.st_size < (off_t)map_size) {
        fprintf(stderr, "      [X] BAR0 không đủ kích thước để truy cập thanh ghi.\n");
        close(fd);
        return 1;
    }
    int result = 1;
    void *map_base = mmap(0, map_size, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
    int use_mmap = (map_base != MAP_FAILED);

    if (!use_mmap) {
        fprintf(stderr, "      [!] mmap failed (%s), falling back to lseek+write\n", strerror(errno));
    }

    /* Read BOOT_0 to validate GPU family */
    uint32_t boot0 = 0;
    if (use_mmap) {
        boot0 = mmap_read_u32(map_base, 0x0);
    } else {
        if (fd_read_u32(fd, 0x0, &boot0) < 0) {
            fprintf(stderr, "      [!] Failed to read BOOT_0 via lseek: %s\n", strerror(errno));
            goto cleanup;
        }
    }

    if ((boot0 & 0xFF000000) != 0x16000000) {
        fprintf(stderr, "      [X] BOOT_0=0x%08X không khớp với BAR0 Turing.\n", boot0);
        goto cleanup;
    }

    printf("      [OK] BAR0 hop le (BOOT_0=0x%08X, Nhan TU116, che do %s).\n",
           boot0, use_mmap ? "mmap C binary" : "lseek fallback");

#define WR32(offset, value) \
    do { \
        if (use_mmap) { \
            mmap_write_u32(map_base, (offset), (value)); \
        } else if (fd_write_u32(fd, (offset), (value)) < 0) { \
            fprintf(stderr, "      [X] Không ghi được MMIO tại 0x%08X: %s\n", (unsigned)(offset), strerror(errno)); \
            goto cleanup; \
        } \
    } while (0)

#define RD32(offset, out) \
    do { \
        if (use_mmap) { \
            *(out) = mmap_read_u32(map_base, (offset)); \
        } else if (fd_read_u32(fd, (offset), (out)) < 0) { \
            fprintf(stderr, "      [X] Không đọc được MMIO tại 0x%08X: %s\n", (unsigned)(offset), strerror(errno)); \
            goto cleanup; \
        } \
        if (*(out) == UINT32_MAX) { \
            fprintf(stderr, "      [X] Giá trị MMIO không hợp lệ tại 0x%08X.\n", (unsigned)(offset)); \
            goto cleanup; \
        } \
    } while (0)

    /* Shadow register sequence (TU116) */
    WR32(0x0008841C, 0xE0B42D00); /* PRIV_MISC_1 */
    WR32(0x0008872C, 0x00000006); /* XVE_OVR = 6 */
    WR32(0x0008C040, 0x80085800); /* LINK_CONFIG_0 */
    WR32(0x0008C1C0, 0x00240036); /* PL_LINK_RATE */
    WR32(0x0008C2C0, 0x068731B3); /* CYA_0 */
    WR32(0x0008872C, 0x00000006); /* XVE_OVR confirm */

    uint32_t orig_cap = 0;
    RD32(0x00088084, &orig_cap);
    WR32(0x00088084, (orig_cap & 0xFFFFFFF0) | 2);

    WR32(0x000880A4, 0x00000006);
    WR32(0x000880F0, 0x00000006);

    uint32_t orig_ctl2 = 0;
    RD32(0x000880A8, &orig_ctl2);
    WR32(0x000880A8, (orig_ctl2 & 0xFFFFFFF0) | 2);

    uint32_t cap_readback;
    uint32_t ctl2_readback;
    RD32(0x00088084, &cap_readback);
    RD32(0x000880A8, &ctl2_readback);
    if ((cap_readback & 0xF) != 2 || (ctl2_readback & 0xF) != 2) {
        fprintf(stderr, "      [X] Giá trị MMIO đọc lại không xác nhận được Gen2 (CAP=0x%08X, CTL2=0x%08X).\n",
                cap_readback, ctl2_readback);
        goto cleanup;
    }

    uint32_t phy_l0 = 0;
    RD32(phy_lane0, &phy_l0);
    const char *phy_desc = "Khac";
    if ((phy_l0 & 0xFF000000) == 0x50000000) {
        phy_desc = "Gen2 (5.0 GT/s)";
    } else if ((phy_l0 & 0xFF000000) == 0x25000000) {
        phy_desc = "Gen1 (2.5 GT/s)";
    }

    printf("      [OK] Da inject MMIO BAR0. PHY Lane 0: 0x%08X (%s).\n", phy_l0, phy_desc);
    result = 0;

cleanup:
    if (use_mmap) {
        munmap(map_base, map_size);
    }
    close(fd);
    return result;
}
