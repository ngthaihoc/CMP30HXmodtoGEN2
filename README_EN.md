<p align="right">
  <b>Language:</b>
  <a href="README.md">Tiếng Việt</a> |
  <b>English</b> |
  <a href="README_ZH.md">简体中文</a>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> NVIDIA CMP 30HX Unlock v3.0.0 (PCIe Gen2 x16)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-Not%20Required-success)](#)

A complete solution to unlock **PCIe Gen2 x16 (~6.4 GB/s)** bandwidth for **NVIDIA CMP 30HX (TU116 die)** graphics cards on Windows 10/11 x64 and Linux (Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux).

- **1-Click Automated Setup**: Automatically configures the system environment, disables PCIe ASPM power saving, configures Memory Integrity, and auto-activates on system boot.
- **Universal Driver Compatibility**: Compatible with all NVIDIA drivers (official, desktop, modded, latest releases, not restricted to version 537.58).
- **Clean & Native System State**: No BIOS/VBIOS flashing required, no EFI bootloader payload needed, and no Windows Test Signing required (safe for gaming and anti-cheat software).

> [!TIP]
> **Support the Author (Donate)**:  
> I developed this project while I was a student. If this tool helps you, please consider supporting me! Thank you very much! ❤️  
> 
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
> 
> - **Account Name**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub Repository**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. Preparation & Download

### 1.1 Hardware & Software Requirements
1. **Hardware**:
   - **CMP 30HX (TU116)** card with physical PCIe lane resistor modding to detect x16 mode.
   - Install the card into a PCIe x16 slot directly wired to the CPU (avoid low-quality riser cables or auxiliary slots routed through the chipset).
2. **Driver**:
   - Install any NVIDIA driver of your choice (Game Ready / Studio from official NVIDIA website, desktop INF-modded driver, or community modded drivers).
   - The card must appear properly in **Device Manager** (initially recognized as Gen1 x16 by default).

### 1.2 Downloading the Toolset
- Download the complete repository by clicking **Code $\rightarrow$ Download ZIP** on GitHub (or use `git clone`).
- Extract to a permanent directory on your drive (e.g., `C:\CMP30HX-Unlock` or `D:\CMP30HX-Unlock`).

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. Step-by-Step Installation Guide

### Method 1: 1-Click Automated Setup (Recommended for all users)

In the extracted folder, right-click **`Setup_CMP30HX_WindowsAIO.bat`** and select **Run as administrator** (or double-click; the script will automatically prompt for elevated admin privileges if needed).

The script will automatically execute a 6-step system optimization pipeline:
1. **Disable PCIe ASPM (Active State Power Management) & Hybrid Sleep**:
   - Prevents Windows from throttling PCIe link speed from Gen2 down to Gen1 when the GPU is idle.
2. **Disable Fast Startup (Hiberboot)**:
   - Prevents Windows from caching kernel sessions across shutdowns, eliminating the stuck-at-Gen1 issue after reboot.
3. **Disable Microsoft Vulnerable Driver Blocklist**:
   - Prevents Windows Defender and Code Integrity (CI) from blocking WinRing0 / ThrottleStop kernel drivers after reboot.
4. **Check & Disable Memory Integrity (Core Isolation / HVCI)**:
   - Disables Windows hypervisor kernel driver blocking in the Registry so the tool can override BAR0 MMIO registers.
5. **Deploy Persistent Files & Register Multi-Trigger Recovery (`CMP30HX_Gen2_Unlock`)**:
   - Automatically copies the toolchain to `%ProgramFiles%\40HXUnlock\` to avoid file loss after reboot.
   - Registers a Scheduled Task under `NT AUTHORITY\SYSTEM` with 3 triggers:
     + **AtStartup** (15-second delay after kernel load).
     + **AtLogOn** (5-second delay upon user login).
     + **Wake from Sleep** (Triggers when system wakes from Sleep / Modern Standby via Power-Troubleshooter Event ID 1).
   - Adds dual redundancy via Registry Run Keys (`HKLM` and `HKCU`), ensuring permanent Gen2 link speed after every reboot and sleep wake-up.
6. **Unlock Gen2 x16 & Enable MRRS 512B Immediately**:
   - Upgrades link bandwidth to Gen2 x16 and optimizes Max Read Request Size to 512B; the GPU is ready immediately without requiring a reboot.

> [!IMPORTANT]
> **NOTE ON SYSTEM REBOOT**:  
> - If your system previously had **Memory Integrity (Core Isolation)** enabled, the script will disable it and display a notification.  
> - In this case, you **MUST REBOOT YOUR COMPUTER** once for Windows to fully unload the hypervisor guard. After logging back into Windows, the system will automatically switch to Gen2 x16.

---

### Method 2: Manual Command Line Setup (For advanced users)

If you prefer manual control via Terminal / Command Prompt / PowerShell (Admin):

1. **Disable PCIe ASPM & Hybrid Sleep**:
   ```cmd
   powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
   powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
   powercfg -setacvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
   powercfg -setdcvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
   powercfg -setactive SCHEME_CURRENT
   ```

2. **Disable Fast Startup & Microsoft Vulnerable Driver Blocklist** (prevents getting stuck at Gen1 after reboot):
   ```cmd
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f
   ```

3. **Disable Memory Integrity (if enabled)**:
   ```cmd
   reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f
   ```
   *(Reboot your computer if this value was changed from 1 to 0).*

4. **Trigger Gen2 x16 Unlock Immediately**:
   ```cmd
   cd /d "Path_to_extracted_folder"
   .\windows-v3.0\release\40HXInstaller.exe -gen2-30hx
   ```

5. **Configure Persistent Gen2 across Reboot & Wake from Sleep**:
   - **Deploy binaries to system directory**:
     ```cmd
     if not exist "%ProgramFiles%\40HXUnlock" mkdir "%ProgramFiles%\40HXUnlock"
     copy /y ".\windows-v3.0\release\40HXInstaller.exe" "%ProgramFiles%\40HXUnlock\"
     copy /y ".\windows-v3.0\release\40HXCheck.exe" "%ProgramFiles%\40HXUnlock\"
     ```
   - **Register SYSTEM Scheduled Task with Multi-triggers (Startup + Logon + Wake) via PowerShell**:
     ```powershell
     $action = New-ScheduledTaskAction -Execute "$env:ProgramFiles\40HXUnlock\40HXInstaller.exe" -Argument '-gen2-30hx -silent' -WorkingDirectory "$env:ProgramFiles\40HXUnlock"
     $t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'
     $t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'
     $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew
     $principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest
     Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force
     ```
   - **Add dual insurance via Registry Run Keys (HKLM & HKCU)**:
     ```cmd
     reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
     reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /t REG_SZ /d "\"%ProgramFiles%\40HXUnlock\40HXInstaller.exe\" -gen2-30hx -silent" /f
     ```

---

### Method 3: Automated Installation on Linux (Ubuntu, Debian, HiveOS, RaveOS, Fedora, Arch Linux)

For Linux workstations, mining rigs, or AI servers. The script **`Setup_CMP30HX_LinuxAIO.sh`** is a standalone All-In-One tool that runs directly without external dependencies:

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

**Automated Features of `Setup_CMP30HX_LinuxAIO.sh`:**
1. **Disable PCIe ASPM & Runtime Power Management**: Configures kernel power policy to `performance` and sets `power/control=on` for all PCI devices, preventing Gen1 downclocking at idle.
2. **Scan & Process All Installed Cards**: Automatically detects and applies settings to all CMP 30HX (`10de:2189`) and CMP 40HX (`10de:1f0b`) cards present on the system.
3. **BAR0 MMIO Direct Injection**: Overrides kernel registers (`XVE_OVR`, `PRIV_MISC_1`, `LINK_CONFIG_0`, `LNKCAP`, `LNKCTL2`).
4. **MRRS 512B Optimization & Link Retrain**: Sets Target Link Speed = Gen2, increases Max Read Request Size to 512 Bytes (DEVCTL), and triggers link retraining to hit the ~6.4 GB/s bandwidth ceiling.
5. **Install Systemd Service & Sleep Hook**: Automatically enables `cmp30hx-gen2-unlock.service` and sleep hook `/lib/systemd/system-sleep/cmp30hx-unlock` to maintain Gen2 speed across reboots and suspend/resume cycles.

**Linux Utility Commands:**
- Check current link status & MRRS:
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --status
  ```
- Completely uninstall systemd service and hooks:
  ```bash
  sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
  ```

---

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 3. Verification & Bandwidth Testing

After activation (or after logging back into Windows), verify using these two methods:

1. **Verify PCIe Link Speed**:
   - Run **`40HXCheck.exe`** (or open **GPU-Z**):
   - Under **Bus Interface / PCIe**: Verify it reports **`PCIe x16 2.0 @ x16 2.0`** (or `Gen2 x16`).
2. **Verify Real-World Transfer Bandwidth (Confirm MRRS 512B)**:
   - Open **AIDA64** $\rightarrow$ Navigate to **Tools** menu $\rightarrow$ Select **GPGPU Benchmark**.
   - Select the CMP 30HX card and click **Run Benchmarks**:
   - Check the **Memory Read** and **Memory Copy** values:
     - **Success**: Achieves approximately **6.3 – 6.4 GB/s** (reaching the theoretical limit of Gen2 x16).
     - **Suboptimal**: If it only reaches ~2.5 GB/s, MRRS is still at the default 128B (re-run `40HXInstaller.exe -gen2-30hx` to activate 512B optimization).

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 4. Troubleshooting

| Symptom | Possible Cause | Resolution |
|---|---|---|
| **Stuck at Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | - Dual-GPU system (CMP 30HX + Intel UHD iGPU) or modded driver (RainCandy) holding DMA/hook context, preventing PCIe speed renegotiation.<br>- Or Windows Memory Integrity (HVCI) is enabled, or poor PCIe riser contact | 1. **Automated**: The latest `Setup_CMP30HX_WindowsAIO.bat` includes an automated **Soft Reset (Disable/Enable via PnP)** cycle to release driver locks and retrain Gen2.<br>2. **Manual**: Open **Device Manager** $\rightarrow$ Right-click **NVIDIA CMP 30HX** $\rightarrow$ Select **Disable device** $\rightarrow$ Immediately select **Enable device** $\rightarrow$ Re-run `Setup_CMP30HX_WindowsAIO.bat`.<br>3. If HVCI is enabled: Disable HVCI and **Reboot your computer**. |
| **Gen2 activation lost after reboot** | - NVIDIA / RainCandy driver loading late or overwriting vBIOS state on startup.<br>- Leftover configuration conflict from old drivers, or Fast Startup is still enabled | 1. `Setup_CMP30HX_WindowsAIO.bat` includes a SYSTEM Scheduled Task (15s delay) with polling logic to automatically detect and retrain.<br>2. **DDU Recommended**: Use **DDU (Display Driver Uninstaller)** in Safe Mode to cleanly remove all prior display drivers before reinstalling modded drivers to avoid registry conflicts. |
| **Throttling to Gen1 x16 when idle** | Windows PCIe ASPM power saving is enabled | Re-run `Setup_CMP30HX_WindowsAIO.bat` (automatically disables ASPM) or configure in Windows Power Options $\rightarrow$ PCI Express $\rightarrow$ Link State Power Management: **Off**. |
| **GPU-Z reports Gen2 x16 but AIDA64 only achieves ~2.5 GB/s** | Card Max Read Request Size (MRRS) is clamped at default 128 Bytes | Run `40HXInstaller.exe -gen2-30hx` to increase MRRS to 512 Bytes and flush DMA queues. |
| **GPU not detected / Code 43 error** | Resistor mod solder joint not making proper contact, or driver not installed properly | 1. Inspect the physical PCIe lane mod solder joints on the card.<br>2. Reinstall the NVIDIA driver (use DDU to clean install the latest driver). |
| **All fixes attempted but still failing** | NVIDIA driver registry corruption, conflicting driver profile, or faulty service | Cleanly remove old drivers using **DDU (Display Driver Uninstaller)** in Safe Mode, then perform a fresh driver installation. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 5. Uninstallation

To restore your system to its original state:
- **Fast Method**: Run `Setup_CMP30HX_WindowsAIO.bat -uninstall`.
- **GUI Method**: Right-click **`40HXUninstaller.exe`** $\rightarrow$ select **Run as administrator**.

The uninstaller cleanly removes:
- Scheduled Task `CMP30HX_Gen2_Unlock` from Windows.
- Background Registry Run Keys (`HKLM` and `HKCU`).
- Application folder `%ProgramFiles%\40HXUnlock`.
- Driver runtime and temporary state files in `%ProgramData%\40HXUnlock`.

---

## <img src="https://api.iconify.design/lucide/info.svg?color=%238b5cf6" width="22" height="22" align="center" /> 6. Technical Notes

> [!NOTE]
> - **Why is Gen2 x16 the ceiling, and why not Gen3?**  
>   On the CMP 30HX TU116 die, NVIDIA physically blew the **Silicon eFuse (Bit 3 - 8.0 GT/s)** at the factory. Therefore, Gen2 x16 (5.0 GT/s) is the absolute hardware physical limit. This tool unlocks this maximum ceiling 100% safely via BAR0 MMIO; it strictly avoids forcing Gen3 to prevent link training lockups.
> - **MRRS 512B**:  
>   Increasing Max Read Request Size from 128B to 512B eliminates TLP packet fragmentation bottlenecks, unlocking the full ~6.4 GB/s DMA memory bandwidth.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Acknowledgments

> Inspired by the foundational work of the **CMP40HX-Unlock** project. If you find this tool helpful, please leave a Star on the repository to support the author!
