package server

import (
	"context"
	"time"

	"kratos-demo/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.4.0"
)

// NewTracingProvider creates a new OpenTelemetry tracing provider
func NewTracingProvider(c *conf.Bootstrap, logger log.Logger) (*trace.TracerProvider, error) {
	helper := log.NewHelper(logger)
	
	// Create Jaeger exporter
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(c.Telemetry.Tracing.Endpoint),
		),
	)
	if err != nil {
		helper.Errorf("failed to create jaeger exporter: %v", err)
		return nil, err
	}
	
	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("kratos-demo"),
			semconv.ServiceVersionKey.String("v1.0.0"),
		),
	)
	if err != nil {
		helper.Errorf("failed to create resource: %v", err)
		return nil, err
	}
	
	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(int(c.Telemetry.Tracing.Batcher)),
			trace.WithBatchTimeout(5*time.Second),
		),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(c.Telemetry.Tracing.Sampler)),
	)
	
	// Set global tracer provider
	otel.SetTracerProvider(tp)
	
	helper.Info("OpenTelemetry tracing initialized")
	return tp, nil
}