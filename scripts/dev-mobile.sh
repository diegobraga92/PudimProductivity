#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MOBILE_DIR="$ROOT_DIR/mobile"
export ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
AVD_NAME="Pixel_9_API_36"
BOOT_TIMEOUT=120

# ─── Colors ────────────────────────────────────────────────────────────────
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}[dev-mobile]${NC} $1"; }
log_ok()    { echo -e "${GREEN}[dev-mobile]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[dev-mobile]${NC} $1"; }
log_error() { echo -e "${RED}[dev-mobile]${NC} $1"; }

# ─── Help ──────────────────────────────────────────────────────────────────
usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Start the PudimProductivity mobile development environment.
Delegates DB/backend/web setup to dev.sh, then boots the emulator,
builds and installs the Android app, and launches it.

Options:
  --no-emulator   Build and install to an already-running device/emulator only
  --no-db         Skip starting Docker services (postgres) — passed to dev.sh
  --no-web        Skip web frontend — passed to dev.sh
  --help          Show this help message and exit
EOF
    exit 0
}

# ─── Parse arguments ───────────────────────────────────────────────────────
SKIP_EMULATOR=false
DEV_SH_ARGS=()
for arg in "$@"; do
    case "$arg" in
        --no-emulator)  SKIP_EMULATOR=true ;;
        --no-db)        DEV_SH_ARGS+=("--no-db") ;;
        --no-web)       DEV_SH_ARGS+=("--no-web") ;;
        --help)         usage ;;
        *) log_warn "Unknown argument: $arg"; usage ;;
    esac
done

# Forward --no-web by default (mobile most commonly conflicts on port 3000)
has_web_arg=false
for a in "${DEV_SH_ARGS[@]}"; do
    if [ "$a" = "--no-web" ]; then
        has_web_arg=true
        break
    fi
done
if [ "$has_web_arg" = false ]; then
    DEV_SH_ARGS+=("--no-web")
fi

# ─── Cleanup handler ───────────────────────────────────────────────────────
cleanup() {
    echo ""
    log_info "Shutting down..."

    # 1. Stop the emulator (via ADB + PID + force-kill children)
    if [ "${EMULATOR_STARTED:-}" = "true" ]; then
        log_info "Stopping emulator..."
        # Graceful shutdown via ADB
        adb emu kill 2>/dev/null || true
        if [ -n "${EMULATOR_PID:-}" ]; then
            # Kill child QEMU processes first
            pkill -P "$EMULATOR_PID" 2>/dev/null || true
            # Polite SIGTERM
            kill "$EMULATOR_PID" 2>/dev/null || true
            sleep 2
            # Force SIGKILL if still alive
            kill -9 "$EMULATOR_PID" 2>/dev/null || true
        fi
        # Catch any orphaned QEMU processes by AVD name
        pkill -9 -f "qemu.*${AVD_NAME}" 2>/dev/null || true
    fi

    # 2. Stop the backend stack launched by dev.sh
    log_info "Stopping backend (go process)..."
    pkill -9 -f "go run.*cmd/server" 2>/dev/null || true

    log_info "Stopping frontend (vite)..."
    pkill -9 -f "vite" 2>/dev/null || true

    log_info "Stopping Docker services..."
    docker compose -f "$ROOT_DIR/docker-compose.yml" down 2>/dev/null || true

    # 3. Stop dev.sh itself (if still alive after children are gone)
    if [ -n "${DEV_PID:-}" ]; then
        kill "$DEV_PID" 2>/dev/null || true
        wait "$DEV_PID" 2>/dev/null || true
    fi

    log_ok "All services stopped. Goodbye!"
    exit 0
}

trap cleanup SIGINT SIGTERM

# ─── 1. Launch backend, database, and other services via dev.sh ────────────
log_info "Starting backend stack via dev.sh..."
"$SCRIPT_DIR/dev.sh" "${DEV_SH_ARGS[@]}" &
DEV_PID=$!
log_ok "dev.sh started (PID $DEV_PID)."

# Give dev.sh time to boot services before starting the emulator
sleep 3

# ─── 2. Start Android emulator ─────────────────────────────────────────────
if [ "$SKIP_EMULATOR" = false ]; then
    # Warn if no display is available
    if [ -z "${DISPLAY:-}" ]; then
        log_warn "DISPLAY is not set. The emulator will boot but the window will not be visible."
        log_warn "  Connect a device via USB or use --no-emulator to skip."
    fi

    DEVICE_COUNT=$("$ANDROID_HOME/platform-tools/adb" devices 2>/dev/null | grep -v "List of devices" | grep -v "^$" | grep -v "offline" | wc -l | tr -d '[:space:]' || echo 0)

    if [ "${DEVICE_COUNT:-0}" -gt 0 ]; then
        log_ok "Device/emulator already connected, skipping boot."
    else
        AVD_LIST=$("$ANDROID_HOME/emulator/emulator" -list-avds 2>/dev/null)

        if ! echo "$AVD_LIST" | grep -q "^${AVD_NAME}$"; then
            log_warn "AVD '$AVD_NAME' not found. Available AVDs:"
            echo "$AVD_LIST"
            log_error "Create it first with: $ANDROID_HOME/cmdline-tools/latest/bin/avdmanager create avd -n $AVD_NAME -k 'system-images;android-36;default;x86_64'"
            exit 1
        fi

        log_info "Booting Android emulator (AVD: $AVD_NAME)..."
        "$ANDROID_HOME/emulator/emulator" -avd "$AVD_NAME" -no-audio -gpu swiftshader_indirect &
        EMULATOR_PID=$!
        EMULATOR_STARTED=true

        log_info "Waiting for emulator to boot (timeout: ${BOOT_TIMEOUT}s)..."
        boot_elapsed=0
        booted=false
        while [ $boot_elapsed -lt $BOOT_TIMEOUT ]; do
            boot_completed=$("$ANDROID_HOME/platform-tools/adb" shell getprop sys.boot_completed 2>/dev/null | tr -d '\r\n' || echo "")
            if [ "$boot_completed" = "1" ]; then
                booted=true
                break
            fi
            sleep 2
            boot_elapsed=$((boot_elapsed + 2))
        done

        if [ "$booted" = true ]; then
            log_ok "Emulator booted successfully."
        else
            log_error "Emulator did not boot within ${BOOT_TIMEOUT}s."
            # Clean up the emulator process we started
            if [ -n "${EMULATOR_PID:-}" ]; then
                kill "$EMULATOR_PID" 2>/dev/null || true
            fi
            exit 1
        fi
    fi
else
    log_info "Skipping emulator (--no-emulator)."
fi

# ─── 3. Build and install the Android app ──────────────────────────────────
log_info "Building and installing the Android app..."

cd "$MOBILE_DIR"
if ! ./gradlew installDebug; then
    log_error "Build or install failed."
    exit 1
fi
log_ok "App installed on device/emulator."

# ─── 4. Launch the app on the device ───────────────────────────────────────
log_info "Launching PudimProductivity..."
"$ANDROID_HOME/platform-tools/adb" shell am start -n com.pudimproductivity/.MainActivity 2>/dev/null || \
    log_warn "Could not auto-launch the app (may need to open manually)."
log_ok "App launched!"

# ─── 5. Print summary ──────────────────────────────────────────────────────
echo ""
log_ok "═══════════════════════════════════════════════════════════"
log_ok "  PudimProductivity mobile environment is running!"
log_ok ""
log_ok "  App:        installed on emulator/device"
log_ok "  Backend:    http://localhost:8080"
log_ok "  API Docs:   http://localhost:8080/api/v1/health"
log_ok ""
log_ok "  Press Ctrl+C to stop all services."
log_ok "═══════════════════════════════════════════════════════════"
echo ""

# Wait for dev.sh to exit (keeps script alive, forwards Ctrl+C)
wait "$DEV_PID"