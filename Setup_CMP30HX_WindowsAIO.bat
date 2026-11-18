@echo off
setlocal DisableDelayedExpansion
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
set "NO_WAIT=0"
set "DO_UNINSTALL=0"
set "DO_CLEAN_TASKS=0"
set "NO_TASK=0"
set "DO_PREFLIGHT_ONLY=0"
set "DO_FIX_NVCPL=0"
set "IS_ADMIN=0"
set "DO_OPTIMIZE=0"
set "MENU_ARGS="
set "AIO_INSTALL_DIR=%ProgramFiles%\40HXUnlock"
set "FINAL_INSTALLER=%AIO_INSTALL_DIR%\UnlockAIO.exe"
set "FINAL_RUNNER=%AIO_INSTALL_DIR%\RunUnlock.bat"
set "TASK_SCRIPT=%~dp0scripts\Register-UnlockTask.ps1"
set "AIO_STATUS_DIR=%ProgramData%\40HXUnlock"
set "AIO_STATUS=%AIO_STATUS_DIR%\cmp30hx_batch_status.txt"
set "HAS_CLI_FLAG=0"
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
    if /i "%%~a"=="-fix-nvcpl" set "DO_FIX_NVCPL=1"
    if /i "%%~a"=="/fix-nvcpl" set "DO_FIX_NVCPL=1"
    if /i "%%~a"=="-fixnvcpl" set "DO_FIX_NVCPL=1"
    if /i "%%~a"=="/fixnvcpl" set "DO_FIX_NVCPL=1"
)

for %%a in (%*) do (
    set "KNOWN_FLAG="
    for %%f in (noadmin install test mock mock-fail mock-winring0 mock-nogpu mock-missing-status nocheck nowait uninstall u cleantasks deltasks clean notask preflight optimize fix-nvcpl fixnvcpl) do (
        if /i "%%~a"=="-%%f" set "KNOWN_FLAG=1"
        if /i "%%~a"=="/%%f" set "KNOWN_FLAG=1"
    )
    if not defined KNOWN_FLAG (
        echo [X] Tham so khong hop le. Dung -install, -preflight, -uninstall hoac -cleantasks.
        exit /b 2
    )
    if /i "%%~a"=="-optimize" set "DO_OPTIMIZE=1"
    if /i not "%%~a"=="-noadmin" if /i not "%%~a"=="/noadmin" set "HAS_CLI_FLAG=1"
)
if "%IS_MOCK_FAIL%"=="1" set "IS_MOCK=1"
if "%IS_MOCK_WINRING0%"=="1" set "IS_MOCK=1"
if "%IS_MOCK_NOGPU%"=="1" set "IS_MOCK=1"
if "%IS_MOCK_MISSING_STATUS%"=="1" set "IS_MOCK=1"

:: ================================================================
:: 2. KIEM TRA QUYEN ADMINISTRATOR (UAC DA TANG PHONG THU)
:: ================================================================
if "%IS_MOCK%"=="1" goto :start_aio
if "%DO_PREFLIGHT_ONLY%"=="1" goto :preflight_only
if "%HAS_CLI_FLAG%"=="0" goto :aio_menu
goto :require_admin

:require_admin
if "%IS_ADMIN%"=="0" fltmc >nul 2>&1 && set "IS_ADMIN=1"

if "%IS_ADMIN%"=="0" (
    echo [!] Dang yeu cau quyen Administrator [UAC]...
    set "CURRENT_SCRIPT=%~f0"
    set "CURRENT_DIR=%~dp0"
    set "SCRIPT_ARGS=%* %MENU_ARGS%"
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$q=[char]34; $p=Start-Process -FilePath $env:ComSpec -ArgumentList ('/d /c ' + $q + $q + $env:CURRENT_SCRIPT + $q + ' ' + $env:SCRIPT_ARGS + $q) -WorkingDirectory $env:CURRENT_DIR -Verb RunAs -PassThru -Wait -ErrorAction Stop; exit $p.ExitCode"
    exit /b
)

cd /d "%~dp0" || exit /b 1

if "%DO_CLEAN_TASKS%"=="1" goto :clean_tasks
if "%DO_UNINSTALL%"=="1" goto :uninstall
if "%DO_FIX_NVCPL%"=="1" goto :FixNvidiaControlPanel
if "%DO_OPTIMIZE%"=="1" goto :OptimizeRiotAndUnity
goto :start_aio

:: ================================================================
:: 3. MENU TUONG TAC TRUYEN THONG CHO NGUOI DUNG
:: ================================================================
:aio_menu
set "DO_OPTIMIZE=0"
set "DO_UNINSTALL=0"
set "DO_FIX_NVCPL=0"
set "MENU_ARGS="
cls
echo ================================================================
echo    CMP 30HX [TU116] - CONG CU MO KHOA PCIE GEN2
echo ================================================================
echo.
echo   [1] Cai dat, mo khoa Gen2 va dang ky tu dong khoi dong
echo   [2] Go bo bo cai Gen2 AIO
echo.
echo   [3] Khoi phuc va Sua loi mat NVIDIA Control Panel
echo       - Dat lai service NVDisplay.ContainerLocalSystem ve tu dong [Auto] va khoi dong
echo       - Phuc hoi dang ky Desktop Context Menu handler
echo.
echo   [4] Toi uu Riot Games & Game Unity (LoL DX11 Borderless, Valorant, Unity D3D12)
echo       - Chay cong cu UnlockRiotGame va cau hinh D3D12 cho Unity engine
echo.
echo   [5] Thoat
echo.
echo ================================================================
%SystemRoot%\System32\choice.exe /c 12345 /m "Nhap lua chon cua ban [1-5]: "
if errorlevel 5 exit /b 0
if errorlevel 4 (
    set "DO_OPTIMIZE=1"
    set "MENU_ARGS=-optimize"
    goto :require_admin
)
if errorlevel 3 (
    set "DO_FIX_NVCPL=1"
    set "MENU_ARGS=-fix-nvcpl"
    goto :require_admin
)
if errorlevel 2 (
    set "DO_UNINSTALL=1"
    set "MENU_ARGS=-uninstall"
    goto :require_admin
)
if errorlevel 1 (
    set "MENU_ARGS=-install"
    goto :require_admin
)
exit /b 1

:OptimizeRiotAndUnity
cls
echo ================================================================
echo    TOI UU RIOT GAMES [VALORANT, LMHT DX11] VA GAME UNITY
echo ================================================================
echo.
echo [*] Dang thuc thi cau hinh GPU High Performance va Borderless Windowed...
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\Set-GpuPreference.ps1"
echo.
echo [*] Dang quet va cau hinh Direct3D 12 (-force-d3d12) cho Game Unity...
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\Set-UnityD3D12.ps1"
echo.
echo [*] Khoi chay cong cu UnlockRiotGame...
if exist "%~dp0windows-v3.0\release\UnlockRiotGame.exe" (
    start "" "%~dp0windows-v3.0\release\UnlockRiotGame.exe"
) else if exist "%ProgramFiles%\40HXUnlock\UnlockRiotGame.exe" (
    start "" "%ProgramFiles%\40HXUnlock\UnlockRiotGame.exe"
) else if exist "%~dp0release\UnlockRiotGame.exe" (
    start "" "%~dp0release\UnlockRiotGame.exe"
) else (
    echo [!] Khong tim thay UnlockRiotGame.exe.
)
echo.
echo [V] Hoan tat! Nhan phim bat ky de quay lai menu...
pause >nul
goto :aio_menu

:: ================================================================
:: 4. DIEU PHOI TIEN TRINH CHINH (MAIN ORCHESTRATION PIPELINE)
:: ================================================================
:start_aio
echo CMP 30HX [TU116] PCIe Gen2
if "%IS_MOCK%"=="1" echo [*] Che do mo phong
set "TASK_OK=0"
set "FINAL_EXIT=1"
if "%IS_MOCK%"=="1" goto :run_mock
call :PreflightDiagnostic
if errorlevel 1 goto :finish_install
call :FindInstallerFiles
if errorlevel 1 goto :finish_install
call :CheckSharedDrivers
if errorlevel 1 goto :finish_install
if "%NO_TASK%"=="0" (
    call :RemovePersistence
    if errorlevel 1 goto :finish_install
)
call :DeployInstallerFiles
if errorlevel 1 goto :finish_install
goto :run_unlock

:run_mock
set "AIO_STATUS_DIR=%TEMP%\CMP30HX-mock-%RANDOM%-%RANDOM%"
mkdir "%AIO_STATUS_DIR%" 2>nul || goto :finish_install
set "AIO_STATUS=%AIO_STATUS_DIR%\cmp30hx_batch_status.txt"

:run_unlock
call :PCIeLinkRetrain
if "%UNLOCK_OK%"=="1" if "%IS_MOCK%"=="0" if "%NO_TASK%"=="0" (
    call :ConfigurePersistence
    if not errorlevel 1 set "TASK_OK=1"
)
call :RenderDiagnosticsSummary
set "FINAL_EXIT=%ERRORLEVEL%"
if "%IS_MOCK%"=="1" if exist "%AIO_STATUS%" del /q "%AIO_STATUS%" >nul 2>&1
if "%IS_MOCK%"=="1" rmdir "%AIO_STATUS_DIR%" >nul 2>&1

:finish_install
if not "%NO_WAIT%"=="1" pause
exit /b %FINAL_EXIT%

:FindInstallerFiles
set "AIO_SOURCE=%~dp0windows-v3.0\release"
if not exist "%AIO_SOURCE%\UnlockAIO.exe" set "AIO_SOURCE=%~dp0release"
if not exist "%AIO_SOURCE%\UnlockAIO.exe" set "AIO_SOURCE=%~dp0."
if not exist "%AIO_SOURCE%\UnlockAIO.exe" goto :deploy_missing
if not exist "%AIO_SOURCE%\RunUnlock.bat" goto :deploy_missing
exit /b 0

:DeployInstallerFiles
if not exist "%AIO_INSTALL_DIR%\" mkdir "%AIO_INSTALL_DIR%" >nul 2>&1 || goto :deploy_failed
if exist "%FINAL_INSTALLER%\" goto :deploy_failed
if exist "%FINAL_RUNNER%\" goto :deploy_failed
copy /y "%AIO_SOURCE%\UnlockAIO.exe" "%FINAL_INSTALLER%" >nul 2>&1 || goto :deploy_failed
copy /y "%AIO_SOURCE%\RunUnlock.bat" "%FINAL_RUNNER%" >nul 2>&1 || goto :deploy_failed
echo [V] Bo cai da duoc sao chep vao "%AIO_INSTALL_DIR%".
exit /b 0

:deploy_missing
echo [X] Thieu UnlockAIO.exe hoac RunUnlock.bat. Hay giai nen day du goi phat hanh.
exit /b 1

:deploy_failed
echo [X] Khong the cap nhat tep trong "%AIO_INSTALL_DIR%".
exit /b 1


:: ================================================================
:: MODULE 2: PREFLIGHT DIAGNOSTICS AND STARTUP TASK
:: ================================================================
:preflight_only
call :PreflightDiagnostic
set "FINAL_EXIT=%ERRORLEVEL%"
goto :finish_install

:PreflightDiagnostic
echo [*] Dang kiem tra GPU CMP 30HX [10DE:2189] dang ket noi...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $g = @(Get-PnpDevice -PresentOnly | Where-Object { $_.InstanceId -like 'PCI\VEN_10DE&DEV_2189&*' }); if ($g.Count -ne 1) { Write-Host '[X] Can dung mot card CMP 30HX [10DE:2189] dang ket noi.'; exit 1 }; $g | Format-Table Status,FriendlyName,InstanceId -AutoSize"
exit /b %ERRORLEVEL%

:ConfigurePersistence
powershell -NoProfile -ExecutionPolicy Bypass -File "%TASK_SCRIPT%" -ExePath "%FINAL_RUNNER%" -WorkingDirectory "%AIO_INSTALL_DIR%"
exit /b %ERRORLEVEL%

:RemovePersistence
powershell -NoProfile -ExecutionPolicy Bypass -File "%TASK_SCRIPT%" -ExePath "%FINAL_RUNNER%" -Remove
exit /b %ERRORLEVEL%


:: ================================================================
:: MODULE 4: PCIE LINK RETRAIN
:: ================================================================
:PCIeLinkRetrain
set "UNLOCK_OK=0"
if "%IS_MOCK%"=="1" (
    if "%IS_MOCK_MISSING_STATUS%"=="1" goto :evaluate_unlock
    if "%IS_MOCK_WINRING0%"=="1" (
        >"%AIO_STATUS%" echo STATUS_CODE=DRV_FAIL
    ) else if "%IS_MOCK_NOGPU%"=="1" (
        >"%AIO_STATUS%" echo STATUS_CODE=NO_GPU
    ) else if "%IS_MOCK_FAIL%"=="1" (
        >"%AIO_STATUS%" echo STATUS_CODE=GEN1_STUCK
    ) else (
        >"%AIO_STATUS%" echo STATUS_CODE=GEN2_SUCCESS
        set "UNLOCK_OK=1"
    )
    goto :evaluate_unlock
)

echo [*] Dang mo khoa PCIe Gen2...
if exist "%AIO_STATUS%\" (
    echo [X] Duong dan trang thai dang la thu muc.
    exit /b 1
)
del /f /q "%AIO_STATUS%" >nul 2>&1
if exist "%AIO_STATUS%" (
    echo [X] Khong the xoa file trang thai cu.
    goto :evaluate_unlock
)
start "" /wait "%FINAL_INSTALLER%" -gen2-30hx -silent -batch
if not errorlevel 1 set "UNLOCK_OK=1"

:evaluate_unlock
call :ShowStatusFileDetails
call :ParseStatusFile
if not "%IS_STATUS_SUCCESS%"=="1" set "UNLOCK_OK=0"
exit /b 0

:ParseStatusFile
set "IS_DRV_FAIL=0"
set "IS_NOGPU_FAIL=0"
set "IS_GEN1_STUCK=0"
set "IS_STATUS_SUCCESS=0"
if not exist "%AIO_STATUS%" exit /b 1
%SystemRoot%\System32\findstr.exe /i /x /c:"STATUS_CODE=GEN2_SUCCESS" "%AIO_STATUS%" >nul 2>&1 && set "IS_STATUS_SUCCESS=1"
%SystemRoot%\System32\findstr.exe /i /x /c:"STATUS_CODE=GEN1_STUCK" "%AIO_STATUS%" >nul 2>&1 && set "IS_GEN1_STUCK=1"
%SystemRoot%\System32\findstr.exe /i /x /c:"STATUS_CODE=DRV_FAIL" "%AIO_STATUS%" >nul 2>&1 && set "IS_DRV_FAIL=1"
%SystemRoot%\System32\findstr.exe /i /x /c:"STATUS_CODE=NO_GPU" "%AIO_STATUS%" >nul 2>&1 && set "IS_NOGPU_FAIL=1"
exit /b 0

:ShowStatusFileDetails
if exist "%AIO_STATUS%" type "%AIO_STATUS%"
echo.
exit /b 0

:CheckSharedDrivers
sc query WinRing0_1_2_0 >nul 2>&1 && goto :shared_driver_found
sc query ThrottleStop >nul 2>&1 && goto :shared_driver_found
if exist "%SystemRoot%\System32\drivers\WinRing0x64.sys" goto :shared_driver_found
if exist "%SystemRoot%\System32\drivers\ThrottleStop.sys" goto :shared_driver_found
exit /b 0
:shared_driver_found
echo [X] Driver WinRing0/ThrottleStop da ton tai; dung de tranh anh huong phan mem khac.
exit /b 1

:: ================================================================
:: MODULE 5: INSTALL RESULT
:: ================================================================
:RenderDiagnosticsSummary
echo.
if "%UNLOCK_OK%"=="1" goto :dispatch_success
if not exist "%AIO_STATUS%" (
    call :SummaryFailMissingStatus
    exit /b 1
)
if "%IS_DRV_FAIL%"=="1" (
    call :SummaryFailDrv
    exit /b 1
)
if "%IS_NOGPU_FAIL%"=="1" (
    call :SummaryFailNoGpu
    exit /b 1
)
if "%IS_GEN1_STUCK%"=="1" (
    call :SummaryFailGen1
) else (
    echo [X] Bo cai khong xac nhan thanh cong. Xem chi tiet trang thai ben tren.
)
exit /b 1

:dispatch_success
if "%IS_MOCK%"=="1" (
    echo [V] Kiem thu mo phong thanh cong.
    exit /b 0
)
if "%NO_TASK%"=="1" (
    echo [V] Mo khoa thanh cong cho phien hien tai. Khong dang ky tac vu moi.
    exit /b 0
)
if "%TASK_OK%"=="0" (
    echo [X] Mo khoa phien hien tai thanh cong, nhung khong dang ky duoc tac vu khoi dong.
    exit /b 1
)
echo [V] Mo khoa thanh cong. Tac vu SYSTEM se chay sau khoi dong 45 giay hoac dang nhap 10 giay.
echo     De go bo, chay lai script nay va chon muc [2].
exit /b 0

:SummaryFailMissingStatus
echo [X] Khong co tep trang thai tu UnlockAIO.exe. Kiem tra quyen Admin va Windows Security.
exit /b 1

:SummaryFailDrv
echo [X] Driver phan cung hoac thao tac PCIe gap loi. Xem chi tiet trang thai ben tren.
exit /b 1

:SummaryFailNoGpu
echo [X] Khong tim thay CMP 30HX [10DE:2189] tren bus PCIe.
exit /b 1

:SummaryFailGen1
echo [X] Chua dat muc tieu Gen2. Kiem tra toc do PCIe duoi tai, khe cam va cap riser.
exit /b 1


:: ================================================================
:: MODULE 7: UNINSTALL & SYSTEM CLEANUP
:: ================================================================
:clean_tasks
call :RemovePersistence
set "FINAL_EXIT=%ERRORLEVEL%"
if "%FINAL_EXIT%"=="0" echo [V] Da don dep tac vu khoi dong cua bo cai batch.
goto :finish_install

:uninstall
call :RemovePersistence
set "FINAL_EXIT=1"
if errorlevel 1 goto :finish_install
if exist "%AIO_STATUS%\" goto :uninstall_failed
if exist "%AIO_STATUS%" del /f /q "%AIO_STATUS%"
if exist "%AIO_STATUS%" goto :uninstall_failed
for %%f in (UnlockAIO.exe RunUnlock.bat 40HXInstaller.exe 40HXCheck.exe UnlockRiotGame.exe) do (
    if exist "%AIO_INSTALL_DIR%\%%f\" goto :uninstall_failed
    if exist "%AIO_INSTALL_DIR%\%%f" del /f /q "%AIO_INSTALL_DIR%\%%f"
    if exist "%AIO_INSTALL_DIR%\%%f" goto :uninstall_failed
)
for %%f in (drivers\WinRing0x64.sys drivers\ThrottleStop.sys scripts\Disable-ASPM.ps1 scripts\Disable-NvidiaASPM.ps1 scripts\Get-SysInfo.ps1 scripts\Register-ReBarTask.ps1 scripts\Register-UnlockTask.ps1 scripts\Set-GpuPreference.ps1 scripts\Set-UnityD3D12.ps1 scripts\Test-AIO-Integration.ps1 scripts\Test-AIO-Status.ps1) do (
    if exist "%AIO_INSTALL_DIR%\%%f\" goto :uninstall_failed
    if exist "%AIO_INSTALL_DIR%\%%f" del /f /q "%AIO_INSTALL_DIR%\%%f"
    if exist "%AIO_INSTALL_DIR%\%%f" goto :uninstall_failed
)
for %%d in (drivers scripts) do if exist "%AIO_INSTALL_DIR%\%%d\" rmdir "%AIO_INSTALL_DIR%\%%d" 2>nul
if exist "%AIO_INSTALL_DIR%\" rmdir "%AIO_INSTALL_DIR%" 2>nul
echo [V] Da go bo bo cai batch. Khoi dong lai Windows de xoa trang thai mo khoa hien tai.
set "FINAL_EXIT=0"
goto :finish_install

:uninstall_failed
echo [X] Chua the go bo het tep. Dong cong cu Unlock va chay lai lenh -uninstall.
goto :finish_install


:: ================================================================
:: MODULE 8: ENSURE & FIX NVIDIA CONTROL PANEL HEALTH
:: ================================================================
:FixNvidiaControlPanel
echo ================================================================
echo    KHOI PHUC VA SUA LOI MAT NVIDIA CONTROL PANEL [DCH ^& SHELL]
echo ================================================================
echo.
echo [*] Dang kiem tra va khoi phuc service NVDisplay.ContainerLocalSystem...
sc config NVDisplay.ContainerLocalSystem start= auto >nul 2>&1 || goto :nvcpl_failed
sc query NVDisplay.ContainerLocalSystem 2>nul | %SystemRoot%\System32\findstr.exe /i "RUNNING" >nul 2>&1
if errorlevel 1 (
    net start NVDisplay.ContainerLocalSystem >nul 2>&1 || goto :nvcpl_failed
)
echo       [OK] Service NVDisplay.ContainerLocalSystem dang chay.
echo.
echo [*] Dang dang ky lai Desktop Context Menu Handler cho NVIDIA Control Panel...
reg add "HKCR\Directory\Background\shellex\ContextMenuHandlers\NvCplDesktopContext" /ve /t REG_SZ /d "{3D1975AF-48C6-4f8e-A182-BE0E08FA86A9}" /f >nul 2>&1 || goto :nvcpl_failed
echo       [OK] Da khoi phuc khoa Registry Context Menu [NvCplDesktopContext].
echo.
echo ================================================================
echo  [V] HOAN TAT KHOI PHUC NVIDIA CONTROL PANEL!
echo  - Ban co the nhap chuot phai tren Desktop de kiem tra lai menu NVIDIA Control Panel.
echo ================================================================
echo.
if not "%NO_WAIT%"=="1" pause
exit /b 0

:nvcpl_failed
echo [X] Khong the khoi phuc NVIDIA Control Panel. Kiem tra driver NVIDIA va quyen Admin.
set "FINAL_EXIT=1"
goto :finish_install
