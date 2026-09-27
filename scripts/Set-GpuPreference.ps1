$reg = 'HKCU:\Software\Microsoft\DirectX\UserGpuPreferences'
if (-not (Test-Path $reg)) {
    New-Item -Path $reg -Force | Out-Null
}

$found = 0
$drives = (Get-PSDrive -PSProvider FileSystem).Root
$targets = @(
    'Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe',
    'Riot Games\League of Legends\Game\League of Legends.exe'
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
