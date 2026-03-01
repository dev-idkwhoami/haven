@echo off
setlocal

set "DATA_DIR=%LOCALAPPDATA%\Haven"

echo.
echo === Haven Client - Full Reset ===
echo.
echo   This will delete ALL client state:
echo   - Identity key (Windows Credential Manager)
echo   - Local profile, saved servers, message history
echo   - Settings and all local data
echo.

set /p CONFIRM="   Are you sure? (y/N): "
if /i not "%CONFIRM%"=="y" (
    echo   Cancelled.
    exit /b 0
)

echo.

:: Delete identity key from Windows Credential Manager
:: go-keyring stores as "service:username" = "haven-client:identity-key"
cmdkey /delete:haven-client:identity-key >nul 2>&1
if %errorlevel% equ 0 (
    echo   [OK]   Identity key removed from Credential Manager
) else (
    echo   [--]   No identity key found in Credential Manager
)

:: Delete data directory
if exist "%DATA_DIR%" (
    rmdir /s /q "%DATA_DIR%"
    echo   [OK]   Data directory deleted (%DATA_DIR%)
) else (
    echo   [--]   Data directory not found
)

echo.
echo   Reset complete. Next launch will be a fresh start.
echo.
