package match

import (
	"fmt"
)

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
	if n.IsLeafNode() {
		str := fmt.Sprintf("%v\t", n.Data)
		nodeF := n.FatherNode
		for nodeF != nil {
			if nodeF.FatherNode != nil {
				str = fmt.Sprintf("%v%v\t", str, nodeF.Data)
			} else {
				str = fmt.Sprintf("%v\n----------------------------\n", str)
			}
			nodeF = nodeF.FatherNode
		}
		// 这里假设DEBUG和match_log会在后续实现
		// 暂时使用fmt.Println
		fmt.Println(str)
	} else {
		for _, node := range n.Nodes {
			node.DumpNodes()
		}
	}
}