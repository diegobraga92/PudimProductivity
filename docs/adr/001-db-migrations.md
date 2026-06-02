# ADR 001: Database Migration Strategy

**Status:** Accepted  
**Date:** 2026-01-06  
**Author:** Diego Braga

## Context

The project needs a way to evolve the PostgreSQL schema across environments (local dev, CI, staging, production) in a repeatable, auditable manner. The key requirements are:

- Migrations must be applied automatically on application startup (no manual step).
- The process must work in containerized environments without additional CLI tools.
- Each migration must be transactional and idempotent.
- The team must be able to trace which migrations have been applied in any given environment.

## Decision

We will use **embedded SQL migrations** applied programmatically via Go's `embed.FS` package, rather than an external migration CLI tool.

### How It Works

1. Every `.sql` file placed in `backend/internal/db/migrations/` is embedded into the Go binary at compile time via `//go:embed migrations/*.sql`.
2. On startup, the `RunMigrations()` function:
   - Creates a `schema_migrations` tracking table if it doesn't exist.
   - Reads the embedded directory, sorts filenames lexicographically, and applies any unseen `.sql` files inside a database transaction.
   - Records the filename in `schema_migrations` on success, rolls back on failure.
3. Migration order is dictated by alphabetical filename sorting (e.g., `001_create_tasks.sql` before `002_create_feature_flags.sql`).

### Rationale

- **Zero external dependencies:** No need for `golang-migrate` CLI, `goose`, or any other tool — just `go build`.
- **Single binary deployment:** The migration logic is baked into the server binary — no separate migration step in CI/CD pipelines.
- **Simplicity:** The implementation is ~90 lines of Go code (see `backend/internal/db/migrate.go`).
- **Auditability:** The `schema_migrations` table provides a clear record of what was applied and when.

### Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **golang-migrate CLI** | Mature, widely adopted, supports rollbacks, multiple DB drivers | Requires separate CLI binary in CI/CD; adds complexity to deployment pipeline; rollbacks are rarely used in practice and can cause data loss |
| **goose** | Similar to golang-migrate, Go-native | Same CLI dependency issue; less widely adopted |
| **Flyway (Java-based)** | Powerful, many DB features | Not Go-native; requires JVM in the container |
| **Manual SQL scripts** | No tooling overhead | Error-prone; no tracking; no idempotency |

### Consequences

**Positive:**
- Single Go binary contains everything needed to run and migrate the database.
- No version conflicts between migration tool and application.
- Transactional migrations ensure atomicity.

**Negative:**
- **No rollback support.** To undo a migration, you must write a compensating forward migration (e.g., `007_revert_006_add_list_id_to_tasks.sql`). This is an accepted trade-off — rollbacks in production often cause more harm than good.
- **Migration filename changes are dangerous.** Once a migration is applied in any environment, the filename must never change. We enforce this by code review.
- **Every migration must be safe for concurrent application.** The startup uses a locking mechanism only via the transaction — if two instances start simultaneously, the duplicate key on `schema_migrations` will cause a rollback on one of them (acceptable for small deployments).

### Migration Naming Convention

```
{NNN}_{description}.sql
```

Where:
- `NNN` — zero-padded, 3-digit sequence number (001, 002, ..., 999)
- `description` — snake_case, concise description of the change

Example: `001_create_tasks.sql`, `002_create_feature_flags.sql`

### Migration Writing Guidelines

1. **Always wrap destructive changes** (ALTER, DROP) in idempotent constructs: `IF NOT EXISTS` / `IF EXISTS`.
2. **Seed data** should use `ON CONFLICT DO NOTHING` to allow re-applying the same migration.
3. **Test migrations** by running `docker compose up` with a fresh database.
4. **Never edit an existing migration file** after it has been applied in any environment — create a new one.