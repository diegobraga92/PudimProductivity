package shared

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TODO: Add wiring, prometheus server, maybe Grafana

type Metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
	dbQueriesTotal   *prometheus.CounterVec
	dbQueryDuration  *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		requestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently in flight",
			},
		),
		dbQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation"},
		),
		dbQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "pg_query_duration_seconds",
				Help:    "Duration of PostgreSQL queries in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
			},
			[]string{"operation"},
		),
	}
}

func (m *Metrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requestsInFlight.Inc()
		start := time.Now()

		wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)
		path := r.URL.Path

		m.requestsTotal.WithLabelValues(r.Method, path, status).Inc()
		m.requestDuration.WithLabelValues(r.Method, path).Observe(duration)
		m.requestsInFlight.Dec()
	})
}

func MetricsHandler() http.HandlerFunc {
	return promhttp.Handler().ServeHTTP
}

func (m *Metrics) RecordDBQuery(operation string) {
	m.dbQueriesTotal.WithLabelValues(operation).Inc()
}

// RecordDBQueryDuration observes the duration of a database query, broken down
// by operation label (e.g. "list_tasks", "create_task", "get_completions").
func (m *Metrics) RecordDBQueryDuration(operation string, duration time.Duration) {
	m.dbQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Hijack preserves http.Hijacker support through the metrics middleware so
// WebSocket upgrades (used by the sync hub) keep working.
func (w *metricsResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hj.Hijack()
}

// Flush forwards flushes to the underlying writer, if supported.
func (w *metricsResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func RegisterMetricsRoutes(r chi.Router, metrics *Metrics) {
	r.Use(metrics.MetricsMiddleware)

	r.Get("/metrics", MetricsHandler())
}

// This should be started on a separate port that is not exposed publicly.
func SetupInternalMetricsServer(metrics *Metrics) *http.Server {
	mux := chi.NewRouter()
	mux.Get("/metrics", MetricsHandler())
	mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return &http.Server{
		Addr:    ":9090",
		Handler: mux,
	}
}
