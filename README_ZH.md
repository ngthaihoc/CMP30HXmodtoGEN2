<p align="right">
  <b>语言:</b>
  <a href="README.md">Tiếng Việt</a> |
  <a href="README_EN.md">English</a> |
  <b>简体中文</b>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> 解锁 NVIDIA CMP 30HX v3.0.0 (PCIe Gen2 x16)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-%E6%97%A0%E9%9C%80%E5%BC%80%E5%90%AF-success)](#)

专为 **NVIDIA CMP 30HX（TU116 核心）** 显卡打造的 **PCIe Gen2 x16 (~6.4 GB/s)** 带宽解锁方案，支持 Windows 10/11 x64 和 Linux。

- **一键安装** — 脚本自动处理一切，只需双击并等待即可。
- **兼容所有驱动** — 兼容 NVIDIA 官方驱动、桌面版驱动、魔改驱动及最新版本。
- **反作弊安全** — 无需开启测试签名（Test Signing），兼容 Riot Vanguard、EAC 和 BattlEye。
- **无需修改 BIOS** — 纯软件层拦截干预，无需刷写主板 BIOS 或显卡 VBIOS。

> [!TIP]
> **赞助作者 (Donate)**
>
> 本项目由作者在大学期间独立研究开发。如果可以的话，欢迎小额赞助支持一下！非常感谢大家！❤️
>
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
>
> - **收款人**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub 仓库**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. 准备工作

**前置需求：**

- 已完成 PCIe x16 通道补阻改线的 **CMP 30HX (TU116)** 显卡，且插入与 CPU 直连的 PCIe x16 插槽。
- 已预先安装任意版本的 NVIDIA 驱动（显卡在设备管理器中能正常显示）。

**下载工具包：**

在 GitHub 页面点击 **Code → Download ZIP**，解压到本地固定目录（例如：`D:\CMP30HX-Unlock`）。

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. 在 Windows 上安装

### ⭐ 最快方式：Web 控制中心（推荐）

> **仅需 1 步** — 双击 `Launch_WebUI.bat` 并跟随界面操作。

浏览器将自动打开控制面板，完整显示 GPU 信息、PCIe 链路状态，并提供一键解锁 Gen2 按钮。

### 方式二：一键自动化脚本

> **仅需 1 步** — 右键点击 `Setup_CMP30HX_WindowsAIO.bat` → **以管理员身份运行**。

脚本会自动处理所有配置：关闭 ASPM、关闭快速启动、配置注册表、解锁 Gen2 x16、提升 MRRS 至 512B 并注册每次开机自启。

> [!IMPORTANT]
> 如果您的系统当前**开启了内存完整性（内核隔离 / HVCI）**，脚本会自动关闭它并提示您**重启一次电脑**。重新登录系统后，程序将自动切换至 Gen2 x16。

<details>
<summary>📋 详情：脚本执行了哪些操作？</summary>

1. **关闭 PCIe ASPM 和混合睡眠** — 防止 Windows 在 GPU 空闲时将 PCIe 链路降速。
2. **关闭快速启动（hiberboot）** — 防止重启后卡在 Gen1 速率。
3. **关闭微软易受攻击驱动程序阻止列表** — 允许加载 WinRing0 / ThrottleStop 内核驱动。
4. **关闭内存完整性（HVCI）** — 允许向 BAR0 MMIO 寄存器写入数据。
5. **注册开机自启维持机制** — 创建系统计划任务（开机 + 登录 + 睡眠唤醒）和注册表自启项，确保持久维持 Gen2 速率。
6. **解锁 Gen2 x16 + MRRS 512B** — 显卡即刻就绪，无需重启（关闭 HVCI 的情况除外）。

</details>

<details>
<summary>🔧 方式三：命令行手动安装（适合高级用户）</summary>

以管理员身份打开 **PowerShell** 或 **命令提示符**：

**步骤 1 — 关闭 PCIe ASPM 和混合睡眠：**
```cmd
powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
powercfg -setacvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
powercfg -setdcvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
powercfg -setactive SCHEME_CURRENT
```

**步骤 2 — 关闭快速启动和驱动阻止列表：**
```cmd
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f
```

**步骤 3 — 关闭内存完整性（若已开启）：**
```cmd
reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f
```
*（若此项值从 1 改为 0，需重启电脑生效）。*

**步骤 4 — 解锁 Gen2 x16：**
```cmd
cd /d "解压目录路径"
.\windows-v3.0\release\40HXInstaller.exe -gen2-30hx
```

**步骤 5 — 注册开机自动维持：**
```cmd
if not exist "%ProgramFiles%\40HXUnlock" mkdir "%ProgramFiles%\40HXUnlock"
copy /y ".\windows-v3.0\release\40HXInstaller.exe" "%ProgramFiles%\40HXUnlock\"
copy /y ".\windows-v3.0\release\40HXCheck.exe" "%ProgramFiles%\40HXUnlock\"
```

```powershell
$action = New-ScheduledTaskAction -Execute "$env:ProgramFiles\40HXUnlock\40HXInstaller.exe" -Argument '-gen2-30hx -silent' -WorkingDirectory "$env:ProgramFiles\40HXUnlock"
$t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'
$t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew
$principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force
```

```cmd
reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
```

</details>

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 3. 在 Linux 上安装

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

脚本自动处理：关闭 ASPM、扫描系统中所有 CMP 30HX/40HX 显卡、写入 BAR0 MMIO 寄存器、提升 MRRS 至 512B、重训 Gen2 链路，并安装 systemd 服务以在每次重启后自动维持。

| 命令 | 功能 |
|---|---|
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --status` | 检查链路状态与 MRRS |
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` | 卸载 systemd 服务与钩子 |
| `./Setup_CMP30HX_LinuxAIO.sh -test --no-root` | 模拟测试（无需物理 GPU） |

<details>
<summary>📋 Linux 脚本技术详情</summary>

1. **关闭 PCIe ASPM 和运行时电源管理** — 将内核电源策略设置为 `performance`，并将所有 PCI 设备的 `power/control` 设为 `on`。
2. **扫描全部显卡** — 自动检测 CMP 30HX (`10de:2189`) 与 CMP 40HX (`10de:1f0b`)。
3. **BAR0 MMIO 直接注入** — 干预写入 `XVE_OVR`、`PRIV_MISC_1`、`LINK_CONFIG_0`、`LNKCAP`、`LNKCTL2` 寄存器。
4. **MRRS 512B + 链路重训** — 目标速率 = Gen2，MRRS 设为 512 字节，重训后达到 ~6.4 GB/s。
5. **Systemd 服务 + 睡眠钩子** — 安装 `cmp30hx-gen2-unlock.service` 与 `/lib/systemd/system-sleep/cmp30hx-unlock`。
6. **StatusContract** — 输出 `gen2_status.txt`（或通过 `--status-file` 指定路径）：
   ```text
   STATUS_CODE=GEN2_SUCCESS
   SPEED_CURRENT=2
   WIDTH_CURRENT=16
   TLS_TARGET=2
   ERROR_CODE=NONE
   TIMESTAMP=2026-09-26T05:30:00Z
   ```

</details>

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 4. 验证结果

安装完成后，可通过以下两种方式之一进行验证：

| 工具 | 检查方式 | 正确结果 |
|---|---|---|
| **GPU-Z** 或 `40HXCheck.exe` | **Bus Interface（总线接口）** 项 | `PCIe x16 2.0 @ x16 2.0` |
| **AIDA64** → Tools → GPGPU Benchmark | **Memory Read / Memory Copy** 行 | **6.3 – 6.4 GB/s** |

> 若 AIDA64 仅达到约 2.5 GB/s → 说明 MRRS 仍处于默认的 128B 限制 → 重新运行 `40HXInstaller.exe -gen2-30hx`。

---

## <img src="https://api.iconify.design/lucide/shield-check.svg?color=%2306b6d4" width="22" height="22" align="center" /> 5. 反作弊系统兼容性 (Riot Vanguard, EAC, BattlEye)

解锁后完全可以正常畅玩 **Valorant（无畏契约）、英雄联盟、Apex 英雄、堡垒之夜** 等游戏：

- **CMP 30HX** — 在 Windows 内部通过 BAR0 MMIO 解锁 Gen2（无需加载 EFI 引导），原生保持 Secure Boot 开启 → 100% 完美兼容 Riot Vanguard。
- **CMP 40HX** — 需额外运行 `UnlockRiotGame.exe` 为 EFI 固件进行数字签名，并将证书导入主板安全启动 db 数据库。
- **用完即卸载机制** — 内核驱动仅加载数毫秒便自动卸载清理。游戏运行时，系统内核处于绝对纯净状态。
- **无需执行 `bcdedit /set testsigning on`** — 该命令会被 Vanguard 100% 拦截，本工具绝不需要开启测试模式。

<details>
<summary>📋 详情：CMP 40HX (TU106) 的 Riot Vanguard 解决方案</summary>

Windows 11 上的 Riot Vanguard 强制要求 Secure Boot = Enabled 与 TPM 2.0。由于 `40HXUNLK.EFI` 固件未经微软官方签名，`UnlockRiotGame.exe` 将自动完成：

1. 生成 SHA256 自签名 X.509 证书（`CMP40HX_Key.cer`）。
2. 使用 Authenticode 为 `40HXUNLK.EFI` 进行数字签名（同时签署发布目录与 ESP 分区中的固件）。
3. 将证书导出至 `C:\`、桌面及 ESP 分区（`\EFI\40HX\`）。
4. 引导用户重启进入主板 BIOS → **Custom Mode（自定义模式）** → 将 `CMP40HX_Key.cer` 导入 **db**（Key Management → Authorized Signatures → Append Key）。
5. 达成效果：安全启动保持开启状态（满足 Vanguard 要求），同时 EFI 固件成功执行并解锁 Tensor Core（~50 TFLOPS）与 PCIe Gen2。

</details>

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 6. 常见故障排查 (Troubleshooting)

| 故障现象 | 产生原因 | 解决方案 |
|---|---|---|
| **卡在 Gen1 无法提升 (GPU TLS=Gen1, Root TLS=Gen2)** | - 双显卡系统（CMP 30HX + AMD/Intel 核显）或魔改驱动（RainCandy）重新初始化，自动将 GPU TLS 重置为 Gen1。<br>- 或 Windows 开启了内存完整性（HVCI），或显卡延长线接触不良 | 1. **自动解决**：新版 `Setup_CMP30HX_WindowsAIO.bat` 激活了**常驻守护（Resident Guard）**模式与登录任务（延迟 10 秒），自动补提 Gen2。<br>2. **手动解决**：在设备管理器中保持显卡处于**启用（Enable）**状态（**严禁**禁用设备，否则会切断 BAR0 MMIO 连接）→ 待桌面加载稳定后运行 `Setup_CMP30HX_WindowsAIO.bat` 或 `40HXInstaller.exe -gen2-30hx`。<br>3. 若开启了 HVCI：关闭内存完整性并**重启电脑**。 |
| **重启电脑后 Gen2 失效回退** | - NVIDIA / RainCandy 驱动在 APU 平台上加载较慢，或在开机后覆盖了 vBIOS 状态。<br>- 旧驱动残留配置冲突，或快速启动（Fast Startup）仍处于开启状态 | 1. `Setup_CMP30HX_WindowsAIO.bat` 已集成多层计划任务（登录延迟 10 秒 + 开机延迟 45 秒）与 **DriverStrategy=2 (Resident Guard)** 模式，每分钟自动检测并在失去 Gen2 时重新重训。<br>2. **DDU 建议**：在安全模式下使用 **DDU (Display Driver Uninstaller)** 彻底清除所有旧显卡驱动残留，再安装驱动，以防注册表冲突。 |
| **显卡空闲时自动掉速至 Gen1 x16** | Windows 系统的 PCIe ASPM 节能策略处于开启状态 | 重新运行 `Setup_CMP30HX_WindowsAIO.bat`（脚本会自动关闭 ASPM），或在电源选项 → PCI Express → 链接状态电源管理中设置为：**关闭**。 |
| **GPU-Z 显示 Gen2 x16 但 AIDA64 测速只有 ~2.5 GB/s** | 显卡的最大读取请求大小（MRRS）被限制在默认的 128 字节 | 运行命令 `40HXInstaller.exe -gen2-30hx`，将 MRRS 提升至 512 字节并刷新 DMA 队列。 |
| **无法识别显卡 / 报错代码 43** | 改焊 x16 通道电阻虚焊或接触不良，或者显卡未正确识别驱动 | 1. 重新检查显卡上的改电阻焊接点。<br>2. 重新安装 NVIDIA 驱动（建议使用 DDU 彻底清除后重装最新版）。 |
| **尝试了所有方法仍无法解决** | NVIDIA 驱动配置冲突、注册表配置文件损坏或驱动服务异常 | 在 Windows 安全模式下使用 **DDU (Display Driver Uninstaller)** 清除旧驱动后重新安装驱动。 |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 7. 卸载与清理

| 操作系统 | 命令 |
|---|---|
| **Windows**（脚本） | `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows**（图形界面） | 右键点击 `40HXUninstaller.exe` → 以管理员身份运行 |
| **Linux** | `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

工具将自动删除系统计划任务、注册表自启项、程序安装目录以及临时文件。

---

## <img src="https://api.iconify.design/lucide/info.svg?color=%238b5cf6" width="22" height="22" align="center" /> 8. 技术说明

> [!NOTE]
> - **为什么只能达到 Gen2 而无法开启 Gen3？**
>   NVIDIA 在出厂时于 TU116 芯片上物理熔断了硬件电子熔丝（Silicon eFuse Bit 3 - 8.0 GT/s）。Gen2 x16 (5.0 GT/s) 是其绝对的硬件物理上限 — 本工具通过 BAR0 MMIO 安全解锁至该上限，绝不强行协商 Gen3，避免造成链路重训卡死。
> - **MRRS 512B 优化**：
>   将最大读取请求大小（Max Read Request Size）从 128B 提升至 512B，消除了 TLP 数据包分片瓶颈，跑满 ~6.4 GB/s 的 DMA 内存吞吐带宽。

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> 致谢

> 本项目受 **CMP40HX-Unlock** 早期研究工作的启发。如果您觉得本工具有所帮助，请在仓库右上角点一个 Star 给予作者支持！
