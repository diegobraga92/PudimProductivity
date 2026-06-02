# ADR 002: Modular Monolith Architecture

**Status:** Accepted  
**Date:** 2026-01-06  
**Author:** Diego Braga

## Context

The application needs to support multiple domains (tasks, habits, pomodoro, meal planning, book tracking) while maintaining developer productivity, testability, and the option to split into microservices later. The key requirements are:

- Domains must be loosely coupled — changes in one should not affect others.
- Each domain should be independently testable.
- Deployment must be simple (single binary in early phases).
- The architecture must not prevent splitting into microservices in the future.

## Decision

We will use a **modular monolith** architecture. The system is deployed as a single Go binary, but the code is organized into distinct, isolated modules under `internal/` following ports-and-adapters (hexagonal) architecture.

### Module Structure

Each module follows the same structure:

```
internal/{module}/
    domain.go           — Core domain types and business logic
    repository.go       — Port (interface) for persistence
    postgres_repository.go — Adapter: Postgres implementation
    service.go          — Use cases / business operations
    handler.go          — HTTP transport layer (Chi routes)
    module.go           — DI wiring and route registration
```

### Module Boundaries

| Module | Dependencies | Exports |
|--------|--------------|---------|
| `task` | `shared` | Service interface, TaskResponse, ToTaskResponse |
| `tasklist` | `shared`, `task` (Service interface) | Handler |
| `pomodoro` | `shared` | — |
| `featureflag` | `shared` | Service |
| `audit` | `shared` | Logger interface |

### Communication Between Modules

- **Service interfaces** are the sole contract between modules. No module directly calls another's repository or accesses its database tables.
- Example: `tasklist` uses `task.Service` (the interface) to list tasks by list ID — it never calls `task.PostgresTaskRepository` directly.
- **Shared context:** All modules access user identity via `shared.GetUserID(ctx)` and `shared.GetUserRole(ctx)`.

### When to Split

The monolith should be split into microservices when any of these conditions are met:

1. **Independent scaling:** One module needs significantly more replicas than others (e.g., notification delivery requires 10x the compute of task CRUD).
2. **Different data stores:** A module requires a non-PostgreSQL database (e.g., Redis for real-time state, S3 for file storage).
3. **Team boundaries:** More than one team needs to deploy changes independently.
4. **Performance isolation:** A noisy neighbor in one module degrades another module's latency.

The anticipated split points would be:

- `internal/notification/` → standalone Notification Service (Phase 3)
- `internal/mealplan/` + `internal/booktrack/` → could remain or split depending on load (Phase 5)

### Rationale

- **Simpler operations:** One binary, one deployment, one set of metrics.
- **Faster development:** No network calls between modules, no serialization overhead.
- **Atomic migrations:** All database changes in a single deployment.
- **Clear migration path:** Because modules communicate via interfaces, extracting a module into a microservice only requires implementing the same interface over HTTP/gRPC.

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Microservices** | Independent scaling, team autonomy, polyglot | Operational complexity, network overhead, distributed transactions |
| **Package-by-layer** (controllers/services/repos) | Simple, familiar | Low cohesion, easy to blur boundaries, harder to extract |
| **Lambda functions** | No ops, auto-scaling | Cold starts, state management, testing complexity |

### Consequences

**Positive:**
- Clear module boundaries enforced by Go's import cycle detection.
- Each module can be tested in isolation via interface mocks.
- Modules can be extracted to microservices without rewriting business logic.

**Negative:**
- Single binary means a crash takes down all features — mitigated by graceful degradation.
- Shared database means schema changes require coordination across modules — mitigated by forward-only migrations (see ADR 001).
- Import cycles can be frustrating to untangle — mitigated by the explicit dependency hierarchy above.