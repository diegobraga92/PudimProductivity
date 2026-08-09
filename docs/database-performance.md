# Database Performance Review

> Phase 6 deep-dive: `EXPLAIN ANALYZE` on the top queries, index effectiveness,
> and connection pool analysis for the PudimProductivity PostgreSQL schema.
> **Date:** 2026-08-09

## 1. Scope & Methodology

- **Target:** the five most frequent queries in the task module.
- **Tooling:** `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)` against a seeded
  PostgreSQL 16 (200 rows — current realistic dev scale — then scaled to
  ~50k tasks / 51k completions to expose index behaviour).
- **Queries reviewed:**

| # | Query | Used by |
|---|-------|---------|
| 1 | `SELECT … FROM tasks WHERE list_id IS NULL AND recurrence_days IS NOT NULL ORDER BY created_at DESC` | Habit list (web + mobile) |
| 2 | `SELECT … FROM tasks WHERE id = $1` | Task detail / update / delete |
| 3 | `SELECT … FROM task_completions WHERE completed_date BETWEEN $1 AND $2 ORDER BY task_id, completed_date` | Batch completions (heatmaps, dashboards) |
| 4 | `SELECT … FROM tasks WHERE list_id = $1 ORDER BY created_at DESC` | Task list grouping |
| 5 | `SELECT … FROM tasks WHERE list_id IS NULL ORDER BY created_at DESC` | Default task view |

## 2. Findings at Current Scale (200 tasks / 1k completions)

**All queries are sub-millisecond sequential scans** — at this data volume the
planner correctly prefers seq scans over index lookups (verified: `id = $1`
with a real UUID still chooses `Seq Scan` because the whole table fits in a few
pages). **None of the existing custom indexes have ever been scanned:**

```
indexrelname                  | idx_scan
------------------------------+---------
 idx_tasks_status             |       0
 idx_tasks_list_id            |       0
 idx_task_completions_task_date |     0
```

**Conclusion:** the database is not a bottleneck at current scale; the existing
single-column indexes (`status`, `list_id`) are dead weight today (write
overhead with no reads using them). They are retained as forward-looking but
could be dropped if write throughput ever mattered.

## 3. Findings at ~50k Rows (index behaviour becomes visible)

After seeding 50k tasks + 51k completions and adding **migration 013**, the
planner starts using indexes:

### Query 1 — habit list (partial index `idx_tasks_habits`)
```sql
CREATE INDEX idx_tasks_habits ON tasks (created_at DESC)
  WHERE recurrence_days IS NOT NULL;
```
| Scale | Plan | Cost |
|-------|------|------|
| 200 rows | Seq Scan + Sort | 0.019 ms |
| 50k rows (before) | Seq Scan + Sort (full table filtered) | scans 50k |
| 50k rows (after 013) | **Index Scan** (`idx_tasks_habits`, no sort) | 0.125 ms for 50 rows |

The partial index directly serves the query: index-ordered scan, no explicit
sort. **Verified improvement.**

### Query 3 — batch completions (existing composite index wins)
The `(task_id, completed_date)` unique index serves the date-range query with
index-ordered access (matches the `ORDER BY`), so the standalone
`idx_task_completions_date` added in 013 is **redundant for the current query
shape** — the planner chose the composite index. This is a useful negative
result: the index was added speculatively; the report records that it should be
dropped if write overhead ever matters.

## 4. Connection Pool

### Current configuration (`.env.example`)
| Parameter | Value | Reasoning |
|-----------|-------|-----------|
| `DATABASE_MAX_CONNS` | 20 | ~10 app instances × 2 conns headroom; dev machine |
| `DATABASE_MIN_CONNS` | 2 | Keep warm connections for latency |
| `DATABASE_MAX_CONN_LIFETIME` | 30m | Avoid stale connections after long sessions |
| `DATABASE_MAX_CONN_IDLETIME` | 5m | Return idle connections to the OS |

### Baseline (pgbench, PostgreSQL 16 container, dev laptop)
| Concurrent clients | TPS | Avg latency |
|-------------------|-----|-------------|
| 10 | 982 | 10.2 ms |
| 20 | (scaling, see load-test report) | — |
| 50 | (scaling) | — |

The app's own pool is not a bottleneck at the tested concurrency; the k6 load
tests in `infra/k6/` provide application-level p95/p99 numbers.

**Recommendation:** keep 20/2 for dev. For production, size `max_conns` as
`(peak concurrent requests × avg query count per request × headroom)` and cap it
below the RDS max connections; enable `pgbouncer` only if connection churn
becomes an issue.

## 5. Load-Test Results (k6)

Run against the backend on the 50k-row dataset (docker container, dev laptop).

### Tasks (`infra/k6/tasks-load.js` — list, habits, create, update, scheduled)
| Metric | Result |
|--------|--------|
| Requests | 1,470 (~48 rps) |
| **p95 latency** | **25 ms** (SLO: < 200 ms) |
| **Error rate** | **0%** (SLO: < 1%) |
| Max VUs | 20 |

### Habits (`infra/k6/habits-load.js` — batch completions + complete/uncomplete)
| Metric | Result |
|--------|--------|
| Requests | 700 (~25 rps) |
| **p95 latency** | **36 ms** (SLO: < 200 ms) |
| **Error rate** | **0%** |
| Max VUs | 15 |

**Conclusion:** both hot paths meet the SLOs with >5x headroom on latency, even
against a 50k-row dataset on a laptop. The database is not a bottleneck.

### Test-design note
A first run of the habits test failed the 1% error threshold (2.27%) because all
VUs shared one habit ID — concurrent completes on the same date return 409
(`ErrCompletionAlreadyExists`), which is correct backend behaviour. Fixed by
giving each VU a distinct completion date. The SLO failure was a test artifact,
not a backend defect.

## 6. Slow-Query Visibility

- **New metric:** `pg_query_duration_seconds` (histogram by operation) recorded
  via a `pgx.QueryTracer` (`internal/db/metrics.go`) — every statement through
  the pool is timed, no repository changes.
- **Dashboard:** `infra/grafana/red-dashboard.json` gains a
  "Database Query Latency (p95 per operation)" panel using
  `histogram_quantile(0.95, sum(rate(pg_query_duration_seconds_bucket[5m])) by (le, operation))`.
- **Threshold:** any operation with p95 > 250 ms warrants investigation.

## 7. Summary of Actions

| Action | Status |
|--------|--------|
| `EXPLAIN ANALYZE` on top 5 queries | Done — all sub-ms at current scale |
| Partial index for habit list (`idx_tasks_habits`) | Added (migration 013) — verified used at 50k rows |
| `completed_date` index | Added speculatively (013) — found redundant for current query shape; candidate for removal |
| Pool config documented | Done (20/2 dev; sizing guidance for prod) |
| DB query latency metric + Grafana panel | Added (`pg_query_duration_seconds`) |
| Load tests | `infra/k6/` (see load-test results section) |

## 8. Future Work

- Add pagination to task list queries (the largest omitted optimization; a
  `LIMIT/OFFSET` or keyset pagination would bound every query above).
- Re-run this review at >1M rows.
- Consider `pg_stat_statements` + a Postgres exporter for DB-wide visibility.
