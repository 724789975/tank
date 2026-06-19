// Package usermgr 提供用户管理功能
// 负责用户会话管理、登录流程处理以及 NATS 消息订阅管理
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
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/golang/protobuf/proto"
	_nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

// User 表示一个在线用户
// 包含用户ID、会话信息和 NATS 订阅
type User struct {
	id      string              // 用户唯一标识
	session isession.ISession   // 用户的 WebSocket 会话
	sub     *_nats.Subscription // NATS 消息订阅（用于接收用户消息）
}

// UserMgr 管理所有在线用户
// 使用三个并发安全的 map 来管理用户、会话和操作回调
type UserMgr struct {
	users    *util.RWMap[string, *User]                    // 用户ID -> User 对象
	sessions *util.RWMap[isession.ISession, string]        // Session -> 用户ID（反向映射）
	ops      *util.RWMap[int64, func(ctx context.Context)] // 登录操作索引 -> 回调函数
}

// addUser 添加用户到管理器
// userId: 用户ID
// _session: 用户的 WebSocket 会话
// _sub: 用户的 NATS 订阅
func (u *UserMgr) addUser(userId string, _session isession.ISession, _sub *_nats.Subscription) {
	u.users.Set(userId, &User{
		id:      userId,
		session: _session,
		sub:     _sub,
	})
	u.sessions.Set(_session, userId)
}

// removeUser 移除用户
// 会取消用户的 NATS 订阅并清理相关资源
// userId: 要移除的用户ID
func (u *UserMgr) removeUser(userId string) {
	if user, ok := u.users.Get(userId); ok {
		u.sessions.Delete(user.session)
		// 取消 NATS 订阅
		if err := user.sub.Unsubscribe(); err != nil {
			klog.Errorf("[GATEWAY-UNSUBSCRIBE-FAIL] unsubscribe %s failed, err: %v", userId, err)
		}
	}
	u.users.Delete(userId)
}

// RemoveSession 根据 Session 移除用户
// 当 WebSocket 连接关闭时调用此方法清理用户信息
// session: 要移除的会话
func (u *UserMgr) RemoveSession(session isession.ISession) {
	if userId, ok := u.sessions.Get(session); ok {
		u.removeUser(userId)
	}
}

// getUserSession 获取用户的会话
// userId: 用户ID
// 返回: 会话实例和是否存在
func (u *UserMgr) getUserSession(userId string) (isession.ISession, bool) {
	user, b := u.users.Get(userId)
	if b {
		return user.session, true
	}
	return nil, false
}

// addOp 添加登录操作回调
// 在登录流程中使用，用于异步处理登录响应
// idx: 操作索引（来自 Redis 的自增ID）
// f: 回调函数
func (u *UserMgr) addOp(idx int64, f func(ctx context.Context)) {
	u.ops.Set(idx, f)
}

// removeOp 移除登录操作回调
// idx: 操作索引
func (u *UserMgr) removeOp(idx int64) {
	u.ops.Delete(idx)
}

// getOp 获取登录操作回调
// idx: 操作索引
// 返回: 回调函数和是否存在
func (u *UserMgr) getOp(idx int64) (func(ctx context.Context), bool) {
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
			ops:      util.NewRWMap[int64, func(ctx context.Context)](),
		}
	})

	return usermgr
}

func InitUserMgr() {
	if _, err := nats.GetNatsConn().Subscribe(constant.UserLoginMsg, func(msg *_nats.Msg) {
		carrier := propagation.MapCarrier{}
		for k, v := range msg.Header {
			carrier[k] = v[0]
		}
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
		ctx, span := otel.Tracer("nats-consume").Start(ctx, "nats-consume", trace.WithAttributes(
			attribute.String("subject", msg.Subject),
		))
		defer span.End()

		klog.CtxInfof(ctx, "[GATEWAY-NATS-CONSUME] recv %s", msg.Subject)
		ncMsg := &gate_way.NatsLoginRequest{}
		err := proto.Unmarshal(msg.Data, ncMsg)
		if err != nil {
			klog.CtxErrorf(ctx, "[GATEWAY-NATS-CONSUME] unmarshal %s failed, err: %v", msg.Subject, err)
			return
		}

		if session, ok := GetUserMgr().getUserSession(ncMsg.Id); ok {
			session.Send(nil)
			GetUserMgr().removeUser(ncMsg.Id)
		}

		if op, ok := GetUserMgr().getOp(ncMsg.Idx); ok {
			op(ctx)
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

		loginRequest := &gate_way.LoginRequest{}
		err := any.UnmarshalTo(loginRequest)
		if err != nil {
			klog.Errorf("[GATEWAY-LOGIN-UNMARSHAL] unmarshal %s failed, err: %v", any.TypeUrl, err)
			return err
		}

		if common_redis.GetRedis().SetNX(context.TODO(), fmt.Sprintf(constant.UserLoginRedisKey, loginRequest.Id), 1, time.Second*3).Val() == false {
			loginResp.Code = common.ErrorCode_USER_SESSION_EXIST
			any1.MarshalFrom(loginResp)
			session.Send(any1)

			klog.Errorf("[GATEWAY-SESSION-EXIST] session %s is logining", loginRequest.Id)
			return nil
		}
		idx, _ := common_redis.GetRedis().Incr(context.TODO(), constant.UserLoginMsgIdx).Result()
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

		GetUserMgr().addOp(idx, func(ctx context.Context) {
			if sub, err := nats.GetNatsConn().Subscribe(fmt.Sprintf(constant.UserMsg, loginRequest.Id), func(msg *_nats.Msg) {
				any := &anypb.Any{}
				err := proto.Unmarshal(msg.Data, any)
				if err != nil {
					GetUserMgr().removeOp(idx) // 失败时也要清理
					klog.CtxErrorf(ctx, "[GATEWAY-USER-MSG-UNMARSHAL] unmarshal %s failed, err: %v", string(msg.Data), err)
					return
				}
				klog.CtxInfof(ctx, "[GATEWAY-USER-MSG-RECV] recv %s", any.String())
				session.Send(any)
			}); err != nil {
				klog.CtxErrorf(ctx, "[GATEWAY-SUBSCRIBE-FAIL] subscribe %s failed, err: %v", fmt.Sprintf(constant.UserMsg, loginRequest.Id), err)
				return
			} else {
				GetUserMgr().addUser(loginRequest.Id, session, sub)
				GetUserMgr().removeOp(idx)
			}

			klog.CtxInfof(ctx, "[GATEWAY-LOGIN-SEND] send %s", loginResp.String())
			session.Send(any1)
		})

		nats.GetNatsConn().Publish(constant.UserLoginMsg, ncMsgb)
		// nats.GetNatsConn().Flush()
		klog.Infof("[GATEWAY-NATS-PUBLISH] publish %s success", constant.UserLoginMsg)

		return nil
	})

}
