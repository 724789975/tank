package redis

import (
	"homepage_server/config"

	"github.com/redis/go-redis/v9"
)

var client redis.UniversalClient

func InitRedis() {
	addrs := make([]string, 0)
	for _, addr := range config.Get("redis.addrs").([]interface{}) {
		addrs = append(addrs, addr.(string))
	}
	client = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    addrs,
		Password: config.Get("redis.password").(string),
		DB:       0,
	})
}

func GetRedis() redis.UniversalClient {
	return client
}
