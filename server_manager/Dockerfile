FROM golang:1.24 AS builder

WORKDIR /build

ENV GOPROXY=https://goproxy.cn,direct 

# 1. 先复制依赖文件并下载（缓存优化层）
COPY go.mod go.sum ./
RUN go mod tidy

# 2. 复制完整代码并构建
COPY . .
# -ldflags '-w -s' 编译优化项(减小镜像大小),加上后将无法使用dlv远程调试可执行文件
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go  build -ldflags '-w -s'  -o application && chmod 755 application

FROM alpine:latest

# 用来标识部署环境,一般为test或prod,
# 对应etc/conf.test.yaml etc/conf.prod.yaml 以此区分线上和测试配置
ARG DEPLOYENV=test

# test2 环境需要通过 docker 命令行控制宿主机 Docker 守护进程来启动 game-server / ai-client,
# 因此仅在 test2 镜像中安装 docker-cli (其它环境使用 k8s API, 无需安装)
RUN if [ "$DEPLOYENV" = "test2" ]; then apk add --no-cache docker-cli; fi

WORKDIR /app
# ← 复制生成的go.sum到本地
COPY --from=builder /build/go.sum ./
COPY --from=builder /build/application ./
COPY --from=builder /build/etc/* /app/etc/
ENV  CONF_ENV_PATH /app/etc
ENV  CONF_ENV_FILE server-${DEPLOYENV}.yaml
# 将 DEPLOYENV 持久化为运行时环境变量, 供 server_manager 判断部署方式:
# test2 -> 使用 Docker 启动 game-server / ai-client, 其它 -> 使用 k8s API
ENV  DEPLOYENV ${DEPLOYENV}
RUN echo 'Asia/Shanghai' >/etc/timezone
#CMD ["tail", "-f", "/dev/null"]
# http端口，固定不变
EXPOSE 8080
EXPOSE 11001
CMD ["./application"]