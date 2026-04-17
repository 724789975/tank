package manager

import (
	"context"
	"sync"
	"testing"
	"time"

	"auction_module/kitex_gen/auction"
	"auction_module/kitex_gen/common"
	"auction_module/redis"
)

// 性能测试用例: 并发创建出售订单
func TestAuctionManager_Performance_Sell(t *testing.T) {
	setupTest()

	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	manager := GetAuctionManager()
	userID := "test_user_performance"
	testCtx := context.WithValue(ctx, "userId", userID)

	// 并发数
	concurrency := 10
	// 每个用户创建的订单数
	ordersPerUser := 8

	var wg sync.WaitGroup
	errorChan := make(chan error, concurrency*ordersPerUser)
	startTime := time.Now()

	// 并发创建出售订单
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userIdx int) {
			defer wg.Done()
			for j := 0; j < ordersPerUser; j++ {
				itemID := "test_item_perf_" + string(rune(userIdx+'0')) + "_" + string(rune(j+'0'))
				req := &auction.SellReq{
					ItemId:       itemID,
					Quantity:     1,
					Price:        100,
					ItemInfo:     "Test Item",
					IdempotentId: "test_sell_perf_" + string(rune(userIdx+'0')) + "_" + string(rune(j+'0')) + "_" + time.Now().String(),
				}

				resp, err := manager.Sell(testCtx, req)
				if err != nil || resp.Code != common.ErrorCode_OK {
					errorChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	duration := time.Since(startTime)
	totalOrders := concurrency * ordersPerUser
	errors := 0
	for range errorChan {
		errors++
	}

	t.Logf("并发创建出售订单性能测试结果:")
	t.Logf("并发数: %d", concurrency)
	t.Logf("每个用户创建订单数: %d", ordersPerUser)
	t.Logf("总订单数: %d", totalOrders)
	t.Logf("总耗时: %v", duration)
	t.Logf("错误数: %d", errors)
	t.Logf("成功率: %.2f%%", float64(totalOrders-errors)/float64(totalOrders)*100)
	t.Logf("每秒处理订单数: %.2f", float64(totalOrders)/duration.Seconds())
}

// 性能测试用例: 并发创建购买订单
func TestAuctionManager_Performance_Buy(t *testing.T) {
	setupTest()

	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	manager := GetAuctionManager()
	userID := "test_user_performance"
	testCtx := context.WithValue(ctx, "userId", userID)

	// 先创建一些出售订单
	for i := 0; i < 10; i++ {
		itemID := "test_item_perf_buy_" + string(rune(i+'0'))
		req := &auction.SellReq{
			ItemId:       itemID,
			Quantity:     100,
			Price:        100,
			ItemInfo:     "Test Item",
			IdempotentId: "test_sell_for_buy_perf_" + string(rune(i+'0')) + "_" + time.Now().String(),
		}
		_, err := manager.Sell(testCtx, req)
		if err != nil {
			t.Skip("Failed to create sell orders for buy performance test:", err)
		}
	}

	// 并发数
	concurrency := 10
	// 每个用户创建的订单数
	ordersPerUser := 8

	var wg sync.WaitGroup
	errorChan := make(chan error, concurrency*ordersPerUser)
	startTime := time.Now()

	// 并发创建购买订单
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userIdx int) {
			defer wg.Done()
			for j := 0; j < ordersPerUser; j++ {
				itemID := "test_item_perf_buy_" + string(rune(j%10+'0'))
				req := &auction.BuyReq{
					ItemId:       itemID,
					Quantity:     1,
					Price:        100,
					IdempotentId: "test_buy_perf_" + string(rune(userIdx+'0')) + "_" + string(rune(j+'0')) + "_" + time.Now().String(),
				}

				resp, err := manager.Buy(testCtx, req)
				if err != nil || resp.Code != common.ErrorCode_OK {
					errorChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	duration := time.Since(startTime)
	totalOrders := concurrency * ordersPerUser
	errors := 0
	for range errorChan {
		errors++
	}

	t.Logf("并发创建购买订单性能测试结果:")
	t.Logf("并发数: %d", concurrency)
	t.Logf("每个用户创建订单数: %d", ordersPerUser)
	t.Logf("总订单数: %d", totalOrders)
	t.Logf("总耗时: %v", duration)
	t.Logf("错误数: %d", errors)
	t.Logf("成功率: %.2f%%", float64(totalOrders-errors)/float64(totalOrders)*100)
	t.Logf("每秒处理订单数: %.2f", float64(totalOrders)/duration.Seconds())
}

// 性能测试用例: 并发获取交易历史
// func TestAuctionManager_Performance_GetTransactionHistory(t *testing.T) {
// 	setupTest()
// 	defer teardownTest()

// 	// 检查Redis连接是否正常
// 	ctx := context.Background()
// 	_, err := redis.GetRedis().Ping(ctx).Result()
// 	if err != nil {
// 		t.Skip("Redis connection failed, skipping test:", err)
// 	}

// 	manager := GetAuctionManager()
// 	userID := "test_user_performance"
// 	testCtx := context.WithValue(ctx, "userId", userID)

// 	// 创建一些交易记录
// 	for i := 0; i < 10; i++ {
// 		itemID := "test_item_perf_history_" + string(rune(i+'0'))

// 		// 创建出售订单
// 		sellReq := &auction.SellReq{
// 			ItemId:       itemID,
// 			Quantity:     5,
// 			Price:        100,
// 			ItemInfo:     "Test Item",
// 			IdempotentId: "test_sell_for_history_perf_" + string(rune(i+'0')) + "_" + time.Now().String(),
// 		}
// 		_, err = manager.Sell(testCtx, sellReq)
// 		if err != nil {
// 			t.Skip("Failed to create sell orders for history performance test:", err)
// 		}

// 		// 创建购买订单
// 		buyReq := &auction.BuyReq{
// 			ItemId:       itemID,
// 			Quantity:     5,
// 			Price:        100,
// 			IdempotentId: "test_buy_for_history_perf_" + string(rune(i+'0')) + "_" + time.Now().String(),
// 		}
// 		_, err = manager.Buy(testCtx, buyReq)
// 		if err != nil {
// 			t.Skip("Failed to create buy orders for history performance test:", err)
// 		}

// 		// 等待交易完成
// 		time.Sleep(1 * time.Second)
// 	}

// 	// 并发数
// 	concurrency := 100
// 	// 每个用户查询次数
// 	queriesPerUser := 10

// 	var wg sync.WaitGroup
// 	errorChan := make(chan error, concurrency*queriesPerUser)
// 	startTime := time.Now()

// 	// 并发获取交易历史
// 	for i := 0; i < concurrency; i++ {
// 		wg.Add(1)
// 		go func(userIdx int) {
// 			defer wg.Done()
// 			for j := 0; j < queriesPerUser; j++ {
// 				// 获取交易历史
// 				getReq := &auction.GetTransactionHistoryReq{
// 					OrderId: "test_sell_for_history_perf_" + string(rune(j%10+'0')),
// 				}

// 				resp, err := manager.GetTransactionHistory(testCtx, getReq)
// 				if err != nil || resp.Code != common.ErrorCode_OK {
// 					errorChan <- err
// 				}
// 			}
// 		}(i)
// 	}

// 	wg.Wait()
// 	close(errorChan)

// 	duration := time.Since(startTime)
// 	totalQueries := concurrency * queriesPerUser
// 	errors := 0
// 	for range errorChan {
// 		errors++
// 	}

// 	t.Logf("并发获取交易历史性能测试结果:")
// 	t.Logf("并发数: %d", concurrency)
// 	t.Logf("每个用户查询次数: %d", queriesPerUser)
// 	t.Logf("总查询次数: %d", totalQueries)
// 	t.Logf("总耗时: %v", duration)
// 	t.Logf("错误数: %d", errors)
// 	t.Logf("成功率: %.2f%%", float64(totalQueries-errors)/float64(totalQueries)*100)
// 	t.Logf("每秒处理查询数: %.2f", float64(totalQueries)/duration.Seconds())
// }
