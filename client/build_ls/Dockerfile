FROM ubuntu:24.04

WORKDIR /app

COPY . .

RUN echo 'Asia/Shanghai' >/etc/timezone
RUN chmod +x ./tank.x86_64

EXPOSE 10085
