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
	"route_module/kitex_gen/server_mgr_service/servermgrservice"
)

var (
	ServerMgrClient servermgrservice.Client
	once_server_mgr sync.Once
)

func InitServerMgrClient() (err error) {
	once_server_mgr.Do(func() {
		ServerMgrClient, err = servermgrservice.NewClient(common_config.Get("server_mgr.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[ROUTE-RPC-SERVER_MGR-INIT] Failed to initialize server_mgr client: ", err)
			return
		}
		clients[common_config.Get("server_mgr.service_name").(string)] = func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
			return CallRPC(ServerMgrClient, rpc_name, ctx, body_any)
		}
	})
	return err
}