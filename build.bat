@echo off
chcp 65001 >nul 2>&1
setlocal

set VERSION=1.0.0
set BINARY_NAME=datacollector.exe
set BUILD_DIR=dist
set WEB_DIR=web
set EMBED_DIR=internal\web\dist

echo ========================================
echo  DataCollector Build Script
echo ========================================
echo.

:: 1. 检查 Node.js
where node >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Node.js not found, please install Node.js first
    exit /b 1
)

:: 2. 检查 Go
where go >nul 2>&1
if %errorlevel% neq 0 (
    if exist "C:\Go\bin\go.exe" (
        set "PATH=C:\Go\bin;%PATH%"
    ) else (
        echo [ERROR] Go not found, please install Go first
        exit /b 1
    )
)

:: 3. 检查 GCC (CGO)
where gcc >nul 2>&1
if %errorlevel% neq 0 (
    echo [WARN] GCC not found, SQLite requires CGO
    echo [WARN] Please install MinGW-w64: scoop install mingw
    echo [WARN] Trying to build without CGO...
    set CGO_ENABLED=0
) else (
    set CGO_ENABLED=1
)

:: 4. 安装前端依赖
echo [1/5] Installing frontend dependencies...
cd %WEB_DIR%
if not exist node_modules (
    call npm install
    if %errorlevel% neq 0 (
        echo [ERROR] npm install failed
        exit /b 1
    )
)
cd ..

:: 5. 构建前端
echo [2/5] Building frontend...
cd %WEB_DIR%
call npm run build
if %errorlevel% neq 0 (
    echo [ERROR] Frontend build failed
    exit /b 1
)
cd ..

:: 6. 复制前端产物到 Go embed 目录
echo [3/5] Copying frontend assets...
if exist %EMBED_DIR% rmdir /s /q %EMBED_DIR%
xcopy /e /i /q /y %WEB_DIR%\dist %EMBED_DIR%

:: 7. 构建 Go 后端
echo [4/5] Building Go backend (CGO_ENABLED=%CGO_ENABLED%)...
if not exist %BUILD_DIR% mkdir %BUILD_DIR%
go build -ldflags "-s -w -X main.version=%VERSION%" -o %BUILD_DIR%\%BINARY_NAME% ./cmd/server
if %errorlevel% neq 0 (
    echo [ERROR] Go build failed
    exit /b 1
)

:: 8. 完成
echo [5/5] Done!
echo.
echo ========================================
echo  Build successful!
echo  Binary: %BUILD_DIR%\%BINARY_NAME%
echo  Run:    %BUILD_DIR%\%BINARY_NAME%
echo ========================================
