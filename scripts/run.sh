#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"
BACKEND_DIR="$ROOT_DIR/backend"

# ─── Colors & logging (shared helpers) ─────────────────────────────────────
# shellcheck source=lib/common.sh
source "$ROOT_DIR/scripts/lib/common.sh"
PREFIX="dev"

# ─── Help ──────────────────────────────────────────────────────────────────
usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Start the PudimProductivity development environment.

Options:
  --no-db    Skip starting Docker services (postgres, redis)
  --no-web   Skip starting the web frontend
  --clean    Remove Docker volumes, node_modules, and Go build cache before starting
  --help     Show this help message and exit
EOF
}

# ─── Cleanup handler ───────────────────────────────────────────────────────
# Runs on SIGINT/SIGTERM and on any exit (EXIT trap), so crashed background
# children and early error paths tear the stack down too. CLEANED guards
# against double-cleanup.
CLEANED=0
cleanup() {
    (( CLEANED )) && return 0
    CLEANED=1

    # Nothing was started yet (e.g. --help or an early error) — nothing to
    # tear down.
    if [ -z "${BACKEND_PID:-}" ] && [ -z "${FRONTEND_PID:-}" ] && [ "${DOCKER_STARTED:-}" != "true" ]; then
        return 0
    fi

    echo ""
    log_info "Shutting down..."

    # Kill background processes (frontend + backend)
    if [ -n "${BACKEND_PID:-}" ]; then
        log_info "Stopping backend (PID $BACKEND_PID)..."
        kill "$BACKEND_PID" 2>/dev/null || true
        sleep 2
        kill -9 "$BACKEND_PID" 2>/dev/null || true
        wait "$BACKEND_PID" 2>/dev/null || true
    fi

    # Broad fallback: catch any Go backend process that escaped the PID kill
    pkill -9 -f "go run.*cmd/server" 2>/dev/null || true

    if [ -n "${FRONTEND_PID:-}" ]; then
        log_info "Stopping frontend (PID $FRONTEND_PID)..."
        kill "$FRONTEND_PID" 2>/dev/null || true
        sleep 2
        kill -9 "$FRONTEND_PID" 2>/dev/null || true
        wait "$FRONTEND_PID" 2>/dev/null || true
    fi

    # Stop Docker services if we started them
    if [ "${DOCKER_STARTED:-}" = "true" ]; then
        log_info "Stopping Docker services..."
        docker compose -f "$ROOT_DIR/docker-compose.yml" down
    fi

    log_ok "All services stopped. Goodbye!"
}

trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM
trap 'cleanup' EXIT

# ─── Parse arguments ───────────────────────────────────────────────────────
SKIP_DOCKER=false
SKIP_WEB=false
CLEAN=false
for arg in "$@"; do
    case "$arg" in
        --no-db)  SKIP_DOCKER=true ;;
        --no-web) SKIP_WEB=true ;;
        --clean)  CLEAN=true ;;
        --help)   usage; exit 0 ;;
        *) log_warn "Unknown argument: $arg"; usage >&2; exit 2 ;;
    esac
done

# ─── 0. Clean (if requested) ───────────────────────────────────────────────
if [ "$CLEAN" = true ]; then
    log_info "Cleaning environment..."

    # Tear down Docker volumes (postgres + redis data)
    if docker compose -f "$ROOT_DIR/docker-compose.yml" ps --quiet 2>/dev/null | grep -q .; then
        log_info "Removing Docker containers and volumes..."
        docker compose -f "$ROOT_DIR/docker-compose.yml" down -v
    fi

    # Remove node_modules
    if [ -d "$WEB_DIR/node_modules" ]; then
        log_info "Removing node_modules..."
        rm -rf "$WEB_DIR/node_modules"
    fi

    # Clean Go build cache
    log_info "Cleaning Go build cache..."
    (cd "$BACKEND_DIR" && go clean -cache)

    log_ok "Clean complete."
fi

# ─── 1. Install frontend dependencies if needed ────────────────────────────
if [ ! -d "$WEB_DIR/node_modules" ]; then
    log_info "node_modules not found. Running npm install..."
    (cd "$WEB_DIR" && npm install)
    log_ok "npm install completed."
else
    log_ok "node_modules found, skipping npm install."
fi

# ─── 2. Download Go dependencies if needed ─────────────────────────────────
if [ ! -f "$BACKEND_DIR/go.sum" ]; then
    log_info "go.sum not found. Running go mod download..."
    (cd "$BACKEND_DIR" && go mod download)
    log_ok "go mod download completed."
else
    log_ok "go.sum found, skipping go mod download."
fi

# ─── 3. Export .env variables (required before starting any service) ────────
if [ -f "$ROOT_DIR/.env" ]; then
    set -a
    source "$ROOT_DIR/.env"
    set +a
    log_ok ".env loaded into environment."
else
    log_error ".env file not found at $ROOT_DIR/.env"
    exit 1
fi

# ─── 4. Start Docker services ──────────────────────────────────────────────
if [ "$SKIP_DOCKER" = false ]; then
    log_info "Starting Docker services (postgres, redis)..."
    # `--wait` blocks until every started service passes its healthcheck.
    # Healthchecks use the .env credentials via the compose file, so no roles
    # are hardcoded here.
    docker compose -f "$ROOT_DIR/docker-compose.yml" up -d --wait postgres redis
    DOCKER_STARTED=true
    log_ok "All Docker services are healthy."
else
    log_info "Skipping Docker services (--no-db)."
fi

# ─── 5. Start backend ──────────────────────────────────────────────────────
log_info "Starting backend (go run ./cmd/server)..."
(cd "$BACKEND_DIR" && go run ./cmd/server) &
BACKEND_PID=$!
log_ok "Backend started (PID $BACKEND_PID)."

# ─── 6. Start frontend (unless --no-web) ────────────────────────────────────
if [ "$SKIP_WEB" = false ]; then
    log_info "Starting frontend (npm run dev)..."
    (cd "$WEB_DIR" && npm run dev) &
    FRONTEND_PID=$!
    log_ok "Frontend started (PID $FRONTEND_PID)."
else
    log_info "Skipping web frontend (--no-web)."
fi

# ─── 7. Wait for the backend health endpoint ────────────────────────────────
# Replaces the old fixed `sleep 2` with a readiness poll. The first `go run`
# compiles the backend, so allow generous time; this is informational only —
# the frontend already started in parallel and we continue either way.
if command -v curl >/dev/null 2>&1; then
    log_info "Waiting for backend health check..."
    backend_ready=false
    poll_count=0
    while [ "$poll_count" -lt 60 ]; do
        if curl -fsS "http://localhost:8080/api/v1/health" >/dev/null 2>&1; then
            backend_ready=true
            break
        fi
        sleep 1
        poll_count=$((poll_count + 1))
    done
    if [ "$backend_ready" = true ]; then
        log_ok "Backend is healthy."
    else
        log_warn "Backend not responding on /api/v1/health after 60s — continuing (first build can be slow)."
    fi
else
    log_warn "curl not found — skipping backend health check."
fi

# ─── 8. Print summary ──────────────────────────────────────────────────────
echo ""
log_ok "═══════════════════════════════════════════════════════════"
log_ok "  PudimProductivity is running!"
log_ok ""
log_ok "  Frontend:  http://localhost:3000"
log_ok "  Backend:   http://localhost:8080"
log_ok "  Backend Health: http://localhost:8080/api/v1/health"
log_ok ""
log_ok "  Press Ctrl+C to stop all services."
log_ok "═══════════════════════════════════════════════════════════"
echo ""

# Wait for any background process to exit (keeps script alive)
wait