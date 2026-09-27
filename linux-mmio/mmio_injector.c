#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <unistd.h>
#include <string.h>
#include <errno.h>

uint32_t read_u32(void *map_base, off_t offset) {
    return *((volatile uint32_t *)((char *)map_base + offset));
}

void write_u32(void *map_base, off_t offset, uint32_t value) {
    *((volatile uint32_t *)((char *)map_base + offset)) = value;
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

    size_t map_size = 0x100000;
    void *map_base = mmap(0, map_size, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
    if (map_base == MAP_FAILED) {
        fprintf(stderr, "      [!] Failed to mmap %s: %s\n", res_path, strerror(errno));
        close(fd);
        return 1;
    }

    uint32_t boot0 = read_u32(map_base, 0x0);
    if ((boot0 & 0xFF000000) != 0x16000000) {
        printf("      [!] BOOT_0=0x%08X khong phai TU116/TU10x (ky vong 0x16xxxxxx), bo qua inject MMIO.\n", boot0);
        munmap(map_base, map_size);
        close(fd);
        return 0; // Success, just skipped
    }

    printf("      [OK] BAR0 hop le (BOOT_0=0x%08X, Nhan TU116, che do mmap C binary).\n", boot0);

    // 0x0008841C: PRIV_MISC_1
    write_u32(map_base, 0x0008841C, 0xE0B42D00);
    // 0x0008872C: XVE_OVR = 6
    write_u32(map_base, 0x0008872C, 0x00000006);
    // 0x0008C040: LINK_CONFIG_0
    write_u32(map_base, 0x0008C040, 0x80085800);
    // 0x0008C1C0: PL_LINK_RATE
    write_u32(map_base, 0x0008C1C0, 0x00240036);
    // 0x0008C2C0: CYA_0
    write_u32(map_base, 0x0008C2C0, 0x068731B3);
    // 0x0008872C: XVE_OVR confirm
    write_u32(map_base, 0x0008872C, 0x00000006);

    uint32_t orig_cap = read_u32(map_base, 0x00088084);
    write_u32(map_base, 0x00088084, (orig_cap & 0xFFFFFFF0) | 2);

    write_u32(map_base, 0x000880A4, 0x00000006);
    write_u32(map_base, 0x000880F0, 0x00000006);

    uint32_t orig_ctl2 = read_u32(map_base, 0x000880A8);
    write_u32(map_base, 0x000880A8, (orig_ctl2 & 0xFFFFFFF0) | 2);

    uint32_t phy_l0 = read_u32(map_base, 0x0008C4B0);
    const char *phy_desc = "Khac";
    if ((phy_l0 & 0xFF000000) == 0x50000000) {
        phy_desc = "Gen2 (5.0 GT/s)";
    } else if ((phy_l0 & 0xFF000000) == 0x25000000) {
        phy_desc = "Gen1 (2.5 GT/s)";
    }

    printf("      [OK] Da inject MMIO BAR0. PHY Lane 0: 0x%08X (%s).\n", phy_l0, phy_desc);

    munmap(map_base, map_size);
    close(fd);
    return 0;
}
