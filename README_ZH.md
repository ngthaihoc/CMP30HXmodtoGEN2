<p align="right">
  <b>语言:</b>
  <a href="README.md">Tiếng Việt</a> |
  <a href="README_EN.md">English</a> |
  <b>简体中文</b>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> 解锁 NVIDIA CMP 30HX 与 40HX v3.0.0 (PCIe Gen2 x16 & Riot Vanguard 支持)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116%20%26%20TU106-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-%E5%85%B3%E9%97%AD%20(%E5%AE%89%E5%85%A8)-success)](#)
[![Web Control Center](https://img.shields.io/badge/Web%20UI-Localhost-blue?logo=html5&logoColor=white)](#-最快方式web-控制中心推荐)

专为 **NVIDIA CMP 30HX（TU116 核心）** 与 **CMP 40HX（TU106 核心）** 显卡打造的 **PCIe Gen2 x16 (~6.4 GB/s)** 带宽解锁与 3D 图形渲染优化方案，支持 Windows 10/11 x64 和 Linux。

- **一键安装** — 支持直观的 Web 控制中心面板或全自动批处理脚本。
- **全驱动兼容** — 兼容 NVIDIA 官方驱动、桌面魔改驱动以及最新的 5xx/6xx 分支驱动。
- **Riot 游戏与反作弊深度优化** — 内置 DirectX Global Router 全局图形路由，彻底修复《英雄联盟》图形初始化崩溃，100% 兼容 Riot Vanguard、EAC、BattlEye，无需开启测试签名（Test Signing）。
- **无需修改 BIOS** — 在 Windows/Linux 系统内部通过 BAR0 MMIO 安全解锁，无需刷写显卡 VBIOS 或主板 BIOS（针对 30HX）。
- **MRRS 512B 吞吐解锁** — 将最大读取请求大小提升至 512 Bytes，跑满 DMA 内存吞吐带宽上限 ~6.4 GB/s。

> [!TIP]
> **赞助作者 (Donate)**
>
> 本项目由作者在大学期间独立研究开发，并向开源社区完全免费分享。如果该工具对您有所帮助，欢迎赞助支持作者的持续维护！非常感谢大家！❤️
>
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
>
> - **收款人**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub 仓库**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/cpu.svg?color=%2310b981" width="22" height="22" align="center" /> 硬件支持对比

| 功能特性 | CMP 30HX (TU116) | CMP 40HX (TU106) |
|---|---|---|
| **硬件物理限制 (eFuse)** | 出厂熔断 Gen3 熔丝 (最高限速 Gen2 x16) | 支持 Gen2 x16 / Gen3 x4 (依配置而定) |
| **PCIe 解锁机制** | 操作系统运行时 BAR0 MMIO (Windows / Linux) | EFI 引导加载器 (`40HXUNLK.EFI`) 或运行时 MMIO |
| **Tensor Core 计算单元** | TU116 架构无物理 Tensor Core | 支持硬件计算 (~50 TFLOPS，通过 EFI 解锁) |
| **Riot Vanguard / 安全启动** | 系统内直接解锁，原生保持 Secure Boot 开启 | 使用 `UnlockRiotGame.exe` 为 EFI 签名并导入 Secure Boot `db` |
| **DirectX Global Router** | 自动路由至 CMP 独显进行渲染 | 自动路由至 CMP 独显进行渲染 |

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. 准备工作

**硬件与系统需求：**

- 已完成 PCIe x16 通道补阻改线的 **CMP 30HX / 40HX** 显卡，且插入与 CPU 直连的 PCIe x16 插槽。
- 显示器信号线插在核心显卡（iGPU）或副显卡输出接口上。
- 已预先安装 NVIDIA 显卡驱动（显卡在设备管理器中能正常识别，无黄色感叹号）。

**下载工具包：**

在 GitHub 页面点击 **Code → Download ZIP** 或在 Releases 页面下载发布包，解压到本地固定目录（例如：`D:\CMP30HX-Unlock`）。

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. 在 Windows 上安装

### ⭐ 最快方式：Web 控制中心（推荐）

> **仅需 1 步** — 双击运行 `Launch_WebUI.bat`。

浏览器将自动打开本地控制面板（`http://127.0.0.1:8080`），完整提供实时的 16 通道 PCIe 链路矩阵可视化、GPU 核心参数、温度监控以及 **一键解锁 Gen2** 按钮。

### 方式二：一键全自动脚本 (All-In-One)

> 右键点击 `Setup_CMP30HX_WindowsAIO.bat` → **以管理员身份运行**。

脚本会自动执行完整的配置流程：
1. **关闭 PCIe ASPM 和混合睡眠** — 防止 Windows 在 GPU 空闲时将 PCIe 链路降速。
2. **关闭快速启动 (hiberboot)** — 防止重启后卡在 Gen1 速率。
3. **关闭微软易受攻击驱动程序阻止列表** — 允许加载临时内核驱动以执行寄存器写入。
4. **关闭内存完整性 (HVCI / 内核隔离)** — 允许写入 BAR0 MMIO 寄存器（若系统当前开启，脚本会自动关闭并提示您重启一次电脑）。
5. **注册常驻守护机制 (Resident Guard)** — 创建多级计划任务（开机 + 登录 + 唤醒）与注册表自启项，在显卡被驱动重置时自动补提 Gen2。
6. **解锁 Gen2 x16 + MRRS 512B** — 显卡即刻就绪，跑满 DMA 内存带宽。

---

## <img src="https://api.iconify.design/lucide/gamepad-2.svg?color=%23ec4899" width="22" height="22" align="center" /> 3. Riot 游戏与 3D 渲染优化 (UnlockRiotGame)

如果您需要游玩 **《英雄联盟》(LoL)、《云顶之弈》(TFT)、《无畏契约》(Valorant)** 或其他 3D 游戏：

运行 `windows-v3.0\release\UnlockRiotGame.exe` 并点击 **[⚡ 1-CHẠM 优化]**（或通过命令行执行 `UnlockRiotGame.exe -silent -optimize`）。

工具将在 2 秒内全自动完成以下优化：
- **动态识别显示与渲染 GPU**：自动扫描连接显示器的 GPU 并设为显示宿主（`EnableMsHybrid=2`），同时将 CMP 显卡绑定为 3D 渲染独显（`EnableMsHybrid=1`，`CASO=1`，`FeatureScore=0xD1`）。
- **DirectX Global Router (全局图形路由)**：将 CMP 显卡的硬件 ID 注册至 `DirectXUserGlobalSettings` (`HighPerfAdapter`)。所有 3D 游戏将自动调用 CMP 显卡渲染，无需在系统图形设置中手动添加每一个 exe 文件。
- **根治《英雄联盟》“无法初始化图形设备”报错**：自动扫描游戏安装目录（Riot / 国服 / 台服），强制写入 `game.cfg` 配置为无边框窗口模式与 DirectX 11（`WindowMode=2`，`BorderlessWindow=1`，`PreferDX9LegacyMode=0`，`DX11Support=1`）。
- **针对 5xx/6xx 驱动的 Vanguard 解除拦截**：关闭 VulnerableDriverBlocklist，自动将魔改驱动证书导入系统根证书存储区，并清理残留内核驱动，确保测试签名保持关闭状态以通过 Vanguard 检测。

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 4. 在 Linux 上安装

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

Linux 脚本全自动处理：关闭 ASPM、扫描系统中所有 CMP 30HX/40HX 显卡、写入 BAR0 MMIO 寄存器、将 MRRS 提升至 512B、重训 Gen2 链路，并安装 systemd 系统服务与休眠唤醒钩子，保证重启与休眠唤醒后持续生效。

| 命令 | 功能 |
|---|---|
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --status` | 查看当前链路速率与 MRRS 状态 |
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` | 卸载自启服务与休眠钩子 |
| `./Setup_CMP30HX_LinuxAIO.sh -test --no-root` | 模拟测试（无需物理 GPU） |

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 5. 验证结果

安装完成后，可通过以下两种方式验证速率：

| 工具 | 检查项 | 正确结果 |
|---|---|---|
| **GPU-Z** 或 `40HXCheck.exe` | **Bus Interface（总线接口）** 项 | `PCIe x16 2.0 @ x16 2.0` |
| **AIDA64** → Tools → GPGPU Benchmark | **Memory Read / Memory Copy** 行 | **6.3 – 6.4 GB/s** |

> [!NOTE]
> 若 GPU-Z 已显示 Gen2 x16，但 AIDA64 测速只有约 2.5 GB/s，说明 MRRS 仍处于默认的 128 字节限制。运行 `windows-v3.0\release\40HXInstaller.exe -gen2-30hx` 即可重新激活 MRRS 512B。

---

## <img src="https://api.iconify.design/lucide/wrench.svg?color=%23f59e0b" width="22" height="22" align="center" /> 6. 辅助工具

仓库中还提供了针对常见硬件环境的辅助脚本：
- **`Fix_CMP30HX_CSM_Error43.bat`**：修复非纯 UEFI 模式（开启 CSM）下主板导致的显卡报代码 43 与资源冲突错误。
- **`Setup_ReBar_CMP30HX.bat`**：为 CMP 显卡开启 Resizable BAR 支持，解除显存访问瓶颈并提升游戏帧率。

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 7. 常见故障排查 (Troubleshooting)

| 故障现象 | 产生原因 | 解决方案 |
|---|---|---|
| **卡在 Gen1 无法提升 (GPU TLS=Gen1, Root TLS=Gen2)** | - 双显卡系统（CMP + 核显）或魔改驱动初始化时将 GPU TLS 重置回 Gen1。<br>- 系统开启了内核隔离 (HVCI)，或显卡延长线接触不良 | 1. **自动修复**：`Setup_CMP30HX_WindowsAIO.bat` 的 **Resident Guard** 常驻守护机制会自动补提速率。<br>2. **手动修复**：在设备管理器中保持显卡处于**启用 (Enable)** 状态（**切勿禁用设备**，否则会断开 MMIO 通道）→ 重新运行 `40HXInstaller.exe -gen2-30hx`。<br>3. 若开启了 HVCI：关闭内核隔离并重启电脑。 |
| **重启电脑后 Gen2 失效回退** | - APU 平台上驱动加载晚于系统自启，或 vBIOS 重置了链路。<br>- Windows 快速启动处于开启状态 | 1. 脚本已注册多级计划任务与常驻守护模式，定时自动巡检并修复。<br>2. 在系统电源选项中关闭快速启动 (Fast Startup)。<br>3. 建议在安全模式下使用 **DDU (Display Driver Uninstaller)** 彻底清除旧驱动后重新安装。 |
| **显卡空闲时自动掉速至 Gen1 x16** | Windows 系统的 PCIe ASPM 节能策略处于开启状态 | 重新运行 `Setup_CMP30HX_WindowsAIO.bat`（自动关闭 ASPM），或在电源选项 → PCI Express → 链接状态电源管理中设置为：**关闭**。 |
| **GPU-Z 显示 Gen2 x16 但 AIDA64 只有 ~2.5 GB/s** | 最大读取请求大小 (MRRS) 被限制在 128 Bytes | 执行 `40HXInstaller.exe -gen2-30hx` 命令将 MRRS 提升至 512 Bytes 并刷新 DMA 队列。 |
| **无法识别显卡 / 报错代码 43** | 改焊 x16 通道电阻虚焊或显卡未正确安装驱动 | 1. 检查显卡电阻焊接点。<br>2. 若主板开启了 CSM，运行 `Fix_CMP30HX_CSM_Error43.bat`。<br>3. 使用 DDU 彻底卸载旧驱动后重新安装。 |
| **运行《英雄联盟》提示无法初始化图形设备** | 无输出显卡在独占全屏模式下的 DirectX 适配冲突 | 运行 `UnlockRiotGame.exe` 点击 **1-CHẠM 优化**，强制将游戏配置设为无边框模式并启用 DirectX 11。 |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. 卸载与清理

| 操作系统 | 卸载方式 |
|---|---|
| **Windows** (脚本) | 运行 `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows** (图形界面) | 右键点击 `windows-v3.0\release\40HXUninstaller.exe` → **以管理员身份运行** |
| **Linux** | 运行命令 `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

卸载程序将自动删除所有计划任务、注册表自启项与程序文件，并将系统设置还原为初始状态。

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> 致谢

本项目受 **CMP40HX-Unlock** 开源社区早期研究工作的启发。如果您觉得本工具有所帮助，请在 GitHub 仓库右上角点一个 **Star** 给予作者支持！
