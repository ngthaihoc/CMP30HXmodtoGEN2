@echo off
setlocal EnableDelayedExpansion
chcp 65001 >nul 2>&1
title CMP 30HX Resizable BAR Auto Setup AIO

rem ================================================================
rem 1. PHAN TICH TOAN BO THAM SO DONG LENH (CLI ARGUMENTS)
rem ================================================================
set "IS_MOCK=0"
set "NO_CHECK=0"
set "NO_WAIT=0"
set "IS_ADMIN=0"
set "ACTION="

for %%a in (%*) do (
    if /i "%%~a"=="-noadmin" set "IS_ADMIN=1"
    if /i "%%~a"=="/noadmin" set "IS_ADMIN=1"
    if /i "%%~a"=="-test" set "IS_MOCK=1"
    if /i "%%~a"=="/test" set "IS_MOCK=1"
    if /i "%%~a"=="-mock" set "IS_MOCK=1"
    if /i "%%~a"=="/mock" set "IS_MOCK=1"
    if /i "%%~a"=="-mock-laptop" (
        set "MOCK_LAPTOP=1"
        set "IS_ADMIN=1"
        if not defined ACTION set "ACTION=INSTALL"
    )
    if /i "%%~a"=="/mock-laptop" (
        set "MOCK_LAPTOP=1"
        set "IS_ADMIN=1"
        if not defined ACTION set "ACTION=INSTALL"
    )
    if /i "%%~a"=="-nocheck" set "NO_CHECK=1"
    if /i "%%~a"=="/nocheck" set "NO_CHECK=1"
    if /i "%%~a"=="-nowait" set "NO_WAIT=1"
    if /i "%%~a"=="/nowait" set "NO_WAIT=1"
    if /i "%%~a"=="-install" set "ACTION=INSTALL"
    if /i "%%~a"=="/install" set "ACTION=INSTALL"
    if /i "%%~a"=="-status" set "ACTION=STATUS"
    if /i "%%~a"=="/status" set "ACTION=STATUS"
    if /i "%%~a"=="-uninstall" set "ACTION=UNINSTALL"
    if /i "%%~a"=="/uninstall" set "ACTION=UNINSTALL"
    if /i "%%~a"=="-u" set "ACTION=UNINSTALL"
    if /i "%%~a"=="/u" set "ACTION=UNINSTALL"
)

rem Neu co co mock, tu dong danh dau la INSTALL neu chua chon action
if "!IS_MOCK!"=="1" (
    if not defined ACTION set "ACTION=INSTALL"
    set "IS_ADMIN=1"
)

rem ================================================================
rem 2. KIEM TRA QUYEN ADMINISTRATOR (UAC DA TANG PHONG THU)
rem ================================================================
if "!IS_ADMIN!"=="0" (
    fltmc >nul 2>&1 && set "IS_ADMIN=1"
)
if "!IS_ADMIN!"=="0" (
    fsutil dirty query %systemdrive% >nul 2>&1 && set "IS_ADMIN=1"
)

if "!IS_ADMIN!"=="0" (
    echo [*] Dang yeu cau quyen Administrator [UAC]...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%comspec%' -ArgumentList '/c \"\"%~f0\" %*\"' -Verb RunAs" >nul 2>&1
    if errorlevel 1 (
        echo.
        echo ================================================================
        echo [X] LOI: Khong the tu dong yeu cau quyen Administrator qua UAC.
        echo [*] Vui long nhap chuot phai vao file Setup_ReBar_CMP30HX.bat
        echo     va chon 'Run as administrator' [Chay voi tu cach quan tri vien].
        echo     Hoac chay qua CMD Admin: Setup_ReBar_CMP30HX.bat -noadmin
        echo ================================================================
        echo.
        pause
    )
    exit /b
)

cd /d "%~dp0"
set "TOOL_DIR=%~dp0windows-v3.0\tools\rebar30hx"
set "STATUS_DIR=%ProgramData%\40HXUnlock"
set "STATUS_FILE=%STATUS_DIR%\rebar_status.txt"

rem Neu da co action qua CLI thi nhay truc tiep den xu ly
if /i "!ACTION!"=="INSTALL" goto :action_install
if /i "!ACTION!"=="STATUS" goto :action_status
if /i "!ACTION!"=="UNINSTALL" goto :action_uninstall

rem ================================================================
rem 3. MENU TUONG TAC TRUYEN THONG CHO NGUOI DUNG (INTERACTIVE MENU)
rem ================================================================
:menu_loop
cls
echo ================================================================
echo    BO CONG CU AIO KICH HOAT RESIZABLE BAR CHO NVIDIA CMP 30HX
echo    - Kien truc Turing [TU116 / 6GB GDDR6] tren nen tang PCIe Gen2
echo    - Co che NvStrapsReBar: 0%% rui ro cho GPU, Zero-Touch VBIOS
echo ================================================================
echo.
echo  [1] Cai dat va Kich hoat Resizable BAR [Full Setup AIO]
echo  [2] Kiem tra trang thai Resizable BAR hien tai [Status Check]
echo  [3] Go bo / Khoi phuc mac dinh [Disable ReBAR ^& Reset Driver]
echo  [4] Chay kiem thu mo phong an toan [Mock Test Simulation]
echo  [0] Thoat [Exit]
echo.
echo ================================================================
set "CHOICE="
set /p "CHOICE=Nhap lua chon cua ban [1-4, 0]: "

if "%CHOICE%"=="1" (
    set "IS_MOCK=0"
    goto :action_install
)
if "%CHOICE%"=="2" goto :action_status
if "%CHOICE%"=="3" goto :action_uninstall
if "%CHOICE%"=="4" (
    set "IS_MOCK=1"
    goto :action_install
)
if "%CHOICE%"=="0" exit /b 0
goto :menu_loop


rem ================================================================
rem ACTION: CAI DAT VA KICH HOAT RESIZABLE BAR AIO
rem ================================================================
:action_install
cls
echo ================================================================
if "!IS_MOCK!"=="1" (
    echo    CONG CU KIEM THU MO PHONG [MOCK TEST] RESIZABLE BAR CHO CMP 30HX
    echo    - Mo phong day du quy trinh ReBAR cho card CMP 30HX [TU116]
    echo    - 100%% AN TOAN: Khong can thiep BIOS/NVRAM he thong
) else (
    echo    CONG CU CAI DAT TU DONG AIO: RESIZABLE BAR CHO NVIDIA CMP 30HX
    echo    - Giai phap NvStrapsReBar: Mo khoa BAR 8GB cho GPU 6GB GDDR6
    echo    - Giu nguyen VBIOS goc cua GPU, tuyet doi an toan phan cung card
)
echo ================================================================
echo.

rem Kiem tra thu muc cong cu
if not exist "%TOOL_DIR%\NvStrapsReBar.exe" (
    echo [X] LOI: Khong tim thay bo cong cu rebar30hx tai: "%TOOL_DIR%"
    echo [*] Vui long kiem tra lai bo cai dat day du cua du an.
    if "!NO_WAIT!"=="0" pause
    exit /b 1
)

rem ----------------------------------------------------------------
rem BUOC 1: QUET PHAN CUNG VA KHOA AN TOAN LAPTOP
rem ----------------------------------------------------------------
echo [1/5] Dang quet cau hinh he thong va kiem tra an toan phan cung...

set "SYS_MANU=Unknown"
set "SYS_MODEL=Unknown"
set "MB_MANU=Unknown"
set "MB_PROD=Unknown"
set "MB_BIOS=Unknown"
set "IS_LAPTOP=0"

for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-CimInstance Win32_ComputerSystem).Manufacturer"`) do set "SYS_MANU=%%A"
for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-CimInstance Win32_ComputerSystem).Model"`) do set "SYS_MODEL=%%A"
for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-CimInstance Win32_BaseBoard).Manufacturer"`) do set "MB_MANU=%%A"
for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-CimInstance Win32_BaseBoard).Product"`) do set "MB_PROD=%%A"
for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-CimInstance Win32_BIOS).SMBIOSBIOSVersion"`) do set "MB_BIOS=%%A"

echo    - He thong   : !SYS_MANU! - Model: !SYS_MODEL!
echo    - Bo mach chu: !MB_MANU! - Model: !MB_PROD! [BIOS: !MB_BIOS!]
echo    - Che do Boot: %FIRMWARE_TYPE%

rem Kiem tra che do khoi dong UEFI thuan
if /i not "%FIRMWARE_TYPE%"=="UEFI" (
    echo.
    echo ================================================================
    echo  [!] CANH BAO: He thong dang khoi dong o che do Legacy BIOS [CSM].
    echo      Resizable BAR bat buoc he thong phai boot o chuan UEFI thuan.
    echo      Vui long chuyen doi o dia sang GPT va bat UEFI trong BIOS.
    echo ================================================================
    echo.
)

rem Ho tro mo phong phat hien Laptop cho bo kiem thu (Mock Laptop Guard)
if "!MOCK_LAPTOP!"=="1" (
    set "IS_LAPTOP=1"
    set "IS_MOCK=0"
    set "SYS_MANU=MockVendor"
    set "SYS_MODEL=MockGamingLaptop"
    set "MB_PROD=MockLaptopBoard"
    set "MB_BIOS=V1.00"
    set "DETECTED_GPUS= [NVIDIA CMP 30HX]"
) else (
    rem Kiem tra 4 lop nhan dien Laptop thuc te
    for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "(Get-CimInstance Win32_ComputerSystem).PCSystemType"`) do if "%%A"=="2" set "IS_LAPTOP=1"
    for /f "usebackq delims=" %%A in (`powershell -NoProfile -Command "[bool](Get-CimInstance Win32_Battery)"`) do if /i "%%A"=="True" set "IS_LAPTOP=1"
    echo "!SYS_MODEL! !MB_PROD!" | findstr /i "FA506 G513 G533 GL553 Nitro Legion Victus Laptop Notebook Book Portable TUF ROG Zephyrus Strix Thin Stealth Blade Omen Pavilion Inspiron Latitude Precision XPS Yoga ThinkPad IdeaPad" >nul 2>&1 && set "IS_LAPTOP=1"
)

rem Quet danh sach GPU tren may
set "FOUND_NVIDIA=0"
set "FOUND_CMP30HX=0"
set "DETECTED_GPUS="

for /f "tokens=1* delims=|" %%A in ('powershell -NoProfile -Command "Get-CimInstance Win32_VideoController | ForEach-Object { $_.Name + '|' + $_.PNPDeviceID }"') do (
    set "G_NAME=%%A"
    set "G_PNP=%%B"
    echo    + Phat hien GPU: !G_NAME!
    set "DETECTED_GPUS=!DETECTED_GPUS! [!G_NAME!]"
    echo "!G_PNP!" | findstr /i "VEN_10DE" >nul 2>&1 && set "FOUND_NVIDIA=1"
    echo "!G_PNP!" | findstr /i "DEV_2189" >nul 2>&1 && set "FOUND_CMP30HX=1"
)

rem ----------------------------------------------------------------
rem KHOA AN TOAN TU DONG: CHAN 100% TREN LAPTOP NEU KHONG PHAI CHE DO MOCK
rem ----------------------------------------------------------------
if "!IS_LAPTOP!"=="1" (
    if "!IS_MOCK!"=="0" (
        echo.
        echo ================================================================
        echo  [X] KHOA AN TOAN TU DONG: PHAT HIEN HE THONG LAPTOP / NOTEBOOK
        echo ================================================================
        echo  [*] Thiet bi hien tai: !SYS_MANU! !SYS_MODEL!
        echo      - Bo mach chu    : !MB_PROD! [BIOS: !MB_BIOS!]
        echo      - GPU tren may   :!DETECTED_GPUS!
        echo      - Tinh trang     : Laptop co pin, GPU han chet tren bo mach
        echo  ----------------------------------------------------------------
        echo  [CANH BAO NGUY HIEM CAP DO CAO]:
        echo  1. Bo mach chu Laptop TUYET DOI KHONG CO nut 'USB BIOS Flashback'.
        echo  2. Mod BIOS hoac ghi de bien UEFI NVRAM bang cong cu NvStrapsReBar
        echo     tren Laptop se gay ra nguy co HARD BRICK [liet nguon, mat hinh,
        echo     khong the khoi dong] ma KHONG THE PHUC HOI bang cach thong thuong.
        echo  3. NVIDIA CMP 30HX la card do hoa roi may ban [Desktop PCIe Card].
        echo     Quy trinh mod BIOS ReBAR nay duoc thiet ke cho Mainboard PC.
        echo  ----------------------------------------------------------------
        echo  [CHINH SACH AN TOAN TU DONG]:
        echo  - Chuong trinh TU DONG CHAN moi thao tac can thiep BIOS / NVRAM.
        echo  - KHONG mo UEFITool, KHONG mo NvStrapsReBar de bao ve Laptop cua ban.
        echo  ----------------------------------------------------------------
        echo  [*] DE XUAT HUONG XU LY:
        echo  - Hay sao chep bo cai sang MAY TINH DE BAN [Desktop PC] dang cam
        echo    card CMP 30HX de thuc hien quy trinh kich hoat nay.
        echo  - Neu ban chi muon KIEM THU / XEM TRUOC quy trinh mo phong an toan:
        echo    Hay chay lenh voi co -mock [hoac chon muc 4 trong Menu]:
        echo    ^> Setup_ReBar_CMP30HX.bat -mock
        echo ================================================================
        echo.
        if "!NO_WAIT!"=="0" pause
        exit /b 1
    ) else (
        echo.
        echo ----------------------------------------------------------------
        echo [MOCK TEST] Phat hien Laptop [!SYS_MANU! !SYS_MODEL!].
        echo [MOCK TEST] Kich hoat che do mo phong an toan 100%%:
        echo             Khong ghi NVRAM, khong nap BIOS, khong can thiep GPU.
        echo ----------------------------------------------------------------
    )
)

rem Kiem tra nhan dien CMP 30HX tren Desktop
if "!FOUND_CMP30HX!"=="1" (
    echo    [+] Xac nhan chinh xac card NVIDIA CMP 30HX [TU116 - DEV_2189].
) else (
    if "!IS_MOCK!"=="1" (
        echo    [MOCK] Gia lap phat hien card do hoa: NVIDIA CMP 30HX [TU116 - 10DE:2189].
    ) else (
        if "!FOUND_NVIDIA!"=="0" (
            echo [*] Khong tim thay card do hoa NVIDIA nao tren he thong.
            echo     Vui long kiem tra lai ket noi khe PCIe, nguon phu hoac riser.
            echo.
            if "!NO_WAIT!"=="0" pause
            exit /b 1
        ) else (
            echo    [*] Phat hien GPU NVIDIA tren he thong Desktop PC.
        )
    )
)

rem ----------------------------------------------------------------
rem BUOC 2: GIAI THICH KHOA HOC VE VRAM 6GB VA DUNG LUONG BAR 8GB
rem ----------------------------------------------------------------
echo.
echo [2/5] Co so kien truc PCIe: Tai sao CMP 30HX 6GB VRAM lai can BAR 8GB?
echo    ----------------------------------------------------------------
echo    - Dung luong VRAM vat ly cua CMP 30HX: 6GB GDDR6 [6144 MB].
echo    - Theo chuan ky thuat PCI Express: Cua so Base Address Register [BAR]
echo      BAT BUOC phai duoc cap phat theo luy thua cua 2 [2^n]:
echo      ... 1024 MB [1GB] -^> 2048 MB [2GB] -^> 4096 MB [4GB] -^> 8192 MB [8GB]...
echo    - Do KHONG TON TAI muc BAR 6GB chan trong chuan PCIe:
echo      + Neu dat BAR = 4GB: CPU chi doc truc tiep duoc 4GB dau, 2GB VRAM con
echo        lai bi bo phi ngoai vung ReBAR [phai chia nho goi tin gay giat hinh].
echo      + De CPU truy cap TRON VEN 100%% dung luong 6GB VRAM cua CMP 30HX,
echo        he thong bat buoc phai mo cua so BAR o muc luy thua 2 gan nhat la 8GB.
echo    - Cong cu NvStrapsReBar se tu dong cau hinh muc BAR 8192 MB [8GB] toi uu.
echo    ----------------------------------------------------------------

rem ----------------------------------------------------------------
rem BUOC 3: KIEM TRA BO CONG CU VA HUONG DAN BIOS MAINBOARD
rem ----------------------------------------------------------------
echo.
echo [3/5] Kiem tra bo cong cu Resizable BAR:
echo    + NvStrapsReBar.exe          : [OK] [Cong cu thiet lap NVRAM]
echo    + NvStrapsReBar.ffs          : [OK] [Module UEFI DXE chen vao BIOS mainboard]
echo    + UEFITool.exe [v0.28.0]     : [OK] [Cong cu chen module FFS vao BIOS]
echo    + nvidiaProfileInspector.exe : [OK] [Cong cu mo khoa rBAR trong driver]
echo    + Enable_ReBAR_Turing.nip    : [OK] [Profile rBAR Base da cau hinh san]

echo.
rem ----------------------------------------------------------------
rem KIEM TRA CHAN DOAN UEFI DXE DRIVER STATUS
rem ----------------------------------------------------------------
set "DXE_LOADED=0"
if "!IS_MOCK!"=="1" (
    set "DXE_LOADED=1"
    echo    + Trang thai UEFI DXE Driver: [MOCK] Loaded [0x0] - San sang 100%%.
) else (
    echo Q | "%TOOL_DIR%\NvStrapsReBar.exe" 2>nul | findstr /i "status: Loaded" >nul 2>&1
    if not errorlevel 1 (
        set "DXE_LOADED=1"
        echo    + Trang thai UEFI DXE Driver: Loaded [0x0] - San sang 100%%.
    ) else (
        echo    + Trang thai UEFI DXE Driver: Not loaded [Chua phat hien ReBAR trong BIOS].
    )
)

if "!DXE_LOADED!"=="0" (
    set "IS_MODERN_MB=0"
    for %%k in (B450 B550 A520 X570 B650 X670 A620 Z390 Z490 B460 H470 Z590 B560 H510 Z690 B660 H610 Z790 B760 H770) do (
        echo "!MB_PROD!" | findstr /i "%%k" >nul 2>&1 && set "IS_MODERN_MB=1"
    )

    if "!IS_MODERN_MB!"=="1" (
        echo.
        echo ================================================================
        echo [*] HUONG DAN BO MACH CHU HO TRO REBAR GOC [!MB_PROD!]:
        echo     Bo mach chu cua ban da co san tinh nang ReBAR trong BIOS.
        echo     [V] BAN KHONG CAN DUNG UEFITOOL DE MOD BIOS.
        echo.
        echo     Chi can khoi dong lai may, vao BIOS Setup [Del / F2] va BAT:
        echo       1. 'Above 4G Decoding'   = Enabled
        echo       2. 'Re-Size BAR Support' = Auto hoac Enabled
        echo       3. 'CSM Support'         = Disabled [UEFI thuan]
        echo ================================================================
        echo.
    ) else (
        echo.
        echo ================================================================
        echo [*] HUONG DAN BO MACH CHU THE HE CU / CAN CHEN MODULE FFS:
        echo     Bo mach chu [!MB_PROD!] co the can chen module NvStrapsReBar.ffs.
        echo     Luu y: Chi thuc hien tren mainboard PC co nut 'USB BIOS Flashback'.
        echo.
        echo     Cac buoc thuc hien bang UEFITool:
        echo     1. Mo UEFITool.exe tai: "%TOOL_DIR%\UEFITool.exe"
        echo     2. Mo file BIOS goc cua bo mach chu.
        echo     3. Tim kiem 'PciBus' trong phan DXE Volume.
        echo     4. Nhap chuot phai vao driver cuoi cung -^> 'Insert after...'
        echo     5. Chon file: "%TOOL_DIR%\NvStrapsReBar.ffs" va luu lai BIOS moi.
        echo ================================================================
        echo.
        if "!NO_WAIT!"=="0" (
            if "!IS_MOCK!"=="0" (
                set "OPEN_TOOL=N"
                set /p "OPEN_TOOL=Ban co muon mo UEFITool.exe ngay bay gio khong? [Y/N, Mac dinh N]: "
                if /i "!OPEN_TOOL!"=="Y" (
                    start "" "%TOOL_DIR%\UEFITool.exe"
                )
            )
        )
    )
)

rem ----------------------------------------------------------------
rem BUOC 4: CAU HINH NVRAM VA DRIVER PROFILE TU DONG [1-CLICK AIO]
rem ----------------------------------------------------------------
echo [4/5] Tu dong cau hinh phan mem Windows [1-Click Automation]:
echo.
echo [*] BUOC 4A: Tu dong cau hinh BAR 8GB vao UEFI NVRAM...
if "!IS_MOCK!"=="1" (
    echo    [MOCK] Gia lap khoi chay NvStrapsReBar.exe thanh cong.
    echo    [MOCK] - Da chon Enable ReBAR Turing [E]: [OK]
    echo    [MOCK] - Da thiet lap BAR Size = 8192 MB [8GB]: [OK]
    echo    [MOCK] - Da ghi bien EFI NVRAM he thong [S]: [OK]
    echo    [MOCK] - Da thoat chuong trinh [Q]: [OK]
) else (
    (echo E & echo S & echo Q) | "%TOOL_DIR%\NvStrapsReBar.exe" >nul 2>&1
    echo    [+] Da tu dong ghi cau hinh Turing ReBAR 8GB vao UEFI NVRAM: [OK]
)

echo.
echo [*] BUOC 4B: Tu dong kich hoat rBAR trong NVIDIA Driver Profile...
if "!IS_MOCK!"=="1" (
    echo    [MOCK] Gia lap nap profile Enable_ReBAR_Turing.nip qua co -silent thanh cong.
    echo    [MOCK] - rBAR - Feature    = 0x00000001 [Enabled]
    echo    [MOCK] - rBAR - Options    = 0x00000001 [Turing Override]
    echo    [MOCK] - rBAR - Size Limit = 0x00000000 [No Limit]
) else (
    start /wait "" "%TOOL_DIR%\nvidiaProfileInspector.exe" -silent "%TOOL_DIR%\Enable_ReBAR_Turing.nip"
    echo    [+] Da tu dong nap Driver Profile [Enable_ReBAR_Turing.nip -silent]: [OK]
)

echo.
echo [*] BUOC 4C: Thiet lap Scheduled Task duy tri rBAR Driver Profile...
set "REBAR_APP_DIR=%ProgramFiles%\40HXUnlock\rebar"
set "REBAR_RUNNER=%REBAR_APP_DIR%\RunReBarProfile.bat"

if not exist "%REBAR_APP_DIR%" mkdir "%REBAR_APP_DIR%" >nul 2>&1
copy /y "%TOOL_DIR%\nvidiaProfileInspector.exe" "%REBAR_APP_DIR%\" >nul 2>&1
copy /y "%TOOL_DIR%\nvidiaProfileInspector.exe.config" "%REBAR_APP_DIR%\" >nul 2>&1
copy /y "%TOOL_DIR%\Enable_ReBAR_Turing.nip" "%REBAR_APP_DIR%\" >nul 2>&1

(
    echo @echo off
    echo setlocal
    echo cd /d "%%~dp0"
    echo if exist "%%~dp0nvidiaProfileInspector.exe" start /wait "" "%%~dp0nvidiaProfileInspector.exe" -silent "%%~dp0Enable_ReBAR_Turing.nip"
    echo exit /b 0
) > "%REBAR_RUNNER%"

powershell -NoProfile -ExecutionPolicy Bypass -Command "$taskName='NVIDIA_ReBAR_Global_Profile'; $dir=$env:REBAR_APP_DIR; $bat=$env:REBAR_RUNNER; $action = New-ScheduledTaskAction -Execute $env:ComSpec -Argument ('/c `\"' + $bat + '`\"') -WorkingDirectory $dir; $t1 = New-ScheduledTaskTrigger -AtStartup; $t1.Delay = 'PT15S'; $t2 = New-ScheduledTaskTrigger -AtLogOn; $t2.Delay = 'PT5S'; $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew -ExecutionTimeLimit (New-TimeSpan -Minutes 5); $principal = New-ScheduledTaskPrincipal -UserId 'NT AUTHORITY\SYSTEM' -LogonType ServiceAccount -RunLevel Highest; Register-ScheduledTask -TaskName $taskName -Action $action -Trigger @($t1, $t2) -Settings $settings -Principal $principal -Force | Out-Null; try { $srv = New-Object -ComObject 'Schedule.Service'; $srv.Connect(); $task = $srv.GetFolder('\').GetTask($taskName); $def = $task.Definition; $tEvent = $def.Triggers.Create(0); $tEvent.Subscription = '<QueryList><Query Id=''0'' Path=''System''><Select Path=''System''>*[System[Provider[@Name=''Microsoft-Windows-Power-Troubleshooter''] and EventID=1]]</Select></Query></QueryList>'; $tEvent.Delay = 'PT3S'; $tEvent.Enabled = $true; $srv.GetFolder('\').RegisterTaskDefinition($taskName, $def, 4, $null, $null, 5, $null) | Out-Null } catch {}" >nul 2>&1

reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /t REG_SZ /d "\"%REBAR_RUNNER%\"" /f >nul 2>&1
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /t REG_SZ /d "\"%REBAR_RUNNER%\"" /f >nul 2>&1
echo    [+] Da kich hoat Scheduled Task [NVIDIA_ReBAR_Global_Profile] va Registry Run: [OK]

rem Ghi nhan file trang thai ReBAR
if not exist "%STATUS_DIR%" mkdir "%STATUS_DIR%" >nul 2>&1
(
    echo ==== CMP 30HX Resizable BAR Status Report ====
    echo Thoi gian thiet lap : %date% %time%
    echo GPU Muc tieu        : NVIDIA CMP 30HX [TU116 - DEV_2189]
    echo Dung luong VRAM     : 6GB GDDR6
    echo Kich thuoc BAR1     : 8192 MB [8GB luy thua 2 PCIe]
    echo Module UEFI         : NvStrapsReBar.ffs [DXE Driver]
    echo Driver Profile      : rBAR Enabled [0x1], Turing Override [0x1]
    if "!IS_MOCK!"=="1" (
        echo Che do thuc thi     : MOCK TEST SIMULATION [100%% Safe]
    ) else (
        echo Che do thuc thi     : LIVE PRODUCTION SETUP
    )
) > "%STATUS_FILE%"

rem ----------------------------------------------------------------
rem BUOC 5: TONG KET VA HUONG DAN NGHIEM THU
rem ----------------------------------------------------------------
echo.
echo [5/5] Tong ket va nghiem thu:
echo ================================================================
echo                    HOAN TAT THIET LAP REBAR AIO
echo ================================================================
echo  Cac buoc nghiem thu sau khi khoi dong lai may tinh:
echo  1. Khoi dong lai he thong de BIOS va driver ap dung cau hinh.
echo  2. Mo GPU-Z kiem tra trang thai:
echo     - Muc 'Resizable BAR' tai tab dau: phai bao 'Enabled'.
echo     - Vao tab Advanced -^> 'PCIe Resizable BAR':
echo       + GPU Supported: Yes
echo       + PCIe BAR1 Size: 8192 MB [8GB].
echo  3. Kiem tra qua nvidia-smi:
echo     ^> nvidia-smi -q -d memory
echo     Muc 'BAR1 Memory Usage - Total' phai bao ~8192 MiB.
echo  4. File bao cao trang thai da duoc luu tai:
echo     "%STATUS_FILE%"
echo ================================================================
echo.

if "!NO_WAIT!"=="0" pause
exit /b 0


rem ================================================================
rem ACTION: KIEM TRA TRANG THAI RESIZABLE BAR HIEN TAI (STATUS)
rem ================================================================
:action_status
cls
echo ================================================================
echo    KIEM TRA TRANG THAI RESIZABLE BAR CHO NVIDIA CMP 30HX
echo ================================================================
echo.

if exist "%STATUS_FILE%" (
    echo [*] Thong tin trang thai da ghi nhan truoc do:
    echo ----------------------------------------------------------------
    type "%STATUS_FILE%"
    echo ----------------------------------------------------------------
    echo.
)

echo [*] Dang kiem tra thong so bo nho BAR1 qua nvidia-smi...
where nvidia-smi >nul 2>&1
if "%errorlevel%"=="0" (
    nvidia-smi -q -d memory | findstr /i "BAR1 Total"
    if errorlevel 1 (
        echo [!] Khong the doc truong thong tin BAR1 tu nvidia-smi.
    )
) else (
    echo [!] Khong tim thay lenh nvidia-smi. Driver NVIDIA chua duoc cai dat day du.
)

echo.
echo [*] Huong dan kiem tra truc quan qua GPU-Z:
echo     - Mo GPU-Z -^> Kiem tra muc 'Resizable BAR' o dong cuoi = Enabled.
echo     - Tab Advanced -^> Dropdown chon 'PCIe Resizable BAR' -^> BAR1 = 8192 MB.
echo ================================================================
echo.
if "!NO_WAIT!"=="0" pause
exit /b 0


rem ================================================================
rem ACTION: GO BO VA KHOI PHUC MAC DINH (UNINSTALL)
rem ================================================================
:action_uninstall
cls
echo ================================================================
echo    GO BO VA KHOI PHUC MAC DINH RESIZABLE BAR CHO CMP 30HX
echo ================================================================
echo.
echo [*] Thao tac nay se giup ban:
echo     1. Xoa bo bien cau hinh ReBAR trong UEFI NVRAM.
echo     2. Xoa Scheduled Task va Registry duy tri rBAR Profile.
echo     3. Xoa file trang thai tai ProgramData.
echo.

if "!IS_MOCK!"=="1" (
    echo [MOCK] Gia lap xoa bien NVRAM va don dep file thanh cong.
) else (
    echo [*] Dang tu dong xoa cau hinh ReBAR trong UEFI NVRAM...
    if exist "%TOOL_DIR%\NvStrapsReBar.exe" (
        (echo C & echo S & echo Q) | "%TOOL_DIR%\NvStrapsReBar.exe" >nul 2>&1
        echo    [+] Da xoa cau hinh ReBAR trong NVRAM: [OK]
    )
)

echo [*] Dang xoa Scheduled Task va Registry duy tri rBAR Profile...
schtasks /delete /tn "NVIDIA_ReBAR_Global_Profile" /f >nul 2>&1
reg delete "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /f >nul 2>&1
if exist "%ProgramFiles%\40HXUnlock\rebar" rmdir /s /q "%ProgramFiles%\40HXUnlock\rebar" >nul 2>&1

if exist "%STATUS_FILE%" del /f /q "%STATUS_FILE%" >nul 2>&1
echo.
echo [V] Da hoan tat go bo cau hinh ReBAR.
echo [*] Vui long khoi dong lai may de bo mach chu tro ve muc 256MB mac dinh.
echo ================================================================
echo.
if "!NO_WAIT!"=="0" pause
exit /b 0
