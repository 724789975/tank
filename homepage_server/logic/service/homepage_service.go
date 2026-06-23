package service

import (
	"context"
	"net"
	"net/http"
	"sync"
	"homepage_server/config"
	"homepage_server/driver"
	"homepage_server/etcd"
	homepage_http "homepage_server/http"
	"homepage_server/logic/manager"
	"homepage_server/rpc_middleware"

	"homepage_server/kitex_gen/homepage"
	"homepage_server/kitex_gen/homepage_service/homepageservice"

	"github.com/cloudwego/kitex/pkg/klog"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/streaming"
	"github.com/gin-gonic/gin"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

type HomepageService struct {
	*kitex.ServiceInfo
}

var (
	homepage_srv      HomepageService
	once_homepage_srv sync.Once
)

func GetHomepageService() IService {
	once_homepage_srv.Do(func() {
		homepage_srv = HomepageService{
			ServiceInfo: homepageservice.NewServiceInfo(),
		}
	})
	return &homepage_srv
}

func (s *HomepageService) ListenAndServe(ctx context.Context) {
	clientRouter := homepage_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0")
	group2 := group.Group("public").Group("homepage_server")
	group2.Use(homepage_http.AuthForClient(10))
	group2.POST(":method", s.ginRoute)
	group3 := group.Group("get").Group("homepage_server")
	group3.GET(":method", s.ginRoute2)

	httpServer := &http.Server{
		Addr:    config.Get("homepage_http.addr").(string),
		Handler: clientRouter,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.CtxErrorf(ctx, "[HOMEPAGE-SVR-HTTP-SERVER-START] run server http error:%s", err.Error())
			panic(err)
		}
	}()

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", config.Get("homepage_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := config.Get("homepage_rpc.service_name").(string)

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

	ser := NewKitexServer()
	homepageservice.RegisterService(ser, s)

	go func() {
		if err := ser.Run(); err != nil {
			klog.CtxErrorf(ctx, "[HOMEPAGE-SVR-RPC-SERVER-START] run server rpc error:%s", err.Error())
			panic(err)
		}
	}()
}

func (s *HomepageService) ginRoute(ctx *gin.Context) {
	methodName := ctx.Param("method")

	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(ctx, "[HOMEPAGE-SVR-METHOD-NOT-FOUND] not found: %s", ctx.FullPath())
		return
	}
	str := driver.NewHttpStream(ctx.Writer, ctx.Request)
	handler := info.Handler()
	if err := handler(context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(string)), s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(ctx, "[HOMEPAGE-SVR-METHOD-HANDLER-ERROR] %s", err.Error())
	}
}

func (s *HomepageService) ginRoute2(ctx *gin.Context) {
	methodName := ctx.Param("method")

	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(ctx, "[HOMEPAGE-SVR-METHOD-NOT-FOUND] not found: %s", ctx.FullPath())
		return
	}
	str := driver.NewGetStream(ctx.Writer, ctx.Request)
	handler := info.Handler()
	if err := handler(ctx.Request.Context(), s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(ctx, "[HOMEPAGE-SVR-METHOD-HANDLER-ERROR] %s", err.Error())
	}
}

func (s *HomepageService) Close() {
}

func (x *HomepageService) GetRoleInfo(ctx context.Context, req *homepage.GetRoleInfoReq) (resp *homepage.GetRoleInfoRsp, err error) {
	return manager.GetHomepageManager().GetRoleInfo(ctx, req)
}

func (x *HomepageService) UpdateRoleExp(ctx context.Context, req *homepage.UpdateRoleExpReq) (resp *homepage.UpdateRoleExpRsp, err error) {
	return manager.GetHomepageManager().UpdateRoleExp(ctx, req)
}