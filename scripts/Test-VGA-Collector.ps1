$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Path $PSScriptRoot -Parent
$batPath = Join-Path -Path $repoRoot -ChildPath 'Collect_VGA_Report.bat'
$reportCjsPath = Join-Path -Path $repoRoot -ChildPath 'scripts\vga-report\report.cjs'

$script:Passed = 0
$script:Failed = 0

$loaded = $false
try {
    if (-not (Test-Path -LiteralPath $batPath)) {
        throw "Missing collector batch file: $batPath"
    }
    $raw = [System.IO.File]::ReadAllText($batPath, [System.Text.Encoding]::UTF8)
    $marker = '# === BEGIN_POWERSHELL ==='
    $idx = $raw.LastIndexOf($marker)
    if ($idx -lt 0) {
        throw "Missing '$marker' marker in $batPath"
    }
    $env:VGA_COLLECTOR_LIB_ONLY = '1'
    $code = $raw.Substring($idx + $marker.Length)
    . ([ScriptBlock]::Create($code))
    $loaded = $true
} catch {
    Write-Host "RED load failure: $($_.Exception.Message)"
}

if (-not $loaded) {
    Write-Host "Cannot run collector tests until Collect_VGA_Report.bat implements the PowerShell library."
    exit 1
}

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) {
        throw "Assertion failed: $Message"
    }
}

function Assert-Equal {
    param($Actual, $Expected, [string]$Message)
    if ($Actual -ne $Expected) {
        throw "Assertion failed ($Message): expected '$Expected', got '$Actual'"
    }
}

function Invoke-TestCase {
    param([string]$Name, [scriptblock]$Body)
    try {
        & $Body
        $script:Passed += 1
        Write-Host "PASS $Name"
    } catch {
        $script:Failed += 1
        Write-Host "FAIL $Name : $($_.Exception.Message)"
    }
}

$expectedNvidia18 = @(
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

# =========================================================================
# Task 4 Tests (12 cases)
# =========================================================================

Invoke-TestCase 'inventory_two_gpus_noncontiguous_bdf' {
    function Get-RawCimDisplayDevices {
        return @(
            [pscustomobject]@{ Name = 'AMD Radeon Graphics'; PNPDeviceID = 'PCI\VEN_1002&DEV_1638&SUBSYS_88091043&REV_C9\4&1111&0&0041'; DriverVersion = '31.0.21921.1000'; Status = 'OK'; ConfigManagerErrorCode = 0 },
            [pscustomobject]@{ Name = 'NVIDIA CMP 30HX'; PNPDeviceID = 'PCI\VEN_10DE&DEV_2189&SUBSYS_000010DE&REV_A1\4&2222&0&0009'; DriverVersion = '32.0.15.6094'; Status = 'OK'; ConfigManagerErrorCode = 0 }
        )
    }
    function Get-RawPnpDisplayDevices {
        return @(
            [pscustomobject]@{ InstanceId = 'PCI\VEN_10DE&DEV_2189&SUBSYS_000010DE&REV_A1\4&2222&0&0009'; FriendlyName = 'NVIDIA CMP 30HX'; Status = 'OK' },
            [pscustomobject]@{ InstanceId = 'PCI\VEN_1002&DEV_1638&SUBSYS_88091043&REV_C9\4&1111&0&0041'; FriendlyName = 'AMD Radeon Graphics'; Status = 'OK' }
        )
    }
    function Get-RawPnpDeviceProperties {
        param([string]$InstanceId)
        if ($InstanceId -like '*VEN_10DE*') {
            return @(
                [pscustomobject]@{ KeyName = 'DEVPKEY_Device_BusNumber'; Data = 7 },
                [pscustomobject]@{ KeyName = 'DEVPKEY_Device_Address'; Data = 0 },
                [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_MaxLinkSpeed'; Data = 2 },
                [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_CurrentLinkSpeed'; Data = 2 },
                [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_MaxLinkWidth'; Data = 16 },
                [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_CurrentLinkWidth'; Data = 16 },
                [pscustomobject]@{ KeyName = 'DEVPKEY_Device_DriverVersion'; Data = '32.0.15.6094' }
            )
        }
        return @(
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_BusNumber'; Data = 0 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_Address'; Data = 131072 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_DriverVersion'; Data = '31.0.21921.1000' }
        )
    }

    $inv = @(Get-VgaInventory)
    Assert-Equal $inv.Count 2 'expected 2 GPUs'
    $nv = $inv | Where-Object { $_.vendor_id.value -eq '10DE' } | Select-Object -First 1
    Assert-Equal $nv.bdf.value '0000:07:00.0' 'non-contiguous BDF for NVIDIA'
    Assert-Equal $nv.driver_version.value '32.0.15.6094' 'driver matched by InstanceId not index'
    Assert-Equal $nv.pcie.max_gen.value 2 'pcie max_gen'
}

Invoke-TestCase 'inventory_pnp_present_driver_missing_is_unsupported' {
    function Get-RawCimDisplayDevices { return @() }
    function Get-RawPnpDisplayDevices {
        return @(
            [pscustomobject]@{ InstanceId = 'PCI\VEN_10DE&DEV_2189&SUBSYS_000010DE&REV_A1\4&2222&0&0009'; FriendlyName = 'Microsoft Basic Display Adapter'; Status = 'Error' }
        )
    }
    function Get-RawPnpDeviceProperties {
        param([string]$InstanceId)
        return @(
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_BusNumber'; Data = 1 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_Address'; Data = 0 }
        )
    }

    $inv = @(Get-VgaInventory)
    Assert-Equal $inv.Count 1 'expected 1 GPU'
    Assert-Equal $inv[0].driver_version.status 'unsupported' 'missing driver marked unsupported'
    Assert-True ($null -eq $inv[0].driver_version.value) 'missing driver value must be null'
}

Invoke-TestCase 'no_nvidia_gpu_marks_nvidia_fields_skipped' {
    $inv = @(
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:00:02.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '8086'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '9A49'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '10438670'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = '01'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'Intel UHD Graphics'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '31.0.101.4502'; source = 'cim'; unit = $null; status = 'ok' }
        }
    )
    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.complete $false 'complete false when no NVIDIA'
    Assert-Equal $nv.missing.Count 18 'all 18 fields listed missing'
    Assert-Equal $nv.fields['name'].status 'skipped' 'name skipped'
    Assert-True ($null -eq $nv.fields['memory.total'].value) 'memory.total null'
}

Invoke-TestCase 'nvidia_all_18_fields_ok_sets_complete_true' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:07:00.0, 0x218910DE, 0x000010DE, 6144, 7.5, 2, 2, 16, 16, 42.0, 25.5, 300, 405, 1785, P8'
        }
    }

    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:07:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
    })

    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.complete $true 'complete must be true for 18/18'
    Assert-Equal $nv.missing.Count 0 'missing must be empty'
    Assert-Equal $nv.fields['memory.total'].value 6144 'memory.total parsed'
    Assert-Equal $nv.fields['pci.bus_id'].value '0000:07:00.0' 'normalized bus_id'
}

Invoke-TestCase 'nvidia_missing_field_sets_complete_false_and_lists_it' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:07:00.0, 0x218910DE, 0x000010DE, 6144, [N/A], 2, 2, 16, 16, 42.0, 25.5, 300, 405, 1785, P8'
        }
    }

    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:07:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
    })

    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.complete $false 'complete false when compute_cap is N/A'
    Assert-True ($nv.missing -contains 'compute_cap') 'compute_cap in missing'
}

Invoke-TestCase 'nvidia_na_and_not_supported_map_to_unsupported_not_zero' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:07:00.0, 0x218910DE, 0x000010DE, 6144, [N/A], 2, 2, 16, 16, [Not Supported], [N/A], 300, 405, 1785, P8'
        }
    }

    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:07:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
    })

    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.fields['temperature.gpu'].status 'unsupported' 'Not Supported -> unsupported'
    Assert-True ($null -eq $nv.fields['temperature.gpu'].value) 'temperature.gpu null not 0'
    Assert-Equal $nv.fields['power.draw'].status 'unsupported' '[N/A] -> unsupported'
    Assert-True ($null -eq $nv.fields['power.draw'].value) 'power.draw null not 0'
}

Invoke-TestCase 'nvidia_one_unknown_field_does_not_lose_other_fields' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            # Old driver without compute_cap
            $subset = $expectedNvidia18 | Where-Object { $_ -ne 'compute_cap' }
            return [pscustomobject]@{ ExitCode = 0; Output = ($subset -join "`n") }
        }
        $q = ($Arguments | Where-Object { $_ -like '--query-gpu=*' }) -replace '^--query-gpu=', ''
        if ($q -match 'compute_cap') {
            return [pscustomobject]@{ ExitCode = 2; Output = 'Field "compute_cap" is not a valid field to query.' }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA CMP 30HX, 470.57, 90.16.46.00.01, 00000000:07:00.0, 0x218910DE, 0x000010DE, 6144, 2, 2, 16, 16, 40.0, 22.0, 300, 405, 1785, P8'
        }
    }

    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:07:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '30.0.14.7057'; source = 'cim'; unit = $null; status = 'ok' }
    })

    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.complete $false 'missing compute_cap -> complete false'
    Assert-True ($nv.missing -contains 'compute_cap') 'compute_cap listed in missing'
    Assert-Equal $nv.fields['compute_cap'].status 'unsupported' 'compute_cap unsupported'
    Assert-Equal $nv.fields['name'].value 'NVIDIA CMP 30HX' 'remaining 17 fields preserved'
    Assert-Equal $nv.fields['clocks.max.sm'].value 1785 'clocks.max.sm preserved'
}

Invoke-TestCase 'nvidia_matches_two_gpus_by_bus_id_not_index' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        # Reversed order relative to inventory!
        $row1 = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:08:00.0, 0x218910DE, 0x000010DE, 6144, 7.5, 2, 2, 16, 16, 44.0, 26.0, 300, 405, 1785, P8'
        $row2 = 'NVIDIA GeForce RTX 2070, 560.94, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.0, 40.0, 300, 405, 2175, P8'
        return [pscustomobject]@{ ExitCode = 0; Output = "$row1`n$row2" }
    }

    $inv = @(
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        },
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:08:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        }
    )

    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $inv[0].nvidia.fields['name'].value 'NVIDIA GeForce RTX 2070' 'GPU 0 matched by 0000:01:00.0'
    Assert-Equal $inv[1].nvidia.fields['name'].value 'NVIDIA CMP 30HX' 'GPU 1 matched by 0000:08:00.0'
    Assert-Equal $nv.fields['pci.bus_id'].value '0000:08:00.0' 'CMP target prioritized for top-level nvidia'
}

Invoke-TestCase 'nvidia_smi_not_found_marks_skipped' {
    function Find-NvidiaSmiExecutable { return $null }
    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:07:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
    })

    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.complete $false 'complete false when nvidia-smi missing'
    Assert-Equal $nv.fields['name'].status 'skipped' 'fields marked skipped'
}

Invoke-TestCase 'remove_pii_strips_user_path_mac_ipv4' {
    $raw = 'File at C:\Users\NguyenVanA\Desktop\test.log mac 00:1A:2B:3C:4D:5E ip 192.168.10.22 uuid GPU-11111111-2222-3333-4444-555555555555'
    $clean = Remove-Pii -Text $raw
    Assert-True ($clean -notmatch 'NguyenVanA') 'user path stripped'
    Assert-True ($clean -notmatch '00:1A:2B:3C:4D:5E') 'mac stripped'
    Assert-True ($clean -notmatch '192\.168\.10\.22') 'ipv4 stripped'
    Assert-True ($clean -notmatch 'GPU-11111111') 'gpu uuid stripped'
}

Invoke-TestCase 'report_validates_against_schema' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA GeForce RTX 2070, 621.01, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.7, 40.04, 300, 405, 2175, P8'
        }
    }
    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.16.2101'; source = 'cim'; unit = $null; status = 'ok' }
        pcie = [pscustomobject]@{
            max_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
            current_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
            max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
        }
    })
    $nv = Get-NvidiaInfo -Inventory $inv
    $ctx = [pscustomobject]@{ mod_state = 'stock'; riser = 'direct'; slot = 'cpu_x16'; stage = 'baseline'; notes = 'C:\Users\Secret\file.txt' }
    $rep = New-Report -Inventory $inv -Nvidia $nv -Context $ctx
    $json = $rep | ConvertTo-Json -Depth 12

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("vga-validate-" + [guid]::NewGuid() + ".json")
    try {
        [IO.File]::WriteAllText($tmp, $json, (New-Object Text.UTF8Encoding($false)))
        $nodeOut = & node -e "const fs=require('fs'), r=require(process.argv[1]), d=JSON.parse(fs.readFileSync(process.argv[2],'utf8')); const v=r.validateReport(d), p=r.findPii(d); if(!v.ok || p.length){ console.log(JSON.stringify({v,p})); process.exit(1); }" $reportCjsPath $tmp
        Assert-Equal $LASTEXITCODE 0 "Node schema validation failed: $nodeOut"
    } finally {
        Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'path_with_spaces_accents_and_shell_chars_is_saved' {
    $baseTemp = Join-Path ([IO.Path]::GetTempPath()) ("vga-path-" + [guid]::NewGuid())
    $trickyDir = Join-Path $baseTemp "Báo cáo & thử (x16) 'vga'"
    try {
        $rep = [pscustomobject]@{
            schema_version = 1
            collector_version = '1.0.0'
            report_id = [guid]::NewGuid().ToString()
            created_utc = '2026-11-21T07:00:00Z'
            experiment_id = [guid]::NewGuid().ToString()
            consent = $true
            gpus = @()
            nvidia = [pscustomobject]@{ complete = $false; missing = @(); fields = @{} }
            pcie = [pscustomobject]@{}
            platform = [pscustomobject]@{}
            context = [pscustomobject]@{ mod_state = 'unknown'; riser = 'unknown'; slot = 'unknown'; stage = 'unknown'; notes = '' }
        }
        $saved = Save-LocalReport -Report $rep -Dir $trickyDir
        Assert-True (Test-Path -LiteralPath $saved.JsonPath) 'JsonPath created'
        Assert-True (Test-Path -LiteralPath $saved.SummaryPath) 'SummaryPath created'
        Assert-True (Test-Path -LiteralPath $saved.ZipPath) 'ZipPath created'
    } finally {
        Remove-Item -LiteralPath $baseTemp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# =========================================================================
# Task 5 Tests (12 cases)
# =========================================================================

function Start-MockHttpServer {
    param([scriptblock]$OnRequest)
    $listener = New-Object System.Net.Sockets.TcpListener([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    $port = ([System.Net.IPEndPoint]$listener.LocalEndpoint).Port

    $state = [hashtable]::Synchronized(@{
        Listener = $listener
        Requests = [System.Collections.ArrayList]::Synchronized((New-Object System.Collections.ArrayList))
        Responses = [System.Collections.Queue]::Synchronized((New-Object System.Collections.Queue))
        Stop = $false
    })

    $ps = [powershell]::Create()
    [void]$ps.AddScript({
        param($State)
        while (-not $State.Stop) {
            try {
                if (-not $State.Listener.Pending()) {
                    Start-Sleep -Milliseconds 10
                    continue
                }
                $client = $State.Listener.AcceptTcpClient()
                try {
                    $stream = $client.GetStream()
                    $stream.ReadTimeout = 5000
                    $ms = New-Object System.IO.MemoryStream
                    $buf = New-Object byte[] 4096
                    $headerEnd = -1
                    $rawBytes = [byte[]]@()
                    while ($true) {
                        $read = $stream.Read($buf, 0, $buf.Length)
                        if ($read -le 0) { break }
                        $ms.Write($buf, 0, $read)
                        $rawBytes = $ms.ToArray()
                        $textSoFar = [System.Text.Encoding]::ASCII.GetString($rawBytes)
                        $idx = $textSoFar.IndexOf("`r`n`r`n")
                        if ($idx -ge 0) {
                            $headerEnd = $idx + 4
                            break
                        }
                    }
                    if ($headerEnd -lt 0) { continue }
                    $headerText = [System.Text.Encoding]::ASCII.GetString($rawBytes, 0, $headerEnd)
                    $headerLines = $headerText -split "`r`n"
                    $reqLine = $headerLines[0]
                    $headers = @{}
                    $contentLength = 0
                    for ($i = 1; $i -lt $headerLines.Length; $i += 1) {
                        $line = $headerLines[$i]
                        if ([string]::IsNullOrWhiteSpace($line)) { continue }
                        $colon = $line.IndexOf(':')
                        if ($colon -gt 0) {
                            $k = $line.Substring(0, $colon).Trim()
                            $v = $line.Substring($colon + 1).Trim()
                            $headers[$k] = $v
                            if ($k.Equals('Content-Length', [StringComparison]::OrdinalIgnoreCase)) {
                                $contentLength = [int]$v
                            }
                        }
                    }
                    while ($ms.Length -lt ($headerEnd + $contentLength)) {
                        $read = $stream.Read($buf, 0, $buf.Length)
                        if ($read -le 0) { break }
                        $ms.Write($buf, 0, $read)
                    }
                    $allBytes = $ms.ToArray()
                    $bodyLen = [Math]::Max(0, [Math]::Min($contentLength, $allBytes.Length - $headerEnd))
                    $bodyBytes = New-Object byte[] $bodyLen
                    if ($bodyLen -gt 0) {
                        [Array]::Copy($allBytes, $headerEnd, $bodyBytes, 0, $bodyLen)
                    }
                    $bodyText = [System.Text.Encoding]::UTF8.GetString($bodyBytes)
                    [void]$State.Requests.Add([pscustomobject]@{
                        RequestLine = $reqLine
                        Headers     = $headers
                        BodyBytes   = $bodyBytes
                        BodyText    = $bodyText
                        SecurityProtocolAtReceive = [System.Net.ServicePointManager]::SecurityProtocol
                    })

                    $resp = if ($State.Responses.Count -gt 0) { $State.Responses.Dequeue() } else { @{ StatusCode = 500; Body = '{}' } }
                    if ($resp.DelayMs -and $resp.DelayMs -gt 0) {
                        Start-Sleep -Milliseconds ([int]$resp.DelayMs)
                    }
                    $respBody = if ($resp.Body) { [string]$resp.Body } else { '{}' }
                    $respBytes = [System.Text.Encoding]::UTF8.GetBytes($respBody)
                    $code = [int]$resp.StatusCode
                    $extraHdrs = ''
                    if ($resp.Headers) {
                        foreach ($hk in $resp.Headers.Keys) {
                            $extraHdrs += "${hk}: $($resp.Headers[$hk])`r`n"
                        }
                    }
                    $headStr = "HTTP/1.1 $code OK`r`nContent-Type: application/json; charset=utf-8`r`nContent-Length: $($respBytes.Length)`r`nConnection: close`r`n$extraHdrs`r`n"
                    $headBytes = [System.Text.Encoding]::ASCII.GetBytes($headStr)
                    $stream.Write($headBytes, 0, $headBytes.Length)
                    $stream.Write($respBytes, 0, $respBytes.Length)
                    $stream.Flush()
                } finally {
                    $client.Close()
                }
            } catch {
                if ($State.Stop) { break }
            }
        }
    }).AddArgument($state)
    $async = $ps.BeginInvoke()

    return [pscustomobject]@{
        Url       = "http://127.0.0.1:$port/v1/reports"
        State     = $state
        PowerShell = $ps
        Async     = $async
    }
}

function Stop-MockHttpServer {
    param($Server)
    if (-not $Server) { return }
    $Server.State.Stop = $true
    try { $Server.State.Listener.Stop() } catch {}
    try { $Server.PowerShell.Stop() } catch {}
    try { $Server.PowerShell.Dispose() } catch {}
}

function New-SampleTestReport {
    return [ordered]@{
        schema_version = 1
        collector_version = '1.0.0'
        report_id = [guid]::NewGuid().ToString()
        created_utc = '2026-11-21T07:00:00Z'
        experiment_id = [guid]::NewGuid().ToString()
        consent = $true
        gpus = @()
        nvidia = [ordered]@{ complete = $false; missing = @(); fields = @{} }
        pcie = [ordered]@{}
        platform = [ordered]@{}
        context = [ordered]@{ mod_state = 'unknown'; riser = 'unknown'; slot = 'unknown'; stage = 'unknown'; notes = 'Tiếng Việt thử nghiệm' }
    }
}

$testSignKey = 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA='

Invoke-TestCase 'send_201_returns_sent_and_url' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{
            StatusCode = 201
            Body = "{`"report_id`":`"$($rep.report_id)`",`"status`":`"issue_created`",`"issue_url`":`"https://github.com/ngthaihoc/vga-report-sandbox/issues/101`"}"
        })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -AllowLoopbackHttp
        Assert-Equal $res.Status 'sent' '201 returns sent'
        Assert-Equal $res.IssueUrl 'https://github.com/ngthaihoc/vga-report-sandbox/issues/101' 'returns issue_url'
        Assert-Equal (@(Get-ChildItem -LiteralPath $tmpDir -Filter '*.pending.json').Count) 0 'pending removed after 201'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'send_200_returns_duplicate_same_url' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{
            StatusCode = 200
            Body = "{`"report_id`":`"$($rep.report_id)`",`"status`":`"duplicate`",`"issue_url`":`"https://github.com/ngthaihoc/vga-report-sandbox/issues/101`"}"
        })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -AllowLoopbackHttp
        Assert-Equal $res.Status 'duplicate' '200 returns duplicate'
        Assert-Equal $res.IssueUrl 'https://github.com/ngthaihoc/vga-report-sandbox/issues/101' 'returns same issue_url'
        Assert-Equal (@(Get-ChildItem -LiteralPath $tmpDir -Filter '*.pending.json').Count) 0 'pending removed after 200'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'send_422_returns_rejected_no_retry' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{ StatusCode = 422; Body = '{"error":"pii_detected","categories":["mac"]}' })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -AllowLoopbackHttp
        Assert-Equal $res.Status 'rejected' '422 returns rejected'
        Assert-Equal $srv.State.Requests.Count 1 '422 must not retry'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'send_429_honors_retry_after' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{
            StatusCode = 429
            Headers = @{ 'Retry-After' = '120' }
            Body = '{"error":"rate_limited","reason":"ip"}'
        })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -AllowLoopbackHttp
        Assert-Equal $res.Status 'retry_later' '429 returns retry_later'
        Assert-Equal $res.RetryAfter 120 'RetryAfter parsed from header'
        Assert-Equal (@(Get-ChildItem -LiteralPath $tmpDir -Filter '*.pending.json').Count) 1 'pending kept on 429'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'send_503_keeps_zip_and_returns_retry_later' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{ StatusCode = 503; Body = '{"error":"service_unavailable"}' })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -AllowLoopbackHttp
        Assert-Equal $res.Status 'retry_later' '503 returns retry_later'
        Assert-Equal (@(Get-ChildItem -LiteralPath $tmpDir -Filter '*.zip').Count) 1 'zip kept on 503'
        Assert-Equal (@(Get-ChildItem -LiteralPath $tmpDir -Filter '*.pending.json').Count) 1 'pending kept on 503'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'send_timeout_returns_offline' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{ StatusCode = 201; DelayMs = 2500; Body = '{}' })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -TimeoutSec 1 -AllowLoopbackHttp
        Assert-Equal $res.Status 'offline' 'timeout returns offline'
        Assert-Equal (@(Get-ChildItem -LiteralPath $tmpDir -Filter '*.zip').Count) 1 'zip kept on timeout'
        Assert-Equal (@(Get-ChildItem -LiteralPath $tmpDir -Filter '*.pending.json').Count) 1 'pending kept on timeout'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'endpoint_must_be_https_and_fixed_host' {
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $resHttp = Send-Report -Report $rep -Endpoint 'http://vga.example.workers.dev/v1/reports' -AllowedHost 'vga.example.workers.dev' -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir
        Assert-Equal $resHttp.Status 'rejected' 'http rejected without AllowLoopbackHttp'

        $resWrongHost = Send-Report -Report $rep -Endpoint 'https://evil.example.com/v1/reports' -AllowedHost 'vga.example.workers.dev' -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir
        Assert-Equal $resWrongHost.Status 'rejected' 'mismatched host rejected'
    } finally {
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'tls12_enabled_before_request' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    $origProto = [System.Net.ServicePointManager]::SecurityProtocol
    try {
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{ StatusCode = 201; Body = '{"issue_url":"https://github.com/ngthaihoc/vga-report-sandbox/issues/1"}' })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -AllowLoopbackHttp
        Assert-Equal $res.Status 'sent' 'request succeeded'
        Assert-True (([int]([System.Net.ServicePointManager]::SecurityProtocol) -band [int]([System.Net.SecurityProtocolType]::Tls12)) -ne 0) 'TLS 1.2 enabled'
    } finally {
        [System.Net.ServicePointManager]::SecurityProtocol = $origProto
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'signature_matches_worker_test_vector' {
    $body = '{"hello":"thế giới"}'
    $headers = New-RequestSignature -Body $body -KeyId 'k1' -Key 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=' -Timestamp 1700000000
    Assert-Equal $headers['X-Report-Key-Id'] 'k1' 'key id header'
    Assert-Equal $headers['X-Report-Timestamp'] '1700000000' 'timestamp header'
    Assert-Equal $headers['X-Report-Signature'] 'a1ef303751bfa35a80ffba19567a172abfdd3047b0c9d36ffd4edd5847ddb9a7' 'matches worker vector'
}

Invoke-TestCase 'signature_covers_exact_utf8_body_bytes' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $rep.context.notes = 'Báo cáo tiếng Việt: Đủ 18/18 trường — không lỗi!'
        $srv.State.Responses.Enqueue(@{ StatusCode = 201; Body = '{"issue_url":"https://github.com/ngthaihoc/vga-report-sandbox/issues/5"}' })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -NowUnix 1700000000 -AllowLoopbackHttp
        Assert-Equal $res.Status 'sent' 'sent OK'
        $captured = $srv.State.Requests[0]
        $sha = [System.Security.Cryptography.SHA256]::Create()
        $bodyHashBytes = $sha.ComputeHash($captured.BodyBytes)
        $bodyHashHex = -join ($bodyHashBytes | ForEach-Object { $_.ToString('x2') })
        $hmac = New-Object System.Security.Cryptography.HMACSHA256 (,[Convert]::FromBase64String($testSignKey))
        $expectedSig = -join ($hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes("1700000000.$bodyHashHex")) | ForEach-Object { $_.ToString('x2') })
        Assert-Equal $captured.Headers['X-Report-Signature'] $expectedSig 'signature matches exact transmitted UTF-8 bytes'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'clock_skew_401_retries_once_with_server_time' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{ StatusCode = 401; Body = '{"error":"unauthorized","reason":"clock_skew","server_time":1795245000}' })
        $srv.State.Responses.Enqueue(@{ StatusCode = 201; Body = '{"issue_url":"https://github.com/ngthaihoc/vga-report-sandbox/issues/77"}' })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -NowUnix 1795244000 -AllowLoopbackHttp
        Assert-Equal $res.Status 'sent' 'retried once and succeeded'
        Assert-Equal $res.IssueUrl 'https://github.com/ngthaihoc/vga-report-sandbox/issues/77' 'issue url returned'
        Assert-Equal $srv.State.Requests.Count 2 'exactly 2 requests sent'
        Assert-Equal $srv.State.Requests[1].Headers['X-Report-Timestamp'] '1795245000' 'retry used server_time'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'second_401_returns_rejected_no_loop' {
    $srv = Start-MockHttpServer
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-send-" + [guid]::NewGuid())
    try {
        $rep = New-SampleTestReport
        $srv.State.Responses.Enqueue(@{ StatusCode = 401; Body = '{"error":"unauthorized","reason":"clock_skew","server_time":1795245000}' })
        $srv.State.Responses.Enqueue(@{ StatusCode = 401; Body = '{"error":"unauthorized","reason":"clock_skew","server_time":1795246000}' })
        $srv.State.Responses.Enqueue(@{ StatusCode = 201; Body = '{"issue_url":"https://should.not.reach"}' })
        $res = Send-Report -Report $rep -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -ReportDir $tmpDir -NowUnix 1795244000 -AllowLoopbackHttp
        Assert-Equal $res.Status 'rejected' 'second 401 rejected without looping'
        Assert-Equal $srv.State.Requests.Count 2 'stopped after 2 attempts'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'remove_pii_strips_ipv6_without_changing_pci_bdf' {
    foreach ($ip in @('2001:db8:85a3:0:0:8a2e:370:7334', '2001:db8::1', '::', '::1', '::ffff:192.0.2.1', 'fe80::1%12', 'fe80::1%eth0')) {
        Assert-Equal (Remove-Pii -Text "adresse [$ip]") 'adresse [[IPV6]]' "IPv6 stripped: $ip"
    }
    Assert-Equal (Remove-Pii -Text 'PCI 0000:01:00.0 and 00000000:01:00.0') 'PCI 0000:01:00.0 and 00000000:01:00.0' 'PCI BDF unchanged'
    Assert-Equal (Remove-Pii -Text 'URL https://[2001:db8::1]/ and 2001:db8::2.') 'URL https://[[IPV6]]/ and [IPV6].' 'bracket URL and sentence punctuation redacted'
}

# =========================================================================
# P0 Reliability Tests (8 cases)
# =========================================================================

Invoke-TestCase 'profile_ids_match_40hxcore' {
    Assert-Equal (Get-GpuProfileName -VendorId '10DE' -DeviceId '1F0B') 'CMP 40HX (TU106)' '1F0B maps to CMP 40HX'
    Assert-Equal (Get-GpuProfileName -VendorId '10de' -DeviceId '2189') 'CMP 30HX (TU116)' '2189 case-insensitive maps to CMP 30HX'
    Assert-True ($null -eq (Get-GpuProfileName -VendorId '10DE' -DeviceId '1F06')) '1F06 must not map to CMP 40HX'
    Assert-True ($null -eq (Get-GpuProfileName -VendorId '10DE' -DeviceId '1F07')) '1F07 unknown profile returns null'
}

Invoke-TestCase 'cmp40hx_behind_rtx_is_target' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        $row1 = 'NVIDIA GeForce RTX 2070, 560.94, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.0, 40.0, 300, 405, 2175, P8'
        $row2 = 'NVIDIA CMP 40HX, 560.94, 90.06.59.00.01, 00000000:02:00.0, 0x1F0B10DE, 0x000010DE, 8192, 7.5, 2, 2, 16, 16, 41.0, 28.0, 300, 405, 1710, P8'
        return [pscustomobject]@{ ExitCode = 0; Output = "$row1`n$row2" }
    }
    $inv = @(
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
            pcie = [ordered]@{
                max_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
                current_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
                max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
                current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            }
        },
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:02:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F0B'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA CMP 40HX'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
            pcie = [ordered]@{
                max_gen = [pscustomobject]@{ value = 2; source = 'pnp'; unit = 'gen'; status = 'ok' }
                current_gen = [pscustomobject]@{ value = 2; source = 'pnp'; unit = 'gen'; status = 'ok' }
                max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
                current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            }
        }
    )
    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.fields['pci.bus_id'].value '0000:02:00.0' 'CMP 40HX behind RTX selected as target'
    $ctx = [pscustomobject]@{ mod_state = 'stock'; riser = 'direct'; slot = 'cpu_x16'; stage = 'baseline'; notes = '' }
    $rep = New-Report -Inventory $inv -Nvidia $nv -Context $ctx
    Assert-Equal $rep.pcie.max_gen.value 2 'top-level pcie comes from target CMP 40HX'
}

Invoke-TestCase 'device_1f06_not_treated_as_cmp' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        $row1 = 'NVIDIA GeForce RTX 2070, 560.94, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.0, 40.0, 300, 405, 2175, P8'
        $row2 = 'NVIDIA GeForce RTX 2060 SUPER, 560.94, 90.06.44.00.01, 00000000:02:00.0, 0x1F0610DE, 0x000010DE, 8192, 7.5, 3, 3, 16, 16, 39.0, 35.0, 300, 405, 1650, P8'
        return [pscustomobject]@{ ExitCode = 0; Output = "$row1`n$row2" }
    }
    $inv = @(
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        },
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:02:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F06'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2060 SUPER'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        }
    )
    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.fields['pci.bus_id'].value '0000:01:00.0' '1F06 must not take priority over first 10DE'
}

Invoke-TestCase 'explicit_target_bdf_overrides_auto' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        $row1 = 'NVIDIA GeForce RTX 2070, 560.94, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.0, 40.0, 300, 405, 2175, P8'
        $row2 = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:08:00.0, 0x218910DE, 0x000010DE, 6144, 7.5, 2, 2, 16, 16, 44.0, 26.0, 300, 405, 1785, P8'
        return [pscustomobject]@{ ExitCode = 0; Output = "$row1`n$row2" }
    }
    $inv = @(
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
            pcie = [ordered]@{
                max_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
                current_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
                max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
                current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            }
        },
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:08:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
            pcie = [ordered]@{
                max_gen = [pscustomobject]@{ value = 2; source = 'pnp'; unit = 'gen'; status = 'ok' }
                current_gen = [pscustomobject]@{ value = 2; source = 'pnp'; unit = 'gen'; status = 'ok' }
                max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
                current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            }
        }
    )
    $nv = Get-NvidiaInfo -Inventory $inv -TargetBdf '0000:01:00.0'
    Assert-Equal $nv.fields['pci.bus_id'].value '0000:01:00.0' 'explicit TargetBdf picks RTX over CMP'
    $ctx = [pscustomobject]@{ mod_state = 'stock'; riser = 'direct'; slot = 'cpu_x16'; stage = 'baseline'; notes = '' }
    $rep = New-Report -Inventory $inv -Nvidia $nv -Context $ctx -TargetBdf '0000:01:00.0'
    Assert-Equal $rep.pcie.max_gen.value 3 'top-level pcie matches explicit TargetBdf'
}

Invoke-TestCase 'unknown_target_bdf_falls_back_to_auto' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        $row1 = 'NVIDIA GeForce RTX 2070, 560.94, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.0, 40.0, 300, 405, 2175, P8'
        $row2 = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:08:00.0, 0x218910DE, 0x000010DE, 6144, 7.5, 2, 2, 16, 16, 44.0, 26.0, 300, 405, 1785, P8'
        return [pscustomobject]@{ ExitCode = 0; Output = "$row1`n$row2" }
    }
    $inv = @(
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        },
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:08:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        }
    )
    $nv = Get-NvidiaInfo -Inventory $inv -TargetBdf '0000:09:00.0'
    Assert-Equal $nv.fields['pci.bus_id'].value '0000:08:00.0' 'unknown TargetBdf falls back to CMP auto-selection'
}

Invoke-TestCase 'target_without_nvidia_row_does_not_borrow_other_gpu' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        # Only RTX has an nvidia-smi row; CMP at 0000:02:00.0 is missing from nvidia-smi
        $row1 = 'NVIDIA GeForce RTX 2070, 560.94, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.0, 40.0, 300, 405, 2175, P8'
        return [pscustomobject]@{ ExitCode = 0; Output = $row1 }
    }
    $inv = @(
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        },
        [pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:02:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F0B'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA CMP 40HX'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'Error'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = $null; source = 'cim'; unit = $null; status = 'unsupported' }
        }
    )
    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.complete $false 'target without nvidia row is not complete'
    Assert-Equal $nv.fields['name'].status 'unsupported' 'missing target nvidia block marked unsupported'
    Assert-True ($nv.fields['pci.bus_id'].value -ne '0000:01:00.0') 'must not borrow RTX bus_id'
    $ctx = [pscustomobject]@{ mod_state = 'stock'; riser = 'direct'; slot = 'cpu_x16'; stage = 'baseline'; notes = '' }
    $rep = New-Report -Inventory $inv -Nvidia $nv -Context $ctx
    $json = $rep | ConvertTo-Json -Depth 12
    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("vga-p0-validate-" + [guid]::NewGuid() + ".json")
    try {
        [IO.File]::WriteAllText($tmp, $json, (New-Object Text.UTF8Encoding($false)))
        $nodeOut = & node -e "const fs=require('fs'), r=require(process.argv[1]), d=JSON.parse(fs.readFileSync(process.argv[2],'utf8')); const v=r.validateReport(d), p=r.findPii(d); if(!v.ok || p.length){ console.log(JSON.stringify({v,p})); process.exit(1); }" $reportCjsPath $tmp
        Assert-Equal $LASTEXITCODE 0 "Node schema validation failed: $nodeOut"
    } finally {
        Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'summary_warns_gen_conflict_and_basic_wording' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA GeForce RTX 2070, 621.01, 90.06.18.80.06, 00000000:01:00.0, 0x1F0710DE, 0x86701043, 8192, 7.5, 3, 3, 16, 16, 35.7, 40.04, 300, 405, 2175, P8'
        }
    }
    $invConflict = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.16.2101'; source = 'cim'; unit = $null; status = 'ok' }
        pcie = [ordered]@{
            max_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
            current_gen = [pscustomobject]@{ value = 1; source = 'pnp'; unit = 'gen'; status = 'ok' }
            max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
        }
    })
    $nv = Get-NvidiaInfo -Inventory $invConflict
    $ctx = [pscustomobject]@{ mod_state = 'stock'; riser = 'direct'; slot = 'cpu_x16'; stage = 'baseline'; notes = '' }
    $repConflict = New-Report -Inventory $invConflict -Nvidia $nv -Context $ctx
    $sumConflict = Format-VietnameseSummary -Report $repConflict
    Assert-True ($sumConflict.Contains('Gen1 ×16')) 'normalized Gen1 ×16 in summary'
    Assert-True ($sumConflict.Contains('Gen3 ×16')) 'normalized Gen3 ×16 in summary'
    Assert-True ($sumConflict.Contains('Cảnh báo: PnP')) 'conflict warning rendered'
    Assert-True ($sumConflict.Contains('NVIDIA cơ bản')) 'basic wording rendered'
    Assert-True ($sumConflict.Contains('18/18')) '18/18 count rendered'

    $invSame = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.16.2101'; source = 'cim'; unit = $null; status = 'ok' }
        pcie = [ordered]@{
            max_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
            current_gen = [pscustomobject]@{ value = 3; source = 'pnp'; unit = 'gen'; status = 'ok' }
            max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
        }
    })
    $repSame = New-Report -Inventory $invSame -Nvidia $nv -Context $ctx
    $sumSame = Format-VietnameseSummary -Report $repSame
    Assert-True (-not $sumSame.Contains('Cảnh báo: PnP')) 'no conflict warning when PnP matches nvidia-smi'
}

Invoke-TestCase 'summary_driver_missing_fields_shows_count' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:07:00.0, 0x218910DE, 0x000010DE, 6144, [N/A], 2, 2, 16, 16, 42.0, 25.5, 300, 405, 1785, P8'
        }
    }
    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:07:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
    })
    $nv = Get-NvidiaInfo -Inventory $inv
    $ctx = [pscustomobject]@{ mod_state = 'stock'; riser = 'direct'; slot = 'cpu_x16'; stage = 'baseline'; notes = '' }
    $rep = New-Report -Inventory $inv -Nvidia $nv -Context $ctx
    $sum = Format-VietnameseSummary -Report $rep
    Assert-True ($sum.Contains('17/18')) 'summary shows 17/18 when 1 field is N/A'
}

# =========================================================================
# P1 Schema v2 & Basic Collector Tests (6 cases)
# =========================================================================

Invoke-TestCase 'p1_profiles_cover_p106_p100_and_flag_ambiguous_p100_name' {
    $p106 = Get-GpuProfileDetails -VendorId '10DE' -DeviceId '1C07' -Name 'NVIDIA P106-100'
    Assert-Equal $p106.name 'P106-100 (GP106)' '1C07 profile name'
    Assert-Equal $p106.category 'mining' '1C07 category'
    Assert-Equal $p106.tensor_applicable 'no' 'Pascal tensor not applicable'

    $p106090 = Get-GpuProfileDetails -VendorId '10DE' -DeviceId '1C09' -Name 'P106-090'
    Assert-Equal $p106090.name 'P106-090 (GP106)' '1C09 profile name'

    $p100 = Get-GpuProfileDetails -VendorId '10DE' -DeviceId '15F8' -Name 'Tesla P100-PCIE-16GB'
    Assert-Equal $p100.name 'Tesla P100 PCIe 16GB (GP100)' '15F8 profile name'
    Assert-Equal $p100.target_pcie_gen 3 'P100 target Gen3'
    Assert-Equal $p100.tensor_applicable 'no' 'P100 Pascal tensor not applicable'

    $p100sxm2 = Get-GpuProfileDetails -VendorId '10DE' -DeviceId '15F9' -Name 'Tesla P100-SXM2-16GB'
    Assert-Equal $p100sxm2.name 'Tesla P100 SXM2 (GP100)' '15F9 profile name'

    $ambig = Get-GpuProfileDetails -VendorId '10DE' -DeviceId '9999' -Name 'NVIDIA P100 Custom'
    Assert-True ($null -eq $ambig.name) 'unknown ID must not assume Tesla P100'
    Assert-True ($ambig.identity_note -match 'P100') 'ambiguous P100 name flagged'
}

Invoke-TestCase 'p1_faulty_gpu_property_does_not_drop_healthy_gpu' {
    function Get-RawCimDisplayDevices {
        return @(
            [pscustomobject]@{ Name = 'Healthy CMP 40HX'; PNPDeviceID = 'PCI\VEN_10DE&DEV_1F0B&SUBSYS_000010DE&REV_A1\4&2222&0&0009'; DriverVersion = '32.0.15.6094'; Status = 'OK'; ConfigManagerErrorCode = 0 }
        )
    }
    function Get-RawPnpDisplayDevices {
        return @(
            [pscustomobject]@{ InstanceId = 'PCI\VEN_1002&DEV_1638&SUBSYS_88091043&REV_C9\4&1111&0&0041'; FriendlyName = 'Faulty GPU'; Status = 'Error' },
            [pscustomobject]@{ InstanceId = 'PCI\VEN_10DE&DEV_1F0B&SUBSYS_000010DE&REV_A1\4&2222&0&0009'; FriendlyName = 'NVIDIA CMP 40HX'; Status = 'OK' }
        )
    }
    function Get-RawPnpDeviceProperties {
        param([string]$InstanceId)
        if ($InstanceId -like '*VEN_1002*') {
            throw 'Simulated PnP device property failure'
        }
        return @(
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_BusNumber'; Data = 2 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_Address'; Data = 0 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_DriverProvider'; Data = 'NVIDIA' },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_DriverInfPath'; Data = 'C:\Windows\INF\oem99.inf' },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_Service'; Data = 'nvlddmkm' },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_ProblemCode'; Data = 0 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_Device_LocationPaths'; Data = @('PCIROOT(0)#PCI(0101)#PCI(0000)') },
            [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_MaxLinkSpeed'; Data = 2 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_CurrentLinkSpeed'; Data = 2 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_MaxLinkWidth'; Data = 16 },
            [pscustomobject]@{ KeyName = 'DEVPKEY_PciDevice_CurrentLinkWidth'; Data = 16 }
        )
    }

    $inv = @(Get-VgaInventory)
    Assert-Equal $inv.Count 2 'both GPUs retained even when one fails PnP property read'
    $faulty = $inv | Where-Object { $_.vendor_id.value -eq '1002' } | Select-Object -First 1
    $healthy = $inv | Where-Object { $_.vendor_id.value -eq '10DE' } | Select-Object -First 1
    Assert-Equal $faulty.bdf.status 'error' 'faulty GPU records error status'
    Assert-Equal $healthy.bdf.value '0000:02:00.0' 'healthy GPU still parsed'
    Assert-Equal $healthy.driver.inf_name.value 'oem99.inf' 'INF stripped to filename only'
    Assert-Equal $healthy.driver.provider.value 'NVIDIA' 'driver provider populated'
    Assert-Equal $healthy.driver.problem_code.value 0 'problem_code populated'
    Assert-Equal $healthy.profile.name 'CMP 40HX (TU106)' 'profile attached to GPU'
}

Invoke-TestCase 'p1_security_and_boot_fields_handle_supported_and_unsupported' {
    function Get-RawSecurityState {
        return [pscustomobject]@{
            BootMode        = 'Legacy'
            SecureBootState = $null
            HvciEnabled     = $null
            HvciStatus      = 'permission_denied'
            TestSigning     = $false
        }
    }
    $secLegacy = Get-SecurityAndBootInfo
    Assert-Equal $secLegacy.boot_mode.value 'Legacy' 'Legacy boot mode'
    Assert-Equal $secLegacy.secure_boot.status 'unsupported' 'Secure Boot unsupported on Legacy BIOS'
    Assert-True ($null -eq $secLegacy.secure_boot.value) 'Secure Boot null on unsupported'
    Assert-Equal $secLegacy.hvci_running.status 'permission_denied' 'HVCI permission_denied preserved'
    Assert-Equal $secLegacy.test_signing.value $false 'TestSigning false'

    function Get-RawSecurityState {
        return [pscustomobject]@{
            BootMode        = 'UEFI'
            SecureBootState = 1
            HvciEnabled     = $true
            HvciStatus      = 'ok'
            TestSigning     = $true
        }
    }
    $secUefi = Get-SecurityAndBootInfo
    Assert-Equal $secUefi.boot_mode.value 'UEFI' 'UEFI boot mode'
    Assert-Equal $secUefi.secure_boot.value $true 'Secure Boot enabled'
    Assert-Equal $secUefi.hvci_running.value $true 'HVCI running'
    Assert-Equal $secUefi.test_signing.value $true 'TestSigning true'
}

Invoke-TestCase 'p1_nvidia_extended_power_and_throttle_parsed_without_breaking_basic_18' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            $allHelp = $expectedNvidia18 + @(
                'power.limit','enforced.power.limit','power.default_limit','power.min_limit','power.max_limit',
                'temperature.gpu.tlimit','temperature.memory','fan.speed','utilization.gpu','utilization.memory',
                'memory.used','memory.free','ecc.mode.current','clocks_event_reasons.active',
                'clocks_event_reasons.sw_power_cap','clocks_event_reasons.hw_slowdown',
                'clocks_event_reasons.hw_thermal_slowdown','clocks_event_reasons.hw_power_brake_slowdown'
            )
            return [pscustomobject]@{ ExitCode = 0; Output = ($allHelp -join "`n") }
        }
        $q = ($Arguments | Where-Object { $_ -like '--query-gpu=*' }) -replace '^--query-gpu=', ''
        if ($q -match 'enforced\.power\.limit') {
            return [pscustomobject]@{
                ExitCode = 0
                Output = '00000000:07:00.0, 125.00, 120.00, 125.00, 100.00, 130.00, 44, [N/A], 35, 10, 5, 256, 5888, [N/A], 0x0000000000000004, Active, Not Active, Not Active, Not Active'
            }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA CMP 30HX, 560.94, 90.16.46.00.01, 00000000:07:00.0, 0x218910DE, 0x000010DE, 6144, 7.5, 2, 2, 16, 16, 42.0, 25.5, 300, 405, 1785, P8'
        }
    }
    $inv = @([pscustomobject]@{
        bdf = [pscustomobject]@{ value = '0000:07:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
        vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
        device_id = [pscustomobject]@{ value = '2189'; source = 'pnp'; unit = $null; status = 'ok' }
        subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
        revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
        name = [pscustomobject]@{ value = 'NVIDIA CMP 30HX'; source = 'cim'; unit = $null; status = 'ok' }
        pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
        driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
    })

    $nv = Get-NvidiaInfo -Inventory $inv
    Assert-Equal $nv.complete $true 'basic 18 fields stay complete=true even with extended unsupported fields'
    Assert-Equal $nv.extended['enforced.power.limit'].value 120 'enforced.power.limit parsed'
    Assert-Equal $nv.extended['power.max_limit'].value 130 'power.max_limit parsed'
    Assert-Equal $nv.extended['clocks_throttle_reasons.sw_power_cap'].value 'Active' 'throttle reason mapped from clocks_event_reasons'
    Assert-True ($nv.extended_missing -contains 'ecc.mode.current') 'unsupported extended field listed in extended_missing'
}

Invoke-TestCase 'p1_legacy_status_allowlist_ignores_free_text_and_pii' {
    $tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-legacy-" + [guid]::NewGuid())
    try {
        [void][IO.Directory]::CreateDirectory($tmpDir)
        $statusContent = @"
STATUS_CODE=GEN1_STUCK
SPEED_CURRENT=1
WIDTH_CURRENT=16
TLS_TARGET=2
ERROR_CODE=WINRING0_BLOCKED
DEVICE_ID=2189
MRRS_BYTES=512
OBSERVED_AT=2026-11-21T15:00:00Z
EXTRA_BAD=should_be_ignored
==== 40HX Gen2 Ket qua 2026-11-21 15:00:00 ====
Free text log at C:\Users\SecretOwner\Desktop\installer.log ip 192.168.1.99 mac 00:11:22:33:44:55
"@
        [IO.File]::WriteAllText((Join-Path $tmpDir 'gen2_status.txt'), $statusContent, (New-Object Text.UTF8Encoding($false)))
        $leg = Get-LegacyUnlockStatus -StatusDir $tmpDir
        Assert-Equal $leg.present $true 'legacy status detected'
        Assert-Equal $leg.source 'gen2_status' 'source recorded'
        Assert-Equal $leg.status_code 'GEN1_STUCK' 'status_code parsed'
        Assert-Equal $leg.error_code 'WINRING0_BLOCKED' 'error_code parsed'
        Assert-Equal $leg.speed_current 1 'speed_current parsed'
        Assert-Equal $leg.device_id '2189' 'device_id parsed'
        $legJson = $leg | ConvertTo-Json -Depth 5
        Assert-True (-not $legJson.Contains('SecretOwner')) 'user path never included'
        Assert-True (-not $legJson.Contains('192.168.1.99')) 'IPv4 never included'
        Assert-True (-not $legJson.Contains('EXTRA_BAD')) 'unknown keys ignored'
    } finally {
        Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'p1_v2_report_validates_in_node_and_links_previous_report_id' {
    function Find-NvidiaSmiExecutable { return 'mock-nvidia-smi.exe' }
    function Invoke-NvidiaSmiRaw {
        param([string[]]$Arguments)
        if ($Arguments -contains '--help-query-gpu') {
            return [pscustomobject]@{ ExitCode = 0; Output = ($expectedNvidia18 -join "`n") }
        }
        return [pscustomobject]@{
            ExitCode = 0
            Output = 'NVIDIA CMP 40HX, 560.94, 90.06.59.00.01, 00000000:02:00.0, 0x1F0B10DE, 0x000010DE, 8192, 7.5, 2, 2, 16, 16, 41.0, 28.0, 300, 405, 1710, P8'
        }
    }
    function Get-RawSecurityState {
        return [pscustomobject]@{
            BootMode        = 'UEFI'
            SecureBootState = 0
            HvciEnabled     = $false
            HvciStatus      = 'ok'
            TestSigning     = $false
        }
    }
    $expDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-exp-" + [guid]::NewGuid())
    try {
        [void][IO.Directory]::CreateDirectory($expDir)
        $inv = @([pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:02:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F0B'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA CMP 40HX'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
            profile = (Get-GpuProfileDetails -VendorId '10DE' -DeviceId '1F0B' -Name 'NVIDIA CMP 40HX')
            driver = [ordered]@{
                provider      = [pscustomobject]@{ value = 'NVIDIA'; source = 'pnp'; unit = $null; status = 'ok' }
                date          = [pscustomobject]@{ value = '2026-10-19'; source = 'pnp'; unit = $null; status = 'ok' }
                inf_name      = [pscustomobject]@{ value = 'oem2.inf'; source = 'pnp'; unit = $null; status = 'ok' }
                service       = [pscustomobject]@{ value = 'nvlddmkm'; source = 'pnp'; unit = $null; status = 'ok' }
                problem_code  = [pscustomobject]@{ value = 0; source = 'pnp'; unit = $null; status = 'ok' }
                location_path = [pscustomobject]@{ value = 'PCIROOT(0)#PCI(0101)#PCI(0000)'; source = 'pnp'; unit = $null; status = 'ok' }
            }
            pcie = [ordered]@{
                max_gen = [pscustomobject]@{ value = 2; source = 'pnp'; unit = 'gen'; status = 'ok' }
                current_gen = [pscustomobject]@{ value = 2; source = 'pnp'; unit = 'gen'; status = 'ok' }
                max_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
                current_width = [pscustomobject]@{ value = 16; source = 'pnp'; unit = 'lanes'; status = 'ok' }
            }
        })
        $ctx = [pscustomobject]@{ mod_state = 'modded'; mod_type = 'efi'; riser = 'direct'; slot = 'cpu_x16'; stage = 'after_unlock'; notes = '' }
        $nv1 = Get-NvidiaInfo -Inventory $inv
        $rep1 = New-Report -Inventory $inv -Nvidia $nv1 -Context $ctx -ExperimentDir $expDir -LegacyStatusDir $expDir -KeepHistory
        Assert-Equal $rep1.schema_version 2 'v2 schema emitted'
        Assert-True ($null -eq $rep1.previous_report_id) 'first report has null previous_report_id'
        [void](Save-LocalReport -Report $rep1 -Dir $expDir -KeepHistory)

        $nv2 = Get-NvidiaInfo -Inventory $inv
        $rep2 = New-Report -Inventory $inv -Nvidia $nv2 -Context $ctx -ExperimentDir $expDir -LegacyStatusDir $expDir -KeepHistory
        Assert-Equal $rep2.experiment_id $rep1.experiment_id 'same experiment_id across runs'
        Assert-Equal $rep2.previous_report_id $rep1.report_id 'second report links previous_report_id'

        $json = $rep2 | ConvertTo-Json -Depth 12
        $tmp = Join-Path $expDir 'validate-v2.json'
        [IO.File]::WriteAllText($tmp, $json, (New-Object Text.UTF8Encoding($false)))
        $nodeOut = & node -e "const fs=require('fs'), r=require(process.argv[1]), d=JSON.parse(fs.readFileSync(process.argv[2],'utf8')); const v=r.validateReport(d), p=r.findPii(d); if(!v.ok || p.length){ console.log(JSON.stringify({v,p})); process.exit(1); }" $reportCjsPath $tmp
        Assert-Equal $LASTEXITCODE 0 "Node v2 schema validation failed: $nodeOut"

        $sum1 = Format-VietnameseSummary -Report $rep1
        Assert-True ($sum1.Contains('Mã thử nghiệm:')) 'summary includes experiment_id'
        Assert-True ($sum1.Contains('báo cáo đầu tiên trong chuỗi thử nghiệm')) 'summary marks first report in chain'
        Assert-True ($sum1.Contains('Cửa sổ đo UTC:')) 'summary includes collection window'
        Assert-True ($sum1.Contains('target_pcie_gen=Gen2')) 'summary includes target_pcie_gen'
        Assert-True ($sum1.Contains('tensor_applicable=unverified')) 'summary includes tensor_applicable'
        Assert-True ($sum1.Contains('PCIe(PnP):')) 'summary includes per-GPU PnP PCIe'
        Assert-True ($sum1.Contains('Liên kết PCIe (nvidia-smi):')) 'summary includes nvidia-smi PCIe link'
        Assert-True ($sum1.Contains('Trường NVIDIA mở rộng (power/thermal/throttle):')) 'summary includes extended field count'
        Assert-True ($sum1.Contains('present=false')) 'summary explicitly reports absent legacy status'

        $gen2Txt = @"
STATUS_CODE=GEN2_SUCCESS
ERROR_CODE=NONE
SPEED_CURRENT=2
WIDTH_CURRENT=16
TLS_TARGET=2
DEVICE_ID=1F0B
MRRS_BYTES=512
OBSERVED_AT=2026-11-22T00:00:00Z
"@
        [IO.File]::WriteAllText((Join-Path $expDir 'gen2_status.txt'), $gen2Txt, (New-Object Text.UTF8Encoding($false)))
        $rep3 = New-Report -Inventory $inv -Nvidia $nv2 -Context $ctx -ExperimentDir $expDir -LegacyStatusDir $expDir
        $sum3 = Format-VietnameseSummary -Report $rep3
        Assert-True ($sum3.Contains('device_id=1F0B')) 'summary includes legacy device_id'
        Assert-True ($sum3.Contains('mrrs=512')) 'summary includes legacy mrrs_bytes'
    } finally {
        Remove-Item -LiteralPath $expDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'privacy_default_no_history_does_not_write_tracking_files' {
    $expDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-nohist-" + [guid]::NewGuid())
    try {
        [void][IO.Directory]::CreateDirectory($expDir)
        $origKeep = $env:VGA_KEEP_HISTORY
        $env:VGA_KEEP_HISTORY = $null
        $inv = @([pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.16.2101'; source = 'cim'; unit = $null; status = 'ok' }
        })
        $ctx = [pscustomobject]@{ mod_state = 'unknown'; mod_type = 'unknown'; riser = 'unknown'; slot = 'unknown'; stage = 'unknown'; notes = '' }
        $rep1 = New-Report -Inventory $inv -Nvidia (New-EmptyNvidiaBlock -Status 'skipped') -Context $ctx -ExperimentDir $expDir -LegacyStatusDir $expDir
        [void](Save-LocalReport -Report $rep1 -Dir $expDir)
        $rep2 = New-Report -Inventory $inv -Nvidia (New-EmptyNvidiaBlock -Status 'skipped') -Context $ctx -ExperimentDir $expDir -LegacyStatusDir $expDir

        Assert-True (-not (Test-Path -LiteralPath (Join-Path $expDir 'experiment_id.txt'))) 'experiment_id.txt must not exist by default'
        Assert-True (-not (Test-Path -LiteralPath (Join-Path $expDir 'last_report_id.txt'))) 'last_report_id.txt must not exist by default'
        Assert-True ($null -eq $rep2.previous_report_id) 'previous_report_id stays null without KeepHistory'
        Assert-True ($rep1.experiment_id -ne $rep2.experiment_id) 'each run gets an ephemeral experiment_id without KeepHistory'
    } finally {
        $env:VGA_KEEP_HISTORY = $origKeep
        Remove-Item -LiteralPath $expDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'privacy_redact_report_object_scrubs_mutated_pii_before_save_and_send' {
    $expDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-redact-" + [guid]::NewGuid())
    $srv = Start-MockHttpServer
    try {
        [void][IO.Directory]::CreateDirectory($expDir)
        $rep = New-SampleTestReport
        $rep.context.notes = 'Path C:\Users\Alice\secret.txt IP 192.168.1.50'
        $saved = Save-LocalReport -Report $rep -Dir $expDir
        $savedJson = [IO.File]::ReadAllText($saved.JsonPath, (New-Object Text.UTF8Encoding($false)))
        Assert-True (-not $savedJson.Contains('Alice')) 'user path scrubbed before Save-LocalReport'
        Assert-True (-not $savedJson.Contains('192.168.1.50')) 'IPv4 scrubbed before Save-LocalReport'
        Assert-True ($savedJson.Contains('[PATH_USER]') -and $savedJson.Contains('[IPV4]')) 'PII placeholders stored'

        $v2Inv = @([pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.16.2101'; source = 'cim'; unit = $null; status = 'ok' }
        })
        $v2Rep = New-Report -Inventory $v2Inv -Nvidia (New-EmptyNvidiaBlock -Status 'skipped') -Context ([pscustomobject]@{ mod_state = 'unknown'; mod_type = 'unknown'; riser = 'unknown'; slot = 'unknown'; stage = 'unknown'; notes = '' }) -ExperimentDir $expDir
        $v2Saved = Save-LocalReport -Report $v2Rep -Dir $expDir
        $nodeValOut = & node -e "const fs=require('fs'), r=require(process.argv[1]), d=JSON.parse(fs.readFileSync(process.argv[2],'utf8')); const v=r.validateReport(d), p=r.findPii(d); if(!v.ok || p.length){ console.log(JSON.stringify({v,p})); process.exit(1); }" $reportCjsPath $v2Saved.JsonPath
        Assert-Equal $LASTEXITCODE 0 "Saved v2 report stays schema-valid after Redact-ReportObject: $nodeValOut"

        $repSend = New-SampleTestReport
        $repSend.context.notes = 'MAC 00:11:22:33:44:55 and C:\Users\Bob\test.log'
        $srv.State.Responses.Enqueue(@{ StatusCode = 201; Body = '{"issue_url":"https://github.com/ngthaihoc/vga-report-sandbox/issues/88"}' })
        $res = Send-Report -Report $repSend -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -AllowLoopbackHttp
        Assert-Equal $res.Status 'sent' 'send succeeds with redacted payload'
        $sentBody = $srv.State.Requests[0].BodyText
        Assert-True (-not $sentBody.Contains('Bob')) 'user path scrubbed before Send-Report'
        Assert-True (-not $sentBody.Contains('00:11:22:33:44:55')) 'MAC scrubbed before Send-Report'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $expDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'p2_load_bench_updates_completeness_and_evidence_level' {
    function Invoke-VgaBenchRaw {
        param([string]$TargetBdf)
        return '{"status":"ok","bdf":"0000:02:00.0","device_name":"NVIDIA CMP 40HX","compute_capability":"7.5","driver_version":13050,"transfer_size_mb":64,"iterations":5,"pinned_host":true,"h2d_bandwidth_gbps":5.82,"d2h_bandwidth_gbps":6.14,"fp32_kernel_ok":true,"fp32_gflops":75.5,"duration_ms":320}'
    }
    $expDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-p2-" + [guid]::NewGuid())
    try {
        [void][IO.Directory]::CreateDirectory($expDir)
        $inv = @([pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:02:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F0B'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '000010DE'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA CMP 40HX'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.15.6094'; source = 'cim'; unit = $null; status = 'ok' }
        })
        $ctx = [pscustomobject]@{ mod_state = 'modded'; mod_type = 'efi'; riser = 'direct'; slot = 'cpu_x16'; stage = 'after_unlock'; notes = '' }
        $benchOk = Get-VgaBenchInfo -TargetBdf '0000:02:00.0'
        $repOk = New-Report -Inventory $inv -Nvidia (New-EmptyNvidiaBlock -Status 'skipped') -Context $ctx -ExperimentDir $expDir -LoadBench $benchOk
        Assert-Equal $repOk.evidence_level 'observed' 'P2 ok upgrades evidence_level to observed'
        Assert-Equal $repOk.completeness.link_load.status 'ok' 'link_load marked ok'
        Assert-Equal $repOk.completeness.compute.status 'ok' 'compute marked ok'
        Assert-True ($repOk.completeness.link_load.reason.Contains('5.82 GB/s')) 'H2D bandwidth recorded in link_load reason'
        Assert-True ($repOk.completeness.compute.reason.Contains('75.5 GFLOPS')) 'FP32 GFLOPS recorded in compute reason'

        $savedOk = Save-LocalReport -Report $repOk -Dir $expDir
        $nodeOut = & node -e "const fs=require('fs'), r=require(process.argv[1]), d=JSON.parse(fs.readFileSync(process.argv[2],'utf8')); const v=r.validateReport(d), p=r.findPii(d); if(!v.ok || p.length){ console.log(JSON.stringify({v,p})); process.exit(1); }" $reportCjsPath $savedOk.JsonPath
        Assert-Equal $LASTEXITCODE 0 "P2 ok report passes schema validation: $nodeOut"

        function Invoke-VgaBenchRaw {
            param([string]$TargetBdf)
            return '{"status":"timeout","bdf":"0000:02:00.0","reason":"watchdog_timeout_during_pcie_copy"}'
        }
        $benchTo = Get-VgaBenchInfo -TargetBdf '0000:02:00.0'
        $repTo = New-Report -Inventory $inv -Nvidia (New-EmptyNvidiaBlock -Status 'skipped') -Context $ctx -ExperimentDir $expDir -LoadBench $benchTo
        Assert-Equal $repTo.evidence_level 'reported' 'P2 timeout keeps evidence_level=reported'
        Assert-Equal $repTo.completeness.link_load.status 'error' 'timeout mapped to schema-valid error status'
        Assert-Equal $repTo.completeness.link_load.reason 'watchdog_timeout_during_pcie_copy' 'timeout reason recorded'
    } finally {
        Remove-Item -LiteralPath $expDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# =========================================================================
# Step 0A RED Tests: PII spaced/JSON-escaped paths, fallback clipboard, pending parse, offline 0-network
# =========================================================================

Invoke-TestCase 'remove_pii_handles_spaced_paths_unc_and_json_escaped' {
    $s1 = Remove-Pii -Text 'Error at C:\Users\Nguyen Van A\My Docs\secret log.txt while testing'
    Assert-True ($s1 -notmatch 'Nguyen|Van|My Docs|secret|log\.txt') "spaced Windows user path leaked: $s1"
    Assert-True ($s1 -match 'while testing') "non-PII trailing prose preserved: $s1"

    $s2 = Remove-Pii -Text 'Share \\file server\shared logs\vga debug\out.txt done'
    Assert-True ($s2 -notmatch 'file server|shared logs|vga debug|out\.txt') "spaced UNC path leaked: $s2"

    $s3 = Remove-Pii -Text 'Linux /home/john doe/my folder/test file.txt ok'
    Assert-True ($s3 -notmatch 'john|doe|my folder|test file\.txt') "spaced Unix path leaked: $s3"

    $jsonEscaped = '{"notes":"Seen at C:\\Users\\Nguyen Van A\\Desktop\\a b.txt","unc":"\\\\srv name\\share name\\f.txt"}'
    $s4 = Remove-Pii -Text $jsonEscaped
    Assert-True ($s4 -notmatch 'Nguyen|Van|Desktop|srv name|share name') "JSON-escaped path leaked: $s4"
}

Invoke-TestCase 'redact_preserves_empty_and_single_element_arrays_and_nested_notes' {
    $expDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-arr-" + [guid]::NewGuid())
    try {
        [void][IO.Directory]::CreateDirectory($expDir)
        $inv = @([pscustomobject]@{
            bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
            vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
            device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
            subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
            revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
            name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
            pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
            driver_version = [pscustomobject]@{ value = '32.0.16.2101'; source = 'cim'; unit = $null; status = 'ok' }
        })
        $ctx = [pscustomobject]@{ mod_state = 'stock'; mod_type = 'none'; riser = 'direct'; slot = 'cpu_x16'; stage = 'baseline'; notes = 'C:\Users\Van A\My Folder\n.txt' }
        $rep = New-Report -Inventory $inv -Nvidia (New-EmptyNvidiaBlock -Status 'skipped') -Context $ctx -ExperimentDir $expDir
        $saved = Save-LocalReport -Report $rep -Dir $expDir
        $nodeOut = & node -e "const fs=require('fs'), r=require(process.argv[1]), d=JSON.parse(fs.readFileSync(process.argv[2],'utf8')); if(!Array.isArray(d.gpus) || d.gpus.length!==1 || !Array.isArray(d.completeness.inventory.missing)){ console.log('ARRAY_UNWRAPPED'); process.exit(2); } const v=r.validateReport(d), p=r.findPii(d); if(!v.ok || p.length){ console.log(JSON.stringify({v,p})); process.exit(1); }" $reportCjsPath $saved.JsonPath
        Assert-Equal $LASTEXITCODE 0 "Arrays preserved & validated: $nodeOut"
    } finally {
        Remove-Item -LiteralPath $expDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'open_fallback_form_sanitizes_before_clipboard' {
    $script:CapturedClipboard = $null
    function Set-Clipboard { param([string]$Value) $script:CapturedClipboard = $Value }
    function Start-Process { param([string]$FilePath) }

    $rep = New-SampleTestReport
    $rep.context.notes = 'Path C:\Users\Alice User\Desktop\dump file.txt IP 192.168.1.99'
    Open-FallbackForm -Report $rep
    Assert-True ($null -ne $script:CapturedClipboard) 'clipboard populated from object'
    Assert-True (-not $script:CapturedClipboard.Contains('Alice User')) 'user path scrubbed from clipboard object'
    Assert-True (-not $script:CapturedClipboard.Contains('192.168.1.99')) 'IPv4 scrubbed from clipboard object'

    $rawJson = $rep | ConvertTo-Json -Depth 12
    $rawJson = $rawJson.Replace('[PATH_USER]', 'C:\\Users\\Second User\\x.log')
    $script:CapturedClipboard = $null
    Open-FallbackForm -Report $rawJson
    Assert-True (-not $script:CapturedClipboard.Contains('Second User')) 'user path scrubbed from clipboard string'
}

Invoke-TestCase 'send_report_pending_string_parses_validates_and_sanitizes_and_keeps_corrupt_local' {
    $expDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-pend-" + [guid]::NewGuid())
    $srv = Start-MockHttpServer
    try {
        [void][IO.Directory]::CreateDirectory($expDir)
        $badJsonPath = Join-Path $expDir 'corrupt.pending.json'
        [IO.File]::WriteAllText($badJsonPath, '{bad json', (New-Object Text.UTF8Encoding($false)))
        $resBad = Send-Report -Report '{bad json' -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -AllowLoopbackHttp -ExistingPendingPath $badJsonPath
        Assert-Equal $resBad.Status 'rejected_local' 'corrupt pending JSON rejected locally'
        Assert-Equal $srv.State.Requests.Count 0 'zero HTTP requests on corrupt pending JSON'
        Assert-True (Test-Path -LiteralPath $badJsonPath) 'corrupt pending file kept on disk'

        $badSchemaPath = Join-Path $expDir 'badschema.pending.json'
        $badSchemaText = '{"schema_version":2,"report_id":"not-a-uuid"}'
        [IO.File]::WriteAllText($badSchemaPath, $badSchemaText, (New-Object Text.UTF8Encoding($false)))
        $resSchema = Send-Report -Report $badSchemaText -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -AllowLoopbackHttp -ExistingPendingPath $badSchemaPath
        Assert-Equal $resSchema.Status 'rejected_local' 'invalid schema pending JSON rejected locally'
        Assert-Equal $srv.State.Requests.Count 0 'zero HTTP requests on invalid schema pending JSON'
        Assert-True (Test-Path -LiteralPath $badSchemaPath) 'invalid schema pending file kept on disk'

        $rep = New-SampleTestReport
        $rep.context.notes = 'Pending C:\Users\Pending User\a b.log and 10.0.0.55'
        $rawPending = $rep | ConvertTo-Json -Depth 12
        $goodPendingPath = Join-Path $expDir 'good.pending.json'
        [IO.File]::WriteAllText($goodPendingPath, $rawPending, (New-Object Text.UTF8Encoding($false)))
        $srv.State.Responses.Enqueue(@{ StatusCode = 201; Body = '{"issue_url":"https://github.com/ngthaihoc/vga-report-sandbox/issues/91"}' })
        $resOk = Send-Report -Report $rawPending -Endpoint $srv.Url -KeyId 'k1' -Key $testSignKey -AllowLoopbackHttp -ExistingPendingPath $goodPendingPath
        Assert-Equal $resOk.Status 'sent' 'valid pending sent after structured sanitize'
        Assert-Equal $srv.State.Requests.Count 1 'one HTTP request made for valid pending'
        $sentBody = $srv.State.Requests[0].BodyText
        Assert-True (-not $sentBody.Contains('Pending User')) 'pending spaced path redacted'
        Assert-True (-not $sentBody.Contains('10.0.0.55')) 'pending IPv4 redacted'
        Assert-True (-not (Test-Path -LiteralPath $goodPendingPath)) 'valid pending cleaned up after 201'
    } finally {
        Stop-MockHttpServer $srv
        Remove-Item -LiteralPath $expDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Invoke-TestCase 'save_only_and_choice_1_zero_network_even_with_pending' {
    $expDir = Join-Path ([IO.Path]::GetTempPath()) ("vga-offline-" + [guid]::NewGuid())
    $outSub = Join-Path $expDir 'vga-reports-out'
    $origDir = $env:VGA_BAT_DIR
    $script:NetworkCalls = 0
    try {
        [void][IO.Directory]::CreateDirectory($outSub)
        $pendFile = Join-Path $outSub 'old.pending.json'
        $samplePend = (New-SampleTestReport) | ConvertTo-Json -Depth 12
        [IO.File]::WriteAllText($pendFile, $samplePend, (New-Object Text.UTF8Encoding($false)))

        $env:VGA_BAT_DIR = $expDir
        function Get-VgaInventory {
            return @([pscustomobject]@{
                bdf = [pscustomobject]@{ value = '0000:01:00.0'; source = 'pnp'; unit = $null; status = 'ok' }
                vendor_id = [pscustomobject]@{ value = '10DE'; source = 'pnp'; unit = $null; status = 'ok' }
                device_id = [pscustomobject]@{ value = '1F07'; source = 'pnp'; unit = $null; status = 'ok' }
                subsystem_id = [pscustomobject]@{ value = '86701043'; source = 'pnp'; unit = $null; status = 'ok' }
                revision = [pscustomobject]@{ value = 'A1'; source = 'pnp'; unit = $null; status = 'ok' }
                name = [pscustomobject]@{ value = 'NVIDIA GeForce RTX 2070'; source = 'cim'; unit = $null; status = 'ok' }
                pnp_status = [pscustomobject]@{ value = 'OK'; source = 'pnp'; unit = $null; status = 'ok' }
                driver_version = [pscustomobject]@{ value = '32.0.16.2101'; source = 'cim'; unit = $null; status = 'ok' }
            })
        }
        function Get-NvidiaInfo { param($Inventory, $TargetBdf) return (New-EmptyNvidiaBlock -Status 'skipped') }
        function Invoke-HttpPostBytes { $script:NetworkCalls += 1; throw 'MUST_NOT_CALL_NETWORK_OFFLINE' }
        function Read-Host { param($Prompt) $script:Prompts += $Prompt; return '1' }

        $script:Prompts = @()
        Start-CollectorInteractive -SaveOnly
        Assert-Equal $script:NetworkCalls 0 '-SaveOnly makes 0 network calls even with pending.json'
        Assert-Equal $script:Prompts.Count 0 '-SaveOnly makes 0 Read-Host prompts even with pending.json'

        $script:Prompts = @()
        Start-CollectorInteractive
        Assert-Equal $script:NetworkCalls 0 'Choice [1] makes 0 network calls even with pending.json'
        Assert-Equal $script:Prompts.Count 1 'Interactive choice [1] has only 1 prompt even with pending.json'
    } finally {
        $env:VGA_BAT_DIR = $origDir
        Remove-Item -LiteralPath $expDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "Summary: Passed=$script:Passed Failed=$script:Failed"
if ($script:Failed -gt 0) { exit 1 }
exit 0
