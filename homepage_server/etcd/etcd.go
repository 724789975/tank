package etcd

import (
	"homepage_server/config"
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

func InitEtcd() {
	syncOnce.Do(func() {
		addrs := make([]string, 0)

		etcdAddrsVal := config.Get("etcd.addrs")
		if etcdAddrsVal == nil {
			panic("etcd.addrs is nil")
		}
		for _, addr := range etcdAddrsVal.([]interface{}) {
			addrs = append(addrs, addr.(string))
		}

		usernameVal := config.Get("etcd.username")
		username := ""
		if usernameVal != nil {
			username, _ = usernameVal.(string)
		}

		passwordVal := config.Get("etcd.password")
		password := ""
		if passwordVal != nil {
			password, _ = passwordVal.(string)
		}

		r, err := _etcd.NewEtcdRegistryWithAuth(addrs, username, password)
		if err != nil {
			panic(err)
		}
		etcdClient = r

		r2, err := _etcd.NewEtcdResolverWithAuth(addrs, username, password)
		if err != nil {
			panic(err)
		}
		etcdResolver = r2
	})
}

func GetEtcdClient() registry.Registry {
	InitEtcd()
	return etcdClient
}

func GetEtcdResolver() discovery.Resolver {
	InitEtcd()
	return etcdResolver
}
