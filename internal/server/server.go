package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/google/wire"
	"hephaestus/internal/conf"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(
	NewGRPCServer, NewHTTPServer,
	NewRegistry, NewMiddlewares,
)

type Middlewares []middleware.Middleware

func NewMiddlewares(c *conf.Telemetry) (m Middlewares) {
	m = make(Middlewares, 0, 4)
	m = append(m,
		// In a normal application, calling the function panic() would make the app exit.
		// We want the service running at all time and do not stop at all, so we shall recover from the panic
		// by using the recovery middleware.
		recovery.Recovery(),
		// If the amount of requests exceeded the server's capabilities, we will reduce the number of requests
		// sent to this service.
		ratelimit.Server(),
	)
	// Provide the metric capabilities to the framework. Metrics include the usage of hardware, runtime-related
	// information (e.g. GC STW duration, number of goroutines, etc.), and many other aspects to help the
	// system administrator grasp the overall status quo.
	if c.Metrics.Enabled {
		m = append(m, NewMetricsMiddleware(c.Metrics))
	}
	if c.Traces.Enabled {
		m = append(m, NewTracingMiddleware(c.Traces))
	}
	return
}
