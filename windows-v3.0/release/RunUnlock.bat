@echo off
setlocal
cd /d "%~dp0"
where nvidia-smi >nul 2>&1 && nvidia-smi -pm 1 >nul 2>&1
"40HXInstaller.exe" -gen2-30hx -silent
where nvidia-smi >nul 2>&1 && nvidia-smi -pm 1 >nul 2>&1
powershell -noProfile -ExecutionPolicy Bypass -Command "Start-Sleep -Seconds 15; $statusFile = [System.IO.Path]::Combine($env:ProgramData, '40HXUnlock\gen2_status.txt'); if (Test-Path $statusFile) { $c = Get-Content $statusFile -Raw; if ($c -match 'GPU TLS=Gen1|chua dat|chưa đạt') { $devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'VEN_10DE&(DEV_2189|DEV_1F0B)' }; foreach ($d in $devs) { try { & pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}; try { Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 800; Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } catch {}; try { $st = (Get-PnpDevice -InstanceId $d.InstanceId -ErrorAction SilentlyContinue).Status; if ($st -ne 'OK') { Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } } catch {} }; Start-Sleep -Seconds 2; try { Restart-Service NVDisplay.ContainerLocalSystem -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1 } catch {}; Start-Process -FilePath (Join-Path $pwd.Path '40HXInstaller.exe') -ArgumentList '-gen2-30hx -silent' -Wait; try { & 'nvidia-smi' -pm 1 } catch {} } }" >nul 2>&1
:: Don dep sach se driver BYOVD khoi kernel va System32 (tranh xung dot Riot Vanguard / Easy Anti-Cheat)
sc stop WinRing0_1_2_0 >nul 2>&1
sc delete WinRing0_1_2_0 >nul 2>&1
sc stop ThrottleStop >nul 2>&1
sc delete ThrottleStop >nul 2>&1
del /f /q "%SystemRoot%\System32\drivers\WinRing0x64.sys" >nul 2>&1
del /f /q "%SystemRoot%\System32\drivers\ThrottleStop.sys" >nul 2>&1
:: Dam bao service NVDisplay.ContainerLocalSystem va NVIDIA Control Panel hoat dong
sc config NVDisplay.ContainerLocalSystem start= auto >nul 2>&1
sc start NVDisplay.ContainerLocalSystem >nul 2>&1
endlocal
