$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("aio-review-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tempRoot | Out-Null
try {
    $source = Get-Content -LiteralPath (Join-Path $root 'Setup_CMP30HX_WindowsAIO.bat') -Raw
    $start = [regex]::Match($source, '(?m)^:PCIeLinkRetrain\r?$').Index
    $end = [regex]::Match($source, '(?m)^:CleanupBYOVD\r?$').Index
    if ($start -le 0 -or $end -le $start) { throw 'Cannot locate batch routines' }
    $routines = $source.Substring($start, $end - $start).Replace('call :PnpSoftReset', 'call :FakeReset')
    $exe = Join-Path $tempRoot 'fake-installer.exe'
    Add-Type -TypeDefinition @'
using System;
using System.IO;
class FakeInstaller {
    static int Main() {
        var mode = Environment.GetEnvironmentVariable("FAKE_MODE");
        if (mode != "missing") {
            var path = Path.Combine(Environment.GetEnvironmentVariable("ProgramData"), "40HXUnlock", "gen2_status.txt");
            Directory.CreateDirectory(Path.GetDirectoryName(path));
            File.WriteAllText(path, mode == "unknown" ? "unknown" : "STATUS_CODE=GEN2_SUCCESS");
        }
        return mode == "error" ? 1 : 0;
    }
}
'@ -OutputAssembly $exe -OutputType ConsoleApplication
    $batch = Join-Path $tempRoot 'harness.bat'
    $head = @'
@echo off
setlocal
set "IS_MOCK=0"
set "IS_MOCK_FAIL=0"
set "IS_MOCK_WINRING0=0"
set "IS_MOCK_NOGPU=0"
set "IS_MOCK_MISSING_STATUS=0"
set "FINAL_INSTALLER=%FAKE_INSTALLER%"
call :PCIeLinkRetrain
if "%UNLOCK_OK%"=="1" exit /b 0
exit /b 1
'@
    [IO.File]::WriteAllText($batch, $head + "`r`n" + $routines + "`r`n:FakeReset`r`nexit /b 1`r`n", [Text.Encoding]::ASCII)
    $env:FAKE_INSTALLER = $exe
    $env:ProgramData = $tempRoot
    $status = Join-Path $tempRoot '40HXUnlock\gen2_status.txt'
    foreach ($case in @(
        @{ Mode='success'; Expected=0 },
        @{ Mode='missing'; Expected=1 },
        @{ Mode='unknown'; Expected=1 },
        @{ Mode='error'; Expected=1 }
    )) {
        New-Item -ItemType Directory -Force -Path (Split-Path $status) | Out-Null
        [IO.File]::WriteAllText($status, 'STATUS_CODE=GEN2_SUCCESS')
        $env:FAKE_MODE = $case.Mode
        & $env:ComSpec /d /c $batch | Out-Null
        if ($LASTEXITCODE -ne $case.Expected) { throw "$($case.Mode): expected $($case.Expected), got $LASTEXITCODE" }
        Write-Host "PASS $($case.Mode)"
    }
    $pnp = ($source -split "\r?\n" | Where-Object { $_ -like 'powershell *Get-PnpDevice*restart-device*' })
    if (@($pnp).Count -ne 1) { throw 'Cannot locate PnP command' }
    $pnp = $pnp.Replace('$d = @(', "function Get-PnpDevice { [pscustomobject]@{ HardwareID='PCI\VEN_10DE&DEV_2189'; InstanceId='fake'; Status='OK' } }; function pnputil.exe { `$global:LASTEXITCODE=0 }; `$d = @(")
    [IO.File]::WriteAllText($batch, "@echo off`r`n" + $pnp + "`r`nexit /b %ERRORLEVEL%`r`n", [Text.Encoding]::ASCII)
    & $env:ComSpec /d /c $batch
    if ($LASTEXITCODE -ne 0) { throw "PnP command failed: $LASTEXITCODE" }
    Write-Host 'PASS PnP command parsing'
    $taskBlock = [regex]::Match($source, '(?ms)^if "%TASK_OK%"=="0" \(\r?\n.*?^\)\r?$').Value
    if (-not $taskBlock) { throw 'Cannot locate task fallback' }
    Add-Type -TypeDefinition @'
using System;
class FakeSchtasks {
    static int Main(string[] args) {
        var mode = Environment.GetEnvironmentVariable("TASK_FAIL");
        foreach (var arg in args) {
            if (String.Equals(arg, mode, StringComparison.OrdinalIgnoreCase)) return 1;
        }
        return 0;
    }
}
'@ -OutputAssembly (Join-Path $tempRoot 'schtasks.exe') -OutputType ConsoleApplication
    $taskHarness = "@echo off`r`nset TASK_OK=0`r`nset FINAL_RUNNER=fake.bat`r`n" + $taskBlock + "`r`nif %TASK_OK%==1 (exit /b 0) else (exit /b 1)`r`n"
    [IO.File]::WriteAllText($batch, $taskHarness, [Text.Encoding]::ASCII)
    Push-Location $tempRoot
    try {
        foreach ($case in @(
            @{ Mode='none'; Expected=0 },
            @{ Mode='onstart'; Expected=1 },
            @{ Mode='onlogon'; Expected=1 }
        )) {
            $env:TASK_FAIL = $case.Mode
            & $env:ComSpec /d /c $batch
            if ($LASTEXITCODE -ne $case.Expected) { throw "task $($case.Mode): expected $($case.Expected), got $LASTEXITCODE" }
            Write-Host "PASS task $($case.Mode)"
        }
    } finally { Pop-Location }
} finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
