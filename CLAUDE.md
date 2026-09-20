# CLAUDE.md — CMP 40HX & CMP 30HX Unlock Project

Project context file: Read `PROJECT_CONTEXT.md` for full hardware details, register maps, benchmark findings, and technical history.

## Tech Stack
- Language: Go 1.20+ (Win32 API via `golang.org/x/sys/windows`, Walk GUI framework)
- C / Assembly: EFI loader payloads (`windows-v3.0/tools/unlock40x/`)
- Platform: Windows 10/11 x64 (Kernel driver `WinRing0x64.sys` / `ThrottleStop.sys`)

## Key Commands
- Build Installer: `cd windows-v3.0/tools/inst40hx && go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXInstaller.exe" .`
- Build Diagnostic: `cd windows-v3.0/tools/check40x && go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXCheck.exe" .`
- Build Uninstaller: `cd windows-v3.0/tools/uninstall40x && go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXUninstaller.exe" .`
- Run Tests: `cd windows-v3.0/tools/40hxcore && go test -v ./...`
- Unlock 30HX on Logon: `schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f`
- Stop Gen3 Retry Loop: `schtasks /delete /tn "40HXGen2Retry" /f`

## Critical Hardware Rules & Boundaries
1. **CMP 30HX is Hardware-Locked to Gen2**: Silicon eFuse bit 3 (8.0 GT/s) is physically blown by NVIDIA in TU116. Do NOT attempt to force Gen3 or higher. Target must be clamped to Gen2 (`targetGen = 2`).
2. **Never Flash 40HX Firmware on 30HX**: Do not apply `40HXUNLK.EFI` or GA102/TU106 microcode blobs to TU116 (CMP 30HX).
3. **No Dangerous Resets on 30HX**: Do not invoke `gen2RootLinkDisable`, `gen2HardFallback`, or PnP resets on CMP 30HX.
4. **DEVCTL MRRS**: Ensure Max Read Request Size is set to 512B (`0x2000`) and restart `NVDisplay.ContainerLocalSystem` for full ~6.4 GB/s throughput.
5. **Localization**: Maintain all logs, messages, and UI strings in Vietnamese.
