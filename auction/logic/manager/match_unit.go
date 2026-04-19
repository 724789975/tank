package manager

import (
	"auction_module/kitex_gen/auction"
	"auction_module/redis"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/google/btree"
)

// matchUnit 撮合单元，处理特定itemId的道具撮合（私有，不对外服务）
type matchUnit struct {
	itemId           string
	sellOrders       *btree.BTree // 卖单BTree，按价格升序
	buyOrders        *btree.BTree // 买单BTree，按价格降序
	opChannel        chan func()  // 用于外部操作的channel
	currentHour      int64        // 当前小时时间戳（小时级）
	hourlyTotalPrice int64        // 小时内总成交价格
	hourlyTotalQty   int32        // 小时内总成交数量
	hourlyAvgPrice   int64        // 小时内平均成交价格
}

// newMatchUnit 创建新的撮合单元（私有方法）
func newMatchUnit(itemId string) *matchUnit {
	// 计算当前小时时间戳（小时级）
	now := time.Now()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location()).Unix()

	// 初始化平均价格
	hourlyAvgPrice := int64(100) // 默认值为1

	// 从Redis读取最新的按小时记录的平均成交价格
	ctx := context.Background()
	hourlyPriceKey := "auction:hourly:price:" + itemId
	avgPriceStr, err := redis.GetRedis().Get(ctx, hourlyPriceKey).Result()
	if err != nil {
		// 如果没有找到数据，记录日志
		klog.Infof("[AUCTION-MATCH-UNIT] No hourly price found in Redis for itemId=%s, hour=%s, initializing to 100",
			itemId, now.Format("2006-01-02 15:00"))
	} else {
		// 解析平均价格
		_, err := fmt.Sscanf(avgPriceStr, "%d", &hourlyAvgPrice)
		if err != nil {
			klog.Errorf("[AUCTION-MATCH-UNIT] Parse hourly price error: %v, initializing to 100", err)
			hourlyAvgPrice = 100
		}
	}

	return &matchUnit{
		itemId:           itemId,
		sellOrders:       btree.New(32),           // 创建卖单BTree，默认度为32
		buyOrders:        btree.New(32),           // 创建买单BTree，默认度为32
		opChannel:        make(chan func(), 1024), // 初始化操作channel，容量为1024
		currentHour:      currentHour,             // 当前小时时间戳（小时级）
		hourlyTotalPrice: hourlyAvgPrice,          // 小时内总成交价格（初始化为平均价格）
		hourlyTotalQty:   0,                       // 小时内总成交数量（初始化为1）
		hourlyAvgPrice:   hourlyAvgPrice,          // 小时内平均成交价格
	}
}

// SellOrderByPriceAsc 卖单按价格升序排序
type SellOrderByPriceAsc auction.SellData

// Less 实现btree.Item接口，按价格升序，价格相同按创建时间升序
func (s SellOrderByPriceAsc) Less(than btree.Item) bool {
	other := than.(SellOrderByPriceAsc)
	if s.Price != other.Price {
		return s.Price < other.Price
	}
	if s.CreateTime != other.CreateTime {
		return s.CreateTime < other.CreateTime
	}
	return s.OrderId < other.OrderId
}

// AddSellOrder 添加主动性卖单（按买方报价成交）
func (mu *matchUnit) AddSellOrder(ctx context.Context, order *auction.SellData) {
	// 设置创建时间
	if order.CreateTime == 0 {
		order.CreateTime = time.Now().Unix()
	}

	// 先尝试撮合（主动性卖单，按买方报价成交）
	mu.matchSellOrder(ctx, order)

	// 如果订单未完全成交，将剩余数量添加到卖单BTree
	if order.Quantity > 0 {
		sellOrder := SellOrderByPriceAsc(*order)
		mu.sellOrders.ReplaceOrInsert(sellOrder)
		klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Add sell order: orderId=%s, itemId=%s, quantity=%d, price=%d",
			order.OrderId, order.ItemId, order.Quantity, order.Price)
	}
}

// matchSellOrder 撮合卖单（主动性卖单按买方报价成交）
func (mu *matchUnit) matchSellOrder(ctx context.Context, sellOrder *auction.SellData) {
	// 遍历买单BTree，按价格降序（从高到低）
	mu.buyOrders.Ascend(func(item btree.Item) bool {
		buyOrder := item.(BuyOrderByPriceDesc)

		// 如果卖单价格 <= 买单价格，且卖单数量 > 0
		if sellOrder.Price <= buyOrder.Price && sellOrder.Quantity > 0 {
			// 计算成交数量（取两者较小值）
			quantity := sellOrder.Quantity
			if buyOrder.Quantity < quantity {
				quantity = buyOrder.Quantity
			}

			// 记录成交信息并调用matchResult
			klog.CtxInfof(ctx, "[AUCTION-MATCH] Match sell order: sellOrder=%s, buyOrder=%s, price=%d, quantity=%d",
				sellOrder.OrderId, buyOrder.OrderId, buyOrder.Price, quantity)

			// 调用matchResult处理成交结果
			mu.matchResult(ctx, buyOrder.Price, quantity, sellOrder, (*auction.BuyData)(&buyOrder))

			// 更新订单数量
			sellOrder.Quantity -= quantity

			// 如果买单完全成交，从BTree中移除
			if buyOrder.Quantity <= quantity {
				mu.buyOrders.Delete(item)
				return sellOrder.Quantity > 0 // 如果卖单还有剩余，继续撮合
			}

			// 更新买单剩余数量
			updateOrder := buyOrder
			updateOrder.Quantity -= quantity
			mu.buyOrders.Delete(item)
			mu.buyOrders.ReplaceOrInsert(updateOrder)
		}

		// 如果卖单已完成或买单价格低于卖单价格，停止撮合
		return sellOrder.Quantity > 0 && buyOrder.Price >= sellOrder.Price
	})
}

// BuyOrderByPriceDesc 买单按价格降序排序
type BuyOrderByPriceDesc auction.BuyData

// Less 实现btree.Item接口，按价格降序，价格相同按创建时间升序
func (b BuyOrderByPriceDesc) Less(than btree.Item) bool {
	other := than.(BuyOrderByPriceDesc)
	if b.Price != other.Price {
		return b.Price > other.Price
	}
	if b.CreateTime != other.CreateTime {
		return b.CreateTime < other.CreateTime
	}
	return b.OrderId < other.OrderId
}

// AddBuyOrder 添加主动性买单（按卖方报价成交）
func (mu *matchUnit) AddBuyOrder(ctx context.Context, order *auction.BuyData) {
	// 设置创建时间
	if order.CreateTime == 0 {
		order.CreateTime = time.Now().Unix()
	}

	// 先尝试撮合（主动性买单，按卖方报价成交）
	mu.matchBuyOrder(ctx, order)

	// 如果订单未完全成交，将剩余数量添加到买单BTree
	if order.Quantity > 0 {
		buyOrder := BuyOrderByPriceDesc(*order)
		mu.buyOrders.ReplaceOrInsert(buyOrder)
		klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Add buy order: orderId=%s, itemId=%s, quantity=%d, price=%d",
			order.OrderId, order.ItemId, order.Quantity, order.Price)
	}
}

// matchBuyOrder 撮合买单（主动性买单按卖方报价成交）
func (mu *matchUnit) matchBuyOrder(ctx context.Context, buyOrder *auction.BuyData) {
	// 遍历卖单BTree，按价格升序（从低到高）
	mu.sellOrders.Ascend(func(item btree.Item) bool {
		sellOrder := item.(SellOrderByPriceAsc)

		// 如果买单价格 >= 卖单价格，且买单数量 > 0
		if buyOrder.Price >= sellOrder.Price && buyOrder.Quantity > 0 {
			// 计算成交数量（取两者较小值）
			quantity := buyOrder.Quantity
			if sellOrder.Quantity < quantity {
				quantity = sellOrder.Quantity
			}

			// 记录成交信息并调用MatchResult
			klog.CtxInfof(ctx, "[AUCTION-MATCH] Match buy order: buyOrder=%s, sellOrder=%s, price=%d, quantity=%d",
				buyOrder.OrderId, sellOrder.OrderId, sellOrder.Price, quantity)

			// 调用matchResult处理成交结果
			mu.matchResult(ctx, sellOrder.Price, quantity, (*auction.SellData)(&sellOrder), buyOrder)

			// 更新订单数量
			buyOrder.Quantity -= quantity

			// 如果卖单完全成交，从BTree中移除
			if sellOrder.Quantity <= quantity {
				mu.sellOrders.Delete(item)
				return buyOrder.Quantity > 0 // 如果买单还有剩余，继续撮合
			}

			// 更新卖单剩余数量
			updateOrder := sellOrder
			updateOrder.Quantity -= quantity
			mu.sellOrders.Delete(item)
			mu.sellOrders.ReplaceOrInsert(updateOrder)
		}

		// 如果买单已完成或卖单价格高于买单价格，停止撮合
		return buyOrder.Quantity > 0 && sellOrder.Price <= buyOrder.Price
	})
}

// RemoveSellOrder 移除卖单
func (mu *matchUnit) RemoveSellOrder(ctx context.Context, orderId string) bool {
	found := false
	mu.sellOrders.Ascend(func(item btree.Item) bool {
		order := item.(SellOrderByPriceAsc)
		if order.OrderId == orderId {
			mu.sellOrders.Delete(item)
			found = true
			klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Remove sell order: orderId=%s, itemId=%s",
				orderId, order.ItemId)
			return false
		}
		return true
	})
	return found
}

// RemoveBuyOrder 移除买单
func (mu *matchUnit) RemoveBuyOrder(ctx context.Context, orderId string) bool {
	found := false
	mu.buyOrders.Ascend(func(item btree.Item) bool {
		order := item.(BuyOrderByPriceDesc)
		if order.OrderId == orderId {
			mu.buyOrders.Delete(item)
			found = true
			klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Remove buy order: orderId=%s, itemId=%s",
				orderId, order.ItemId)
			return false
		}
		return true
	})
	return found
}

// matchResult 处理撮合结果（私有方法）
func (mu *matchUnit) matchResult(ctx context.Context, price int64, quantity int32, sellData *auction.SellData, buyData *auction.BuyData) {
	// 记录成交信息
	klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Match found: buyOrder=%s, sellOrder=%s, price=%d, quantity=%d",
		buyData.OrderId, sellData.OrderId, price, quantity)

	// 验证交易合法性
	if !mu.validateTransaction(ctx, sellData, buyData, quantity) {
		klog.CtxErrorf(ctx, "[AUCTION-MATCH-UNIT] Invalid transaction: buyOrder=%s, sellOrder=%s",
			buyData.OrderId, sellData.OrderId)
		return
	}

	// 处理小时数据
	mu.processHourlyData(price, quantity)

	// 生成交易ID（使用雪花算法）
	transactionId := idClient.Generate().String()

	// 计算税费（假设为交易金额的1%）
	tax := (price * int64(quantity)) / 100

	// 计算各种key
	transactionKey := "auction:transaction:" + transactionId
	sellOrderTransactionsKey := "auction:order:" + sellData.OrderId + ":transactions"
	buyOrderTransactionsKey := "auction:order:" + buyData.OrderId + ":transactions"
	sellOrderKey := "auction:sell:" + sellData.OrderId
	buyOrderKey := "auction:buy:" + buyData.OrderId
	sellStatusKey := "auction:order:" + sellData.OrderId + ":status"
	buyStatusKey := "auction:order:" + buyData.OrderId + ":status"

	// 获取卖家和买家ID
	sellerId, _ := redis.GetRedis().HGet(ctx, sellOrderKey, "user_id").Result()
	buyerId, _ := redis.GetRedis().HGet(ctx, buyOrderKey, "user_id").Result()

	// 执行优化的Lua脚本
	luaScript := `
		-- 优化的交易处理脚本
		local transactionId = ARGV[1]
		local sellOrderId = ARGV[2]
		local buyOrderId = ARGV[3]
		local itemId = ARGV[4]
		local itemInfo = ARGV[5]
		local quantity = tonumber(ARGV[6])
		local price = tonumber(ARGV[7])
		local tax = tonumber(ARGV[8])
		local transactionTime = tonumber(ARGV[9])
		local transactionKey = ARGV[10]
		local sellOrderTransactionsKey = ARGV[11]
		local buyOrderTransactionsKey = ARGV[12]
		local sellerId = ARGV[13]
		local buyerId = ARGV[14]
		local sellOrderKey = ARGV[15]
		local buyOrderKey = ARGV[16]
		local sellStatusKey = ARGV[17]
		local buyStatusKey = ARGV[18]

		-- 1. 存储交易记录
		redis.call('HMSET', transactionKey,
			'transaction_id', transactionId,
			'buy_order_id', buyOrderId,
			'sell_order_id', sellOrderId,
			'item_id', itemId,
			'item_info', itemInfo,
			'quantity', quantity,
			'price', price,
			'tax', tax,
			'transaction_time', transactionTime
		)

		-- 2. 关联交易ID与订单ID
		redis.call('SADD', sellOrderTransactionsKey, transactionId)
		redis.call('SADD', buyOrderTransactionsKey, transactionId)

		-- 3. 获取卖单当前状态信息
		local sellCreateTime = redis.call('HGET', sellOrderKey, 'create_time')
		local currentSellFinalPrice = tonumber(redis.call('HGET', sellStatusKey, 'final_price') or '0')
		local currentSellFinalQuantity = tonumber(redis.call('HGET', sellStatusKey, 'final_quantity') or '0')
		local currentSellTax = tonumber(redis.call('HGET', sellStatusKey, 'tax') or '0')

		-- 4. 获取买单当前状态信息
		local buyCreateTime = redis.call('HGET', buyOrderKey, 'create_time')
		local currentBuyFinalPrice = tonumber(redis.call('HGET', buyStatusKey, 'final_price') or '0')
		local currentBuyFinalQuantity = tonumber(redis.call('HGET', buyStatusKey, 'final_quantity') or '0')

		-- 5. 计算累加后的数值
		local newSellFinalPrice = currentSellFinalPrice + (price * quantity)
		local newSellFinalQuantity = currentSellFinalQuantity + quantity
		local newSellTax = currentSellTax + tax
		local newBuyFinalPrice = currentBuyFinalPrice + (price * quantity)
		local newBuyFinalQuantity = currentBuyFinalQuantity + quantity

		-- 6. 更新卖单剩余数量
		local sellQuantity = tonumber(redis.call('HGET', sellOrderKey, 'quantity'))
		local newSellQuantity = sellQuantity - quantity

		-- 7. 更新买单剩余数量
		local buyQuantity = tonumber(redis.call('HGET', buyOrderKey, 'quantity'))
		local newBuyQuantity = buyQuantity - quantity

		-- 8. 更新卖单状态
		if newSellQuantity <= 0 then
			-- 卖单已完成，更新状态为交易完成
			redis.call('HMSET', sellStatusKey, 
				'order_id', sellOrderId,
				'trade_direction', 'sell',
				'status', '交易完成',
				'final_price', newSellFinalPrice,
				'final_quantity', newSellFinalQuantity,
				'item_id', itemId,
				'tax', newSellTax,
				'create_time', sellCreateTime,
				'user_id', sellerId
			)
			-- 删除卖单
			redis.call('DEL', sellOrderKey)
			-- 从用户出售列表中移除
			if sellerId and sellerId ~= '' then
				local sellerSellsKey = 'user:' .. sellerId .. ':sells'
				redis.call('SREM', sellerSellsKey, sellOrderKey)
			end
			-- 从全局出售列表中移除
			redis.call('SREM', 'auction:sells', sellOrderKey)
		else
			-- 卖单未完成，更新剩余数量
			redis.call('HSET', sellOrderKey, 'quantity', newSellQuantity)
			-- 更新卖单状态（保持原状态，只更新累计数值）
			redis.call('HMSET', sellStatusKey, 
				'order_id', sellOrderId,
				'trade_direction', 'sell',
				'status', '卖',
				'final_price', newSellFinalPrice,
				'final_quantity', newSellFinalQuantity,
				'item_id', itemId,
				'tax', newSellTax,
				'create_time', sellCreateTime,
				'user_id', sellerId
			)
		end

		-- 9. 更新买单状态
		if newBuyQuantity <= 0 then
			-- 买单已完成，更新状态为交易完成
			redis.call('HMSET', buyStatusKey, 
				'order_id', buyOrderId,
				'trade_direction', 'buy',
				'status', '交易完成',
				'final_price', newBuyFinalPrice,
				'final_quantity', newBuyFinalQuantity,
				'item_id', itemId,
				'tax', 0,
				'create_time', buyCreateTime,
				'user_id', buyerId
			)
			-- 删除买单
			redis.call('DEL', buyOrderKey)
			-- 从用户求购列表中移除
			if buyerId and buyerId ~= '' then
				local buyerBuysKey = 'user:' .. buyerId .. ':buys'
				redis.call('SREM', buyerBuysKey, buyOrderKey)
			end
			-- 从全局求购列表中移除
			redis.call('SREM', 'auction:buys', buyOrderKey)
		else
			-- 买单未完成，更新剩余数量
			redis.call('HSET', buyOrderKey, 'quantity', newBuyQuantity)
			-- 更新买单状态（保持原状态，只更新累计数值）
			redis.call('HMSET', buyStatusKey, 
				'order_id', buyOrderId,
				'trade_direction', 'buy',
				'status', '买',
				'final_price', newBuyFinalPrice,
				'final_quantity', newBuyFinalQuantity,
				'item_id', itemId,
				'tax', 0,
				'create_time', buyCreateTime,
				'user_id', buyerId
			)
		end

		return {success = true}
	`

	// 执行Lua脚本
	_, err := redis.GetRedis().Eval(ctx, luaScript, []string{},
		transactionId,
		sellData.OrderId,
		buyData.OrderId,
		sellData.ItemId,
		sellData.ItemInfo,
		quantity,
		price,
		tax,
		time.Now().Unix(),
		transactionKey,
		sellOrderTransactionsKey,
		buyOrderTransactionsKey,
		sellerId,
		buyerId,
		sellOrderKey,
		buyOrderKey,
		sellStatusKey,
		buyStatusKey,
	).Result()

	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MATCH-UNIT] Execute transaction script error: %v", err)
		return
	}

	// 记录数据变更日志
	klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Transaction completed: transactionId=%s, buyOrder=%s, sellOrder=%s, quantity=%d",
		transactionId, buyData.OrderId, sellData.OrderId, quantity)
}

// validateTransaction 验证交易合法性
func (mu *matchUnit) validateTransaction(ctx context.Context, sellData *auction.SellData, buyData *auction.BuyData, quantity int32) bool {
	// 验证订单ID
	if sellData.OrderId == "" || buyData.OrderId == "" {
		klog.CtxErrorf(ctx, "[AUCTION-MATCH-UNIT] Invalid transaction: orderId is empty")
		return false
	}

	// 验证物品ID匹配
	if sellData.ItemId != buyData.ItemId {
		klog.CtxErrorf(ctx, "[AUCTION-MATCH-UNIT] Invalid transaction: itemId mismatch, sellItemId=%s, buyItemId=%s",
			sellData.ItemId, buyData.ItemId)
		return false
	}

	// 验证数量有效
	if quantity <= 0 {
		klog.CtxErrorf(ctx, "[AUCTION-MATCH-UNIT] Invalid transaction: quantity <= 0, quantity=%d", quantity)
		return false
	}

	// 验证价格匹配（卖单价格 <= 买单价格）
	if sellData.Price > buyData.Price {
		klog.CtxErrorf(ctx, "[AUCTION-MATCH-UNIT] Invalid transaction: price mismatch, sellPrice=%d, buyPrice=%d",
			sellData.Price, buyData.Price)
		return false
	}

	return true
}

// processHourlyData 处理小时内的成交数据
func (mu *matchUnit) processHourlyData(price int64, quantity int32) {
	// 累加当前小时的成交价格和数量
	mu.hourlyTotalPrice += price * int64(quantity)
	mu.hourlyTotalQty += quantity
}

// saveHourlyData 保存小时数据到Redis
func (mu *matchUnit) saveHourlyData() {
	// 构造Redis键（固定格式）
	hourlyPriceKey := "auction:hourly:price:" + mu.itemId

	// 计算平均价格
	avgPrice := max(mu.hourlyAvgPrice, 100)

	// 如果当前小时成交量为0，平均价格下降10%，直到价格为100
	if mu.hourlyTotalQty == 0 && avgPrice > 100 {
		avgPrice = int64(float64(avgPrice) * 0.9)
		// 确保价格不低于1
		if avgPrice < 100 {
			avgPrice = 100
		}
		// 更新结构体中的平均价格
		mu.hourlyAvgPrice = avgPrice
		klog.Infof("[AUCTION-MATCH-UNIT] No transaction in this hour, average price decreased by 10%%: itemId=%s, newAvgPrice=%d",
			mu.itemId, avgPrice)
	} else if mu.hourlyTotalQty > 0 {
		// 有成交数据，重新计算平均价格
		avgPrice = mu.hourlyTotalPrice / int64(mu.hourlyTotalQty)
		mu.hourlyAvgPrice = avgPrice
	}
	avgPrice = max(mu.hourlyAvgPrice, 100)

	// 写入Redis
	err := redis.GetRedis().Set(getMatchManager().ctx, hourlyPriceKey, avgPrice, 24*time.Hour*7).Err()
	if err != nil {
		klog.Errorf("[AUCTION-MATCH-UNIT] Save hourly price to Redis error: %v", err)
	} else {
		klog.Infof("[AUCTION-MATCH-UNIT] Save hourly price to Redis: itemId=%s, avgPrice=%d",
			mu.itemId, avgPrice)
	}
}

// getAuctionInfo 获取买5卖5信息（聚合相同价格的订单，优化遍历算法）
// 内部方法，不直接调用，通过opChannel执行
func (mu *matchUnit) getAuctionInfo(ctx context.Context) *auction.ItemAuctionInfo {
	// 初始化返回结果
	info := &auction.ItemAuctionInfo{
		ItemId: mu.itemId,
		Sells:  make([]*auction.OrderInfo, 0, 5),
		Buys:   make([]*auction.OrderInfo, 0, 5),
	}

	// 聚合卖单（相同价格的订单聚合，遍历到足够的价格数量）
	sellPriceMap := make(map[int64]int32)
	mu.sellOrders.Ascend(func(item btree.Item) bool {
		sellOrder := item.(SellOrderByPriceAsc)
		sellPriceMap[sellOrder.Price] += sellOrder.Quantity
		// 当收集到6个不同的价格时，停止遍历
		// 这样可以确保所有相同价格的订单都被处理
		return len(sellPriceMap) < 6
	})

	// 按价格升序排序
	sellPrices := make([]int64, 0, len(sellPriceMap))
	for price := range sellPriceMap {
		sellPrices = append(sellPrices, price)
	}
	// 使用sort包排序（升序）
	sort.Slice(sellPrices, func(i, j int) bool {
		return sellPrices[i] < sellPrices[j]
	})

	// 构造卖单列表，只取前5个
	sellCount := 0
	for _, price := range sellPrices {
		if sellCount >= 5 {
			break
		}
		orderInfo := &auction.OrderInfo{
			ItemId:   mu.itemId,
			Quantity: sellPriceMap[price],
			Price:    price,
		}
		info.Sells = append(info.Sells, orderInfo)
		sellCount++
	}

	// 聚合买单（相同价格的订单聚合，遍历到足够的价格数量）
	buyPriceMap := make(map[int64]int32)
	mu.buyOrders.Ascend(func(item btree.Item) bool {
		buyOrder := item.(BuyOrderByPriceDesc)
		buyPriceMap[buyOrder.Price] += buyOrder.Quantity
		// 当收集到6个不同的价格时，停止遍历
		// 这样可以确保所有相同价格的订单都被处理
		return len(buyPriceMap) < 6
	})

	// 按价格降序排序
	buyPrices := make([]int64, 0, len(buyPriceMap))
	for price := range buyPriceMap {
		buyPrices = append(buyPrices, price)
	}
	// 使用sort包排序（降序）
	sort.Slice(buyPrices, func(i, j int) bool {
		return buyPrices[i] > buyPrices[j]
	})

	// 构造买单列表，只取前5个
	buyCount := 0
	for _, price := range buyPrices {
		if buyCount >= 5 {
			break
		}
		orderInfo := &auction.OrderInfo{
			ItemId:   mu.itemId,
			Quantity: buyPriceMap[price],
			Price:    price,
		}
		info.Buys = append(info.Buys, orderInfo)
		buyCount++
	}

	klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Get auction info: itemId=%s, sells=%d, buys=%d",
		mu.itemId, len(info.Sells), len(info.Buys))

	return info
}

func (mu *matchUnit) startMatchProcess(ctx context.Context) {
	// 启动定时任务，在每个小时的整点保存小时数据
	go func(ctx context.Context) {
		now := time.Now()
		nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
		duration := nextHour.Sub(now)

		// 等待到下一个小时
		timer := time.NewTimer(duration)
		needReset := true
		for {
			// 计算下一个小时的开始时间
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case op := <-mu.opChannel:
				op()
			case <-timer.C:
				// 保存当前小时的数据
				mu.saveHourlyData()

				if needReset {
					needReset = false
					timer.Reset(time.Hour)
				}

				// 重置当前小时的数据
				mu.currentHour = nextHour.Unix()
				mu.hourlyTotalPrice = 0
				mu.hourlyTotalQty = 0
				mu.hourlyAvgPrice = 0
			}
		}
	}(ctx)

	klog.CtxInfof(ctx, "[AUCTION-MATCH-UNIT] Match process started for item: %s", mu.itemId)
}
