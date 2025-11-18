package manager

import (
	"context"
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
		Code: common.ErrorCode_OK,
	}

	defer func() {
		klog.CtxInfof(ctx, "[SERVER-MGR-CREATE-004] resp: %d", resp.Code)
	}()

	if err1, clusterIP, tcpPort, _ := pod.StartGameServer(ctx, idClient.Generate().Int64(), []string{""}); err1 != nil {
		klog.CtxErrorf(ctx, "[SERVER-MGR-CREATE-007] CreateServer: failed to start game server, error: %v", err1)
		resp.Code = common.ErrorCode_SERVER_MGR_CREATE_FAILED
		resp.Msg = "failed to start game server"
		err = err1
	} else {
		resp.GamePort = tcpPort
		resp.GameAddr = clusterIP
		klog.CtxInfof(ctx, "[SERVER-MGR-CREATE-008] CreateServer: successfully created server, tcpPort: %d", tcpPort)
	}

	resp.Msg = resp.Code.String()
	return resp, err
}

func (s *ServerManager) CreateAiClient(ctx context.Context, req *server_mgr.CreateAiClientReq) (resp *server_mgr.CreateAiClientRsp, err error) {
	resp = &server_mgr.CreateAiClientRsp{
		Code: common.ErrorCode_OK,
	}
	defer func() {
		klog.CtxInfof(ctx, "[SERVER-MGR-CREATE-005] resp: %d", resp.Code)
	}()

	if err1, clusterIP, tcpPort, _ := pod.StartAiClient(ctx, idClient.Generate().Int64(), []string{"-server_ip", req.GameAddr}); err1 != nil {
		klog.CtxErrorf(ctx, "[SERVER-MGR-CREATE-007] CreateServer: failed to start game server, error: %v", err1)
		resp.Code = common.ErrorCode_SERVER_MGR_CREATE_FAILED
		resp.Msg = "failed to start ai client"
		err = err1
	} else {
		resp.GamePort = tcpPort
		resp.GameAddr = clusterIP
		klog.CtxInfof(ctx, "[SERVER-MGR-CREATE-008] CreateAiClient: successfully created ai client, tcpPort: %d", tcpPort)
	}

	return resp, err
}
