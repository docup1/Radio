# Content Service — build & codegen
#
# Swagger docs (docs/docs.go, docs/swagger.json, docs/swagger.yaml) are
# GENERATED automatically from the endpoint annotations in the code on every
# `make build`. Never edit docs/ by hand — change the handlers and rebuild.

SWAG    ?= $(shell go env GOPATH)/bin/swag
GOOS    ?= linux
GOARCH  ?= $(shell go env GOHOSTARCH)
BINDIR  := bin
COMPOSE := docker compose -f $(CURDIR)/../docker-compose.yaml

.PHONY: generate build docker-build clean

# Regenerate Swagger specs from the code annotations (JSON + YAML only, no
# embedded docs.go).
generate:
	$(SWAG) init -g internal/api/rest/doc.go -o docs --parseInternal --ot json,yaml

# Build the Linux binaries (used by the Docker images). Always regenerates
# the Swagger docs first so they never drift from the endpoints.
build: generate
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINDIR)/public ./cmd/public
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINDIR)/stream ./cmd/stream
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINDIR)/access  ./cmd/access

# Rebuild and (re)start the public + stream containers with Swagger enabled.
docker-build: build
	$(COMPOSE) up -d --build content-service-public content-service-stream

clean:
	rm -f $(BINDIR)/public $(BINDIR)/stream $(BINDIR)/access
