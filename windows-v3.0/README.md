# Windows CMP Unlock Guide

This repository package contains Windows and Linux entry points. The Windows core declares support for CMP 30HX (TU116, PCI ID `10DE:2189`) and CMP 40HX (TU106, PCI ID `10DE:1F0B`). Results depend on the GPU, motherboard, slot, riser, firmware, and driver. Gen2 x16 is a target, not a guarantee.

## CMP 30HX AIO

Keep the complete downloaded ZIP or extracted project folder. The uninstaller uses files from the package, so do not delete or move the folder before uninstalling. Open an elevated terminal in the project root and run:

```cmd
Setup_CMP30HX_WindowsAIO.bat
```

The menu has five choices:

1. Install and unlock Gen2.
2. Uninstall the batch package.
3. Repair NVIDIA Control Panel.
4. Run optional Riot/Unity configuration.
5. Exit.

Choices 3 and 4 are explicit utilities that can change NVIDIA or game settings. They do not run as part of the default unlock flow.

Command-line options:

| Command | Behavior |
| --- | --- |
| `Setup_CMP30HX_WindowsAIO.bat -preflight` | Read-only check requiring exactly one connected GPU with PCI ID `10DE:2189`. Zero or multiple matches fail. |
| `Setup_CMP30HX_WindowsAIO.bat -notask` | Installs/unlocks for the current session without registering persistence on a fresh install. |
| `Setup_CMP30HX_WindowsAIO.bat -cleantasks` | Removes the batch task and known legacy tasks only when they run the installed `RunUnlock.bat` or `UnlockAIO.exe`; removes matching Run values and keeps other tasks and tool files. |
| `Setup_CMP30HX_WindowsAIO.bat -uninstall` | Removes the batch package's persistence and owned files. Retains shared drivers, services, and configuration. |

After a successful unlock, the SYSTEM task `CMP30HX_Batch_Gen2_Unlock` runs 45 seconds after startup or 10 seconds after logon. The installer files are placed in `%ProgramFiles%\40HXUnlock`; batch status is written to `%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt`.

To verify installation, inspect the success output and check PCIe under GPU load. Reboot and verify the link again. The task can be inspected with `Get-ScheduledTask -TaskName 'CMP30HX_Batch_Gen2_Unlock'`. To remove persistence, choose menu option [2] or run `-cleantasks`. Confirm removal with `schtasks /query /tn "CMP30HX_Batch_Gen2_Unlock"`, then reboot and verify that the unlock no longer runs automatically. Keep the ZIP or extracted folder until uninstall completes.

The default unlock flow does not globally change power settings, Windows security, NVIDIA, or Riot settings. Reboot after uninstall to clear runtime unlock state for the current session. Uninstall does not reverse settings changed manually or by separate utilities.

## Control Center

When included, `Launch_WebUI.bat` starts the CMP Control Center. Its AIO interface provides Gen2 unlock, diagnostics, Riot & 3D, subtools, and uninstall tabs. The diagnostic panel reports PCIe link details, MRRS, GSP, clock/temperature, and lane-matrix information. Riot/Unity and NVIDIA repair functions are optional actions.

## CMP 40HX

CMP 40HX uses the TU106 profile. The release package includes CMP 40HX installer, uninstaller, diagnostic, EFI, and recovery files. Follow the README shipped with the selected release for its exact BIOS prerequisites and setup steps. The EFI workflow can require UEFI/BIOS configuration. Never use CMP 40HX EFI firmware on CMP 30HX.

If a CMP 40HX system fails to boot after the EFI workflow, consult [EFI Emergency Recovery](release/EFI应急修复指南.md) only for the failures described there. The guide uses Windows recovery media to remove the CMP EFI entry/files and rebuild Windows boot data. Do not alter EFI files when Windows boots normally.

## Linux

The repository includes `Setup_CMP30HX_LinuxAIO.sh` for installation, status, mock execution, and service removal:

```bash
sudo ./Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh --status
sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
./Setup_CMP30HX_LinuxAIO.sh -test --no-root
```

Linux removal cleans up the service, sleep hook, and tool files installed by the script. It does not reverse manual changes or guarantee hardware settings return to their prior values.

## Diagnostics and limitations

Check PCIe link status under GPU load. Idle power management may reduce the reported link speed to Gen1. Benchmark throughput varies by workload and platform. Mock tests cover software paths and error handling, not physical CMP operation or long-term stability. Anti-cheat compatibility with Vanguard, EAC, or BattlEye is not guaranteed.

This project was inspired by early research from the CMP40HX-Unlock community.
