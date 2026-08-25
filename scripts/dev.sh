#!/bin/sh
# Dev orchestration for the full stack:
#   - backend via docker compose (rebuild, logs to console)
#   - frontend via vite (HMR, logs to console)
# Both run in the foreground of this script; Ctrl-C (SIGINT) stops everything
# (vite killed, containers brought down).
set -e

cd "$(dirname "$0")/.."

mkdir -p gateway/dist

docker compose up --build & COMPOSE_PID=$!

(cd frontend && npm install && npm run dev) & VITE_PID=$!

cleanup() {
	kill "$COMPOSE_PID" "$VITE_PID" 2>/dev/null
	pkill -f vite 2>/dev/null
	docker compose down 2>/dev/null
}
trap cleanup EXIT INT TERM

wait
