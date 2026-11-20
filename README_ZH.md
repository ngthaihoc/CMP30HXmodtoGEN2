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

- **一键安装**: 支持直观的 Web 控制中心面板或全自动批处理脚本，双击运行即可。
- **全驱动兼容**: 兼容 NVIDIA 官方驱动、桌面魔改驱动以及最新的 5xx/6xx 分支驱动。
- **Riot 游戏与反作弊深度优化**: 内置 DirectX Global Router 全局图形路由，彻底修复《英雄联盟》图形初始化崩溃，100% 兼容 Riot Vanguard、EAC、BattlEye，无需开启测试签名（Test Signing）。
- **无需修改 BIOS**: 在 Windows/Linux 系统内部通过 BAR0 MMIO 安全解锁，无需刷写显卡 VBIOS 或主板 BIOS（针对 30HX）。
- **MRRS 512B 吞吐解锁**: 将最大读取请求大小提升至 512 Bytes，跑满 DMA 内存吞吐带宽上限 ~6.4 GB/s。

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
| **本工具的 PCIe 目标** | Gen2 x16 | Gen2 x16 |
| **PCIe 解锁机制** | 操作系统运行时 BAR0 MMIO (Windows / Linux) | EFI 引导加载器 (`40HXUNLK.EFI`) 或运行时 MMIO |
| **Tensor Core 计算单元** | TU116 架构无物理 Tensor Core | 支持硬件计算 (~50 TFLOPS，通过 EFI 解锁) |
| **Riot Vanguard / 安全启动** | 系统内直接解锁，原生保持 Secure Boot 开启 | 使用 `UnlockRiotGame.exe` 为 EFI 签名并导入 Secure Boot `db` |
| **DirectX Global Router** | 自动路由至 CMP 独显进行渲染 | 自动路由至 CMP 独显进行渲染 |

在受测的 TU116 样卡上，请求 Gen3 后读回的目标速率仍为 Gen2。造成这一现象的硬件原因尚未确定。

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. 准备工作

**硬件与系统需求：**

- 已完成 PCIe x16 通道补阻改线的 **CMP 30HX / 40HX** 显卡，且插入与 CPU 直连的 PCIe x16 插槽。
- 显示器信号线插在核心显卡（iGPU）或副显卡输出接口上。
- 已预先安装 NVIDIA 显卡驱动（显卡在设备管理器中能正常识别，无黄色感叹号）。

**下载工具包：**

在 GitHub 页面点击 **Code -> Download ZIP** 或在 Releases 页面下载发布包，解压到本地固定目录（例如：`D:\CMP30HX-Unlock`）。

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. 在 Windows 上安装

### ⭐ CMP 控制中心

> **仅需 1 步**: 双击运行 `Launch_WebUI.bat`。

启动器优先打开 Wails 桌面应用 `UnlockAIO.exe`。该文件缺失时，启动器会通过 `40HXInstaller.exe` 启动本地网页界面，并在浏览器中打开工具显示的地址。

### 方式二：一键全自动脚本 (All-In-One)

> 右键点击 `Setup_CMP30HX_WindowsAIO.bat` -> **以管理员身份运行**。

此脚本适用于 CMP 30HX（`10DE:2189`）。
1. 检查显卡、安装文件及所需驱动。
2. 复制运行文件，尝试配置 PCIe Gen2 并读取结果。在负载下验证 Gen2 x16。
3. 解锁成功后注册 Windows 开机和登录任务。驱动被拦截或任务注册失败时，安装程序会报告错误。

---

## <img src="https://api.iconify.design/lucide/gamepad-2.svg?color=%23ec4899" width="22" height="22" align="center" /> 3. Riot 游戏与 3D 渲染优化 (UnlockRiotGame)

如果您需要游玩 **《英雄联盟》(LoL)、《云顶之弈》(TFT)、《无畏契约》(Valorant)** 或其他 3D 游戏：

`windows-v3.0\release\UnlockRiotGame.exe` 的 **[⚡ 一键优化]** 操作可能更改 Windows 安全配置。命令行模式 `-silent -optimize` 需要通过 `-confirm-riot` 或 `-y` 明确确认。

优化功能包括：
- **动态识别显示与渲染 GPU**: 自动扫描连接显示器的 GPU 并设为显示宿主（`EnableMsHybrid=2`），同时将 CMP 显卡绑定为 3D 渲染独显（`EnableMsHybrid=1`，`CASO=1`，`FeatureScore=0xD1`）。
- **DirectX Global Router (全局图形路由)**: 将 CMP 显卡的硬件 ID 注册至 `DirectXUserGlobalSettings` (`HighPerfAdapter`)。所有 3D 游戏将自动调用 CMP 显卡渲染，无需在系统图形设置中手动添加每一个 exe 文件。
- **根治《英雄联盟》“无法初始化图形设备”报错**: 自动扫描游戏安装目录（Riot / 国服 / 台服），强制写入 `game.cfg` 配置为无边框窗口模式与 DirectX 11（`WindowMode=2`，`BorderlessWindow=1`，`PreferDX9LegacyMode=0`，`DX11Support=1`）。
- **安全配置**：视系统配置而定，优化可能更改驱动策略或 Windows 证书存储。Vanguard 兼容性取决于所测试的驱动和环境。

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 4. 在 Linux 上安装

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

Linux 脚本适用于 CMP 30HX（`10DE:2189`）。脚本以 PCIe Gen2 x16 和至少 512 字节的 MRRS 为目标，也可安装 systemd 服务和唤醒钩子。请在重启及唤醒后，在负载下验证链路速率。

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
| **AIDA64** -> Tools -> GPGPU Benchmark | **Memory Read / Memory Copy** 行 | **6.3 - 6.4 GB/s** |

> [!NOTE]
> 若 GPU-Z 已显示 Gen2 x16，但 AIDA64 测速只有约 2.5 GB/s，说明 MRRS 仍处于默认的 128 字节限制。运行 `windows-v3.0\release\40HXInstaller.exe -gen2-30hx` 即可重新激活 MRRS 512B。

---

## <img src="https://api.iconify.design/lucide/wrench.svg?color=%23f59e0b" width="22" height="22" align="center" /> 6. 辅助工具

仓库中还提供了针对常见硬件环境的辅助脚本：
- **`Fix_CMP30HX_CSM_Error43.bat`**: 修复非纯 UEFI 模式（开启 CSM）下主板导致的显卡报代码 43 与资源冲突错误。
- **`Setup_ReBar_CMP30HX.bat`**: 为 CMP 显卡开启 Resizable BAR 支持，解除显存访问瓶颈并提升游戏帧率。

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 7. 常见故障排查 (Troubleshooting)

| 故障现象 | 产生原因 | 解决方案 |
|---|---|---|
| **卡在 Gen1（GPU TLS=Gen1，Root TLS=Gen2）** | 驱动重新初始化、系统拦截硬件访问驱动，或插槽与延长线连接异常 | 在设备管理器中保持显卡启用。运行 `Setup_CMP30HX_WindowsAIO.bat -preflight` 检查显卡识别情况，安装程序报告驱动错误时查看诊断结果。 |
| **重启后恢复到 Gen1** | 启动任务失败、未注册，或驱动改变了显卡状态 | 在任务计划程序中检查 `CMP30HX_Batch_Gen2_Unlock`，并对比登录后的诊断结果。 |
| **显卡空闲时降到 Gen1 x16** | 电源管理可能在显卡空闲时降低链路速率 | 比较目标链路速率（TLS）和负载下实测的速率与宽度。 |
| **GPU-Z 显示 Gen2 x16 但 AIDA64 测速只有 ~2.5 GB/s** | 显卡的最大读取请求大小（MRRS）被限制在默认的 128 字节 | 运行命令 `40HXInstaller.exe -gen2-30hx`，将 MRRS 提升至 512 字节并刷新 DMA 队列。 |
| **无法识别显卡 / 报错代码 43** | 改焊 x16 通道电阻虚焊或接触不良，或者显卡未正确识别驱动 | 1. 重新检查显卡上的改电阻焊接点。<br>2. 若主板开启了 CSM 模式，运行 `Fix_CMP30HX_CSM_Error43.bat`。<br>3. 重新安装 NVIDIA 驱动（建议使用 DDU 彻底清除后重装最新版）。 |
| **尝试了所有方法仍无法解决** | NVIDIA 驱动配置冲突、注册表配置文件损坏或驱动服务异常 | 在 Windows 安全模式下使用 **DDU (Display Driver Uninstaller)** 清除旧驱动后重新安装驱动。 |
| **《英雄联盟》无法初始化图形设备或黑屏** | 无显示输出显卡的独占全屏模式冲突 | 参照第 3 节的优化功能，将 `game.cfg` 配置为无边框窗口和 DirectX 11 模式。 |
| **3D 游戏不调用 CMP 显卡，仅使用核显 (iGPU)** | Windows 尚未对 DirectX 应用建立全局图形重定向 | 运行 `UnlockRiotGame.exe` 点击 **[⚡ 1-CHẠM 优化]**。工具会自动将 CMP 硬件 ID 写入 `DirectXUserGlobalSettings` (`HighPerfAdapter`)，自动驱动所有 3D 游戏使用 CMP 渲染，无需手动逐个添加游戏。 |
| **Vanguard 或 Windows 拦截驱动或报告签名错误** | Windows 安全策略生效或驱动签名不受信任 | 确认任何更改前，先检查驱动签名和 Windows 错误详情。 |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. 卸载与清理

| 操作系统 | 卸载方式 |
|---|---|
| **Windows（CMP 30HX 批处理安装）** | 运行 `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows（CMP 40HX 通过 40HXInstaller 安装）** | 以管理员身份运行 `windows-v3.0\release\40HXUninstaller.exe` |
| **Linux（CMP 30HX）** | 运行 `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

Windows 批处理安装程序的 `-uninstall` 会清理其管理的任务、旧启动项及文件。仅需清理可识别的任务与启动项时使用 `-cleantasks`。Linux 卸载会清理 systemd 服务、唤醒钩子及已安装文件。CMP 40HX 安装程序的清理与恢复范围见[卸载指南](windows-v3.0/release/README.md#8-gỡ-cài-đặt-và-hoàn-nguyên)。

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> 致谢

本项目受 **CMP40HX-Unlock** 开源社区早期研究工作的启发。如果您觉得本工具有所帮助，请在 GitHub 仓库右上角点一个 **Star** 给予作者支持！
