package manager

import (
	"context"
	common_config "match_server/config"
	"match_server/kitex_gen/common"
	"match_server/kitex_gen/user_center"
	common_redis "match_server/redis"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
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

	return resp, nil
}
