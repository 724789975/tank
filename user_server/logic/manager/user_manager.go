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
	idtoken "google.golang.org/api/idtoken"
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
	redis_google_play   = "google_play_platform"
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

func (x *UserManager) GoogleLogin(ctx context.Context, req *user_center.GooglePlayLoginReq) (resp *user_center.GooglePlayLoginRsp, err error) {
	resp = &user_center.GooglePlayLoginRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}

	userId := ""
	defer func() {
		klog.CtxInfof(ctx, "[GOOGLE-LOGIN-RESULT] uuid: %s, resp: %d", userId, resp.Code)
	}()

	userId = ctx.Value("userId").(string)

	// 验证 token 是否存在
	if req.Token == "" {
		klog.CtxErrorf(ctx, "[GOOGLE-LOGIN-ERROR] token is empty")
		resp.Code = common.ErrorCode_USER_LOGIN_GOOGLE_ERROR
		resp.Msg = "token is empty"
		return resp, nil
	}

	// 向 Google 服务器验证 token 并获取用户信息
	// 这里是简化实现，实际应该使用 Google API 客户端库进行验证
	// 示例：https://oauth2.googleapis.com/tokeninfo?id_token={token}
	googleUserInfo, err := verifyGoogleToken(ctx, req.Token)
	if err != nil {
		klog.CtxErrorf(ctx, "[GOOGLE-LOGIN-VERIFY-ERROR] verify google token err: %v", err)
		resp.Code = common.ErrorCode_USER_LOGIN_GOOGLE_VERIFY_FAILED
		resp.Msg = "failed to verify google token"
		return resp, nil
	}

	// 验证返回的用户信息
	if googleUserInfo.Openid == "" {
		klog.CtxErrorf(ctx, "[GOOGLE-LOGIN-ERROR] invalid google user info")
		resp.Code = common.ErrorCode_USER_LOGIN_GOOGLE_ERROR
		resp.Msg = "invalid google user info"
		return resp, nil
	}

	// 确保所有字段都有值
	if googleUserInfo.Avatar == "" {
		googleUserInfo.Avatar = ""
	}
	if googleUserInfo.Gender == "" {
		googleUserInfo.Gender = ""
	}
	if googleUserInfo.Name == "" {
		googleUserInfo.Name = ""
	}
	if googleUserInfo.Unionid == "" {
		googleUserInfo.Unionid = ""
	}

	// 使用Lua脚本确保原子操作，防止并发重复创建
	luaScript := `
	local google_platform_key = KEYS[1]
	local openid = ARGV[1]
	local avatar = ARGV[2]
	local gender = ARGV[3]
	local name = ARGV[4]
	local unionid = ARGV[5]

	-- 构建用户信息 key
	local user_info_key = 'user_info:' .. openid

	-- 检查用户是否存在
	local exists = redis.call('HEXISTS', user_info_key, 'openid')
	if exists == 1 then
		-- 用户存在，更新 googlePlatformKey 为当前 openid
		redis.call('SET', google_platform_key, openid)
		-- 返回用户信息
		local user_info = redis.call('HGETALL', user_info_key)
		return {1, user_info}
	else
		-- 用户不存在，创建新用户
		redis.call('SET', google_platform_key, openid)
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

	googlePlatformKey := fmt.Sprintf("%s:%s", redis_google_play, userId)

	keys := []string{googlePlatformKey}
	args := []interface{}{
		googleUserInfo.Openid,
		googleUserInfo.Avatar,
		googleUserInfo.Gender,
		googleUserInfo.Name,
		googleUserInfo.Unionid,
	}

	result, err := common_redis.GetRedis().Eval(ctx, luaScript, keys, args).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[GOOGLE-LOGIN-LUA-ERROR] eval lua script err: %v", err)
		resp.Code = common.ErrorCode_FAILED
		resp.Msg = "lua script execution failed"
		return resp, err
	}

	// 解析Lua脚本返回结果
	resultArray, ok := result.([]interface{})
	if !ok {
		klog.CtxErrorf(ctx, "[GOOGLE-LOGIN-LUA-ERROR] invalid lua result format")
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

	// 确保所有字段都有值
	avatar := userInfo["avatar"]
	gender := userInfo["gender"]
	name := userInfo["name"]
	openid := userInfo["openid"]
	unionid := userInfo["unionid"]

	// 设置用户信息到响应中
	resp.TapInfo = &user_center.TapInfo{
		Avatar:  avatar,
		Gender:  gender,
		Name:    name,
		Openid:  openid,
		Unionid: unionid,
	}

	return resp, nil
}

// verifyGoogleToken 验证 Google token 并获取用户信息
func verifyGoogleToken(ctx context.Context, token string) (*user_center.TapInfo, error) {
	// 从配置中获取 Google 客户端 ID
	// 实际实现中应该从配置文件或环境变量中获取
	googleClientID := common_config.Get("google.client_id").(string)
	if googleClientID == "" {
		// 如果没有配置客户端 ID，使用默认值（仅用于测试）
		googleClientID = "YOUR_GOOGLE_CLIENT_ID"
	}

	// 验证 Google ID token
	payload, err := idtoken.Validate(ctx, token, googleClientID)
	if err != nil {
		klog.Errorf("[GOOGLE-TOKEN-VALIDATE-ERROR] validate google token err: %v", err)
		return nil, err
	}

	// 提取用户信息
	userID := payload.Subject
	name := ""
	avatar := ""

	// 从 claims 中提取姓名
	if nameVal, ok := payload.Claims["name"]; ok {
		if nameStr, ok := nameVal.(string); ok {
			name = nameStr
		}
	}

	// 从 claims 中提取头像
	if pictureVal, ok := payload.Claims["picture"]; ok {
		if pictureStr, ok := pictureVal.(string); ok {
			avatar = pictureStr
		}
	}

	// 构建并返回 TapInfo，确保所有字段都有值
	return &user_center.TapInfo{
		Avatar:  avatar,
		Gender:  "", // Google 登录不提供性别信息，设置为空字符串
		Name:    name,
		Openid:  userID,
		Unionid: "", // Google 登录不提供 unionid 信息，设置为空字符串
	}, nil
}
