# ADR 003: Caching Strategy

**Status:** Accepted  
**Date:** 2026-07-06  
**Author:** Diego Braga

## Context

The application has two distinct caching needs that emerged during Phase 1 development:

### 1. Feature flags (`internal/featureflag/`)

- **Per-deployment** configuration — flags like `"tasks"`, `"habits"`, `"focus_timer"` are global toggles, not per-user.
- **Hot path** — `IsEnabled` is called on nearly every request (often multiple times per request in middleware, handlers, and services).
- **Small payload** — each entry is a single boolean, ~7 flags total.
- **Rarely changes** — only when an admin toggles a feature on/off.
- **Must remain available** — a feature flag that silently returns `false` (disabled) because the cache is down could disable critical functionality.

### 2. Task / task-list data (`internal/task/cached_service.go`)

- **User-scoped** — each user sees their own tasks.
- **Larger payloads** — task lists can contain dozens of tasks with full JSON structures.
- **Read-heavy with occasional writes** — users browse tasks far more often than they create or update them.
- **Cross-replica consistency matters** — a user may create a task on one replica and immediately read it from another.

### Existing infrastructure

The project already had a `shared.Cache` abstraction wrapping Redis (with graceful degradation to no-op when Redis is unavailable), used by `CachedTaskService`. The feature flag module was written separately with its own in-memory cache — a design choice that needs to be documented and justified.

## Decision

We will use **two different caching strategies**, each chosen to fit the specific requirements of the data being cached:

| Aspect | Feature flags | Task data |
|--------|---------------|-----------|
| **Cache type** | In-memory `map[string]bool` | Redis via `shared.Cache` |
| **Payload** | Single boolean | Full JSON-serialized structs |
| **Hit cost** | ~10 ns (local memory read) | ~1 ms (network RTT + JSON unmarshal) |
| **Miss cost** | Same as hit + DB query | Same as hit + DB query |
| **Serialization** | None (native Go `bool`) | JSON marshal/unmarshal every round-trip |
| **Cache population** | Synchronous, inline | Asynchronous goroutine |
| **Graceful degradation** | Always works (local memory) | No-op when Redis unreachable |
| **Cross-replica consistency** | Bounded by TTL — `SetEnabled` only refreshes local instance | Immediate — single Redis instance shared by all replicas |
| **Dependency** | None | Redis (fallible) |

### Feature flag cache (`internal/featureflag/`)

```
┌────────────┐     ┌──────────────┐     ┌──────────┐
│  Request   │────▶│ IsEnabled()  │────▶│  DB      │
└────────────┘     │  (cache hit) │     │  (miss)  │
                   │              │     └──────────┘
                   │  map[string] │◀──── cache write
                   │  bool        │
                   └──────────────┘
```

- TTL-based invalidation (configurable, 0 disables caching entirely).
- Read-lock for concurrent reads, write-lock for TTL expiry and cache population.
- `RefreshCache()` resets the local timestamp — only affects the current replica.

### Task data cache (`internal/task/cached_service.go`)

```
┌────────────┐     ┌─────────────────┐     ┌──────────┐
│  Request   │────▶│ ListTasks()     │────▶│  DB      │
└────────────┘     │  (cache hit)    │     │  (miss)  │
                   │                 │     └──────────┘
                   │  shared.Cache   │◀──── goroutine
                   │  (Redis)        │      (async set)
                   └─────────────────┘
```

- Cache-aside pattern: read from cache first, fall through to DB on miss.
- Async population (non-blocking write to Redis).
- Cache keys follow `shared.CacheKey` conventions for consistency (`task:{id}`, `tasks:list`).
- Graceful degradation: if Redis is down, reads fall through to DB with no error; writes are silently dropped.

### Neither case uses write-through caching

Both caches are read-only — writes go directly to the database. The cache is populated on read (lazy/cache-aside pattern) and invalidated explicitly:

- **Feature flags:** `SetEnabled` calls `RefreshCache()` to invalidate the local in-memory cache.
- **Task data:** No explicit invalidation yet — relies on TTL expiry (planned for future phases).

## Rationale

### Why not use Redis for feature flags?

1. **Latency overhead on the hot path.** Feature flags are checked on nearly every request (often multiple times). A Redis round-trip per check would add ~1 ms per flag check. With ~7 flags and dozens of requests per second, this multiplies latency unnecessarily.
2. **Availability coupling.** If Redis is down, the `shared.Cache` degrades to a no-op — every `IsEnabled` call would hit the database. This adds load to the database during an already degraded state and risks making a bad situation worse. An in-memory cache has no external dependency.
3. **No benefit from shared state.** Feature flags are per-deployment and change infrequently. The consistency window introduced by per-replica TTLs is acceptable (seconds to minutes of staleness after a toggle). No user data is at risk.

### Why not use in-memory cache for task data?

1. **Memory pressure.** Task data includes JSON payloads for potentially thousands of tasks per user. Storing this in each replica's heap would multiply memory usage by the number of replicas and risk OOM under load.
2. **Consistency matters.** Unlike feature flags, task data is user-facing — a user who creates a task expects to see it on the next page load. With per-replica in-memory caching, a request routed to a different replica would miss the newly created task.
3. **Redis is already available.** The project already runs Redis in `docker-compose.yml`. Using it for task caching adds no new infrastructure.

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Redis for everything** | Single caching abstraction, cross-replica consistency | Hot-path latency, availability coupling to Redis |
| **In-memory for everything** | No network dependency, lowest latency | Memory pressure, task data inconsistency across replicas |
| **No caching** | Simplest possible system | Database becomes bottleneck under load |
| **Local LRU with Redis fallback (feature flags)** | Best of both worlds | Over-engineered for seven boolean values |

## Consequences

### Positive

- Each cache is optimized for its specific access pattern and payload size.
- Feature flag checks remain fast and reliable regardless of Redis health.
- Task data benefits from cross-replica consistency and off-heap storage.
- The two approaches serve as contrasting examples of caching strategy in the same codebase.

### Negative

- Two caching abstractions to maintain (in-memory map + Redis client).
- Feature flag toggles may not be visible immediately across all replicas — bounded by `cacheTTL`.
- Engineers must understand both strategies to work with the codebase.

### When to Revisit

- **Consider switching feature flags to Redis** when: (a) the number of flags grows to hundreds or thousands, (b) per-user flag overrides are introduced, or (c) sub-second cross-replica consistency becomes a requirement.
- **Consider adding a local cache layer in front of Redis (feature flags)** if: Redis latency becomes a bottleneck for flag checks (extremely unlikely at this scale — revisit only if profiling shows otherwise).