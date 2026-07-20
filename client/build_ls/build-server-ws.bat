@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM ============================================================
REM  Unity Linux Dedicated Server build script (CLIENT_WS)
REM  - Target : StandaloneLinux64 (x86_64)
REM  - Server : StandaloneBuildSubtarget.Server (defines UNITY_SERVER)
REM  - Extra  : CLIENT_WS (WebSocketSharp network stack)
REM  - Scene  : Assets/scene/server.unity
REM  - Output : build_ls/tank.x86_64 (shared with the non-CLIENT_WS build; cleaned before build)
REM ============================================================

REM Unity editor path (edit if your version/install location differs)
set "UNITY_PATH=C:\Program Files\Unity\Hub\Editor\2022.3.62f3c1\Editor\Unity.exe"

REM Project root (this script lives in build_ls, parent is the project root)
set "PROJECT_PATH=%~dp0.."

REM Output dir and executable (shared with the non-CLIENT_WS build; old artifacts are cleaned first)
set "OUTPUT_DIR=%~dp0"
set "OUTPUT_PATH=%OUTPUT_DIR%tank.x86_64"
set "LOG_FILE=%OUTPUT_DIR%build_ws.log"

if not exist "%UNITY_PATH%" (
    echo [ERROR] Unity not found: %UNITY_PATH%
    echo Please edit UNITY_PATH in this script to your Unity 2022.3.62f3c1 install path.
    exit /b 1
)

echo ============================================================
echo  Building Linux Dedicated Server (CLIENT_WS / WebSocketSharp)
echo  Project : %PROJECT_PATH%
echo  Output  : %OUTPUT_PATH%
echo  Log     : %LOG_FILE%
echo ============================================================

"%UNITY_PATH%" -quit -batchmode -nographics ^
    -projectPath "%PROJECT_PATH%" ^
    -buildTarget Linux64 ^
    -executeMethod ServerBuild.BuildLinuxServerWS ^
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
