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

// setup 初始化测试环境
func setup(t *testing.T) {
	os.Setenv("CONF_PATH", "../../etc")

	// 加载配置
	if err := common_config.LoadConfig(); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 测试Redis连接
	rdb := common_redis.GetRedis()
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// 清理测试数据
	cleanupTestData(ctx, t)
}

// cleanupTestData 清理测试数据
func cleanupTestData(ctx context.Context, t *testing.T) {
	rdb := common_redis.GetRedis()

	// 删除测试用户的所有道具
	userKey := fmt.Sprintf("item:user:{%s}:*", testUserId)
	keys, err := rdb.Keys(ctx, userKey).Result()
	if err != nil {
		t.Fatalf("Failed to get keys: %v", err)
	}

	if len(keys) > 0 {
		if err := rdb.Del(ctx, keys...).Err(); err != nil {
			t.Fatalf("Failed to delete keys: %v", err)
		}
	}

	// 删除测试用户的道具集合
	setKey := fmt.Sprintf("item:user:{%s}:items", testUserId)
	if err := rdb.Del(ctx, setKey).Err(); err != nil {
		t.Fatalf("Failed to delete set: %v", err)
	}

	// 删除幂等键
	idempotentKey := fmt.Sprintf("idempotent:{%s}:*", testUserId)
	idempotentKeys, err := rdb.Keys(ctx, idempotentKey).Result()
	if err != nil {
		t.Fatalf("Failed to get idempotent keys: %v", err)
	}

	if len(idempotentKeys) > 0 {
		if err := rdb.Del(ctx, idempotentKeys...).Err(); err != nil {
			t.Fatalf("Failed to delete idempotent keys: %v", err)
		}
	}
}

func TestItemManager_AddItem(t *testing.T) {
	// 初始化测试环境
	setup(t)
	defer cleanupTestData(context.Background(), t)

	ctx := context.WithValue(context.Background(), "userId", testUserId)

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
