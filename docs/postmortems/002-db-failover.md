# Postmortem 002: Database Outage (Simulated Failover)

**Date:** 2026-08-09
**Severity:** SEV-2 (simulated)
**Type:** Infrastructure / dependency outage
**Simulated in:** Phase 10 final incident exercise (see `docs/capacity-planning.md`
and `docs/runbooks/db-pool-exhaustion.md`)

## Summary

We stopped the PostgreSQL container backing a running backend and observed the
system's behaviour for the duration of the outage, then restarted the database
and confirmed automatic recovery. The backend degraded **cleanly and predictably**
(health flipped to 503/degraded, API returned 500s with clear errors) and
**recovered without a process restart** once the database came back.

## Impact

| Window | Effect |
|--------|--------|
| During outage | `GET /api/v1/health` → **503** `{"status":"degraded","db":"disconnected"}`; task API → **500** `{"error":"failed to list tasks"}` |
| After DB restart | Health → **200** `{"status":"ok","db":"connected"}`; task API → **200** — no backend restart required |
| Data | None lost (database was the only component stopped; its data volume persisted) |

## Timeline

| Time (sim) | Event |
|-----------|-------|
| 18:35:33 | Backend running, healthy (`db=connected`), DB on port 5546 |
| 18:36:1x | **DB stopped** (`docker stop` simulates failover / host crash) |
| +2s | Health endpoint returns **503 degraded**; task list returns **500** |
| 18:36:17 | Backend logs `ERR ... failed to list tasks error="list tasks: failed to connect ... connection refused"` |
| 18:37:0x | **DB restarted** (`docker start`) |
| +4s | Health returns **200 ok / connected**; task list returns **200** — pool reconnected transparently |

## Root Cause

**Not a code defect.** A PostgreSQL outage is an external dependency failure.
The failure mode is fully anticipated by `docs/graceful-degradation.md`
(database down → API errors, health degraded) and the health endpoint's
`db: connected/disconnected` reporting.

## What Worked Well

1. **Health signal quality** — the health endpoint distinguished `ok` vs
   `degraded`/`disconnected` (503), so a load balancer would remove the instance
   from rotation during DB outages.
2. **Automatic pool recovery** — pgx re-established connections after the
   database returned; no backend restart, no manual intervention.
3. **Clear error surfaces** — API errors were structured
   (`{"error":"failed to list tasks"}`) and logged with the underlying dial
   error for debugging.
4. **Graceful degradation is documented** — runbook `db-pool-exhaustion.md`
   covers the related failure; this exercise validated the restore path end-to-end.

## Gaps & Findings

| # | Finding | Severity | Action |
|---|---------|----------|--------|
| 1 | **No health-driven load-balancer integration** in the compose deployment (health is exposed but nothing consumes it to drop traffic) | P2 | When a real ingress/LB exists (ADR 006 forward path), wire the 503 health response to instance drain |
| 2 | **DB failover to a standby is not configured** — single-host compose has one Postgres; a host loss is an outage until restore | P2 (accepted for MVP scale) | Documented restore path in runbook; RDS Multi-AZ at S2 scale (see capacity plan) |
| 3 | **Startup logs show the RabbitMQ fallback default** (`@rabbitmq:5672`) when `RABBITMQ_URL` is empty — noisy but harmless | P3 | Consider explicit `RABBITMQ_URL=""` handling in `main.go` to skip the fallback when not using compose |
| 4 | `DATABASE_MAX_CONNS` (20) exhausted during a slow-outage window would manifest as pool errors — no backpressure alert yet | P2 | Add a Prometheus alert on pool queue/acquire duration (see `docs/runbooks/db-pool-exhaustion.md`) |

## Corrective Actions (tracked)

- [ ] Wire health 503 → instance drain in the deployment layer when an ingress exists (ADR 006 checklist).
- [ ] Add a Prometheus alert for DB connection-pool acquire time (covers #4).
- [ ] Suppress the noisy RabbitMQ fallback warning when `RABBITMQ_URL` is empty.

## Lesson Learned

The modular monolith's **single in-process connection pool** makes DB failover
transparent at the code level: the pool re-dials, health re-reports, and the API
resumes. The operational gap is *detection and routing* (alerts, LB drain),
which is exactly what the Phase 10 runbooks and capacity plan address — the code
already did its job.
