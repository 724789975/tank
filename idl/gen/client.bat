@echo off
echo [gen/client] Generating C# proto files for client...

.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\common.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\tank_common.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\tank_game.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\user_center.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\gate_way.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\match_proto.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\item.proto
.\bin\protoc.exe --csharp_out=.\proto_gen .\proto\ranking.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\tank_game_service.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\gateway_service.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\user_center_service.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\item_service.proto
.\bin\protoc.exe --csharp_out=.\proto_gen --grpc_out=.\proto_gen --plugin=protoc-gen-grpc=.\bin\grpc_csharp_plugin.exe .\proto\ranking_service.proto

copy .\proto_gen\*.cs ..\client\Assets\script\proto\

echo [gen/client] Done.