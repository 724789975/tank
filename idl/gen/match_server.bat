@echo off
echo [gen/match_server] Generating kitex files for match_server...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\gateway_service\gatewayservice
mkdir kitex_gen\match_service\matchservice
mkdir kitex_gen\server_mgr_service\servermgrservice

.\bin\kitex -module match_server -type protobuf -no-fast-api proto/gateway_service.proto
.\bin\kitex -module match_server -type protobuf -no-fast-api proto/match_service.proto
.\bin\kitex -module match_server -type protobuf -no-fast-api proto/server_mgr_service.proto

rmdir /s /q ..\match_server\kitex_gen 2>nul
move .\kitex_gen ..\match_server\

echo [gen/match_server] Done.