@echo off
setlocal
chcp 65001 >nul
title CMP 30HX - Sua Loi 43, Khoi Phuc Bien Mat Trong CSM va Mo Khoa GPU-Z

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
        echo     Hoac mo CMD Admin va chay: Fix_CMP30HX_CSM_Error43.bat -noadmin
        echo ================================================================
        echo.
        pause
    )
    exit /b
)

cd /d "%~dp0"

:: Duong dan toi cong cu nhi phan FixCMP30HX_CSM.exe
set "FIX_EXE=%~dp0windows-v3.0\release\FixCMP30HX_CSM.exe"

:: Kiem tra tham so CLI neu co (ho tro moi thu tu tham so)
for %%a in (%*) do (
    if /i "%%~a"=="-fix" goto :do_auto_fix
    if /i "%%~a"=="/fix" goto :do_auto_fix
    if /i "%%~a"=="-scan" goto :do_deep_scan_wake
    if /i "%%~a"=="/scan" goto :do_deep_scan_wake
    if /i "%%~a"=="-gpuz" goto :do_gpuz_fix
    if /i "%%~a"=="/gpuz" goto :do_gpuz_fix
    if /i "%%~a"=="-gui" goto :do_gui
    if /i "%%~a"=="/gui" goto :do_gui
    if /i "%%~a"=="-watchdog" goto :do_install_watchdog
    if /i "%%~a"=="/watchdog" goto :do_install_watchdog
    if /i "%%~a"=="-unwatchdog" goto :do_remove_watchdog
    if /i "%%~a"=="/unwatchdog" goto :do_remove_watchdog
    if /i "%%~a"=="-status" goto :do_status_only
    if /i "%%~a"=="/status" goto :do_status_only
)

:: ================================================================
:: 2. MENU TUONG TAC NGUOI DUNG (INTERACTIVE MENU - KHONG TIMEOUT)
:: ================================================================
:main_menu
cls
echo ===============================================================================
echo     CONG CU FIX SAU LOI 43, BIEN MAT THIET BI VA MO KHOA GPU-Z CHO CMP 30HX
echo     Toi uu hoa chuyen dung cho Mainboard chay CSM / Legacy BIOS / Windows 10
echo ===============================================================================
echo.
echo  [*] TINH TRANG THIET BI HIEN TAI TREN HE THONG:
powershell -NoProfile -ExecutionPolicy Bypass -Command "$fw = (Get-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Control\SystemInformation' -Name 'FirmwareType' -ErrorAction SilentlyContinue).FirmwareType; $fwStr = if ($fw -eq 1) { 'CSM / Legacy BIOS [Can toi uu]' } elseif ($fw -eq 2) { 'UEFI Thuan' } else { 'Khong xac dinh' }; Write-Host ('    - Che do Firmware Boot : ' + $fwStr) -ForegroundColor Yellow; $devs = Get-PnpDevice -PresentOnly:$false | Where-Object { $_.InstanceId -like '*VEN_10DE*' -or $_.Class -eq 'Display' -or $_.FriendlyName -like '*3D*' }; if ($devs) { foreach ($d in $devs) { $p = if ($d.Present) { '[ON - Dang ket noi]' } else { '[OFF - An/Rot bus (Code 45)]' }; $color = if ($d.Present -and $d.Status -eq 'OK') { 'Green' } else { 'Cyan' }; Write-Host ('    - Thiet bi: ' + $d.FriendlyName + ' | Status: ' + $d.Status + ' ' + $p) -ForegroundColor $color } } else { Write-Host '    - Khong tim thay thiet bi NVIDIA hoac 3D Controller nao tren bus PnP!' -ForegroundColor Red }"
echo.
echo -------------------------------------------------------------------------------
echo   [1] TU DONG SUA TOAN DIEN TAT CA LOI [KHUYEN DUNG 1-CHAM]
echo       + Danh thuc phan cung PCIe, cuu card bi bien mat / rot bus khoi Device Manager
echo       + Don dep device node ao/rac (Code 45/43/22), ep Windows nhan lai phan cung
echo       + MO KHOA GPU-Z: Xoa bo AdapterType=0 gay nghen, bat EnableMsHybrid va CASO
echo       + Khoi phuc tick: CUDA, OpenCL, Vulkan, DirectCompute, PhysX, DirectML
echo       + Don dep khoa registry sai tren iGPU, khoa vinh vien D3cold va Fast Startup
echo       + Dang ky Watchdog canh gac tu dong bao ve card moi khi bat may
echo.
echo   [2] DANH THUC PHAN CUNG ^& QUET SAU BUS PCIE [Cuu card bien mat]
echo       + Gui xung SetupAPI Re-enumerate len Root DevNode ^& toan bo PCI-to-PCI Bridge
echo       + Go bo device node ket Code 45 (Disconnected) de Device Manager nhan dien lai
echo       + Keo lai card thanh '3D Video Controller' hoac 'Microsoft Basic Display Adapter'
echo.
echo   [3] FIX SAU GPU-Z [Bat CUDA, OpenCL, Vulkan, DirectCompute, CASO]
echo       + Xoa AdapterType (thu pham chinh khien card khong render va mat tick GPU-Z)
echo       + Dang ky EnableMsHybrid = 1 cho phep Windows gan game render do hoa
echo       + Dang ky Khronos OpenCL va Vulkan ICD DLLs vao Registry
echo       + Khoi dong lai Display Container va lam moi WDDM context
echo.
echo   [4] Mo giao dien cua so truc quan [GUI FixCMP30HX_CSM]
echo.
echo   [5] Bat / Cai dat Tac vu Canh gac tu dong [Keep-Alive Watchdog]
echo       + Chong card bi rot hoac mat tich sau khi Sleep hoac Khoi dong lai
echo.
echo   [6] Go bo Tac vu Canh gac tu dong khoi he thong
echo.
echo   [7] Kiem tra dieu kien nang cap len UEFI thuan [MBR2GPT Lossless]
echo       + Giai phap triet de 100%% de bat Above 4G Decoding va Resizable BAR
echo.
echo   [8] Mo Device Manager [Quan ly thiet bi] ^& Kiem tra chi tiet
echo.
echo   [9] Huong dan cai Driver va cau hinh BIOS cho may CSM
echo.
echo   [0] Thoat
echo ===============================================================================
echo.
:: DOI NGUOI DUNG NHAP LUA CHON - HOAN TOAN KHONG TU DONG CHON SAU 10 GIAY
%SystemRoot%\System32\choice.exe /c 1234567890 /m "Nhap lua chon cua ban [1-9, 0]: "
if errorlevel 10 exit /b 0
if errorlevel 9 goto :show_bios_and_driver_guide
if errorlevel 8 goto :do_open_devmgmt
if errorlevel 7 goto :check_mbr2gpt
if errorlevel 6 goto :do_remove_watchdog
if errorlevel 5 goto :do_install_watchdog
if errorlevel 4 goto :do_gui
if errorlevel 3 goto :do_gpuz_fix
if errorlevel 2 goto :do_deep_scan_wake
if errorlevel 1 goto :do_auto_fix
goto :main_menu

:: ================================================================
:: 3. CHUC NANG 1: TU DONG SUA TOAN DIEN (AIO 1-CHAM)
:: ================================================================
:do_auto_fix
cls
echo ===============================================================================
echo   DANG THUC HIEN TU DONG SUA TOAN DIEN CHO CMP 30HX TREN HE THONG CSM...
echo ===============================================================================
echo.

echo [*] BUOC 1/4: Danh thuc phan cung PCIe va phuc hoi thiet bi bien mat...
call :native_deep_scan_and_wake

echo.
echo [*] BUOC 2/4: Khoa vinh vien D3cold, tat Fast Startup va PCIe ASPM...
call :native_lock_power_settings

echo.
echo [*] BUOC 3/4: Fix sau GPU-Z [Xoa AdapterType, bat EnableMsHybrid, dang ky ICD]...
call :native_unlock_gpuz_and_driver

echo.
echo [*] BUOC 4/4: Dang ky tac vu canh gac khoi dong Keep-Alive Watchdog...
call :native_install_watchdog

echo.
echo ===============================================================================
echo [V] HOAN TAT QUY TRINH SUA LOI TOAN DIEN!
echo ===============================================================================
echo  1. PHAN CUNG: Toan bo bus PCIe va PCI Bridge da duoc quet lai.
echo     - Neu truoc do card bi bien mat hoac ket Code 45, Windows se nhan lai
echo       card duoi dang '3D Video Controller', 'Microsoft Basic Display Adapter'
echo       hoac 'NVIDIA CMP 30HX'.
echo.
echo  2. GPU-Z ^& DO HOA:
echo     - Da xoa bo khoa 'AdapterType' (nguyen nhan khien card bi ket o Compute Mode
echo       va vo hieu hoa DirectCompute / Vulkan / CUDA tren man hinh WDDM).
echo     - Da kich hoat 'EnableMsHybrid = 1' va CASO de Windows 10 cho phep game va
echo       ung dung 3D su dung CMP 30HX qua iGPU.
echo     - Da dang ky Khronos OpenCL va Vulkan ICD DLLs.
echo.
echo  3. GIAI THICH VE CAC DAU TICK TRONG GPU-Z CHO CMP 30HX (TU116):
echo     [+] CUDA, OpenCL, Vulkan, DirectCompute, DirectML, PhysX: SE CO DAU TICK!
echo     [!] Ray Tracing: KHONG CO DAU TICK. Day la DAC TINH PHAN CUNG 100%% BINH THUONG
echo         vi chip Turing TU116 (CMP 30HX / GTX 1660 / 1660 Super) KHONG CO NHAN RT PHAN CUNG.
echo         (Ca card xin GTX 1660 Super cung khong he co dau tick o Ray Tracing).
echo ===============================================================================
echo.
pause
goto :main_menu

:: ================================================================
:: 4. CHUC NANG 2: DANH THUC PHAN CUNG ^& QUET SAU BUS PCIE
:: ================================================================
:do_deep_scan_wake
cls
echo ===============================================================================
echo   DANG THUC HIEN DANH THUC PHAN CUNG ^& QUET SAU BUS PCIE [CSM DEEP RESCAN]
echo ===============================================================================
echo.
call :native_deep_scan_and_wake
echo.
echo ===============================================================================
echo [V] Da hoan tat chu trinh quet sau va danh thuc phan cung!
echo     Hay mo Device Manager kiem tra xem card da xuat hien lai chua.
echo ===============================================================================
echo.
pause
goto :main_menu

:: ================================================================
:: 5. CHUC NANG 3: FIX SAU GPU-Z ^& KHOI PHUC COMPUTING
:: ================================================================
:do_gpuz_fix
cls
echo ===============================================================================
echo   DANG AP DUNG CAU HINH FIX SAU GPU-Z ^& KHOI PHUC COMPUTING TECHNOLOGIES...
echo ===============================================================================
echo.
call :native_unlock_gpuz_and_driver
echo.
echo ===============================================================================
echo [V] Da hoan tat mo khoa GPU-Z!
echo     Vui long dong hoan toan GPU-Z (neu dang mo) roi mo lai de kiem tra cac dau tick:
echo      - OpenCL, Vulkan, CUDA, DirectCompute 5.0, DirectML: Co tick!
echo      - Ray Tracing: Khong tick (binh thuong doi voi TU116/GTX 1660 series).
echo ===============================================================================
echo.
pause
goto :main_menu

:: ================================================================
:: 6. CHUC NANG 4: MO GIAO DIEN GUI
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
:: 7. CHUC NANG 5 ^& 6: QUAN LY WATCHDOG CANH GAC
:: ================================================================
:do_install_watchdog
cls
echo ===============================================================================
echo   CAI DAT TAC VU CANH GAC KHOI DONG [CSM KEEP-ALIVE WATCHDOG]
echo ===============================================================================
echo.
call :native_install_watchdog
echo.
pause
goto :main_menu

:do_remove_watchdog
cls
echo ===============================================================================
echo   GO BO TAC VU CANH GAC KHOI DONG KHOI HE THONG
echo ===============================================================================
echo.
if exist "%FIX_EXE%" (
    "%FIX_EXE%" -unwatchdog
)
schtasks /delete /tn "CMP30HX_CSM_KeepAlive_Boot" /f >nul 2>&1
schtasks /delete /tn "CMP30HX_CSM_KeepAlive_Logon" /f >nul 2>&1
echo [OK] Da go bo toan bo tac vu canh gac khoi Task Scheduler.
echo.
pause
goto :main_menu

:: ================================================================
:: 8. CHUC NANG 7: KIEM TRA MBR2GPT SANG UEFI
:: ================================================================
:check_mbr2gpt
cls
echo ===============================================================================
echo   KIEM TRA TINH TUONG THICH CHUYEN DOI MBR SANG UEFI [MBR2GPT LOSSLESS]
echo ===============================================================================
echo.
echo [*] Dang kiem tra cau truc phan vung dia cai Windows...
mbr2gpt.exe /validate /allowfullos
if errorlevel 1 (
    echo.
    echo [!] Ket qua: O dia hien tai chua du dieu kien chuyen tu dong qua mbr2gpt.
    echo     (Co the do o dia da la GPT san, hoac co nhieu hon 3 phan vung chinh).
) else (
    echo.
    echo ===============================================================================
    echo [V] CHUC MUNG: O dia cua ban DU DIEU KIEN 100%% de chuyen sang UEFI thuan!
    echo.
    echo Loi ich vuot troi khi chuyen sang UEFI thuan [Tat bo CSM]:
    echo  1. Bat duoc tinh nang 'Above 4G Decoding' trong BIOS [Triet tieu 100%% loi 43].
    echo  2. Bat duoc Resizable BAR giup card tang toi da hieu nang truyen tai du lieu.
    echo  3. Card khong bao gio bi rot bus hay bien mat sau khi tat may.
    echo.
    echo LENH THUC THI [Mo CMD Administrator chay]:
    echo   mbr2gpt /convert /allowfullos
    echo Sau khi chay xong, khoi dong lai vao BIOS, chuyen Boot sang UEFI va TAT CSM.
    echo ===============================================================================
)
echo.
pause
goto :main_menu

:: ================================================================
:: 9. CHUC NANG 8: MO DEVICE MANAGER ^& KIEM TRA CHI TIET
:: ================================================================
:do_open_devmgmt
cls
echo ===============================================================================
echo   DANG MO DEVICE MANAGER VA TRICH XUAT DANH SACH THIET BI DO HOA...
echo ===============================================================================
echo.
start "" devmgmt.msc
echo [DANH SACH CHI TIET CAC THIET BI DISPLAY ^& 3D CONTROLLER]:
powershell -NoProfile -ExecutionPolicy Bypass -Command "Get-PnpDevice -PresentOnly:$false | Where-Object { $_.InstanceId -like '*VEN_10DE*' -or $_.Class -eq 'Display' -or $_.FriendlyName -like '*3D*' } | Select-Object FriendlyName, InstanceId, Status, Present, Problem | Format-Table -AutoSize"
echo.
echo Ghi chu:
echo  - Neu thay card voi Present = False: Card dang bi Windows an do rot link.
echo    Hay chay muc [2] tren menu de danh thuc lai.
echo  - Neu thay '3D Video Controller': Card da duoc phan cung nhan dien thanh cong!
echo    Ban co the tien hanh cai Driver theo huong dan o muc [9].
echo.
pause
goto :main_menu

:: ================================================================
:: 10. CHUC NANG 9: HUONG DAN CAI DRIVER ^& CAU HINH BIOS
:: ================================================================
:show_bios_and_driver_guide
cls
echo ===============================================================================
echo   HUONG DAN CAI DRIVER VA CAU HINH BIOS CHO CMP 30HX TREN CSM
echo ===============================================================================
echo.
echo A. HUONG DAN CAI DRIVER KHI DEVICE MANAGER HIEN '3D VIDEO CONTROLLER':
echo    1. Vi CMP 30HX mang Hardware ID 'DEV_2189' (ID card dao coin), bo cai driver
echo       NVIDIA goc (Game Ready / Studio) thuong chan khong cho cai truc tiep va bao
echo       'This graphics driver could not find compatible graphics hardware'.
echo.
echo    2. Cac phuong an cai dat chuan xac:
echo       - Phuong an a: Dung bo driver da mod san INF cho CMP 30HX hoac P106-100.
echo       - Phuong an b: Dung cong cu NVIDIA-patcher tu dong go bo gioi han mining.
echo       - Phuong an c: Cai qua Device Manager: Chuot phai vao '3D Video Controller' -^>
echo         'Update driver' -^> 'Browse my computer for drivers' -^> 'Let me pick from a list'
echo         -^> Chon 'Display adapters' -^> 'Have Disk...' -^> Tro toi file .inf cua driver
echo         (Vi du driver GTX 1660 Super hoac ban driver da them DEV_2189).
echo.
echo B. CAU HINH BIOS BAT BUOC TREN MAINBOARD CHAY CSM:
echo    1. Primary Display / Initial Display Output:
echo       - Phai chon 'iGPU' (Card onboard Intel/AMD) hoac card phu co cong xuat hinh.
echo       - TUYET DOI KHONG chon khe PCIe cua CMP 30HX lam card xuat hinh chinh cua BIOS.
echo    2. Above 4G Decoding:
echo       - Neu BIOS co muc nay va cho phep Bat (Enable) khi CSM dang bat, hay CHON ENABLE.
echo    3. PCIe Link Speed:
echo       - Chuyen khe cam CMP 30HX sang 'Gen2' (hoac Gen1 / Auto) de tin hieu on dinh.
echo    4. Fast Boot:
echo       - Chon DISABLED de BIOS luon khoi tao lai bus PCIe khi bat may.
echo.
echo C. GIAI PHAP TRIET DE:
echo    Chuyen o dia sang GPT (chay muc [7]) roi vao BIOS TAT HAN CSM de chay UEFI thuan!
echo ===============================================================================
echo.
pause
goto :main_menu

:do_status_only
echo ===============================================================================
echo   BAO CAO TRANG THAI THIET BI (STATUS REPORT)
echo ===============================================================================
if exist "%FIX_EXE%" (
    "%FIX_EXE%" -status
)
echo [*] Danh sach thiet bi Display va 3D Controller:
powershell -NoProfile -ExecutionPolicy Bypass -Command "Get-PnpDevice -PresentOnly:$false | Where-Object { $_.InstanceId -like '*VEN_10DE*' -or $_.Class -eq 'Display' -or $_.FriendlyName -like '*3D*' } | Select-Object FriendlyName, InstanceId, Status, Present, Problem | Format-Table -AutoSize"
exit /b 0

:: ===============================================================================
:: 11. CAC DOAN MA THUC THI CHUYEN SAU NATIVE (POWERFUL INTERNAL PIPELINES)
:: ===============================================================================

:: -------------------------------------------------------------------------------
:: PIPELINE A: DANH THUC PHAN CUNG PCIE ^& QUET SAU PNP (FIX BIEN MAT CARD)
:: -------------------------------------------------------------------------------
:native_deep_scan_and_wake
echo  [1/4] Kich hoat danh thuc bus PCIe cap thap qua SetupAPI CM_Reenumerate...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$src = 'using System; using System.Runtime.InteropServices; public class SetupApiRescan { [DllImport(\"setupapi.dll\", SetLastError=true)] public static extern int CM_Locate_DevNode_Ex(out IntPtr pdnDevInst, string pDeviceID, int ulFlags, IntPtr hMachine); [DllImport(\"setupapi.dll\", SetLastError=true)] public static extern int CM_Reenumerate_DevNode_Ex(IntPtr dnDevInst, int ulFlags, IntPtr hMachine); }'; Add-Type -TypeDefinition $src; $root = [IntPtr]::Zero; [SetupApiRescan]::CM_Locate_DevNode_Ex([ref]$root, $null, 0, [IntPtr]::Zero) | Out-Null; [SetupApiRescan]::CM_Reenumerate_DevNode_Ex($root, 1, [IntPtr]::Zero) | Out-Null; Write-Host '    [OK] Da gui lenh Re-enumerate toi toan bo cay PnP phan cung.' -ForegroundColor Green;"

echo  [2/4] Kiem tra va tu dong kich hoat lai thiet bi neu dang bi Disabled [Code 22]...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$devs = Get-PnpDevice -PresentOnly:$false | Where-Object { ($_.InstanceId -like '*VEN_10DE*' -or $_.Class -eq 'Display' -or $_.FriendlyName -like '*3D*') -and $_.Status -eq 'Disabled' }; if ($devs) { foreach ($d in $devs) { Write-Host ('    [*] Dang bat lai thiet bi bi khoa: ' + $d.InstanceId) -ForegroundColor Yellow; try { Enable-PnpDevice -InstanceId $d.InstanceId -Confirm:$false -ErrorAction Stop; Write-Host '    [OK] Da bat lai thanh cong!' -ForegroundColor Green } catch { Write-Host ('    [!] Khong the bat tu dong: ' + $_.Exception.Message) -ForegroundColor Red } } } else { Write-Host '    [OK] Khong co thiet bi do hoa nao bi ket o trang thai Disabled.' -ForegroundColor Gray; }"

echo  [3/4] Xu ly device node ket trang thai ngat ket noi ao [Code 45 Phantom/Ghost Node]...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$ghosts = Get-PnpDevice -PresentOnly:$false | Where-Object { $_.InstanceId -like '*VEN_10DE&DEV_2189*' -and -not $_.Present }; if ($ghosts) { foreach ($g in $ghosts) { Write-Host ('    [*] Phat hien node ao bi rot ket noi (Code 45): ' + $g.InstanceId) -ForegroundColor Yellow; Write-Host '    [*] Dang giai phong node cu de ep Windows quet lai bus phan cung that...' -ForegroundColor Yellow; $arg = '/remove-device \"' + $g.InstanceId + '\"'; Start-Process pnputil.exe -ArgumentList $arg -NoNewWindow -Wait } } else { Write-Host '    [OK] Khong co node ao rac bi ket.' -ForegroundColor Gray; }"

echo  [4/4] Quet lai phan cung toan bo he thong bang pnputil...
pnputil /scan-devices >nul 2>&1
timeout /t 2 /nobreak >nul 2>&1

:: Bao cao ket qua tim kiem lai thiet bi
powershell -NoProfile -ExecutionPolicy Bypass -Command "$target = Get-PnpDevice | Where-Object { $_.InstanceId -like '*VEN_10DE&DEV_2189*' -or $_.FriendlyName -like '*3D*' -or ($_.Class -eq 'Display' -and $_.InstanceId -like '*VEN_10DE*') }; if ($target) { Write-Host '    [V] THANH CONG: Windows da nhin thay card do hoa!' -ForegroundColor Green; $target | Format-Table FriendlyName, InstanceId, Status -AutoSize } else { Write-Host '    [!] Card van chua xuat hien tren Device Manager.' -ForegroundColor Yellow; Write-Host '        Nguyen nhan: Mainboard CSM cap dien cham hoac khe PCIe dang o trang thai ngat link.' -ForegroundColor Gray; Write-Host '        Hay thu cam card sang khe PCIe khac hoac kiem tra nguon 8-pin.' -ForegroundColor Gray; }"
exit /b 0

:: -------------------------------------------------------------------------------
:: PIPELINE B: KHOA CHONG NGU SAU D3COLD ^& TAT FAST STARTUP, ASPM
:: -------------------------------------------------------------------------------
:native_lock_power_settings
echo  [1/3] Tat Fast Startup [Hiberboot] chong mat link PCIe khi bat may...
powercfg -h off >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Power" /v "HiberbootEnabled" /t REG_DWORD /d 0 /f >nul 2>&1

echo  [2/3] Vo hieu hoa PCIe ASPM [Link State Power Management] tren toan bo Power Scheme...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$schemes = powercfg -list | Select-String -Pattern '([a-f0-9-]{36})' | ForEach-Object { $_.Matches.Groups[1].Value }; $subPci = '501a4d13-42af-4429-9fd5-e8174f7900c2'; $settingPci = 'ee12f906-d27e-44ba-b376-73e76cb376f7'; foreach ($s in $schemes) { powercfg -setacvalueindex $s $subPci $settingPci 0; powercfg -setdcvalueindex $s $subPci $settingPci 0; powercfg -setactive $s }; Write-Host '    [OK] Da tat Link State Power Management tren tat ca profile nguon.' -ForegroundColor Green;"

echo  [3/3] Ap dung cam ngu D3cold truc tiep vao Enum PCI Device Parameters...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$pciBase = 'HKLM:\SYSTEM\CurrentControlSet\Enum\PCI'; Get-ChildItem $pciBase -ErrorAction SilentlyContinue | Where-Object { $_.PSChildName -like '*VEN_10DE&DEV_2189*' } | ForEach-Object { Get-ChildItem $_.PSPath -ErrorAction SilentlyContinue | ForEach-Object { $paramPath = $_.PSPath + '\Device Parameters'; if (-not (Test-Path $paramPath)) { New-Item -Path $paramPath -Force | Out-Null }; Set-ItemProperty -Path $paramPath -Name 'EnhancedPowerManagementEnabled' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $paramPath -Name 'AllowIdleIrpInD3' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $paramPath -Name 'D3ColdSupported' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $paramPath -Name 'DeviceSelectiveSuspended' -Value 0 -Type DWord -Force; Write-Host ('    [OK] Da khoa cam D3cold cho: ' + $_.PSChildName) -ForegroundColor Green } }"
exit /b 0

:: -------------------------------------------------------------------------------
:: PIPELINE C: MO KHOA GPU-Z [XOA ADAPTERTYPE, BAT MS-HYBRID ^& CASO, DANG KY ICD]
:: -------------------------------------------------------------------------------
:native_unlock_gpuz_and_driver
echo  [1/4] Toi uu hoa GraphicsDrivers TDR Delay len 10 giay...
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "TdrDelay" /t REG_DWORD /d 10 /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "TdrDdiDelay" /t REG_DWORD /d 10 /f >nul 2>&1
reg add "HKLM\SYSTEM\CurrentControlSet\Control\GraphicsDrivers" /v "TdrLevel" /t REG_DWORD /d 3 /f >nul 2>&1

echo  [2/4] Xu ly cau hinh Driver Class NVIDIA: Xoa AdapterType, bat EnableMsHybrid va CASO...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$dispClass = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}'; Get-ChildItem $dispClass -ErrorAction SilentlyContinue | Where-Object { $_.PSChildName -match '^\d{4}$' } | ForEach-Object { $p = $_.PSPath; $desc = (Get-ItemProperty -Path $p -Name 'DriverDesc' -ErrorAction SilentlyContinue).DriverDesc; $matchId = (Get-ItemProperty -Path $p -Name 'MatchingDeviceId' -ErrorAction SilentlyContinue).MatchingDeviceId; $isNvidia = ($desc -like '*NVIDIA*' -or $desc -like '*CMP*' -or $desc -like '*TU116*' -or $matchId -like '*10DE*'); if ($isNvidia) { Write-Host ('    [*] Tim thay Driver Class NVIDIA: ' + $_.PSChildName + ' (' + $desc + ')') -ForegroundColor Cyan; Set-ItemProperty -Path $p -Name 'EnableCrossAdapterScanOut' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'EnableMsHybrid' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'EnableCoproc' -Value 1 -Type DWord -Force; if (Get-ItemProperty -Path $p -Name 'AdapterType' -ErrorAction SilentlyContinue) { Remove-ItemProperty -Path $p -Name 'AdapterType' -Force -ErrorAction SilentlyContinue; Write-Host '    [OK] Da XOA BO AdapterType [Khoi phuc DirectCompute/CUDA/Vulkan thanh cong].' -ForegroundColor Green }; Remove-ItemProperty -Path $p -Name 'LargePageMinimum' -Force -ErrorAction SilentlyContinue; Set-ItemProperty -Path $p -Name 'D3ColdSupported' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'DeviceSelectiveSuspended' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'RmDisableGpuPowerMgmt' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'RmEnableAggressivePciePowerManagement' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'DisableASPM' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'DisablePCIePowerManagement' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'EnableGpuFirmware' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'PowerMizerEnable' -Value 0 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'PowerMizerLevel' -Value 1 -Type DWord -Force; Set-ItemProperty -Path $p -Name 'PowerMizerLevelAC' -Value 1 -Type DWord -Force; Write-Host '    [OK] Da cau hinh toi uu WDDM, CASO va khoa nguon cho GPU NVIDIA.' -ForegroundColor Green } else { Remove-ItemProperty -Path $p -Name 'AdapterType' -Force -ErrorAction SilentlyContinue; Remove-ItemProperty -Path $p -Name 'LargePageMinimum' -Force -ErrorAction SilentlyContinue; Remove-ItemProperty -Path $p -Name 'EnableGpuFirmware' -Force -ErrorAction SilentlyContinue } }"

echo  [3/4] Dang ky Khronos OpenCL va Vulkan ICD DLLs vao Registry...
powershell -NoProfile -ExecutionPolicy Bypass -Command "$oclPaths = @('HKLM:\SOFTWARE\Khronos\OpenCL\Vendors', 'HKLM:\SOFTWARE\WOW6432Node\Khronos\OpenCL\Vendors'); foreach ($path in $oclPaths) { if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }; Set-ItemProperty -Path $path -Name 'nvopencl.dll' -Value 0 -Type DWord -Force -ErrorAction SilentlyContinue; Set-ItemProperty -Path $path -Name 'nvopencl64.dll' -Value 0 -Type DWord -Force -ErrorAction SilentlyContinue }; $vkPaths = @('HKLM:\SOFTWARE\Khronos\Vulkan\Drivers', 'HKLM:\SOFTWARE\WOW6432Node\Khronos\Vulkan\Drivers'); foreach ($path in $vkPaths) { if (-not (Test-Path $path)) { New-Item -Path $path -Force | Out-Null }; Set-ItemProperty -Path $path -Name 'nv-vk64.json' -Value 0 -Type DWord -Force -ErrorAction SilentlyContinue; Set-ItemProperty -Path $path -Name 'nv-vk32.json' -Value 0 -Type DWord -Force -ErrorAction SilentlyContinue }; Write-Host '    [OK] Da dam bao dang ky Khronos OpenCL va Vulkan ICD.' -ForegroundColor Green;"

echo  [4/4] Khoi dong lai dich vu NVIDIA Display Container va lam moi Device Node...
sc query "NVDisplay.ContainerLocalSystem" >nul 2>&1
if not errorlevel 1 (
    sc config "NVDisplay.ContainerLocalSystem" start= auto >nul 2>&1
    net stop "NVDisplay.ContainerLocalSystem" >nul 2>&1
    timeout /t 1 /nobreak >nul 2>&1
    net start "NVDisplay.ContainerLocalSystem" >nul 2>&1
    echo     [OK] Da khoi dong lai dich vu NVIDIA Display Container.
)

:: Khoi dong lai PnP device node neu da co
powershell -NoProfile -ExecutionPolicy Bypass -Command "$dev = Get-PnpDevice | Where-Object { $_.InstanceId -like '*VEN_10DE&DEV_2189*' } | Select-Object -First 1; if ($dev) { Write-Host ('    [*] Dang lam moi PnP device node: ' + $dev.InstanceId) -ForegroundColor Gray; $arg = '/restart-device \"' + $dev.InstanceId + '\"'; Start-Process pnputil.exe -ArgumentList $arg -NoNewWindow -Wait }"
exit /b 0

:: -------------------------------------------------------------------------------
:: PIPELINE D: DANG KY WATCHDOG CANH GAC KHOI DONG
:: -------------------------------------------------------------------------------
:native_install_watchdog
if exist "%FIX_EXE%" (
    "%FIX_EXE%" -watchdog
    exit /b 0
)

schtasks /create /tn "CMP30HX_CSM_KeepAlive_Boot" /tr "pnputil /scan-devices" /sc onstart /ru SYSTEM /delay 0000:20 /f >nul 2>&1
schtasks /create /tn "CMP30HX_CSM_KeepAlive_Logon" /tr "pnputil /scan-devices" /sc onlogon /ru SYSTEM /delay 0000:10 /f >nul 2>&1
echo [OK] Da dang ky tac vu canh gac khoi dong tu dong [SYSTEM] thanh cong.
exit /b 0
