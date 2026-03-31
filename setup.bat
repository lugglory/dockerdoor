@echo off
setlocal

set "TOOL_DIR=%~dp0"
set "TOOL_DIR=%TOOL_DIR:~0,-1%"

:: Build binary if not present
if not exist "%TOOL_DIR%\dockerdoor.exe" (
    where go >nul 2>&1
    if errorlevel 1 (
        echo Go is not installed. Please install it from https://go.dev/dl/ and re-run this script.
        exit /b 1
    )
    echo Building dockerdoor.exe...
    pushd "%TOOL_DIR%"
    go build -o dockerdoor.exe .
    if errorlevel 1 (
        echo Build failed.
        popd
        exit /b 1
    )
    popd
    echo Build succeeded.
)

:: Check User PATH (not session PATH) for duplicates
powershell -ExecutionPolicy Bypass -Command "if(([Environment]::GetEnvironmentVariable('PATH','User') -split ';') -contains '%TOOL_DIR%'){exit 0}else{exit 1}"
if %errorlevel% == 0 (
    echo Already in PATH: %TOOL_DIR%
    goto :end
)

:: setx has a 1024-char truncation bug, so use PowerShell to safely append
powershell -ExecutionPolicy Bypass -Command "[Environment]::SetEnvironmentVariable('PATH', [Environment]::GetEnvironmentVariable('PATH','User') + ';%TOOL_DIR%', 'User')"

if %errorlevel% == 0 (
    echo Added to PATH: %TOOL_DIR%
    echo Restart your terminal for the change to take effect.
) else (
    echo Failed to update PATH.
    exit /b 1
)

:end
endlocal
