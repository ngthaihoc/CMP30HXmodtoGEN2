# 40HX Unlock 应急修复指南（引导损坏恢复）

> **本文件仅在"装了解锁后开机引导出错"时使用**——正常使用请不要碰。
> 适用范围：开机蓝屏报错 `0xc000000f` / `0xc000007b` / `0xc0000098`，
> 提示找不到 `\EFI\40HX\40HXUNLK.EFI`，或卡在 40HX 解锁画面无法进系统。

---

## 0. 为什么会这样 & 修复思路

解锁工具会往 **EFI 系统分区（ESP）** 写入解锁固件并注册一个固件启动项。
极少数情况下（固件文件不完整、主板时序、人为中断），会导致引导链断裂：

```
开机 → 主板按 NVRAM 启动项去找 \EFI\40HX\40HXUNLK.EFI
     → 文件缺失/损坏 → Windows 引导管理器报 0xc000000f
```

**核心原则：这不是系统坏了，是"引导入口"被卡住了。**
修复 = ① 删掉卡住的 40HX 启动项与残留文件 → ② 恢复/重建 Windows 自己的 BCD 引导数据库。

**你需要准备一个 Windows 安装 U 盘**（官方媒体创建工具做的即可），
或一个 Ubuntu 等 Linux 启动 U 盘（仅用于删文件，修不了 BCD）。

---

## 1. 首选方案：Windows 安装 U 盘（一条路走完，推荐）

### 1.1 启动到 Windows 恢复环境

1. 插入 **Windows 安装 U 盘**，开机从它启动（UEFI 开头的那一项）。
2. 看到安装界面后，点左下角 **修复计算机** → **疑难解答** → **高级选项** → **命令提示符**。

### 1.2 挂载 EFI 分区

在命令提示符里执行：

```bat
diskpart
list disk
select disk 0        ← 你的系统盘，若不止一块盘先用 list disk 确认
list volume
```

找到 **类型为"系统"（ESP，FAT32，一般 100~500MB）** 的那一卷，记下它的卷号（例如 2），然后：

```bat
select volume 2      ← 换成你看到的卷号
assign letter=Z:     ← 把 EFI 分区挂成 Z 盘
exit
```

### 1.3 删除 40HX 解锁残留（关键）

```bat
Z:
dir \EFI
```

看到 `\EFI\40HX` 目录就删掉：

```bat
rmdir /s /q Z:\EFI\40HX
```

顺手清理根目录和回退引导残留（有就删，没有跳过）：

```bat
del Z:\40hx_log.txt
del Z:\40hx_vbios.bin
del Z:\EFI\Boot\bootx64.efi.40hx.bak
```

### 1.4 重建 Windows BCD 引导数据库（最关键）

```bat
cd /d Z:\EFI\Microsoft\Boot
ren BCD BCD.old              ← 旧的备份起来（好习惯）
bootrec /rebuildbcd
```

按提示输入 `Y` 把扫描到的 Windows 加入引导。完成后可顺手修复引导记录（保险）：

```bat
bootrec /fixmbr
bootrec /fixboot
```

> 若 `bootrec /rebuildbcd` 扫不到系统，可改用（把 Z: 换成你的 ESP 盘符）：
> ```bat
> bcdboot C:\Windows /s Z: /f UEFI
> ```

### 1.5 收尾重启

```bat
exit
```

重启（拔掉 U 盘）。此时应能直接进 Windows。

### 1.6 进系统后的最后清理

以管理员运行发布包里的 **`40HXUninstaller.exe`**：
它会删除剩余的计划任务 / 固件启动项 / 驱动与服务 / GSP 设置，让系统彻底回到出厂状态。

> 用 **EasyUEFI 或 BIOS 启动菜单**检查一下：确认启动顺序里 Windows Boot Manager 在第一位，没有 40HX Unlock 残留项。

---

## 2. 备选：Ubuntu / Linux U 盘（只能删文件，不能修 BCD）

> 只在没有 Windows 安装 U 盘时用来应急。**Linux 修不了 Windows 的 BCD**，
> 删完文件后仍要用方案 1 的 `bootrec`/`bcdboot` 重建引导（或用 Windows U 盘）。

1. Ubuntu U 盘启动时，**务必选 `UEFI:` 开头的那一项**（否则看不到 EFI 变量）。
2. 打开终端：

```bash
# 找到 EFI 分区（一般几百 MB 的 FAT32）
sudo lsblk -f

# 挂载（nvme0n1p2 换成你的 EFI 分区）
sudo mkdir -p /mnt/efi
sudo mount /dev/nvme0n1p2 /mnt/efi

# 删除 40HX 残留（目录 + 根目录杂物 + 回退备份）
sudo rm -rf /mnt/efi/EFI/40HX
sudo rm -f  /mnt/efi/40hx_log.txt /mnt/efi/40hx_vbios.bin
sudo rm -f  /mnt/efi/EFI/Boot/bootx64.efi.40hx.bak

# 确认没有残留
sudo find /mnt/efi -iname "*40hx*" -o -iname "*unlk*"
```

3. 用 efibootmgr 删除 NVRAM 里的 40HX 启动项：

```bash
sudo efibootmgr -v          # 找到 "40HX Unlock" 对应的 BootXXXX
sudo efibootmgr -b 0001 -B  # 删除它（0001 换成实际编号）
```

4. `sudo umount /mnt/efi`，重启。
5. **仍不能进系统的话** → 回到方案 1，用 Windows U 盘重建 BCD。

---

## 3. 兜底：BIOS 层面

进 BIOS（开机狂按 Del / F2）：

1. **Boot 菜单** → 找启动项列表，把 `40HX Unlock` 设为 **Disabled** 或删除；
2. 确认 **Windows Boot Manager** 是第一启动项；
3. 若开机菜单里 40HX Unlock 一直删不掉，尝试主板 **CMOS 清零**
   （拔主板纽扣电池 30 秒，或 BIOS 里 Load Optimized Defaults）——这会清空 NVRAM 启动项。

---

## 4. 修复成功标志

- 开机直接进 Windows，无任何中间菜单 / 蓝屏报错；
- BIOS 启动菜单里没有 40HX Unlock；
- 事件日志不再报 `\EFI\40HX\40HXUNLK.EFI` 找不到。

---

## 5. 怎么避免再次发生

1. **别在解锁固件工作时断电/强制重启**——固件写入 ESP 中途断电是损坏主因；
2. 升级系统/驱动前建议先卸载解锁（`40HXUninstaller.exe`），完事再装回来；
3. 有条件的话，用 DiskGenius / Macrium Reflect 定期备份 ESP 分区（很小，几百 KB）；
4. 本次修复的完整复盘与命令备份见本目录 README 第 5 节"失败排查"与第 8 节——遇到同款报错直接按本文件第 1 节走即可。

---

*仅供个人硬件研究与学习使用，请遵守当地法律与硬件厂商条款。*
