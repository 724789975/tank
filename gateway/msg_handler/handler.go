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
}

