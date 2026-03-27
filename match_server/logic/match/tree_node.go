package match

import "fmt"

// TreeNode 表示树中的一个节点
type TreeNode struct {
	Data       interface{} // 节点数据
	FatherNode *TreeNode   // 父节点
	Nodes      []*TreeNode // 子节点列表
}

// NewTreeNode 创建一个新的树节点
func NewTreeNode() *TreeNode {
	return &TreeNode{
		Data:       0,
		FatherNode: nil,
		Nodes:      []*TreeNode{},
	}
}

// GetChildNode 根据数据查找子节点
func (n *TreeNode) GetChildNode(key interface{}) *TreeNode {
	for _, node := range n.Nodes {
		if node.Data == key {
			return node
		}
	}
	return nil
}

// GetData 获取节点数据
func (n *TreeNode) GetData() interface{} {
	return n.Data
}

// AddChildNode 添加子节点
func (n *TreeNode) AddChildNode(data interface{}) *TreeNode {
	if n.GetChildNode(data) == nil {
		childNode := NewTreeNode()
		childNode.Data = data
		childNode.FatherNode = n
		n.Nodes = append(n.Nodes, childNode)
		return childNode
	}
	return nil
}

// RemoveChildNode 移除子节点
func (n *TreeNode) RemoveChildNode(key interface{}) bool {
	for i, node := range n.Nodes {
		if node.Data == key {
			// 从切片中移除元素
			n.Nodes = append(n.Nodes[:i], n.Nodes[i+1:]...)
			return true
		}
	}
	return false
}

// IsLeafNode 判断是否为叶子节点
func (n *TreeNode) IsLeafNode() bool {
	return len(n.Nodes) == 0
}

// IsRootNode 判断是否为根节点
func (n *TreeNode) IsRootNode() bool {
	return n.FatherNode == nil
}

// DumpNodes 打印节点信息
func (n *TreeNode) DumpNodes() {
	n.dumpNodesRecursive(0, []bool{})
}

// dumpNodesRecursive 递归打印节点信息
func (n *TreeNode) dumpNodesRecursive(level int, isLast []bool) {
	// 打印当前节点
	for i := 0; i < level; i++ {
		if i == level-1 {
			if isLast[i] {
				print("`-- ")
			} else {
				print("|-- ")
			}
		} else {
			if isLast[i] {
				print("    ")
			} else {
				print("|   ")
			}
		}
	}
	fmt.Printf("%v\n", n.Data)

	// 递归打印子节点
	for i, node := range n.Nodes {
		newIsLast := make([]bool, level+1)
		copy(newIsLast, isLast)
		newIsLast[level] = (i == len(n.Nodes)-1)
		node.dumpNodesRecursive(level+1, newIsLast)
	}
}
