package match

import (
	"sort"
)

// MatchTree 表示匹配树结构
type MatchTree struct {
	TreeNode *TreeNode
}

// NewMatchTree 创建一个新的匹配树
func NewMatchTree() *MatchTree {
	return &MatchTree{
		TreeNode: NewTreeNode(),
	}
}

// GetNode 获取根节点
func (mt *MatchTree) GetNode() *TreeNode {
	return mt.TreeNode
}

// BuildMatchTree 构建匹配树
func (mt *MatchTree) BuildMatchTree(num int) {
	partitions := [][]int{}

	// 深度拷贝切片
	deepCopy := func(src []int) []int {
		result := make([]int, len(src))
		copy(result, src)
		return result
	}

	// 递归生成所有分区组合
	var partition func(n, maxNum int, vec []int)
	partition = func(n, maxNum int, vec []int) {
		if n == 0 {
			sortedVec := deepCopy(vec)
			sort.Ints(sortedVec)
			partitions = append(partitions, sortedVec)
			return
		}
		if maxNum == 0 {
			partitions = append(partitions, deepCopy(vec))
			return
		}
		for i := maxNum; i >= 1; i-- {
			vec1 := deepCopy(vec)
			vec1 = append(vec1, i)
			partition(n-i, min(i, n-i), vec1)
		}
	}

	// 反转切片指定范围
	reverse := func(vec []int, from, to int) {
		for from < to {
			vec[from], vec[to] = vec[to], vec[from]
			from++
			to--
		}
	}

	// 字典序排列
	dicSort := func(vec []int) bool {
		length := len(vec)
		i := length - 2 // Go索引从0开始，对应Lua的#vec-2
		j := length - 1 // Go索引从0开始，对应Lua的#vec-1
		var m int

		// 找到第一个升序位置
		for i >= 0 {
			if vec[i+1] > vec[i] {
				break
			}
			i--
		}
		if i < 0 {
			return false
		}
		m = i
		i++

		// 找到第一个大于vec[m]的元素
		for j > i {
			if vec[j] > vec[m] {
				break
			}
			j--
		}

		// 交换并反转
		vec[j], vec[m] = vec[m], vec[j]
		reverse(vec, m+1, length-1)
		return true
	}

	// 生成所有分区
	partition(num, num, []int{})

	// 将所有分区及其排列添加到树中
	for _, partition := range partitions {
		// 处理原始分区
		node := mt.TreeNode
		for _, val := range partition {
			childNode := node.GetChildNode(val)
			if childNode != nil {
				node = childNode
			} else {
				node = node.AddChildNode(val)
			}
		}

		// 处理所有字典序排列
		for {
			if !dicSort(partition) {
				break
			}
			node1 := mt.TreeNode
			for _, val := range partition {
				childNode := node1.GetChildNode(val)
				if childNode != nil {
					node1 = childNode
				} else {
					node1 = node1.AddChildNode(val)
				}
			}
		}
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
