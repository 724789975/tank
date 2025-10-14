package main

import (
	"context"
	rpcservice "gateway/rpc_service"
	"gateway/session"
	"gateway/session/isession"
	usermgr "gateway/user_mgr"
	"gateway/ws"
	"os"
	"os/signal"
	"syscall"

	"devops.xfein.com/codeup/75560f9d-2fab-4efe-bbe6-90b71f3ff9e4/xf-x/backend/common"
)

func main() {
	_, cancel := context.WithCancel(context.Background())
	common.Init(nil, nil)
	common.GetNatsConn()

	usermgr.InitUserMgr()

	ws.WebSocketServer(common.Get("ws.addr").(string), common.Get("ws.uri").(string), func() isession.ISession {
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
