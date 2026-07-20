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

if not exist "%UNITY_PATH%" (
    echo [ERROR] Unity not found: %UNITY_PATH%
    echo Please edit UNITY_PATH in this script to your Unity 2022.3.62f3c1 install path.
    exit /b 1
)

echo ============================================================
echo  Building Linux Dedicated Server
echo  Project : %PROJECT_PATH%
echo  Output  : %OUTPUT_PATH%
echo  Log     : %LOG_FILE%
echo ============================================================

"%UNITY_PATH%" -quit -batchmode -nographics ^
    -projectPath "%PROJECT_PATH%" ^
    -buildTarget Linux64 ^
    -executeMethod ServerBuild.BuildLinuxServer ^
    -serverBuildOutput "%OUTPUT_PATH%" ^
    -logFile "%LOG_FILE%"

set "EXIT_CODE=%ERRORLEVEL%"
if not "%EXIT_CODE%"=="0" (
    echo [ERROR] Build failed with code %EXIT_CODE%. See %LOG_FILE%
    exit /b %EXIT_CODE%
)

echo [OK] Build succeeded: %OUTPUT_PATH%
echo Next: run build-image.bat to build the Docker image.
endlocal
