@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM ============================================================
REM  Unity Linux Dedicated Server build script
REM  - Target : StandaloneLinux64 (x86_64)
REM  - Server : StandaloneBuildSubtarget.Server (defines UNITY_SERVER)
REM  - Scene  : Assets/scene/server.unity
REM  - Output : build_ls/tank.x86_64
REM ============================================================

REM Unity editor path (edit if your version/install location differs)
set "UNITY_PATH=C:\Program Files\Unity\Hub\Editor\2022.3.62f3c1\Editor\Unity.exe"

REM Project root (this script lives in build_ls, parent is the project root)
set "PROJECT_PATH=%~dp0.."

REM Output dir and executable (name must match tank.x86_64 in Dockerfile)
set "OUTPUT_DIR=%~dp0"
set "OUTPUT_PATH=%OUTPUT_DIR%tank.x86_64"
set "LOG_FILE=%OUTPUT_DIR%build.log"
set "EXIT_FILE=%OUTPUT_DIR%build.exitcode"
set "BUILD_NAME=Linux Dedicated Server (FxNet)"

REM Child mode: launched below via start /b to run the real Unity build
if "%~1"=="__build__" goto :run_unity

if not exist "%UNITY_PATH%" (
    echo [ERROR] Unity not found: %UNITY_PATH%
    echo Please edit UNITY_PATH in this script to your Unity 2022.3.62f3c1 install path.
    exit /b 1
)

del "%EXIT_FILE%" >nul 2>&1

REM Carriage-return char, used to refresh the single status line in place
for /f %%a in ('copy /Z "%~f0" nul') do set "CR=%%a"

REM One-line build status; progress (elapsed time + Unity stage) refreshes on the same line
<nul set /p "=[BUILD] !BUILD_NAME! - starting Unity...!CR!"
start "" /b "%COMSPEC%" /c call "%~f0" __build__

set /a ELAPSED=0
set "STAGE= starting Unity..."
:wait_build
if exist "%EXIT_FILE%" goto :build_done
ping -n 3 127.0.0.1 >nul
set /a ELAPSED+=2
if exist "%LOG_FILE%" for /f "tokens=1* delims=:" %%A in ('findstr /c:"DisplayProgressbar:" "%LOG_FILE%" 2^>nul') do set "STAGE=%%B"
set "LINE=[BUILD] !BUILD_NAME! !ELAPSED!s -!STAGE!"
set "LINE=!LINE!                                                                                "
<nul set /p "=!LINE:~0,79!!CR!"
goto :wait_build

:build_done
set /p EXIT_CODE=<"%EXIT_FILE%"
del "%EXIT_FILE%" >nul 2>&1
echo(
if not "%EXIT_CODE%"=="0" (
    echo [ERROR] Build failed with code %EXIT_CODE%. See %LOG_FILE%
    exit /b %EXIT_CODE%
)
echo [OK] Build succeeded: %OUTPUT_PATH%
echo Next: run build-image.bat to build the Docker image.
endlocal
exit /b 0

:run_unity
"%UNITY_PATH%" -quit -batchmode -nographics ^
    -projectPath "%PROJECT_PATH%" ^
    -buildTarget Linux64 ^
    -executeMethod ServerBuild.BuildLinuxServer ^
    -serverBuildOutput "%OUTPUT_PATH%" ^
    -logFile "%LOG_FILE%"
>"%EXIT_FILE%" echo %ERRORLEVEL%
exit /b
