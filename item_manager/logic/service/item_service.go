package service

import (
	"context"
	common_config "item_manager/config"
	"item_manager/driver"
	"item_manager/etcd"
	item_http "item_manager/http"
	"item_manager/kitex_gen/item"
	"item_manager/kitex_gen/item_service/itemservice"
	"item_manager/logic/manager"
	"net"
	"net/http"
	"sync"

	"github.com/cloudwego/kitex/pkg/klog"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/streaming"
	"github.com/gin-gonic/gin"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

type ItemService struct {
	*kitex.ServiceInfo
}

var (
	item_srv      ItemService
	once_item_srv sync.Once
)

func GetItemService() IService {
	once_item_srv.Do(func() {
		item_srv = ItemService{
			ServiceInfo: itemservice.NewServiceInfo(),
		}
	})
	return &item_srv
}

func (s *ItemService) ListenAndServe(ctx context.Context) {
	clientRouter := item_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0").Group("public").Group("item_manager")
	group.POST(":method", s.ginRoute)

	httpServer := &http.Server{
		Addr:    common_config.Get("item_http.addr").(string),
		Handler: clientRouter,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Errorf("[ITEM-SVR-HTTP-SERVER-START] run server http error:%s", err.Error())
			panic(err)
		}
	}()

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", common_config.Get("item_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := common_config.Get("item_rpc.service_name").(string)

		server := kitexserver.NewServer(
			kitexserver.WithServiceAddr(address),
			kitexserver.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			}),
			kitexserver.WithSuite(tracing.NewServerSuite()),
			kitexserver.WithRegistry(etcd.GetEtcdClient()),
		)

		return server
	}

	ser := NewKitexServer()
	itemservice.RegisterService(ser, s)

	go func() {
		if err := ser.Run(); err != nil {
			klog.Errorf("[ITEM-SVR-RPC-SERVER-START] run server rpc error:%s", err.Error())
			panic(err)
		}
	}()
}

func (s *ItemService) ginRoute(ctx *gin.Context) {
	methodName := ctx.Param("method")

	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(ctx, "[ITEM-SVR-METHOD-NOT-FOUND] not found: %s", ctx.FullPath())
		return
	}

	str := driver.NewHttpStream(ctx.Writer, ctx.Request)
	handler := info.Handler()
	if err := handler(context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(string)), s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(ctx, "[ITEM-SVR-METHOD-HANDLER-ERROR] %s", err.Error())
	}
}

func (s *ItemService) Close() {
}

func (x *ItemService) AddItem(ctx context.Context, req *item.AddItemReq) (resp *item.AddItemRsp, err error) {
	return manager.GetItemManager().AddItem(ctx, req)
}

func (x *ItemService) DeleteItem(ctx context.Context, req *item.DeleteItemReq) (resp *item.DeleteItemRsp, err error) {
	return manager.GetItemManager().DeleteItem(ctx, req)
}

func (x *ItemService) GetAllItems(ctx context.Context, req *item.GetAllItemsReq) (resp *item.GetAllItemsRsp, err error) {
	return manager.GetItemManager().GetAllItems(ctx, req)
}

func (x *ItemService) GetItem(ctx context.Context, req *item.GetItemReq) (resp *item.GetItemRsp, err error) {
	return manager.GetItemManager().GetItem(ctx, req)
}
