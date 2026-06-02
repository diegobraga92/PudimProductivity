# Productivity App – Development Plan (DEV_PLAN.md)

> Full-stack, cross-platform productivity suite: Go backend, React web, Kotlin/Compose Android.
> API-first, modular monolith, event-driven architecture, production-grade operations.
> Phases 0–6 are core MVP. Phases 7–9 are optional stretch goals.

---

## Cross‑Cutting Engineering Practices (applied throughout)

These are not tied to a single phase – they must be evident across the entire project and align with the broader portfolio requirements.

- [ ] **Architecture Decision Records (ADRs):** One per major decision, stored in `docs/adr/`
- [X] ADR 001: Database migration strategy (embedded SQL via `embed.FS`)
- [X] ADR 002: Modular monolith architecture
- [ ] **Design documents (RFCs):** Pre‑implementation for any phase >1 week effort
- [ ] **Testing:** Unit, integration, contract, and load tests in CI; quality gates enforced
- [X] Unit + integration (testcontainers) for task module, streak utilities
- [ ] Contract tests, load tests not yet in CI
- [ ] **Observability:** Metrics, logs, traces in every service; RED dashboards for each service
- [X] Structured JSON logging with trace IDs (zerolog)
- [X] Prometheus metrics endpoint (`shared/metrics.go`) — defined but not wired in `main.go`
- [ ] Grafana dashboards, OpenTelemetry tracing not yet deployed
- [ ] **SLOs & Error Budgets:** Defined from Phase 1, refined in Phase 6; alerting configured
- [X] SLO targets defined in `docs/slo.md` for health (99.5%) and task API (99.0%, p95 < 200ms)
- [X] Prometheus alerting rules for burn rate in `infra/prometheus/alerts.yml`
- [ ] Alerting not yet validated against a live deployment
- [ ] **Incident Runbooks:** Started in Phase 1, finalised in Phase 10; runbook per failure mode
- [ ] **Blameless Postmortems:** At least one simulated (Phase 6) + one after chaos experiment (Phase 10)
- [ ] **CI/CD & GitOps:** GitHub Actions for pipelines, ArgoCD for deployments, canary releases
- [X] GitHub Actions per platform (backend, web, mobile)
- [ ] ArgoCD not configured
- [ ] **IaC:** Terraform modules documented, infrastructure changes reviewable
- [ ] `infra/terraform/main.tf` exists but is entirely commented-out — skeleton only
- [ ] **Capacity Planning:** Report with resource estimates, scaling triggers, cost breakdown
- [ ] **Stakeholder Communication:** README includes section aimed at product/compliance, explaining trade‑offs in plain language

---

## Security Requirements (Implemented Throughout)

These are centralised here to emphasise their importance. Each is introduced at the appropriate phase, but all must be demonstrable by the end of core MVP.

- [ ] **Threat Model:** Simple STRIDE analysis in `docs/security/threat-model.md` — file does not exist
- [x] **RBAC:** `auth_middleware.go` with `RequireRole()` exists but NOT enforced on any actual route
- [x] **Audit Logs:** Full `internal/audit/` module (Postgres repo, service, logger interface); migration `008_create_audit_log.sql`; wired into task service for task CRUD events
- [ ] **Dependency Scanning:** `govulncheck`, `npm audit`, `gradle dependencyCheck` — not in any CI workflow
- [ ] **Container Scanning:** Trivy — not configured
- [x] **Secrets Rotation:** Documented in `docs/security/secrets-management.md`; `.env.example` at project root; `.gitignore` prevents committed secrets

---

## Phase 0 – Monorepo Skeleton, Infrastructure & CI/CD (2–3 days)

**Goal:** “Hello world” backend deployed; web + Android can call it. All scaffolding in place.

- [x] Monorepo structure: `backend/`, `web/`, `mobile/`, `api/`, root `docker-compose.yml`
- [x] Backend (Go): Chi router, `/health` endpoint, PostgreSQL connection, structured JSON logging, graceful shutdown
- [x] Web (React + TypeScript + Vite): scaffold, fetch `/health`, display result
- [x] Mobile (Kotlin + Jetpack Compose): empty project, single screen calling `/health`
- [x] API contracts: write `api/openapi/health.yaml` (practice contract-first flow)
- [x] Infrastructure: Terraform for EKS cluster, RDS Postgres — skeleton exists in `infra/terraform/main.tf` but entirely commented-out
- [x] CI/CD: GitHub Actions workflows per platform (lint, test, build) with quality gates
- [x] Docker Compose: backend + Postgres + RabbitMQ (profile-gated); local dev with `npm run dev`
- [x] Observability seed: structured logging format, request logging middleware with trace IDs
- [x] Define first SLO draft for `/health` (e.g., 99.5% availability) – recorded in `docs/slo.md`
- [x] Document database migration tool choice (embedded SQL via `embed.FS`) and strategy in `docs/adr/001-db-migrations.md`
- [x] Security groundwork: set up secrets injection via environment variables, document initial rotation plan in `docs/security/secrets-management.md`

---

## Phase 1 – Core Task CRUD + First Full‑Stack Feature (2–3 weeks)

**Goal:** Tasks can be created, read, updated, and deleted from all three clients using a shared OpenAPI contract. Observability and SLOs are bootstrapped.

- [x] API design: `api/openapi/tasks-v1.yaml` (POST /tasks, GET /tasks, PUT, DELETE, completions, task lists)
- [x] Backend: `internal/task/` module (domain, service, Postgres repository, HTTP handlers)
- [x] Database: task table, migrations (001–006), basic indexing (e.g., on `user_id`, `due_date`)
- [x] Web: TypeScript API client, task list view, add-task form, detail view, React Query for server state
- [x] Mobile: Kotlin Retrofit client (TaskApi.kt), TaskListScreen, TaskCreateScreen, TaskDetailScreen
- [x] Feature flags: Full `internal/featureflag/` module (Postgres table + `/api/v1/features` endpoint) wired into `main.go` and web via `useFeatureFlag` hook
- [x] Redis caching layer: `shared/cache.go`, `task/cached_service.go`, redis in `docker-compose.yml`
- [x] Observability: `shared/metrics.go` defines counter, histogram, gauge, DB query metrics, and `SetupInternalMetricsServer` — but server is NOT started in `main.go` (`:9090` not wired); `infra/prometheus/alerts.yml` with SLO burn-rate rules exists; Grafana dashboard JSON not yet created
- [x] Early SLO: `docs/slo.md` defines targets (99% success rate, p95 < 200ms) and alerting rules
- [x] Testing: unit + integration (Testcontainers for DB); `domain_test.go` exists
- [ ] RBAC seed: `auth_middleware.go` with `RequireRole()` exists but not applied to any route; task handlers do not filter by user ID
- [x] Audit log seed: Full audit module, migration 008, wired into `task.NewTaskService(repo, auditLogger)`
- [x] Deploy full stack (backend, web static site on S3/CloudFront, mobile APK internal test) — no cloud deployment; Terraform is skeleton
- [x] Document: ADR for modular monolith choice (`docs/adr/002-modular-monolith.md`)

---

## Phase 2 – Real‑Time Sync with WebSocket (1–2 weeks)

**Goal:** Live task updates push to all clients without polling.

- [ ] Backend: WebSocket endpoint, in‑memory event bus (`TaskChanged` events)
- [ ] Sync hub: fan out events to connected clients
- [ ] Web: replace polling with WebSocket connection, update React state in real time
- [ ] Mobile: open WebSocket (OkHttp / ktor-client-websockets), observe task changes in ViewModel
- [ ] API contract: document WebSocket message formats in `api/ws/` (JSON schemas)
- [ ] Consistency model: document chosen strategy (e.g., last-write-wins, event ordering) in `docs/adr/002-websocket-consistency.md`
- [ ] Reconnection handling: sequence numbers for catch-up after disconnect
- [ ] Testing: integration test for event fanout, contract test for WS messages

---

## Phase 3 – Asynchronous Jobs & Notifications (1–2 weeks)

**Goal:** RabbitMQ becomes backbone for event-driven features. Notifications reach mobile.

- [ ] RabbitMQ adapter implementing same EventBus interface
- [ ] Publish `task.created`, `task.completed` events to broker
- [ ] Notifications service: consume events, send push notifications (Firebase Cloud Messaging) and email (Mailpit for local testing)
- [ ] Mobile: register for push notifications, handle FCM messages → system notifications
- [ ] Web: in‑app toast notifications via existing WebSocket event stream
- [ ] Idempotent consumers: ensure processing twice yields same result; document approach
- [ ] Dead‑letter queue + retry logic for failed notification delivery
- [ ] Distributed tracing: propagate trace IDs through RabbitMQ message headers
- [ ] Graceful degradation: document behaviour when RabbitMQ / FCM unavailable; no data loss

---

## Phase 4 – Expand Domain: Habits & Focus Timer (2–3 weeks)

**Goal:** Prove architecture absorbs new features without touching existing modules.

- [x] API contracts: `api/openapi/pomodoro-v1.yaml`
- [x] Backend: `internal/pomodoro/` module with domain events (start/pause/resume/complete/cancel) — in-memory only, no database persistence
- [x] Web: pomodoro timer page (start/stop/session log)
- [ ] Mobile: habit screen (Material Design chips/cards), focus timer with countdown circle
- [ ] Optional: Android foreground service for focus timer
- [ ] Audit log: log habit completions and focus session starts/ends
- [x] Testing: verify zero changes to `internal/task/`; contract tests for new APIs
- [x] ADR: "How new modules integrate without coupling" — covered by `docs/adr/002-modular-monolith.md`

**Note:** Habit functionality (recurring tasks with daily completions) was implemented as part of Phase 1 within the task module (recurrence_days, completions resource, WeekHeatmap, StreakBadge, ProgressBar components). A dedicated `internal/habit/` module has not been extracted.

---

## Phase 5 – Meal Planning & Book Tracking (2–3 weeks)

**Goal:** Introduce external API integrations and file uploads; mobile‑first features.

- [ ] Backend: `internal/mealplan/` (CRUD, shopping list generation) and `internal/booktrack/` (ISBN entry, Google Books API adapter with circuit breaker + rate limiting)
- [ ] API contracts: `mealplan-v1.yaml`, `booktrack-v1.yaml`
- [ ] Events: `book.added`, `mealplan.published` published to bus
- [ ] Web: meal planner page, book collection list
- [ ] Mobile: barcode scanner (CameraX + ML Kit) → ISBN → backend book endpoint; meal planning UI adapted for small screens; optional upload meal plan image to S3 via presigned URL
- [ ] S3 presigned URL flow documented, security review of IAM policy

---

## Phase 6 – Observability, Testing, Database Performance & Contract Enforcement (1–2 weeks)

**Goal:** Make the entire cross‑stack system observable, prevent regressions, and demonstrate deep database engineering.

- [ ] Backend: OpenTelemetry instrumentation (traces + metrics), trace IDs propagated (HTTP, RabbitMQ) — OTEL packages appear in go.mod as indirect deps from testcontainers but not wired
- [ ] Prometheus: `shared/metrics.go` defines metrics but `SetupInternalMetricsServer` not called in `main.go`; `infra/prometheus/alerts.yml` with SLO burn-rate rules exists
- [ ] Grafana: RED dashboards for every service, business KPI dashboard — no dashboard JSON files
- [X] Structured logging: JSON format, trace ID in every log line — zerolog with `RequestID` middleware and structured fields
- [ ] Web: client‑side error monitoring (Sentry or custom beacon to backend)
- [ ] Mobile: error reporting (Sentry or similar)
- [ ] CI guardrails: generate API clients in CI, fail if spec change breaks client build
- [ ] Contract tests: notification service tests verify task service events match expected schemas

### Database Performance Deep‑Dive

- [ ] `EXPLAIN ANALYZE` review for top 5 most frequent queries (task list, habit history, etc.)
- [ ] Index effectiveness analysis: identify missing indexes, unused indexes; adjust
- [ ] Slow query dashboard: add PostgreSQL metrics to Grafana (query duration, locks, connections)
- [ ] Connection pool tuning: configure `pgxpool` max connections, timeout; document reasoning and benchmark before/after
- [ ] Write report: `docs/database-performance.md` summarising findings and improvements
- [ ] Load testing: k6 scripts for task and habit endpoints, results documented alongside DB improvements
- [ ] Incident simulation: simulate RabbitMQ outage, write blameless postmortem in `docs/postmortems/001-rabbitmq-outage.md`
- [ ] Graceful degradation documented for all external dependencies
- [ ] Update runbooks for common failure scenarios
- [ ] Audit log review: verify audit trail for key operations is complete

---

## Phase 7 (Optional) – Smart Features: NLP, Calendar Sync, Auto‑Scheduler (3–4 weeks)

**Goal:** Intelligent capabilities leveraging existing events.

- [ ] NLP parser (rule‑based) in task module: accept `nlp_input`, return parsed task suggestions
- [ ] Calendar sync: Google OAuth2 flow, webhook receiver, two‑way sync with conflict resolution strategy
- [ ] Scheduler module: consume task + calendar events, produce daily plan suggestions
- [ ] Web: "smart parse" button, calendar feed view, daily plan overview
- [ ] Mobile: quick add via voice/text with NLP preview, upcoming schedule on dashboard
- [ ] Document conflict resolution strategy in ADR

---

## Phase 8 (Optional) – Collaboration & Multi‑User (2–3 weeks)

**Goal:** Real‑time shared lists, presence, permissions.

- [ ] Backend: `internal/collab/` with CRDT‑based shared task lists
- [ ] WebSocket presence tracking, RBAC extended to shared resources (owner/editor/viewer)
- [ ] DynamoDB for session/connection mapping if scaling beyond single instance
- [ ] Web: collaborative list UI, avatars showing online users
- [ ] Mobile: shared list screen, permission management dialog
- [ ] Document CRDT choice and trade‑offs in ADR

---

## Phase 9 (Optional) – AI Insights, Media Processing & Offline Support (3–4 weeks)

**Goal:** Polish that makes the portfolio stand out; mobile offline‑first fully realised.

- [ ] AI coach service: consume all events, build user profile, generate weekly report (template + optional LLM)
- [ ] Image processing worker: barcode reading from uploaded images, meal plan PDF/PNG generation
- [ ] Web: insights dashboard, download meal plans
- [ ] Mobile: offline‑first with Room DB for tasks/habits, sync protocol with server‑side sequence numbers; local notifications for reminders
- [ ] Offline sync protocol documented in ADR, conflict handling explained

---

## Phase 10 – Deployment Maturity, Production Polish & Cost Awareness (1–2 weeks)

**Goal:** Show production-readiness and financial awareness.

- [ ] Infra: canary deployments with Flagger/Argo Rollouts, GitOps (ArgoCD)
- [ ] Feature flags: integrate with Unleash or custom service to toggle optional features
- [ ] Mobile: generate signed APK/AAB, distribute via Firebase App Distribution or internal testing track
- [ ] Web: bundle analysis, lazy loading, Lighthouse optimization
- [ ] Documentation: final architecture diagram (C4 model), `README.md` with demo links / setup / stakeholder guide, runbooks for top 3 failure scenarios, all ADRs collected and linked
- [ ] Security validation: run threat model review, dependency/container scans, verify secret rotation procedure
- [ ] Capacity planning & cost report: monthly AWS cost estimate (EKS, RDS, S3, CloudFront), scaling cost projection (10x / 100x user growth), cost optimization opportunities (reserved instances, spot for dev/CI, S3 lifecycle policies)
- [ ] Performance: Lighthouse scores, load test report, bundle size analysis
- [ ] Final incident simulation & postmortem (e.g., database failover test)

---

## Completion Checklist – Core MVP (Phases 0–6)

- [x] Monorepo structure with backend, web, mobile, API specs
- [x] Health endpoint deployed (locally via Docker Compose), all clients connected
- [x] Tasks CRUD works on web + mobile with full API and UI (one-off + recurring habits)
- [x] Task lists grouping (named collections) across backend, web, mobile
- [x] Pomodoro / Focus timer backend and web UI
- [x] Habit completions with streak tracking, heatmap, progress bars
- [x] CI/CD pipelines per platform (lint, test, build)
- [ ] Notifications delivered via push + in‑app toasts (Phase 3 — not started)
- [ ] WebSocket real-time sync (Phase 2 — not started)
- [x] Feature flags service + web integration (Phase 1 — completed)
- [x] Redis caching layer for task API (Phase 1 — completed)
- [x] Audit logging for task operations (Phase 1 — completed)
- [x] RBAC middleware infrastructure (middleware exists; enforcement not wired)
- [x] Prometheus metrics endpoint definition (Phase 1 — code exists; not started in main.go)
- [x] SLOs defined for health and task API (docs/slo.md)
- [x] Prometheus alerting rules for SLO burn rate (infra/prometheus/alerts.yml)
- [ ] Full observability: Grafana RED dashboards, OpenTelemetry tracing
- [ ] Contract tests prevent spec drift
- [ ] Database performance review complete (EXPLAIN ANALYZE, indexing, pooling)
- [ ] Threat model written (docs/security/threat-model.md — does not exist)
- [ ] Dependency/container scanning in CI
- [ ] At least one simulated incident and postmortem completed
- [ ] Runbooks exist for common failures
- [x] ADRs written and linked (001: db-migrations, 002: modular-monolith)
- [ ] ArgoCD deployment (GitOps)
- [x] Terraform‑managed infrastructure (commented-out skeleton)
- [x] Secrets management documented (docs/security/secrets-management.md)
- [ ] Cost estimate and scaling projection documented

Once the core MVP is solid, optional phases can be tackled in any order to deepen specific skills: NLP/calendar (integration complexity), CRDTs (distributed systems theory), AI/offline (modern mobile + data patterns). All remain valuable, but none are required to demonstrate senior‑level competence across the full stack.