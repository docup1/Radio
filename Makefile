GOARCH := $(shell uname -m | sed -e 's/^x86_64$$/amd64/' -e 's/^aarch64$$/arm64/')

FRONTEND_DIR := frontend
GATEWAY_DIST := gateway/dist

.PHONY: dev prod build down clean \
         front-install front-dev front-build front-dist \
         user-dev cs-build cs-dev cs-down \
         ss-build ss-dev snd-build snd-dev \
         gw-build

# ---------- frontend ----------
front-install:
	cd $(FRONTEND_DIR) && npm install

front-build: front-install
	cd $(FRONTEND_DIR) && npm run build

front-dist: front-build
	rm -rf $(GATEWAY_DIST)
	cp -r $(FRONTEND_DIR)/dist $(GATEWAY_DIST)

front-dev: front-install
	cd $(FRONTEND_DIR) && npm run dev

# ---------- user-service ----------
build:
	@echo "building user-service linux/$(GOARCH)"
	cd user-service && CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/user-service-linux .

user-dev: build
	@echo "freeing user-service ports"
	@lsof -ti tcp:18080 -sTCP:LISTEN 2>/dev/null | while read pid; do \
		cmd=$$(ps -o command= -p $$pid 2>/dev/null); \
		case "$$cmd" in \
			*OrbStack*|*com.docker*|*Docker*) echo "skip $$pid (container port binding)";; \
			*) echo "killing $$pid ($$cmd)"; kill -9 $$pid 2>/dev/null || true;; \
		esac; \
	done
	docker compose up -d --build --force-recreate user-service

# ---------- content-service ----------
cs-build:
	@echo "building content-service linux/$(GOARCH)"
	cd content-service && CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/content-service ./cmd/content

cs-dev: cs-build
	@echo "freeing content-service ports"
	@lsof -ti tcp:18092 -sTCP:LISTEN 2>/dev/null | while read pid; do \
		cmd=$$(ps -o command= -p $$pid 2>/dev/null); \
		case "$$cmd" in \
			*OrbStack*|*com.docker*|*Docker*) echo "skip $$pid (container port binding)";; \
			*) echo "killing $$pid ($$cmd)"; kill -9 $$pid 2>/dev/null || true;; \
		esac; \
	done
	docker compose up -d --build --force-recreate content-service-migrations content-service

cs-down:
	docker compose stop content-service-migrations content-service

# ---------- stream-service ----------
ss-build:
	@echo "building stream-service linux/$(GOARCH)"
	cd stream-service && CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/stream ./cmd/stream

ss-dev: ss-build
	docker compose up -d --build --force-recreate stream-service

# ---------- sender-service ----------
snd-build:
	@echo "building sender-service linux/$(GOARCH)"
	cd sender-service && CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/sender ./cmd/sender

snd-dev: snd-build
	docker compose up -d --build --force-recreate sender-service

# ---------- gateway ----------
gw-build:
	@echo "building gateway linux/$(GOARCH)"
	cd gateway && go install github.com/swaggo/swag/cmd/swag@v1.16.4 && \
		swag init -g api/router.go -o docs --parseInternal --ot json,yaml
	cd gateway && CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -ldflags="-s -w" -o bin/gateway .

# ---------- full stack ----------
dev:
	sh $(CURDIR)/scripts/dev.sh

prod: build cs-build ss-build snd-build gw-build front-dist
	docker compose build
	docker compose up -d

down:
	docker compose down

clean:
	rm -rf user-service/bin content-service/bin stream-service/bin sender-service/bin gateway/bin
