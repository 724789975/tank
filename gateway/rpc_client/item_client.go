package rpc_client

import (
	"context"
	"sync"

	common_config "gate_way_module/config"
	"gate_way_module/etcd"
	"gate_way_module/kitex_gen/item_service/itemservice"
	"gate_way_module/rpc_middleware"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	any1 "github.com/golang/protobuf/ptypes/any"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

var (
	ItemClient itemservice.Client
	once_item  sync.Once
)

func InitItemClient() (err error) {
	once_item.Do(func() {
		ItemClient, err = itemservice.NewClient(
			common_config.Get("item.service_name").(string),
			client.WithResolver(etcd.GetEtcdResolver()),
			client.WithSuite(tracing.NewClientSuite()),
			client.WithMiddleware(rpc_middleware.UserIdClientMiddleware),
		)
		if err != nil {
			klog.Error("[GATEWAY-RPC-ITEM-INIT] Failed to initialize item client: ", err)
			return
		}
		clients[common_config.Get("item.service_name").(string)] = func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
			return callRPC(ctx, ItemClient, rpc_name, body_any)
		}
	})
	return err
}
