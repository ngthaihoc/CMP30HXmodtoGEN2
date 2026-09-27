$devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'VEN_10DE&(DEV_2189|DEV_1F0B)' }

if ($devs) {
    foreach ($d in $devs) {
        try { & pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}
        try {
            Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue
            Start-Sleep -Milliseconds 800
            Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue
        } catch {}
        
        try {
            $st = (Get-PnpDevice -InstanceId $d.InstanceId -ErrorAction SilentlyContinue).Status
            if ($st -ne 'OK') {
                Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue
            }
        } catch {}
    }
    
    Start-Sleep -Seconds 2
    
    try {
        sc.exe config NVDisplay.ContainerLocalSystem start= auto | Out-Null
        Restart-Service NVDisplay.ContainerLocalSystem -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 1
        $s = Get-Service -Name NVDisplay.ContainerLocalSystem -ErrorAction SilentlyContinue
        if ($s -and $s.Status -ne 'Running') {
            Start-Service NVDisplay.ContainerLocalSystem -ErrorAction SilentlyContinue
        }
    } catch {}
} else {
    Write-Host 'Khong tim thay Instance ID qua PnP'
}
