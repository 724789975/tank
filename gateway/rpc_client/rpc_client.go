package rpc_client

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cloudwego/kitex/pkg/klog"
	any1 "github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/proto"
)

type CB func(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any)

var clients = make(map[string]CB)

func InitRpc() {
	if err := InitUserCenterClient(); err != nil {
		panic(err)
	}

	if err := InitMatchClient(); err != nil {
		panic(err)
	}

	if err := InitTankGameClient(); err != nil {
		panic(err)
	}

	if err := InitAuctionClient(); err != nil {
		panic(err)
	}

	if err := InitItemClient(); err != nil {
		panic(err)
	}

	if err := InitRankingClient(); err != nil {
		panic(err)
	}

	if err := InitServerMgrClient(); err != nil {
		panic(err)
	}

}

func GetClient(serviceName string) (CB, error) {
	if client, ok := clients[serviceName]; ok {
		return client, nil
	}
	return nil, fmt.Errorf("unknown service: %s", serviceName)
}

type RPCClient interface{}

func callRPC(ctx context.Context, client RPCClient, rpcName string, bodyAny *any1.Any) (error, *any1.Any) {

	clientValue := reflect.ValueOf(client)
	method := clientValue.MethodByName(rpcName)
	if !method.IsValid() {
		return fmt.Errorf("unknown rpc method: %s", rpcName), nil
	}

	methodType := method.Type()
	if methodType.NumIn() < 2 {
		return fmt.Errorf("rpc method %s has insufficient parameters", rpcName), nil
	}

	reqType := methodType.In(1)
	if reqType.Kind() != reflect.Ptr {
		return fmt.Errorf("rpc method %s param type must be pointer", rpcName), nil
	}

	req := reflect.New(reqType.Elem()).Interface()
	if err := bodyAny.UnmarshalTo(req.(proto.Message)); err != nil {
		return fmt.Errorf("unmarshal request failed: %v", err), nil
	}
	klog.CtxInfof(ctx, "[GATEWAY-RPC-REQUEST] req: %v", req)

	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(req),
	})

	klog.CtxInfof(ctx, "[GATEWAY-RPC-RESPONSE] results: %v", results)

	if len(results) != 2 {
		return fmt.Errorf("rpc method %s has unexpected return count", rpcName), nil
	}

	errVal := results[1]
	if !errVal.IsNil() {
		return errVal.Interface().(error), nil
	}

	anyr := &any1.Any{}
	anyr.MarshalFrom(results[0].Interface().(proto.Message))
	klog.CtxInfof(ctx, "[GATEWAY-RPC-RESPONSE] rsp: %v", anyr)

	return nil, anyr
}
