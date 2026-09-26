@echo off
rem ============================================================
rem  40HX Gen2 BYOVD unlock - remove autostart + purge drivers
rem ============================================================
setlocal
schtasks /Delete /TN "40HXGen2" /F 2>nul
sc stop ThrottleStop       >nul 2>&1
sc delete ThrottleStop     >nul 2>&1
sc stop WinRing0_1_2_0     >nul 2>&1
sc delete WinRing0_1_2_0   >nul 2>&1
del /f "%SystemRoot%\System32\drivers\ThrottleStop.sys" >nul 2>&1
del /f "%SystemRoot%\System32\drivers\WinRing0x64.sys"  >nul 2>&1
sc config NVDisplay.ContainerLocalSystem start= auto >nul 2>&1
sc start NVDisplay.ContainerLocalSystem >nul 2>&1
reg add "HKCR\Directory\Background\shellex\ContextMenuHandlers\NvCplDesktopContext" /ve /t REG_SZ /d "{3D1975AF-48C6-4f8e-A182-BE0E08FA86A9}" /f >nul 2>&1
echo Cleaned: task removed, services deleted, driver files removed.
pause
