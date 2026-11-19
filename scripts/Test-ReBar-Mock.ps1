$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$sourceScript = Join-Path $root 'Setup_ReBar_CMP30HX.bat'
$sandbox = Join-Path $env:TEMP ('CMP30HX_ReBar_Mock_' + [guid]::NewGuid().ToString('N'))
$dataDir = Join-Path $sandbox 'ProgramData'
$filesDir = Join-Path $sandbox 'ProgramFiles'
$toolsDir = Join-Path $sandbox 'windows-v3.0\tools\rebar30hx'
$scriptsDir = Join-Path $sandbox 'scripts'
$shimDir = Join-Path $sandbox 'shims'
$mutationLog = Join-Path $sandbox 'blocked-mutations.txt'
New-Item -ItemType Directory -Path $dataDir, $filesDir, $toolsDir, $scriptsDir, $shimDir -Force | Out-Null

try {
    $scriptText = [IO.File]::ReadAllText($sourceScript)
    $scriptText = $scriptText.Replace('%ProgramData%', $dataDir).Replace('%ProgramFiles%', $filesDir)
    $isolatedScript = Join-Path $sandbox 'Setup_ReBar_CMP30HX.bat'
    [IO.File]::WriteAllText($isolatedScript, $scriptText, [Text.Encoding]::UTF8)

    foreach ($name in @('NvStrapsReBar.exe', 'nvidiaProfileInspector.exe', 'Enable_ReBAR_Turing.nip')) {
        [IO.File]::WriteAllBytes((Join-Path $toolsDir $name), [byte[]]@())
    }
    Set-Content -LiteralPath (Join-Path $scriptsDir 'Get-SysInfo.ps1') -Value @(
        'Write-Output "SYS_MANU=MockVendor"'
        'Write-Output "SYS_MODEL=MockDesktop"'
        'Write-Output "MB_MANU=MockVendor"'
        'Write-Output "MB_PROD=MockBoard"'
        'Write-Output "MB_BIOS=MockBIOS"'
        'Write-Output "IS_LAPTOP=0"'
        'Write-Output "SYSINFO_OK=1"'
        'Write-Output "FIRMWARE_TYPE=UEFI"'
    )
    foreach ($name in @('reg.cmd', 'schtasks.cmd')) {
        Set-Content -LiteralPath (Join-Path $shimDir $name) -Value @(
            '@echo off'
            '>>"%REBAR_TEST_MUTATIONS%" echo blocked %~n0 %*'
            'exit /b 1'
        )
    }

    function Invoke-IsolatedMock([string]$arguments) {
        $start = New-Object System.Diagnostics.ProcessStartInfo
        $start.FileName = $env:ComSpec
        $start.Arguments = '/d /c call "' + $isolatedScript + '" ' + $arguments
        $start.WorkingDirectory = $sandbox
        $start.UseShellExecute = $false
        $start.RedirectStandardOutput = $true
        $start.RedirectStandardError = $true
        $start.EnvironmentVariables['PATH'] = $shimDir + ';' + $env:PATH
        $start.EnvironmentVariables['REBAR_TEST_MUTATIONS'] = $mutationLog
        $process = [System.Diagnostics.Process]::Start($start)
        $output = $process.StandardOutput.ReadToEnd()
        $errorOutput = $process.StandardError.ReadToEnd()
        if (-not $process.WaitForExit(30000)) {
            $process.Kill()
            throw "Mock timed out: $arguments"
        }
        if ($process.ExitCode -ne 0) {
            throw "Mock failed ($($process.ExitCode)): $arguments`n$output`n$errorOutput"
        }
        return $output
    }

    $output = Invoke-IsolatedMock '-mock -nowait -noadmin'
    if ($output -notmatch 'Mock chi mo phong; khong thay doi he thong va khong can reboot') {
        throw 'Mock install did not report that no reboot or system change is needed.'
    }
    if (Test-Path -LiteralPath (Join-Path $dataDir '40HXUnlock\rebar_status.txt')) {
        throw 'Mock install wrote a status file.'
    }
    if (Test-Path -LiteralPath (Join-Path $filesDir '40HXUnlock\rebar')) {
        throw 'Mock install copied a persistent driver profile.'
    }
    if (Test-Path -LiteralPath $mutationLog) {
        throw 'Mock install attempted a task or Registry operation.'
    }

    $sentinelDir = Join-Path $dataDir '40HXUnlock'
    New-Item -ItemType Directory -Path $sentinelDir -Force | Out-Null
    $sentinel = Join-Path $sentinelDir 'rebar_status.txt'
    Set-Content -LiteralPath $sentinel -Value 'must remain'
    $null = Invoke-IsolatedMock '-uninstall -mock -nowait -noadmin'
    if ((Get-Content -LiteralPath $sentinel -Raw).Trim() -ne 'must remain') {
        throw 'Mock uninstall changed an existing status file.'
    }
    if (Test-Path -LiteralPath $mutationLog) {
        throw 'Mock uninstall attempted a task or Registry operation.'
    }
    Write-Output 'ReBAR isolated mock: PASS'
} finally {
    $resolvedSandbox = [IO.Path]::GetFullPath($sandbox)
    $resolvedTemp = [IO.Path]::GetFullPath($env:TEMP).TrimEnd('\') + '\'
    if ($resolvedSandbox.StartsWith($resolvedTemp, [StringComparison]::OrdinalIgnoreCase) -and
        [IO.Path]::GetFileName($resolvedSandbox).StartsWith('CMP30HX_ReBar_Mock_', [StringComparison]::Ordinal)) {
        Remove-Item -LiteralPath $resolvedSandbox -Recurse -Force
    }
}
