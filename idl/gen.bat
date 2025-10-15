.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\common.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\tank_common.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\tank_game.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\user_center.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\gate_way.proto
copy .\proto_gen\*.cs ..\client\Assets\script\proto\
copy .\proto_gen\*.cs ..\server\Assets\script\proto\

@REM .\bin\protoc.exe --go_out=.\proto_gen .\proto\common.proto
@REM .\bin\protoc.exe --go_out=.\proto_gen .\proto\user_center.proto
@REM .\bin\protoc.exe --go_out=.\proto_gen .\proto\user_center_service.proto

@REM rmdir /s /q ..\user_server\proto
@REM move .\proto_gen\user_server\proto ..\user_server\

del go.mod
.\bin\kitex -module user_server -type protobuf -no-fast-api proto/user_center_service.proto
.\bin\kitex -module user_server -type protobuf -no-fast-api proto/gateway_service.proto
rmdir /s /q ..\user_server\kitex_gen
move .\kitex_gen ..\user_server\

del go.mod
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/user_center_service.proto
.\bin\kitex -module gate_way_module -type protobuf -no-fast-api proto/gateway_service.proto
rmdir /s /q ..\gateway\kitex_gen
move .\kitex_gen ..\gateway\
