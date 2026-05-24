@echo off
setlocal EnableExtensions EnableDelayedExpansion

cd /d "%~dp0.."
set "EXIT_CODE=0"
set "NO_PAUSE=0"
set "SHOW_USAGE=0"
set "DEV_MODE=stable"
set "ACTIVE_WAILS_BIN="
set "LIMITED_WATCHER_PID_FILE=tmp-frontend-limited-watcher.pid"
set "PREFERRED_FRONTEND_PORT=5218"
set "FRONTEND_PORT="
set "WATCHER_PID="
set "WATCHER_STARTED=0"

call :parse_args %*
if errorlevel 1 (
    set "EXIT_CODE=1"
    goto :finish
)

if "%SHOW_USAGE%"=="1" (
    call :print_usage
    goto :finish
)

if /I "%DEV_MODE%"=="stable" (
    call :run_stable
    set "EXIT_CODE=%errorlevel%"
    goto :finish
)

if /I "%DEV_MODE%"=="live" (
    call :run_live 0
    set "EXIT_CODE=%errorlevel%"
    goto :finish
)

if /I "%DEV_MODE%"=="limited" (
    call :run_live 1
    set "EXIT_CODE=%errorlevel%"
    goto :finish
)

echo [ERROR] Unsupported dev mode: %DEV_MODE%
set "EXIT_CODE=1"

:finish
if "%WATCHER_STARTED%"=="1" call :cleanup_watcher >nul 2>&1
if "%NO_PAUSE%"=="1" exit /b %EXIT_CODE%
if "%CI%"=="1" exit /b %EXIT_CODE%

pause
exit /b %EXIT_CODE%

:parse_args
if "%~1"=="" exit /b 0
if /I "%~1"=="--no-pause" (
    set "NO_PAUSE=1"
    shift
    goto :parse_args
)
if /I "%~1"=="--help" (
    set "SHOW_USAGE=1"
    shift
    goto :parse_args
)
if /I "%~1"=="-h" (
    set "SHOW_USAGE=1"
    shift
    goto :parse_args
)
if /I "%~1"=="--wails3" (
    shift
    goto :parse_args
)
if /I "%~1"=="stable" (
    set "DEV_MODE=stable"
    shift
    goto :parse_args
)
if /I "%~1"=="live" (
    set "DEV_MODE=live"
    shift
    goto :parse_args
)
if /I "%~1"=="limited" (
    set "DEV_MODE=limited"
    shift
    goto :parse_args
)

echo [ERROR] Unsupported argument: %~1
echo.
call :print_usage
exit /b 1

:print_usage
echo Usage:
echo   bat\dev.bat [stable^|live^|limited] [--wails3] [--no-pause]
echo.
echo Modes:
echo   stable   Default. Build frontend static assets and start Wails without Vite dev server.
echo   live     Start Vite watcher and connect Wails to the frontend dev server.
echo   limited  Same as live, but add Windows Job Object memory limits to the watcher chain.
echo   --wails3 Accepted for compatibility. This branch always runs the Wails3 shell.
echo.
echo Examples:
echo   bat\dev.bat
echo   bat\dev.bat live
echo   bat\dev.bat limited --no-pause
exit /b 0

:run_stable
call :run_wails3
exit /b %errorlevel%

:run_live
echo [WARN] Wails3 live/limited frontend mode is not wired yet; using static frontend assets.
call :run_wails3
exit /b %errorlevel%

:apply_proxy_settings
if defined DEV_PROXY_URL (
    set "HTTP_PROXY=%DEV_PROXY_URL%"
    set "HTTPS_PROXY=%DEV_PROXY_URL%"
    set "http_proxy=%DEV_PROXY_URL%"
    set "https_proxy=%DEV_PROXY_URL%"
)
if defined DEV_NO_PROXY (
    set "NO_PROXY=%DEV_NO_PROXY%"
    set "no_proxy=%DEV_NO_PROXY%"
)
if defined DEV_GOPROXY set "GOPROXY=%DEV_GOPROXY%"
if not defined DEV_GOPROXY if not defined GOPROXY set "GOPROXY=https://goproxy.cn,direct"
exit /b 0

:resolve_wails3_command
set "ACTIVE_WAILS_BIN="
if defined WAILS_BIN set "ACTIVE_WAILS_BIN=%WAILS_BIN%"
if not defined ACTIVE_WAILS_BIN if defined WAILS3_BIN set "ACTIVE_WAILS_BIN=%WAILS3_BIN%"
if not defined ACTIVE_WAILS_BIN set "ACTIVE_WAILS_BIN=wails3"
echo Wails3 command: %ACTIVE_WAILS_BIN%
"%ACTIVE_WAILS_BIN%" version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Cannot run Wails3 command: %ACTIVE_WAILS_BIN%
    echo         Set WAILS_BIN or WAILS3_BIN to a valid executable path.
    exit /b 1
)
exit /b 0

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
            echo.
            exit /b 0
        )
    )
)

echo [ERROR] Cannot find Go command.
echo         Install Go, add it to PATH, or place a portable Go toolchain under .tmp\toolchains\go*\go.
exit /b 1

:print_proxy_settings
if defined DEV_PROXY_URL (
    echo HTTP/HTTPS proxy: %DEV_PROXY_URL%
) else (
    echo HTTP/HTTPS proxy: disabled
)
if defined DEV_NO_PROXY (
    echo NO_PROXY: %DEV_NO_PROXY%
)
echo Go proxy: %GOPROXY%
echo.
exit /b 0

:cleanup_app_processes
echo Cleaning stale app processes...
taskkill /F /IM trace-browser-dev.exe >nul 2>&1
taskkill /F /IM trace-browser-dev.exe >nul 2>&1
echo.
exit /b 0

:cleanup_frontend_dev_processes
echo Cleaning stale frontend dev processes...
node frontend\scripts\dev-port-helper.mjs cleanup
if errorlevel 1 (
    if /I "%~1"=="warn" (
        echo [WARN] Failed to clean stale frontend dev processes. Continuing...
        echo.
        exit /b 0
    )
    echo [ERROR] Failed to clean stale frontend dev processes.
    echo.
    exit /b 1
)
echo.
exit /b 0

:cleanup_dev_binary
echo Removing stale dev binary...
if exist "build\bin\trace-browser-dev.exe" (
    powershell -NoProfile -Command "$p='build\\bin\\trace-browser-dev.exe'; for($i=0;$i -lt 5;$i++){ if(-not (Test-Path $p)){ exit 0 }; Remove-Item -Path $p -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1 }; if(Test-Path $p){ exit 2 } else { exit 0 }"
    if errorlevel 1 (
        echo [ERROR] Cannot remove build\bin\trace-browser-dev.exe.
        echo         End trace-browser-dev.exe in Task Manager and retry.
        exit /b 1
    )
)
if exist "build\bin\trace-browser-dev.exe~" del /F /Q "build\bin\trace-browser-dev.exe~" >nul 2>&1
if exist "build\bin\trace-browser-dev.exe" del /F /Q "build\bin\trace-browser-dev.exe" >nul 2>&1
echo.
exit /b 0

:run_wails3
echo ========================================
echo   Trace Browser - Wails3 Dev Launcher
echo ========================================
echo.
echo Current workdir: %CD%
echo Mode: Wails3 shell
if /I not "%DEV_MODE%"=="stable" (
    echo [WARN] Wails3 live/limited frontend mode is not wired yet; using static frontend assets.
)
echo.

call :cleanup_dev_logs
call :apply_proxy_settings
call :print_proxy_settings

call :ensure_go_command
if errorlevel 1 exit /b 1

call :cleanup_app_processes
call :cleanup_frontend_dev_processes warn
if errorlevel 1 exit /b 1

call :cleanup_dev_binary
if errorlevel 1 exit /b 1

call :resolve_wails3_command
if errorlevel 1 exit /b 1

call :download_go_dependencies
if errorlevel 1 exit /b 1

call :install_frontend_dependencies
if errorlevel 1 exit /b 1

echo Regenerating Wails3 bindings...
call bat\generate-bindings.bat --no-pause --wails3
if errorlevel 1 exit /b 1

call :build_frontend_assets
if errorlevel 1 exit /b 1

echo Building Wails3 dev binary...
powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path 'build/bin' | Out-Null"
go build -o build\bin\trace-browser-dev.exe .
if errorlevel 1 (
    echo [ERROR] Failed to build Wails3 dev binary.
    exit /b 1
)

echo Starting Wails3 dev binary...
echo.
build\bin\trace-browser-dev.exe
exit /b %errorlevel%

:resolve_frontend_dev_port
echo Resolving frontend dev port...
set "FRONTEND_PORT="
for /f "usebackq delims=" %%a in (`node frontend\scripts\dev-port-helper.mjs resolve --preferred %PREFERRED_FRONTEND_PORT%`) do (
    if not defined FRONTEND_PORT set "FRONTEND_PORT=%%a"
)
if not defined FRONTEND_PORT (
    echo [ERROR] Failed to resolve frontend dev port.
    exit /b 1
)
echo [OK] Frontend dev port: %FRONTEND_PORT%
echo.
exit /b 0

:prepare_tooling
call :check_dependencies
if errorlevel 1 exit /b 1

call :download_go_dependencies
if errorlevel 1 exit /b 1

call :install_frontend_dependencies
if errorlevel 1 exit /b 1

call :regenerate_bindings
if errorlevel 1 exit /b 1

exit /b 0

:check_dependencies
echo Checking dependencies...
if not exist "go.mod" (
    echo [ERROR] go.mod not found in repository root.
    exit /b 1
)
if not exist "build\config.yml" (
    echo [ERROR] build\config.yml not found in repository root.
    exit /b 1
)
exit /b 0

:download_go_dependencies
echo Downloading Go dependencies...
go mod download
if errorlevel 1 (
    echo [ERROR] Failed to download Go dependencies.
    exit /b 1
)
exit /b 0

:install_frontend_dependencies
if not exist "frontend\node_modules" (
    echo Installing frontend dependencies...
    pushd frontend
    call npm install
    set "NPM_INSTALL_EXIT_CODE=!errorlevel!"
    popd
    if not "!NPM_INSTALL_EXIT_CODE!"=="0" (
        echo [ERROR] Failed to install frontend dependencies.
        exit /b 1
    )
)
echo.
exit /b 0

:regenerate_bindings
echo Regenerating Wails bindings...
call bat\generate-bindings.bat --no-pause
if errorlevel 1 (
    echo [ERROR] Failed to generate Wails bindings.
    exit /b 1
)
if not exist "frontend\src\wails3" (
    echo [ERROR] Wails bindings output folder not found.
    exit /b 1
)
echo.
exit /b 0

:build_frontend_assets
echo Building frontend static assets...
pushd frontend
call npm run build
set "FRONTEND_BUILD_EXIT_CODE=!errorlevel!"
popd
if not "!FRONTEND_BUILD_EXIT_CODE!"=="0" (
    echo [ERROR] Frontend build failed.
    exit /b 1
)
if not exist "frontend\dist\index.html" (
    echo [ERROR] frontend\dist\index.html was not generated.
    exit /b 1
)
echo.
exit /b 0

:ensure_embed_dist
if not exist "frontend\dist" (
    mkdir "frontend\dist" >nul 2>&1
)
if not exist "frontend\dist\__wails_placeholder__.txt" (
    echo placeholder> "frontend\dist\__wails_placeholder__.txt"
)
if not exist "frontend\dist" (
    echo [ERROR] Failed to prepare frontend\dist for go:embed.
    exit /b 1
)
exit /b 0

:wait_for_frontend_dev_server
powershell -NoProfile -Command "$port=%FRONTEND_PORT%; $pid=%WATCHER_PID%; $deadline=(Get-Date).AddSeconds(20); while((Get-Date) -lt $deadline){ $listener = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue | Select-Object -First 1; if($listener){ exit 0 }; if(-not (Get-Process -Id $pid -ErrorAction SilentlyContinue)){ exit 2 }; Start-Sleep -Milliseconds 500 }; exit 1"
if "%errorlevel%"=="0" (
    echo [OK] Frontend dev server is listening on %FRONTEND_PORT%.
    exit /b 0
)
if "%errorlevel%"=="2" (
    echo [ERROR] Frontend watcher exited before the dev server became ready.
) else (
    echo [ERROR] Timed out waiting for the frontend dev server on port %FRONTEND_PORT%.
)
if exist "tmp-npm-dev.err.log" type "tmp-npm-dev.err.log"
exit /b 1

:cleanup_watcher
if defined WATCHER_PID (
    taskkill /F /T /PID %WATCHER_PID% >nul 2>&1
)
if exist "%LIMITED_WATCHER_PID_FILE%" del /F /Q "%LIMITED_WATCHER_PID_FILE%" >nul 2>&1
node frontend\scripts\dev-port-helper.mjs cleanup >nul 2>&1
set "WATCHER_STARTED=0"
exit /b 0

:start_watcher
echo Starting frontend watcher...
set "WATCHER_PID="
if "%FRONTEND_LIMITED_MODE%"=="1" (
    for /f "usebackq delims=" %%a in (`powershell -NoProfile -Command "$p = Start-Process -FilePath 'powershell.exe' -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-File','scripts/run-limited-frontend-dev.ps1','-WorkingDirectory','%CD%','-MemoryLimitMB','%FRONTEND_PROCESS_MEMORY_LIMIT_MB%','-MaxOldSpaceMB','%FRONTEND_NODE_MAX_OLD_SPACE_SIZE_MB%','-MaxSemiSpaceMB','%FRONTEND_NODE_MAX_SEMI_SPACE_SIZE_MB%','-PidFile','%LIMITED_WATCHER_PID_FILE%') -WorkingDirectory '%CD%' -RedirectStandardOutput 'tmp-npm-dev.log' -RedirectStandardError 'tmp-npm-dev.err.log' -PassThru; Write-Output $p.Id"`) do (
        if not defined WATCHER_PID set "WATCHER_PID=%%a"
    )
) else (
    for /f "usebackq delims=" %%a in (`powershell -NoProfile -Command "$p = Start-Process -FilePath 'node' -ArgumentList @('frontend/scripts/dev-watcher.mjs') -WorkingDirectory '%CD%' -RedirectStandardOutput 'tmp-npm-dev.log' -RedirectStandardError 'tmp-npm-dev.err.log' -PassThru; Write-Output $p.Id"`) do (
        if not defined WATCHER_PID set "WATCHER_PID=%%a"
    )
)
if not defined WATCHER_PID (
    echo [ERROR] Failed to start frontend watcher.
    exit /b 1
)
set "WATCHER_STARTED=1"
echo [OK] Frontend watcher PID: %WATCHER_PID%
echo Watcher logs: tmp-npm-dev.log / tmp-npm-dev.err.log
echo.
exit /b 0

:cleanup_dev_logs
for %%f in (
    "tmp-npm-dev.err.log"
    "tmp-npm-dev.log"
    "tmp-frontend-limited-watcher.pid"
    "tmp-wails-err.log"
    "tmp-wails-out.log"
    "tmp-wails3-err.log"
    "tmp-wails3-out.log"
    "tmp-wails.err"
    "wails-dev-capture.log"
    "wails-dev-run.log"
    "wails-dev-stderr.log"
    "wails-dev-stdout.log"
) do (
    if exist %%~f del /F /Q %%~f >nul 2>&1
)
exit /b 0
