# Postmortem 001 — RabbitMQ Outage

> Simulated incident (Phase 6). Blameless postmortem; the goal is to learn, not
> to assign fault.

**Date:** 2026-08-09
**Severity:** SEV-3 (partial feature degradation — notifications only; core task
CRUD and real-time sync unaffected)
**Duration:** ~4 minutes (controlled simulation)
**Detected by:** manual simulation script

---

## Summary

The RabbitMQ broker was stopped while the backend was running. Task creation
continued to work (HTTP 201), real-time WebSocket sync continued to push events
to all clients, and the only degraded path was email/push notification delivery,
which paused for the duration and lost the events published while the broker was
down. After restarting RabbitMQ **and the backend**, notifications resumed.

## Timeline

| Time | Event |
|------|-------|
| 17:52 | Baseline: task created → RabbitMQ → worker → email delivered to Mailpit ✓ |
| 17:53 | `docker stop rabbitmq` |
| 17:53 | Task created during outage → HTTP 201; **no email**; log: `composite bus: child publish failed (continuing) error="Exception (504) Reason: "channel/connection is not open""` |
| 17:53 | WebSocket client confirmed **live events still flowing** during the outage |
| 17:54 | RabbitMQ restarted; backend restarted (required — see root cause); new task → email delivered ✓ |

## Impact

- **Unaffected:** HTTP task CRUD, WebSocket real-time sync, WebSocket replay.
- **Affected:** Email + push notification delivery for events published during
  the outage. Those events were **permanently lost** (in-memory bus has no
  durability — the composite logs the failure but does not persist).
- **Verification:** the `notifications` table contains exactly 2 rows (baseline
  + post-recovery) and no row for the during-outage task.

## Root Cause

The RabbitMQ adapter opens one connection + channel at startup
(`internal/rabbitmq/adapter.go`). On connection loss it logs
`rabbitmq connection lost` but does **not reconnect**; subsequent `Publish`
calls fail with `channel/connection is not open`. The notification worker's
consumer channel is also dead. **Recovery therefore requires a backend restart.**

The loss of during-outage events is by design: the event bus is in-memory and
best-effort (ADR 004/005). The composite bus isolates the failure so the sync
hub (in-memory child) never sees it — which is why WebSocket sync stayed up.

## Detection & Alerting

- No alert fired (this is a simulation). In production, the following should
  alert:
  - RabbitMQ connectivity: a `rabbitmq_up`/probe metric or broker health check.
  - Notification lag: `notifications.dlq` depth or a counter of
    `composite bus: child publish failed` log lines.

## Action Items

| # | Action | Priority | Owner |
|---|--------|----------|-------|
| 1 | Add auto-reconnect to the RabbitMQ adapter (re-dial + re-subscribe consumers with backoff) so backend restarts are not required for broker recovery | P1 | Backend |
| 2 | Add a RabbitMQ health check + Prometheus alert (e.g., `rabbitmq_health_down`) | P1 | Ops |
| 3 | Consider an outbox table so events published while the broker is down are not lost (ADR 005 "When to Revisit") | P2 | Backend |
| 4 | Update `docs/runbooks/rabbitmq-unavailable.md` with the current "restart backend after broker recovery" procedure | P0 (done) | Ops |

## Lessons Learned

1. **The interface seam paid off:** the composite bus made the outage invisible
   to the task service and the sync hub — the single code change to isolate the
   broker was `CompositeBus`.
2. **Graceful degradation works, but "recoverable" needs auto-reconnect.**
   Degrading is only half the story; healing without a restart is the other half.
3. **`docker stop` on a `--rm` container removes it** — the simulation's
   "restart RabbitMQ" step required recreating the container, which is fine for
   a simulation but a reminder that the compose stack manages lifecycle for us.
