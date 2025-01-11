//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"hephaestus/internal/biz"
	"hephaestus/internal/conf"
	"hephaestus/internal/data"
	"hephaestus/internal/server"
	"hephaestus/internal/service"
)

func wireApp(
	*conf.Registry, *conf.Server, *conf.Telemetry, log.Logger,
) (*kratos.App, func(), error) {
	panic(
		wire.Build(
			server.ProviderSet,
			data.ProviderSet,
			biz.ProviderSet,
			service.ProviderSet,
			newApp,
		),
	)
}
