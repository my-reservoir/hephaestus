package server

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
	v1 "hephaestus/api/lua/v1"
	"hephaestus/internal/conf"
	"hephaestus/internal/service"
)

func NewGRPCServer(c *conf.Server, s *service.HephaestusService, m Middlewares) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(m...),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterHephaestusServer(srv, s)
	return srv
}
