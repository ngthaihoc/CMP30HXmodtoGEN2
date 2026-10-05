<p align="right">
  <b>Language:</b>
  <a href="README.md">Tiếng Việt</a> |
  <b>English</b> |
  <a href="README_ZH.md">简体中文</a>
</p>

# <img src="https://api.iconify.design/carbon/chip.svg?color=%2310b981" width="32" height="32" align="center" /> Unlock NVIDIA CMP 30HX v3.0.0 (PCIe Gen2 x16)

[![GitHub Repo](https://img.shields.io/badge/GitHub-ngthaihoc%2FCMP30HXmodtoGEN2-181717?logo=github&logoColor=white)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-0078D6?logo=linux&logoColor=white)](https://kernel.org)
[![GPU](https://img.shields.io/badge/NVIDIA-TU116-76B900?logo=nvidia&logoColor=white)](https://nvidia.com)
[![PCIe](https://img.shields.io/badge/PCIe-Gen2%20x16%20(~6.4%20GB%2Fs)-orange)](https://pcisig.com)
[![Test Signing](https://img.shields.io/badge/Test%20Signing-Not%20Required-success)](#)

Bandwidth unlock solution for **PCIe Gen2 x16 (~6.4 GB/s)** on **NVIDIA CMP 30HX (TU116 die)** graphics cards on Windows 10/11 x64 and Linux.

- **1-Click Installation** — the script handles everything automatically, just double-click and wait.
- **Universal Driver Compatibility** — works with official NVIDIA drivers, desktop drivers, modded drivers, and latest releases.
- **Anti-Cheat Safe** — no Test Signing required; compatible with Riot Vanguard, EAC, and BattlEye.
- **No BIOS Modification** — software-level interception; no BIOS/VBIOS flashing required.

> [!TIP]
> **Support the Author (Donate)**
>
> I developed this project while I was a student. If possible, please consider donating a little to support me! Thank you all very much! ❤️
>
> <p align="center">
>   <img src="assets/donate_momo.jpg" alt="Donate MoMo VietQR - NGUYEN THAI HOC" width="220" style="border-radius: 12px;" />
> </p>
>
> - **Account Name**: NGUYEN THAI HOC (MoMo / VietQR)
> - **GitHub Repository**: [https://github.com/ngthaihoc/CMP30HXmodtoGEN2](https://github.com/ngthaihoc/CMP30HXmodtoGEN2)

---

## <img src="https://api.iconify.design/lucide/download.svg?color=%230284c7" width="22" height="22" align="center" /> 1. Preparation

**Requirements:**

- A **CMP 30HX (TU116)** card with physical PCIe x16 lane resistor modding, installed in a PCIe x16 slot directly wired to the CPU.
- Any NVIDIA driver already installed (the card displays normally in Device Manager).

**Download the toolset:**

Click **Code → Download ZIP** on GitHub, then extract to a fixed directory (e.g., `D:\CMP30HX-Unlock`).

---

## <img src="https://api.iconify.design/lucide/terminal.svg?color=%2310b981" width="22" height="22" align="center" /> 2. Installation on Windows

### ⭐ Fastest Method: Web UI (Recommended)

> **Just 1 step** — double-click `Launch_WebUI.bat` and follow the interface.

Your browser will open the control dashboard with complete GPU information, PCIe link status, and a 1-click button to unlock Gen2.

### Method 2: 1-Click Automated Script

> **Just 1 step** — right-click `Setup_CMP30HX_WindowsAIO.bat` → **Run as administrator**.

The script will automatically handle everything: disable ASPM, disable Fast Startup, configure the registry, unlock Gen2 x16, increase MRRS to 512B, and register auto-activation on every boot.

> [!IMPORTANT]
> If your system currently has **Memory Integrity (Core Isolation / HVCI) enabled**, the script will disable it and prompt you to **reboot once**. After logging back in, the system will automatically transition to Gen2 x16.

<details>
<summary>📋 Details: What does the script do?</summary>

1. **Disable PCIe ASPM and Hybrid Sleep** — prevents Windows from throttling PCIe link speed when the GPU is idle.
2. **Disable Fast Startup (hiberboot)** — prevents getting stuck at Gen1 link speed after reboot.
3. **Disable Microsoft Vulnerable Driver Blocklist** — allows loading the WinRing0 / ThrottleStop kernel drivers.
4. **Disable Memory Integrity (HVCI)** — permits writing to BAR0 MMIO registers.
5. **Register Auto-Activation** — creates Scheduled Tasks (startup + logon + wake from sleep) and Registry Run Keys, ensuring Gen2 is persistently maintained.
6. **Unlock Gen2 x16 + MRRS 512B** — GPU is ready immediately, no reboot required (except when disabling HVCI).

</details>

<details>
<summary>🔧 Method 3: Manual Command Line Installation (For advanced users)</summary>

Open **PowerShell** or **Command Prompt** as administrator:

**Step 1 — Disable PCIe ASPM and Hybrid Sleep:**
```cmd
powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0
powercfg -setacvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
powercfg -setdcvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0
powercfg -setactive SCHEME_CURRENT
```

**Step 2 — Disable Fast Startup and Vulnerable Driver Blocklist:**
```cmd
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f
```

**Step 3 — Disable Memory Integrity (if enabled):**
```cmd
reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f
```
*(Reboot if this value was changed from 1 to 0.)*

**Step 4 — Unlock Gen2 x16:**
```cmd
cd /d "Path_to_extracted_folder"
.\windows-v3.0\release\40HXInstaller.exe -gen2-30hx
```

**Step 5 — Register Auto-Activation on boot:**
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

## <img src="https://api.iconify.design/lucide/monitor.svg?color=%2310b981" width="22" height="22" align="center" /> 3. Installation on Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
```

The script automatically handles: disabling ASPM, scanning all CMP 30HX/40HX cards, writing BAR0 MMIO registers, setting MRRS to 512B, retraining to Gen2 link speed, and installing a systemd service to persist across reboots.

| Command | Function |
|---|---|
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --status` | Check link status and MRRS |
| `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` | Uninstall service and sleep hooks |
| `./Setup_CMP30HX_LinuxAIO.sh -test --no-root` | Run mock test without requiring a physical GPU |

<details>
<summary>📋 Linux Script Technical Details</summary>

1. **Disable PCIe ASPM and Runtime Power Management** — sets kernel policy to `performance` and `power/control=on` for all PCI devices.
2. **Scan All Cards** — auto-detects CMP 30HX (`10de:2189`) and CMP 40HX (`10de:1f0b`).
3. **BAR0 MMIO Direct Injection** — writes registers `XVE_OVR`, `PRIV_MISC_1`, `LINK_CONFIG_0`, `LNKCAP`, `LNKCTL2`.
4. **MRRS 512B + Link Retrain** — Target Link Speed = Gen2, MRRS 512 Bytes, retrains link to reach ~6.4 GB/s.
5. **Systemd Service + Sleep Hook** — `cmp30hx-gen2-unlock.service` and `/lib/systemd/system-sleep/cmp30hx-unlock`.
6. **StatusContract** — outputs `gen2_status.txt` (or via `--status-file` flag):
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

## <img src="https://api.iconify.design/lucide/check-circle.svg?color=%2306b6d4" width="22" height="22" align="center" /> 4. Verifying Results

After installation is complete, verify using either of these methods:

| Tool | How to Check | Correct Result |
|---|---|---|
| **GPU-Z** or `40HXCheck.exe` | **Bus Interface** field | `PCIe x16 2.0 @ x16 2.0` |
| **AIDA64** → Tools → GPGPU Benchmark | **Memory Read / Memory Copy** row | **6.3 – 6.4 GB/s** |

> If AIDA64 only achieves ~2.5 GB/s → MRRS is still at the default 128B → re-run `40HXInstaller.exe -gen2-30hx`.

---

## <img src="https://api.iconify.design/lucide/shield-check.svg?color=%2306b6d4" width="22" height="22" align="center" /> 5. Anti-Cheat Compatibility (Riot Vanguard, EAC, BattlEye)

You can comfortably play **Valorant, League of Legends, Apex Legends, and Fortnite** after unlocking:

- **CMP 30HX** — unlocks Gen2 via BAR0 MMIO inside Windows (no EFI loading required), keeping Secure Boot enabled → 100% compatible with Riot Vanguard.
- **CMP 40HX** — requires running `UnlockRiotGame.exe` to digitally sign EFI firmware and enroll the certificate into the Secure Boot db.
- **Transient BYOVD Model** — kernel drivers load for only a few milliseconds and then cleanly unload and delete themselves. When games run, the kernel is completely clean.
- **No `bcdedit /set testsigning on` required** — this command is 100% blocked by Vanguard and is not required by this tool.

<details>
<summary>📋 Details: Riot Vanguard Solution for CMP 40HX (TU106)</summary>

Riot Vanguard on Windows 11 requires Secure Boot = Enabled and TPM 2.0. Because `40HXUNLK.EFI` firmware is not signed by Microsoft, `UnlockRiotGame.exe` will automatically:

1. Generate a self-signed X.509 certificate (`CMP40HX_Key.cer`) with SHA256.
2. Digitally sign `40HXUNLK.EFI` via Authenticode (in both release directory and ESP partition).
3. Export the certificate to `C:\`, Desktop, and ESP (`\EFI\40HX\`).
4. Guide you through rebooting into BIOS → **Custom Mode** → enrolling `CMP40HX_Key.cer` into **db** (Key Management → Authorized Signatures → Append Key).
5. Result: Secure Boot remains enabled (satisfying Vanguard), while the EFI firmware unlocks Tensor Cores (~50 TFLOPS) and PCIe Gen2.

</details>

---

## <img src="https://api.iconify.design/lucide/help-circle.svg?color=%23f43f5e" width="22" height="22" align="center" /> 6. Troubleshooting

| Symptom | Cause | Resolution |
|---|---|---|
| **Stuck at Gen1 (GPU TLS=Gen1, Root TLS=Gen2)** | - Dual-GPU system (CMP 30HX + AMD/Intel iGPU) or modded driver (RainCandy) re-initialization resetting GPU TLS to Gen1.<br>- Or Windows has Memory Integrity (HVCI) enabled, or poor PCIe riser cable contact | 1. **Automated**: The new `Setup_CMP30HX_WindowsAIO.bat` activates **Resident Guard** mode and a Logon task (10s delay) to automatically compensate Gen2 speed.<br>2. **Manual**: Keep the card **Enabled** in Device Manager (strictly **DO NOT** Disable as it disconnects BAR0 MMIO) → Run `Setup_CMP30HX_WindowsAIO.bat` or `40HXInstaller.exe -gen2-30hx` once the desktop has stabilized.<br>3. If HVCI is enabled: Disable HVCI and **Reboot the computer**. |
| **Gen2 activation lost after reboot** | - NVIDIA / RainCandy driver loading late on APU systems or overwriting vBIOS state after boot.<br>- Old driver configuration conflicts or Fast Startup still enabled | 1. `Setup_CMP30HX_WindowsAIO.bat` includes multi-tier Scheduled Tasks (Logon delay 10s + Startup delay 45s) with **DriverStrategy=2 (Resident Guard)** mode that checks every minute and retrains if Gen2 is lost.<br>2. **DDU Recommendation**: Use **DDU (Display Driver Uninstaller)** in Safe Mode to cleanly remove all old display drivers before reinstalling modded drivers to avoid registry conflicts. |
| **Throttling to Gen1 x16 when idle** | Windows PCIe ASPM power saving feature is enabled | Re-run `Setup_CMP30HX_WindowsAIO.bat` (script automatically disables ASPM) or set Power Options → PCI Express → Link State Power Management to: **Off**. |
| **GPU-Z reports Gen2 x16 but AIDA64 only achieves ~2.5 GB/s** | Card Max Read Request Size (MRRS) is clamped at default 128 Bytes | Run command `40HXInstaller.exe -gen2-30hx` to increase MRRS to 512 Bytes and flush DMA queues. |
| **GPU not detected / Code 43 error** | Resistor mod solder joint for x16 lanes has poor contact, or driver not installed properly | 1. Inspect the resistor mod solder joints on the card.<br>2. Reinstall the NVIDIA driver (use DDU to clean install the latest driver). |
| **All fixes attempted but still failing** | NVIDIA driver registry conflicts, overwritten profiles, or faulty driver services | Cleanly remove old drivers using **DDU (Display Driver Uninstaller)** in Safe Mode, then perform a fresh driver installation. |

---

## <img src="https://api.iconify.design/lucide/trash-2.svg?color=%23ef4444" width="22" height="22" align="center" /> 7. Uninstallation

| Operating System | Command |
|---|---|
| **Windows** (script) | `Setup_CMP30HX_WindowsAIO.bat -uninstall` |
| **Windows** (GUI) | Right-click `40HXUninstaller.exe` → Run as administrator |
| **Linux** | `sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall` |

The uninstaller will automatically delete the Scheduled Task, Registry Run Keys, program directories, and temporary files.

---

## <img src="https://api.iconify.design/lucide/info.svg?color=%238b5cf6" width="22" height="22" align="center" /> 8. Technical Notes

> [!NOTE]
> - **Why only Gen2 and not Gen3?**
>   NVIDIA physically blew the hardware fuse (Silicon eFuse Bit 3 - 8.0 GT/s) on the TU116 silicon at the factory. Gen2 x16 (5.0 GT/s) is the absolute physical hardware ceiling — this tool safely unlocks this ceiling via BAR0 MMIO without attempting to force Gen3, avoiding link training lockups.
> - **MRRS 512B**:
>   Increasing Max Read Request Size from 128B to 512B eliminates TLP packet fragmentation bottlenecks, unlocking the full ~6.4 GB/s DMA bandwidth.

---

## <img src="https://api.iconify.design/lucide/heart.svg?color=%23f43f5e" width="22" height="22" align="center" /> Acknowledgments

> This project is inspired by the original work of **CMP40HX-Unlock**. If you find this tool helpful, please leave a Star on the repository to support the author!
