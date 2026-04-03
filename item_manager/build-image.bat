@echo off
setlocal

set IMAGE_NAME=item_manager:latest
set DOCKERFILE_PATH=./Dockerfile

echo Building Docker image: %IMAGE_NAME%
docker build -t %IMAGE_NAME% -f %DOCKERFILE_PATH% .

if %ERRORLEVEL% equ 0 (
    echo Docker image built successfully: %IMAGE_NAME%
) else (
    echo Failed to build Docker image
    exit /b 1
)

endlocal