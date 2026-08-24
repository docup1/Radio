SERVICE := user-service
SERVICE_DIR := $(SERVICE)
BIN := bin/user-service-linux

GOARCH := $(shell uname -m | sed -e 's/^x86_64$$/amd64/' -e 's/^aarch64$$/arm64/')
GOBUILD := CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o $(BIN)

PORT_FILE := .env
SERVICE_PORT := $(shell grep -E '^USER_SERVICE_PORT=' $(PORT_FILE) 2>/dev/null | cut -d= -f2 | tr -d '[:space:]')
SERVICE_PORT := $(if $(SERVICE_PORT),$(SERVICE_PORT),18080)

.PHONY: dev build down clean

dev: build
	@echo "freeing port $(SERVICE_PORT)"
	@lsof -ti tcp:$(SERVICE_PORT) -sTCP:LISTEN 2>/dev/null | while read pid; do \
		cmd=$$(ps -o command= -p $$pid 2>/dev/null); \
		case "$$cmd" in \
			*OrbStack*|*com.docker*|*Docker*) echo "skip $$pid (container port binding)";; \
			*) echo "killing $$pid ($$cmd)"; kill -9 $$pid 2>/dev/null || true;; \
		esac; \
	done
	docker compose up -d --build --force-recreate $(SERVICE)

build:
	@echo "building $(BIN) (linux/$(GOARCH))"
	cd $(SERVICE_DIR) && $(GOBUILD) .

down:
	docker compose down

clean:
	rm -rf $(SERVICE_DIR)/bin

# ---- content-service ----
CS_DIR := content-service
CS_PORTS := 18092

.PHONY: cs-build cs-dev cs-down
cs-build:
	echo "building content-service linux/$(GOARCH) binaries"
	cd $(CS_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/content-service ./cmd/content-service
cs-dev: cs-build
	echo "freeing content-service ports $(CS_PORTS)"
	for p in $(CS_PORTS); do lsof -ti tcp:$$p -sTCP:LISTEN 2>/dev/null | while read pid; do cmd=$$(ps -o command= -p $$pid 2>/dev/null); case "$$cmd" in *OrbStack*|*com.docker*|*Docker*) echo "skip $$pid (container port binding)";; *) echo "killing $$pid ($$cmd)"; kill -9 $$pid 2>/dev/null || true;; esac; done; done
	docker compose up -d --build --force-recreate content-service-migrations content-service
cs-down:
	docker compose stop content-service-migrations content-service
