package service

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	common_config "route_module/config"
	route_http "route_module/http"
	"route_module/rpc"

	any1 "github.com/golang/protobuf/ptypes/any"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gin-gonic/gin"
)

type RouteService struct{}

var (
	routeSrv     RouteService
	onceRouteSrv sync.Once
)

func GetRouteService() *RouteService {
	onceRouteSrv.Do(func() {
		routeSrv = RouteService{}
	})
	return &routeSrv
}

func (s *RouteService) ListenAndServe(ctx context.Context) {
	clientRouter := route_http.GetClientRouter()
	group := clientRouter.Group("api").Group("1.0")
	group2 := group.Group("public").Group("route")
	group2.Use(route_http.AuthForClient(10))
	group2.POST(":service_name/:rpc", s.ginRoute)

	httpServer := &http.Server{
		Addr:    common_config.Get("http.addr").(string),
		Handler: clientRouter,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.CtxErrorf(ctx, "[ROUTE-SVR-HTTP-SERVER-START] run server http error:%s", err.Error())
			panic(err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			klog.CtxErrorf(ctx, "[ROUTE-SVR-HTTP-SHUTDOWN] shutdown error:%s", err.Error())
		}
	}()
}

func (s *RouteService) ginRoute(ctx *gin.Context) {
	serviceName := ctx.Param("service_name")
	rpcMethod := ctx.Param("rpc")
	klog.CtxInfof(ctx.Request.Context(), "[ROUTE-REQUEST] service: %s, rpc: %s", serviceName, rpcMethod)

	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		klog.CtxErrorf(ctx.Request.Context(), "[ROUTE-LOG] read body error: %s", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "read body failed"})
		return
	}
	defer ctx.Request.Body.Close()

	bodyAny := &any1.Any{Value: bodyBytes}

	ginCtx := context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(string))
	client, err := rpc.GetClient(serviceName)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": err.Error()})
		return
	}

	err, resp := client(ginCtx, rpcMethod, bodyAny)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": -1, "msg": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": resp.GetValue()})
}
