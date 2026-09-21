@echo off
chcp 65001 >nul 2>&1
title 40HX Unlock - Tao USB boot cuu ho mo khoa v3.0.0
setlocal EnableDelayedExpansion

rem ============================================================
rem  CMP 40HX Windows Unlock - Cong cu tao USB boot thu cong v3.0.0
rem  Muc dich: Du phong khi BIOS/firmware khong thay muc boot 40HX Unlock / loi chuoi boot
rem  Cach dung: Nhap dup chuot de chay     -> Tao USB mo khoa
rem             make_usb_efi.bat /restore -> Khoi phuc USB ve trang thai ban dau
rem  Luu y: Script chi ghi len USB, khong thay doi bat ky muc boot nao tren o cung
rem  Chi tiet: Xem README muc 2.2
rem ============================================================

rem ---- 0. Dinh vi file EFI mo khoa (dung cho ca quy trinh tao va khoi phuc) ----
set "SRC=%~dp0files\40HXUNLK.EFI"
if not exist "%SRC%" set "SRC=%~dp0gen2\40HXUNLK.EFI"
if not exist "%SRC%" set "SRC=%~dp040HXUNLK.EFI"

if /i "%~1"=="/restore" goto RESTORE
if /i "%~1"=="restore" goto RESTORE
if /i "%~1"=="/r" goto RESTORE

echo ============================================================
echo   40HX - Cong cu tao USB boot mo khoa thu cong v3.0.0
echo   Muc dich: Khi BIOS khong nhan muc boot 40HX Unlock hoac loi boot,
echo         sao chep EFI mo khoa vao USB de boot thu cong tu USB.
echo   Luu y: Script chi ghi len USB, khong sua o cung may tinh.
echo ============================================================
echo.

rem ---- 1. Kiem tra file EFI mo khoa ----
if not exist "%SRC%" (
    echo [!!] Khong tim thay 40HXUNLK.EFI ^(trong thu muc files\ hoac gen2\^)
    echo      Vui long dam bao script nay nam cung thu muc voi bo phat hanh.
    echo.
    pause
    exit /b 1
)
set "SRC_SIZE=0"
for %%F in ("%SRC%") do set "SRC_SIZE=%%~zF"
if !SRC_SIZE! LSS 65536 (
    echo [!!] File EFI mo khoa chi co !SRC_SIZE! bytes, nghi ngo bi loi, da dung lai.
    echo.
    pause
    exit /b 1
)
echo EFI mo khoa: %SRC%
echo          Kich thuoc !SRC_SIZE! bytes
echo.

rem ---- 2. Tu dong quet o USB (Removable + co he thong tep) ----
set "TMPL=%TEMP%\40hx_usb_list.tmp"
if exist "%TMPL%" del /f /q "%TMPL%" >nul 2>&1
powershell -NoProfile -Command "Get-Volume | Where-Object { $_.DriveType -eq 'Removable' -and $_.DriveLetter -and $_.FileSystem } | ForEach-Object { Write-Output ($_.DriveLetter.ToString() + ';' + $_.FileSystem + ';' + $_.DriveType + ';' + [string][math]::Round($_.SizeRemaining/1GB,1)) }" > "%TMPL%" 2>nul

set "CNT=0"
set "PICK="
set "PICK_FS="
set "PICK_FREE="
if exist "%TMPL%" (
    for /f "usebackq tokens=1-4 delims=;" %%a in ("%TMPL%") do (
        set /a CNT+=1
        echo   O dia !CNT!: %%a:   Dinh dang=%%b   Trong=%%d GB
        if "!CNT!"=="1" (
            set "PICK=%%a"
            set "PICK_FS=%%b"
            set "PICK_FREE=%%d"
        )
    )
    del /f /q "%TMPL%" >nul 2>&1
)

echo.
if "%CNT%"=="0" (
    echo [i] Khong tu dong phat hien o dia di dong nao.
    echo    Nguyen nhan: Chua cam USB / chua format / bi nhan dien la o cung local.
    echo    Khong sao ca, ban co the nhap truc tiep ky tu o dia o ben duoi.
    echo.
) else (
    echo [i] Da tu dong quet duoc !CNT! o dia di dong ^(danh sach o tren^).
    echo.
)

rem ---- 3. Xac dinh ky tu o dia (tu dong chon hoac nhap thu cong) ----
set "DRV=%~1"
if defined PICK if not defined DRV set "DRV=!PICK!"
if not defined DRV (
    set /p "DRV=Vui long nhap ky tu o USB (chi nhap 1 chu cai, vi du: E) roi Enter: "
) else (
    if not "%~1"=="" (
        rem Truyen tu command line, dung truc tiep
    ) else (
        if defined PICK (
            set /p "DRV=Ky tu o USB (nhan Enter de dung !PICK!:, hoac nhap chu cai khac): "
        ) else (
            set /p "DRV=Vui long nhap ky tu o USB (chi nhap 1 chu cai, vi du: E) roi Enter: "
        )
    )
)
set "DRV=%DRV::=%"
set "DRV=%DRV:\=%"
set "DRV=%DRV:/=%"
set "DRV=%DRV:.=%"
set "DRV=%DRV: =%"
set "DRV=%DRV:"=%"
if not defined DRV (
    echo.
    echo [!!] Chua nhap ky tu o dia, thoat.
    pause
    exit /b 1
)
set "DRV=%DRV:~0,1%"
for %%L in (A B C D E F G H I J K L M N O P Q R S T U V W X Y Z) do if /i "%DRV%"=="%%L" set "DRV=%%L"
set "USR=%DRV%:"

if not exist "%USR%\" (
    echo.
    echo [!!] O dia %USR% khong ton tai, vui long kiem tra va thu lai.
    pause
    exit /b 1
)
if /i "%DRV%"=="C" (
    echo.
    echo [!!] Khong duoc phep ghi vao o he thong C:, vui long chon o USB.
    pause
    exit /b 1
)
if /i "%USR%"=="%SystemDrive%" (
    echo.
    echo [!!] %USR% la o chua he dieu hanh, da huy thao tac.
    pause
    exit /b 1
)

rem ---- 4. Kiem tra dinh dang he thong tep va loai o dia ----
set "TMPF=%TEMP%\40hx_usb_vol.tmp"
set "VINFO="
powershell -NoProfile -Command "$v=Get-Volume -DriveLetter '%DRV%' -ErrorAction SilentlyContinue; if($v){ Write-Output ($v.FileSystem + ';' + $v.DriveType) }" > "%TMPF%" 2>nul
if exist "%TMPF%" set /p VINFO=<"%TMPF%"
del /f /q "%TMPF%" >nul 2>&1
set "V_FS=UNKNOWN"
set "V_TYPE=UNKNOWN"
if defined VINFO for /f "tokens=1,2 delims=;" %%a in ("%VINFO%") do (
    set "V_FS=%%a"
    set "V_TYPE=%%b"
)
if not defined V_FS set "V_FS=UNKNOWN"
if not defined V_TYPE set "V_TYPE=UNKNOWN"

echo.
echo O muc tieu: %USR%\   Dinh dang: %V_FS%   Loai: %V_TYPE%
echo.

set "WARN=0"
if /i not "%V_FS%"=="FAT32" (
    set "WARN=1"
    echo [!!] O dia nay khong phai FAT32 ^(hien tai: %V_FS%^). Firmware UEFI thuong chi boot duoc tu FAT32,
    echo      Khuyen nghi nen sao luu du lieu va format USB sang FAT32 truoc khi tiep tuc.
)
if /i not "%V_TYPE%"=="Removable" (
    set "WARN=1"
    echo [!]  O dia khong phai la Removable ^(hien tai: %V_TYPE%^), hay dam bao day khong phai o cung trong may.
)
if /i "%V_TYPE%"=="CD-ROM" (
    set "WARN=1"
    echo [!]  Day la o CD-ROM, khong the ghi du lieu.
)

if "%WARN%"=="1" (
    echo.
    set "GO="
    set /p "GO=Ban van muon ghi vao %USR%\ ? Nhap Y de tiep tuc, phim khac de huy: "
    if /i not "!GO!"=="Y" (
        echo Da huy thao tac, khong co thay doi nao.
        echo.
        pause
        exit /b 0
    )
) else (
    set "GO="
    set /p "GO=Xac nhan ghi vao %USR%\ ? [Y/N]: "
    if /i not "!GO!"=="Y" (
        echo Da huy thao tac, khong co thay doi nao.
        echo.
        pause
        exit /b 0
    )
)

rem ---- 5. Ghi du lieu ----
echo.
mkdir "%USR%\EFI" >nul 2>&1
mkdir "%USR%\EFI\40HX" >nul 2>&1
mkdir "%USR%\EFI\Boot" >nul 2>&1
if not exist "%USR%\EFI\Boot\" (
    echo [!!] Khong the tao thu muc %USR%\EFI\Boot
    echo      Nguyen nhan: USB bi khoa chong ghi (write-protected) / khong co quyen / chon sai o.
    echo      Hay nhap chuot phai vao script va chon Run as administrator de thu lai.
    echo.
    pause
    exit /b 1
)

set "BAK=%USR%\EFI\Boot\bootx64.efi.40hx.bak"
if exist "%USR%\EFI\Boot\bootx64.efi" (
    fc /b "%SRC%" "%USR%\EFI\Boot\bootx64.efi" >nul 2>&1
    if errorlevel 1 (
        if not exist "%BAK%" (
            copy /y "%USR%\EFI\Boot\bootx64.efi" "%BAK%" >nul
            echo [i] Da sao luu file bootx64.efi cu tren USB thanh bootx64.efi.40hx.bak
        ) else (
            echo [i] Da co ban sao luu cu bootx64.efi.40hx.bak, bo qua ghi de.
        )
    ) else (
        echo [i] File bootx64.efi tren USB da la ban EFI mo khoa nay, bo qua sao luu.
    )
)

copy /y "%SRC%" "%USR%\EFI\40HX\40HXUNLK.EFI" >nul
if errorlevel 1 goto COPYFAIL
copy /y "%SRC%" "%USR%\EFI\Boot\bootx64.efi" >nul
if errorlevel 1 goto COPYFAIL

rem ---- 6. Kiem tra lai sau khi ghi ----
set "FAIL=0"
echo.
echo [1/2] Kiem tra \EFI\40HX\40HXUNLK.EFI ...
fc /b "%SRC%" "%USR%\EFI\40HX\40HXUNLK.EFI" >nul 2>&1
if not errorlevel 1 (
    echo       Kiem tra OK
) else (
    echo       [!!] Kiem tra THAT BAI
    set "FAIL=1"
)
echo [2/2] Kiem tra \EFI\Boot\bootx64.efi ...
fc /b "%SRC%" "%USR%\EFI\Boot\bootx64.efi" >nul 2>&1
if not errorlevel 1 (
    echo       Kiem tra OK
) else (
    echo       [!!] Kiem tra THAT BAI
    set "FAIL=1"
)

if "%FAIL%"=="1" (
    echo.
    echo [!!] Kiem tra ghi file khong dat, vui long khong dung USB nay de boot mo khoa.
    echo      Nguyen nhan: USB chong ghi / dung luong day / tiep xuc kem.
    echo      Co the doi USB khac, hoac chay make_usb_efi.bat /restore de khoi phuc roi thu lai.
    echo.
    pause
    exit /b 1
)

echo.
echo ============================================================
echo   Tao USB thanh cong! Huong dan su dung:
echo   1. Tat may, cam USB nay vao may va bat nguon, an phim Boot Menu
echo      (Asus/Gigabyte: F8, MSI: F11, Lenovo: F12)
echo      Chon muc USB co chu dau la UEFI:
echo   2. Xuat hien dong chu mo khoa 40HX khoang 10~30 giay = mo khoa thanh cong
echo   3. Neu khong tu vao Windows: Khoi dong lai, vao Boot Menu chon o cung Windows
echo   4. Vao Windows chay 40HXCheck.exe de kiem tra (SS0=0x88888888 la thanh cong)
echo.
echo   Luu y: Neu mainboard dang bat Secure Boot, EFI khong chung thuc se bi chan,
echo         can vao BIOS tat Secure Boot truoc, neu khong USB se bi bo qua.
echo.
echo   Sau khi dung xong chay make_usb_efi.bat /restore de khoi phuc lai USB.
echo ============================================================
echo.
pause
exit /b 0

:COPYFAIL
echo.
echo [!!] Sao chep that bai: %USR% khong the ghi du lieu.
echo      Nguyen nhan: USB chong ghi / day dung luong / thieu quyen / chon sai o.
echo      Hay nhap chuot phai vao script - Run as administrator de thu lai.
echo.
pause
exit /b 1

:RESTORE
echo ============================================================
echo   40HX - Khoi phuc USB mo khoa
echo   Dua USB ve trang thai truoc khi cong cu ghi du lieu
echo ============================================================
echo.

set "DRV=%~2"
if not defined DRV set /p "DRV=Vui long nhap ky tu o USB (chi nhap 1 chu cai, vi du: E) roi Enter: "
set "DRV=%DRV::=%"
set "DRV=%DRV:\=%"
set "DRV=%DRV:/=%"
set "DRV=%DRV: =%"
set "DRV=%DRV:"=%"
if not defined DRV (
    echo [!!] Chua nhap ky tu o dia, thoat.
    pause
    exit /b 1
)
set "DRV=%DRV:~0,1%"
for %%L in (A B C D E F G H I J K L M N O P Q R S T U V W X Y Z) do if /i "%DRV%"=="%%L" set "DRV=%%L"
set "USR=%DRV%:"

if not exist "%USR%\" (
    echo [!!] O dia %USR% khong ton tai, vui long kiem tra va thu lai.
    pause
    exit /b 1
)
if /i "%DRV%"=="C" (
    echo [!!] Tu choi thao tac tren o he thong C:.
    pause
    exit /b 1
)
if /i "%USR%"=="%SystemDrive%" (
    echo [!!] %USR% la o he thong, da dung thao tac.
    pause
    exit /b 1
)

set "BAK=%USR%\EFI\Boot\bootx64.efi.40hx.bak"
if exist "%BAK%" (
    copy /y "%BAK%" "%USR%\EFI\Boot\bootx64.efi" >nul
    if errorlevel 1 (
        echo [!!] Khoi phuc that bai, file sao luu van con tai: %BAK%
    ) else (
        del /f /q "%BAK%" >nul 2>&1
        echo [i] Da khoi phuc lai bootx64.efi goc tren USB
    )
) else (
    if exist "%USR%\EFI\Boot\bootx64.efi" (
        if exist "%SRC%" (
            fc /b "%SRC%" "%USR%\EFI\Boot\bootx64.efi" >nul 2>&1
            if errorlevel 1 (
                echo [i] bootx64.efi khong phai file do cong cu nay tao, giu nguyen.
            ) else (
                del /f /q "%USR%\EFI\Boot\bootx64.efi" >nul 2>&1
                echo [i] Da xoa file bootx64.efi do cong cu tao
            )
        ) else (
            echo [i] Khong tim thay 40HXUNLK.EFI goc tren may de so sanh, giu nguyen.
        )
    ) else (
        echo [i] Tren USB khong co bootx64.efi, khong can khoi phuc.
    )
)

if exist "%USR%\EFI\40HX\40HXUNLK.EFI" (
    del /f /q "%USR%\EFI\40HX\40HXUNLK.EFI" >nul 2>&1
    echo [i] Da xoa \EFI\40HX\40HXUNLK.EFI
)
rmdir "%USR%\EFI\40HX" >nul 2>&1
rmdir "%USR%\EFI\Boot" >nul 2>&1
rmdir "%USR%\EFI" >nul 2>&1

echo.
echo [ok] Qua trinh khoi phuc hoan tat, USB da tro ve trang thai binh thuong.
echo.
pause
exit /b 0
