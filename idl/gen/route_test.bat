@echo off
echo [gen/route_test] Generating kitex files for route_test...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\user_center_service\usercenterservice

.\bin\kitex -module route_test -type protobuf -no-fast-api proto/user_center_service.proto

rmdir /s /q ..\route_test\kitex_gen 2>nul
move .\kitex_gen ..\route_test\

echo [gen/route_test] Done.