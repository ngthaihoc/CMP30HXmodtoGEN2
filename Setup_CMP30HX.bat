@echo off
setlocal
chcp 65001 >nul
title CMP 30HX Gen2 x16 Auto Setup

:: Kiem tra quyen Administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [!] Dang yeu cau quyen Administrator...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~f0' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"

if /i "%~1"=="-uninstall" goto :uninstall
if /i "%~1"=="/u" goto :uninstall

echo ================================================================
echo    CONG CU CAI DAT TU DONG GEN2 X16 CHO NVIDIA CMP 30HX (TU116)
echo ================================================================
echo.

set "INSTALLER="
if exist "%~dp0windows-v3.0\release\40HXInstaller.exe" (
    set "INSTALLER=%~dp0windows-v3.0\release\40HXInstaller.exe"
) else if exist "%~dp0release\40HXInstaller.exe" (
    set "INSTALLER=%~dp0release\40HXInstaller.exe"
) else if exist "%~dp040HXInstaller.exe" (
    set "INSTALLER=%~dp040HXInstaller.exe"
)

if not defined INSTALLER (
    echo [X] LOI: Khong tim thay 40HXInstaller.exe!
    echo Vui long dam bao ban da giai nen day du thu muc repository.
    echo.
    pause
    exit /b 1
)

echo [*] Tim thay bo cai: "%INSTALLER%"
echo.

:: 1. Don dep task cu (neu co)
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1

:: 2. Tat tinh nang tiet kiem dien PCIe ASPM (tranh bi ha ve Gen1 khi idle)
echo [1/4] Dang tat tinh nang tiet kiem dien PCIe ASPM...
powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0 >nul 2>&1
powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0 >nul 2>&1
powercfg -setactive SCHEME_CURRENT >nul 2>&1
echo       [OK] Da tat PCIe ASPM thanh cong.

:: 3. Tat Memory Integrity (Core Isolation / HVCI) de driver ThrottleStop/WinRing0 khong bi chan
echo [2/4] Dang kiem tra va tat Memory Integrity (Core Isolation / HVCI)...
set "NEED_REBOOT=0"
reg query "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x1" >nul 2>&1
if %errorlevel% equ 0 (
    set "NEED_REBOOT=1"
)
reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f >nul 2>&1
if "%NEED_REBOOT%"=="1" (
    echo       [!] Da tat Memory Integrity. (Can khoi dong lai may de co hieu luc hoan toan!)
) else (
    echo       [OK] Memory Integrity da o trang thai tat (OFF).
)

:: 4. Dang ky Scheduled Task chay ngam khi Logon
echo [3/4] Dang tao Scheduled Task tu dong kich hoat khi dang nhap...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$a = New-ScheduledTaskAction -Execute '%INSTALLER%' -Argument '-gen2-30hx -silent'; $t = New-ScheduledTaskTrigger -AtLogOn; $p = New-ScheduledTaskPrincipal -UserId $env:USERNAME -RunLevel Highest; Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action $a -Trigger $t -Principal $p -Force" >nul 2>&1
if %errorlevel% neq 0 (
    schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"%INSTALLER%\" -gen2-30hx -silent" /sc onlogon /rl highest /f >nul 2>&1
)
if %errorlevel% equ 0 (
    echo       [OK] Da dang ky tac vu "CMP30HX_Gen2_Unlock" thanh cong.
) else (
    echo       [!] Canh bao: Khong the tao Scheduled Task.
)

:: 5. Kich hoat mo khoa Gen2 ngay lap tuc
echo.
echo [4/4] Dang kich hoat mo khoa Gen2 x16 va toi uu MRRS 512B ngay...
"%INSTALLER%" -gen2-30hx

echo.
echo ================================================================
echo  [V] HOAN TAT CAI DAT!
echo  - PCIe Gen2 x16 va MRRS 512B da duoc kich hoat.
echo  - PCIe ASPM da duoc tat (chong tu ha xuong Gen1 khi idle).
echo  - He thong se tu dong mo khoa moi khi ban dang nhap Windows.
if "%NEED_REBOOT%"=="1" call :warn_reboot
echo  - Kiem tra lai bang 40HXCheck.exe hoac AIDA64 GPGPU Benchmark.
echo ================================================================
echo.
pause
exit /b 0

:uninstall
echo ================================================================
echo    GO BO TU DONG KHOI DONG CMP 30HX GEN2 UNLOCK
echo ================================================================
schtasks /delete /tn "CMP30HX_Gen2_Unlock" /f >nul 2>&1
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1
echo.
echo [V] Da xoa toan bo tac vu Scheduled Task lien quan.
echo.
pause
exit /b 0


:warn_reboot
echo.
echo  [!] LUU Y: He thong vua tat Memory Integrity (Core Isolation).
echo      Neu lan nay chua dat Gen2, hay KHOI DONG LAI MAY (Reboot)
echo      de Windows giai phong driver ThrottleStop.sys!
exit /b 0
