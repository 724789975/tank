FROM x-project-cn-xf-ga-01-vecps.cr.cloud.xunfeng.cn/xf-x/go-1.22-build:v0.0.2 AS builder

WORKDIR /build
ADD . /build
# -ldflags '-w -s' 编译优化项(减小镜像大小),加上后将无法使用dlv远程调试可执行文件
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go  build -ldflags '-w -s'  -o application && chmod 755 application

# 目前使用镜像alpine:3.20.3
FROM x-project-cn-xf-ga-01-vecps.cr.cloud.xunfeng.cn/xf-x/alpine:v0.0.2

# 用来标识部署环境,一般为test或prod,
# 对应etc/conf.test.yaml etc/conf.prod.yaml 以此区分线上和测试配置
ARG DEPLOYENV

WORKDIR /app
COPY --from=builder /build/application .
COPY --from=builder /build/etc/server-${DEPLOYENV}.yaml .
ENV  XF_X_CONF_PATH /app
ENV  XF_X_CONF_FILE server-${DEPLOYENV}.yaml
#CMD ["tail", "-f", "/dev/null"]
# http端口，固定不变
EXPOSE 10002
EXPOSE 10001
CMD ["./application"]