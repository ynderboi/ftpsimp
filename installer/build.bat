@echo off
setlocal
cd /d "%~dp0\.."

where rsrc >nul 2>&1
if %errorlevel%==0 (
  rsrc -ico assets\ftpsimp.ico -arch amd64 -o rsrc_windows_amd64.syso
) else if exist "%USERPROFILE%\go\bin\rsrc.exe" (
  "%USERPROFILE%\go\bin\rsrc.exe" -ico assets\ftpsimp.ico -arch amd64 -o rsrc_windows_amd64.syso
)

go build -ldflags="-s -w" -o ftpsimp.exe .
if errorlevel 1 exit /b 1
"%LOCALAPPDATA%\Programs\Inno Setup 6\ISCC.exe" "%~dp0ftpsimp.iss"
if errorlevel 1 exit /b 1
echo.
echo Installer: %~dp0..\dist\ftpsimp-setup-1.1.2.exe
