package isession

import (
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/types/known/anypb"
)

type ISession interface {
	Conn(*websocket.Conn)
	ErrChan() *chan error
	Read() error
	Write(int, []byte) error
	Send(any *anypb.Any)		// 发送空值代表关闭连接
	Close() error
}
