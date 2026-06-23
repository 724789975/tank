package manager

import (
	"context"
	"fmt"
	"sync"

	"homepage_server/kitex_gen/common"
	"homepage_server/kitex_gen/homepage"
	"homepage_server/redis"
)

type HomepageManager struct {
}

var (
	homepage_mgr        HomepageManager
	once_homepage_mgr   sync.Once
	redis_role_info_key = "role_info"
)

func GetHomepageManager() *HomepageManager {
	once_homepage_mgr.Do(func() {
		homepage_mgr = HomepageManager{}
	})
	return &homepage_mgr
}

func (x *HomepageManager) GetRoleInfo(ctx context.Context, req *homepage.GetRoleInfoReq) (resp *homepage.GetRoleInfoRsp, err error) {
	resp = &homepage.GetRoleInfoRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}

	userId := ctx.Value("userId").(string)

	experience := redis.GetRedis().Get(ctx, fmt.Sprintf("%s:%s:experience", redis_role_info_key, userId)).Val()
	if experience == "" {
		experience = "0"
	}

	expVal := int64(0)
	fmt.Sscanf(experience, "%d", &expVal)

	level := int32(expVal/100) + 1

	resp.Data = &homepage.GetRoleInfoRsp_Data{
		RoleInfo: &homepage.RoleInfo{
			Level:      level,
			Experience: expVal,
		},
	}

	return resp, nil
}

func (x *HomepageManager) UpdateRoleExp(ctx context.Context, req *homepage.UpdateRoleExpReq) (resp *homepage.UpdateRoleExpRsp, err error) {
	resp = &homepage.UpdateRoleExpRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}

	return resp, nil
}
