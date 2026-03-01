@echo off
setlocal enabledelayedexpansion

set "ROOT=%~dp0.."
set "CONFIG=%ROOT%\haven-server.toml"
set "DATA_DIR=%ROOT%\server\data"

echo.
echo === Haven Server - Dependency Check ===
echo.

set ERRORS=0

:: Go
where go >nul 2>&1
if !errorlevel! neq 0 (
    echo   [FAIL] Go not found. Install from https://go.dev/dl/
    set /a ERRORS+=1
    goto :check_gomod
)
for /f "tokens=3" %%v in ('go version') do set "GO_VER=%%v"
echo   [OK]   Go installed (%GO_VER%)

:check_gomod
if exist "%ROOT%\server\go.mod" (
    echo   [OK]   server/go.mod found
) else (
    echo   [FAIL] server/go.mod not found
    set /a ERRORS+=1
)

:: Config file (server auto-generates if missing)
if exist "%CONFIG%" (
    echo   [OK]   Config found
) else (
    echo   [..]   Config not found - server will create a default on startup
)

:check_datadir
if not exist "%DATA_DIR%" mkdir "%DATA_DIR%"
echo   [OK]   Data directory ready

if !ERRORS! gtr 0 (
    echo.
    echo   [FAIL] !ERRORS! dependency issue found. Fix and try again.
    exit /b 1
)

echo.
echo === Building Server ===
echo.

if not exist "%ROOT%\build" mkdir "%ROOT%\build"
if exist "%ROOT%\build\haven-server.exe" (
    del /f "%ROOT%\build\haven-server.exe" >nul 2>&1
    if exist "%ROOT%\build\haven-server.exe" (
        echo   [FAIL] Cannot delete old binary. Is the server still running?
        exit /b 1
    )
)
cd /d "%ROOT%\server"
go build -o "%ROOT%\build\haven-server.exe" .
if !errorlevel! neq 0 (
    echo   [FAIL] Build failed
    exit /b 1
)

echo   [OK]   Built: %ROOT%\build\haven-server.exe

echo.
echo === Starting Server ===
echo.

"%ROOT%\build\haven-server.exe" -config "%CONFIG%" -data "%DATA_DIR%" -verbose
