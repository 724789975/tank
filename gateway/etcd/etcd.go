package etcd

import (
	common_config "gate_way_module/config"
	"time"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/registry"
	_etcd "github.com/kitex-contrib/registry-etcd"
	"github.com/kitex-contrib/registry-etcd/retry"
)

func GetEtcdClient() registry.Registry {
	addrs := make([]string, 0)
	for _, addr := range common_config.Get("etcd.addrs").([]interface{}) {
		addrs = append(addrs, addr.(string))
	}
	r, err := _etcd.NewEtcdRegistryWithRetry(addrs, retry.NewRetryConfig(
		retry.WithMaxAttemptTimes(10),
		retry.WithObserveDelay(20*time.Second),
		retry.WithRetryDelay(5*time.Second),
	), _etcd.WithAuthOpt(common_config.Get("etcd.username").(string), common_config.Get("etcd.password").(string)))
	if err != nil {
		panic(err)
	}
	return r
}

func GetEtcdResolver() discovery.Resolver {
	addrs := make([]string, 0)
	for _, addr := range common_config.Get("etcd.addrs").([]interface{}) {
		addrs = append(addrs, addr.(string))
	}

	r2, err := _etcd.NewEtcdResolverWithAuth(addrs, common_config.Get("etcd.username").(string), common_config.Get("etcd.password").(string)) // r should not be reused.
	if err != nil {
		panic(err)
	}
	return r2
}
