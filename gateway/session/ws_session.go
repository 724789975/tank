package session

import (
	"errors"
	msghandler "gate_way_module/msg_handler"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Session struct {
	conn *websocket.Conn
	errChan chan error

	CloseFunc func(*Session)

	msgType int
}

func NewSession(closeFunc func(*Session)) *Session {
	s := &Session{
		errChan: make(chan error, 1),
		CloseFunc: closeFunc,
	}
	return s
}

// ErrChan implements isession.ISession.
func (s *Session) Conn(conn *websocket.Conn) {
	s.conn = conn
}

func (s *Session) ErrChan() *chan error {
	return &s.errChan
}

func (s *Session) Read() error {
	// 读取消息
	for {
		msgType, msg, err := s.conn.ReadMessage()
		if err != nil {
			return err
		}

		if msgType == websocket.TextMessage {
			if s.msgType == websocket.BinaryMessage {
				return errors.New("receive binary message, but expect text message")
			}
			s.msgType = websocket.TextMessage
			s.processTextMessage(msg)
		} else if msgType == websocket.BinaryMessage {
			if s.msgType == websocket.TextMessage {
				return errors.New("receive text message, but expect binary message")
			}
			s.msgType = websocket.BinaryMessage
			s.processBinaryMessage(msg)
		} else if msgType == websocket.CloseMessage {
			return &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "client closed connection"}
		} else {
		}
	}
}

func (s *Session) Write(msgType int, msg []byte) error {
	// 发送消息
	err := s.conn.WriteMessage(msgType, msg)
	return err
}

func (s *Session) write2(any *anypb.Any) error {
	if s.msgType == websocket.BinaryMessage {
		msg, e := proto.Marshal(any)
		if e != nil {
			return e
		}
		return s.Write(websocket.BinaryMessage, msg)
	}

	if s.msgType == websocket.TextMessage {
		msg, e := protojson.Marshal(any)
		if e != nil {
			return e
		}
		return s.Write(websocket.TextMessage, msg)
	}

	return nil
}

func (s *Session) Send(any *anypb.Any) {
	if any == nil {
		s.errChan <- errors.New("any is nil")
		return
	}
	if e := s.write2(any); e != nil {
		s.errChan <- e
	}
}

func (s *Session) Close() error {
	s.CloseFunc(s)
	return nil
}

func (s *Session) processTextMessage(msg []byte) {
	any := &anypb.Any{}
	if e := protojson.Unmarshal(msg, any); e != nil {
		s.errChan <- e
		return
	}

	if e := msghandler.Handle(s, any); e != nil {
		s.errChan <- e
		return
	}
}

func (s *Session) processBinaryMessage(msg []byte) {
	any := &anypb.Any{}
	proto.Unmarshal(msg, any)
	if e := protojson.Unmarshal(msg, any); e != nil {
		s.errChan <- e
		return
	}

	if e := msghandler.Handle(s, any); e != nil {
		s.errChan <- e
		return
	}
}

