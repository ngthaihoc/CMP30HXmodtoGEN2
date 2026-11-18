$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("aio-review-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tempRoot | Out-Null
try {
    $source = Get-Content -LiteralPath (Join-Path $root 'Setup_CMP30HX_WindowsAIO.bat') -Raw
    $start = [regex]::Match($source, '(?m)^:PCIeLinkRetrain\r?$').Index
    $end = [regex]::Match($source, '(?m)^:CheckSharedDrivers\r?$').Index
    if ($start -le 0 -or $end -le $start) { throw 'Cannot locate batch routines' }
    $routines = $source.Substring($start, $end - $start)
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
            string status = mode == "unknown" ? "unknown" : mode == "malformed" ? "STATUS_CODE=GEN2_SUCCESS_EXTRA" : "STATUS_CODE=GEN2_SUCCESS";
            File.WriteAllText(path, status + "\r\n");
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
set "AIO_STATUS=%ProgramData%\40HXUnlock\gen2_status.txt"
call :PCIeLinkRetrain
if "%UNLOCK_OK%"=="1" exit /b 0
exit /b 1
'@
    [IO.File]::WriteAllText($batch, $head + "`r`n" + $routines, [Text.Encoding]::ASCII)
    $env:FAKE_INSTALLER = $exe
    $env:ProgramData = $tempRoot
    $status = Join-Path $tempRoot '40HXUnlock\gen2_status.txt'
    foreach ($case in @(
        @{ Mode='success'; Expected=0 },
        @{ Mode='missing'; Expected=1 },
        @{ Mode='unknown'; Expected=1 },
        @{ Mode='malformed'; Expected=1 },
        @{ Mode='error'; Expected=1 }
    )) {
        New-Item -ItemType Directory -Force -Path (Split-Path $status) | Out-Null
        [IO.File]::WriteAllText($status, 'STATUS_CODE=GEN2_SUCCESS')
        $env:FAKE_MODE = $case.Mode
        & $env:ComSpec /d /c $batch | Out-Null
        if ($LASTEXITCODE -ne $case.Expected) { throw "$($case.Mode): expected $($case.Expected), got $LASTEXITCODE" }
        Write-Host "PASS $($case.Mode)"
    }
    Remove-Item -LiteralPath $status
    New-Item -ItemType Directory -Path $status | Out-Null
    $sentinel = Join-Path $status 'sentinel.txt'
    Set-Content -LiteralPath $sentinel -Value 'Keep'
    $env:FAKE_MODE = 'success'
    & $env:ComSpec /d /c $batch | Out-Null
    if ($LASTEXITCODE -ne 1 -or -not (Test-Path $sentinel)) { throw 'Accepted a blocked status path or removed its contents' }
    Write-Host 'PASS blocked status path'
} finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
