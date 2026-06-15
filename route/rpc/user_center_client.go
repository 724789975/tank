package rpc

import (
	"context"
	"sync"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	any1 "github.com/golang/protobuf/ptypes/any"
	common_config "route_module/config"
	"route_module/etcd"
	"route_module/kitex_gen/user_center_service/usercenterservice"
)

var (
	UserCenterClient usercenterservice.Client
	once_user_center sync.Once
)

func InitUserCenterClient() (err error) {
	once_user_center.Do(func() {
		UserCenterClient, err = usercenterservice.NewClient(common_config.Get("user_center.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[ROUTE-RPC-USER_CENTER-INIT] Failed to initialize user_center client: ", err)
			return
		}
		clients[common_config.Get("user_center.service_name").(string)] = func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
			return CallRPC(UserCenterClient, rpc_name, ctx, body_any)
		}
	})
	return err
}