package match

import (
	"sort"
	"testing"
	"time"
)

// 创建测试用的匹配组
func createTestMatchGroup(id, level uint32, count int) *MatchGroup {
	mg := NewMatchGroup()
	mg.Initialize(id, level, count)
	return mg
}

// 创建测试用的匹配组并设置特定的创建时间
func createTestMatchGroupWithTime(id, level uint32, count int, createTime int64) *MatchGroup {
	mg := createTestMatchGroup(id, level, count)
	mg.CreateTime = createTime
	return mg
}

// 比较两个uint32切片是否相等（忽略顺序）
func slicesEqualIgnoreOrder(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
	sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMatchProcess_Match(t *testing.T) {
	// 确保匹配树已经初始化
	mp := GetMatchProcess()

	// 创建一个辅助函数来清理测试数据
	cleanupTestData := func() {
		ch := make(chan bool)
		mp.opchan <- func() {
			mp.matchGroups = make(map[uint32]*MatchGroup)
			ch <- true
		}
		<-ch
	}

	// 首先清理可能存在的匹配组
	cleanupTestData()

	// 设置当前时间基准，用于测试CreateTime的影响
	now := time.Now().Unix()

	tests := []struct {
		name     string
		r        uint32
		b        uint32
		level    uint32
		groups   []*MatchGroup
		want     []uint32
		want2    []uint32
		want3    bool
		describe string
	}{
		{
			name:   "CountMatchTest",
			r:      4,
			b:      4,
			level:  1,
			groups: []*MatchGroup{
				createTestMatchGroup(1, 1, 2), // 2人组
				createTestMatchGroup(2, 1, 1), // 1人组
				createTestMatchGroup(3, 1, 3), // 3人组
				createTestMatchGroup(4, 1, 1), // 1人组
				createTestMatchGroup(5, 1, 1), // 1人组
			},
			want:   []uint32{3, 2},
			want2:  []uint32{1, 5, 4}, // 2+1+1=4，但我们需要3，所以应该选2+1
			want3:  true,
			describe: "测试不同Count值的匹配组组合",
		},
		{
			name:   "SuccessfulMatch_MultipleGroups",
			r:      2,
			b:      2,
			level:  1,
			groups: []*MatchGroup{
				createTestMatchGroupWithTime(1, 1, 1, now),
				createTestMatchGroupWithTime(2, 1, 1, now-1),
				createTestMatchGroupWithTime(3, 1, 1, now-2),
				createTestMatchGroupWithTime(4, 1, 1, now-3),
			},
			want:   []uint32{3, 4},
			want2:  []uint32{1, 2},
			want3:  true,
			describe: "四个1人匹配组，应该成功匹配为2v2",
		},
		{
			name:   "LevelMismatch",
			r:      1,
			b:      1,
			level:  1,
			groups: []*MatchGroup{
				createTestMatchGroup(1, 1, 1),
				createTestMatchGroup(2, 100, 1), // 等级差距过大
			},
			want:   []uint32{},
			want2:  []uint32{},
			want3:  false,
			describe: "等级差距过大的匹配组，应该匹配失败",
		},
		{
			name:     "EmptyGroups",
			r:        0,
			b:        0,
			level:    1,
			groups:   []*MatchGroup{},
			want:     []uint32{},
			want2:    []uint32{},
			want3:    false,
			describe: "空匹配组列表，应该匹配失败",
		},
		{
			name:   "SuccessfulMatch_1v1",
			r:      1,
			b:      1,
			level:  1,
			groups: []*MatchGroup{createTestMatchGroup(1, 1, 1), createTestMatchGroup(2, 1, 1)},
			want:   []uint32{1},
			want2:  []uint32{2},
			want3:  true,
			describe: "两个1人匹配组，应该成功匹配为1v1",
		},
		{
			name:   "SuccessfulMatch_2v2",
			r:      2,
			b:      2,
			level:  1,
			groups: []*MatchGroup{createTestMatchGroup(1, 1, 2), createTestMatchGroup(2, 1, 2)},
			want:   []uint32{1},
			want2:  []uint32{2},
			want3:  true,
			describe: "两个2人匹配组，应该成功匹配为2v2",
		},
		{
			name:   "InsufficientGroups",
			r:      1,
			b:      1,
			level:  1,
			groups: []*MatchGroup{createTestMatchGroup(1, 1, 1)},
			want:   []uint32{},
			want2:  []uint32{},
			want3:  false,
			describe: "只有一个匹配组，应该匹配失败",
		},
		{
			name:   "SuccessfulMatch_WithExistingPlayers",
			r:      3,
			b:      1,
			level:  1,
			groups: []*MatchGroup{
				createTestMatchGroup(1, 1, 1),
				createTestMatchGroup(2, 1, 1),
				createTestMatchGroup(3, 1, 1),
				createTestMatchGroup(4, 1, 1),
			},
			want:   []uint32{1, 2, 3},
			want2:  []uint32{4},
			want3:  true,
			describe: "已有部分玩家，应该成功匹配剩余玩家",
		},
		{
			name:   "SuccessfulMatch_MixedGroupSizes",
			r:      2,
			b:      2,
			level:  1,
			groups: []*MatchGroup{
				createTestMatchGroup(1, 1, 2), // 2人组
				createTestMatchGroup(2, 1, 1), // 1人组
				createTestMatchGroup(3, 1, 1), // 1人组
			},
			want:   []uint32{1},
			want2:  []uint32{2, 3},
			want3:  true,
			describe: "混合大小的匹配组，应该成功匹配为2v2",
		},
		{
			name:   "CreateTimePriority",
			r:      2,
			b:      2,
			level:  1,
			groups: []*MatchGroup{
				createTestMatchGroupWithTime(1, 1, 1, now-10), // 更早创建
				createTestMatchGroupWithTime(2, 1, 1, now-5),  // 中间创建
				createTestMatchGroupWithTime(3, 1, 1, now-8),  // 较早创建
				createTestMatchGroupWithTime(4, 1, 1, now-2),  // 较晚创建
			},
			want:   []uint32{1, 3}, // 期望创建时间早的组优先匹配
			want2:  []uint32{2, 4},
			want3:  true,
			describe: "创建时间早的匹配组应该优先匹配",
		},
		{
			name:   "WeightsAndCreateTimeInteraction",
			r:      2,
			b:      2,
			level:  1,
			groups: []*MatchGroup{
				// Weights不同时，Weights小的优先
				func() *MatchGroup { g := createTestMatchGroupWithTime(1, 1, 1, now-10); g.Weights = 1; return g }(),
				func() *MatchGroup { g := createTestMatchGroupWithTime(2, 1, 1, now-5); g.Weights = 2; return g }(),
				// Weights相同时，CreateTime早的优先
				func() *MatchGroup { g := createTestMatchGroupWithTime(3, 1, 1, now-8); g.Weights = 1; return g }(),
				func() *MatchGroup { g := createTestMatchGroupWithTime(4, 1, 1, now-2); g.Weights = 2; return g }(),
			},
			want:   []uint32{2, 4}, // Weights=1的两个组优先匹配
			want2:  []uint32{1, 3},
			want3:  true,
			describe: "测试Weights和CreateTime的交互影响",
		},
		{
			name:   "LevelExtensionTest",
			r:      1,
			b:      1,
			level:  8,
			groups: []*MatchGroup{
				func() *MatchGroup { 
					g := createTestMatchGroup(1, 1, 1)
					// 模拟扩展后的等级范围
					g.MinLevel = 1
					g.MaxLevel = 10
					return g 
				}(),
				createTestMatchGroup(2, 8, 1),
			},
			want:   []uint32{1},
			want2:  []uint32{2},
			want3:  true,
			describe: "测试等级扩展后的匹配逻辑",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 确保每次测试前清理数据
			cleanupTestData()

			// 将测试用的匹配组添加到matchGroups中
			ch := make(chan bool)
			mp.opchan <- func() {
				for _, group := range tt.groups {
					mp.matchGroups[group.ID] = group
				}
				ch <- true
			}
			<-ch

			// 调用待测试的方法
			got, got2, got3 := mp.Match(tt.r, tt.b, tt.level)

			// 检查结果
			if !slicesEqualIgnoreOrder(got, tt.want) {
				t.Errorf("Match() got = %v, want %v - %s", got, tt.want, tt.describe)
			}
			if !slicesEqualIgnoreOrder(got2, tt.want2) {
				t.Errorf("Match() got2 = %v, want %v - %s", got2, tt.want2, tt.describe)
			}
			if got3 != tt.want3 {
				t.Errorf("Match() got3 = %v, want %v - %s", got3, tt.want3, tt.describe)
			}

			// 测试完成后清理数据
			cleanupTestData()
		})
	}
}

