package main

import (
	"context"
	common_config "gate_way_module/config"
	"gate_way_module/etcd"
	"gate_way_module/nats"
	common_redis "gate_way_module/redis"
	rpcservice "gate_way_module/rpc_service"
	"gate_way_module/session"
	"gate_way_module/session/isession"
	usermgr "gate_way_module/user_mgr"
	"gate_way_module/ws"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/kitex-contrib/obs-opentelemetry/logging/logrus"
)

func main() {
	klog.SetLogger(logrus.NewLogger())
	klog.SetLevel(klog.LevelDebug)
	_, cancel := context.WithCancel(context.Background())
	common_config.LoadConfig()
	common_redis.GetRedis()
	nats.GetNatsConn()
	etcd.GetEtcdClient()

	usermgr.InitUserMgr()

	ws.WebSocketServer(common_config.Get("ws.addr").(string), common_config.Get("ws.uri").(string), func() isession.ISession {
		return session.NewSession(func(s *session.Session) {
			usermgr.GetUserMgr().RemoveSession(s)
		})
		// return &session.Session{
		// 	CloseFunc: func(s *session.Session) {
		// 		usermgr.GetUserMgr().RemoveSession(s)
		// 	},
		// }
	})

	rpcservice.InitGatewayService()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	s := <-quit
	switch s {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP:
		cancel()
	default:
	}
}
