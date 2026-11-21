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

Write-Host "Summary: Passed=$script:Passed Failed=$script:Failed"
if ($script:Failed -gt 0) { exit 1 }
exit 0
