#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/diego/dev/OpenToDoApp"

cleanup() {
  echo ""
  echo "Stopping services..."
  jobs -p | xargs -r kill
}

trap cleanup EXIT INT TERM

echo "=============================================="
echo "Starting OpenToDoApp Development Environment"
echo "=============================================="
echo ""

echo "1. Starting backend on http://localhost:3000 ..."
cargo run --manifest-path "$ROOT_DIR/backend/Cargo.toml" &

echo "2. Starting web frontend on http://localhost:5173 ..."
npm --prefix "$ROOT_DIR/web" run dev &

echo ""
echo "=============================================="
echo "Development services started!"
echo "=============================================="
echo ""
echo "Backend:      http://localhost:3000"
echo "Web Frontend: http://localhost:5173"
echo ""
echo "For Android development:"
echo "1. Edit android/gradle.properties and set:"
echo "   backend.url=http://localhost:3000"
echo "2. Rebuild Android app:"
echo "   cd android && ./gradlew assembleDebug"
echo ""
echo "Press Ctrl+C to stop all services."
echo "=============================================="

wait
# TODO
