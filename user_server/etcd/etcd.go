package etcd

import (
	"sync"
	common_config "user_server/config"

	"github.com/cloudwego/kitex/pkg/registry"
	_etcd "github.com/kitex-contrib/registry-etcd"
)

var (
	syncOnce   sync.Once
	etcdClient registry.Registry
)

func GetEtcdClient() registry.Registry {
	syncOnce.Do(func() {
		addrs := make([]string, 0)	
		for _, addr := range common_config.Get("etcd.addrs").([]interface{}) {
				addrs = append(addrs, addr.(string))
			}
		r, err := _etcd.NewEtcdRegistryWithAuth(addrs, common_config.Get("etcd.username").(string), common_config.Get("etcd.password").(string)) // r should not be reused.
		if err != nil {
			panic(err)
		}
		etcdClient = r
	})
	return etcdClient
}