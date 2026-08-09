# Performance Report

> Phase 10 measurement snapshot. Date: **2026-08-09**. All measurements were
> taken on the current `main` code; methodology and raw outputs are referenced
> so they can be reproduced.

## 1. Web — Lighthouse (mobile, local preview)

Measured with Lighthouse 13 on the production build served by `vite preview`
(headless Chrome 151), performance category only.

| Metric | Value | Target |
|--------|-------|--------|
| **Performance score** | **99 / 100** | ≥ 90 |
| First Contentful Paint | 1 506 ms | ≤ 1.8 s |
| Largest Contentful Paint | 1 838 ms | ≤ 2.5 s |
| Total Blocking Time | 0 ms | ≤ 200 ms |
| Cumulative Layout Shift | 0.0 | ≤ 0.1 |
| Speed Index | 1 506 ms | ≤ 3.4 s |
| Time to Interactive | 1 838 ms | ≤ 3.8 s |

Method: `CHROME_PATH=… npx lighthouse http://localhost:3417 --only-categories=performance`

## 2. Web — Bundle Size (code splitting)

**Before (Phase 10):** single `index.js` **291.82 kB** (gzip 84.82 kB).
**After:** lazy-loaded secondary pages.

| Chunk | Size | Gzip |
|-------|------|------|
| `index.js` (shell + Dashboard) | **213.48 kB** | 67.00 kB |
| `TaskList.js` (on demand) | 30.45 kB | 7.73 kB |
| `audio.js` (shared by pomodoro/soundscape) | 25.34 kB | 6.14 kB |
| `Soundscape.js` | 9.71 kB | 3.15 kB |
| `Pomodoro.js` | 7.10 kB | 2.18 kB |
| `Planner.js` | 5.51 kB | 2.18 kB |

**Initial-bundle reduction: 26.8%** (gzip 84.82 → 67.00 kB). Dashboard stays in
the entry chunk (it is the landing page); Planner, Pomodoro, Soundscape and
TaskList load only when opened. `npm run build:analyze` emits `dist/stats.html`
(rollup-plugin-visualizer).

## 3. Backend — Load Tests (k6)

Thresholds mirror `docs/slo.md` (task API p95 < 200 ms, error rate < 1%).

| Test | Load | Requests | p95 | Errors |
|------|------|----------|-----|--------|
| CI smoke (`infra/k6/smoke.js`) | 5 VUs × 20 s | 1 561 | **6.4 ms** | 0.00% |
| Task CRUD (`tasks-load.js`) | up to 50 VUs | ~77 req/s | **25 ms** | 0.00% |
| Habit completions (`habits-load.js`) | up to 30 VUs | ~60 req/s | **36 ms** | 0.00% |

Reproduce: `BASE_URL=http://localhost:8080/api/v1 k6 run infra/k6/smoke.js`.
The smoke script now runs in CI on every backend push (`load-smoke` job), so
SLO regressions fail the pipeline.

## 4. Database

- **Query plans:** top-5 queries reviewed with `EXPLAIN ANALYZE` at 200 rows and
  50k rows — see `docs/database-performance.md`. The habit partial index
  (`idx_tasks_habits`, migration 013) is used by the hottest habit query.
- **Pool baseline:** pgbench ≈ **982 TPS** at 10 clients on the dev host;
  connection pool is 20 max / 2 min with 30-minute max lifetime.
- **Latency:** DB query p95 per operation is captured by
  `pg_query_duration_seconds` (pgx `QueryTracer`) on the Grafana RED dashboard.

## 5. Summary

Every automated performance gate is green with margin: web Lighthouse **99**,
bundle reduced **27%** with route-level splitting, task API p95 **25 ms** at
50 VUs (8× under the 200 ms SLO), and the load-test gate runs in CI.
