package rpc

import (
	"context"
	"sync"

	common_config "route_module/config"
	"route_module/etcd"
	"route_module/kitex_gen/gateway_service/gatewayservice"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	any1 "github.com/golang/protobuf/ptypes/any"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

var (
	GatewayClient gatewayservice.Client
	once_gateway  sync.Once
)

func InitGateWayClient() (err error) {
	once_gateway.Do(func() {
		GatewayClient, err = gatewayservice.NewClient(common_config.Get("gateway.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[ROUTE-RPC-GATEWAY-INIT] Failed to initialize gateway client: ", err)
			return
		}
		clients[common_config.Get("gateway.service_name").(string)] = func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
			return CallRPC(GatewayClient, rpc_name, ctx, body_any)
		}
	})
	return err
}
