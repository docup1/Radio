#!/bin/sh
# Dev orchestration for the full stack:
#   - frontend build → gateway/dist (static assets for gateway)
#   - backend via docker compose (rebuild, logs to console)
#   - frontend via vite (HMR, logs to console)
# Both run in the foreground of this script; Ctrl-C (SIGINT) stops everything
# (vite killed, containers brought down).
set -e

cd "$(dirname "$0")/.."

# Build frontend and copy to gateway/dist so the gateway Docker image has static assets
echo "building frontend..."
(cd frontend && npm install && npm run build)
rm -rf gateway/dist
cp -r frontend/dist gateway/dist
echo "frontend copied to gateway/dist"

# Build gateway binary
echo "building gateway..."
make gw-build

docker compose up --build & COMPOSE_PID=$!

(cd frontend && npm run dev) & VITE_PID=$!

cleanup() {
	kill "$COMPOSE_PID" "$VITE_PID" 2>/dev/null
	pkill -f vite 2>/dev/null
	docker compose down 2>/dev/null
}
trap cleanup EXIT INT TERM

wait
