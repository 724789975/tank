package match

import "testing"

func TestMatchTree_BuildMatchTree(t *testing.T) {
	type args struct {
		num int
	}
	tests := []struct {
		name string
		mt   *MatchTree
		args args
	}{
		// TODO: Add test cases.
		{
			name: "BuildMatchTree",
			mt:   NewMatchTree(),
			args: args{num: 10},
		},
		{
			name: "BuildMatchTree_8",
			mt:   NewMatchTree(),
			args: args{num: 8},
		},
		{
			name: "BuildMatchTree_5",
			mt:   NewMatchTree(),
			args: args{num: 5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mt.BuildMatchTree(tt.args.num)
			tt.mt.GetNode().DumpNodes()
		})
	}
}
