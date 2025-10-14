package main

import (
	"context"
	common_config "gate_way_module/config"
	rpcservice "gate_way_module/rpc_service"
	"gate_way_module/session"
	"gate_way_module/session/isession"
	usermgr "gate_way_module/user_mgr"
	"gate_way_module/ws"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	_, cancel := context.WithCancel(context.Background())

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
