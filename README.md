# PudimProductivity

Starting as a To Do App. Hopefully adding other stuff later.

## Quick Start (Development)

Prerequisites:
- Go 1.25+
- Node 20+ (see `.nvmrc` or `web/package.json` for the exact version)
- Docker & Docker Compose

```bash
# Copy environment file and edit the passwords
cp .env.example .env

# Start the full dev stack — infrastructure (PostgreSQL, Redis, RabbitMQ)
# plus the backend and frontend dev servers
./scripts/run.sh
```

## Deploy on LAN

Deploy the entire application as a self-contained stack on any Linux machine in your local network.

### Prerequisites

- A Linux server on your LAN
- Docker & Docker Compose
- `git`

### Steps

```bash
# 1. Clone the repository on your LAN server
git clone https://github.com/diegobraga92/PudimProductivity.git ~/pudimproductivity
cd ~/pudimproductivity

# 2. Create and configure environment file
cp .env.example .env
# ⚠️ Edit .env now — change POSTGRES_PASSWORD and RABBITMQ_PASS (both default
#    to "change_me_in_production"). All host ports are configurable there too.

# 3. Start all services (PostgreSQL, Redis, RabbitMQ, Backend, Frontend)
docker compose up -d

# 4. Access the app from any device on your LAN
# Open http://<server-lan-ip>:<port> in your browser
# (default port is 3000, configurable via FRONTEND_PORT in .env)
```

The frontend (nginx) serves the built React app and proxies `/api/` requests to the backend. All services are wired together via Docker Compose networking. The first `docker compose up -d` builds the backend (Go) and frontend (React) images, so allow a few minutes before the app responds.

> ⚠️ **LAN exposure note:** the RabbitMQ management UI (`http://<server-ip>:15672`)
> is also reachable from any device on your LAN. That's fine for a personal
> network; if you want it private, remove its `ports:` entry from
> `docker-compose.yml` or block it at the firewall.

### Services

All host ports are **configurable via `.env`** — see `.env.example` for defaults.

| Service    | Default port | Description                        |
|------------|-------------|------------------------------------|
| Frontend   | 3000       | React SPA (nginx)                  |
| Backend    | 8080       | Go API (chi)                       |
| PostgreSQL | 5433       | Database                           |
| Redis      | 6379       | Cache / real-time sync store       |
| RabbitMQ   | 5672       | Async notification event bus       |
|            | 15672      | RabbitMQ management UI (web)       |

### Desktop app (Electron)

There's also a native desktop app (`desktop/`) that reuses the same React web
client. Full instructions live in [`desktop/README.md`](desktop/README.md).
Short version — to deploy it on another Linux machine and point it at this LAN
server:

1. **On the server** — the desktop app serves its UI from the `app://bundle`
   origin, so allow it to call the API cross-origin and rebuild the backend:

   ```bash
   echo 'CORS_ALLOWED_ORIGINS=app://bundle' >> .env
   docker compose up -d --build backend
   ```

2. **On the desktop machine** — build with the server URL baked in, then
   install the generated package:

   ```bash
   # web/.env.desktop:
   VITE_API_BASE_URL=http://<server-lan-ip>:8080/api/v1   # or :3000 via nginx

   cd desktop
   npm install && npm run package
   sudo apt install ./release/pudimproductivity-desktop_*_amd64.deb   # or run the .AppImage
   ```

The desktop app then talks to the server over HTTP + WebSocket (real-time sync
with the browser/Android included). A runtime override
(`window.desktop.setApiBaseUrl(...)` or the `PUDIM_API_BASE_URL` env var) wins
over the baked URL — see `desktop/README.md`.

## API

Using API-first, `api/openapi/` should be the source of truth.

## Monitoring & Observability

- **Metrics:** the backend exposes a Prometheus scrape endpoint on an internal-only
  port (`:9090/metrics`) — it is deliberately not exposed on the public port.
  Metrics include request rate/duration/in-flight, DB query counts, and Go runtime stats.
- **Alerting:** SLO burn-rate rules live in `infra/prometheus/alerts.yml`
  (health 99.5%, task API 99.0% / p95 < 200ms — see `docs/slo.md`).
- **Dashboards:** Grafana dashboard JSON files are in `infra/grafana/`
  (`red-dashboard.json` for RED metrics + SLOs, `business-kpi.json` for product KPIs).
  When provisioning, set the Prometheus datasource `uid` to `prometheus`.
- **Security:** a STRIDE threat model is maintained in `docs/security/threat-model.md`.

## Tracing (OpenTelemetry)

- Every HTTP request gets a W3C `trace_id`; the ID appears in JSON logs (`trace_id` /
  `span_id`) and is stamped onto event-bus events for downstream consumers.
- Spans export to stdout by default. To view them in Jaeger:
  `docker compose --profile tracing up -d`, set `OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318`
  in `.env`, restart the backend, then open http://localhost:16686.
- Instrumentation lives in `backend/internal/observability/`.

## Real-Time Sync

- Task changes push to all clients over WebSocket (`GET /api/v1/ws`), so updates
  made on any client (web, mobile) appear everywhere immediately — no polling.
- The sync hub uses monotonic sequence numbers and a bounded replay buffer:
  reconnecting clients pass `?last_seq=N` to catch up, or receive a `stale`
  signal to do a full REST refetch. See `docs/adr/004-websocket-consistency.md`.
- Message schema: `api/ws/events-v1.json`. Both the Vite dev proxy and the
  nginx config are wired to forward WebSocket upgrades.

## Notifications (Phase 3)

- Task events fan out to **RabbitMQ** (`task.events` exchange) via a
  `CompositeBus` alongside the in-memory WebSocket bus. A worker in
  `internal/notification/` consumes them and sends push notifications
  (Firebase Cloud Messaging).
- **Idempotency:** the `notifications` table (`UNIQUE(event_id, channel)`)
  dedupes at-least-once redeliveries.
- **Retry + DLQ:** failed sends are dead-lettered and republished up to 5 times
  (`x-retry-count`), then discarded.
- **Tracing:** W3C `traceparent` travels in AMQP headers, so worker logs share
  the producer's `trace_id`.
- **Web:** in-app toasts (`useTaskNotifier`) surface task events from the
  WebSocket stream.
- **Mobile:** `PudimFirebaseMessagingService` handles FCM push — add
  `mobile/app/google-services.json` + the Google Services Gradle plugin to use a
  real Firebase project, and set `FCM_DEVICE_TOKEN` on the backend.
- See `docs/adr/005-async-notifications.md` for the design and degradation
  matrix.

## Architecture

- **C4 model:** [System Context](docs/architecture/c4-system-context.md) and
  [Container](docs/architecture/c4-container.md) diagrams (Mermaid).
- **ADRs:** [Index of all architecture decision records](docs/adr/README.md) —
  migrations, modular monolith, caching, WebSocket consistency, async
  notifications, deployment strategy, external API integrations.

## For Product & Compliance Stakeholders

This section explains, in plain language, the trade-offs that matter for
non-engineering readers. Engineering detail lives in `docs/`.

### What the product is

A personal productivity suite: **tasks & habits**, a **weekly planner**, a
**focus timer (pomodoro)**, **real-time sync** between devices,
**notifications** (email + push), a **recipe manager** with image upload, and
a **media library** (movies, series, books and games — with CSV import, plus
optional score ratings auto-looked-up from configurable rating providers such
as IMDb (OMDb) for film and Metacritic (RAWG) for games).
Web, Desktop (Electron), Android, and a Go API backend.

### The data model: one source of truth

Every task, habit completion, and planner entry lives in one PostgreSQL
database. When you tick a task on one device, the change is saved to the
database first, then pushed to your other devices in real time. If a device
loses connectivity and misses an update, it re-synchronises from the database
the next time it connects. **You can never "lose" a completed task because a
push failed** — the database always wins.

### How changes reach other devices

We use a live connection (WebSocket) so updates feel instant. If that
connection drops — flaky Wi-Fi, phone in airplane mode, backend restart — the
device reconnects and catches up automatically. The trade-off: a notification
such as a "habit reminder" email may be delayed while the messaging service is
down. Notifications are best-effort convenience; they never change your data.

### What is (and isn't) available offline

- **Data is stored on the server**, not just on your phone. A fresh install
  pulls everything back from the database.
- **The Android app is local-first (Phase 9c):** tasks, habits and completions
  are read from a local SQLite copy, so adding/editing/ticking works offline and
  the change is saved immediately on the device. The change is pushed to the
  server as soon as connectivity returns (network regain or WebSocket reconnect)
  or on the next periodic sync; the server wins on timestamp conflicts (ADR 012).
- **The focus timer also works fully offline** (a foreground service keeps the
  countdown accurate).

### Security & privacy posture

- All sensitive configuration (database passwords, messaging credentials) lives
  in environment variables — never in the code or the repo. The one exception
  is the library rating-provider API keys (OMDb/RAWG), which can be configured
  at runtime via the admin UI (Server Settings); they are stored server-side,
  never returned by the API (masked `api_key_set` only), and excluded from
  backups (see [ADR 014](docs/adr/014-runtime-score-provider-config.md)).
- The current build uses **development-only identity headers** instead of real
  user accounts. This is the single most important known gap: before any
  multi-user or public deployment, authentication (JWT/session) and per-user
  data isolation must be implemented. It is tracked as a P0 item in our threat
  model.
- Dependency scanning runs in CI for Go, npm, and Android dependencies; a
  container image scan (Trivy) is part of the backend pipeline.

### Reliability expectations (SLOs)

- **Health of the service:** 99.5% uptime target.
- **Task API:** 99.0% of requests succeed, and 95% of them respond in under
  200 ms. These targets are monitored with burn-rate alerting.

### Data retention

We do not auto-delete data. Deleting a task is permanent (the database honours
the delete). Audit logs (who did what, when) are append-only and are not
deleted by the app.

### Cost & deployment

The MVP runs as a single-server stack (one Docker host) — deliberately cheap and
simple at this scale. The deployment is fully described in code (infrastructure
as code), so it is reproducible and reviewable. A cluster-based deployment with
automated rollouts is designed but not activated, and would only be switched on
when traffic justifies it.

## License

MIT