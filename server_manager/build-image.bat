REM 删除打包文件
del /F /Q server-mgr.tar.gz
docker rmi server-mgr:v1.0.0
docker build -t server-mgr:v1.0.0 .
docker save -o server-mgr.tar.gz server-mgr:v1.0.0
