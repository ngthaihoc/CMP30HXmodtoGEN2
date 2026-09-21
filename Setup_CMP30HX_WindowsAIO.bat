@echo off
setlocal
chcp 65001 >nul
title CMP 30HX Gen2 x16 Auto Setup

:: Phan tich toan bo tham so dong lenh
set "IS_MOCK=0"
set "NO_CHECK=0"
set "NO_WAIT=0"
for %%a in (%*) do (
    if /i "%%~a"=="-noadmin" set "IS_ADMIN=1"
    if /i "%%~a"=="/noadmin" set "IS_ADMIN=1"
    if /i "%%~a"=="-test" set "IS_MOCK=1"
    if /i "%%~a"=="/test" set "IS_MOCK=1"
    if /i "%%~a"=="-mock" set "IS_MOCK=1"
    if /i "%%~a"=="/mock" set "IS_MOCK=1"
    if /i "%%~a"=="-nocheck" set "NO_CHECK=1"
    if /i "%%~a"=="/nocheck" set "NO_CHECK=1"
    if /i "%%~a"=="-nowait" set "NO_WAIT=1"
    if /i "%%~a"=="-uninstall" set "DO_UNINSTALL=1"
    if /i "%%~a"=="/uninstall" set "DO_UNINSTALL=1"
    if /i "%%~a"=="-u" set "DO_UNINSTALL=1"
    if /i "%%~a"=="/u" set "DO_UNINSTALL=1"
)
if "%DO_UNINSTALL%"=="1" goto :uninstall

:: Kiem tra quyen Administrator (UAC da tang phong thu, thay the net session cu)
if not defined IS_ADMIN (
    set "IS_ADMIN=0"
    fltmc >nul 2>&1 && set "IS_ADMIN=1"
    if "%IS_ADMIN%"=="0" (
        fsutil dirty query %systemdrive% >nul 2>&1 && set "IS_ADMIN=1"
    )
    if "%IS_ADMIN%"=="0" (
        copy /b nul "%SystemRoot%\System32\__admintest_%random%.tmp" >nul 2>&1 && (
            del "%SystemRoot%\System32\__admintest_%random%.tmp" >nul 2>&1
            set "IS_ADMIN=1"
        )
    )
)

if "%IS_ADMIN%"=="0" (
    echo [!] Dang yeu cau quyen Administrator [UAC]...
    set "CURRENT_SCRIPT=%~f0"
    set "CURRENT_DIR=%~dp0"
    set "SCRIPT_ARGS=%*"
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$script=$env:CURRENT_SCRIPT; $dir=$env:CURRENT_DIR; $args=$env:SCRIPT_ARGS; $procArgs = if ($args) { '/c `\"' + $script + '`\" ' + $args } else { '/c `\"' + $script + '`\"' }; Start-Process -FilePath $env:ComSpec -ArgumentList $procArgs -WorkingDirectory $dir -Verb RunAs" >nul 2>&1
    if errorlevel 1 (
        echo.
        echo ================================================================
        echo [X] LOI: Khong the tu dong yeu cau quyen Administrator qua UAC.
        echo [!] Vui long nhap chuot phai vao file Setup_CMP30HX_WindowsAIO.bat
        echo     va chon 'Run as administrator' [Chay voi tu cach quan tri vien].
        echo     Hoac chay qua CMD Admin: Setup_CMP30HX_WindowsAIO.bat -noadmin
        echo ================================================================
        echo.
        pause
    )
    exit /b
)

cd /d "%~dp0"

echo ================================================================
if "%IS_MOCK%"=="1" (
    echo    CONG CU KIEM THU MO PHONG [MOCK TEST] GEN2 X16 CHO CMP 30HX
    echo    - Mo phong GPU CMP 30HX [TU116] ket Gen1 o lan goi dau
    echo    - Kich hoat chu trinh Soft Reset tu dong qua PnP
    echo    - Mo phong khoi phuc thanh cong Gen2 x16 [5.0 GT/s] o lan 2
    echo    - He thong: Powercfg, ASPM, Blocklist, Task SYSTEM chay that 100%%
) else (
    echo    CONG CU CAI DAT TU DONG GEN2 X16 CHO NVIDIA CMP 30HX (TU116)
)
echo ================================================================
echo.

set "SRC_INSTALLER="
if exist "%~dp0windows-v3.0\release\40HXInstaller.exe" (
    set "SRC_INSTALLER=%~dp0windows-v3.0\release\40HXInstaller.exe"
) else if exist "%~dp0release\40HXInstaller.exe" (
    set "SRC_INSTALLER=%~dp0release\40HXInstaller.exe"
) else if exist "%~dp040HXInstaller.exe" (
    set "SRC_INSTALLER=%~dp040HXInstaller.exe"
)

if not defined SRC_INSTALLER (
    echo [X] LOI: Khong tim thay 40HXInstaller.exe!
    echo Vui long dam bao ban da giai nen day du thu muc repository.
    echo.
    pause
    exit /b 1
)

set "SRC_CHECK="
if exist "%~dp0windows-v3.0\release\40HXCheck.exe" (
    set "SRC_CHECK=%~dp0windows-v3.0\release\40HXCheck.exe"
) else if exist "%~dp0release\40HXCheck.exe" (
    set "SRC_CHECK=%~dp0release\40HXCheck.exe"
) else if exist "%~dp040HXCheck.exe" (
    set "SRC_CHECK=%~dp040HXCheck.exe"
)

echo [*] Tim thay bo cai nguon: "%SRC_INSTALLER%"

:: 0. Trien khai co dinh vao thu muc he thong Program Files (tranh loi mat file sau reboot)
set "TARGET_DIR=%ProgramFiles%\40HXUnlock"
set "TARGET_INSTALLER=%TARGET_DIR%\40HXInstaller.exe"
set "TARGET_CHECK=%TARGET_DIR%\40HXCheck.exe"

if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%" >nul 2>&1
copy /y "%SRC_INSTALLER%" "%TARGET_INSTALLER%" >nul 2>&1
if defined SRC_CHECK copy /y "%SRC_CHECK%" "%TARGET_CHECK%" >nul 2>&1

:: Sao chep driver gen2 du phong vao ProgramData va ProgramFiles neu co
set "SRC_DRV="
if exist "%~dp0windows-v3.0\release\gen2\drivers" (
    set "SRC_DRV=%~dp0windows-v3.0\release\gen2\drivers"
) else if exist "%~dp0gen2\drivers" (
    set "SRC_DRV=%~dp0gen2\drivers"
) else if exist "%~dp0drivers" (
    set "SRC_DRV=%~dp0drivers"
)
if defined SRC_DRV (
    if not exist "%ProgramData%\40HXUnlock\drivers" mkdir "%ProgramData%\40HXUnlock\drivers" >nul 2>&1
    copy /y "%SRC_DRV%\*.sys" "%ProgramData%\40HXUnlock\drivers\" >nul 2>&1
    if not exist "%TARGET_DIR%\drivers" mkdir "%TARGET_DIR%\drivers" >nul 2>&1
    copy /y "%SRC_DRV%\*.sys" "%TARGET_DIR%\drivers\" >nul 2>&1
)

if exist "%TARGET_INSTALLER%" (
    echo [V] Da dong bo bo cai vao thu muc he thong: "%TARGET_DIR%"
    set "FINAL_INSTALLER=%TARGET_INSTALLER%"
    set "FINAL_DIR=%TARGET_DIR%"
) else (
    echo [!] Khong the copy vao Program Files, su dung bo cai tai cho: "%SRC_INSTALLER%"
    set "FINAL_INSTALLER=%SRC_INSTALLER%"
    set "FINAL_DIR=%~dp0"
)

:: Tao script runner tu dong polling va Soft Reset neu bi ket Gen1 sau reboot
set "FINAL_RUNNER=%FINAL_DIR%\RunUnlock.bat"
set "SRC_RUNNER="
if exist "%~dp0windows-v3.0\release\RunUnlock.bat" (
    set "SRC_RUNNER=%~dp0windows-v3.0\release\RunUnlock.bat"
) else if exist "%~dp0release\RunUnlock.bat" (
    set "SRC_RUNNER=%~dp0release\RunUnlock.bat"
) else if exist "%~dp0RunUnlock.bat" (
    set "SRC_RUNNER=%~dp0RunUnlock.bat"
)

if defined SRC_RUNNER (
    copy /y "%SRC_RUNNER%" "%FINAL_RUNNER%" >nul 2>&1
)

if not exist "%FINAL_RUNNER%" (
    powershell -NoProfile -ExecutionPolicy Bypass -Command "[System.IO.File]::WriteAllBytes($env:FINAL_RUNNER, [System.Convert]::FromBase64String('QGVjaG8gb2ZmDQpzZXRsb2NhbA0KY2QgL2QgIiV+ZHAwIg0KIjQwSFhJbnN0YWxsZXIuZXhlIiAtZ2VuMi0zMGh4IC1zaWxlbnQNCnBvd2Vyc2hlbGwgLU5vUHJvZmlsZSAtRXhlY3V0aW9uUG9saWN5IEJ5cGFzcyAtQ29tbWFuZCAiU3RhcnQtU2xlZXAgLVNlY29uZHMgMTU7ICRzdGF0dXNGaWxlID0gW1N5c3RlbS5JTy5QYXRoXTo6Q29tYmluZShgJGVudjpQcm9ncmFtRGF0YSwgJzQwSFhVbmxvY2tcZ2VuMl9zdGF0dXMudHh0Jyk7IGlmIChUZXN0LVBhdGggYCRzdGF0dXNGaWxlKSB7IGAkYyA9IEdldC1Db250ZW50IGAkc3RhdHVzRmlsZSAtUmF3OyBpZiAoYCRjIC1tYXRjaCAnR1BVIFRMUz1HZW4xfGNodWEgZGF0fGNoxrBhIMSR4bqhdCcpIHsgYCRkZXZzID0gR2V0LVBucERldmljZSAtUHJlc2VudE9ubHkgLUVycm9yQWN0aW9uIFNpbGVudGx5Q29udGludWUgfCBXaGVyZS1PYmplY3QgeyBgJF8uSGFyZHdhcmVJRCAtbWF0Y2ggJ1ZFTl8xMERFJihERVZfMjE4OXxERVZfMUYwQiknIH07IGZvcmVhY2ggKGAkZCBpbiBgJGRldnMpIHsgdHJ5IHsgcG5wdXRpbCAvcmVzdGFydC1kZXZpY2UgYCRkLkluc3RhbmNlSWQgPmAkbnVsbCAyPiYxIH0gY2F0Y2gge307IHRyeSB7IERpc2FibGUtUG5wRGV2aWNlIC1JbnN0YW5jZUlkIGAkZC5JbnN0YW5jZUlkIC1Db25maXJtOmAkZmFsc2UgLUVycm9yQWN0aW9uIFNpbGVudGx5Q29udGludWU7IFN0YXJ0LVNsZWVwIC1NaWxsaXNlY29uZHMgODAwOyBFbmFibGUtUG5wRGV2aWNlIC1JbnN0YW5jZUlkIGAkZC5JbnN0YW5jZUlkIC1Db25maXJtOmAkZmFsc2UgLUVycm9yQWN0aW9uIFNpbGVudGx5Q29udGludWUgfSBjYXRjaCB7fSB9OyBTdGFydC1TbGVlcCAtU2Vjb25kcyAyOyBTdGFydC1Qcm9jZXNzIC1GaWxlUGF0aCAoSm9pbi1QYXRoIGAkcHdkLlBhdGggJzQwSFhJbnN0YWxsZXIuZXhlJykgLUFyZ3VtZW50TGlzdCAnLWdlbjItMzBoeCAtc2lsZW50JyAtV2FpdCB9IH0iID5udWwgMj4mMQplbmRsb2NhbA0K'))" >nul 2>&1
)
echo [V] Da thiet lap script duy tri khoi dong: "%FINAL_RUNNER%"
echo.

:: 1. Don dep task retry cu (neu co), tranh vong lap retry Gen3/Gen2 cu
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1

:: 2. Tat triet de Fast Startup, Hybrid Sleep va PCIe ASPM tren toan bo Power Plan
echo [1/6] Dang tat Fast Startup, Hybrid Sleep va PCIe ASPM toan he thong...
powercfg -h off >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f >nul 2>&1

powershell -NoProfile -ExecutionPolicy Bypass -Command "$schemes = powercfg -list | ForEach-Object { if ($_ -match '([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})') { $matches[1] } }; foreach ($s in $schemes) { powercfg -setacvalueindex $s SUB_PCIEXPRESS ASPM 0 2>$null; powercfg -setdcvalueindex $s SUB_PCIEXPRESS ASPM 0 2>$null; powercfg -setacvalueindex $s SUB_SLEEP HYBRIDSLEEP 0 2>$null; powercfg -setdcvalueindex $s SUB_SLEEP HYBRIDSLEEP 0 2>$null }; powercfg -setactive SCHEME_CURRENT 2>$null" >nul 2>&1

powershell -NoProfile -ExecutionPolicy Bypass -Command "$base = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}'; Get-ChildItem $base -ErrorAction SilentlyContinue | ForEach-Object { $p = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue; if ($p.ProviderName -match 'NVIDIA' -or $p.DriverDesc -match 'NVIDIA|CMP') { Set-ItemProperty -Path $_.PSPath -Name 'DisableAspm' -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue; Set-ItemProperty -Path $_.PSPath -Name 'RMDisableLinkDownshift' -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue } }" >nul 2>&1

where nvidia-smi >nul 2>&1 && nvidia-smi -pm 1 >nul 2>&1
sc config NVDisplay.ContainerLocalSystem start= auto >nul 2>&1
sc start NVDisplay.ContainerLocalSystem >nul 2>&1

echo       [OK] Da vo hieu hoa Fast Startup (Hiberboot) va PCIe ASPM toan he thong.

:: 3. Tat Microsoft Vulnerable Driver Blocklist (tranh Windows chan driver sau reboot)
echo [2/6] Dang tat Microsoft Vulnerable Driver Blocklist...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f >nul 2>&1
if errorlevel 1 (
    echo       [X] Khong tat duoc Driver Blocklist. Kiem tra chinh sach bao mat Windows.
) else (
    reg query "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x0" >nul 2>&1
    if errorlevel 1 (echo       [!] Driver Blocklist chua xac nhan OFF.) else (echo       [OK] Driver Blocklist da tat.)
)

:: 4. Tat Memory Integrity (Core Isolation / HVCI) de driver MMIO khong bi chan
echo [3/6] Dang kiem tra va tat Memory Integrity (HVCI)...
set "NEED_REBOOT=0"
reg query "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x1" >nul 2>&1
if not errorlevel 1 set "NEED_REBOOT=1"
reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f >nul 2>&1
if errorlevel 1 (
    echo       [X] Khong tat duoc Memory Integrity. Windows co the van chan driver kernel.
) else (
    reg query "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x1" >nul 2>&1
    if not errorlevel 1 (
        echo       [X] Memory Integrity van dang ON. Can reboot va kiem tra lai Windows Security.
        set "NEED_REBOOT=1"
    ) else (
        if "%NEED_REBOOT%"=="1" (echo       [!] Da dat HVCI=OFF; can reboot de driver duoc giai phong.) else (echo       [OK] Memory Integrity dang OFF.)
    )
)

:: 5. Dang ky Scheduled Task SYSTEM da kich hoat (Startup 15s + Logon 5s + Wake from Sleep)
echo [4/6] Dang tao Scheduled Task SYSTEM va Run Key duy tri Gen2...
set "TASK_OK=0"
set "FINAL_EXE=%FINAL_RUNNER%"
if not exist "%FINAL_EXE%" set "FINAL_EXE=%FINAL_INSTALLER%"
set "FINAL_WD=%FINAL_DIR%"

powershell -NoProfile -ExecutionPolicy Bypass -Command "$exe=$env:FINAL_EXE; $dir=$env:FINAL_WD; if (-not [IO.Path]::IsPathRooted($exe)) { throw 'Duong dan installer khong hop le' }; $action = if ($exe -match '\.bat$') { New-ScheduledTaskAction -Execute $env:ComSpec -Argument ('/c `\"' + $exe + '`\"') -WorkingDirectory $dir } else { New-ScheduledTaskAction -Execute $exe -Argument '-gen2-30hx -silent' -WorkingDirectory $dir }; $t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'; $t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'; $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit (New-TimeSpan -Minutes 5); $principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest; Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force; try { $srv = New-Object -ComObject 'Schedule.Service'; $srv.Connect(); $task = $srv.GetFolder('\').GetTask('CMP30HX_Gen2_Unlock'); $def = $task.Definition; $tEvent = $def.Triggers.Create(0); $tEvent.Subscription = '<QueryList><Query Id=''0'' Path=''System''><Select Path=''System''>*[System[Provider[@Name=''Microsoft-Windows-Power-Troubleshooter''] and EventID=1]]</Select></Query></QueryList>'; $tEvent.Delay = 'PT3S'; $tEvent.Enabled = $true; $srv.GetFolder('\').RegisterTaskDefinition('CMP30HX_Gen2_Unlock', $def, 4, $null, $null, 5, $null) } catch {}" >nul 2>&1

if not errorlevel 1 set "TASK_OK=1"
if "%TASK_OK%"=="1" schtasks /query /tn "CMP30HX_Gen2_Unlock" >nul 2>&1 || set "TASK_OK=0"

if "%TASK_OK%"=="0" (
    schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"%FINAL_RUNNER%\"" /sc onstart /delay 0000:15 /rl highest /ru "NT AUTHORITY\SYSTEM" /f >nul 2>&1
    if not errorlevel 1 set "TASK_OK=1"
)

:: Bao hiem kep: Dang ky Registry Run key cho HKLM va HKCU (phong thu neu Task Scheduler bi chan)
reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /t REG_SZ /d "\"%FINAL_RUNNER%\"" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /t REG_SZ /d "\"%FINAL_RUNNER%\"" /f >nul 2>&1

if "%TASK_OK%"=="1" (
    echo       [OK] Scheduled Task SYSTEM da duoc kich hoat thanh cong.
    echo           - Kich hoat khi he thong khoi dong [AtStartup: delay 15 giay].
    echo           - Kich hoat khi nguoi dung dang nhap [AtLogOn: delay 5 giay].
    echo           - Kich hoat khi thuc giac tu che do ngu [Wake from Sleep: delay 3 giay].
    echo           - Tich hop them Run Key du phong tai Registry HKLM va HKCU.
) else (
    echo       [!] Scheduled Task SYSTEM gap truc trac, da kich hoat che do du phong Registry Run.
)

:: 6. Kich hoat mo khoa Gen2 ngay lap tuc
echo.
if "%NEED_REBOOT%"=="1" (
    echo [!] HVCI vua duoc dat OFF trong Registry nhung chua co hieu luc trong phien nay.
    echo     Buoc hien tai co the khong nap duoc driver kernel; sau khi ket thuc hay reboot truoc khi danh gia.
)
echo [5/6] Dang kich hoat Gen2 x16 va toi uu MRRS 512B ngay...
if "%IS_MOCK%"=="1" (
    echo       [*] [MOCK TEST] Mo phong Installer lan 1: Phat hien CMP 30HX nhung dang bi ket Gen1...
    if not exist "%ProgramData%\40HXUnlock" mkdir "%ProgramData%\40HXUnlock" >nul 2>&1
    (
        echo ==== 40HX Gen2 Ket qua [MOCK TEST - LAN 1] ====
        echo GPU: NVIDIA CMP 30HX [TU116] [DEV_2189]
        echo PCIe Link Width: x16
        echo PCIe Link Speed: GPU TLS=Gen1 [2.5 GT/s]
        echo Trang thai: chua dat muc tieu Gen2! [Driver mod / iGPU dang giu DMA context]
        echo Quyen thuc thi: Quan tri vien / SYSTEM
    ) > "%ProgramData%\40HXUnlock\gen2_status.txt"
    set "UNLOCK_OK=0"
) else (
    "%FINAL_INSTALLER%" -gen2-30hx -silent
    if errorlevel 1 (
        echo       [X] Installer bao loi khi chay [exit code khac 0].
        echo           Kiem tra driver WinRing0/ThrottleStop, HVCI va quyen Administrator.
        set "UNLOCK_OK=0"
    ) else (
        set "UNLOCK_OK=1"
    )
)

if exist "%ProgramData%\40HXUnlock\gen2_status.txt" (
    echo       [OK] Chi tiet ket qua tu installer:
    echo       --------------------------------------------------------
    type "%ProgramData%\40HXUnlock\gen2_status.txt"
    echo.
    echo       --------------------------------------------------------
) else (
    echo       [!] Canh bao: Installer chay xong nhung khong tim thay file gen2_status.txt.
    echo           Hay chay lai 40HXInstaller.exe -status de kiem tra.
)

:: Kiem tra chu trinh Soft Reset neu GPU TLS bi ket o Gen1 (dac biet tren he thong GPU kep iGPU + CMP 30HX hoac driver mod)
set "NEED_DEV_RESET=0"
if "%UNLOCK_OK%"=="0" set "NEED_DEV_RESET=1"
if exist "%ProgramData%\40HXUnlock\gen2_status.txt" (
    findstr /i "GPU TLS=Gen1" "%ProgramData%\40HXUnlock\gen2_status.txt" >nul 2>&1 && set "NEED_DEV_RESET=1"
    findstr /i "chua dat" "%ProgramData%\40HXUnlock\gen2_status.txt" >nul 2>&1 && set "NEED_DEV_RESET=1"
    findstr /i "chưa đạt" "%ProgramData%\40HXUnlock\gen2_status.txt" >nul 2>&1 && set "NEED_DEV_RESET=1"
)

if "%NEED_DEV_RESET%"=="1" (
    echo.
    echo       [!] Phat hien GPU TLS van o Gen1 [driver mod / iGPU dang giu DMA context].
    echo       [*] Dang tu dong thuc hien chu trinh Soft Reset [Disable - Enable qua PnP]...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'VEN_10DE&(DEV_2189|DEV_1F0B)' }; if ($devs) { foreach ($d in $devs) { try { pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}; try { Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 800; Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } catch {} }; Start-Sleep -Seconds 2 } else { Write-Host 'Khong tim thay Instance ID qua PnP' }" >nul 2>&1
    echo       [*] Dang chay lai lenh mo khoa Gen2 sau khi Soft Reset card...
    if "%IS_MOCK%"=="1" (
        echo       [*] [MOCK TEST] Mo phong Installer lan 2: Soft Reset thanh cong, GPU bung Gen2 x16 [5.0 GT/s]!
        (
            echo ==== 40HX Gen2 Ket qua [MOCK TEST - LAN 2 SAU SOFT RESET] ====
            echo GPU: NVIDIA CMP 30HX [TU116] [DEV_2189]
            echo PCIe Link Width: x16
            echo PCIe Link Speed: GPU TLS=Gen2 [5.0 GT/s]
            echo Ket qua: da dat muc tieu Gen2 thanh cong!
            echo MRRS: 512B [Da toi uu]
            echo Quyen thuc thi: Quan tri vien / SYSTEM
        ) > "%ProgramData%\40HXUnlock\gen2_status.txt"
        set "UNLOCK_OK=1"
    ) else (
        "%FINAL_INSTALLER%" -gen2-30hx -silent
        if not errorlevel 1 set "UNLOCK_OK=1"
    )
    if exist "%ProgramData%\40HXUnlock\gen2_status.txt" (
        echo.
        echo       [OK] Ket qua sau khi Soft Reset:
        echo       --------------------------------------------------------
        type "%ProgramData%\40HXUnlock\gen2_status.txt"
        echo.
        echo       --------------------------------------------------------
    )
)

:: 7. Kiem tra trang thai chan doan
echo.
echo [6/6] Kiem tra trang thai sau khi mo khoa...
if not "%NO_CHECK%"=="1" (
    if exist "%TARGET_CHECK%" (
        echo [*] Tim thay cong cu chan doan: "%TARGET_CHECK%"
        echo [*] Dang khoi chay cua so chan doan 40HXCheck...
        start "" "%TARGET_CHECK%"
    ) else (
        "%FINAL_INSTALLER%" -status
    )
) else (
    echo [*] Bo qua khoi chay 40HXCheck [-nocheck].
)

echo.
echo ================================================================
if "%UNLOCK_OK%"=="1" (
    if "%IS_MOCK%"=="1" (
        echo  [V] KIEM THU MO PHONG [MOCK TEST] HOAN TAT MY MAN:
        echo  - Mo phong phat hien GPU CMP 30HX ket Gen1 o lan chay 1: [THANH CONG].
        echo  - Tu dong kich hoat chu trinh Soft Reset card qua PnP: [THANH CONG].
        echo  - Mo phong tai bung toc do Gen2 x16 [5.0 GT/s] o lan chay 2: [THANH CONG].
        echo  - Toan bo cac buoc he thong da thuc hien that 100%%:
        echo    + Powercfg: Tat Fast Startup, Hybrid Sleep, PCIe ASPM tat ca Power Plan.
        echo    + CI Policy: Tat Microsoft Vulnerable Driver Blocklist.
        echo    + HVCI: Kiem tra trang thai Memory Integrity.
        echo    + Persistence: Scheduled Task SYSTEM va Registry Run Key duy tri Gen2.
        echo.
        echo  - Ban co the dung: Setup_CMP30HX_WindowsAIO.bat -uninstall de don dep sau kiem thu.
    ) else (
        echo  [V] CAI DAT HOAN TAT - Gen2 da duoc cau hinh ben vung.
        echo  - Da thiet lap da co che: Scheduled Task SYSTEM + Registry Run Key.
        echo  - Tu dong duy tri Gen2 tren moi lan Boot, Dang nhap va Wake from Sleep!
        echo.
        echo  - LUU Y QUAN TRONG VE GEN 1 KHI VUA KHOI DONG / IDLE:
        echo    + Link PCIe se o Gen1 x16 khi card o che do ranh [Idle Power Saving].
        echo    + Khi co tai 3D/CUDA/AIDA64/FurMark, card se tu dong bung toc do len Gen2 x16.
        echo    + Neu GPU-Z bao Gen1: nhap vao dau cham hoi [?] canh Bus Interface de chay Render Test!
        echo  - Neu sau khi reboot co tai ma GPU van Gen1: kiem tra HVCI, riser, tiep xuc lane va BIOS khe PCIe.
    )
) else (
    echo  [X] CAI DAT CHUA HOAN TAT - chua xac nhan duoc Gen2 trong phien hien tai.
    if "%NEED_REBOOT%"=="1" (
        echo  - LUU Y: Memory Integrity [HVCI] vua duoc tat, nhung can KHOI DONG LAI MAY de ap dung.
        echo    Sau khi reboot, Scheduled Task SYSTEM se tu dong thu nap driver va mo khoa Gen2.
    ) else (
        echo  - Khong ket luan thanh cong chi dua tren Root Port Gen2.
        echo  - Kiem tra file status va log chi tiet: %TEMP%\40HX_installer.log
        echo  - Neu da thu moi cach van loi: go bo sach driver cu bang DDU roi cai lai driver NVIDIA moi nhat.
    )
)
if "%TASK_OK%"=="1" (
    echo  - Sau khi reboot, he thong se tu dong mo khoa sau 15 giay khoi dong hoac 5 giay dang nhap.
) else (
    echo  - [Canh bao] Scheduled Task chua san sang; sau reboot phai chay lai Setup hoac lenh mo khoa thu cong.
)
if "%NEED_REBOOT%"=="1" call :warn_reboot
echo ================================================================
echo.
if not "%NO_WAIT%"=="1" pause
if "%UNLOCK_OK%"=="1" exit /b 0
exit /b 1

:uninstall
echo ================================================================
echo    GO BO TU DONG KHOI DONG CMP 30HX GEN2 UNLOCK
echo ================================================================
schtasks /delete /tn "CMP30HX_Gen2_Unlock" /f >nul 2>&1
schtasks /delete /tn "CMP30HX_Gen2_Unlock_User" /f >nul 2>&1
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1
schtasks /delete /tn "40HX PCIe Gen2 Bring-up" /f >nul 2>&1
reg delete "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /f >nul 2>&1
if exist "%ProgramFiles%\40HXUnlock" (
    rmdir /s /q "%ProgramFiles%\40HXUnlock" >nul 2>&1
)
if exist "%ProgramData%\40HXUnlock\gen2_status.txt" (
    del /f /q "%ProgramData%\40HXUnlock\gen2_status.txt" >nul 2>&1
)
echo [V] Da xoa toan bo cac Scheduled Task, Run key va thu muc he thong lien quan.
echo.
if not "%NO_WAIT%"=="1" pause
exit /b 0


:warn_reboot
echo.
echo  [!] LUU Y BAT BUOC: He thong vua tat Memory Integrity [Core Isolation].
echo      Ban PHAI KHOI DONG LAI MAY de Windows giai phong driver kernel.
echo      Sau reboot: cho Scheduled Task chay du 15 giay, tao tai 3D/CUDA,
echo      sau do moi dung GPU-Z/40HXCheck de danh gia Gen2.
exit /b 0
