# User Service — build & codegen
#
# Swagger docs (docs/swagger.json, docs/swagger.yaml) are GENERATED automatically
# from the endpoint annotations in the code on every `make build`. Never edit
# docs/ by hand — change the handlers and rebuild.

SWAG    ?= $(shell go env GOPATH)/bin/swag
GOOS    ?= linux
GOARCH  ?= $(shell go env GOHOSTARCH)
BINDIR  := bin
COMPOSE := docker compose -f $(CURDIR)/../docker-compose.yaml

.PHONY: generate build docker-build clean

# Regenerate Swagger specs from the code annotations (JSON + YAML only, no
# embedded docs.go).
generate:
	$(SWAG) init -g infra/doc.go -d . -o docs --parseInternal --ot json,yaml

# Build the Linux binary (used by the Docker image). Always regenerates the
# Swagger docs first so they never drift from the endpoints.
build: generate
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINDIR)/user-service-linux .

# Rebuild and (re)start the user-service container with Swagger enabled.
docker-build: build
	$(COMPOSE) up -d --build user-service

clean:
	rm -f $(BINDIR)/user-service-linux
