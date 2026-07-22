FROM ubuntu:24.04

WORKDIR /app

# Unity Linux 专用服务器运行所需的基础依赖
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

# 拷贝 AI 机器人构建产物（FxNet 或 CLIENT_WS/WebSocketSharp 均为同名 tank.x86_64）
COPY tank.x86_64 ./
COPY tank_Data ./tank_Data
COPY UnityPlayer.so ./

# 设置时区
ENV TZ=Asia/Shanghai
RUN echo 'Asia/Shanghai' >/etc/timezone
RUN chmod +x ./tank.x86_64

# AI 机器人是客户端，主动连接游戏服务器，本身不监听端口，因此无需 EXPOSE。
# 运行时通过命令行参数指定要连接的服务器，例如：
#   docker run --rm ai-client:v1.0.0 \
#       -server_ip <server_ip> -service_name 123456 -etcd_addr <addr>
ENTRYPOINT ["./tank.x86_64", "-batchmode", "-nographics", "-logFile", "/dev/stdout"]
