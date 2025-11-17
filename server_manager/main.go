package main

import (
	"context"
	"os"
	"os/signal"
	common_config "server_manager/config"
	"server_manager/logic/service"
	common_redis "server_manager/redis"
	"server_manager/rpc"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
)

func main() {
	klog.SetLogger(logrus.NewLogger())
	klog.SetLevel(klog.LevelDebug)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	common_config.LoadConfig()
	common_redis.GetRedis()
	service.GetServerMgrService().ListenAndServe(ctx)
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
