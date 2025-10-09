package manager

import (
	"context"
	"sync"
	common_config "user_server/config"
	"user_server/kitex_gen/common"
	"user_server/kitex_gen/user_center"
	"user_server/logic/tap"
	common_redis "user_server/redis"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	item_operator         = "item_operator"
	winedroplets_giveback = "winedroplets_giveback"
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
			klog.Fatal("UserManager: gen uuid creator err: %v", err)
		}

		nodeIdx := n % (1 << snowflake.NodeBits)
		if node, err := snowflake.NewNode(nodeIdx); err != nil {
			klog.Fatal("UserManager: gen uuid creator err: %v", err)
		} else {
			klog.Infof("UserManager: gen uuid creator success, node: %d", nodeIdx)
			IdClient = node
		}
	})
	return &user_mgr
}

func (x *UserManager) Login(ctx context.Context, req *user_center.LoginReq) (resp *user_center.LoginRsp, err error) {
	resp = &user_center.LoginRsp{
		Code:       common.ErrorCode_Ok,
		Msg:        "success",
		Name:       "",
		ServerAddr: common_config.Get("game.addr").(string),
		ServerPort: int32(common_config.Get("game.port").(int)),
	}

	uuid := ""
	defer func() {
		klog.CtxInfof(ctx, "uuid: %d, resp: %d", uuid, resp.Code)
	}()

	uuid = ctx.Value("userId").(string)
	resp.Id = uuid

	tapResp, err := tap.GetHandle(ctx, req.Kid, req.MacKey, common_config.Get("tap.base_info_uri").(string))
	if err != nil {
		klog.CtxErrorf(ctx, "tap GetHandle err: %v", err)
		return nil, err
	}
	tapBaseInfo := user_center.TapBaseInfo{}
	err = protojson.Unmarshal([]byte(tapResp), &tapBaseInfo)
	if err != nil {
		klog.CtxErrorf(ctx, "tap UnmarshalTo err: %v", err)
		return nil, err
	}
	if !tapBaseInfo.Success {
		klog.CtxErrorf(ctx, "tap GetHandle err: %v", err)
		return nil, err
	}
	resp.TapInfo = tapBaseInfo.Data

	return resp, nil
}
