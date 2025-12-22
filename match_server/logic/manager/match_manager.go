package manager

import (
	"context"
	"fmt"
	common_config "match_server/config"
	"match_server/kitex_gen/common"
	"match_server/kitex_gen/gate_way"
	"match_server/kitex_gen/match_proto"
	"match_server/kitex_gen/server_mgr"
	"match_server/logic/match"
	common_redis "match_server/redis"
	"match_server/rpc"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/protobuf/types/known/anypb"
)

type MatchManager struct {
}

var (
	match_mgr      MatchManager
	once_match_mgr sync.Once
	idClient       *snowflake.Node

	userGameInfoKey = "match_server:user_game:%s"
	createGameKey   = "match_server:create_game:%s"
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
			resp_create_server, err := rpc.ServerMgrClient.CreateServer(context.Background(), &server_mgr.CreateServerReq{})
			if err != nil {
				klog.CtxErrorf(context.Background(), "[MATCH-EXIST] uuid: %v %v create server failed, err: %v", r, b, err)
				return
			}

			time.Sleep(time.Second * 1)
			match_info_ntf := &match_proto.MatchInfoNtf{
				R: make([]string, 0),
				B: make([]string, 0),
			}
			game_info_ntf := &match_proto.GameInfoNtf{
				GameAddr: common_config.Get("game.addr").(string),
				GamePort: int32(resp_create_server.GamePort),
			}
			for _, v := range r {
				members, _ := common_redis.GetRedis().SMembers(context.Background(), fmt.Sprintf("match_group:%d", v)).Result()
				match_info_ntf.R = append(match_info_ntf.R, members...)
				common_redis.GetRedis().Del(context.Background(), fmt.Sprintf("match_group:%d", v))

				common_redis.GetRedis().HSetEX(context.TODO(), fmt.Sprintf(userGameInfoKey, v), "game_port", strconv.Itoa(int(game_info_ntf.GamePort)))
				common_redis.GetRedis().HSetEX(context.TODO(), fmt.Sprintf(userGameInfoKey, v), "game_addr", game_info_ntf.GameAddr)

				common_redis.GetRedis().Expire(context.TODO(), fmt.Sprintf(userGameInfoKey, v), time.Second*60*50)
			}
			for _, v := range b {
				members, _ := common_redis.GetRedis().SMembers(context.Background(), fmt.Sprintf("match_group:%d", v)).Result()
				match_info_ntf.B = append(match_info_ntf.B, members...)
				common_redis.GetRedis().Del(context.Background(), fmt.Sprintf("match_group:%d", v))

				common_redis.GetRedis().HSetEX(context.TODO(), fmt.Sprintf(userGameInfoKey, v), "game_port", strconv.Itoa(int(game_info_ntf.GamePort)))
				common_redis.GetRedis().HSetEX(context.TODO(), fmt.Sprintf(userGameInfoKey, v), "game_addr", game_info_ntf.GameAddr)

				common_redis.GetRedis().Expire(context.TODO(), fmt.Sprintf(userGameInfoKey, v), time.Second*60*50)
			}

			any1 := &anypb.Any{}
			if err := any1.MarshalFrom(match_info_ntf); err != nil {
				klog.Errorf("[MATCH-MANAGER-NTF] MatchManager: marshal match_info_ntf err: %v", err)
			}
			any2 := &anypb.Any{}
			if err := any2.MarshalFrom(game_info_ntf); err != nil {
				klog.Errorf("[MATCH-MANAGER-NTF] MatchManager: marshal game_info_ntf err: %v", err)
				return
			}
			for _, v := range match_info_ntf.R {
				rpc.GatewayClient.UserMsg(context.Background(), &gate_way.UserMsgReq{
					Id:  v,
					Msg: any1,
				})
				rpc.GatewayClient.UserMsg(context.Background(), &gate_way.UserMsgReq{
					Id:  v,
					Msg: any2,
				})
				common_redis.GetRedis().Del(context.Background(), fmt.Sprintf("match_user:%s", v))
			}
			for _, v := range match_info_ntf.B {
				rpc.GatewayClient.UserMsg(context.Background(), &gate_way.UserMsgReq{
					Id:  v,
					Msg: any1,
				})
				rpc.GatewayClient.UserMsg(context.Background(), &gate_way.UserMsgReq{
					Id:  v,
					Msg: any2,
				})
				common_redis.GetRedis().Del(context.Background(), fmt.Sprintf("match_user:%s", v))
			}
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

	if r, err := common_redis.GetRedis().SetNX(ctx, fmt.Sprintf(createGameKey, userId), userId, time.Second*1).Result(); err != nil {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s create game failed, err: %v", userId, err)
		return resp, err
	} else if !r {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s create game failed, err: %v", userId, err)
		return resp, err
	}

	defer func() {
		common_redis.GetRedis().Del(ctx, fmt.Sprintf(createGameKey, userId))
	}()

	game_info, err := common_redis.GetRedis().HGetAll(ctx, fmt.Sprintf(userGameInfoKey, userId)).Result()
	if err != nil {
		if err != redis.Nil {
			resp.Code = common.ErrorCode_FAILED
			klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s get game info failed, err: %v", userId, err)
			return resp, err
		}
	} else {
		game_info_ntf := &match_proto.GameInfoNtf{
			GameAddr: common_config.Get("game.addr").(string),
			GamePort: 0,
		}
		if conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", game_info["game_addr"], "10085"), time.Second*1); err == nil {
			conn.Close()
			if port, ok := game_info["game_port"]; ok {
				if p, err := strconv.Atoi(port); err == nil {
					game_info_ntf.GamePort = int32(p)
				}
				game_info_ntf.GameAddr = common_config.Get("game.addr").(string)
			}
			game_info_ntf.GameAddr = common_config.Get("game.addr").(string)
			rpc.GatewayClient.UserMsg(ctx, &gate_way.UserMsgReq{
				Id: userId,
				Msg: func() *anypb.Any {
					any := &anypb.Any{}
					if err = any.MarshalFrom(game_info_ntf); err != nil {
						klog.Errorf("[MATCH-MANAGER-NTF] MatchManager: marshal game_info_ntf err: %v", err)
					}
					return any
				}(),
			})
			return resp, nil
		} else {
			klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s addr %v connect game info failed, err: %v", userId, game_info, err)
		}
	}

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

func (x *MatchManager) Pve(ctx context.Context, req *match_proto.PveReq) (resp *match_proto.PveResp, err error) {
	resp = &match_proto.PveResp{
		Code: common.ErrorCode_OK,
	}

	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[MATCH-RESULT] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)

	if r, err := common_redis.GetRedis().SetNX(ctx, fmt.Sprintf(createGameKey, userId), userId, time.Second*1).Result(); err != nil {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s create game failed, err: %v", userId, err)
		return resp, err
	} else if !r {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s create game failed, err: %v", userId, err)
		return resp, err
	}

	defer func() {
		common_redis.GetRedis().Del(ctx, fmt.Sprintf(createGameKey, userId))
	}()

	game_info_ntf := &match_proto.GameInfoNtf{
		GameAddr: common_config.Get("game.addr").(string),
		GamePort: 0,
	}

	game_info, err := common_redis.GetRedis().HGetAll(ctx, fmt.Sprintf(userGameInfoKey, userId)).Result()
	if err != nil {
		if err != redis.Nil {
			resp.Code = common.ErrorCode_FAILED
			klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s get game info failed, err: %v", userId, err)
			return resp, err
		}
	} else {
		if conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", game_info["game_addr"], "10085"), time.Second*1); err == nil {
			conn.Close()
			if port, ok := game_info["game_port"]; ok {
				if p, err := strconv.Atoi(port); err == nil {
					game_info_ntf.GamePort = int32(p)
				}
				game_info_ntf.GameAddr = common_config.Get("game.addr").(string)
			}
			game_info_ntf.GameAddr = common_config.Get("game.addr").(string)
			rpc.GatewayClient.UserMsg(ctx, &gate_way.UserMsgReq{
				Id: userId,
				Msg: func() *anypb.Any {
					any := &anypb.Any{}
					if err = any.MarshalFrom(game_info_ntf); err != nil {
						klog.Errorf("[MATCH-MANAGER-NTF] MatchManager: marshal game_info_ntf err: %v", err)
					}
					return any
				}(),
			})
			return resp, nil
		} else {
			klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s addr %v connect game info failed, err: %v", userId, game_info, err)
		}
	}

	resp_create_server, err := rpc.ServerMgrClient.CreateServer(ctx, &server_mgr.CreateServerReq{})
	if err != nil {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s create server failed, err: %v", userId, err)
		return resp, err
	}
	time.Sleep(time.Second * 1)
	_, err = rpc.ServerMgrClient.CreateAiClient(ctx, &server_mgr.CreateAiClientReq{
		GameAddr: resp_create_server.GameAddr,
	})
	if err != nil {
		resp.Code = common.ErrorCode_FAILED
		klog.CtxErrorf(ctx, "[MATCH-EXIST] uuid: %s create ai client failed, err: %v", userId, err)
		return resp, err
	}

	game_info_ntf.GameAddr = common_config.Get("game.addr").(string)
	game_info_ntf.GamePort = resp_create_server.GamePort

	time.Sleep(time.Second * 1)

	common_redis.GetRedis().HSet(ctx, fmt.Sprintf(userGameInfoKey, userId), "game_port", strconv.Itoa(int(game_info_ntf.GamePort)), "game_addr", resp_create_server.GameAddr)

	common_redis.GetRedis().Expire(ctx, fmt.Sprintf(userGameInfoKey, userId), time.Second*60*5)

	rpc.GatewayClient.UserMsg(ctx, &gate_way.UserMsgReq{
		Id: userId,
		Msg: func() *anypb.Any {
			any := &anypb.Any{}
			if err = any.MarshalFrom(game_info_ntf); err != nil {
				klog.Errorf("[MATCH-MANAGER-NTF] MatchManager: marshal game_info_ntf err: %v", err)
			}
			return any
		}(),
	})

	return resp, nil
}
