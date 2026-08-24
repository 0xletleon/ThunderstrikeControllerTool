@echo off
chcp 65001 >nul
echo ==========================================
echo   Thunderstrike Controller Tool - Build
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
go build -o tsct.exe .
if errorlevel 1 (
    echo [ERROR] build failed
    pause
    exit /b 1
)

echo [3/3] done
echo.
echo   Output: tsct.exe
echo.
pause
