package etcd

import (
	common_config "match_server/config"
	"sync"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/registry"
	_etcd "github.com/kitex-contrib/registry-etcd"
)

var (
	syncOnce     sync.Once
	etcdClient   registry.Registry
	etcdResolver discovery.Resolver
)

func initEtcd() {
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

		r2, err := _etcd.NewEtcdResolverWithAuth(addrs, common_config.Get("etcd.username").(string), common_config.Get("etcd.password").(string)) // r should not be reused.
		if err != nil {
			panic(err)
		}
		etcdResolver = r2
	})
}

func GetEtcdClient() registry.Registry {
	initEtcd()
	return etcdClient
}

func GetEtcdResolver() discovery.Resolver {
	initEtcd()
	return etcdResolver
}
