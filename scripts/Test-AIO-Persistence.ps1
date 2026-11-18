param (
    [int]$TaskWaitTimeoutSeconds = 30
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
$tempRoot = Join-Path 'D:\Temp' ("aio-persistence-" + [guid]::NewGuid())
$null = New-Item -ItemType Directory -Path $tempRoot -Force
$runnerPath = Join-Path $tempRoot 'Program Files with spaces\RunUnlock.bat'
$installDirectory = Split-Path $runnerPath -Parent
$statePath = Join-Path $tempRoot 'state.json'
$logPath = Join-Path $tempRoot 'calls.log'
$childPath = Join-Path $tempRoot 'scheduler-child.ps1'

function Assert-Condition([bool]$Condition, [string]$Message) {
    if (-not $Condition) {
        throw $Message
    }
}

function Invoke-SchedulerCase([string]$Case, [switch]$Remove, [string]$WorkingDirectory = $installDirectory) {
    $env:PERSISTENCE_CASE = $Case
    $env:PERSISTENCE_RUNNER = $runnerPath
    $env:PERSISTENCE_WORKDIR = $WorkingDirectory
    $env:PERSISTENCE_STATE = $statePath
    $env:PERSISTENCE_LOG = $logPath
    $env:PERSISTENCE_TARGET = Join-Path $repoRoot 'scripts\Register-UnlockTask.ps1'
    $env:PERSISTENCE_TIMEOUT = "$TaskWaitTimeoutSeconds"
    $arguments = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $childPath)
    if ($Remove) {
        $arguments += '-Remove'
    }
    & powershell.exe @arguments 2>&1 | Out-Null
    return $LASTEXITCODE
}

try {
    $null = New-Item -ItemType Directory -Path $installDirectory -Force
    Copy-Item -LiteralPath (Join-Path $repoRoot 'windows-v3.0\release\RunUnlock.bat') -Destination $runnerPath

    Add-Type -TypeDefinition @'
using System;
using System.IO;
public static class FakeUnlockAIO {
    public static int Main() {
        string dir = Path.Combine(Environment.GetEnvironmentVariable("ProgramData"), "40HXUnlock");
        Directory.CreateDirectory(dir);
        string status = Environment.GetEnvironmentVariable("AIO_FAKE_STATUS");
        if (status != "missing") {
            File.WriteAllText(Path.Combine(dir, "cmp30hx_batch_status.txt"), (status ?? "STATUS_CODE=GEN2_SUCCESS") + Environment.NewLine);
        }
        string code = Environment.GetEnvironmentVariable("AIO_FAKE_EXIT") ?? "0";
        return Int32.Parse(code);
    }
}
'@ -OutputAssembly (Join-Path $installDirectory 'UnlockAIO.exe') -OutputType ConsoleApplication

    $env:ProgramData = Join-Path $tempRoot 'Program Data'
    $statusDirectory = Join-Path $env:ProgramData '40HXUnlock'
    $null = New-Item -ItemType Directory -Path $statusDirectory -Force

    foreach ($case in @(
        @{ Name = 'success'; Status = 'STATUS_CODE=GEN2_SUCCESS'; Exit = '0'; Expected = 0 },
        @{ Name = 'nonzero installer'; Status = 'STATUS_CODE=GEN2_SUCCESS'; Exit = '7'; Expected = 7 },
        @{ Name = 'missing status'; Status = 'missing'; Exit = '0'; Expected = 1 },
        @{ Name = 'non-success status'; Status = 'STATUS_CODE=GEN1_STUCK'; Exit = '0'; Expected = 1 },
        @{ Name = 'stale status cleared'; Status = 'STATUS_CODE=GEN2_SUCCESS'; Exit = '0'; Expected = 0 }
    )) {
        $statusFile = Join-Path $statusDirectory 'cmp30hx_batch_status.txt'
        [IO.File]::WriteAllText($statusFile, 'STATUS_CODE=GEN2_SUCCESS')
        $env:AIO_FAKE_STATUS = $case.Status
        $env:AIO_FAKE_EXIT = $case.Exit
        & $env:ComSpec /d /c ('"' + $runnerPath + '"')
        Assert-Condition ($LASTEXITCODE -eq $case.Expected) "Runner $($case.Name): expected $($case.Expected), got $LASTEXITCODE"
        if ($case.Status -eq 'missing') {
            Assert-Condition (-not (Test-Path -LiteralPath $statusFile)) 'Runner accepted a stale status file'
        }
        Write-Host "PASS runner $($case.Name)"
    }

    $statusFile = Join-Path $statusDirectory 'cmp30hx_batch_status.txt'
    Remove-Item -LiteralPath $statusFile -Force -ErrorAction SilentlyContinue
    $null = New-Item -ItemType Directory -Path $statusFile -Force
    $statusSentinel = Join-Path $statusFile 'preserve.txt'
    [IO.File]::WriteAllText($statusSentinel, 'preserve')
    & $env:ComSpec /d /c ('"' + $runnerPath + '"')
    Assert-Condition ($LASTEXITCODE -ne 0) 'Runner accepted a directory at the status file path'
    Assert-Condition (Test-Path -LiteralPath $statusSentinel) 'Runner deleted contents of a status path directory'
    Write-Host 'PASS runner preserves status path directory'
    Remove-Item -LiteralPath $statusFile -Recurse -Force

    $child = @'
$ErrorActionPreference = 'Stop'
$runner = $env:PERSISTENCE_RUNNER
$target = $env:PERSISTENCE_TARGET
$log = $env:PERSISTENCE_LOG
$statePath = $env:PERSISTENCE_STATE
$caseName = $env:PERSISTENCE_CASE
$taskName = 'CMP30HX_Batch_Gen2_Unlock'
$otherTaskName = 'CMP30HX_Gen2_Unlock'
$installedExe = Join-Path (Split-Path -Parent $runner) 'UnlockAIO.exe'
$global:PERSISTENCE_TASKS = @()
$global:PERSISTENCE_LIST_COUNT = 0

function Write-Call([string]$value) {
    Add-Content -LiteralPath $log -Value $value
}

function Get-ScheduledTask {
    param($TaskPath, $TaskName, $ErrorAction)
    Write-Call 'list'
    if ($caseName -eq 'scheduler unavailable') {
        throw 'Mock scheduler unavailable'
    }
    $global:PERSISTENCE_LIST_COUNT++
    if ($caseName -eq 'running task completes' -and $global:PERSISTENCE_LIST_COUNT -gt 1) {
        foreach ($task in $global:PERSISTENCE_TASKS) { $task.State = 'Ready' }
    }
    return @($global:PERSISTENCE_TASKS)
}

function Disable-ScheduledTask {
    param($InputObject, $ErrorAction)
    Write-Call ("disable:" + $InputObject.TaskName)
}

function Stop-ScheduledTask {
    param($InputObject, $ErrorAction)
    Write-Call ("stop:" + $InputObject.TaskName)
}

function Unregister-ScheduledTask {
    param($InputObject, [switch]$Confirm, $ErrorAction)
    Write-Call ("unregister:" + $InputObject.TaskName)
    if ($caseName -eq 'unregister fails') {
        throw 'Mock unregister failure'
    }
    $global:PERSISTENCE_TASKS = @($global:PERSISTENCE_TASKS | Where-Object TaskName -NE $InputObject.TaskName)
}

function Test-Path {
    param($LiteralPath, $PathType)
    if ($LiteralPath -like 'Registry::*') {
        Write-Call 'registry-cleanup:'
        return $caseName -in 'remove unrelated Go task', 'remove legacy installed AIO' -and
            $LiteralPath -eq 'Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Run'
    }
    Microsoft.PowerShell.Management\Test-Path -LiteralPath $LiteralPath -PathType $PathType
}

function Get-Item {
    param($LiteralPath)
    if ($LiteralPath -like 'Registry::*') {
        $valuePath = if ($caseName -eq 'remove legacy installed AIO') { $installedExe } else { 'D:\Other\UnlockAIO.exe' }
        $item = [pscustomobject]@{ ValuePath = $valuePath }
        Add-Member -InputObject $item -MemberType ScriptMethod -Name GetValue -Value {
            param($name, $default, $options)
            if ($name -ne '40HXGen2') { return $null }
            return '"' + $this.ValuePath + '" -gen2 -silent'
        }
        return $item
    }
    Microsoft.PowerShell.Management\Get-Item -LiteralPath $LiteralPath
}

function Remove-ItemProperty {
    param($LiteralPath, $Name, $ErrorAction)
    Write-Call ("registry-remove:" + $Name)
}

function Get-ChildItem {
    param($LiteralPath, $ErrorAction)
    if ($LiteralPath -eq 'Registry::HKEY_USERS') { return @() }
    Microsoft.PowerShell.Management\Get-ChildItem -LiteralPath $LiteralPath -ErrorAction $ErrorAction
}

if ($caseName -in 'remove unrelated Go task', 'remove legacy installed AIO', 'register collision') {
    $name = if ($caseName -eq 'register collision') { $taskName } else { $otherTaskName }
    $execute = if ($caseName -eq 'remove legacy installed AIO') { $installedExe } else { 'D:\Other\UnlockAIO.exe' }
    $global:PERSISTENCE_TASKS += [pscustomobject]@{
        TaskName = $name
        State = 'Ready'
        Actions = @([pscustomobject]@{ Execute = $execute; Arguments = '-gen2 -silent' })
    }
} elseif ($caseName -eq 'remove task mentions runner') {
    $global:PERSISTENCE_TASKS += [pscustomobject]@{
        TaskName = $taskName
        State = 'Ready'
        Actions = @([pscustomobject]@{ Execute = 'cmd.exe'; Arguments = ('/d /c echo "' + $runner + '"') })
    }
} elseif ($caseName -in 'remove matching', 'unregister fails', 'running task completes') {
    $state = if ($caseName -eq 'running task completes') { 'Running' } else { 'Ready' }
    $global:PERSISTENCE_TASKS += [pscustomobject]@{
        TaskName = $taskName
        State = $state
        Actions = @([pscustomobject]@{ Execute = 'cmd.exe'; Arguments = ('/d /c ""' + $runner + '""') })
    }
}

if ($caseName -eq 'register') {
    function New-ScheduledTaskAction { param($Execute, $Argument, $WorkingDirectory); [pscustomobject]@{ Execute=$Execute; Arguments=$Argument; WorkingDirectory=$WorkingDirectory } }
    function New-ScheduledTaskTrigger { param([switch]$AtStartup, [switch]$AtLogOn); [pscustomobject]@{ Type=if($AtStartup){'Startup'}else{'Logon'}; Delay=$null } }
    function New-ScheduledTaskSettingsSet { param($MultipleInstances, [switch]$AllowStartIfOnBatteries, [switch]$DontStopIfGoingOnBatteries, [switch]$StartWhenAvailable, $RestartCount, $RestartInterval, $ExecutionTimeLimit); [pscustomobject]@{ MultipleInstances=$MultipleInstances } }
    function New-ScheduledTaskPrincipal { param($UserId, $LogonType, $RunLevel); [pscustomobject]@{ UserId=$UserId } }
    function Register-ScheduledTask { param($TaskName, $Action, $Trigger, $Settings, $Principal, [switch]$Force, $ErrorAction); Write-Call ("register:" + $TaskName); ConvertTo-Json -InputObject @{TaskName=$TaskName; Action=$Action; Trigger=$Trigger; Settings=$Settings; Principal=$Principal} -Depth 5 | Set-Content -LiteralPath $statePath }
}

try {
    if ($args -contains '-Remove') {
        & $target -ExePath $runner -Remove -TaskWaitTimeoutSeconds ([int]$env:PERSISTENCE_TIMEOUT)
    } else {
        & $target -ExePath $runner -WorkingDirectory $env:PERSISTENCE_WORKDIR
    }
} catch {
    Write-Call ("error:" + $_.Exception.Message)
    exit 1
}
'@
    [IO.File]::WriteAllText($childPath, $child, [Text.Encoding]::UTF8)

    Remove-Item -LiteralPath $logPath, $statePath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'register'
    Assert-Condition ($code -eq 0) "Mock register failed with exit code $code"
    $registered = Get-Content -LiteralPath $statePath -Raw | ConvertFrom-Json
    Assert-Condition ($registered.TaskName -eq 'CMP30HX_Batch_Gen2_Unlock') 'Registration used the wrong task name'
    Assert-Condition ($registered.Action.Execute -eq $env:ComSpec) 'Registration did not invoke cmd.exe'
    Assert-Condition ($registered.Action.Arguments -eq ('/d /c ""' + $runnerPath + '""')) 'Runner path arguments were not quoted correctly'
    Assert-Condition ($registered.Settings.MultipleInstances -eq 'IgnoreNew') 'Task runs were not serialized'
    Assert-Condition (@($registered.Trigger).Count -eq 2) 'Expected startup and logon triggers'
    Write-Host 'PASS scheduler registration'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'register' -WorkingDirectory (Join-Path $tempRoot 'missing folder')
    Assert-Condition ($code -ne 0) 'Registration accepted a missing working directory'
    Assert-Condition (-not (Select-String -LiteralPath $logPath -Pattern '^register:' -Quiet)) 'Registration proceeded after a missing working directory'
    Write-Host 'PASS scheduler working directory validation'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'scheduler unavailable'
    Assert-Condition ($code -ne 0) 'Scheduler listing error was swallowed'
    Write-Host 'PASS scheduler unavailable error propagation'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'register collision'
    Assert-Condition ($code -ne 0) ("Registration overwrote an unrelated task using the batch task name. " + (Get-Content -LiteralPath $logPath -Raw))
    Assert-Condition (-not (Select-String -LiteralPath $logPath -Pattern '^register:' -Quiet)) 'Registration proceeded after a task name collision'
    Write-Host 'PASS registration preserves unrelated task'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'remove unrelated Go task' -Remove
    Assert-Condition ($code -eq 0) "Unrelated task removal case failed with exit code $code"
    Assert-Condition (-not (Select-String -LiteralPath $logPath -Pattern '^(disable|stop|unregister):' -Quiet)) 'Unrelated Go task was modified'
    Assert-Condition (-not (Select-String -LiteralPath $logPath -Pattern '^registry-remove:' -Quiet)) 'Unrelated Run value was removed'
    Write-Host 'PASS removal preserves unrelated Go task'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'remove task mentions runner' -Remove
    Assert-Condition ($code -eq 0) "Unrelated task mention case failed with exit code $code"
    Assert-Condition (-not (Select-String -LiteralPath $logPath -Pattern '^(disable|stop|unregister):' -Quiet)) 'Task mentioning runner was modified'
    Write-Host 'PASS removal ignores a task that only mentions the runner'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'remove legacy installed AIO' -Remove
    Assert-Condition ($code -eq 0) "Legacy installed AIO removal failed with exit code $code"
    Assert-Condition (Select-String -LiteralPath $logPath -Pattern '^unregister:CMP30HX_Gen2_Unlock$' -Quiet) 'Legacy installed AIO task was not removed'
    Assert-Condition (Select-String -LiteralPath $logPath -Pattern '^registry-remove:40HXGen2$' -Quiet) 'Installed AIO Run value was not removed'
    Write-Host 'PASS removal of legacy installed AIO task'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'remove matching' -Remove
    Assert-Condition ($code -eq 0) "Matching task removal failed with exit code $code"
    $calls = Get-Content -LiteralPath $logPath
    Assert-Condition (($calls -join ',') -match 'disable:CMP30HX_Batch_Gen2_Unlock.*unregister:CMP30HX_Batch_Gen2_Unlock') 'Matching task was not disabled and unregistered'
    Write-Host 'PASS matching task removal'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'running task completes' -Remove
    Assert-Condition ($code -eq 0) "Running task completion case failed with exit code $code"
    $calls = Get-Content -LiteralPath $logPath
    Assert-Condition (($calls -join ',') -notmatch 'stop:') 'Removal forcibly stopped the active task'
    Assert-Condition (($calls -join ',') -match 'disable:CMP30HX_Batch_Gen2_Unlock.*unregister:CMP30HX_Batch_Gen2_Unlock') 'Removal did not wait for the active task to finish'
    Write-Host 'PASS running task completes without force stop'

    Remove-Item -LiteralPath $logPath -Force -ErrorAction SilentlyContinue
    $code = Invoke-SchedulerCase 'unregister fails' -Remove
    Assert-Condition ($code -ne 0) 'Unregistration failure was swallowed'
    Assert-Condition (-not (Select-String -LiteralPath $logPath -Pattern '^registry-cleanup:' -Quiet)) 'Registry cleanup continued after task removal failed'
    Write-Host 'PASS unregister failure propagation'
} finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
