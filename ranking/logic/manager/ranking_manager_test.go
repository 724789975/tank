package manager

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"ranking_module/kitex_gen/common"
	"ranking_module/kitex_gen/ranking"
)

func TestRankingManager_UpdateScore(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	manager := &RankingManager{rdb: rdb}

	tests := []struct {
		name      string
		ctx       context.Context
		req       *ranking.UpdateScoreReq
		mockSetup func()
		wantCode  common.ErrorCode
		wantScore int64
		wantRank  int64
	}{
		{
			name: "success update score",
			ctx:  context.WithValue(context.Background(), "userId", "test_user"),
			req: &ranking.UpdateScoreReq{
				Type:      ranking.RankingType_RANKING_TYPE_SCORE,
				Score:     100,
				Timestamp: time.Now().UnixMilli(),
			},
			mockSetup: func() {
				mock.ExpectZAdd("ranking:1", redis.Z{Score: 100, Member: "test_user"}).SetVal(1)
				mock.ExpectZRevRank("ranking:1", "test_user").SetVal(0) // rank 0 means 1st
			},
			wantCode:  common.ErrorCode_OK,
			wantScore: 100,
			wantRank:  1,
		},
		{
			name: "invalid ranking type",
			ctx:  context.WithValue(context.Background(), "userId", "test_user"),
			req: &ranking.UpdateScoreReq{
				Type:      ranking.RankingType_RANKING_TYPE_UNKNOWN,
				Score:     100,
				Timestamp: time.Now().UnixMilli(),
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_TYPE,
		},
		{
			name: "invalid negative score",
			ctx:  context.WithValue(context.Background(), "userId", "test_user"),
			req: &ranking.UpdateScoreReq{
				Type:      ranking.RankingType_RANKING_TYPE_SCORE,
				Score:     -10,
				Timestamp: time.Now().UnixMilli(),
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_SCORE,
		},
		{
			name: "invalid empty user id",
			ctx:  context.WithValue(context.Background(), "userId", ""),
			req: &ranking.UpdateScoreReq{
				Type:      ranking.RankingType_RANKING_TYPE_SCORE,
				Score:     100,
				Timestamp: time.Now().UnixMilli(),
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_USER_ID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			rsp, err := manager.UpdateScore(tt.ctx, tt.req)
			assert.Nil(t, err)
			assert.Equal(t, tt.wantCode, rsp.Code)
			if tt.wantCode == common.ErrorCode_OK {
				assert.Equal(t, tt.wantScore, rsp.Score)
				assert.Equal(t, tt.wantRank, rsp.Rank)
				assert.NotZero(t, rsp.UpdateTime)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRankingManager_GetUserRank(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	manager := &RankingManager{rdb: rdb}

	tests := []struct {
		name           string
		ctx            context.Context
		req            *ranking.GetUserRankReq
		mockSetup      func()
		wantCode       common.ErrorCode
		wantRank       int64
		wantScore      int64
		wantTotalCount int64
	}{
		{
			name: "success get user rank",
			ctx:  context.WithValue(context.Background(), "userId", "test_user"),
			req: &ranking.GetUserRankReq{
				Type: ranking.RankingType_RANKING_TYPE_SCORE,
			},
			mockSetup: func() {
				mock.ExpectZScore("ranking:1", "test_user").SetVal(100)
				mock.ExpectZRevRank("ranking:1", "test_user").SetVal(2) // 3rd place
				mock.ExpectZCount("ranking:1", "100", "100").SetVal(1)
				mock.ExpectZCard("ranking:1").SetVal(10)
			},
			wantCode:       common.ErrorCode_OK,
			wantRank:       3,
			wantScore:      100,
			wantTotalCount: 10,
		},
		{
			name: "user not found",
			ctx:  context.WithValue(context.Background(), "userId", "not_found_user"),
			req: &ranking.GetUserRankReq{
				Type: ranking.RankingType_RANKING_TYPE_SCORE,
			},
			mockSetup: func() {
				mock.ExpectZScore("ranking:1", "not_found_user").RedisNil()
			},
			wantCode: common.ErrorCode_RANKING_USER_NOT_FOUND,
		},
		{
			name: "invalid ranking type",
			ctx:  context.WithValue(context.Background(), "userId", "test_user"),
			req: &ranking.GetUserRankReq{
				Type: ranking.RankingType_RANKING_TYPE_UNKNOWN,
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_TYPE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			rsp, err := manager.GetUserRank(tt.ctx, tt.req)
			assert.Nil(t, err)
			assert.Equal(t, tt.wantCode, rsp.Code)
			if tt.wantCode == common.ErrorCode_OK {
				assert.Equal(t, tt.wantRank, rsp.RankInfo.Rank)
				assert.Equal(t, tt.wantScore, rsp.RankInfo.Score)
				assert.Equal(t, tt.wantTotalCount, rsp.TotalCount)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRankingManager_GetRankRange(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	manager := &RankingManager{rdb: rdb}

	ctx := context.Background()

	tests := []struct {
		name      string
		req       *ranking.GetRankRangeReq
		mockSetup func()
		wantCode  common.ErrorCode
		wantCount int
	}{
		{
			name: "success get rank range",
			req: &ranking.GetRankRangeReq{
				Type:      ranking.RankingType_RANKING_TYPE_SCORE,
				StartRank: 1,
				EndRank:   5,
				Page:      1,
				PageSize:  100,
			},
			mockSetup: func() {
				mock.ExpectZCard("ranking:1").SetVal(100)
				mock.ExpectZRevRangeWithScores("ranking:1", 0, 4).SetVal([]redis.Z{
					{Score: 1000, Member: "user1"},
					{Score: 900, Member: "user2"},
					{Score: 800, Member: "user3"},
					{Score: 700, Member: "user4"},
					{Score: 600, Member: "user5"},
				})
			},
			wantCode:  common.ErrorCode_OK,
			wantCount: 5,
		},
		{
			name: "invalid ranking type",
			req: &ranking.GetRankRangeReq{
				Type:      ranking.RankingType_RANKING_TYPE_UNKNOWN,
				StartRank: 1,
				EndRank:   5,
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_TYPE,
		},
		{
			name: "invalid range - start equals end",
			req: &ranking.GetRankRangeReq{
				Type:      ranking.RankingType_RANKING_TYPE_SCORE,
				StartRank: 5,
				EndRank:   5,
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_RANGE,
		},
		{
			name: "invalid range - negative rank",
			req: &ranking.GetRankRangeReq{
				Type:      ranking.RankingType_RANKING_TYPE_SCORE,
				StartRank: -1,
				EndRank:   5,
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_RANGE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			rsp, err := manager.GetRankRange(ctx, tt.req)
			assert.Nil(t, err)
			assert.Equal(t, tt.wantCode, rsp.Code)
			if tt.wantCode == common.ErrorCode_OK {
				assert.Equal(t, tt.wantCount, len(rsp.Entries))
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRankingManager_BatchGetUserRank(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	manager := &RankingManager{rdb: rdb}

	ctx := context.Background()

	tests := []struct {
		name      string
		req       *ranking.BatchGetUserRankReq
		mockSetup func()
		wantCode  common.ErrorCode
		wantCount int
	}{
		{
			name: "success batch get",
			req: &ranking.BatchGetUserRankReq{
				UserIds: []string{"user1", "user2", "user3"},
				Type:    ranking.RankingType_RANKING_TYPE_SCORE,
			},
			mockSetup: func() {
				mock.ExpectZCard("ranking:1").SetVal(100)
				mock.ExpectZScore("ranking:1", "user1").SetVal(1000)
				mock.ExpectZRevRank("ranking:1", "user1").SetVal(0)
				mock.ExpectZCount("ranking:1", "1000", "1000").SetVal(1)
				mock.ExpectZScore("ranking:1", "user2").SetVal(900)
				mock.ExpectZRevRank("ranking:1", "user2").SetVal(1)
				mock.ExpectZCount("ranking:1", "900", "900").SetVal(1)
				mock.ExpectZScore("ranking:1", "user3").SetVal(800)
				mock.ExpectZRevRank("ranking:1", "user3").SetVal(2)
				mock.ExpectZCount("ranking:1", "800", "800").SetVal(1)
			},
			wantCode:  common.ErrorCode_OK,
			wantCount: 3,
		},
		{
			name: "batch size exceeded",
			req: &ranking.BatchGetUserRankReq{
				UserIds: make([]string, 150),
				Type:    ranking.RankingType_RANKING_TYPE_SCORE,
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_BATCH_LIMIT_EXCEEDED,
		},
		{
			name: "empty user ids",
			req: &ranking.BatchGetUserRankReq{
				UserIds: []string{},
				Type:    ranking.RankingType_RANKING_TYPE_SCORE,
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_USER_ID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			rsp, err := manager.BatchGetUserRank(ctx, tt.req)
			assert.Nil(t, err)
			assert.Equal(t, tt.wantCode, rsp.Code)
			if tt.wantCode == common.ErrorCode_OK {
				assert.Equal(t, tt.wantCount, len(rsp.RankInfos))
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRankingManager_GetRankingStats(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	manager := &RankingManager{rdb: rdb}

	ctx := context.Background()

	tests := []struct {
		name         string
		req          *ranking.GetRankingStatsReq
		mockSetup    func()
		wantCode     common.ErrorCode
		wantTotal    int64
		wantMaxScore int64
	}{
		{
			name: "success get stats",
			req: &ranking.GetRankingStatsReq{
				Type: ranking.RankingType_RANKING_TYPE_SCORE,
			},
			mockSetup: func() {
				mock.ExpectZCard("ranking:1").SetVal(10)
				mock.ExpectZRevRangeWithScores("ranking:1", 0, 0).SetVal([]redis.Z{
					{Score: 1000, Member: "user1"},
				})
				mock.ExpectZRangeWithScores("ranking:1", 0, 0).SetVal([]redis.Z{
					{Score: 100, Member: "user_last"},
				})
				mock.ExpectZRevRangeWithScores("ranking:1", 0, -1).SetVal([]redis.Z{
					{Score: 1000, Member: "user1"},
					{Score: 900, Member: "user2"},
					{Score: 800, Member: "user3"},
					{Score: 700, Member: "user4"},
					{Score: 600, Member: "user5"},
				})
			},
			wantCode:     common.ErrorCode_OK,
			wantTotal:    10,
			wantMaxScore: 1000,
		},
		{
			name: "empty ranking",
			req: &ranking.GetRankingStatsReq{
				Type: ranking.RankingType_RANKING_TYPE_SCORE,
			},
			mockSetup: func() {
				mock.ExpectZCard("ranking:1").SetVal(0)
			},
			wantCode:  common.ErrorCode_OK,
			wantTotal: 0,
		},
		{
			name: "invalid ranking type",
			req: &ranking.GetRankingStatsReq{
				Type: ranking.RankingType_RANKING_TYPE_UNKNOWN,
			},
			mockSetup: func() {},
			wantCode:  common.ErrorCode_RANKING_INVALID_TYPE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			rsp, err := manager.GetRankingStats(ctx, tt.req)
			assert.Nil(t, err)
			assert.Equal(t, tt.wantCode, rsp.Code)
			if tt.wantCode == common.ErrorCode_OK {
				assert.Equal(t, tt.wantTotal, rsp.Stats.TotalUsers)
				if tt.wantMaxScore > 0 {
					assert.Equal(t, tt.wantMaxScore, rsp.Stats.MaxScore)
				}
				assert.NotZero(t, rsp.Stats.LastUpdateTime)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestValidateRankingType(t *testing.T) {
	tests := []struct {
		name     string
		rankType ranking.RankingType
		want     bool
	}{
		{"valid score", ranking.RankingType_RANKING_TYPE_SCORE, true},
		{"valid activity", ranking.RankingType_RANKING_TYPE_ACTIVITY, true},
		{"valid daily", ranking.RankingType_RANKING_TYPE_DAILY, true},
		{"valid weekly", ranking.RankingType_RANKING_TYPE_WEEKLY, true},
		{"valid monthly", ranking.RankingType_RANKING_TYPE_MONTHLY, true},
		{"valid arena", ranking.RankingType_RANKING_TYPE_ARENA, true},
		{"valid level", ranking.RankingType_RANKING_TYPE_LEVEL, true},
		{"invalid unknown", ranking.RankingType_RANKING_TYPE_UNKNOWN, false},
		{"invalid negative", ranking.RankingType(-1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, validateRankingType(tt.rankType))
		})
	}
}

func TestValidateScore(t *testing.T) {
	tests := []struct {
		name  string
		score int64
		want  bool
	}{
		{"positive", 100, true},
		{"zero", 0, true},
		{"negative", -10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, validateScore(tt.score))
		})
	}
}

func TestValidateUserId(t *testing.T) {
	tests := []struct {
		name   string
		userId string
		want   bool
	}{
		{"valid", "user123", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, validateUserId(tt.userId))
		})
	}
}
