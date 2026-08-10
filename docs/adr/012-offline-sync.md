# ADR 012: Offline-First Sync Protocol

**Status:** Accepted
**Date:** 2026-08-10
**Author:** Diego Braga

## Context

Phase 9c makes the mobile app offline-first: users can read and edit tasks,
habits, completions and task lists with no network, then converge with the
server when connectivity returns. Three design questions:

1. **How does the client learn about server changes** since it last synced?
2. **How do locally-made offline changes reach the server**, and what happens
   on conflicts?
3. **What local persistence layer** backs the offline experience?

## Decision

### Incremental timestamp-based pull

`GET /api/v1/sync?since=RFC3339` returns every task, completion, task list and
share **created/updated after `since`**, plus the IDs of rows **soft-deleted
after `since`** (migration 019 adds `deleted_at` and changes DELETE handlers to
`UPDATE … SET deleted_at = NOW()`; every read query filters `deleted_at IS
NULL`). The response carries a fresh `timestamp` the client persists and sends
back as `since` on the next pull. Omitted `since` = full snapshot.

Rationale: timestamps are simpler than a global `seq` table and need no new
write path on every mutation. The only caveat — clock skew — is bounded by the
server being the single writer of `timestamp`.

### Optimistic local writes + server-LWW merge

Local writes are **optimistic**: the row is written to the local DB with
`dirty=1` and rendered immediately; a background sync then pushes dirty rows
through the existing REST endpoints (`POST/PUT/DELETE /tasks`,
`POST/DELETE /tasks/{id}/complete`, `POST/DELETE /task-lists`). Conflicts are
resolved by the Phase 8 **LWW merge** semantics (ADR 010): the server keeps the
winning state, the losing client's dirty flag is cleared, and the next pull
converges the local copy. Deletes are tombstones (`dirty=1, deleted=1`) pushed
as DELETE and kept until the server confirms.

### Tombstones and re-share

Soft-delete keeps tombstone rows so offline clients see deletions. Because
`task_list_shares` has a composite PK, a revoked share keeps its row with
`deleted_at` set; re-sharing revives it (created_at reset) — the sync endpoint
surfaces both directions via `shares` / `deleted_share_keys`.

### Local persistence: hand-rolled SQLiteOpenHelper (Room deferred)

The original plan called for **Room**. During implementation, adding KSP (Room's
annotation processor) failed twice on this toolchain:

- AGP 9.1's **built-in Kotlin** is unsupported by KSP (`"KSP is not compatible
  with Android Gradle Plugin's built-in Kotlin"`).
- Switching to the classic `kotlin("android")` plugin fails with AGP 9
  (`ApplicationExtension … cannot be cast to BaseExtension`) for Kotlin 2.2.21;
  a Kotlin upgrade to 2.3.x is the only supported path, which is a larger build
  migration than Phase 9c justifies.

So the app uses a small **`SQLiteOpenHelper` layer with a Room-style DAO API**
(`local/LocalDatabase.kt`: `upsertTasks`, `queryTasks`, `markTaskDirty`, …).
The offline architecture is identical to a Room design (entities, DAO methods,
`dirty`/`deleted` flags) and can be swapped to Room unchanged once the
toolchain permits. ADR impact is limited to the persistence implementation —
the sync protocol is orthogonal.

### Background sync & local notifications

- **WorkManager** (`SyncWorker`) runs an immediate sync on app start and every
  15 minutes while connected; the repository also syncs after every local
  write and after WS events (Phase 2) refresh local state.
- **`HabitReminderWorker`** posts a local notification at 08:00 for habits not
  yet completed today — read from the local DB, so reminders work fully
  offline (no FCM dependency).

## Alternatives Considered

- **Global `seq` per row / change log**: more robust against clock skew but
  adds a write path on every mutation and a new table; not worth it at this
  scale (documented revisit trigger below).
- **Room now**: blocked by the KSP/AGP 9 incompatibility (see above).
- **Server-authoritative full re-fetch**: simple but wasteful and cannot
  support offline-first UX without a local cache anyway.

## Consequences

- The mobile app is fully usable offline; the UI shows a "📡 Offline" banner
  when the last sync failed.
- The backend gained soft-delete on 4 tables + the sync endpoint; all existing
  reads were audited to filter `deleted_at IS NULL`.
- Local deletes that race a server change converge deterministically via LWW.

## When to Revisit

- If multiple clients sync more than a few hundred changes per sync, or clock
  skew becomes observable, move to a global `seq`-based change log.
- When AGP/Kotlin/KSP versions align, swap `LocalDatabase` for Room — the DAO
  API and sync protocol do not change.
