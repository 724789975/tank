package main

import (
	"context"
	common_config "match_server/config"
	"match_server/logic/match"
	"match_server/logic/service"
	common_redis "match_server/redis"
	"match_server/rpc"
	"match_server/tracer"
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
	common_config.LoadConfig()
	tracer.InitTracer(common_config.Get("match_rpc.service_name").(string), common_config.Get("tracer.address").(string))
	defer tracer.FinitTracer()
	common_redis.GetRedis()
	service.GetMatchService().ListenAndServe(ctx)
	rpc.InitRpc()
	match.GetMatchProcess()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	s := <-quit
	switch s {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP:
		cancel()
	default:
	}
}
