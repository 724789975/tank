package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	any1 "github.com/golang/protobuf/ptypes/any"
	common_config "route_module/config"
	route_http "route_module/http"
	"route_module/kitex_gen/auction"
	"route_module/kitex_gen/auction_service/auctionservice"
	"route_module/kitex_gen/gate_way"
	"route_module/kitex_gen/gateway_service/gatewayservice"
	"route_module/kitex_gen/item"
	"route_module/kitex_gen/item_service/itemservice"
	"route_module/kitex_gen/match_proto"
	"route_module/kitex_gen/match_service/matchservice"
	"route_module/kitex_gen/server_mgr"
	"route_module/kitex_gen/server_mgr_service/servermgrservice"
	"route_module/kitex_gen/tank_game_service"
	"route_module/kitex_gen/tank_game_service/tankgameservice"
	"route_module/kitex_gen/user_center"
	"route_module/kitex_gen/user_center_service/usercenterservice"
	"route_module/rpc"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gin-gonic/gin"
)

type RouteService struct{}

var (
	routeSrv      RouteService
	onceRouteSrv  sync.Once
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

	// Wrap bodyBytes into proto.Any
	bodyAny := &any1.Any{Value: bodyBytes}

	// Call RPC based on service_name
	ginCtx := context.WithValue(ctx.Request.Context(), "userId", ctx.Value("userId").(string))
	client, err := rpc.GetClient(serviceName)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": err.Error()})
		return
	}

	handler, ok := serviceHandlers[serviceName]
	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": fmt.Sprintf("unknown service: %s", serviceName)})
		return
	}
	resp := handler(client, rpcMethod, bodyAny, ginCtx)

	ctx.JSON(http.StatusOK, resp)
}

type serviceHandler func(client interface{}, method string, body *any1.Any, ctx context.Context) interface{}

var serviceHandlers = map[string]serviceHandler{
	"gateway":     callGateway,
	"user_center": callUserCenter,
	"match":       callMatch,
	"server_mgr":  callServerMgr,
	"item":        callItem,
	"tank_game":   callTankGame,
	"auction":     callAuction,
}

func callGateway(client interface{}, method string, body *any1.Any, ctx context.Context) interface{} {
	switch method {
	case "UserMsg":
		var req gate_way.UserMsgReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(gatewayservice.Client).UserMsg(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	default:
		return gin.H{"code": -1, "msg": fmt.Sprintf("unknown method: %s", method)}
	}
}

func callUserCenter(client interface{}, method string, body *any1.Any, ctx context.Context) interface{} {
	switch method {
	case "Login":
		var req user_center.LoginReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(usercenterservice.Client).Login(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "UserInfo":
		var req user_center.UserInfoReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(usercenterservice.Client).UserInfo(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "TestLogin":
		var req user_center.TestLoginReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(usercenterservice.Client).TestLogin(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	default:
		return gin.H{"code": -1, "msg": fmt.Sprintf("unknown method: %s", method)}
	}
}

func callMatch(client interface{}, method string, body *any1.Any, ctx context.Context) interface{} {
	switch method {
	case "Match":
		var req match_proto.MatchReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(matchservice.Client).Match(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "Pve":
		var req match_proto.PveReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(matchservice.Client).Pve(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	default:
		return gin.H{"code": -1, "msg": fmt.Sprintf("unknown method: %s", method)}
	}
}

func callServerMgr(client interface{}, method string, body *any1.Any, ctx context.Context) interface{} {
	switch method {
	case "CreateServer":
		var req server_mgr.CreateServerReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(servermgrservice.Client).CreateServer(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "CreateAiClient":
		var req server_mgr.CreateAiClientReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(servermgrservice.Client).CreateAiClient(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	default:
		return gin.H{"code": -1, "msg": fmt.Sprintf("unknown method: %s", method)}
	}
}

func callItem(client interface{}, method string, body *any1.Any, ctx context.Context) interface{} {
	switch method {
	case "AddItem":
		var req item.AddItemReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(itemservice.Client).AddItem(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "DeleteItem":
		var req item.DeleteItemReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(itemservice.Client).DeleteItem(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "GetAllItems":
		var req item.GetAllItemsReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(itemservice.Client).GetAllItems(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "GetItem":
		var req item.GetItemReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(itemservice.Client).GetItem(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	default:
		return gin.H{"code": -1, "msg": fmt.Sprintf("unknown method: %s", method)}
	}
}

func callTankGame(client interface{}, method string, body *any1.Any, ctx context.Context) interface{} {
	switch method {
	case "Ping":
		var req tank_game_service.Ping
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(tankgameservice.Client).Ping(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	default:
		return gin.H{"code": -1, "msg": fmt.Sprintf("unknown method: %s", method)}
	}
}

func callAuction(client interface{}, method string, body *any1.Any, ctx context.Context) interface{} {
	switch method {
	case "Ping":
		var req auction.PingReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(auctionservice.Client).Ping(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "Sell":
		var req auction.SellReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(auctionservice.Client).Sell(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	case "Buy":
		var req auction.BuyReq
		if err := json.Unmarshal(body.GetValue(), &req); err != nil {
			return gin.H{"code": -1, "msg": "invalid request"}
		}
		r, err := client.(auctionservice.Client).Buy(ctx, &req)
		if err != nil {
			return gin.H{"code": -1, "msg": err.Error()}
		}
		return gin.H{"code": 0, "data": r}
	default:
		return gin.H{"code": -1, "msg": fmt.Sprintf("unknown method: %s", method)}
	}
}
