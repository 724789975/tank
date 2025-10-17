package msghandler

import (
	"errors"
	"gate_way_module/session/isession"

	"google.golang.org/protobuf/types/known/anypb"
)

var handler map[string]func(session isession.ISession, any *anypb.Any) error = make(map[string]func(session isession.ISession, any *anypb.Any) error)

func RegisterHandler(msgType string, f func(session isession.ISession, any *anypb.Any) error) {
	handler[msgType] = f
}

func Handle(session isession.ISession, any *anypb.Any) error {
	msgType := any.GetTypeUrl()
	f, ok := handler[msgType]
	if ok {
		return f(session, any)
	}
	return errors.New("unsupported message type " + msgType)
	// 注意：实际使用时可以在调用方根据错误类型返回相应的错误码
	// 这里仅作为参考，具体实现可能需要在session的响应中设置错误码
}

