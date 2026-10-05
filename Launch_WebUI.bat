@echo off
setlocal
chcp 65001 >nul
title CMP 40HX and 30HX Web Control Center

REM 1. Kiem tra tham so dong lenh bo qua Admin
if /i "%~1"=="-noadmin" goto :run_app
if /i "%~1"=="/noadmin" goto :run_app
if /i "%~1"=="-elevated" goto :check_elevated

REM 2. Kiem tra co lap UAC de tranh loop vo tan
if "%__ELEVATING__%"=="1" goto :elevation_failed

:check_elevated
REM 3. Kiem tra quyen Administrator bang da tang phong thu (fltmc, fsutil, write check)
REM Khong dung 'net session' vi neu dich vu LanmanServer bi tat se luon bao loi
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

if "%IS_ADMIN%"=="1" goto :run_app

REM Neu da duoc goi qua UAC ma van khong co Admin, dung lai tranh loop
if /i "%~1"=="-elevated" goto :elevation_failed

echo [!] Yeu cau quyen Administrator. Dang tu dong kich hoat UAC...
set "CURRENT_SCRIPT=%~f0"
set "CURRENT_DIR=%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -Command "$script=$env:CURRENT_SCRIPT; $dir=$env:CURRENT_DIR; $q=[char]34; Start-Process -FilePath $env:ComSpec -ArgumentList ('/c ' + $q + $script + $q + ' -elevated') -WorkingDirectory $dir -Verb RunAs" >nul 2>&1
if errorlevel 1 goto :elevation_failed
exit /b

:elevation_failed
echo.
echo ================================================================
echo [X] LOI: Khong the tu dong kich hoat quyen Administrator qua UAC.
echo [!] Vui long nhap chuot phai vao file Launch_WebUI.bat
echo     va chon 'Run as administrator' (Chay voi tu cach quan tri vien).
echo ================================================================
echo.
pause
exit /b 1

:run_app
cd /d "%~dp0windows-v3.0\release"
if not exist "40HXInstaller.exe" (
    echo [!] Khong tim thay file windows-v3.0\release\40HXInstaller.exe
    pause
    exit /b 1
)

echo [*] Dang khoi chay CMP Control Center Web UI...
start "" "40HXInstaller.exe"
exit /b 0
