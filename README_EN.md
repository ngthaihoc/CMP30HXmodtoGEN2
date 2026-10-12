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

- **1-Click Setup** — Interactive Web Control Center interface or fully automated All-In-One batch scripts.
- **Universal Driver Compatibility** — Works with official NVIDIA drivers, modded desktop drivers, and latest 5xx/6xx branch drivers.
- **Riot Games & Anti-Cheat Optimization** — Built-in DirectX Global Router, solves League of Legends display initialization crash, 100% compatible with Riot Vanguard, EAC, and BattlEye with Test Signing disabled.
- **No BIOS Modification** — Safe software-level unlock via BAR0 MMIO inside Windows/Linux; no VBIOS or motherboard BIOS flashing needed (for 30HX).
- **MRRS 512B Max Throughput** — Adjusts Max Read Request Size to 512 Bytes, achieving full ~6.4 GB/s DMA memory bandwidth.

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
| **Hardware eFuse Limit** | Factory eFuse blown for Gen3 (capped at Gen2 x16) | Supports Gen2 x16 / Gen3 x4 (configuration dependent) |
| **PCIe Unlock Mechanism** | OS runtime BAR0 MMIO (Windows / Linux) | EFI Loader (`40HXUNLK.EFI`) or runtime MMIO |
| **Tensor Core Compute** | None on TU116 architecture | Hardware supported (~50 TFLOPS via EFI unlock) |
| **Riot Vanguard / Secure Boot** | Direct in-OS unlock, keeps Secure Boot enabled | Sign EFI with `UnlockRiotGame.exe` into Secure Boot `db` |
| **DirectX Global Router** | Auto-routed to CMP discrete card | Auto-routed to CMP discrete card |

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. Preparation

**Hardware & System Requirements:**

- A **CMP 30HX / 40HX** card with physical PCIe x16 lane resistor mods, installed in a PCIe x16 slot directly wired to the CPU.
- Display monitor plugged into your integrated GPU (iGPU) or secondary output adapter.
- NVIDIA drivers installed (the card shows without exclamation marks in Device Manager).

**Download Toolset:**

Click **Code → Download ZIP** on GitHub or grab the latest release archive, then extract to a dedicated local directory (e.g., `D:\CMP30HX-Unlock`).

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. Installation on Windows

### ⭐ Fastest Method: Web Control Center (Recommended)

> **Just 1 step** — Double-click `Launch_WebUI.bat`.

Your browser will automatically open the local dashboard (`http://127.0.0.1:8080`) featuring real-time 16-lane PCIe matrix telemetry, GPU clocks, temperature monitoring, and a **1-Click Unlock Gen2** action button.

### Method 2: All-In-One Automated Script

> Right-click `Setup_CMP30HX_WindowsAIO.bat` → **Run as administrator**.

The script handles the full configuration process:
1. **Disables PCIe ASPM and Hybrid Sleep** — Prevents Windows from downclocking the PCIe link during idle states.
2. **Disables Fast Startup (hiberboot)** — Prevents links from getting stuck at Gen1 after restart.
3. **Disables Vulnerable Driver Blocklist** — Allows temporary kernel driver loading to write hardware registers.
4. **Disables Memory Integrity (HVCI / Core Isolation)** — Permits writing to BAR0 MMIO registers (if enabled, the script disables it and prompts for a single reboot).
5. **Registers Resident Guard Service** — Creates multi-tier Scheduled Tasks (Startup + Logon + Wake) and Registry Run Keys to automatically restore Gen2 speed if reset by the driver.
6. **Unlocks Gen2 x16 + MRRS 512B** — GPU is ready immediately, maximizing DMA throughput.

---

## <img src="https://api.iconify.design/lucide/gamepad-2.svg?color=%23ec4899" width="22" height="22" align="center" /> 3. Riot Games & 3D Render Optimization (UnlockRiotGame)

If you play **League of Legends (LoL), Teamfight Tactics (TFT), Valorant**, or other 3D games:

Run `windows-v3.0\release\UnlockRiotGame.exe` and click **[⚡ 1-CLICK Optimize]** (or run `UnlockRiotGame.exe -silent -optimize` via terminal).

The tool executes automatically within 2 seconds:
- **Dynamic Display & Render Detection**: Scans active display outputs to assign Display Host (`EnableMsHybrid=2`), while binding the CMP card as the 3D Render dGPU (`EnableMsHybrid=1`, `CASO=1`, `FeatureScore=0xD1`).
- **DirectX Global Router**: Registers CMP Hardware ID into `DirectXUserGlobalSettings` (`HighPerfAdapter`). All 3D titles automatically render on the CMP card without manually adding executable files in Windows Settings.
- **Resolves "Could not initialize graphics device" Error**: Scans LoL installation paths and configures `game.cfg` to borderless presentation and DirectX 11 (`WindowMode=2`, `BorderlessWindow=1`, `PreferDX9LegacyMode=0`, `DX11Support=1`).
- **Vanguard Unblocker for 5xx/6xx Drivers**: Disables VulnerableDriverBlocklist, automatically imports modified driver certificates into Windows Root Certificate Store, and cleans temporary drivers to remain strictly compatible with Vanguard (`testsigning` remains off).

---

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 4. Installation on Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

The Linux script automatically handles: disabling ASPM, scanning all CMP 30HX/40HX cards, writing BAR0 MMIO registers, setting MRRS to 512B, retraining the Gen2 link, and deploying systemd services plus sleep hooks to persist across reboots and suspend cycles.

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
| **AIDA64** → Tools → GPGPU Benchmark | **Memory Read / Memory Copy** rows | **6.3 – 6.4 GB/s** |

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
| **Stuck at Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | - Dual-GPU system (CMP + iGPU) or modded driver reset GPU TLS to Gen1 upon re-init.<br>- Windows has HVCI enabled, or riser cable has poor signal | 1. **Automated**: `Setup_CMP30HX_WindowsAIO.bat` features a **Resident Guard** mode to automatically restore Gen2 speed.<br>2. **Manual**: Keep the card **Enabled** in Device Manager (strictly **DO NOT Disable** the card as it cuts MMIO communication) → Re-run `40HXInstaller.exe -gen2-30hx`.<br>3. If HVCI is enabled: Turn off Memory Integrity and reboot. |
| **Gen2 speed lost after reboot** | - Driver loaded late on APU system or vBIOS reset link speed.<br>- Fast Startup is enabled in Windows | 1. The script deploys multi-tier Scheduled Tasks and Resident Guard to automatically detect and correct speed.<br>2. Turn off Fast Startup in Windows Power Options.<br>3. Recommended: Cleanly remove old drivers with **DDU (Display Driver Uninstaller)** in Safe Mode before reinstalling. |
| **Throttling to Gen1 x16 when idle** | Windows PCIe ASPM power saving is enabled | Re-run `Setup_CMP30HX_WindowsAIO.bat` (automatically disables ASPM) or configure Power Options → PCI Express → Link State Power Management to **Off**. |
| **GPU-Z shows Gen2 x16 but AIDA64 only gets ~2.5 GB/s** | Max Read Request Size (MRRS) is clamped at 128 Bytes | Execute `40HXInstaller.exe -gen2-30hx` to raise MRRS to 512 Bytes and flush DMA queues. |
| **GPU not detected / Code 43 error** | Resistor mod solder joint has poor contact, or missing driver | 1. Check resistor mod soldering.<br>2. If your motherboard uses CSM mode, run `Fix_CMP30HX_CSM_Error43.bat`.<br>3. Use DDU to clean install the latest driver. |
| **Crash initializing graphics when launching LoL** | Exclusive fullscreen display conflict on headless mining card | Run `UnlockRiotGame.exe` and click **1-CLICK Optimize** to enforce borderless presentation and DirectX 11 mode. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 8. Uninstallation

| Operating System | Method |
|---|---|
| **Windows** (script) | Run `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows** (GUI) | Right-click `windows-v3.0\release\40HXUninstaller.exe` → **Run as administrator** |
| **Linux** | Run `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

The uninstaller removes all Scheduled Tasks, Registry Run Keys, and application files, restoring system settings to default.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Acknowledgments

This project is inspired by the early research of the **CMP40HX-Unlock** community. If this tool helped you, please consider giving a **Star** on the GitHub repository to support the author!
