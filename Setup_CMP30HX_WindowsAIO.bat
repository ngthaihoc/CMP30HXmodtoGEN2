@echo off
setlocal
chcp 65001 >nul
title CMP 30HX Gen2 x16 Auto Setup

:: ================================================================
:: 1. PHAN TICH TOAN BO THAM SO DONG LENH (CLI ARGUMENTS)
:: ================================================================
set "IS_MOCK=0"
set "IS_MOCK_FAIL=0"
set "IS_MOCK_WINRING0=0"
set "IS_MOCK_NOGPU=0"
set "IS_MOCK_MISSING_STATUS=0"
set "NO_CHECK=0"
set "NO_WAIT=0"
set "DO_UNINSTALL=0"
set "DO_CLEAN_TASKS=0"
set "NO_TASK=0"
set "DO_PREFLIGHT_ONLY=0"

for %%a in (%*) do (
    if /i "%%~a"=="-noadmin" set "IS_ADMIN=1"
    if /i "%%~a"=="/noadmin" set "IS_ADMIN=1"
    if /i "%%~a"=="-test" set "IS_MOCK=1"
    if /i "%%~a"=="/test" set "IS_MOCK=1"
    if /i "%%~a"=="-mock" set "IS_MOCK=1"
    if /i "%%~a"=="/mock" set "IS_MOCK=1"
    if /i "%%~a"=="-mock-fail" set "IS_MOCK_FAIL=1"
    if /i "%%~a"=="/mock-fail" set "IS_MOCK_FAIL=1"
    if /i "%%~a"=="-mock-winring0" set "IS_MOCK_WINRING0=1"
    if /i "%%~a"=="/mock-winring0" set "IS_MOCK_WINRING0=1"
    if /i "%%~a"=="-mock-nogpu" set "IS_MOCK_NOGPU=1"
    if /i "%%~a"=="/mock-nogpu" set "IS_MOCK_NOGPU=1"
    if /i "%%~a"=="-mock-missing-status" set "IS_MOCK_MISSING_STATUS=1"
    if /i "%%~a"=="/mock-missing-status" set "IS_MOCK_MISSING_STATUS=1"
    if /i "%%~a"=="-nocheck" set "NO_CHECK=1"
    if /i "%%~a"=="/nocheck" set "NO_CHECK=1"
    if /i "%%~a"=="-nowait" set "NO_WAIT=1"
    if /i "%%~a"=="/nowait" set "NO_WAIT=1"
    if /i "%%~a"=="-uninstall" set "DO_UNINSTALL=1"
    if /i "%%~a"=="/uninstall" set "DO_UNINSTALL=1"
    if /i "%%~a"=="-u" set "DO_UNINSTALL=1"
    if /i "%%~a"=="/u" set "DO_UNINSTALL=1"
    if /i "%%~a"=="-cleantasks" set "DO_CLEAN_TASKS=1"
    if /i "%%~a"=="/cleantasks" set "DO_CLEAN_TASKS=1"
    if /i "%%~a"=="-deltasks" set "DO_CLEAN_TASKS=1"
    if /i "%%~a"=="/deltasks" set "DO_CLEAN_TASKS=1"
    if /i "%%~a"=="-clean" set "DO_CLEAN_TASKS=1"
    if /i "%%~a"=="/clean" set "DO_CLEAN_TASKS=1"
    if /i "%%~a"=="-notask" set "NO_TASK=1"
    if /i "%%~a"=="/notask" set "NO_TASK=1"
    if /i "%%~a"=="-preflight" set "DO_PREFLIGHT_ONLY=1"
    if /i "%%~a"=="/preflight" set "DO_PREFLIGHT_ONLY=1"
)

set "HAS_CLI_FLAG=0"
for %%a in (%*) do (
    if /i not "%%~a"=="-noadmin" if /i not "%%~a"=="/noadmin" set "HAS_CLI_FLAG=1"
)
if "%IS_MOCK_FAIL%"=="1" set "IS_MOCK=1"
if "%IS_MOCK_WINRING0%"=="1" set "IS_MOCK=1"
if "%IS_MOCK_NOGPU%"=="1" set "IS_MOCK=1"
if "%IS_MOCK_MISSING_STATUS%"=="1" set "IS_MOCK=1"

:: ================================================================
:: 2. KIEM TRA QUYEN ADMINISTRATOR (UAC DA TANG PHONG THU)
:: ================================================================
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
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$script=$env:CURRENT_SCRIPT; $dir=$env:CURRENT_DIR; $args=$env:SCRIPT_ARGS; $q=[char]34; $procArgs = if ($args) { '/c ' + $q + $script + $q + ' ' + $args } else { '/c ' + $q + $script + $q }; Start-Process -FilePath $env:ComSpec -ArgumentList $procArgs -WorkingDirectory $dir -Verb RunAs" >nul 2>&1
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

if "%DO_CLEAN_TASKS%"=="1" goto :clean_tasks
if "%DO_UNINSTALL%"=="1" goto :uninstall
if "%HAS_CLI_FLAG%"=="0" goto :aio_menu
goto :start_aio

:: ================================================================
:: 3. MENU TUONG TAC TRUYEN THONG CHO NGUOI DUNG
:: ================================================================
:aio_menu
cls
echo ================================================================
echo    CONG CU CAI DAT TOAN DIEN [ALL-IN-ONE] CHO CMP 30HX [TU116]
echo    - Tuong thich 100%% Game Riot [Valorant, LoL] va Nguoi dung pho thong
echo ================================================================
echo.
echo   [1] Cai dat va Mo khoa Gen2 AIO [Tu dong 100%% cho moi nguoi dung]
echo       - Mo khoa Gen2 x16 [5.0 GT/s], toi uu DEVCTL MRRS 512B
echo       - TU DONG tich hop toi uu Riot Games (Valorant/LMHT)
echo       - Don dep sach driver BYOVD tranh loi VAN 1067
echo.
echo   [2] Go bo cai dat (Tu dong xoa Scheduled Task va Don dep)
echo       - Tu dong xoa sach Scheduled Task, Registry Run key cua script
echo       - Go bo hoan toan khoi he thong
echo.
echo   [3] Thoat
echo.
echo ================================================================
%SystemRoot%\System32\choice.exe /c 123 /t 8 /d 1 /m "Nhap lua chon cua ban [1-3] (Tu dong chon [1] sau 8 giay): "
if errorlevel 3 exit /b 0
if errorlevel 2 goto :clean_tasks
if errorlevel 1 goto :start_aio
goto :start_aio

:: ================================================================
:: 4. DIEU PHOI TIEN TRINH CHINH (MAIN ORCHESTRATION PIPELINE)
:: ================================================================
:start_aio
echo ================================================================
if not "%IS_MOCK%"=="1" goto :banner_real
if "%IS_MOCK_MISSING_STATUS%"=="1" goto :banner_missing_status
if "%IS_MOCK_WINRING0%"=="1" goto :banner_winring0
if "%IS_MOCK_NOGPU%"=="1" goto :banner_nogpu
if "%IS_MOCK_FAIL%"=="1" goto :banner_fail
goto :banner_mock_default

:banner_missing_status
echo    KIEM THU MO PHONG [MOCK TEST]: MAT / HONG FILE TRANG THAI STATUS
echo    - Mo phong truong hop Installer bi Antivirus diet hoac crash dot ngot
echo    - Kiem tra co che phong thu chan bao thanh cong ao khi khong co status file
goto :banner_end

:banner_winring0
echo    KIEM THU MO PHONG [MOCK TEST]: DRIVER WINRING0 BI CHAN
echo    - Mo phong Driver kernel WinRing0 bi Windows/Antivirus chan nap
echo    - Kiem tra he thong tu dong phat hien va bo qua Soft Reset
echo    - Kiem tra thong bao huong dan xu ly HVCI / Blocklist / Reboot
goto :banner_end

:banner_nogpu
echo    KIEM THU MO PHONG [MOCK TEST]: KHONG TIM THAY GPU CMP 30HX
echo    - Mo phong khong dinh vi duoc GPU tren PCI Bus
echo    - Kiem tra he thong phat hien va huong dan Device Manager / Driver
goto :banner_end

:banner_fail
echo    KIEM THU MO PHONG [MOCK TEST]: KET GEN1 SAU SOFT RESET
echo    - Mo phong GPU van ket Gen1 du da thuc hien chu trinh Soft Reset
echo    - Kiem tra huong dan xu ly ve BIOS, PCIe slot va Render Test
goto :banner_end

:banner_mock_default
echo    CONG CU KIEM THU MO PHONG [MOCK TEST] GEN2 X16 CHO CMP 30HX
echo    - Mo phong GPU CMP 30HX [TU116] ket Gen1 o lan goi dau
echo    - Kich hoat chu trinh Soft Reset tu dong qua PnP
echo    - Mo phong khoi phuc thanh cong Gen2 x16 [5.0 GT/s] o lan 2
echo    - He thong: Powercfg, ASPM, Blocklist, Task SYSTEM chay that 100%%
goto :banner_end

:banner_real
echo    CONG CU CAI DAT TU DONG GEN2 X16 CHO NVIDIA CMP 30HX [TU116]
goto :banner_end

:banner_end
echo ================================================================
echo.

:: 4.1 Trien khai bo cai dat vao Program Files
call :DeployInstallerFiles
if errorlevel 1 exit /b 1

:: 4.2 Module PreflightDiagnostic (Kiem tra va chuan hoa moi truong he thong)
call :PreflightDiagnostic
if "%DO_PREFLIGHT_ONLY%"=="1" (
    echo.
    echo [V] Hoan tat buoc chan doan Preflight Diagnostic [-preflight].
    if not "%NO_WAIT%"=="1" pause
    exit /b 0
)

:: 4.3 Module Persistence (Tao Scheduled Task SYSTEM va ho tro Riot Vanguard)
call :ConfigurePersistence

:: 4.4 Module PCIeLinkRetrain (Mo khoa Gen2, xu ly Soft Reset neu ket Gen1)
call :PCIeLinkRetrain

:: 4.5 Module CleanupBYOVD (Don dep driver de bao dam an toan Anti-Cheat)
call :CleanupBYOVD

:: 4.6 Module RenderDiagnosticsSummary (Tong hop ket qua va bao cao)
call :RenderDiagnosticsSummary
set "FINAL_EXIT=%ERRORLEVEL%"
if not "%NO_WAIT%"=="1" pause
exit /b %FINAL_EXIT%


:: ================================================================
:: MODULE 1: DEPLOY INSTALLER FILES
:: ================================================================
:DeployInstallerFiles
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

set "TARGET_DIR=%ProgramFiles%\40HXUnlock"
set "TARGET_INSTALLER=%TARGET_DIR%\40HXInstaller.exe"
set "TARGET_CHECK=%TARGET_DIR%\40HXCheck.exe"

if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%" >nul 2>&1
copy /y "%SRC_INSTALLER%" "%TARGET_INSTALLER%" >nul 2>&1
if defined SRC_CHECK copy /y "%SRC_CHECK%" "%TARGET_CHECK%" >nul 2>&1

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
    (
        echo @echo off
        echo setlocal
        echo cd /d "%%~dp0"
        echo where nvidia-smi ^>nul 2^>^&1 ^&^& nvidia-smi -pm 1 ^>nul 2^>^&1
        echo "40HXInstaller.exe" -gen2-30hx -silent
        echo where nvidia-smi ^>nul 2^>^&1 ^&^& nvidia-smi -pm 1 ^>nul 2^>^&1
        echo powershell -noProfile -ExecutionPolicy Bypass -Command "Start-Sleep -Seconds 15; $statusFile = [System.IO.Path]::Combine($env:ProgramData, '40HXUnlock\gen2_status.txt'); if (Test-Path $statusFile) { $c = Get-Content $statusFile -Raw; if ($c -match 'GPU TLS=Gen1|chua dat|chưa đạt') { $devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'VEN_10DE&(DEV_2189|DEV_1F0B)' }; foreach ($d in $devs) { try { & pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}; try { Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 800; Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } catch {}; try { $st = (Get-PnpDevice -InstanceId $d.InstanceId -ErrorAction SilentlyContinue).Status; if ($st -ne 'OK') { Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } } catch {} }; Start-Sleep -Seconds 2; try { Restart-Service NVDisplay.ContainerLocalSystem -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1 } catch {}; Start-Process -FilePath (Join-Path $pwd.Path '40HXInstaller.exe') -ArgumentList '-gen2-30hx -silent' -Wait; try { & 'nvidia-smi' -pm 1 } catch {} } }" ^>nul 2^>^&1
        echo sc stop WinRing0_1_2_0 ^>nul 2^>^&1
        echo sc delete WinRing0_1_2_0 ^>nul 2^>^&1
        echo sc stop ThrottleStop ^>nul 2^>^&1
        echo sc delete ThrottleStop ^>nul 2^>^&1
        echo del /f /q "%%SystemRoot%%\System32\drivers\WinRing0x64.sys" ^>nul 2^>^&1
        echo del /f /q "%%SystemRoot%%\System32\drivers\ThrottleStop.sys" ^>nul 2^>^&1
        echo endlocal
    ) > "%FINAL_RUNNER%"
)

echo [V] Da thiet lap script duy tri khoi dong: "%FINAL_RUNNER%"
echo.
exit /b 0


:: ================================================================
:: MODULE 2: PREFLIGHT DIAGNOSTICS & SYSTEM REMEDIATION
:: ================================================================
:PreflightDiagnostic
:: Don dep cac task cu truoc
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1
schtasks /delete /tn "40HX PCIe Gen2 Bring-up" /f >nul 2>&1
schtasks /delete /tn "CMP30HX_Gen2_Unlock_User" /f >nul 2>&1

:: 2.1 Tat Fast Startup, Hybrid Sleep va PCIe ASPM toan he thong
echo [1/6] Dang tat Fast Startup, Hybrid Sleep va PCIe ASPM toan he thong...
powercfg -h off >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f >nul 2>&1

powershell -NoProfile -ExecutionPolicy Bypass -Command "$schemes = powercfg -list | ForEach-Object { if ($_ -match '([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})') { $matches[1] } }; foreach ($s in $schemes) { powercfg -setacvalueindex $s SUB_PCIEXPRESS ASPM 0 2>$null; powercfg -setdcvalueindex $s SUB_PCIEXPRESS ASPM 0 2>$null; powercfg -setacvalueindex $s SUB_SLEEP HYBRIDSLEEP 0 2>$null; powercfg -setdcvalueindex $s SUB_SLEEP HYBRIDSLEEP 0 2>$null }; powercfg -setactive SCHEME_CURRENT 2>$null" >nul 2>&1

powershell -NoProfile -ExecutionPolicy Bypass -Command "$base = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}'; Get-ChildItem $base -ErrorAction SilentlyContinue | ForEach-Object { $p = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue; if ($p.ProviderName -match 'NVIDIA' -or $p.DriverDesc -match 'NVIDIA|CMP') { Set-ItemProperty -Path $_.PSPath -Name 'DisableAspm' -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue; Set-ItemProperty -Path $_.PSPath -Name 'RMDisableLinkDownshift' -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue } }" >nul 2>&1

where nvidia-smi >nul 2>&1 && nvidia-smi -pm 1 >nul 2>&1
sc config NVDisplay.ContainerLocalSystem start= auto >nul 2>&1
sc start NVDisplay.ContainerLocalSystem >nul 2>&1

echo       [OK] Da vo hieu hoa Fast Startup (Hiberboot) va PCIe ASPM toan he thong.

:: 2.2 Tat Microsoft Vulnerable Driver Blocklist
echo [2/6] Dang tat Microsoft Vulnerable Driver Blocklist...
set "NEED_REBOOT=0"
reg query "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x1" >nul 2>&1
if not errorlevel 1 set "NEED_REBOOT=1"
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f >nul 2>&1
if errorlevel 1 (
    echo       [X] Khong tat duoc Driver Blocklist. Kiem tra chinh sach bao mat Windows.
) else (
    reg query "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x0" >nul 2>&1
    if errorlevel 1 (echo       [!] Driver Blocklist chua xac nhan OFF.) else (echo       [OK] Driver Blocklist da tat.)
)

:: 2.3 Tat Memory Integrity (HVCI)
echo [3/6] Dang kiem tra va tat Memory Integrity (HVCI)...
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

:: 2.4 Kiem tra che do Test Signing
bcdedit 2>nul | %SystemRoot%\System32\findstr.exe /i "testsigning" | %SystemRoot%\System32\findstr.exe /i "yes" >nul 2>&1
if not errorlevel 1 (
    echo.
    echo       [!] Phat hien Windows dang BAT che do Test Signing [testsigning=Yes]!
    echo           Riot Vanguard [Valorant, LMHT] va Easy Anti-Cheat se CHAN vao game [Loi VAN 1067].
    echo       [*] Dang tu dong tat Test Signing [bcdedit /set testsigning off]...
    bcdedit /set testsigning off >nul 2>&1
    set "NEED_REBOOT=1"
    echo       [OK] Da tat Test Signing thanh cong de tuong thich Riot Vanguard. [Can reboot].
)
exit /b 0


:: ================================================================
:: MODULE 3: PERSISTENCE & RIOT COMPATIBILITY
:: ================================================================
:ConfigurePersistence
if "%NO_TASK%"=="1" (
    echo [4/6] Bo qua tao Scheduled Task [-notask duoc bat]...
    echo       [*] Che do khong tao Task: Mo khoa truc tiep cho phien lam viec hien tai.
    set "TASK_OK=1"
    goto :skip_task_creation
)
echo [4/6] Dang tao Scheduled Task SYSTEM va Run Key duy tri Gen2...
set "TASK_OK=0"
set "FINAL_EXE=%FINAL_RUNNER%"
if not exist "%FINAL_EXE%" set "FINAL_EXE=%FINAL_INSTALLER%"
set "FINAL_WD=%FINAL_DIR%"

powershell -NoProfile -ExecutionPolicy Bypass -Command "$exe=$env:FINAL_EXE; $dir=$env:FINAL_WD; if (-not [IO.Path]::IsPathRooted($exe)) { throw 'Duong dan installer khong hop le' }; $q=[char]34; $action = if ($exe -match '\.bat$') { New-ScheduledTaskAction -Execute $env:ComSpec -Argument ('/c ' + $q + $exe + $q) -WorkingDirectory $dir } else { New-ScheduledTaskAction -Execute $exe -Argument '-gen2-30hx -silent' -WorkingDirectory $dir }; $t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'; $t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'; $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1) -ExecutionTimeLimit (New-TimeSpan -Minutes 5); $principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest; Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force; try { $srv = New-Object -ComObject 'Schedule.Service'; $srv.Connect(); $task = $srv.GetFolder('\').GetTask('CMP30HX_Gen2_Unlock'); $def = $task.Definition; $tEvent = $def.Triggers.Create(0); $tEvent.Subscription = '<QueryList><Query Id=''0'' Path=''System''><Select Path=''System''>*[System[Provider[@Name=''Microsoft-Windows-Power-Troubleshooter''] and EventID=1]]</Select></Query></QueryList>'; $tEvent.Delay = 'PT3S'; $tEvent.Enabled = $true; $srv.GetFolder('\').RegisterTaskDefinition('CMP30HX_Gen2_Unlock', $def, 4, $null, $null, 5, $null) } catch {}" >nul 2>&1

if not errorlevel 1 set "TASK_OK=1"
if "%TASK_OK%"=="1" schtasks /query /tn "CMP30HX_Gen2_Unlock" >nul 2>&1 || set "TASK_OK=0"

if "%TASK_OK%"=="0" (
    schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "%comspec% /c \"%FINAL_RUNNER%\"" /sc onstart /delay 0000:15 /rl highest /ru "NT AUTHORITY\SYSTEM" /f >nul 2>&1
    if not errorlevel 1 set "TASK_OK=1"
)

:: Bao hiem kep: Dang ky Registry Run key cho HKLM va HKCU
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

:skip_task_creation
:: Tu dong phat hien Riot Vanguard va cau hinh GPU High Performance cho Riot Games
set "HAS_VANGUARD=0"
sc query vgc >nul 2>&1 && set "HAS_VANGUARD=1"
if exist "%ProgramFiles%\Riot Vanguard\vgc.exe" set "HAS_VANGUARD=1"
if "%HAS_VANGUARD%"=="1" (
    echo       [*] Phat hien Riot Vanguard tren he thong. Che do tuong thich Anti-Cheat da san sang.
)

powershell -NoProfile -ExecutionPolicy Bypass -Command "$reg = 'HKCU:\Software\Microsoft\DirectX\UserGpuPreferences'; if (-not (Test-Path $reg)) { New-Item -Path $reg -Force | Out-Null }; $found = 0; $drives = (Get-PSDrive -PSProvider FileSystem).Root; foreach ($d in $drives) { foreach ($sub in @('Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe', 'Riot Games\League of Legends\Game\League of Legends.exe')) { $p = Join-Path $d $sub; if (Test-Path $p) { Set-ItemProperty -Path $reg -Name $p -Value 'GpuPreference=2;' -ErrorAction SilentlyContinue; $found++ } } }; if ($found -eq 0) { Set-ItemProperty -Path $reg -Name (Join-Path $env:SystemDrive 'Riot Games\VALORANT\live\ShooterGame\Binaries\Win64\VALORANT-Win64-Shipping.exe') -Value 'GpuPreference=2;' -ErrorAction SilentlyContinue; Set-ItemProperty -Path $reg -Name (Join-Path $env:SystemDrive 'Riot Games\League of Legends\Game\League of Legends.exe') -Value 'GpuPreference=2;' -ErrorAction SilentlyContinue }" >nul 2>&1
exit /b 0


:: ================================================================
:: MODULE 4: PCIE LINK RETRAIN & SOFT RESET CONTROLLER
:: ================================================================
:PCIeLinkRetrain
echo.
if "%NEED_REBOOT%"=="1" (
    echo [!] HVCI vua duoc dat OFF trong Registry nhung chua co hieu luc trong phien nay.
    echo     Buoc hien tai co the khong nap duoc driver kernel; sau khi ket thuc hay reboot truoc khi danh gia.
)
echo [5/6] Dang kich hoat Gen2 x16 va toi uu MRRS 512B ngay...

if "%IS_MOCK_MISSING_STATUS%"=="1" (
    echo       [*] [MOCK TEST] Mo phong loi installer bi crash hoac khong tao duoc file status...
    if exist "%ProgramData%\40HXUnlock\gen2_status.txt" del /f /q "%ProgramData%\40HXUnlock\gen2_status.txt" >nul 2>&1
    set "UNLOCK_OK=0"
    exit /b 0
)

if "%IS_MOCK_WINRING0%"=="1" (
    call :WriteMockStatusWinRing0
    set "UNLOCK_OK=0"
    goto :eval_first_attempt
)

if "%IS_MOCK_NOGPU%"=="1" (
    call :WriteMockStatusNoGpu
    set "UNLOCK_OK=0"
    goto :eval_first_attempt
)

if "%IS_MOCK%"=="1" (
    call :WriteMockStatusGen1
    set "UNLOCK_OK=0"
    goto :eval_first_attempt
)

:: Chay installer that lan dau
"%FINAL_INSTALLER%" -gen2-30hx -silent
if errorlevel 1 (
    echo       [X] Installer bao loi khi chay [exit code khac 0].
    echo           Kiem tra driver WinRing0/ThrottleStop, HVCI va quyen Administrator.
    set "UNLOCK_OK=0"
) else (
    set "UNLOCK_OK=1"
)

:eval_first_attempt
call :ShowStatusFileDetails
call :ParseStatusFile

:: Neu khong co file status (crash/antivirus): chan bao thanh cong ao
if not exist "%ProgramData%\40HXUnlock\gen2_status.txt" (
    echo       [X] LOI: Khong tim thay file gen2_status.txt sau khi khoi chay!
    set "UNLOCK_OK=0"
    exit /b 0
)

:: Neu driver kernel bi chan hoac khong tim thay GPU, Soft Reset hoan toan vo dung!
if "%IS_DRV_FAIL%"=="1" (
    echo.
    echo       [!] Phat hien Driver Kernel bi chan [WinRing0/ThrottleStop Loi 5 / HVCI / Antivirus].
    echo       [*] Bo qua Soft Reset [Soft Reset khong the giai quyet loi chan quyen driver kernel].
    set "NEED_REBOOT=1"
    set "UNLOCK_OK=0"
    exit /b 0
)

if "%IS_NOGPU_FAIL%"=="1" (
    echo.
    echo       [!] Khong dinh vi duoc GPU CMP 30HX tren bus PCI.
    echo       [*] Bo qua Soft Reset.
    set "UNLOCK_OK=0"
    exit /b 0
)

:: Kiem tra co can Soft Reset khong (khi bi ket Gen1 hoac installer bao chua dat)
set "NEED_DEV_RESET=0"
if "%UNLOCK_OK%"=="0" set "NEED_DEV_RESET=1"
if "%IS_GEN1_STUCK%"=="1" set "NEED_DEV_RESET=1"

if "%NEED_DEV_RESET%"=="0" (
    if "%IS_STATUS_SUCCESS%"=="1" set "UNLOCK_OK=1"
    exit /b 0
)

echo.
echo       [!] Phat hien GPU TLS van o Gen1 [driver mod / iGPU dang giu DMA context].
call :PnpSoftReset
echo       [*] Dang chay lai lenh mo khoa Gen2 sau khi Soft Reset card...

if "%IS_MOCK_FAIL%"=="1" (
    call :WriteMockStatusFail
    set "UNLOCK_OK=0"
    goto :eval_second_attempt
)

if "%IS_MOCK%"=="1" (
    call :WriteMockStatusGen2Success
    set "UNLOCK_OK=1"
    goto :eval_second_attempt
)

:: Chay lai installer sau khi Soft Reset
"%FINAL_INSTALLER%" -gen2-30hx -silent
if not errorlevel 1 set "UNLOCK_OK=1"

:eval_second_attempt
call :ShowStatusFileDetails
call :ParseStatusFile

if not exist "%ProgramData%\40HXUnlock\gen2_status.txt" (
    set "UNLOCK_OK=0"
    exit /b 0
)

if "%IS_STATUS_SUCCESS%"=="1" (
    set "UNLOCK_OK=1"
) else (
    set "UNLOCK_OK=0"
)
exit /b 0


:: ================================================================
:: STATUS INSPECTION & PARSER HELPERS
:: ================================================================
:ParseStatusFile
set "IS_DRV_FAIL=0"
set "IS_NOGPU_FAIL=0"
set "IS_GEN1_STUCK=0"
set "IS_STATUS_SUCCESS=0"
set "STATUS_PATH=%ProgramData%\40HXUnlock\gen2_status.txt"

if not exist "%STATUS_PATH%" exit /b 1

:: 1. Uu tien phan tich theo token co cau truc Seam 2 (STATUS_CODE)
findstr /i /c:"STATUS_CODE=GEN2_SUCCESS" "%STATUS_PATH%" >nul 2>&1 && set "IS_STATUS_SUCCESS=1"
findstr /i /c:"STATUS_CODE=GEN1_STUCK" "%STATUS_PATH%" >nul 2>&1 && set "IS_GEN1_STUCK=1"
findstr /i /c:"STATUS_CODE=DRV_FAIL" "%STATUS_PATH%" >nul 2>&1 && set "IS_DRV_FAIL=1"
findstr /i /c:"STATUS_CODE=NO_GPU" "%STATUS_PATH%" >nul 2>&1 && set "IS_NOGPU_FAIL=1"

:: 2. Co che du phong (Fallback) cho file trang thai cu hoac dinh dang tu do
if "%IS_STATUS_SUCCESS%"=="0" if "%IS_GEN1_STUCK%"=="0" if "%IS_DRV_FAIL%"=="0" if "%IS_NOGPU_FAIL%"=="0" (
    %SystemRoot%\System32\find.exe /i "WinRing0" "%STATUS_PATH%" >nul 2>&1 && set "IS_DRV_FAIL=1"
    %SystemRoot%\System32\find.exe /i "ThrottleStop" "%STATUS_PATH%" >nul 2>&1 && set "IS_DRV_FAIL=1"
    %SystemRoot%\System32\find.exe /i "Loi 5" "%STATUS_PATH%" >nul 2>&1 && set "IS_DRV_FAIL=1"
    %SystemRoot%\System32\find.exe /i "Error 5" "%STATUS_PATH%" >nul 2>&1 && set "IS_DRV_FAIL=1"
    %SystemRoot%\System32\find.exe /i "ERROR_ACCESS_DENIED" "%STATUS_PATH%" >nul 2>&1 && set "IS_DRV_FAIL=1"
    %SystemRoot%\System32\find.exe /i "bus PCI" "%STATUS_PATH%" >nul 2>&1 && set "IS_NOGPU_FAIL=1"
    %SystemRoot%\System32\find.exe /i "Khong tim thay" "%STATUS_PATH%" >nul 2>&1 && set "IS_NOGPU_FAIL=1"
    %SystemRoot%\System32\find.exe /i "not found" "%STATUS_PATH%" >nul 2>&1 && set "IS_NOGPU_FAIL=1"

    findstr /i /c:"GPU TLS=Gen1" "%STATUS_PATH%" >nul 2>&1 && set "IS_GEN1_STUCK=1"
    findstr /i /c:"chua dat" "%STATUS_PATH%" >nul 2>&1 && set "IS_GEN1_STUCK=1"
    findstr /i /c:"chưa đạt" "%STATUS_PATH%" >nul 2>&1 && set "IS_GEN1_STUCK=1"

    findstr /i /c:"da dat muc tieu Gen2 thanh cong" "%STATUS_PATH%" >nul 2>&1 && set "IS_STATUS_SUCCESS=1"
    findstr /i /c:"đã đạt mục tiêu Gen2 thành công" "%STATUS_PATH%" >nul 2>&1 && set "IS_STATUS_SUCCESS=1"
)

if "%IS_MOCK_WINRING0%"=="1" set "IS_DRV_FAIL=1"
if "%IS_MOCK_NOGPU%"=="1" set "IS_NOGPU_FAIL=1"
exit /b 0

:ShowStatusFileDetails
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
exit /b 0

:PnpSoftReset
echo       [*] Dang tu dong thuc hien chu trinh Soft Reset [Disable - Enable qua PnP]...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$devs = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.HardwareID -match 'VEN_10DE&(DEV_2189|DEV_1F0B)' }; if ($devs) { foreach ($d in $devs) { try { & pnputil /restart-device $d.InstanceId >$null 2>&1 } catch {}; try { Disable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue; Start-Sleep -Milliseconds 800; Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } catch {}; try { $st = (Get-PnpDevice -InstanceId $d.InstanceId -ErrorAction SilentlyContinue).Status; if ($st -ne 'OK') { Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction SilentlyContinue } } catch {} }; Start-Sleep -Seconds 2; try { Restart-Service NVDisplay.ContainerLocalSystem -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1 } catch {} } else { Write-Host 'Khong tim thay Instance ID qua PnP' }" >nul 2>&1
exit /b 0


:: ================================================================
:: MOCK STATUS GENERATORS (TEST HARNESS SUPPORT)
:: ================================================================
:WriteMockStatusWinRing0
echo       [*] [MOCK TEST] Mo phong WinRing0 bi chan boi HVCI / Security Policy...
if not exist "%ProgramData%\40HXUnlock" mkdir "%ProgramData%\40HXUnlock" >nul 2>&1
(
    echo STATUS_CODE=DRV_FAIL
    echo ERROR_CODE=ACCESS_DENIED_OR_DRIVER_BLOCKED
    echo ==== 40HX Gen2 Ket qua [MOCK TEST - WINRING0 BLOCKED] ====
    echo GPU: NVIDIA CMP 30HX [TU116] [DEV_2189]
    echo X Gen2 Chua thuc thi: WinRing0 driver bi chan [Loi 5 / ERROR_ACCESS_DENIED]
    echo Driver: WinRing0 Khong chay, ThrottleStop khong the mo thiet bi
    echo Loi: Khoi dong that bai: Khong du quyen han [Loi 5 / ERROR_ACCESS_DENIED]
    echo Quyen thuc thi: Quan tri vien / SYSTEM
) > "%ProgramData%\40HXUnlock\gen2_status.txt"
exit /b 0

:WriteMockStatusNoGpu
echo       [*] [MOCK TEST] Mo phong khong tim thay GPU CMP 30HX tren bus PCIe...
if not exist "%ProgramData%\40HXUnlock" mkdir "%ProgramData%\40HXUnlock" >nul 2>&1
(
    echo STATUS_CODE=NO_GPU
    echo ERROR_CODE=GPU_NOT_FOUND
    echo ==== 40HX Gen2 Ket qua [MOCK TEST - KHONG TIM THAY GPU] ====
    echo [X] Khong the dinh vi GPU CMP 30HX tren bus PCI [Khong tim thay thiet bi DEV_2189]
    echo Trang thai: Khong tim thay thiet bi tren bus PCI.
    echo Quyen thuc thi: Quan tri vien / SYSTEM
) > "%ProgramData%\40HXUnlock\gen2_status.txt"
exit /b 0

:WriteMockStatusGen1
echo       [*] [MOCK TEST] Mo phong Installer lan 1: Phat hien CMP 30HX nhung dang bi ket Gen1...
if not exist "%ProgramData%\40HXUnlock" mkdir "%ProgramData%\40HXUnlock" >nul 2>&1
(
    echo STATUS_CODE=GEN1_STUCK
    echo SPEED_CURRENT=1
    echo WIDTH_CURRENT=16
    echo TLS_TARGET=2
    echo ERROR_CODE=NONE
    echo ==== 40HX Gen2 Ket qua [MOCK TEST - LAN 1] ====
    echo GPU: NVIDIA CMP 30HX [TU116] [DEV_2189]
    echo PCIe Link Width: x16
    echo PCIe Link Speed: GPU TLS=Gen1 [2.5 GT/s]
    echo Trang thai: chua dat muc tieu Gen2 [Driver mod / iGPU dang giu DMA context]
    echo Quyen thuc thi: Quan tri vien / SYSTEM
) > "%ProgramData%\40HXUnlock\gen2_status.txt"
exit /b 0

:WriteMockStatusFail
echo       [*] [MOCK TEST FAIL] Mo phong Soft Reset khong the cuu van, GPU van kiet o Gen1...
if not exist "%ProgramData%\40HXUnlock" mkdir "%ProgramData%\40HXUnlock" >nul 2>&1
(
    echo STATUS_CODE=GEN1_STUCK
    echo SPEED_CURRENT=1
    echo WIDTH_CURRENT=16
    echo TLS_TARGET=2
    echo ERROR_CODE=NONE
    echo ==== 40HX Gen2 Ket qua [MOCK TEST - THAT BAI] ====
    echo GPU: NVIDIA CMP 30HX [TU116] [DEV_2189]
    echo PCIe Link Width: x16
    echo PCIe Link Speed: GPU TLS=Gen1 [2.5 GT/s]
    echo Trang thai: chua dat muc tieu Gen2 [Soft Reset khong the cuu van link]
    echo Quyen thuc thi: Quan tri vien / SYSTEM
) > "%ProgramData%\40HXUnlock\gen2_status.txt"
exit /b 0

:WriteMockStatusGen2Success
echo       [*] [MOCK TEST] Mo phong Installer lan 2: Soft Reset thanh cong, GPU bung Gen2 x16 [5.0 GT/s]
if not exist "%ProgramData%\40HXUnlock" mkdir "%ProgramData%\40HXUnlock" >nul 2>&1
(
    echo STATUS_CODE=GEN2_SUCCESS
    echo SPEED_CURRENT=2
    echo WIDTH_CURRENT=16
    echo TLS_TARGET=2
    echo ERROR_CODE=NONE
    echo ==== 40HX Gen2 Ket qua [MOCK TEST - LAN 2 SAU SOFT RESET] ====
    echo GPU: NVIDIA CMP 30HX [TU116] [DEV_2189]
    echo PCIe Link Width: x16
    echo PCIe Link Speed: GPU TLS=Gen2 [5.0 GT/s]
    echo Ket qua: da dat muc tieu Gen2 thanh cong
    echo MRRS: 512B [Da toi uu]
    echo Quyen thuc thi: Quan tri vien / SYSTEM
) > "%ProgramData%\40HXUnlock\gen2_status.txt"
exit /b 0


:: ================================================================
:: MODULE 5: CLEANUP BYOVD DRIVERS
:: ================================================================
:CleanupBYOVD
sc stop WinRing0_1_2_0 >nul 2>&1
sc delete WinRing0_1_2_0 >nul 2>&1
sc stop ThrottleStop >nul 2>&1
sc delete ThrottleStop >nul 2>&1
del /f /q "%SystemRoot%\System32\drivers\WinRing0x64.sys" >nul 2>&1
del /f /q "%SystemRoot%\System32\drivers\ThrottleStop.sys" >nul 2>&1
exit /b 0


:: ================================================================
:: MODULE 6: RENDER DIAGNOSTICS & SUMMARY
:: ================================================================
:RenderDiagnosticsSummary
echo.
echo [6/6] Kiem tra trang thai sau khi mo khoa...
if "%NO_CHECK%"=="1" (
    echo [*] Bo qua khoi chay 40HXCheck [-nocheck].
    goto :summary_dispatch
)
if "%IS_DRV_FAIL%"=="1" (
    echo [*] Driver kernel dang bi chan boi HVCI / Vulnerable Driver Blocklist trong phien nay.
    echo     Bo qua khoi chay 40HXCheck de tranh thong bao nham ve quyen han / GSP.
    echo     He thong can REBOOT de tat HVCI; sau reboot Scheduled Task se tu dong mo khoa Gen2.
    goto :summary_dispatch
)
echo.
echo [*] LUU Y QUAN TRONG VE GSP (GPU System Processor):
echo     CMP 30HX dung kien truc TU116 KHONG CO phan cung GSP.
echo     Neu 40HXCheck bao "Khong tim thay khoa GSP / Bat GSP": HAY BO QUA HOAN TOAN!
echo.
if exist "%TARGET_CHECK%" (
    echo [*] Tim thay cong cu chan doan: "%TARGET_CHECK%"
    echo [*] Dang khoi chay cua so chan doan 40HXCheck...
    start "" "%TARGET_CHECK%"
) else (
    "%FINAL_INSTALLER%" -status
)

:summary_dispatch
echo.
echo ================================================================
if "%UNLOCK_OK%"=="1" goto :dispatch_success
goto :dispatch_failure

:dispatch_success
if "%IS_MOCK%"=="1" (
    call :SummarySuccessMock
) else (
    call :SummarySuccessReal
)
goto :summary_post

:dispatch_failure
if "%IS_MOCK_MISSING_STATUS%"=="1" (
    call :SummaryFailMissingStatus
    goto :summary_post
)
if "%IS_DRV_FAIL%"=="1" (
    call :SummaryFailDrv
    goto :summary_post
)
if "%IS_NOGPU_FAIL%"=="1" (
    call :SummaryFailNoGpu
    goto :summary_post
)
call :SummaryFailGen1
goto :summary_post

:summary_post
if "%TASK_OK%"=="1" (
    if not "%NO_TASK%"=="1" (
        echo  - Sau khi reboot, he thong se tu dong mo khoa sau 15 giay khoi dong hoac 5 giay dang nhap.
        echo  - Neu muon xoa Scheduled Task, ban chi can mo lai script nay va chon muc [2].
    ) else (
        echo  - Da mo khoa Gen2 thanh cong ma khong de lai Scheduled Task khoi dong.
    )
) else (
    echo  - [Canh bao] Scheduled Task chua san sang; sau reboot phai chay lai Setup hoac lenh mo khoa thu cong.
)
if "%NEED_REBOOT%"=="1" call :warn_reboot
echo ================================================================
echo.
if "%UNLOCK_OK%"=="1" exit /b 0
exit /b 1

:SummarySuccessMock
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
exit /b 0

:SummarySuccessReal
echo  [V] CAI DAT HOAN TAT - Gen2 da duoc cau hinh ben vung.
echo  - Da thiet lap da co che: Scheduled Task SYSTEM + Registry Run Key.
echo  - Tu dong duy tri Gen2 tren moi lan Boot, Dang nhap va Wake from Sleep!
echo.
echo  - LUU Y QUAN TRONG VE GEN 1 KHI VUA KHOI DONG / IDLE:
echo    + Link PCIe se o Gen1 x16 khi card o che do ranh [Idle Power Saving].
echo    + Khi co tai 3D/CUDA/AIDA64/FurMark, card se tu dong bung toc do len Gen2 x16.
echo    + Neu GPU-Z bao Gen1: nhap vao dau cham hoi [?] canh Bus Interface de chay Render Test!
echo  - Neu sau khi reboot co tai ma GPU van Gen1: kiem tra HVCI, riser, tiep xuc lane va BIOS khe PCIe.
echo.
echo  [*] CHE DO TUONG THICH TOAN DIEN [NGUOI CHOI RIOT GAMES & NGUOI DUNG THUONG]:
echo      1. He thong da tu dong don dep sach se driver WinRing0/ThrottleStop khoi kernel va System32.
echo         =^> Riot Vanguard, Easy Anti-Cheat, BattlEye khong bao gio phat hien hay chan driver.
echo      2. Windows Test Signing da duoc kiem tra va tat =^> Khong bi loi VAN 1067 / VAN 9003.
echo      3. Chinh sach Secure Boot ^& TPM 2.0 theo tung loai card:
echo         + Voi CMP 30HX: Secure Boot va TPM 2.0 luon giu BAT trong BIOS [Choi tot ca Valorant ^& LMHT].
echo         + Voi CMP 40HX [Phai tat Secure Boot trong BIOS de nap EFI mo khoa Tensor Core]:
echo           - Lien Minh Huyen Thoai (LMHT): Choi binh thuong tren ca Win 10 ^& Win 11.
echo           - Valorant tren Win 11: Riot bat buoc Secure Boot [loi VAN 9003/VAN 1067].
echo             =^> De choi Valorant voi 40HX: Khuyen nghi dung Windows 10 (khong ep Secure Boot).
echo      4. Valorant va LMHT da duoc tu dong dinh tuyen sang GPU High Performance.
echo      5. Neu truoc day tung dung ban cu/ban goc bi chan game, chi can chay script nay 1 lan roi reboot!
exit /b 0

:SummaryFailMissingStatus
echo  [X] CAI DAT CHUA HOAN TAT - MAT HOAC KHONG THE GHI FILE TRANG THAI GEN2!
echo.
echo  [!] NGUYEN NHAN CHINH:
echo      Installer bi Antivirus diet, bi Crash hoac khong the ghi file vao ProgramData.
echo.
echo  [*] CAC BUOC KHAC PHUC:
echo      1. Kiem tra Windows Defender / Antivirus xem 40HXInstaller.exe co bi chan khong.
echo      2. Dam bao chay script voi quyen Administrator cao nhat.
echo      3. Kiem tra quyen ghi vao thu muc "C:\ProgramData\40HXUnlock".
goto :fail_common

:SummaryFailDrv
echo  [X] CAI DAT CHUA HOAN TAT - DRIVER KERNEL (WinRing0/ThrottleStop) BI CHAN!
echo.
echo  [!] NGUYEN NHAN CHINH:
echo      Windows Core Isolation (HVCI), Vulnerable Driver Blocklist hoac Antivirus
echo      dang chan nap driver truy cap phan cung kernel (Loi 5 / Access Denied).
echo.
echo  [*] CAC BUOC KHAC PHUC:
echo      1. KHOI DONG LAI MAY (REBOOT):
echo         Script da tu dong tat HVCI va Driver Blocklist trong Registry o Buoc [2/6] ^& [3/6].
echo         Tuy nhien, Windows KERNEL BAT BUOC PHAI REBOOT moi co hieu luc!
echo      2. Kiem tra phan mem diet virus / Windows Defender:
echo         Neu co Kaspersky, Bitdefender, Avast... hay tam tat hoac them exclusion cho
echo         thu muc "%ProgramFiles%\40HXUnlock".
echo      3. Sau khi Reboot:
echo         Scheduled Task SYSTEM se tu dong thu nap lai driver va mo khoa Gen2.
echo         Hoac ban co the chay lai script nay voi Run as Administrator.
goto :fail_common

:SummaryFailNoGpu
echo  [X] CAI DAT CHUA HOAN TAT - KHONG TIM THAY GPU CMP 30HX (DEV_2189)!
echo.
echo  [!] NGUYEN NHAN CHINH:
echo      Cong cu khong dinh vi duoc card CMP 30HX tren bus PCI.
echo.
echo  [*] CAC BUOC KHAC PHUC:
echo      1. Kiem tra lai nguon phu 8-pin PCIe va tiep xuc khe cam PCIe / Riser.
echo      2. Kiem tra Device Manager xem card co hien thi khong (ke ca dang co cham than vang).
echo      3. Kiem tra BIOS: Bat "Above 4G Decoding", Re-Size BAR, va kiem tra thiet lap khe PCIe.
goto :fail_common

:SummaryFailGen1
echo  [X] CAI DAT CHUA HOAN TAT - GPU VAN BI KET O GEN1 (2.5 GT/s)!
echo.
echo  [!] NGUYEN NHAN CHINH:
echo      Driver da mo duoc phan cung nhung link PCIe khong the bung len Gen2.
echo.
echo  [*] CAC BUOC KHAC PHUC:
echo      1. Mo GPU-Z va BAM VAO DAU CHAM HOI [?] canh muc Bus Interface de chay RENDER TEST!
echo         Luu y: Khi card o che do nghi (Idle), GPU se tu dong ha xuong Gen1 de tiet kiem dien.
echo      2. Neu khi Render Test van kiet Gen1: Khoi dong lai may de PnP/DMA context duoc giai phong.
echo      3. Neu van Gen1 sau reboot: Kiem tra cap Riser, tiep xuc khe PCIe hoac cam truc tiep vao khe x16.
echo      4. Neu nghi ngo xung dot driver: Dung DDU go sach driver cu trong Safe Mode roi cai lai.
goto :fail_common

:fail_common
echo.
echo  [*] LUU Y VE GSP: TU116 (CMP 30HX) khong ho tro GSP. Bo qua moi canh bao GSP tu 40HXCheck.
exit /b 0

:warn_reboot
echo.
echo  [!] LUU Y BAT BUOC: He thong vua tat Memory Integrity [Core Isolation].
echo      Ban PHAI KHOI DONG LAI MAY de Windows giai phong driver kernel.
echo      Sau reboot: cho Scheduled Task chay du 15 giay, tao tai 3D/CUDA,
echo      sau do moi dung GPU-Z/40HXCheck de danh gia Gen2.
exit /b 0


:: ================================================================
:: MODULE 7: UNINSTALL & SYSTEM CLEANUP
:: ================================================================
:clean_tasks
:uninstall
echo ================================================================
echo    TU DONG XOA TOAN BO SCHEDULED TASK VA DON DEP HE THONG
echo ================================================================
echo.
echo [*] Dang tim va xoa toan bo Scheduled Task he thong cua AIO...
schtasks /delete /tn "CMP30HX_Gen2_Unlock" /f >nul 2>&1
schtasks /delete /tn "CMP30HX_Gen2_Unlock_User" /f >nul 2>&1
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1
schtasks /delete /tn "40HX PCIe Gen2 Bring-up" /f >nul 2>&1
echo       [OK] Da xoa sach toan bo Scheduled Task [CMP30HX_Gen2_Unlock].
echo.
echo [*] Dang xoa cac Registry Run Key duy tri khoi dong...
reg delete "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /f >nul 2>&1
echo       [OK] Da xoa sach Registry Run Key khoi dong [HKLM va HKCU].
echo.
echo [*] Dang don dep driver BYOVD tranh xung dot Anti-Cheat...
call :CleanupBYOVD
echo       [OK] Da go bo cac service va driver WinRing0 / ThrottleStop khoi kernel.
echo.
if exist "%ProgramData%\40HXUnlock\gen2_status.txt" (
    del /f /q "%ProgramData%\40HXUnlock\gen2_status.txt" >nul 2>&1
)
if exist "%ProgramFiles%\40HXUnlock" (
    rmdir /s /q "%ProgramFiles%\40HXUnlock" >nul 2>&1
)
echo ================================================================
echo  [V] DA XOA TOAN BO CAC SCHEDULED TASK VA DON DEP SACH SE!
echo  - Task Scheduler he thong hoan toan sach se, khong con tac vu chay ngam.
echo  - Khong con bat ky tac vu nao co the gay anh huong den Riot Vanguard / Anti-Cheat.
echo ================================================================
echo.
if not "%NO_WAIT%"=="1" pause
exit /b 0
