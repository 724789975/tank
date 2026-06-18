package manager

import (
	"context"
	"fmt"
	"ranking_module/kitex_gen/common"
	"ranking_module/kitex_gen/ranking"
	"ranking_module/redis"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// Redis key前缀
	RANKING_KEY_PREFIX = "ranking:"
	// 最大分页大小
	MAX_PAGE_SIZE = 100
	// 批量查询最大数量
	MAX_BATCH_SIZE = 100
)

// RankingManager 排行榜管理器
type RankingManager struct {
	rdb goredis.UniversalClient
}

var (
	rankingManager     *RankingManager
	onceRankingManager sync.Once
)

// GetRankingManager 获取排行榜管理器实例
func GetRankingManager() *RankingManager {
	onceRankingManager.Do(func() {
		rankingManager = &RankingManager{
			rdb: getRedisClient(),
		}
	})
	return rankingManager
}

// getRedisClient 获取Redis客户端（延迟初始化）
func getRedisClient() goredis.UniversalClient {
	return redis.GetRedis()
}

// getRankingKey 获取排行榜的Redis key
func getRankingKey(rankingType ranking.RankingType) string {
	return fmt.Sprintf("%s%d", RANKING_KEY_PREFIX, int32(rankingType))
}

// validateRankingType 验证排行榜类型
func validateRankingType(rankingType ranking.RankingType) bool {
	return rankingType != ranking.RankingType_RANKING_TYPE_UNKNOWN &&
		rankingType >= ranking.RankingType_RANKING_TYPE_SCORE &&
		rankingType <= ranking.RankingType_RANKING_TYPE_LEVEL
}

// validateScore 验证分数值
func validateScore(score int64) bool {
	return score >= 0
}

// validateUserId 验证用户ID
func validateUserId(userId string) bool {
	return userId != ""
}

// UpdateScore 更新用户分数
// 使用Redis的ZADD命令，确保原子性
func (m *RankingManager) UpdateScore(ctx context.Context, req *ranking.UpdateScoreReq) (resp *ranking.UpdateScoreRsp, err error) {
	resp = &ranking.UpdateScoreRsp{
		Code: common.ErrorCode_OK,
	}

	// 参数验证
	if !validateRankingType(req.Type) {
		resp.Code = common.ErrorCode_RANKING_INVALID_TYPE
		resp.Msg = "无效的排行榜类型"
		klog.CtxErrorf(ctx, "[RANKING-UPDATE-SCORE] invalid ranking type: %d", req.Type)
		return resp, nil
	}

	// 从context获取用户ID
	userId := ctx.Value("userId").(string)
	if !validateUserId(userId) {
		resp.Code = common.ErrorCode_RANKING_INVALID_USER_ID
		resp.Msg = "无效的用户ID"
		klog.CtxErrorf(ctx, "[RANKING-UPDATE-SCORE] invalid user id")
		return resp, nil
	}

	if !validateScore(req.Score) {
		resp.Code = common.ErrorCode_RANKING_INVALID_SCORE
		resp.Msg = "分数必须为非负整数"
		klog.CtxErrorf(ctx, "[RANKING-UPDATE-SCORE] invalid score: %d", req.Score)
		return resp, nil
	}

	// 防重放攻击：检查时间戳
	if req.Timestamp > 0 {
		now := time.Now().UnixMilli()
		if req.Timestamp < now-60000 { // 60秒前的时间戳视为过期
			resp.Code = common.ErrorCode_RANKING_PARAM_OUT_OF_RANGE
			resp.Msg = "请求时间戳已过期"
			klog.CtxErrorf(ctx, "[RANKING-UPDATE-SCORE] timestamp expired: req=%d, now=%d", req.Timestamp, now)
			return resp, nil
		}
	}

	// 获取Redis key
	key := getRankingKey(req.Type)

	// 使用ZADD命令更新分数（原子操作）
	// Redis的ZADD命令本身就是原子性的
	member := userId
	err = m.rdb.ZAdd(ctx, key, goredis.Z{
		Score:  float64(req.Score),
		Member: member,
	}).Err()

	if err != nil {
		resp.Code = common.ErrorCode_RANKING_REDIS_ERROR
		resp.Msg = "Redis操作失败"
		klog.CtxErrorf(ctx, "[RANKING-UPDATE-SCORE] redis zadd error: %s", err.Error())
		return nil, err
	}

	// 获取更新后的排名
	rank, err := m.rdb.ZRevRank(ctx, key, member).Result()
	if err != nil {
		resp.Code = common.ErrorCode_RANKING_REDIS_ERROR
		resp.Msg = "获取排名失败"
		klog.CtxErrorf(ctx, "[RANKING-UPDATE-SCORE] redis zrevrank error: %s", err.Error())
		return nil, err
	}

	// Redis返回的rank是从0开始的，转换为从1开始
	rank = rank + 1

	resp.Score = req.Score
	resp.Rank = rank
	resp.UpdateTime = time.Now().UnixMilli()

	klog.CtxInfof(ctx, "[RANKING-UPDATE-SCORE] success: userId=%s, type=%d, score=%d, rank=%d", userId, req.Type, req.Score, rank)
	return resp, nil
}

// GetUserRank 获取用户当前排名
func (m *RankingManager) GetUserRank(ctx context.Context, req *ranking.GetUserRankReq) (resp *ranking.GetUserRankRsp, err error) {
	resp = &ranking.GetUserRankRsp{
		Code: common.ErrorCode_OK,
	}

	// 参数验证
	if !validateRankingType(req.Type) {
		resp.Code = common.ErrorCode_RANKING_INVALID_TYPE
		resp.Msg = "无效的排行榜类型"
		klog.CtxErrorf(ctx, "[RANKING-GET-USER-RANK] invalid ranking type: %d", req.Type)
		return resp, nil
	}

	// 从context获取用户ID
	userId := ctx.Value("userId").(string)
	if !validateUserId(userId) {
		resp.Code = common.ErrorCode_RANKING_INVALID_USER_ID
		resp.Msg = "无效的用户ID"
		klog.CtxErrorf(ctx, "[RANKING-GET-USER-RANK] invalid user id")
		return resp, nil
	}

	key := getRankingKey(req.Type)

	// 获取用户分数
	score, err := m.rdb.ZScore(ctx, key, userId).Result()
	if err == goredis.Nil {
		resp.Code = common.ErrorCode_RANKING_USER_NOT_FOUND
		resp.Msg = "用户未在排行榜中"
		klog.CtxInfof(ctx, "[RANKING-GET-USER-RANK] user not found: userId=%s", userId)
		return resp, nil
	}
	if err != nil {
		resp.Code = common.ErrorCode_RANKING_REDIS_ERROR
		resp.Msg = "Redis操作失败"
		klog.CtxErrorf(ctx, "[RANKING-GET-USER-RANK] redis zscore error: %s", err.Error())
		return nil, err
	}

	// 获取用户排名
	rank, err := m.rdb.ZRevRank(ctx, key, userId).Result()
	if err == goredis.Nil {
		resp.Code = common.ErrorCode_RANKING_USER_NOT_FOUND
		resp.Msg = "用户未在排行榜中"
		return resp, nil
	}
	if err != nil {
		resp.Code = common.ErrorCode_RANKING_REDIS_ERROR
		resp.Msg = "获取排名失败"
		klog.CtxErrorf(ctx, "[RANKING-GET-USER-RANK] redis zrevrank error: %s", err.Error())
		return nil, err
	}

	// 转换为从1开始的排名
	rank = rank + 1

	// 获取同分数用户数量
	sameScoreCount, err := m.rdb.ZCount(ctx, key, strconv.FormatFloat(score, 'f', -1, 64), strconv.FormatFloat(score, 'f', -1, 64)).Result()
	if err != nil {
		klog.CtxWarnf(ctx, "[RANKING-GET-USER-RANK] redis zcount error: %s", err.Error())
		sameScoreCount = 1 // 如果出错，默认为1
	}

	// 获取排行榜总用户数
	totalCount, err := m.rdb.ZCard(ctx, key).Result()
	if err != nil {
		klog.CtxWarnf(ctx, "[RANKING-GET-USER-RANK] redis zcard error: %s", err.Error())
		totalCount = 0
	}

	resp.RankInfo = &ranking.UserRankInfo{
		UserId:         userId,
		Rank:           rank,
		Score:          int64(score),
		SameScoreCount: sameScoreCount,
	}
	resp.TotalCount = totalCount

	klog.CtxInfof(ctx, "[RANKING-GET-USER-RANK] success: userId=%s, rank=%d, score=%d", userId, rank, int64(score))
	return resp, nil
}

// GetRankRange 获取排名区间
func (m *RankingManager) GetRankRange(ctx context.Context, req *ranking.GetRankRangeReq) (resp *ranking.GetRankRangeRsp, err error) {
	resp = &ranking.GetRankRangeRsp{
		Code: common.ErrorCode_OK,
	}

	// 参数验证
	if !validateRankingType(req.Type) {
		resp.Code = common.ErrorCode_RANKING_INVALID_TYPE
		resp.Msg = "无效的排行榜类型"
		klog.CtxErrorf(ctx, "[RANKING-GET-RANK-RANGE] invalid ranking type: %d", req.Type)
		return resp, nil
	}

	if req.StartRank <= 0 || req.EndRank <= 0 {
		resp.Code = common.ErrorCode_RANKING_INVALID_RANGE
		resp.Msg = "排名必须为正整数"
		klog.CtxErrorf(ctx, "[RANKING-GET-RANK-RANGE] invalid range: start=%d, end=%d", req.StartRank, req.EndRank)
		return resp, nil
	}

	if req.StartRank == req.EndRank {
		resp.Code = common.ErrorCode_RANKING_INVALID_RANGE
		resp.Msg = "起始排名和结束排名不能相同"
		klog.CtxErrorf(ctx, "[RANKING-GET-RANK-RANGE] start equals end: %d", req.StartRank)
		return resp, nil
	}

	// 分页参数处理
	if req.PageSize <= 0 {
		req.PageSize = MAX_PAGE_SIZE
	}
	if req.PageSize > MAX_PAGE_SIZE {
		req.PageSize = MAX_PAGE_SIZE
	}

	if req.Page <= 0 {
		req.Page = 1
	}

	key := getRankingKey(req.Type)

	// 获取总用户数
	totalCount, err := m.rdb.ZCard(ctx, key).Result()
	if err != nil {
		resp.Code = common.ErrorCode_RANKING_REDIS_ERROR
		resp.Msg = "获取排行榜总数失败"
		klog.CtxErrorf(ctx, "[RANKING-GET-RANK-RANGE] redis zcard error: %s", err.Error())
		return nil, err
	}

	// 确定查询范围（支持正向和逆向）
	var start, end int64
	if req.StartRank < req.EndRank {
		// 正向查询
		start = int64(req.StartRank) - 1 // Redis索引从0开始
		end = int64(req.EndRank) - 1
	} else {
		// 逆向查询
		start = int64(req.StartRank) - 1
		end = int64(req.EndRank) - 1
	}

	// 应用分页
	offset := (req.Page - 1) * req.PageSize
	pageStart := start + int64(offset)
	pageEnd := pageStart + int64(req.PageSize) - 1

	// 确保不超过实际范围
	if pageEnd > end {
		pageEnd = end
	}
	if pageStart > end {
		resp.Code = common.ErrorCode_RANKING_PARAM_OUT_OF_RANGE
		resp.Msg = "页码超出范围"
		return resp, nil
	}

	// 使用ZREVRANGE获取排名区间（按分数从高到低）
	results, err := m.rdb.ZRevRangeWithScores(ctx, key, pageStart, pageEnd).Result()
	if err != nil {
		resp.Code = common.ErrorCode_RANKING_REDIS_ERROR
		resp.Msg = "Redis操作失败"
		klog.CtxErrorf(ctx, "[RANKING-GET-RANK-RANGE] redis zrevrange error: %s", err.Error())
		return nil, err
	}

	// 构建响应
	entries := make([]*ranking.RankEntry, 0, len(results))
	for i, z := range results {
		rank := pageStart + int64(i) + 1 // 转换为从1开始的排名
		entries = append(entries, &ranking.RankEntry{
			UserId: z.Member.(string),
			Rank:   rank,
			Score:  int64(z.Score),
		})
	}

	// 计算总页数
	rangeSize := end - start + 1
	totalPages := int32(rangeSize / int64(req.PageSize))
	if rangeSize%int64(req.PageSize) > 0 {
		totalPages++
	}

	resp.Entries = entries
	resp.StartRank = req.StartRank
	resp.EndRank = req.EndRank
	resp.TotalCount = int32(totalCount)
	resp.CurrentPage = req.Page
	resp.TotalPages = totalPages

	klog.CtxInfof(ctx, "[RANKING-GET-RANK-RANGE] success: type=%d, start=%d, end=%d, page=%d, count=%d", req.Type, req.StartRank, req.EndRank, req.Page, len(entries))
	return resp, nil
}

// BatchGetUserRank 批量获取用户排名
func (m *RankingManager) BatchGetUserRank(ctx context.Context, req *ranking.BatchGetUserRankReq) (resp *ranking.BatchGetUserRankRsp, err error) {
	resp = &ranking.BatchGetUserRankRsp{
		Code: common.ErrorCode_OK,
	}

	// 参数验证
	if !validateRankingType(req.Type) {
		resp.Code = common.ErrorCode_RANKING_INVALID_TYPE
		resp.Msg = "无效的排行榜类型"
		klog.CtxErrorf(ctx, "[RANKING-BATCH-GET-USER-RANK] invalid ranking type: %d", req.Type)
		return resp, nil
	}

	if len(req.UserIds) == 0 {
		resp.Code = common.ErrorCode_RANKING_INVALID_USER_ID
		resp.Msg = "用户ID列表为空"
		klog.CtxErrorf(ctx, "[RANKING-BATCH-GET-USER-RANK] empty user ids")
		return resp, nil
	}

	if len(req.UserIds) > MAX_BATCH_SIZE {
		resp.Code = common.ErrorCode_RANKING_BATCH_LIMIT_EXCEEDED
		resp.Msg = fmt.Sprintf("批量查询数量超过限制，最大%d个", MAX_BATCH_SIZE)
		klog.CtxErrorf(ctx, "[RANKING-BATCH-GET-USER-RANK] batch size exceeded: %d", len(req.UserIds))
		return resp, nil
	}

	key := getRankingKey(req.Type)

	// 获取排行榜总用户数
	totalCount, err := m.rdb.ZCard(ctx, key).Result()
	if err != nil {
		klog.CtxWarnf(ctx, "[RANKING-BATCH-GET-USER-RANK] redis zcard error: %s", err.Error())
		totalCount = 0
	}

	// 批量获取用户排名信息
	rankInfos := make([]*ranking.UserRankInfo, 0, len(req.UserIds))
	for _, userId := range req.UserIds {
		if !validateUserId(userId) {
			continue // 跳过无效的用户ID
		}

		// 获取用户分数
		score, err := m.rdb.ZScore(ctx, key, userId).Result()
		if err == goredis.Nil {
			// 用户未在排行榜中，跳过
			continue
		}
		if err != nil {
			klog.CtxWarnf(ctx, "[RANKING-BATCH-GET-USER-RANK] redis zscore error for user %s: %s", userId, err.Error())
			continue
		}

		// 获取用户排名
		rank, err := m.rdb.ZRevRank(ctx, key, userId).Result()
		if err != nil {
			klog.CtxWarnf(ctx, "[RANKING-BATCH-GET-USER-RANK] redis zrevrank error for user %s: %s", userId, err.Error())
			continue
		}

		// 转换为从1开始的排名
		rank = rank + 1

		// 获取同分数用户数量
		sameScoreCount, err := m.rdb.ZCount(ctx, key, strconv.FormatFloat(score, 'f', -1, 64), strconv.FormatFloat(score, 'f', -1, 64)).Result()
		if err != nil {
			sameScoreCount = 1
		}

		rankInfos = append(rankInfos, &ranking.UserRankInfo{
			UserId:         userId,
			Rank:           rank,
			Score:          int64(score),
			SameScoreCount: sameScoreCount,
		})
	}

	resp.RankInfos = rankInfos
	resp.TotalCount = totalCount

	klog.CtxInfof(ctx, "[RANKING-BATCH-GET-USER-RANK] success: type=%d, requested=%d, found=%d", req.Type, len(req.UserIds), len(rankInfos))
	return resp, nil
}

// GetRankingStats 获取排行榜统计信息
func (m *RankingManager) GetRankingStats(ctx context.Context, req *ranking.GetRankingStatsReq) (resp *ranking.GetRankingStatsRsp, err error) {
	resp = &ranking.GetRankingStatsRsp{
		Code: common.ErrorCode_OK,
	}

	// 参数验证
	if !validateRankingType(req.Type) {
		resp.Code = common.ErrorCode_RANKING_INVALID_TYPE
		resp.Msg = "无效的排行榜类型"
		klog.CtxErrorf(ctx, "[RANKING-GET-STATS] invalid ranking type: %d", req.Type)
		return resp, nil
	}

	key := getRankingKey(req.Type)

	// 获取总用户数
	totalUsers, err := m.rdb.ZCard(ctx, key).Result()
	if err != nil {
		resp.Code = common.ErrorCode_RANKING_REDIS_ERROR
		resp.Msg = "获取排行榜总数失败"
		klog.CtxErrorf(ctx, "[RANKING-GET-STATS] redis zcard error: %s", err.Error())
		return nil, err
	}

	if totalUsers == 0 {
		// 排行榜为空
		resp.Stats = &ranking.RankingStats{
			TotalUsers:     0,
			MaxScore:       0,
			MinScore:       0,
			AvgScore:       0,
			LastUpdateTime: time.Now().UnixMilli(),
		}
		return resp, nil
	}

	// 获取最高分（排名第一的用户）
	maxResult, err := m.rdb.ZRevRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil || len(maxResult) == 0 {
		klog.CtxWarnf(ctx, "[RANKING-GET-STATS] redis zrevrange error for max: %s", err.Error())
	}
	var maxScore int64 = 0
	if len(maxResult) > 0 {
		maxScore = int64(maxResult[0].Score)
	}

	// 获取最低分（排名最后的用户）
	minResult, err := m.rdb.ZRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil || len(minResult) == 0 {
		klog.CtxWarnf(ctx, "[RANKING-GET-STATS] redis zrange error for min: %s", err.Error())
	}
	var minScore int64 = 0
	if len(minResult) > 0 {
		minScore = int64(minResult[0].Score)
	}

	// 计算平均分（需要遍历所有分数）
	// 使用Lua脚本计算平均值会更高效，这里简化处理
	var avgScore int64 = 0
	if totalUsers > 0 {
		// 获取所有分数并计算平均值
		allScores, err := m.rdb.ZRevRangeWithScores(ctx, key, 0, -1).Result()
		if err != nil {
			klog.CtxWarnf(ctx, "[RANKING-GET-STATS] redis zrevrange error for all: %s", err.Error())
		} else {
			var sum int64 = 0
			for _, z := range allScores {
				sum += int64(z.Score)
			}
			avgScore = sum / totalUsers
		}
	}

	resp.Stats = &ranking.RankingStats{
		TotalUsers:     totalUsers,
		MaxScore:       maxScore,
		MinScore:       minScore,
		AvgScore:       avgScore,
		LastUpdateTime: time.Now().UnixMilli(),
	}

	klog.CtxInfof(ctx, "[RANKING-GET-STATS] success: type=%d, total=%d, max=%d, min=%d, avg=%d", req.Type, totalUsers, maxScore, minScore, avgScore)
	return resp, nil
}
