package manager

import (
	"context"
	"fmt"
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
	user_mgr            UserManager
	once_user_bag_mgr   sync.Once
	IdClient            *snowflake.Node
	redis_taptap_str    = "taptap_platform"
	redis_user_info_key = "user_info"
	redis_test_platform = "test_platform"
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
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}

	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[USER-LOGIN-RESULT] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)

	realId := common_redis.GetRedis().Get(ctx, fmt.Sprintf("%s:%s", redis_taptap_str, userId)).Val()
	if realId == "" {
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
			klog.CtxErrorf(ctx, "[USER-LOGIN-TAP-FAIL] tap GetHandle failed")
			resp.Code = common.ErrorCode_USER_LOGIN_TAP_FAIL
			return resp, nil
		}
		realId = userId
		common_redis.GetRedis().Set(ctx, fmt.Sprintf("%s:%s", redis_taptap_str, userId), realId, 0)
		common_redis.GetRedis().HSet(ctx, fmt.Sprintf("%s:%s", redis_user_info_key, realId), "avatar", tapBaseInfo.Data.Avatar, "gender", tapBaseInfo.Data.Gender, "name", tapBaseInfo.Data.Name, "openid", tapBaseInfo.Data.Openid, "unionid", tapBaseInfo.Data.Unionid)
	}

	userInfo := common_redis.GetRedis().HGetAll(ctx, fmt.Sprintf("%s:%s", redis_user_info_key, realId)).Val()

	resp.TapInfo = &user_center.TapInfo{
		Avatar:  userInfo["avatar"],
		Gender:  userInfo["gender"],
		Name:    userInfo["name"],
		Openid:  userInfo["openid"],
		Unionid: userInfo["unionid"],
	}

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

func (x *UserManager) UserInfo(ctx context.Context, req *user_center.UserInfoReq) (resp *user_center.UserInfoRsp, err error) {
	resp = &user_center.UserInfoRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}

	userInfo := common_redis.GetRedis().HGetAll(ctx, fmt.Sprintf("%s:%s", redis_user_info_key, req.Openid)).Val()

	//检查redis是否有该用户信息
	if _, ok := userInfo["openid"]; !ok {
		resp.Code = common.ErrorCode_USER_INFO_NOT_FOUND
		resp.Msg = "user info not found"
		return resp, nil
	}

	resp.Data = &user_center.UserInfoRsp_Data{
		TapInfo: &user_center.TapInfo{
			Avatar:  userInfo["avatar"],
			Gender:  userInfo["gender"],
			Name:    userInfo["name"],
			Openid:  userInfo["openid"],
			Unionid: userInfo["unionid"],
		},
	}

	return resp, nil
}

func (x *UserManager) TestLogin(ctx context.Context, req *user_center.TestLoginReq) (resp *user_center.TestLoginRsp, err error) {
	resp = &user_center.TestLoginRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}

	userId := ctx.Value("userId").(string)

	// 使用Lua脚本确保原子操作，防止并发重复创建
	luaScript := `
	local test_platform_key = KEYS[1]
	local openid_arg = ARGV[1]
	local avatar = ARGV[2]
	local gender = ARGV[3]
	local name = ARGV[4]
	local unionid = ARGV[5]

	-- 先尝试从 testPlatformKey 获取 openid
	local saved_openid = redis.call('GET', test_platform_key)
	local openid = saved_openid
	if openid == false or openid == nil then
		-- 如果没有保存的 openid，则使用传入的 openid
		openid = openid_arg
	end

	-- 构建用户信息 key
	local user_info_key = 'user_info:' .. openid

	-- 检查用户是否存在
	local exists = redis.call('HEXISTS', user_info_key, 'openid')
	if exists == 1 then
		-- 用户存在，更新 testPlatformKey 为当前 openid
		redis.call('SET', test_platform_key, openid)
		-- 返回用户信息
		local user_info = redis.call('HGETALL', user_info_key)
		return {1, user_info}
	else
		-- 用户不存在，创建新用户
		redis.call('SET', test_platform_key, openid)
		redis.call('HSET', user_info_key, 
			'avatar', avatar, 
			'gender', gender, 
			'name', name, 
			'openid', openid, 
			'unionid', unionid)
		-- 返回创建的用户信息
		local user_info = redis.call('HGETALL', user_info_key)
		return {0, user_info}
	end
	`

	testPlatformKey := fmt.Sprintf("%s:%s", redis_test_platform, userId)

	keys := []string{testPlatformKey}
	args := []interface{}{
		req.TapInfo.Openid,
		req.TapInfo.Avatar,
		req.TapInfo.Gender,
		req.TapInfo.Name,
		req.TapInfo.Unionid,
	}

	result, err := common_redis.GetRedis().Eval(ctx, luaScript, keys, args).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[TEST-LOGIN-LUA-ERROR] eval lua script err: %v", err)
		resp.Code = common.ErrorCode_FAILED
		resp.Msg = "lua script execution failed"
		return resp, err
	}

	// 解析Lua脚本返回结果
	resultArray, ok := result.([]interface{})
	if !ok {
		klog.CtxErrorf(ctx, "[TEST-LOGIN-LUA-ERROR] invalid lua result format")
		resp.Code = common.ErrorCode_FAILED
		resp.Msg = "invalid lua result format"
		return resp, nil
	}

	userExists := resultArray[0].(int64)
	// 解析用户信息
	userInfoArray := resultArray[1].([]interface{})
	userInfo := make(map[string]string)
	for i := 0; i < len(userInfoArray); i += 2 {
		key := userInfoArray[i].(string)
		value := userInfoArray[i+1].(string)
		userInfo[key] = value
	}

	if userExists == 1 {
		// 用户存在
		resp.Msg = "user exists"
	} else {
		// 用户不存在，已创建新用户
		resp.Msg = "user created"
	}

	// 设置用户信息到响应中
	resp.Data = &user_center.TestLoginRsp_Data{
		TapInfo: &user_center.TapInfo{
			Avatar:  userInfo["avatar"],
			Gender:  userInfo["gender"],
			Name:    userInfo["name"],
			Openid:  userInfo["openid"],
			Unionid: userInfo["unionid"],
		},
	}

	return resp, nil
}
