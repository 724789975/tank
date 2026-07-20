FROM ubuntu:24.04

WORKDIR /app

# Unity Linux 专用服务器运行所需的基础依赖
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

# 仅拷贝非 CLIENT_WS 版本构建产物（tank.x86_64 及 tank_Data、共享的 UnityPlayer.so）
COPY tank.x86_64 ./
COPY tank_Data ./tank_Data
COPY UnityPlayer.so ./

# 设置时区
ENV TZ=Asia/Shanghai
RUN echo 'Asia/Shanghai' >/etc/timezone
RUN chmod +x ./tank.x86_64

# 游戏服务端口（TCP + UDP，参考 Config.port = 10085）
EXPOSE 10085/tcp
EXPOSE 10085/udp

# 以专用服务器无图形模式启动，日志输出到标准输出
# 运行时可追加参数，例如：
#   docker run --rm -p 10085:10085/tcp -p 10085:10085/udp game-server:v1.0.0 \
#       -service_name 123456 -server_ip <ip> -etcd_addr <addr>
ENTRYPOINT ["./tank.x86_64", "-batchmode", "-nographics", "-logFile", "/dev/stdout"]
