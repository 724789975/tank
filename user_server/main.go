package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	common_config "user_server/config"
	"user_server/logic/service"
	common_redis "user_server/redis"
	"user_server/rpc"
	"user_server/tracer"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
)

func main() {
	klog.SetLogger(logrus.NewLogger())
	klog.SetLevel(klog.LevelDebug)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	common_config.LoadConfig()
	tracer.InitTracer(common_config.Get("user_rpc.service_name").(string), common_config.Get("tracer.address").(string))
	common_redis.GetRedis()
	service.GetUserService().ListenAndServe(ctx)
	rpc.InitRpc()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	s := <-quit
	switch s {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP:
		cancel()
	default:
	}
}
