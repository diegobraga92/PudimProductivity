# 🍮 PudimProductivity

A full‑stack personal productivity platform — task management, habit tracking, focus timers, meal planning, book tracking, smart scheduling, collaboration, and AI‑powered insights.

## Architecture

```
pudimproductivity/
├── backend/          # Go (Chi, pgx, RabbitMQ, OpenTelemetry)
├── web/              # React + TypeScript (Vite, React Query)
├── mobile/           # Android (Kotlin, Jetpack Compose)
├── api/              # OpenAPI specs
├── infra/            # Terraform (AWS EKS + RDS)
└── .github/          # CI/CD workflows
```

- **Modular monolith**: bounded contexts in `backend/internal/` — no cross‑imports.
- **Ports & adapters**: domain interfaces, Postgres implementations.
- **Event‑driven**: in‑memory bus (Phase 2) → RabbitMQ (Phase 3).
- **API‑first**: all endpoints defined in `api/openapi/*.yaml`.

## Quick Start

### Prerequisites

- Go 1.22+
- Node 18+
- Docker & Docker Compose

### 1. Start infrastructure

```bash
docker compose up -d
```

This starts PostgreSQL (port 5432) and RabbitMQ (ports 5672, 15672).

### 2. Start the backend

```bash
cd backend
go run ./cmd/server/
```

Health check: [http://localhost:8080/api/v1/health](http://localhost:8080/api/v1/health)

### 3. Start the web frontend

```bash
cd web
npm install
npm run dev
```

Opens at [http://localhost:3000](http://localhost:3000) — proxies `/api` to the backend.

### 4. Mobile (Android)

Open `mobile/` in Android Studio, sync Gradle, and run on an emulator.
The app connects to `http://10.0.2.2:8080/api/v1` (Android emulator loopback).

## Development

### Backend

```bash
cd backend
go test ./... -v -race
go build ./cmd/server/
```

### Web

```bash
cd web
npm run lint        # ESLint
npm run typecheck   # TypeScript
npm run build       # Production build
```

### API Specs

All OpenAPI specs live in `api/openapi/`. They are the source of truth for client‑server contracts.

## License

MIT
