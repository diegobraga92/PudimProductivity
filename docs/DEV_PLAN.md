# Productivity App – Development Plan (DEV_PLAN.md)

> Full-stack, cross-platform productivity suite: Go backend, React web, Kotlin/Compose Android.
> API-first, modular monolith, event-driven architecture, production-grade operations.
> Phases 0–6 are core MVP. Phases 7–9 are optional stretch goals.

---

## Cross‑Cutting Engineering Practices (applied throughout)

These are not tied to a single phase – they must be evident across the entire project and align with the broader portfolio requirements.

- **Architecture Decision Records (ADRs):** One per major decision, stored in `docs/adr/`
- **Design documents (RFCs):** Pre‑implementation for any phase >1 week effort
- **Testing:** Unit, integration, contract, and load tests in CI; quality gates enforced
- **Observability:** Metrics, logs, traces in every service; RED dashboards for each service
- **SLOs & Error Budgets:** Defined from Phase 1, refined in Phase 6; alerting configured
- **Incident Runbooks:** Started in Phase 1, finalised in Phase 10; runbook per failure mode
- **Blameless Postmortems:** At least one simulated (Phase 6) + one after chaos experiment (Phase 10)
- **CI/CD & GitOps:** GitHub Actions for pipelines, ArgoCD for deployments, canary releases
- **IaC:** Terraform modules documented, infrastructure changes reviewable
- **Capacity Planning:** Report with resource estimates, scaling triggers, cost breakdown
- **Stakeholder Communication:** README includes section aimed at product/compliance, explaining trade‑offs in plain language

---

## Security Requirements (Implemented Throughout)

These are centralised here to emphasise their importance. Each is introduced at the appropriate phase, but all must be demonstrable by the end of core MVP.

- **Threat Model:** Simple STRIDE analysis covering the backend API, mobile app, and infrastructure; documented in `docs/security/threat-model.md`
- **RBAC:** Basic `user` / `admin` role separation with middleware; applied to task APIs and feature flags
- **Audit Logs:** `audit_log` table capturing actor, action, resource, timestamp, and old/new values for sensitive operations (task changes, permission changes)
- **Dependency Scanning:** `govulncheck` (Go), `npm audit` (web), `gradle dependencyCheck` (mobile) in CI; alert on critical vulns
- **Container Scanning:** Trivy or similar scanning Docker images in CI pipeline; block high-severity findings
- **Secrets Rotation:** All credentials stored in environment variables / secrets manager; documented rotation procedure for DB creds, API keys, signing keys

---

## Phase 0 – Monorepo Skeleton, Infrastructure & CI/CD (2–3 days)

**Goal:** “Hello world” backend deployed; web + Android can call it. All scaffolding in place.

- [ ] Monorepo structure: `backend/`, `web/`, `mobile/`, `api/`, root `docker-compose.yml`
- [ ] Backend (Go): Chi router, `/health` endpoint, PostgreSQL connection, structured JSON logging, graceful shutdown
- [ ] Web (React + TypeScript + Vite): scaffold, fetch `/health`, display result
- [ ] Mobile (Kotlin + Jetpack Compose): empty project, single screen calling `/health`
- [ ] API contracts: write `api/openapi/health.yaml` (practice contract-first flow)
- [ ] Infrastructure: Terraform for EKS cluster, RDS Postgres (organised in modules, not flat)
- [ ] CI/CD: GitHub Actions workflows per platform (lint, test, build) with quality gates
- [ ] Docker Compose: backend + Postgres + RabbitMQ; local dev with `npm run dev` and emulator pointing to `10.0.2.2:8080`
- [ ] Observability seed: structured logging format, request logging middleware with trace IDs
- [ ] Define first SLO draft for `/health` (e.g., 99.5% availability) – record in `docs/slo.md`
- [ ] Document database migration tool choice (golang-migrate) and strategy in `docs/adr/001-db-migrations.md`
- [ ] Security groundwork: set up secrets injection via environment variables, document initial rotation plan

---

## Phase 1 – Core Task CRUD + First Full‑Stack Feature (2–3 weeks)

**Goal:** Tasks can be created, read, updated, and deleted from all three clients using a shared OpenAPI contract. Observability and SLOs are bootstrapped.

- [ ] API design: `api/openapi/tasks-v1.yaml` (POST /tasks, GET /tasks, etc.)
- [ ] Backend: `internal/task/` module (domain, service, Postgres repository, HTTP handlers)
- [ ] Database: task table, migrations, basic indexing (e.g., on `user_id`, `due_date`)
- [ ] Web: generate TypeScript client from OpenAPI, task list view, add-task form, React Query for server state
- [ ] Mobile: generate Kotlin client (or Retrofit with DTOs), LazyColumn display, FAB to add task
- [ ] Feature flags: simple flag service (Postgres table) + `/api/v1/features` endpoint; toggle “task notes” feature in web/mobile
- [ ] Redis caching layer: add to docker-compose; cache GET /tasks responses with invalidation on mutation
- [ ] Observability: Prometheus metrics endpoint (`/metrics`), basic Grafana dashboard skeleton, request latency histograms
- [ ] Early SLO: define SLO for task API (e.g., 99% success rate, p95 latency < 200ms); set up alerting rules for burn rate
- [ ] Testing: unit + integration (Testcontainers for DB), first contract test verifying server matches OpenAPI spec
- [ ] RBAC seed: implement `admin` role check middleware; regular users see only own tasks
- [ ] Audit log seed: log task creation/deletion events to `audit_log` table
- [ ] Deploy full stack (backend, web static site on S3/CloudFront, mobile APK internal test)
- [ ] Document: ADR for modular monolith choice, runbook entry for deployment, update architecture diagram

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
- [ ] Notifications service: consume events, send:
  - [ ] Push notifications (Firebase Cloud Messaging integration, device token storage in Postgres)
  - [ ] Email (optional; Mailpit for local testing)
- [ ] Mobile: register for push notifications, handle FCM messages → system notifications
- [ ] Web: in‑app toast notifications via existing WebSocket event stream
- [ ] Idempotent consumers: ensure processing twice yields same result; document approach
- [ ] Dead‑letter queue + retry logic for failed notification delivery
- [ ] Distributed tracing: propagate trace IDs through RabbitMQ message headers
- [ ] Graceful degradation: document behaviour when RabbitMQ / FCM unavailable; no data loss

---

## Phase 4 – Expand Domain: Habits & Focus Timer (2–3 weeks)

**Goal:** Prove architecture absorbs new features without touching existing modules.

- [ ] API contracts: `api/openapi/habits-v1.yaml`, `api/openapi/focus-v1.yaml`
- [ ] Backend: `internal/habit/`, `internal/focus/` modules, each with own domain events
- [ ] Web: habit tracker page (daily checkboxes, streak counter), focus timer page (start/stop, session log)
- [ ] Mobile: habit screen (Material Design chips/cards), focus timer with countdown circle
- [ ] Optional: Android foreground service for focus timer
- [ ] Audit log: log habit completions and focus session starts/ends
- [ ] Testing: verify zero changes to `internal/task/`; contract tests for new APIs
- [ ] ADR: “How new modules integrate without coupling”

---

## Phase 5 – Meal Planning & Book Tracking (2–3 weeks)

**Goal:** Introduce external API integrations and file uploads; mobile‑first features.

- [ ] Backend:
  - [ ] `internal/mealplan/`: CRUD, shopping list generation
  - [ ] `internal/booktrack/`: manual ISBN entry, Google Books API adapter with circuit breaker + rate limiting
- [ ] API contracts: `mealplan-v1.yaml`, `booktrack-v1.yaml`
- [ ] Events: `book.added`, `mealplan.published` published to bus
- [ ] Web: meal planner page, book collection list
- [ ] Mobile:
  - [ ] Barcode scanner (CameraX + ML Kit) → ISBN → backend book endpoint
  - [ ] Meal planning UI adapted for small screens
  - [ ] Optional: upload meal plan image to S3 via presigned URL (IAM policy documented)
- [ ] S3 presigned URL flow documented, security review of IAM policy

---

## Phase 6 – Observability, Testing, Database Performance & Contract Enforcement (1–2 weeks)

**Goal:** Make the entire cross‑stack system observable, prevent regressions, and demonstrate deep database engineering.

- [ ] Backend: OpenTelemetry instrumentation (traces + metrics), trace IDs propagated (HTTP, RabbitMQ)
- [ ] Prometheus: all services expose metrics; configure alerting rules based on SLOs
- [ ] Grafana: RED dashboards for every service, business KPI dashboard
- [ ] Structured logging: JSON format, trace ID in every log line
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
- [ ] Web: “smart parse” button, calendar feed view, daily plan overview
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
- [ ] Mobile:
  - [ ] Offline‑first: Room DB for tasks/habits, sync protocol with server‑side sequence numbers
  - [ ] Local notifications for reminders
- [ ] Offline sync protocol documented in ADR, conflict handling explained

---

## Phase 10 – Deployment Maturity, Production Polish & Cost Awareness (1–2 weeks)

**Goal:** Show production-readiness and financial awareness.

- [ ] Infra: canary deployments with Flagger/Argo Rollouts, GitOps (ArgoCD)
- [ ] Feature flags: integrate with Unleash or custom service to toggle optional features
- [ ] Mobile: generate signed APK/AAB, distribute via Firebase App Distribution or internal testing track
- [ ] Web: bundle analysis, lazy loading, Lighthouse optimization
- [ ] Documentation:
  - [ ] Final architecture diagram (C4 model)
  - [ ] `README.md` with demo links, setup instructions, stakeholder guide
  - [ ] Runbooks for top 3 failure scenarios
  - [ ] All ADRs collected and linked
- [ ] Security validation: run threat model review, dependency/container scans, verify secret rotation procedure
- [ ] Capacity planning & cost report:
  - [ ] Monthly AWS cost estimate (breakdown: EKS, RDS, S3, CloudFront, etc.)
  - [ ] Scaling cost projection (10x / 100x user growth)
  - [ ] Cost optimization opportunities (reserved instances, spot for dev/CI, S3 lifecycle policies)
- [ ] Performance: Lighthouse scores, load test report, bundle size analysis
- [ ] Final incident simulation & postmortem (e.g., database failover test)

---

## Completion Checklist – Core MVP (Phases 0–6)

- [ ] Health endpoint deployed, all clients connected
- [ ] Tasks CRUD works on web + mobile with real‑time sync
- [ ] Notifications delivered via push + in‑app toasts
- [ ] Habits and focus timer fully functional across platforms
- [ ] Meal planning and book tracking with external API integration
- [ ] Full observability: traces, metrics, logs, RED dashboards, alerting
- [ ] SLOs defined and monitored
- [ ] Contract tests prevent spec drift
- [ ] Database performance review complete (EXPLAIN ANALYZE, indexing, pooling)
- [ ] RBAC implemented with audit logging for sensitive operations
- [ ] Threat model written, dependency/container scanning in CI
- [ ] At least one simulated incident and postmortem completed
- [ ] Runbooks exist for common failures
- [ ] All ADRs written and linked
- [ ] CI/CD pipeline with automated quality gates
- [ ] Terraform‑managed infrastructure, ArgoCD deployment
- [ ] Cost estimate and scaling projection documented

Once the core MVP is solid, optional phases can be tackled in any order to deepen specific skills: NLP/calendar (integration complexity), CRDTs (distributed systems theory), AI/offline (modern mobile + data patterns). All remain valuable, but none are required to demonstrate senior‑level competence across the full stack.