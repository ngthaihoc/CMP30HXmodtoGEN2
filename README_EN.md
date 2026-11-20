<p align="right">
  <b>Language:</b>
  <a href="README.md">Tiếng Việt</a> |
  <b>English</b> |
  <a href="README_ZH.md">简体中文</a>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Unlock NVIDIA CMP 30HX & 40HX v3.0.0 (PCIe Gen2 x16 & Riot Vanguard)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116%20%26%20TU106-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-Disabled%20(Safe)-success)](#)
[![Web Control Center](https://img.shields.io/badge/Web%20UI-Localhost-blue?logo=html5&logoColor=white)](#-fastest-method-web-control-center-recommended)

Bandwidth unlock solution for **PCIe Gen2 x16 (~6.4 GB/s)** and dedicated 3D graphics rendering optimization for **NVIDIA CMP 30HX (TU116 die)** and **CMP 40HX (TU106 die)** mining GPUs on Windows 10/11 x64 and Linux.

- **1-Click Setup**: Interactive Web Control Center interface or fully automated All-In-One batch scripts.
- **Universal Driver Compatibility**: Works with official NVIDIA drivers, modded desktop drivers, and latest 5xx/6xx branch drivers.
- **Riot Games & Anti-Cheat Optimization**: Built-in DirectX Global Router, solves League of Legends display initialization crash, 100% compatible with Riot Vanguard, EAC, and BattlEye with Test Signing disabled.
- **No BIOS Modification**: Safe software-level unlock via BAR0 MMIO inside Windows/Linux; no VBIOS or motherboard BIOS flashing needed (for 30HX).
- **MRRS 512B Max Throughput**: Adjusts Max Read Request Size to 512 Bytes, achieving full ~6.4 GB/s DMA memory bandwidth.

> [!TIP]
> **Support the Author (Donate)**
>
> I developed this project while studying at university and shared it completely free for the community. If this tool is helpful to you, please consider making a small donation to keep development going! Thank you very much! ❤️
>
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
>
> - **Account Name**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub Repository**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/cpu.svg?color=%2310b981" width="22" height="22" align="center" /> Hardware Support Comparison

| Feature | CMP 30HX (TU116) | CMP 40HX (TU106) |
|---|---|---|
| **PCIe target in this project** | Gen2 x16 | Gen2 x16 |
| **PCIe Unlock Mechanism** | OS runtime BAR0 MMIO (Windows / Linux) | EFI Loader (`40HXUNLK.EFI`) or runtime MMIO |
| **Tensor Core Compute** | None on TU116 architecture | Hardware supported (~50 TFLOPS via EFI unlock) |
| **Riot Vanguard / Secure Boot** | Direct in-OS unlock, keeps Secure Boot enabled | Sign EFI with `UnlockRiotGame.exe` into Secure Boot `db` |
| **DirectX Global Router** | Auto-routed to CMP discrete card | Auto-routed to CMP discrete card |

On the TU116 sample tested, the target speed still read Gen2 after a request for Gen3. The underlying hardware cause remains undetermined.

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. Preparation

**Hardware & System Requirements:**

- A **CMP 30HX / 40HX** card with physical PCIe x16 lane resistor mods, installed in a PCIe x16 slot directly wired to the CPU.
- Display monitor plugged into your integrated GPU (iGPU) or secondary output adapter.
- NVIDIA drivers installed (the card shows without exclamation marks in Device Manager).

**Download Toolset:**

Click **Code -> Download ZIP** on GitHub or grab the latest release archive, then extract to a dedicated local directory (e.g., `D:\CMP30HX-Unlock`).

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. Installation on Windows

### ⭐ CMP Control Center

> **Just 1 step**: Double-click `Launch_WebUI.bat`.

The launcher opens the `UnlockAIO.exe` Wails desktop app when present. If that file is missing, the launcher starts the local web interface from `40HXInstaller.exe` and opens the address shown by the tool in your browser.

### Method 2: All-In-One Automated Script

> Right-click `Setup_CMP30HX_WindowsAIO.bat` -> **Run as administrator**.

This script is for CMP 30HX (`10DE:2189`).
1. Checks the GPU, installer files, and required drivers.
2. Copies the runtime files, attempts to configure PCIe Gen2, and reads the result. Verify Gen2 x16 under load.
3. After a successful unlock, registers a Windows startup and logon task. Setup reports a failure if the driver is blocked or the task cannot be registered.

---

## <img src="https://api.iconify.design/lucide/gamepad-2.svg?color=%23ec4899" width="22" height="22" align="center" /> 3. Riot Games & 3D Render Optimization (UnlockRiotGame)

If you play **League of Legends (LoL), Teamfight Tactics (TFT), Valorant**, or other 3D games:

The **[⚡ 1-CLICK Optimize]** action in `windows-v3.0\release\UnlockRiotGame.exe` can change Windows security settings. The `-silent -optimize` command requires explicit confirmation through `-confirm-riot` or `-y`.

The optimization includes:
- **Dynamic Display & Render Detection**: Scans active display outputs to assign Display Host (`EnableMsHybrid=2`), while binding the CMP card as the 3D Render dGPU (`EnableMsHybrid=1`, `CASO=1`, `FeatureScore=0xD1`).
- **DirectX Global Router**: Registers CMP Hardware ID into `DirectXUserGlobalSettings` (`HighPerfAdapter`). All 3D titles automatically render on the CMP card without manually adding executable files in Windows Settings.
- **Resolves "Could not initialize graphics device" Error**: Scans LoL installation paths and configures `game.cfg` to borderless presentation and DirectX 11 (`WindowMode=2`, `BorderlessWindow=1`, `PreferDX9LegacyMode=0`, `DX11Support=1`).
- **Security settings**: Depending on the system configuration, optimization can change driver policies or the Windows certificate store. Vanguard compatibility depends on the tested driver and environment.

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 4. Installation on Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

The Linux script targets CMP 30HX (`10DE:2189`). It configures PCIe with a Gen2 x16 target and an MRRS of at least 512 bytes. It can also install a systemd service and a wake hook. Verify link speed under load after reboot and wake.

| Command | Purpose |
|---|---|
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --status` | Check current PCIe link status and MRRS |
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` | Remove systemd persistence and sleep hooks |
| `./Setup_CMP30HX_LinuxAIO.sh -test --no-root` | Run mock simulation without requiring physical GPUs |

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 5. Verifying Results

After running the unlock, verify performance using either tool:

| Utility | Check Location | Expected Result |
|---|---|---|
| **GPU-Z** or `40HXCheck.exe` | **Bus Interface** row | `PCIe x16 2.0 @ x16 2.0` |
| **AIDA64** -> Tools -> GPGPU Benchmark | **Memory Read / Memory Copy** rows | **6.3 - 6.4 GB/s** |

> [!NOTE]
> If GPU-Z shows Gen2 x16 but AIDA64 only achieves ~2.5 GB/s, MRRS is still stuck at the default 128 Bytes limit. Run `windows-v3.0\release\40HXInstaller.exe -gen2-30hx` to reactivate MRRS 512B.

---

## <img src="https://api.iconify.design/lucide/wrench.svg?color=%23f59e0b" width="22" height="22" align="center" /> 6. Companion Utilities

The repository provides companion tools for common environment configurations:
- **`Fix_CMP30HX_CSM_Error43.bat`**: Resolves Code 43 and motherboard resource conflict issues when CSM legacy mode is enabled.
- **`Setup_ReBar_CMP30HX.bat`**: Configures Resizable BAR support on CMP cards to eliminate VRAM bus bottlenecks and improve gaming frame rates.

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 7. Troubleshooting

| Symptom | Cause | Solution |
|---|---|---|
| **Stuck at Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | Driver reinitialization, a blocked hardware access driver, or a slot or riser issue | Keep the card enabled in Device Manager. Run `Setup_CMP30HX_WindowsAIO.bat -preflight` to check GPU detection and review the diagnostic output if setup reports a driver error. |
| **Gen2 activation lost after reboot** | Startup task failed or is missing, or the driver changed the GPU state | Check `CMP30HX_Batch_Gen2_Unlock` in Task Scheduler and compare the diagnostics after logging in. |
| **Link drops to Gen1 x16 when idle** | Power management can reduce the link speed while the GPU is idle | Compare the target link speed (TLS) with the current speed and width under load. |
| **GPU-Z reports Gen2 x16 but AIDA64 only achieves ~2.5 GB/s** | Card Max Read Request Size (MRRS) is clamped at default 128 Bytes | Run command `40HXInstaller.exe -gen2-30hx` to increase MRRS to 512 Bytes and flush DMA queues. |
| **GPU not detected / Code 43 error** | Resistor mod solder joint for x16 lanes has poor contact, or driver not installed properly | 1. Inspect the resistor mod solder joints on the card.<br>2. If your motherboard uses CSM mode, run `Fix_CMP30HX_CSM_Error43.bat`.<br>3. Reinstall the NVIDIA driver (use DDU to clean install the latest driver). |
| **All fixes attempted but still failing** | NVIDIA driver registry conflicts, overwritten profiles, or faulty driver services | Cleanly remove old drivers using **DDU (Display Driver Uninstaller)** in Safe Mode, then perform a fresh driver installation. |
| **Crash initializing graphics / black screen in LoL** | Exclusive fullscreen conflict on a card without display output | Use the optimization in section 3 to set borderless mode and DirectX 11 in `game.cfg`. |
| **3D games do not utilize CMP card, only running on iGPU** | Windows has not configured global graphics routing for DirectX applications | Run `UnlockRiotGame.exe` and click **[⚡ 1-CLICK Optimize]**. The tool binds CMP Hardware ID to `DirectXUserGlobalSettings` (`HighPerfAdapter`), forcing all 3D games to render on CMP without adding each game manually. |
| **Vanguard or Windows blocks a driver / driver signature error** | Windows security policy or an untrusted driver signature | Check the driver signature and the Windows error details before confirming any changes. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. Uninstallation

| Operating System | Method |
|---|---|
| **Windows (CMP 30HX batch setup)** | Run `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows (CMP 40HX via 40HXInstaller)** | Run `windows-v3.0\release\40HXUninstaller.exe` as administrator |
| **Linux (CMP 30HX)** | Run `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

The Windows batch `-uninstall` cleans up tasks, legacy startup entries, and files owned by the installer. Use `-cleantasks` to clean only the tasks and startup entries the installer recognizes. Linux removal cleans up the systemd service, wake hook, and installed files. For the 40HX installer, see [what its uninstaller removes and restores](windows-v3.0/release/README.md#8-gỡ-cài-đặt-và-hoàn-nguyên).

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Acknowledgments

This project is inspired by the early research of the **CMP40HX-Unlock** community. If this tool helped you, please consider giving a **Star** on the GitHub repository to support the author!
