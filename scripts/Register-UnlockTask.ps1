param (
    [Parameter(Mandatory)]
    [string]$ExePath,
    [string]$WorkingDirectory,
    [switch]$Remove,
    [ValidateRange(0, 300)]
    [int]$TaskWaitTimeoutSeconds = 30
)

$ErrorActionPreference = 'Stop'
$taskName = 'CMP30HX_Batch_Gen2_Unlock'
$legacyTaskNames = @(
    'CMP30HX_Gen2_Unlock',
    'CMP30HX_Gen2_Unlock_User',
    '40HXGen2Retry',
    '40HX PCIe Gen2 Bring-up'
)

if (-not [IO.Path]::IsPathRooted($ExePath)) {
    throw 'Duong dan runner khong hop le'
}

$runnerPath = [IO.Path]::GetFullPath($ExePath)
$ownedExePath = Join-Path (Split-Path -Parent $runnerPath) 'UnlockAIO.exe'
$ownedTargets = @($runnerPath, $ownedExePath)

function Test-ActionTargetsRunner($Task, [string]$Path) {
    foreach ($action in $Task.Actions) {
        $command = [Environment]::ExpandEnvironmentVariables([string]$action.Execute).Trim('"')
        $arguments = ([string]$action.Arguments).Trim()
        if ($command -ieq $Path) { return $true }
        if ($command -ieq $env:ComSpec -or $command -ieq 'cmd.exe') {
            if ($arguments -ieq ('/d /c ""' + $Path + '""') -or
                $arguments -ieq ('/c ""' + $Path + '""')) { return $true }
        }
    }
    return $false
}

function Test-ActionTargetsOwned($Task, [string[]]$Paths) {
    foreach ($path in $Paths) {
        if (Test-ActionTargetsRunner $Task $path) { return $true }
    }
    return $false
}

function Test-ValueTargetsRunner([string]$Value, [string]$Path) {
    $expandedValue = [Environment]::ExpandEnvironmentVariables($Value).Trim()
    $quotedPath = '"' + $Path + '"'
    if ($expandedValue -ieq $Path -or $expandedValue -ieq $quotedPath) { return $true }
    if ($expandedValue.StartsWith($quotedPath, [StringComparison]::OrdinalIgnoreCase)) {
        $tail = $expandedValue.Substring($quotedPath.Length)
        return $tail.Length -gt 0 -and [char]::IsWhiteSpace($tail[0])
    }
    return $false
}

function Test-ValueTargetsOwned([string]$Value, [string[]]$Paths) {
    foreach ($path in $Paths) {
        if (Test-ValueTargetsRunner $Value $path) { return $true }
    }
    return $false
}

function Remove-RunValue([string]$KeyPath, [string]$ValueName, [string[]]$Paths) {
    if (-not (Test-Path -LiteralPath $KeyPath)) {
        return
    }
    $key = Get-Item -LiteralPath $KeyPath
    $value = $key.GetValue($ValueName, $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    if ($null -ne $value -and (Test-ValueTargetsOwned ([string]$value) $Paths)) {
        Remove-ItemProperty -LiteralPath $KeyPath -Name $ValueName -ErrorAction Stop
    }
}

function Get-RootTask([string]$Name) {
    Get-ScheduledTask -TaskPath '\' -ErrorAction Stop | Where-Object TaskName -EQ $Name
}

if ($Remove) {
    $taskNames = @($taskName) + $legacyTaskNames
    foreach ($name in $taskNames) {
        $task = Get-RootTask $name
        $targets = if ($name -eq $taskName) { @($runnerPath) } else { $ownedTargets }
        if ($null -eq $task -or -not (Test-ActionTargetsOwned $task $targets)) {
            continue
        }
        Disable-ScheduledTask -InputObject $task -ErrorAction Stop | Out-Null
        if ($task.State -eq 'Running') {
            $deadline = [DateTime]::UtcNow.AddSeconds($TaskWaitTimeoutSeconds)
            do {
                Start-Sleep -Milliseconds 250
                $task = Get-RootTask $name
            } while ($null -ne $task -and $task.State -eq 'Running' -and [DateTime]::UtcNow -lt $deadline)
            if ($null -ne $task -and $task.State -eq 'Running') {
                throw "Scheduled Task $name is still running"
            }
        }
        if ($null -ne $task) {
            Unregister-ScheduledTask -InputObject $task -Confirm:$false -ErrorAction Stop
        }
        if (Get-RootTask $name) {
            throw "Khong the xoa Scheduled Task $name"
        }
    }

    Remove-RunValue 'Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Run' 'CMP30HX_Gen2' $ownedTargets
    Remove-RunValue 'Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Run' '40HXGen2' $ownedTargets
    Remove-RunValue 'Registry::HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run' '40HXGen2' $ownedTargets
    foreach ($hive in Get-ChildItem -LiteralPath 'Registry::HKEY_USERS' -ErrorAction Stop) {
        Remove-RunValue "$($hive.PSPath)\Software\Microsoft\Windows\CurrentVersion\Run" '40HXGen2' $ownedTargets
    }
    exit 0
}

if (-not [IO.Path]::IsPathRooted($WorkingDirectory)) {
    throw 'Thu muc lam viec khong hop le'
}
if (-not (Test-Path -LiteralPath $WorkingDirectory -PathType Container)) {
    throw 'Thu muc lam viec khong ton tai'
}
if (-not (Test-Path -LiteralPath $runnerPath -PathType Leaf)) {
    throw 'Runner khong ton tai'
}
if ([IO.Path]::GetExtension($runnerPath) -ine '.bat') {
    throw 'Runner phai la file .bat'
}

$existingTask = Get-RootTask $taskName
if ($existingTask -and -not (Test-ActionTargetsRunner $existingTask $runnerPath)) {
    throw "Scheduled Task $taskName da duoc su dung boi tac vu khac"
}

$actionArguments = '/d /c ""' + $runnerPath + '""'
$action = New-ScheduledTaskAction -Execute $env:ComSpec -Argument $actionArguments -WorkingDirectory $WorkingDirectory
$startupTrigger = New-ScheduledTaskTrigger -AtStartup
$startupTrigger.Delay = 'PT45S'
$logonTrigger = New-ScheduledTaskTrigger -AtLogOn
$logonTrigger.Delay = 'PT10S'
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit (New-TimeSpan -Minutes 5)
$principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask -TaskName $taskName -Action $action -Trigger @($startupTrigger, $logonTrigger) -Settings $settings -Principal $principal -Force -ErrorAction Stop | Out-Null
