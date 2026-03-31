@echo off
setlocal

set "TOOL_DIR=%~dp0"
set "TOOL_DIR=%TOOL_DIR:~0,-1%"

:: Build binary if not present or if any .go source is newer than the exe
set "NEED_BUILD=0"
if not exist "%TOOL_DIR%\bin\dockerdoor.exe" (
    set "NEED_BUILD=1"
) else (
    powershell -ExecutionPolicy Bypass -Command "$exe = Get-Item '%TOOL_DIR%\bin\dockerdoor.exe'; $newer = Get-ChildItem '%TOOL_DIR%' -Filter '*.go' | Where-Object { $_.LastWriteTime -gt $exe.LastWriteTime }; if ($newer) { exit 1 } else { exit 0 }"
    if errorlevel 1 set "NEED_BUILD=1"
)

if not exist "%TOOL_DIR%\bin" mkdir "%TOOL_DIR%\bin"

:: Copy scripts from scripts/ to bin/
for %%f in ("%TOOL_DIR%\scripts\*") do (
    copy /y "%%f" "%TOOL_DIR%\bin\" >nul
)

if "%NEED_BUILD%"=="1" (
    where go >nul 2>&1
    if errorlevel 1 (
        echo Go is not installed. Please install it from https://go.dev/dl/ and re-run this script.
        exit /b 1
    )
    echo Building dockerdoor.exe...
    pushd "%TOOL_DIR%"
    go build -o bin\dockerdoor.exe .
    if errorlevel 1 (
        echo Build failed.
        popd
        exit /b 1
    )
    popd
    echo Build succeeded.
)

:: Check User PATH (not session PATH) for duplicates
powershell -ExecutionPolicy Bypass -Command "if(([Environment]::GetEnvironmentVariable('PATH','User') -split ';') -contains '%TOOL_DIR%\bin'){exit 0}else{exit 1}"
if %errorlevel% == 0 (
    echo Already in PATH: %TOOL_DIR%
    goto :end
)

:: setx has a 1024-char truncation bug, so use PowerShell to safely append
powershell -ExecutionPolicy Bypass -Command "[Environment]::SetEnvironmentVariable('PATH', [Environment]::GetEnvironmentVariable('PATH','User') + ';%TOOL_DIR%\bin', 'User')"

if %errorlevel% == 0 (
    echo Added to PATH: %TOOL_DIR%
    echo Restart your terminal for the change to take effect.
) else (
    echo Failed to update PATH.
    exit /b 1
)

:end
endlocal
