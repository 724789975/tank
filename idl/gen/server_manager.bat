@echo off
echo [gen/server_manager] Generating kitex files for server_manager...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\server_mgr_service\servermgrservice
mkdir kitex_gen\match_service\matchservice

.\bin\kitex -module server_manager -type protobuf -no-fast-api proto/server_mgr_service.proto
.\bin\kitex -module server_manager -type protobuf -no-fast-api proto/match_service.proto

rmdir /s /q ..\server_manager\kitex_gen 2>nul
move .\kitex_gen ..\server_manager\

echo [gen/server_manager] Done.