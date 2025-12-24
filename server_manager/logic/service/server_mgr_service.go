package service

import (
	"context"
	"net"
	"net/http"
	common_config "server_manager/config"
	"server_manager/driver"
	"server_manager/etcd"
	match_http "server_manager/http"
	"server_manager/logic/manager"
	"sync"

	"server_manager/kitex_gen/server_mgr"
	"server_manager/kitex_gen/server_mgr_service/servermgrservice"

	"github.com/cloudwego/kitex/pkg/klog"
	kitex "github.com/cloudwego/kitex/pkg/serviceinfo"
	"github.com/cloudwego/kitex/pkg/streaming"
	"github.com/gin-gonic/gin"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

type ServerMgrService struct {
	*kitex.ServiceInfo
}

var (
	user_srv      ServerMgrService
	once_user_srv sync.Once
)

func GetServerMgrService() IService {
	once_user_srv.Do(func() {
		user_srv = ServerMgrService{
			ServiceInfo: servermgrservice.NewServiceInfo(),
		}
	})
	return &user_srv
}

func (s *ServerMgrService) ListenAndServe(ctx context.Context) {
	clientRouter := match_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0").Group("public").Group("server_mgr")
	group.POST(":method", s.ginRoute)

	httpServer := &http.Server{
		Addr:    common_config.Get("server_mgr_http.addr").(string),
		Handler: clientRouter,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Errorf("[SERVER-MGR-HTTP-SERVER-START] run server http error:%s", err.Error())
			panic(err)
		}
	}()

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", common_config.Get("server_mgr_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := common_config.Get("server_mgr_rpc.service_name").(string)

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
	servermgrservice.RegisterService(ser, s)

	go func() {
		if err := ser.Run(); err != nil {
			klog.Errorf("[SERVER-MGR-RPC-SERVER-START] run server rpc error:%s", err.Error())
			panic(err)
		}
	}()
}

func (s *ServerMgrService) ginRoute(ctx *gin.Context) {
	methodName := ctx.Param("method")

	// c := context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(int64))
	info, ok := s.ServiceInfo.Methods[methodName]
	if !ok {
		klog.CtxErrorf(ctx, "[SERVER-MGR-METHOD-NOT-FOUND] not found: %s", ctx.FullPath())
		return
	}
	str := driver.NewHttpStream(ctx.Writer, ctx.Request)
	handler := info.Handler()
	if err := handler(ctx, s, &streaming.Args{Stream: str}, nil); err != nil {
		klog.CtxErrorf(ctx, "[SERVER-MGR-METHOD-HANDLER-ERROR] %s", err.Error())
	}
}

func (s *ServerMgrService) Close() {
}

func (s *ServerMgrService) CreateServer(ctx context.Context, req *server_mgr.CreateServerReq) (resp *server_mgr.CreateServerRsp, err error) {
	return manager.GetServerManager().CreateServer(ctx, req)
}

func (s *ServerMgrService) CreateAiClient(ctx context.Context, req *server_mgr.CreateAiClientReq) (resp *server_mgr.CreateAiClientRsp, err error) {
	return manager.GetServerManager().CreateAiClient(ctx, req)
}
