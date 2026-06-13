FROM golang:1.25 AS builder

WORKDIR /build
ADD . /build
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags '-w -s' -o application && chmod 755 application

FROM alpine:latest

ARG DEPLOYENV=dev

WORKDIR /app
COPY --from=builder /build/application .
COPY --from=builder /build/etc/* /app/etc/
ENV CONF_ENV_PATH /app/etc
ENV CONF_ENV_FILE server-${DEPLOYENV}.yaml
RUN echo 'Asia/Shanghai' >/etc/timezone

EXPOSE 8080
CMD ["./application"]
