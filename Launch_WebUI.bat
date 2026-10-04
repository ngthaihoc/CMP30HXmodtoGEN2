@echo off
setlocal
chcp 65001 >nul
title CMP 40HX & 30HX Web Control Center

:: Tự động nâng quyền Administrator nếu chưa có
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [!] Yêu cầu quyền Administrator. Đang tự động kích hoạt UAC...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~f0' -Verb RunAs"
    exit /b
)

cd /d "%~dp0windows-v3.0\release"
if not exist "40HXInstaller.exe" (
    echo [!] Không tìm thấy file windows-v3.0\release\40HXInstaller.exe
    pause
    exit /b 1
)

echo [*] Đang khởi chạy CMP Control Center Web UI...
start "" "40HXInstaller.exe"
exit /b 0
