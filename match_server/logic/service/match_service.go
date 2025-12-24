package service

import (
	"context"
	common_config "match_server/config"
	"match_server/driver"
	"match_server/etcd"
	match_http "match_server/http"
	"match_server/logic/manager"
	"net"
	"net/http"
	"sync"

	"match_server/kitex_gen/match_proto"
	"match_server/kitex_gen/match_service/matchservice"

	"github.com/cloudwego/kitex/pkg/klog"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/streaming"
	"github.com/gin-gonic/gin"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

type MatchService struct {
	*kitex.ServiceInfo
}

var (
	user_srv      MatchService
	once_user_srv sync.Once
)

func GetMatchService() IService {
	once_user_srv.Do(func() {
		user_srv = MatchService{
			ServiceInfo: matchservice.NewServiceInfo(),
		}
	})
	return &user_srv
}

func (s *MatchService) ListenAndServe(ctx context.Context) {
	clientRouter := match_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0").Group("public").Group("match_server")
	group.POST(":method", s.ginRoute)

	httpServer := &http.Server{
		Addr:    common_config.Get("match_http.addr").(string),
		Handler: clientRouter,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Errorf("[HTTP-SERVER-START] run server http error:%s", err.Error())
			panic(err)
		}
	}()

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", common_config.Get("match_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := common_config.Get("match_rpc.service_name").(string)

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
	matchservice.RegisterService(ser, s)

	go func() {
		if err := ser.Run(); err != nil {
			klog.Errorf("[RPC-SERVER-START] run server rpc error:%s", err.Error())
			panic(err)
		}
	}()
}

func (s *MatchService) ginRoute(ctx *gin.Context) {
	methodName := ctx.Param("method")

	// c := context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(int64))
	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(ctx.Request.Context(), "[METHOD-NOT-FOUND] not found: %s", ctx.FullPath())
		return
	}
	str := driver.NewHttpStream(ctx.Writer, ctx.Request)
	handler := info.Handler()

	if err := handler(context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(string)), s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(ctx.Request.Context(), "[METHOD-HANDLER-ERROR] %s", err.Error())
	}
}

func (s *MatchService) Close() {
}

func (x *MatchService) Match(ctx context.Context, req *match_proto.MatchReq) (resp *match_proto.MatchResp, err error) {
	return manager.GetMatchManager().Match(ctx, req)
}

func (x *MatchService) Pve(ctx context.Context, req *match_proto.PveReq) (resp *match_proto.PveResp, err error) {
	return manager.GetMatchManager().Pve(ctx, req)
}
