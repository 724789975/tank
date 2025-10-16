package match

import "time"

// ExtendFlag 扩展标志枚举
type ExtendFlag int

// ExtendFlagByteSet 扩展标志位集合
type ExtendFlagByteSet struct {
	Data [ExtendFlagCount/8 + 1]byte
}

// NewExtendFlagByteSet 创建新的扩展标志位集合
func NewExtendFlagByteSet() *ExtendFlagByteSet {
	return &ExtendFlagByteSet{}
}

// SetByte 设置特定位的值
func (bs *ExtendFlagByteSet) SetByte(flag ExtendFlag, value bool) bool {
	if flag >= ExtendFlag(ExtendFlagCount) {
		return false
	}
	index := flag / 8
	bit := flag % 8
	if value {
		bs.Data[index] |= (1 << bit)
	} else {
		bs.Data[index] &= ^(1 << bit)
	}
	return true
}

// GetByte 获取特定位的值
func (bs *ExtendFlagByteSet) GetByte(flag ExtendFlag) bool {
	if flag >= ExtendFlag(ExtendFlagCount) {
		return false
	}
	index := flag / 8
	bit := flag % 8
	return (bs.Data[index] & (1 << bit)) != 0
}

// Reset 重置所有位
func (bs *ExtendFlagByteSet) Reset() {
	for i := range bs.Data {
		bs.Data[i] = 0
	}
}

// MatchGroup 匹配组
type MatchGroup struct {
	ID           uint32
	Count        uint32
	MinLevel     uint32
	MaxLevel     uint32
	ExtendFlag   *ExtendFlagByteSet
	Weights      uint32
	UpweightTime float32
	ExtendTime   float32
	ExtendTimes  uint32
	CreateTime   int64
}

// NewMatchGroup 创建新的匹配组
func NewMatchGroup() *MatchGroup {
	return &MatchGroup{
		ExtendFlag: NewExtendFlagByteSet(),
	}
}

// Initialize 初始化匹配组
func (mg *MatchGroup) Initialize(id, level, count uint32) {
	mg.Reset()
	mg.ID = id
	mg.MinLevel = level
	mg.MaxLevel = level
	mg.Count = count
	mg.Weights = count
	mg.CreateTime = time.Now().Unix()
}

// Reset 重置匹配组
func (mg *MatchGroup) Reset() {
	mg.ExtendFlag.Reset()
	mg.Weights = 0
	mg.UpweightTime = 0.0
	mg.ExtendTime = 0.0
	mg.ExtendTimes = 0
}

// Update 更新匹配组状态
func (mg *MatchGroup) Update(deltaTime float64) {
	mg.UpweightTime += float32(deltaTime)
	if mg.UpweightTime >= 1 {
		mg.Weights++
		mg.UpweightTime = 0
	}

	mg.ExtendTime += float32(deltaTime)
	if mg.ExtendTime >= EXTENDTIME {
		if mg.Extend() {
			mg.ExtendTimes++
		}
		mg.ExtendTime = 0
	}
}

// Matching 判断是否可以与另一个匹配组匹配
func (mg *MatchGroup) Matching(other *MatchGroup) bool {
	// 当前组的等级范围完全小于另一组的最小等级
	if mg.MaxLevel < other.MinLevel {
		return false
	}
	// 当前组的等级范围完全大于另一组的最大等级
	if mg.MinLevel > other.MaxLevel {
		return false
	}
	// 两个等级范围有重叠
	return true
}

func (mg *MatchGroup) CheckLevel(level uint32) bool {
	return mg.MinLevel <= level && mg.MaxLevel >= level
}

// Extend 扩展等级范围
func (mg *MatchGroup) Extend() bool {
	// 已经达到最大扩展范围
	if mg.MinLevel <= MINLEVEL && mg.MaxLevel >= MAXLEVEL {
		return false
	}

	hasExtended := false
	// 向下扩展
	if mg.MinLevel > MINLEVEL {
		mg.MinLevel--
		mg.ExtendFlag.SetByte(ExtendFlag(mg.MinLevel-1), true)
		hasExtended = true
	}

	// 向上扩展
	if mg.MaxLevel < MAXLEVEL {
		mg.MaxLevel++
		mg.ExtendFlag.SetByte(ExtendFlag(mg.MaxLevel-1), true)
		hasExtended = true
	}

	return hasExtended
}

// MatchGroupCompareDes 用于按权重降序排列的比较器
func MatchGroupCompareDes(l, r *MatchGroup) bool {
	return l.Weights > r.Weights
}
