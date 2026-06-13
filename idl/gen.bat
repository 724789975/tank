.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\common.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\tank_common.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\tank_game.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\user_center.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\gate_way.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\match_proto.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\item.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\tank_game_service.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\gateway_service.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\user_center_service.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\item_service.proto

copy .\proto_gen\*.cs ..\client\Assets\script\proto\
@REM copy .\proto_gen\*.cs ..\server\Assets\script\proto\

@REM .\bin\protoc.exe --go_out=.\proto_gen .\proto\common.proto
@REM .\bin\protoc.exe --go_out=.\proto_gen .\proto\user_center.proto
@REM .\bin\protoc.exe --go_out=.\proto_gen .\proto\user_center_service.proto

@REM rmdir /s /q ..\user_server\proto
@REM move .\proto_gen\user_server\proto ..\user_server\

@REM delete previous generated files and go modules
del go.mod
del go.sum
rmdir /s /q kitex_gen

@REM Pre-create nested directories for service clients (workaround for kitex Windows bug)
mkdir kitex_gen\user_center_service\usercenterservice
mkdir kitex_gen\gateway_service\gatewayservice

@REM Generate user_server
.\bin\kitex -module user_server -type protobuf -no-fast-api proto/user_center_service.proto
.\bin\kitex -module user_server -type protobuf -no-fast-api proto/gateway_service.proto
rmdir /s /q ..\user_server\kitex_gen
move .\kitex_gen ..\user_server\

@REM delete previous generated files and go modules
del go.mod
del go.sum
rmdir /s /q kitex_gen

@REM Pre-create nested directories for service clients
mkdir kitex_gen\gateway_service\gatewayservice

@REM Generate gateway
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/gateway_service.proto
rmdir /s /q ..\gateway\kitex_gen
move .\kitex_gen ..\gateway\

@REM delete previous generated files and go modules
del go.mod
del go.sum
rmdir /s /q kitex_gen

@REM Pre-create nested directories for service clients
mkdir kitex_gen\gateway_service\gatewayservice
mkdir kitex_gen\match_service\matchservice
mkdir kitex_gen\server_mgr_service\servermgrservice

@REM Generate match_server
.\bin\kitex -module match_server -type protobuf -no-fast-api proto/gateway_service.proto
.\bin\kitex -module match_server -type protobuf -no-fast-api proto/match_service.proto
.\bin\kitex -module match_server -type protobuf -no-fast-api proto/server_mgr_service.proto
rmdir /s /q ..\match_server\kitex_gen
move .\kitex_gen ..\match_server\

@REM delete previous generated files and go modules
del go.mod
del go.sum
rmdir /s /q kitex_gen

@REM Pre-create nested directories for service clients
mkdir kitex_gen\server_mgr_service\servermgrservice
mkdir kitex_gen\match_service\matchservice

@REM Generate server_manager
.\bin\kitex -module server_manager -type protobuf -no-fast-api proto/server_mgr_service.proto
.\bin\kitex -module server_manager -type protobuf -no-fast-api proto/match_service.proto
rmdir /s /q ..\server_manager\kitex_gen
move .\kitex_gen ..\server_manager\

@REM delete previous generated files and go modules
del go.mod
del go.sum
rmdir /s /q kitex_gen

@REM Pre-create nested directories for service clients
mkdir kitex_gen\item_service\itemservice

@REM Generate item_manager
.\bin\kitex -module item_manager -type protobuf -no-fast-api proto/item_service.proto
rmdir /s /q ..\item_manager\kitex_gen
move .\kitex_gen ..\item_manager\

@REM delete previous generated files and go modules
del go.mod
del go.sum
rmdir /s /q kitex_gen

@REM Pre-create nested directories for service clients
mkdir kitex_gen\auction_service\auctionservice

@REM Generate auction
.\bin\kitex -module auction_module -type protobuf -no-fast-api proto/auction_service.proto
rmdir /s /q ..\auction\kitex_gen
move .\kitex_gen ..\auction\

@REM delete previous generated files and go modules
del go.mod
del go.sum
rmdir /s /q kitex_gen

@REM Pre-create nested directories for service clients
mkdir kitex_gen\gateway_service\gatewayservice
mkdir kitex_gen\user_center_service\usercenterservice
mkdir kitex_gen\tank_game_service\tankgameservice
mkdir kitex_gen\match_service\matchservice
mkdir kitex_gen\server_mgr_service\servermgrservice
mkdir kitex_gen\item_service\itemservice
mkdir kitex_gen\auction_service\auctionservice

@REM Generate route_module
.\bin\kitex -module route_module -type protobuf -no-fast-api proto/gateway_service.proto
.\bin\kitex -module route_module -type protobuf -no-fast-api proto/user_center_service.proto
.\bin\kitex -module route_module -type protobuf -no-fast-api proto/tank_game_service.proto
.\bin\kitex -module route_module -type protobuf -no-fast-api proto/match_service.proto
.\bin\kitex -module route_module -type protobuf -no-fast-api proto/server_mgr_service.proto
.\bin\kitex -module route_module -type protobuf -no-fast-api proto/item_service.proto
.\bin\kitex -module route_module -type protobuf -no-fast-api proto/auction_service.proto
rmdir /s /q ..\route\kitex_gen
move .\kitex_gen ..\route\
