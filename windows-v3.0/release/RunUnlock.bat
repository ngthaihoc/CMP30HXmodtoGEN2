@echo off
setlocal
cd /d "%~dp0"
"40HXInstaller.exe" -gen2-30hx -silent
powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Sleep -Seconds 15; $statusFile = [System.IO.Path]::Combine($env:ProgramData, '40HXUnlock\gen2_status.txt'); if (Test-Path $statusFile) { $c = Get-Content $statusFile -Raw; if ($c -match 'GPU TLS=Gen1|chua dat|chưa đạt') { $devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'VEN_10DE&(DEV_2189|DEV_1F0B)' }; foreach ($d in $devs) { try { pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}; try { Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 800; Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } catch {} }; Start-Sleep -Seconds 2; Start-Process -FilePath (Join-Path $pwd.Path '40HXInstaller.exe') -ArgumentList '-gen2-30hx -silent' -Wait } }" >nul 2>&1
endlocal
