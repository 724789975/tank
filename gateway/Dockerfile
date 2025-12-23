FROM golang:1.23 AS builder

WORKDIR /build
ADD . /build
ENV GOPROXY=https://goproxy.cn,direct 
# -ldflags '-w -s' 编译优化项(减小镜像大小),加上后将无法使用dlv远程调试可执行文件
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go  build -ldflags '-w -s'  -o application && chmod 755 application

FROM alpine:latest

# 用来标识部署环境,一般为test或prod,
# 对应etc/conf.test.yaml etc/conf.prod.yaml 以此区分线上和测试配置
ARG DEPLOYENV=test

WORKDIR /app
COPY --from=builder /build/application .
COPY --from=builder /build/etc/* /app/etc/
ENV  CONF_ENV_PATH /app/etc
ENV  CONF_ENV_FILE server-${DEPLOYENV}.yaml
#CMD ["tail", "-f", "/dev/null"]
RUN echo 'Asia/Shanghai' >/etc/timezone
# http端口，固定不变
EXPOSE 8080
EXPOSE 11001
EXPOSE 12001
CMD ["./application"]
