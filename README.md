# PudimProductivity

Starting as a To Do App. Hopefully adding other stuff later.

## Quick Start (Development)

Prerequisites:
- Go 1.25+
- Node 18+
- Docker & Docker Compose

```bash
# Start infrastructure (PostgreSQL, RabbitMQ)
docker compose up -d postgres rabbitmq

# Start backend and frontend dev servers
./scripts/dev.sh
```

## Deploy on LAN

Deploy the entire application as a self-contained stack on any Linux machine in your local network.

### Prerequisites

- A Linux server on your LAN
- Docker & Docker Compose

### Steps

```bash
# 1. Clone the repository on your LAN server
git clone <repo-url> /opt/pudimproductivity
cd /opt/pudimproductivity

# 2. Start all services (PostgreSQL, RabbitMQ, Backend, Frontend)
docker compose up -d

# 3. Access the app from any device on your LAN
# Open http://<server-lan-ip>:3000 in your browser
```

The frontend (nginx) serves the built React app on port `3000` and proxies `/api/` requests to the backend. All services are wired together via Docker Compose networking.

### Services

| Service    | Port | Description              |
|------------|------|--------------------------|
| Frontend   | 3000 | React SPA (nginx)        |
| Backend    | 8080 | Go API (chi)             |
| PostgreSQL | 5433 | Database                  |
| RabbitMQ   | 5672 | Message broker           |
| RabbitMQ UI| 15672| Management dashboard     |

## API

Using API-first, `api/openapi/` should be the source of truth.

## License

MIT
