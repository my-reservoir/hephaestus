package server

import (
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	hephaestus "hephaestus/api/lua/v1"
	"hephaestus/internal/conf"
	"hephaestus/internal/service"
)

func NewHTTPServer(c *conf.Server, s *service.HephaestusService, m Middlewares) *http.Server {
	opts := []http.ServerOption{
		http.Middleware(m...),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	srv.Handle("/metrics", promhttp.Handler())
	hephaestus.RegisterHephaestusHTTPServer(srv, s)
	return srv
}
