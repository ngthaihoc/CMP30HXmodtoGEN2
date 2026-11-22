@echo off
setlocal DisableDelayedExpansion
chcp 65001 >nul
title Thu thập báo cáo VGA - CMP30HXmodtoGEN2

if /i not "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    if /i not "%PROCESSOR_ARCHITEW6432%"=="AMD64" (
        echo [LỖI] Công cụ này chỉ hỗ trợ Windows 10/11 64-bit ^(x64^).
        pause
        exit /b 1
    )
)

set "VGA_BAT_SELF=%~f0"
set "VGA_BAT_DIR=%~dp0"
set "VGA_COLLECTOR_LIB_ONLY=0"

:parse_args
if "%~1"=="" goto args_done
if /i "%~1"=="-SaveOnly" set "VGA_SAVE_ONLY=1"
if /i "%~1"=="-KeepHistory" set "VGA_KEEP_HISTORY=1"
if /i "%~1"=="-DetailedContext" set "VGA_DETAILED_CONTEXT=1"
if /i "%~1"=="-SelectGpu" set "VGA_SELECT_GPU=1"
if /i "%~1"=="-RunLoadTest" set "VGA_RUN_LOAD_TEST=1"
shift
goto parse_args
:args_done

powershell.exe -NoProfile -Command "& { $raw = [System.IO.File]::ReadAllText($env:VGA_BAT_SELF, [System.Text.Encoding]::UTF8); $m = '# === BEGIN_POWERSHELL ==='; $idx = $raw.LastIndexOf($m); if ($idx -lt 0) { Write-Error 'Không tìm thấy khối PowerShell trong file BAT.'; exit 1 }; $code = $raw.Substring($idx + $m.Length); . ([ScriptBlock]::Create($code)) }"
set "EXIT_CODE=%ERRORLEVEL%"
exit /b %EXIT_CODE%

# === BEGIN_POWERSHELL ===
$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
$OutputEncoding = [Console]::OutputEncoding

if ($PSVersionTable.PSVersion.Major -lt 5 -or ($PSVersionTable.PSVersion.Major -eq 5 -and $PSVersionTable.PSVersion.Minor -lt 1)) {
    throw 'Yêu cầu Windows PowerShell 5.1 trở lên.'
}
if (-not [Environment]::Is64BitOperatingSystem) {
    throw 'Yêu cầu Windows 10/11 64-bit (x64).'
}

$script:CollectorVersion = '1.1.0'
$script:SchemaVersion = 2
# Khóa định danh client công khai dùng để giới hạn tần suất (anti-spam rate-limit tag) trên Worker.
# Đây KHÔNG phải khóa bí mật cá nhân. Mã nguồn máy chủ công khai tại: worker/vga-reports/src/index.js
# Nếu muốn vô hiệu hóa hoàn toàn tính năng gửi mạng, hãy đổi giá trị $script:SigningKey thành 'REPLACE_WITH_DISABLED'.
$script:AllowedEndpointHost = 'cmp-vga-reports.wellynguyen.workers.dev'
$script:DefaultEndpoint = "https://$($script:AllowedEndpointHost)/v1/reports"
$script:TargetHostname = $script:AllowedEndpointHost
$script:SigningKeyId = 'k1'
$script:SigningKey = 'bEsyUvbb55hLMnm5i8ul/dBC+Eo4iVXPLDNoCuJOrs0='
$script:FallbackIssueUrl = 'https://github.com/ngthaihoc/CMP30HXmodtoGEN2/issues/new?template=vga-report.yml'

$script:GpuProfileDetails = [ordered]@{
    '10DE:1F0B' = [ordered]@{
        id                = '10DE:1F0B'
        name              = 'CMP 40HX (TU106)'
        family            = 'TU106'
        category          = 'cmp'
        target_pcie_gen   = 2
        tensor_applicable = 'unverified'
        identity_note     = $null
    }
    '10DE:2189' = [ordered]@{
        id                = '10DE:2189'
        name              = 'CMP 30HX (TU116)'
        family            = 'TU116'
        category          = 'cmp'
        target_pcie_gen   = 2
        tensor_applicable = 'no'
        identity_note     = $null
    }
    '10DE:1C07' = [ordered]@{
        id                = '10DE:1C07'
        name              = 'P106-100 (GP106)'
        family            = 'GP106'
        category          = 'mining'
        target_pcie_gen   = $null
        tensor_applicable = 'no'
        identity_note     = $null
    }
    '10DE:1C09' = [ordered]@{
        id                = '10DE:1C09'
        name              = 'P106-090 (GP106)'
        family            = 'GP106'
        category          = 'mining'
        target_pcie_gen   = $null
        tensor_applicable = 'no'
        identity_note     = $null
    }
    '10DE:15F8' = [ordered]@{
        id                = '10DE:15F8'
        name              = 'Tesla P100 PCIe 16GB (GP100)'
        family            = 'GP100'
        category          = 'datacenter'
        target_pcie_gen   = 3
        tensor_applicable = 'no'
        identity_note     = 'Xác minh SUBSYS/dạng card trước khi áp định mức 250W'
    }
    '10DE:15F7' = [ordered]@{
        id                = '10DE:15F7'
        name              = 'Tesla P100 PCIe 12GB (GP100)'
        family            = 'GP100'
        category          = 'datacenter'
        target_pcie_gen   = 3
        tensor_applicable = 'no'
        identity_note     = 'Xác minh SUBSYS/dạng card trước khi áp định mức 250W'
    }
    '10DE:15F9' = [ordered]@{
        id                = '10DE:15F9'
        name              = 'Tesla P100 SXM2 (GP100)'
        family            = 'GP100'
        category          = 'datacenter'
        target_pcie_gen   = 3
        tensor_applicable = 'no'
        identity_note     = 'SXM2 dùng baseline công suất/kết nối riêng'
    }
}

$script:GpuProfiles = [ordered]@{}
foreach ($pk in $script:GpuProfileDetails.Keys) {
    $script:GpuProfiles[$pk] = $script:GpuProfileDetails[$pk].name
}

$script:NvidiaRequiredFields = @(
    'name',
    'driver_version',
    'vbios_version',
    'pci.bus_id',
    'pci.device_id',
    'pci.sub_device_id',
    'memory.total',
    'compute_cap',
    'pcie.link.gen.max',
    'pcie.link.gen.gpucurrent',
    'pcie.link.width.max',
    'pcie.link.width.current',
    'temperature.gpu',
    'power.draw',
    'clocks.sm',
    'clocks.mem',
    'clocks.max.sm',
    'pstate'
)

$script:NvidiaExtendedFields = @(
    'power.limit',
    'enforced.power.limit',
    'power.default_limit',
    'power.min_limit',
    'power.max_limit',
    'temperature.gpu.tlimit',
    'temperature.memory',
    'fan.speed',
    'utilization.gpu',
    'utilization.memory',
    'memory.used',
    'memory.free',
    'ecc.mode.current',
    'clocks_throttle_reasons.active',
    'clocks_throttle_reasons.sw_power_cap',
    'clocks_throttle_reasons.hw_slowdown',
    'clocks_throttle_reasons.hw_thermal_slowdown',
    'clocks_throttle_reasons.hw_power_brake_slowdown'
)

$script:NumericNvidiaFields = @{
    'memory.total'            = 'MiB'
    'pcie.link.gen.max'       = 'gen'
    'pcie.link.gen.gpucurrent'= 'gen'
    'pcie.link.width.max'     = 'lanes'
    'pcie.link.width.current' = 'lanes'
    'temperature.gpu'         = 'C'
    'power.draw'              = 'W'
    'clocks.sm'               = 'MHz'
    'clocks.mem'              = 'MHz'
    'clocks.max.sm'           = 'MHz'
    'power.limit'             = 'W'
    'enforced.power.limit'    = 'W'
    'power.default_limit'     = 'W'
    'power.min_limit'         = 'W'
    'power.max_limit'         = 'W'
    'temperature.gpu.tlimit'  = 'C'
    'temperature.memory'      = 'C'
    'fan.speed'               = '%'
    'utilization.gpu'         = '%'
    'utilization.memory'      = '%'
    'memory.used'             = 'MiB'
    'memory.free'             = 'MiB'
}

$script:LastSourceTimestamps = @{
    pnp_utc        = $null
    cim_utc        = $null
    nvidia_smi_utc = $null
    security_utc   = $null
}

function Remove-Pii {
    param([AllowNull()][string]$Text)
    if ($null -eq $Text -or $Text -eq '') { return '' }
    $out = $Text
    $out = [regex]::Replace($out, '(?i)[A-Za-z]:\\Users\\[^\\\s"''<>|]+(?:\\[^\s"''<>|]*)*', '[PATH_USER]')
    $out = [regex]::Replace($out, '(?i)\\\\[^\\\s"''<>|]+\\[^\\\s"''<>|]+(?:\\[^\s"''<>|]*)*', '[PATH_USER]')
    $out = [regex]::Replace($out, '(?i)(?:^|(?<=[\s"'']))/(?:home|Users)/[^/\s"''<>|]+(?:/[^\s"''<>|]*)*', '[PATH_USER]')
    $out = [regex]::Replace($out, '(?i)%USERPROFILE%', '[PATH_USER]')
    $out = [regex]::Replace($out, '(?<![0-9a-fA-F:-])(?:[0-9a-fA-F]{2}[:-]){5}[0-9a-fA-F]{2}(?![0-9a-fA-F:-])', '[MAC]')
    $out = [regex]::Replace($out, '(?<![\w:.])(?:[a-fA-F0-9]*:){2,}[a-fA-F0-9:.]*(?:%[a-zA-Z0-9_.~-]+)?', {
        param($match)
        $candidate = $match.Value.TrimEnd('.')
        $address = $null
        if ([System.Net.IPAddress]::TryParse(($candidate -split '%')[0], [ref]$address) -and $address.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetworkV6) {
            return '[IPV6]' + $match.Value.Substring($candidate.Length)
        }
        return $match.Value
    })
    $out = [regex]::Replace($out, '(?<![.\dA-Za-z])(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]\d|\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]\d|\d)){3}(?![.\dA-Za-z])', '[IPV4]')
    $out = [regex]::Replace($out, '(?i)\b(?:MIG-)?GPU-[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b', '[UUID_GPU]')
    $out = [regex]::Replace($out, '(?i)\b(?:serial(?:\s*(?:number|no|#))?|s/n|sn)\s*[:=]\s*[A-Za-z0-9_-]{4,}', '[SERIAL]')
    return $out
}

function Invoke-RedactReportInPlace {
    param($Node)
    if ($null -eq $Node) { return }
    if ($Node -is [System.Collections.IDictionary]) {
        $keys = @($Node.Keys)
        foreach ($k in $keys) {
            $val = $Node[$k]
            if ($val -is [string]) {
                $Node[$k] = Remove-Pii -Text $val
            } elseif ($null -ne $val -and -not ($val.GetType().IsPrimitive -or $val -is [decimal])) {
                Invoke-RedactReportInPlace -Node $val
            }
        }
        return
    }
    if ($Node -is [System.Collections.IList]) {
        for ($i = 0; $i -lt $Node.Count; $i += 1) {
            $item = $Node[$i]
            if ($item -is [string]) {
                $Node[$i] = Remove-Pii -Text $item
            } elseif ($null -ne $item -and -not ($item.GetType().IsPrimitive -or $item -is [decimal])) {
                Invoke-RedactReportInPlace -Node $item
            }
        }
        return
    }
    if ($Node -is [pscustomobject]) {
        foreach ($p in @($Node.PSObject.Properties)) {
            if ($p.IsSettable) {
                $val = $p.Value
                if ($val -is [string]) {
                    $p.Value = Remove-Pii -Text $val
                } elseif ($null -ne $val -and -not ($val.GetType().IsPrimitive -or $val -is [decimal])) {
                    Invoke-RedactReportInPlace -Node $val
                }
            }
        }
    }
}

function Redact-ReportObject {
    param($InputObject)
    if ($null -eq $InputObject) { return $null }
    if ($InputObject -is [string]) {
        return (Remove-Pii -Text $InputObject)
    }
    Invoke-RedactReportInPlace -Node $InputObject
    if ($InputObject -is [System.Collections.IList]) {
        return ,$InputObject
    }
    return $InputObject
}

function New-ReportField {
    param(
        $Value,
        [string]$Source,
        [AllowNull()][string]$Unit = $null,
        [string]$Status = 'ok'
    )
    if ($Status -ne 'ok') {
        return [ordered]@{
            value  = $null
            source = $Source
            unit   = $Unit
            status = $Status
        }
    }
    if ($null -eq $Value) {
        return [ordered]@{
            value  = $null
            source = $Source
            unit   = $Unit
            status = 'unsupported'
        }
    }
    if ($Value -is [string]) {
        $trimmed = $Value.Trim()
        if ($trimmed -eq '' -or $trimmed -eq '[N/A]' -or $trimmed -eq '[Not Supported]' -or $trimmed -eq 'N/A') {
            return [ordered]@{
                value  = $null
                source = $Source
                unit   = $Unit
                status = 'unsupported'
            }
        }
        $clean = Remove-Pii -Text $trimmed
        if ($clean.Length -gt 500) {
            $clean = $clean.Substring(0, 500)
        }
        return [ordered]@{
            value  = $clean
            source = $Source
            unit   = $Unit
            status = 'ok'
        }
    }
    return [ordered]@{
        value  = $Value
        source = $Source
        unit   = $Unit
        status = 'ok'
    }
}

function Normalize-Bdf {
    param([AllowNull()][string]$BusId)
    if ([string]::IsNullOrWhiteSpace($BusId)) { return $null }
    $t = $BusId.Trim().ToLowerInvariant()
    if ($t -match '^0000([0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7])$') {
        return $Matches[1]
    }
    if ($t -match '^([0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7])$') {
        return $Matches[1]
    }
    if ($t -match '^([0-9a-f]{2}:[0-9a-f]{2}\.[0-7])$') {
        return "0000:$($Matches[1])"
    }
    return $t
}

function Get-RawCimDisplayDevices {
    try {
        return @(Get-CimInstance -ClassName Win32_VideoController -ErrorAction Stop | Select-Object Name, PNPDeviceID, DriverVersion, Status, ConfigManagerErrorCode)
    } catch {
        return @()
    }
}

function Get-RawPnpDisplayDevices {
    try {
        return @(Get-PnpDevice -Class Display -PresentOnly -ErrorAction Stop | Select-Object InstanceId, FriendlyName, Status)
    } catch {
        return @()
    }
}

function Get-RawPnpDeviceProperties {
    param([string]$InstanceId)
    try {
        return @(Get-PnpDeviceProperty -InstanceId $InstanceId -ErrorAction Stop | Select-Object KeyName, Data)
    } catch {
        return @()
    }
}

function Get-RawPlatformInfo {
    $cpuName = $null
    $board = $null
    $bios = $null
    $os = $null
    try {
        $c = Get-CimInstance -ClassName Win32_Processor -ErrorAction Stop | Select-Object -First 1 Name
        if ($c -and $c.Name) { $cpuName = $c.Name.Trim() }
    } catch {}
    try {
        $b = Get-CimInstance -ClassName Win32_BaseBoard -ErrorAction Stop | Select-Object -First 1 Manufacturer, Product
        if ($b) {
            $parts = @($b.Manufacturer, $b.Product) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | ForEach-Object { $_.Trim() }
            if ($parts.Count -gt 0) { $board = ($parts -join ' ') }
        }
    } catch {}
    try {
        $bi = Get-CimInstance -ClassName Win32_BIOS -ErrorAction Stop | Select-Object -First 1 SMBIOSBIOSVersion
        if ($bi -and $bi.SMBIOSBIOSVersion) { $bios = $bi.SMBIOSBIOSVersion.Trim() }
    } catch {}
    try {
        $o = Get-CimInstance -ClassName Win32_OperatingSystem -ErrorAction Stop | Select-Object -First 1 Version, BuildNumber
        if ($o -and $o.Version) { $os = $o.Version.Trim() }
        elseif ($o -and $o.BuildNumber) { $os = $o.BuildNumber.Trim() }
    } catch {}
    return [pscustomobject]@{
        Cpu       = $cpuName
        Mainboard = $board
        Bios      = $bios
        OsBuild   = $os
    }
}

function Parse-PciIdsFromInstanceId {
    param([AllowNull()][string]$InstanceId)
    $ven = $null; $dev = $null; $subsys = $null; $rev = $null
    if ($InstanceId) {
        if ($InstanceId -match 'VEN_([0-9A-Fa-f]{4})') { $ven = $Matches[1].ToUpperInvariant() }
        if ($InstanceId -match 'DEV_([0-9A-Fa-f]{4})') { $dev = $Matches[1].ToUpperInvariant() }
        if ($InstanceId -match 'SUBSYS_([0-9A-Fa-f]{8})') { $subsys = $Matches[1].ToUpperInvariant() }
        if ($InstanceId -match 'REV_([0-9A-Fa-f]{2})') { $rev = $Matches[1].ToUpperInvariant() }
    }
    return [pscustomobject]@{ Vendor = $ven; Device = $dev; Subsys = $subsys; Revision = $rev }
}

function Get-VgaInventory {
    $cimList = @(Get-RawCimDisplayDevices)
    $script:LastSourceTimestamps['cim_utc'] = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    $pnpList = @(Get-RawPnpDisplayDevices)
    $script:LastSourceTimestamps['pnp_utc'] = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

    $cimById = @{}
    foreach ($c in $cimList) {
        if ($c -and $c.PNPDeviceID) {
            $cimById[$c.PNPDeviceID.ToUpperInvariant()] = $c
        }
    }

    $seenIds = @{}
    $records = @()

    foreach ($p in $pnpList) {
        if (-not $p -or -not $p.InstanceId) { continue }
        $key = $p.InstanceId.ToUpperInvariant()
        $seenIds[$key] = $true
        $cim = if ($cimById.ContainsKey($key)) { $cimById[$key] } else { $null }
        $propMap = @{}
        $propError = $false
        try {
            $props = @(Get-RawPnpDeviceProperties -InstanceId $p.InstanceId)
            foreach ($pr in $props) {
                if ($pr -and $pr.KeyName) { $propMap[$pr.KeyName] = $pr.Data }
            }
        } catch {
            $propError = $true
        }
        $records += [pscustomobject]@{
            InstanceId = $p.InstanceId
            Pnp        = $p
            Cim        = $cim
            Props      = $propMap
            PropError  = $propError
        }
    }

    foreach ($c in $cimList) {
        if (-not $c -or -not $c.PNPDeviceID) { continue }
        $key = $c.PNPDeviceID.ToUpperInvariant()
        if (-not $seenIds.ContainsKey($key)) {
            $records += [pscustomobject]@{
                InstanceId = $c.PNPDeviceID
                Pnp        = $null
                Cim        = $c
                Props      = @{}
                PropError  = $false
            }
        }
    }

    $gpus = @()
    foreach ($rec in $records) {
        $ids = Parse-PciIdsFromInstanceId -InstanceId $rec.InstanceId
        $propMap = $rec.Props
        $propErr = [bool]$rec.PropError

        $nameVal = $null; $nameSrc = 'pnp'
        if ($rec.Cim -and -not [string]::IsNullOrWhiteSpace($rec.Cim.Name)) {
            $nameVal = $rec.Cim.Name
            $nameSrc = 'cim'
        } elseif ($rec.Pnp -and -not [string]::IsNullOrWhiteSpace($rec.Pnp.FriendlyName)) {
            $nameVal = $rec.Pnp.FriendlyName
            $nameSrc = 'pnp'
        }

        $pnpStatusVal = $null
        if ($rec.Pnp -and $rec.Pnp.Status) { $pnpStatusVal = [string]$rec.Pnp.Status }
        elseif ($rec.Cim -and $rec.Cim.Status) { $pnpStatusVal = [string]$rec.Cim.Status }

        try {
            $bdfStr = $null
            if ($propMap.ContainsKey('DEVPKEY_Device_BusNumber') -and $null -ne $propMap['DEVPKEY_Device_BusNumber'] -and $propMap.ContainsKey('DEVPKEY_Device_Address') -and $null -ne $propMap['DEVPKEY_Device_Address']) {
                $busNum = [int]$propMap['DEVPKEY_Device_BusNumber']
                $addrVal = [int]$propMap['DEVPKEY_Device_Address']
                $devNum = ($addrVal -shr 16) -band 0x1F
                $fnNum = $addrVal -band 0x07
                $bdfStr = ('0000:{0:x2}:{1:x2}.{2}' -f $busNum, $devNum, $fnNum)
            } elseif ($propMap.ContainsKey('DEVPKEY_Device_LocationInfo') -and $propMap['DEVPKEY_Device_LocationInfo'] -match '(\d+)\D+(\d+)\D+(\d+)') {
                $bdfStr = ('0000:{0:x2}:{1:x2}.{2}' -f [int]$Matches[1], [int]$Matches[2], [int]$Matches[3])
            }

            $drvVal = $null; $drvSrc = 'cim'
            if ($rec.Cim -and -not [string]::IsNullOrWhiteSpace($rec.Cim.DriverVersion)) {
                $drvVal = [string]$rec.Cim.DriverVersion
                $drvSrc = 'cim'
            } elseif ($propMap.ContainsKey('DEVPKEY_Device_DriverVersion') -and -not [string]::IsNullOrWhiteSpace([string]$propMap['DEVPKEY_Device_DriverVersion'])) {
                $drvVal = [string]$propMap['DEVPKEY_Device_DriverVersion']
                $drvSrc = 'pnp'
            }

            $maxGen = if ($propMap.ContainsKey('DEVPKEY_PciDevice_MaxLinkSpeed') -and $null -ne $propMap['DEVPKEY_PciDevice_MaxLinkSpeed']) { [int]$propMap['DEVPKEY_PciDevice_MaxLinkSpeed'] } else { $null }
            $curGen = if ($propMap.ContainsKey('DEVPKEY_PciDevice_CurrentLinkSpeed') -and $null -ne $propMap['DEVPKEY_PciDevice_CurrentLinkSpeed']) { [int]$propMap['DEVPKEY_PciDevice_CurrentLinkSpeed'] } else { $null }
            $maxW = if ($propMap.ContainsKey('DEVPKEY_PciDevice_MaxLinkWidth') -and $null -ne $propMap['DEVPKEY_PciDevice_MaxLinkWidth']) { [int]$propMap['DEVPKEY_PciDevice_MaxLinkWidth'] } else { $null }
            $curW = if ($propMap.ContainsKey('DEVPKEY_PciDevice_CurrentLinkWidth') -and $null -ne $propMap['DEVPKEY_PciDevice_CurrentLinkWidth']) { [int]$propMap['DEVPKEY_PciDevice_CurrentLinkWidth'] } else { $null }

            if ($curGen -eq 0) { $curGen = $null }
            if ($maxGen -eq 0) { $maxGen = $null }
            if ($curW -eq 0) { $curW = $null }
            if ($maxW -eq 0) { $maxW = $null }

            $drvProvider = if ($propMap.ContainsKey('DEVPKEY_Device_DriverProvider') -and -not [string]::IsNullOrWhiteSpace([string]$propMap['DEVPKEY_Device_DriverProvider'])) { [string]$propMap['DEVPKEY_Device_DriverProvider'] } else { $null }
            $drvDateStr = $null
            if ($propMap.ContainsKey('DEVPKEY_Device_DriverDate') -and $null -ne $propMap['DEVPKEY_Device_DriverDate']) {
                $rawDate = $propMap['DEVPKEY_Device_DriverDate']
                if ($rawDate -is [datetime]) {
                    $drvDateStr = $rawDate.ToString('yyyy-MM-dd')
                } elseif ($rawDate -is [DateTimeOffset]) {
                    $drvDateStr = $rawDate.DateTime.ToString('yyyy-MM-dd')
                } else {
                    $parsedDate = [datetime]::MinValue
                    if ([datetime]::TryParse([string]$rawDate, [ref]$parsedDate)) {
                        $drvDateStr = $parsedDate.ToString('yyyy-MM-dd')
                    }
                }
            }
            $infName = $null
            if ($propMap.ContainsKey('DEVPKEY_Device_DriverInfPath') -and -not [string]::IsNullOrWhiteSpace([string]$propMap['DEVPKEY_Device_DriverInfPath'])) {
                $infName = [System.IO.Path]::GetFileName(([string]$propMap['DEVPKEY_Device_DriverInfPath']).Trim())
            }
            $drvService = if ($propMap.ContainsKey('DEVPKEY_Device_Service') -and -not [string]::IsNullOrWhiteSpace([string]$propMap['DEVPKEY_Device_Service'])) { [string]$propMap['DEVPKEY_Device_Service'] } else { $null }
            $problemCode = $null; $problemSrc = 'pnp'
            if ($propMap.ContainsKey('DEVPKEY_Device_ProblemCode') -and $null -ne $propMap['DEVPKEY_Device_ProblemCode']) {
                $problemCode = [int]$propMap['DEVPKEY_Device_ProblemCode']
            } elseif ($rec.Cim -and $null -ne $rec.Cim.ConfigManagerErrorCode) {
                $problemCode = [int]$rec.Cim.ConfigManagerErrorCode
                $problemSrc = 'cim'
            }
            $locPath = $null
            if ($propMap.ContainsKey('DEVPKEY_Device_LocationPaths') -and $null -ne $propMap['DEVPKEY_Device_LocationPaths']) {
                $lpRaw = @($propMap['DEVPKEY_Device_LocationPaths']) | Where-Object { -not [string]::IsNullOrWhiteSpace([string]$_) } | Select-Object -First 1
                if ($lpRaw) { $locPath = [string]$lpRaw }
            }

            $bdfField = if ($propErr -and $null -eq $bdfStr) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $bdfStr -Source 'pnp' }
            $pcieStatus = if ($propErr) { 'error' } else { 'ok' }
            $drvDetailStatus = if ($propErr) { 'error' } else { 'ok' }

            $gpus += [pscustomobject]@{
                bdf            = $bdfField
                vendor_id      = New-ReportField -Value $ids.Vendor -Source 'pnp'
                device_id      = New-ReportField -Value $ids.Device -Source 'pnp'
                subsystem_id   = New-ReportField -Value $ids.Subsys -Source 'pnp'
                revision       = New-ReportField -Value $ids.Revision -Source 'pnp'
                name           = New-ReportField -Value $nameVal -Source $nameSrc
                pnp_status     = New-ReportField -Value $pnpStatusVal -Source 'pnp'
                driver_version = if ($propErr -and $null -eq $drvVal) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $drvVal -Source $drvSrc }
                profile        = Get-GpuProfileDetails -VendorId $ids.Vendor -DeviceId $ids.Device -Name $nameVal
                driver         = [ordered]@{
                    provider      = if ($propErr -and $null -eq $drvProvider) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $drvProvider -Source 'pnp' }
                    date          = if ($propErr -and $null -eq $drvDateStr) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $drvDateStr -Source 'pnp' }
                    inf_name      = if ($propErr -and $null -eq $infName) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $infName -Source 'pnp' }
                    service       = if ($propErr -and $null -eq $drvService) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $drvService -Source 'pnp' }
                    problem_code  = if ($propErr -and $null -eq $problemCode) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $problemCode -Source $problemSrc }
                    location_path = if ($propErr -and $null -eq $locPath) { New-ReportField -Value $null -Source 'pnp' -Status 'error' } else { New-ReportField -Value $locPath -Source 'pnp' }
                }
                pcie           = [ordered]@{
                    max_gen       = if ($propErr -and $null -eq $maxGen) { New-ReportField -Value $null -Source 'pnp' -Unit 'gen' -Status $pcieStatus } else { New-ReportField -Value $maxGen -Source 'pnp' -Unit 'gen' }
                    current_gen   = if ($propErr -and $null -eq $curGen) { New-ReportField -Value $null -Source 'pnp' -Unit 'gen' -Status $pcieStatus } else { New-ReportField -Value $curGen -Source 'pnp' -Unit 'gen' }
                    max_width     = if ($propErr -and $null -eq $maxW) { New-ReportField -Value $null -Source 'pnp' -Unit 'lanes' -Status $pcieStatus } else { New-ReportField -Value $maxW -Source 'pnp' -Unit 'lanes' }
                    current_width = if ($propErr -and $null -eq $curW) { New-ReportField -Value $null -Source 'pnp' -Unit 'lanes' -Status $pcieStatus } else { New-ReportField -Value $curW -Source 'pnp' -Unit 'lanes' }
                }
                nvidia         = $null
            }
        } catch {
            $gpus += [pscustomobject]@{
                bdf            = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                vendor_id      = New-ReportField -Value $ids.Vendor -Source 'pnp'
                device_id      = New-ReportField -Value $ids.Device -Source 'pnp'
                subsystem_id   = New-ReportField -Value $ids.Subsys -Source 'pnp'
                revision       = New-ReportField -Value $ids.Revision -Source 'pnp'
                name           = New-ReportField -Value $nameVal -Source $nameSrc
                pnp_status     = New-ReportField -Value $pnpStatusVal -Source 'pnp'
                driver_version = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                profile        = Get-GpuProfileDetails -VendorId $ids.Vendor -DeviceId $ids.Device -Name $nameVal
                driver         = [ordered]@{
                    provider      = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                    date          = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                    inf_name      = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                    service       = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                    problem_code  = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                    location_path = New-ReportField -Value $null -Source 'pnp' -Status 'error'
                }
                pcie           = [ordered]@{
                    max_gen       = New-ReportField -Value $null -Source 'pnp' -Unit 'gen' -Status 'error'
                    current_gen   = New-ReportField -Value $null -Source 'pnp' -Unit 'gen' -Status 'error'
                    max_width     = New-ReportField -Value $null -Source 'pnp' -Unit 'lanes' -Status 'error'
                    current_width = New-ReportField -Value $null -Source 'pnp' -Unit 'lanes' -Status 'error'
                }
                nvidia         = $null
            }
        }
    }
    return $gpus
}

function Find-NvidiaSmiExecutable {
    $cmd = Get-Command -Name 'nvidia-smi.exe' -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source) { return $cmd.Source }
    $sysCandidate = Join-Path -Path $env:SystemRoot -ChildPath 'System32\nvidia-smi.exe'
    if (Test-Path -LiteralPath $sysCandidate) { return $sysCandidate }
    $nvSmiCandidate = Join-Path -Path $env:ProgramFiles -ChildPath 'NVIDIA Corporation\NVSMI\nvidia-smi.exe'
    if (Test-Path -LiteralPath $nvSmiCandidate) { return $nvSmiCandidate }
    return $null
}

function Invoke-NvidiaSmiRaw {
    param([string[]]$Arguments)
    $exe = Find-NvidiaSmiExecutable
    if (-not $exe) {
        return [pscustomobject]@{ ExitCode = -1; Output = '' }
    }
    $out = & $exe @Arguments 2>&1 | Out-String
    return [pscustomobject]@{ ExitCode = $LASTEXITCODE; Output = $out }
}

function Get-NvidiaSupportedFields {
    $exe = Find-NvidiaSmiExecutable
    if (-not $exe) { return @() }
    $res = Invoke-NvidiaSmiRaw -Arguments @('--help-query-gpu')
    if (-not $res -or $res.ExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($res.Output)) {
        return @()
    }
    $supported = New-Object System.Collections.Generic.HashSet[string]
    $lines = $res.Output -split "`r?`n"
    foreach ($line in $lines) {
        $trimmed = $line.Trim()
        if ($trimmed.StartsWith('"')) {
            foreach ($m in [regex]::Matches($trimmed, '"([a-z0-9_.]+)"')) {
                [void]$supported.Add($m.Groups[1].Value)
            }
        } elseif ($trimmed -match '^([a-z0-9_.]+)\s*$') {
            [void]$supported.Add($Matches[1])
        }
    }
    return @($supported)
}

function Convert-NvidiaCellToField {
    param([string]$FieldName, [AllowNull()][string]$RawValue)
    $unit = if ($script:NumericNvidiaFields.ContainsKey($FieldName)) { $script:NumericNvidiaFields[$FieldName] } else { $null }
    if ($null -eq $RawValue) {
        return New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status 'unsupported'
    }
    $t = $RawValue.Trim()
    if ($t -eq '' -or $t -eq '[N/A]' -or $t -eq '[Not Supported]' -or $t -eq 'N/A' -or $t -eq '[Unknown Error]') {
        return New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status 'unsupported'
    }
    if ($FieldName -eq 'pci.bus_id') {
        $t = Normalize-Bdf -BusId $t
    }
    if ($script:NumericNvidiaFields.ContainsKey($FieldName)) {
        $num = 0.0
        if ([double]::TryParse($t, [System.Globalization.NumberStyles]::Float, [System.Globalization.CultureInfo]::InvariantCulture, [ref]$num)) {
            if ($FieldName -in @('memory.total','pcie.link.gen.max','pcie.link.gen.gpucurrent','pcie.link.width.max','pcie.link.width.current','clocks.sm','clocks.mem','clocks.max.sm','fan.speed','utilization.gpu','utilization.memory','memory.used','memory.free')) {
                return New-ReportField -Value ([int][Math]::Round($num)) -Source 'nvidia-smi' -Unit $unit -Status 'ok'
            }
            return New-ReportField -Value $num -Source 'nvidia-smi' -Unit $unit -Status 'ok'
        }
        return New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status 'unsupported'
    }
    return New-ReportField -Value $t -Source 'nvidia-smi' -Unit $unit -Status 'ok'
}

function New-EmptyNvidiaExtendedBlock {
    param([string]$Status = 'unsupported')
    $extended = [ordered]@{}
    $extMissing = New-Object 'System.Collections.Generic.List[string]'
    foreach ($k in $script:NvidiaExtendedFields) {
        $unit = if ($script:NumericNvidiaFields.ContainsKey($k)) { $script:NumericNvidiaFields[$k] } else { $null }
        $extended[$k] = New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status $Status
        [void]$extMissing.Add($k)
    }
    return [ordered]@{
        extended         = $extended
        extended_missing = $extMissing.ToArray()
    }
}

function Build-NvidiaExtendedFromMap {
    param([hashtable]$RawMap)
    $extended = [ordered]@{}
    $extMissing = New-Object 'System.Collections.Generic.List[string]'
    foreach ($k in $script:NvidiaExtendedFields) {
        if ($RawMap.ContainsKey($k)) {
            $f = Convert-NvidiaCellToField -FieldName $k -RawValue $RawMap[$k]
        } else {
            $unit = if ($script:NumericNvidiaFields.ContainsKey($k)) { $script:NumericNvidiaFields[$k] } else { $null }
            $f = New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status 'unsupported'
        }
        $extended[$k] = $f
        if ($f.status -ne 'ok') {
            [void]$extMissing.Add($k)
        }
    }
    return [ordered]@{
        extended         = $extended
        extended_missing = $extMissing.ToArray()
    }
}

function New-EmptyNvidiaBlock {
    param([string]$Status = 'skipped')
    $fields = [ordered]@{}
    $missing = New-Object 'System.Collections.Generic.List[string]'
    foreach ($k in $script:NvidiaRequiredFields) {
        $unit = if ($script:NumericNvidiaFields.ContainsKey($k)) { $script:NumericNvidiaFields[$k] } else { $null }
        $fields[$k] = New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status $Status
        [void]$missing.Add($k)
    }
    $extObj = New-EmptyNvidiaExtendedBlock -Status $Status
    return [ordered]@{
        complete         = $false
        missing          = $missing.ToArray()
        fields           = $fields
        extended         = $extObj.extended
        extended_missing = $extObj.extended_missing
    }
}

function Build-NvidiaBlockFromMap {
    param([hashtable]$RawMap)
    $fields = [ordered]@{}
    $missing = New-Object 'System.Collections.Generic.List[string]'
    foreach ($k in $script:NvidiaRequiredFields) {
        if ($RawMap.ContainsKey($k)) {
            $f = Convert-NvidiaCellToField -FieldName $k -RawValue $RawMap[$k]
        } else {
            $unit = if ($script:NumericNvidiaFields.ContainsKey($k)) { $script:NumericNvidiaFields[$k] } else { $null }
            $f = New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status 'unsupported'
        }
        $fields[$k] = $f
        if ($f.status -ne 'ok') {
            [void]$missing.Add($k)
        }
    }
    return [ordered]@{
        complete = ($missing.Count -eq 0)
        missing  = $missing.ToArray()
        fields   = $fields
    }
}

function Get-GpuProfileDetails {
    param(
        [AllowNull()][string]$VendorId,
        [AllowNull()][string]$DeviceId,
        [AllowNull()][string]$Name = $null
    )
    if (-not [string]::IsNullOrWhiteSpace($VendorId) -and -not [string]::IsNullOrWhiteSpace($DeviceId)) {
        $key = '{0}:{1}' -f $VendorId.Trim().ToUpperInvariant(), $DeviceId.Trim().ToUpperInvariant()
        if ($script:GpuProfileDetails.Contains($key)) {
            $src = $script:GpuProfileDetails[$key]
            return [ordered]@{
                id                = $src.id
                name              = $src.name
                family            = $src.family
                category          = $src.category
                target_pcie_gen   = $src.target_pcie_gen
                tensor_applicable = $src.tensor_applicable
                identity_note     = $src.identity_note
            }
        }
    }
    $note = $null
    if (-not [string]::IsNullOrWhiteSpace($Name) -and $Name -match '(?i)\bP100\b') {
        $note = 'Tên báo P100 nhưng PCI ID chưa khớp Tesla GP100; cần xác minh trước khi áp thông số Tesla'
    }
    return [ordered]@{
        id                = $null
        name              = $null
        family            = $null
        category          = 'unknown'
        target_pcie_gen   = $null
        tensor_applicable = 'unknown'
        identity_note     = $note
    }
}

function Get-GpuProfileName {
    param(
        [AllowNull()][string]$VendorId,
        [AllowNull()][string]$DeviceId
    )
    $details = Get-GpuProfileDetails -VendorId $VendorId -DeviceId $DeviceId
    return $details.name
}

function Select-TargetGpu {
    param(
        [Parameter(Mandatory = $false)][object[]]$Inventory,
        [AllowNull()][string]$TargetBdf = $null
    )
    $invArray = @($Inventory | Where-Object { $null -ne $_ })
    if ($invArray.Count -eq 0) { return $null }

    if (-not [string]::IsNullOrWhiteSpace($TargetBdf) -and $TargetBdf.Trim().ToLowerInvariant() -ne 'auto') {
        $wantedBdf = Normalize-Bdf -BusId $TargetBdf
        foreach ($gpu in $invArray) {
            $gpuBdf = if ($gpu.bdf -and $gpu.bdf.status -eq 'ok') { Normalize-Bdf -BusId ([string]$gpu.bdf.value) } else { $null }
            if ($gpuBdf -and $gpuBdf -eq $wantedBdf) {
                return $gpu
            }
        }
    }

    foreach ($gpu in $invArray) {
        $vid = if ($gpu.vendor_id -and $gpu.vendor_id.status -eq 'ok') { [string]$gpu.vendor_id.value } else { $null }
        $did = if ($gpu.device_id -and $gpu.device_id.status -eq 'ok') { [string]$gpu.device_id.value } else { $null }
        if (Get-GpuProfileName -VendorId $vid -DeviceId $did) {
            return $gpu
        }
    }

    foreach ($gpu in $invArray) {
        $vid = if ($gpu.vendor_id -and $gpu.vendor_id.status -eq 'ok') { [string]$gpu.vendor_id.value } else { $null }
        if ($vid -and $vid.Trim().ToUpperInvariant() -eq '10DE') {
            return $gpu
        }
    }

    return $invArray[0]
}

function Select-TargetNvidiaBlock {
    param(
        [object[]]$Inventory,
        [hashtable]$BlocksByBdf,
        [AllowNull()][string]$TargetBdf = $null
    )
    $targetGpu = Select-TargetGpu -Inventory $Inventory -TargetBdf $TargetBdf
    if ($targetGpu) {
        $bdf = if ($targetGpu.bdf -and $targetGpu.bdf.status -eq 'ok') { Normalize-Bdf -BusId ([string]$targetGpu.bdf.value) } else { $null }
        if ($bdf -and $BlocksByBdf.ContainsKey($bdf)) {
            return $BlocksByBdf[$bdf]
        }
        $vid = if ($targetGpu.vendor_id -and $targetGpu.vendor_id.status -eq 'ok') { ([string]$targetGpu.vendor_id.value).Trim().ToUpperInvariant() } else { $null }
        if ($vid -eq '10DE') {
            return New-EmptyNvidiaBlock -Status 'unsupported'
        }
        return New-EmptyNvidiaBlock -Status 'skipped'
    }
    foreach ($key in $BlocksByBdf.Keys) {
        return $BlocksByBdf[$key]
    }
    return New-EmptyNvidiaBlock -Status 'unsupported'
}

function Add-NvidiaRowsToMap {
    param(
        [string]$Output,
        [string[]]$QueryFields,
        [hashtable]$RowsByBdf
    )
    if ([string]::IsNullOrWhiteSpace($Output)) { return }
    $lines = @($Output -split "`r?`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    foreach ($line in $lines) {
        $cols = $line -split ',\s*'
        if ($cols.Count -eq $QueryFields.Count) {
            $rowMap = @{}
            for ($i = 0; $i -lt $QueryFields.Count; $i += 1) {
                $rowMap[$QueryFields[$i]] = $cols[$i]
            }
            $bdfKey = Normalize-Bdf -BusId $rowMap['pci.bus_id']
            if ($bdfKey) {
                $RowsByBdf[$bdfKey] = $rowMap
            }
        }
    }
}

function Get-NvidiaInfo {
    param(
        [Parameter(Mandatory = $false)]$Inventory,
        [AllowNull()][string]$TargetBdf = $null
    )
    $invArray = @($Inventory)
    $hasNvidia = $false
    foreach ($g in $invArray) {
        if ($g -and $g.vendor_id -and $g.vendor_id.value -eq '10DE') {
            $hasNvidia = $true
            break
        }
    }
    if (-not $hasNvidia) {
        $script:LastSourceTimestamps['nvidia_smi_utc'] = $null
        return New-EmptyNvidiaBlock -Status 'skipped'
    }

    $exe = Find-NvidiaSmiExecutable
    if (-not $exe) {
        $script:LastSourceTimestamps['nvidia_smi_utc'] = $null
        return New-EmptyNvidiaBlock -Status 'skipped'
    }

    $rowsByBdf = @{}
    $supported = @()
    $allFields = @($script:NvidiaRequiredFields)
    $combinedRes = Invoke-NvidiaSmiRaw -Arguments @("--query-gpu=$($allFields -join ',')", '--format=csv,noheader,nounits')
    if ($combinedRes -and $combinedRes.ExitCode -eq 0 -and -not [string]::IsNullOrWhiteSpace($combinedRes.Output)) {
        Add-NvidiaRowsToMap -Output $combinedRes.Output -QueryFields $allFields -RowsByBdf $rowsByBdf
    } else {
        $supported = @(Get-NvidiaSupportedFields)
        $queryFields = if ($supported.Count -gt 0) {
            @($script:NvidiaRequiredFields | Where-Object { $supported -contains $_ })
        } else {
            @($script:NvidiaRequiredFields)
        }
        if ($queryFields -notcontains 'pci.bus_id') {
            $queryFields = @('pci.bus_id') + $queryFields
        }
        $filteredRes = Invoke-NvidiaSmiRaw -Arguments @("--query-gpu=$($queryFields -join ',')", '--format=csv,noheader,nounits')
        if ($filteredRes -and $filteredRes.ExitCode -eq 0 -and -not [string]::IsNullOrWhiteSpace($filteredRes.Output)) {
            Add-NvidiaRowsToMap -Output $filteredRes.Output -QueryFields $queryFields -RowsByBdf $rowsByBdf
        } else {
            # Truy vấn lại từng trường để không mất dữ liệu còn lại khi driver cũ thiếu một trường
            foreach ($field in $script:NvidiaRequiredFields) {
                $pairFields = if ($field -eq 'pci.bus_id') { @('pci.bus_id') } else { @('pci.bus_id', $field) }
                $singleRes = Invoke-NvidiaSmiRaw -Arguments @("--query-gpu=$($pairFields -join ',')", '--format=csv,noheader,nounits')
                if ($singleRes -and $singleRes.ExitCode -eq 0 -and -not [string]::IsNullOrWhiteSpace($singleRes.Output)) {
                    $lines = @($singleRes.Output -split "`r?`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
                    foreach ($line in $lines) {
                        $cols = $line -split ',\s*'
                        if ($cols.Count -eq $pairFields.Count) {
                            $bdfKey = Normalize-Bdf -BusId $cols[0]
                            if ($bdfKey) {
                                if (-not $rowsByBdf.ContainsKey($bdfKey)) {
                                    $rowsByBdf[$bdfKey] = @{ 'pci.bus_id' = $cols[0] }
                                }
                                if ($field -ne 'pci.bus_id') {
                                    $rowsByBdf[$bdfKey][$field] = $cols[1]
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    $script:LastSourceTimestamps['nvidia_smi_utc'] = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

    if ($rowsByBdf.Count -eq 0) {
        $targetOnEmpty = Select-TargetGpu -Inventory $invArray -TargetBdf $TargetBdf
        if ($targetOnEmpty -and $targetOnEmpty.vendor_id -and $targetOnEmpty.vendor_id.value -ne '10DE') {
            return New-EmptyNvidiaBlock -Status 'skipped'
        }
        return New-EmptyNvidiaBlock -Status 'error'
    }

    if ($supported.Count -eq 0) {
        $supported = @(Get-NvidiaSupportedFields)
    }

    $extQueryNames = New-Object 'System.Collections.Generic.List[string]'
    $queryToCanonical = @{}
    foreach ($cf in $script:NvidiaExtendedFields) {
        if ($supported.Count -gt 0) {
            if ($supported -contains $cf) {
                [void]$extQueryNames.Add($cf)
                $queryToCanonical[$cf] = $cf
            } elseif ($cf.StartsWith('clocks_throttle_reasons.')) {
                $alt = $cf.Replace('clocks_throttle_reasons.', 'clocks_event_reasons.')
                if ($supported -contains $alt) {
                    [void]$extQueryNames.Add($alt)
                    $queryToCanonical[$alt] = $cf
                }
            }
        } else {
            [void]$extQueryNames.Add($cf)
            $queryToCanonical[$cf] = $cf
        }
    }

    $extRowsByBdf = @{}
    if ($extQueryNames.Count -gt 0) {
        $extQueryFull = @('pci.bus_id') + $extQueryNames.ToArray()
        $extRes = Invoke-NvidiaSmiRaw -Arguments @("--query-gpu=$($extQueryFull -join ',')", '--format=csv,noheader,nounits')
        $tempExt = @{}
        if ($extRes -and $extRes.ExitCode -eq 0 -and -not [string]::IsNullOrWhiteSpace($extRes.Output)) {
            Add-NvidiaRowsToMap -Output $extRes.Output -QueryFields $extQueryFull -RowsByBdf $tempExt
        } else {
            foreach ($qf in $extQueryNames) {
                $pair = @('pci.bus_id', $qf)
                $sRes = Invoke-NvidiaSmiRaw -Arguments @("--query-gpu=$($pair -join ',')", '--format=csv,noheader,nounits')
                if ($sRes -and $sRes.ExitCode -eq 0 -and -not [string]::IsNullOrWhiteSpace($sRes.Output)) {
                    $lines = @($sRes.Output -split "`r?`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
                    foreach ($line in $lines) {
                        $cols = $line -split ',\s*'
                        if ($cols.Count -eq 2) {
                            $bk = Normalize-Bdf -BusId $cols[0]
                            if ($bk -and $bk -match '^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]$') {
                                if (-not $tempExt.ContainsKey($bk)) { $tempExt[$bk] = @{} }
                                $tempExt[$bk][$qf] = $cols[1]
                            }
                        }
                    }
                }
            }
        }
        foreach ($bk in $tempExt.Keys) {
            if ($bk -match '^[0-9a-f]{4}:[0-9a-f]{2}:[0-9a-f]{2}\.[0-7]$' -and $rowsByBdf.ContainsKey($bk)) {
                $mapped = @{}
                foreach ($qk in $tempExt[$bk].Keys) {
                    if ($queryToCanonical.ContainsKey($qk)) {
                        $mapped[$queryToCanonical[$qk]] = $tempExt[$bk][$qk]
                    }
                }
                $extRowsByBdf[$bk] = $mapped
            }
        }
    }

    $blocksByBdf = @{}
    foreach ($bdfKey in $rowsByBdf.Keys) {
        $blk = Build-NvidiaBlockFromMap -RawMap $rowsByBdf[$bdfKey]
        $extRaw = if ($extRowsByBdf.ContainsKey($bdfKey)) { $extRowsByBdf[$bdfKey] } else { @{} }
        $extBuilt = Build-NvidiaExtendedFromMap -RawMap $extRaw
        $blk['extended'] = $extBuilt.extended
        $blk['extended_missing'] = $extBuilt.extended_missing
        $blocksByBdf[$bdfKey] = $blk
    }

    foreach ($gpu in $invArray) {
        if ($gpu -and $gpu.bdf -and $gpu.bdf.value) {
            $normBdf = Normalize-Bdf -BusId ([string]$gpu.bdf.value)
            if ($normBdf -and $blocksByBdf.ContainsKey($normBdf)) {
                $gpu | Add-Member -MemberType NoteProperty -Name 'nvidia' -Value $blocksByBdf[$normBdf] -Force
            }
        }
    }

    return Select-TargetNvidiaBlock -Inventory $invArray -BlocksByBdf $blocksByBdf -TargetBdf $TargetBdf
}

function Get-RawSecurityState {
    $bootMode = $null
    if (-not [string]::IsNullOrWhiteSpace($env:firmware_type)) {
        $fw = $env:firmware_type.Trim().ToUpperInvariant()
        if ($fw -eq 'UEFI') { $bootMode = 'UEFI' }
        elseif ($fw -eq 'LEGACY') { $bootMode = 'Legacy' }
    }
    if (-not $bootMode) {
        if (Test-Path -LiteralPath 'HKLM:\SYSTEM\CurrentControlSet\Control\SecureBoot\State') {
            $bootMode = 'UEFI'
        } else {
            $bootMode = 'Legacy'
        }
    }

    $sbState = $null
    $sbStatus = 'unsupported'
    if ($bootMode -eq 'UEFI') {
        try {
            $sbVal = (Get-ItemProperty -LiteralPath 'HKLM:\SYSTEM\CurrentControlSet\Control\SecureBoot\State' -Name 'UEFISecureBootEnabled' -ErrorAction Stop).UEFISecureBootEnabled
            if ($null -ne $sbVal) {
                $sbState = [int]$sbVal
                $sbStatus = 'ok'
            }
        } catch [System.Security.SecurityException], [System.UnauthorizedAccessException] {
            $sbStatus = 'permission_denied'
        } catch {
            $sbStatus = 'unsupported'
        }
    }

    $hvciEnabled = $null
    $hvciStatus = 'unsupported'
    $hvciSource = 'cim'
    try {
        $dg = Get-CimInstance -Namespace 'root\Microsoft\Windows\DeviceGuard' -ClassName 'Win32_DeviceGuard' -ErrorAction Stop | Select-Object -First 1 SecurityServicesRunning
        if ($dg -and $null -ne $dg.SecurityServicesRunning) {
            $hvciEnabled = (@($dg.SecurityServicesRunning) -contains 2)
            $hvciStatus = 'ok'
            $hvciSource = 'cim'
        }
    } catch {
        $hvciSource = 'registry'
        try {
            $hvKey = 'HKLM:\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity'
            if (Test-Path -LiteralPath $hvKey) {
                $hvVal = (Get-ItemProperty -LiteralPath $hvKey -Name 'Enabled' -ErrorAction Stop).Enabled
                $hvciEnabled = ([int]$hvVal -eq 1)
                $hvciStatus = 'ok'
            } else {
                $hvciEnabled = $false
                $hvciStatus = 'ok'
            }
        } catch [System.Security.SecurityException], [System.UnauthorizedAccessException] {
            $hvciStatus = 'permission_denied'
        } catch {
            $hvciStatus = 'unsupported'
        }
    }

    $tsEnabled = $null
    $tsStatus = 'unsupported'
    try {
        $sso = (Get-ItemProperty -LiteralPath 'HKLM:\SYSTEM\CurrentControlSet\Control' -Name 'SystemStartOptions' -ErrorAction Stop).SystemStartOptions
        $tsEnabled = ([string]$sso -match '(?i)\bTESTSIGNING\b')
        $tsStatus = 'ok'
    } catch [System.Security.SecurityException], [System.UnauthorizedAccessException] {
        $tsStatus = 'permission_denied'
    } catch {
        $tsStatus = 'unsupported'
    }

    return [pscustomobject]@{
        BootMode         = $bootMode
        SecureBootState  = $sbState
        SecureBootStatus = $sbStatus
        HvciEnabled      = $hvciEnabled
        HvciStatus       = $hvciStatus
        HvciSource       = $hvciSource
        TestSigning      = $tsEnabled
        TestSigningStatus= $tsStatus
    }
}

function Get-SecurityAndBootInfo {
    $raw = Get-RawSecurityState
    $script:LastSourceTimestamps['security_utc'] = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

    $bootField = if ($raw -and -not [string]::IsNullOrWhiteSpace([string]$raw.BootMode)) {
        New-ReportField -Value ([string]$raw.BootMode) -Source 'windows_env'
    } else {
        New-ReportField -Value $null -Source 'windows_env' -Status 'unsupported'
    }

    $sbStatus = if ($raw -and $raw.SecureBootStatus) { [string]$raw.SecureBootStatus } elseif ($raw -and $null -ne $raw.SecureBootState) { 'ok' } else { 'unsupported' }
    if ($raw -and $raw.BootMode -eq 'Legacy') { $sbStatus = 'unsupported' }
    $sbField = if ($sbStatus -eq 'ok' -and $null -ne $raw.SecureBootState) {
        New-ReportField -Value ([int]$raw.SecureBootState -eq 1) -Source 'registry'
    } else {
        New-ReportField -Value $null -Source 'registry' -Status $sbStatus
    }

    $hvSrc = if ($raw -and $raw.HvciSource) { [string]$raw.HvciSource } else { 'cim' }
    $hvStatus = if ($raw -and $raw.HvciStatus) { [string]$raw.HvciStatus } elseif ($raw -and $null -ne $raw.HvciEnabled) { 'ok' } else { 'unsupported' }
    $hvField = if ($hvStatus -eq 'ok' -and $null -ne $raw.HvciEnabled) {
        New-ReportField -Value ([bool]$raw.HvciEnabled) -Source $hvSrc
    } else {
        New-ReportField -Value $null -Source $hvSrc -Status $hvStatus
    }

    $tsStatus = if ($raw -and $raw.TestSigningStatus) { [string]$raw.TestSigningStatus } elseif ($raw -and $null -ne $raw.TestSigning) { 'ok' } else { 'unsupported' }
    $tsField = if ($tsStatus -eq 'ok' -and $null -ne $raw.TestSigning) {
        New-ReportField -Value ([bool]$raw.TestSigning) -Source 'registry'
    } else {
        New-ReportField -Value $null -Source 'registry' -Status $tsStatus
    }

    return [ordered]@{
        boot_mode    = $bootField
        secure_boot  = $sbField
        hvci_running = $hvField
        test_signing = $tsField
    }
}

function Get-LegacyUnlockStatus {
    param([AllowNull()][string]$StatusDir = $null)
    $dir = if (-not [string]::IsNullOrWhiteSpace($StatusDir)) { $StatusDir } elseif (-not [string]::IsNullOrWhiteSpace($env:ProgramData)) { Join-Path $env:ProgramData '40HXUnlock' } else { $null }
    $empty = [ordered]@{
        present       = $false
        source        = 'none'
        status_code   = 'NONE'
        error_code    = 'NONE'
        speed_current = $null
        width_current = $null
        tls_target    = $null
        device_id     = $null
        mrrs_bytes    = $null
        observed_utc  = $null
    }
    if (-not $dir -or -not (Test-Path -LiteralPath $dir)) {
        return $empty
    }

    $candidates = @(
        @{ Name = 'gen2_status.txt'; Source = 'gen2_status' },
        @{ Name = 'cmp30hx_batch_status.txt'; Source = 'batch_status' }
    )
    $chosenFile = $null
    $chosenSource = 'none'
    foreach ($c in $candidates) {
        $p = Join-Path $dir $c.Name
        if (Test-Path -LiteralPath $p -PathType Leaf) {
            $chosenFile = $p
            $chosenSource = $c.Source
            break
        }
    }
    if (-not $chosenFile) {
        return $empty
    }

    $allowedCodes = @('GEN2_SUCCESS', 'GEN2_SKIPPED', 'GEN1_STUCK', 'GEN2_HARDWARE_LIMIT', 'DRV_FAIL', 'NO_GPU', 'UNKNOWN', 'NONE')
    $res = [ordered]@{
        present       = $true
        source        = $chosenSource
        status_code   = 'UNKNOWN'
        error_code    = 'NONE'
        speed_current = $null
        width_current = $null
        tls_target    = $null
        device_id     = $null
        mrrs_bytes    = $null
        observed_utc  = $null
    }

    try {
        $rawLines = [System.IO.File]::ReadAllLines($chosenFile, [System.Text.Encoding]::UTF8)
        foreach ($rawLine in $rawLines) {
            $line = $rawLine.Trim()
            if ([string]::IsNullOrEmpty($line) -or $line.IndexOf('=') -le 0) { continue }
            $eqIdx = $line.IndexOf('=')
            $k = $line.Substring(0, $eqIdx).Trim()
            $v = $line.Substring($eqIdx + 1).Trim()
            switch ($k) {
                'STATUS_CODE' {
                    $up = $v.ToUpperInvariant()
                    if ($allowedCodes -contains $up) { $res.status_code = $up }
                }
                'ERROR_CODE' {
                    $up = $v.ToUpperInvariant()
                    if ($up -match '^[A-Z0-9_]{1,64}$') { $res.error_code = $up }
                }
                'SPEED_CURRENT' {
                    $iv = 0
                    if ([int]::TryParse($v, [ref]$iv) -and $iv -ge 0) { $res.speed_current = $iv }
                }
                'WIDTH_CURRENT' {
                    $iv = 0
                    if ([int]::TryParse($v, [ref]$iv) -and $iv -ge 0) { $res.width_current = $iv }
                }
                'TLS_TARGET' {
                    $iv = 0
                    if ([int]::TryParse($v, [ref]$iv) -and $iv -ge 0) { $res.tls_target = $iv }
                }
                'MRRS_BYTES' {
                    $iv = 0
                    if ([int]::TryParse($v, [ref]$iv) -and $iv -ge 0) { $res.mrrs_bytes = $iv }
                }
                'DEVICE_ID' {
                    if ($v -match '^[0-9A-Fa-f]{4}$') { $res.device_id = $v.ToUpperInvariant() }
                }
                'OBSERVED_AT' {
                    $dto = [DateTimeOffset]::MinValue
                    if ([DateTimeOffset]::TryParse($v, [System.Globalization.CultureInfo]::InvariantCulture, [System.Globalization.DateTimeStyles]::AssumeUniversal, [ref]$dto)) {
                        $res.observed_utc = $dto.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
                    }
                }
            }
        }
    } catch {
        return $empty
    }
    return $res
}

function Get-OrCreateExperimentId {
    param(
        [AllowNull()][string]$BaseDir,
        [switch]$KeepHistory
    )
    if (-not $KeepHistory -and $env:VGA_KEEP_HISTORY -ne '1') {
        return [guid]::NewGuid().ToString()
    }
    $dir = if (-not [string]::IsNullOrWhiteSpace($BaseDir)) { $BaseDir } else { $env:VGA_BAT_DIR }
    if (-not [string]::IsNullOrWhiteSpace($dir)) {
        try {
            $idPath = Join-Path -Path $dir -ChildPath 'experiment_id.txt'
            if (Test-Path -LiteralPath $idPath) {
                $existing = ([System.IO.File]::ReadAllText($idPath, [System.Text.Encoding]::UTF8)).Trim()
                $parsed = [guid]::Empty
                if ([guid]::TryParse($existing, [ref]$parsed)) {
                    return $parsed.ToString()
                }
            }
            [void][System.IO.Directory]::CreateDirectory($dir)
            $newId = [guid]::NewGuid().ToString()
            [System.IO.File]::WriteAllText($idPath, $newId, (New-Object System.Text.UTF8Encoding($false)))
            return $newId
        } catch {}
    }
    return [guid]::NewGuid().ToString()
}

function Get-PreviousReportId {
    param(
        [AllowNull()][string]$BaseDir,
        [switch]$KeepHistory
    )
    if (-not $KeepHistory -and $env:VGA_KEEP_HISTORY -ne '1') {
        return $null
    }
    $dir = if (-not [string]::IsNullOrWhiteSpace($BaseDir)) { $BaseDir } else { $env:VGA_BAT_DIR }
    if (-not [string]::IsNullOrWhiteSpace($dir)) {
        try {
            $prevPath = Join-Path -Path $dir -ChildPath 'last_report_id.txt'
            if (Test-Path -LiteralPath $prevPath) {
                $existing = ([System.IO.File]::ReadAllText($prevPath, [System.Text.Encoding]::UTF8)).Trim()
                $parsed = [guid]::Empty
                if ([guid]::TryParse($existing, [ref]$parsed)) {
                    return $parsed.ToString()
                }
            }
        } catch {}
    }
    return $null
}

function Save-LastReportId {
    param(
        [AllowNull()][string]$BaseDir,
        [Parameter(Mandatory = $true)][string]$ReportId,
        [switch]$KeepHistory
    )
    if (-not $KeepHistory -and $env:VGA_KEEP_HISTORY -ne '1') {
        return
    }
    $dir = if (-not [string]::IsNullOrWhiteSpace($BaseDir)) { $BaseDir } else { $env:VGA_BAT_DIR }
    if (-not [string]::IsNullOrWhiteSpace($dir)) {
        try {
            [void][System.IO.Directory]::CreateDirectory($dir)
            $prevPath = Join-Path -Path $dir -ChildPath 'last_report_id.txt'
            [System.IO.File]::WriteAllText($prevPath, $ReportId, (New-Object System.Text.UTF8Encoding($false)))
        } catch {}
    }
}

function Invoke-VgaBenchRaw {
    param(
        [Parameter(Mandatory = $true)][string]$TargetBdf,
        [int]$SizeMB = 64,
        [int]$Iterations = 5,
        [int]$TimeoutMs = 15000
    )
    $rootDir = if (-not [string]::IsNullOrWhiteSpace($env:VGA_BAT_DIR)) { $env:VGA_BAT_DIR } else { (Get-Location).Path }
    $exeCandidates = @(
        (Join-Path $rootDir 'windows-v3.0\release\vgabench.exe'),
        (Join-Path $rootDir 'windows-v3.0\tools\vgabench\vgabench.exe')
    )
    $exePath = $null
    foreach ($c in $exeCandidates) {
        if (Test-Path -LiteralPath $c -PathType Leaf) {
            $exePath = $c
            break
        }
    }
    $toolDir = Join-Path $rootDir 'windows-v3.0\tools\vgabench'
    $argsList = @('-bdf', $TargetBdf, '-size-mb', "$SizeMB", '-iters', "$Iterations", '-timeout-ms', "$TimeoutMs")
    try {
        if ($exePath) {
            $out = & $exePath @argsList 2>$null
            return ($out -join "`n")
        }
        $goCmd = Get-Command go -ErrorAction SilentlyContinue
        if ($goCmd -and (Test-Path -LiteralPath (Join-Path $toolDir 'main.go') -PathType Leaf)) {
            Push-Location $toolDir
            try {
                $out = & $goCmd.Source run . @argsList 2>$null
                return ($out -join "`n")
            } finally {
                Pop-Location
            }
        }
    } catch {
        return (@{ status = 'error'; reason = $_.Exception.Message } | ConvertTo-Json -Compress)
    }
    return $null
}

function Get-VgaBenchInfo {
    param([AllowNull()][string]$TargetBdf)
    if ([string]::IsNullOrWhiteSpace($TargetBdf)) {
        return [pscustomobject]@{ status = 'skipped'; reason = 'missing_target_bdf' }
    }
    $raw = Invoke-VgaBenchRaw -TargetBdf $TargetBdf
    if ([string]::IsNullOrWhiteSpace($raw)) {
        return [pscustomobject]@{ status = 'unsupported'; reason = 'vgabench_helper_not_found' }
    }
    try {
        return ($raw | ConvertFrom-Json)
    } catch {
        return [pscustomobject]@{ status = 'error'; reason = 'invalid_vgabench_json' }
    }
}

function New-Report {
    param(
        [Parameter(Mandatory = $false)]$Inventory,
        [Parameter(Mandatory = $false)]$Nvidia,
        [Parameter(Mandatory = $false)]$Context,
        [AllowNull()][string]$ExperimentDir = $null,
        [AllowNull()][string]$TargetBdf = $null,
        [Parameter(Mandatory = $false)]$Security = $null,
        [AllowNull()][string]$LegacyStatusDir = $null,
        [AllowNull()][string]$CollectionStartedUtc = $null,
        [switch]$KeepHistory,
        [Parameter(Mandatory = $false)]$LoadBench = $null
    )
    $startedUtc = if (-not [string]::IsNullOrWhiteSpace($CollectionStartedUtc)) { $CollectionStartedUtc } else { (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ') }
    $invArray = @($Inventory)
    $gpuEntries = New-Object System.Collections.ArrayList
    $hasInvError = $false
    foreach ($g in $invArray) {
        $vidVal = if ($g.vendor_id -and $g.vendor_id.status -eq 'ok') { [string]$g.vendor_id.value } else { $null }
        $didVal = if ($g.device_id -and $g.device_id.status -eq 'ok') { [string]$g.device_id.value } else { $null }
        $nameVal = if ($g.name -and $g.name.status -eq 'ok') { [string]$g.name.value } else { $null }
        $profObj = if ($null -ne $g.profile) { $g.profile } else { Get-GpuProfileDetails -VendorId $vidVal -DeviceId $didVal -Name $nameVal }
        $drvObj = if ($null -ne $g.driver) {
            $g.driver
        } else {
            [ordered]@{
                provider      = New-ReportField -Value $null -Source 'pnp' -Status 'unsupported'
                date          = New-ReportField -Value $null -Source 'pnp' -Status 'unsupported'
                inf_name      = New-ReportField -Value $null -Source 'pnp' -Status 'unsupported'
                service       = New-ReportField -Value $null -Source 'pnp' -Status 'unsupported'
                problem_code  = New-ReportField -Value $null -Source 'pnp' -Status 'unsupported'
                location_path = New-ReportField -Value $null -Source 'pnp' -Status 'unsupported'
            }
        }
        if ($g.bdf -and $g.bdf.status -eq 'error') { $hasInvError = $true }
        $entry = [ordered]@{
            bdf            = $g.bdf
            vendor_id      = $g.vendor_id
            device_id      = $g.device_id
            subsystem_id   = $g.subsystem_id
            revision       = $g.revision
            name           = $g.name
            pnp_status     = $g.pnp_status
            driver_version = $g.driver_version
            profile        = $profObj
            driver         = $drvObj
        }
        if ($null -ne $g.pcie) { $entry['pcie'] = $g.pcie }
        if ($null -ne $g.nvidia) { $entry['nvidia'] = $g.nvidia }
        [void]$gpuEntries.Add($entry)
    }

    $primaryGpu = Select-TargetGpu -Inventory $invArray -TargetBdf $TargetBdf
    $resolvedTargetBdf = if ($primaryGpu -and $primaryGpu.bdf -and $primaryGpu.bdf.status -eq 'ok') {
        Normalize-Bdf -BusId ([string]$primaryGpu.bdf.value)
    } else {
        $null
    }

    $topPcie = $null
    if ($primaryGpu -and $primaryGpu.pcie) {
        $topPcie = $primaryGpu.pcie
    } elseif ($Nvidia -and $Nvidia.fields) {
        $topPcie = [ordered]@{
            max_gen       = $Nvidia.fields['pcie.link.gen.max']
            current_gen   = $Nvidia.fields['pcie.link.gen.gpucurrent']
            max_width     = $Nvidia.fields['pcie.link.width.max']
            current_width = $Nvidia.fields['pcie.link.width.current']
        }
    } else {
        $topPcie = [ordered]@{
            max_gen       = New-ReportField -Value $null -Source 'pnp' -Unit 'gen' -Status 'unsupported'
            current_gen   = New-ReportField -Value $null -Source 'pnp' -Unit 'gen' -Status 'unsupported'
            max_width     = New-ReportField -Value $null -Source 'pnp' -Unit 'lanes' -Status 'unsupported'
            current_width = New-ReportField -Value $null -Source 'pnp' -Unit 'lanes' -Status 'unsupported'
        }
    }

    $plat = Get-RawPlatformInfo
    $secObj = if ($null -ne $Security) { $Security } else { Get-SecurityAndBootInfo }
    $legacyObj = Get-LegacyUnlockStatus -StatusDir $LegacyStatusDir
    $notesClean = if ($Context -and $Context.notes) { Remove-Pii -Text ([string]$Context.notes) } else { '' }
    if ($notesClean.Length -gt 500) { $notesClean = $notesClean.Substring(0, 500) }

    $ctxObj = [ordered]@{
        mod_state = if ($Context -and $Context.mod_state) { [string]$Context.mod_state } else { 'unknown' }
        mod_type  = if ($Context -and $Context.mod_type) { [string]$Context.mod_type } else { 'unknown' }
        riser     = if ($Context -and $Context.riser) { [string]$Context.riser } else { 'unknown' }
        slot      = if ($Context -and $Context.slot) { [string]$Context.slot } else { 'unknown' }
        stage     = if ($Context -and $Context.stage) { [string]$Context.stage } else { 'unknown' }
        notes     = $notesClean
    }

    $missingList = New-Object 'System.Collections.Generic.List[string]'
    if ($Nvidia -and $null -ne $Nvidia.missing) {
        foreach ($m in @($Nvidia.missing)) {
            if (-not [string]::IsNullOrEmpty([string]$m)) {
                [void]$missingList.Add([string]$m)
            }
        }
    }
    $nvBlock = [ordered]@{
        complete = if ($Nvidia) { [bool]$Nvidia.complete } else { $false }
        missing  = $missingList.ToArray()
        fields   = if ($Nvidia -and $Nvidia.fields) { $Nvidia.fields } else { (New-EmptyNvidiaBlock -Status 'skipped').fields }
    }
    if ($Nvidia -and $null -ne $Nvidia.extended) {
        $extMiss = New-Object 'System.Collections.Generic.List[string]'
        if ($null -ne $Nvidia.extended_missing) {
            foreach ($em in @($Nvidia.extended_missing)) {
                if (-not [string]::IsNullOrEmpty([string]$em)) { [void]$extMiss.Add([string]$em) }
            }
        }
        $nvBlock['extended'] = $Nvidia.extended
        $nvBlock['extended_missing'] = $extMiss.ToArray()
    }

    $prevReportId = Get-PreviousReportId -BaseDir $ExperimentDir -KeepHistory:$KeepHistory
    $finishedUtc = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    $pnpTs = if ($script:LastSourceTimestamps['pnp_utc']) { $script:LastSourceTimestamps['pnp_utc'] } else { $finishedUtc }
    $cimTs = if ($script:LastSourceTimestamps['cim_utc']) { $script:LastSourceTimestamps['cim_utc'] } else { $finishedUtc }
    $nvTs = $script:LastSourceTimestamps['nvidia_smi_utc']
    $secTs = if ($script:LastSourceTimestamps['security_utc']) { $script:LastSourceTimestamps['security_utc'] } else { $finishedUtc }

    $linkIdleMissing = New-Object 'System.Collections.Generic.List[string]'
    if (-not ($topPcie.current_gen -and $topPcie.current_gen.status -eq 'ok')) { [void]$linkIdleMissing.Add('pcie.current_gen') }
    if (-not ($topPcie.current_width -and $topPcie.current_width.status -eq 'ok')) { [void]$linkIdleMissing.Add('pcie.current_width') }
    if (-not $nvBlock.complete) { [void]$linkIdleMissing.Add('nvidia_basic_fields') }
    $linkIdleOk = ($linkIdleMissing.Count -eq 0)

    $invMissing = New-Object 'System.Collections.Generic.List[string]'
    if ($invArray.Count -eq 0) { [void]$invMissing.Add('gpus') }
    if ($hasInvError) { [void]$invMissing.Add('gpu_pnp_properties') }

    $compMissing = New-Object 'System.Collections.Generic.List[string]'
    [void]$compMissing.Add($(if ($prevReportId) { 'paired_load_metrics' } else { 'previous_report_id' }))

    $newCompEntry = {
        param([bool]$Complete, [string]$Status, [string[]]$Missing, [string]$Reason)
        $list = New-Object 'System.Collections.Generic.List[string]'
        foreach ($m in @($Missing)) {
            if (-not [string]::IsNullOrEmpty([string]$m)) { [void]$list.Add([string]$m) }
        }
        return [ordered]@{
            complete = $Complete
            status   = $Status
            missing  = $list.ToArray()
            reason   = $Reason
        }
    }

    $truncateReason = {
        param([string]$Txt)
        $clean = Remove-Pii -Text $Txt
        if ($clean.Length -gt 200) { return $clean.Substring(0, 200) }
        return $clean
    }

    $evidenceLevel = 'reported'
    $linkLoadEntry = & $newCompEntry $false 'skipped' @('h2d_d2h', 'load_timeseries') 'Chưa chạy bài thử tải GPU (chế độ P2)'
    $computeEntry = & $newCompEntry $false 'skipped' @('cuda_runtime', 'tensor_check') 'Chưa chạy helper kiểm tra tính toán/Tensor (chế độ P2)'
    if ($null -ne $LoadBench) {
        if ($LoadBench.status -eq 'ok' -and $LoadBench.fp32_kernel_ok -eq $true) {
            $evidenceLevel = 'observed'
            $loadReason = & $truncateReason ("H2D=$($LoadBench.h2d_bandwidth_gbps) GB/s, D2H=$($LoadBench.d2h_bandwidth_gbps) GB/s ($($LoadBench.transfer_size_mb) MiB, pinned=$($LoadBench.pinned_host))")
            $compReason = & $truncateReason ("CUDA PTX FP32 OK ($($LoadBench.fp32_gflops) GFLOPS, CC $($LoadBench.compute_capability))")
            $linkLoadEntry = & $newCompEntry $true 'ok' @() $loadReason
            $computeEntry = & $newCompEntry $true 'ok' @() $compReason
        } else {
            $bStatus = if ($LoadBench.status -in @('ok', 'partial', 'skipped', 'unsupported', 'error')) { [string]$LoadBench.status } else { 'error' }
            if ($bStatus -eq 'ok') { $bStatus = 'error' }
            $bReason = if ($LoadBench.reason) { & $truncateReason ([string]$LoadBench.reason) } else { "vgabench status=$bStatus" }
            $linkLoadEntry = & $newCompEntry $false $bStatus @('h2d_d2h') $bReason
            $computeEntry = & $newCompEntry $false $bStatus @('cuda_runtime', 'fp32_kernel') $bReason
        }
    }

    $completeness = [ordered]@{
        inventory  = (& $newCompEntry ($invMissing.Count -eq 0) $(if ($invMissing.Count -eq 0) { 'ok' } elseif ($invArray.Count -gt 0) { 'partial' } else { 'error' }) $invMissing.ToArray() $(if ($invMissing.Count -eq 0) { '' } else { 'Một hoặc nhiều GPU lỗi đọc thuộc tính PnP' }))
        topology   = (& $newCompEntry $false 'partial' @('upstream_root_port', 'tls', 'aspm', 'rebar_capability') 'Chế độ cơ bản P1 chưa đọc topology upstream và PCIe BAR (chờ P3)')
        link_idle  = (& $newCompEntry $linkIdleOk $(if ($linkIdleOk) { 'ok' } else { 'partial' }) $linkIdleMissing.ToArray() $(if ($linkIdleOk) { '' } else { 'Thiếu một phần thông số link hoặc nvidia-smi cơ bản ở trạng thái nhàn rỗi' }))
        link_load  = $linkLoadEntry
        compute    = $computeEntry
        stability  = (& $newCompEntry $false 'skipped' @('whea_events', 'tdr_events') 'Chưa ghi cửa sổ ổn định dưới tải (chế độ P2)')
        comparison = (& $newCompEntry $false $(if ($prevReportId) { 'partial' } else { 'skipped' }) $compMissing.ToArray() $(if ($prevReportId) { 'Đã liên kết báo cáo trước nhưng chưa có dữ liệu đối chứng dưới tải' } else { 'Chưa có báo cáo trước trong experiment này' }))
    }

    return [ordered]@{
        schema_version          = $script:SchemaVersion
        collector_version       = $script:CollectorVersion
        report_id               = [guid]::NewGuid().ToString()
        previous_report_id      = $prevReportId
        created_utc             = $finishedUtc
        collection_started_utc  = $startedUtc
        collection_finished_utc = $finishedUtc
        source_timestamps       = [ordered]@{
            pnp_utc        = $pnpTs
            cim_utc        = $cimTs
            nvidia_smi_utc = $nvTs
            security_utc   = $secTs
        }
        experiment_id           = Get-OrCreateExperimentId -BaseDir $ExperimentDir -KeepHistory:$KeepHistory
        target_bdf              = $resolvedTargetBdf
        evidence_level          = $evidenceLevel
        consent                 = $true
        gpus                    = $gpuEntries.ToArray()
        nvidia                  = $nvBlock
        pcie                    = $topPcie
        platform                = [ordered]@{
            cpu             = New-ReportField -Value $plat.Cpu -Source 'cim'
            mainboard_model = New-ReportField -Value $plat.Mainboard -Source 'cim'
            bios_version    = New-ReportField -Value $plat.Bios -Source 'cim'
            os_build        = New-ReportField -Value $plat.OsBuild -Source 'cim'
        }
        security                = $secObj
        completeness            = $completeness
        legacy_status           = $legacyObj
        context                 = $ctxObj
    }
}

function Get-ObjProp {
    param($Obj, [string]$Name)
    if ($null -eq $Obj) { return $null }
    if ($Obj -is [System.Collections.IDictionary]) {
        if ($Obj.Contains($Name)) { return $Obj[$Name] }
        return $null
    }
    $prop = $Obj.PSObject.Properties[$Name]
    if ($prop) { return $prop.Value }
    return $null
}

function Format-PcieLink {
    param($GenField, $WidthField)
    $genText = if ($GenField -and $GenField.status -eq 'ok' -and $null -ne $GenField.value) { "$($GenField.value)" } else { '?' }
    $widthText = if ($WidthField -and $WidthField.status -eq 'ok' -and $null -ne $WidthField.value) { "$($WidthField.value)" } else { '?' }
    return "Gen$genText ×$widthText"
}

function Get-PcieSourceWarnings {
    param($Report)
    $warnings = New-Object 'System.Collections.Generic.List[string]'
    if (-not $Report -or -not $Report.pcie -or -not $Report.nvidia -or -not $Report.nvidia.fields) {
        return $warnings.ToArray()
    }
    $pcie = $Report.pcie
    $nvFields = $Report.nvidia.fields

    $pnpCurGen = Get-ObjProp $pcie 'current_gen'
    $pnpCurW = Get-ObjProp $pcie 'current_width'
    $nvCurGen = Get-ObjProp $nvFields 'pcie.link.gen.gpucurrent'
    $nvCurW = Get-ObjProp $nvFields 'pcie.link.width.current'

    $curGenDiff = ($pnpCurGen -and $pnpCurGen.source -eq 'pnp' -and $pnpCurGen.status -eq 'ok' -and $nvCurGen -and $nvCurGen.status -eq 'ok' -and ([string]$pnpCurGen.value -ne [string]$nvCurGen.value))
    $curWDiff = ($pnpCurW -and $pnpCurW.source -eq 'pnp' -and $pnpCurW.status -eq 'ok' -and $nvCurW -and $nvCurW.status -eq 'ok' -and ([string]$pnpCurW.value -ne [string]$nvCurW.value))
    if ($curGenDiff -or $curWDiff) {
        [void]$warnings.Add("Cảnh báo: PnP báo hiện tại $(Format-PcieLink $pnpCurGen $pnpCurW), nvidia-smi báo $(Format-PcieLink $nvCurGen $nvCurW). Hai nguồn đọc khác thời điểm; link có thể hạ tốc khi nhàn rỗi. Chưa kết luận nguồn nào sai.")
    }

    $pnpMaxGen = Get-ObjProp $pcie 'max_gen'
    $pnpMaxW = Get-ObjProp $pcie 'max_width'
    $nvMaxGen = Get-ObjProp $nvFields 'pcie.link.gen.max'
    $nvMaxW = Get-ObjProp $nvFields 'pcie.link.width.max'

    $maxGenDiff = ($pnpMaxGen -and $pnpMaxGen.source -eq 'pnp' -and $pnpMaxGen.status -eq 'ok' -and $nvMaxGen -and $nvMaxGen.status -eq 'ok' -and ([string]$pnpMaxGen.value -ne [string]$nvMaxGen.value))
    $maxWDiff = ($pnpMaxW -and $pnpMaxW.source -eq 'pnp' -and $pnpMaxW.status -eq 'ok' -and $nvMaxW -and $nvMaxW.status -eq 'ok' -and ([string]$pnpMaxW.value -ne [string]$nvMaxW.value))
    if ($maxGenDiff -or $maxWDiff) {
        [void]$warnings.Add("Cảnh báo: PnP báo tối đa $(Format-PcieLink $pnpMaxGen $pnpMaxW), nvidia-smi báo $(Format-PcieLink $nvMaxGen $nvMaxW). Hai nguồn đọc khác thời điểm hoặc khác cách báo cáo khả năng khe/card. Chưa kết luận nguồn nào sai.")
    }

    return $warnings.ToArray()
}

function Format-VietnameseSummary {
    param($Report)
    $sb = New-Object System.Text.StringBuilder
    [void]$sb.AppendLine('=== TÓM TẮT BÁO CÁO CHẨN ĐOÁN VGA ===')
    [void]$sb.AppendLine("Mã báo cáo   : $($Report.report_id)")
    [void]$sb.AppendLine("Mã thử nghiệm: $($Report.experiment_id)")
    if ($Report.previous_report_id) {
        [void]$sb.AppendLine("Báo cáo trước: $($Report.previous_report_id)")
    } elseif ($Report.schema_version -eq 2) {
        [void]$sb.AppendLine('Báo cáo trước: null (báo cáo đầu tiên trong chuỗi thử nghiệm)')
    }
    [void]$sb.AppendLine("Thời gian UTC: $($Report.created_utc)")
    if ($Report.collection_started_utc -or $Report.collection_finished_utc) {
        [void]$sb.AppendLine("Cửa sổ đo UTC: $($Report.collection_started_utc) -> $($Report.collection_finished_utc)")
    }
    [void]$sb.AppendLine("Phiên bản    : $($Report.collector_version) (schema v$($Report.schema_version))")
    if ($Report.evidence_level) {
        [void]$sb.AppendLine("Mức bằng chứng: $($Report.evidence_level)")
    }

    $nvFields = if ($Report.nvidia -and $Report.nvidia.fields) { $Report.nvidia.fields } else { $null }
    $explicitBdf = if ($Report.target_bdf) { Normalize-Bdf -BusId ([string]$Report.target_bdf) } else { $null }
    $nvBusField = Get-ObjProp $nvFields 'pci.bus_id'
    $nvTargetBdf = if ($nvBusField -and $nvBusField.status -eq 'ok') { Normalize-Bdf -BusId ([string]$nvBusField.value) } else { $null }
    $targetHintBdf = if ($explicitBdf) { $explicitBdf } else { $nvTargetBdf }
    $targetGpu = Select-TargetGpu -Inventory @($Report.gpus) -TargetBdf $targetHintBdf
    if ($targetGpu) {
        $tBdf = if ($targetGpu.bdf.status -eq 'ok') { $targetGpu.bdf.value } else { $targetGpu.bdf.status }
        $tName = if ($targetGpu.name.status -eq 'ok') { $targetGpu.name.value } else { $targetGpu.name.status }
        $tVid = if ($targetGpu.vendor_id.status -eq 'ok') { [string]$targetGpu.vendor_id.value } else { $null }
        $tDid = if ($targetGpu.device_id.status -eq 'ok') { [string]$targetGpu.device_id.value } else { $null }
        $tProfDetails = if ($targetGpu.profile) { $targetGpu.profile } else { Get-GpuProfileDetails -VendorId $tVid -DeviceId $tDid -Name $tName }
        $tProfText = if ($tProfDetails.name) { $tProfDetails.name } else { 'chưa nhận diện' }
        $tFam = if ($tProfDetails.family) { $tProfDetails.family } else { 'unknown' }
        $tCat = if ($tProfDetails.category) { $tProfDetails.category } else { 'unknown' }
        $tTargetGen = if ($tProfDetails.target_pcie_gen) { "Gen$($tProfDetails.target_pcie_gen)" } else { 'chưa đặt' }
        $tTensor = if ($tProfDetails.tensor_applicable) { $tProfDetails.tensor_applicable } else { 'unknown' }
        [void]$sb.AppendLine("GPU mục tiêu : [$tBdf] $tName — hồ sơ: $tProfText (family=$tFam, category=$tCat, target_pcie_gen=$tTargetGen, tensor_applicable=$tTensor)")
    } else {
        [void]$sb.AppendLine('GPU mục tiêu : chưa xác định')
    }
    if ($Report.source_timestamps) {
        $ts = $Report.source_timestamps
        [void]$sb.AppendLine("Mốc thời gian: pnp=$($ts.pnp_utc) | cim=$($ts.cim_utc) | nvidia=$($ts.nvidia_smi_utc) | sec=$($ts.security_utc)")
    }
    [void]$sb.AppendLine('')

    [void]$sb.AppendLine('--- GPU phát hiện ---')
    $targetNormBdf = if ($targetGpu -and $targetGpu.bdf -and $targetGpu.bdf.status -eq 'ok') { Normalize-Bdf -BusId ([string]$targetGpu.bdf.value) } else { $null }
    foreach ($g in @($Report.gpus)) {
        $bdf = if ($g.bdf.status -eq 'ok') { $g.bdf.value } else { $g.bdf.status }
        $name = if ($g.name.status -eq 'ok') { $g.name.value } else { $g.name.status }
        $vidRaw = if ($g.vendor_id.status -eq 'ok') { [string]$g.vendor_id.value } else { $null }
        $didRaw = if ($g.device_id.status -eq 'ok') { [string]$g.device_id.value } else { $null }
        $vid = if ($vidRaw) { $vidRaw } else { '?' }
        $did = if ($didRaw) { $didRaw } else { '?' }
        $drv = if ($g.driver_version.status -eq 'ok') { $g.driver_version.value } else { $g.driver_version.status }
        $profDetails = if ($g.profile) { $g.profile } else { Get-GpuProfileDetails -VendorId $vidRaw -DeviceId $didRaw -Name $name }
        $profText = if ($profDetails.name) { $profDetails.name } else { 'chưa nhận diện' }
        $gFam = if ($profDetails.family) { $profDetails.family } else { 'unknown' }
        $gCat = if ($profDetails.category) { $profDetails.category } else { 'unknown' }
        $gTargetGen = if ($profDetails.target_pcie_gen) { "Gen$($profDetails.target_pcie_gen)" } else { 'chưa đặt' }
        $gTensor = if ($profDetails.tensor_applicable) { $profDetails.tensor_applicable } else { 'unknown' }
        $gPcieTag = if ($g.pcie) {
            " | PCIe(PnP): $(Format-PcieLink -GenField (Get-ObjProp $g.pcie 'current_gen') -WidthField (Get-ObjProp $g.pcie 'current_width')) / max $(Format-PcieLink -GenField (Get-ObjProp $g.pcie 'max_gen') -WidthField (Get-ObjProp $g.pcie 'max_width'))"
        } else { '' }
        $gNormBdf = if ($g.bdf -and $g.bdf.status -eq 'ok') { Normalize-Bdf -BusId ([string]$g.bdf.value) } else { $null }
        $targetTag = if ($targetNormBdf -and $gNormBdf -eq $targetNormBdf) { ' (mục tiêu)' } else { '' }
        $infTag = if ($g.driver -and $g.driver.inf_name -and $g.driver.inf_name.status -eq 'ok') { " | INF: $($g.driver.inf_name.value)" } else { '' }
        $probTag = if ($g.driver -and $g.driver.problem_code -and $g.driver.problem_code.status -eq 'ok') { " | ProblemCode: $($g.driver.problem_code.value)" } else { '' }
        [void]$sb.AppendLine("  [$bdf] $name ($vid`:$did) | Hồ sơ: $profText [$gFam/$gCat, Target=$gTargetGen, Tensor=$gTensor]$gPcieTag | Driver: $drv$infTag$probTag$targetTag")
        if ($profDetails.identity_note) {
            [void]$sb.AppendLine("    Lưu ý danh tính: $($profDetails.identity_note)")
        }
    }
    [void]$sb.AppendLine('')

    if ($Report.security) {
        $sec = $Report.security
        $bm = if ($sec.boot_mode -and $sec.boot_mode.status -eq 'ok') { $sec.boot_mode.value } else { $sec.boot_mode.status }
        $sbv = if ($sec.secure_boot -and $sec.secure_boot.status -eq 'ok') { $sec.secure_boot.value } else { $sec.secure_boot.status }
        $hv = if ($sec.hvci_running -and $sec.hvci_running.status -eq 'ok') { $sec.hvci_running.value } else { $sec.hvci_running.status }
        $tsv = if ($sec.test_signing -and $sec.test_signing.status -eq 'ok') { $sec.test_signing.value } else { $sec.test_signing.status }
        [void]$sb.AppendLine("Bảo mật & Boot: Boot=$bm | SecureBoot=$sbv | HVCI=$hv | TestSigning=$tsv")
        [void]$sb.AppendLine('')
    }

    if ($Report.pcie) {
        $maxLink = Format-PcieLink -GenField (Get-ObjProp $Report.pcie 'max_gen') -WidthField (Get-ObjProp $Report.pcie 'max_width')
        $curLink = Format-PcieLink -GenField (Get-ObjProp $Report.pcie 'current_gen') -WidthField (Get-ObjProp $Report.pcie 'current_width')
        $curGenField = Get-ObjProp $Report.pcie 'current_gen'
        $pcieSrc = if ($curGenField -and $curGenField.source) { $curGenField.source } else { 'unknown' }
        [void]$sb.AppendLine("Liên kết PCIe ($pcieSrc): tối đa $maxLink | hiện tại $curLink")
        if ($nvFields) {
            $nvMaxLink = Format-PcieLink -GenField (Get-ObjProp $nvFields 'pcie.link.gen.max') -WidthField (Get-ObjProp $nvFields 'pcie.link.width.max')
            $nvCurLink = Format-PcieLink -GenField (Get-ObjProp $nvFields 'pcie.link.gen.gpucurrent') -WidthField (Get-ObjProp $nvFields 'pcie.link.width.current')
            [void]$sb.AppendLine("Liên kết PCIe (nvidia-smi): tối đa $nvMaxLink | hiện tại $nvCurLink")
        }
        foreach ($warn in @(Get-PcieSourceWarnings -Report $Report)) {
            [void]$sb.AppendLine($warn)
        }
        [void]$sb.AppendLine('')
    }

    if ($Report.nvidia) {
        $okCount = 0
        foreach ($k in $script:NvidiaRequiredFields) {
            $f = Get-ObjProp $nvFields $k
            if ($f -and $f.status -eq 'ok') { $okCount += 1 }
        }
        [void]$sb.AppendLine("Trường NVIDIA cơ bản (nvidia-smi): $okCount/$($script:NvidiaRequiredFields.Count) (đủ 18/18: $($Report.nvidia.complete))")
        [void]$sb.AppendLine('Lưu ý: 18/18 chỉ là nhóm trường NVIDIA cơ bản, không phải đủ dữ liệu nghiên cứu/mod.')
        if (@($Report.nvidia.missing).Count -gt 0) {
            [void]$sb.AppendLine("Các trường thiếu / không hỗ trợ: $((@($Report.nvidia.missing)) -join ', ')")
        }
        if ($Report.nvidia.extended) {
            $ext = $Report.nvidia.extended
            $extOkCount = 0
            foreach ($ek in $script:NvidiaExtendedFields) {
                $ef = Get-ObjProp $ext $ek
                if ($ef -and $ef.status -eq 'ok') { $extOkCount += 1 }
            }
            [void]$sb.AppendLine("Trường NVIDIA mở rộng (power/thermal/throttle): $extOkCount/$($script:NvidiaExtendedFields.Count)")
            $enf = Get-ObjProp $ext 'enforced.power.limit'
            $def = Get-ObjProp $ext 'power.default_limit'
            $maxP = Get-ObjProp $ext 'power.max_limit'
            $act = Get-ObjProp $ext 'clocks_throttle_reasons.active'
            $enfStr = if ($enf -and $enf.status -eq 'ok') { "$($enf.value) W" } elseif ($enf) { $enf.status } else { 'unsupported' }
            $defStr = if ($def -and $def.status -eq 'ok') { "$($def.value) W" } elseif ($def) { $def.status } else { 'unsupported' }
            $maxPStr = if ($maxP -and $maxP.status -eq 'ok') { "$($maxP.value) W" } elseif ($maxP) { $maxP.status } else { 'unsupported' }
            $actStr = if ($act -and $act.status -eq 'ok') { "$($act.value)" } elseif ($act) { $act.status } else { 'unsupported' }
            [void]$sb.AppendLine("Công suất & giảm xung: enforced=$enfStr | default=$defStr | max=$maxPStr | active_reasons=$actStr")
        }
    }

    if ($Report.legacy_status) {
        $leg = $Report.legacy_status
        if ($leg.present) {
            $devTxt = if ($leg.device_id) { $leg.device_id } else { 'null' }
            $mrrsTxt = if ($null -ne $leg.mrrs_bytes) { $leg.mrrs_bytes } else { 'null' }
            [void]$sb.AppendLine("Trạng thái mở khóa cũ ($($leg.source)): $($leg.status_code) (error=$($leg.error_code), device_id=$devTxt, Gen$($leg.speed_current) ×$($leg.width_current), TLS=$($leg.tls_target), mrrs=$mrrsTxt)")
        } else {
            [void]$sb.AppendLine('Trạng thái mở khóa cũ: không phát hiện tệp trạng thái cục bộ (present=false)')
        }
    }

    if ($Report.completeness) {
        [void]$sb.AppendLine('')
        [void]$sb.AppendLine('--- Mức độ đầy đủ theo nhóm bằng chứng ---')
        foreach ($gk in @('inventory', 'topology', 'link_idle', 'link_load', 'compute', 'stability', 'comparison')) {
            $grp = Get-ObjProp $Report.completeness $gk
            if ($grp) {
                $missTxt = if (@($grp.missing).Count -gt 0) { " | thiếu: $((@($grp.missing)) -join ', ')" } else { '' }
                [void]$sb.AppendLine("  [$gk] $($grp.status)$missTxt")
            }
        }
    }
    return $sb.ToString()
}

function Save-LocalReport {
    param(
        [Parameter(Mandatory = $true)]$Report,
        [Parameter(Mandatory = $true)][string]$Dir,
        [switch]$KeepHistory
    )
    [void][System.IO.Directory]::CreateDirectory($Dir)
    [void](Redact-ReportObject -InputObject $Report)
    $stamp = (Get-Date).ToUniversalTime().ToString('yyyyMMdd-HHmmss')
    $shortId = ([string]$Report.report_id).Split('-')[0]
    $baseName = "vga-report-$stamp-$shortId"

    $jsonPath = Join-Path -Path $Dir -ChildPath "$baseName.json"
    $summaryPath = Join-Path -Path $Dir -ChildPath "$baseName-summary.txt"
    $zipPath = Join-Path -Path $Dir -ChildPath "$baseName.zip"

    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    $jsonText = Remove-Pii -Text ($Report | ConvertTo-Json -Depth 12)
    $summaryText = Remove-Pii -Text (Format-VietnameseSummary -Report $Report)

    [System.IO.File]::WriteAllText($jsonPath, $jsonText, $utf8NoBom)
    [System.IO.File]::WriteAllText($summaryPath, $summaryText, $utf8NoBom)
    Save-LastReportId -BaseDir $Dir -ReportId ([string]$Report.report_id) -KeepHistory:$KeepHistory

    $stageDir = Join-Path -Path ([System.IO.Path]::GetTempPath()) ("vga-zip-" + [guid]::NewGuid().ToString())
    try {
        [void][System.IO.Directory]::CreateDirectory($stageDir)
        [System.IO.File]::Copy($jsonPath, (Join-Path -Path $stageDir -ChildPath 'report.json'), $true)
        [System.IO.File]::Copy($summaryPath, (Join-Path -Path $stageDir -ChildPath 'summary.txt'), $true)
        if (Test-Path -LiteralPath $zipPath) {
            Remove-Item -LiteralPath $zipPath -Force
        }
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::CreateFromDirectory($stageDir, $zipPath)
    } finally {
        Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    return [pscustomobject]@{
        JsonPath    = $jsonPath
        SummaryPath = $summaryPath
        ZipPath     = $zipPath
        ReportId    = $Report.report_id
        JsonText    = $jsonText
        BaseName    = $baseName
    }
}

function New-RequestSignature {
    param(
        [Parameter(Mandatory = $false)][AllowEmptyString()][string]$Body = '',
        [Parameter(Mandatory = $false)][byte[]]$BodyBytes = $null,
        [Parameter(Mandatory = $true)][string]$KeyId,
        [Parameter(Mandatory = $true)][string]$Key,
        [Parameter(Mandatory = $false)][long]$Timestamp = 0
    )
    if ($Timestamp -le 0) {
        $Timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    }
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    $bytes = if ($null -ne $BodyBytes) { $BodyBytes } else { $utf8NoBom.GetBytes($Body) }

    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $hashBytes = $sha.ComputeHash($bytes)
    } finally {
        $sha.Dispose()
    }
    $bodyShaHex = -join ($hashBytes | ForEach-Object { $_.ToString('x2') })
    $payloadBytes = $utf8NoBom.GetBytes("$Timestamp.$bodyShaHex")

    $keyBytes = [Convert]::FromBase64String($Key)
    $hmac = New-Object System.Security.Cryptography.HMACSHA256 (,$keyBytes)
    try {
        $sigBytes = $hmac.ComputeHash($payloadBytes)
    } finally {
        $hmac.Dispose()
    }
    $sigHex = -join ($sigBytes | ForEach-Object { $_.ToString('x2') })

    return @{
        'X-Report-Key-Id'    = $KeyId
        'X-Report-Timestamp' = [string]$Timestamp
        'X-Report-Signature' = $sigHex
    }
}

function Invoke-HttpPostBytes {
    param(
        [string]$Url,
        [byte[]]$BodyBytes,
        [hashtable]$Headers,
        [int]$TimeoutSec = 15
    )
    $req = [System.Net.HttpWebRequest]::Create($Url)
    $req.Method = 'POST'
    $req.ContentType = 'application/json; charset=utf-8'
    $req.Timeout = $TimeoutSec * 1000
    $req.ReadWriteTimeout = $TimeoutSec * 1000
    $req.AllowAutoRedirect = $false
    $req.ServicePoint.Expect100Continue = $false
    $req.ContentLength = $BodyBytes.Length
    foreach ($hk in $Headers.Keys) {
        $req.Headers[$hk] = [string]$Headers[$hk]
    }

    try {
        $reqStream = $req.GetRequestStream()
        try {
            $reqStream.Write($BodyBytes, 0, $BodyBytes.Length)
        } finally {
            $reqStream.Close()
        }
        $resp = [System.Net.HttpWebResponse]$req.GetResponse()
        try {
            $sr = New-Object System.IO.StreamReader($resp.GetResponseStream(), [System.Text.Encoding]::UTF8)
            $body = $sr.ReadToEnd()
            $sr.Close()
            return [pscustomobject]@{
                StatusCode     = [int]$resp.StatusCode
                Body           = $body
                RetryAfter     = $null
                TransportError = $null
            }
        } finally {
            $resp.Close()
        }
    } catch [System.Net.WebException] {
        $wex = $_.Exception
        $errResp = $wex.Response
        if ($null -ne $errResp) {
            $httpResp = [System.Net.HttpWebResponse]$errResp
            $retrySec = $null
            $raVal = $httpResp.Headers['Retry-After']
            $parsedRa = 0
            if ($raVal -and [int]::TryParse($raVal.Trim(), [ref]$parsedRa)) {
                $retrySec = $parsedRa
            }
            $sr = New-Object System.IO.StreamReader($httpResp.GetResponseStream(), [System.Text.Encoding]::UTF8)
            $errBody = $sr.ReadToEnd()
            $sr.Close()
            $code = [int]$httpResp.StatusCode
            $httpResp.Close()
            return [pscustomobject]@{
                StatusCode     = $code
                Body           = $errBody
                RetryAfter     = $retrySec
                TransportError = $null
            }
        }
        return [pscustomobject]@{
            StatusCode     = 0
            Body           = ''
            RetryAfter     = $null
            TransportError = $wex.Message
        }
    }
}

function Send-Report {
    param(
        [Parameter(Mandatory = $true)]$Report,
        [Parameter(Mandatory = $false)][string]$Endpoint = $script:DefaultEndpoint,
        [Parameter(Mandatory = $false)][string]$KeyId = $script:SigningKeyId,
        [Parameter(Mandatory = $false)][string]$Key = $script:SigningKey,
        [Parameter(Mandatory = $false)][string]$AllowedHost = $script:AllowedEndpointHost,
        [Parameter(Mandatory = $false)][int]$TimeoutSec = 15,
        [Parameter(Mandatory = $false)][AllowNull()][string]$ReportDir = $null,
        [Parameter(Mandatory = $false)][long]$NowUnix = 0,
        [Parameter(Mandatory = $false)][switch]$AllowLoopbackHttp,
        [Parameter(Mandatory = $false)][AllowNull()][string]$ExistingPendingPath = $null,
        [Parameter(Mandatory = $false)][switch]$KeepHistory
    )

    $parsedUri = $null
    if (-not [Uri]::TryCreate($Endpoint, [UriKind]::Absolute, [ref]$parsedUri)) {
        return [pscustomobject]@{ Status = 'rejected'; IssueUrl = $null; Error = 'URL endpoint không hợp lệ.'; RetryAfter = $null }
    }
    if ($AllowLoopbackHttp) {
        if ($parsedUri.Scheme -notin @('http', 'https') -or $parsedUri.Host -notin @('127.0.0.1', '::1')) {
            return [pscustomobject]@{ Status = 'rejected'; IssueUrl = $null; Error = 'Loopback test chỉ cho phép 127.0.0.1.'; RetryAfter = $null }
        }
    } else {
        if ($parsedUri.Scheme -ne 'https' -or -not $parsedUri.Host.Equals($AllowedHost, [StringComparison]::OrdinalIgnoreCase)) {
            return [pscustomobject]@{ Status = 'rejected'; IssueUrl = $null; Error = 'Endpoint phải dùng HTTPS và đúng hostname cố định.'; RetryAfter = $null }
        }
    }

    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor [System.Net.SecurityProtocolType]::Tls12

    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    $pendingPath = $ExistingPendingPath
    $bodyText = $null

    if ($Report -is [string]) {
        $bodyText = Remove-Pii -Text $Report
    } else {
        [void](Redact-ReportObject -InputObject $Report)
        if (-not [string]::IsNullOrWhiteSpace($ReportDir)) {
            $saved = Save-LocalReport -Report $Report -Dir $ReportDir -KeepHistory:$KeepHistory
            $bodyText = $saved.JsonText
            if ([string]::IsNullOrWhiteSpace($pendingPath)) {
                $pendingPath = Join-Path -Path $ReportDir -ChildPath "$($saved.BaseName).pending.json"
                [System.IO.File]::WriteAllText($pendingPath, $bodyText, $utf8NoBom)
            }
        } else {
            $bodyText = Remove-Pii -Text ($Report | ConvertTo-Json -Depth 12)
        }
    }

    if ($Endpoint -match 'REPLACE_WITH_' -or $Key -match 'REPLACE_WITH_') {
        return [pscustomobject]@{
            Status     = 'offline'
            IssueUrl   = $null
            Error      = 'Endpoint hoặc khóa ký chưa được cấu hình (xem Task 6 trong Plan.MD).'
            RetryAfter = $null
        }
    }

    $bodyBytes = $utf8NoBom.GetBytes($bodyText)
    $ts = if ($NowUnix -gt 0) { $NowUnix } else { [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() }

    for ($attempt = 0; $attempt -lt 2; $attempt += 1) {
        $sigHeaders = New-RequestSignature -BodyBytes $bodyBytes -KeyId $KeyId -Key $Key -Timestamp $ts
        $httpRes = Invoke-HttpPostBytes -Url $Endpoint -BodyBytes $bodyBytes -Headers $sigHeaders -TimeoutSec $TimeoutSec

        if ($httpRes.StatusCode -eq 0) {
            return [pscustomobject]@{ Status = 'offline'; IssueUrl = $null; Error = $httpRes.TransportError; RetryAfter = $null }
        }

        $parsedBody = $null
        if (-not [string]::IsNullOrWhiteSpace($httpRes.Body)) {
            try { $parsedBody = $httpRes.Body | ConvertFrom-Json } catch {}
        }

        if ($httpRes.StatusCode -in @(200, 201)) {
            $issueUrl = if ($parsedBody -and $parsedBody.issue_url) { [string]$parsedBody.issue_url } else { $null }
            if ([string]::IsNullOrWhiteSpace($issueUrl)) {
                return [pscustomobject]@{ Status = 'retry_later'; IssueUrl = $null; Error = 'Máy chủ không trả về issue_url hợp lệ.'; RetryAfter = $null }
            }
            if ($pendingPath -and (Test-Path -LiteralPath $pendingPath)) {
                Remove-Item -LiteralPath $pendingPath -Force -ErrorAction SilentlyContinue
            }
            $st = if ($httpRes.StatusCode -eq 201) { 'sent' } else { 'duplicate' }
            return [pscustomobject]@{ Status = $st; IssueUrl = $issueUrl; Error = $null; RetryAfter = $null }
        }

        if ($httpRes.StatusCode -eq 401) {
            $reason = if ($parsedBody -and $parsedBody.reason) { [string]$parsedBody.reason } else { 'unauthorized' }
            if ($attempt -eq 0 -and $reason -eq 'clock_skew' -and $parsedBody -and $null -ne $parsedBody.server_time) {
                $ts = [long]$parsedBody.server_time
                continue
            }
            return [pscustomobject]@{ Status = 'rejected'; IssueUrl = $null; Error = "Xác thực thất bại (401: $reason)."; RetryAfter = $null }
        }

        if ($httpRes.StatusCode -in @(400, 409, 413, 415, 422)) {
            return [pscustomobject]@{ Status = 'rejected'; IssueUrl = $null; Error = "Báo cáo bị từ chối ($($httpRes.StatusCode)): $($httpRes.Body)"; RetryAfter = $null }
        }

        if ($httpRes.StatusCode -eq 429) {
            return [pscustomobject]@{ Status = 'retry_later'; IssueUrl = $null; Error = 'Giới hạn tần suất (429). Vui lòng thử lại sau.'; RetryAfter = $httpRes.RetryAfter }
        }

        return [pscustomobject]@{ Status = 'retry_later'; IssueUrl = $null; Error = "Máy chủ lỗi hoặc bảo trì ($($httpRes.StatusCode))."; RetryAfter = $httpRes.RetryAfter }
    }

    return [pscustomobject]@{ Status = 'rejected'; IssueUrl = $null; Error = 'Không thể xác thực sau khi đồng bộ giờ.'; RetryAfter = $null }
}

function Open-FallbackForm {
    param([Parameter(Mandatory = $true)]$Report)
    $jsonText = if ($Report -is [string]) { $Report } else { $Report | ConvertTo-Json -Depth 12 }
    try {
        Set-Clipboard -Value $jsonText -ErrorAction Stop
        Write-Host 'Đã chép nội dung JSON báo cáo vào clipboard.'
    } catch {
        Write-Host 'Không chép được tự động vào clipboard; vui lòng sao chép từ tệp JSON cục bộ.'
    }
    try {
        Start-Process -FilePath $script:FallbackIssueUrl | Out-Null
    } catch {
        Write-Host "Vui lòng mở liên kết sau trên trình duyệt: $($script:FallbackIssueUrl)"
    }
}

function Read-ChoiceWithDefault {
    param([string]$Prompt, [hashtable]$Map, [string]$DefaultKey)
    Write-Host $Prompt
    foreach ($k in $Map.Keys) {
        Write-Host "  [$k] $($Map[$k].Label)"
    }
    $ans = Read-Host "Chọn (mặc định [$DefaultKey])"
    if ([string]::IsNullOrWhiteSpace($ans)) { return $Map[$DefaultKey].Value }
    $key = $ans.Trim()
    if ($Map.ContainsKey($key)) { return $Map[$key].Value }
    return $Map[$DefaultKey].Value
}

function Start-CollectorInteractive {
    param(
        [switch]$SaveOnly,
        [switch]$KeepHistory,
        [switch]$DetailedContext,
        [switch]$SelectGpu,
        [switch]$RunLoadTest
    )
    $useSaveOnly = $SaveOnly -or ($env:VGA_SAVE_ONLY -eq '1')
    $useKeepHistory = $KeepHistory -or ($env:VGA_KEEP_HISTORY -eq '1')
    $useDetailedContext = $DetailedContext -or ($env:VGA_DETAILED_CONTEXT -eq '1')
    $useSelectGpu = $SelectGpu -or ($env:VGA_SELECT_GPU -eq '1')
    $useRunLoadTest = $RunLoadTest -or ($env:VGA_RUN_LOAD_TEST -eq '1')

    Write-Host '================================================================'
    Write-Host '  THU THẬP BÁO CÁO CHẨN ĐOÁN VGA (CHỈ ĐỌC - KHÔNG GHI HỆ THỐNG)'
    Write-Host '================================================================'
    Write-Host 'Công cụ chỉ đọc thông tin VGA/PCIe/nền tảng (không sửa registry,'
    Write-Host 'không sửa MMIO, không cài đặt Persistence, không cần Admin).'
    Write-Host 'Quyền riêng tư: Chế độ [1] Lưu cục bộ là OFFLINE 100% (không gửi mạng).'
    Write-Host 'Mặc định KHÔNG lưu tệp theo dõi phiên (experiment_id.txt / last_report_id.txt).'
    Write-Host "Nếu chọn [2] Gửi: dữ liệu đẩy qua $($script:TargetHostname) lên GitHub Issue CÔNG KHAI."
    Write-Host 'Mã nguồn mở máy chủ: worker/vga-reports/src/index.js | Chính sách: docs/vga-report/policy.md'
    Write-Host ''

    $outDir = if (-not [string]::IsNullOrWhiteSpace($env:VGA_BAT_DIR)) { Join-Path $env:VGA_BAT_DIR 'vga-reports-out' } else { Join-Path (Get-Location).Path 'vga-reports-out' }
    if (Test-Path -LiteralPath $outDir) {
        $pendingFiles = @(Get-ChildItem -LiteralPath $outDir -Filter '*.pending.json' -File -ErrorAction SilentlyContinue)
        if ($pendingFiles.Count -gt 0) {
            Write-Host "[PHÁT HIỆN] Có $($pendingFiles.Count) báo cáo đang chờ gửi lại trong: $outDir"
            $resendChoice = Read-Host 'Bạn có muốn thử gửi lại các báo cáo này trước không? (C/K, mặc định K)'
            if ($resendChoice -match '^(c|co|có|y|yes)$') {
                foreach ($pf in $pendingFiles) {
                    $rawPending = [System.IO.File]::ReadAllText($pf.FullName, [System.Text.Encoding]::UTF8)
                    $resendRes = Send-Report -Report $rawPending -ExistingPendingPath $pf.FullName -KeepHistory:$useKeepHistory
                    if ($resendRes.Status -in @('sent', 'duplicate') -and -not [string]::IsNullOrWhiteSpace($resendRes.IssueUrl)) {
                        Write-Host "  [OK] Đã gửi ($($resendRes.Status)): $($resendRes.IssueUrl)"
                    } else {
                        Write-Host "  [CHƯA GỬI] $($pf.Name): $($resendRes.Status) - $($resendRes.Error)"
                    }
                }
            }
            Write-Host ''
        }
    }

    Write-Host 'Đang quét thiết bị VGA và nvidia-smi...'
    $collectionStartedUtc = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
    $inventory = @(Get-VgaInventory)
    $targetBdf = $null
    if (-not [string]::IsNullOrWhiteSpace($env:VGA_TARGET_BDF)) {
        $targetBdf = $env:VGA_TARGET_BDF.Trim()
    } elseif ($inventory.Count -ge 2 -and $useSelectGpu) {
        $defaultGpu = Select-TargetGpu -Inventory $inventory
        $defaultIdx = 1
        Write-Host 'Phát hiện nhiều GPU:'
        for ($i = 0; $i -lt $inventory.Count; $i += 1) {
            $g = $inventory[$i]
            $num = $i + 1
            $bdfStr = if ($g.bdf -and $g.bdf.status -eq 'ok') { [string]$g.bdf.value } else { 'unknown' }
            $nameStr = if ($g.name -and $g.name.status -eq 'ok') { [string]$g.name.value } else { 'unknown' }
            $vidStr = if ($g.vendor_id -and $g.vendor_id.status -eq 'ok') { [string]$g.vendor_id.value } else { '?' }
            $didStr = if ($g.device_id -and $g.device_id.status -eq 'ok') { [string]$g.device_id.value } else { '?' }
            $prof = Get-GpuProfileName -VendorId $vidStr -DeviceId $didStr
            $profTag = if ($prof) { $prof } else { 'chưa nhận diện' }
            if ($defaultGpu -and [object]::ReferenceEquals($g, $defaultGpu)) { $defaultIdx = $num }
            Write-Host "  [$num] $bdfStr $nameStr ($vidStr`:$didStr) — $profTag"
        }
        $gpuPick = Read-Host "Chọn GPU mục tiêu (mặc định [$defaultIdx])"
        $pickedIdx = $defaultIdx
        if (-not [string]::IsNullOrWhiteSpace($gpuPick)) {
            $parsedIdx = 0
            if ([int]::TryParse($gpuPick.Trim(), [ref]$parsedIdx) -and $parsedIdx -ge 1 -and $parsedIdx -le $inventory.Count) {
                $pickedIdx = $parsedIdx
            }
        }
        $chosenGpu = $inventory[$pickedIdx - 1]
        if ($chosenGpu -and $chosenGpu.bdf -and $chosenGpu.bdf.status -eq 'ok') {
            $targetBdf = [string]$chosenGpu.bdf.value
        }
    } else {
        $autoGpu = Select-TargetGpu -Inventory $inventory
        if ($autoGpu -and $autoGpu.bdf -and $autoGpu.bdf.status -eq 'ok') {
            $targetBdf = [string]$autoGpu.bdf.value
        }
    }
    $nvidia = Get-NvidiaInfo -Inventory $inventory -TargetBdf $targetBdf

    if ($useDetailedContext) {
        $modState = Read-ChoiceWithDefault -Prompt '1. Trạng thái hàn trở / VBIOS của card:' -DefaultKey '3' -Map ([ordered]@{
            '1' = @{ Label = 'Nguyên bản (stock)'; Value = 'stock' }
            '2' = @{ Label = 'Đã mod (modded)'; Value = 'modded' }
            '3' = @{ Label = 'Không biết'; Value = 'unknown' }
        })
        $modType = Read-ChoiceWithDefault -Prompt '1b. Loại can thiệp/mod đã áp dụng:' -DefaultKey '8' -Map ([ordered]@{
            '1' = @{ Label = 'Không mod / nguyên bản (none)'; Value = 'none' }
            '2' = @{ Label = 'Mod driver / INF (driver)'; Value = 'driver' }
            '3' = @{ Label = 'EFI / UEFI loader (efi)'; Value = 'efi' }
            '4' = @{ Label = 'Flash VBIOS (vbios)'; Value = 'vbios' }
            '5' = @{ Label = 'Hàn trở / strap phần cứng (strap)'; Value = 'strap' }
            '6' = @{ Label = 'Bộ mở khóa AIO / LNKCTL2 (software_aio)'; Value = 'software_aio' }
            '7' = @{ Label = 'Kết hợp nhiều cách (multi)'; Value = 'multi' }
            '8' = @{ Label = 'Không biết (unknown)'; Value = 'unknown' }
        })
        $riser = Read-ChoiceWithDefault -Prompt '2. Cách cắm card:' -DefaultKey '3' -Map ([ordered]@{
            '1' = @{ Label = 'Cắm trực tiếp vào mainboard'; Value = 'direct' }
            '2' = @{ Label = 'Dùng dây riser'; Value = 'riser' }
            '3' = @{ Label = 'Không biết'; Value = 'unknown' }
        })
        $slot = Read-ChoiceWithDefault -Prompt '3. Khe PCIe đang cắm:' -DefaultKey '4' -Map ([ordered]@{
            '1' = @{ Label = 'Khe x16 nối CPU'; Value = 'cpu_x16' }
            '2' = @{ Label = 'Khe qua chipset'; Value = 'chipset' }
            '3' = @{ Label = 'Khác'; Value = 'other' }
            '4' = @{ Label = 'Không biết'; Value = 'unknown' }
        })
        $stage = Read-ChoiceWithDefault -Prompt '4. Giai đoạn thu thập:' -DefaultKey '5' -Map ([ordered]@{
            '1' = @{ Label = 'Trước khi mở khóa'; Value = 'before_unlock' }
            '2' = @{ Label = 'Sau khi mở khóa'; Value = 'after_unlock' }
            '3' = @{ Label = 'Đang xử lý sự cố'; Value = 'troubleshooting' }
            '4' = @{ Label = 'Card đối chứng (không phải CMP)'; Value = 'baseline' }
            '5' = @{ Label = 'Không biết'; Value = 'unknown' }
        })
        $notes = Read-Host '5. Ghi chú thêm (tối đa 500 ký tự, Enter để bỏ qua)'
        $ctx = [pscustomobject]@{
            mod_state = $modState
            mod_type  = $modType
            riser     = $riser
            slot      = $slot
            stage     = $stage
            notes     = $notes
        }
    } else {
        $ctx = [pscustomobject]@{
            mod_state = 'unknown'
            mod_type  = 'unknown'
            riser     = 'unknown'
            slot      = 'unknown'
            stage     = 'unknown'
            notes     = ''
        }
    }

    $loadBench = $null
    if ($useRunLoadTest) {
        Write-Host 'Đang chạy bài thử tải nhẹ P2 (vgabench CUDA Driver API / PCIe H2D-D2H)...'
        $loadBench = Get-VgaBenchInfo -TargetBdf $targetBdf
    }

    $outDir = if (-not [string]::IsNullOrWhiteSpace($env:VGA_BAT_DIR)) { Join-Path $env:VGA_BAT_DIR 'vga-reports-out' } else { Join-Path (Get-Location).Path 'vga-reports-out' }
    $report = New-Report -Inventory $inventory -Nvidia $nvidia -Context $ctx -ExperimentDir $outDir -TargetBdf $targetBdf -CollectionStartedUtc $collectionStartedUtc -KeepHistory:$useKeepHistory -LoadBench $loadBench

    $rawPreviewJson = $report | ConvertTo-Json -Depth 12
    [void](Redact-ReportObject -InputObject $report)
    $sanitizedPreviewJson = Remove-Pii -Text ($report | ConvertTo-Json -Depth 12)

    Write-Host ''
    Write-Host (Format-VietnameseSummary -Report $report)
    if (-not $report.nvidia.complete) {
        Write-Host "[CẢNH BÁO] nvidia-smi không lấy đủ 18/18 trường NVIDIA cơ bản. Các trường thiếu:"
        Write-Host "  $((@($report.nvidia.missing)) -join ', ')"
    }
    if ($sanitizedPreviewJson -eq $rawPreviewJson) {
        Write-Host '[KIỂM TRA PII] Không phát hiện thông tin cá nhân (đường dẫn người dùng/IP/MAC/serial/UUID).'
    } else {
        Write-Host '[CẢNH BÁO PII] Đã phát hiện và tự động lọc sạch chuỗi nhạy cảm trong báo cáo.'
    }
    Write-Host ''
    if ($useSaveOnly) {
        $action = '1'
    } else {
        Write-Host 'Lựa chọn hành động:'
        Write-Host '  [1] Chỉ lưu báo cáo cục bộ (JSON + summary + ZIP — Offline 100%)'
        Write-Host '  [2] Gửi báo cáo lên GitHub Issue (đồng ý công khai dữ liệu trên)'
        Write-Host '  [3] Thoát không lưu'
        $action = Read-Host 'Nhập lựa chọn (1/2/3)'
    }
    if ($action -eq '1') {
        $saved = Save-LocalReport -Report $report -Dir $outDir -KeepHistory:$useKeepHistory
        Write-Host "Đã lưu báo cáo cục bộ tại:"
        Write-Host "  JSON: $($saved.JsonPath)"
        Write-Host "  ZIP : $($saved.ZipPath)"
    } elseif ($action -eq '2') {
        $sendRes = Send-Report -Report $report -ReportDir $outDir -KeepHistory:$useKeepHistory
        if ($sendRes.Status -in @('sent', 'duplicate') -and -not [string]::IsNullOrWhiteSpace($sendRes.IssueUrl)) {
            $label = if ($sendRes.Status -eq 'duplicate') { 'Báo cáo đã tồn tại (trùng lặp)' } else { 'Đã gửi báo cáo thành công' }
            Write-Host "${label}: $($sendRes.IssueUrl)"
        } else {
            Write-Host "[CHƯA GỬI ĐƯỢC] Trạng thái: $($sendRes.Status)"
            if ($sendRes.Error) { Write-Host "  Chi tiết: $($sendRes.Error)" }
            Write-Host "  Báo cáo và tệp .pending.json vẫn được giữ an toàn trong: $outDir"
            if ($sendRes.Status -in @('offline', 'retry_later')) {
                $fb = Read-Host 'Bạn có muốn mở Issue Form dự phòng trên trình duyệt (chép JSON vào clipboard)? (C/K)'
                if ($fb -match '^(c|co|có|y|yes)$') {
                    Open-FallbackForm -Report $report
                }
            }
        }
    } else {
        Write-Host 'Đã thoát theo yêu cầu.'
    }
}

if ($env:VGA_COLLECTOR_LIB_ONLY -ne '1') {
    Start-CollectorInteractive
}
