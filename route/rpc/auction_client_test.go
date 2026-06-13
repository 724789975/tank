package rpc

import (
	"context"
	"os"
	"testing"

	common_config "route_module/config"
	"route_module/kitex_gen/auction"

	any1 "github.com/golang/protobuf/ptypes/any"
)

func Test_auctionClientWrapper(t *testing.T) {
	// Prepare test request bytes
	err := os.Chdir("..")
	if err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}
	pingReq := &auction.PingReq{Message: "123"}
	common_config.LoadConfig()
	InitAuctionClient()
	tests := []struct {
		name      string
		rpc_name  string
		body_any  *any1.Any
		wantErr   bool
		wantValue bool
	}{
		{
			name:     "Ping - valid request",
			rpc_name: "Ping",
			body_any: func() *any1.Any {
				a := &any1.Any{}
				err := a.MarshalFrom(pingReq)
				if err != nil {
					panic(err)
				}
				return a
			}(),
			wantErr:   false,
			wantValue: true,
		},
		{
			name:     "Ping - empty body",
			rpc_name: "Ping",
			body_any: func() *any1.Any {
				a := &any1.Any{}
				pingReq2 := &auction.PingReq{}
				err := a.MarshalFrom(pingReq2)
				if err != nil {
					panic(err)
				}
				return a
			}(),
			wantErr:   false,
			wantValue: true,
		},
		{
			name:     "unknown method",
			rpc_name: "UnknownMethod",
			body_any: func() *any1.Any {
				a := &any1.Any{}
				err := a.MarshalFrom(pingReq)
				if err != nil {
					panic(err)
				}
				return a
			}(),
			wantErr:   true,
			wantValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := auctionClientWrapper(context.Background(), tt.rpc_name, tt.body_any)
			if (got != nil) != tt.wantErr {
				t.Errorf("auctionClientWrapper() error = %v, wantErr %v", got, tt.wantErr)
			}
			if tt.wantValue && got2 == nil {
				t.Errorf("auctionClientWrapper() got2 = nil, want non-nil")
			}
		})
	}
}
