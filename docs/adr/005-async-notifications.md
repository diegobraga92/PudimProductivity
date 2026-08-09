# ADR 005: Asynchronous Notifications via RabbitMQ

**Status:** Accepted
**Date:** 2026-08-09
**Author:** Diego Braga

## Context

Phase 2 delivered real-time sync over WebSocket using an in-memory event bus.
The WebSocket hub needs low-latency fan-out, but durable, asynchronous
consumers (notifications) require a broker. Phase 3 introduces RabbitMQ as the
event backbone: task events must be delivered to a notifications worker that
sends email (Mailpit locally) and push notifications (Firebase Cloud Messaging).

Requirements:

- **Same `Bus` interface** as Phase 2 so producers (task service) don't change.
- **Real-time fan-out must not be blocked** by a slow/down broker.
- **At-least-once delivery** with **idempotent consumers** and **bounded retry**.
- **Trace propagation** across the broker (continuation of Phase 6 OTEL).
- **Graceful degradation** when RabbitMQ, SMTP, or FCM are unavailable.

## Decision

### 1. CompositeBus fans to in-memory + RabbitMQ

`internal/eventbus/composite.go` implements the same `Bus` interface and fans
`Publish` to multiple child buses **concurrently in goroutines**. The task
service keeps a single reference; the sync hub subscribes to the in-memory bus,
the notifications worker to the RabbitMQ bus. A child failure (e.g. RabbitMQ
down) is logged, never propagated — WebSocket fan-out is never delayed.

### 2. RabbitMQ topology

- Exchange `task.events` (fanout, durable) — every task event.
- Queue `notifications` (durable) bound to `task.events`, with
  `x-dead-letter-exchange: task.events.dlx`.
- Exchange `task.events.dlx` + queue `notifications.dlq` — holds failed
  messages.
- **No queue-level TTL retry loop** — RabbitMQ's dead-letter cycle protection
  drops messages that would re-enter a queue they already passed through.
  Instead, a **retry pump** in the adapter consumes from the DLQ and
  republishes to the main exchange with an explicit `x-retry-count` header,
  up to `MaxRetries` (default 5), then discards.

### 3. Idempotent consumers

RabbitMQ gives at-least-once delivery: a message can be redelivered if the
worker crashes between sending and acking. The worker dedupes using the message
ID (`event.ID` = AMQP `message_id`) + channel in the `notifications` table
(`UNIQUE(event_id, channel)`). `MarkSent` uses `ON CONFLICT DO NOTHING`, so a
redelivery is a no-op.

### 4. Trace propagation

On publish, the W3C `traceparent` is injected into AMQP headers. On consume,
the adapter extracts it back into the context before invoking the handler, so
the worker's spans and log lines continue the producer's trace (verified:
HTTP request → event bus → RabbitMQ header → worker log all share the same
`trace_id`). The retry pump copies the original headers, preserving the trace
across retries.

### 5. Graceful degradation

| Dependency down | Behaviour |
|-----------------|-----------|
| RabbitMQ at startup | Backend starts; composite has only the in-memory bus; worker disabled (logged) |
| RabbitMQ at runtime | Publish fails in the goroutine → logged; sync unaffected; messages lost (in-memory bus has no durability) |
| SMTP / Mailpit | Worker send fails → message rejected to DLQ → retried up to 5× → discarded |
| FCM (no credentials) | `FCMSender` becomes a no-op; email still delivered |
| Notifications table | `MarkSent`/`AlreadySent` fail → send is retried via DLQ (may duplicate; accepted) |

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Single bus everywhere** | Simpler | Slow broker would block WebSocket fan-out |
| **Outbox table (DB-published events)** | Durable publish | More moving parts; overkill while publish path is in-process |
| **Queue-level TTL retry loop** | No extra code | RabbitMQ cycle protection silently drops messages — rejected after testing |
| **Only in-memory bus** | Zero infra | No durability, no notifications |

## Consequences

**Positive:**

- Notifications pipeline is durable, retried, idempotent, and traced.
- Task service + sync hub unchanged from Phase 2 (interface seam held).
- Verified end-to-end: task create → RabbitMQ → worker → Mailpit email;
  failed sends retry 5× then discard.

**Negative:**

- RabbitMQ is a new always-on dependency (in `docker-compose.yml`).
- In-memory bus events are lost on server restart (stale → full refetch on
  WebSocket reconnect covers sync clients; notifications would need the outbox
  pattern for zero-loss — deferred).
- `FCM_DEVICE_TOKEN` is a static single-user config; a device registry endpoint
  is Phase 8 (multi-user) work.

## When to Revisit

- **Multi-user (Phase 8):** per-user device registry, `user_id`-scoped events,
  and per-user notification preferences.
- **Zero-loss requirement:** adopt an outbox table so events survive process
  restarts before reaching RabbitMQ.
