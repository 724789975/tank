REM 删除打包文件
del /F /Q ai-client.tar.gz
docker rmi ai-client:v1.0.0
docker build -t ai-client:v1.0.0 .
docker save -o ai-client.tar.gz ai-client:v1.0.0
