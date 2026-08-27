#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# common.sh — shared helpers for PudimProductivity shell scripts.
#
# Provides color constants (auto-disabled when stdout is not a TTY) and the
# log_* helpers used by run.sh, run-mobile.sh, ci-check.sh and
# deadcode-audit.sh.
#
# Usage:
#   ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
#   # shellcheck source=lib/common.sh
#   source "$ROOT_DIR/scripts/lib/common.sh"
#   PREFIX="dev"   # tag shown in brackets, e.g. "[dev]"
# ──────────────────────────────────────────────────────────────────────────────

# ─── Colors ────────────────────────────────────────────────────────────────
# Disable ANSI codes when stdout is not a terminal (e.g. CI logs, pipes).
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    CYAN=''
    BOLD=''
    NC=''
fi

# ─── Logging helpers ────────────────────────────────────────────────────────
# PREFIX is the tag printed in brackets. Set it after sourcing; the functions
# read it at call time, so a late assignment works fine.
PREFIX="${PUDIM_LOG_PREFIX:-dev}"

log_info()   { echo -e "${CYAN}[$PREFIX]${NC} $*"; }
log_ok()     { echo -e "${GREEN}[$PREFIX]${NC} $*"; }
log_warn()   { echo -e "${YELLOW}[$PREFIX]${NC} $*"; }
log_error()  { echo -e "${RED}[$PREFIX]${NC} $*"; }
