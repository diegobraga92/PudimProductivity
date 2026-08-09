package db

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// metricsTracer is a pgx.QueryTracer that records per-query count and duration
// metrics for every statement executed through the pool — no repository changes
// needed. Operation labels are derived from the SQL (verb + primary table).
type metricsTracer struct {
	metrics *shared.Metrics
}

type queryStartKey struct{}

type queryStartInfo struct {
	sql string
	at  time.Time
}

// TraceQueryStart records the SQL and start time in the context.
func (t *metricsTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, queryStartInfo{sql: data.SQL, at: time.Now()})
}

// TraceQueryEnd records count + duration.
func (t *metricsTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	if info, ok := ctx.Value(queryStartKey{}).(queryStartInfo); ok {
		op := operationFor(info.sql)
		t.metrics.RecordDBQuery(op)
		t.metrics.RecordDBQueryDuration(op, time.Since(info.at))
	}
}

// ConnectPoolWithMetrics is ConnectPool with a query tracer attached, so every
// statement's count + latency is exposed on the :9090 metrics endpoint.
func ConnectPoolWithMetrics(ctx context.Context, dbCfg shared.DatabaseConfig, metrics *shared.Metrics) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dbCfg.URL)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = int32(dbCfg.MaxConns)
	poolConfig.MinConns = int32(dbCfg.MinConns)
	poolConfig.MaxConnLifetime = dbCfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = dbCfg.MaxConnIdleTime
	poolConfig.ConnConfig.Tracer = &metricsTracer{metrics: metrics}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, dbPingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// operationFor derives a stable, low-cardinality label from a SQL statement:
// the verb plus the first table name (e.g. "SELECT * FROM tasks" → "select
// tasks", "INSERT INTO task_completions" → "insert task_completions").
func operationFor(sql string) string {
	s := strings.ToLower(strings.Join(strings.Fields(sql), " "))
	if s == "" {
		return "unknown"
	}

	verb := ""
	switch {
	case strings.HasPrefix(s, "select"):
		verb = "select"
	case strings.HasPrefix(s, "insert"):
		verb = "insert"
	case strings.HasPrefix(s, "update"):
		verb = "update"
	case strings.HasPrefix(s, "delete"):
		verb = "delete"
	case strings.HasPrefix(s, "create"):
		verb = "create"
	case strings.HasPrefix(s, "alter"):
		verb = "alter"
	case strings.HasPrefix(s, "drop"):
		verb = "drop"
	case strings.HasPrefix(s, "with"):
		verb = "cte"
	default:
		verb = "other"
	}

	// Extract the first table identifier: after FROM / INTO / UPDATE, or after
	// TABLE (handling "CREATE TABLE IF NOT EXISTS <name>").
	fields := strings.Fields(s)
	for i, f := range fields {
		if f == "table" && i+1 < len(fields) {
			name := fields[i+1]
			if name == "if" { // CREATE TABLE IF NOT EXISTS <name>
				for j := i + 2; j < len(fields); j++ {
					if fields[j] != "not" && fields[j] != "exists" {
						name = fields[j]
						break
					}
				}
			}
			return verb + " " + strings.Trim(name, "\"`;()")
		}
		if (f == "from" || f == "into" || f == "update") && i+1 < len(fields) {
			return verb + " " + strings.Trim(fields[i+1], "\"`;()")
		}
	}
	return verb
}
