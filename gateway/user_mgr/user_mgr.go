package usermgr

import (
	"context"
	"fmt"
	"gate_way_module/constant"
	"gate_way_module/kitex_gen/common"
	"gate_way_module/kitex_gen/gate_way"
	msghandler "gate_way_module/msg_handler"
	"gate_way_module/nats"
	common_redis "gate_way_module/redis"
	"gate_way_module/session/isession"
	"gate_way_module/util"
	"sync"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/golang/protobuf/proto"
	_nats "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

type User struct {
	id      string
	session isession.ISession
	sub     *_nats.Subscription
}

type UserMgr struct {
	users *util.RWMap[string, *User]

	sessions *util.RWMap[isession.ISession, string]
	ops      *util.RWMap[int64, func()]
}

func (u *UserMgr) addUser(userId string, _session isession.ISession, _sub *_nats.Subscription) {
	u.users.Set(userId, &User{
		id:      userId,
		session: _session,
		sub:     _sub,
	})
	u.sessions.Set(_session, userId)
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
	user, b := u.users.Get(userId)
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
	once    sync.Once
)

func GetUserMgr() *UserMgr {
	once.Do(func() {
		usermgr = &UserMgr{
			users:    util.NewRWMap[string, *User](),
			sessions: util.NewRWMap[isession.ISession, string](),
			ops:      util.NewRWMap[int64, func()](),
		}
	})

	return usermgr
}

func InitUserMgr() {
	if _, err := nats.GetNatsConn().Subscribe(constant.UserLoginMsg, func(msg *_nats.Msg) {
		ncMsg := &gate_way.NatsLoginRequest{}
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

	any := &anypb.Any{}
	any.MarshalFrom(&gate_way.LoginRequest{Id: "123"})
	b, _ := protojson.Marshal(any)
	fmt.Print(string(b))
	msghandler.RegisterHandler(any.TypeUrl, func(session isession.ISession, any *anypb.Any) error {
		loginResp := &gate_way.LoginResp{
			Code: common.ErrorCode_OK,
		}

		any1 := &anypb.Any{}
		err1 := any1.MarshalFrom(loginResp)
		if err1 != nil {
			klog.Errorf("[GATEWAY-LOGIN-MARSHAL] marshal %s failed, err: %v", loginResp.String(), err1)
			return err1
		}
		if id, ok := GetUserMgr().sessions.Get(session); ok {
			klog.Errorf("[GATEWAY-SESSION-EXIST] session %s already login", id)
			loginResp.Code = common.ErrorCode_USER_SESSION_EXIST
			any1.MarshalFrom(loginResp)
			session.Send(any1)
			return nil
		}
		idx, _ := common_redis.GetRedis().Incr(context.TODO(), constant.UserLoginMsgIdx).Result()
		loginRequest := &gate_way.LoginRequest{}
		err := any.UnmarshalTo(loginRequest)
		if err != nil {
			klog.Errorf("[GATEWAY-LOGIN-UNMARSHAL] unmarshal %s failed, err: %v", any.TypeUrl, err)
			return err
		}

		klog.Infof("[GATEWAY-LOGIN-RECV] recv %s", loginRequest.String())
		ncMsg := &gate_way.NatsLoginRequest{
			Id:  loginRequest.Id,
			Idx: idx,
		}
		ncMsgb, err := proto.Marshal(ncMsg)
		if err != nil {
			klog.Errorf("[GATEWAY-NATS-MARSHAL] marshal %s failed, err: %v", ncMsg.String(), err)
			return err
		}

		GetUserMgr().addOp(idx, func() {
			if sub, err := nats.GetNatsConn().Subscribe(fmt.Sprintf(constant.UserMsg, loginRequest.Id), func(msg *_nats.Msg) {
				any := &anypb.Any{}
				err := proto.Unmarshal(msg.Data, any)
				if err != nil {
					klog.Errorf("[GATEWAY-USER-MSG-UNMARSHAL] unmarshal %s failed, err: %v", string(msg.Data), err)
					return
				}
				klog.Infof("[GATEWAY-USER-MSG-RECV] recv %s", any.String())
				session.Send(any)
			}); err != nil {
				klog.Errorf("[GATEWAY-SUBSCRIBE-FAIL] subscribe %s failed, err: %v", fmt.Sprintf(constant.UserMsg, loginRequest.Id), err)
				return
			} else {
				GetUserMgr().addUser(loginRequest.Id, session, sub)
				GetUserMgr().removeOp(idx)
			}

			klog.Infof("[GATEWAY-LOGIN-SEND] send %s", loginResp.String())
			session.Send(any1)
		})

		nats.GetNatsConn().Publish(constant.UserLoginMsg, ncMsgb)
		// nats.GetNatsConn().Flush()
		klog.Infof("[GATEWAY-NATS-PUBLISH] publish %s success", constant.UserLoginMsg)

		return nil
	})

}
