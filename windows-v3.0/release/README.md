# Windows CMP Unlock Release Package

This ZIP keeps the repository layout and includes the Windows and Linux entry points. Check the release tag and contents on [GitHub Releases](https://github.com/ngthaihoc/CMP30HXmodtoGEN2/releases), then follow this README for the package you downloaded.

Windows code declares CMP 30HX support for TU116 PCI ID `10DE:2189` and CMP 40HX support for TU106 PCI ID `10DE:1F0B`. Results vary with hardware and platform. Gen2 x16 is a target, not a guarantee.

## Main files

| Path | Purpose |
| --- | --- |
| `Setup_CMP30HX_WindowsAIO.bat` | Interactive CMP 30HX install/uninstall and utility menu |
| `Setup_CMP30HX_LinuxAIO.sh` | Linux setup, status, mock, and uninstall entry point |
| `Launch_WebUI.bat` | Launches the CMP Control Center when included |
| `windows-v3.0/release/UnlockAIO.exe` | AIO interface for unlock, diagnostics, utilities, and uninstall |
| `windows-v3.0/release/40HXInstaller.exe` | CMP 40HX installer and management tool |
| `windows-v3.0/release/40HXUninstaller.exe` | CMP 40HX package uninstaller |
| `windows-v3.0/release/40HXCheck.exe` | Hardware and PCIe diagnostic tool |
| `windows-v3.0/release/UnlockRiotGame.exe` | Optional Riot/game configuration utility |
| `windows-v3.0/release/files/40HXUNLK.EFI` | EFI component for the CMP 40HX workflow only |
| `windows-v3.0/release/make_usb_efi.bat` | Writes CMP 40HX EFI files to a selected USB and provides restore mode |
| `windows-v3.0/release/EFI应急修复指南.md` | Recovery guidance for specified EFI boot failures |

Do not use CMP 40HX EFI files on CMP 30HX. File availability may vary by release. Keep the complete ZIP/extracted directory available until uninstall is complete.

## CMP 30HX Windows setup

Run `Setup_CMP30HX_WindowsAIO.bat` as Administrator from the package root. The five menu options are install/unlock, uninstall, NVIDIA Control Panel repair, optional Riot/Unity configuration, and exit.

`-preflight` performs a read-only hardware check and requires exactly one connected GPU with PCI ID `10DE:2189`. `-notask` skips persistence registration for a fresh install. After successful unlock, the SYSTEM task `CMP30HX_Batch_Gen2_Unlock` runs 45 seconds after startup or 10 seconds after logon.

The batch installer files are placed in `%ProgramFiles%\40HXUnlock`. Status is written to `%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt`.

- Select menu option [2] or use `-uninstall` to remove persistence and owned installer files while retaining shared drivers, services, and configuration.
- Use `-cleantasks` to remove the batch task and known legacy tasks only when they run `RunUnlock.bat` or `UnlockAIO.exe` from `%ProgramFiles%\40HXUnlock`. Matching Run values are removed; other tasks and installed tool files are kept.
- Reboot after uninstall to clear runtime unlock state. Keep the ZIP/extracted folder through uninstall.

Verify after installation by checking the success result and PCIe link under GPU load. Reboot and check again. View the task with `Get-ScheduledTask -TaskName 'CMP30HX_Batch_Gen2_Unlock'`. After removing persistence, `schtasks /query /tn "CMP30HX_Batch_Gen2_Unlock"` should report that the task cannot be found. Reboot and confirm the unlock no longer starts automatically.

The default unlock flow does not globally change Windows power/security settings or NVIDIA/Riot configuration. The separate menu utilities can make their documented changes. Uninstall does not undo manual firmware or operating system changes.

## CMP Control Center

When the package includes `UnlockAIO.exe`, `Launch_WebUI.bat` starts the CMP Control Center. The AIO has tabs for Gen2 unlock, diagnostics, Riot & 3D, subtools, and uninstall. Its diagnostic view includes PCIe, MRRS, GSP, clock/temperature, and lane-matrix readings. Optional actions remain user-selected.

## CMP 40HX

Read the included CMP 40HX guide before setup. EFI unlock requires suitable firmware support and may require BIOS changes. Use `40HXCheck.exe` to inspect diagnostics after following the package instructions. `make_usb_efi.bat` writes only to the selected USB volume; verify the drive letter before confirming. Its `/restore` option restores a prior `bootx64.efi` when the script created a backup.

If the system fails to boot after the EFI workflow, use [EFI Emergency Recovery](EFI应急修复指南.md) only for the specified boot failures. The guide describes Windows recovery media steps to remove the CMP EFI files and rebuild Windows boot data. Do not change EFI files when the system boots normally.

## Linux

Run the Linux script from the package root:

```bash
sudo ./Setup_CMP30HX_LinuxAIO.sh
sudo ./Setup_CMP30HX_LinuxAIO.sh --status
sudo ./Setup_CMP30HX_LinuxAIO.sh --uninstall
./Setup_CMP30HX_LinuxAIO.sh -test --no-root
```

Uninstall removes the service, sleep hook, and tool files installed by the script. It does not promise to restore all hardware configuration to its earlier state.

## Verification limits

GPU link status may show Gen1 at idle. Check under load and interpret benchmark results for the specific platform and workload. Mock tests validate software behavior but cannot establish operation on physical CMP hardware or long-term stability. Compatibility with Vanguard, EAC, or BattlEye is not guaranteed.

This project was inspired by early research from the CMP40HX-Unlock community.
