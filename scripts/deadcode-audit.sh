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
# Usage:
#   ./scripts/deadcode-audit.sh [--skip-mobile] [--offline] [--verbose] [--help]
#
#   --skip-mobile  skip the Android Gradle checks (slow; needs Android SDK)
#   --offline      pass --offline to Gradle (use cached dependencies only)
#   --verbose      print the full output of passing checks
#
# The Go tools (deadcode, staticcheck) are installed on first use to
# $PUDIM_AUDIT_TOOL_BIN (default: ~/.cache/pudim-deadcode/bin).
#
# Exit code: 0 if every enabled check passed, 1 if anything failed.

set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOL_BIN="${PUDIM_AUDIT_TOOL_BIN:-$HOME/.cache/pudim-deadcode/bin}"

SKIP_MOBILE=0
GRADLE_OFFLINE=""
VERBOSE=0

for arg in "$@"; do
  case "$arg" in
    --skip-mobile) SKIP_MOBILE=1 ;;
    --offline)     GRADLE_OFFLINE="--offline" ;;
    --verbose)     VERBOSE=1 ;;
    -h|--help)
      sed -n '2,22p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
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

log()  { printf '\n\033[1m%s\033[0m\n' "$*"; }
note() { printf '  \033[36m%s\033[0m\n' "$*"; }
ok()   { printf '  \033[32m✔ %s\033[0m\n' "$*"; }
bad()  { printf '  \033[31m✘ %s\033[0m\n' "$*"; }
skip() { SKIP=$((SKIP + 1)); printf '  \033[33m- %s\033[0m\n' "$*"; }

# check <name> <cmd...>: run a check, record and report the result.
check() {
  local name="$1"
  shift
  local log
  log="$(mktemp)"
  if "$@" >"$log" 2>&1; then
    PASS=$((PASS + 1))
    RESULTS+=("PASS|$name")
    ok "$name"
    if [[ "$VERBOSE" == "1" && -s "$log" ]]; then
      sed 's/^/       /' "$log"
    fi
  else
    FAIL=$((FAIL + 1))
    RESULTS+=("FAIL|$name")
    bad "$name"
    tail -40 "$log" | sed 's/^/       /' >&2
  fi
  rm -f "$log"
}

# run_in <dir> <cmd...>: run a command with a specific working directory.
run_in() {
  local dir="$1"
  shift
  (cd "$dir" && "$@")
}

has() { command -v "$1" >/dev/null 2>&1; }

# ─── Prerequisites ────────────────────────────────────────────────────────
[[ "$SKIP_MOBILE" == "1" ]] && note "mobile checks disabled (--skip-mobile)"
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
  check "backend: go build"          run_in "$ROOT/backend" go build ./...
  check "backend: go vet"            run_in "$ROOT/backend" go vet ./...
  check "backend: go mod tidy -diff" run_in "$ROOT/backend" go mod tidy -diff
  if install_go_tool "golang.org/x/tools/cmd/deadcode@latest" deadcode \
    && install_go_tool "honnef.co/go/tools/cmd/staticcheck@latest" staticcheck; then
    check "backend: deadcode -test ./..." run_in "$ROOT/backend" "$TOOL_BIN/deadcode" -test ./...
    check "backend: staticcheck ./..."    run_in "$ROOT/backend" "$TOOL_BIN/staticcheck" ./...
  else
    skip "backend: go tool install failed"
  fi
else
  skip "backend: checks skipped (go missing)"
fi

# ─── Web ──────────────────────────────────────────────────────────────────
log "web"
if has node; then
  check "web: tsc --noEmit" run_in "$ROOT/web" npx tsc --noEmit
  check "web: eslint"       run_in "$ROOT/web" npm run lint
  check "web: knip"         run_in "$ROOT/web" npx -y knip --reporter compact
  check "web: api typegen stable" bash -c '
    set -euo pipefail
    gen_dir="'"$ROOT"'/web/src/api/generated"
    backup="$(mktemp -d)"
    # Always restore the working tree, even if the generator crashes midway.
    trap "rm -rf \"$gen_dir\"; mkdir -p \"$gen_dir\"; cp -r \"$backup\"/. \"$gen_dir\"/; rm -rf \"$backup\"" EXIT
    cp -r "$gen_dir"/. "$backup"/
    ( cd "'"$ROOT"'/web" && node scripts/generate-api-types.mjs >/dev/null )
    diff -rq "$backup" "$gen_dir" >/dev/null
  '
else
  skip "web: checks skipped (node missing)"
fi

# ─── Desktop ──────────────────────────────────────────────────────────────
log "desktop"
if has node; then
  check "desktop: tsc --noEmit" run_in "$ROOT/desktop" npm run typecheck
else
  skip "desktop: checks skipped (node missing)"
fi

# ─── Mobile ───────────────────────────────────────────────────────────────
log "mobile"
if [[ "$SKIP_MOBILE" == "1" ]]; then
  skip "mobile: checks skipped (--skip-mobile)"
elif has java; then
  check "mobile: compileDebugKotlin" run_in "$ROOT/mobile" ./gradlew :app:compileDebugKotlin $GRADLE_OFFLINE
  check "mobile: lintDebug"          run_in "$ROOT/mobile" ./gradlew :app:lintDebug          $GRADLE_OFFLINE
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
printf '  \033[1m%d passed, %d failed, %d skipped\033[0m\n' "$PASS" "$FAIL" "$SKIP"
if [[ "$FAIL" == "0" ]]; then
  log "audit passed"
  exit 0
fi
log "audit failed"
exit 1
