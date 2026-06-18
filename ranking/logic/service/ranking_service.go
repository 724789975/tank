package service

import (
	"context"
	"net"
	nethttp "net/http"
	"ranking_module/config"
	"ranking_module/driver"
	"ranking_module/etcd"
	ranking_http "ranking_module/http"
	"ranking_module/kitex_gen/ranking"
	"ranking_module/kitex_gen/ranking_service/rankingservice"
	"ranking_module/logic/manager"
	"ranking_module/rpc_middleware"
	"sync"

	"github.com/cloudwego/kitex/pkg/klog"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/streaming"
	"github.com/gin-gonic/gin"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

type RankingService struct {
	*kitex.ServiceInfo
}

var (
	ranking_srv      RankingService
	once_ranking_srv sync.Once
)

func GetRankingService() IService {
	once_ranking_srv.Do(func() {
		ranking_srv = RankingService{
			ServiceInfo: rankingservice.NewServiceInfo(),
		}
	})
	return &ranking_srv
}

func (s *RankingService) ListenAndServe(ctx context.Context) {
	clientRouter := ranking_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0").Group("public").Group("ranking")
	group.POST(":method", s.ginRoute)

	httpServer := &nethttp.Server{
		Addr:    config.Get("ranking_http.addr").(string),
		Handler: clientRouter,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
			klog.Errorf("[RANKING-SVR-HTTP-SERVER-START] run server http error:%s", err.Error())
			panic(err)
		}
	}()

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", config.Get("ranking_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := config.Get("ranking_rpc.service_name").(string)

		server := kitexserver.NewServer(
			kitexserver.WithServiceAddr(address),
			kitexserver.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			}),
			kitexserver.WithSuite(tracing.NewServerSuite()),
			kitexserver.WithRegistry(etcd.GetEtcdClient()),
			kitexserver.WithMiddleware(rpc_middleware.UserIdServerMiddleware),
		)

		return server
	}

	server := NewKitexServer()
	err := server.RegisterService(s.ServiceInfo, s)
	if err != nil {
		panic(err)
	}
	if err := server.Run(); err != nil {
		klog.Errorf("[RANKING-SVR-KITEX-SERVER-START] run server rpc error:%s", err.Error())
	}
}

func (s *RankingService) Close() {
}

func (s *RankingService) ginRoute(ctx *gin.Context) {
	methodName := ctx.Param("method")

	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(ctx, "[RANKING-SVR-METHOD-NOT-FOUND] not found: %s", ctx.FullPath())
		return
	}
	str := driver.NewHttpStream(ctx.Writer, ctx.Request)
	handler := info.Handler()
	if err := handler(context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(string)), s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(ctx, "[RANKING-SVR-METHOD-HANDLER-ERROR] %s", err.Error())
	}
}

// UpdateScore 更新用户分数
func (x *RankingService) UpdateScore(ctx context.Context, req *ranking.UpdateScoreReq) (resp *ranking.UpdateScoreRsp, err error) {
	return manager.GetRankingManager().UpdateScore(ctx, req)
}

// GetUserRank 获取用户当前排名
func (x *RankingService) GetUserRank(ctx context.Context, req *ranking.GetUserRankReq) (resp *ranking.GetUserRankRsp, err error) {
	return manager.GetRankingManager().GetUserRank(ctx, req)
}

// GetRankRange 获取排名区间
func (x *RankingService) GetRankRange(ctx context.Context, req *ranking.GetRankRangeReq) (resp *ranking.GetRankRangeRsp, err error) {
	return manager.GetRankingManager().GetRankRange(ctx, req)
}

// BatchGetUserRank 批量获取用户排名
func (x *RankingService) BatchGetUserRank(ctx context.Context, req *ranking.BatchGetUserRankReq) (resp *ranking.BatchGetUserRankRsp, err error) {
	return manager.GetRankingManager().BatchGetUserRank(ctx, req)
}

// GetRankingStats 获取排行榜统计信息
func (x *RankingService) GetRankingStats(ctx context.Context, req *ranking.GetRankingStatsReq) (resp *ranking.GetRankingStatsRsp, err error) {
	return manager.GetRankingManager().GetRankingStats(ctx, req)
}
