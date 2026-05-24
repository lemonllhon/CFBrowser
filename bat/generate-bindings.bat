@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM Change to repository root (parent directory of this script).
cd /d "%~dp0.."

set "NO_PAUSE=0"
:parse_args
if "%~1"=="" goto :after_parse_args
if /I "%~1"=="--no-pause" (
    set "NO_PAUSE=1"
    shift
    goto :parse_args
)
if /I "%~1"=="--wails3" (
    shift
    goto :parse_args
)
echo [ERROR] Unsupported argument: %~1
if not "%NO_PAUSE%"=="1" pause
exit /b 1

:after_parse_args

set "TEMP_DIST_CREATED=0"
set "TEMP_PLACEHOLDER_CREATED=0"
set "ACTIVE_WAILS_BIN="

echo ========================================
echo   Generate Wails Bindings
echo ========================================
echo.
echo Working directory: %CD%
echo Wails target: v3
echo.

if not exist "build\config.yml" (
    echo [ERROR] build\config.yml not found in repository root.
    echo         This Wails3 branch must keep a complete Wails3 source tree.
    if not "%NO_PAUSE%"=="1" pause
    exit /b 1
)

echo [1/3] Ensure frontend\dist exists...
if not exist "frontend\dist" (
    mkdir "frontend\dist"
    set "TEMP_DIST_CREATED=1"
    echo Created temporary dist directory.
) else (
    echo Dist directory already exists.
)

if not exist "frontend\dist\__wails_placeholder__.txt" (
    echo placeholder> "frontend\dist\__wails_placeholder__.txt"
    set "TEMP_PLACEHOLDER_CREATED=1"
    echo Created temporary placeholder file.
)

echo.
echo [2/3] Regenerating Wails bindings...
call :ensure_go_command
if errorlevel 1 goto :cleanup_fail

call :resolve_wails_bin
if errorlevel 1 goto :cleanup_fail

"%ACTIVE_WAILS_BIN%" generate bindings -ts -d frontend\src\wails3 .
if errorlevel 1 (
    echo Failed to regenerate Wails bindings.
    goto :cleanup_fail
)

echo.
echo [3/3] Verify bindings output...
if exist "frontend\src\wails3" (
    echo Wails3 bindings generated in frontend\src\wails3.
) else (
    echo Cannot find generated Wails3 bindings in frontend\src\wails3.
    goto :cleanup_fail
)

call :cleanup

echo.
echo ========================================
echo   Done
echo ========================================
echo.

if not "%NO_PAUSE%"=="1" pause
exit /b 0

:cleanup
if "!TEMP_PLACEHOLDER_CREATED!"=="1" (
    del /Q "frontend\dist\__wails_placeholder__.txt" >nul 2>&1
)
if "!TEMP_DIST_CREATED!"=="1" (
    rmdir /S /Q "frontend\dist" >nul 2>&1
)
exit /b 0

:cleanup_fail
call :cleanup
if not "%NO_PAUSE%"=="1" pause
exit /b 1

:ensure_go_command
go version >nul 2>&1
if not errorlevel 1 exit /b 0

for /f "delims=" %%g in ('dir /B /AD ".tmp\toolchains\go*" 2^>nul') do (
    if exist ".tmp\toolchains\%%g\go\bin\go.exe" (
        set "GOROOT=%CD%\.tmp\toolchains\%%g\go"
        if not defined GOPATH set "GOPATH=%CD%\.tmp\go"
        if not defined GOBIN set "GOBIN=!GOPATH!\bin"
        set "PATH=!GOROOT!\bin;!GOBIN!;!PATH!"
        go version >nul 2>&1
        if not errorlevel 1 (
            echo Go command: !GOROOT!\bin\go.exe
            exit /b 0
        )
    )
)

echo [ERROR] Cannot find Go command.
echo         Install Go, add it to PATH, or place a portable Go toolchain under .tmp\toolchains\go*\go.
exit /b 1

:resolve_wails_bin
if defined WAILS_BIN set "ACTIVE_WAILS_BIN=%WAILS_BIN%"
if not defined ACTIVE_WAILS_BIN if defined WAILS3_BIN set "ACTIVE_WAILS_BIN=%WAILS3_BIN%"
if not defined ACTIVE_WAILS_BIN set "ACTIVE_WAILS_BIN=wails3"
echo Wails command: %ACTIVE_WAILS_BIN%
"%ACTIVE_WAILS_BIN%" version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Cannot run Wails command: %ACTIVE_WAILS_BIN%
    echo         Set WAILS_BIN or WAILS3_BIN to a valid executable path.
    exit /b 1
)
exit /b 0

