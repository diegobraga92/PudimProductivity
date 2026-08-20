package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// headersCarrier adapts an amqp.Table to propagation.TextMapCarrier so the
// W3C TraceContext propagator can inject/extract traceparent headers on
// RabbitMQ messages.
type headersCarrier amqp.Table

var _ propagation.TextMapCarrier = headersCarrier{}

func (h headersCarrier) Get(key string) string {
	if v, ok := h[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (h headersCarrier) Set(key, value string) {
	h[key] = value
}

func (h headersCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

// injectTrace writes the trace context from ctx into the AMQP headers (a
// no-op when no span is active in ctx).
func injectTrace(ctx context.Context, headers amqp.Table) {
	otel.GetTextMapPropagator().Inject(ctx, headersCarrier(headers))
}

// extractTrace reads the trace context from AMQP headers and returns a context
// carrying it as the current span parent. The returned context carries the
// producer's trace ID, so consumer-side spans and log lines continue the same
// trace across the broker boundary.
func extractTrace(ctx context.Context, headers amqp.Table) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	carrier := headersCarrier(headers)
	// Only touch the context if there is actually a traceparent to extract.
	if carrier.Get("traceparent") == "" {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
