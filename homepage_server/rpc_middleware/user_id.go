package rpc_middleware

import (
	"context"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/pkg/endpoint"
)

const userIdMetaKey = "userId"

func UserIdServerMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, req, resp interface{}) error {
		if userId, ok := metainfo.GetValue(ctx, userIdMetaKey); ok {
			ctx = context.WithValue(ctx, userIdMetaKey, userId)
		}
		return next(ctx, req, resp)
	}
}

func SetUserIdToContext(ctx context.Context, userId string) context.Context {
	ctx = context.WithValue(ctx, userIdMetaKey, userId)
	return metainfo.WithValue(ctx, userIdMetaKey, userId)
}
