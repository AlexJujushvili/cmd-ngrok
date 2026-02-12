@echo off
REM Android ARM64 Go build script

REM ცვლილება შენი ფაილის სახელის მიხედვით
SET GOFILE=main.go
SET OUTPUT=app-arm64

REM Cross-compile Android ARM64
SET GOOS=android
SET GOARCH=arm64

echo Building %GOFILE% for Android ARM64...
go build -o %OUTPUT% %GOFILE%

IF %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build succeeded! Output file: %OUTPUT%
pause
