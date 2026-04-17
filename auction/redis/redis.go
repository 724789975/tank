package redis

import (
	"context"
	"auction_module/config"
	"sync"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

var cache struct {
	sync.Once
	rdb redis.UniversalClient
}

func GetRedis() redis.UniversalClient {
	cache.Do(func() {
		addrs := []string{}
		password := ""

		if val, ok := config.Get("redis.addrs").([]interface{}); ok && len(val) > 0 {
			addrs = make([]string, 0)
			for _, addr := range val {
				addrs = append(addrs, addr.(string))
			}
		}

		if val, ok := config.Get("redis.password").(string); ok {
			password = val
		}

		rdb := redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:           addrs,
			Password:        password,
			DB:              0,
			MaxIdleConns:    16,
			ConnMaxIdleTime: time.Minute * 5,
		})
		err := rdb.Ping(context.Background()).Err()
		if err != nil {
			panic("redis not connected:" + err.Error())
		}

		if err := redisotel.InstrumentTracing(rdb); err != nil {
			panic(err)
		}

		if err := redisotel.InstrumentMetrics(rdb); err != nil {
			panic(err)
		}
		cache.rdb = rdb
	})

	return cache.rdb
}