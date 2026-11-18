<p align="right">
  <b>Language:</b>
  <a href="README.md">Tiếng Việt</a> |
  <b>English</b> |
  <a href="README_ZH.md">简体中文</a>
</p>

# NVIDIA CMP 30HX / 40HX PCIe Unlock

A research tool for enabling PCIe Gen2 links on NVIDIA CMP 30HX (TU116, PCI ID `10DE:2189`) and CMP 40HX (TU106, PCI ID `10DE:1F0B`). Windows code supports these two IDs. Results depend on the GPU, motherboard, slot, riser, firmware, and driver. Gen2 x16 is a technical target, not a performance guarantee.

[![GitHub Releases](https://img.shields.io/badge/GitHub-Releases-181717?logo=github)](https://github.com/ngthaihoc/CMP30HXmodtoGEN2/releases)

## Choose a tool

- **Windows CMP 30HX:** run `Setup_CMP30HX_WindowsAIO.bat` from the project folder as Administrator.
- **Windows CMP 40HX:** see the [Windows guide](windows-v3.0/README.md) and the guide included with the release package. CMP 40HX EFI files are for CMP 40HX only.
- **Linux:** use `Setup_CMP30HX_LinuxAIO.sh` to install, check status, or remove the service. Read the script's help before running it.
- **Resizable BAR / CSM:** `Setup_ReBar_CMP30HX.bat` and `Fix_CMP30HX_CSM_Error43.bat` are separate utilities that may change system or device configuration. Review their prompts and documentation before confirming.

## Windows CMP 30HX

Download and extract the complete project ZIP or release package. Keep the extracted folder available for uninstall. Open an elevated Terminal or Command Prompt in that folder.

```cmd
Setup_CMP30HX_WindowsAIO.bat
Setup_CMP30HX_WindowsAIO.bat -preflight
Setup_CMP30HX_WindowsAIO.bat -notask
Setup_CMP30HX_WindowsAIO.bat -cleantasks
Setup_CMP30HX_WindowsAIO.bat -uninstall
```

The interactive menu has five choices: [1] install and unlock, [2] uninstall, [3] repair NVIDIA Control Panel, [4] optional Riot/Unity configuration, and [5] exit. Choices [3] and [4] are separate actions and can change NVIDIA or game settings.

- `-preflight` checks hardware only. Exactly **one** connected GPU with CMP 30HX PCI ID `10DE:2189` is required. The script rejects zero or multiple matches.
- After a successful unlock, the script registers the SYSTEM task `CMP30HX_Batch_Gen2_Unlock`. It runs 45 seconds after startup or 10 seconds after logon. The installer files are placed in `%ProgramFiles%\40HXUnlock`, and batch status is written to `%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt`.
- `-notask` skips persistence registration on a fresh install. The unlock then applies only to the current session.
- `-cleantasks` removes this installer’s persistence task and known legacy task names. It does not uninstall the tool files.
- `-uninstall` removes the installer’s persistence and owned files while retaining shared drivers, services, and configuration. Reboot to clear runtime unlock state for the current session.

To verify installation, inspect the result and check PCIe status under GPU load, then reboot and check again. Inspect the scheduled task with:

```powershell
Get-ScheduledTask -TaskName 'CMP30HX_Batch_Gen2_Unlock'
```

Choose [2] in the menu or run `-cleantasks` to remove persistence. Verify removal with:

```cmd
schtasks /query /tn "CMP30HX_Batch_Gen2_Unlock"
```

After successful removal, Windows reports that the task cannot be found. Reboot and confirm that the link is no longer unlocked automatically. Keep the ZIP or extracted project folder until uninstall is complete.

## Control Center and utilities

`Launch_WebUI.bat` starts the CMP Control Center when it is present in the package. The AIO interface has tabs for Gen2 unlock, diagnostics, Riot & 3D, subtools, and uninstall. Diagnostics show PCIe, MRRS, GSP, clock/temperature, and lane-matrix information. Riot/Unity configuration and NVIDIA Control Panel repair are optional actions. They do not run as part of the default unlock flow.

## Linux

```bash
chmod +x Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh --status
sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
./Setup_CMP30HX_LinuxAIO.sh -test --no-root
```

Linux uninstall removes the service, sleep hook, and tool files installed by the script. It does not reverse manual changes or every hardware configuration state. Reboot if you need to refresh PCIe runtime state.

## Verification, recovery, and limits

Use GPU-Z or a diagnostic tool under GPU load. An idle GPU may report Gen1 due to power management. Benchmark results vary with platform and workload, so no single bandwidth figure is guaranteed.

Mock tests verify software paths, not unlock operation on physical CMP hardware, stability across sleep/reboot, or anti-cheat compatibility. Vanguard, EAC, and BattlEye compatibility is not guaranteed. Validate on the target hardware before daily use.

If a CMP 40HX system fails to boot after using the EFI workflow, consult the [EFI recovery guide](windows-v3.0/release/EFI应急修复指南.md) only for the boot failures described there. Do not modify the EFI partition when the system boots normally.

## Downloads, support, and attribution

Download packages from [GitHub Releases](https://github.com/ngthaihoc/CMP30HXmodtoGEN2/releases). Follow the guide shipped with the specific package because contents may differ by version.

The project is shared free of charge. If it helps you, you can support the author via MoMo/VietQR using the [QR image](assets/donate_momo.jpg). This project was inspired by early research from the CMP40HX-Unlock community.

Follow local laws, manufacturer terms, and the rules of the software you use.
