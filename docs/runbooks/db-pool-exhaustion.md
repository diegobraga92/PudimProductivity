# Runbook — PostgreSQL Connection Pool Exhaustion

Severity: **SEV-1/SEV-2** (task CRUD down; health + pomodoro still serve)

## Symptoms
- Task API returns 5xx; request logs show timeouts or `connection pool exhausted`.
- `GET /api/v1/health` may still return `"db": "connected"` (pool Ping uses a
  short timeout) but task queries stall.
- Prometheus `db_queries_total` flat / `pg_query_duration_seconds` p95 rising.

## Diagnosis
```bash
# Server-side pool usage
curl -s http://localhost:9090/metrics | grep -E 'pgx|db_queries'
# Active backends in Postgres
docker exec <pg> psql -U pudim -d pudimproductivity \
  -c "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"
# Long-running / stuck queries
docker exec <pg> psql -U pudim -d pudimproductivity \
  -c "SELECT pid, now()-query_start AS age, left(query,60) FROM pg_stat_activity WHERE state='active' ORDER BY age DESC LIMIT 10;"
```

## Recovery
1. Identify the stuck/runaway query (above); cancel it:
   `SELECT pg_cancel_backend(<pid>);`
2. If connections are legitimately exhausted, restart the backend to reset the
   pool, or lower `DATABASE_MAX_CONNS` temporarily.
3. If a slow query is the cause, apply the fix from `docs/database-performance.md`
   (indexes, pool sizing) and re-deploy.

## Prevention
- Monitor `pg_query_duration_seconds` (Grafana RED dashboard panel).
- Add pagination to task list queries (largest open optimization).
- Size the pool per `docs/database-performance.md` §4.
