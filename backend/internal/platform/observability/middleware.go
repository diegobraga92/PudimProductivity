package observability

import (
	"bufio"
	"net"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is the instrumentation scope for all PudimProductivity spans.
const tracerName = "pudimproductivity/backend"

// TracingMiddleware extracts incoming W3C trace context (if any), starts a
// server span for the request, and records method, URL, and status code.
//
// Register it as the outermost middleware so spans are active for the whole
// chain (metrics, request logging, handlers, and the WebSocket sync hub).
func TracingMiddleware(next http.Handler) http.Handler {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		spanCtx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		span.SetAttributes(
			semconv.HTTPRequestMethodKey.String(r.Method),
			semconv.URLFullKey.String(r.URL.String()),
			semconv.URLPathKey.String(r.URL.Path),
		)
		defer span.End()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(spanCtx))

		span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(rec.status))
	})
}

// statusRecorder captures the response status code so it can be attached to the
// span. It forwards Hijack/Flush so WebSocket upgrades and streaming responses
// keep working through the tracing middleware.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hj.Hijack()
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
