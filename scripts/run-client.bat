@echo off
setlocal enabledelayedexpansion

set "ROOT=%~dp0.."
set "FRONTEND_DIR=%ROOT%\client\frontend"

:: Auto-detect MSYS2 MinGW64 and prepend to PATH
if exist "C:\msys64\mingw64\bin" (
    set "PATH=C:\msys64\mingw64\bin;!PATH!"
)

:: Auto-detect GOPATH/bin for wails and other Go-installed tools
for /f "delims=" %%g in ('go env GOPATH 2^>nul') do set "GOPATH_BIN=%%g\bin"
if defined GOPATH_BIN if exist "!GOPATH_BIN!" (
    set "PATH=!GOPATH_BIN!;!PATH!"
)

echo.
echo === Haven Client - Dependency Check ===
echo.

set ERRORS=0
set WARNINGS=0

:: Go
where go >nul 2>&1
if !errorlevel! neq 0 (
    echo   [FAIL] Go not found. Install from https://go.dev/dl/
    set /a ERRORS+=1
    goto :check_gcc
)
for /f "tokens=3" %%v in ('go version') do set "GO_VER=%%v"
echo   [OK]   Go installed (%GO_VER%)

:check_gcc
where gcc >nul 2>&1
if !errorlevel! neq 0 (
    echo   [FAIL] GCC not found. Install MSYS2 + mingw-w64-x86_64-gcc
    echo          https://www.msys2.org/
    set /a ERRORS+=1
    goto :check_pkgconfig
)
for /f "delims=" %%v in ('gcc -dumpversion 2^>nul') do set "GCC_VER=%%v"
echo   [OK]   GCC found (%GCC_VER%)

:: Test GCC actually compiles
echo int main(){return 0;} > "%TEMP%\haven_test.c"
gcc "%TEMP%\haven_test.c" -o "%TEMP%\haven_test.exe" >nul 2>&1
if !errorlevel! neq 0 (
    echo   [FAIL] GCC cannot compile. Reinstall via MSYS2.
    set /a ERRORS+=1
) else (
    echo   [OK]   GCC compiles successfully
)
del "%TEMP%\haven_test.c" "%TEMP%\haven_test.exe" >nul 2>&1

:check_pkgconfig
set "PKGCFG="
where pkg-config >nul 2>&1
if !errorlevel! equ 0 (
    set "PKGCFG=pkg-config"
    echo   [OK]   pkg-config found
    goto :check_libs
)
where pkgconf >nul 2>&1
if !errorlevel! equ 0 (
    set "PKGCFG=pkgconf"
    echo   [OK]   pkgconf found
    goto :check_libs
)
echo   [!!]   pkg-config not found
echo          MSYS2: pacman -S mingw-w64-x86_64-pkgconf
set /a WARNINGS+=1
goto :check_node

:check_libs
:: PortAudio
call :check_lib "PortAudio" "portaudio-2.0" "pacman -S mingw-w64-x86_64-portaudio"
:: Opus
call :check_lib "Opus" "opus" "pacman -S mingw-w64-x86_64-opus"
:: SQLite3
call :check_lib "SQLite3" "sqlite3" "pacman -S mingw-w64-x86_64-sqlite3"

:check_node
where node >nul 2>&1
if !errorlevel! neq 0 (
    echo   [FAIL] Node.js not found. Install from https://nodejs.org/
    set /a ERRORS+=1
    goto :check_npm
)
for /f "delims=" %%v in ('node --version') do set "NODE_VER=%%v"
echo   [OK]   Node.js installed (%NODE_VER%)

:check_npm
where npm >nul 2>&1
if !errorlevel! neq 0 (
    echo   [FAIL] npm not found
    set /a ERRORS+=1
    goto :check_wails
)
for /f "delims=" %%v in ('npm --version') do set "NPM_VER=%%v"
echo   [OK]   npm installed (v%NPM_VER%)

:check_wails
set "WAILS_CMD="
where wails.exe >nul 2>&1
if !errorlevel! equ 0 (
    set "WAILS_CMD=wails"
    goto :wails_ok
)
if defined GOPATH_BIN if exist "!GOPATH_BIN!\wails.exe" (
    set "WAILS_CMD=!GOPATH_BIN!\wails.exe"
    goto :wails_ok
)
echo   [FAIL] Wails CLI not found (required for desktop app build)
echo          Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest
set /a ERRORS+=1
goto :done_checks
:wails_ok
echo   [OK]   Wails CLI found

:done_checks
echo.
echo ---
if !ERRORS! gtr 0 (
    echo   [FAIL] !ERRORS! dependency issue found. Fix and try again.
    exit /b 1
)
if !WARNINGS! gtr 0 (
    echo   [!!]   !WARNINGS! non-critical warning. Build may still work.
)

echo.
echo === Installing Frontend Dependencies ===
echo.

if not exist "%FRONTEND_DIR%\package.json" (
    echo   [!!]   No package.json found at %FRONTEND_DIR%
    goto :build_client
)
if exist "%FRONTEND_DIR%\node_modules" (
    echo   [OK]   Frontend dependencies already installed
    goto :build_client
)
cd /d "%FRONTEND_DIR%"
call npm install
if !errorlevel! neq 0 (
    echo   [FAIL] npm install failed
    exit /b 1
)
echo   [OK]   Frontend dependencies installed
cd /d "%ROOT%"

:build_client
echo.
echo === Starting Client (wails dev) ===
echo.

cd /d "%ROOT%\client"
set CGO_ENABLED=1

if exist "%ROOT%\client\build\bin\haven-client.exe" (
    del /f "%ROOT%\client\build\bin\haven-client.exe" >nul 2>&1
    if exist "%ROOT%\client\build\bin\haven-client.exe" (
        echo   [FAIL] Cannot delete old binary. Is the client still running?
        exit /b 1
    )
)

echo   Using: !WAILS_CMD!
"!WAILS_CMD!" dev
exit /b 0

:check_lib
set "LIB_NAME=%~1"
set "PKG_NAME=%~2"
set "INSTALL_HINT=%~3"
%PKGCFG% --exists %PKG_NAME% >nul 2>&1
if !errorlevel! neq 0 (
    echo   [FAIL] %LIB_NAME% not found
    echo          MSYS2: %INSTALL_HINT%
    set /a ERRORS+=1
    exit /b 0
)
for /f "delims=" %%v in ('%PKGCFG% --modversion %PKG_NAME% 2^>nul') do set "LIB_VER=%%v"
echo   [OK]   %LIB_NAME% (%LIB_VER%)
exit /b 0
