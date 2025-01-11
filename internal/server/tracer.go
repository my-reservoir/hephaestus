package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.4.0"
	"hephaestus/internal/conf"
)

// NewTracingMiddleware initializes the connection to tracing service endpoints.
//
// The objective of using distributed calling path tracer is to minimize the difficulty of debugging problems of
// intro-service calls. Administrators can easily locate where the problem occurs via user-friendly graphic interfaces
// with the help of tracer framework.
func NewTracingMiddleware(traces *conf.Traces) middleware.Middleware {
	// Create a Jaeger exporter
	exp, err := jaeger.New(
		jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(traces.Endpoint)),
	)
	if err != nil {
		panic(err)
	}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(exp),
		trace.WithResource(resource.NewSchemaless(
			semconv.ServiceNameKey.String("hephaestus-trace"),
		)),
	)
	return tracing.Server(
		tracing.WithTracerProvider(tp),
	)
}
