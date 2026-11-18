@echo off
setlocal
cd /d "%~dp0" || exit /b 1

:: The installer can remove shared driver services. Do not touch existing ones.
"%SystemRoot%\System32\sc.exe" query WinRing0_1_2_0 >nul 2>&1 && exit /b 1
"%SystemRoot%\System32\sc.exe" query ThrottleStop >nul 2>&1 && exit /b 1
if exist "%SystemRoot%\System32\drivers\WinRing0x64.sys" exit /b 1
if exist "%SystemRoot%\System32\drivers\ThrottleStop.sys" exit /b 1

if not exist "%ProgramData%\40HXUnlock" mkdir "%ProgramData%\40HXUnlock" >nul 2>&1
if not exist "%ProgramData%\40HXUnlock\" exit /b 1
if exist "%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt\" exit /b 1
del /f /q "%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt" >nul 2>&1
if exist "%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt" exit /b 1

start "" /wait "%~dp0UnlockAIO.exe" -gen2-30hx -silent -batch
set "INSTALLER_EXIT=%ERRORLEVEL%"
if not "%INSTALLER_EXIT%"=="0" exit /b %INSTALLER_EXIT%
if not exist "%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt" exit /b 1

"%SystemRoot%\System32\findstr.exe" /i /x /c:"STATUS_CODE=GEN2_SUCCESS" "%ProgramData%\40HXUnlock\cmp30hx_batch_status.txt" >nul 2>&1
if errorlevel 1 exit /b 1
exit /b 0
