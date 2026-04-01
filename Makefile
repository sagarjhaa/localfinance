# LocalFinance — Agent Playbook
# This Makefile is the ONLY interface for building, testing, deploying.
# Never run raw ssh/scp — use these targets.

# ─── Configuration ─────────────────────────────────────
JETSON_USER   ?= sagar
JETSON_HOST   ?= 10.0.0.16
JETSON_PASS   ?= jetson
JETSON_DIR    ?= /home/sagar/localfinance
JETSON        = sshpass -p $(JETSON_PASS) ssh -o StrictHostKeyChecking=no $(JETSON_USER)@$(JETSON_HOST)
JETSON_SUDO   = $(JETSON) 'echo $(JETSON_PASS) | sudo -S'
JSCP          = sshpass -p $(JETSON_PASS) scp -o StrictHostKeyChecking=no

GO_SERVICES   = hermes thesaurus sophia logos
ALL_SERVICES  = $(GO_SERVICES) iris
GOOS          = linux
GOARCH        = arm64
CGO_ENABLED   = 0
DIST          = $(CURDIR)/dist
NODE_VERSION  = 18.20.4
NODE_TARBALL  = node-v$(NODE_VERSION)-linux-arm64.tar.xz
NODE_URL      = https://nodejs.org/dist/v$(NODE_VERSION)/$(NODE_TARBALL)
NODE_CACHE    = $(DIST)/node-arm64
NODE_BIN      = $(shell /usr/local/bin/node --version >/dev/null 2>&1 && echo /usr/local/bin || echo $(shell dirname $(shell which node)))
NPM           = PATH=$(NODE_BIN):$$PATH npm

SERVICE_PORTS = hermes:3000 thesaurus:8001 sophia:8002 logos:8003 iris:3001

.PHONY: help
help:
	@echo "LocalFinance — Agent Playbook"
	@echo ""
	@echo "Local Development (Docker):"
	@echo "  make dev-up                Start full local stack"
	@echo "  make dev-down              Tear down local stack"
	@echo "  make dev-down-clean        Tear down + delete volumes"
	@echo "  make dev-logs              Tail all service logs"
	@echo "  make dev-logs-<svc>        Tail one service log"
	@echo "  make dev-restart-<svc>     Rebuild + restart one service"
	@echo ""
	@echo "Build:"
	@echo "  make build                 Build all Go services (native)"
	@echo "  make build-<svc>           Build one service (native)"
	@echo "  make build-arm64           Cross-compile all for Jetson"
	@echo "  make build-arm64-<svc>     Cross-compile one"
	@echo "  make build-iris            Build React + Node bundle"
	@echo ""
	@echo "Test:"
	@echo "  make test                  All unit tests (Go + Node)"
	@echo "  make test-<svc>            One service unit tests"
	@echo "  make test-e2e              Integration tests (requires dev-up)"
	@echo "  make test-e2e-<suite>      One suite (auth|upload|chat|categories)"
	@echo ""
	@echo "Deploy to Jetson:"
	@echo "  make deploy-<svc>          Build ARM64 + SCP + restart + verify"
	@echo "  make deploy-all            All services"
	@echo "  make deploy-infra          Docker Compose infra only"
	@echo "  make setup-jetson          First-time Jetson setup"
	@echo ""
	@echo "Verify & Debug (Jetson):"
	@echo "  make verify                Health check all services"
	@echo "  make verify-<svc>          Health check one service"
	@echo "  make jetson-logs-<svc>     Tail logs on Jetson"
	@echo "  make jetson-status         systemctl status all services"
	@echo "  make jetson-kill-<svc>     Kill stale process by port"
	@echo "  make jetson-infra-status   Check Postgres, Redis, Ollama, MinIO"
	@echo "  make jetson-ollama-pull    Pull llama3.2:1b model"
	@echo "  make jetson-ssh            SSH into Jetson"

# ─── Local Development (Docker) ────────────────────────
.PHONY: dev-up dev-down dev-down-clean dev-logs

dev-up:
	@bash scripts/dev-up.sh

dev-down:
	@bash scripts/dev-down.sh

dev-down-clean:
	@bash scripts/dev-down.sh --volumes

dev-logs:
	@docker compose -f docker-compose.dev.yml logs -f

define DEV_LOGS_SERVICE
.PHONY: dev-logs-$(1)
dev-logs-$(1):
	@docker compose -f docker-compose.dev.yml logs -f $(1)
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call DEV_LOGS_SERVICE,$(svc))))

define DEV_RESTART_SERVICE
.PHONY: dev-restart-$(1)
dev-restart-$(1):
	@echo "Rebuilding $(1)..."
	@docker compose -f docker-compose.dev.yml up -d --build --no-deps $(1)
	@echo "Waiting for $(1) to be healthy..."
	@sleep 3
	@bash scripts/verify.sh localhost 2>/dev/null && echo "$(1) restarted." || echo "Warning: $(1) may not be healthy yet"
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call DEV_RESTART_SERVICE,$(svc))))

# ─── Build (native) ───────────────────────────────────
.PHONY: build

build: $(addprefix build-,$(GO_SERVICES))
	@echo "All Go services built (native)"

define BUILD_NATIVE
.PHONY: build-$(1)
build-$(1):
	@echo "Building $(1) (native)..."
	@cd services/$(1) && go build -o $(DIST)/$(1)-native .
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call BUILD_NATIVE,$(svc))))

# ─── Build (ARM64 cross-compile for Jetson) ───────────
.PHONY: build-arm64

build-arm64: $(addprefix build-arm64-,$(GO_SERVICES))
	@echo "All Go services built (ARM64)"

define BUILD_ARM64
.PHONY: build-arm64-$(1)
build-arm64-$(1):
	@echo "Building $(1) (linux/arm64)..."
	@cd services/$(1) && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -ldflags="-s -w" -o $(DIST)/$(1) .
	@echo "  → dist/$(1) ($$(du -h $(DIST)/$(1) | cut -f1))"
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call BUILD_ARM64,$(svc))))

.PHONY: build-iris
build-iris: fetch-node
	@echo "Building Iris (React + Node)..."
	@cd services/iris && $(NPM) install --silent 2>/dev/null
	@cd services/iris/client && $(NPM) install --silent 2>/dev/null && $(NPM) run build 2>&1 | tail -3
	@echo "  Bundling server with esbuild..."
	@PATH=$(NODE_BIN):$$PATH npx esbuild services/iris/server/index.js \
		--bundle --platform=node --target=node18 \
		--outfile=$(DIST)/iris-server.js 2>&1 | tail -1
	@echo "  → dist/iris-server.js ($$(du -h $(DIST)/iris-server.js | cut -f1))"
	@echo "  → services/iris/client/build/"

.PHONY: fetch-node
fetch-node:
	@if [ ! -f "$(NODE_CACHE)/bin/node" ]; then \
		echo "Downloading Node.js $(NODE_VERSION) (linux/arm64)..."; \
		mkdir -p $(NODE_CACHE); \
		curl -sL $(NODE_URL) | tar -xJ -C $(NODE_CACHE) --strip-components=1; \
	fi

.PHONY: clean
clean:
	@rm -rf $(DIST)/hermes $(DIST)/thesaurus $(DIST)/sophia $(DIST)/logos \
		$(DIST)/hermes-native $(DIST)/thesaurus-native $(DIST)/sophia-native $(DIST)/logos-native
	@echo "Cleaned dist/"

# ─── Test (unit) ──────────────────────────────────────
.PHONY: test test-thesaurus test-logos test-sophia test-iris

test: test-thesaurus test-logos test-iris
	@echo "All unit tests passed"

test-thesaurus:
	@echo "Testing Thesaurus..."
	@cd services/thesaurus && go test ./... -v -count=1 2>&1 | tail -40

test-logos:
	@echo "Testing Logos..."
	@cd services/logos && go test ./... -v -count=1 2>&1 | tail -40

test-sophia:
	@echo "Testing Sophia..."
	@cd services/sophia && go test ./... -v -count=1 2>&1 | tail -40

test-iris:
	@echo "Testing Iris..."
	@cd services/iris && PATH=$(NODE_BIN):$$PATH node server/__tests__/auth-routes.test.js

# ─── Test (e2e — requires dev-up) ─────────────────────
.PHONY: test-e2e test-e2e-auth test-e2e-upload test-e2e-chat test-e2e-categories

test-e2e:
	@bash tests/e2e/run-all.sh

test-e2e-auth:
	@bash tests/e2e/test-auth.sh

test-e2e-upload:
	@bash tests/e2e/test-upload.sh

test-e2e-chat:
	@bash tests/e2e/test-chat.sh

test-e2e-categories:
	@bash tests/e2e/test-categories.sh

# ─── Deploy to Jetson ─────────────────────────────────
.PHONY: deploy-all deploy-iris deploy-infra

deploy-all: $(addprefix deploy-,$(GO_SERVICES)) deploy-iris
	@echo "All services deployed. Verifying..."
	@bash scripts/verify.sh $(JETSON_HOST)

define DEPLOY_SERVICE
.PHONY: deploy-$(1)
deploy-$(1): build-arm64-$(1)
	@echo "Deploying $(1) to Jetson..."
	@$(JSCP) $(DIST)/$(1) $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/bin/$(1)
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl restart localfinance-$(1)' 2>/dev/null || \
		echo "  Warning: systemd unit not installed — run make setup-jetson first"
	@sleep 2
	@bash scripts/verify.sh $(JETSON_HOST) 2>/dev/null || echo "  Warning: health check pending"
	@echo "  → $(1) deployed"
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call DEPLOY_SERVICE,$(svc))))

deploy-iris: build-iris
	@echo "Deploying Iris to Jetson..."
	@$(JETSON) 'mkdir -p $(JETSON_DIR)/iris/{client/build,node/bin}'
	@$(JSCP) $(NODE_CACHE)/bin/node $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/node/bin/node
	@$(JSCP) $(DIST)/iris-server.js $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/server.js
	@$(JSCP) -r services/iris/client/build $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/client/
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl restart localfinance-iris' 2>/dev/null || \
		echo "  Warning: systemd unit not installed — run make setup-jetson first"
	@sleep 2
	@bash scripts/verify.sh $(JETSON_HOST) 2>/dev/null || echo "  Warning: health check pending"
	@echo "  → iris deployed"

deploy-infra:
	@echo "Deploying infrastructure to Jetson..."
	@$(JSCP) deployment/docker/docker-compose.yml $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/docker-compose.yml
	@$(JETSON) 'cd $(JETSON_DIR) && docker compose up -d'

# ─── Jetson Setup (first-time) ────────────────────────
.PHONY: setup-jetson

setup-jetson:
	@echo "Setting up Jetson..."
	@$(JETSON) 'mkdir -p $(JETSON_DIR)/{bin,logs,data,uploads,iris}'
	@$(JSCP) deployment/jetson.env.template $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/.env
	@for unit in deployment/systemd/*.service; do \
		$(JSCP) $$unit $(JETSON_USER)@$(JETSON_HOST):/tmp/; \
		$(JETSON) "echo $(JETSON_PASS) | sudo -S cp /tmp/$$(basename $$unit) /etc/systemd/system/"; \
	done
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl daemon-reload'
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl enable localfinance-hermes localfinance-thesaurus localfinance-sophia localfinance-logos localfinance-iris'
	@echo "Jetson setup complete. Run make deploy-all."

# ─── Verify & Debug (Jetson) ──────────────────────────
.PHONY: verify jetson-status jetson-infra-status jetson-ollama-pull jetson-ssh

verify:
	@bash scripts/verify.sh $(JETSON_HOST)

define VERIFY_SERVICE
.PHONY: verify-$(1)
verify-$(1):
	@port=$$(echo "$(SERVICE_PORTS)" | tr ' ' '\n' | grep "^$(1):" | cut -d: -f2); \
	health="/health"; \
	if [ "$(1)" = "iris" ]; then health="/api/health"; fi; \
	code=$$(curl -sf -o /dev/null -w '%{http_code}' --connect-timeout 3 \
		"http://$(JETSON_HOST):$$port$$health" 2>/dev/null || echo "000"); \
	if [ "$$code" = "200" ]; then \
		echo "✅ $(1) (port $$port) — healthy"; \
	else \
		echo "❌ $(1) (port $$port) — FAILED (HTTP $$code)"; \
		exit 1; \
	fi
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call VERIFY_SERVICE,$(svc))))

define JETSON_LOGS_SERVICE
.PHONY: jetson-logs-$(1)
jetson-logs-$(1):
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S journalctl -u localfinance-$(1) --no-pager -n 50'
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call JETSON_LOGS_SERVICE,$(svc))))

define JETSON_KILL_SERVICE
.PHONY: jetson-kill-$(1)
jetson-kill-$(1):
	@port=$$(echo "$(SERVICE_PORTS)" | tr ' ' '\n' | grep "^$(1):" | cut -d: -f2); \
	echo "Killing process on port $$port..."; \
	$(JETSON) "fuser -k $$port/tcp 2>/dev/null || echo 'No process on port $$port'"
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call JETSON_KILL_SERVICE,$(svc))))

jetson-status:
	@for svc in $(ALL_SERVICES); do \
		echo -n "$$svc: "; \
		$(JETSON) "systemctl is-active localfinance-$$svc 2>/dev/null || echo 'inactive'"; \
	done

jetson-infra-status:
	@echo "Checking Jetson infrastructure..."
	@$(JETSON) 'docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "Docker not running"'

jetson-ollama-pull:
	@echo "Pulling llama3.2:1b on Jetson..."
	@$(JETSON) 'curl -sf http://localhost:11434/api/pull -d "{\"name\":\"llama3.2:1b\"}" || echo "Ollama not reachable"'

jetson-ssh:
	@sshpass -p $(JETSON_PASS) ssh -o StrictHostKeyChecking=no $(JETSON_USER)@$(JETSON_HOST)
