package manager

import (
	"context"
	"fmt"
	"match_server/kitex_gen/common"
	"match_server/kitex_gen/match_proto"
	"match_server/logic/match"
	common_redis "match_server/redis"
	"match_server/shell"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type MatchManager struct {
}

var (
	match_mgr      MatchManager
	once_match_mgr sync.Once
	idClient       *snowflake.Node
)

func GetMatchManager() *MatchManager {
	once_match_mgr.Do(func() {
		match_mgr = MatchManager{}

		key := "match_server:snowflake:node"
		n, err := common_redis.GetRedis().Incr(context.Background(), key).Result()
		if err != nil {
			klog.Fatal("[MATCH-MANAGER-INIT] MatchManager: gen uuid creator err: %v", err)
		}

		nodeIdx := n % (1 << snowflake.NodeBits)
		if node, err := snowflake.NewNode(nodeIdx); err != nil {
			klog.Fatal("[MATCH-MANAGER-NODE] MatchManager: gen uuid creator err: %v", err)
		} else {
			klog.Infof("[MATCH-MANAGER-NODE-OK] MatchManager: gen uuid creator success, node: %d", nodeIdx)
			idClient = node
		}

		match.GetMatchProcess().SetAfterMatched(func(r, b []int64) {
			shell.StartCmd(fmt.Sprintf("echo match success, r: %v, b: %v", r, b))
		})
	})
	return &match_mgr
}

func (x *MatchManager) Match(ctx context.Context, req *match_proto.MatchReq) (resp *match_proto.MatchResp, err error) {
	resp = &match_proto.MatchResp{
		Code: common.ErrorCode_OK,
	}

	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[MATCH-RESULT] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)

	if common_redis.GetRedis().SetNX(ctx, fmt.Sprintf("match_server_op:user:%s", userId), userId, time.Second*1).Val() == false {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s", userId)
		return resp, nil
	}

	if e := common_redis.GetRedis().Get(ctx, fmt.Sprintf("match_user:%s", userId)).Err(); e != redis.Nil {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s already match, err: %v", userId, e)
		return resp, e
	}

	id := idClient.Generate().Int64()
	common_redis.GetRedis().Set(ctx, fmt.Sprintf("match_user:%s", userId), id, 0)
	common_redis.GetRedis().SAdd(ctx, fmt.Sprintf("match_group:%d", id), userId)

	if !match.GetMatchProcess().AddMatch(id, 1, 1) {
		resp.Code = common.ErrorCode_FAILED
		common_redis.GetRedis().Del(ctx, fmt.Sprintf("match_user:%s", userId))
		common_redis.GetRedis().Del(ctx, fmt.Sprintf("match_group:%d", id))
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s add match failed", userId)
		return resp, nil
	}

	return resp, nil
}
