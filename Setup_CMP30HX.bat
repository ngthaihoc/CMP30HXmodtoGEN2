@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion
title CMP 30HX Gen2 x16 Auto Setup

:: Tự động yêu cầu quyền Administrator nếu chưa có
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [!] Đang yêu cầu quyền Administrator...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process -FilePath '%~f0' -Verb RunAs"
    exit /b
)

cd /d "%~dp0"

:: Hỗ trợ tham số gỡ cài đặt
if /i "%~1"=="-uninstall" goto :uninstall
if /i "%~1"=="/u" goto :uninstall

echo ================================================================
echo    CÔNG CỤ CÀI ĐẶT TỰ ĐỘNG GEN2 X16 CHO NVIDIA CMP 30HX (TU116)
echo ================================================================
echo.

:: Tìm đường dẫn 40HXInstaller.exe
set "INSTALLER="
if exist "%~dp0windows-v3.0\release\40HXInstaller.exe" (
    set "INSTALLER=%~dp0windows-v3.0\release\40HXInstaller.exe"
) else if exist "%~dp0release\40HXInstaller.exe" (
    set "INSTALLER=%~dp0release\40HXInstaller.exe"
) else if exist "%~dp040HXInstaller.exe" (
    set "INSTALLER=%~dp040HXInstaller.exe"
)

if not defined INSTALLER (
    echo [X] LỖI: Không tìm thấy 40HXInstaller.exe!
    echo Vui lòng đảm bảo bạn giải nén / clone đầy đủ repo.
    echo.
    pause
    exit /b 1
)

echo [*] Tìm thấy: "!INSTALLER!"
echo.

:: 1. Dọn dẹp task cũ (nếu có)
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1

:: 2. Đăng ký Scheduled Task chạy ngầm khi Logon
echo [1/2] Đang tạo Scheduled Task tự kích hoạt khi đăng nhập Windows...
schtasks /create /tn "CMP30HX_Gen2_Unlock" /tr "\"!INSTALLER!\" -gen2-30hx -silent" /sc onlogon /rl highest /f >nul 2>&1
if %errorlevel% equ 0 (
    echo       [OK] Đã đăng ký tác vụ "CMP30HX_Gen2_Unlock" thành công.
) else (
    echo       [!] Cảnh báo: Không thể tạo Scheduled Task.
)

:: 3. Kích hoạt mở khoá Gen2 ngay lập tức
echo.
echo [2/2] Đang kích hoạt mở khoá Gen2 x16 và tối ưu MRRS 512B ngay...
"!INSTALLER!" -gen2-30hx

echo.
echo ================================================================
echo  [V] HOÀN TẤT CÀI ĐẶT!
echo  - PCIe Gen2 x16 và MRRS 512B đã được kích hoạt.
echo  - Hệ thống sẽ tự động mở khoá mỗi khi bạn đăng nhập Windows.
echo  - Kiểm tra lại bằng 40HXCheck.exe hoặc AIDA64 GPGPU Benchmark.
echo ================================================================
echo.
pause
exit /b 0

:uninstall
echo ================================================================
echo    GỠ BỎ TỰ ĐỘNG KHỞI ĐỘNG CMP 30HX GEN2 UNLOCK
echo ================================================================
schtasks /delete /tn "CMP30HX_Gen2_Unlock" /f >nul 2>&1
schtasks /delete /tn "40HXGen2Retry" /f >nul 2>&1
echo.
echo [V] Đã xoá toàn bộ tác vụ Scheduled Task liên quan.
echo.
pause
exit /b 0
