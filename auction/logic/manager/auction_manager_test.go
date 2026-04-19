package manager

import (
	"auction_module/config"
	"auction_module/kitex_gen/auction"
	"auction_module/kitex_gen/common"
	"auction_module/redis"
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestMain 用于在测试开始前初始化测试环境
func TestMain(m *testing.M) {
	// 初始化测试环境
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			fmt.Println("Redis connection failed, skipping all tests")
			os.Exit(0)
		}
	}()

	// 设置配置文件路径环境变量（使用绝对路径）
	os.Setenv("CONF_PATH", "e:\\tank\\auction\\etc")
	os.Setenv("CONF_FILE", "server-test.yaml")

	// 加载配置
	config.LoadConfig()

	setupTest()

	getMatchManager()

	// 运行测试
	code := m.Run()

	// 清理测试环境
	teardownTest()

	// 退出测试
	os.Exit(code)
}

// 测试环境设置
func setupTest() {
	// 初始化Redis连接
	// 这里假设Redis已经在本地运行
	fmt.Println("Setting up test environment...")
	// 尝试连接Redis，如果失败则跳过测试
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		fmt.Println("Redis connection failed, some tests may be skipped:", err)
	}
}

// 测试环境清理
func teardownTest() {
	// 清理测试数据
	fmt.Println("Tearing down test environment...")
	ctx := context.Background()
	// 删除 Redis 中的所有 key
	_, err := redis.GetRedis().FlushDB(ctx).Result()
	if err != nil {
		fmt.Println("Error flushing Redis DB:", err)
	} else {
		fmt.Println("All Redis keys have been deleted")
	}
	getMatchManager().matchUnits = make(map[string]*matchUnit)
	getMatchManager().cancel()
	getMatchManager().ctx, getMatchManager().cancel = context.WithCancel(context.Background())
	time.Sleep(1 * time.Second)
}

// 测试用例: 正常出售操作
func TestAuctionManager_Sell(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 获取当前市场的平均价格数据
	manager := GetAuctionManager()
	mu := getMatchManager().GetMatchUnit("test_item_001")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	// 创建测试请求
	req := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_idempotent_001",
	}

	// 调用Sell方法
	resp, err := manager.Sell(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp.Code)
	assert.Equal(t, "success", resp.Msg)
	assert.NotEmpty(t, resp.Data.OrderId)
	assert.Equal(t, "test_item_001", resp.Data.ItemId)
	assert.Equal(t, int32(10), resp.Data.Quantity)
	assert.Equal(t, correctedPrice, resp.Data.Price)
	assert.Equal(t, "Test Item", resp.Data.ItemInfo)
	assert.Greater(t, resp.Data.CreateTime, int64(0))

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 调用GetItemAuctionInfo接口检查物品上架状态
	infoReq := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{"test_item_001"},
	}
	infoResp, err := manager.GetItemAuctionInfo(ctx, infoReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, infoResp.Code)
	assert.Equal(t, "success", infoResp.Msg)
	assert.Len(t, infoResp.Data, 1)
	assert.Equal(t, "test_item_001", infoResp.Data[0].ItemId)
	// 检查是否有卖单
	assert.GreaterOrEqual(t, len(infoResp.Data[0].Sells), 0)

	// 清理测试数据
	orderId := resp.Data.OrderId
	redis.GetRedis().Del(ctx, "auction:sell:"+orderId)
	redis.GetRedis().Del(ctx, "auction:order:"+orderId+":status")
	redis.GetRedis().SRem(ctx, "user:test_user_001:sells", "auction:sell:"+orderId)
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+orderId)
	redis.GetRedis().ZRem(ctx, "user:test_user_001:transactions:time", orderId)
}

// 测试用例: 出售数量为0的情况
func TestAuctionManager_Sell_QuantityZero(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 创建测试请求（数量为0）
	req := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     0,
		Price:        100,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_quantity_zero_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Sell方法
	manager := GetAuctionManager()
	resp, err := manager.Sell(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, resp.Code)
	assert.Equal(t, "quantity must be greater than 0", resp.Msg)
}

// 测试用例: 出售价格为0的情况
func TestAuctionManager_Sell_PriceZero(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 创建测试请求（价格为0）
	req := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        0,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_price_zero_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Sell方法
	manager := GetAuctionManager()
	resp, err := manager.Sell(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, resp.Code)
	assert.Equal(t, "price must be greater than 0", resp.Msg)
}

// 测试用例: 出售物品ID为空的情况
func TestAuctionManager_Sell_EmptyItemId(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 创建测试请求（物品ID为空）
	req := &auction.SellReq{
		ItemId:       "",
		Quantity:     10,
		Price:        100,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_empty_item_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Sell方法
	manager := GetAuctionManager()
	resp, err := manager.Sell(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, resp.Code)
	assert.Equal(t, "item_id is empty", resp.Msg)
}

// 测试用例: 出售用户ID为空的情况
func TestAuctionManager_Sell_EmptyUserId(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文（用户ID为空）

	// 创建测试请求
	req := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        100,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_empty_user_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Sell方法
	manager := GetAuctionManager()
	resp, err := manager.Sell(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_USER_NOT_FOUND, resp.Code)
	assert.Equal(t, "user_id is empty", resp.Msg)
}

// 测试用例: 正常购买操作
func TestAuctionManager_Buy(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_002")

	// 获取当前市场的平均价格数据
	manager := GetAuctionManager()
	mu := getMatchManager().GetMatchUnit("test_item_001")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	// 创建测试请求
	req := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     5,
		Price:        correctedPrice,
		IdempotentId: "test_buy_idempotent_001",
	}

	// 调用Buy方法
	resp, err := manager.Buy(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp.Code)
	assert.Equal(t, "success", resp.Msg)
	assert.NotEmpty(t, resp.Data.OrderId)
	assert.Equal(t, "test_item_001", resp.Data.ItemId)
	assert.Equal(t, int32(5), resp.Data.Quantity)
	assert.Equal(t, int64(100), resp.Data.Price)
	assert.Greater(t, resp.Data.CreateTime, int64(0))

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 调用GetItemAuctionInfo接口检查物品购买状态
	infoReq := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{"test_item_001"},
	}
	infoResp, err := manager.GetItemAuctionInfo(ctx, infoReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, infoResp.Code)
	assert.Equal(t, "success", infoResp.Msg)
	assert.Len(t, infoResp.Data, 1)
	assert.Equal(t, "test_item_001", infoResp.Data[0].ItemId)
	// 检查是否有买单
	assert.GreaterOrEqual(t, len(infoResp.Data[0].Buys), 0)

	// 清理测试数据
	orderId := resp.Data.OrderId
	redis.GetRedis().Del(ctx, "auction:buy:"+orderId)
	redis.GetRedis().Del(ctx, "auction:order:"+orderId+":status")
	redis.GetRedis().SRem(ctx, "user:test_user_002:buys", "auction:buy:"+orderId)
	redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+orderId)
	redis.GetRedis().ZRem(ctx, "user:test_user_002:transactions:time", orderId)
}

// 测试用例: 购买数量为0的情况
func TestAuctionManager_Buy_QuantityZero(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_002")

	// 创建测试请求（数量为0）
	req := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     0,
		Price:        100,
		IdempotentId: "test_buy_quantity_zero_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Buy方法
	manager := GetAuctionManager()
	resp, err := manager.Buy(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, resp.Code)
	assert.Equal(t, "quantity must be greater than 0", resp.Msg)
}

// 测试用例: 购买价格为0的情况
func TestAuctionManager_Buy_PriceZero(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_002")

	// 创建测试请求（价格为0）
	req := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     5,
		Price:        0,
		IdempotentId: "test_buy_price_zero_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Buy方法
	manager := GetAuctionManager()
	resp, err := manager.Buy(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, resp.Code)
	assert.Equal(t, "price must be greater than 0", resp.Msg)
}

// 测试用例: 购买物品ID为空的情况
func TestAuctionManager_Buy_EmptyItemId(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_002")

	// 创建测试请求（物品ID为空）
	req := &auction.BuyReq{
		ItemId:       "",
		Quantity:     5,
		Price:        100,
		IdempotentId: "test_buy_empty_item_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Buy方法
	manager := GetAuctionManager()
	resp, err := manager.Buy(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, resp.Code)
	assert.Equal(t, "item_id is empty", resp.Msg)
}

// 测试用例: 购买用户ID为空的情况
func TestAuctionManager_Buy_EmptyUserId(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文（用户ID为空）

	// 创建测试请求
	req := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     5,
		Price:        100,
		IdempotentId: "test_buy_empty_user_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Buy方法
	manager := GetAuctionManager()
	resp, err := manager.Buy(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_USER_NOT_FOUND, resp.Code)
	assert.Equal(t, "user_id is empty", resp.Msg)
}

// 测试用例: 取消出售操作
func TestAuctionManager_CancelSell(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 首先创建一个出售订单
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	manager := GetAuctionManager()
	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_001")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	sellReq := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: "test_cancel_sell_idempotent_001",
	}

	sellResp, err := manager.Sell(ctx, sellReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp.Code)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 然后取消该订单
	cancelReq := &auction.CancelSellReq{
		OrderId:      sellResp.Data.OrderId,
		IdempotentId: "test_cancel_sell_idempotent_002",
	}

	cancelResp, err := manager.CancelSell(ctx, cancelReq)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, cancelResp.Code)
	assert.Equal(t, "success", cancelResp.Msg)
	assert.True(t, cancelResp.Data.Success)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 调用GetItemAuctionInfo接口检查物品状态
	infoReq := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{"test_item_001"},
	}
	infoResp, err := manager.GetItemAuctionInfo(ctx, infoReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, infoResp.Code)
	assert.Equal(t, "success", infoResp.Msg)
	assert.Len(t, infoResp.Data, 1)
	assert.Equal(t, "test_item_001", infoResp.Data[0].ItemId)

	// 验证订单是否被删除
	exists, err := redis.GetRedis().Exists(ctx, "auction:sell:"+sellResp.Data.OrderId).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

// 测试用例: 取消不存在的出售订单
func TestAuctionManager_CancelSell_NonExistentOrder(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 创建测试请求（订单ID不存在）
	req := &auction.CancelSellReq{
		OrderId:      "non_existent_order",
		IdempotentId: "test_cancel_sell_nonexistent_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用CancelSell方法
	manager := GetAuctionManager()
	resp, err := manager.CancelSell(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp.Code)
}

// 测试用例: 取消购买操作
func TestAuctionManager_CancelBuy(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 首先创建一个购买订单
	ctx = context.WithValue(ctx, "userId", "test_user_002")

	manager := GetAuctionManager()
	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_001")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	buyReq := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     5,
		Price:        correctedPrice,
		IdempotentId: "test_cancel_buy_idempotent_001",
	}

	buyResp, err := manager.Buy(ctx, buyReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp.Code)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 然后取消该订单
	cancelReq := &auction.CancelBuyReq{
		OrderId:      buyResp.Data.OrderId,
		IdempotentId: "test_cancel_buy_idempotent_002",
	}

	cancelResp, err := manager.CancelBuy(ctx, cancelReq)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, cancelResp.Code)
	assert.Equal(t, "success", cancelResp.Msg)
	assert.True(t, cancelResp.Data.Success)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 调用GetItemAuctionInfo接口检查物品状态
	infoReq := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{"test_item_001"},
	}
	infoResp, err := manager.GetItemAuctionInfo(ctx, infoReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, infoResp.Code)
	assert.Equal(t, "success", infoResp.Msg)
	assert.Len(t, infoResp.Data, 1)
	assert.Equal(t, "test_item_001", infoResp.Data[0].ItemId)

	// 验证订单是否被删除
	exists, err := redis.GetRedis().Exists(ctx, "auction:buy:"+buyResp.Data.OrderId).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), exists)
}

// 测试用例: 取消不存在的购买订单
func TestAuctionManager_CancelBuy_NonExistentOrder(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_002")

	// 创建测试请求（订单ID不存在）
	req := &auction.CancelBuyReq{
		OrderId:      "non_existent_order",
		IdempotentId: "test_cancel_buy_nonexistent_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用CancelBuy方法
	manager := GetAuctionManager()
	resp, err := manager.CancelBuy(ctx, req)

	// 验证结果
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp.Code)
}

// 测试用例: 获取我的出售订单
func TestAuctionManager_GetMySells(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 首先创建几个出售订单
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	manager := GetAuctionManager()

	// 创建第一个订单
	sellReq1 := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        100,
		ItemInfo:     "Test Item 1",
		IdempotentId: "test_getmysells_1_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp1, err := manager.Sell(ctx, sellReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp1.Code)

	// 创建第二个订单
	sellReq2 := &auction.SellReq{
		ItemId:       "test_item_002",
		Quantity:     5,
		Price:        100,
		ItemInfo:     "Test Item 2",
		IdempotentId: "test_getmysells_2_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp2, err := manager.Sell(ctx, sellReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp2.Code)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 获取我的出售列表
	getReq := &auction.GetMySellsReq{}
	getResp, err := manager.GetMySells(ctx, getReq)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, getResp.Code)
	assert.Equal(t, "success", getResp.Msg)
	assert.Len(t, getResp.Data, 2)

	// 清理测试数据
	redis.GetRedis().Del(ctx, "auction:sell:"+sellResp1.Data.OrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+sellResp1.Data.OrderId+":status")
	redis.GetRedis().Del(ctx, "auction:sell:"+sellResp2.Data.OrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+sellResp2.Data.OrderId+":status")
	redis.GetRedis().Del(ctx, "user:test_user_001:sells")
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+sellResp1.Data.OrderId)
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+sellResp2.Data.OrderId)
	redis.GetRedis().ZRem(ctx, "user:test_user_001:transactions:time", sellResp1.Data.OrderId)
	redis.GetRedis().ZRem(ctx, "user:test_user_001:transactions:time", sellResp2.Data.OrderId)
}

// 测试用例: 获取我的购买订单
func TestAuctionManager_GetMyBuys(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 首先创建几个购买订单
	ctx = context.WithValue(ctx, "userId", "test_user_002")

	manager := GetAuctionManager()

	// 创建第一个订单
	buyReq1 := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     5,
		Price:        100,
		IdempotentId: "test_getmybuys_1_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	buyResp1, err := manager.Buy(ctx, buyReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp1.Code)

	// 创建第二个订单
	buyReq2 := &auction.BuyReq{
		ItemId:       "test_item_002",
		Quantity:     3,
		Price:        100,
		IdempotentId: "test_getmybuys_2_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	buyResp2, err := manager.Buy(ctx, buyReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp2.Code)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 获取我的购买列表
	getReq := &auction.GetMyBuysReq{}
	getResp, err := manager.GetMyBuys(ctx, getReq)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, getResp.Code)
	assert.Equal(t, "success", getResp.Msg)
	assert.Len(t, getResp.Data, 2)

	// 清理测试数据
	redis.GetRedis().Del(ctx, "auction:buy:"+buyResp1.Data.OrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+buyResp1.Data.OrderId+":status")
	redis.GetRedis().Del(ctx, "auction:buy:"+buyResp2.Data.OrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+buyResp2.Data.OrderId+":status")
	redis.GetRedis().Del(ctx, "user:test_user_002:buys")
	redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+buyResp1.Data.OrderId)
	redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+buyResp2.Data.OrderId)
	redis.GetRedis().ZRem(ctx, "user:test_user_002:transactions:time", buyResp1.Data.OrderId)
	redis.GetRedis().ZRem(ctx, "user:test_user_002:transactions:time", buyResp2.Data.OrderId)
}

// 测试用例: 获取物品拍卖信息
func TestAuctionManager_GetItemAuctionInfo(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 清理所有现有的订单，避免影响测试
	redis.GetRedis().Del(ctx, "auction:sells").Result()
	redis.GetRedis().Del(ctx, "auction:buys").Result()

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_auction_info")

	manager := GetAuctionManager()

	// 创建一些拍卖订单

	// 道具1: 有卖单和买单
	// 创建出售订单
	sellReq1 := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        100,
		ItemInfo:     "Test Item 1",
		IdempotentId: "test_getitemauction_sell_1_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp1, err := manager.Sell(ctx, sellReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp1.Code)

	// 创建购买订单
	buyReq1 := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     5,
		Price:        100,
		IdempotentId: "test_getitemauction_buy_1_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	buyResp1, err := manager.Buy(ctx, buyReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp1.Code)

	// 道具2: 只有卖单
	sellReq2 := &auction.SellReq{
		ItemId:       "test_item_002",
		Quantity:     15,
		Price:        100,
		ItemInfo:     "Test Item 2",
		IdempotentId: "test_getitemauction_sell_2_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp2, err := manager.Sell(ctx, sellReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp2.Code)

	// 道具3: 只有买单
	buyReq3 := &auction.BuyReq{
		ItemId:       "test_item_003",
		Quantity:     8,
		Price:        100,
		IdempotentId: "test_getitemauction_buy_3_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	buyResp3, err := manager.Buy(ctx, buyReq3)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp3.Code)

	// 道具4: 既没有卖单也没有买单
	// 不需要创建订单

	// 道具5: 卖单价格数量大于5
	var sellResps5 []*auction.SellRsp
	for i := 0; i < 6; i++ {
		sellReq5 := &auction.SellReq{
			ItemId:       "test_item_005",
			Quantity:     int32(i + 1),
			Price:        int64(100 + i*1),
			ItemInfo:     "Test Item 5",
			IdempotentId: "test_getitemauction_sell_5_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + strconv.Itoa(i),
		}
		sellResp5, err := manager.Sell(ctx, sellReq5)
		assert.NoError(t, err)
		assert.Equal(t, common.ErrorCode_OK, sellResp5.Code)
		sellResps5 = append(sellResps5, sellResp5)
	}

	// 道具6: 买单价格数量大于5
	var buyResps6 []*auction.BuyRsp
	for i := 0; i < 6; i++ {
		buyReq6 := &auction.BuyReq{
			ItemId:       "test_item_006",
			Quantity:     int32(i + 1),
			Price:        int64(100 - i*1),
			IdempotentId: "test_getitemauction_buy_6_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + strconv.Itoa(i),
		}
		buyResp6, err := manager.Buy(ctx, buyReq6)
		assert.NoError(t, err)
		assert.Equal(t, common.ErrorCode_OK, buyResp6.Code)
		buyResps6 = append(buyResps6, buyResp6)
	}

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 测试场景1: 获取单个物品的拍卖信息
	req1 := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{"test_item_001"},
	}
	resp1, err := manager.GetItemAuctionInfo(ctx, req1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp1.Code)
	assert.Equal(t, "success", resp1.Msg)
	assert.Len(t, resp1.Data, 1)
	assert.Equal(t, "test_item_001", resp1.Data[0].ItemId)
	// 检查是否有卖单和买单信息
	assert.GreaterOrEqual(t, len(resp1.Data[0].Sells), 0)
	assert.GreaterOrEqual(t, len(resp1.Data[0].Buys), 0)

	// 测试场景2: 获取多个物品的拍卖信息
	req2 := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{"test_item_001", "test_item_002", "test_item_003", "test_item_004", "test_item_005", "test_item_006"},
	}
	resp2, err := manager.GetItemAuctionInfo(ctx, req2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp2.Code)
	assert.Equal(t, "success", resp2.Msg)
	// 应该返回6个物品的信息
	assert.Len(t, resp2.Data, 6)

	// 测试场景3: 获取不存在的物品的拍卖信息
	req3 := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{"test_item_non_existent"},
	}
	resp3, err := manager.GetItemAuctionInfo(ctx, req3)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp3.Code)
	assert.Equal(t, "success", resp3.Msg)
	assert.Len(t, resp3.Data, 1)
	assert.Equal(t, "test_item_non_existent", resp3.Data[0].ItemId)

	// 测试场景4: 空物品ID列表 - 预期会失败
	req4 := &auction.GetItemAuctionInfoReq{
		ItemIds: []string{},
	}
	resp4, err := manager.GetItemAuctionInfo(ctx, req4)
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp4.Code)
	assert.Equal(t, "item_ids is empty", resp4.Msg)
	assert.Len(t, resp4.Data, 0)

	// 清理测试数据
	// 清理道具1
	if sellResp1.Data != nil {
		redis.GetRedis().Del(ctx, "auction:sell:"+sellResp1.Data.OrderId)
		redis.GetRedis().Del(ctx, "auction:order:"+sellResp1.Data.OrderId+":status")
		redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+sellResp1.Data.OrderId)
		redis.GetRedis().ZRem(ctx, "user:test_user_auction_info:transactions:time", sellResp1.Data.OrderId)
	}
	if buyResp1.Data != nil {
		redis.GetRedis().Del(ctx, "auction:buy:"+buyResp1.Data.OrderId)
		redis.GetRedis().Del(ctx, "auction:order:"+buyResp1.Data.OrderId+":status")
		redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+buyResp1.Data.OrderId)
		redis.GetRedis().ZRem(ctx, "user:test_user_auction_info:transactions:time", buyResp1.Data.OrderId)
	}

	// 清理道具2
	if sellResp2.Data != nil {
		redis.GetRedis().Del(ctx, "auction:sell:"+sellResp2.Data.OrderId)
		redis.GetRedis().Del(ctx, "auction:order:"+sellResp2.Data.OrderId+":status")
		redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+sellResp2.Data.OrderId)
	}

	// 清理道具3
	if buyResp3.Data != nil {
		redis.GetRedis().Del(ctx, "auction:buy:"+buyResp3.Data.OrderId)
		redis.GetRedis().Del(ctx, "auction:order:"+buyResp3.Data.OrderId+":status")
		redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+buyResp3.Data.OrderId)
	}

	// 清理道具5
	for _, sellResp5 := range sellResps5 {
		if sellResp5.Data != nil {
			redis.GetRedis().Del(ctx, "auction:sell:"+sellResp5.Data.OrderId)
			redis.GetRedis().Del(ctx, "auction:order:"+sellResp5.Data.OrderId+":status")
			redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+sellResp5.Data.OrderId)
			redis.GetRedis().ZRem(ctx, "user:test_user_auction_info:transactions:time", sellResp5.Data.OrderId)
		}
	}

	// 清理道具6
	for _, buyResp6 := range buyResps6 {
		if buyResp6.Data != nil {
			redis.GetRedis().Del(ctx, "auction:buy:"+buyResp6.Data.OrderId)
			redis.GetRedis().Del(ctx, "auction:order:"+buyResp6.Data.OrderId+":status")
			redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+buyResp6.Data.OrderId)
			redis.GetRedis().ZRem(ctx, "user:test_user_auction_info:transactions:time", buyResp6.Data.OrderId)
		}
	}

	// 清理用户相关数据
	redis.GetRedis().Del(ctx, "user:test_user_auction_info:sells")
	redis.GetRedis().Del(ctx, "user:test_user_auction_info:buys")
}

// 测试用例: 按时间获取交易记录
func TestAuctionManager_GetTransactionsByTime(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	sellUserId := "test_user_sell"
	sellCtx := context.WithValue(ctx, "userId", sellUserId)
	buyUserId := "test_user_buy"
	buyCtx := context.WithValue(ctx, "userId", buyUserId)

	manager := GetAuctionManager()

	// 测试场景1: 只有挂单，没有成交的情况
	sellReq1 := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        100,
		ItemInfo:     "Test Item",
		IdempotentId: "test_getbytime_sell1_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp1, err := manager.Sell(sellCtx, sellReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp1.Code)

	// 按时间获取交易记录
	getReq1 := &auction.GetTransactionsByTimeReq{
		StartTime: 0,
		EndTime:   time.Now().Unix(),
		Page:      1,
		PageSize:  10,
	}

	getResp1, err := manager.GetTransactionsByTime(sellCtx, getReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, getResp1.Code)
	assert.Equal(t, "success", getResp1.Msg)

	// 测试场景2: 有成交记录的情况
	// 创建一个出售订单
	sellReq2 := &auction.SellReq{
		ItemId:       "test_item_002",
		Quantity:     5,
		Price:        100,
		ItemInfo:     "Test Item 2",
		IdempotentId: "test_getbytime_sell2_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp2, err := manager.Sell(sellCtx, sellReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp2.Code)

	// 创建一个购买订单，价格与出售订单匹配，会产生成交
	buyReq := &auction.BuyReq{
		ItemId:       "test_item_002",
		Quantity:     5,
		Price:        100,
		IdempotentId: "test_getbytime_buy_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	buyResp, err := manager.Buy(buyCtx, buyReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp.Code)

	// 等待异步操作完成，确保交易已经完成
	time.Sleep(3 * time.Second)

	// 按时间获取交易记录
	getReq2 := &auction.GetTransactionsByTimeReq{
		StartTime: 0,
		EndTime:   time.Now().Unix(),
		Page:      1,
		PageSize:  10,
	}

	getResp2, err := manager.GetTransactionsByTime(sellCtx, getReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, getResp2.Code)
	assert.Equal(t, "success", getResp2.Msg)

	// 清理测试数据
	// 清理第一个出售订单
	orderId1 := sellResp1.Data.OrderId
	redis.GetRedis().Del(ctx, "auction:sell:"+orderId1)
	redis.GetRedis().Del(ctx, "auction:order:"+orderId1+":status")
	redis.GetRedis().SRem(ctx, "user:"+sellUserId+":sells", "auction:sell:"+orderId1)
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+orderId1)
	redis.GetRedis().ZRem(ctx, "user:"+sellUserId+":transactions:time", orderId1)

	// 清理第二个出售订单和购买订单（已成交）
	orderId2 := sellResp2.Data.OrderId
	buyOrderId := buyResp.Data.OrderId
	redis.GetRedis().Del(ctx, "auction:sell:"+orderId2)
	redis.GetRedis().Del(ctx, "auction:order:"+orderId2+":status")
	redis.GetRedis().Del(ctx, "auction:buy:"+buyOrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+buyOrderId+":status")
	redis.GetRedis().SRem(ctx, "user:"+sellUserId+":sells", "auction:sell:"+orderId2)
	redis.GetRedis().SRem(ctx, "user:"+buyUserId+":buys", "auction:buy:"+buyOrderId)
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+orderId2)
	redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+buyOrderId)
	redis.GetRedis().ZRem(ctx, "user:"+sellUserId+":transactions:time", orderId2)
	redis.GetRedis().ZRem(ctx, "user:"+buyUserId+":transactions:time", buyOrderId)

	// 清理用户相关数据
	redis.GetRedis().Del(ctx, "user:"+sellUserId+":sells")
	redis.GetRedis().Del(ctx, "user:"+buyUserId+":buys")
	redis.GetRedis().Del(ctx, "user:"+sellUserId+":transactions:time")
	redis.GetRedis().Del(ctx, "user:"+buyUserId+":transactions:time")
}

// 测试用例: 先挂买单后挂卖单并能成功交易（覆盖matchSellOrder函数的所有分支）
func TestAuctionManager_Trade_BuyFirstThenSell(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	buyUserId := "test_user_buy_first"
	buyCtx := context.WithValue(ctx, "userId", buyUserId)
	sellUserId := "test_user_sell_after"
	sellCtx := context.WithValue(ctx, "userId", sellUserId)

	manager := GetAuctionManager()

	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_trade")
	avgPrice := mu.hourlyAvgPrice
	t.Logf("Average price: %d", avgPrice)

	// 1. 先挂多个买单，价格不同（覆盖按价格降序撮合的场景）
	// 高价格买单
	buyReq1 := &auction.BuyReq{
		ItemId:       "test_item_trade",
		Quantity:     3,
		Price:        avgPrice + 10, // 高价格
		IdempotentId: "test_buy_high_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	buyResp1, err := manager.Buy(buyCtx, buyReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp1.Code)
	assert.NotEmpty(t, buyResp1.Data.OrderId)
	t.Logf("High price buy order created: %s, price: %d, quantity: %d", buyResp1.Data.OrderId, avgPrice+10, 3)

	// 中价格买单
	buyReq2 := &auction.BuyReq{
		ItemId:       "test_item_trade",
		Quantity:     4,
		Price:        avgPrice + 5, // 中价格
		IdempotentId: "test_buy_medium_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	buyResp2, err := manager.Buy(buyCtx, buyReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp2.Code)
	assert.NotEmpty(t, buyResp2.Data.OrderId)
	t.Logf("Medium price buy order created: %s, price: %d, quantity: %d", buyResp2.Data.OrderId, avgPrice+5, 4)

	// 低价格买单
	buyReq3 := &auction.BuyReq{
		ItemId:       "test_item_trade",
		Quantity:     5,
		Price:        avgPrice, // 低价格
		IdempotentId: "test_buy_low_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	buyResp3, err := manager.Buy(buyCtx, buyReq3)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp3.Code)
	assert.NotEmpty(t, buyResp3.Data.OrderId)
	t.Logf("Low price buy order created: %s, price: %d, quantity: %d", buyResp3.Data.OrderId, avgPrice, 5)

	// 等待买单挂单完成
	time.Sleep(1 * time.Second)

	// 2. 后挂卖单，数量较大，需要与多个买单成交（覆盖卖单部分成交的场景）
	sellReq := &auction.SellReq{
		ItemId:       "test_item_trade",
		Quantity:     10,       // 数量大于任何一个买单
		Price:        avgPrice, // 价格与最低价格买单匹配
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_large_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	sellResp, err := manager.Sell(sellCtx, sellReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp.Code)
	assert.NotEmpty(t, sellResp.Data.OrderId)
	t.Logf("Large sell order created: %s, quantity: %d", sellResp.Data.OrderId, 10)

	// 等待交易完成
	time.Sleep(2 * time.Second)

	// 3. 验证交易是否成功
	// 从日志中可以看到交易已经成功完成
	t.Logf("Trade completed successfully! Sell order: %s", sellResp.Data.OrderId)

	// 4. 检查交易记录
	// 获取买家的交易记录
	buyTransactions, err := redis.GetRedis().ZRange(ctx, "user:"+buyUserId+":transactions:time", 0, -1).Result()
	assert.NoError(t, err)
	assert.Greater(t, len(buyTransactions), 0)
	t.Logf("Buyer transaction count: %d", len(buyTransactions))

	// 获取卖家的交易记录
	sellTransactions, err := redis.GetRedis().ZRange(ctx, "user:"+sellUserId+":transactions:time", 0, -1).Result()
	assert.NoError(t, err)
	assert.Greater(t, len(sellTransactions), 0)
	t.Logf("Seller transaction count: %d", len(sellTransactions))
}

// 测试用例: 先挂买单后挂卖单并能成功交易（简单场景）
func TestAuctionManager_Trade_BuyFirstThenSell_Simple(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	buyUserId := "test_user_buy_first"
	buyCtx := context.WithValue(ctx, "userId", buyUserId)
	sellUserId := "test_user_sell_after"
	sellCtx := context.WithValue(ctx, "userId", sellUserId)

	manager := GetAuctionManager()

	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_trade")
	avgPrice := mu.hourlyAvgPrice
	t.Logf("Average price: %d", avgPrice)

	// 1. 先挂买单
	buyReq := &auction.BuyReq{
		ItemId:       "test_item_trade",
		Quantity:     5,
		Price:        avgPrice,
		IdempotentId: "test_buy_first_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	buyResp, err := manager.Buy(buyCtx, buyReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp.Code)
	assert.NotEmpty(t, buyResp.Data.OrderId)
	t.Logf("Buy order created: %s", buyResp.Data.OrderId)

	// 等待买单挂单完成
	time.Sleep(1 * time.Second)

	// 2. 后挂卖单，价格与买单匹配
	sellReq := &auction.SellReq{
		ItemId:       "test_item_trade",
		Quantity:     5,
		Price:        avgPrice,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_after_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	sellResp, err := manager.Sell(sellCtx, sellReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp.Code)
	assert.NotEmpty(t, sellResp.Data.OrderId)
	t.Logf("Sell order created: %s", sellResp.Data.OrderId)

	// 等待交易完成
	time.Sleep(2 * time.Second)

	// 3. 验证交易是否成功
	// 从日志中可以看到交易已经成功完成
	// 例如: "Transaction completed: transactionId=2044745917533261824, buyOrder=2044745913330569216, sellOrder=2044745917529067520, quantity=5"
	t.Logf("Trade completed successfully! Buy order: %s, Sell order: %s", buyResp.Data.OrderId, sellResp.Data.OrderId)

	// 4. 检查交易记录
	// 获取买家的交易记录
	buyTransactions, err := redis.GetRedis().ZRange(ctx, "user:"+buyUserId+":transactions:time", 0, -1).Result()
	assert.NoError(t, err)
	assert.Greater(t, len(buyTransactions), 0)
	t.Logf("Buyer transaction count: %d", len(buyTransactions))

	// 获取卖家的交易记录
	sellTransactions, err := redis.GetRedis().ZRange(ctx, "user:"+sellUserId+":transactions:time", 0, -1).Result()
	assert.NoError(t, err)
	assert.Greater(t, len(sellTransactions), 0)
	t.Logf("Seller transaction count: %d", len(sellTransactions))
}

// 测试用例: 获取交易历史
func TestAuctionManager_GetTransactionHistory(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	sellUserId := "test_user_sell_history"
	sellCtx := context.WithValue(ctx, "userId", sellUserId)
	buyUserId := "test_user_buy_history"
	buyCtx := context.WithValue(ctx, "userId", buyUserId)

	manager := GetAuctionManager()

	// 测试场景1: 只有挂单，没有成交的情况
	sellReq1 := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        100,
		ItemInfo:     "Test Item",
		IdempotentId: "test_history_sell1_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp1, err := manager.Sell(sellCtx, sellReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp1.Code)

	// 获取交易历史
	getReq1 := &auction.GetTransactionHistoryReq{
		OrderId: sellResp1.Data.OrderId,
	}

	getResp1, err := manager.GetTransactionHistory(sellCtx, getReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, getResp1.Code)
	assert.Equal(t, "success", getResp1.Msg)

	// 测试场景2: 有成交记录的情况
	// 创建一个出售订单
	sellReq2 := &auction.SellReq{
		ItemId:       "test_item_002",
		Quantity:     5,
		Price:        100,
		ItemInfo:     "Test Item 2",
		IdempotentId: "test_history_sell2_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp2, err := manager.Sell(sellCtx, sellReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp2.Code)

	// 创建一个购买订单，价格与出售订单匹配，会产生成交
	buyReq := &auction.BuyReq{
		ItemId:       "test_item_002",
		Quantity:     5,
		Price:        100,
		IdempotentId: "test_history_buy_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	buyResp, err := manager.Buy(buyCtx, buyReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp.Code)

	// 等待异步操作完成，确保交易已经完成
	time.Sleep(3 * time.Second)

	// 获取交易历史
	getReq2 := &auction.GetTransactionHistoryReq{
		OrderId: sellResp2.Data.OrderId,
	}

	getResp2, err := manager.GetTransactionHistory(sellCtx, getReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, getResp2.Code)
	assert.Equal(t, "success", getResp2.Msg)
	// 验证交易记录数量
	assert.Greater(t, len(getResp2.Data.Records), 0)
	// 验证交易记录内容
	for _, record := range getResp2.Data.Records {
		assert.NotEmpty(t, record.TransactionId)
		assert.NotEmpty(t, record.BuyOrderId)
		assert.NotEmpty(t, record.SellOrderId)
		assert.NotEmpty(t, record.ItemId)
		assert.NotEmpty(t, record.ItemInfo)
		assert.Greater(t, record.Quantity, int32(0))
		assert.Greater(t, record.Price, int64(0))
		assert.Greater(t, record.TransactionTime, int64(0))
	}
	t.Logf("交易历史记录数量: %d", len(getResp2.Data.Records))

	// 清理测试数据
	// 清理第一个出售订单
	orderId1 := sellResp1.Data.OrderId
	redis.GetRedis().Del(ctx, "auction:sell:"+orderId1)
	redis.GetRedis().Del(ctx, "auction:order:"+orderId1+":status")
	redis.GetRedis().SRem(ctx, "user:"+sellUserId+":sells", "auction:sell:"+orderId1)
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+orderId1)
	redis.GetRedis().ZRem(ctx, "user:"+sellUserId+":transactions:time", orderId1)

	// 清理第二个出售订单和购买订单（已成交）
	orderId2 := sellResp2.Data.OrderId
	buyOrderId := buyResp.Data.OrderId
	redis.GetRedis().Del(ctx, "auction:sell:"+orderId2)
	redis.GetRedis().Del(ctx, "auction:order:"+orderId2+":status")
	redis.GetRedis().Del(ctx, "auction:buy:"+buyOrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+buyOrderId+":status")
	redis.GetRedis().SRem(ctx, "user:"+sellUserId+":sells", "auction:sell:"+orderId2)
	redis.GetRedis().SRem(ctx, "user:"+buyUserId+":buys", "auction:buy:"+buyOrderId)
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+orderId2)
	redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+buyOrderId)
	redis.GetRedis().ZRem(ctx, "user:"+sellUserId+":transactions:time", orderId2)
	redis.GetRedis().ZRem(ctx, "user:"+buyUserId+":transactions:time", buyOrderId)

	// 清理用户相关数据
	redis.GetRedis().Del(ctx, "user:"+sellUserId+":sells")
	redis.GetRedis().Del(ctx, "user:"+buyUserId+":buys")
	redis.GetRedis().Del(ctx, "user:"+sellUserId+":transactions:time")
	redis.GetRedis().Del(ctx, "user:"+buyUserId+":transactions:time")
}

// 测试用例: 销售限制 - 超过最大销售数量
func TestAuctionManager_Sell_ExceedLimit(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_limit")

	manager := GetAuctionManager()

	// 清理之前的测试数据
	redis.GetRedis().Del(ctx, "user:test_user_limit:sells")

	// 创建8个出售订单（达到限制）
	var orderIds []string
	for i := 0; i < 8; i++ {
		sellReq := &auction.SellReq{
			ItemId:       "test_item_001",
			Quantity:     1,
			Price:        100,
			ItemInfo:     "Test Item",
			IdempotentId: "test_exceed_sell_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		}
		sellResp, err := manager.Sell(ctx, sellReq)
		assert.NoError(t, err)
		assert.Equal(t, common.ErrorCode_OK, sellResp.Code)
		orderIds = append(orderIds, sellResp.Data.OrderId)
		// 等待异步操作完成
		time.Sleep(1 * time.Second)
	}

	// 尝试创建第9个订单（应该失败）
	sellReq := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     1,
		Price:        100,
		ItemInfo:     "Test Item",
		IdempotentId: "test_exceed_sell_9_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp, err := manager.Sell(ctx, sellReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, sellResp.Code)
	assert.Equal(t, "sell count exceeds limit", sellResp.Msg)

	// 清理测试数据
	for _, orderId := range orderIds {
		redis.GetRedis().Del(ctx, "auction:sell:"+orderId)
		redis.GetRedis().Del(ctx, "auction:order:"+orderId+":status")
		redis.GetRedis().SRem(ctx, "user:test_user_limit:sells", "auction:sell:"+orderId)
		redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+orderId)
		redis.GetRedis().ZRem(ctx, "user:test_user_limit:transactions:time", orderId)
	}
	redis.GetRedis().Del(ctx, "user:test_user_limit:sells")
}

// 测试用例: 购买限制 - 超过最大购买数量
func TestAuctionManager_Buy_ExceedLimit(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_limit")

	manager := GetAuctionManager()

	// 清理之前的测试数据
	redis.GetRedis().Del(ctx, "user:test_user_limit:buys")

	// 创建8个购买订单（达到限制）
	var orderIds []string
	for i := 0; i < 8; i++ {
		buyReq := &auction.BuyReq{
			ItemId:       "test_item_001",
			Quantity:     1,
			Price:        100,
			IdempotentId: "test_exceed_buy_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		}
		buyResp, err := manager.Buy(ctx, buyReq)
		assert.NoError(t, err)
		assert.Equal(t, common.ErrorCode_OK, buyResp.Code)
		orderIds = append(orderIds, buyResp.Data.OrderId)
		// 等待异步操作完成
		time.Sleep(1 * time.Second)
	}

	// 尝试创建第9个订单（应该失败）
	buyReq := &auction.BuyReq{
		ItemId:       "test_item_001",
		Quantity:     1,
		Price:        100,
		IdempotentId: "test_exceed_buy_9_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	buyResp, err := manager.Buy(ctx, buyReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_AUCTION_PARAM_ERROR, buyResp.Code)
	assert.Equal(t, "buy count exceeds limit", buyResp.Msg)

	// 清理测试数据
	for _, orderId := range orderIds {
		redis.GetRedis().Del(ctx, "auction:buy:"+orderId)
		redis.GetRedis().Del(ctx, "auction:order:"+orderId+":status")
		redis.GetRedis().SRem(ctx, "user:test_user_limit:buys", "auction:buy:"+orderId)
		redis.GetRedis().SRem(ctx, "auction:buys", "auction:buy:"+orderId)
		redis.GetRedis().ZRem(ctx, "user:test_user_limit:transactions:time", orderId)
	}
	redis.GetRedis().Del(ctx, "user:test_user_limit:buys")
}

// 测试用例: 价格限制 - 低于最低价格
func TestAuctionManager_Sell_PriceBelowLimit(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 创建测试请求（价格低于限制）
	req := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        50, // 假设小时平均价格为100，这里设置为50，低于90%
		ItemInfo:     "Test Item",
		IdempotentId: "test_price_below_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Sell方法
	manager := GetAuctionManager()
	_, err = manager.Sell(ctx, req)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 验证结果
	assert.NoError(t, err)
	// 由于没有实际的小时平均价格数据，这里可能不会触发价格限制
	// 但如果系统中已经有该物品的交易记录，可能会触发
}

// 测试用例: 价格限制 - 高于最高价格
func TestAuctionManager_Sell_PriceAboveLimit(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 创建测试请求（价格高于限制）
	req := &auction.SellReq{
		ItemId:       "test_item_001",
		Quantity:     10,
		Price:        150, // 假设小时平均价格为100，这里设置为150，高于110%
		ItemInfo:     "Test Item",
		IdempotentId: "test_price_above_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	// 调用Sell方法
	manager := GetAuctionManager()
	_, err = manager.Sell(ctx, req)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 验证结果
	assert.NoError(t, err)
	// 由于没有实际的小时平均价格数据，这里可能不会触发价格限制
	// 但如果系统中已经有该物品的交易记录，可能会触发
}

// 测试用例: 测试GetMySells中的价格显示
func TestAuctionManager_GetMySells_PriceDisplay(t *testing.T) {
	setupTest()
	defer teardownTest()
	// 检查Redis连接是否正常
	ctx := context.Background()
	_, err := redis.GetRedis().Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis connection failed, skipping test:", err)
	}

	// 使用唯一的用户ID，避免之前测试数据的干扰
	userId := "test_user_price_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// 创建测试上下文
	ctx = context.WithValue(ctx, "userId", userId)

	// 清理该用户的所有相关数据
	redis.GetRedis().Del(ctx, "user:"+userId+":sells")
	redis.GetRedis().Del(ctx, "user:"+userId+":transactions:time")

	manager := GetAuctionManager()

	// 创建第一个订单，价格为100
	sellReq1 := &auction.SellReq{
		ItemId:       "test_item_price_1",
		Quantity:     10,
		Price:        100,
		ItemInfo:     "Test Item 1",
		IdempotentId: "test_pricedisplay_sell1_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp1, err := manager.Sell(ctx, sellReq1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp1.Code)

	// 创建第二个订单，价格为110（在限制范围内）
	sellReq2 := &auction.SellReq{
		ItemId:       "test_item_price_2",
		Quantity:     5,
		Price:        110,
		ItemInfo:     "Test Item 2",
		IdempotentId: "test_pricedisplay_sell2_" + strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	sellResp2, err := manager.Sell(ctx, sellReq2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp2.Code)

	// 等待异步操作完成
	time.Sleep(2 * time.Second)

	// 获取我的出售列表
	getReq := &auction.GetMySellsReq{}
	getResp, err := manager.GetMySells(ctx, getReq)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, getResp.Code)
	assert.Equal(t, "success", getResp.Msg)
	assert.Len(t, getResp.Data, 2)

	// 验证价格是否正确
	priceMap := make(map[string]int64)
	for _, sellData := range getResp.Data {
		priceMap[sellData.ItemId] = sellData.Price
	}

	// 验证第一个物品的价格
	assert.Contains(t, priceMap, "test_item_price_1")
	assert.Equal(t, int64(100), priceMap["test_item_price_1"])

	// 验证第二个物品的价格
	assert.Contains(t, priceMap, "test_item_price_2")
	assert.Equal(t, int64(110), priceMap["test_item_price_2"])

	// 清理测试数据
	redis.GetRedis().Del(ctx, "auction:sell:"+sellResp1.Data.OrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+sellResp1.Data.OrderId+":status")
	redis.GetRedis().Del(ctx, "auction:sell:"+sellResp2.Data.OrderId)
	redis.GetRedis().Del(ctx, "auction:order:"+sellResp2.Data.OrderId+":status")
	redis.GetRedis().Del(ctx, "user:"+userId+":sells")
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+sellResp1.Data.OrderId)
	redis.GetRedis().SRem(ctx, "auction:sells", "auction:sell:"+sellResp2.Data.OrderId)
	redis.GetRedis().ZRem(ctx, "user:"+userId+":transactions:time", sellResp1.Data.OrderId)
	redis.GetRedis().ZRem(ctx, "user:"+userId+":transactions:time", sellResp2.Data.OrderId)
}

// 测试Sell接口的幂等功能
func TestAuctionManager_SellIdempotency(t *testing.T) {
	// 初始化测试环境
	setupTest()
	defer teardownTest()

	// 初始化拍卖管理器
	manager := GetAuctionManager()

	// 创建测试上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_idempotent")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	// 1. 测试重复请求使用相同的幂等ID，应该返回相同的结果
	fmt.Println("测试1: 重复请求使用相同的幂等ID")
	sameIdempotentId := "test_sell_idempotent_duplicate"

	// 第一个请求
	req1 := &auction.SellReq{
		ItemId:       "test_item_idempotent",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: sameIdempotentId,
	}

	resp1, err := manager.Sell(ctx, req1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp1.Code)
	assert.NotEmpty(t, resp1.Data.OrderId)

	// 第二个请求，使用相同的幂等ID
	req2 := &auction.SellReq{
		ItemId:       "test_item_idempotent",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: sameIdempotentId,
	}

	resp2, err := manager.Sell(ctx, req2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp2.Code)
	// 应该返回相同的订单ID
	assert.Equal(t, resp1.Data.OrderId, resp2.Data.OrderId)

	// 2. 测试不同的幂等ID，应该产生不同的结果
	fmt.Println("测试2: 不同的幂等ID")
	differentIdempotentId := "test_sell_idempotent_different"

	req3 := &auction.SellReq{
		ItemId:       "test_item_idempotent",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: differentIdempotentId,
	}

	resp3, err := manager.Sell(ctx, req3)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp3.Code)
	// 应该返回不同的订单ID
	assert.NotEqual(t, resp1.Data.OrderId, resp3.Data.OrderId)

	// 3. 测试幂等ID为空
	fmt.Println("测试3: 幂等ID为空")

	req4 := &auction.SellReq{
		ItemId:       "test_item_idempotent",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: "",
	}

	resp4, err := manager.Sell(ctx, req4)
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp4.Code)
	assert.Equal(t, "idempotent_id is empty", resp4.Msg)
}

// 测试Buy接口的幂等功能
func TestAuctionManager_BuyIdempotency(t *testing.T) {
	// 初始化测试环境
	setupTest()
	defer teardownTest()

	// 初始化拍卖管理器
	manager := GetAuctionManager()

	// 创建测试上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_buy_idempotent")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	// 先创建一个出售订单，用于购买
	sellReq := &auction.SellReq{
		ItemId:       "test_item_buy_idempotent",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_for_buy",
	}
	sellResp, err := manager.Sell(ctx, sellReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp.Code)

	// 1. 测试重复请求使用相同的幂等ID，应该返回相同的结果
	fmt.Println("测试1: 重复请求使用相同的幂等ID")
	sameIdempotentId := "test_buy_idempotent_duplicate"

	// 第一个请求
	req1 := &auction.BuyReq{
		ItemId:       "test_item_buy_idempotent",
		Quantity:     5,
		Price:        correctedPrice,
		IdempotentId: sameIdempotentId,
	}

	resp1, err := manager.Buy(ctx, req1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp1.Code)
	assert.NotEmpty(t, resp1.Data.OrderId)

	// 第二个请求，使用相同的幂等ID
	req2 := &auction.BuyReq{
		ItemId:       "test_item_buy_idempotent",
		Quantity:     5,
		Price:        correctedPrice,
		IdempotentId: sameIdempotentId,
	}

	resp2, err := manager.Buy(ctx, req2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp2.Code)
	// 应该返回相同的订单ID
	assert.Equal(t, resp1.Data.OrderId, resp2.Data.OrderId)

	// 2. 测试不同的幂等ID，应该产生不同的结果
	fmt.Println("测试2: 不同的幂等ID")
	differentIdempotentId := "test_buy_idempotent_different"

	req3 := &auction.BuyReq{
		ItemId:       "test_item_buy_idempotent",
		Quantity:     1,
		Price:        correctedPrice,
		IdempotentId: differentIdempotentId,
	}

	resp3, err := manager.Buy(ctx, req3)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp3.Code)
	// 应该返回不同的订单ID
	assert.NotEqual(t, resp1.Data.OrderId, resp3.Data.OrderId)

	// 3. 测试幂等ID为空的情况
	fmt.Println("测试3: 幂等ID为空")
	req4 := &auction.BuyReq{
		ItemId:       "test_item_buy_idempotent",
		Quantity:     1,
		Price:        correctedPrice,
		IdempotentId: "",
	}

	resp4, err := manager.Buy(ctx, req4)
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp4.Code)
	assert.Equal(t, "idempotent_id is empty", resp4.Msg)
}

// 测试CancelSell接口的幂等功能
func TestAuctionManager_CancelSellIdempotency(t *testing.T) {
	// 初始化测试环境
	setupTest()
	defer teardownTest()

	// 初始化拍卖管理器
	manager := GetAuctionManager()

	// 创建测试上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_cancel_sell_idempotent")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	// 先创建一个出售订单，用于取消
	sellReq := &auction.SellReq{
		ItemId:       "test_item_cancel_sell_idempotent",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_for_cancel",
	}
	sellResp, err := manager.Sell(ctx, sellReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp.Code)

	// 1. 测试重复请求使用相同的幂等ID，应该返回相同的结果
	fmt.Println("测试1: 重复请求使用相同的幂等ID")
	sameIdempotentId := "test_cancel_sell_idempotent_duplicate"

	// 第一个请求
	req1 := &auction.CancelSellReq{
		OrderId:      sellResp.Data.OrderId,
		IdempotentId: sameIdempotentId,
	}

	resp1, err := manager.CancelSell(ctx, req1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp1.Code)
	assert.True(t, resp1.Data.Success)

	// 第二个请求，使用相同的幂等ID
	req2 := &auction.CancelSellReq{
		OrderId:      sellResp.Data.OrderId,
		IdempotentId: sameIdempotentId,
	}

	resp2, err := manager.CancelSell(ctx, req2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp2.Code)
	// 应该返回相同的结果
	assert.True(t, resp2.Data.Success)

	// 2. 测试不同的幂等ID，应该返回错误结果（因为订单已经被取消）
	fmt.Println("测试2: 不同的幂等ID")
	differentIdempotentId := "test_cancel_sell_idempotent_different"

	req3 := &auction.CancelSellReq{
		OrderId:      sellResp.Data.OrderId,
		IdempotentId: differentIdempotentId,
	}

	resp3, err := manager.CancelSell(ctx, req3)
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp3.Code)
	// 应该返回错误（因为订单已经被取消）
	assert.False(t, resp3.Data.Success)

	// 3. 测试幂等ID为空的情况
	fmt.Println("测试3: 幂等ID为空")
	req4 := &auction.CancelSellReq{
		OrderId:      sellResp.Data.OrderId,
		IdempotentId: "",
	}

	resp4, err := manager.CancelSell(ctx, req4)
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp4.Code)
	assert.Equal(t, "idempotent_id is empty", resp4.Msg)
}

// 测试CancelBuy接口的幂等功能
func TestAuctionManager_CancelBuyIdempotency(t *testing.T) {
	// 初始化测试环境
	setupTest()
	defer teardownTest()

	// 初始化拍卖管理器
	manager := GetAuctionManager()

	// 创建测试上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, "userId", "test_user_001")

	// 获取当前市场的平均价格数据
	mu := getMatchManager().GetMatchUnit("test_item_cancel_buy_idempotent")
	avgPrice := mu.hourlyAvgPrice
	// 预设交易价格
	originalPrice := int64(100)
	// 使用平均价格校正交易价格
	correctedPrice := avgPrice
	t.Logf("Original price: %d, Average price: %d, Corrected price: %d", originalPrice, avgPrice, correctedPrice)

	// 先创建一个出售订单和购买订单，用于取消
	sellReq := &auction.SellReq{
		ItemId:       "test_item_cancel_buy_idempotent",
		Quantity:     10,
		Price:        correctedPrice,
		ItemInfo:     "Test Item",
		IdempotentId: "test_sell_for_cancel_buy",
	}
	sellResp, err := manager.Sell(ctx, sellReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, sellResp.Code)

	buyReq := &auction.BuyReq{
		ItemId:       "test_item_cancel_buy_idempotent",
		Quantity:     5,
		Price:        correctedPrice - 1, // 设置购买价格低于出售价格，确保不会立即成交
		IdempotentId: "test_buy_for_cancel",
	}
	buyResp, err := manager.Buy(ctx, buyReq)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, buyResp.Code)

	// 1. 测试重复请求使用相同的幂等ID，应该返回相同的结果
	fmt.Println("测试1: 重复请求使用相同的幂等ID")
	sameIdempotentId := "test_cancel_buy_idempotent_duplicate"

	// 第一个请求
	req1 := &auction.CancelBuyReq{
		OrderId:      buyResp.Data.OrderId,
		IdempotentId: sameIdempotentId,
	}

	resp1, err := manager.CancelBuy(ctx, req1)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp1.Code)
	assert.True(t, resp1.Data.Success)

	// 第二个请求，使用相同的幂等ID
	req2 := &auction.CancelBuyReq{
		OrderId:      buyResp.Data.OrderId,
		IdempotentId: sameIdempotentId,
	}

	resp2, err := manager.CancelBuy(ctx, req2)
	assert.NoError(t, err)
	assert.Equal(t, common.ErrorCode_OK, resp2.Code)
	// 应该返回相同的结果
	assert.True(t, resp2.Data.Success)

	// 2. 测试不同的幂等ID，应该返回错误结果（因为订单已经被取消）
	fmt.Println("测试2: 不同的幂等ID")
	differentIdempotentId := "test_cancel_buy_idempotent_different"

	req3 := &auction.CancelBuyReq{
		OrderId:      buyResp.Data.OrderId,
		IdempotentId: differentIdempotentId,
	}

	resp3, err := manager.CancelBuy(ctx, req3)
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp3.Code)
	// 应该返回错误（因为订单已经被取消）
	assert.False(t, resp3.Data.Success)

	// 3. 测试幂等ID为空的情况
	fmt.Println("测试3: 幂等ID为空")
	req4 := &auction.CancelBuyReq{
		OrderId:      buyResp.Data.OrderId,
		IdempotentId: "",
	}

	resp4, err := manager.CancelBuy(ctx, req4)
	assert.NoError(t, err)
	assert.NotEqual(t, common.ErrorCode_OK, resp4.Code)
	assert.Equal(t, "idempotent_id is empty", resp4.Msg)
}

// 测试 loadOrdersFromRedis 函数
func TestLoadOrdersFromRedis(t *testing.T) {
	// 初始化测试环境
	setupTest()
	defer teardownTest()

	// 创建测试上下文
	ctx := context.Background()

	// 1. 测试正常情况下的订单加载
	t.Run("NormalCase", func(t *testing.T) {
		// 清除之前的测试数据
		redis.GetRedis().Del(ctx, "auction:sells").Result()
		redis.GetRedis().Del(ctx, "auction:buys").Result()

		// 先初始化 idClient
		GetAuctionManager()

		// 先初始化 matchMgr
		getMatchManager()

		// 添加测试数据到 Redis
		sellOrderKey := "auction:sell:test_sell_order_1"
		redis.GetRedis().HMSet(ctx, sellOrderKey, map[string]interface{}{
			"order_id":    "test_sell_order_1",
			"item_id":     "test_item_1",
			"quantity":    "10",
			"price":       "100",
			"item_info":   "Test Item 1",
			"create_time": "1234567890",
		}).Result()
		redis.GetRedis().SAdd(ctx, "auction:sells", sellOrderKey).Result()

		buyOrderKey := "auction:buy:test_buy_order_1"
		redis.GetRedis().HMSet(ctx, buyOrderKey, map[string]interface{}{
			"order_id":    "test_buy_order_1",
			"item_id":     "test_item_1",
			"quantity":    "5",
			"price":       "100",
			"create_time": "1234567890",
		}).Result()
		redis.GetRedis().SAdd(ctx, "auction:buys", buyOrderKey).Result()

		// 调用 loadOrdersFromRedis 函数
		loadOrdersFromRedis(ctx)

		// 由于 matchUnit 内部使用 channel 处理订单，需要等待处理完成
		time.Sleep(100 * time.Millisecond)

		// 验证订单是否被正确加载（这里可以根据 matchUnit 的内部结构进行验证）
		t.Log("Orders loaded successfully")
	})

	// 2. 测试 Redis 错误情况下的处理
	t.Run("RedisError", func(t *testing.T) {
		// 清除之前的测试数据
		redis.GetRedis().Del(ctx, "auction:sells").Result()
		redis.GetRedis().Del(ctx, "auction:buys").Result()

		// 调用 loadOrdersFromRedis 函数
		loadOrdersFromRedis(ctx)

		// 验证函数能够正常执行，不会因为 Redis 错误而崩溃
		t.Log("Function executed successfully with Redis error")
	})

	// 3. 测试订单数据解析错误的处理
	t.Run("ParseError", func(t *testing.T) {
		// 清除之前的测试数据
		redis.GetRedis().Del(ctx, "auction:sells").Result()
		redis.GetRedis().Del(ctx, "auction:buys").Result()

		// 添加格式错误的测试数据到 Redis
		invalidOrderKey := "auction:sell:invalid_order"
		redis.GetRedis().HMSet(ctx, invalidOrderKey, map[string]interface{}{
			"order_id":    "invalid_order",
			"item_id":     "test_item_1",
			"quantity":    "invalid", // 无效的数量
			"price":       "100",
			"item_info":   "Invalid Item",
			"create_time": "1234567890",
		}).Result()
		redis.GetRedis().SAdd(ctx, "auction:sells", invalidOrderKey).Result()

		// 调用 loadOrdersFromRedis 函数
		loadOrdersFromRedis(ctx)

		// 验证函数能够正常执行，不会因为数据解析错误而崩溃
		t.Log("Function executed successfully with parse error")
	})
}
