REM 删除打包文件
del /F /Q auction-server.tar.gz
docker rmi auction-server:v1.0.0
docker build -t auction-server:v1.0.0 .
docker save -o auction-server.tar.gz auction-server:v1.0.0