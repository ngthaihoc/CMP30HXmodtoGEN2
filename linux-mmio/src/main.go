package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <sysfs_resource_path>\n", os.Args[0])
		os.Exit(1)
	}

	resPath := os.Args[1]
	fd, err := os.OpenFile(resPath, os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "      [!] Failed to open %s: %v\n", resPath, err)
		os.Exit(1)
	}
	defer fd.Close()

	mapSize := 0x100000
	mapBase, mmapErr := syscall.Mmap(int(fd.Fd()), 0, mapSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	useMmap := mmapErr == nil

	if !useMmap {
		fmt.Fprintf(os.Stderr, "      [!] mmap failed (%v), falling back to lseek+write\n", mmapErr)
	} else {
		defer syscall.Munmap(mapBase)
	}

	// readU32 / writeU32 dispatch to mmap or file I/O
	readU32 := func(offset int) uint32 {
		if useMmap {
			return *(*uint32)(unsafe.Pointer(&mapBase[offset]))
		}
		var buf [4]byte
		if _, err := fd.ReadAt(buf[:], int64(offset)); err != nil && err != io.EOF {
			return 0
		}
		return binary.LittleEndian.Uint32(buf[:])
	}
	writeU32 := func(offset int, value uint32) {
		if useMmap {
			*(*uint32)(unsafe.Pointer(&mapBase[offset])) = value
			return
		}
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], value)
		fd.WriteAt(buf[:], int64(offset)) //nolint:errcheck — best-effort MMIO write
	}

	boot0 := readU32(0x0)
	if (boot0 & 0xFF000000) != 0x16000000 {
		fmt.Printf("      [!] BOOT_0=0x%08X khong phai TU116/TU10x (ky vong 0x16xxxxxx), bo qua inject MMIO.\n", boot0)
		return
	}

	mode := "mmap Go binary"
	if !useMmap {
		mode = "lseek fallback"
	}
	fmt.Printf("      [OK] BAR0 hop le (BOOT_0=0x%08X, Nhan TU116, che do %s).\n", boot0, mode)

	writeU32(0x0008841C, 0xE0B42D00)
	writeU32(0x0008872C, 0x00000006)
	writeU32(0x0008C040, 0x80085800)
	writeU32(0x0008C1C0, 0x00240036)
	writeU32(0x0008C2C0, 0x068731B3)
	writeU32(0x0008872C, 0x00000006)

	origCap := readU32(0x00088084)
	writeU32(0x00088084, (origCap&0xFFFFFFF0)|2)
	writeU32(0x000880A4, 0x00000006)
	writeU32(0x000880F0, 0x00000006)

	origCtl2 := readU32(0x000880A8)
	writeU32(0x000880A8, (origCtl2&0xFFFFFFF0)|2)

	phyL0 := readU32(0x0008C4B0)
	phyDesc := "Khac"
	if (phyL0 & 0xFF000000) == 0x50000000 {
		phyDesc = "Gen2 (5.0 GT/s)"
	} else if (phyL0 & 0xFF000000) == 0x25000000 {
		phyDesc = "Gen1 (2.5 GT/s)"
	}

	fmt.Printf("      [OK] Da inject MMIO BAR0. PHY Lane 0: 0x%08X (%s).\n", phyL0, phyDesc)
}
