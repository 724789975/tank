@echo off
echo [gen/homepage_server] Generating kitex files for homepage_server...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\homepage_service\homepageservice

.\bin\kitex -module homepage_server -type protobuf -no-fast-api proto/homepage_service.proto

rmdir /s /q ..\homepage_server\kitex_gen 2>nul
move .\kitex_gen ..\homepage_server\

echo [gen/homepage_server] Done.