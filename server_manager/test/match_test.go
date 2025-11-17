package test

import (
	// "match_server/logic/match"
	"testing"
)

// TestMatch 测试匹配系统
func TestMatch(t *testing.T) {
	// // 创建并初始化匹配服务
	// matchService := match.NewMatchService()
	// matchService.Initialize()
	// // 启动匹配服务
	// matchService.Start()

	// // 添加测试匹配请求
	// matchService.AddMatching(1, 4, 1)
	// matchService.AddMatching(2, 2, 5)
	// matchService.AddMatching(3, 1, 9)
	// matchService.AddMatching(4, 1, 1)
	// matchService.AddMatching(5, 3, 1)
	// matchService.AddMatching(6, 5, 2)
	// matchService.AddMatching(7, 1, 3)
	// matchService.AddMatching(8, 1, 5)
	// matchService.AddMatching(9, 1, 7)
	// matchService.AddMatching(31, 1, 2)
	// matchService.AddMatching(32, 1, 2)
	// matchService.AddMatching(33, 1, 3)

	// // 尝试执行匹配
	// fmt.Println("开始匹配测试...")
	// matchingClients, success := matchService.Matching(2, 5, 2)
	// if success {
	// 	fmt.Println("匹配成功!")
	// 	fmt.Print("红队: ")
	// 	for i := 0; i < match.TREE_COUNT; i++ {
	// 		if matchingClients[i] > 0 {
	// 			fmt.Printf("%d ", matchingClients[i])
	// 		}
	// 	}
	// 	fmt.Println()

	// 	fmt.Print("蓝队: ")
	// 	for i := match.TREE_COUNT; i < match.TREE_COUNT*2; i++ {
	// 		if matchingClients[i] > 0 {
	// 			fmt.Printf("%d ", matchingClients[i])
	// 		}
	// 	}
	// 	fmt.Println()
	// } else {
	// 	fmt.Println("匹配失败")
	// }

	// // 持续更新一段时间
	// fmt.Println("持续更新5秒...")
	// for i := 0; i < 30; i++ {
	// 	matchService.Update()
	// 	time.Sleep(time.Second)
	// }
}
