REM 删除打包文件
del /F /Q gate-server.tar.gz
docker rmi gate-server:v1.0.0
docker build -t gate-server:v1.0.0 .
docker save -o gate-server.tar.gz gate-server:v1.0.0
