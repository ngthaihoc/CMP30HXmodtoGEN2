Start-Sleep -Seconds 15
$statusFile = [System.IO.Path]::Combine($env:ProgramData, '40HXUnlock\gen2_status.txt')

if (Test-Path $statusFile) {
    $c = Get-Content $statusFile -Raw
    if ($c -match 'GPU TLS=Gen1|chua dat|chua d?t') {
        $devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'VEN_10DE&(DEV_2189|DEV_1F0B)' }
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
            Restart-Service NVDisplay.ContainerLocalSystem -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 1
        } catch {}
        
        $installerPath = Join-Path $pwd.Path '40HXInstaller.exe'
        if (Test-Path $installerPath) {
            Start-Process -FilePath $installerPath -ArgumentList '-gen2-30hx -silent' -Wait
        }
        
        try { & 'nvidia-smi' -pm 1 } catch {}
    }
}
