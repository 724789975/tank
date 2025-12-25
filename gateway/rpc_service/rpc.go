package rpcservice

import (
	"context"
	"fmt"
	common_config "gate_way_module/config"
	"gate_way_module/constant"
	"gate_way_module/etcd"
	"gate_way_module/kitex_gen/common"
	"gate_way_module/kitex_gen/gate_way"
	"gate_way_module/kitex_gen/gateway_service/gatewayservice"
	"gate_way_module/nats"
	"net"

	_nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/cloudwego/kitex/pkg/klog"
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
	resp = &gate_way.UserMsgResp{
		Id:   req.Id,
		Code: common.ErrorCode_OK,
	}
	m := _nats.NewMsg(fmt.Sprintf(constant.UserMsg, req.Id))
	m.Data = ncMsgb

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	for k, v := range carrier {
		m.Header.Set(k, v)
	}

	if err = nats.GetNatsConn().PublishMsg(m); err != nil {
		klog.CtxErrorf(ctx, "[GATEWAY-RPC-PUBLISH] publish %s failed %s", fmt.Sprintf(constant.UserMsg, req.Id), err.Error())
		resp.Code = common.ErrorCode_FAILED
		return resp, err
	}
	klog.CtxInfof(ctx, "[GATEWAY-RPC-PUBLISH] publish %s success", fmt.Sprintf(constant.UserMsg, req.Id))
	return resp, nil
}
