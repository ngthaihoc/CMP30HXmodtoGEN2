@echo off
setlocal DisableDelayedExpansion
chcp 65001 >nul
title Thu thap bao cao VGA - CMP30HXmodtoGEN2

if /i not "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    if /i not "%PROCESSOR_ARCHITEW6432%"=="AMD64" (
        echo [LOI] Cong cu nay chi ho tro Windows 10/11 64-bit ^(x64^).
        pause
        exit /b 1
    )
)

set "VGA_BAT_SELF=%~f0"
set "VGA_BAT_DIR=%~dp0"
set "VGA_COLLECTOR_LIB_ONLY=0"

powershell.exe -NoProfile -Command "& { $raw = [System.IO.File]::ReadAllText($env:VGA_BAT_SELF, [System.Text.Encoding]::UTF8); $m = '# === BEGIN_POWERSHELL ==='; $idx = $raw.LastIndexOf($m); if ($idx -lt 0) { Write-Error 'Khong tim thay khoi PowerShell trong file BAT.'; exit 1 }; $code = $raw.Substring($idx + $m.Length); . ([ScriptBlock]::Create($code)) }"
set "EXIT_CODE=%ERRORLEVEL%"
exit /b %EXIT_CODE%

# === BEGIN_POWERSHELL ===
$ErrorActionPreference = 'Stop'

if ($PSVersionTable.PSVersion.Major -lt 5 -or ($PSVersionTable.PSVersion.Major -eq 5 -and $PSVersionTable.PSVersion.Minor -lt 1)) {
    throw 'Yêu cầu Windows PowerShell 5.1 trở lên.'
}
if (-not [Environment]::Is64BitOperatingSystem) {
    throw 'Yêu cầu Windows 10/11 64-bit (x64).'
}

$script:CollectorVersion = '1.0.0'
$script:SchemaVersion = 1
$script:TargetHostname = 'chua-cau-hinh.workers.dev'

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
    $out = [regex]::Replace($out, '(?<![.\dA-Za-z])(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]\d|\d)(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]\d|\d)){3}(?![.\dA-Za-z])', '[IPV4]')
    $out = [regex]::Replace($out, '(?i)\b(?:MIG-)?GPU-[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b', '[UUID_GPU]')
    $out = [regex]::Replace($out, '(?i)\b(?:serial(?:\s*(?:number|no|#))?|s/n|sn)\s*[:=]\s*[A-Za-z0-9_-]{4,}', '[SERIAL]')
    return $out
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
    $pnpList = @(Get-RawPnpDisplayDevices)

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
        $props = @(Get-RawPnpDeviceProperties -InstanceId $p.InstanceId)
        $propMap = @{}
        foreach ($pr in $props) {
            if ($pr -and $pr.KeyName) { $propMap[$pr.KeyName] = $pr.Data }
        }
        $records += [pscustomobject]@{
            InstanceId = $p.InstanceId
            Pnp        = $p
            Cim        = $cim
            Props      = $propMap
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
            }
        }
    }

    $gpus = @()
    foreach ($rec in $records) {
        $ids = Parse-PciIdsFromInstanceId -InstanceId $rec.InstanceId
        $propMap = $rec.Props

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

        $gpus += [pscustomobject]@{
            bdf            = New-ReportField -Value $bdfStr -Source 'pnp'
            vendor_id      = New-ReportField -Value $ids.Vendor -Source 'pnp'
            device_id      = New-ReportField -Value $ids.Device -Source 'pnp'
            subsystem_id   = New-ReportField -Value $ids.Subsys -Source 'pnp'
            revision       = New-ReportField -Value $ids.Revision -Source 'pnp'
            name           = New-ReportField -Value $nameVal -Source $nameSrc
            pnp_status     = New-ReportField -Value $pnpStatusVal -Source 'pnp'
            driver_version = New-ReportField -Value $drvVal -Source $drvSrc
            pcie           = [ordered]@{
                max_gen       = New-ReportField -Value $maxGen -Source 'pnp' -Unit 'gen'
                current_gen   = New-ReportField -Value $curGen -Source 'pnp' -Unit 'gen'
                max_width     = New-ReportField -Value $maxW -Source 'pnp' -Unit 'lanes'
                current_width = New-ReportField -Value $curW -Source 'pnp' -Unit 'lanes'
            }
            nvidia         = $null
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
            if ($FieldName -in @('memory.total','pcie.link.gen.max','pcie.link.gen.gpucurrent','pcie.link.width.max','pcie.link.width.current','clocks.sm','clocks.mem','clocks.max.sm')) {
                return New-ReportField -Value ([int][Math]::Round($num)) -Source 'nvidia-smi' -Unit $unit -Status 'ok'
            }
            return New-ReportField -Value $num -Source 'nvidia-smi' -Unit $unit -Status 'ok'
        }
        return New-ReportField -Value $null -Source 'nvidia-smi' -Unit $unit -Status 'unsupported'
    }
    return New-ReportField -Value $t -Source 'nvidia-smi' -Unit $unit -Status 'ok'
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
    return [ordered]@{
        complete = $false
        missing  = $missing.ToArray()
        fields   = $fields
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

function Select-TargetNvidiaBlock {
    param([object[]]$Inventory, [hashtable]$BlocksByBdf)
    # Ưu tiên CMP 30HX (2189) hoặc CMP 40HX (1F06), rồi bất kỳ GPU NVIDIA nào khớp BDF
    foreach ($gpu in $Inventory) {
        if ($gpu.vendor_id.value -eq '10DE' -and $gpu.device_id.value -in @('2189', '1F06')) {
            $bdf = Normalize-Bdf -BusId $gpu.bdf.value
            if ($bdf -and $BlocksByBdf.ContainsKey($bdf)) { return $BlocksByBdf[$bdf] }
        }
    }
    foreach ($gpu in $Inventory) {
        if ($gpu.vendor_id.value -eq '10DE') {
            $bdf = Normalize-Bdf -BusId $gpu.bdf.value
            if ($bdf -and $BlocksByBdf.ContainsKey($bdf)) { return $BlocksByBdf[$bdf] }
        }
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
    param([Parameter(Mandatory = $false)]$Inventory)
    $invArray = @($Inventory)
    $hasNvidia = $false
    foreach ($g in $invArray) {
        if ($g -and $g.vendor_id -and $g.vendor_id.value -eq '10DE') {
            $hasNvidia = $true
            break
        }
    }
    if (-not $hasNvidia) {
        return New-EmptyNvidiaBlock -Status 'skipped'
    }

    $exe = Find-NvidiaSmiExecutable
    if (-not $exe) {
        return New-EmptyNvidiaBlock -Status 'skipped'
    }

    $rowsByBdf = @{}
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

    if ($rowsByBdf.Count -eq 0) {
        return New-EmptyNvidiaBlock -Status 'error'
    }

    $blocksByBdf = @{}
    foreach ($bdfKey in $rowsByBdf.Keys) {
        $blocksByBdf[$bdfKey] = Build-NvidiaBlockFromMap -RawMap $rowsByBdf[$bdfKey]
    }

    foreach ($gpu in $invArray) {
        if ($gpu -and $gpu.bdf -and $gpu.bdf.value) {
            $normBdf = Normalize-Bdf -BusId ([string]$gpu.bdf.value)
            if ($normBdf -and $blocksByBdf.ContainsKey($normBdf)) {
                $gpu | Add-Member -MemberType NoteProperty -Name 'nvidia' -Value $blocksByBdf[$normBdf] -Force
            }
        }
    }

    return Select-TargetNvidiaBlock -Inventory $invArray -BlocksByBdf $blocksByBdf
}

function Get-OrCreateExperimentId {
    param([AllowNull()][string]$BaseDir)
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
            $newId = [guid]::NewGuid().ToString()
            [System.IO.File]::WriteAllText($idPath, $newId, (New-Object System.Text.UTF8Encoding($false)))
            return $newId
        } catch {}
    }
    return [guid]::NewGuid().ToString()
}

function New-Report {
    param(
        [Parameter(Mandatory = $false)]$Inventory,
        [Parameter(Mandatory = $false)]$Nvidia,
        [Parameter(Mandatory = $false)]$Context,
        [AllowNull()][string]$ExperimentDir = $null
    )
    $invArray = @($Inventory)
    $gpuEntries = New-Object System.Collections.ArrayList
    foreach ($g in $invArray) {
        $entry = [ordered]@{
            bdf            = $g.bdf
            vendor_id      = $g.vendor_id
            device_id      = $g.device_id
            subsystem_id   = $g.subsystem_id
            revision       = $g.revision
            name           = $g.name
            pnp_status     = $g.pnp_status
            driver_version = $g.driver_version
        }
        if ($null -ne $g.pcie) { $entry['pcie'] = $g.pcie }
        if ($null -ne $g.nvidia) { $entry['nvidia'] = $g.nvidia }
        [void]$gpuEntries.Add($entry)
    }

    $primaryGpu = $null
    foreach ($g in $invArray) {
        if ($g.vendor_id.value -eq '10DE' -and $g.device_id.value -in @('2189', '1F06')) { $primaryGpu = $g; break }
    }
    if (-not $primaryGpu) {
        foreach ($g in $invArray) {
            if ($g.vendor_id.value -eq '10DE') { $primaryGpu = $g; break }
        }
    }
    if (-not $primaryGpu -and $invArray.Count -gt 0) { $primaryGpu = $invArray[0] }

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
    $notesClean = if ($Context -and $Context.notes) { Remove-Pii -Text ([string]$Context.notes) } else { '' }
    if ($notesClean.Length -gt 500) { $notesClean = $notesClean.Substring(0, 500) }

    $ctxObj = [ordered]@{
        mod_state = if ($Context -and $Context.mod_state) { [string]$Context.mod_state } else { 'unknown' }
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

    return [ordered]@{
        schema_version    = $script:SchemaVersion
        collector_version = $script:CollectorVersion
        report_id         = [guid]::NewGuid().ToString()
        created_utc       = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
        experiment_id     = Get-OrCreateExperimentId -BaseDir $ExperimentDir
        consent           = $true
        gpus              = $gpuEntries.ToArray()
        nvidia            = $nvBlock
        pcie              = $topPcie
        platform          = [ordered]@{
            cpu             = New-ReportField -Value $plat.Cpu -Source 'cim'
            mainboard_model = New-ReportField -Value $plat.Mainboard -Source 'cim'
            bios_version    = New-ReportField -Value $plat.Bios -Source 'cim'
            os_build        = New-ReportField -Value $plat.OsBuild -Source 'cim'
        }
        context           = $ctxObj
    }
}

function Format-VietnameseSummary {
    param($Report)
    $sb = New-Object System.Text.StringBuilder
    [void]$sb.AppendLine('=== TOM TAT BAO CAO CHAN DOAN VGA ===')
    [void]$sb.AppendLine("Ma bao cao   : $($Report.report_id)")
    [void]$sb.AppendLine("Thoi gian UTC: $($Report.created_utc)")
    [void]$sb.AppendLine("Phien ban    : $($Report.collector_version)")
    [void]$sb.AppendLine('')
    [void]$sb.AppendLine('--- GPU phat hien ---')
    foreach ($g in @($Report.gpus)) {
        $bdf = if ($g.bdf.status -eq 'ok') { $g.bdf.value } else { $g.bdf.status }
        $name = if ($g.name.status -eq 'ok') { $g.name.value } else { $g.name.status }
        $vid = if ($g.vendor_id.status -eq 'ok') { $g.vendor_id.value } else { '?' }
        $did = if ($g.device_id.status -eq 'ok') { $g.device_id.value } else { '?' }
        $drv = if ($g.driver_version.status -eq 'ok') { $g.driver_version.value } else { $g.driver_version.status }
        [void]$sb.AppendLine("  [$bdf] $name ($vid`:$did) | Driver: $drv")
    }
    [void]$sb.AppendLine('')
    if ($Report.nvidia) {
        [void]$sb.AppendLine("nvidia-smi du 18/18 truong: $($Report.nvidia.complete)")
        if (@($Report.nvidia.missing).Count -gt 0) {
            [void]$sb.AppendLine("Cac truong thieu / khong ho tro: $((@($Report.nvidia.missing)) -join ', ')")
        }
    }
    return $sb.ToString()
}

function Save-LocalReport {
    param(
        [Parameter(Mandatory = $true)]$Report,
        [Parameter(Mandatory = $true)][string]$Dir
    )
    [void][System.IO.Directory]::CreateDirectory($Dir)
    $stamp = (Get-Date).ToUniversalTime().ToString('yyyyMMdd-HHmmss')
    $shortId = ([string]$Report.report_id).Split('-')[0]
    $baseName = "vga-report-$stamp-$shortId"

    $jsonPath = Join-Path -Path $Dir -ChildPath "$baseName.json"
    $summaryPath = Join-Path -Path $Dir -ChildPath "$baseName-summary.txt"
    $zipPath = Join-Path -Path $Dir -ChildPath "$baseName.zip"

    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    $jsonText = $Report | ConvertTo-Json -Depth 12
    $summaryText = Format-VietnameseSummary -Report $Report

    [System.IO.File]::WriteAllText($jsonPath, $jsonText, $utf8NoBom)
    [System.IO.File]::WriteAllText($summaryPath, $summaryText, $utf8NoBom)

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
    }
}

function Read-ChoiceWithDefault {
    param([string]$Prompt, [hashtable]$Map, [string]$DefaultKey)
    Write-Host $Prompt
    foreach ($k in $Map.Keys) {
        Write-Host "  [$k] $($Map[$k].Label)"
    }
    $ans = Read-Host "Chon (mac dinh [$DefaultKey])"
    if ([string]::IsNullOrWhiteSpace($ans)) { return $Map[$DefaultKey].Value }
    $key = $ans.Trim()
    if ($Map.ContainsKey($key)) { return $Map[$key].Value }
    return $Map[$DefaultKey].Value
}

function Start-CollectorInteractive {
    Write-Host '================================================================'
    Write-Host '  THU THAP BAO CAO CHAN DOAN VGA (CHI DOC - KHONG GHI HE THONG)'
    Write-Host '================================================================'
    Write-Host 'Cong cu chi doc thong tin VGA/PCIe/nen tang (khong sua registry,'
    Write-Host 'khong sua MMIO, khong cai dat Persistence, khong can Admin).'
    Write-Host "May chu dich du kien: $($script:TargetHostname)"
    Write-Host 'Chinh sach du lieu: Issue GitHub la CONG KHAI. Khong thu thap'
    Write-Host 'ten nguoi dung, ten may, duong dan ca nhan, IP/MAC, serial, UUID GPU.'
    Write-Host ''

    Write-Host 'Dang quet thiet bi VGA va nvidia-smi...'
    $inventory = @(Get-VgaInventory)
    $nvidia = Get-NvidiaInfo -Inventory $inventory

    $modState = Read-ChoiceWithDefault -Prompt '1. Trang thai han tro / VBIOS cua card:' -DefaultKey '3' -Map ([ordered]@{
        '1' = @{ Label = 'Nguyen ban (stock)'; Value = 'stock' }
        '2' = @{ Label = 'Da mod (modded)'; Value = 'modded' }
        '3' = @{ Label = 'Khong biet'; Value = 'unknown' }
    })
    $riser = Read-ChoiceWithDefault -Prompt '2. Cach cam card:' -DefaultKey '3' -Map ([ordered]@{
        '1' = @{ Label = 'Cam truc tiep vao mainboard'; Value = 'direct' }
        '2' = @{ Label = 'Dung day riser'; Value = 'riser' }
        '3' = @{ Label = 'Khong biet'; Value = 'unknown' }
    })
    $slot = Read-ChoiceWithDefault -Prompt '3. Khe PCIe dang cam:' -DefaultKey '4' -Map ([ordered]@{
        '1' = @{ Label = 'Khe x16 noi CPU'; Value = 'cpu_x16' }
        '2' = @{ Label = 'Khe qua chipset'; Value = 'chipset' }
        '3' = @{ Label = 'Khac'; Value = 'other' }
        '4' = @{ Label = 'Khong biet'; Value = 'unknown' }
    })
    $stage = Read-ChoiceWithDefault -Prompt '4. Giai doan thu thap:' -DefaultKey '5' -Map ([ordered]@{
        '1' = @{ Label = 'Truoc khi mo khoa'; Value = 'before_unlock' }
        '2' = @{ Label = 'Sau khi mo khoa'; Value = 'after_unlock' }
        '3' = @{ Label = 'Dang xu ly su co'; Value = 'troubleshooting' }
        '4' = @{ Label = 'Card doi chung (khong phai CMP)'; Value = 'baseline' }
        '5' = @{ Label = 'Khong biet'; Value = 'unknown' }
    })
    $notes = Read-Host '5. Ghi chu them (toi da 500 ky tu, Enter de bo qua)'

    $ctx = [pscustomobject]@{
        mod_state = $modState
        riser     = $riser
        slot      = $slot
        stage     = $stage
        notes     = $notes
    }

    $outDir = if (-not [string]::IsNullOrWhiteSpace($env:VGA_BAT_DIR)) { Join-Path $env:VGA_BAT_DIR 'vga-reports-out' } else { Join-Path (Get-Location).Path 'vga-reports-out' }
    $report = New-Report -Inventory $inventory -Nvidia $nvidia -Context $ctx -ExperimentDir $outDir

    Write-Host ''
    Write-Host (Format-VietnameseSummary -Report $report)
    if (-not $report.nvidia.complete) {
        Write-Host "[CANH BAO] nvidia-smi khong lay du 18/18 truong. Cac truong thieu:"
        Write-Host "  $((@($report.nvidia.missing)) -join ', ')"
    }
    Write-Host ''
    Write-Host 'Lua chon hanh dong:'
    Write-Host '  [1] Chi luu bao cao cuc bo (JSON + summary + ZIP)'
    Write-Host '  [2] Gui bao cao len GitHub Issue (dong y cong khai du lieu tren)'
    Write-Host '  [3] Thoat khong luu'
    $action = Read-Host 'Nhap lua chon (1/2/3)'
    if ($action -eq '1') {
        $saved = Save-LocalReport -Report $report -Dir $outDir
        Write-Host "Da luu bao cao cuc bo tai:"
        Write-Host "  JSON: $($saved.JsonPath)"
        Write-Host "  ZIP : $($saved.ZipPath)"
    } elseif ($action -eq '2') {
        $saved = Save-LocalReport -Report $report -Dir $outDir
        Write-Host "Chua cau hinh endpoint gui tu dong trong phien ban co ban nay."
        Write-Host "Bao cao da duoc luu an toan tai: $($saved.ZipPath)"
    } else {
        Write-Host 'Da thoat theo yeu cau.'
    }
}

if ($env:VGA_COLLECTOR_LIB_ONLY -ne '1') {
    Start-CollectorInteractive
}
