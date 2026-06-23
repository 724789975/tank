package main

import (
	"context"

	"homepage_server/config"
	"homepage_server/etcd"
	"homepage_server/http"
	"homepage_server/logic/service"
	"homepage_server/redis"
	"homepage_server/tracer"

	"github.com/cloudwego/kitex/pkg/klog"
)

func main() {
	config.InitConfig()

	shutdown := tracer.InitTracer()
	defer shutdown()

	redis.InitRedis()

	etcd.InitEtcd()

	http.InitClientRouter()

	ctx := context.Background()

	klog.Info("[HOMEPAGE-SVR-START] starting homepage server...")

	service.GetHomepageService().ListenAndServe(ctx)

	select {}
}
