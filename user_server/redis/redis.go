package common_redis

import (
	"context"
	"sync"
	"time"
	common_config "user_server/config"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

var cache struct {
	sync.Once
	rdb redis.UniversalClient
}

func GetRedis() redis.UniversalClient {
	cache.Do(func() {
		rdb := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:           common_config.Get("redis.addrs").([]string),
			Password:        common_config.Get("redis.password").(string),
			DB:              0,
			MaxIdleConns:    16,
			ConnMaxIdleTime: time.Minute * 5,
		})
		err := rdb.Ping(context.Background()).Err()
		if err != nil {
			panic("redis not connected:" + err.Error())
		}

		// 开启 tracing instrumentation.
		if err := redisotel.InstrumentTracing(rdb); err != nil {
			panic(err)
		}

		// 开启 metrics instrumentation.
		if err := redisotel.InstrumentMetrics(rdb); err != nil {
			panic(err)
		}
		cache.rdb = rdb
	})

	return cache.rdb
}
