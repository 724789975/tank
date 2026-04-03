package test

import (
	"context"
	"fmt"
	"item_manager/logic/manager"
	"item_manager/kitex_gen/item"
	"testing"
)

func TestAddItem(t *testing.T) {
	ctx := context.Background()
	
	itemInfoList := []*item.ItemInfo{
		{
			ItemId:        1001,
			ItemUniqueId:  "item_1001_001",
			ItemType:      1,
			Properties:    `{"level": 1, "rarity": "common"}`,
			Count:         5,
		},
		{
			ItemId:        1002,
			ItemUniqueId:  "item_1002_001",
			ItemType:      2,
			Properties:    `{"level": 2, "rarity": "rare"}`,
			Count:         3,
		},
	}
	
	req := &item.AddItemReq{
		ItemInfoList:   itemInfoList,
		IdempotentId:   "test_add_idempotent_123",
		OperationReason: "test_add",
	}
	
	result, err := manager.GetItemManager().AddItem(ctx, req)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	
	fmt.Printf("AddItem result: %v\n", result)
	
	if len(result.GetData().ItemInfoList) != len(itemInfoList) {
		t.Errorf("Expected %d items, got %d", len(itemInfoList), len(result.GetData().ItemInfoList))
	}
}

func TestDeleteItem(t *testing.T) {
	ctx := context.Background()
	
	itemIds := []string{"item_1001_001", "item_1002_001"}
	
	req := &item.DeleteItemReq{
		ItemUniqueIdList: itemIds,
		IdempotentId:     "test_delete_idempotent_123",
		OperationReason:  "test_delete",
	}
	
	result, err := manager.GetItemManager().DeleteItem(ctx, req)
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
	
	fmt.Printf("DeleteItem result: %v\n", result)
	
	if len(result.GetData().SuccessList) != len(itemIds) {
		t.Errorf("Expected %d results, got %d", len(itemIds), len(result.GetData().SuccessList))
	}
}