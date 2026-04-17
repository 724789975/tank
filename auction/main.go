package main

import (
	"auction_module/config"
	"auction_module/logic/manager"
	"auction_module/logic/service"
	"auction_module/redis"
	"auction_module/tracer"
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
	tracer.InitTracer(config.Get("auction_rpc.service_name").(string), config.Get("tracer.address").(string))
	redis.GetRedis()
	service.GetAuctionService().ListenAndServe(ctx)
	manager.GetAuctionManager()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	s := <-quit
	switch s {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP:
		cancel()
	default:
	}
}
