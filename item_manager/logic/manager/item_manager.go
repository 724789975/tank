package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"item_manager/kitex_gen/common"
	"item_manager/kitex_gen/item"
	common_redis "item_manager/redis"
	"os"
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/redis/go-redis/v9"
)

const (
	ITEM_KEY_PREFIX = "item:user:{%s}:"
)

type ItemConfig struct {
	ItemId   int `json:"item_id"`
	IsUnique int `json:"is_unique"`
}

type ItemManager struct {
	rdb         redis.UniversalClient
	itemConfigs map[int]*ItemConfig
}

var (
	itemManager *ItemManager
	once        sync.Once
	IdClient    *snowflake.Node
)

func GetItemManager() *ItemManager {
	once.Do(func() {
		// 初始化snowflake ID生成器
		key := "item_svr:snowflake:node"
		n, err := common_redis.GetRedis().Incr(context.Background(), key).Result()
		if err != nil {
			klog.Fatal("[ITEM-MANAGER-INIT] ItemManager: gen uuid creator err: %v", err)
		}

		nodeIdx := n % (1 << snowflake.NodeBits)
		if node, err := snowflake.NewNode(nodeIdx); err != nil {
			klog.Fatal("[ITEM-MANAGER-NODE] ItemManager: gen uuid creator err: %v", err)
		} else {
			klog.Infof("[ITEM-MANAGER-NODE-OK] ItemManager: gen uuid creator success, node: %d", nodeIdx)
			IdClient = node
		}

		itemManager = &ItemManager{
			rdb:         common_redis.GetRedis(),
			itemConfigs: make(map[int]*ItemConfig),
		}

		// 读取道具配置文件
		if err := itemManager.loadItemConfigs(); err != nil {
			klog.Errorf("[ITEM-MANAGER-LOAD-CONFIG-ERROR] Failed to load item config: %v", err)
		}
	})
	return itemManager
}

func (m *ItemManager) loadItemConfigs() error {
	// 从项目根目录读取配置文件
	configPath := "etc/item.json"

	data, err := os.ReadFile(configPath)
	if err != nil {
		// 如果当前目录找不到，尝试从上级目录查找
		configPath = "../etc/item.json"
		data, err = os.ReadFile(configPath)
		if err != nil {
			configPath = "../../etc/item.json"
			data, err = os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	var configs []*ItemConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	for _, config := range configs {
		m.itemConfigs[config.ItemId] = config
	}

	klog.Infof("[ITEM-MANAGER-LOAD-CONFIG-SUCCESS] Loaded %d item configs", len(m.itemConfigs))
	return nil
}

func (m *ItemManager) getUserKey(userId string) string {
	return fmt.Sprintf(ITEM_KEY_PREFIX, userId)
}

func (m *ItemManager) AddItem(ctx context.Context, req *item.AddItemReq) (resp *item.AddItemRsp, err error) {
	userId := ctx.Value("userId").(string)
	userKey := m.getUserKey(userId)

	klog.CtxInfof(ctx, "[ITEM-ADD-START] userId: %s, itemCount: %d, idempotentId: %s", userId, len(req.ItemAddList), req.IdempotentId)

	luaScript := `
		local user_key = KEYS[1]
		local idempotent_key = KEYS[2]
		local item_data = cjson.decode(ARGV[1])
		
		local cached_result = redis.call('get', idempotent_key)
		if cached_result then
			return cached_result
		end
		
		local results = {}
		local user_items_set_key = user_key .. 'items'
		
		for i, item in ipairs(item_data) do
			local item_unique_id = item.item_unique_id
			local item_key = user_key .. item_unique_id
			
			local exists = redis.call('exists', item_key)
			if exists == 1 then
				-- 道具已存在，增加数量
				local current_count = tonumber(redis.call('hget', item_key, 'count'))
				local new_count = current_count + item.count
				redis.call('hset', item_key, 'count', new_count)
				-- 构建返回数据
				local result = {
					item_id = tonumber(redis.call('hget', item_key, 'item_id')),
					item_unique_id = redis.call('hget', item_key, 'item_unique_id'),
					item_type = tonumber(redis.call('hget', item_key, 'item_type')),
					properties = redis.call('hget', item_key, 'properties'),
					count = new_count
				}
				table.insert(results, result)
			else
				-- 道具不存在，创建新道具
				redis.call('hset', item_key, 'item_id', item.item_id)
				redis.call('hset', item_key, 'item_unique_id', item.item_unique_id)
				redis.call('hset', item_key, 'item_type', item.item_type)
				redis.call('hset', item_key, 'properties', item.properties)
				redis.call('hset', item_key, 'count', item.count)
				redis.call('sadd', user_items_set_key, item_unique_id)
				table.insert(results, item)
			end
		end
		
		-- 确保results是数组格式，即使是空表
		local result_json = cjson.encode({success = true, results = results})
		redis.call('set', idempotent_key, result_json, 'EX', 604800)
		return result_json
	`

	items := make([]map[string]interface{}, 0, len(req.ItemAddList))
	for _, itemAdd := range req.ItemAddList {
		// 获取道具配置
		itemId := int(itemAdd.ItemId)
		config, exists := m.itemConfigs[itemId]

		if !exists {
			klog.CtxErrorf(ctx, "[ITEM-ADD-CONFIG-NOT-FOUND] userId: %s, itemId: %d, config not found", userId, itemId)
			return &item.AddItemRsp{
				Code: common.ErrorCode_ITEM_ADD_FAILED,
				Msg:  fmt.Sprintf("Item config not found for itemId: %d", itemId),
			}, nil
		}

		// 根据IsUnique属性生成uniqueid
		var uniqueId string
		if config.IsUnique == 1 {
			// 唯一道具，必须生成新的uniqueid
			uniqueId = IdClient.Generate().String()
			klog.CtxInfof(ctx, "[ITEM-ADD-GENERATE-UNIQUEID] userId: %s, itemId: %d, generated uniqueId: %s", userId, itemId, uniqueId)
		} else {
			// 非唯一道具，使用itemId作为uniqueId
			uniqueId = fmt.Sprintf("%d", itemAdd.ItemId)
			klog.CtxInfof(ctx, "[ITEM-ADD-NON-UNIQUE] userId: %s, itemId: %d, IsUnique: %d", userId, itemId, config.IsUnique)
		}

		// 从配置中获取道具类型和属性
		itemType := int32(itemId) // 使用itemId作为类型
		properties := fmt.Sprintf(`{"item_id": %d, "is_unique": %d}`, itemId, config.IsUnique)

		itemMap := map[string]interface{}{
			"item_id":        itemAdd.ItemId,
			"item_unique_id": uniqueId,
			"item_type":      itemType,
			"properties":     properties,
			"count":          itemAdd.Count,
		}
		items = append(items, itemMap)
	}

	itemsJSON, err := json.Marshal(items)
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-ADD-JSON-ERROR] userId: %s, error: %v", userId, err)
		return &item.AddItemRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON marshaling failed: %v", err),
		}, nil
	}

	idempotentKey := fmt.Sprintf("idempotent:{%s}:%s", userId, req.IdempotentId)
	keys := []string{userKey, idempotentKey}
	args := []interface{}{string(itemsJSON)}

	val, err := m.rdb.Eval(ctx, luaScript, keys, args...).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-ADD-REDIS-ERROR] userId: %s, error: %v", userId, err)
		return &item.AddItemRsp{
			Code: common.ErrorCode_ITEM_REDIS_OPERATION_ERROR,
			Msg:  fmt.Sprintf("Redis operation failed: %v", err),
		}, nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(val.(string)), &response); err != nil {
		klog.CtxErrorf(ctx, "[ITEM-ADD-UNMARSHAL-ERROR] userId: %s, error: %v", userId, err)
		return &item.AddItemRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON unmarshaling failed: %v", err),
		}, nil
	}

	if !response["success"].(bool) {
		errorMsg := response["error"].(string)
		klog.CtxWarnf(ctx, "[ITEM-ADD-FAIL] userId: %s, error: %s", userId, errorMsg)

		if errorMsg == "duplicate idempotent request" {
			return &item.AddItemRsp{
				Code: common.ErrorCode_ITEM_IDEMPOTENT_DUPLICATE,
				Msg:  "Duplicate idempotent request",
			}, nil
		}

		return &item.AddItemRsp{
			Code: common.ErrorCode_ITEM_ADD_FAILED,
			Msg:  errorMsg,
		}, nil
	}

	resultsValue := response["results"]
	results, ok := resultsValue.([]interface{})
	if !ok {
		klog.CtxErrorf(ctx, "[ITEM-ADD-RESULTS-TYPE-ERROR] userId: %s, results is not array: %T", userId, resultsValue)
		return &item.AddItemRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  "Invalid results format",
		}, nil
	}

	resultItemInfos := make([]*item.ItemInfo, 0, len(results))
	for _, r := range results {
		resultMap, ok := r.(map[string]interface{})
		if !ok {
			klog.CtxErrorf(ctx, "[ITEM-ADD-ITEM-TYPE-ERROR] userId: %s, item is not map: %T", userId, r)
			continue
		}

		var itemId, itemType, count float64

		// 处理item_id字段
		switch v := resultMap["item_id"].(type) {
		case string:
			itemId, _ = strconv.ParseFloat(v, 64)
		case float64:
			itemId = v
		}

		// 处理item_type字段
		switch v := resultMap["item_type"].(type) {
		case string:
			itemType, _ = strconv.ParseFloat(v, 64)
		case float64:
			itemType = v
		}

		// 处理count字段
		switch v := resultMap["count"].(type) {
		case string:
			count, _ = strconv.ParseFloat(v, 64)
		case float64:
			count = v
		}

		itemInfo := &item.ItemInfo{
			ItemId:       int32(itemId),
			ItemUniqueId: resultMap["item_unique_id"].(string),
			ItemType:     int32(itemType),
			Properties:   resultMap["properties"].(string),
			Count:        int32(count),
		}
		resultItemInfos = append(resultItemInfos, itemInfo)
	}

	klog.CtxInfof(ctx, "[ITEM-ADD-SUCCESS] userId: %s, addedCount: %d", userId, len(resultItemInfos))

	return &item.AddItemRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
		Data: &item.AddItemRsp_Data{
			ItemInfoList: resultItemInfos,
		},
	}, nil
}

func (m *ItemManager) DeleteItem(ctx context.Context, req *item.DeleteItemReq) (resp *item.DeleteItemRsp, err error) {
	userId := ctx.Value("userId").(string)
	userKey := m.getUserKey(userId)

	klog.CtxInfof(ctx, "[ITEM-DELETE-START] userId: %s, deleteCount: %d, idempotentId: %s", userId, len(req.ItemDeleteList), req.IdempotentId)

	luaScript := `
		local user_key = KEYS[1]
		local idempotent_key = KEYS[2]
		local delete_data = cjson.decode(ARGV[1])
		
		local cached_result = redis.call('get', idempotent_key)
		if cached_result then
			return cached_result
		end
		
		local user_items_set_key = user_key .. 'items'
		
		-- 第一阶段：检查所有道具数量是否足够
		for i, delete_item in ipairs(delete_data) do
			local item_unique_id = delete_item.item_unique_id
			local delete_count = tonumber(delete_item.count)
			local item_key = user_key .. item_unique_id
			
			local exists = redis.call('exists', item_key)
			if exists == 1 then
				local current_count = tonumber(redis.call('hget', item_key, 'count'))
				
				if delete_count > current_count then
					-- 删除数量大于现有数量，删除失败
					local result_json = cjson.encode({success = false, error = 'delete count exceeds available count'})
					redis.call('set', idempotent_key, result_json, 'EX', 604800)
					return result_json
				end
			else
				-- 道具不存在，删除失败
				local result_json = cjson.encode({success = false, error = 'item not found'})
				redis.call('set', idempotent_key, result_json, 'EX', 604800)
				return result_json
			end
		end
		
		-- 第二阶段：执行删除操作
		for i, delete_item in ipairs(delete_data) do
			local item_unique_id = delete_item.item_unique_id
			local delete_count = tonumber(delete_item.count)
			local item_key = user_key .. item_unique_id
			
			local current_count = tonumber(redis.call('hget', item_key, 'count'))
			
			if delete_count == current_count then
				-- 删除数量等于现有数量，删除整个道具
				redis.call('del', item_key)
				redis.call('srem', user_items_set_key, item_unique_id)
			else
				-- 删除数量小于现有数量，减少数量
				local new_count = current_count - delete_count
				redis.call('hset', item_key, 'count', new_count)
			end
		end
		
		local result_json = cjson.encode({success = true})
		redis.call('set', idempotent_key, result_json, 'EX', 604800)
		return result_json
	`

	deleteData := make([]map[string]interface{}, 0, len(req.ItemDeleteList))
	for _, deleteItem := range req.ItemDeleteList {
		deleteData = append(deleteData, map[string]interface{}{
			"item_unique_id": deleteItem.ItemUniqueId,
			"count":          int(deleteItem.Count),
		})
	}

	klog.CtxInfof(ctx, "[ITEM-DELETE-DATA] userId: %s, deleteData: %v", userId, deleteData)

	deleteDataJSON, err := json.Marshal(deleteData)
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-DELETE-JSON-ERROR] userId: %s, error: %v", userId, err)
		return &item.DeleteItemRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON marshaling failed: %v", err),
		}, nil
	}

	idempotentKey := fmt.Sprintf("idempotent:{%s}:%s", userId, req.IdempotentId)
	keys := []string{userKey, idempotentKey}
	args := []interface{}{string(deleteDataJSON)}

	val, err := m.rdb.Eval(ctx, luaScript, keys, args...).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-DELETE-REDIS-ERROR] userId: %s, error: %v", userId, err)
		return &item.DeleteItemRsp{
			Code: common.ErrorCode_ITEM_REDIS_OPERATION_ERROR,
			Msg:  fmt.Sprintf("Redis operation failed: %v", err),
		}, nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(val.(string)), &response); err != nil {
		klog.CtxErrorf(ctx, "[ITEM-DELETE-UNMARSHAL-ERROR] userId: %s, error: %v", userId, err)
		return &item.DeleteItemRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON unmarshaling failed: %v", err),
		}, nil
	}

	if !response["success"].(bool) {
		errorMsg := response["error"].(string)
		klog.CtxWarnf(ctx, "[ITEM-DELETE-FAIL] userId: %s, error: %s", userId, errorMsg)

		if errorMsg == "duplicate idempotent request" {
			return &item.DeleteItemRsp{
				Code: common.ErrorCode_ITEM_IDEMPOTENT_DUPLICATE,
				Msg:  "Duplicate idempotent request",
			}, nil
		}

		return &item.DeleteItemRsp{
			Code: common.ErrorCode_ITEM_DELETE_FAILED,
			Msg:  errorMsg,
		}, nil
	}

	klog.CtxInfof(ctx, "[ITEM-DELETE-SUCCESS] userId: %s, deleteCount: %d", userId, len(req.ItemDeleteList))

	return &item.DeleteItemRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}, nil
}

func (m *ItemManager) GetAllItems(ctx context.Context, req *item.GetAllItemsReq) (resp *item.GetAllItemsRsp, err error) {
	userId := ctx.Value("userId").(string)
	userKey := m.getUserKey(userId)

	klog.CtxInfof(ctx, "[ITEM-GET-ALL-START] userId: %s", userId)

	luaScript := `
		local user_key = KEYS[1]
		local user_items_set_key = user_key .. 'items'
		local results = {}
		
		local item_ids = redis.call('smembers', user_items_set_key)
		for i, item_id in ipairs(item_ids) do
			local item_key = user_key .. item_id
			local exists = redis.call('exists', item_key)
			if exists == 1 then
				local item_data = redis.call('hgetall', item_key)
				local item = {}
				for j = 1, #item_data, 2 do
					item[item_data[j]] = item_data[j+1]
				end
				table.insert(results, item)
			end
		end
		
		return cjson.encode({success = true, results = results})
	`

	keys := []string{userKey}
	val, err := m.rdb.Eval(ctx, luaScript, keys).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-GET-ALL-REDIS-ERROR] userId: %s, error: %v", userId, err)
		return &item.GetAllItemsRsp{
			Code: common.ErrorCode_ITEM_REDIS_OPERATION_ERROR,
			Msg:  fmt.Sprintf("Redis operation failed: %v", err),
		}, nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(val.(string)), &response); err != nil {
		klog.CtxErrorf(ctx, "[ITEM-GET-ALL-UNMARSHAL-ERROR] userId: %s, error: %v", userId, err)
		return &item.GetAllItemsRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON unmarshaling failed: %v", err),
		}, nil
	}

	if !response["success"].(bool) {
		errorMsg := response["error"].(string)
		klog.CtxWarnf(ctx, "[ITEM-GET-ALL-FAIL] userId: %s, error: %s", userId, errorMsg)
		return &item.GetAllItemsRsp{
			Code: common.ErrorCode_FAILED,
			Msg:  errorMsg,
		}, nil
	}

	resultsValue := response["results"]
	results, ok := resultsValue.([]interface{})
	if !ok {
		klog.CtxErrorf(ctx, "[ITEM-GET-ALL-RESULTS-TYPE-ERROR] userId: %s, results is not array: %T", userId, resultsValue)
		return &item.GetAllItemsRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  "Invalid results format",
		}, nil
	}

	resultItemInfos := make([]*item.ItemInfo, 0, len(results))
	for _, r := range results {
		resultMap, ok := r.(map[string]interface{})
		if !ok {
			klog.CtxErrorf(ctx, "[ITEM-ADD-ITEM-TYPE-ERROR] userId: %s, item is not map: %T", userId, r)
			continue
		}

		var itemId, itemType, count float64

		// 处理item_id字段
		switch v := resultMap["item_id"].(type) {
		case string:
			itemId, _ = strconv.ParseFloat(v, 64)
		case float64:
			itemId = v
		}

		// 处理item_type字段
		switch v := resultMap["item_type"].(type) {
		case string:
			itemType, _ = strconv.ParseFloat(v, 64)
		case float64:
			itemType = v
		}

		// 处理count字段
		switch v := resultMap["count"].(type) {
		case string:
			count, _ = strconv.ParseFloat(v, 64)
		case float64:
			count = v
		}

		itemInfo := &item.ItemInfo{
			ItemId:       int32(itemId),
			ItemUniqueId: resultMap["item_unique_id"].(string),
			ItemType:     int32(itemType),
			Properties:   resultMap["properties"].(string),
			Count:        int32(count),
		}
		resultItemInfos = append(resultItemInfos, itemInfo)
	}

	klog.CtxInfof(ctx, "[ITEM-GET-ALL-SUCCESS] userId: %s, itemCount: %d", userId, len(resultItemInfos))

	return &item.GetAllItemsRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
		Data: &item.GetAllItemsRsp_Data{
			ItemInfoList: resultItemInfos,
		},
	}, nil
}

func (m *ItemManager) GetItem(ctx context.Context, req *item.GetItemReq) (resp *item.GetItemRsp, err error) {
	userId := ctx.Value("userId").(string)
	userKey := m.getUserKey(userId)
	itemKey := userKey + req.ItemUniqueId

	klog.CtxInfof(ctx, "[ITEM-GET-SINGLE-START] userId: %s, itemUniqueId: %s", userId, req.ItemUniqueId)

	luaScript := `
		local item_key = KEYS[1]
		local exists = redis.call('exists', item_key)
		
		if exists == 0 then
			return cjson.encode({success = false, error = 'item not found'})
		end
		
		local item_data = redis.call('hgetall', item_key)
		local result = {}
		
		for i = 1, #item_data, 2 do
			result[item_data[i]] = item_data[i+1]
		end
		
		return cjson.encode({success = true, result = result})
	`

	keys := []string{itemKey}
	val, err := m.rdb.Eval(ctx, luaScript, keys).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-GET-SINGLE-REDIS-ERROR] userId: %s, itemUniqueId: %s, error: %v", userId, req.ItemUniqueId, err)
		return &item.GetItemRsp{
			Code: common.ErrorCode_ITEM_REDIS_OPERATION_ERROR,
			Msg:  fmt.Sprintf("Redis operation failed: %v", err),
		}, nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(val.(string)), &response); err != nil {
		klog.CtxErrorf(ctx, "[ITEM-GET-SINGLE-UNMARSHAL-ERROR] userId: %s, itemUniqueId: %s, error: %v", userId, req.ItemUniqueId, err)
		return &item.GetItemRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON unmarshaling failed: %v", err),
		}, nil
	}

	if !response["success"].(bool) {
		errorMsg := response["error"].(string)
		klog.CtxWarnf(ctx, "[ITEM-GET-SINGLE-FAIL] userId: %s, itemUniqueId: %s, error: %s", userId, req.ItemUniqueId, errorMsg)

		if errorMsg == "item not found" {
			return &item.GetItemRsp{
				Code: common.ErrorCode_ITEM_NOT_FOUND,
				Msg:  "Item not found",
			}, nil
		}

		return &item.GetItemRsp{
			Code: common.ErrorCode_FAILED,
			Msg:  errorMsg,
		}, nil
	}

	resultMap := response["result"].(map[string]interface{})
	var itemId, itemType, count float64

	// 处理item_id字段
	switch v := resultMap["item_id"].(type) {
	case string:
		itemId, _ = strconv.ParseFloat(v, 64)
	case float64:
		itemId = v
	}

	// 处理item_type字段
	switch v := resultMap["item_type"].(type) {
	case string:
		itemType, _ = strconv.ParseFloat(v, 64)
	case float64:
		itemType = v
	}

	// 处理count字段
	switch v := resultMap["count"].(type) {
	case string:
		count, _ = strconv.ParseFloat(v, 64)
	case float64:
		count = v
	}

	itemInfo := &item.ItemInfo{
		ItemId:       int32(itemId),
		ItemUniqueId: resultMap["item_unique_id"].(string),
		ItemType:     int32(itemType),
		Properties:   resultMap["properties"].(string),
		Count:        int32(count),
	}

	klog.CtxInfof(ctx, "[ITEM-GET-SINGLE-SUCCESS] userId: %s, itemUniqueId: %s, itemId: %d", userId, req.ItemUniqueId, itemInfo.ItemId)

	return &item.GetItemRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
		Data: &item.GetItemRsp_Data{
			ItemInfo: itemInfo,
		},
	}, nil
}

// DeleteItemById 通过道具id删除道具
func (m *ItemManager) DeleteItemById(ctx context.Context, req *item.DeleteItemByIdReq) (resp *item.DeleteItemByIdRsp, err error) {
	userId := ctx.Value("userId").(string)
	userKey := m.getUserKey(userId)

	klog.CtxInfof(ctx, "[ITEM-DELETE-BY-ID-START] userId: %s, deleteCount: %d, idempotentId: %s", userId, len(req.ItemDeleteList), req.IdempotentId)

	deleteData := make([]map[string]interface{}, 0, len(req.ItemDeleteList))
	for _, deleteItem := range req.ItemDeleteList {
		// 获取道具配置
		itemId := int(deleteItem.ItemId)
		config, exists := m.itemConfigs[itemId]

		if !exists {
			klog.CtxErrorf(ctx, "[ITEM-DELETE-BY-ID-CONFIG-NOT-FOUND] userId: %s, itemId: %d, config not found", userId, itemId)
			return &item.DeleteItemByIdRsp{
				Code: common.ErrorCode_ITEM_DELETE_FAILED,
				Msg:  fmt.Sprintf("Item config not found for itemId: %d", itemId),
			}, nil
		}

		// 如果是唯一道具，不能通过id删除
		if config.IsUnique == 1 {
			klog.CtxErrorf(ctx, "[ITEM-DELETE-BY-ID-UNIQUE-ITEM] userId: %s, itemId: %d, unique item cannot be deleted by id", userId, itemId)
			return &item.DeleteItemByIdRsp{
				Code: common.ErrorCode_ITEM_DELETE_FAILED,
				Msg:  "Unique item cannot be deleted by id",
			}, nil
		}

		// 非唯一道具，使用itemId作为uniqueId
		itemUniqueId := fmt.Sprintf("%d", deleteItem.ItemId)
		deleteData = append(deleteData, map[string]interface{}{
			"item_unique_id": itemUniqueId,
			"count":          int(deleteItem.Count),
		})
	}

	klog.CtxInfof(ctx, "[ITEM-DELETE-BY-ID-DATA] userId: %s, deleteData: %v", userId, deleteData)

	deleteDataJSON, err := json.Marshal(deleteData)
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-DELETE-BY-ID-JSON-ERROR] userId: %s, error: %v", userId, err)
		return &item.DeleteItemByIdRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON marshaling failed: %v", err),
		}, nil
	}

	luaScript := `
		local user_key = KEYS[1]
		local idempotent_key = KEYS[2]
		local delete_data = cjson.decode(ARGV[1])
		
		local cached_result = redis.call('get', idempotent_key)
		if cached_result then
			return cached_result
		end
		
		local user_items_set_key = user_key .. 'items'
		
		-- 第一阶段：检查所有道具数量是否足够
		for i, delete_item in ipairs(delete_data) do
			local item_unique_id = delete_item.item_unique_id
			local delete_count = tonumber(delete_item.count)
			local item_key = user_key .. item_unique_id
			
			local exists = redis.call('exists', item_key)
			if exists == 1 then
				local current_count = tonumber(redis.call('hget', item_key, 'count'))
				
				if delete_count > current_count then
					-- 删除数量大于现有数量，删除失败
					local result_json = cjson.encode({success = false, error = 'delete count exceeds available count'})
					redis.call('set', idempotent_key, result_json, 'EX', 604800)
					return result_json
				end
			else
				-- 道具不存在，删除失败
				local result_json = cjson.encode({success = false, error = 'item not found'})
				redis.call('set', idempotent_key, result_json, 'EX', 604800)
				return result_json
			end
		end
		
		-- 第二阶段：执行删除操作
		for i, delete_item in ipairs(delete_data) do
			local item_unique_id = delete_item.item_unique_id
			local delete_count = tonumber(delete_item.count)
			local item_key = user_key .. item_unique_id
			
			local current_count = tonumber(redis.call('hget', item_key, 'count'))
			
			if delete_count == current_count then
				-- 删除数量等于现有数量，删除整个道具
				redis.call('del', item_key)
				redis.call('srem', user_items_set_key, item_unique_id)
			else
				-- 删除数量小于现有数量，减少数量
				local new_count = current_count - delete_count
				redis.call('hset', item_key, 'count', new_count)
			end
		end
		
		local result_json = cjson.encode({success = true})
		redis.call('set', idempotent_key, result_json, 'EX', 604800)
		return result_json
	`

	idempotentKey := fmt.Sprintf("idempotent:{%s}:%s", userId, req.IdempotentId)
	keys := []string{userKey, idempotentKey}
	args := []interface{}{string(deleteDataJSON)}

	val, err := m.rdb.Eval(ctx, luaScript, keys, args...).Result()
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-DELETE-BY-ID-REDIS-ERROR] userId: %s, error: %v", userId, err)
		return &item.DeleteItemByIdRsp{
			Code: common.ErrorCode_ITEM_REDIS_OPERATION_ERROR,
			Msg:  fmt.Sprintf("Redis operation failed: %v", err),
		}, nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(val.(string)), &response); err != nil {
		klog.CtxErrorf(ctx, "[ITEM-DELETE-BY-ID-UNMARSHAL-ERROR] userId: %s, error: %v", userId, err)
		return &item.DeleteItemByIdRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON unmarshaling failed: %v", err),
		}, nil
	}

	if !response["success"].(bool) {
		errorMsg := response["error"].(string)
		klog.CtxWarnf(ctx, "[ITEM-DELETE-BY-ID-FAIL] userId: %s, error: %s", userId, errorMsg)

		if errorMsg == "duplicate idempotent request" {
			return &item.DeleteItemByIdRsp{
				Code: common.ErrorCode_ITEM_IDEMPOTENT_DUPLICATE,
				Msg:  "Duplicate idempotent request",
			}, nil
		}

		return &item.DeleteItemByIdRsp{
			Code: common.ErrorCode_ITEM_DELETE_FAILED,
			Msg:  errorMsg,
		}, nil
	}

	klog.CtxInfof(ctx, "[ITEM-DELETE-BY-ID-SUCCESS] userId: %s, deleteCount: %d", userId, len(req.ItemDeleteList))

	return &item.DeleteItemByIdRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
	}, nil
}
