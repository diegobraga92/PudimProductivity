# ADR 010: CRDT-Based Collaboration — LWW-Register + Membership Scoping

**Status:** Accepted
**Date:** 2026-08-10
**Author:** Diego Braga

## Context

Phase 8 adds multi-user collaboration: a task list can be shared with other
users (editor/viewer), and two users may edit a task concurrently. We need:

1. **A merge strategy** for concurrent task edits that converges on one state
   without locks, conflict UIs, or operator intervention.
2. **Access control** (owner/editor/viewer) on shared resources.
3. **Presence** so members see who is online.

## Decision

### Merge strategy: document-level Last-Writer-Wins (LWW) Register

`PATCH /api/v1/tasks/{taskId}/merge` treats each task as a single LWW register:

- The client sends its edit **plus the timestamp at which it was made**
  (`updated_at`, RFC3339).
- The server applies the write only if `client_updated_at` is **strictly newer**
  than the stored `updated_at`, or **equal** AND the client's user ID sorts
  after the stored `updated_by` (deterministic tie-break).
- A winning write is persisted with `updated_by` recorded and a `task.merged`
  event published so all clients converge on the winning state.
- A losing write returns **HTTP 409 with the winning task state** so the client
  can reconcile and retry without a manual conflict dialog.

This is the simplest CRDT that gives eventual convergence. A document-level
register was chosen over a **field-level register** (per-field timestamps) and
over an **RGA/OR-Set for list ordering** because:

- Concurrent edits to a task are rare; a whole-task register is sufficient.
- Field-level registers add a `task_fields` table and N-way merge complexity
  with no user-visible benefit at this scale.
- Task ordering in a list is server-determined (created_at DESC); clients do
  not reorder tasks concurrently, so an ordered CRDT (RGA) is not warranted.
- Tombstones are unnecessary: the latest timestamp IS the truth; the "old"
  version is simply rejected and never stored.

**Trade-off:** LWW is lossy — the older writer's edit is discarded entirely
rather than merged field-by-field. This is the accepted price for simplicity and
is consistent with ADR 004's server-authoritative LWW push model. If a future
phase needs field-level merging (e.g. two editors editing different fields
simultaneously), the natural evolution is a field-level LWW register behind the
same endpoint — the API shape (`updated_at` + fields) already supports it.

### Access model: owner / editor / viewer

- `task_lists.owner_id` is written at creation (migration 017; legacy rows
  default to `dev-user`).
- `task_list_shares(list_id, shared_with, role)` grants `editor` or `viewer`.
- Enforcement lives in `internal/tasklist`: `CheckAccess` verifies the effective
  role (owner → share role → denied) for every mutation and read of a list and
  its tasks. Admins bypass as owners (dev-mode role model).
- Mutations on a list require the owner (`share`, `unshare`, `delete`).

### Presence + event scoping in the sync hub

- WebSocket connections now carry the user identity (from the dev headers via
  `shared.AuthMiddleware`).
- A `MembershipResolver` (Postgres-backed in `internal/collab`) computes the
  user's list membership; the hub stores it per connection.
- `presence.online` / `presence.offline` events are published through the bus
  (so they are replayed to late joiners) and a REST snapshot
  (`GET /api/v1/presence/{listId}`) bootstraps page loads.
- List-scoped events (`task.created/updated/merged`, `tasklist.shared/unshared`)
  are dispatched **only to connected members of that list**; non-list events
  (completions, book/mealplan/recipe, presence) are broadcast as before.
- On `tasklist.shared`/`tasklist.unshared`, live connections of the affected
  user are re-resolved so mid-session scope changes apply immediately.

## Alternatives Considered

- **Server lock per task (pessimistic):** simplest but serializes collaborators
  and requires lease expiry / ownership bookkeeping.
- **Operational Transform (OT):** powerful for text, overkill for discrete task
  fields, and requires a transform server.
- **CRDTs with per-field registers + RGA ordering:** the "textbook" answer, but
  ~3× the code for no user-visible benefit at this scale (see above).
- **No merge (last request wins silently):** violates the plan's requirement to
  *demonstrate* a resolution strategy and would silently drop user edits.

## Consequences

- Concurrent edits converge deterministically; the losing client reconciles via
  the 409 body and the `task.merged` event.
- One new table (`task_list_shares`), two new columns (`owner_id`,
  `updated_by`), one new module (`internal/collab`).
- The WS hub is no longer identity-less: presence and scoping require the
  resolver, which depends on the DB. When the resolver is unavailable (tests,
  degraded startup) the hub degrades to the legacy broadcast behavior.
- `task_list_shares` membership is refreshed on share/unshare events; a user
  whose access is revoked mid-session loses live events immediately.

## When to Revisit

- If concurrent edits to the same task become frequent, move to field-level LWW
  registers behind the same endpoint.
- If shared lists grow beyond ~10 members, replace the per-request DB membership
  query with a cached membership store.
- When real JWT auth lands (threat model P0), the resolver and dev headers
  inherit the same identity source — no API change needed.
