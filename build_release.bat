@echo off
chcp 65001 >nul
echo ==========================================
echo   Thunderstrike Controller Tool - Release
echo ==========================================
echo.

cd /d "%~dp0"

set GOPROXY=https://goproxy.cn,direct

echo [1/3] go mod tidy
go mod tidy
if errorlevel 1 (
    echo [ERROR] go mod tidy failed
    pause
    exit /b 1
)

echo [2/3] go build
go build -ldflags="-s -w" -o Release\tsct.exe .
if errorlevel 1 (
    echo [ERROR] build failed
    pause
    exit /b 1
)

echo [3/3] zip release
if exist "Release\Release.zip" del "Release\Release.zip"
powershell -NoProfile -Command "Compress-Archive -Path 'Release\tsct.exe','Release\blkz\','Release\ext4imgtool\' -DestinationPath 'Release\Release.zip' -Force"
if errorlevel 1 (
    echo [ERROR] zip failed
    pause
    exit /b 1
)

echo.
echo   Release build complete!
echo   exe:  Release\tsct.exe
echo   zip:  Release\Release.zip
echo.
pause
