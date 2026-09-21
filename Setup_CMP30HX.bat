@echo off
setlocal
chcp 65001 >nul
title CMP 30HX Gen2 x16 Auto Setup

:: Kiem tra quyen Administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [!] Dang yeu cau quyen Administrator...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~f0' -ArgumentList '%*' -Verb RunAs"
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

:: 1. Don dep task retry cu (neu co), tranh vong lap retry Gen3/Gen2 cu
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1

:: 2. Tat tinh nang tiet kiem dien PCIe ASPM va Hybrid Sleep (tranh bi ha ve Gen1 khi idle)
echo [1/6] Dang tat PCIe ASPM va Hybrid Sleep...
set "POWERCFG_OK=1"
powercfg -setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0 >nul 2>&1
if errorlevel 1 set "POWERCFG_OK=0"
powercfg -setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0 >nul 2>&1
if errorlevel 1 set "POWERCFG_OK=0"
powercfg -setacvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0 >nul 2>&1
if errorlevel 1 set "POWERCFG_OK=0"
powercfg -setdcvalueindex SCHEME_CURRENT SUB_SLEEP HYBRIDSLEEP 0 >nul 2>&1
if errorlevel 1 set "POWERCFG_OK=0"
powercfg -setactive SCHEME_CURRENT >nul 2>&1
if errorlevel 1 set "POWERCFG_OK=0"
if "%POWERCFG_OK%"=="1" (
    echo       [OK] Da tat PCIe ASPM va Hybrid Sleep.
) else (
    echo       [X] Khong tat duoc power setting. Kiem tra quyen Administrator va Power Plan hien tai.
)

:: 3. Tat Fast Startup de tranh cache kernel giu link Gen1 sau khoi dong lai
echo [2/6] Dang tat Fast Startup (Hiberboot)...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f >nul 2>&1
if errorlevel 1 (
    echo       [X] Khong tat duoc Fast Startup. Hay chay lai bang Administrator.
) else (
    reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x0" >nul 2>&1
    if errorlevel 1 (echo       [!] Fast Startup chua xac nhan OFF.) else (echo       [OK] Fast Startup da tat.)
)

:: 4. Tat Microsoft Vulnerable Driver Blocklist (tranh Windows chan driver sau reboot)
echo [3/6] Dang tat Microsoft Vulnerable Driver Blocklist...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" /t REG_DWORD /d 0 /f >nul 2>&1
if errorlevel 1 (
    echo       [X] Khong tat duoc Driver Blocklist. Kiem tra chinh sach bao mat Windows.
) else (
    reg query "HKLM\SYSTEM\CurrentControlSet\Control\CI\Config" /v "VulnerableDriverBlocklistEnable" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x0" >nul 2>&1
    if errorlevel 1 (echo       [!] Driver Blocklist chua xac nhan OFF.) else (echo       [OK] Driver Blocklist da tat.)
)

:: 5. Tat Memory Integrity (Core Isolation / HVCI) de driver MMIO khong bi chan
echo [4/6] Dang kiem tra va tat Memory Integrity (HVCI)...
set "NEED_REBOOT=0"
reg query "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x1" >nul 2>&1
if not errorlevel 1 set "NEED_REBOOT=1"
reg add "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" /t REG_DWORD /d 0 /f >nul 2>&1
if errorlevel 1 (
    echo       [X] Khong tat duoc Memory Integrity. Windows co the van chan driver kernel.
    set "HVCI_OK=0"
) else (
    reg query "HKLM\SYSTEM\CurrentControlSet\Control\DeviceGuard\Scenarios\HypervisorEnforcedCodeIntegrity" /v "Enabled" 2>nul | %SystemRoot%\System32\findstr.exe /i "0x1" >nul 2>&1
    if not errorlevel 1 (
        echo       [X] Memory Integrity van dang ON. Can reboot va kiem tra lai Windows Security.
        set "HVCI_OK=0"
        set "NEED_REBOOT=1"
    ) else (
        set "HVCI_OK=1"
        if "%NEED_REBOOT%"=="1" (echo       [!] Da dat HVCI=OFF; can reboot de driver duoc giai phong.) else (echo       [OK] Memory Integrity dang OFF.)
    )
)

:: 6. Dang ky Scheduled Task SYSTEM (startup, delay 45s)
echo [5/6] Dang tao Scheduled Task duy tri Gen2 sau moi lan khoi dong...
set "TASK_OK=0"
set "CMP30HX_INSTALLER=%INSTALLER%"
powershell -NoProfile -ExecutionPolicy Bypass -Command "$exe=$env:CMP30HX_INSTALLER; if (-not [IO.Path]::IsPathRooted($exe)) { throw 'Duong dan installer khong hop le' }; $a1=New-ScheduledTaskAction -Execute 'powercfg.exe' -Argument '-setacvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0'; $a2=New-ScheduledTaskAction -Execute 'powercfg.exe' -Argument '-setdcvalueindex SCHEME_CURRENT SUB_PCIEXPRESS ASPM 0'; $a3=New-ScheduledTaskAction -Execute 'powercfg.exe' -Argument '-setactive SCHEME_CURRENT'; $a4=New-ScheduledTaskAction -Execute $exe -Argument '-gen2-30hx -silent'; $t=New-ScheduledTaskTrigger -AtStartup; $t.Delay='PT45S'; $p=New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest; Register-ScheduledTask -TaskName 'CMP30HX_Gen2_Unlock' -Action @($a1,$a2,$a3,$a4) -Trigger $t -Principal $p -Force" >nul 2>&1
if not errorlevel 1 set "TASK_OK=1"
if "%TASK_OK%"=="1" (
    schtasks /query /tn "CMP30HX_Gen2_Unlock" >nul 2>&1
    if errorlevel 1 (
        echo       [X] Scheduled Task SYSTEM khong ton tai sau khi tao. Khong dam bao persistence sau reboot.
        set "TASK_OK=0"
    ) else (
        echo       [OK] Scheduled Task SYSTEM da duoc xac nhan; se chay khi Startup sau 45 giay.
        echo           Sau reboot: dang nhap Windows, cho du 45 giay roi moi kiem tra Gen2.
        echo           Task dang chay tu duong dan hien tai; khong di chuyen/xoa file nay sau khi cai dat.
    )
) else (
    echo       [X] Khong tao duoc Scheduled Task SYSTEM. Sau reboot can chay lai lenh mo khoa thu cong.
    echo           Ban fallback user da bi bo qua de tranh task user khong nap duoc driver kernel.
)

:: 7. Kich hoat mo khoa Gen2 ngay lap tuc
echo.
if "%NEED_REBOOT%"=="1" (
    echo [!] HVCI vua duoc dat OFF trong Registry nhung chua co hieu luc trong phien nay.
    echo     Buoc hien tai co the khong nap duoc driver kernel; sau khi ket thuc hay reboot truoc khi danh gia.
)
echo [6/6] Dang kich hoat Gen2 x16 va toi uu MRRS 512B ngay...
"%INSTALLER%" -gen2-30hx
if errorlevel 1 (
    echo       [X] Installer bao loi khi chay (exit code khac 0).
    echo           Kiem tra driver WinRing0/ThrottleStop, HVCI va quyen Administrator.
    set "UNLOCK_OK=0"
) else (
    set "UNLOCK_OK=1"
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
)

echo.
echo ================================================================
if "%UNLOCK_OK%"=="1" (
    echo  [V] CAI DAT HOAN TAT - Gen2 da duoc cau hinh.
    echo  - Scheduled Task SYSTEM se tu dong chay sau khi khoi dong 45 giay.
    echo  - LUU Y QUAN TRONG VE GEN 1 KHI VUA KHOI DONG / IDLE:
    echo    + Link PCIe se o Gen1 x16 khi card o che do ranh (Idle Power Saving / ASPM).
    echo    + Khi co tai 3D/CUDA/AIDA64/FurMark, card se tu dong bung toc do len Gen2 x16.
    echo    + Neu GPU-Z bao Gen1: nhap vao dau cham hoi (?) canh Bus Interface de chay Render Test!
    echo  - Neu sau 45 giay va co tai ma GPU van Gen1: kiem tra HVCI, riser, tiep xuc lane va BIOS khe PCIe.
) else (
    echo  [X] CAI DAT CHUA HOAN TAT - chua xac nhan duoc Gen2 trong phien hien tai.
    if "%NEED_REBOOT%"=="1" (
        echo  - LUU Y: Memory Integrity (HVCI) vua duoc tat, nhung can KHOI DONG LAI MAY de ap dung.
        echo    Sau khi reboot, Scheduled Task SYSTEM se tu dong thu nap driver va mo khoa Gen2.
    ) else (
        echo  - Khong ket luan thanh cong chi dua tren Root Port Gen2.
        echo  - Kiem tra file status va log chi tiet: %TEMP%\40HX_installer.log
        echo  - Neu da thu moi cach van loi: go bo sach driver cu bang DDU roi cai lai driver NVIDIA moi nhat.
    )
)
if "%TASK_OK%"=="1" (
    echo  - Sau khi reboot, cho task chay du 45 giay roi kiem tra bang 40HXCheck.exe.
) else (
    echo  - [Canh bao] Scheduled Task chua san sang; sau reboot phai chay lai Setup hoac lenh mo khoa thu cong.
)
if "%NEED_REBOOT%"=="1" call :warn_reboot
echo ================================================================
echo.
pause
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
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "40HXGen2" /f >nul 2>&1
echo [V] Da xoa toan bo cac Scheduled Task va Run key tu khoi dong lien quan.
echo.
pause
exit /b 0


:warn_reboot
echo.
echo  [!] LUU Y BAT BUOC: He thong vua tat Memory Integrity (Core Isolation).
echo      Ban PHAI KHOI DONG LAI MAY de Windows giai phong driver kernel.
echo      Sau reboot: cho Scheduled Task chay du 45 giay, tao tai 3D/CUDA,
echo      sau do moi dung GPU-Z/40HXCheck de danh gia Gen2.
exit /b 0
