// Package observability wires OpenTelemetry tracing into the backend so every
// HTTP request, WebSocket dispatch, and log line carries a W3C trace ID. The
// trace context also flows through the event bus (see internal/eventbus), ready
// for the Phase 3 RabbitMQ adapter to propagate it via message headers.
package observability

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// Config controls tracing initialization.
type Config struct {
	// ServiceName identifies this service in traces (attribute service.name).
	ServiceName string
	// Version is the build version (attribute service.version).
	Version string
	// OTLPEndpoint is the base URL of an OTLP/HTTP collector (e.g. Jaeger at
	// http://jaeger:4318 or http://localhost:4318). Empty disables OTLP export.
	OTLPEndpoint string
	// Stdout enables a human-readable stdout span exporter. Default true.
	Stdout bool
}

// InitTracing builds a TracerProvider, installs it as the global provider, and
// configures W3C TraceContext + Baggage propagation. It must be called once at
// startup; the returned provider should be Shutdown() on graceful stop.
func InitTracing(ctx context.Context, cfg Config) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
		),
	)
	if err != nil {
		return nil, err
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}

	if cfg.Stdout {
		ex, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(ex)))
	}

	if cfg.OTLPEndpoint != "" {
		// Normalize so users can set a bare collector URL (http://jaeger:4318)
		// without worrying about the OTLP/HTTP path.
		endpoint := strings.TrimRight(cfg.OTLPEndpoint, "/")
		if !strings.HasSuffix(endpoint, "/v1/traces") {
			endpoint += "/v1/traces"
		}
		ex, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(ex)))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return tp, nil
}
