package rpc

import (
	"context"
	"fmt"

	any1 "github.com/golang/protobuf/ptypes/any"
)

type CB func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any)

var clients = make(map[string]CB)

func InitRpc() {
	if err := InitGateWayClient(); err != nil {
		panic(err)
	}

	if err := InitUserCenterClient(); err != nil {
		panic(err)
	}

	if err := InitMatchClient(); err != nil {
		panic(err)
	}

	if err := InitServerMgrClient(); err != nil {
		panic(err)
	}

	if err := InitItemClient(); err != nil {
		panic(err)
	}

	if err := InitTankGameClient(); err != nil {
		panic(err)
	}

	if err := InitAuctionClient(); err != nil {
		panic(err)
	}
}

func GetClient(serviceName string) (CB, error) {
	if client, ok := clients[serviceName]; ok {
		return client, nil
	}
	return nil, fmt.Errorf("unknown service: %s", serviceName)
}
