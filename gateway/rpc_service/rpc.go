package rpcservice

import (
	"context"
	"fmt"
	"gateway/constant"

	"devops.xfein.com/codeup/75560f9d-2fab-4efe-bbe6-90b71f3ff9e4/xf-x/backend/common"
	"devops.xfein.com/codeup/75560f9d-2fab-4efe-bbe6-90b71f3ff9e4/xf-x/backend/idl/kitex_gen/gateway"
	"devops.xfein.com/codeup/75560f9d-2fab-4efe-bbe6-90b71f3ff9e4/xf-x/backend/idl/kitex_gen/gateway_service/gatewayservice"
	"github.com/cloudwego/kitex/pkg/serviceinfo"
)

type GatewayService struct {
	*serviceinfo.ServiceInfo
}

var service *GatewayService
func InitGatewayService() {
	service     = &GatewayService{
		ServiceInfo: gatewayservice.NewServiceInfo(),
	}
	ser         := common.NewKitexServer()
	gatewayservice.RegisterService(ser, service)

	go ser.Run()
}

func (x *GatewayService) UserMsg(ctx context.Context, req *gateway.UserMsgReq) (resp *gateway.UserMsgResp, err error) {

	common.GetNatsConn().Publish(fmt.Sprintf(constant.UserMsg, req.Id), []byte(""))
	resp = &gateway.UserMsgResp{
		Id  : req.Id,
		Code: 0,
	}
	return resp, nil
}
