package rpc

import (
	"context"
	"sync"

	common_config "route_module/config"
	"route_module/etcd"
	"route_module/kitex_gen/tank_game_service/tankgameservice"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	any1 "github.com/golang/protobuf/ptypes/any"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

var (
	TankGameClient tankgameservice.Client
	once_tank_game sync.Once
)

func InitTankGameClient() (err error) {
	once_tank_game.Do(func() {
		TankGameClient, err = tankgameservice.NewClient(common_config.Get("tank_game.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[ROUTE-RPC-TANK_GAME-INIT] Failed to initialize tank_game client: ", err)
			return
		}
		clients[common_config.Get("tank_game.service_name").(string)] = func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
			return CallRPC(TankGameClient, rpc_name, ctx, body_any)
		}
	})
	return err
}
