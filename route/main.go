package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	common_config "route_module/config"
	"route_module/logic/service"
	"route_module/rpc"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
)

func main() {
	klog.SetLogger(logrus.NewLogger())
	klog.SetLevel(klog.LevelDebug)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	common_config.LoadConfig()
	rpc.InitRpc()

	service.GetRouteService().ListenAndServe(ctx)

	klog.CtxInfof(ctx, "[ROUTE-SVR-START] route server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	s := <-quit
	switch s {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP:
		klog.CtxInfof(ctx, "[ROUTE-SVR-SHUTDOWN] route server shutting down, signal: %v", s)
		cancel()
	default:
	}
}
