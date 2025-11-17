package manager

import (
	"context"
	common_config "server_manager/config"
	"server_manager/kitex_gen/common"
	"server_manager/kitex_gen/server_mgr"
	"server_manager/pod"
	common_redis "server_manager/redis"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
)

type ServerManager struct {
}

var (
	match_mgr      ServerManager
	once_match_mgr sync.Once
	idClient       *snowflake.Node
)

func GetServerManager() *ServerManager {
	once_match_mgr.Do(func() {
		match_mgr = ServerManager{}

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
	})
	return &match_mgr
}

func (s *ServerManager) CreateServer(ctx context.Context, req *server_mgr.CreateServerReq) (resp *server_mgr.CreateServerRsp, err error) {
	resp = &server_mgr.CreateServerRsp{
		Code:     common.ErrorCode_OK,
		GameAddr: common_config.Get("pod.game_addr").(string),
	}

	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[MATCH-RESULT] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)

	if err, tcpPort, _ := pod.StartGameServer(ctx, idClient.Generate().Int64(), userId); err != nil {
		klog.CtxErrorf(ctx, "start game server failed, err: %v", err)
		return &server_mgr.CreateServerRsp{
			Code: common.ErrorCode_FAILED,
			Msg:  "start game server failed",
		}, err
	} else {
		resp.GamePort = tcpPort
	}
	resp.Msg = resp.Code.String()
	return resp, nil
}

func (s *ServerManager) CreateAiClient(ctx context.Context, req *server_mgr.CreateAiClientReq) (resp *server_mgr.CreateAiClientRsp, err error) {
	return &server_mgr.CreateAiClientRsp{}, nil
}
