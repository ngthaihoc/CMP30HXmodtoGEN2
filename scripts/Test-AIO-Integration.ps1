$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
$tempRoot = Join-Path ([IO.Path]::GetTempPath()) ("aio integration " + [guid]::NewGuid())
$previous = @{}
foreach ($name in 'AIO_TEST_PROGRAMFILES', 'ProgramData', 'FAKE_MODE', 'TASK_MODE', 'PREFLIGHT_MODE', 'REMOVE_MODE') {
    $previous[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
New-Item -ItemType Directory -Path $tempRoot | Out-Null
try {
    $source = Get-Content -LiteralPath (Join-Path $root 'Setup_CMP30HX_WindowsAIO.bat') -Raw
    $source = $source.Replace('%ProgramFiles%', '%AIO_TEST_PROGRAMFILES%')
    foreach ($name in 'PreflightDiagnostic', 'ConfigurePersistence', 'RemovePersistence', 'CheckSharedDrivers') {
        $source = $source.Replace("call :$name", "call :Test$name")
    }
    $source += @'

:TestPreflightDiagnostic
if "%PREFLIGHT_MODE%"=="no-gpu" exit /b 1
exit /b 0
:TestConfigurePersistence
if not exist "%ProgramData%\40HXUnlock\count" exit /b 91
if "%TASK_MODE%"=="fail" exit /b 1
echo registered>"%ProgramData%\40HXUnlock\registered"
exit /b 0
:TestRemovePersistence
if "%REMOVE_MODE%"=="fail" exit /b 1
if exist "%ProgramData%\40HXUnlock\registered" del /q "%ProgramData%\40HXUnlock\registered"
exit /b 0
:TestCheckSharedDrivers
exit /b 0
'@
    $batch = Join-Path $tempRoot 'setup.bat'
    [IO.File]::WriteAllText($batch, $source, [Text.UTF8Encoding]::new($false))
    Copy-Item -LiteralPath (Join-Path $root 'windows-v3.0/release/RunUnlock.bat') -Destination $tempRoot
    Add-Type -TypeDefinition @'
using System;
using System.IO;
class FakeInstaller {
    static int Main(string[] args) {
        if (String.Join(" ", args) != "-gen2-30hx -silent -batch") return 91;
        string dir = Path.Combine(Environment.GetEnvironmentVariable("ProgramData"), "40HXUnlock");
        Directory.CreateDirectory(dir);
        string countPath = Path.Combine(dir, "count");
        int count = File.Exists(countPath) ? int.Parse(File.ReadAllText(countPath)) : 0;
        File.WriteAllText(countPath, (++count).ToString());
        string mode = Environment.GetEnvironmentVariable("FAKE_MODE");
        if (mode != "missing") {
            string code = mode == "nogpu" ? "NO_GPU" : mode == "gen1" ? "GEN1_STUCK" : mode == "badtoken" ? "GEN2_SUCCESS_EXTRA" : "GEN2_SUCCESS";
            File.WriteAllText(Path.Combine(dir, "cmp30hx_batch_status.txt"), "STATUS_CODE=" + code + "\r\n");
        }
        return mode == "error" ? 1 : 0;
    }
}
'@ -OutputAssembly (Join-Path $tempRoot 'UnlockAIO.exe') -OutputType ConsoleApplication
    foreach ($case in @(
        @{Mode='preflight'; Expected=0; Count=0; Flags='-preflight'},
        @{Mode='preflight_fail'; Expected=1; Count=0; Flags='-preflight'; Preflight='no-gpu'},
        @{Mode='wrong_hardware'; Expected=1; Count=0; Flags=''; Preflight='no-gpu'},
        @{Mode='unknown_flag'; Expected=2; Count=0; Flags='-typo'},
        @{Mode='success'; Expected=0; Count=1; Flags=''},
        @{Mode='missing'; Expected=1; Count=1; Flags=''},
        @{Mode='nogpu'; Expected=1; Count=1; Flags=''},
        @{Mode='gen1'; Expected=1; Count=1; Flags=''},
        @{Mode='badtoken'; Expected=1; Count=1; Flags=''},
        @{Mode='error'; Expected=1; Count=1; Flags=''},
        @{Mode='task_success'; Expected=0; Count=1; Flags=''; Task='success'},
        @{Mode='task_failure'; Expected=1; Count=1; Flags=''; Task='fail'},
        @{Mode='task_nogpu'; Expected=1; Count=1; Flags=''; Task='success'; InstallerMode='nogpu'},
        @{Mode='mock'; Expected=0; Count=0; Flags='-mock'},
        @{Mode='mockfail'; Expected=1; Count=0; Flags='-mock-fail'},
        @{Mode='mockdriver'; Expected=1; Count=0; Flags='-mock-winring0'},
        @{Mode='mocknogpu'; Expected=1; Count=0; Flags='-mock-nogpu'},
        @{Mode='mockmissing'; Expected=1; Count=0; Flags='-mock-missing-status'}
    )) {
        $env:AIO_TEST_PROGRAMFILES = Join-Path $tempRoot ("Program Files " + $case.Mode)
        $env:ProgramData = Join-Path $tempRoot ("Program Data " + $case.Mode)
        $env:FAKE_MODE = if ($case.InstallerMode) { $case.InstallerMode } else { $case.Mode }
        $env:TASK_MODE = $case.Task
        $env:PREFLIGHT_MODE = $case.Preflight
        $env:REMOVE_MODE = ''
        $stateDir = New-Item -ItemType Directory -Path (Join-Path $env:ProgramData '40HXUnlock') -Force
        $sharedStatus = Join-Path $stateDir.FullName 'gen2_status.txt'
        Set-Content -LiteralPath $sharedStatus -Value 'Keep shared status'
        if ($case.Mode -eq 'missing') {
            Set-Content -LiteralPath (Join-Path $stateDir.FullName 'cmp30hx_batch_status.txt') -Value 'STATUS_CODE=GEN2_SUCCESS'
        }
        $taskFlag = if ($case.Task) { '' } else { '-notask' }
        $output = & $env:ComSpec /d /c "call `"$batch`" -noadmin -nocheck $taskFlag -nowait $($case.Flags)" 2>&1
        if ($LASTEXITCODE -ne $case.Expected) { throw "$($case.Mode): exit $LASTEXITCODE instead of $($case.Expected)`n$($output -join "`n")" }
        $countPath = Join-Path $env:ProgramData '40HXUnlock\count'
        $count = if (Test-Path $countPath) { [int](Get-Content $countPath -Raw) } else { 0 }
        if ($count -ne $case.Count) { throw "$($case.Mode): expected $($case.Count) invocations, got $count" }
        $installDir = Join-Path $env:AIO_TEST_PROGRAMFILES '40HXUnlock'
        $runnerExists = Test-Path (Join-Path $installDir 'RunUnlock.bat')
        if ($runnerExists -ne ($case.Count -gt 0)) { throw "$($case.Mode): unexpected deployment result" }
        if ((Get-Content -LiteralPath $sharedStatus -Raw).Trim() -ne 'Keep shared status') { throw "$($case.Mode): shared status was overwritten" }
        $registered = Test-Path (Join-Path $env:ProgramData '40HXUnlock\registered')
        if ($registered -ne ($case.Mode -eq 'task_success')) { throw "$($case.Mode): unexpected persistence result" }
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

    $env:AIO_TEST_PROGRAMFILES = Join-Path $tempRoot 'Program Files task_success'
    $env:ProgramData = Join-Path $tempRoot 'Program Data task_success'
    $installDir = Join-Path $env:AIO_TEST_PROGRAMFILES '40HXUnlock'
    $stateDir = Join-Path $env:ProgramData '40HXUnlock'
    $registration = Join-Path $stateDir 'registered'
    $batchStatus = Join-Path $stateDir 'cmp30hx_batch_status.txt'
    $sharedStatus = Join-Path $stateDir 'gen2_status.txt'
    $keepFile = Join-Path $installDir 'user-note.txt'
    Set-Content -LiteralPath $keepFile -Value 'Keep'
    $env:REMOVE_MODE = 'fail'
    $output = & $env:ComSpec /d /c "`"$batch`" -uninstall -noadmin -nowait" 2>&1
    if ($LASTEXITCODE -ne 1 -or -not (Test-Path $registration) -or -not (Test-Path (Join-Path $installDir 'UnlockAIO.exe'))) {
        throw "Uninstall proceeded after task removal failed`n$($output -join "`n")"
    }
    Write-Host 'PASS uninstall task removal failure'
    $env:REMOVE_MODE = ''
    $output = & $env:ComSpec /d /c "`"$batch`" -cleantasks -noadmin -nowait" 2>&1
    if ($LASTEXITCODE -ne 0 -or (Test-Path $registration) -or -not (Test-Path $batchStatus) -or -not (Test-Path (Join-Path $installDir 'UnlockAIO.exe'))) {
        throw "Unexpected task-only cleanup result`n$($output -join "`n")"
    }
    Write-Host 'PASS task-only cleanup'
    foreach ($pass in 1, 2) {
        $output = & $env:ComSpec /d /c "`"$batch`" -uninstall -noadmin -nowait" 2>&1
        if ($LASTEXITCODE -ne 0 -or (Test-Path $batchStatus) -or -not (Test-Path $sharedStatus)) { throw "Unexpected uninstall result on pass $pass" }
        if ((Test-Path (Join-Path $installDir 'UnlockAIO.exe')) -or (Test-Path (Join-Path $installDir 'RunUnlock.bat')) -or -not (Test-Path $keepFile)) {
            throw "Uninstall did not remove only the deployed files"
        }
        Write-Host "PASS uninstall pass $pass"
    }

    $launcher = Join-Path $tempRoot 'Launch_WebUI.bat'
    Copy-Item -LiteralPath (Join-Path $root 'Launch_WebUI.bat') -Destination $launcher
    $launcherArgs = '"' + $launcher + '" -noadmin'
    $output = & $env:ComSpec /d /c $launcherArgs 2>&1
    if ($LASTEXITCODE -ne 1) { throw "Launcher accepted missing release directory: $output" }
    Write-Host 'PASS launcher rejects missing release directory'

    $releaseDir = Join-Path $tempRoot 'windows-v3.0/release'
    New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null
    Copy-Item -LiteralPath (Join-Path $tempRoot 'UnlockAIO.exe') -Destination $releaseDir
    $output = & $env:ComSpec /d /c $launcherArgs 2>&1
    if ($LASTEXITCODE -ne 0) { throw "Launcher failed from a path with spaces: $output" }
    Write-Host 'PASS launcher starts the packaged executable from a path with spaces'
} finally {
    foreach ($name in $previous.Keys) { [Environment]::SetEnvironmentVariable($name, $previous[$name], 'Process') }
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
