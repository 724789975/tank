@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM ============================================================
REM  Unity Android client build script (FxNet)
REM  - Target : Android (APK)
REM  - Net    : FxNet (CLIENT_WS off)
REM  - Scenes : login.unity (main) + match.unity + tank.unity
REM  - Output : build_android/tank.apk (cleaned before build)
REM  - Package name / signing follow Player Settings (same as Editor build).
REM    If a custom keystore is enabled in Player Settings, set env vars
REM    ANDROID_KEYSTORE_PASS (and optional ANDROID_KEYALIAS_PASS) before
REM    running this script; otherwise the build fails instead of silently
REM    falling back to the debug signature.
REM ============================================================

REM Unity editor path (edit if your version/install location differs)
set "UNITY_PATH=C:\Program Files\Unity\Hub\Editor\2022.3.62f3c1\Editor\Unity.exe"

REM Project root (this script lives in build_android, parent is the project root)
set "PROJECT_PATH=%~dp0.."

REM Output dir and APK file
set "OUTPUT_DIR=%~dp0"
set "OUTPUT_PATH=%OUTPUT_DIR%tank.apk"
set "LOG_FILE=%OUTPUT_DIR%build.log"

if not exist "%UNITY_PATH%" (
    echo [ERROR] Unity not found: %UNITY_PATH%
    echo Please edit UNITY_PATH in this script to your Unity 2022.3.62f3c1 install path.
    exit /b 1
)

echo ============================================================
echo  Building Android Client (FxNet)
echo  Project : %PROJECT_PATH%
echo  Output  : %OUTPUT_PATH%
echo  Log     : %LOG_FILE%
echo ============================================================

"%UNITY_PATH%" -quit -batchmode -nographics ^
    -projectPath "%PROJECT_PATH%" ^
    -buildTarget Android ^
    -executeMethod AndroidBuild.BuildAndroid ^
    -androidBuildOutput "%OUTPUT_PATH%" ^
    -logFile "%LOG_FILE%"

set "EXIT_CODE=%ERRORLEVEL%"
if not "%EXIT_CODE%"=="0" (
    echo [ERROR] Build failed with code %EXIT_CODE%. See %LOG_FILE%
    exit /b %EXIT_CODE%
)

echo [OK] Build succeeded: %OUTPUT_PATH%
endlocal
