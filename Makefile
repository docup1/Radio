SERVICE := user-service
SERVICE_DIR := $(SERVICE)
BIN := bin/user-service-linux

GOARCH := $(shell uname -m | sed -e 's/^x86_64$$/amd64/' -e 's/^aarch64$$/arm64/')
GOBUILD := CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o $(BIN)

PORT_FILE := .env
SERVICE_PORT := $(shell grep -E '^USER_SERVICE_PORT=' $(PORT_FILE) 2>/dev/null | cut -d= -f2 | tr -d '[:space:]')
SERVICE_PORT := $(if $(SERVICE_PORT),$(SERVICE_PORT),18080)

FRONTEND_DIR := frontend
GATEWAY_DIST := gateway/dist

.PHONY: dev prod user-dev build down clean \
         front-install front-dev front-build front-dist

# ---------- frontend ----------
front-install:
	cd $(FRONTEND_DIR) && npm install

front-build: front-install
	cd $(FRONTEND_DIR) && npm run build

# Copy the built SPA into the gateway image build context so `make prod` can bake it in.
front-dist: front-build
	rm -rf $(GATEWAY_DIST)
	cp -r $(FRONTEND_DIR)/dist $(GATEWAY_DIST)

front-dev: front-install
	cd $(FRONTEND_DIR) && npm run dev

# ---------- backend (user-service only) ----------
user-dev: build
	@echo "freeing port $(SERVICE_PORT)"
	@lsof -ti tcp:$(SERVICE_PORT) -sTCP:LISTEN 2>/dev/null | while read pid; do \
		cmd=$$(ps -o command= -p $$pid 2>/dev/null); \
		case "$$cmd" in \
			*OrbStack*|*com.docker*|*Docker*) echo "skip $$pid (container port binding)";; \
			*) echo "killing $$pid ($$cmd)"; kill -9 $$pid 2>/dev/null || true;; \
		esac; \
	done
	docker compose up -d --build --force-recreate $(SERVICE)

# ---------- full stack (dev) ----------
# Backend (docker compose) + frontend vite, both in the foreground with logs to the
# console. Ctrl-C stops everything. Uses the already-built images (no rebuild);
# rebuild backend separately when needed.
dev:
	sh $(CURDIR)/scripts/dev.sh

# ---------- full stack (prod) ----------
# Build the SPA, bake it into the gateway image (STATIC_DIR), then bring up everything.
prod: build cs-build front-dist
	docker compose build
	docker compose up -d

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
	cd $(CS_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/content-service ./cmd/content
cs-dev: cs-build
	echo "freeing content-service ports $(CS_PORTS)"
	for p in $(CS_PORTS); do lsof -ti tcp:$$p -sTCP:LISTEN 2>/dev/null | while read pid; do cmd=$$(ps -o command= -p $$pid 2>/dev/null); case "$$cmd" in *OrbStack*|*com.docker*|*Docker*) echo "skip $$pid (container port binding)";; *) echo "killing $$pid ($$cmd)"; kill -9 $$pid 2>/dev/null || true;; esac; done; done
	docker compose up -d --build --force-recreate content-service-migrations content-service
cs-down:
	docker compose stop content-service-migrations content-service
