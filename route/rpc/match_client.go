package rpc

import (
	"context"
	"sync"

	common_config "route_module/config"
	"route_module/etcd"
	"route_module/kitex_gen/match_service/matchservice"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	any1 "github.com/golang/protobuf/ptypes/any"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

var (
	MatchClient matchservice.Client
	once_match  sync.Once
)

func InitMatchClient() (err error) {
	once_match.Do(func() {
		MatchClient, err = matchservice.NewClient(common_config.Get("match.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[ROUTE-RPC-MATCH-INIT] Failed to initialize match client: ", err)
			return
		}
		clients[common_config.Get("match.service_name").(string)] = func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
			return CallRPC(MatchClient, rpc_name, ctx, body_any)
		}
	})
	return err
}
