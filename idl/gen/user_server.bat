@echo off
echo [gen/user_server] Generating kitex files for user_server...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\user_center_service\usercenterservice
mkdir kitex_gen\gateway_service\gatewayservice
mkdir kitex_gen\homepage_service\homepageservice

.\bin\kitex -module user_server -type protobuf -no-fast-api proto/user_center_service.proto
.\bin\kitex -module user_server -type protobuf -no-fast-api proto/gateway_service.proto

rmdir /s /q ..\user_server\kitex_gen 2>nul
move .\kitex_gen ..\user_server\

echo [gen/user_server] Done.