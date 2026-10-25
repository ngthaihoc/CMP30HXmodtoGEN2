# PCIe Bandwidth, Compute Unlock & Anti-Cheat Compatibility

Orchestrates PCIe Gen2 link negotiation, eFuse hardware protection, Tensor Core compute enablement, and anti-cheat MSHybrid CASO routing across NVIDIA CMP mining cards.

## Language

**LinkNegotiator**:
The deep module orchestrating PCIe link retrain, profile eFuse enforcement, MMIO shadow registers, and hardware recovery across GPU endpoints.
_Avoid_: LinkManager, PcieHandler, SpeedSetter

**ComputeInspector**:
The deep module validating BAR0 BOOT_0 architecture identity and decoding Tensor Core hardware unlock state (SS0/SS1).
_Avoid_: TensorChecker, ComputeStatus, GpuAuditor

**HardwareBus**:
The hardware abstraction seam isolating physical PCI configuration and MMIO access from negotiation logic.
_Avoid_: DriverBus, IoBridge, HardwareService

**StatusContract**:
The structured typed communication protocol exchanging execution verdicts between the Go engine and consuming shell scripts.
_Avoid_: StatusLog, ResultFile, Gen2Output

**GPUTopologyDetector**:
The deep module dynamically scanning active display outputs via Win32 `EnumDisplayDevicesW` and `GpuClassPath` to classify active display host GPUs (`EnableMsHybrid=2`) versus discrete render-only CMP cards (`FeatureScore=0xD1`, `EnableMsHybrid=1`, `CASO=1`).
_Avoid_: ScreenChecker, OutputScanner, DisplayFinder

**DirectX Global Router**:
The system configuring `DirectXUserGlobalSettings` (`HighPerfAdapter=<CMP_DID>;SwapEffectUpgradeEnable=1;`) to automatically route all 3D gaming workloads to CMP without manual per-executable entry.
_Avoid_: GpuPicker, GameRouter, DxPreferencesHack

**Vanguard Blocklist Remediator**:
The module disabling Microsoft Vulnerable Driver Blocklist (`VulnerableDriverBlocklistEnable=0`) in `Control\CI\Config` to prevent Windows Code Integrity and Riot Vanguard (`vgk.sys`) from revoking driver mod binaries (5xx/6xx).
_Avoid_: BlocklistBypass, AnticheatPatcher, CertCracker

**Borderless Presentation Enforcement**:
The automated configuration setting League of Legends `game.cfg` across all regional installs (Riot, VNG, Garena) to `WindowMode=2`, `BorderlessWindow=1`, `PreferDX9LegacyMode=0`, and `DX11Support=1`, resolving the fatal `Could not Initialize Graphics Device` error.
_Avoid_: LolFix, GraphicsErrorPatch, WindowHack

**Authenticode Classifier**:
The module distinguishing genuine Microsoft WHQL signatures (`WHQL_GENUINE`), Dartraiden PKI modified drivers (`DARTRAIDEN_MOD`), and self-signed certificates (`SELF_SIGNED`) in user-mode graphics drivers (`nvwgf2umx.dll`).
_Avoid_: CertChecker, SignatureProbe, DllAuditor

**GPU Profile**:
The configuration entity defining silicon architecture boundaries, maximum allowable PCIe generations, and firmware unlock requirements for a device ID.
_Avoid_: CardConfig, DeviceInfo, ModelSpec

**eFuse Lock**:
Physical hardware silicon bit cut restricting the maximum achievable link speed on specific dies (e.g. TU116 bit 3 limited to 5.0 GT/s).
_Avoid_: HardCap, SiliconFuse, HardwareBlock

**Shadow Sequence**:
The ordered table of private MMIO register writes applied to GPU BAR0 to unlock PCIe link capabilities before retrain.
_Avoid_: RegPatch, MagicSequence, MMIOTable

**Stage 2 Recovery**:
The fallback sequence combining Root Port Link Disable or PnP device reset with register re-injection when standard link retrain fails to achieve target speed.
_Avoid_: HardReset, ForceRecover, FallbackHack

**MRRS 512B**:
Maximum Read Request Size optimization configured in PCI Device Control to prevent DMA memory fragmentation and maximize bus throughput.
_Avoid_: BufferTweak, DevctlHack, DmaBoost

**MSHybrid (Microsoft Hybrid Graphics)**:
The multi-adapter display architecture pairing headless discrete render GPU (EnableMsHybrid=1, FeatureScore=0xD1) with host scan-out adapter (EnableMsHybrid=2) over Desktop Window Manager (DWM).
_Avoid_: OptimusHack, DualGpuFix, SwitchableGpu

**CASO (Cross-Adapter Scan-Out)**:
Hardware direct scan-out feature across PCIe bus allowing the display controller to scan out dGPU render buffers directly without intermediate blit copies (EnableCrossAdapterScanOut=1).
_Avoid_: FastScanout, DirectPcie, CasoHack

**BYOVD Remediation**:
The sanitization routine removing vulnerable third-party kernel drivers (WinRing0, ThrottleStop) and disabling testsigning to comply with Riot Vanguard (vgk.sys) and EAC anti-cheat code integrity.
_Avoid_: DriverCleanup, AnticheatFix, KernelCleaner

**Release Binary Tracking**:
Prebuilt production executables (`windows-v3.0/release/*.exe`), EFI loader (`files/40HXUNLK.EFI`), and kernel driver assets (`files/*.sys`, `gen2/drivers/*.sys`) are deliberately tracked in git repository to allow end-users to clone and run immediately via batch scripts without Go, C, or EDK2 toolchains.
_Avoid_: Untracking release binaries, adding release PE files to .gitignore, stripping prebuilts from repository.

**Web Control Center**:
The embedded hardware control workstation providing real-time telemetry, 16-lane PCIe matrix visualization, and 1-click execution over localhost loopback via Go `embed.FS` and Server-Sent Events (`/api/logs/stream`).
_Avoid_: LocalhostSite, AdminPortal, WebWrapper, ElectronApp
