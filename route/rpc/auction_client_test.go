package rpc

import (
	"context"
	"os"
	"testing"

	"route_module/kitex_gen/auction"

	"github.com/golang/protobuf/proto"
	any1 "github.com/golang/protobuf/ptypes/any"
)

func Test_CallRPC(t *testing.T) {
	err := os.Chdir("..")
	if err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}

	tests := []struct {
		name      string
		rpcName   string
		bodyAny   *any1.Any
		wantErr   bool
		wantValue bool
	}{
		{
			name:    "unknown method",
			rpcName: "UnknownMethod",
			bodyAny: func() *any1.Any {
				pingReq := &auction.PingReq{Message: "123"}
				bytes, _ := proto.Marshal(pingReq)
				return &any1.Any{Value: bytes}
			}(),
			wantErr:   true,
			wantValue: false,
		},
		{
			name:    "empty method name",
			rpcName: "",
			bodyAny: func() *any1.Any {
				pingReq := &auction.PingReq{Message: "123"}
				bytes, _ := proto.Marshal(pingReq)
				return &any1.Any{Value: bytes}
			}(),
			wantErr:   true,
			wantValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb, err := GetClient("auction")
			if err != nil {
				t.Fatalf("GetClient failed: %v", err)
			}
			got, got2 := cb(context.Background(), tt.rpcName, tt.bodyAny)
			if (got != nil) != tt.wantErr {
				t.Errorf("cb() error = %v, wantErr %v", got, tt.wantErr)
			}
			if tt.wantValue && got2 == nil {
				t.Errorf("cb() got2 = nil, want non-nil")
			}
		})
	}
}
