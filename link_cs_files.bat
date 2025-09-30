setlocal enabledelayedexpansion
cd client

set CLIENT_DIR=.\Assets\script
set SERVER_DIR=..\server

del /s /q "%SERVER_DIR%\*.cs"

for /r %%f in (*.cs) do (
    set "fullpath=%%f"
    set s=!fullpath:%cd%=!
    echo !s! %%~nxf

    set "server_path=%SERVER_DIR%!s!"
    @REM echo !server_path:%%~nxf=!
    mkdir "!server_path:%%~nxf=!" 
    mklink /h "!server_path!" "!fullpath!"  
)

cd ..