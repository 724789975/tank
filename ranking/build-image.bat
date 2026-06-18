REM 删除打包文件
del /F /Q ranking-server.tar.gz
docker rmi ranking-server:v1.0.0
docker build -t ranking-server:v1.0.0 .
docker save -o ranking-server.tar.gz ranking-server:v1.0.0