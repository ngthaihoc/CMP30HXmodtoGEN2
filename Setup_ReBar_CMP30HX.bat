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
    set "CURRENT_SCRIPT=%~f0"
    set "CURRENT_DIR=%~dp0"
    set "SCRIPT_ARGS=%*"
    powershell -NoProfile -ExecutionPolicy Bypass -Command "$script=$env:CURRENT_SCRIPT; $dir=$env:CURRENT_DIR; $args=$env:SCRIPT_ARGS; $q=[char]34; $procArgs = if ($args) { '/c ' + $q + $script + $q + ' ' + $args } else { '/c ' + $q + $script + $q }; Start-Process -FilePath $env:ComSpec -ArgumentList $procArgs -WorkingDirectory $dir -Verb RunAs" >nul 2>&1
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
echo    - NvStrapsReBar can UEFI DXE va cau hinh BIOS phu hop; can xac minh sau reboot.
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
    echo    - Thu cau hinh GPU BAR qua NvStrapsReBar va NVIDIA Driver Profile.
    echo    - Viec sua firmware mainboard co rui ro; hay theo huong dan cua nha san xuat.
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
set "SYSINFO_OK=0"

for /f "usebackq tokens=1* delims==" %%A in (`powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\Get-SysInfo.ps1"`) do (
    set "%%A=%%B"
)

if "!IS_MOCK!"=="0" if not "!MOCK_LAPTOP!"=="1" if not "!SYSINFO_OK!"=="1" (
    echo [X] Khong doc duoc thong tin he thong; dung truoc khi ghi EFI.
    if "!NO_WAIT!"=="0" pause
    exit /b 2
)
echo    - He thong   : !SYS_MANU! - Model: !SYS_MODEL!
echo    - Bo mach chu: !MB_MANU! - Model: !MB_PROD! [BIOS: !MB_BIOS!]
echo    - Che do Boot: %FIRMWARE_TYPE%

rem UEFI firmware is required for the DXE driver to run.
if "!IS_MOCK!"=="0" if not "!MOCK_LAPTOP!"=="1" if /i not "%FIRMWARE_TYPE%"=="UEFI" (
    echo [X] ReBAR requires UEFI boot with CSM disabled.
    echo [*] Enable Above 4G Decoding and check UEFI boot mode before retrying.
    if "!NO_WAIT!"=="0" pause
    exit /b 2
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
    rem IS_LAPTOP da duoc lay tu Get-SysInfo.ps1
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

rem Only configure the supported target GPU during a live install.
if "!FOUND_CMP30HX!"=="1" (
    echo    [+] Xac nhan card NVIDIA CMP 30HX [TU116 - DEV_2189].
) else (
    if "!IS_MOCK!"=="1" (
        echo    [MOCK] Gia lap phat hien NVIDIA CMP 30HX [10DE:2189].
    ) else (
        echo [X] Khong tim thay NVIDIA CMP 30HX [10DE:2189].
        echo [*] Dung cai dat de tranh thay doi cau hinh GPU khac.
        if "!NO_WAIT!"=="0" pause
        exit /b 2
    )
)

rem ----------------------------------------------------------------
rem BUOC 2: PHAN BIET BAR0 VA BAR1
rem ----------------------------------------------------------------
echo.
echo [2/5] Kiem tra muc tieu BAR:
echo    - BAR0 16 MB la vung thanh ghi GPU; can kiem tra BAR1 cua VRAM.
echo    - 8 GiB la muc tieu cho VRAM 6 GiB neu GPU va firmware ho tro.
echo    - NvStrapsReBar co the can cau hinh ca GPU-side va PCI-side BAR.
echo    - Chi GPU-Z/nvidia-smi sau reboot moi cho biet BAR1 thuc te.
echo    ----------------------------------------------------------------

rem ----------------------------------------------------------------
rem BUOC 3: KIEM TRA BO CONG CU VA HUONG DAN BIOS MAINBOARD
rem ----------------------------------------------------------------
echo.
echo [3/5] Kiem tra bo cong cu Resizable BAR:
for %%F in (NvStrapsReBar.exe nvidiaProfileInspector.exe Enable_ReBAR_Turing.nip) do (
    if not exist "%TOOL_DIR%\%%F" (
        echo [X] Thieu tep bat buoc: %%F
        if "!NO_WAIT!"=="0" pause
        exit /b 2
    )
)

echo.
rem ----------------------------------------------------------------
rem Inspect DXE status; Bridge Found means the driver saw a PCI bridge,
rem but BAR1 still needs post-reboot verification.
set "DXE_ACTIVE=0"
set "DXE_STATUS=Other status; see console output"
set "DXE_BRIDGE=0"
set "DXE_GPU_SEEN=0"
if "!IS_MOCK!"=="1" (
    echo    [MOCK] Gia lap DXE Bridge Found.
) else (
    set "DXE_REPORT=%TEMP%\CMP30HX_ReBar_DXE_%RANDOM%%RANDOM%.txt"
    echo Q | "%TOOL_DIR%\NvStrapsReBar.exe" > "!DXE_REPORT!" 2>&1
    set "DXE_EXIT=!errorlevel!"
    findstr /i /c:"UEFI DXE driver status:" "!DXE_REPORT!" >nul 2>&1
    if not errorlevel 1 set "DXE_ACTIVE=1"
    findstr /i /c:"status: Loaded" "!DXE_REPORT!" >nul 2>&1
    if not errorlevel 1 set "DXE_STATUS=Loaded"
    findstr /i /c:"status: Unconfigured" "!DXE_REPORT!" >nul 2>&1
    if not errorlevel 1 set "DXE_STATUS=Unconfigured"
    findstr /i /c:"status: Bridge Found" "!DXE_REPORT!" >nul 2>&1
    if not errorlevel 1 (
        set "DXE_BRIDGE=1"
        set "DXE_STATUS=Bridge Found"
    )
    findstr /i /c:"status: Not loaded" /c:"status: Missing" /c:"status: Bad" /c:"status: EFI Error" "!DXE_REPORT!" >nul 2>&1
    if not errorlevel 1 set "DXE_ACTIVE=0"
    findstr /i /c:"10DE:2189" "!DXE_REPORT!" >nul 2>&1
    if not errorlevel 1 set "DXE_GPU_SEEN=1"
    type "!DXE_REPORT!"
    if not "!DXE_EXIT!"=="0" set "DXE_ACTIVE=0"
    del /f /q "!DXE_REPORT!" >nul 2>&1
    if "!DXE_ACTIVE!"=="0" (
        echo [X] Khong xac nhan duoc UEFI DXE driver. Dung truoc khi ghi EFI.
        echo [*] Kiem tra UEFI boot, CSM va Above 4G Decoding trong BIOS.
        if "!NO_WAIT!"=="0" pause
        exit /b 2
    )
    if "!DXE_GPU_SEEN!"=="0" (
        echo [X] NvStrapsReBar khong liet ke GPU 10DE:2189. Dung truoc khi ghi EFI.
        if "!NO_WAIT!"=="0" pause
        exit /b 2
    )
    if "!DXE_BRIDGE!"=="1" (
        echo [!] Bridge Found: DXE da thay PCI bridge, BAR1 chua duoc xac minh.
        echo [*] Mainboard cu co the can cau hinh PCI-side BAR [P] thu cong.
    )
)

rem ----------------------------------------------------------------
rem BUOC 4: CAU HINH NVRAM VA DRIVER PROFILE TU DONG [1-CLICK AIO]
rem ----------------------------------------------------------------
echo [4/5] Tu dong cau hinh phan mem Windows [1-Click Automation]:
echo.
echo [*] BUOC 4A: Thu cau hinh GPU-side BAR qua UEFI NVRAM...
if "!IS_MOCK!"=="1" (
    echo    [MOCK] Gia lap gui lenh E/S/Q; khong ghi EFI NVRAM.
    echo    [MOCK] BAR1 sau reboot chua duoc xac minh.
) else (
    (echo E & echo S & echo Q) | "%TOOL_DIR%\NvStrapsReBar.exe"
    if errorlevel 1 (
        echo [X] NvStrapsReBar failed while saving EFI configuration.
        if "!NO_WAIT!"=="0" pause
        exit /b 2
    )
    echo    [+] Da gui cau hinh EFI. Can khoi dong lai va kiem tra BAR1 thuc te.
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
    if errorlevel 1 (
        echo [X] Khong the nap NVIDIA Driver Profile.
        if "!NO_WAIT!"=="0" pause
        exit /b 2
    )
    echo    [+] Driver Profile da nap; ReBAR van can xac minh sau reboot.
)

echo.
echo [*] BUOC 4C: Duy tri rBAR Driver Profile...
if "!IS_MOCK!"=="1" (
    echo    [MOCK] Bo qua Scheduled Task, Registry va file he thong.
    goto :after_profile_persistence
)
set "REBAR_APP_DIR=%ProgramFiles%\40HXUnlock\rebar"
set "REBAR_RUNNER=%REBAR_APP_DIR%\RunReBarProfile.bat"

if not exist "%REBAR_APP_DIR%" mkdir "%REBAR_APP_DIR%" >nul 2>&1
if not exist "%REBAR_APP_DIR%" (
    echo [X] Khong tao duoc thu muc duy tri Driver Profile.
    exit /b 2
)
for %%F in (nvidiaProfileInspector.exe nvidiaProfileInspector.exe.config Enable_ReBAR_Turing.nip) do (
    copy /y "%TOOL_DIR%\%%F" "%REBAR_APP_DIR%\" >nul 2>&1
    if errorlevel 1 (
        echo [X] Khong sao chep duoc %%F; profile sau reboot co the khong duoc nap.
        exit /b 2
    )
)

(
    echo @echo off
    echo setlocal
    echo cd /d "%%~dp0"
    echo if exist "%%~dp0nvidiaProfileInspector.exe" start /wait "" "%%~dp0nvidiaProfileInspector.exe" -silent "%%~dp0Enable_ReBAR_Turing.nip"
    echo exit /b 0
) > "%REBAR_RUNNER%"

if not exist "%REBAR_RUNNER%" (
    echo [X] Khong tao duoc RunReBarProfile.bat.
    exit /b 2
)
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\Register-ReBarTask.ps1" -TaskName "NVIDIA_ReBAR_Global_Profile" -BatPath "%REBAR_RUNNER%" -WorkingDirectory "%REBAR_APP_DIR%" >nul 2>&1
if errorlevel 1 (
    echo [X] Khong dang ky duoc Scheduled Task duy tri Driver Profile.
    exit /b 2
)
schtasks /query /tn "NVIDIA_ReBAR_Global_Profile" >nul 2>&1
if errorlevel 1 (
    echo [X] Khong xac nhan duoc Scheduled Task sau khi dang ky.
    exit /b 2
)
reg add "HKLM\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /t REG_SZ /d "\"%REBAR_RUNNER%\"" /f >nul 2>&1
if errorlevel 1 echo [!] Khong ghi duoc HKLM Run; Scheduled Task van duoc dang ky.
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "NVIDIA_ReBAR_Profile" /t REG_SZ /d "\"%REBAR_RUNNER%\"" /f >nul 2>&1
if errorlevel 1 echo [!] Khong ghi duoc HKCU Run; Scheduled Task van duoc dang ky.
echo    [+] Da dang ky va xac nhan Scheduled Task duy tri Driver Profile.

rem Ghi nhan file trang thai ReBAR
if not exist "%STATUS_DIR%" mkdir "%STATUS_DIR%" >nul 2>&1
if not exist "%STATUS_DIR%" (
    echo [X] Khong tao duoc thu muc bao cao ReBAR.
    exit /b 2
)
if exist "%STATUS_FILE%" del /f /q "%STATUS_FILE%" >nul 2>&1
if exist "%STATUS_FILE%" (
    echo [X] Khong the thay the bao cao ReBAR cu.
    exit /b 2
)
(
    echo ==== CMP 30HX Resizable BAR Setup Attempt ====
    echo Thoi gian thiet lap : %date% %time%
    echo GPU Muc tieu        : NVIDIA CMP 30HX [10DE:2189]
    echo DXE truoc cai dat   : !DXE_STATUS!
    echo EFI config          : Da gui lenh E/S; can xac minh sau reboot
    echo BAR1 hien tai       : CHUA XAC MINH
    echo Driver Profile      : Da gui lenh nap profile
    echo Trang thai          : PENDING POST-REBOOT VERIFICATION
) > "%STATUS_FILE%"
if not exist "%STATUS_FILE%" (
    echo [X] Khong ghi duoc bao cao ReBAR.
    exit /b 2
)
:after_profile_persistence

rem ----------------------------------------------------------------
rem BUOC 5: TONG KET VA HUONG DAN NGHIEM THU
rem ----------------------------------------------------------------
echo.
echo [5/5] Tong ket va nghiem thu:
echo ================================================================
if "!IS_MOCK!"=="1" (
    echo                    MOCK TEST - KHONG GUI CAU HINH
) else (
    echo                    DA GUI CAU HINH REBAR AIO
)
echo ================================================================
if "!IS_MOCK!"=="1" (
    echo  Mock chi mo phong; khong thay doi he thong va khong can reboot.
) else (
    echo  Trang thai: CHO XAC MINH SAU KHI KHOI DONG LAI
    echo  1. Khoi dong lai he thong.
    echo  2. Mo GPU-Z tab Advanced -^> PCIe Resizable BAR:
    echo     - Kiem tra BAR1 hien tai va BIOS/UEFI support.
    echo     - Neu van Disabled, gui model mainboard, BIOS version,
    echo       anh GPU-Z va toan bo output NvStrapsReBar.
    echo  3. Kiem tra qua nvidia-smi:
    echo     ^> nvidia-smi -q -d memory
    echo     So sanh BAR1 Memory Usage - Total voi gia tri truoc khi cai dat.
    echo  4. File bao cao ghi lai lan THU cai dat, khong xac nhan BAR:
    echo     "%STATUS_FILE%"
)
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
    echo [*] Lan thu cai dat truoc do - chua xac minh BAR:
    echo ----------------------------------------------------------------
    type "%STATUS_FILE%"
    echo ----------------------------------------------------------------
    echo.
)

echo [*] Dang kiem tra thong so bo nho BAR1 qua nvidia-smi...
where nvidia-smi >nul 2>&1
if "%errorlevel%"=="0" (
    nvidia-smi -q -d memory | powershell -NoProfile -Command "$text=$input | Out-String; if($text -match '(?s)BAR1 Memory Usage\s+Total\s*:\s*([^\r\n]+)'){Write-Output ('BAR1 Total (first NVIDIA GPU): '+$Matches[1])}else{exit 1}"
    if errorlevel 1 (
        echo [!] Khong the doc truong thong tin BAR1 tu nvidia-smi.
    )
) else (
    echo [!] Khong tim thay lenh nvidia-smi. Driver NVIDIA chua duoc cai dat day du.
)

echo.
echo [*] Huong dan kiem tra truc quan qua GPU-Z:
echo     - Mo GPU-Z -^> Advanced -^> PCIe Resizable BAR.
echo     - Kiem tra BAR1 va trang thai firmware; BAR0 16 MB la muc rieng.
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
    echo [MOCK] Bo qua moi thao tac xoa EFI, task, registry va file.
    if "!NO_WAIT!"=="0" pause
    exit /b 0
) else (
    echo [*] Dang xoa cau hinh ReBAR trong UEFI NVRAM...
    if exist "%TOOL_DIR%\NvStrapsReBar.exe" (
        (echo C & echo S & echo Q) | "%TOOL_DIR%\NvStrapsReBar.exe" >nul 2>&1
        if errorlevel 1 (
            echo [X] Khong the xoa cau hinh EFI.
            if "!NO_WAIT!"=="0" pause
            exit /b 2
        )
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
