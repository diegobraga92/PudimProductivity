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
git clone https://github.com/diegobraga92/PudimProductivity.git /opt/pudimproductivity
cd /opt/pudimproductivity

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

## License

MIT