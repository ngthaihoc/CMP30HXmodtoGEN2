$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("aio integration " + [guid]::NewGuid())
$previous = @{}
foreach ($name in 'AIO_TEST_PROGRAMFILES', 'ProgramData', 'FAKE_MODE') {
    $previous[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
New-Item -ItemType Directory -Path $tempRoot | Out-Null
try {
    $source = Get-Content -LiteralPath (Join-Path $root 'Setup_CMP30HX_WindowsAIO.bat') -Raw
    $source = $source.Replace('%ProgramFiles%', '%AIO_TEST_PROGRAMFILES%')
    foreach ($name in 'PreflightDiagnostic', 'ConfigurePersistence', 'CleanupBYOVD', 'CheckSharedDrivers', 'PnpSoftReset') {
        $source = $source.Replace("call :$name", "call :Test$name")
    }
    $source += @'

:TestPreflightDiagnostic
set "NEED_REBOOT=0"
exit /b 0
:TestConfigurePersistence
set "TASK_OK=1"
exit /b 0
:TestCleanupBYOVD
exit /b 0
:TestCheckSharedDrivers
exit /b 0
:TestPnpSoftReset
exit /b 0
'@
    $batch = Join-Path $tempRoot 'setup.bat'
    [IO.File]::WriteAllText($batch, $source, [Text.Encoding]::UTF8)
    Add-Type -TypeDefinition @'
using System;
using System.IO;
class FakeInstaller {
    static int Main() {
        string dir = Path.Combine(Environment.GetEnvironmentVariable("ProgramData"), "40HXUnlock");
        Directory.CreateDirectory(dir);
        string countPath = Path.Combine(dir, "count");
        int count = File.Exists(countPath) ? int.Parse(File.ReadAllText(countPath)) : 0;
        File.WriteAllText(countPath, (++count).ToString());
        string mode = Environment.GetEnvironmentVariable("FAKE_MODE");
        if (mode != "missing") {
            string code = mode == "nogpu" ? "NO_GPU" : mode == "retry" && count == 1 ? "GEN1_STUCK" : "GEN2_SUCCESS";
            File.WriteAllText(Path.Combine(dir, "gen2_status.txt"), "STATUS_CODE=" + code);
        }
        return 0;
    }
}
'@ -OutputAssembly (Join-Path $tempRoot 'UnlockAIO.exe') -OutputType ConsoleApplication
    foreach ($case in @(
        @{Mode='success'; Expected=0; Count=1; Flags=''},
        @{Mode='missing'; Expected=1; Count=1; Flags=''},
        @{Mode='nogpu'; Expected=1; Count=1; Flags=''},
        @{Mode='retry'; Expected=0; Count=2; Flags=''},
        @{Mode='mock'; Expected=0; Count=0; Flags='-mock'},
        @{Mode='mockfail'; Expected=1; Count=0; Flags='-mock-fail'},
        @{Mode='mockdriver'; Expected=1; Count=0; Flags='-mock-winring0'},
        @{Mode='mocknogpu'; Expected=1; Count=0; Flags='-mock-nogpu'},
        @{Mode='mockmissing'; Expected=1; Count=0; Flags='-mock-missing-status'}
    )) {
        $env:AIO_TEST_PROGRAMFILES = Join-Path $tempRoot ("Program Files " + $case.Mode)
        $env:ProgramData = Join-Path $tempRoot ("Program Data " + $case.Mode)
        $env:FAKE_MODE = $case.Mode
        $output = & $env:ComSpec /d /c "`"$batch`" -noadmin -nocheck -notask -nowait $($case.Flags)" 2>&1
        if ($LASTEXITCODE -ne $case.Expected) { throw "$($case.Mode): exit $LASTEXITCODE instead of $($case.Expected)`n$($output -join "`n")" }
        $countPath = Join-Path $env:ProgramData '40HXUnlock\count'
        $count = if (Test-Path $countPath) { [int](Get-Content $countPath -Raw) } else { 0 }
        if ($count -ne $case.Count) { throw "$($case.Mode): expected $($case.Count) invocations, got $count" }
        if (-not (Test-Path (Join-Path $env:AIO_TEST_PROGRAMFILES '40HXUnlock\RunUnlock.bat'))) { throw "$($case.Mode): runner missing`n$($output -join "`n")" }
        Write-Host "PASS integration $($case.Mode)"
    }
    $env:AIO_TEST_PROGRAMFILES = Join-Path $tempRoot 'Program Files copy failure'
    $env:ProgramData = Join-Path $tempRoot 'Program Data copy failure'
    New-Item -ItemType Directory -Path $env:AIO_TEST_PROGRAMFILES | Out-Null
    [IO.File]::WriteAllText((Join-Path $env:AIO_TEST_PROGRAMFILES '40HXUnlock'), 'block directory creation')
    $output = & $env:ComSpec /d /c "`"$batch`" -noadmin -nocheck -notask -nowait" 2>&1
    if ($LASTEXITCODE -ne 1) { throw "copy failure: expected exit 1, got $LASTEXITCODE" }
    if (Test-Path (Join-Path $env:ProgramData '40HXUnlock\count')) { throw 'Installer ran after copy failure' }
    Write-Host 'PASS integration copy failure'
} finally {
    foreach ($name in $previous.Keys) { [Environment]::SetEnvironmentVariable($name, $previous[$name], 'Process') }
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
