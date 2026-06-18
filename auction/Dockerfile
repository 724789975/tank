FROM golang:1.25 AS builder

WORKDIR /build

ENV GOPROXY=https://goproxy.cn,direct 

# 1. 先复制依赖文件并下载（缓存优化层）
COPY go.mod .
RUN go mod tidy

# 2. 复制完整代码并构建
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go  build -ldflags '-w -s'  -o application && chmod 755 application

FROM alpine:latest

ARG DEPLOYENV=test

WORKDIR /app
# ← 复制生成的go.sum到本地
COPY --from=builder /build/go.sum ./
COPY --from=builder /build/application ./
COPY --from=builder /build/etc/* /app/etc/
ENV  CONF_ENV_PATH /app/etc
ENV  CONF_ENV_FILE server-${DEPLOYENV}.yaml
RUN echo 'Asia/Shanghai' >/etc/timezone
EXPOSE 8080
EXPOSE 11001
CMD ["./application"]