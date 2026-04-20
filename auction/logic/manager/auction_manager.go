package manager

import (
	"auction_module/kitex_gen/auction"
	"auction_module/kitex_gen/common"
	"auction_module/redis"
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
)

type AuctionManager struct {
}

// 检查分布式锁
func (m *AuctionManager) checkLock(ctx context.Context, userId string) (bool, error) {
	lockKey := "auction:user:lock:" + userId
	lockAcquired, err := redis.GetRedis().SetNX(ctx, lockKey, "1", 5*time.Second).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR] acquire lock error: %s", err.Error())
		return false, err
	}
	return lockAcquired, nil
}

// 释放分布式锁
func (m *AuctionManager) releaseLock(ctx context.Context, userId string) error {
	lockKey := "auction:user:lock:" + userId
	_, err := redis.GetRedis().Del(ctx, lockKey).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR] release lock error: %s", err.Error())
		return err
	}
	return nil
}

// 检查幂等性，返回已存在的结果或插入"正在处理"状态
func (m *AuctionManager) checkIdempotency(ctx context.Context, idempotentKey string, prefix string) (map[string]interface{}, error) {
	// 使用Lua脚本实现原子操作：检查是否存在幂等键，如果不存在则插入"正在处理"状态
	luaScript := `
		local key = KEYS[1]
		local exists = redis.call('EXISTS', key)
		if exists == 1 then
			local result = redis.call('HGETALL', key)
			return result
		else
			redis.call('HMSET', key, 'code', 1306, 'msg', 'processing', 'timestamp', ARGV[1])
			redis.call('EXPIRE', key, 2592000) -- 1个月过期
			return {}
		end
	`

	result, luaErr := redis.GetRedis().Eval(ctx, luaScript, []string{idempotentKey}, time.Now().Unix()).Result()
	if luaErr != nil {
		klog.CtxErrorf(ctx, "%s Lua script error: %s", prefix, luaErr.Error())
		return nil, luaErr
	}

	// 解析Lua脚本返回结果
	resultMap := make(map[string]interface{})
	if resultSlice, ok := result.([]interface{}); ok && len(resultSlice) > 0 {
		// 将 []interface{} 转换为 map[string]interface{}
		for i := 0; i < len(resultSlice); i += 2 {
			if i+1 < len(resultSlice) {
				key, ok := resultSlice[i].(string)
				if ok {
					resultMap[key] = resultSlice[i+1]
				}
			}
		}
	}

	return resultMap, nil
}

var (
	auctionManager *AuctionManager
	once           sync.Once
	idClient       *snowflake.Node
	maxSellCount   int32 = 8
	maxBuyCount    int32 = 8
)

func GetAuctionManager() *AuctionManager {
	once.Do(func() {
		// 使用Redis获取雪花算法节点ID（借鉴match_server实现）
		key := "auction_server:snowflake:node"
		n, err := redis.GetRedis().Incr(context.Background(), key).Result()
		if err != nil {
			klog.Fatal("[AUCTION-MGR-INIT] AuctionManager: gen uuid creator err: %v", err)
		}

		nodeIdx := n % (1 << snowflake.NodeBits)
		if node, err := snowflake.NewNode(nodeIdx); err != nil {
			klog.Fatal("[AUCTION-MGR-NODE] AuctionManager: gen uuid creator err: %v", err)
		} else {
			klog.Infof("[AUCTION-MGR-NODE-OK] AuctionManager: gen uuid creator success, node: %d", nodeIdx)
			idClient = node
		}

		getMatchManager()
		auctionManager = &AuctionManager{}

	})
	return auctionManager
}

func (m *AuctionManager) Ping(ctx context.Context, req *auction.PingReq) (resp *auction.PingRsp, err error) {
	resp = &auction.PingRsp{
		Code: common.ErrorCode_OK,
		Msg:  "pong: " + req.GetMessage(),
	}
	return
}

// 出售协议 - 将SellData的各个属性直接存入Redis hash，order_id使用雪花算法
func (m *AuctionManager) Sell(ctx context.Context, req *auction.SellReq) (resp *auction.SellRsp, err error) {
	// 获取用户ID
	userId := ""
	var mu *matchUnit
	var userSellsKey string
	if val, ok := ctx.Value("userId").(string); ok {
		userId = val
	}

	// 初始化resp，确保defer中不需要判断是否为空
	resp = &auction.SellRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
	}

	// 获取分布式锁（5秒超时）
	lockAcquired, err := m.checkLock(ctx, userId)
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] acquire lock error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "acquire lock error"
		return
	}
	if !lockAcquired {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "user is busy"
		klog.CtxInfof(ctx, "[AUCTION-MGR-SELL] User %s is busy, cannot sell", userId)
		return
	}
	// 释放锁
	defer func() {
		_ = m.releaseLock(ctx, userId)
	}()

	// 幂等性检查
	idempotentId := req.GetIdempotentId()
	if idempotentId == "" {
		resp.Msg = "idempotent_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-SELL] User %s idempotent_id is empty, cannot sell", userId)
		return
	}

	// 构建幂等性检查的Redis键
	idempotentKey := "auction:idempotent:" + userId + ":" + idempotentId
	klog.CtxInfof(ctx, "[AUCTION-MGR-SELL] Building idempotent key: %s", idempotentKey)

	// 检查幂等性
	resultMap, err := m.checkIdempotency(ctx, idempotentKey, "[AUCTION-MGR-SELL]")
	if err != nil {
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "check idempotent error"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] Check idempotent error: %s", err.Error())
		return
	}

	if len(resultMap) > 0 {
		// 存在之前的处理结果，直接返回
		code, _ := strconv.ParseInt(fmt.Sprintf("%v", resultMap["code"]), 10, 32)
		resp.Code = common.ErrorCode(code)
		resp.Msg = fmt.Sprintf("%v", resultMap["msg"])

		// 如果是成功的结果，解析SellData
		if resp.Code == common.ErrorCode_OK {
			sellData := &auction.SellData{
				OrderId:    fmt.Sprintf("%v", resultMap["order_id"]),
				ItemId:     fmt.Sprintf("%v", resultMap["item_id"]),
				Quantity:   int32(parseInt(fmt.Sprintf("%v", resultMap["quantity"]))),
				Price:      parseInt64(fmt.Sprintf("%v", resultMap["price"])),
				ItemInfo:   fmt.Sprintf("%v", resultMap["item_info"]),
				CreateTime: parseInt64(fmt.Sprintf("%v", resultMap["create_time"])),
			}
			resp.Data = sellData
		}

		klog.CtxWarnf(ctx, "[AUCTION-MGR-SELL] Idempotent request detected, returning cached result: idempotentId=%s", idempotentId)
		return
	}

	defer func() {
		orderId := ""
		if resp.GetData() != nil {
			orderId = resp.GetData().GetOrderId()
		}
		klog.CtxInfof(ctx, "[AUCTION-SELL-RESULT] userId: %s, orderId: %s, resp: %d", userId, orderId, resp.GetCode())

		// 统一处理幂等性结果存储到Redis
		data := map[string]interface{}{
			"code":      int(resp.Code),
			"msg":       resp.Msg,
			"timestamp": time.Now().Unix(),
		}

		// 如果响应成功，添加订单信息
		if resp.Code == common.ErrorCode_OK && resp.Data != nil {
			sellData := resp.Data
			data["order_id"] = sellData.OrderId
			data["item_id"] = sellData.ItemId
			data["quantity"] = sellData.Quantity
			data["price"] = sellData.Price
			data["item_info"] = sellData.ItemInfo
			data["create_time"] = sellData.CreateTime
		}

		hmsetErr := redis.GetRedis().HMSet(ctx, idempotentKey, data).Err()
		if hmsetErr != nil {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] Store idempotent result error: %s", hmsetErr.Error())
			// 存储失败不影响主流程，只记录错误
		}

		// 设置过期时间，防止Redis数据过多（1个月）
		expireErr, _ := redis.GetRedis().Expire(ctx, idempotentKey, 30*24*time.Hour).Result()
		if !expireErr {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] Set idempotent key expire error")
		}
	}()

	// 参数检查
	if req.GetItemId() == "" {
		resp.Msg = "item_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-SELL] User %s item_id is empty, cannot sell", userId)
		return
	}
	if req.GetQuantity() <= 0 {
		resp.Msg = "quantity must be greater than 0"
		klog.CtxInfof(ctx, "[AUCTION-MGR-SELL] User %s quantity must be greater than 0, cannot sell", userId)
		return
	}
	if req.GetPrice() <= 0 {
		resp.Msg = "price must be greater than 0"
		klog.CtxInfof(ctx, "[AUCTION-MGR-SELL] User %s price must be greater than 0, cannot sell", userId)
		return
	}
	if userId == "" {
		resp.Code = common.ErrorCode_AUCTION_USER_NOT_FOUND
		resp.Msg = "user_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-SELL] User %s user_id is empty, cannot sell", userId)
		return
	}

	// 销售限制检查：确保单个用户在拍卖系统中最多只能销售maxSellCount单商品
	userSellsKey = "user:" + userId + ":sells"
	sellCount, err := redis.GetRedis().SCard(ctx, userSellsKey).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] Get user sells count error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get user sells count error"
		return
	}
	if int32(sellCount) >= maxSellCount {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] Sell count exceeds limit, userId: %s, currentCount: %d, maxLimit: %d", userId, sellCount, maxSellCount)
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "sell count exceeds limit"
		return
	}

	// 价格限制检查：所有买卖价格必须控制在matchunit.hourlyAvgPrice的±10%范围内
	mu = getMatchManager().GetMatchUnit(req.GetItemId())
	avgPrice := mu.hourlyAvgPrice
	minPrice := int64(float64(avgPrice) * 0.9)
	maxPrice := int64(float64(avgPrice) * 1.1)
	if req.GetPrice() < minPrice || req.GetPrice() > maxPrice {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] Price exceeds limit, userId: %s, itemId: %s, price: %d, minPrice: %d, maxPrice: %d, avgPrice: %d", userId, req.GetItemId(), req.GetPrice(), minPrice, maxPrice, avgPrice)
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "price exceeds limit"
		return
	}

	// 使用雪花算法生成order_id
	orderId := idClient.Generate().String()

	// 创建SellData结构体
	sellData := &auction.SellData{
		OrderId:    orderId,
		ItemId:     req.GetItemId(),
		Quantity:   req.GetQuantity(),
		Price:      req.GetPrice(),
		ItemInfo:   req.GetItemInfo(),
		CreateTime: time.Now().Unix(),
	}

	// 计算各种key
	key := "auction:sell:" + orderId
	statusKey := "auction:order:" + orderId + ":status"
	userTimeKey := "user:" + userId + ":transactions:time"
	userSellsKey = "user:" + userId + ":sells"

	// 使用Lua脚本将数据存入Redis：
	// 1. 将订单信息存入 auction:sell:{orderId}（包含order_id字段）
	// 2. 将订单ID添加到用户的出售列表中
	// 3. 将订单ID添加到全局出售列表中
	// 4. 记录订单状态到新的Redis结构
	var luaScript string
	luaScript = ` 
		-- 将订单信息存入hash表（包含order_id字段）
		redis.call('HMSET', KEYS[1], 
			'order_id', ARGV[7],
			'item_id', ARGV[1],
			'quantity', ARGV[2],
			'price', ARGV[3],
			'item_info', ARGV[4],
			'create_time', ARGV[5],
			'user_id', ARGV[6]
		)
		
		-- 将订单ID添加到用户的出售列表
		redis.call('SADD', ARGV[8], KEYS[1])
		
		-- 将订单ID添加到全局出售列表
		redis.call('SADD', 'auction:sells', KEYS[1])
		
		-- 记录订单状态到新的Redis结构
		redis.call('HMSET', KEYS[2], 
			'order_id', ARGV[7],
			'trade_direction', 'sell',
			'status', '卖',
			'final_price', 0,
			'final_quantity', ARGV[2],
			'item_id', ARGV[1],
			'tax', 0,
			'create_time', ARGV[5],
			'user_id', ARGV[6]
		)
		
		-- 添加到用户交易时间排序集合（按用户维度的时间排序）
		redis.call('ZADD', KEYS[3], ARGV[5], ARGV[7])
		
		return 1
	`

	if _, err = redis.GetRedis().Eval(ctx, luaScript, []string{key, statusKey, userTimeKey},
		sellData.ItemId,
		sellData.Quantity,
		sellData.Price,
		sellData.ItemInfo,
		sellData.CreateTime,
		userId,
		orderId,
		userSellsKey,
	).Result(); err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-SELL] redis eval error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "save to redis error"
		return
	}

	mu = getMatchManager().GetMatchUnit(sellData.ItemId)
	r := make(chan bool, 0)
	mu.opChannel <- func() {
		mu.AddSellOrder(ctx, sellData)
		r <- true
	}
	<-r
	// 更新响应为成功状态
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	// 响应数据 在往mu投递以后 之前的sellData可能会发生变化
	resp.Data = &auction.SellData{
		OrderId:    orderId,
		ItemId:     req.GetItemId(),
		Quantity:   req.GetQuantity(),
		Price:      req.GetPrice(),
		ItemInfo:   req.GetItemInfo(),
		CreateTime: sellData.CreateTime,
	}

	return
}

// 求购协议 - 将BuyData的各个属性直接存入Redis hash，order_id使用雪花算法
func (m *AuctionManager) Buy(ctx context.Context, req *auction.BuyReq) (resp *auction.BuyRsp, err error) {
	// 获取用户ID
	userId := ""
	var mu *matchUnit
	var userBuysKey string
	if val, ok := ctx.Value("userId").(string); ok {
		userId = val
	}

	// 初始化resp，确保defer中不需要判断是否为空
	resp = &auction.BuyRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
	}

	// 获取分布式锁（5秒超时）
	lockAcquired, err := m.checkLock(ctx, userId)
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] acquire lock error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "acquire lock error"
		return
	}
	if !lockAcquired {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "user is busy"
		klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] User %s is busy, cannot buy", userId)
		return
	}
	// 释放锁
	defer func() {
		_ = m.releaseLock(ctx, userId)
	}()

	// 幂等性检查
	idempotentId := req.GetIdempotentId()
	if idempotentId == "" {
		resp.Msg = "idempotent_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] User %s idempotent_id is empty, cannot buy", userId)
		return
	}

	// 构建幂等性检查的Redis键
	idempotentKey := "auction:idempotent:" + userId + ":" + idempotentId
	klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] Building idempotent key: %s", idempotentKey)

	// 检查幂等性
	resultMap, err := m.checkIdempotency(ctx, idempotentKey, "[AUCTION-MGR-BUY]")
	if err != nil {
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "check idempotent error"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] Check idempotent error: %s", err.Error())
		return
	}

	if len(resultMap) > 0 {
		// 存在之前的处理结果，直接返回
		code, _ := strconv.ParseInt(fmt.Sprintf("%v", resultMap["code"]), 10, 32)
		resp.Code = common.ErrorCode(code)
		resp.Msg = fmt.Sprintf("%v", resultMap["msg"])

		// 如果是成功的结果，解析BuyData
		if resp.Code == common.ErrorCode_OK {
			buyData := &auction.BuyData{
				OrderId:    fmt.Sprintf("%v", resultMap["order_id"]),
				ItemId:     fmt.Sprintf("%v", resultMap["item_id"]),
				Quantity:   int32(parseInt(fmt.Sprintf("%v", resultMap["quantity"]))),
				Price:      parseInt64(fmt.Sprintf("%v", resultMap["price"])),
				CreateTime: parseInt64(fmt.Sprintf("%v", resultMap["create_time"])),
			}
			resp.Data = buyData
		}

		klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] Idempotent request detected, returning cached result: idempotentId=%s", idempotentId)
		return
	}

	defer func() {
		orderId := ""
		if resp.GetData() != nil {
			orderId = resp.GetData().GetOrderId()
		}
		klog.CtxInfof(ctx, "[AUCTION-BUY-RESULT] userId: %s, orderId: %s, resp: %d", userId, orderId, resp.GetCode())

		// 统一处理幂等性结果存储到Redis
		data := map[string]interface{}{
			"code":      int(resp.Code),
			"msg":       resp.Msg,
			"timestamp": time.Now().Unix(),
		}

		// 如果响应成功，添加订单信息
		if resp.Code == common.ErrorCode_OK && resp.Data != nil {
			buyData := resp.Data
			data["order_id"] = buyData.OrderId
			data["item_id"] = buyData.ItemId
			data["quantity"] = buyData.Quantity
			data["price"] = buyData.Price
			data["create_time"] = buyData.CreateTime
		}

		if hmsetErr := redis.GetRedis().HMSet(ctx, idempotentKey, data).Err(); hmsetErr != nil {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] Store idempotent result error: %s", hmsetErr.Error())
			// 存储失败不影响主流程，只记录错误
		}

		// 设置过期时间，防止Redis数据过多（1个月）
		if expireErr, _ := redis.GetRedis().Expire(ctx, idempotentKey, 30*24*time.Hour).Result(); !expireErr {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] Set idempotent key expire error")
		}
	}()

	// 参数检查
	if req.GetItemId() == "" {
		resp.Msg = "item_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] User %s item_id is empty, cannot buy", userId)
		return
	}
	if req.GetQuantity() <= 0 {
		resp.Msg = "quantity must be greater than 0"
		klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] User %s quantity must be greater than 0, cannot buy", userId)
		return
	}
	if req.GetPrice() <= 0 {
		resp.Msg = "price must be greater than 0"
		klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] User %s price must be greater than 0, cannot buy", userId)
		return
	}
	if userId == "" {
		resp.Code = common.ErrorCode_AUCTION_USER_NOT_FOUND
		resp.Msg = "user_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-BUY] User %s user_id is empty, cannot buy", userId)
		return
	}

	// 购买限制检查：确保单个用户在拍卖系统中最多只能购买maxBuyCount单商品
	userBuysKey = "user:" + userId + ":buys"
	if buyCount, err2 := redis.GetRedis().SCard(ctx, userBuysKey).Result(); err2 != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] Get user buys count error: %s", err2.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get user buys count error"
		return
	} else if int32(buyCount) >= maxBuyCount {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] Buy count exceeds limit, userId: %s, currentCount: %d, maxLimit: %d", userId, buyCount, maxBuyCount)
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "buy count exceeds limit"
		return
	}

	// 价格限制检查：所有买卖价格必须控制在matchunit.hourlyAvgPrice的±10%范围内
	mu = getMatchManager().GetMatchUnit(req.GetItemId())
	avgPrice := mu.hourlyAvgPrice
	minPrice := int64(float64(avgPrice) * 0.9)
	maxPrice := int64(float64(avgPrice) * 1.1)
	if req.GetPrice() < minPrice || req.GetPrice() > maxPrice {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] Price exceeds limit, userId: %s, itemId: %s, price: %d, minPrice: %d, maxPrice: %d, avgPrice: %d", userId, req.GetItemId(), req.GetPrice(), minPrice, maxPrice, avgPrice)
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "price exceeds limit"
		return
	}

	// 使用雪花算法生成order_id
	orderId := idClient.Generate().String()

	// 创建BuyData结构体
	buyData := &auction.BuyData{
		OrderId:    orderId,
		ItemId:     req.GetItemId(),
		Quantity:   req.GetQuantity(),
		Price:      req.GetPrice(),
		CreateTime: time.Now().Unix(),
	}

	// 计算各种key
	key := "auction:buy:" + orderId
	statusKey := "auction:order:" + orderId + ":status"
	userTimeKey := "user:" + userId + ":transactions:time"
	userBuysKey = "user:" + userId + ":buys"

	// 使用Lua脚本将数据存入Redis：
	// 1. 将订单信息存入 auction:buy:{orderId}（包含order_id字段）
	// 2. 将订单ID添加到用户的求购列表中
	// 3. 将订单ID添加到全局求购列表中
	// 4. 记录订单状态到新的Redis结构
	var luaScript string
	luaScript = ` 
		-- 将订单信息存入hash表（包含order_id字段）
		redis.call('HMSET', KEYS[1], 
			'order_id', ARGV[6],
			'item_id', ARGV[1],
			'quantity', ARGV[2],
			'price', ARGV[3],
			'create_time', ARGV[4],
			'user_id', ARGV[5]
		)
		
		-- 将订单ID添加到用户的求购列表
		redis.call('SADD', ARGV[7], KEYS[1])
		
		-- 将订单ID添加到全局求购列表
		redis.call('SADD', 'auction:buys', KEYS[1])
		
		-- 记录订单状态到新的Redis结构
		redis.call('HMSET', KEYS[2], 
			'order_id', ARGV[6],
			'trade_direction', 'buy',
			'status', '买',
			'final_price', 0,
			'final_quantity', ARGV[2],
			'item_id', ARGV[1],
			'tax', 0,
			'create_time', ARGV[4],
			'user_id', ARGV[5]
		)
		
		-- 添加到用户交易时间排序集合（按用户维度的时间排序）
		redis.call('ZADD', KEYS[3], ARGV[4], ARGV[6])
		
		return 1
	`

	if _, err = redis.GetRedis().Eval(ctx, luaScript, []string{key, statusKey, userTimeKey},
		buyData.ItemId,
		buyData.Quantity,
		buyData.Price,
		buyData.CreateTime,
		userId,
		orderId,
		userBuysKey,
	).Result(); err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-BUY] redis eval error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "save to redis error"
		return
	}

	mu = getMatchManager().GetMatchUnit(buyData.ItemId)
	r := make(chan bool, 0)
	mu.opChannel <- func() {
		mu.AddBuyOrder(ctx, buyData)
		r <- true
	}
	<-r
	// 更新响应为成功状态
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	// 响应数据 在往mu投递以后 之前的buyData可能会发生变化
	resp.Data = &auction.BuyData{
		OrderId:    orderId,
		ItemId:     req.GetItemId(),
		Quantity:   req.GetQuantity(),
		Price:      req.GetPrice(),
		CreateTime: buyData.CreateTime,
	}

	return
}

// 取消出售协议 - 原子性地删除Redis中的卖单数据
func (m *AuctionManager) CancelSell(ctx context.Context, req *auction.CancelSellReq) (resp *auction.CancelSellRsp, err error) {
	// 获取用户ID
	userId := ""
	if val, ok := ctx.Value("userId").(string); ok {
		userId = val
	}

	// 初始化resp，确保defer中不需要判断是否为空
	resp = &auction.CancelSellRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
		Data: &auction.CancelSellData{
			Success: false,
		},
	}

	// 获取分布式锁（5秒超时）
	lockAcquired, err := m.checkLock(ctx, userId)
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-SELL] acquire lock error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "acquire lock error"
		return
	}
	if !lockAcquired {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "user is busy"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] User %s is busy, cannot cancel sell", userId)
		return
	}
	// 释放锁
	defer func() {
		_ = m.releaseLock(ctx, userId)
	}()

	// 幂等性检查
	idempotentId := req.GetIdempotentId()
	if idempotentId == "" {
		resp.Msg = "idempotent_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] User %s idempotent_id is empty, cannot cancel sell", userId)
		return
	}

	// 构建幂等性检查的Redis键
	idempotentKey := "auction:idempotent:" + userId + ":" + idempotentId
	klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] Building idempotent key: %s", idempotentKey)

	// 检查幂等性
	resultMap, err := m.checkIdempotency(ctx, idempotentKey, "[AUCTION-MGR-CANCEL-SELL]")
	if err != nil {
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "check idempotent error"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-SELL] Check idempotent error: %s", err.Error())
		return
	}

	if len(resultMap) > 0 {
		// 存在之前的处理结果，直接返回
		code, _ := strconv.ParseInt(fmt.Sprintf("%v", resultMap["code"]), 10, 32)
		resp.Code = common.ErrorCode(code)
		resp.Msg = fmt.Sprintf("%v", resultMap["msg"])

		// 解析success字段
		success := false
		successStr := fmt.Sprintf("%v", resultMap["success"])
		if successStr == "true" || successStr == "1" {
			success = true
		}
		resp.Data.Success = success

		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] Idempotent request detected, returning cached result: idempotentId=%s", idempotentId)
		return
	}

	defer func() {
		klog.CtxInfof(ctx, "[AUCTION-CANCEL-SELL-RESULT] userId: %s, orderId: %s, resp: %d", userId, req.GetOrderId(), resp.GetCode())

		// 统一处理幂等性结果存储到Redis
		data := map[string]interface{}{
			"code":      int(resp.Code),
			"msg":       resp.Msg,
			"success":   resp.Data.Success,
			"order_id":  req.GetOrderId(),
			"timestamp": time.Now().Unix(),
		}

		hmsetErr := redis.GetRedis().HMSet(ctx, idempotentKey, data).Err()
		if hmsetErr != nil {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-SELL] Store idempotent result error: %s", hmsetErr.Error())
			// 存储失败不影响主流程，只记录错误
		}

		// 设置过期时间，防止Redis数据过多（1个月）
		expireErr, _ := redis.GetRedis().Expire(ctx, idempotentKey, 30*24*time.Hour).Result()
		if !expireErr {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-SELL] Set idempotent key expire error")
		}
	}()

	// 参数检查
	if req.GetOrderId() == "" {
		resp.Msg = "order_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] User %s order_id is empty, cannot cancel sell", userId)
		return
	}
	if userId == "" {
		resp.Code = common.ErrorCode_AUCTION_USER_NOT_FOUND
		resp.Msg = "user_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] User %s user_id is empty, cannot cancel sell", userId)
		return
	}

	// 先查询订单信息，获取道具ID
	orderKey := "auction:sell:" + req.GetOrderId()

	// 先检查订单是否存在
	exists, err := redis.GetRedis().Exists(ctx, orderKey).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] check order exists error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "check order exists error"
		return
	}

	if exists == 0 {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "order not found"
		return
	}

	// 获取物品ID
	itemId, err := redis.GetRedis().HGet(ctx, orderKey, "item_id").Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] get item_id error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get item_id error"
		return
	}

	// 从matchUnit中移除订单（通过opChannel保证线程安全）
	mu := getMatchManager().GetMatchUnit(itemId)
	r := make(chan bool, 0)
	mu.opChannel <- func() {
		if mu.RemoveSellOrder(ctx, req.GetOrderId()) {
			klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] remove sell order success: %s", req.GetOrderId())
		}
		r <- true
	}

	<-r
	// 使用Lua脚本原子性地删除Redis中的卖单数据
	var luaScript string
	var result interface{}
	luaScript = ` 
		-- 参数：orderId, userId
		local orderId = ARGV[1]
		local userId = ARGV[2]
		local orderKey = 'auction:sell:' .. orderId
		
		-- 检查订单是否存在
		local exists = redis.call('EXISTS', orderKey)
		if exists == 0 then
			return {'err', 'order_not_found'}
		end
		
		-- 检查订单是否属于该用户
		local orderUserId = redis.call('HGET', orderKey, 'user_id')
		if orderUserId ~= userId then
			return {'err', 'order_not_belong_to_user'}
		end
		
		-- 获取订单信息
		local itemId = redis.call('HGET', orderKey, 'item_id')
		local createTime = redis.call('HGET', orderKey, 'create_time')
		
		-- 更新订单状态到新的Redis结构
		local statusKey = 'auction:order:' .. orderId .. ':status'
		redis.call('HMSET', statusKey, 
			'order_id', orderId,
			'trade_direction', 'sell',
			'status', '取消',
			'final_price', 0,
			'final_quantity', 0,
			'item_id', itemId,
			'tax', 0,
			'create_time', createTime,
			'user_id', userId
		)
		
		-- 删除订单
		redis.call('DEL', orderKey)
		
		-- 从用户出售列表中移除
		redis.call('SREM', 'user:' .. userId .. ':sells', orderKey)
		
		-- 从全局出售列表中移除
		redis.call('SREM', 'auction:sells', orderKey)
		
		return {'success', 'true'}
	`

	// 执行Lua脚本
	result, err = redis.GetRedis().Eval(ctx, luaScript, []string{},
		req.GetOrderId(),
		userId,
	).Result()

	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-SELL] redis eval error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "cancel sell order error"
		return
	}

	// 获取结果，Lua脚本返回的是[]interface{}
	resultSlice, ok := result.([]interface{})
	klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-SELL] result: %+v %+v %v", result, resultSlice, ok)

	// 检查结果类型
	if !ok {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-SELL] invalid result format")
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "invalid result format"
		return
	}

	// 处理[]interface{}类型的结果
	// 检查是否有错误信息
	for i := 0; i < len(resultSlice); i += 2 {
		if i+1 < len(resultSlice) {
			key, ok1 := resultSlice[i].(string)
			value, ok2 := resultSlice[i+1].(string)
			if ok1 && ok2 && key == "err" {
				klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-SELL] cancel sell error: %s", value)
				resp.Msg = value
				return
			}
		}
	}

	// 更新响应为成功状态
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	resp.Data.Success = true

	return
}

// 取消求购协议 - 原子性地删除Redis中的买单数据
func (m *AuctionManager) CancelBuy(ctx context.Context, req *auction.CancelBuyReq) (resp *auction.CancelBuyRsp, err error) {
	// 获取用户ID
	userId := ""
	if val, ok := ctx.Value("userId").(string); ok {
		userId = val
	}

	// 初始化resp，确保defer中不需要判断是否为空
	resp = &auction.CancelBuyRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
		Data: &auction.CancelBuyData{
			Success: false,
		},
	}

	// 获取分布式锁（5秒超时）
	lockAcquired, err := m.checkLock(ctx, userId)
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] acquire lock error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "acquire lock error"
		return
	}
	if !lockAcquired {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "user is busy"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] User %s is busy, cannot cancel buy", userId)
		return
	}
	// 释放锁
	defer func() {
		_ = m.releaseLock(ctx, userId)
	}()

	// 幂等性检查
	idempotentId := req.GetIdempotentId()
	if idempotentId == "" {
		resp.Msg = "idempotent_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] User %s idempotent_id is empty, cannot cancel buy", userId)
		return
	}

	// 构建幂等性检查的Redis键
	idempotentKey := "auction:idempotent:" + userId + ":" + idempotentId
	klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] Building idempotent key: %s", idempotentKey)

	// 检查幂等性
	resultMap, err := m.checkIdempotency(ctx, idempotentKey, "[AUCTION-MGR-CANCEL-BUY]")
	if err != nil {
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "check idempotent error"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] Check idempotent error: %s", err.Error())
		return
	}

	if len(resultMap) > 0 {
		// 存在之前的处理结果，直接返回
		code, _ := strconv.ParseInt(fmt.Sprintf("%v", resultMap["code"]), 10, 32)
		resp.Code = common.ErrorCode(code)
		resp.Msg = fmt.Sprintf("%v", resultMap["msg"])

		// 解析success字段
		success := false
		successStr := fmt.Sprintf("%v", resultMap["success"])
		if successStr == "true" || successStr == "1" {
			success = true
		}
		resp.Data.Success = success

		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] Idempotent request detected, returning cached result: idempotentId=%s", idempotentId)
		return
	}

	defer func() {
		klog.CtxInfof(ctx, "[AUCTION-CANCEL-BUY-RESULT] userId: %s, orderId: %s, resp: %d", userId, req.GetOrderId(), resp.GetCode())

		// 统一处理幂等性结果存储到Redis
		data := map[string]interface{}{
			"code":      int(resp.Code),
			"msg":       resp.Msg,
			"success":   resp.Data.Success,
			"order_id":  req.GetOrderId(),
			"timestamp": time.Now().Unix(),
		}

		hmsetErr := redis.GetRedis().HMSet(ctx, idempotentKey, data).Err()
		if hmsetErr != nil {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] Store idempotent result error: %s", hmsetErr.Error())
			// 存储失败不影响主流程，只记录错误
		}

		// 设置过期时间，防止Redis数据过多（1个月）
		expireErr, _ := redis.GetRedis().Expire(ctx, idempotentKey, 30*24*time.Hour).Result()
		if !expireErr {
			klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] Set idempotent key expire error")
		}
	}()

	// 参数检查
	if req.GetOrderId() == "" {
		resp.Msg = "order_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] User %s order_id is empty, cannot cancel buy", userId)
		return
	}
	if userId == "" {
		resp.Code = common.ErrorCode_AUCTION_USER_NOT_FOUND
		resp.Msg = "user_id is empty"
		klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] User %s user_id is empty, cannot cancel buy", userId)
		return
	}

	// 先查询订单信息，获取道具ID
	orderKey := "auction:buy:" + req.GetOrderId()

	// 先检查订单是否存在
	exists, err := redis.GetRedis().Exists(ctx, orderKey).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] check order exists error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "check order exists error"
		return
	}

	if exists == 0 {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "order not found"
		return
	}

	// 获取物品ID
	itemId, err := redis.GetRedis().HGet(ctx, orderKey, "item_id").Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] get item_id error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get item_id error"
		return
	}

	// 从matchUnit中移除订单（通过opChannel保证线程安全）
	mu := getMatchManager().GetMatchUnit(itemId)
	r := make(chan bool, 0)
	mu.opChannel <- func() {
		if mu.RemoveBuyOrder(ctx, req.GetOrderId()) {
			klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] remove buy order success: %s", req.GetOrderId())
		}
		r <- true
	}

	<-r
	// 使用Lua脚本原子性地删除Redis中的买单数据
	var luaScript string
	var result interface{}
	luaScript = ` 
		-- 参数：orderId, userId
		local orderId = ARGV[1]
		local userId = ARGV[2]
		local orderKey = 'auction:buy:' .. orderId
		
		-- 检查订单是否存在
		local exists = redis.call('EXISTS', orderKey)
		if exists == 0 then
			return {'err', 'order_not_found'}
		end
		
		-- 检查订单是否属于该用户
		local orderUserId = redis.call('HGET', orderKey, 'user_id')
		if orderUserId ~= userId then
			return {'err', 'order_not_belong_to_user'}
		end
		
		-- 获取订单信息
		local itemId = redis.call('HGET', orderKey, 'item_id')
		local createTime = redis.call('HGET', orderKey, 'create_time')
		
		-- 更新订单状态到新的Redis结构
		local statusKey = 'auction:order:' .. orderId .. ':status'
		redis.call('HMSET', statusKey, 
			'order_id', orderId,
			'trade_direction', 'buy',
			'status', '取消',
			'final_price', 0,
			'final_quantity', 0,
			'item_id', itemId,
			'tax', 0,
			'create_time', createTime,
			'user_id', userId
		)
		
		-- 删除订单
		redis.call('DEL', orderKey)
		
		-- 从用户求购列表中移除
		redis.call('SREM', 'user:' .. userId .. ':buys', orderKey)
		
		-- 从全局求购列表中移除
		redis.call('SREM', 'auction:buys', orderKey)
		
		return {'success', 'true'}
	`

	// 执行Lua脚本
	result, err = redis.GetRedis().Eval(ctx, luaScript, []string{},
		req.GetOrderId(),
		userId,
	).Result()

	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] redis eval error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "cancel buy order error"
		return
	}

	// 获取结果，Lua脚本返回的是[]interface{}
	resultSlice, ok := result.([]interface{})
	klog.CtxInfof(ctx, "[AUCTION-MGR-CANCEL-BUY] result: %+v %+v %v", result, resultSlice, ok)

	// 检查结果类型
	if !ok {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] invalid result format")
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "invalid result format"
		return
	}

	// 处理[]interface{}类型的结果
	// 检查是否有错误信息
	for i := 0; i < len(resultSlice); i += 2 {
		if i+1 < len(resultSlice) {
			key, ok1 := resultSlice[i].(string)
			value, ok2 := resultSlice[i+1].(string)
			if ok1 && ok2 && key == "err" {
				klog.CtxErrorf(ctx, "[AUCTION-MGR-CANCEL-BUY] cancel buy error: %s", value)
				resp.Msg = value
				return
			}
		}
	}

	// 更新响应为成功状态
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	resp.Data.Success = true

	return
}

// 查看自己所有出售道具协议
func (m *AuctionManager) GetMySells(ctx context.Context, req *auction.GetMySellsReq) (resp *auction.GetMySellsRsp, err error) {
	// 初始化resp
	resp = &auction.GetMySellsRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
		Data: []*auction.SellData{},
	}

	// 获取用户ID
	userId := ""
	if val, ok := ctx.Value("userId").(string); ok {
		userId = val
	}

	// 检查用户ID
	if userId == "" {
		resp.Code = common.ErrorCode_AUCTION_USER_NOT_FOUND
		resp.Msg = "user_id is empty"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-MY-SELLS] User ID is empty")
		return
	}

	// 使用Lua脚本获取用户的所有出售道具
	luaScript := `
		local userSellsKey = 'user:' .. ARGV[1] .. ':sells'
		local sellOrderKeys = redis.call('SMEMBERS', userSellsKey)
		local result = {}
		
		for _, key in ipairs(sellOrderKeys) do
			local sellData = redis.call('HGETALL', key)
			if sellData then
				table.insert(result, sellData)
			end
		end
		
		return result
	`

	// 执行Lua脚本
	result, err := redis.GetRedis().Eval(ctx, luaScript, []string{}, userId).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-MY-SELLS] Execute Lua script error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get user sells error"
		return
	}

	// 处理Lua脚本返回的结果
	sellDataList := make([]*auction.SellData, 0)
	if resultList, ok := result.([]interface{}); ok {
		for _, item := range resultList {
			if sellDataMap, ok := item.([]interface{}); ok {
				// 将[]interface{}转换为map[string]string
				dataMap := make(map[string]string)
				for i := 0; i < len(sellDataMap); i += 2 {
					if key, ok := sellDataMap[i].(string); ok {
						if value, ok := sellDataMap[i+1].(string); ok {
							dataMap[key] = value
						}
					}
				}

				// 构造SellData
				sellData := &auction.SellData{
					OrderId:  dataMap["order_id"],
					ItemId:   dataMap["item_id"],
					ItemInfo: dataMap["item_info"],
				}

				// 解析数值字段
				if quantity, err := strconv.ParseInt(dataMap["quantity"], 10, 32); err == nil {
					sellData.Quantity = int32(quantity)
				}
				if price, err := strconv.ParseInt(dataMap["price"], 10, 64); err == nil {
					sellData.Price = price
				}
				if createTime, err := strconv.ParseInt(dataMap["create_time"], 10, 64); err == nil {
					sellData.CreateTime = createTime
				}

				// 添加到列表
				sellDataList = append(sellDataList, sellData)
			}
		}
	}

	// 更新响应
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	resp.Data = sellDataList
	klog.CtxInfof(ctx, "[AUCTION-MGR-GET-MY-SELLS] Get my sells success, userId: %s, count: %d", userId, len(sellDataList))
	return
}

// 查看自己所有求购道具协议
func (m *AuctionManager) GetMyBuys(ctx context.Context, req *auction.GetMyBuysReq) (resp *auction.GetMyBuysRsp, err error) {
	// 初始化resp
	resp = &auction.GetMyBuysRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
		Data: []*auction.BuyData{},
	}

	// 获取用户ID
	userId := ""
	if val, ok := ctx.Value("userId").(string); ok {
		userId = val
	}

	// 检查用户ID
	if userId == "" {
		resp.Code = common.ErrorCode_AUCTION_USER_NOT_FOUND
		resp.Msg = "user_id is empty"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-MY-BUYS] User ID is empty")
		return
	}

	// 使用Lua脚本获取用户的所有求购道具
	luaScript := `
		local userBuysKey = 'user:' .. ARGV[1] .. ':buys'
		local buyOrderKeys = redis.call('SMEMBERS', userBuysKey)
		local result = {}
		
		for _, key in ipairs(buyOrderKeys) do
			local buyData = redis.call('HGETALL', key)
			if buyData then
				table.insert(result, buyData)
			end
		end
		
		return result
	`

	// 执行Lua脚本
	result, err := redis.GetRedis().Eval(ctx, luaScript, []string{}, userId).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-MY-BUYS] Execute Lua script error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get user buys error"
		return
	}

	// 处理Lua脚本返回的结果
	buyDataList := make([]*auction.BuyData, 0)
	if resultList, ok := result.([]interface{}); ok {
		for _, item := range resultList {
			if buyDataMap, ok := item.([]interface{}); ok {
				// 将[]interface{}转换为map[string]string
				dataMap := make(map[string]string)
				for i := 0; i < len(buyDataMap); i += 2 {
					if key, ok := buyDataMap[i].(string); ok {
						if value, ok := buyDataMap[i+1].(string); ok {
							dataMap[key] = value
						}
					}
				}

				// 构造BuyData
				buyData := &auction.BuyData{
					OrderId: dataMap["order_id"],
					ItemId:  dataMap["item_id"],
				}

				// 解析数值字段
				if quantity, err := strconv.ParseInt(dataMap["quantity"], 10, 32); err == nil {
					buyData.Quantity = int32(quantity)
				}
				if price, err := strconv.ParseInt(dataMap["price"], 10, 64); err == nil {
					buyData.Price = price
				}
				if createTime, err := strconv.ParseInt(dataMap["create_time"], 10, 64); err == nil {
					buyData.CreateTime = createTime
				}

				// 添加到列表
				buyDataList = append(buyDataList, buyData)
			}
		}
	}

	// 更新响应
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	resp.Data = buyDataList
	klog.CtxInfof(ctx, "[AUCTION-MGR-GET-MY-BUYS] Get my buys success, userId: %s, count: %d", userId, len(buyDataList))
	return
}

// 获取道具出售求购详情协议
func (m *AuctionManager) GetItemAuctionInfo(ctx context.Context, req *auction.GetItemAuctionInfoReq) (resp *auction.GetItemAuctionInfoRsp, err error) {
	// 初始化resp
	resp = &auction.GetItemAuctionInfoRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
		Data: make([]*auction.ItemAuctionInfo, 0),
	}

	// 检查请求参数
	if len(req.GetItemIds()) == 0 {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "item_ids is empty"
		return
	}

	// 获取matchManager
	matchMgr := getMatchManager()

	// 遍历道具ID列表，获取每个道具的拍卖信息
	itemInfoList := make([]*auction.ItemAuctionInfo, 0, len(req.GetItemIds()))
	for _, itemId := range req.GetItemIds() {
		// 获取或创建matchUnit
		matchUnit := matchMgr.GetMatchUnit(itemId)
		// matchUnit必定存在
		// 直接通过opChannel执行getAuctionInfo，确保线程安全
		resultChan := make(chan *auction.ItemAuctionInfo, 1)
		matchUnit.opChannel <- func() {
			info := matchUnit.getAuctionInfo(ctx)
			resultChan <- info
		}
		// 等待结果
		info := <-resultChan
		itemInfoList = append(itemInfoList, info)
	}

	// 更新响应
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	resp.Data = itemInfoList
	return
}

// 按时间获取交易记录协议
func (m *AuctionManager) GetTransactionsByTime(ctx context.Context, req *auction.GetTransactionsByTimeReq) (resp *auction.GetTransactionsByTimeRsp, err error) {
	// 初始化resp
	resp = &auction.GetTransactionsByTimeRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
		Data: &auction.TransactionsByTimeData{
			Total:    0,
			Page:     req.GetPage(),
			PageSize: req.GetPageSize(),
			Records:  make([]*auction.TimeTransactionRecord, 0),
		},
	}

	// 检查请求参数
	if req.GetPage() < 1 {
		req.Page = 1
		resp.Data.Page = 1
	}
	if req.GetPageSize() < 1 || req.GetPageSize() > 100 {
		req.PageSize = 10
		resp.Data.PageSize = 10
	}

	// 获取用户ID
	userId := ""
	if val, ok := ctx.Value("userId").(string); ok {
		userId = val
	}

	// 检查用户ID
	if userId == "" {
		resp.Code = common.ErrorCode_AUCTION_USER_NOT_FOUND
		resp.Msg = "user_id is empty"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-TRANSACTIONS-BY-TIME] User ID is empty")
		return
	}

	// 构建用户交易时间排序集合键
	transactionsTimeKey := "user:" + userId + ":transactions:time"

	// 计算时间范围
	startTime := req.GetStartTime()
	endTime := req.GetEndTime()
	if endTime == 0 {
		endTime = time.Now().Unix()
	}

	// 使用Lua脚本获取交易记录
	luaScript := `
		local transactionsTimeKey = ARGV[1]
		local startTime = tonumber(ARGV[2])
		local endTime = tonumber(ARGV[3])
		local offset = tonumber(ARGV[4])
		local count = tonumber(ARGV[5])
		
		-- 1. 通过'user:{user_id}:transactions:time'获取订单ID（按时间降序）
		local orderIds = redis.call('ZREVRANGEBYSCORE', transactionsTimeKey, endTime, startTime, 'LIMIT', offset, count)
		
		-- 获取总记录数
		local total = redis.call('ZCOUNT', transactionsTimeKey, startTime, endTime)
		
		-- 2. 使用获取到的订单ID，查询对应的订单详细信息
		local transactions = {}
		for _, orderId in ipairs(orderIds) do
			-- 通过'auction:order:{order_id}:status'查询订单详细信息
			local statusKey = 'auction:order:' .. orderId .. ':status'
			local orderStatus = redis.call('HGETALL', statusKey)
			if orderStatus then
				-- 将订单状态添加到结果中
				table.insert(transactions, orderStatus)
			end
		end
		
		return {total, transactions}
	`

	// 计算偏移量
	offset := (req.GetPage() - 1) * req.GetPageSize()

	// 执行Lua脚本
	result, err := redis.GetRedis().Eval(ctx, luaScript, []string{},
		transactionsTimeKey,
		startTime,
		endTime,
		offset,
		req.GetPageSize(),
	).Result()

	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-TRANSACTIONS-BY-TIME] Execute Lua script error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get transactions error"
		return
	}

	// 处理Lua脚本返回的结果
	if resultList, ok := result.([]interface{}); ok && len(resultList) == 2 {
		// 获取总记录数
		if totalInterface, ok := resultList[0].(int64); ok {
			resp.Data.Total = int32(totalInterface)
		}

		// 获取交易记录
		if transactionsInterface, ok := resultList[1].([]interface{}); ok {
			records := make([]*auction.TimeTransactionRecord, 0, len(transactionsInterface))
			for _, transactionInterface := range transactionsInterface {
				if orderStatusMap, ok := transactionInterface.([]interface{}); ok {
					// 将[]interface{}转换为map[string]string
					dataMap := make(map[string]string)
					for i := 0; i < len(orderStatusMap); i += 2 {
						if key, ok := orderStatusMap[i].(string); ok {
							if value, ok := orderStatusMap[i+1].(string); ok {
								dataMap[key] = value
							}
						}
					}

					// 构造TimeTransactionRecord
					record := &auction.TimeTransactionRecord{
						TransactionId:  dataMap["order_id"],
						ItemId:         dataMap["item_id"],
						ItemInfo:       dataMap["item_info"],
						TradeDirection: dataMap["trade_direction"],
						Status:         dataMap["status"],
						UserId:         dataMap["user_id"],
					}

					// 解析数值字段
					if quantity, err := strconv.ParseInt(dataMap["final_quantity"], 10, 32); err == nil {
						record.Quantity = int32(quantity)
					}
					if price, err := strconv.ParseInt(dataMap["final_price"], 10, 64); err == nil {
						record.Price = price
					}
					if tax, err := strconv.ParseInt(dataMap["tax"], 10, 64); err == nil {
						record.Tax = tax
					}
					if createTime, err := strconv.ParseInt(dataMap["create_time"], 10, 64); err == nil {
						record.TransactionTime = createTime
					}
					if finalPrice, err := strconv.ParseInt(dataMap["final_price"], 10, 64); err == nil {
						record.FinalPrice = finalPrice
					}
					if finalQuantity, err := strconv.ParseInt(dataMap["final_quantity"], 10, 32); err == nil {
						record.FinalQuantity = int32(finalQuantity)
					}

					// 添加到记录列表
					records = append(records, record)
				}
			}
			resp.Data.Records = records
		}
	}

	// 更新响应
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	klog.CtxInfof(ctx, "[AUCTION-MGR-GET-TRANSACTIONS-BY-TIME] Get transactions by time success, userId: %s, count: %d", userId, len(resp.Data.Records))
	return
}

// 获取交易历史协议
func (m *AuctionManager) GetTransactionHistory(ctx context.Context, req *auction.GetTransactionHistoryReq) (resp *auction.GetTransactionHistoryRsp, err error) {
	// 初始化resp
	resp = &auction.GetTransactionHistoryRsp{
		Code: common.ErrorCode_AUCTION_PARAM_ERROR,
		Msg:  "default error",
		Data: &auction.TransactionHistoryData{
			Records: make([]*auction.TransactionRecord, 0),
		},
	}

	// 检查请求参数
	orderId := req.GetOrderId()
	if orderId == "" {
		resp.Code = common.ErrorCode_AUCTION_PARAM_ERROR
		resp.Msg = "order_id is empty"
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-TRANSACTION-HISTORY] Order ID is empty")
		return
	}

	// 使用Lua脚本获取交易记录
	luaScript := `
		local orderId = ARGV[1]
		
		-- 1. 从"auction:order:{order_id}:transactions" 获取交易id
		local transactionsKey = 'auction:order:' .. orderId .. ':transactions'
		local transactionIds = redis.call('SMEMBERS', transactionsKey)
		
		-- 2. 遍历每个交易id，从"auction:transaction:{transactionId}" 获取交易明细
		local transactions = {}
		for _, transactionId in ipairs(transactionIds) do
			local transactionKey = 'auction:transaction:' .. transactionId
			local transaction = redis.call('HGETALL', transactionKey)
			if transaction then
				table.insert(transactions, transaction)
			end
		end
		
		return transactions
	`

	// 执行Lua脚本
	result, err := redis.GetRedis().Eval(ctx, luaScript, []string{}, orderId).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MGR-GET-TRANSACTION-HISTORY] Execute Lua script error: %s", err.Error())
		resp.Code = common.ErrorCode_AUCTION_REDIS_ERROR
		resp.Msg = "get transaction history error"
		return
	}

	// 处理Lua脚本返回的结果
	if transactionsInterface, ok := result.([]interface{}); ok {
		records := make([]*auction.TransactionRecord, 0, len(transactionsInterface))
		for _, transactionInterface := range transactionsInterface {
			if transactionMap, ok := transactionInterface.([]interface{}); ok {
				// 将[]interface{}转换为map[string]string
				dataMap := make(map[string]string)
				for i := 0; i < len(transactionMap); i += 2 {
					if key, ok := transactionMap[i].(string); ok {
						if value, ok := transactionMap[i+1].(string); ok {
							dataMap[key] = value
						}
					}
				}

				// 构造TransactionRecord
				record := &auction.TransactionRecord{
					TransactionId: dataMap["transaction_id"],
					BuyOrderId:    dataMap["buy_order_id"],
					SellOrderId:   dataMap["sell_order_id"],
					ItemId:        dataMap["item_id"],
					ItemInfo:      dataMap["item_info"],
				}

				// 解析数值字段
				if quantity, err := strconv.ParseInt(dataMap["quantity"], 10, 32); err == nil {
					record.Quantity = int32(quantity)
				}
				if price, err := strconv.ParseInt(dataMap["price"], 10, 64); err == nil {
					record.Price = price
				}
				if tax, err := strconv.ParseInt(dataMap["tax"], 10, 64); err == nil {
					record.Tax = tax
				}
				if transactionTime, err := strconv.ParseInt(dataMap["transaction_time"], 10, 64); err == nil {
					record.TransactionTime = transactionTime
				}

				// 添加到记录列表
				records = append(records, record)
			}
		}
		resp.Data.Records = records
	}

	// 更新响应
	resp.Code = common.ErrorCode_OK
	resp.Msg = "success"
	klog.CtxInfof(ctx, "[AUCTION-MGR-GET-TRANSACTION-HISTORY] Get transaction history success, orderId: %s, count: %d", orderId, len(resp.Data.Records))
	return
}
