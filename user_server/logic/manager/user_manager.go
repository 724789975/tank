package manager

import (
	"context"
	"sync"
	common_config "user_server/config"
	"user_server/kitex_gen/common"
	"user_server/kitex_gen/gate_way"
	"user_server/kitex_gen/user_center"
	"user_server/logic/tap"
	common_redis "user_server/redis"
	"user_server/rpc"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
)

type UserManager struct {
}

var (
	user_mgr          UserManager
	once_user_bag_mgr sync.Once
	IdClient          *snowflake.Node
)

func GetUserManager() *UserManager {
	once_user_bag_mgr.Do(func() {
		user_mgr = UserManager{}

		key := "user_svr:snowflake:node"
		n, err := common_redis.GetRedis().Incr(context.Background(), key).Result()
		if err != nil {
			klog.Fatal("[USER-MANAGER-INIT] UserManager: gen uuid creator err: %v", err)
		}

		nodeIdx := n % (1 << snowflake.NodeBits)
		if node, err := snowflake.NewNode(nodeIdx); err != nil {
			klog.Fatal("[USER-MANAGER-NODE] UserManager: gen uuid creator err: %v", err)
		} else {
			klog.Infof("[USER-MANAGER-NODE-OK] UserManager: gen uuid creator success, node: %d", nodeIdx)
			IdClient = node
		}
	})
	return &user_mgr
}

func (x *UserManager) Login(ctx context.Context, req *user_center.LoginReq) (resp *user_center.LoginRsp, err error) {
	resp = &user_center.LoginRsp{
		Code:       common.ErrorCode_OK,
		Msg:        "success",
		ServerAddr: common_config.Get("game.addr").(string),
		ServerPort: int32(common_config.Get("game.port").(int)),
	}

	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[USER-LOGIN-RESULT] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)

	tapResp, err := tap.GetHandle(ctx, req.Kid, req.MacKey, common_config.Get("tap.base_info_uri").(string))
	if err != nil {
		klog.CtxErrorf(ctx, "[USER-LOGIN-TAP-ERROR] tap GetHandle err: %v", err)
		resp.Code = common.ErrorCode_USER_LOGIN_TAP_ERROR
		return nil, err
	}
	tapBaseInfo := user_center.TapBaseInfo{}
	err = protojson.Unmarshal([]byte(tapResp), &tapBaseInfo)
	if err != nil {
		klog.CtxErrorf(ctx, "[USER-LOGIN-UNMARSHAL] tap UnmarshalTo err: %v", err)
		resp.Code = common.ErrorCode_USER_LOGIN_UNMARSHAL
		return nil, err
	}
	if !tapBaseInfo.Success {
		klog.CtxErrorf(ctx, "[USER-LOGIN-TAP-FAIL] tap GetHandle err: %v", err)
		resp.Code = common.ErrorCode_USER_LOGIN_TAP_FAIL
		return nil, err
	}
	resp.TapInfo = tapBaseInfo.Data

	test := &gate_way.Test{
		Test: "test",
	}

	any := &anypb.Any{}
	err = any.MarshalFrom(test)
	if err != nil {
		return nil, err
	}

	rpc.GatewayClient.UserMsg(ctx, &gate_way.UserMsgReq{
		Id:  userId,
		Msg: any,
	})

	return resp, nil
}
