#!/bin/sh
# Runs the full test suite for every Go service. The repo has no go.work, so
# each service is a separate module and is tested in its own directory.
set -eu

cd /app

services="user-service content-service stream-service sender-service gateway"

for svc in $services; do
	echo
	echo "=== $svc ==="
	( cd "$svc" && go build ./... && go test -count=1 -race -cover ./... )
done