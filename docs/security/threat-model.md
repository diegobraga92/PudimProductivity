# Threat Model — PudimProductivity

> Simple STRIDE analysis for the PudimProductivity full-stack productivity suite.
> Status: **Draft** — reviewed as part of Phase 1 security groundwork; must be
> updated whenever authentication, external integrations, or the deployment
> topology change.
>
> Last reviewed: 2026-08-08

---

## 1. System Overview

### Components in Scope

| Component | Technology | Notes |
|-----------|-----------|-------|
| Backend API | Go (chi), modular monolith | PostgreSQL, Redis (task cache), in-memory pomodoro |
| Web client | React + TypeScript + Vite | Talks to backend via REST |
| Mobile client | Kotlin + Jetpack Compose (Retrofit) | Talks to backend via REST |
| Database | PostgreSQL 16 | Hosted via Docker locally; RDS in planned prod |
| Cache | Redis 7 | Graceful degradation to no-op |
| Observability | Prometheus (:9090), Grafana (planned) | Metrics endpoint is internal-only |

### Trust Boundaries

1. **Internet / LAN → Web & Mobile clients** (HTTPS in production; HTTP locally)
2. **Clients → Backend API** (REST over HTTP)
3. **Backend → PostgreSQL / Redis** (internal network, no TLS locally)
4. **Backend :9090 → Prometheus** (internal-only scrape endpoint)
5. **Backend → Google Books API** (Phase 5, future)

### Actors & Roles

| Actor | Role | Privileges |
|-------|------|------------|
| Anonymous visitor | `anonymous` | Read-only API access (GET endpoints) |
| Registered user | `user` | Read + create/update/delete tasks, task lists, pomodoro |
| Administrator | `admin` | Everything above + feature-flag toggles |
| DevOps / SRE | — | Infrastructure, secrets, observability |

**Identity mechanism (current, dev-mode):** `AuthMiddleware` reads `X-User-ID` /
`X-User-Role` request headers and places them in the request context. There is no
token validation. **This is a known, accepted risk in development only** — a real
JWT/session mechanism is required before any non-local deployment.

---

## 2. STRIDE Analysis

### 2.1 Spoofing (Identity / Impersonation)

| # | Threat | Affected Asset | Likelihood | Impact | Current Mitigation | Recommendation |
|---|--------|----------------|-----------|--------|--------------------|----------------|
| S1 | Attacker forges `X-User-ID` / `X-User-Role` headers to impersonate a user or admin | All user-scoped data; feature-flag toggles | **High** (dev-mode) | **High** — anyone can mutate tasks as any user, or toggle flags as admin | Header-based trust only; `RequireRole` checks the header value | Replace header trust with signed JWTs or server-side sessions before production; keep dev headers behind a build flag |
| S2 | Attacker impersonates the backend (DNS spoofing / ARP on LAN) | Client ↔ backend traffic | Medium (LAN) | Medium — client could send credentials to a rogue server | None (HTTP locally) | Use TLS/HTTPS everywhere except localhost-only dev; pin certificates in mobile |
| S3 | Mobile app impersonation (repackaging) | Backend API | Low | Medium | None | App attestation (Play Integrity / App Check) when auth is added |

### 2.2 Tampering (Data Modification)

| # | Threat | Affected Asset | Likelihood | Impact | Current Mitigation | Recommendation |
|---|--------|----------------|-----------|--------|--------------------|----------------|
| T1 | MITM modifies task data / audit events in transit | Tasks, completions, audit log | Medium (HTTP on LAN) | Medium | None — HTTP locally | TLS in production; consider request signing or MAC for highly sensitive writes |
| T2 | Unauthorized client modifies data by calling mutation endpoints | Tasks, task lists | **High** — anonymous callers get HTTP 403, but any header can claim `user` | Medium | `RequireRole("admin","user")` on all task/task-list mutations; 403 for anonymous | JWT with server-side verification; per-user ownership filtering (`WHERE user_id = ...`) |
| T3 | Client tampers with local Redis cache | Cached task data | Low | Low | Cache is read-only (lazy populating) | Ensure cache keys are server-generated; treat cache as untrusted input |

### 2.3 Repudiation (Deny Actions)

| # | Threat | Affected Asset | Likelihood | Impact | Current Mitigation | Recommendation |
|---|--------|----------------|-----------|--------|--------------------|----------------|
| R1 | User denies creating/updating/deleting a task | Audit log | Medium | Medium | `internal/audit/` logs task CRUD + completions to Postgres (`audit_log` table), with actor ID | Audit logs are decent for task ops; add audit for pomodoro sessions and feature-flag toggles (see Phase 4) |
| R2 | Admin denies toggling a feature flag | Feature flags | Medium | Medium | **None** — no audit record for flag toggles | Add audit logging to `featureflag` toggle handler |



### 2.4 Information Disclosure (Confidentiality)

| # | Threat | Affected Asset | Likelihood | Impact | Current Mitigation | Recommendation |
|---|--------|----------------|-----------|--------|--------------------|----------------|
| I1 | Anyone can list all tasks (GET endpoints unauthenticated) | Task data (all users) | **High** (dev-mode) | **High** — full task inventory leak across users | Read endpoints deliberately left open for the demo | Enforce authentication on reads in production; scope queries to `user_id` |
| I2 | `:9090` metrics endpoint exposed | Request volumes, latency distributions | Medium | Medium | Metrics served on separate port; not in the public docker-compose port mapping | Keep :9090 bound to internal network only; never publish in prod Ingress/ELB |
| I3 | Debug logging (`LOG_LEVEL=debug`, OkHttp BODY logging) leaks request/response bodies | Task titles, user headers | Medium | Low (dev) | Default `.env.example` uses `info`; mobile logging interceptor level BODY | Set log level to `info` in all non-local envs; remove BODY logging from release builds |
| I4 | Secrets committed to git | DB passwords, API keys | Medium | **Critical** | `.env` in `.gitignore`; `.env.example` contains only placeholders | Add `gitleaks` pre-commit hook and CI secret scan (see secrets-management.md) |

### 2.5 Denial of Service

| # | Threat | Affected Asset | Likelihood | Impact | Current Mitigation | Recommendation |
|---|--------|----------------|-----------|--------|--------------------|----------------|
| D1 | Request flood against the API | Backend availability | Medium | Medium | `middleware.Timeout` caps request duration; connection pool caps DB load | Add rate limiting per client/IP; enforce max body sizes; horizontal scaling via EKS |
| D2 | Connection pool exhaustion (slow queries + many clients) | PostgreSQL | Medium | High | pgxpool with configured min/max conns | Tune pool (`DATABASE_MAX_CONNS`), add slow-query monitoring, index hot paths (Phase 6) |
| D3 | Redis outage causes cascading DB load | Backend + DB | Low | Medium | `shared.Cache` degrades to no-op when Redis is down | Document graceful degradation; monitor cache hit rate |

### 2.6 Elevation of Privilege

| # | Threat | Affected Asset | Likelihood | Impact | Current Mitigation | Recommendation |
|---|--------|----------------|-----------|--------|--------------------|----------------|
| E1 | Forge `X-User-Role: admin` to toggle feature flags | Feature flags, product behavior | **High** (dev-mode) | **High** | `RequireRole("admin")` on `PUT /features/{name}/toggle` | JWT with verified role claims; never trust client-supplied role headers |
| E2 | SQL injection via task title / list names | Database | Low | **Critical** | All queries use parameterized SQL via pgx | Keep parameterized queries as a hard rule; add SQL static analysis to CI |
| E3 | Path traversal / IDOR via `{taskId}`, `{listId}` | Other users' tasks/lists | Medium | Medium | None — handlers look up by ID without ownership check | Scope all queries by authenticated `user_id`; validate UUID format |

---

## 3. Risk Summary & Priority

| Priority | Threat IDs | Action Required |
|----------|-----------|-----------------|
| **P0 — fix before any shared/prod deployment** | S1, E1, I1, T2, E3 | Replace dev headers with JWT auth; enforce per-user data scoping |
| **P1 — before Phase 3 (external integrations)** | D4, I4, S3 | Gitleaks in CI; audit feature-flag toggles; secrets via AWS Secrets Manager |
| **P2 — during Phase 6 hardening** | D1, D2, D3, I2, I3 | Rate limiting, pool tuning, metrics hardening, log level policy |
| **P3 — ongoing** | R1, R2, T1, S2 | Extend audit coverage; TLS everywhere; periodic re-review |

---

## 4. Explicitly Out of Scope (Current MVP)

- OAuth2 / third-party identity (Phase 7)
- Payment / PII handling
- Multi-tenant isolation beyond `user_id` scoping (Phase 8)
- File upload security (S3 presigned URLs, Phase 5)

---

## 5. Re-Review Triggers

- Introduction of authentication (JWT) — mandatory
- Any change to trust boundaries (e.g., exposing :9090)
- New external API integrations (Google Books, calendar sync)
- Deployment topology change (Docker → EKS/RDS)

