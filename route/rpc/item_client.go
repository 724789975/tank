package rpc

import (
	"context"
	"sync"

	common_config "route_module/config"
	"route_module/etcd"
	"route_module/kitex_gen/item_service/itemservice"

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
		ItemClient, err = itemservice.NewClient(common_config.Get("item.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[ROUTE-RPC-ITEM-INIT] Failed to initialize item client: ", err)
			return
		}
		clients[common_config.Get("item.service_name").(string)] = func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
			return CallRPC(ItemClient, rpc_name, ctx, body_any)
		}
	})
	return err
}
