package observability

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// GetTraceID returns the W3C trace ID of the active span in ctx, or "" when no
// span is active. Used by the event bus to stamp events with their origin trace.
func GetTraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

// GetSpanID returns the ID of the active span in ctx, or "" when none is active.
func GetSpanID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.SpanID().String()
}
