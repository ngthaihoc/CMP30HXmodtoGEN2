@echo off
setlocal EnableDelayedExpansion
chcp 65001 >nul
title CMP 30HX Gen2 Automated Mock Test Suite

:: Kiem tra co truyen tham so bo qua UAC khong (-noadmin)
set "NO_ADMIN=0"
for %%a in (%*) do (
    if /i "%%~a"=="-noadmin" set "NO_ADMIN=1"
    if /i "%%~a"=="/noadmin" set "NO_ADMIN=1"
)

:: Kiem tra quyen Administrator
set "IS_ELEVATED=0"
if "%NO_ADMIN%"=="1" (
    set "IS_ELEVATED=1"
) else (
    fltmc >nul 2>&1 && set "IS_ELEVATED=1"
    if "!IS_ELEVATED!"=="0" (
        fsutil dirty query %systemdrive% >nul 2>&1 && set "IS_ELEVATED=1"
    )
)

if "!IS_ELEVATED!"=="0" (
    echo [!] Dang yeu cau quyen Administrator [UAC]...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~f0' -ArgumentList '%*' -WorkingDirectory '%~dp0' -Verb RunAs" >nul 2>&1
    if errorlevel 1 (
        echo [X] LOI: Khong the nang quyen Administrator qua UAC.
        exit /b 1
    )
    exit /b
)

cd /d "%~dp0"

set "TARGET_BAT=%~dp0Setup_CMP30HX_WindowsAIO.bat"
if not exist "%TARGET_BAT%" (
    echo [X] Khong tim thay: "%TARGET_BAT%"
    exit /b 1
)

set "REBAR_BAT=%~dp0Setup_ReBar_CMP30HX.bat"
if not exist "%REBAR_BAT%" (
    echo [X] Khong tim thay: "%REBAR_BAT%"
    exit /b 1
)

set "STATUS_FILE=%ProgramData%\40HXUnlock\gen2_status.txt"
set "REBAR_STATUS_FILE=%ProgramData%\40HXUnlock\rebar_status.txt"
set "APP_DIR=%ProgramFiles%\40HXUnlock"
set "RUN_BAT=%APP_DIR%\RunUnlock.bat"

set /a TOTAL_TESTS=0
set /a PASSED_TESTS=0
set /a FAILED_TESTS=0
set "IS_CLI=0"

:: Phan tich tham so dong lenh truc tiep
set "TARGET_CMD="
for %%a in (%*) do (
    if /i "%%~a"=="-clean" set "TARGET_CMD=cmd_clean"
    if /i "%%~a"=="-uninstall" set "TARGET_CMD=cmd_clean"
    if /i "%%~a"=="/u" set "TARGET_CMD=cmd_clean"
    if /i "%%~a"=="-run" set "TARGET_CMD=cmd_run"
    if /i "%%~a"=="-test" set "TARGET_CMD=cmd_run"
    if /i "%%~a"=="-fail" set "TARGET_CMD=cmd_fail"
    if /i "%%~a"=="-winring0" set "TARGET_CMD=cmd_winring0"
    if /i "%%~a"=="-nogpu" set "TARGET_CMD=cmd_nogpu"
    if /i "%%~a"=="-missing-status" set "TARGET_CMD=cmd_missing_status"
    if /i "%%~a"=="-preflight" set "TARGET_CMD=cmd_preflight"
    if /i "%%~a"=="-rebar" set "TARGET_CMD=cmd_rebar"
    if /i "%%~a"=="-rebar-happy" set "TARGET_CMD=cmd_rebar_happy"
    if /i "%%~a"=="-rebar-laptop" set "TARGET_CMD=cmd_rebar_laptop"
    if /i "%%~a"=="-rebar-clean" set "TARGET_CMD=cmd_rebar_clean"
    if /i "%%~a"=="-go" set "TARGET_CMD=cmd_gotest"
    if /i "%%~a"=="-gotest" set "TARGET_CMD=cmd_gotest"
    if /i "%%~a"=="-linux" set "TARGET_CMD=cmd_linux"
    if /i "%%~a"=="-wsl" set "TARGET_CMD=cmd_linux"
    if /i "%%~a"=="-all" set "TARGET_CMD=cmd_auto"
    if /i "%%~a"=="-auto" set "TARGET_CMD=cmd_auto"
)
if defined TARGET_CMD (
    set "IS_CLI=1"
    goto :!TARGET_CMD!
)

:menu
cls
echo ================================================================
echo    BO TRINH KIEM THU TU DONG HOA CMP 30HX GEN2 & REBAR AIO
echo ================================================================
echo.
echo  --- CAC TEST SUITE GEN2 [Setup_CMP30HX_WindowsAIO.bat] ---
echo  [1] Test Suite 1: Kiem thu Gen2 nhanh thanh cong (Gen1 - Soft Reset - Gen2)
echo  [2] Test Suite 2: Kiem thu Gen2 nhanh that bai (Soft Reset khong the cuu van)
echo  [3] Test Suite 3: Kiem thu Gen2 driver WinRing0 bi chan (HVCI / Blocklist)
echo  [4] Test Suite 4: Kiem thu Gen2 khong tim thay GPU CMP 30HX tren bus PCI
echo  [5] Test Suite 5: Don dep / Go bo Gen2 he thong (Uninstall Gen2)
echo  [6] Test Suite 9: Kiem thu Co che phong thu chan bao thanh cong ao khi mat Status
echo  [7] Test Suite 10: Kiem thu Che do chay doc lap Chan doan he thong (Preflight Only)
echo.
echo  --- CAC TEST SUITE RESIZABLE BAR [Setup_ReBar_CMP30HX.bat] ---
echo  [8] Test Suite 6: Kiem thu Resizable BAR 1-Click AIO (Happy Path)
echo  [9] Test Suite 7: Kiem thu Khoa an toan ReBAR chan Laptop (Laptop Guard)
echo  [10] Test Suite 8: Don dep / Go bo Resizable BAR (Uninstall ReBAR)
echo.
echo  --- CAC TEST SUITE NEN TANG (GO ENGINE & LINUX / WSL) ---
echo  [11] Test Suite 11: Kiem thu Go Unit Tests (40hxcore: 13 tests, unlockriot: 18 tests)
echo  [12] Test Suite 12: Kiem thu Linux / WSL AIO Mock Test Suite (8 Suites, 42 Assertions)
echo.
echo  --- FULL AUTOMATED TEST RUNNER ---
echo  [A] Full Test Suite: Chay tat ca 12 Suites + Assertions + Report
echo  [0] Thoat
echo.
echo ================================================================
set /p "CHOICE=Nhap lua chon cua ban [1-12, A, 0] (Mac dinh: A): "
if "%CHOICE%"=="" set "CHOICE=A"

if /i "%CHOICE%"=="1" goto :cmd_run
if /i "%CHOICE%"=="2" goto :cmd_fail
if /i "%CHOICE%"=="3" goto :cmd_winring0
if /i "%CHOICE%"=="4" goto :cmd_nogpu
if /i "%CHOICE%"=="5" goto :cmd_clean
if /i "%CHOICE%"=="6" goto :cmd_missing_status
if /i "%CHOICE%"=="7" goto :cmd_preflight
if /i "%CHOICE%"=="8" goto :cmd_rebar_happy
if /i "%CHOICE%"=="9" goto :cmd_rebar_laptop
if /i "%CHOICE%"=="10" goto :cmd_rebar_clean
if /i "%CHOICE%"=="11" goto :cmd_gotest
if /i "%CHOICE%"=="12" goto :cmd_linux
if /i "%CHOICE%"=="A" goto :cmd_auto
if /i "%CHOICE%"=="0" exit /b 0

echo [!] Lua chon khong hop le.
timeout /t 2 >nul
goto :menu

:cmd_run
cls
echo [*] KHOI CHAY TEST SUITE 1 (HAPPY PATH)...
call :test_suite_happy
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" (
    echo.
    set /p "CLEAN_NOW=Ban co muon don dep sach ngay bay gio khong? (Y/N, mac dinh Y): "
    if "!CLEAN_NOW!"=="" set "CLEAN_NOW=y"
    if /i "!CLEAN_NOW!"=="y" (
        call :test_suite_uninstall
    )
    pause
)
exit /b !SUB_EC!

:cmd_fail
cls
echo [*] KHOI CHAY TEST SUITE 2 (FAILURE BRANCH)...
call :test_suite_fail
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" (
    echo.
    set /p "CLEAN_NOW=Ban co muon don dep sach ngay bay gio khong? (Y/N, mac dinh Y): "
    if "!CLEAN_NOW!"=="" set "CLEAN_NOW=y"
    if /i "!CLEAN_NOW!"=="y" (
        call :test_suite_uninstall
    )
    pause
)
exit /b !SUB_EC!

:cmd_winring0
cls
echo [*] KHOI CHAY TEST SUITE 3 (WINRING0 DRIVER BLOCKED)...
call :test_suite_winring0
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" (
    echo.
    set /p "CLEAN_NOW=Ban co muon don dep sach ngay bay gio khong? (Y/N, mac dinh Y): "
    if "!CLEAN_NOW!"=="" set "CLEAN_NOW=y"
    if /i "!CLEAN_NOW!"=="y" (
        call :test_suite_uninstall
    )
    pause
)
exit /b !SUB_EC!

:cmd_nogpu
cls
echo [*] KHOI CHAY TEST SUITE 4 (NO GPU CMP 30HX FOUND)...
call :test_suite_nogpu
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" (
    echo.
    set /p "CLEAN_NOW=Ban co muon don dep sach ngay bay gio khong? (Y/N, mac dinh Y): "
    if "!CLEAN_NOW!"=="" set "CLEAN_NOW=y"
    if /i "!CLEAN_NOW!"=="y" (
        call :test_suite_uninstall
    )
    pause
)
exit /b !SUB_EC!

:cmd_clean
cls
echo [*] KHOI CHAY TEST SUITE 5 (UNINSTALL ^& CLEANUP)...
call :test_suite_uninstall
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_missing_status
cls
echo [*] KHOI CHAY TEST SUITE 9 (MISSING STATUS GUARD)...
call :test_suite_missing_status
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_preflight
cls
echo [*] KHOI CHAY TEST SUITE 10 (PREFLIGHT DIAGNOSTIC STANDALONE)...
call :test_suite_preflight
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_rebar_happy
cls
echo [*] KHOI CHAY TEST SUITE 6 (REBAR 1-CLICK AIO HAPPY PATH)...
call :test_suite_rebar_happy
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_rebar_laptop
cls
echo [*] KHOI CHAY TEST SUITE 7 (REBAR LAPTOP SAFETY GUARD)...
call :test_suite_rebar_laptop
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_rebar_clean
cls
echo [*] KHOI CHAY TEST SUITE 8 (REBAR UNINSTALL)...
call :test_suite_rebar_uninstall
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_gotest
cls
echo [*] KHOI CHAY TEST SUITE 11 (GO UNIT TESTS)...
call :test_suite_gotest
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_linux
cls
echo [*] KHOI CHAY TEST SUITE 12 (LINUX / WSL MOCK TEST SUITE)...
call :test_suite_linux
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_rebar
cls
echo ================================================================
echo    CHAY KIEM THU REBAR 3 SUITES (6, 7, 8)
echo ================================================================
call :test_suite_rebar_happy
call :test_suite_rebar_laptop
call :test_suite_rebar_uninstall
call :print_summary
set "FINAL_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !FINAL_EC!

:cmd_auto
cls
echo ================================================================
echo    CHAY KIEM THU TU DONG TOAN BO 12 TEST SUITES VA ASSERTIONS
echo ================================================================
call :test_suite_happy
call :test_suite_fail
call :test_suite_winring0
call :test_suite_nogpu
call :test_suite_missing_status
call :test_suite_preflight
call :test_suite_uninstall
call :test_suite_rebar_happy
call :test_suite_rebar_laptop
call :test_suite_rebar_uninstall
call :test_suite_gotest
call :test_suite_linux
call :print_summary
set "FINAL_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !FINAL_EC!

:: ================================================================
:: TEST SUITES
:: ================================================================
:test_suite_happy
echo.
echo [*] [TEST SUITE 1] Kiem thu mo phong thanh cong (Gen1 - Soft Reset - Gen2)...
call :clean_baseline
call "%TARGET_BAT%" -test -nocheck -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 1) ---
call :assert_exit_code "0" "!EC!" "Setup tra ve ExitCode 0 [Thanh cong]"
call :assert_file_exists "%STATUS_FILE%" "File gen2_status.txt duoc tao hop le"
call :assert_file_contains "%STATUS_FILE%" "STATUS_CODE=GEN2_SUCCESS" "Seam 2: Token STATUS_CODE=GEN2_SUCCESS hop le"
call :assert_file_contains "%STATUS_FILE%" "GPU TLS=Gen2" "Nhan trang thai GPU TLS=Gen2"
call :assert_file_contains "%STATUS_FILE%" "MRRS: 512B" "Nhan trang thai MRRS: 512B da toi uu"
call :assert_file_contains "%STATUS_FILE%" "da dat muc tieu Gen2 thanh cong" "Xac nhan dong trang thai hoan tat"
call :assert_task_exists "CMP30HX_Gen2_Unlock" "Scheduled Task SYSTEM da duoc dang ky"
call :assert_reg_exists "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" "CMP30HX_Gen2" "Registry Run Key HKLM da duoc tao"
call :assert_file_exists "%RUN_BAT%" "File script duy tri RunUnlock.bat ton tai"
exit /b 0

:test_suite_fail
echo.
echo [*] [TEST SUITE 2] Kiem thu nhanh that bai (Soft Reset khong cuu van duoc)...
call :clean_baseline
call "%TARGET_BAT%" -mock-fail -nocheck -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 2) ---
call :assert_exit_code "1" "!EC!" "Setup tra ve ExitCode 1 khi mo phong that bai"
call :assert_file_exists "%STATUS_FILE%" "File gen2_status.txt duoc ghi nhan"
call :assert_file_contains "%STATUS_FILE%" "STATUS_CODE=GEN1_STUCK" "Seam 2: Token STATUS_CODE=GEN1_STUCK hop le"
call :assert_file_contains "%STATUS_FILE%" "chua dat muc tieu Gen2" "Xac nhan trang thai chua dat Gen2"
exit /b 0

:test_suite_winring0
echo.
echo [*] [TEST SUITE 3] Kiem thu driver WinRing0 bi chan (HVCI / Blocklist / Error 5)...
call :clean_baseline
call "%TARGET_BAT%" -mock-winring0 -nocheck -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 3) ---
call :assert_exit_code "1" "!EC!" "Setup tra ve ExitCode 1 khi driver bi chan"
call :assert_file_exists "%STATUS_FILE%" "File gen2_status.txt duoc tao"
call :assert_file_contains "%STATUS_FILE%" "STATUS_CODE=DRV_FAIL" "Seam 2: Token STATUS_CODE=DRV_FAIL hop le"
call :assert_file_contains "%STATUS_FILE%" "WinRing0" "File status ghi nhan loi driver WinRing0"
exit /b 0

:test_suite_nogpu
echo.
echo [*] [TEST SUITE 4] Kiem thu khong tim thay GPU CMP 30HX tren bus PCI...
call :clean_baseline
call "%TARGET_BAT%" -mock-nogpu -nocheck -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 4) ---
call :assert_exit_code "1" "!EC!" "Setup tra ve ExitCode 1 khi khong tim thay GPU"
call :assert_file_exists "%STATUS_FILE%" "File gen2_status.txt duoc tao"
call :assert_file_contains "%STATUS_FILE%" "STATUS_CODE=NO_GPU" "Seam 2: Token STATUS_CODE=NO_GPU hop le"
call :assert_file_contains "%STATUS_FILE%" "PCI" "File status ghi nhan loi PCI bus"
exit /b 0

:test_suite_missing_status
echo.
echo [*] [TEST SUITE 9] Kiem thu co che phong thu chan bao thanh cong ao khi mat Status file...
call :clean_baseline
call "%TARGET_BAT%" -mock-missing-status -nocheck -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 9) ---
call :assert_exit_code "1" "!EC!" "Setup tra ve ExitCode 1 khi khong co status file"
call :assert_file_not_exists "%STATUS_FILE%" "Xac nhan file gen2_status.txt khong ton tai (mo phong crash)"
exit /b 0

:test_suite_preflight
echo.
echo [*] [TEST SUITE 10] Kiem thu che do chay doc lap Preflight Diagnostic [-preflight]...
call :clean_baseline
call "%TARGET_BAT%" -preflight -nocheck -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 10) ---
call :assert_exit_code "0" "!EC!" "Preflight diagnostic doc lap tra ve ExitCode 0"
call :assert_file_not_exists "%STATUS_FILE%" "Khong khoi chay installer khi chi kiem tra preflight"
call :assert_task_not_exists "CMP30HX_Gen2_Unlock" "Khong dang ky Scheduled Task khi chi kiem tra preflight"
exit /b 0

:test_suite_uninstall
echo.
echo [*] [TEST SUITE 5] Kiem thu go bo va don dep sach se he thong...
call "%TARGET_BAT%" -uninstall -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 5) ---
call :assert_exit_code "0" "!EC!" "Lenh Uninstall tra ve ExitCode 0"
call :assert_task_not_exists "CMP30HX_Gen2_Unlock" "Scheduled Task da bi xoa triet de"
call :assert_reg_not_exists "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" "CMP30HX_Gen2" "Registry Run Key HKLM da bi xoa"
call :assert_file_not_exists "%STATUS_FILE%" "File gen2_status.txt da duoc don dep"
call :assert_file_not_exists "%APP_DIR%" "Thu muc he thong 40HXUnlock da bi xoa"
exit /b 0

:test_suite_rebar_happy
echo.
echo [*] [TEST SUITE 6] Kiem thu Resizable BAR 1-Click AIO (Happy Path)...
if exist "%REBAR_STATUS_FILE%" del /f /q "%REBAR_STATUS_FILE%" >nul 2>&1
call "%REBAR_BAT%" -test -nocheck -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 6) ---
call :assert_exit_code "0" "!EC!" "ReBAR Setup tra ve ExitCode 0 [Thanh cong]"
call :assert_file_exists "%REBAR_STATUS_FILE%" "File rebar_status.txt duoc tao hop le"
call :assert_file_contains "%REBAR_STATUS_FILE%" "8192 MB" "Xac nhan BAR1 Size = 8192 MB [8GB luy thua 2]"
call :assert_file_contains "%REBAR_STATUS_FILE%" "Turing Override" "Xac nhan Driver Profile rBAR da bat"
call :assert_file_contains "%REBAR_STATUS_FILE%" "MOCK TEST SIMULATION" "Xac nhan che do mo phong an toan"
call :assert_task_exists "NVIDIA_ReBAR_Global_Profile" "Scheduled Task ReBAR Profile da duoc tao"
call :assert_reg_exists "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" "NVIDIA_ReBAR_Profile" "Registry Run Key HKLM ReBAR da duoc tao"
exit /b 0

:test_suite_rebar_laptop
echo.
echo [*] [TEST SUITE 7] Kiem thu Khoa an toan ReBAR chan he thong Laptop...
call "%REBAR_BAT%" -mock-laptop -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 7) ---
call :assert_exit_code "1" "!EC!" "ReBAR Setup chan dung 100%% tren Laptop [ExitCode 1]"
exit /b 0

:test_suite_rebar_uninstall
echo.
echo [*] [TEST SUITE 8] Kiem thu go bo va khoi phuc mac dinh Resizable BAR...
call "%REBAR_BAT%" -uninstall -mock -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 8) ---
call :assert_exit_code "0" "!EC!" "Lenh Uninstall ReBAR tra ve ExitCode 0"
call :assert_file_not_exists "%REBAR_STATUS_FILE%" "File rebar_status.txt da bi xoa triet de"
call :assert_task_not_exists "NVIDIA_ReBAR_Global_Profile" "Scheduled Task ReBAR Profile da bi xoa triet de"
call :assert_reg_not_exists "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" "NVIDIA_ReBAR_Profile" "Registry Run Key HKLM ReBAR da bi xoa"
exit /b 0

:test_suite_gotest
echo.
echo [*] [TEST SUITE 11] Kiem thu Go Unit Tests (40hxcore ^& unlockriot)...
set "GO_BIN="
where go >nul 2>&1 && set "GO_BIN=go"
if not defined GO_BIN (
    echo       [WARN] Khong tim thay Go tren PATH. Bo qua kiem thu Go.
    exit /b 0
)
pushd "%~dp0windows-v3.0\tools\40hxcore"
go test -count=1 ./... >nul 2>&1
set "GO_EC1=!ERRORLEVEL!"
popd
call :assert_exit_code "0" "!GO_EC1!" "Go Core 13 Unit Tests (eFuse, MRRS, BOOT_0, PnP) deu DAT"

pushd "%~dp0windows-v3.0\tools\unlockriot"
go test -count=1 ./... >nul 2>&1
set "GO_EC2=!ERRORLEVEL!"
popd
call :assert_exit_code "0" "!GO_EC2!" "Go UnlockRiot 18 Unit Tests (Riot Policy, Authenticode, Mock UEFI) deu DAT"
exit /b 0

:test_suite_linux
echo.
echo [*] [TEST SUITE 12] Kiem thu Linux / WSL Mock Test Suite (Test_Mock_CMP30HX_Linux.sh)...
set "WSL_BIN="
where wsl >nul 2>&1 && set "WSL_BIN=wsl"
if not defined WSL_BIN (
    echo       [WARN] Khong tim thay WSL tren he thong. Bo qua kiem thu Linux.
    exit /b 0
)
if not exist "%~dp0Test_Mock_CMP30HX_Linux.sh" (
    echo       [WARN] Khong tim thay Test_Mock_CMP30HX_Linux.sh. Bo qua kiem thu Linux.
    exit /b 0
)
wsl bash ./Test_Mock_CMP30HX_Linux.sh >nul 2>&1
set "WSL_EC=!ERRORLEVEL!"
call :assert_exit_code "0" "!WSL_EC!" "WSL Linux Mock Test Suite 8 Suites (42 Assertions) deu DAT"
exit /b 0

:clean_baseline
schtasks /delete /tn "CMP30HX_Gen2_Unlock" /f >nul 2>&1
schtasks /delete /tn "CMP30HX_Gen2_Unlock_User" /f >nul 2>&1
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1
schtasks /delete /tn "40HX PCIe Gen2 Bring-up" /f >nul 2>&1
schtasks /delete /tn "NVIDIA_ReBAR_Global_Profile" /f >nul 2>&1
reg delete "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /f >nul 2>&1
reg delete "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /f >nul 2>&1
if exist "%STATUS_FILE%" del /f /q "%STATUS_FILE%" >nul 2>&1
if exist "%REBAR_STATUS_FILE%" del /f /q "%REBAR_STATUS_FILE%" >nul 2>&1
exit /b 0

:: ================================================================
:: CAC HAM ASSERTION TU DONG
:: ================================================================
:assert_exit_code
set /a TOTAL_TESTS+=1
if "%~1"=="%~2" goto :pass_ec
echo       [FAIL] %~3 [Mong doi: %~1, Thuc te: %~2]
set /a FAILED_TESTS+=1
exit /b 0
:pass_ec
echo       [PASS] %~3 [ExitCode: %~2]
set /a PASSED_TESTS+=1
exit /b 0

:assert_file_exists
set /a TOTAL_TESTS+=1
if exist "%~1" goto :pass_fe
echo       [FAIL] %~2 [Khong tim thay: %~1]
set /a FAILED_TESTS+=1
exit /b 0
:pass_fe
echo       [PASS] %~2
set /a PASSED_TESTS+=1
exit /b 0

:assert_file_not_exists
set /a TOTAL_TESTS+=1
if not exist "%~1" goto :pass_fne
echo       [FAIL] %~2 [File/Thu muc van con ton tai: %~1]
set /a FAILED_TESTS+=1
exit /b 0
:pass_fne
echo       [PASS] %~2
set /a PASSED_TESTS+=1
exit /b 0

:assert_file_contains
set /a TOTAL_TESTS+=1
if not exist "%~1" (
    echo       [FAIL] %~3 [File khong ton tai de kiem tra]
    set /a FAILED_TESTS+=1
    exit /b 0
)
findstr /i /c:"%~2" "%~1" >nul 2>&1
if errorlevel 1 goto :fail_fc
echo       [PASS] %~3
set /a PASSED_TESTS+=1
exit /b 0
:fail_fc
echo       [FAIL] %~3 [Khong chua chuoi: "%~2"]
set /a FAILED_TESTS+=1
exit /b 0

:assert_task_exists
set /a TOTAL_TESTS+=1
schtasks /query /tn "%~1" >nul 2>&1
if errorlevel 1 goto :fail_te
echo       [PASS] %~2
set /a PASSED_TESTS+=1
exit /b 0
:fail_te
echo       [FAIL] %~2 [Scheduled Task khong ton tai: %~1]
set /a FAILED_TESTS+=1
exit /b 0

:assert_task_not_exists
set /a TOTAL_TESTS+=1
schtasks /query /tn "%~1" >nul 2>&1
if not errorlevel 1 goto :fail_tne
echo       [PASS] %~2
set /a PASSED_TESTS+=1
exit /b 0
:fail_tne
echo       [FAIL] %~2 [Scheduled Task van chua bi xoa: %~1]
set /a FAILED_TESTS+=1
exit /b 0

:assert_reg_exists
set /a TOTAL_TESTS+=1
reg query "%~1" /v "%~2" >nul 2>&1
if errorlevel 1 goto :fail_re
echo       [PASS] %~3
set /a PASSED_TESTS+=1
exit /b 0
:fail_re
echo       [FAIL] %~3 [Registry key/value khong ton tai: %~1 -> %~2]
set /a FAILED_TESTS+=1
exit /b 0

:assert_reg_not_exists
set /a TOTAL_TESTS+=1
reg query "%~1" /v "%~2" >nul 2>&1
if not errorlevel 1 goto :fail_rne
echo       [PASS] %~3
set /a PASSED_TESTS+=1
exit /b 0
:fail_rne
echo       [FAIL] %~3 [Registry value van chua bi xoa: %~1 -> %~2]
set /a FAILED_TESTS+=1
exit /b 0

:print_summary
echo.
echo ================================================================
echo                    BANG TONG KET KIEM THU (TEST REPORT)
echo ================================================================
echo  Tong so phep kiem tra (Total Assertions): !TOTAL_TESTS!
echo  So phep dat (Passed)                     : !PASSED_TESTS!
echo  So phep hong (Failed)                    : !FAILED_TESTS!
echo ================================================================
if !FAILED_TESTS! GTR 0 (
    echo  [X] KET QUA: CO !FAILED_TESTS! PHEP KIEM THU THAT BAI!
    echo ================================================================
    exit /b 1
) else (
    echo  [V] KET QUA: TOAN BO !TOTAL_TESTS! PHEP KIEM THU DEU DAT [100%% PASS]!
    echo ================================================================
    exit /b 0
)
