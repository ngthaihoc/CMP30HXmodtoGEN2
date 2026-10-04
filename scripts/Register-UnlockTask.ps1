param (
    [string]$ExePath,
    [string]$WorkingDirectory
)

if (-not [IO.Path]::IsPathRooted($ExePath)) {
    throw 'Duong dan installer khong hop le'
}

$action = if ($ExePath -match '\.bat$') {
    New-ScheduledTaskAction -Execute $env:ComSpec -Argument ("/c `"$ExePath`"") -WorkingDirectory $WorkingDirectory
} else {
    New-ScheduledTaskAction -Execute $ExePath -Argument '-gen2-30hx -silent -guard' -WorkingDirectory $WorkingDirectory
}

$t1 = New-ScheduledTaskTrigger -AtStartup
$t1.Delay = 'PT45S'

$t2 = New-ScheduledTaskTrigger -AtLogOn
$t2.Delay = 'PT10S'

$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries `
    -StartWhenAvailable -MultipleInstances Parallel `
    -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 5)

$principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest

Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force

try {
    $srv = New-Object -ComObject 'Schedule.Service'
    $srv.Connect()
    $task = $srv.GetFolder('\').GetTask('CMP30HX_Gen2_Unlock')
    $def = $task.Definition
    
    $tEvent = $def.Triggers.Create(0)
    $tEvent.Subscription = '<QueryList><Query Id="0" Path="System"><Select Path="System">*[System[Provider[@Name="Microsoft-Windows-Power-Troubleshooter"] and EventID=1]]</Select></Query></QueryList>'
    $tEvent.Delay = 'PT3S'
    $tEvent.Enabled = $true
    
    $srv.GetFolder('\').RegisterTaskDefinition('CMP30HX_Gen2_Unlock', $def, 4, $null, $null, 5, $null) | Out-Null
} catch {
    Write-Warning "Failed to add Event trigger: $_"
}
