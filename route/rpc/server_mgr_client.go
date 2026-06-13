package rpc

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	any1 "github.com/golang/protobuf/ptypes/any"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"google.golang.org/protobuf/proto"
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
		clients[common_config.Get("server_mgr.service_name").(string)] = serverMgrClientWrapper
	})
	return err
}

func serverMgrClientWrapper(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
	clientValue := reflect.ValueOf(ServerMgrClient)
	method := clientValue.MethodByName(rpc_name)
	if !method.IsValid() {
		return fmt.Errorf("unknown rpc method: %s", rpc_name), nil
	}

	methodType := method.Type()
	if methodType.NumIn() < 2 {
		return fmt.Errorf("rpc method %s has insufficient parameters", rpc_name), nil
	}

	reqType := methodType.In(1)
	if reqType.Kind() != reflect.Ptr {
		return fmt.Errorf("rpc method %s param type must be pointer", rpc_name), nil
	}

	req := reflect.New(reqType.Elem()).Interface()
	if err := proto.Unmarshal(body_any.GetValue(), req.(proto.Message)); err != nil {
		return fmt.Errorf("unmarshal request failed: %v", err), nil
	}

	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(req),
	})

	if len(results) != 2 {
		return fmt.Errorf("rpc method %s has unexpected return count", rpc_name), nil
	}

	errVal := results[1]
	if !errVal.IsNil() {
		return errVal.Interface().(error), nil
	}

	rspVal := results[0]
	if rspVal.IsNil() {
		return nil, &any1.Any{}
	}

	rspBytes, err := proto.Marshal(rspVal.Interface().(proto.Message))
	if err != nil {
		return fmt.Errorf("marshal response failed: %v", err), nil
	}

	return nil, &any1.Any{Value: rspBytes}
}