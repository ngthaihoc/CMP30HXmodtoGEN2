# Script cấu hình tối ưu GPU High Performance và Cross-Adapter Scan-Out (CASO) cho CMP 40HX / 30HX

# ================================================================
# 1. DÒ TÌM DRIVER CLASS INDEX ĐỘNG (THAY VÌ HARDCODE 0001)
# ================================================================
$classBase = "HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}"
$targetKey = $null

if (Test-Path $classBase) {
    $subkeys = Get-ChildItem -Path $classBase -ErrorAction SilentlyContinue | Where-Object { $_.PSChildName -match '^\d{4}$' }
    foreach ($sk in $subkeys) {
        $props = Get-ItemProperty -Path $sk.PSPath -ErrorAction SilentlyContinue
        $matchId = "$($props.MatchingDeviceId)"
        $driverDesc = "$($props.DriverDesc)"
        $provider = "$($props.ProviderName)"
        $combined = ("$matchId $driverDesc $provider").ToUpper()

        # Nhận diện NVIDIA CMP 40HX / 30HX (TU106 / TU116)
        if ($combined -match 'DEV_1E38|DEV_1F08|DEV_1F0B|DEV_2187|DEV_2184|DEV_2189' -or
            $combined -match 'CMP 30HX|CMP 40HX|TU116|TU106' -or
            ($combined -match 'NVIDIA' -and $combined -notmatch 'BASIC DISPLAY|MICROSOFT')) {
            $targetKey = $sk.PSPath
            break
        }
    }
}

# Nếu không dò được card cụ thể, fallback về 0001
if (-not $targetKey -and (Test-Path "$classBase\0001")) {
    $targetKey = "$classBase\0001"
}

# Áp dụng CASO và cờ chống ngủ đông vào Driver Class Index
if ($targetKey -and (Test-Path $targetKey)) {
    Set-ItemProperty -Path $targetKey -Name "D3ColdSupported" -Value 0 -Type DWord -Force -ErrorAction SilentlyContinue
    Set-ItemProperty -Path $targetKey -Name "RmDisableGpuPowerMgmt" -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    Set-ItemProperty -Path $targetKey -Name "EnableCrossAdapterScanOut" -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    Set-ItemProperty -Path $targetKey -Name "EnableMsHybrid" -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    Set-ItemProperty -Path $targetKey -Name "EnableCoproc" -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path $targetKey -Name "AdapterType" -Force -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path $targetKey -Name "LargePageMinimum" -Force -ErrorAction SilentlyContinue
}

# ================================================================
# 2. CẤU HÌNH USER GPU PREFERENCES (DIRECTX 11/12 HIGH PERFORMANCE)
# ================================================================
$reg = 'HKCU:\Software\Microsoft\DirectX\UserGpuPreferences'
if (-not (Test-Path $reg)) {
    New-Item -Path $reg -Force | Out-Null
}

$found = 0
$drives = (Get-PSDrive -PSProvider FileSystem).Root
$targets = @(
    'Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe',
    'Riot Games\League of Legends\Game\League of Legends.exe',
    'Riot Games\League of Legends\LeagueClient.exe',
    'Riot Games\Riot Client\RiotClientServices.exe'
)

foreach ($d in $drives) {
    foreach ($sub in $targets) {
        $p = Join-Path $d $sub
        if (Test-Path $p) {
            Set-ItemProperty -Path $reg -Name $p -Value 'GpuPreference=2;' -ErrorAction SilentlyContinue
            $found++
        }
    }
}

if ($found -eq 0) {
    Set-ItemProperty -Path $reg -Name (Join-Path $env:SystemDrive 'Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe') -Value 'GpuPreference=2;' -ErrorAction SilentlyContinue
    Set-ItemProperty -Path $reg -Name (Join-Path $env:SystemDrive 'Riot Games\League of Legends\Game\League of Legends.exe') -Value 'GpuPreference=2;' -ErrorAction SilentlyContinue
}

# ================================================================
# 3. TỐI ƯU LEAGUE OF LEGENDS GAME.CFG (BORDERLESS WINDOWED CHO DWM)
# ================================================================
$cfgRelativePaths = @(
    'Riot Games\League of Legends\Config\game.cfg',
    'League of Legends\Config\game.cfg',
    'Garena\Games\32787\Game\Config\game.cfg'
)

foreach ($d in $drives) {
    foreach ($rel in $cfgRelativePaths) {
        $cfgPath = Join-Path $d $rel
        if (Test-Path $cfgPath) {
            try {
                $lines = Get-Content -Path $cfgPath -ErrorAction Stop
                $newLines = @()
                $inGeneral = $false
                $generalFound = $false
                $windowModeSet = $false
                $borderlessSet = $false

                foreach ($line in $lines) {
                    $trimmed = $line.Trim()
                    if ($trimmed -match '^\[.*\]$') {
                        if ($inGeneral) {
                            if (-not $windowModeSet) { $newLines += "WindowMode=2"; $windowModeSet = $true }
                            if (-not $borderlessSet) { $newLines += "BorderlessWindow=1"; $borderlessSet = $true }
                            $inGeneral = $false
                        }
                        if ($trimmed -ieq '[General]') {
                            $inGeneral = $true
                            $generalFound = $true
                        }
                        $newLines += $line
                        continue
                    }

                    if ($inGeneral) {
                        if ($trimmed -match '^WindowMode\s*=') {
                            $newLines += "WindowMode=2"
                            $windowModeSet = $true
                            continue
                        }
                        if ($trimmed -match '^BorderlessWindow\s*=') {
                            $newLines += "BorderlessWindow=1"
                            $borderlessSet = $true
                            continue
                        }
                    }
                    $newLines += $line
                }

                if ($inGeneral) {
                    if (-not $windowModeSet) { $newLines += "WindowMode=2" }
                    if (-not $borderlessSet) { $newLines += "BorderlessWindow=1" }
                }

                if (-not $generalFound) {
                    $newLines += ""
                    $newLines += "[General]"
                    $newLines += "WindowMode=2"
                    $newLines += "BorderlessWindow=1"
                }

                Set-Content -Path $cfgPath -Value $newLines -Encoding UTF8 -Force
            } catch {
                # Bỏ qua nếu file bị lock hoặc không có quyền ghi
            }
        }
    }
}
