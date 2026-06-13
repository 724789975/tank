package rpc

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	common_config "route_module/config"
	"route_module/etcd"
	"route_module/kitex_gen/auction_service/auctionservice"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/klog"
	any1 "github.com/golang/protobuf/ptypes/any"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"google.golang.org/protobuf/proto"
)

var (
	AuctionClient auctionservice.Client
	once_auction  sync.Once
)

func InitAuctionClient() (err error) {
	once_auction.Do(func() {
		AuctionClient, err = auctionservice.NewClient(common_config.Get("auction.service_name").(string), client.WithResolver(etcd.GetEtcdResolver()), client.WithSuite(tracing.NewClientSuite()))
		if err != nil {
			klog.Error("[ROUTE-RPC-AUCTION-INIT] Failed to initialize auction client: ", err)
			return
		}
		clients[common_config.Get("auction.service_name").(string)] = auctionClientWrapper
	})
	return err
}

func auctionClientWrapper(ctx context.Context, rpc_name string, body_any *any1.Any) (error, *any1.Any) {
	// 获取 client 的反射值
	clientValue := reflect.ValueOf(AuctionClient)
	method := clientValue.MethodByName(rpc_name)
	if !method.IsValid() {
		return fmt.Errorf("unknown rpc method: %s", rpc_name), nil
	}

	// 获取方法的参数类型（第二个参数是请求类型）
	methodType := method.Type()
	if methodType.NumIn() < 2 {
		return fmt.Errorf("rpc method %s has insufficient parameters", rpc_name), nil
	}

	// 获取请求参数类型（索引 1，因为索引 0 是 context）
	reqType := methodType.In(1)
	if reqType.Kind() != reflect.Ptr {
		return fmt.Errorf("rpc method %s param type must be pointer", rpc_name), nil
	}

	// 创建请求对象实例
	req := reflect.New(reqType.Elem()).Interface()
	if err := proto.Unmarshal(body_any.GetValue(), req.(proto.Message)); err != nil {
		return fmt.Errorf("unmarshal request failed: %v", err), nil
	}

	// 调用方法
	results := method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(req),
	})

	// 处理返回值：第一个是响应，第二个是 error
	if len(results) != 2 {
		return fmt.Errorf("rpc method %s has unexpected return count", rpc_name), nil
	}

	// 检查 error
	errVal := results[1]
	if !errVal.IsNil() {
		return errVal.Interface().(error), nil
	}

	// 检查响应
	rspVal := results[0]
	if rspVal.IsNil() {
		return nil, &any1.Any{}
	}

	// 序列化响应
	rspBytes, err := proto.Marshal(rspVal.Interface().(proto.Message))
	if err != nil {
		return fmt.Errorf("marshal response failed: %v", err), nil
	}

	return nil, &any1.Any{Value: rspBytes}
}
