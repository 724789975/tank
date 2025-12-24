package rpc

import (
	common_config "match_server/config"
	"match_server/etcd"
	"match_server/kitex_gen/server_mgr_service/servermgrservice"
	"sync"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

var (
	ServerMgrClient servermgrservice.Client
	once_server_mgr sync.Once
)

func InitServerMgrClient() (err error) {
	once_server_mgr.Do(func() {
		ServerMgrClient, err = servermgrservice.NewClient(common_config.Get("server_mgr.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[MATCH-RPC-SERVER-MGR-INIT] Failed to initialize server_mgr client: ", err)
		}
	})
	return err
}
