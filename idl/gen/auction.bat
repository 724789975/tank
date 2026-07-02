@echo off
echo [gen/auction] Generating kitex files for auction...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\auction_service\auctionservice

.\bin\kitex -module auction_module -type protobuf -no-fast-api proto/auction_service.proto

rmdir /s /q ..\auction\kitex_gen 2>nul
move .\kitex_gen ..\auction\

echo [gen/auction] Done.