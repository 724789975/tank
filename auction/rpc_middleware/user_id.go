package rpc_middleware

import (
	"context"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/pkg/endpoint"
)

// userIdMetaKey 用于 context 和 metainfo 中传递 userId 的 key
const userIdMetaKey = "userId"

// UserIdServerMiddleware Kitex 服务端中间件，用于从 RPC metadata 中提取 userId 并注入到 context
func UserIdServerMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, req, resp interface{}) error {
		// 从 metainfo 中获取 userId
		if userId, ok := metainfo.GetValue(ctx, userIdMetaKey); ok {
			ctx = context.WithValue(ctx, userIdMetaKey, userId)
		}
		return next(ctx, req, resp)
	}
}

// SetUserIdToContext 将 userId 设置到 context（支持跨 RPC 传递）
func SetUserIdToContext(ctx context.Context, userId string) context.Context {
	ctx = context.WithValue(ctx, userIdMetaKey, userId)
	return metainfo.WithValue(ctx, userIdMetaKey, userId)
}
