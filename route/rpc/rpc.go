package rpc

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

	if err := InitRankingClient(); err != nil {
		panic(err)
	}
}

func GetClient(serviceName string) (CB, error) {
	if client, ok := clients[serviceName]; ok {
		return client, nil
	}
	for k, _ := range clients {
		klog.CtxInfof(context.Background(), "[ROUTE-LOG] service_name: %s", k)
	}
	klog.CtxErrorf(context.Background(), "[ROUTE-RPC-CLIENT-NOT-FOUND] Unknown service: %s", serviceName)
	return nil, fmt.Errorf("unknown service: %s", serviceName)
}

type RPCClient interface{}

func callRPC(ctx context.Context, client RPCClient, rpcName string, bodyAny *any1.Any) (error, *any1.Any) {

	clientValue := reflect.ValueOf(client)
	method := clientValue.MethodByName(rpcName)
	if !method.IsValid() {
		klog.CtxErrorf(ctx, "[ROUTE-RPC-METHOD-INVALID] Unknown rpc method: %s, client type: %T", rpcName, client)
		return fmt.Errorf("unknown rpc method: %s", rpcName), nil
	}

	methodType := method.Type()
	if methodType.NumIn() < 2 {
		klog.CtxErrorf(ctx, "[ROUTE-RPC-METHOD-PARAMS] RPC method %s has insufficient parameters, expected at least 2, got %d", rpcName, methodType.NumIn())
		return fmt.Errorf("rpc method %s has insufficient parameters", rpcName), nil
	}

	reqType := methodType.In(1)
	if reqType.Kind() != reflect.Ptr {
		klog.CtxErrorf(ctx, "[ROUTE-RPC-METHOD-PARAM-TYPE] RPC method %s param type must be pointer, got %s", rpcName, reqType.Kind())
		return fmt.Errorf("rpc method %s param type must be pointer", rpcName), nil
	}

	req := reflect.New(reqType.Elem()).Interface()
	if err := bodyAny.UnmarshalTo(req.(proto.Message)); err != nil {
		klog.CtxErrorf(ctx, "[ROUTE-RPC-UNMARSHAL] Failed to unmarshal request for method %s: %v", rpcName, err)
		return fmt.Errorf("unmarshal request failed: %v", err), nil
	}
	klog.CtxInfof(ctx, "[ROUTE-REQUEST] req: %v", req)

	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(req),
	})

	klog.CtxInfof(ctx, "[ROUTE-REQUEST] results: %v", results)

	if len(results) != 2 {
		klog.CtxErrorf(ctx, "[ROUTE-RPC-RETURN-COUNT] RPC method %s has unexpected return count, expected 2, got %d", rpcName, len(results))
		return fmt.Errorf("rpc method %s has unexpected return count", rpcName), nil
	}

	errVal := results[1]
	if !errVal.IsNil() {
		rpcErr := errVal.Interface().(error)
		klog.CtxErrorf(ctx, "[ROUTE-RPC-CALL-FAILED] RPC method %s call failed: %v", rpcName, rpcErr)
		return rpcErr, nil
	}

	rspVal := results[0]
	if rspVal.IsNil() {
		return nil, &any1.Any{}
	}

	anyr := &any1.Any{}
	anyr.MarshalFrom(rspVal.Interface().(proto.Message))
	klog.CtxInfof(ctx, "[ROUTE-REQUEST] rsp: %v", anyr)

	return nil, anyr
}
