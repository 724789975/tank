package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"item_manager/kitex_gen/common"
	"item_manager/kitex_gen/item"
	common_redis "item_manager/redis"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/redis/go-redis/v9"
)

const (
	ITEM_KEY_PREFIX = "item:user:{%s}:"
)

type ItemManager struct {
	rdb redis.UniversalClient
}

var itemManager *ItemManager

func GetItemManager() *ItemManager {
	if itemManager == nil {
		itemManager = &ItemManager{
			rdb: common_redis.GetRedis(),
		}
	}
	return itemManager
}

func (m *ItemManager) getUserKey(userId string) string {
	return fmt.Sprintf(ITEM_KEY_PREFIX, userId)
}

func (m *ItemManager) AddItem(ctx context.Context, req *item.AddItemReq) (resp *item.AddItemRsp, err error) {
	userId := ctx.Value("userId").(string)
	userKey := m.getUserKey(userId)

	klog.CtxInfof(ctx, "[ITEM-ADD-START] userId: %s, itemCount: %d, idempotentId: %s", userId, len(req.ItemInfoList), req.IdempotentId)

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
			
			local existing_item = redis.call('get', item_key)
			if existing_item then
				local existing_data = cjson.decode(existing_item)
				existing_data.count = existing_data.count + item.count
				redis.call('set', item_key, cjson.encode(existing_data))
				table.insert(results, existing_data)
			else
				redis.call('set', item_key, cjson.encode(item))
				redis.call('sadd', user_items_set_key, item_unique_id)
				table.insert(results, item)
			end
		end
		
		local result_json = cjson.encode({success = true, results = results})
		redis.call('set', idempotent_key, result_json, 'EX', 604800)
		return result_json
	`

	items := make([]map[string]interface{}, 0, len(req.ItemInfoList))
	for _, itemInfo := range req.ItemInfoList {
		itemMap := map[string]interface{}{
			"item_id":        itemInfo.ItemId,
			"item_unique_id": itemInfo.ItemUniqueId,
			"item_type":      itemInfo.ItemType,
			"properties":     itemInfo.Properties,
			"count":          itemInfo.Count,
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

	results := response["results"].([]interface{})
	resultItemInfos := make([]*item.ItemInfo, 0, len(results))
	for _, r := range results {
		resultMap := r.(map[string]interface{})
		itemInfo := &item.ItemInfo{
			ItemId:       int32(resultMap["item_id"].(float64)),
			ItemUniqueId: resultMap["item_unique_id"].(string),
			ItemType:     int32(resultMap["item_type"].(float64)),
			Properties:   resultMap["properties"].(string),
			Count:        int32(resultMap["count"].(float64)),
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

	klog.CtxInfof(ctx, "[ITEM-DELETE-START] userId: %s, itemIds: %v, idempotentId: %s", userId, req.ItemUniqueIdList, req.IdempotentId)

	luaScript := `
		local user_key = KEYS[1]
		local idempotent_key = KEYS[2]
		local item_ids = cjson.decode(ARGV[1])
		
		local cached_result = redis.call('get', idempotent_key)
		if cached_result then
			return cached_result
		end
		
		local results = {}
		local user_items_set_key = user_key .. 'items'
		
		for i, item_id in ipairs(item_ids) do
			local item_key = user_key .. item_id
			local exists = redis.call('exists', item_key)
			if exists == 1 then
				redis.call('del', item_key)
				redis.call('srem', user_items_set_key, item_id)
				table.insert(results, true)
			else
				table.insert(results, false)
			end
		end
		
		local result_json = cjson.encode({success = true, results = results})
		redis.call('set', idempotent_key, result_json, 'EX', 604800)
		return result_json
	`

	itemIdsJSON, err := json.Marshal(req.ItemUniqueIdList)
	if err != nil {
		klog.CtxErrorf(ctx, "[ITEM-DELETE-JSON-ERROR] userId: %s, error: %v", userId, err)
		return &item.DeleteItemRsp{
			Code: common.ErrorCode_ITEM_JSON_ERROR,
			Msg:  fmt.Sprintf("JSON marshaling failed: %v", err),
		}, nil
	}

	idempotentKey := fmt.Sprintf("idempotent:{%s}:%s", userId, req.IdempotentId)
	keys := []string{userKey, idempotentKey}
	args := []interface{}{string(itemIdsJSON)}

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

	results := response["results"].([]interface{})
	successList := make([]bool, len(results))
	for i, r := range results {
		successList[i] = r.(bool)
	}

	successCount := 0
	for _, success := range successList {
		if success {
			successCount++
		}
	}

	klog.CtxInfof(ctx, "[ITEM-DELETE-SUCCESS] userId: %s, totalCount: %d, successCount: %d", userId, len(successList), successCount)

	return &item.DeleteItemRsp{
		Code: common.ErrorCode_OK,
		Msg:  "success",
		Data: &item.DeleteItemRsp_Data{
			SuccessList: successList,
		},
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
			local item_data = redis.call('get', item_key)
			if item_data then
				table.insert(results, cjson.decode(item_data))
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

	results := response["results"].([]interface{})
	resultItemInfos := make([]*item.ItemInfo, 0, len(results))
	for _, r := range results {
		resultMap := r.(map[string]interface{})
		itemInfo := &item.ItemInfo{
			ItemId:       int32(resultMap["item_id"].(float64)),
			ItemUniqueId: resultMap["item_unique_id"].(string),
			ItemType:     int32(resultMap["item_type"].(float64)),
			Properties:   resultMap["properties"].(string),
			Count:        int32(resultMap["count"].(float64)),
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
		local item_data = redis.call('get', item_key)
		
		if not item_data then
			return cjson.encode({success = false, error = 'item not found'})
		end
		
		return cjson.encode({success = true, result = cjson.decode(item_data)})
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
	itemInfo := &item.ItemInfo{
		ItemId:       int32(resultMap["item_id"].(float64)),
		ItemUniqueId: resultMap["item_unique_id"].(string),
		ItemType:     int32(resultMap["item_type"].(float64)),
		Properties:   resultMap["properties"].(string),
		Count:        int32(resultMap["count"].(float64)),
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
