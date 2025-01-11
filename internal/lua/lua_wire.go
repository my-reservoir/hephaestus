package lua

import (
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	etcdclient "go.etcd.io/etcd/client/v3"
	"hephaestus/internal/conf"
)

func NewRegistryDiscovery(c *conf.Registry) {
	cli, err := etcdclient.New(etcdclient.Config{
		Endpoints: c.Endpoints,
	})
	if err != nil {
		panic(err)
	}
	reg = etcd.New(cli)
}
