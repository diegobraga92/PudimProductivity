# Graceful Degradation

> How the PudimProductivity stack behaves when each external dependency is
> unavailable, and how it recovers. Each row cites the authoritative design doc.

## Dependency Matrix

| Dependency | Failure mode | Impact | Recovery | Reference |
|------------|--------------|--------|----------|-----------|
| **RabbitMQ** | Down at startup | Backend starts; composite has only the in-memory bus; notifications worker disabled (logged) | Start broker; restart backend (no auto-reconnect yet) | [ADR 005](./adr/005-async-notifications.md), [runbook](./runbooks/rabbitmq-unavailable.md) |
| **RabbitMQ** | Down at runtime | Task API + WebSocket sync unaffected; notifications paused; events published meanwhile are **lost** | Restart broker + backend; next events flow again | [Postmortem 001](./postmortems/001-rabbitmq-outage.md) |
| **PostgreSQL** | Down at startup | Degraded mode: only `/health` (reports degraded) + pomodoro routes; task CRUD unavailable | Restart DB; restart backend | `main.go` degraded-mode branch |
| **PostgreSQL** | Connection pool exhausted | Task API 5xx/timeouts; health may still pass | Cancel stuck queries; restart backend or tune pool | [runbook](./runbooks/db-pool-exhaustion.md), [database-performance.md](./database-performance.md) |
| **Redis** | Down at any time | Task cache degrades to no-op — reads fall through to DB; no data loss | Redis restart; cache auto-reconnects | [ADR 003](./adr/003-caching-strategy.md) |
| **SMTP / Mailpit** | Down | Email send fails → message rejected to DLQ → retried up to 5× → discarded | Restart Mailpit; messages in DLQ at restart get retried by the pump | [ADR 005](./adr/005-async-notifications.md) |
| **FCM** | No credentials / down | Push disabled; email still delivered (senders are independent) | Add `google-services.json` + set `FCM_DEVICE_TOKEN` | [ADR 005](./adr/005-async-notifications.md) |
| **WebSocket clients** | Disconnected | Clients auto-reconnect with exponential backoff; replay from `last_seq` or `stale` → full REST refetch | Self-healing | [ADR 004](./adr/004-websocket-consistency.md), [runbook](./runbooks/ws-disconnect-storm.md) |
| **Rating providers (OMDb / RAWG)** | Down at runtime | Library `score/search` and `score/batch` return 502; the circuit breaker opens (fail-fast) and self-recovers after 30s; item CRUD is unaffected | Provider recovers; breaker half-open probe re-closes | [ADR 007](./adr/007-external-api-integrations.md) |
| **Rating providers (OMDb / RAWG)** | No credentials / flag off | Library `score/search` and `score/batch` return 503 ("not configured"/"disabled"); the score columns stay fully usable for manual entry | Configure providers in the admin UI (Server Settings → score providers, `GET/PUT /api/v1/admin/score-providers`) and enable the `library.score_lookup_enabled` flag | [ADR 007](./adr/007-external-api-integrations.md), [ADR 014](./adr/014-runtime-score-provider-config.md) |

## Design Principles

1. **The database is the system of record.** Caches (Redis), the event bus
   (in-memory), and the WebSocket stream are all conveniences — losing them
   degrades features but never corrupts data.
2. **Interface seams isolate failures.** The composite bus (`CompositeBus`)
   fans to in-memory + RabbitMQ concurrently; a broker failure is logged and
   never blocks the WebSocket fan-out.
3. **Best-effort events.** The in-memory bus is not durable; events published
   while the broker is down are lost. The durable path (outbox table) is a
   documented future item (ADR 005).
4. **Idempotent + retried side effects.** At-least-once delivery with the
   `notifications` table dedup and the bounded DLQ retry pump mean a dependency
   blip results in a retry, not a duplicate or a permanent failure.

## Acceptable Degradation by Phase

| Capability | Normal | RabbitMQ down | Postgres down | Redis down |
|-----------|--------|---------------|---------------|------------|
| Task CRUD | ✅ | ✅ | ❌ | ✅ |
| Real-time sync (WS) | ✅ | ✅ | ❌ | ✅ |
| Notifications (email/push) | ✅ | ❌ (paused, events lost) | ❌ | ✅ |
| Planner | ✅ | ✅ | ❌ | ✅ |
| Focus timer | ✅ | ✅ | ✅ (in-memory) | ✅ |
