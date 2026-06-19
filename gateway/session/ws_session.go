// Package session 提供 WebSocket 会话管理功能
// 负责消息的读写、序列化/反序列化以及会话生命周期管理
package session

import (
	"errors"
	msghandler "gate_way_module/msg_handler"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Session 表示一个 WebSocket 会话
// 每个客户端连接对应一个 Session 实例
type Session struct {
	conn      *websocket.Conn // WebSocket 连接实例
	errChan   chan error      // 错误通道，用于通知上层发生错误
	CloseFunc func(*Session)  // 关闭回调函数，在会话关闭时执行
	msgType   int             // 当前消息类型（文本或二进制）
}

// NewSession 创建一个新的 Session 实例
// closeFunc: 会话关闭时的回调函数，用于清理资源
func NewSession(closeFunc func(*Session)) *Session {
	s := &Session{
		errChan:   make(chan error, 1), // 带缓冲的错误通道，防止阻塞
		CloseFunc: closeFunc,
	}
	return s
}

// Conn 设置 WebSocket 连接
// conn: WebSocket 连接实例
func (s *Session) Conn(conn *websocket.Conn) {
	s.conn = conn
}

// ErrChan 返回错误通道的指针
// 上层可以通过该通道监听会话错误
func (s *Session) ErrChan() *chan error {
	return &s.errChan
}

// Read 持续读取 WebSocket 消息
// 这是一个阻塞方法，会一直循环读取直到出错或连接关闭
// 消息类型判断：文本消息和二进制消息不能混合，首次收到的消息类型决定后续消息类型
func (s *Session) Read() error {
	for {
		// 从 WebSocket 连接读取消息
		msgType, msg, err := s.conn.ReadMessage()
		if err != nil {
			klog.Warnf("[WS-READ-ERROR] Read message error: %v", err)
			return err
		}

		// 获取客户端地址用于日志记录
		remoteAddr := "unknown"
		if s.conn != nil {
			remoteAddr = s.conn.RemoteAddr().String()
		}

		// 记录接收到的消息信息
		klog.Debugf("[WS-MESSAGE-RECV] Received message from %s, type: %d, length: %d", remoteAddr, msgType, len(msg))

		// 根据消息类型进行处理
		if msgType == websocket.TextMessage {
			// 检查消息类型一致性
			if s.msgType == websocket.BinaryMessage {
				err := errors.New("receive binary message, but expect text message")
				klog.Warnf("[WS-MSG-TYPE-MISMATCH] %s: %v", remoteAddr, err)
				return err
			}
			s.msgType = websocket.TextMessage
			s.processTextMessage(msg)
		} else if msgType == websocket.BinaryMessage {
			// 检查消息类型一致性
			if s.msgType == websocket.TextMessage {
				err := errors.New("receive text message, but expect binary message")
				klog.Warnf("[WS-MSG-TYPE-MISMATCH] %s: %v", remoteAddr, err)
				return err
			}
			s.msgType = websocket.BinaryMessage
			s.processBinaryMessage(msg)
		} else if msgType == websocket.CloseMessage {
			klog.Infof("[WS-CLOSE-MESSAGE] Client %s sent close message", remoteAddr)
			return &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "client closed connection"}
		} else {
			klog.Warnf("[WS-UNKNOWN-MSG-TYPE] Received unknown message type %d from %s", msgType, remoteAddr)
		}
	}
}

// Write 向客户端发送原始消息
// msgType: 消息类型（websocket.TextMessage 或 websocket.BinaryMessage）
// msg: 消息内容
func (s *Session) Write(msgType int, msg []byte) error {
	err := s.conn.WriteMessage(msgType, msg)
	return err
}

// write2 将 anypb.Any 消息序列化并发送
// 根据当前会话的消息类型（文本或二进制）选择相应的序列化方式
// any: 要发送的消息（protobuf Any 类型）
func (s *Session) write2(any *anypb.Any) error {
	remoteAddr := "unknown"
	if s.conn != nil {
		remoteAddr = s.conn.RemoteAddr().String()
	}

	// 根据当前消息类型选择序列化方式
	if s.msgType == websocket.BinaryMessage {
		// 二进制模式：使用 proto.Marshal
		msg, e := proto.Marshal(any)
		if e != nil {
			klog.Errorf("[WS-MARSHAL-ERROR] Binary marshal error for %s: %v", remoteAddr, e)
			return e
		}
		klog.Debugf("[WS-MESSAGE-SEND] Sending binary message to %s, type: %s, length: %d", remoteAddr, any.TypeUrl, len(msg))
		return s.Write(websocket.BinaryMessage, msg)
	}

	if s.msgType == websocket.TextMessage {
		// 文本模式：使用 protojson.Marshal
		msg, e := protojson.Marshal(any)
		if e != nil {
			klog.Errorf("[WS-MARSHAL-ERROR] JSON marshal error for %s: %v", remoteAddr, e)
			return e
		}
		klog.Debugf("[WS-MESSAGE-SEND] Sending text message to %s, type: %s, length: %d", remoteAddr, any.TypeUrl, len(msg))
		return s.Write(websocket.TextMessage, msg)
	}

	// 消息类型未设置（还未收到任何消息）
	klog.Warnf("[WS-WRITE-ERROR] No message type set for %s, cannot send message", remoteAddr)
	return nil
}

// Send 向客户端发送消息
// 这是对外暴露的发送接口，会自动处理序列化
// any: 要发送的消息（protobuf Any 类型），nil 表示关闭连接
func (s *Session) Send(any *anypb.Any) {
	if any == nil {
		klog.Error("[WS-SEND-ERROR] Attempt to send nil message")
		s.errChan <- errors.New("any is nil")
		return
	}
	if e := s.write2(any); e != nil {
		klog.Errorf("[WS-SEND-ERROR] Send message failed: %v", e)
		s.errChan <- e
	}
}

// Close 关闭会话
// 执行关闭回调函数并关闭底层 WebSocket 连接
func (s *Session) Close() error {
	remoteAddr := "unknown"
	if s.conn != nil {
		remoteAddr = s.conn.RemoteAddr().String()
	}
	klog.Infof("[WS-SESSION-CLOSE] Closing session for %s", remoteAddr)

	// 执行关闭回调（清理用户管理等）
	s.CloseFunc(s)

	// 关闭 WebSocket 连接
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			klog.Errorf("[WS-CLOSE-ERROR] Failed to close connection for %s: %v", remoteAddr, err)
			return err
		}
		klog.Infof("[WS-CONNECTION-CLOSED] Connection closed for %s", remoteAddr)
	}
	return nil
}

// processTextMessage 处理文本消息
// 将 JSON 格式的消息反序列化为 anypb.Any，然后交给消息处理器处理
// msg: 原始文本消息内容
func (s *Session) processTextMessage(msg []byte) {
	any := &anypb.Any{}
	// 将 JSON 字符串反序列化为 anypb.Any
	if e := protojson.Unmarshal(msg, any); e != nil {
		remoteAddr := "unknown"
		if s.conn != nil {
			remoteAddr = s.conn.RemoteAddr().String()
		}
		klog.Errorf("[WS-UNMARSHAL-ERROR] JSON unmarshal error from %s: %v", remoteAddr, e)
		s.errChan <- e
		return
	}

	// 记录消息处理日志
	remoteAddr := "unknown"
	if s.conn != nil {
		remoteAddr = s.conn.RemoteAddr().String()
	}
	klog.Debugf("[WS-MESSAGE-HANDLE] Handling message type %s from %s", any.TypeUrl, remoteAddr)

	// 交给消息处理器处理
	if e := msghandler.Handle(s, any); e != nil {
		klog.Errorf("[WS-HANDLE-ERROR] Message handler error for %s: %v", remoteAddr, e)
		s.errChan <- e
		return
	}
}

// processBinaryMessage 处理二进制消息
// 将二进制格式的消息反序列化为 anypb.Any，然后交给消息处理器处理
// msg: 原始二进制消息内容
func (s *Session) processBinaryMessage(msg []byte) {
	any := &anypb.Any{}
	// 将二进制数据反序列化为 anypb.Any
	if e := proto.Unmarshal(msg, any); e != nil {
		remoteAddr := "unknown"
		if s.conn != nil {
			remoteAddr = s.conn.RemoteAddr().String()
		}
		klog.Errorf("[WS-UNMARSHAL-ERROR] Binary unmarshal error from %s: %v", remoteAddr, e)
		s.errChan <- e
		return
	}

	// 记录消息处理日志
	remoteAddr := "unknown"
	if s.conn != nil {
		remoteAddr = s.conn.RemoteAddr().String()
	}
	klog.Debugf("[WS-MESSAGE-HANDLE] Handling message type %s from %s", any.TypeUrl, remoteAddr)

	// 交给消息处理器处理
	if e := msghandler.Handle(s, any); e != nil {
		klog.Errorf("[WS-HANDLE-ERROR] Message handler error for %s: %v", remoteAddr, e)
		s.errChan <- e
		return
	}
}
