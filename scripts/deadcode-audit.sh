#!/usr/bin/env bash
#
# deadcode-audit.sh — dead-code audit across the PudimProductivity monorepo.
#
# Runs the same checks used in the manual audit:
#   backend : go build, go vet, go mod tidy -diff, deadcode, staticcheck
#   web     : tsc --noEmit, eslint, knip, OpenAPI typegen stability
#   desktop : tsc --noEmit
#   mobile  : :app:compileDebugKotlin, :app:lintDebug   (skippable)
#
# The Go tools (deadcode, staticcheck) are installed on first use to
# $PUDIM_AUDIT_TOOL_BIN (default: ~/.cache/pudim-deadcode/bin).
#
# Exit code: 0 if every enabled check passed, 1 if anything failed.

set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOL_BIN="${PUDIM_AUDIT_TOOL_BIN:-$HOME/.cache/pudim-deadcode/bin}"

# ─── Help ──────────────────────────────────────────────────────────────────
usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Run a dead-code audit across the PudimProductivity monorepo.

Options:
  --skip-mobile  skip the Android Gradle checks (slow; needs Android SDK)
  --offline      pass --offline to Gradle (use cached dependencies only)
  --verbose      print the full output of passing checks
  -h, --help     show this help message and exit
EOF
}

SKIP_MOBILE=false
GRADLE_OFFLINE=""
VERBOSE=false

for arg in "$@"; do
  case "$arg" in
    --skip-mobile) SKIP_MOBILE=true ;;
    --offline)     GRADLE_OFFLINE="--offline" ;;
    --verbose)     VERBOSE=true ;;
    -h|--help)     usage; exit 0 ;;
    *)
      echo "Unknown option: $arg (see --help)" >&2
      exit 2
      ;;
  esac
done

PASS=0
FAIL=0
SKIP=0
declare -a RESULTS=()

# ─── Colors & logging (shared helpers) ─────────────────────────────────────
# shellcheck source=lib/common.sh
source "$ROOT_DIR/scripts/lib/common.sh"

log()  { printf "\n${BOLD}%s${NC}\n" "$*"; }
note() { printf "  ${CYAN}%s${NC}\n" "$*"; }
ok()   { printf "  ${GREEN}✔ %s${NC}\n" "$*"; }
bad()  { printf "  ${RED}✘ %s${NC}\n" "$*"; }
skip() { SKIP=$((SKIP + 1)); printf "  ${YELLOW}- %s${NC}\n" "$*"; }

# check <name> <cmd...>: run a check, record and report the result.
check() {
  local name="$1"
  shift
  local logfile
  logfile="$(mktemp)"
  if "$@" >"$logfile" 2>&1; then
    PASS=$((PASS + 1))
    RESULTS+=("PASS|$name")
    ok "$name"
    if [ "$VERBOSE" = true ] && [ -s "$logfile" ]; then
      sed 's/^/       /' "$logfile"
    fi
  else
    FAIL=$((FAIL + 1))
    RESULTS+=("FAIL|$name")
    bad "$name"
    tail -40 "$logfile" | sed 's/^/       /' >&2
  fi
  rm -f "$logfile"
}

# run_in <dir> <cmd...>: run a command with a specific working directory.
run_in() {
  local dir="$1"
  shift
  (cd "$dir" && "$@")
}

has() { command -v "$1" >/dev/null 2>&1; }

# ─── Prerequisites ────────────────────────────────────────────────────────
if [ "$SKIP_MOBILE" = true ]; then
  note "mobile checks disabled (--skip-mobile)"
fi
has go   || note "go not found — backend checks will be skipped"
has node || note "node/npm not found — web/desktop checks will be skipped"
has java || note "java not found — mobile checks will be skipped"

# Install the Go dead-code tools on demand (idempotent).
install_go_tool() {
  local pkg="$1"
  local bin="$2"
  if [[ -x "$TOOL_BIN/$bin" ]]; then
    return 0
  fi
  note "installing $bin (first run)..."
  mkdir -p "$TOOL_BIN"
  GOBIN="$TOOL_BIN" go install "$pkg"
}

# ─── Backend ──────────────────────────────────────────────────────────────
log "backend"
if has go; then
  check "backend: go build"          run_in "$ROOT_DIR/backend" go build ./...
  check "backend: go vet"            run_in "$ROOT_DIR/backend" go vet ./...
  check "backend: go mod tidy -diff" run_in "$ROOT_DIR/backend" go mod tidy -diff

  # Pinned versions so audit results are reproducible across machines/days.
  DEADCODE_PKG="golang.org/x/tools/cmd/deadcode@v0.49.0"
  STATICCHECK_PKG="honnef.co/go/tools/cmd/staticcheck@v0.8.1"

  # Each tool is installed and checked independently: a failed install of
  # one must not skip the other.
  if install_go_tool "$DEADCODE_PKG" deadcode; then
    check "backend: deadcode -test ./..." run_in "$ROOT_DIR/backend" "$TOOL_BIN/deadcode" -test ./...
  else
    skip "backend: deadcode install failed"
  fi
  if install_go_tool "$STATICCHECK_PKG" staticcheck; then
    check "backend: staticcheck ./..."    run_in "$ROOT_DIR/backend" "$TOOL_BIN/staticcheck" ./...
  else
    skip "backend: staticcheck install failed"
  fi
else
  skip "backend: checks skipped (go missing)"
fi

# ─── Web ──────────────────────────────────────────────────────────────────
log "web"
if has node; then
  check "web: tsc --noEmit" run_in "$ROOT_DIR/web" npx tsc --noEmit
  check "web: eslint"       run_in "$ROOT_DIR/web" npm run lint
  check "web: knip"         run_in "$ROOT_DIR/web" npx -y knip --reporter compact
  check "web: api typegen stable" bash -c '
    set -euo pipefail
    gen_dir="'"$ROOT_DIR"'/web/src/api/generated"
    backup="$(mktemp -d)"
    # Always restore the working tree, even if the generator crashes midway.
    trap "rm -rf \"$gen_dir\"; mkdir -p \"$gen_dir\"; cp -r \"$backup\"/. \"$gen_dir\"/; rm -rf \"$backup\"" EXIT
    cp -r "$gen_dir"/. "$backup"/
    ( cd "'"$ROOT_DIR"'/web" && node scripts/generate-api-types.mjs >/dev/null )
    diff -rq "$backup" "$gen_dir" >/dev/null
  '
else
  skip "web: checks skipped (node missing)"
fi

# ─── Desktop ──────────────────────────────────────────────────────────────
log "desktop"
if has node; then
  check "desktop: tsc --noEmit" run_in "$ROOT_DIR/desktop" npm run typecheck
else
  skip "desktop: checks skipped (node missing)"
fi

# ─── Mobile ───────────────────────────────────────────────────────────────
log "mobile"
if [ "$SKIP_MOBILE" = true ]; then
  skip "mobile: checks skipped (--skip-mobile)"
elif has java; then
  check "mobile: compileDebugKotlin" run_in "$ROOT_DIR/mobile" ./gradlew :app:compileDebugKotlin $GRADLE_OFFLINE
  check "mobile: lintDebug"          run_in "$ROOT_DIR/mobile" ./gradlew :app:lintDebug          $GRADLE_OFFLINE
else
  skip "mobile: checks skipped (java missing)"
fi

# ─── Summary ──────────────────────────────────────────────────────────────
log "summary"
for r in "${RESULTS[@]}"; do
  case "$r" in
    PASS*) ok "${r#PASS|}" ;;
    FAIL*) bad "${r#FAIL|}" ;;
  esac
done
printf "  ${BOLD}%d passed, %d failed, %d skipped${NC}\n" "$PASS" "$FAIL" "$SKIP"
if [ "$FAIL" = 0 ]; then
  log "audit passed"
  exit 0
fi
log "audit failed"
exit 1
