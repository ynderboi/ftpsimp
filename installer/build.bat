@echo off
setlocal
cd /d "%~dp0\.."
go build -ldflags="-s -w" -o ftpsimp.exe .
if errorlevel 1 exit /b 1
"%LOCALAPPDATA%\Programs\Inno Setup 6\ISCC.exe" "%~dp0ftpsimp.iss"
if errorlevel 1 exit /b 1
echo.
echo Installer: %~dp0..\dist\ftpsimp-setup-1.0.0.exe
