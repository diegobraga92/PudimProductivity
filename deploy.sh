#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="/home/diego/dev/OpenToDoApp"

echo "Building and deploying OpenToDoApp..."
echo ""

echo "1. Building backend..."
cd "$ROOT_DIR/backend"
cargo build --release
echo "   Backend built successfully"

echo ""
echo "2. Building web frontend (production mode)..."
cd "$ROOT_DIR/web"
npm run build
echo "   Web frontend built successfully"

echo ""
echo "3. Starting services..."
echo "   Backend will run on: http://192.168.3.12:3000"
echo "   Web frontend will be served from: $ROOT_DIR/web/dist"
echo ""
echo "   To start the backend in production mode:"
echo "   cd $ROOT_DIR/backend && ./target/release/opentodo"
echo ""
echo "   To serve the web frontend:"
echo "   cd $ROOT_DIR/web && npx serve dist"
echo ""
echo "Deployment preparation complete!"
# TODO
