REM 删除打包文件
del /F /Q game-server.tar.gz
docker rmi game-server:v1.0.0
docker build -t game-server:v1.0.0 .
docker save -o game-server.tar.gz game-server:v1.0.0
