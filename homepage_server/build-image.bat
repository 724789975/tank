REM 删除打包文件
del /F /Q homepage-server.tar.gz
docker rmi homepage-server:v1.0.0
docker build -t homepage-server:v1.0.0 .
docker save -o homepage-server.tar.gz homepage-server:v1.0.0