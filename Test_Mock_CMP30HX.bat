@echo off
setlocal EnableDelayedExpansion
chcp 65001 >nul
title CMP 30HX Gen2 Automated Mock Test Suite

:: Kiem tra quyen Administrator
set "IS_ELEVATED=0"
fltmc >nul 2>&1 && set "IS_ELEVATED=1"
if "!IS_ELEVATED!"=="0" (
    fsutil dirty query %systemdrive% >nul 2>&1 && set "IS_ELEVATED=1"
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

set "STATUS_FILE=%ProgramData%\40HXUnlock\gen2_status.txt"
set "APP_DIR=%ProgramFiles%\40HXUnlock"
set "RUN_BAT=%APP_DIR%\RunUnlock.bat"

set /a TOTAL_TESTS=0
set /a PASSED_TESTS=0
set /a FAILED_TESTS=0
set "IS_CLI=0"

:: Phan tich tham so dong lenh truc tiep
if /i "%~1"=="-clean" (
    set "IS_CLI=1"
    goto :cmd_clean
)
if /i "%~1"=="-uninstall" (
    set "IS_CLI=1"
    goto :cmd_clean
)
if /i "%~1"=="/u" (
    set "IS_CLI=1"
    goto :cmd_clean
)
if /i "%~1"=="-run" (
    set "IS_CLI=1"
    goto :cmd_run
)
if /i "%~1"=="-test" (
    set "IS_CLI=1"
    goto :cmd_run
)
if /i "%~1"=="-fail" (
    set "IS_CLI=1"
    goto :cmd_fail
)
if /i "%~1"=="-auto" (
    set "IS_CLI=1"
    goto :cmd_auto
)

:menu
cls
echo ================================================================
echo    BO TRINH KIEM THU TU DONG HOA CMP 30HX GEN2 (TEST SUITE)
echo ================================================================
echo.
echo  [1] Test Suite 1: Kiem thu nhanh thanh cong (Gen1 - Soft Reset - Gen2)
echo  [2] Test Suite 2: Kiem thu nhanh that bai (Mock Failure Branch)
echo  [3] Test Suite 3: Don dep / Go bo cai dat he thong (Uninstall)
echo  [4] Full Test Suite: Chay tat ca 3 Suites + Assertions + Report
echo  [5] Thoat
echo.
echo ================================================================
set /p "CHOICE=Nhap lua chon cua ban [1-5] (Mac dinh: 4): "
if "%CHOICE%"=="" set "CHOICE=4"

if "%CHOICE%"=="1" goto :cmd_run
if "%CHOICE%"=="2" goto :cmd_fail
if "%CHOICE%"=="3" goto :cmd_clean
if "%CHOICE%"=="4" goto :cmd_auto
if "%CHOICE%"=="5" exit /b 0

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

:cmd_clean
cls
echo [*] KHOI CHAY TEST SUITE 3 (UNINSTALL ^& CLEANUP)...
call :test_suite_uninstall
call :print_summary
set "SUB_EC=!ERRORLEVEL!"
if not "!IS_CLI!"=="1" pause
exit /b !SUB_EC!

:cmd_auto
cls
echo ================================================================
echo    CHAY KIEM THU TU DONG TOAN BO TEST SUITES VA ASSERTIONS
echo ================================================================
call :test_suite_happy
call :test_suite_fail
call :test_suite_uninstall
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
call :assert_file_contains "%STATUS_FILE%" "GPU TLS=Gen2" "Nhan trang thai GPU TLS=Gen2"
call :assert_file_contains "%STATUS_FILE%" "MRRS: 512B" "Nhan trang thai MRRS: 512B da toi uu"
call :assert_file_contains "%STATUS_FILE%" "da dat muc tieu Gen2 thanh cong!" "Xac nhan dong trang thai hoan tat"
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
call :assert_file_contains "%STATUS_FILE%" "chua dat muc tieu Gen2!" "Xac nhan trang thai chua dat Gen2"
exit /b 0

:test_suite_uninstall
echo.
echo [*] [TEST SUITE 3] Kiem thu go bo va don dep sach se he thong...
call "%TARGET_BAT%" -uninstall -nowait -noadmin
set "EC=!ERRORLEVEL!"
echo.
echo     --- Ket qua kiem tra (Assertions - Suite 3) ---
call :assert_exit_code "0" "!EC!" "Lenh Uninstall tra ve ExitCode 0"
call :assert_task_not_exists "CMP30HX_Gen2_Unlock" "Scheduled Task da bi xoa triet de"
call :assert_reg_not_exists "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" "CMP30HX_Gen2" "Registry Run Key HKLM da bi xoa"
call :assert_file_not_exists "%STATUS_FILE%" "File gen2_status.txt da duoc don dep"
call :assert_file_not_exists "%APP_DIR%" "Thu muc he thong 40HXUnlock da bi xoa"
exit /b 0

:clean_baseline
schtasks /delete /tn "CMP30HX_Gen2_Unlock" /f >nul 2>&1
schtasks /delete /tn "CMP30HX_Gen2_Unlock_User" /f >nul 2>&1
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1
schtasks /delete /tn "40HX PCIe Gen2 Bring-up" /f >nul 2>&1
reg delete "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "CMP30HX_Gen2" /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /f >nul 2>&1
if exist "%STATUS_FILE%" del /f /q "%STATUS_FILE%" >nul 2>&1
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
echo       [FAIL] %~3 [Registry key/value khong ton tai: %~1 -^> %~2]
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
echo       [FAIL] %~3 [Registry value van chua bi xoa: %~1 -^> %~2]
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
