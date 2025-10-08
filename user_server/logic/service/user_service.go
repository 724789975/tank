package service

import (
	"context"
	"net"
	"net/http"
	"sync"
	common_config "user_server/config"
	"user_server/driver"
	user_http "user_server/http"

	"user_server/kitex_gen/user_center"
	"user_server/kitex_gen/user_center_service/usercenterservice"

	"github.com/cloudwego/kitex/pkg/klog"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/streaming"
	"github.com/gin-gonic/gin"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

type UserService struct {
	*kitex.ServiceInfo
}

var (
	user_srv      UserService
	once_user_srv sync.Once
)

func GetUserService() IService {
	once_user_srv.Do(func() {
		user_srv = UserService{
			ServiceInfo: usercenterservice.NewServiceInfo(),
		}
	})
	return &user_srv
}

func (s *UserService) ListenAndServe(ctx context.Context) {
	clientRouter := user_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0").Group("public").Group("user_server")
	group.POST(":method", s.ginRoute)

	httpServer := &http.Server{
		Addr:    common_config.Get("user_http.addr").(string),
		Handler: clientRouter,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Errorf("run server http error:%s", err.Error())
			panic(err)
		}
	}()

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", common_config.Get("user_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := common_config.Get("user_rpc.service_name").(string)

		server := kitexserver.NewServer(
			kitexserver.WithServiceAddr(address),
			kitexserver.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			}),
			kitexserver.WithSuite(tracing.NewServerSuite()),
		)

		return server
	}

	ser := NewKitexServer()
	usercenterservice.RegisterService(ser, s)

	go func() {
		if err := ser.Run(); err != nil {
			klog.Errorf("run server rpc error:%s", err.Error())
			panic(err)
		}
	}()
}

func (s *UserService) ginRoute(ctx *gin.Context) {
	methodName := ctx.Param("method")

	c := context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(int64))
	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(c, "not found", ctx.FullPath())
		return
	}
	str := driver.NewHttpStream(ctx.Writer, ctx.Request)
	handler := info.Handler()
	if err := handler(c, s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(c, err.Error())
	}
}

func (s *UserService) Close() {
}

func (x *UserService) Login(ctx context.Context, req *user_center.LoginReq) (resp *user_center.LoginRsp, err error) {
	// return manager.GetUserManager().Login(ctx, req)
	return nil, nil
}
