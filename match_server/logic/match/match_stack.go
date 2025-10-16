package match

// MatchStack 匹配栈，用于存储匹配过程中的匹配组
// 在匹配算法中用于临时存储已选择的匹配组，并提供匹配检查功能
type MatchStack struct {
	matchLevel uint32              // 匹配等级要求，如果为0则不限制等级
	stack      []*MatchGroup       // 存储匹配组的切片，替代C++中的vector
	size       uint32              // 当前栈中所有匹配组的总人数
}

// NewMatchStack 创建一个新的匹配栈实例
// 参数: matchLevel - 匹配等级要求，如果为0则不限制等级
// 返回: 初始化好的MatchStack指针
func NewMatchStack(matchLevel uint32) *MatchStack {
	return &MatchStack{
		matchLevel: matchLevel,
		stack:      make([]*MatchGroup, 0),
		size:       0,
	}
}

// Matching 检查指定的匹配组是否可以与当前栈中的匹配组进行匹配
// 参数: matchingGroup - 要检查的匹配组
// 返回: bool - true表示可以匹配，false表示不可以匹配
func (ms *MatchStack) Matching(matchingGroup *MatchGroup) bool {
	// 如果设置了匹配等级要求，检查是否在等级范围内
	if ms.matchLevel > 0 {
		if ms.matchLevel > matchingGroup.MaxLevel || ms.matchLevel < matchingGroup.MinLevel {
			return false
		}
	}

	// 检查是否与栈中已有匹配组冲突
	for _, group := range ms.stack {
		// 同一ID的匹配组不能重复添加
		if matchingGroup.ID == group.ID {
			return false
		}
		// 检查等级范围是否重叠
		if group.MinLevel > matchingGroup.MaxLevel || group.MaxLevel < matchingGroup.MinLevel {
			return false
		}
	}

	return true
}

// IsEmpty 检查栈是否为空
// 返回: bool - true表示栈为空，false表示栈不为空
func (ms *MatchStack) IsEmpty() bool {
	return len(ms.stack) == 0
}

// Top 获取栈顶的匹配组
// 返回: *MatchGroup - 栈顶的匹配组指针，如果栈为空则返回nil
func (ms *MatchStack) Top() *MatchGroup {
	if ms.IsEmpty() {
		return nil
	}
	return ms.stack[len(ms.stack)-1]
}

// Push 将匹配组压入栈顶
// 参数: matchGroup - 要压入的匹配组指针
func (ms *MatchStack) Push(matchGroup *MatchGroup) {
	ms.stack = append(ms.stack, matchGroup)
	ms.size += matchGroup.Count
}

// Pop 弹出栈顶的匹配组
func (ms *MatchStack) Pop() {
	if !ms.IsEmpty() {
		ms.size -= ms.stack[len(ms.stack)-1].Count
		ms.stack = ms.stack[:len(ms.stack)-1]
	}
}

// Size 获取栈中匹配组的数量
// 返回: uint32 - 栈中匹配组的数量
func (ms *MatchStack) Size() uint32 {
	return uint32(len(ms.stack))
}

// Clear 清空栈
func (ms *MatchStack) Clear() {
	ms.stack = ms.stack[:0]
	ms.size = 0
}

// GetSize 获取栈中所有匹配组的总人数
// 返回: uint32 - 总人数
func (ms *MatchStack) GetSize() uint32 {
	return ms.size
}