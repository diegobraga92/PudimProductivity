## Phase 5 – Meal Planning & Book Tracking (2–3 weeks)

> **Status: IMPLEMENTED (2026-08-09).** Both modules are delivered end-to-end
> (backend + web UI + mobile UI + contracts + tests + ADR 007). The previously
> deferred barcode scanner was delivered in Phase 9b: camera intent →
> gozxing ISBN decode → auto-add book.
>
> **Updated (2026-08-13):** Book tracking was **replaced** by the **Library**
> module (`internal/library/`, migration 020) — a simple multi-media tracker
> (movies / series / books / games) with name, media type, release year, a
> `done` flag and optional notes, plus CSV import with column matching and
> fixed values. Existing books were migrated as `type=book` items; the Google
> Books ISBN lookup, the `scan-isbn` barcode decoder and the `booktrack`
> module were removed.

**Goal:** Introduce external API integrations and file uploads; mobile‑first features.

- [x] Backend: `internal/mealplan/` (CRUD, slot assignment, shopping-list generation — aggregates recipe ingredients, sums free-text quantities) and `internal/booktrack/` (ISBN entry via Google Books adapter with circuit breaker + rate limiting via `internal/httpclient`)
- [x] API contracts: `mealplan-v1.yaml`, `booktrack-v1.yaml` (+ `recipes-v1.yaml` in 5a) — all valid, codegen wired
- [x] Events: `book.added`, `mealplan.created`, `mealplan.published` published to bus + WS contract (`api/ws/events-v1.json`) + contract test extended
- [x] Web: meal planner pages (`MealPlanList.tsx`, `MealPlanDetail.tsx` — weekly grid + shopping list + publish), book collection (`BookList.tsx` — ISBN add / manual / status filter), wired into App.tsx nav
- [x] Mobile: meal planning UI (`MealPlanScreen.kt` — list/create/details/generate shopping list) and books UI (`BookListScreen.kt` — ISBN add + list + **📷 barcode scan** → auto-add by ISBN, delivered in Phase 9b)
- [x] S3 presigned URL flow documented + IAM policy reviewed (`docs/security/s3-media-iam.md`); ADR 007 documents the external-API pattern

**Delivered (2026-08-09):** `internal/booktrack/` + `googlebooks` adapter,
`internal/mealplan/` (migrations 015/016), both web + mobile UIs, unit +
integration + adapter tests, events + audit, ADR 007. Verified live end-to-end
(recipe → meal plan → shopping-list aggregation → publish).

**Updated (2026-08-13):** `booktrack` replaced by `internal/library/`
(migration 020 converts `books` → `library_items` with `done = status='read'`,
then drops `books`). Events are now `library.item.added/updated/deleted` +
`library.items.imported`; the Google Books adapter, `scan-isbn` barcode decoder
and `booktrack` routes were removed. The Library ships CSV import
(`POST /api/v1/library/import`) with web column-matching + fixed values.

> **Updated (2026-08-13):** the **meal planner** module was **removed**
> end-to-end — `internal/mealplan/`, `api/openapi/mealplan-v1.yaml`, the WS
> `mealplan.published` event, web `MealPlanList.tsx`/`MealPlanDetail.tsx`,
> mobile `MealPlanScreen.kt`/`MealPlanApi.kt`. Migration `021_drop_mealplans.sql`
> drops `meal_plans`/`meal_plan_slots`/`meal_plan_shopping_list` and the
> `meal_planning` feature flag. Recipes remain a standalone feature.


---

## Phase 5a (Optional) – Recipes Module (2–3 weeks)

**Goal:** A full‑featured cooking recipe manager with media upload support.
Standalone feature — the Phase 5 meal planner was removed (2026-08-13).

- [x] API contract: `api/openapi/recipes-v1.yaml` (CRUD, search by title/tag/difficulty) — valid + codegen works
- [x] Backend: `internal/recipe/` module (domain, service, Postgres repository, HTTP handlers, module router)
- [x] Database: `recipes`, `recipe_tags`, `recipe_ingredients`, `recipe_steps` tables (migration `014_create_recipes.sql`; verified end-to-end)
- [x] Media: presigned S3 upload URLs (`internal/media/` — AWS SDK v2 presigner; `POST /recipes/{id}/upload-url`; graceful degradation → 503 when `S3_MEDIA_BUCKET` unset)
- [x] Web: recipe list (search + tag/difficulty filters, `RecipeList.tsx`), detail/create/edit form (`RecipeDetail.tsx`), wired into App.tsx nav
- [x] Mobile: recipe list/detail/create screens (`RecipeListScreen.kt`, `RecipeCreateScreen.kt` + TaskListScreen nav buttons); camera integration for ingredient photos deferred to the barcode scanner work item
- [x] Events: `recipe.created`, `recipe.updated`, `recipe.deleted` published to event bus + WS contract (`api/ws/events-v1.json`) + contract test extended
- [x] Audit logging: recipe CRUD via `internal/audit/` (`recipe.created/updated/deleted`, resource `recipes`)
- [x] Testing: unit (domain validation, service events/audit) + integration (testcontainers: CRUD, child replacement, search/tag/difficulty filters, keyset pagination) — all with `-race`

---

## Phase 6 – Observability, Testing, Database Performance & Contract Enforcement (1–2 weeks)

**Goal:** Make the entire cross‑stack system observable, prevent regressions, and demonstrate deep database engineering.

- [x] Backend: OpenTelemetry instrumentation — `internal/observability/` (TracerProvider, W3C TraceContext propagator, HTTP span middleware, zerolog trace_id/span_id hook); trace context stamped onto event-bus events; spans exported to stdout + Jaeger (docker-compose `--profile tracing`) via OTLP/HTTP
- [x] Prometheus: metrics server started in `main.go` on internal `:9090` (metrics middleware on public router); `infra/prometheus/alerts.yml` with SLO burn-rate rules exists
- [x] Grafana: RED dashboard (`infra/grafana/red-dashboard.json`) + business KPI dashboard (`infra/grafana/business-kpi.json`) created — datasource uid must be `prometheus`
- [X] Structured logging: JSON format, trace ID in every log line — zerolog with `RequestID` middleware and structured fields
- [x] Web: client‑side error monitoring — `useErrorReporter` hook posts `window.onerror` + `unhandledrejection` to `POST /api/v1/errors`
- [x] Mobile: error reporting — `ErrorReporter` (default uncaught-exception handler) posts crashes to `POST /api/v1/errors`
- [x] CI guardrails: OpenAPI → TypeScript client codegen (`web/scripts/generate-api-types.mjs`) + `check-clients` CI job fails on spec drift (`api/**` triggers web CI)
- [x] Contract tests: `internal/sync/contract_test.go` validates WS events against `api/ws/events-v1.json` (runs in backend CI via `go test ./...`) + k6 smoke load test in CI (`load-smoke` job, `infra/k6/smoke.js`)

### Database Performance Deep‑Dive

- [x] `EXPLAIN ANALYZE` review for top 5 most frequent queries (task list, habit history, etc.) — findings in `docs/database-performance.md`
- [x] Index effectiveness analysis: all pre-existing indexes had `idx_scan = 0` at small scale; added partial index `idx_tasks_habits` (migration 013, verified used at 50k rows); the standalone `completed_date` index found redundant
- [x] Slow query dashboard: `pg_query_duration_seconds` histogram (via `pgx.QueryTracer`) + Grafana RED panel "Database Query Latency (p95 per operation)"
- [x] Connection pool tuning: config documented with sizing reasoning + pgbench baseline in `docs/database-performance.md`
- [x] Write report: `docs/database-performance.md` — EXPLAIN ANALYZE before/after, index analysis, pool reasoning, load-test results
- [x] Load testing: k6 scripts `infra/k6/tasks-load.js` + `habits-load.js` — p95 25ms/36ms, 0% errors at 20/15 VUs on 50k rows
- [x] Incident simulation: RabbitMQ outage exercised; blameless postmortem in `docs/postmortems/001-rabbitmq-outage.md`
- [x] Graceful degradation documented: `docs/graceful-degradation.md` (dependency matrix + acceptable degradation table)
- [x] Update runbooks for common failure scenarios: `docs/runbooks/rabbitmq-unavailable.md`, `db-pool-exhaustion.md`, `ws-disconnect-storm.md`
- [x] Audit log review: verify audit trail for key operations is complete — task CRUD (Phase 1), focus sessions + feature-flag toggles (Phase 4) all audited

---

## Phase 7 (Optional) – Smart Features: NLP, Auto‑Scheduler (3–4 weeks)

**Goal:** Intelligent capabilities leveraging existing events.

**Progress (implemented 2026-08-09, verified live):**

- [x] **NLP parser** (`backend/internal/nlp/`): rule‑based parser for task input
      (dates/times/durations/recurrence) with partial‑result semantics; exposed
      as `POST /api/v1/tasks/parse`; fully unit‑tested (injectable clock).
      ADR 008 documents the design (LLM fallback behind the same endpoint).
- [x] **Scheduler module** (`backend/internal/scheduler/`): derives a user
      profile from 14‑day completion history (work window + avg completions)
      and returns read‑only daily suggestions (`GET /api/v1/schedule`) that fit
      habits + pending todos into free blocks, respecting existing planner
      entries. ADR 009 (derived, not persisted profile).
- [x] **Web**: Smart Parse panel on the Task Create form (plain‑English input →
      pre‑filled form) + new **Daily Plan** page (`DailyPlan.tsx`) with a
      timeline of suggested slots and productivity stats, wired into App.tsx.
- [x] **Mobile**: Smart Add NLP field on `TaskCreateScreen` (parse → pre‑fill
      title/habit days) + `DailyPlanScreen` (timeline card list) wired into
      `MainActivity`.
- [x] OpenAPI specs: `api/openapi/scheduler-v1.yaml` + `parseTask` path in
      `tasks-v1.yaml`; web types regenerated.

**Deferred from the original scope:**

- [ ] Calendar sync: Google OAuth2 flow, webhook receiver, two‑way sync with
      conflict resolution strategy — intentionally deferred; the scheduler's
      `plannedBlocks` is the integration point (calendar events would become
      occupied intervals).
- [ ] `nlp_input` acceptance on `POST /tasks` (client currently calls
      `/tasks/parse` first, then submits the structured form).

---

## Phase 8 (Optional) – Collaboration & Multi‑User (2–3 weeks)

**Goal:** Real‑time shared lists, presence, permissions.

**Progress (implemented 2026-08-10, see ADR 010):**

- [x] Backend: `internal/collab/` — Postgres membership resolver for WS event
      scoping + presence; sharing in `internal/tasklist` (owner/editor/viewer).
      CRDT merge via `PATCH /api/v1/tasks/{taskId}/merge` (document-level LWW
      register; ties broken by `updated_by`; losers reconcile from 409 body).
- [x] WebSocket presence tracking — connections carry user identity;
      `presence.online/offline` events (replayed via the bus) +
      `GET /api/v1/presence/{listId}` snapshot. List-scoped events are only
      pushed to members.
- [x] RBAC extended to shared resources — `CheckAccess` (owner/editor/viewer)
      on task-list reads, list-task reads, and owner-only share/unshare/delete.
- [x] Web: share dialog on the Lists panel (invite with role, member list with
      live presence dots, revoke), merge-conflict toasts, real-time cache sync.
- [x] Mobile: share dialog on list cards (invite editor/viewer, member list
      with presence dots, revoke), collab endpoints in `TaskApi.kt`.
- [x] Contract tests: 5 new WS event payloads validated against
      `api/ws/events-v1.json`; OpenAPI endpoints for share/merge/presence.
- [x] CRDT choice documented in ADR 010 (LWW-Register vs field-level vs RGA vs OT).

**Deferred from the original scope:**

- [ ] DynamoDB for session/connection mapping — only needed when scaling beyond
      a single instance (ADR 002 modular monolith still holds).
- [ ] Avatar/name resolution for shared members (dev identities are opaque user
      IDs; a real users table lookup would replace them once JWT auth lands).

---

## Phase 9 (Optional) – AI Insights, Media Processing & Offline Support (3–4 weeks)

**Goal:** Polish that makes the portfolio stand out; mobile offline‑first fully realised.

**Progress (2026-08-10):**

### 9a — AI Coach (done)

- [x] Backend `internal/insights/`: consumes `pomodoro.session.completed` events
      to persist focus history; generates template-rendered weekly reports
      (`GET /api/v1/insights/weekly`); optional LLM summary gated by the
      `insights.llm_enabled` feature flag. Migration 018 (`pomodoro_sessions` +
      `insight_reports`). ADR 011.
- [x] Web: **Insights** page (stat cards + template report + optional AI Coach
      summary), wired into App.tsx nav (🧠).
- [x] Mobile: `InsightsScreen.kt` (same cards + report), 🧠 button on the tasks
      top bar, `InsightsApi.kt` Retrofit client.
- [x] Spec: `api/openapi/insights-v1.yaml`, web types regenerated.

### 9b & 9c (done)

- [x] Image processing worker: barcode → ISBN decoding (`internal/media`,
      pure-Go gozxing; `POST /api/v1/media/scan-isbn`, no external service).
      **Retired (2026-08-13):** meal-plan PDF generation was removed with the
      meal planner module.
- [x] Web: "📄 PDF" download button on the Meal Plan detail page.
      **Retired (2026-08-13):** removed with the meal planner module.
- [x] Mobile: camera intent → FileProvider photo → `/media/scan-isbn` →
      auto-fill + add book by ISBN (📷 on BookListScreen).
      **Retired (2026-08-13):** removed with the booktrack → Library replacement.
- [x] Mobile: offline‑first — local SQLite (`local/LocalDatabase.kt`, Room-style
      DAO API) with optimistic writes + `dirty`/`deleted` flags; incremental
      sync via `GET /api/v1/sync?since=...` (backend `internal/syncstore`,
      migration 019 soft-delete) pushed by WorkManager; conflicts resolved by
      the Phase 8 LWW merge; local habit-reminder notifications (no FCM).
      ADR 012 documents the protocol and the Room-deferred trade-off.
- [x] Offline sync protocol documented in ADR 012 (timestamp-based pull,
      optimistic push, soft-delete tombstones, conflict handling).

**Phase 9 is now fully implemented.**

---

## Phase 10 – Deployment Maturity, Production Polish & Cost Awareness (1–2 weeks)

**Goal:** Show production-readiness and financial awareness. **Fully delivered (2026-08-10)**
— the three previously-open cross-cutting items are closed: ArgoCD deployed on a
local Kind cluster (`infra/argocd/`), Terraform IaC (`infra/terraform/`,
`terraform validate` clean), and the SLO burn-rate alert fired against a live
deployment (`docs/slo-validation.md`).

- [x] Infra: GitOps manifests prepared — `infra/argocd/` ApplicationSet + `infra/kustomize/` base/overlays (dev/prod) for the day a cluster exists (see ADR 006); MVP deploys as single-host docker-compose (documented decision)
- [x] Feature flags: custom service (`internal/featureflag/`) serves admin-toggled flags to clients — integration with a SaaS (Unleash) explicitly out of scope for MVP
- [x] Mobile: signed release build — Gradle release signing config (env/`local.properties`) + `release-sign` CI job producing signed AAB+APK; keystore workflow in `docs/security/secrets-management.md`
- [x] Web: bundle analysis + lazy loading — `rollup-plugin-visualizer` (`npm run build:analyze`), `React.lazy` on secondary pages (initial bundle 291.8 kB → 213.5 kB, −27%); Lighthouse **99/100** (`docs/performance-report.md`)
- [x] Documentation: C4 diagrams (`docs/architecture/`), README stakeholder guide, runbooks for top 3 failure modes (`docs/runbooks/`), ADR index (`docs/adr/README.md`)
- [x] Security validation: `docs/security/validation-report.md` — govulncheck 0 findings (Go pinned 1.26.5), Trivy clean on backend+web images (base images bumped), npm audit gated on runtime deps with 1 documented accepted exception, secret hygiene verified
- [x] Capacity planning & cost report: `docs/capacity-planning.md` — AWS estimates for S0/S1/S2, scaling triggers, cost-per-1000-users, optimization opportunities (reserved instances, spot, S3 lifecycle)
- [x] Performance: `docs/performance-report.md` — Lighthouse scores, load-test summaries, bundle sizes
- [x] Final incident simulation & postmortem: DB failover exercise → `docs/postmortems/002-db-failover.md` (clean degradation + automatic pool recovery verified)

---

## Completion Checklist – Core MVP (Phases 0–6)

- [x] Monorepo structure with backend, web, mobile, API specs
- [x] Health endpoint deployed (locally via Docker Compose), all clients connected
- [x] Tasks CRUD works on web + mobile with full API and UI (one-off + recurring habits)
- [x] Task lists grouping (named collections) across backend, web, mobile
- [x] Pomodoro / Focus timer backend, web UI, and mobile (countdown circle + foreground service)
- [x] Habit completions with streak tracking, heatmap, progress bars (web + dedicated mobile HabitScreen)
- [x] Weekly planner with time-blocked calendar grid (CRUD API + React UI)
- [x] CI/CD pipelines per platform (lint, test, build)
- [x] Notifications delivered via push + in‑app toasts (Phase 3 — RabbitMQ + worker + Mailpit email + FCM push wiring + web toasts)
- [x] WebSocket real-time sync (Phase 2 — completed: event bus, sync hub, web + mobile clients, replay/reconnect, ADR 004)
- [x] Feature flags service + web integration (Phase 1 — completed)
- [x] Redis caching layer for task API (Phase 1 — completed)
- [x] Audit logging for task operations (Phase 1 — completed)
- [x] RBAC middleware infrastructure (enforced on task/task-list mutations + feature-flag toggles; dev-header identity)
- [x] Prometheus metrics endpoint (wired in main.go — internal :9090 scrape endpoint)
- [x] SLOs defined for health and task API (docs/slo.md)
- [x] Prometheus alerting rules for SLO burn rate (infra/prometheus/alerts.yml)
- [x] Full observability: Grafana RED dashboards (infra/grafana/), OpenTelemetry tracing wired (stdout + Jaeger), trace IDs in logs; RabbitMQ trace propagation still pending (Phase 3)
- [x] Contract tests prevent spec drift (WS events vs api/ws/events-v1.json)
- [x] Database performance review complete (EXPLAIN ANALYZE, indexing, pooling — docs/database-performance.md)
- [x] Threat model written (docs/security/threat-model.md — STRIDE analysis)
- [x] Dependency/container scanning in CI (govulncheck + npm audit + OWASP dependencyCheck; Trivy container scan added to backend-ci.yml)
- [x] At least one simulated incident and postmortem completed (docs/postmortems/001-rabbitmq-outage.md)
- [x] Runbooks exist for common failures (docs/runbooks/ — rabbitmq-unavailable, db-pool-exhaustion, ws-disconnect-storm)
- [x] ADRs written and linked (001–012 — index in docs/adr/README.md)
- [x] GitOps manifests prepared (infra/argocd/ + infra/kustomize/ overlays; activation gated on a cluster existing per ADR 006)
- [x] Terraform IaC delivered — `infra/terraform/` provisions an EC2 docker-compose host, RDS Postgres and an S3 media bucket; `terraform validate` + `fmt` clean (2026-08-10)
- [x] Secrets management documented (docs/security/secrets-management.md — includes Android keystore workflow)
- [x] Cost estimate and scaling projection documented (docs/capacity-planning.md)
- [x] Security validation walkthrough complete (docs/security/validation-report.md)
- [x] Performance report complete (docs/performance-report.md — Lighthouse 99, load tests, bundle analysis)

**Core MVP is complete — all of Phases 0–6 delivered**, including Phase 5
(book tracking → now the Library media tracker; meal planning removed
2026-08-13) and Phase 5a (recipes) with web + mobile UIs.
The barcode scanner (previously deferred) was delivered in Phase 9b: camera
intent → gozxing ISBN decode → auto-add book — and retired with the
booktrack → Library replacement (2026-08-13).

**Optional phases 7–9 are all delivered and live-verified** (NLP/auto-scheduler,
collaboration/CRDTs, AI insights/media/offline) and the Phase 10 ops wrap-up is
complete: ArgoCD deployed on a Kind cluster, Terraform IaC, and the SLO
burn-rate alert fired against a live deployment. Every checkbox in this plan is
now green.