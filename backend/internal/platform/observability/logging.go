package observability

import (
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// TraceLogHook injects trace_id / span_id into every zerolog event whose
// context carries an active span. Attach it to the global logger so log lines
// emitted from inside a traced request are correlated with the trace.
//
// The context is read from the event (set via zerolog's .Ctx(ctx) call) using
// Event.GetCtx(); events logged without a context get no trace fields.
type TraceLogHook struct{}

// Run implements zerolog.Hook.
func (TraceLogHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	ctx := e.GetCtx()
	if ctx == nil {
		return
	}
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return
	}
	e.Str("trace_id", sc.TraceID().String())
	e.Str("span_id", sc.SpanID().String())
}
