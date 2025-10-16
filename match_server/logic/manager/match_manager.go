package manager

import (
	"context"
	"match_server/kitex_gen/common"
	"match_server/kitex_gen/match_proto"
	common_redis "match_server/redis"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
)

type MatchManager struct {
}

var (
	match_mgr          MatchManager
	once_match_mgr sync.Once
	IdClient          *snowflake.Node
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
			IdClient = node
		}
	})
	return &match_mgr
}

func (x *MatchManager) Match(ctx context.Context, req *match_proto.MatchReq) (resp *match_proto.MatchResp, err error) {
	return &match_proto.MatchResp{
		Error: common.ErrorCode_OK,
	}, nil
}
