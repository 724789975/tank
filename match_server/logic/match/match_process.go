package match

import (
	"sort"
	"sync"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
)

type MatchProcess struct {
	matchGroups map[uint32]*MatchGroup
	tree        []*MatchTree

	opchan chan func()
}

var (
	matchProcessInstance *MatchProcess
	once                 sync.Once
)

func GetMatchProcess() *MatchProcess {
	once.Do(func() {
		matchProcessInstance = &MatchProcess{
			matchGroups: make(map[uint32]*MatchGroup),
			tree:        make([]*MatchTree, 0, MAX_TEAM_COUNT),
			opchan:      make(chan func(), 10),
		}

		// 初始化匹配树
		for i := 1; i <= MAX_TEAM_COUNT; i++ {
			mt := NewMatchTree()
			mt.BuildMatchTree(i)
			matchProcessInstance.tree = append(matchProcessInstance.tree, mt)
		}

		go func() {
			for {
				matchProcessInstance.update()
			}
		}()
	})
	return matchProcessInstance
}

func (mp *MatchProcess) update() {
	select {
	case op := <-mp.opchan:
		op()
	case <-time.After(time.Second):
		// 将matchGroups中所有元素放入数组中
		groups := make([]*MatchGroup, 0, len(mp.matchGroups))
		for _, mg := range mp.matchGroups {
			mg.Update(time.Second.Seconds())
			groups = append(groups, mg)
		}

		// 按照 weight 升序，CreateTime 降序排列
		sort.Slice(groups, func(i, j int) bool {
			if groups[i].Weights != groups[j].Weights {
				return groups[i].Weights < groups[j].Weights
			}
			return groups[i].CreateTime > groups[j].CreateTime
		})

		for {
			mr, mb, r := mp.match(MAX_TEAM_COUNT, MAX_TEAM_COUNT, 0, groups)
			if r {
				go func(r, b []uint32) {
					// 在这里处理匹配成功的情况
					klog.Infof("match success, r: %v, b: %v", mr, mb)
				}(mr, mb)
				// 从groups中删除 mr 和 mb 对应的匹配组
				newGroups := make([]*MatchGroup, 0)
				matchedIDs := make(map[uint32]bool)
				for _, id := range mr {
					matchedIDs[id] = true
				}
				for _, id := range mb {
					matchedIDs[id] = true
				}
				for _, mg := range groups {
					if !matchedIDs[mg.ID] {
						newGroups = append(newGroups, mg)
					}
				}
				groups = newGroups
			}else{
				break
			}
		}

	}
}

// match 匹配玩家
// 输入：r 希望的红色方玩家数量，b 希望的蓝色方玩家数量，level 匹配等级，groups 匹配组列表
// 输出：true 匹配成功，false 匹配失败 mr 红色方玩家列表 mb 蓝色方玩家列表
func (mp *MatchProcess) match(r, b uint32, level uint32, groups []*MatchGroup) (mr, mb []uint32, ok bool) {
	// 检查是否有匹配组
	if len(groups) == 0 {
		return nil, nil, false
	}
	if r > 0{
		rtree := mp.tree[r - 1]
		mr, ok = mp.match_by_tree(rtree.TreeNode, level, groups, []uint32{})
		if !ok {
			return nil, nil, false
		}
	}
	if b > 0{
		btree := mp.tree[b - 1]
		mb, ok = mp.match_by_tree(btree.TreeNode, level, groups, mr)
		if !ok {
			return nil, nil, false
		}
	}
	return mr, mb, ok
}

func (mp *MatchProcess) match_by_tree(node *TreeNode, level uint32, groups []*MatchGroup, exclude []uint32) (t []uint32, ok bool) {
	// 检查是否有匹配组
	if len(groups) == 0 {
		return nil, false
	}

	retl := make([]*MatchGroup, 0, len(groups))
	checkExclude := func(id uint32) bool {
		for _, e := range exclude {
			if e == id {
				return false
			}
		}
		return true
	}
	checkLevel := func(mg *MatchGroup) bool {
		if !mg.CheckLevel(level) {
			return false
		}
		for _, e := range retl {
			if !e.Matching(mg) {
				return false
			}
		}
		return true
	}
	for _, mg := range groups {
		if checkExclude(mg.ID) {
			if node1 := node.GetChildNode(mg.Count); node1 != nil {
				if checkLevel(mg) {
					t = append(t, mg.ID)
					retl = append(retl, mg)
					if node1.IsLeafNode() {
						return t, true
					}
					node = node1
				}
			} else {
				continue
			}

		}
	}
	return nil, false
}

func (mp *MatchProcess) AddMatch(id, level uint32, count int) bool {
	mg := NewMatchGroup()
	mg.Initialize(id, level, count)

	ch := make(chan bool)
	mp.opchan <- func() {
		if _, ok := mp.matchGroups[id]; ok {
			// 匹配组已存在
			klog.Errorf("match group %d already exists", id)
			ch <- false
			return
		}
		mp.matchGroups[id] = mg
		ch <- true
	}
	return <-ch
}

func (mp *MatchProcess) CancelMatch(id uint32) bool {
	ch := make(chan bool)
	mp.opchan <- func() {
		if _, ok := mp.matchGroups[id]; !ok {
			// 匹配组不存在
			klog.Errorf("match group %d not found", id)
			ch <- false
			return
		}
		delete(mp.matchGroups, id)
		ch <- true
	}
	return <-ch
}

func (mp *MatchProcess) Match(r, b uint32, level uint32) (mr, mb []uint32, ok bool) {
	ch := make(chan bool)
	mp.opchan <- func() {
		defer func() { ch <- true }()
		groups := make([]*MatchGroup, 0, len(mp.matchGroups))
		for _, mg := range mp.matchGroups {
			groups = append(groups, mg)
		}

		// 按照 weight 升序，CreateTime 降序排列
		sort.Slice(groups, func(i, j int) bool {
			if groups[i].Weights != groups[j].Weights {
				return groups[i].Weights > groups[j].Weights
			} else if groups[i].CreateTime != groups[j].CreateTime {
				return groups[i].CreateTime < groups[j].CreateTime
			}
			return groups[i].ID < groups[j].ID
		})
		mr, mb, ok = mp.match(r, b, level, groups)
	}
	<-ch
	return mr, mb, ok
}
