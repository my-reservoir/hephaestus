package server

import (
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	"hephaestus/internal/conf"

	etcdclient "go.etcd.io/etcd/client/v3"
)

func NewRegistry(c *conf.Registry) registry.Registrar {
	etcdClient, err := etcdclient.New(etcdclient.Config{ // Here we instantiate an etcd client
		Endpoints:            c.Endpoints,
		Username:             c.Username,
		Password:             c.Password,
		AutoSyncInterval:     c.AutoSyncInterval.AsDuration(),
		DialTimeout:          c.DialTimeout.AsDuration(),
		DialKeepAliveTimeout: c.DialKeepAliveTimeout.AsDuration(),
	})
	if err != nil {
		panic(err)
	}
	return etcd.New(etcdClient)
}
