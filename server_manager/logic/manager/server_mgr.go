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
			klog.Fatal("[SERVER-MGR-INIT-001] ServerManager: failed to generate UUID creator, key: %s, error: %v", key, err)
		}

		nodeIdx := n % (1 << snowflake.NodeBits)
		if node, err := snowflake.NewNode(nodeIdx); err != nil {
			klog.Fatal("[SERVER-MGR-INIT-002] ServerManager: failed to create snowflake node, nodeIdx: %d, error: %v", nodeIdx, err)
		} else {
			klog.Infof("[SERVER-MGR-INIT-003] ServerManager: successfully initialized UUID creator, nodeIdx: %d", nodeIdx)
			idClient = node
		}
	})
	return &match_mgr
}

func (s *ServerManager) CreateServer(ctx context.Context, req *server_mgr.CreateServerReq) (resp *server_mgr.CreateServerRsp, err error) {
	resp = &server_mgr.CreateServerRsp{
		Code:     common.ErrorCode_OK,
		GameAddr: common_config.Get("pod.server_addr").(string),
	}

	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[SERVER-MGR-CREATE-004] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)

	if err, tcpPort, _ := pod.StartGameServer(ctx, idClient.Generate().Int64(), userId); err != nil {
		klog.CtxErrorf(ctx, "[SERVER-MGR-CREATE-007] CreateServer: userId: %s, failed to start game server, error: %v", userId, err)
		return &server_mgr.CreateServerRsp{
			Code: common.ErrorCode_SERVER_MGR_CREATE_FAILED,
			Msg:  "failed to start game server",
		}, err
	} else {
		resp.GamePort = tcpPort
		klog.CtxInfof(ctx, "[SERVER-MGR-CREATE-008] CreateServer: userId: %s, successfully created server, tcpPort: %d", userId, tcpPort)
	}
	
	resp.Msg = resp.Code.String()
	return resp, nil
}

func (s *ServerManager) CreateAiClient(ctx context.Context, req *server_mgr.CreateAiClientReq) (resp *server_mgr.CreateAiClientRsp, err error) {
	resp = &server_mgr.CreateAiClientRsp{
		Code:     common.ErrorCode_OK,
		GameAddr: common_config.Get("pod.server_addr").(string),
	}
	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[SERVER-MGR-CREATE-005] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)
	

	
	return &server_mgr.CreateAiClientRsp{
		Code: common.ErrorCode_SERVER_MGR_AI_CLIENT_ERROR,
		Msg:  "AI client creation not implemented yet",
	}, nil
}
