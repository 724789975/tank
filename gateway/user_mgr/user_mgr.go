package usermgr

import (
	"context"
	"fmt"
	"gateway/constant"
	msghandler "gateway/msg_handler"
	common_redis "gateway/redis"
	"gateway/session/isession"
	"gateway/util"
	"sync"

	"devops.xfein.com/codeup/75560f9d-2fab-4efe-bbe6-90b71f3ff9e4/xf-x/backend/common"
	"devops.xfein.com/codeup/75560f9d-2fab-4efe-bbe6-90b71f3ff9e4/xf-x/backend/idl/kitex_gen/gateway"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)


type User struct {
	id string
	session isession.ISession
	sub *nats.Subscription
}

type UserMgr struct {
	users *util.RWMap[string, *User]

	sessions *util.RWMap[isession.ISession, string]
	ops *util.RWMap[int64, func()]
}

func (u *UserMgr) addUser(userId string, _session isession.ISession, _sub *nats.Subscription) {
	u.users.Set(userId, &User{
		id: userId,
		session: _session,
		sub: _sub,
	})
}

func (u *UserMgr) removeUser(userId string) {
	if user, ok := u.users.Get(userId); ok {
		u.sessions.Delete(user.session)
		user.sub.Unsubscribe()
	}
	u.users.Delete(userId)
}

func (u *UserMgr) RemoveSession(session isession.ISession) {
	if userId, ok := u.sessions.Get(session); ok {
		u.removeUser(userId)
	}
}

func (u *UserMgr) getUserSession(userId string) (isession.ISession, bool) {
	user, b :=  u.users.Get(userId)
	if b {
		return user.session, true
	}
	return nil, false
}

func (u *UserMgr) addOp(idx int64, f func()) {
	u.ops.Set(idx, f)
}

func (u *UserMgr) removeOp(idx int64) {
	u.ops.Delete(idx)
}

func (u *UserMgr) getOp(idx int64) (func(), bool) {
	return u.ops.Get(idx)
}

var (
	usermgr *UserMgr
	once  sync.Once
)
func GetUserMgr() *UserMgr {
	once.Do(func() {
		usermgr = &UserMgr{
			users: util.NewRWMap[string, *User](),
			sessions: util.NewRWMap[isession.ISession, string](),
			ops: util.NewRWMap[int64, func()](),
		}
	})

	return usermgr
}

func InitUserMgr()  {
	{
		any := &anypb.Any{}
		any.MarshalFrom(&gateway.LoginRequest{Id: "123"})
		b, _ := protojson.Marshal(any)
		fmt.Print(string(b))
		msghandler.RegisterHandler(any.TypeUrl, func(session isession.ISession, any *anypb.Any) error {
			idx, _ := common_redis.GetRedis().Incr(context.TODO(), constant.UserLoginMsgIdx).Result()
			loginRequest := &gateway.LoginRequest{}
			err := any.UnmarshalTo(loginRequest)
			if err != nil {
				return err
			}

			ncMsg := &gateway.NatsLoginRequest{
				Id: loginRequest.Id,
				Idx: idx,
			}
			ncMsgb, err := proto.Marshal(ncMsg)
			if err != nil {
				return err
			}

			GetUserMgr().addOp(idx, func() {
				if sub, err := common.GetNatsConn().Subscribe(fmt.Sprintf(constant.UserMsg, loginRequest.Id), func(msg *nats.Msg) {
					any := &anypb.Any{}
					err := proto.Unmarshal(msg.Data, any)
					if err != nil {
						return
					}
					session.Send(any)
				}); err != nil {
					klog.Errorf("subscribe %s failed, err: %v", fmt.Sprintf(constant.UserMsg, loginRequest.Id), err)
					return
				} else {
					GetUserMgr().addUser(loginRequest.Id, session, sub)
					GetUserMgr().removeOp(idx)
				}
			})

			common.GetNatsConn().Publish(constant.UserLoginMsg, ncMsgb)
			common.GetNatsConn().Flush()

			return nil
		})
	}

	if _, err := common.GetNatsConn().Subscribe(constant.UserLoginMsg, func(msg *nats.Msg) {
		ncMsg := &gateway.NatsLoginRequest{}
		err := proto.Unmarshal(msg.Data, ncMsg)
		if err != nil {
			return
		}

		if session, ok := GetUserMgr().getUserSession(ncMsg.Id); ok {
			session.Send(nil)
			GetUserMgr().removeUser(ncMsg.Id)
		}

		if op, ok := GetUserMgr().getOp(ncMsg.Idx); ok {
			op()
		}
	}); err != nil {
		panic(err)
	}
}