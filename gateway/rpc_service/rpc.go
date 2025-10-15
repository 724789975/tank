package rpcservice

import (
	"context"
	"fmt"
	common_config "gate_way_module/config"
	"gate_way_module/constant"
	"gate_way_module/etcd"
	"gate_way_module/kitex_gen/gate_way"
	"gate_way_module/kitex_gen/gateway_service/gatewayservice"
	"gate_way_module/nats"
	"net"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/serviceinfo"
	kitexserver "github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"google.golang.org/protobuf/proto"
)

type GatewayService struct {
	*serviceinfo.ServiceInfo
}

var service *GatewayService

func InitGatewayService() {
	service = &GatewayService{
		ServiceInfo: gatewayservice.NewServiceInfo(),
	}

	NewKitexServer := func() kitexserver.Server {
		address, err := net.ResolveTCPAddr("tcp", common_config.Get("gateway_rpc.addr").(string))
		if err != nil {
			panic(err)
		}
		serviceName := common_config.Get("gateway_rpc.service_name").(string)

		server := kitexserver.NewServer(
			kitexserver.WithServiceAddr(address),
			kitexserver.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
				ServiceName: serviceName,
			}),
			kitexserver.WithSuite(tracing.NewServerSuite()),
			kitexserver.WithRegistry(etcd.GetEtcdClient()),
		)

		return server
	}

	ser := NewKitexServer()
	gatewayservice.RegisterService(ser, service)

	go ser.Run()
}

func (x *GatewayService) UserMsg(ctx context.Context, req *gate_way.UserMsgReq) (resp *gate_way.UserMsgResp, err error) {
	ncMsgb, err := proto.Marshal(req.Msg)
	if err != nil {
		return nil, err
	}
	nats.GetNatsConn().Publish(fmt.Sprintf(constant.UserMsg, req.Id), ncMsgb)
	resp = &gate_way.UserMsgResp{
		Id:   req.Id,
		Code: 0,
	}
	return resp, nil
}
