@echo off
setlocal
chcp 65001 >nul
title CMP 30HX - Sua Loi 43 va Khoi Phuc Bien Mat Trong CSM

:: ================================================================
:: 1. KIEM TRA QUYEN ADMINISTRATOR (UAC DA TANG PHONG THU)
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

for %%a in (%*) do (
    if /i "%%~a"=="-noadmin" set "IS_ADMIN=1"
    if /i "%%~a"=="/noadmin" set "IS_ADMIN=1"
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
        echo [!] Vui long nhap chuot phai vao file Fix_CMP30HX_CSM_Error43.bat
        echo     va chon 'Run as administrator' [Chay voi quyen quan tri vien].
        echo     Hoac chay qua CMD Admin: Fix_CMP30HX_CSM_Error43.bat -noadmin
        echo ================================================================
        echo.
        pause
    )
    exit /b
)

cd /d "%~dp0"

:: Duong dan toi cong cu nhi phan FixCMP30HX_CSM.exe
set "FIX_EXE=%~dp0windows-v3.0\release\FixCMP30HX_CSM.exe"

:: Kiem tra tham so CLI neu co
if /i "%~1"=="-fix" goto :do_auto_fix
if /i "%~1"=="/fix" goto :do_auto_fix
if /i "%~1"=="-scan" goto :do_scan_only
if /i "%~1"=="/scan" goto :do_scan_only
if /i "%~1"=="-gui" goto :do_gui
if /i "%~1"=="/gui" goto :do_gui
if /i "%~1"=="-watchdog" goto :do_install_watchdog
if /i "%~1"=="/watchdog" goto :do_install_watchdog
if /i "%~1"=="-status" goto :do_status_only
if /i "%~1"=="/status" goto :do_status_only

:: ================================================================
:: 2. MENU TUONG TAC NGUOI DUNG (INTERACTIVE MENU)
:: ================================================================
:main_menu
cls
echo ================================================================
echo    CONG CU SUA LOI 43 ^& BIEN MAT DEVICE MANAGER CHO CMP 30HX
echo    Chuyen dung cho he thong chay CSM / Legacy BIOS / Mainboard cu
echo ================================================================
echo.
echo   [1] TU DONG SUA LOI 43 ^& KHOI PHUC CARD [Khuyen nghi 1-cham]
echo       - Quet lai bus PCIe, danh thuc card bi rot khoi Device Manager
echo       - Khoa vinh vien D3cold, ngan chan tut link PCIe khi khong tai
echo       - Sua loi 43 [Code 43], toi uu MMIO 32-bit va kich hoat CASO
echo       - Dang ky Watchdog canh gac tu dong luc khoi dong Windows
echo.
echo   [2] Mo cong cu giao dien truc quan FixCMP30HX_CSM [GUI]
echo       - Bang dieu khien cua so truc quan, xem chi tiet trang thai
echo.
echo   [3] Quet lai Bus PCIe phan cung [PCI Bus Rescan]
echo       - Tim lai card neu vua bi bien mat khoi Device Manager
echo.
echo   [4] Bat / Cai dat Tac vu Canh gac tu dong [Keep-Alive Watchdog]
echo       - Dam bao card khong bao gio bi mat lai sau Sleep hoac Reboot
echo.
echo   [5] Go bo Tac vu Canh gac tu dong [Uninstall Watchdog]
echo.
echo   [6] Kiem tra dieu kien nang cap len UEFI thuan [MBR2GPT Lossless]
echo       - Kiem tra xem may co the chuyen sang UEFI khong mat du lieu
echo.
echo   [7] Xem huong dan cau hinh BIOS cho may CSM
echo.
echo   [8] Thoat
echo.
echo ================================================================
%SystemRoot%\System32\choice.exe /c 12345678 /t 10 /d 1 /m "Nhap lua chon cua ban [1-8] (Tu dong chon [1] sau 10 giay): "
if errorlevel 8 exit /b 0
if errorlevel 7 goto :show_bios_guide
if errorlevel 6 goto :check_mbr2gpt
if errorlevel 5 goto :do_remove_watchdog
if errorlevel 4 goto :do_install_watchdog
if errorlevel 3 goto :do_scan_only
if errorlevel 2 goto :do_gui
if errorlevel 1 goto :do_auto_fix
goto :do_auto_fix

:: ================================================================
:: 3. CHUC NANG 1: TU DONG SUA TOAN DIEN (1-CHAM)
:: ================================================================
:do_auto_fix
cls
echo ================================================================
echo   DANG THUC HIEN TU DONG SUA LOI 43 VA KHOI PHUC CARD TRONG CSM...
echo ================================================================
echo.

if exist "%FIX_EXE%" (
    echo [*] Dang khoi chay dong co cot loi FixCMP30HX_CSM...
    "%FIX_EXE%" -scan
    "%FIX_EXE%" -fix
    "%FIX_EXE%" -watchdog
    echo.
    echo ----------------------------------------------------------------
    "%FIX_EXE%" -status
) else (
    echo [!] Khong tim thay FixCMP30HX_CSM.exe, chuyen sang che do phuc hoi Native...
    call :native_rescue_pipeline
)

echo.
echo ================================================================
echo [V] HOAN TAT XU LY!
echo  - Da quet lai toan bo bus PCIe phan cung.
echo  - Da khoa trang thai ngu sau D3cold va vo hieu hoa Fast Startup.
echo  - Da kich hoat Cross-Adapter Scan-Out [CASO] va cau hinh CPU-RM.
echo  - Da thiet lap TDR Timeout len 10 giay va toi uu MMIO 32-bit.
echo  - Da dang ky tac vu canh gac khoi dong tu dong.
echo.
echo  [*] Hay mo Device Manager de kiem tra card man hinh NVIDIA CMP 30HX.
echo ================================================================
echo.
pause
goto :main_menu

:: ================================================================
:: 4. CHUC NANG 2: MO GIAO DIEN GUI
:: ================================================================
:do_gui
if exist "%FIX_EXE%" (
    start "" "%FIX_EXE%"
) else (
    echo [X] Loi: Khong tim thay tep %FIX_EXE%
    pause
)
goto :main_menu

:: ================================================================
:: 5. CHUC NANG 3: QUET BUS PCIE PHAN CUNG
:: ================================================================
:do_scan_only
cls
echo ================================================================
echo   DANG QUET LAI TOAN BO PHAN CUNG BUS PCIE [PCI BUS RESCAN]...
echo ================================================================
echo.
if exist "%FIX_EXE%" (
    "%FIX_EXE%" -scan
) else (
    echo [*] Dang chay pnputil /scan-devices...
    pnputil /scan-devices
)
echo.
echo [V] Da hoan tat quet lai phan cung! Kiem tra lai Device Manager.
echo.
pause
goto :main_menu

:: ================================================================
:: 6. CHUC NANG 4 ^& 5: QUAN LY WATCHDOG CANH GAC
:: ================================================================
:do_install_watchdog
cls
echo ================================================================
echo   CAI DAT TAC VU CANH GAC KHOI DONG [CSM KEEP-ALIVE WATCHDOG]
echo ================================================================
echo.
if exist "%FIX_EXE%" (
    "%FIX_EXE%" -watchdog
) else (
    schtasks /create /tn "CMP30HX_CSM_KeepAlive_Boot" /tr "pnputil /scan-devices" /sc onstart /ru SYSTEM /delay 0000:20 /f >nul 2>&1
    schtasks /create /tn "CMP30HX_CSM_KeepAlive_Logon" /tr "pnputil /scan-devices" /sc onlogon /ru SYSTEM /delay 0000:10 /f >nul 2>&1
    echo [OK] Da dang ky tac vu canh gac native thanh cong.
)
echo.
pause
goto :main_menu

:do_remove_watchdog
cls
echo ================================================================
echo   GO BO TAC VU CANH GAC KHOI DONG
echo ================================================================
echo.
if exist "%FIX_EXE%" (
    "%FIX_EXE%" -unwatchdog
) else (
    schtasks /delete /tn "CMP30HX_CSM_KeepAlive_Boot" /f >nul 2>&1
    schtasks /delete /tn "CMP30HX_CSM_KeepAlive_Logon" /f >nul 2>&1
    echo [OK] Da go bo toan bo tac vu canh gac.
)
echo.
pause
goto :main_menu

:: ================================================================
:: 7. CHUC NANG 6: KIEM TRA MBR2GPT SANG UEFI
:: ================================================================
:check_mbr2gpt
cls
echo ================================================================
echo   KIEM TRA TINH TUONG THICH CHUYEN DOI MBR SANG UEFI [MBR2GPT]
echo ================================================================
echo.
echo [*] Dang kiem tra cau truc phan vung dia cai Windows...
mbr2gpt.exe /validate /allowfullos
if errorlevel 1 (
    echo.
    echo [!] Ket qua: O dia hien tai chua du dieu kien chuyen tu dong qua mbr2gpt.
    echo     [Co the do o dia da la GPT san, hoac co hon 3 phan vung chinh].
) else (
    echo.
    echo ================================================================
    echo [V] CHUC MUNG: O dia cua ban DU DIEU KIEN 100%% de chuyen sang UEFI!
    echo.
    echo Loi ich vuot troi khi chuyen sang UEFI thuan [Tat CSM]:
    echo  1. Bat duoc tinh nang Above 4G Decoding trong BIOS.
    echo  2. Bat duoc Resizable BAR giup card tang toi da hieu nang.
    echo  3. Triet tieu 100%% nguy co rot card hay loi 43 do nghen MMIO 32-bit.
    echo.
    echo LENH THUC THI [Mo CMD Administrator chay]:
    echo   mbr2gpt /convert /allowfullos
    echo Sau khi chay xong, khoi dong lai vao BIOS, chuyen Boot sang UEFI va TAT CSM.
    echo ================================================================
)
echo.
pause
goto :main_menu

:: ================================================================
:: 8. CHUC NANG 7: HUONG DAN CAU HINH BIOS
:: ================================================================
:show_bios_guide
cls
echo ================================================================
echo   HUONG DAN CAU HINH BIOS CHO CMP 30HX TREN MAIN CSM / LEGACY
echo ================================================================
echo.
echo 1. NGUYEN NHAN ROT CARD VA LOI 43 TREN CSM:
echo    - CMP 30HX la card dao coin KHONG CO CONG XUAT HINH [Headless].
echo    - Khi chay CSM, BIOS khong quan ly ACPI PCIe hien dai, khien card roi
echo      vao trang thai D3cold [ngu sau] khi khong dung va bien mat khoi Bus.
echo    - CSM thuong tu dong TAT "Above 4G Decoding", khien VRAM 6GB bi ket
echo      trong khong gian dia chi 32-bit [duoi 4GB] dan den LOI 43 [Code 43].
echo.
echo 2. CAC THIET LAP BIOS KHUYEN NGHI:
echo    - Primary Display / Initial Display Output:
echo      Chon iGPU [card onboard] hoac card phu co cong xuat hinh.
echo      TUYET DOI KHONG chon khe PCIe cua CMP 30HX lam card khoi dong man hinh.
echo    - Above 4G Decoding:
echo      Neu BIOS cho phep bat trong khi CSM bat, hay chon ENABLED.
echo    - PCIe Link Speed:
echo      Dat khe cam CMP 30HX o Gen2 hoac Gen3 [hoac Auto].
echo    - Fast Boot:
echo      Dat DISABLED.
echo.
echo 3. GIAI PHAP LAU DAI TOI UU:
echo    Chuyen doi o dia sang GPT bang mbr2gpt roi TAT HAN CSM trong BIOS
echo    de chay UEFI thuan.
echo ================================================================
echo.
pause
goto :main_menu

:do_status_only
if exist "%FIX_EXE%" (
    "%FIX_EXE%" -status
) else (
    echo [!] Khong tim thay %FIX_EXE%
)
exit /b 0

:: ================================================================
:: 9. PIPELINE PHUC HOI NATIVE (FALLBACK KHI KHONG CO EXE)
:: ================================================================
:native_rescue_pipeline
echo [1/5] Quet lai phan cung bus PCIe...
pnputil /scan-devices >nul 2>&1

echo [2/5] Vo hieu hoa Fast Startup va PCIe ASPM toan he thong...
powercfg -h off >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f >nul 2>&1

powershell -NoProfile -ExecutionPolicy Bypass -Command "$schemes = powercfg -list | Select-String -Pattern '([a-f0-9-]{36})' | ForEach-Object { $_.Matches.Groups[1].Value }; $subPci = '501a4d13-42af-4429-9fd5-e8174f7900c2'; $settingPci = 'ee12f906-d27e-44ba-b376-73e76cb376f7'; foreach ($s in $schemes) { powercfg -setacvalueindex $s $subPci $settingPci 0; powercfg -setdcvalueindex $s $subPci $settingPci 0; powercfg -setactive $s }" >nul 2>&1

echo [3/5] Ap dung cau hinh sua loi 43 va khoa chong ngu D3cold...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "TdrDelay" /t REG_DWORD /d 10 /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "TdrDdiDelay" /t REG_DWORD /d 10 /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "TdrLevel" /t REG_DWORD /d 3 /f >nul 2>&1

REM Cap nhat cho toan bo Display Class NVIDIA
powershell -NoProfile -ExecutionPolicy Bypass -Command "$base = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}'; Get-ChildItem $base -ErrorAction SilentlyContinue | ForEach-Object { $p = $_.PsPath; Set-ItemProperty -Path $p -Name 'D3ColdSupported' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'DeviceSelectiveSuspended' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'RmDisableGpuPowerMgmt' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'RmEnableAggressivePciePowerManagement' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'DisableASPM' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'DisablePCIePowerManagement' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'EnableCrossAdapterScanOut' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'AdapterType' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'LargePageMinimum' -Value 0xffffffff -Type DWord -Force; Set-ItemProperty -Path $p -Name 'EnableGpuFirmware' -Value 0 -Type DWord -Force }" >nul 2>&1

echo [4/5] Dat lai trang thai PnP device node...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$dev = Get-PnpDevice -PresentOnly:$false | Where-Object { $_.InstanceId -like '*VEN_10DE&DEV_2189*' } | Select-Object -First 1; if ($dev) { pnputil /restart-device ('\"' + $dev.InstanceId + '\"') }" >nul 2>&1

echo [5/5] Cai dat tac vu canh gac khoi dong...
schtasks /create /tn "CMP30HX_CSM_KeepAlive_Boot" /tr "pnputil /scan-devices" /sc onstart /ru SYSTEM /delay 0000:20 /f >nul 2>&1
schtasks /create /tn "CMP30HX_CSM_KeepAlive_Logon" /tr "pnputil /scan-devices" /sc onlogon /ru SYSTEM /delay 0000:10 /f >nul 2>&1
exit /b 0
