package service

import (
	"auction_module/config"
	"auction_module/driver"
	"auction_module/etcd"
	auction_http "auction_module/http"
	"auction_module/kitex_gen/auction"
	"auction_module/kitex_gen/auction_service/auctionservice"
	"auction_module/logic/manager"
	"context"
	"net"
	nethttp "net/http"
	"sync"

	"github.com/cloudwego/kitex/pkg/klog"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/streaming"
	"github.com/gin-gonic/gin"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

type AuctionService struct {
	*kitex.ServiceInfo
}

var (
	auction_srv      AuctionService
	once_auction_srv sync.Once
)

func GetAuctionService() IService {
	once_auction_srv.Do(func() {
		auction_srv = AuctionService{
			ServiceInfo: auctionservice.NewServiceInfo(),
		}
	})
	return &auction_srv
}

func (s *AuctionService) ListenAndServe(ctx context.Context) {
	clientRouter := auction_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0").Group("public").Group("auction")
	group.POST(":method", s.ginRoute)

	httpServer := &nethttp.Server{
		Addr:    config.Get("auction_http.addr").(string),
		Handler: clientRouter,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
			klog.Errorf("[AUCTION-SVR-HTTP-SERVER-START] run server http error:%s", err.Error())
			panic(err)
		}
	}()

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", config.Get("auction_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := config.Get("auction_rpc.service_name").(string)

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
	auctionservice.RegisterService(ser, s)

	go func() {
		if err := ser.Run(); err != nil {
			klog.Errorf("[AUCTION-SVR-RPC-SERVER-START] run server rpc error:%s", err.Error())
			panic(err)
		}
	}()
}

func (s *AuctionService) ginRoute(ctx *gin.Context) {
	methodName := ctx.Param("method")

	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(ctx, "[AUCTION-SVR-METHOD-NOT-FOUND] not found: %s", ctx.FullPath())
		return
	}

	str := driver.NewHttpStream(ctx.Writer, ctx.Request)
	handler := info.Handler()
	if err := handler(context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(string)), s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-SVR-METHOD-HANDLER-ERROR] %s", err.Error())
	}
}

func (s *AuctionService) Close() {
}

func (x *AuctionService) Ping(ctx context.Context, req *auction.PingReq) (resp *auction.PingRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.Ping(ctx, req)
}

func (x *AuctionService) Sell(ctx context.Context, req *auction.SellReq) (resp *auction.SellRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.Sell(ctx, req)
}

func (x *AuctionService) Buy(ctx context.Context, req *auction.BuyReq) (resp *auction.BuyRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.Buy(ctx, req)
}

func (x *AuctionService) CancelSell(ctx context.Context, req *auction.CancelSellReq) (resp *auction.CancelSellRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.CancelSell(ctx, req)
}

func (x *AuctionService) CancelBuy(ctx context.Context, req *auction.CancelBuyReq) (resp *auction.CancelBuyRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.CancelBuy(ctx, req)
}

func (x *AuctionService) GetMySells(ctx context.Context, req *auction.GetMySellsReq) (resp *auction.GetMySellsRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.GetMySells(ctx, req)
}

func (x *AuctionService) GetMyBuys(ctx context.Context, req *auction.GetMyBuysReq) (resp *auction.GetMyBuysRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.GetMyBuys(ctx, req)
}

func (x *AuctionService) GetItemAuctionInfo(ctx context.Context, req *auction.GetItemAuctionInfoReq) (resp *auction.GetItemAuctionInfoRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.GetItemAuctionInfo(ctx, req)
}

func (x *AuctionService) GetTransactionHistory(ctx context.Context, req *auction.GetTransactionHistoryReq) (resp *auction.GetTransactionHistoryRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.GetTransactionHistory(ctx, req)
}

func (x *AuctionService) GetTransactionsByTime(ctx context.Context, req *auction.GetTransactionsByTimeReq) (resp *auction.GetTransactionsByTimeRsp, err error) {
	auctionMgr := manager.GetAuctionManager()
	return auctionMgr.GetTransactionsByTime(ctx, req)
}
