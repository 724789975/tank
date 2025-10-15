package test

import (
	"gate_way_module/kitex_gen/gate_way"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"
)

func TestExample(t *testing.T) {

	lgp := &gate_way.LoginRequest{
		Id: "mzw0536knQSO+bhbdL6dtw==",
	}

	any := &anypb.Any{}
	err := any.MarshalFrom(lgp)
	if err != nil {
		t.Errorf("marshal %s failed, err: %v", lgp.String(), err)
		return
	}

}
