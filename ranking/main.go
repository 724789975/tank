package main

import (
	"ranking_module/config"
	"ranking_module/logic/manager"
	"ranking_module/logic/service"
	"ranking_module/redis"
	"ranking_module/tracer"
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
)

func main() {
	klog.SetLogger(logrus.NewLogger())
	klog.SetLevel(klog.LevelDebug)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config.LoadConfig()
	tracer.InitTracer(config.Get("ranking_rpc.service_name").(string), config.Get("tracer.address").(string))
	redis.GetRedis()
	service.GetRankingService().ListenAndServe(ctx)
	manager.GetRankingManager()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	s := <-quit
	switch s {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP:
		cancel()
	default:
	}
}