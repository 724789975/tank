REM 删除打包文件
del /F /Q match-server.tar.gz
docker rmi match-server:v1.0.0
docker build -t match-server:v1.0.0 .
docker save -o match-server.tar.gz match-server:v1.0.0
