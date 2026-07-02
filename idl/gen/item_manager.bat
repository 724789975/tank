@echo off
echo [gen/item_manager] Generating kitex files for item_manager...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\item_service\itemservice

.\bin\kitex -module item_manager -type protobuf -no-fast-api proto/item_service.proto

rmdir /s /q ..\item_manager\kitex_gen 2>nul
move .\kitex_gen ..\item_manager\

echo [gen/item_manager] Done.