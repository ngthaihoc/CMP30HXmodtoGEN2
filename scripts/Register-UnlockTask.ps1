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

function Test-ActionTargetsRunner($Task, [string]$Path) {
    foreach ($action in $Task.Actions) {
        $actionText = "$($action.Execute) $($action.Arguments)".Replace('/', '\')
        $pathIndex = $actionText.IndexOf($Path, [StringComparison]::OrdinalIgnoreCase)
        if ($pathIndex -lt 0) {
            continue
        }
        $pathEnd = $pathIndex + $Path.Length
        if ($pathEnd -eq $actionText.Length -or $actionText[$pathEnd] -in '"', ' ', [char]9) {
            return $true
        }
    }
    return $false
}

function Test-ValueTargetsRunner([string]$Value, [string]$Path) {
    $expandedValue = [Environment]::ExpandEnvironmentVariables($Value).Replace('/', '\')
    $pathIndex = $expandedValue.IndexOf($Path, [StringComparison]::OrdinalIgnoreCase)
    if ($pathIndex -lt 0) {
        return $false
    }
    $pathEnd = $pathIndex + $Path.Length
    return $pathEnd -eq $expandedValue.Length -or $expandedValue[$pathEnd] -in '"', ' ', [char]9
}

function Remove-RunValue([string]$KeyPath, [string]$ValueName, [string]$Path) {
    if (-not (Test-Path -LiteralPath $KeyPath)) {
        return
    }
    $key = Get-Item -LiteralPath $KeyPath
    $value = $key.GetValue($ValueName, $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    if ($null -ne $value -and (Test-ValueTargetsRunner ([string]$value) $Path)) {
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
        if ($null -eq $task -or -not (Test-ActionTargetsRunner $task $runnerPath)) {
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

    Remove-RunValue 'Registry::HKEY_LOCAL_MACHINE\Software\Microsoft\Windows\CurrentVersion\Run' 'CMP30HX_Gen2' $runnerPath
    Remove-RunValue 'Registry::HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run' '40HXGen2' $runnerPath
    foreach ($hive in Get-ChildItem -LiteralPath 'Registry::HKEY_USERS' -ErrorAction Stop) {
        Remove-RunValue "$($hive.PSPath)\Software\Microsoft\Windows\CurrentVersion\Run" '40HXGen2' $runnerPath
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
