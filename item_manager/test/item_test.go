package test

import (
	"context"
	"fmt"
	"item_manager/kitex_gen/item"
	"item_manager/logic/manager"
	"testing"
)

func TestAddItem(t *testing.T) {
	ctx := context.Background()

	itemInfoList := []*item.ItemAddInfo{
		{
			ItemId: 1001,
			Count:  5,
		},
		{
			ItemId: 1002,
			Count:  3,
		},
	}

	req := &item.AddItemReq{
		ItemAddList:     itemInfoList,
		IdempotentId:    "test_add_idempotent_123",
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

	itemIds := []*item.ItemDeleteInfo{
		{
			ItemUniqueId: "item_1001_001",
			Count:        5,
		},
		{
			ItemUniqueId: "item_1002_001",
			Count:        3,
		},
	}

	req := &item.DeleteItemReq{
		ItemDeleteList:  itemIds,
		IdempotentId:    "test_delete_idempotent_123",
		OperationReason: "test_delete",
	}

	result, err := manager.GetItemManager().DeleteItem(ctx, req)
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}

	fmt.Printf("DeleteItem result: %v\n", result)

	if result.Code != 0 {
		t.Errorf("Expected %d, got %d", 0, result.Code)
	}
}
