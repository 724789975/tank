package rpc

import (
	"sync"
	common_config "user_server/config"
	"user_server/etcd"
	"user_server/kitex_gen/gateway_service/gatewayservice"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
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
			klog.Error("[RPC-GATEWAY-INIT] Failed to initialize gateway client: ", err)
		}
	})
	return err
}
