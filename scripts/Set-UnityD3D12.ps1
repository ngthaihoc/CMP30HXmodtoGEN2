# Script hỗ trợ cấu hình Direct3D 12 (-force-d3d12) cho các Game chạy Unity Engine trên CMP 40HX / 30HX
param(
    [string]$GamePath = "",
    [switch]$ScanDesktop = $true
)

$wsh = New-Object -ComObject WScript.Shell
$regGpuPref = 'HKCU:\Software\Microsoft\DirectX\UserGpuPreferences'
if (-not (Test-Path $regGpuPref)) {
    New-Item -Path $regGpuPref -Force | Out-Null
}

function Configure-UnityExe {
    param([string]$ExePath)

    if (-not (Test-Path $ExePath)) { return }
    $dir = Split-Path -Parent $ExePath
    $exeName = [System.IO.Path]::GetFileNameWithoutExtension($ExePath)
    $dataDir = Join-Path $dir "${exeName}_Data"
    $unityCrash = Join-Path $dir "UnityCrashHandler64.exe"
    $unityCrash32 = Join-Path $dir "UnityCrashHandler32.exe"

    $isUnity = (Test-Path $dataDir) -or (Test-Path $unityCrash) -or (Test-Path $unityCrash32)
    if ($isUnity) {
        Write-Host "  [+] Phát hiện Game Unity: $ExePath" -ForegroundColor Green
        # Gán High Performance GPU
        Set-ItemProperty -Path $regGpuPref -Name $ExePath -Value 'GpuPreference=2;' -ErrorAction SilentlyContinue
    }
}

function Add-D3D12ToShortcut {
    param([string]$LnkPath)

    try {
        $shortcut = $wsh.CreateShortcut($LnkPath)
        $target = $shortcut.TargetPath
        if ($target -and (Test-Path $target) -and ($target.EndsWith(".exe", [System.StringComparison]::OrdinalIgnoreCase))) {
            $dir = Split-Path -Parent $target
            $exeName = [System.IO.Path]::GetFileNameWithoutExtension($target)
            $dataDir = Join-Path $dir "${exeName}_Data"
            $unityCrash = Join-Path $dir "UnityCrashHandler64.exe"

            if ((Test-Path $dataDir) -or (Test-Path $unityCrash)) {
                Write-Host "  [*] Game Unity tìm thấy qua Shortcut: $LnkPath" -ForegroundColor Cyan
                Write-Host "      Mục tiêu: $target"

                # Gán cờ -force-d3d12 vào đối số nếu chưa có
                $currentArgs = $shortcut.Arguments
                if ($currentArgs -notmatch '-force-d3d12') {
                    $shortcut.Arguments = ("$currentArgs -force-d3d12").Trim()
                    $shortcut.Save()
                    Write-Host "      [V] Đã bổ sung tham số -force-d3d12 vào Shortcut thành công!" -ForegroundColor Green
                } else {
                    Write-Host "      [V] Shortcut đã có sẵn tham số -force-d3d12." -ForegroundColor Yellow
                }

                Configure-UnityExe -ExePath $target
            }
        }
    } catch {
        # Bỏ qua nếu shortcut không đọc được
    }
}

# 1. Nếu người dùng chỉ định đường dẫn cụ thể
if ($GamePath) {
    if ($GamePath.EndsWith(".lnk", [System.StringComparison]::OrdinalIgnoreCase)) {
        Add-D3D12ToShortcut -LnkPath $GamePath
    } elseif ($GamePath.EndsWith(".exe", [System.StringComparison]::OrdinalIgnoreCase)) {
        Configure-UnityExe -ExePath $GamePath
    } elseif (Test-Path $GamePath) {
        # Nếu truyền vào thư mục, quét các file exe và lnk
        Get-ChildItem -Path $GamePath -Filter "*.exe" -Recurse -Depth 2 -ErrorAction SilentlyContinue | ForEach-Object {
            Configure-UnityExe -ExePath $_.FullName
        }
    }
}

# 2. Tự động quét Shortcut trên Desktop người dùng và Public Desktop
if ($ScanDesktop) {
    $desktopDirs = @(
        [Environment]::GetFolderPath('Desktop'),
        [Environment]::GetFolderPath('CommonDesktopDirectory')
    )

    foreach ($d in $desktopDirs) {
        if ($d -and (Test-Path $d)) {
            Get-ChildItem -Path $d -Filter "*.lnk" -ErrorAction SilentlyContinue | ForEach-Object {
                Add-D3D12ToShortcut -LnkPath $_.FullName
            }
        }
    }
}

Write-Host "[V] Hoàn tất quét và cấu hình DirectX 12 cho Game Unity." -ForegroundColor Green
