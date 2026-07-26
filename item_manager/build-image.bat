REM 删除打包文件
del /F /Q item-manager.tar.gz
docker rmi item-manager:v1.0.0
docker build -t item-manager:v1.0.0 .
docker save -o item-manager.tar.gz item-manager:v1.0.0
