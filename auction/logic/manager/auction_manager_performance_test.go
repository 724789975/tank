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

	// 并发数
	concurrency := 1000
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
			// 为每个goroutine使用不同的userID
			userID := "test_user_performance_" + string(rune(userIdx+'0'))
			testCtx := context.WithValue(ctx, "userId", userID)
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

	// 先创建一些出售订单
	sellUserID := "test_user_performance_sell"
	sellTestCtx := context.WithValue(ctx, "userId", sellUserID)
	for i := 0; i < 10; i++ {
		itemID := "test_item_perf_buy_" + string(rune(i+'0'))
		req := &auction.SellReq{
			ItemId:       itemID,
			Quantity:     100,
			Price:        100,
			ItemInfo:     "Test Item",
			IdempotentId: "test_sell_for_buy_perf_" + string(rune(i+'0')) + "_" + time.Now().String(),
		}
		_, err := manager.Sell(sellTestCtx, req)
		if err != nil {
			t.Skip("Failed to create sell orders for buy performance test:", err)
		}
	}

	// 并发数
	concurrency := 1000
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
			// 为每个goroutine使用不同的userID
			userID := "test_user_performance_buy_" + string(rune(userIdx+'0'))
			testCtx := context.WithValue(ctx, "userId", userID)
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
func TestAuctionManager_Performance_GetTransactionHistory(t *testing.T) {
	setupTest()
	defer teardownTest()

	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	manager := GetAuctionManager()

	// 并发数
	concurrency := 10
	// 每个用户查询次数
	queriesPerUser := 10

	var wg sync.WaitGroup
	// 创建一些交易记录
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(userIdx int) {
			defer wg.Done()
			// 为创建交易记录使用单独的 userID
			historyUserID := "test_user_performance_history" + string(rune(i+'0'))
			historyTestCtx := context.WithValue(ctx, "userId", historyUserID)

			itemID := "test_item_perf_history_" + string(rune(i+'0'))
			price := getMatchManager().GetMatchUnit(itemID).hourlyAvgPrice

			// 创建出售订单
			sellReq := &auction.SellReq{
				ItemId:       itemID,
				Quantity:     40,
				Price:        price,
				ItemInfo:     "Test Item",
				IdempotentId: "test_sell_for_history_perf_" + string(rune(i+'0')) + "_" + time.Now().String(),
			}
			_, err = manager.Sell(historyTestCtx, sellReq)
			if err != nil {
				t.Skip("Failed to create sell orders for history performance test:", err)
			}

			//购买10次每次5个
			for j := 0; j < 8; j++ {
				// 创建购买订单
				buyReq := &auction.BuyReq{
					ItemId:       itemID,
					Quantity:     5,
					Price:        price,
					IdempotentId: "test_buy_for_history_perf_" + string(rune(i+'0')) + "_" + string(rune(j+'0')) + "_" + time.Now().String(),
				}
				_, err = manager.Buy(historyTestCtx, buyReq)
				if err != nil {
					t.Skip("Failed to create buy orders for history performance test:", err)
				}

				// 等待交易完成
				time.Sleep(1000 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// 先通过GetTransactionsByTime获取交易记录
	getByTimeReq := &auction.GetTransactionsByTimeReq{
		StartTime: 0,
		EndTime:   time.Now().Unix(),
		Page:      1,
		PageSize:  100,
	}

	timeresps := make([]*auction.GetTransactionsByTimeRsp, 0, concurrency)
	for i := 0; i < concurrency; i++ {
		historyUserID := "test_user_performance_history" + string(rune(i+'0'))
		historyTestCtx := context.WithValue(ctx, "userId", historyUserID)
		getByTimeResp, err := manager.GetTransactionsByTime(historyTestCtx, getByTimeReq)
		if err != nil || getByTimeResp.Code != common.ErrorCode_OK {
			t.Skip("Failed to get transactions by time:", err)
		}
		timeresps = append(timeresps, getByTimeResp)
	}

	// 从GetTransactionsByTime响应中提取订单ID列表
	var orderIds = make([][]string, 0)
	for _, resp := range timeresps {
		orders := []string{}
		for _, record := range resp.Data.Records {
			if record.TradeDirection == "sell" {
				orders = append(orders, record.TransactionId)
			}
		}
		orderIds = append(orderIds, orders)
	}
	if len(orderIds) == 0 {
		t.Skip("No transaction records found")
	}
	var wg2 sync.WaitGroup
	errorChan := make(chan error, concurrency*queriesPerUser)
	startTime := time.Now()

	// 并发获取交易历史
	for i := 0; i < len(orderIds); i++ {
		orders := orderIds[i]
		wg2.Add(1)
		go func(userIdx int, orders []string) {
			historyUserID := "test_user_performance_history" + string(rune(i+'0'))
			historyTestCtx := context.WithValue(ctx, "userId", historyUserID)
			defer wg2.Done()
			// 使用与创建交易记录相同的userID，确保能够查询到交易记录
			testCtx := historyTestCtx
			for j := 0; j < len(orders); j++ {
				// 使用从GetTransactionsByTime获取的订单ID获取交易历史
				orderId := orders[j]
				getReq := &auction.GetTransactionHistoryReq{
					OrderId: orderId,
				}

				resp, err := manager.GetTransactionHistory(testCtx, getReq)
				if err != nil || resp.Code != common.ErrorCode_OK {
					errorChan <- err
				}
			}
		}(i, orders)
	}

	wg2.Wait()
	close(errorChan)

	duration := time.Since(startTime)
	totalQueries := concurrency * queriesPerUser
	errors := 0
	for range errorChan {
		errors++
	}

	t.Logf("并发获取交易历史性能测试结果:")
	t.Logf("并发数: %d", concurrency)
	t.Logf("每个用户查询次数: %d", queriesPerUser)
	t.Logf("总查询次数: %d", totalQueries)
	t.Logf("总耗时: %v", duration)
	t.Logf("错误数: %d", errors)
	t.Logf("成功率: %.2f%%", float64(totalQueries-errors)/float64(totalQueries)*100)
	t.Logf("每秒处理查询数: %.2f", float64(totalQueries)/duration.Seconds())
}
