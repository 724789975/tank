package ws

import (
	"net/http"
	"xf_gateway/session/isession"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gorilla/websocket"
)

func WebSocketServer(addr string, uri string, sessionFactory func() isession.ISession) {
	http.HandleFunc(uri, func (resp http.ResponseWriter, req *http.Request) {
		// 初始化 Upgrader
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		} // 使用默认的选项
		// 第三个参数是响应头,默认会初始化

		s := sessionFactory()
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			klog.Error("WebSocket upgrade error: %v", err)
			return
		}
		defer conn.Close()

		s.Conn(conn)

		// 读取客户端的发送额消息,并返回
		go readMessage(s, s.ErrChan())
		select {
			case err := <-(*s.ErrChan()):
				s.Close()
				klog.Infof("WebSocket client closed connection: %s, %+v", conn.RemoteAddr(), err)
				return
		}

	})
	klog.Infof("Starting websocket server at: %s/%s", addr, uri)

	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			klog.Errorf("WebSocket server failed to start: %v", err)
		}
	}()
}

// 读取客户端发送的消息,并返回
func readMessage(s isession.ISession, closechan *chan error) {
	err := s.Read()
	*closechan <- err
}
