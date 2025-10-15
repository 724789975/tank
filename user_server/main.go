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
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	common_config.LoadConfig()
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
