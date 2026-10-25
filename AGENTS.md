# Project: CMP 40HX & CMP 30HX Unlock

## Tech Stack
- Language: Go (minimum version in each tool's `go.mod`), Win32 via `golang.org/x/sys/windows`, Walk / Wails v2 GUI, embedded HTTP server via `embed.FS`
- Frontend: Vanilla HTML5 / Modular CSS3 / ES6 (Zero external runtime dependencies, served via embedded `web/*`)
- C / Assembly: EFI loader payloads (`windows-v3.0/tools/unlock40x/`)
- Platform: Windows 10/11 x64 (Kernel driver `WinRing0x64.sys` / `ThrottleStop.sys`)

## Commands
- Build Installer: `cd windows-v3.0/tools/inst40hx && go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXInstaller.exe" .`
- Build Diagnostic: `cd windows-v3.0/tools/check40x && go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXCheck.exe" .`
- Build Uninstaller: `cd windows-v3.0/tools/uninstall40x && go build -ldflags "-s -w -H=windowsgui" -o "../../release/40HXUninstaller.exe" .`
- Build UnlockRiotGame: `cd windows-v3.0/tools/unlockriot && go build -ldflags "-s -w -H=windowsgui" -o "../../release/UnlockRiotGame.exe" .`
- Build AIO: in `windows-v3.0/tools/aio`, run `wails build -s -webview2 download -ldflags "-s -w -H=windowsgui" -o UnlockAIO.exe`, then `Copy-Item -LiteralPath build/bin/UnlockAIO.exe -Destination ../../release/UnlockAIO.exe`.
- Run All Tests (PowerShell, repo root): `foreach ($toolModule in '40hxcore','riotcore','unlockriot','aio','inst40hx','check40x','uninstall40x','fix30hxcsm') { go -C "windows-v3.0/tools/$toolModule" test ./...; if ($LASTEXITCODE -ne 0) { throw "Kiểm thử thất bại: $toolModule" } }`
- Frontend Check: `node windows-v3.0/tools/aio/tests/telemetry.test.cjs`
- Unlock 30HX on Logon: `schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"D:\ClodeGithub\CMP40HX-Unlock-main\windows-v3.0\release\40HXInstaller.exe\" -gen2-30hx -silent" /sc onlogon /rl highest /f`
- Stop Gen3 Retry Loop: `schtasks /delete /tn "40HXGen2Retry" /f`
- Unlock Riot Zero-Reinstall: `windows-v3.0\release\UnlockRiotGame.exe -silent -optimize`
- Query Riot JSON Status: `windows-v3.0\release\UnlockRiotGame.exe -json-status`
- Launch Web Control Center: `Launch_WebUI.bat`

## Code Conventions
- **The Ladder of Laziness**: Stop at the first rung that holds: 1. YAGNI -> 2. Reuse codebase -> 3. Stdlib -> 4. Native platform -> 5. Installed dependency -> 6. One-liner -> 7. Write minimum code.
- **Bug Fixes**: Address the root cause, not the symptom. Fix the shared function once rather than patching every caller.
- **Simplicity**: No unrequested abstractions, no boilerplate, no unnecessary dependencies. Deletion over addition.
- **Testing**: Non-trivial logic must leave ONE runnable check behind (assert-based demo or small test file). Trivial one-liners need no tests.
- **Localization**: Maintain all logs, messages, and UI strings in Vietnamese.
- **Web UI Architecture**: All web assets must exist identically in both `windows-v3.0/tools/inst40hx/web/` and `windows-v3.0/release/web/`. SSE endpoint is strictly `/api/logs/stream`. Styles are modularized across `base.css`, `hud.css`, `components.css`, and `panels.css`.

## Boundaries
- **Track Release Binaries in Git (IMMUTABLE)**: Prebuilt production executables (`windows-v3.0/release/*.exe`), EFI loader (`files/40HXUNLK.EFI`), and kernel drivers (`files/*.sys`, `gen2/drivers/*.sys`) MUST be permanently tracked in Git. Do NOT add them to `.gitignore` and NEVER remove them from git tracking. End users clone/download the repository directly to run via batch scripts without Go/C/EDK2 compilers.
- **Hardware-Locked Gen2**: CMP 30HX silicon eFuse bit 3 (8.0 GT/s) is physically blown. Do NOT attempt to force Gen3 or higher. Target must be clamped to Gen2 (`targetGen = 2`).
- **Never Flash 40HX Firmware on 30HX**: Do not apply `40HXUNLK.EFI` or GA102/TU106 microcode blobs to TU116 (CMP 30HX).
- **No Dangerous Resets on 30HX**: Do not invoke `gen2RootLinkDisable`, `gen2HardFallback`, or PnP resets on CMP 30HX.
- **DEVCTL MRRS**: Ensure Max Read Request Size is set to 512B (`0x2000`) and restart `NVDisplay.ContainerLocalSystem` for full ~6.4 GB/s throughput.
- **Anti-Cheat & Modern Driver Integrity**: Disable `VulnerableDriverBlocklistEnable=0`, enforce `WindowMode=2` + `DX11Support=1` for LoL, and route global 3D graphics via `HighPerfAdapter` without re-signing or modifying system driver binaries.
- **End-User Distribution**: Package production binaries + scripts into `CMP30HX_Unlock_vX.X.zip` (strictly exclude `windows-v3.0/tools/` source trees, tests, and internal documentation).

## Domain Architecture
- **LinkNegotiator**: Deep module orchestrating PCIe link retrain, profile eFuse enforcement, and MMIO shadow sequences.
- **ComputeInspector**: Deep module validating BAR0 BOOT_0 architecture identity and decoding Tensor Core unlock states.
- **HardwareBus**: Interface isolating physical PCI configuration and MMIO access from negotiation logic.
- **GPUTopologyDetector**: Dynamic scanning of active display host adapters vs headless render CMP dGPUs.
- **StatusContract**: Structured communication protocol between Go engines and orchestration scripts.

## Agent Skills
- **Resume / handoff**: Khi tiếp tục sau khi đổi phiên hoặc compact, đọc `docs/agents/session-handoff.md`, rồi đối chiếu `git status --short` và HEAD trước khi sửa. Đọc phần spec và mã liên quan theo nhiệm vụ; `PROJECT_CONTEXT.md` là tài liệu lịch sử.
- **Issue tracker**: Issues and specs for this repo live as GitHub issues. See `docs/agents/issue-tracker.md`.
- **Domain docs**: Single-context repo layout (`CONTEXT.md` and `docs/adr/` at the root). See `docs/agents/domain.md`.
