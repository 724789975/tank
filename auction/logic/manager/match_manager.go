package manager

import (
	"auction_module/kitex_gen/auction"
	"auction_module/redis"
	"context"
	"strconv"
	"sync"

	"github.com/cloudwego/kitex/pkg/klog"
)

// matchManager 撮合管理器（私有，不对外服务）
type matchManager struct {
	matchUnits map[string]*matchUnit // 道具ID -> matchUnit映射
	mu         sync.RWMutex          // 并发控制锁
	ctx        context.Context       // 上下文，用于日志记录
	cancel     context.CancelFunc    // 取消函数，用于取消撮合管理器
}

var (
	matchMgr  *matchManager
	matchOnce sync.Once
)

// parseInt 将字符串转换为int
func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// parseInt64 将字符串转换为int64
func parseInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

// getMatchManager 获取撮合管理器实例（私有方法）
func getMatchManager() *matchManager {
	matchOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		matchMgr = &matchManager{
			matchUnits: make(map[string]*matchUnit),
			mu:         sync.RWMutex{},
			ctx:        ctx,
			cancel:     cancel,
		}

		// 读取Redis中未成交的订单并放入相应的matchUnit
		loadOrdersFromRedis(ctx)

		klog.CtxInfof(ctx, "[AUCTION-MATCH-MGR] MatchManager initialized")
	})
	return matchMgr
}

// GetMatchUnit 根据道具ID获取matchUnit（懒加载模式）
func (m *matchManager) GetMatchUnit(itemId string) *matchUnit {
	m.mu.RLock()
	matchUnit, exists := m.matchUnits[itemId]
	m.mu.RUnlock()

	if exists {
		return matchUnit
	}

	// 如果不存在，创建新的matchUnit
	m.mu.Lock()
	defer m.mu.Unlock()

	// 再次检查，避免并发创建
	matchUnit, exists = m.matchUnits[itemId]
	if exists {
		return matchUnit
	}

	// 创建新的matchUnit
	matchUnit = newMatchUnit(itemId)
	matchUnit.startMatchProcess(m.ctx)
	m.matchUnits[itemId] = matchUnit
	klog.Info("[AUCTION-MATCH-MGR] Created matchUnit for item: %s", itemId)

	return matchUnit
}

// ProcessMatchResult 处理撮合结果
func (m *matchManager) ProcessMatchResult(ctx context.Context, sellOrder *auction.SellData, buyOrder *auction.BuyData) {
	// TODO: 实现撮合结果的处理逻辑
	// 1. 更新订单状态
	// 2. 处理交易完成后的逻辑
	// 3. 扣除买家税费（1% 交易金额，放在常量中定义）
	klog.CtxInfof(ctx, "[AUCTION-MATCH] Process match result: sellOrder=%s, buyOrder=%s",
		sellOrder.GetOrderId(), buyOrder.GetOrderId())
}

// loadOrdersFromRedis 从Redis加载未成交的订单到matchUnit
func loadOrdersFromRedis(ctx context.Context) {
	// 读取全局出售列表
	sellOrders, err := redis.GetRedis().SMembers(ctx, "auction:sells").Result()
	if err != nil {
		klog.Errorf("[AUCTION-MATCH-MGR] get sell orders error: %s", err.Error())
	} else {
		for _, orderKey := range sellOrders {
			// 获取订单详情
			orderData, err := redis.GetRedis().HGetAll(ctx, orderKey).Result()
			if err != nil {
				klog.Errorf("[AUCTION-MATCH-MGR] get sell order data error: %s", err.Error())
				continue
			}

			// 解析订单数据
			sellData := &auction.SellData{
				OrderId:    orderData["order_id"],
				ItemId:     orderData["item_id"],
				Quantity:   int32(parseInt(orderData["quantity"])),
				Price:      parseInt64(orderData["price"]),
				ItemInfo:   orderData["item_info"],
				CreateTime: parseInt64(orderData["create_time"]),
			}

			// 添加到matchUnit
			mu := matchMgr.GetMatchUnit(sellData.ItemId)
			mu.opChannel <- func() {
				mu.AddSellOrder(ctx, sellData)
			}
		}
		klog.CtxInfof(ctx, "[AUCTION-MATCH-MGR] loaded %d sell orders", len(sellOrders))
	}

	// 读取全局求购列表
	buyOrders, err := redis.GetRedis().SMembers(ctx, "auction:buys").Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[AUCTION-MATCH-MGR] get buy orders error: %s", err.Error())
	} else {
		for _, orderKey := range buyOrders {
			// 获取订单详情
			orderData, err := redis.GetRedis().HGetAll(ctx, orderKey).Result()
			if err != nil {
				klog.CtxErrorf(ctx, "[AUCTION-MATCH-MGR] get buy order data error: %s", err.Error())
				continue
			}

			// 解析订单数据
			buyData := &auction.BuyData{
				OrderId:    orderData["order_id"],
				ItemId:     orderData["item_id"],
				Quantity:   int32(parseInt(orderData["quantity"])),
				Price:      parseInt64(orderData["price"]),
				CreateTime: parseInt64(orderData["create_time"]),
			}

			// 添加到matchUnit
			mu := matchMgr.GetMatchUnit(buyData.ItemId)
			mu.opChannel <- func() {
				mu.AddBuyOrder(ctx, buyData)
			}
		}
		for _, mu := range matchMgr.matchUnits {
			r := make(chan bool, 0)
			mu.opChannel <- func() {
				r <- true
			}
			<-r
		}
		klog.CtxInfof(ctx, "[AUCTION-MATCH-MGR] loaded %d buy orders", len(buyOrders))
	}
}
