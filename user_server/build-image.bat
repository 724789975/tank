REM 删除打包文件
del /F /Q user-server.tar.gz
docker rmi user-server:v1.0.0
docker build -t user-server:v1.0.0 .
docker save -o user-server.tar.gz user-server:v1.0.0
