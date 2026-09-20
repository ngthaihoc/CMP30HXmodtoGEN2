@echo off
chcp 65001 >nul 2>&1
title 40HX Unlock - 制作解锁 U 盘 v3.0.0
setlocal EnableDelayedExpansion

rem ============================================================
rem  CMP 40HX Windows Unlock - U 盘手动引导解锁制作工具 v3.0.0
rem  用途: 固件看不到 40HX Unlock 启动项 / 引导链异常时的兜底方案
rem  用法: 双击运行               -> 制作解锁 U 盘
rem        make_usb_efi.bat /restore -> 还原 U 盘到制作前状态
rem  注意: 本脚本只写 U 盘, 不改本机硬盘的任何引导项
rem  详见: README 2.2 节
rem ============================================================

rem ---- 0. 定位解锁 EFI(主流程与还原流程都要用) ----
set "SRC=%~dp0files\40HXUNLK.EFI"
if not exist "%SRC%" set "SRC=%~dp0gen2\40HXUNLK.EFI"
if not exist "%SRC%" set "SRC=%~dp040HXUNLK.EFI"

if /i "%~1"=="/restore" goto RESTORE
if /i "%~1"=="restore" goto RESTORE
if /i "%~1"=="/r" goto RESTORE

echo ============================================================
echo   40HX U 盘手动引导解锁 - 制作工具 v3.0.0
echo   用途: 固件看不到 40HX Unlock 启动项 / 引导链出问题时,
echo         把解锁 EFI 拷到 U 盘, 手动从 U 盘引导解锁。
echo   说明: 本脚本只写 U 盘, 不改本机硬盘的任何引导项。
echo ============================================================
echo.

rem ---- 1. 校验解锁 EFI ----
if not exist "%SRC%" (
    echo [!!] 找不到 40HXUNLK.EFI ^(应在 files\ 或 gen2\ 目录^)
    echo      请确认本脚本与发布包放在同一文件夹。
    echo.
    pause
    exit /b 1
)
set "SRC_SIZE=0"
for %%F in ("%SRC%") do set "SRC_SIZE=%%~zF"
if !SRC_SIZE! LSS 65536 (
    echo [!!] 解锁 EFI 只有 !SRC_SIZE! 字节, 疑似损坏, 已中止。
    echo.
    pause
    exit /b 1
)
echo 解锁 EFI: %SRC%
echo          大小 !SRC_SIZE! 字节
echo.

rem ---- 2. 自动扫描 U 盘(可移动 + 有文件系统) ----
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
        echo   候选 !CNT!: %%a:   文件系统=%%b   可用=%%d GB
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
    echo [i] 没有自动识别到可移动磁盘。
    echo    常见原因: U 盘没插 / 没格式化 / 被识别成本地磁盘。
    echo    没关系, 下面手动输入盘符即可。
    echo.
) else (
    echo [i] 自动扫描到 !CNT! 个可移动磁盘 ^(上面列出的候选^)。
    echo.
)

rem ---- 3. 确定盘符(自动选中或手动输入) ----
set "DRV=%~1"
if defined PICK if not defined DRV set "DRV=!PICK!"
if not defined DRV (
    set /p "DRV=请输入 U 盘盘符 (只输字母, 例: E) 后回车: "
) else (
    if not "%~1"=="" (
        rem 命令行传入, 直接使用
    ) else (
        if defined PICK (
            set /p "DRV=U 盘盘符 (直接回车使用 !PICK!:, 或输入其它字母): "
        ) else (
            set /p "DRV=请输入 U 盘盘符 (只输字母, 例: E) 后回车: "
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
    echo [!!] 未输入盘符, 退出。
    pause
    exit /b 1
)
set "DRV=%DRV:~0,1%"
for %%L in (A B C D E F G H I J K L M N O P Q R S T U V W X Y Z) do if /i "%DRV%"=="%%L" set "DRV=%%L"
set "USR=%DRV%:"

if not exist "%USR%\" (
    echo.
    echo [!!] 盘符 %USR% 不存在, 请核对后重试。
    pause
    exit /b 1
)
if /i "%DRV%"=="C" (
    echo.
    echo [!!] 不能写入系统盘 C:, 请选择 U 盘盘符。
    pause
    exit /b 1
)
if /i "%USR%"=="%SystemDrive%" (
    echo.
    echo [!!] %USR% 是系统所在盘, 已中止。
    pause
    exit /b 1
)

rem ---- 4. 查询该盘的文件系统与类型 ----
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
echo 目标盘: %USR%\   文件系统: %V_FS%   类型: %V_TYPE%
echo.

set "WARN=0"
if /i not "%V_FS%"=="FAT32" (
    set "WARN=1"
    echo [!!] 该盘不是 FAT32 ^(当前: %V_FS%^)。UEFI 固件一般只能从 FAT32 盘引导,
    echo      建议先备份数据、格式化为 FAT32 再来。
)
if /i not "%V_TYPE%"=="Removable" (
    set "WARN=1"
    echo [!]  该盘未被识别为可移动磁盘 ^(当前: %V_TYPE%^), 请确认它不是本机硬盘。
)
if /i "%V_TYPE%"=="CD-ROM" (
    set "WARN=1"
    echo [!]  该盘是光驱, 无法写入。
)

if "%WARN%"=="1" (
    echo.
    set "GO="
    set /p "GO=仍要继续写入 %USR%\ ? 输入 Y 继续, 其它键取消: "
    if /i not "!GO!"=="Y" (
        echo 已取消, 未做任何改动。
        echo.
        pause
        exit /b 0
    )
) else (
    set "GO="
    set /p "GO=确认写入 %USR%\ ? [Y/N]: "
    if /i not "!GO!"=="Y" (
        echo 已取消, 未做任何改动。
        echo.
        pause
        exit /b 0
    )
)

rem ---- 5. 写入 ----
echo.
mkdir "%USR%\EFI" >nul 2>&1
mkdir "%USR%\EFI\40HX" >nul 2>&1
mkdir "%USR%\EFI\Boot" >nul 2>&1
if not exist "%USR%\EFI\Boot\" (
    echo [!!] 无法创建 %USR%\EFI\Boot
    echo      常见原因: U 盘写保护 / 没有权限 / 盘符选错。
    echo      可右键本脚本 - 以管理员身份运行 再试一次。
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
            echo [i] 已备份 U 盘原 bootx64.efi 为 bootx64.efi.40hx.bak
        ) else (
            echo [i] 已存在旧备份 bootx64.efi.40hx.bak, 未覆盖。
        )
    ) else (
        echo [i] U 盘 bootx64.efi 已是本解锁 EFI, 跳过备份。
    )
)

copy /y "%SRC%" "%USR%\EFI\40HX\40HXUNLK.EFI" >nul
if errorlevel 1 goto COPYFAIL
copy /y "%SRC%" "%USR%\EFI\Boot\bootx64.efi" >nul
if errorlevel 1 goto COPYFAIL

rem ---- 6. 回读校验 ----
set "FAIL=0"
echo.
echo [1/2] 校验 \EFI\40HX\40HXUNLK.EFI ...
fc /b "%SRC%" "%USR%\EFI\40HX\40HXUNLK.EFI" >nul 2>&1
if not errorlevel 1 (
    echo       校验 OK
) else (
    echo       [!!] 校验失败
    set "FAIL=1"
)
echo [2/2] 校验 \EFI\Boot\bootx64.efi ...
fc /b "%SRC%" "%USR%\EFI\Boot\bootx64.efi" >nul 2>&1
if not errorlevel 1 (
    echo       校验 OK
) else (
    echo       [!!] 校验失败
    set "FAIL=1"
)

if "%FAIL%"=="1" (
    echo.
    echo [!!] 写入校验未通过, 请勿使用该 U 盘引导解锁。
    echo      常见原因: U 盘写保护 / 空间不足 / 接触不良。
    echo      可换一个 U 盘, 或运行 make_usb_efi.bat /restore 还原后重试。
    echo.
    pause
    exit /b 1
)

echo.
echo ============================================================
echo   制作完成! 使用方法:
echo   1. 关机后插着这个 U 盘开机, 按启动菜单键
echo      (华硕/技嘉 F8, 微星 F11, 联想 F12)
echo      选择名称以 UEFI: 开头的 U 盘项
echo   2. 出现 40HX 解锁文字约 10~30 秒 = 解锁注入成功
echo   3. 若没自动进 Windows: 重启, 启动菜单选 Windows 硬盘项
echo   4. 进系统后运行 40HXCheck.exe 验证(SS0=0x88888888 即成功)
echo.
echo   注意: 主板开了 Secure Boot 的话, 未签名 EFI 不会被执行,
echo         需先在 BIOS 里关闭 Secure Boot, 否则 U 盘会被跳过。
echo.
echo   用完执行 make_usb_efi.bat /restore 可把 U 盘还原原样。
echo ============================================================
echo.
pause
exit /b 0

:COPYFAIL
echo.
echo [!!] 复制失败: %USR% 无法写入。
echo      常见原因: U 盘写保护 / 空间不足 / 没有权限 / 盘符选错。
echo      可右键本脚本 - 以管理员身份运行 再试一次。
echo.
pause
exit /b 1

:RESTORE
echo ============================================================
echo   40HX 解锁 U 盘 - 还原
echo   把 U 盘恢复成本工具写入前的状态
echo ============================================================
echo.

set "DRV=%~2"
if not defined DRV set /p "DRV=请输入 U 盘盘符 (只输字母, 例: E) 后回车: "
set "DRV=%DRV::=%"
set "DRV=%DRV:\=%"
set "DRV=%DRV:/=%"
set "DRV=%DRV: =%"
set "DRV=%DRV:"=%"
if not defined DRV (
    echo [!!] 未输入盘符, 退出。
    pause
    exit /b 1
)
set "DRV=%DRV:~0,1%"
for %%L in (A B C D E F G H I J K L M N O P Q R S T U V W X Y Z) do if /i "%DRV%"=="%%L" set "DRV=%%L"
set "USR=%DRV%:"

if not exist "%USR%\" (
    echo [!!] 盘符 %USR% 不存在, 请核对后重试。
    pause
    exit /b 1
)
if /i "%DRV%"=="C" (
    echo [!!] 拒绝操作系统盘 C:。
    pause
    exit /b 1
)
if /i "%USR%"=="%SystemDrive%" (
    echo [!!] %USR% 是系统所在盘, 已中止。
    pause
    exit /b 1
)

set "BAK=%USR%\EFI\Boot\bootx64.efi.40hx.bak"
if exist "%BAK%" (
    copy /y "%BAK%" "%USR%\EFI\Boot\bootx64.efi" >nul
    if errorlevel 1 (
        echo [!!] 还原失败, 备份文件仍在: %BAK%
    ) else (
        del /f /q "%BAK%" >nul 2>&1
        echo [i] 已还原 U 盘原 bootx64.efi
    )
) else (
    if exist "%USR%\EFI\Boot\bootx64.efi" (
        if exist "%SRC%" (
            fc /b "%SRC%" "%USR%\EFI\Boot\bootx64.efi" >nul 2>&1
            if errorlevel 1 (
                echo [i] bootx64.efi 不是本工具写入的文件, 保持不动。
            ) else (
                del /f /q "%USR%\EFI\Boot\bootx64.efi" >nul 2>&1
                echo [i] 已删除本工具写入的 bootx64.efi
            )
        ) else (
            echo [i] 找不到本地 40HXUNLK.EFI, 无法判断 bootx64.efi 来源, 保持不动。
        )
    ) else (
        echo [i] U 盘上没有 bootx64.efi, 无需还原。
    )
)

if exist "%USR%\EFI\40HX\40HXUNLK.EFI" (
    del /f /q "%USR%\EFI\40HX\40HXUNLK.EFI" >nul 2>&1
    echo [i] 已删除 \EFI\40HX\40HXUNLK.EFI
)
rmdir "%USR%\EFI\40HX" >nul 2>&1
rmdir "%USR%\EFI\Boot" >nul 2>&1
rmdir "%USR%\EFI" >nul 2>&1

echo.
echo [ok] 还原流程结束, U 盘已恢复普通状态。
echo.
pause
exit /b 0
