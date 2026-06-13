REM 删除打包文件
del /F /Q route-server.tar.gz
docker rmi route-server:v1.0.0
docker build -t route-server:v1.0.0 .
docker save -o route-server.tar.gz route-server:v1.0.0
