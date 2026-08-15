# Architecture Decision Records — Index

> One ADR per major architectural decision. Decisions are recorded with
> **Context → Decision → Alternatives → Consequences → When to Revisit** so
> future contributors can reconstruct *why* the system is shaped the way it is.

| ADR | Title | Status | Date | Topic |
|-----|-------|--------|------|-------|
| [001](001-db-migrations.md) | Database Migration Strategy | Accepted | 2026-01-06 | Embedded SQL migrations via `embed.FS`, applied at startup, recorded in `schema_migrations` |
| [002](002-modular-monolith.md) | Modular Monolith Architecture | Accepted | 2026-01-06 | Single Go process, per-domain packages, interface seams, no microservice tax at MVP scale |
| [003](003-caching-strategy.md) | Caching Strategy | Accepted | 2026-07-06 | Redis read-through cache for task reads; cache-aside with explicit invalidation on writes |
| [004](004-websocket-consistency.md) | WebSocket Real-Time Sync & Consistency Model | Accepted | 2026-08-09 | Server-authoritative LWW push, per-process `seq`, replay buffer, stale→refetch |
| [005](005-async-notifications.md) | Asynchronous Notifications via RabbitMQ | Accepted | 2026-08-09 | `CompositeBus` (in-memory + RabbitMQ), at-least-once with idempotent dedupe, DLQ retry pump |
| [006](006-deployment-strategy.md) | Deployment Strategy — Single-Host Compose, GitOps Forward Path | Accepted | 2026-08-09 | docker-compose on one host for MVP; Kustomize/ArgoCD manifests prepared but not activated |
| [007](007-external-api-integrations.md) | External API Integration Pattern | Accepted | 2026-08-09 | Shared `httpclient` (retry/rate-limit/circuit-breaker) + thin adapters + consumer-side interfaces; presigned-S3 media with degraded mode |
| [008](008-nlp-parser.md) | Rule-Based NLP Parser | Accepted | 2026-08-09 | Deterministic regex parser for task input; partial-result semantics; LLM fallback behind the same endpoint |
| [009](009-scheduler.md) | Auto-Scheduler with Derived Profile | Accepted | 2026-08-09 | Stateless scheduler; profile derived from 14-day history; read-only suggestions fitted into free blocks |
| [010](010-crdt-collaboration.md) | CRDT-Based Collaboration | Accepted | 2026-08-10 | Document-level LWW register for task merges; owner/editor/viewer shares; WS presence + membership scoping |
| [011](011-ai-coach-insights.md) | AI Coach — Template-First Reports | Accepted | 2026-08-10 | Go-template weekly reports from domain events; optional LLM summary behind a feature flag |
| [012](012-offline-sync.md) | Offline-First Sync Protocol | Accepted | 2026-08-10 | Timestamp-based incremental pull; optimistic push + Phase 8 LWW merge; soft-delete tombstones; SQLite local layer (Room deferred) |
| [013](013-widgets.md) | Home-Screen Widgets for Tasks & Habits | Accepted | 2026-08-13 | Companion-widget roadmap; data sourced from the offline-sync store |
| [014](014-runtime-score-provider-config.md) | Runtime Score-Provider Config & Auto-Scoring | Accepted | 2026-08-15 | DB-backed provider config (env = one-time bootstrap), reloadable lookup manager, admin UI, batch score endpoint for CSV auto-scoring |

## Cross-Links

- ADR 002 (modular monolith) is the load-bearing decision: every module in
  `backend/internal/` follows the service+repository+handler seam it defines.
- ADR 004 (WS consistency) and ADR 005 (async notifications) both extend ADR 002's
  event-bus seam; the `Bus` interface survived both without producer changes.
- ADR 010 builds on ADR 004's server-authoritative LWW push model — the CRDT
  merge reuses the same timestamp-wins semantics for concurrent *writers*.
- ADR 006 (deployment) is informed by ADR 002 (single process → single container)
  and ADR 005 (RabbitMQ as an optional-at-startup dependency).

## Writing New ADRs

Number sequentially (`00N-<slug>.md`), link it here, and keep the five-section
template. If a decision is revisited, do not edit history — add a new ADR that
supersedes the old one and mark the old **Superseded by ADR 00N**.
