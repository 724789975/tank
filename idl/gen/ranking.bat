@echo off
echo [gen/ranking] Generating kitex files for ranking...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\ranking_service\rankingservice

.\bin\kitex -module ranking_module -type protobuf -no-fast-api proto/ranking_service.proto

rmdir /s /q ..\ranking\kitex_gen 2>nul
move .\kitex_gen ..\ranking\

echo [gen/ranking] Done.