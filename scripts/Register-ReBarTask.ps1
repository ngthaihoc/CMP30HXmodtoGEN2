param (
    [string]$TaskName,
    [string]$BatPath,
    [string]$WorkingDirectory
)

$action = New-ScheduledTaskAction -Execute $env:ComSpec -Argument ("/c `"$BatPath`"") -WorkingDirectory $WorkingDirectory

$t1 = New-ScheduledTaskTrigger -AtStartup
$t1.Delay = 'PT15S'

$t2 = New-ScheduledTaskTrigger -AtLogOn
$t2.Delay = 'PT5S'

$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
    -StartWhenAvailable -MultipleInstances IgnoreNew `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 5)

$principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force | Out-Null

try {
    $srv = New-Object -ComObject 'Schedule.Service'
    $srv.Connect()
    $task = $srv.GetFolder('\').GetTask($TaskName)
    $def = $task.Definition
    
    $tEvent = $def.Triggers.Create(0)
    $tEvent.Subscription = '<QueryList><Query Id="0" Path="System"><Select Path="System">*[System[Provider[@Name="Microsoft-Windows-Power-Troubleshooter"] and EventID=1]]</Select></Query></QueryList>'
    $tEvent.Delay = 'PT3S'
    $tEvent.Enabled = $true
    
    $srv.GetFolder('\').RegisterTaskDefinition($TaskName, $def, 4, $null, $null, 5, $null) | Out-Null
} catch {
    Write-Warning "Failed to add Event trigger: $_"
}
