// Package ws 提供 WebSocket 服务端实现
// 负责处理客户端的 WebSocket 连接请求，管理连接生命周期
package ws

import (
	"gate_way_module/session/isession"
	"net/http"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gorilla/websocket"
)

// WebSocketServer 启动 WebSocket 服务器
// addr: 监听地址（如 ":8080"）
// uri: WebSocket 端点路径（如 "/ws"）
// sessionFactory: Session 工厂函数，用于创建新的会话实例
func WebSocketServer(addr string, uri string, sessionFactory func() isession.ISession) {
	http.HandleFunc(uri, func(resp http.ResponseWriter, req *http.Request) {
		// 获取客户端地址和 User-Agent
		remoteAddr := req.RemoteAddr
		klog.Infof("[WS-CLIENT-CONNECT] New connection from %s, User-Agent: %s", remoteAddr, req.Header.Get("User-Agent"))

		// 配置 WebSocket Upgrader
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024, // 读取缓冲区大小
			WriteBufferSize: 1024, // 写入缓冲区大小
			// 允许所有来源（生产环境应限制允许的域名）
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		// 创建新的 Session 实例
		s := sessionFactory()

		// 将 HTTP 请求升级为 WebSocket 连接
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			klog.Errorf("[WS-UPGRADE-ERROR] WebSocket upgrade error from %s: %v", remoteAddr, err)
			return
		}
		defer conn.Close() // 函数退出时关闭连接

		// 将 WebSocket 连接绑定到 Session
		s.Conn(conn)
		klog.Infof("[WS-CONNECTED] WebSocket connection established with %s", remoteAddr)

		// 启动消息读取协程
		go readMessage(s, s.ErrChan())

		// 阻塞等待错误信号（连接关闭或出错）
		select {
		case err := <-(*s.ErrChan()):
			s.Close()
			klog.Infof("[WS-CLIENT-CLOSE] WebSocket client %s closed connection: %+v", remoteAddr, err)
			return
		}
	})

	// 记录服务器启动信息
	klog.Infof("[WS-SERVER-START] Starting websocket server at: %s/%s", addr, uri)

	// 启动 HTTP 服务（异步）
	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			klog.Errorf("[WS-SERVER-FAIL] WebSocket server failed to start: %v", err)
		}
	}()
}

// readMessage 从 Session 读取消息
// 这是一个阻塞函数，会持续读取直到出错或连接关闭
// s: Session 实例
// closechan: 错误通道，用于通知主协程发生错误
func readMessage(s isession.ISession, closechan *chan error) {
	err := s.Read()
	*closechan <- err
}
