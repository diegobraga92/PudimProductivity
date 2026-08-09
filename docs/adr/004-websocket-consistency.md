# ADR 004: WebSocket Real-Time Sync & Consistency Model

**Status:** Accepted
**Date:** 2026-08-09
**Author:** Diego Braga

## Context

Phase 1 delivered REST-only task management. The web and mobile clients
refreshed via on-demand queries and page-local state, so changes made on one
client were not visible on others without a manual refresh. Phase 2 adds
real-time sync: task mutations (created/updated/deleted/completed) must push to
all connected clients.

Constraints:

- **Single backend process** (modular monolith). No distributed state in Phase 2.
- **Clients are unreliable** — network drops, backgrounding, and app restarts are
  the norm on mobile.
- **The database is the system of record.** The WebSocket stream is a
  convenience, never a source of truth.
- **Phase 3** will replace the in-memory event bus with RabbitMQ. The event
  envelope and sequence model must survive that migration unchanged.

## Decision

We use a **server-authoritative, last-write-wins push model** with **per-process
monotonic sequence numbers** and **bounded replay on reconnect**.

### Event Bus

`internal/eventbus/` exposes a `Bus` interface (`Publish` / `Subscribe` / `Close`)
with an in-memory implementation. The task service publishes after successful
persistence (`task.created`, `task.updated`, `task.deleted`,
`task.completed`, `task.uncompleted`). Publishing is **best-effort**: a bus
failure is logged but never rolls back the already-committed database
transaction. Phase 3 adds a RabbitMQ implementation of the same interface; the
task service and sync hub do not change.

### Message Envelope

Every event is a JSON envelope (see `api/ws/events-v1.json`):

```json
{ "type": "task.updated", "seq": 42, "timestamp": "…", "payload": { … } }
```

- `payload` carries the **full task representation** for created/updated events,
  so clients apply the change without a follow-up REST call (deleted/completed
  events carry only the minimal reference).
- `seq` is a **per-process monotonically increasing integer**, assigned at
  publish time. Publishes are serialized, so clients observe events in exactly
  seq order — there are no gaps except across a server restart.

### Reconnect & Replay

The sync hub (`internal/sync/`) keeps an in-memory **ring buffer** of the most
recent events (default 1000). Clients persist the last seen `seq` and pass it as
`?last_seq=N` when (re)connecting:

- If `N` is within the buffer, the hub **replays** all events with `seq > N`
  before streaming live events.
- If `N` is older than the oldest buffered event, the hub sends a **`stale`**
  signal; the client must do a full REST refetch, then resume live updates.
- If the client cannot keep up, the hub drops it; it reconnects and replays.

Registration (replay snapshot + adding the client to the fan-out set) is atomic
with respect to event dispatch, so **no event is delivered twice or missed** at
connect time.

### Consistency Semantics

- **Last-write-wins:** mutations are applied to the database first; the event
  that was persisted last has the latest state. Clients apply updates in seq
  order, so the final state converges to the server's.
- **Server-authoritative:** clients never rebroadcast mutations. A client's
  optimistic UI is reconciled by the server-confirmed event.
- **Best-effort delivery with replay:** a client that was offline for < buffer
  window catches up from the buffer; one that was offline longer does a full
  refetch. Either way, convergence is guaranteed because the REST endpoint and
  the WS stream read the same database.
- **Session scoping is a Phase 8 concern.** Phase 2 broadcasts all task events
  to all connected clients (single-user MVP). Per-user filtering will be added
  with authentication (threat model P0 items).

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Polling (status quo)** | Simple, stateless | Latency, wasted requests, no push |
| **Push without sequence numbers** | Simple protocol | Client cannot tell what it missed after a drop — every reconnect forces a full refetch |
| **CRDT-based sync** | Convergent without server authority | Over-engineered for a single-user app; no server-authoritative source; Phase 8 introduces this for shared lists |
| **Event sourcing / outbox table** | Durable event log, replay across restarts | Significant migration cost; overkill while the event bus is in-memory |

## Consequences

**Positive:**

- Near-instant cross-client updates; no polling for tasks.
- Reconnect is cheap for short drops (replay) and correct for long drops (stale
  → refetch).
- The `Bus` interface is the seam for the Phase 3 RabbitMQ migration.

**Negative:**

- **Events are lost on server restart** (in-memory bus + ring buffer). Mitigated
  by the `stale` → full-refetch fallback and by Phase 3's durable RabbitMQ
  exchange.
- **Broadcast fan-out** leaks one user's task titles to other connected clients
  in the multi-user future; requires scoping (Phase 8).
- A client that stays connected while the server restarts sees its connection
  close and must re-handshake — handled by client auto-reconnect.

## When to Revisit

- **Phase 3:** replace `InMemoryBus` with the RabbitMQ adapter. Sequence numbers
  become per-broker (or switch to broker message IDs). Consider an outbox table
  for durable publish.
- **Phase 8 (collaboration):** per-user/room scoping of the fan-out, and CRDT
  merging for shared lists.
