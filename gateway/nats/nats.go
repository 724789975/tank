package nats

import (
	common_config "gateway/config"
	"sync"

	_nats "github.com/nats-io/nats.go"
)

var (
	nc *_nats.Conn
	once sync.Once
)

func GetNatsConn() *_nats.Conn {
	once.Do(func ()  {
		nc_, err := _nats.Connect(common_config.Get("nats.addr").(string), _nats.UserInfo(common_config.Get("nats.username").(string), common_config.Get("nats.password").(string)))	
		if err != nil {
			panic(err)
		}
		nc = nc_
	})
	return nc
}
