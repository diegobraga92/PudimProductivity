#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"
BACKEND_DIR="$ROOT_DIR/backend"

# ─── Colors ────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info()  { echo -e "${CYAN}[dev]${NC} $1"; }
log_ok()    { echo -e "${GREEN}[dev]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[dev]${NC} $1"; }
log_error() { echo -e "${RED}[dev]${NC} $1"; }

# ─── Cleanup handler ───────────────────────────────────────────────────────
cleanup() {
    echo ""
    log_info "Shutting down..."

    # Kill background processes (frontend + backend)
    if [ -n "${BACKEND_PID:-}" ]; then
        log_info "Stopping backend (PID $BACKEND_PID)..."
        kill "$BACKEND_PID" 2>/dev/null || true
        wait "$BACKEND_PID" 2>/dev/null || true
    fi

    if [ -n "${FRONTEND_PID:-}" ]; then
        log_info "Stopping frontend (PID $FRONTEND_PID)..."
        kill "$FRONTEND_PID" 2>/dev/null || true
        wait "$FRONTEND_PID" 2>/dev/null || true
    fi

    # Stop Docker services if we started them
    if [ "${DOCKER_STARTED:-}" = "true" ]; then
        log_info "Stopping Docker services..."
        docker compose -f "$ROOT_DIR/docker-compose.yml" down
    fi

    log_ok "All services stopped. Goodbye!"
    exit 0
}

trap cleanup SIGINT SIGTERM

# ─── Parse arguments ───────────────────────────────────────────────────────
SKIP_DOCKER=false
for arg in "$@"; do
    case "$arg" in
        --no-db) SKIP_DOCKER=true ;;
        *) log_warn "Unknown argument: $arg" ;;
    esac
done

# ─── 1. Install frontend dependencies if needed ────────────────────────────
if [ ! -d "$WEB_DIR/node_modules" ]; then
    log_info "node_modules not found. Running npm install..."
    (cd "$WEB_DIR" && npm install)
    log_ok "npm install completed."
else
    log_ok "node_modules found, skipping npm install."
fi

# ─── 2. Start Docker services ──────────────────────────────────────────────
if [ "$SKIP_DOCKER" = false ]; then
    log_info "Starting Docker services (postgres, rabbitmq)..."
    docker compose -f "$ROOT_DIR/docker-compose.yml" up -d postgres rabbitmq
    DOCKER_STARTED=true

    log_info "Waiting for PostgreSQL to be healthy..."
    until docker compose -f "$ROOT_DIR/docker-compose.yml" exec -T postgres \
        pg_isready -U pudim -d pudimproductivity >/dev/null 2>&1; do
        sleep 1
    done
    log_ok "PostgreSQL is healthy."
else
    log_info "Skipping Docker services (--no-db)."
fi

# ─── 3. Start backend ──────────────────────────────────────────────────────
log_info "Starting backend (go run ./cmd/server)..."
(cd "$BACKEND_DIR" && go run ./cmd/server) &
BACKEND_PID=$!
log_ok "Backend started (PID $BACKEND_PID)."

# Give the backend a moment to start
sleep 2

# ─── 4. Start frontend ─────────────────────────────────────────────────────
log_info "Starting frontend (npm run dev)..."
(cd "$WEB_DIR" && npm run dev) &
FRONTEND_PID=$!
log_ok "Frontend started (PID $FRONTEND_PID)."

# ─── 5. Print summary ──────────────────────────────────────────────────────
echo ""
log_ok "═══════════════════════════════════════════════════════════"
log_ok "  PudimProductivity is running!"
log_ok ""
log_ok "  Frontend:  http://localhost:3000"
log_ok "  Backend:   http://localhost:8080"
log_ok "  API Docs:  http://localhost:8080/api/v1/health"
log_ok ""
log_ok "  Press Ctrl+C to stop all services."
log_ok "═══════════════════════════════════════════════════════════"
echo ""

# Wait for any background process to exit (keeps script alive)
wait
