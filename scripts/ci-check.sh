#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# ci-check.sh - Run all PudimProductivity CI checks locally.
#
# Usage: ./scripts/ci-check.sh [OPTIONS]
# Run `./scripts/ci-check.sh --help` for the full option list.
# ──────────────────────────────────────────────────────────────────────────────
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
WEB_DIR="$ROOT_DIR/web"
BACKEND_DIR="$ROOT_DIR/backend"
MOBILE_DIR="$ROOT_DIR/mobile"
API_DIR="$ROOT_DIR/api"
SCRIPTS_DIR="$ROOT_DIR/scripts"

# ─── Android SDK (used by the mobile checks) ─────────────────────────────────
export ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"

# ─── Colors & logging (shared helpers) ─────────────────────────────────────
# shellcheck source=lib/common.sh
source "$ROOT_DIR/scripts/lib/common.sh"
PREFIX="ci"
log_section() { echo ""; echo -e "${BOLD}━━━ $1 ━━━${NC}"; }

PASS=0
FAIL=0
SKIP=0

# ─── Result tracking ─────────────────────────────────────────────────────────
pass() {
    log_ok "✓ $1"
    PASS=$((PASS + 1))
}

fail() {
    log_error "✗ $1"
    FAIL=$((FAIL + 1))
}

skip() {
    log_warn "− $1 (skipped)"
    SKIP=$((SKIP + 1))
    return 0
}

summary() {
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "  ${BOLD}CI Summary${NC}"
    echo ""
    echo -e "  ${GREEN}Passed:${NC}  $PASS"
    echo -e "  ${RED}Failed:${NC}  $FAIL"
    echo -e "  ${YELLOW}Skipped:${NC} $SKIP"

    if [ "$FAIL" -gt 0 ]; then
        echo ""
        log_error "Some checks failed!"
        exit 1
    fi

    log_ok "All checks passed!"
    exit 0
}

# ─── Help ──────────────────────────────────────────────────────────────────
usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Run all PudimProductivity CI checks locally.

Options:
  --skip-mobile        Skip Android / Gradle checks
  --skip-integration   Skip Docker-dependent integration tests (go test)
  --help               Show this help message and exit
EOF
}

# ─── Parse arguments ───────────────────────────────────────────────────────
SKIP_MOBILE=false
SKIP_INTEGRATION=false

for arg in "$@"; do
    case "$arg" in
        --skip-mobile)      SKIP_MOBILE=true ;;
        --skip-integration) SKIP_INTEGRATION=true ;;
        --help)             usage; exit 0 ;;
        *) log_warn "Unknown argument: $arg"; usage >&2; exit 2 ;;
    esac
done

# ═══════════════════════════════════════════════════════════════════════════
# Preflight: required toolchains
# ═══════════════════════════════════════════════════════════════════════════
# Missing tools surface as failures in the relevant checks; reporting them
# here up front makes the cause obvious instead of a cryptic "not found".
for tool in go gofmt node npm docker; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        log_warn "Required tool not found: $tool — related checks will fail"
    fi
done

# ═══════════════════════════════════════════════════════════════════════════
# Section 1: Go Checks
# ═══════════════════════════════════════════════════════════════════════════
log_section "Go Checks"

# ── 1a. go fmt ────────────────────────────────────────────────────────────
run_go_fmt() {
    if ! command -v gofmt >/dev/null 2>&1; then
        log_error "gofmt not found — is the Go toolchain installed?"
        return 1
    fi
    log_info "Checking Go formatting (gofmt)..."
    pushd "$BACKEND_DIR" > /dev/null || return 1

    UNFORMATTED=$(gofmt -l .)
    GOFMT_RC=$?
    popd > /dev/null || return 1

    if [ "$GOFMT_RC" -ne 0 ]; then
        log_error "gofmt failed with exit code $GOFMT_RC"
        return 1
    fi
    if [ -n "$UNFORMATTED" ]; then
        log_error "Unformatted files:"
        echo "$UNFORMATTED"
        return 1
    fi
    return 0
}

if run_go_fmt; then
    pass "go fmt"
else
    fail "go fmt"
fi

# ── 1b. go vet ────────────────────────────────────────────────────────────
log_info "Running go vet..."
if (cd "$BACKEND_DIR" && go vet ./...) 2>&1; then
    pass "go vet"
else
    fail "go vet"
fi

# ── 1c. go build ──────────────────────────────────────────────────────────
log_info "Running go build..."
if (cd "$BACKEND_DIR" && go build ./...) 2>&1; then
    pass "go build"
else
    fail "go build"
fi

# ── 1d. go test (unit only, or full integration) ──────────────────────────
if [ "$SKIP_INTEGRATION" = true ]; then
    log_info "Running go test (unit only, skipping integration)..."
    # The -short flag is a Go convention: tests using testcontainers
    # should call t.SkipIfShort(). We pass -short here so integration
    # tests can opt out (the test suite doesn't use it yet, but the
    # pattern is established).
    if (cd "$BACKEND_DIR" && go test -short ./...) 2>&1; then
        pass "go test (unit)"
    else
        fail "go test (unit)"
    fi
else
    log_info "Running go test (including integration — requires Docker)..."
    if (cd "$BACKEND_DIR" && go test ./...) 2>&1; then
        pass "go test (full)"
    else
        fail "go test (full)"
    fi
fi

# ── 1e. go mod tidy check ─────────────────────────────────────────────────
log_info "Checking go mod tidy..."
# `-diff` prints the changes tidy would make without touching go.mod/go.sum,
# and exits non-zero if any are needed — no copy/restore dance required.
TIDY_LOG=$(mktemp)
if (cd "$BACKEND_DIR" && go mod tidy -diff) >"$TIDY_LOG" 2>&1; then
    pass "go mod tidy (clean)"
else
    log_error "go.mod or go.sum needs updating — run 'go mod tidy'"
    tail -20 "$TIDY_LOG"
    fail "go mod tidy"
fi
rm -f "$TIDY_LOG"

# ── 1f. govulncheck (optional) ────────────────────────────────────────────
log_info "Checking for govulncheck..."
if command -v govulncheck &> /dev/null; then
    if (cd "$BACKEND_DIR" && govulncheck ./...) 2>&1; then
        pass "govulncheck"
    else
        fail "govulncheck"
    fi
else
    skip "govulncheck (not installed — install with 'go install golang.org/x/vuln/cmd/govulncheck@latest')"
fi

# ═══════════════════════════════════════════════════════════════════════════
# Section 2: Web / Frontend Checks
# ═══════════════════════════════════════════════════════════════════════════
log_section "Web / Frontend Checks"

# ── 2a. npm ci ────────────────────────────────────────────────────────────
log_info "Installing web dependencies (npm ci)..."
if (cd "$WEB_DIR" && npm ci) 2>&1; then
    pass "npm ci"
else
    fail "npm ci"
fi

# ── 2b. ESLint ────────────────────────────────────────────────────────────
log_info "Running ESLint..."
if (cd "$WEB_DIR" && npm run lint) 2>&1; then
    pass "ESLint"
else
    fail "ESLint"
fi

# ── 2c. TypeScript typecheck ──────────────────────────────────────────────
log_info "Running TypeScript typecheck..."
if (cd "$WEB_DIR" && npm run typecheck) 2>&1; then
    pass "TypeScript typecheck"
else
    fail "TypeScript typecheck"
fi

# ── 2d. Vitest unit tests ─────────────────────────────────────────────────
log_info "Running Vitest tests..."
if (cd "$WEB_DIR" && npm run test) 2>&1; then
    pass "Vitest tests"
else
    fail "Vitest tests"
fi

# ── 2e. Vite build ────────────────────────────────────────────────────────
log_info "Running Vite production build..."
if (cd "$WEB_DIR" && npm run build) 2>&1; then
    pass "Vite build"
else
    fail "Vite build"
fi

# ── 2f. npm audit (optional, runtime deps only) ────────────────────────────
log_info "Running npm audit (runtime dependencies)..."
# Capture the real exit code: npm audit exits non-zero when vulnerabilities
# are found. They are reported but treated as non-fatal (skipped) here so a
# manual review can happen.
AUDIT_OUTPUT=$(cd "$WEB_DIR" && npm audit --omit=dev 2>&1)
AUDIT_EXIT=$?
if echo "$AUDIT_OUTPUT" | grep -q "found 0 vulnerabilities"; then
    pass "npm audit"
elif [ -z "$AUDIT_OUTPUT" ]; then
    pass "npm audit (no output)"
else
    log_warn "npm audit reported issues (exit $AUDIT_EXIT):"
    echo "$AUDIT_OUTPUT" | head -20
    skip "npm audit (vulnerabilities found — review manually)"
fi

# ═══════════════════════════════════════════════════════════════════════════
# Section 3: Mobile / Android Checks
# ═══════════════════════════════════════════════════════════════════════════
log_section "Mobile / Android Checks"

if [ "$SKIP_MOBILE" = true ]; then
    skip "Mobile checks (--skip-mobile)"
else
    # Auto-detect Android SDK
    if [ ! -d "$ANDROID_HOME" ]; then
        skip "Mobile checks (Android SDK not found at $ANDROID_HOME)"
    elif [ ! -f "$MOBILE_DIR/gradlew" ]; then
        skip "Mobile checks (gradlew not found at $MOBILE_DIR)"
    else
        # ── 3a. Gradle assembleDebug ──────────────────────────────────────
        log_info "Running Gradle assembleDebug..."
        if (cd "$MOBILE_DIR" && ANDROID_HOME="$ANDROID_HOME" ./gradlew assembleDebug) 2>&1; then
            pass "Gradle assembleDebug"
        else
            fail "Gradle assembleDebug"
        fi

        # ── 3b. Gradle lint (Android lint) ────────────────────────────────
        log_info "Running Android lint..."
        if (cd "$MOBILE_DIR" && ANDROID_HOME="$ANDROID_HOME" ./gradlew lint) 2>&1; then
            pass "Android lint"
        else
            fail "Android lint"
        fi
    fi
fi

# ═══════════════════════════════════════════════════════════════════════════
# Section 4: Infra & Optional Checks
# ═══════════════════════════════════════════════════════════════════════════
log_section "Infrastructure & Quality Checks"

# ── 4a. ShellCheck (optional) ─────────────────────────────────────────────
log_info "Checking for shellcheck..."
if command -v shellcheck &> /dev/null; then
    log_info "Running shellcheck on scripts/*.sh..."
    # `find -exec ... +` handles filenames with spaces, which an unquoted
    # expansion of a plain file list would break on.
    if [ -n "$(find "$SCRIPTS_DIR" -name '*.sh' -type f 2>/dev/null || true)" ]; then
        if find "$SCRIPTS_DIR" -name '*.sh' -type f -exec shellcheck {} +; then
            pass "ShellCheck"
        else
            fail "ShellCheck"
        fi
    else
        skip "ShellCheck (no .sh files in scripts/)"
    fi
else
    skip "ShellCheck (not installed — install with 'apt install shellcheck' or 'brew install shellcheck')"
fi

# ── 4b. hadolint (optional) ───────────────────────────────────────────────
log_info "Checking for hadolint..."
if command -v hadolint &> /dev/null; then
    log_info "Linting Dockerfiles..."
    HADOLINT_FAILED=false

    # `find` instead of a `**` glob: `**` only recurses with `shopt -s globstar`
    # (off by default), so the glob would silently miss nested Dockerfiles.
    while IFS= read -r -d '' dockerfile; do
        if ! hadolint "$dockerfile" 2>&1; then
            HADOLINT_FAILED=true
        fi
    done < <(find "$ROOT_DIR" -name Dockerfile -type f -print0)

    if [ "$HADOLINT_FAILED" = false ]; then
        pass "hadolint"
    else
        fail "hadolint"
    fi
else
    skip "hadolint (not installed — install with 'apt install hadolint' or 'brew install hadolint')"
fi

# ── 4c. OpenAPI spec validation (optional) ────────────────────────────────
log_info "Checking for OpenAPI validator..."
# Try redocly first, then swagger-cli as fallback
VALIDATOR=""
if command -v redocly &> /dev/null; then
    VALIDATOR="redocly"
elif npx --yes @redocly/cli --version &> /dev/null; then
    VALIDATOR="npx --yes @redocly/cli"
elif command -v swagger-cli &> /dev/null; then
    VALIDATOR="swagger-cli"
fi

if [ -n "$VALIDATOR" ]; then
    log_info "Validating OpenAPI specs with ${VALIDATOR}..."
    OAS_FAILED=false
    OAS_FILES=$(find "$API_DIR" -name '*.yaml' -type f 2>/dev/null || true)

    if [ -z "$OAS_FILES" ]; then
        skip "OpenAPI validation (no .yaml files in api/)"
    else
        for spec in $OAS_FILES; do
            spec_name=$(basename "$spec")

            case "$VALIDATOR" in
                redocly)
                    if redocly lint "$spec" 2>&1; then
                        log_ok "  ✓ $spec_name"
                    else
                        log_warn "  ✗ $spec_name"
                        OAS_FAILED=true
                    fi
                    ;;
                "npx --yes @redocly/cli")
                    if npx --yes @redocly/cli lint "$spec" 2>&1; then
                        log_ok "  ✓ $spec_name"
                    else
                        log_warn "  ✗ $spec_name"
                        OAS_FAILED=true
                    fi
                    ;;
                swagger-cli)
                    if swagger-cli validate "$spec" 2>&1; then
                        log_ok "  ✓ $spec_name"
                    else
                        log_warn "  ✗ $spec_name"
                        OAS_FAILED=true
                    fi
                    ;;
            esac
        done

        if [ "$OAS_FAILED" = false ]; then
            pass "OpenAPI validation"
        else
            fail "OpenAPI validation"
        fi
    fi
else
    skip "OpenAPI validation (no validator found — install with 'npm install -g @redocly/cli' or 'npm install -g swagger-cli')"
fi

# ── 4d. Docker compose config validation ──────────────────────────────────
log_info "Validating docker-compose.yml..."
if command -v docker &> /dev/null; then
    if docker compose -f "$ROOT_DIR/docker-compose.yml" config -q 2>&1; then
        pass "docker-compose config"
    else
        fail "docker-compose config"
    fi
else
    skip "docker compose config (Docker not available)"
fi

# ═══════════════════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════════════════
summary