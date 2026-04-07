package manager_test

import (
	"context"
	"fmt"
	common_config "item_manager/config"
	"item_manager/kitex_gen/common"
	"item_manager/kitex_gen/item"
	"item_manager/logic/manager"
	common_redis "item_manager/redis"
	"os"
	"testing"
)

const (
	testUserId = "test_user_123"
)

var testCtx context.Context

// TestMain 在所有测试执行前初始化环境，执行后清理环境
func TestMain(m *testing.M) {
	// 设置环境变量
	os.Setenv("CONF_PATH", "../../etc")

	// 加载配置
	if err := common_config.LoadConfig(); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 测试Redis连接
	rdb := common_redis.GetRedis()
	testCtx = context.WithValue(context.Background(), "userId", testUserId)
	if err := rdb.Ping(testCtx).Err(); err != nil {
		fmt.Printf("Failed to connect to Redis: %v\n", err)
		os.Exit(1)
	}

	// 清理测试数据
	cleanupTestData(testCtx)

	// 运行所有测试
	exitCode := m.Run()

	// 测试结束后再次清理数据
	cleanupTestData(testCtx)

	os.Exit(exitCode)
}

// cleanupTestData 清理测试数据
func cleanupTestData(ctx context.Context) {
	rdb := common_redis.GetRedis()

	// 删除测试用户的所有道具
	userKey := fmt.Sprintf("item:user:{%s}:*", testUserId)
	keys, err := rdb.Keys(ctx, userKey).Result()
	if err != nil {
		fmt.Printf("Failed to get keys: %v\n", err)
		return
	}

	if len(keys) > 0 {
		if err := rdb.Del(ctx, keys...).Err(); err != nil {
			fmt.Printf("Failed to delete keys: %v\n", err)
		}
	}

	// 删除测试用户的道具集合
	setKey := fmt.Sprintf("item:user:{%s}:items", testUserId)
	if err := rdb.Del(ctx, setKey).Err(); err != nil {
		fmt.Printf("Failed to delete set: %v\n", err)
	}

	// 删除幂等键
	idempotentKey := fmt.Sprintf("idempotent:{%s}:*", testUserId)
	idempotentKeys, err := rdb.Keys(ctx, idempotentKey).Result()
	if err != nil {
		fmt.Printf("Failed to get idempotent keys: %v\n", err)
		return
	}

	if len(idempotentKeys) > 0 {
		if err := rdb.Del(ctx, idempotentKeys...).Err(); err != nil {
			fmt.Printf("Failed to delete idempotent keys: %v\n", err)
		}
	}
}

func TestItemManager_AddItem(t *testing.T) {
	// 使用全局测试上下文
	ctx := testCtx

	tests := []struct {
		name    string
		req     *item.AddItemReq
		wantErr bool
		check   func(t *testing.T, resp *item.AddItemRsp)
	}{
		{
			name: "正常添加单个道具",
			req: &item.AddItemReq{
				ItemAddList: []*item.ItemAddInfo{
					{
						ItemId: 1,
						Count:  5,
					},
				},
				IdempotentId:    "test_idempotent_001",
				OperationReason: "test_add_single",
			},
			wantErr: false,
			check: func(t *testing.T, resp *item.AddItemRsp) {
				if resp.Code != common.ErrorCode_OK {
					t.Errorf("Expected code OK, got %v", resp.Code)
				}
				if len(resp.Data.ItemInfoList) != 1 {
					t.Errorf("Expected 1 item, got %d", len(resp.Data.ItemInfoList))
				}
				if resp.Data.ItemInfoList[0].Count != 5 {
					t.Errorf("Expected count 5, got %d", resp.Data.ItemInfoList[0].Count)
				}
			},
		},
		{
			name: "正常添加多个道具",
			req: &item.AddItemReq{
				ItemAddList: []*item.ItemAddInfo{
					{
						ItemId: 2,
						Count:  3,
					},
					{
						ItemId: 3,
						Count:  2,
					},
				},
				IdempotentId:    "test_idempotent_002",
				OperationReason: "test_add_multiple",
			},
			wantErr: false,
			check: func(t *testing.T, resp *item.AddItemRsp) {
				if resp.Code != common.ErrorCode_OK {
					t.Errorf("Expected code OK, got %v", resp.Code)
				}
				if len(resp.Data.ItemInfoList) != 2 {
					t.Errorf("Expected 2 items, got %d", len(resp.Data.ItemInfoList))
				}
			},
		},
		{
			name: "重复添加道具（数量累加）",
			req: &item.AddItemReq{
				ItemAddList: []*item.ItemAddInfo{
					{
						ItemId: 1,
						Count:  3,
					},
				},
				IdempotentId:    "test_idempotent_003",
				OperationReason: "test_add_duplicate",
			},
			wantErr: false,
			check: func(t *testing.T, resp *item.AddItemRsp) {
				if resp.Code != common.ErrorCode_OK {
					t.Errorf("Expected code OK, got %v", resp.Code)
				}
				// 道具1是唯一道具，每次添加都会生成新的uniqueId，所以数量应该是3而不是8
				if resp.Data.ItemInfoList[0].Count != 3 {
					t.Errorf("Expected count 3, got %d", resp.Data.ItemInfoList[0].Count)
				}
			},
		},
		{
			name: "幂等性测试（重复调用相同idempotentId）",
			req: &item.AddItemReq{
				ItemAddList: []*item.ItemAddInfo{
					{
						ItemId: 4,
						Count:  1,
					},
				},
				IdempotentId:    "test_idempotent_004",
				OperationReason: "test_idempotent",
			},
			wantErr: false,
			check: func(t *testing.T, resp *item.AddItemRsp) {
				if resp.Code != common.ErrorCode_OK {
					t.Errorf("Expected code OK, got %v", resp.Code)
				}

				// 验证第一次调用的结果
				if len(resp.Data.ItemInfoList) != 1 {
					t.Errorf("Expected 1 item after first call, got %d", len(resp.Data.ItemInfoList))
				}
				if resp.Data.ItemInfoList[0].Count != 1 {
					t.Errorf("Expected count 1 after first call, got %d", resp.Data.ItemInfoList[0].Count)
				}

				// 第二次调用相同的请求（验证幂等性）
				m := manager.GetItemManager()
				// 创建相同的请求
				secondReq := &item.AddItemReq{
					ItemAddList: []*item.ItemAddInfo{
						{
							ItemId: 4,
							Count:  1,
						},
					},
					IdempotentId:    "test_idempotent_004",
					OperationReason: "test_idempotent",
				}
				secondResp, secondErr := m.AddItem(ctx, secondReq)

				if secondErr != nil {
					t.Errorf("Second call failed: %v", secondErr)
					return
				}

				// 验证第二次调用返回相同的结果
				if secondResp.Code != common.ErrorCode_OK {
					t.Errorf("Expected code OK on second call, got %v", secondResp.Code)
				}

				if len(secondResp.Data.ItemInfoList) != 1 {
					t.Errorf("Expected 1 item after second call, got %d", len(secondResp.Data.ItemInfoList))
				}

				// 关键验证：第二次调用后数量仍然是1，而不是2
				if secondResp.Data.ItemInfoList[0].Count != 1 {
					t.Errorf("Expected count 1 after second call (idempotent), got %d", secondResp.Data.ItemInfoList[0].Count)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := manager.GetItemManager()
			got, gotErr := m.AddItem(ctx, tt.req)

			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("AddItem() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("AddItem() succeeded unexpectedly")
			}

			// 执行检查函数
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

// TestItemManager_DeleteItem 测试删除道具
func TestItemManager_DeleteItem(t *testing.T) {
	// 使用全局测试上下文
	ctx := testCtx

	// 测试场景1：部分删除道具数量
	t.Run("部分删除道具数量", func(t *testing.T) {
		m := manager.GetItemManager()

		// 清理可能存在的测试数据
		cleanupTestData(ctx)

		// 添加测试数据（使用itemId=2）
		addReq := &item.AddItemReq{
			ItemAddList: []*item.ItemAddInfo{
				{
					ItemId: 2,
					Count:  10,
				},
			},
			IdempotentId:    "test_delete_partial_setup",
			OperationReason: "setup_test_data",
		}
		addResp, err := m.AddItem(ctx, addReq)
		if err != nil {
			t.Fatalf("Failed to setup test data: %v", err)
		}
		if addResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to setup test data: %v", addResp.Msg)
		}

		// 执行删除操作（非唯一道具使用itemId作为uniqueId）
		req := &item.DeleteItemReq{
			ItemDeleteList: []*item.ItemDeleteInfo{
				{
					ItemUniqueId: "2",
					Count:        3,
				},
			},
			IdempotentId:    "test_delete_partial",
			OperationReason: "test_delete_partial",
		}
		resp, err := m.DeleteItem(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItem() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_OK {
			t.Fatalf("Expected code OK, got %v", resp.Code)
		}

		// 验证道具数量减少
		getResp, err := m.GetItem(ctx, &item.GetItemReq{ItemUniqueId: "2"})
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		if getResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to get item: %v", getResp.Msg)
		}
		if getResp.Data.ItemInfo.Count != 7 { // 10 - 3 = 7
			t.Fatalf("Expected count 7, got %d", getResp.Data.ItemInfo.Count)
		}

		// 清理测试数据
		cleanupTestData(ctx)
	})

	// 测试场景2：完全删除道具（数量相等）
	t.Run("完全删除道具（数量相等）", func(t *testing.T) {
		m := manager.GetItemManager()

		// 清理可能存在的测试数据
		cleanupTestData(ctx)

		// 添加测试数据（使用itemId=4）
		addReq := &item.AddItemReq{
			ItemAddList: []*item.ItemAddInfo{
				{
					ItemId: 4,
					Count:  5,
				},
			},
			IdempotentId:    "test_delete_complete_setup",
			OperationReason: "setup_test_data",
		}
		addResp, err := m.AddItem(ctx, addReq)
		if err != nil {
			t.Fatalf("Failed to setup test data: %v", err)
		}
		if addResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to setup test data: %v", addResp.Msg)
		}

		// 执行删除操作
		req := &item.DeleteItemReq{
			ItemDeleteList: []*item.ItemDeleteInfo{
				{
					ItemUniqueId: "4",
					Count:        5,
				},
			},
			IdempotentId:    "test_delete_complete",
			OperationReason: "test_delete_complete",
		}
		resp, err := m.DeleteItem(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItem() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_OK {
			t.Fatalf("Expected code OK, got %v", resp.Code)
		}

		// 验证道具已删除
		getResp, err := m.GetItem(ctx, &item.GetItemReq{ItemUniqueId: "4"})
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		if getResp.Code != common.ErrorCode_ITEM_NOT_FOUND {
			t.Fatalf("Expected code ITEM_NOT_FOUND, got %v", getResp.Code)
		}

		// 清理测试数据
		cleanupTestData(ctx)
	})

	// 测试场景3：删除数量大于现有数量（应该失败）
	t.Run("删除数量大于现有数量（应该失败）", func(t *testing.T) {
		m := manager.GetItemManager()

		// 清理可能存在的测试数据
		cleanupTestData(ctx)

		// 添加测试数据（使用itemId=2）
		addReq := &item.AddItemReq{
			ItemAddList: []*item.ItemAddInfo{
				{
					ItemId: 2,
					Count:  10,
				},
			},
			IdempotentId:    "test_delete_exceed_setup",
			OperationReason: "setup_test_data",
		}
		addResp, err := m.AddItem(ctx, addReq)
		if err != nil {
			t.Fatalf("Failed to setup test data: %v", err)
		}
		if addResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to setup test data: %v", addResp.Msg)
		}

		// 执行删除操作
		req := &item.DeleteItemReq{
			ItemDeleteList: []*item.ItemDeleteInfo{
				{
					ItemUniqueId: "2",
					Count:        20, // 大于现有数量10
				},
			},
			IdempotentId:    "test_delete_exceed",
			OperationReason: "test_delete_exceed",
		}
		resp, err := m.DeleteItem(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItem() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_ITEM_DELETE_FAILED {
			t.Fatalf("Expected code ITEM_DELETE_FAILED, got %v", resp.Code)
		}

		// 验证道具数量没有变化
		getResp, err := m.GetItem(ctx, &item.GetItemReq{ItemUniqueId: "2"})
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		if getResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to get item: %v", getResp.Msg)
		}
		if getResp.Data.ItemInfo.Count != 10 {
			t.Fatalf("Expected count 10, got %d", getResp.Data.ItemInfo.Count)
		}

		// 清理测试数据
		cleanupTestData(ctx)
	})

	// 测试场景4：删除不存在的道具
	t.Run("删除不存在的道具", func(t *testing.T) {
		m := manager.GetItemManager()

		// 清理可能存在的测试数据
		cleanupTestData(ctx)

		// 执行删除操作
		req := &item.DeleteItemReq{
			ItemDeleteList: []*item.ItemDeleteInfo{
				{
					ItemUniqueId: "non_existent_item",
					Count:        1,
				},
			},
			IdempotentId:    "test_delete_nonexistent",
			OperationReason: "test_delete_nonexistent",
		}
		resp, err := m.DeleteItem(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItem() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_ITEM_DELETE_FAILED {
			t.Fatalf("Expected code ITEM_DELETE_FAILED, got %v", resp.Code)
		}

		// 清理测试数据
		cleanupTestData(ctx)
	})
}

// TestItemManager_DeleteItemById 测试通过道具id删除道具
func TestItemManager_DeleteItemById(t *testing.T) {
	ctx := context.WithValue(context.Background(), "userId", testUserId)

	// 测试场景1：通过道具id删除非唯一道具（部分删除）
	t.Run("通过道具id删除非唯一道具（部分删除）", func(t *testing.T) {
		m := manager.GetItemManager()

		// 清理可能存在的测试数据
		cleanupTestData(ctx)

		// 添加测试数据（使用itemId=2）
		addReq := &item.AddItemReq{
			ItemAddList: []*item.ItemAddInfo{
				{
					ItemId: 2,
					Count:  10,
				},
			},
			IdempotentId:    "test_delete_by_id_partial_setup",
			OperationReason: "setup_test_data",
		}
		addResp, err := m.AddItem(ctx, addReq)
		if err != nil {
			t.Fatalf("Failed to setup test data: %v", err)
		}
		if addResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to setup test data: %v", addResp.Msg)
		}

		// 执行删除操作
		req := &item.DeleteItemByIdReq{
			ItemDeleteList: []*item.ItemDeleteByIdInfo{
				{
					ItemId: 2,
					Count:  3,
				},
			},
			IdempotentId:    "test_delete_by_id_partial",
			OperationReason: "test_delete_by_id_partial",
		}
		resp, err := m.DeleteItemById(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItemById() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_OK {
			t.Fatalf("Expected code OK, got %v", resp.Code)
		}

		// 验证道具数量减少
		getResp, err := m.GetItem(ctx, &item.GetItemReq{ItemUniqueId: "2"})
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		if getResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to get item: %v", getResp.Msg)
		}
		if getResp.Data.ItemInfo.Count != 7 { // 10 - 3 = 7
			t.Fatalf("Expected count 7, got %d", getResp.Data.ItemInfo.Count)
		}

		// 清理测试数据
		cleanupTestData(ctx)
	})

	// 测试场景2：通过道具id删除非唯一道具（完全删除）
	t.Run("通过道具id删除非唯一道具（完全删除）", func(t *testing.T) {
		m := manager.GetItemManager()

		// 清理可能存在的测试数据
		cleanupTestData(ctx)

		// 添加测试数据（使用itemId=2）
		addReq := &item.AddItemReq{
			ItemAddList: []*item.ItemAddInfo{
				{
					ItemId: 2,
					Count:  10,
				},
			},
			IdempotentId:    "test_delete_by_id_complete_setup",
			OperationReason: "setup_test_data",
		}
		addResp, err := m.AddItem(ctx, addReq)
		if err != nil {
			t.Fatalf("Failed to setup test data: %v", err)
		}
		if addResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to setup test data: %v", addResp.Msg)
		}

		// 执行删除操作
		req := &item.DeleteItemByIdReq{
			ItemDeleteList: []*item.ItemDeleteByIdInfo{
				{
					ItemId: 2,
					Count:  10,
				},
			},
			IdempotentId:    "test_delete_by_id_complete",
			OperationReason: "test_delete_by_id_complete",
		}
		resp, err := m.DeleteItemById(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItemById() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_OK {
			t.Fatalf("Expected code OK, got %v", resp.Code)
		}

		// 验证道具已删除
		getResp, err := m.GetItem(ctx, &item.GetItemReq{ItemUniqueId: "2"})
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		if getResp.Code != common.ErrorCode_ITEM_NOT_FOUND {
			t.Fatalf("Expected code ITEM_NOT_FOUND, got %v", getResp.Code)
		}

		// 清理测试数据
		cleanupTestData(ctx)
	})

	// 测试场景3：通过道具id删除非唯一道具（删除数量大于现有数量，应该失败）
	t.Run("通过道具id删除非唯一道具（删除数量大于现有数量，应该失败）", func(t *testing.T) {
		m := manager.GetItemManager()

		// 清理可能存在的测试数据
		cleanupTestData(ctx)

		// 添加测试数据（使用itemId=2）
		addReq := &item.AddItemReq{
			ItemAddList: []*item.ItemAddInfo{
				{
					ItemId: 2,
					Count:  10,
				},
			},
			IdempotentId:    "test_delete_by_id_exceed_setup",
			OperationReason: "setup_test_data",
		}
		addResp, err := m.AddItem(ctx, addReq)
		if err != nil {
			t.Fatalf("Failed to setup test data: %v", err)
		}
		if addResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to setup test data: %v", addResp.Msg)
		}

		// 执行删除操作
		req := &item.DeleteItemByIdReq{
			ItemDeleteList: []*item.ItemDeleteByIdInfo{
				{
					ItemId: 2,
					Count:  20, // 大于现有数量10
				},
			},
			IdempotentId:    "test_delete_by_id_exceed",
			OperationReason: "test_delete_by_id_exceed",
		}
		resp, err := m.DeleteItemById(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItemById() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_ITEM_DELETE_FAILED {
			t.Fatalf("Expected code ITEM_DELETE_FAILED, got %v", resp.Code)
		}

		// 验证道具数量没有变化
		getResp, err := m.GetItem(ctx, &item.GetItemReq{ItemUniqueId: "2"})
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		if getResp.Code != common.ErrorCode_OK {
			t.Fatalf("Failed to get item: %v", getResp.Msg)
		}
		if getResp.Data.ItemInfo.Count != 10 {
			t.Fatalf("Expected count 10, got %d", getResp.Data.ItemInfo.Count)
		}

		// 清理测试数据
		cleanupTestData(ctx)
	})

	// 测试场景4：尝试删除唯一道具（应该失败）
	t.Run("尝试删除唯一道具（应该失败）", func(t *testing.T) {
		m := manager.GetItemManager()

		// 执行删除操作
		req := &item.DeleteItemByIdReq{
			ItemDeleteList: []*item.ItemDeleteByIdInfo{
				{
					ItemId: 1, // itemId=1 是唯一道具
					Count:  1,
				},
			},
			IdempotentId:    "test_delete_by_id_unique",
			OperationReason: "test_delete_by_id_unique",
		}
		resp, err := m.DeleteItemById(ctx, req)
		if err != nil {
			t.Fatalf("DeleteItemById() failed: %v", err)
		}

		if resp.Code != common.ErrorCode_ITEM_DELETE_FAILED {
			t.Fatalf("Expected code ITEM_DELETE_FAILED, got %v", resp.Code)
		}
		if resp.Msg != "Unique item cannot be deleted by id" {
			t.Fatalf("Expected message 'Unique item cannot be deleted by id', got '%s'", resp.Msg)
		}
	})
}
