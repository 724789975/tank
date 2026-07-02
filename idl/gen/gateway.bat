@echo off
echo [gen/gateway] Generating kitex files for gate_way_module...

del go.mod 2>nul
del go.sum 2>nul
rmdir /s /q kitex_gen 2>nul

mkdir kitex_gen\gateway_service\gatewayservice
mkdir kitex_gen\user_center_service\usercenterservice
mkdir kitex_gen\match_service\matchservice
mkdir kitex_gen\tank_game_service\tankgameservice
mkdir kitex_gen\auction_service\auctionservice
mkdir kitex_gen\item_service\itemservice
mkdir kitex_gen\ranking_service\rankingservice
mkdir kitex_gen\server_mgr_service\servermgrservice

.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/gateway_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/user_center_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/match_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/tank_game_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/auction_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/item_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/ranking_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/server_mgr_service.proto

rmdir /s /q ..\gateway\kitex_gen 2>nul
move .\kitex_gen ..\gateway\

echo [gen/gateway] Done.