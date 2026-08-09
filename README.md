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

# Start infrastructure (PostgreSQL)
docker compose up -d postgres

# Start backend and frontend dev servers
./scripts/dev.sh
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

# 3. Start all services (PostgreSQL, Backend, Frontend)
docker compose up -d

# 4. Access the app from any device on your LAN
# Open http://<server-lan-ip>:<port> in your browser
# (default port is 3000, configurable via FRONTEND_PORT in .env)
```

The frontend (nginx) serves the built React app and proxies `/api/` requests to the backend. All services are wired together via Docker Compose networking.

### Services

All host ports are **configurable via `.env`** — see `.env.example` for defaults.

| Service    | Default port | Description              |
|------------|-------------|--------------------------|
| Frontend   | 3000       | React SPA (nginx)        |
| Backend    | 8080       | Go API (chi)             |
| PostgreSQL | 5433       | Database                  |

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

## License

MIT