<p align="right">
  <b>语言:</b>
  <a href="README.md">Tiếng Việt</a> |
  <a href="README_EN.md">English</a> |
  <b>简体中文</b>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> NVIDIA CMP 30HX 解锁工具 v3.0.0 (PCIe Gen2 x16)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-%E6%97%A0%E9%9C%80%E5%BC%80%E5%90%AF-success)](#)

专为 **NVIDIA CMP 30HX（TU116 核心）** 显卡打造的 **PCIe Gen2 x16（~6.4 GB/s）** 完整带宽解锁方案，支持 Windows 10/11 x64 及 Linux（Ubuntu、Debian、HiveOS、RaveOS、Fedora、Arch Linux）。

- **一键自动化部署**：自动优化系统环境、关闭 PCIe ASPM 节能、配置内存完整性，并在开机与唤醒时全自动维持 Gen2。
- **全驱动版本兼容**：支持所有 NVIDIA 驱动（官方 Game Ready / Studio、桌面改 INF 驱动、第三方魔改驱动，无 537.58 等版本限制）。
- **纯净安全不破坏系统**：无需刷写 BIOS/VBIOS，无需挂载 EFI 引导补丁，无需开启 Windows 测试模式（Test Signing），完全不影响竞技游戏反作弊系统。

> [!TIP]
> **赞助与支持作者 (Donate)**：  
> 本项目由作者在大学期间独立研究开发。如果本工具对您有所帮助，欢迎支持鼓励！非常感谢大家！❤️  
> 
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
> 
> - **账户名**：NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub 仓库**：[https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. 准备与下载

### 1.1 软硬件要求
1. **硬件要求**：
   - **CMP 30HX (TU116)** 显卡需已完成物理 PCIe 通道（Lane）补阻改装，能够识别为 x16 模式。
   - 必须插入 CPU 直连的 PCIe x16 插槽（避免使用劣质延长线或芯片组转接的副插槽）。
2. **驱动要求**：
   - 安装任意版本的 NVIDIA 显卡驱动（官方驱动、桌面魔改驱动均可）。
   - 在 **设备管理器** 中显卡状态显示正常（未解锁前默认识别为 Gen1 x16）。

### 1.2 下载工具包
- 在 GitHub 页面点击 **Code $\rightarrow$ Download ZIP** 下载整个仓库（或使用 `git clone`）。
- 解压到本地固定目录（例如：`C:\CMP30HX-Unlock` 或 `D:\CMP30HX-Unlock`）。

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. 详细安装指南

### 方法一：一键自动安装（推荐绝大多数用户使用）

在解压后的目录中，右键点击 **`Setup_CMP30HX_WindowsAIO.bat`** 并选择 **以管理员身份运行**（也可以直接双击，脚本会自动请求管理员权限）。

脚本将自动执行以下 6 项系统优化与解锁步骤：
1. **关闭 PCIe ASPM（活动状态电源管理）与混合睡眠**：
   - 防止 Windows 在 GPU 闲置时自动将 PCIe 链路降速至 Gen1。
2. **关闭快速启动（Fast Startup / Hiberboot）**：
   - 防止关机时缓存内核会话，彻底消除重启后卡回 Gen1 的问题。
3. **关闭微软易受攻击驱动程序阻止列表（Vulnerable Driver Blocklist）**：
   - 防止 Windows Defender / 代码完整性服务阻止加载 WinRing0 / ThrottleStop 内核驱动。
4. **检查并关闭内存完整性（内核隔离 / HVCI）**：
   - 调整注册表以解除 Windows 对内核驱动的拦截，确保工具能向 BAR0 MMIO 寄存器写入数据。
5. **部署驻留文件并注册多重维持机制 (`CMP30HX_Gen2_Unlock`)**：
   - 自动复制工具集到 `%ProgramFiles%\40HXUnlock\`，防止重启后文件丢失。
   - 在 `NT AUTHORITY\SYSTEM` 权限下创建系统计划任务，配置三重触发器（Multi-triggers）：
     + **AtStartup**（内核加载后延迟 15 秒执行）。
     + **AtLogOn**（用户登录后延迟 5 秒执行）。
     + **Wake from Sleep**（系统从睡眠或待机唤醒时通过 Power-Troubleshooter 事件 ID 1 触发）。
   - 结合注册表自启项（`HKLM` 与 `HKCU`）双重保险，确保无论重启还是睡眠唤醒，永久锁定在 Gen2 x16。
6. **立即激活 Gen2 x16 并开启 MRRS 512B 优化**：
   - 即刻将链路带宽提升至 Gen2 x16，并将最大读取请求大小（MRRS）优化至 512 字节，显卡即刻生效，无需重启即可开始使用。

> [!IMPORTANT]
> **关于重启电脑的重要提示**：  
> - 如果您的电脑此前**开启了内存完整性（内核隔离 / HVCI）**，脚本会自动将其关闭并弹出提示。  
> - 在这种情况下，您**必须重启电脑一次**，以使 Windows 彻底解除虚拟机监控保护。重启并进入系统后，程序会自动将链路切换至 Gen2 x16。

---

### 方法二：命令行手动安装（适合高级用户）

如果您希望在终端（CMD / PowerShell 管理员窗口）中手动执行每一步：

1. **关闭 PCIe ASPM 与混合睡眠**：
   ```cmd
   powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
   powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
   powercfg -setacvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
   powercfg -setdcvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
   powercfg -setactive SCHEME_CURRENT
   ```

2. **关闭快速启动与驱动阻止列表**（防止重启后回落到 Gen1）：
   ```cmd
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f
   ```

3. **关闭内存完整性（若此前已开启）**：
   ```cmd
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f
   ```
   *（若此项数值由 1 修改为 0，需重启电脑生效）。*

4. **立即触发解锁 Gen2 x16**：
   ```cmd
   cd /d "解压目录路径"
   .\windows-v3.0\release\40HXInstaller.exe -gen2-30hx
   ```

5. **配置开机与睡眠唤醒自动维持 Gen2**：
   - **复制工具到系统目录**：
     ```cmd
     if not exist "%ProgramFiles%\40HXUnlock" mkdir "%ProgramFiles%\40HXUnlock"
     copy /y ".\windows-v3.0\release\40HXInstaller.exe" "%ProgramFiles%\40HXUnlock\"
     copy /y ".\windows-v3.0\release\40HXCheck.exe" "%ProgramFiles%\40HXUnlock\"
     ```
   - **通过 PowerShell 注册带有多重触发器的 SYSTEM 计划任务**：
     ```powershell
     $action = New-ScheduledTaskAction -Execute "$env:ProgramFiles\40HXUnlock\40HXInstaller.exe" -Argument '-gen2-30hx -silent' -WorkingDirectory "$env:ProgramFiles\40HXUnlock"
     $t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'
     $t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'
     $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew
     $principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
     Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force
     ```
   - **通过注册表添加双重开机启动项（HKLM 和 HKCU）**：
     ```cmd
     reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
     reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
     ```

---

### 方法三：Linux 系统全自动安装（Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux）

适用于 Linux 操作系统、矿机系统或 AI 计算节点。脚本 **`Setup_CMP30HX_LinuxAIO.sh`** 为独立的一体化脚本，无需额外依赖即可直接执行：

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

**`Setup_CMP30HX_LinuxAIO.sh` 自动化特性：**
1. **关闭 PCIe ASPM 与运行时电源管理**：将内核电源策略设为 `performance`，并将所有 PCI 设备的 `power/control` 设为 `on`，防止闲置掉速。
2. **扫描并处理系统中所有显卡**：自动识别系统中安装的所有 CMP 30HX (`10de:2189`) 和 CMP 40HX (`10de:1f0b`) 显卡。
3. **BAR0 MMIO 寄存器直写注入**：干预内核寄存器（`XVE_OVR`、`PRIV_MISC_1`、`LINK_CONFIG_0`、`LNKCAP`、`LNKCTL2`）。
4. **优化 MRRS 512B 并触发链路重协商**：配置目标速率为 Gen2，将 DEVCTL 最大读取请求大小（MRRS）拉升至 512 字节，跑满 ~6.4 GB/s 理论极限。
5. **配置 Systemd 服务与睡眠钩子**：自动安装并启动 `cmp30hx-gen2-unlock.service` 以及睡眠钩子 `/lib/systemd/system-sleep/cmp30hx-unlock`，确保重启或唤醒后稳定维持 Gen2。

**Linux 常用管理命令：**
- 查看当前链路与 MRRS 状态：
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --status
  ```
- 彻底卸载 systemd 服务与钩子：
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
  ```
- 在 WSL / Linux / CI 上运行自动化 Mock 单元测试（无需物理 GPU）：
  ```bash
  chmod +x Test_Mock_CMP30HX_Linux.sh && ./Test_Mock_CMP30HX_Linux.sh
  ```

**StatusContract (Seam 2) 状态格式：**
脚本会将机器可读的结构化状态头写入 `/var/run/cmp30hx_gen2_status.txt`（或通过 `--status-file` 指定路径）：
```text
STATUS_CODE=GEN2_SUCCESS
SPEED_CURRENT=2
WIDTH_CURRENT=16
TLS_TARGET=2
ERROR_CODE=NONE
TIMESTAMP=2026-09-26T05:30:00Z
```

---

---

## <img src="https://api.iconify.design/lucide/shield-check.svg?color=%2306b6d4" width="22" height="22" align="center" /> 3. 完美兼容反作弊系统 (Riot Vanguard, EAC, BattlEye)

CMP 40HX 与 CMP 30HX 用户可畅玩具有高强度反作弊机制的游戏，如 **Valorant（无畏契约）、英雄联盟（Riot Vanguard）、Apex 英雄、堡垒之夜（EAC / BattlEye）**：

- **通过 `UnlockRiotGame.exe` 解决 Windows 11 下 Riot Vanguard 限制**：
  - **针对 CMP 40HX (TU106)**：Windows 11 下的 Riot Vanguard 强制要求 Secure Boot = Enabled 与 TPM 2.0。然而预引导固件 `40HXUNLK.EFI` 属于第三方未经微软认证的 EFI 程序。自动化工具 `windows-v3.0/release/UnlockRiotGame.exe` 可全自动：
    1. 生成自签名 X.509 安全证书（`CMP40HX_Key.cer`，SHA256 签名）。
    2. 使用 Authenticode 技术为 `40HXUNLK.EFI` 进行数字签名（同时签署发布目录与活动 ESP 分区中的文件）。
    3. 将证书文件 `CMP40HX_Key.cer` 导出到 C 盘根目录、桌面以及 ESP 分区（`\EFI\40HX\`）。
    4. 弹出图文说明界面，提示用户拍照保存步骤，重启电脑进入主板 BIOS，将安全启动切换为 **Custom Mode（自定义模式）**，并将 `CMP40HX_Key.cer` 导入可信签名数据库 **`db`**（Key Management -> Authorized Signatures -> Append Key）。
    5. 达成效果：系统既保持 **Secure Boot 开启状态** 以满足 Riot Vanguard (`vgk.sys`) 的检测，主板又能执行 `40HXUNLK.EFI` 固件以解锁全部 Tensor Core 算力（`SS0=0x88888888`，~50 TFLOPS）与 PCIe Gen2。
  - **针对 CMP 30HX (TU116)**：由于 TU116 核心在物理层面上没有 Tensor Core，且 Gen2 完全通过 Windows ring-0 MMIO 寄存器解锁（无需加载任何 EFI 引导程序），用户**在 BIOS 中正常保持 Secure Boot 开启**即可，无需导入任何密钥，天然 100% 兼容 Riot Vanguard。
- **用完即释放机制 (Transient BYOVD on-demand)**：
  - 内核驱动（`WinRing0x64.sys`、`ThrottleStop.sys`）仅在系统开机或用户登录时加载几毫秒以配置 PCIe 寄存器。
  - 一旦链路协商完成，工具立即停止驱动服务（`sc stop`）、删除服务（`sc delete`）并将 `.sys` 驱动文件从系统目录中移除。
  - 当游戏或 Vanguard 扫描内核空间时，操作系统处于完全干净状态，不存在任何被列入黑名单的驱动程序或常驻钩子。
- **无需开启 Windows 测试模式 (Test Signing)**：
  - 不需要执行危险的 `bcdedit /set testsigning on`（该命令会被 Vanguard 100% 拦截封堵）。
  - 完整保留微软 Windows 原生代码完整性认证。
- **CMP 40HX Pre-boot EFI 引导**：
  - Tensor Core 在 Windows 内核加载之前的 UEFI 阶段就已完成解锁。当 Windows 和反作弊驱动启动时，显卡在硬件底层已处于自然解锁状态。

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 4. 常见问题排查 (Troubleshooting)

| 故障现象 | 产生原因 | 解决方案 |
|---|---|---|
| **卡在 Gen1 无法提升 (GPU TLS=Gen1, Root TLS=Gen2)** | - 双显卡系统（CMP 30HX + Intel 核显）或魔改驱动（RainCandy）持有了 DMA 句柄，阻止 GPU 重协商 PCIe 速率。<br>- 或 Windows 开启了内存完整性（HVCI），或显卡延长线接触不良 | 1. **自动解决**：最新版 `Setup_CMP30HX_WindowsAIO.bat` 已内置自动 **软复位机制（通过 PnP 禁用并立即重新启用设备）**，释放驱动锁定并重新协商 Gen2。<br>2. **手动解决**：打开 **设备管理器** $\rightarrow$ 右键点击 **NVIDIA CMP 30HX** $\rightarrow$ 选择 **禁用设备** $\rightarrow$ 紧接着选择 **启用设备** $\rightarrow$ 重新运行 `Setup_CMP30HX_WindowsAIO.bat`。<br>3. 若开启了 HVCI：关闭内存完整性并**重启电脑**。 |
| **重启电脑后 Gen2 失效回退** | - NVIDIA / RainCandy 驱动加载较慢或在开机后重置了 vBIOS 状态。<br>- 旧驱动残留配置冲突，或快速启动（Fast Startup）仍处于开启状态 | 1. `Setup_CMP30HX_WindowsAIO.bat` 已配置 SYSTEM 计划任务（延迟 15 秒）与轮询脚本，开机后自动检测并重新补提。<br>2. **DDU 推荐**：在安全模式下使用 **DDU (Display Driver Uninstaller)** 彻底清除旧显卡驱动残留，再安装新驱动，避免注册表冲突。 |
| **显卡空闲时自动掉速至 Gen1 x16** | Windows 系统的 PCIe ASPM 电源节能策略正在运行 | 重新运行 `Setup_CMP30HX_WindowsAIO.bat`（会自动关闭 ASPM），或在 Windows 电源选项 $\rightarrow$ PCI Express $\rightarrow$ 链接状态电源管理中设置为：**关闭**。 |
| **GPU-Z 显示 Gen2 x16 但 AIDA64 测速只有 ~2.5 GB/s** | 显卡的最大读取请求大小（MRRS）被卡在默认的 128 字节 | 运行命令 `40HXInstaller.exe -gen2-30hx`，将 MRRS 提升至 512 字节并刷新 DMA 队列。 |
| **无法识别显卡 / 报错代码 43** | 改焊 x16 通道电阻虚焊或接触不良，或者显卡未正确安装驱动 | 1. 重新检查显卡上的物理改电阻焊接点。<br>2. 重新安装 NVIDIA 驱动（建议使用 DDU 彻底清除后重装）。 |
| **尝试了所有方法仍无法成功** | NVIDIA 驱动配置冲突、注册表配置文件损坏或驱动服务异常 | 在 Windows 安全模式下使用 **DDU (Display Driver Uninstaller)** 清除旧驱动并重新安装。 |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 5. 卸载与清理 (Uninstall)

如需将系统还原至初始状态：
- **快速命令**：运行 `Setup_CMP30HX_WindowsAIO.bat -uninstall`。
- **图形界面**：右键点击 **`40HXUninstaller.exe`** $\rightarrow$ 选择 **以管理员身份运行**。

卸载程序将自动清理以下项目：
- 删除 Windows 计划任务 `CMP30HX_Gen2_Unlock`。
- 清除注册表后台启动项（`HKLM` 与 `HKCU`）。
- 删除安装目录 `%ProgramFiles%\40HXUnlock`。
- 清理 `%ProgramData%\40HXUnlock` 下的临时驱动与状态文件。

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 6. 验证与带宽测试

在完成解锁后（或重新登录系统后），可通过以下两款工具进行验证：

1. **检查 PCIe 链路速度**：
   - 运行 **`40HXCheck.exe`**（或打开 **GPU-Z**）：
   - 查看 **Bus Interface / PCIe** 栏目：应正确显示 **`PCIe x16 2.0 @ x16 2.0`**（或 `Gen2 x16`）。
2. **测试实际传输带宽（验证 MRRS 512B 优化）**：
   - 打开 **AIDA64** $\rightarrow$ 点击菜单栏 **Tools（工具）** $\rightarrow$ 选择 **GPGPU Benchmark**。
   - 选中 CMP 30HX 显卡，点击 **Run Benchmarks**：
   - 观察 **Memory Read** 与 **Memory Copy** 两项读写数据：
     - **成功达标**：达到约 **6.3 – 6.4 GB/s**（达到 Gen2 x16 理论带宽上限）。
     - **未优化**：若仅有 ~2.5 GB/s，说明 MRRS 仍处于默认的 128B 状态（需重新运行 `40HXInstaller.exe -gen2-30hx` 激活 512B 优化）。

---

## <img src="https://api.iconify.design/lucide/info.svg?color=%238b5cf6" width="22" height="22" align="center" /> 7. 核心技术说明

> [!NOTE]
> - **为什么上限只能到 Gen2 x16，无法开启 Gen3？**  
>   在 CMP 30HX 的 TU116 核心中，NVIDIA 在出厂时通过物理手段烧断了 **硅晶圆电子熔丝（Silicon eFuse Bit 3 - 8.0 GT/s）**。因此，Gen2 x16 (5.0 GT/s) 是该芯片的硬件物理极限。本工具通过 BAR0 MMIO 安全解锁至此物理上限，绝不强行协商 Gen3，避免引发重训死锁或系统卡死。
> - **MRRS 512B 优化原理**：  
>   将最大读取请求大小（MRRS）从 128 字节提升至 512 字节，消除了 TLP 数据包传输中的严重分片瓶颈，从而彻底释放 ~6.4 GB/s 的 DMA 内存吞吐带宽。

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> 致谢 (Acknowledgments)

> 本项目受 **CMP40HX-Unlock** 早期研究工作的启发。如果您觉得本工具好用，请在 GitHub 仓库右上角点一个 Star 给予作者支持！
