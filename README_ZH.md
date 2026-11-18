<p align="right">
  <b>语言：</b>
  <a href="README.md">Tiếng Việt</a> |
  <a href="README_EN.md">English</a> |
  <b>简体中文</b>
</p>

# NVIDIA CMP 30HX / 40HX PCIe 解锁

用于研究 NVIDIA CMP 30HX（TU116，PCI ID `10DE:2189`) 和 CMP 40HX（TU106，PCI ID `10DE:1F0B`) PCIe Gen2 链路的工具。Windows 代码支持这两个设备 ID。实际结果取决于显卡、主板、插槽、转接线、固件和驱动。Gen2 x16 是技术目标，不保证每套系统都能达到。

[![GitHub Releases](https://img.shields.io/badge/GitHub-Releases-181717?logo=github)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2/releases)

## 选择工具

- **Windows CMP 30HX：**以管理员身份从项目目录运行 `Setup_CMP30HX_WindowsAIO.bat`。
- **Windows CMP 40HX：**阅读 [Windows 指南](windows-v3.0/README.md) 和发布包内附带的指南。CMP 40HX EFI 文件仅适用于 CMP 40HX。
- **Linux：**使用 `Setup_CMP30HX_LinuxAIO.sh` 安装、检查状态或移除服务。运行前请阅读脚本帮助。
- **Resizable BAR / CSM：**`Setup_ReBar_CMP30HX.bat` 和 `Fix_CMP30HX_CSM_Error43.bat` 是独立工具，可能更改系统或设备配置。确认前请阅读其提示和文档。

## Windows CMP 30HX

下载并完整解压项目 ZIP 或发布包。卸载完成前请保留解压目录。在该目录中以管理员权限打开终端或命令提示符。

```cmd
Setup_CMP30HX_WindowsAIO.bat
Setup_CMP30HX_WindowsAIO.bat -preflight
Setup_CMP30HX_WindowsAIO.bat -notask
Setup_CMP30HX_WindowsAIO.bat -cleantasks
Setup_CMP30HX_WindowsAIO.bat -uninstall
```

交互菜单有五个选项：[1] 安装并解锁，[2] 卸载，[3] 修复 NVIDIA Control Panel，[4] 可选的 Riot/Unity 配置，[5] 退出。[3] 和 [4] 是独立操作，可能更改 NVIDIA 或游戏设置。

- `-preflight` 只检查硬件。必须恰好有 **一张**已连接且 PCI ID 为 CMP 30HX `10DE:2189` 的显卡。没有匹配设备或匹配多张时脚本会拒绝继续。
- 解锁成功后，脚本注册 SYSTEM 计划任务 `CMP30HX_Batch_Gen2_Unlock`，在开机 45 秒后或登录 10 秒后运行。安装文件位于 `%ProgramFiles%\40HXUnlock`，批处理状态文件位于 `%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt`。
- 新安装时使用 `-notask` 可跳过持久化任务注册，解锁仅作用于当前会话。
- `-cleantasks` 仅在批处理任务或已知旧任务运行 `%ProgramFiles%\40HXUnlock` 中的 `RunUnlock.bat` 或 `UnlockAIO.exe` 时移除它们，同时清理指向这些文件的 Run 注册表值。其他任务和工具文件会保留。
- `-uninstall` 移除安装器的持久化设置和自有文件，同时保留共享驱动、服务及配置。卸载后重启，以清除当前会话的运行时解锁状态。

验证安装时，检查安装结果，并在 GPU 负载下检查 PCIe 状态。随后重启并再次检查。使用以下命令查看计划任务：

```powershell
Get-ScheduledTask -TaskName 'CMP30HX_Batch_Gen2_Unlock'
```

在菜单中选择 [2] 或运行 `-cleantasks` 来移除持久化任务。使用以下命令确认任务已移除：

```cmd
schtasks /query /tn "CMP30HX_Batch_Gen2_Unlock"
```

移除成功后，Windows 会提示找不到该任务。重启并确认链路不会自动解锁。完成卸载前请保留 ZIP 文件或解压目录。

## Control Center 与其他工具

如果发布包包含 CMP Control Center，`Launch_WebUI.bat` 会启动它。AIO 界面包含 Gen2 解锁、诊断、Riot & 3D、辅助工具和卸载标签页。诊断信息包括 PCIe、MRRS、GSP、时钟/温度和链路矩阵。Riot/Unity 配置与 NVIDIA Control Panel 修复是可选操作，不会随默认解锁流程自动执行。

## Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh --status
sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
./Setup_CMP30HX_LinuxAIO.sh -test --no-root
```

Linux 卸载会移除脚本安装的服务、睡眠恢复钩子和工具文件，但不会撤销手动更改或所有硬件配置状态。如需刷新 PCIe 运行时状态，请重启。

## 验证、恢复与限制

请在 GPU 负载下使用 GPU-Z 或诊断工具检查。GPU 空闲时可能因电源管理而显示 Gen1。基准结果会因平台和工作负载而异，不保证单一带宽数值。

模拟测试验证软件流程，不能证明真实 CMP 硬件上的解锁、睡眠/重启稳定性或反作弊兼容性。本项目不保证兼容 Vanguard、EAC 或 BattlEye。日常使用前请在目标硬件上验证。

如果使用 EFI 流程后 CMP 40HX 系统无法启动，请仅在遇到指南中列出的启动故障时查阅 [EFI 救援指南](windows-v3.0/release/EFI应急修复指南.md)。系统正常启动时不要修改 EFI 分区。

## 下载、支持与致谢

从 [GitHub Releases](https://github.com/ngthaihoc/CMP30HXmodtoGEN2/releases) 下载软件包。请使用对应软件包中附带的说明，因为不同版本的文件可能不同。

本项目免费分享。如果工具对你有帮助，可以通过 [MoMo/VietQR 二维码图片](assets/donate_momo.jpg) 支持作者。本项目受到 CMP40HX-Unlock 社区早期研究工作的启发。

请遵守当地法律、制造商条款以及所用软件的规则。
