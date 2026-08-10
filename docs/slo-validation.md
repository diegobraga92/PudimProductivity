# SLO Burn-Rate Validation

**Date:** 2026-08-10
**Status:** ✅ Alerting rules fired against a live deployment

This document records the first live validation of the SLO alerting rules in
`infra/prometheus/alerts.yml` (defined in `docs/slo.md`). Previously the rules
were configured but had **never been observed firing** — this exercise closes
that gap.

## Target

| SLO | Alert | Threshold | `for` |
|-----|-------|-----------|-------|
| Task API p95 < 200ms | `TaskApiHighLatency` | p95 > 0.2s over 5m | 5m |
| Task API error rate < 1% | `TaskApiHighErrorRate` | errors > 1% over 10m | 10m |
| Health 99.5% | `HealthEndpointHighErrorRate` | errors > 0.5% over 5m | 5m |

## Setup

1. **Stack**: full `docker-compose` (postgres + redis + rabbitmq + backend)
   booted on `:18080` (host port remapped; the port was occupied by an unrelated
   monitoring stack on `:8080`).
2. **Prometheus**: standalone container on the compose network scraping
   `pudim-backend:9090` (the internal metrics port), loading
   `infra/prometheus/alerts.yml`, with `scrape_interval`/`evaluation_interval`
   = 5s to speed up the demonstration.
3. **Bottleneck**: a real degradation was introduced — the DB pool was shrunk
   to `DATABASE_MAX_CONNS=1` and `REDIS_URL` pointed at a dead Redis so the
   Phase 1 read cache degraded to a no-op. This simulates the documented
   pool-exhaustion scenario (`docs/runbooks/db-pool-exhaustion.md`) and forces
   every read through a single Postgres connection.

## Load

`k6` hammered the task endpoints (`/tasks`, `/tasks/scheduled`,
`/tasks/completions`) with a staged profile ramping to **5000 VUs**, sustaining
peak throughput around **8–10k req/s**.

Observed during the sustained window:

- `http_requests_total{path=~"/api/v1/tasks.*"}` ~8,700–10,000 req/s
- **p95 latency crossed the 200ms SLO** and stayed above it for > 5 minutes
  (peaking around 248ms).

## Alert firing evidence

Prometheus `/api/v1/alerts` (captured at 14:59 UTC):

```json
{
  "labels": { "alertname": "TaskApiHighLatency", "severity": "warning" },
  "state": "firing",
  "activeAt": "2026-08-10T14:53:50.232720703Z",
  "value": 2.48e-01,
  "annotations": {
    "summary": "Task API p95 latency exceeds 200ms",
    "description": "The p95 latency is 0.2479s over the last 5 minutes."
  }
}
```

The rule progressed `pending → firing` after its 5-minute `for` window with p95
sustained above 0.2s — exactly the burn-rate semantics documented in
`docs/slo.md`. After the load stopped, the alert cleared once the metric window
rolled past the breach.

## Findings

1. ✅ **Alerting machinery works end-to-end**: rules load, evaluate every 5s,
   and fire with the expected `for` semantics on a live deployment.
2. ✅ **The SLO is comfortably met at low/moderate load**: at 50 VUs the task
   API p95 was ~5ms; even at 3,000–8,000 req/s (before the pool squeeze) p95
   stayed ~18ms. Breaching the 200ms SLO required a **deliberate degradation**
   (single DB connection + dead cache) — a strong result for the baseline.
3. ⚠️ The `TaskApiHighErrorRate` (10m `for`) and `HealthEndpointHighErrorRate`
   (5m `for`) rules were loaded and evaluated but not force-fired in this run —
   the latency rule is the burn-rate validator for the primary SLO.

## Repeatable run

```sh
# 1. Boot the stack with the degradation (see docker-compose.override.yml notes):
#    DATABASE_MAX_CONNS=1, REDIS_URL=redis://localhost:6399/0
# 2. Run Prometheus with infra/prometheus/alerts.yml against pudim-backend:9090.
# 3. Generate load:
BASE_URL=http://localhost:18080/api/v1 k6 run /tmp/load-max.js
# 4. Watch the alert:
curl 'http://localhost:19090/api/v1/alerts'
```
