# LocalFinance — Agent Playbook
# This Makefile is the ONLY interface for building, testing, running locally.

# ─── Configuration ─────────────────────────────────────
GO_SERVICES   = hermes thesaurus sophia logos
ALL_SERVICES  = $(GO_SERVICES) iris
DIST          = $(CURDIR)/dist
NODE_VERSION  = 18.20.4
NODE_BIN      = $(shell /usr/local/bin/node --version >/dev/null 2>&1 && echo /usr/local/bin || echo $(shell dirname $(shell which node)))
NPM           = PATH=$(NODE_BIN):$$PATH npm

SERVICE_PORTS = hermes:3000 thesaurus:8001 sophia:8002 logos:8003 iris:3001

# Default model used by Sophia. Override per command: make dev-up CHAT_MODEL=qwen2.5:7b
CHAT_MODEL    ?= llama3.1:8b

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
	@echo "  make build-iris            Build React + Node bundle"
	@echo ""
	@echo "Test:"
	@echo "  make test                  All unit tests (Go + Node)"
	@echo "  make test-<svc>            One service unit tests"
	@echo "  make test-e2e              Integration tests (requires dev-up)"
	@echo "  make test-e2e-<suite>      One suite (auth|upload|chat|categories)"
	@echo ""
	@echo "Eval:"
	@echo "  make eval-hallucination    Run Phase 0 hallucination eval (live Ollama)"
	@echo ""
	@echo "Verify:"
	@echo "  make verify                Health check all services on localhost"
	@echo "  make verify-<svc>          Health check one service"

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

# ─── Build (native macOS) ──────────────────────────────
.PHONY: build clean

build: $(addprefix build-,$(GO_SERVICES))
	@echo "All Go services built"

define BUILD_NATIVE
.PHONY: build-$(1)
build-$(1):
	@echo "Building $(1)..."
	@cd services/$(1) && go build -o $(DIST)/$(1) .
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call BUILD_NATIVE,$(svc))))

.PHONY: build-iris
build-iris:
	@echo "Building Iris (React + Node)..."
	@cd services/iris && $(NPM) install --silent 2>/dev/null
	@cd services/iris/client && $(NPM) install --silent 2>/dev/null && $(NPM) run build 2>&1 | tail -3
	@PATH=$(NODE_BIN):$$PATH npx esbuild services/iris/server/index.js \
		--bundle --platform=node --target=node18 \
		--outfile=$(DIST)/iris-server.js 2>&1 | tail -1
	@echo "  → dist/iris-server.js ($$(du -h $(DIST)/iris-server.js | cut -f1))"

clean:
	@rm -rf $(DIST)/hermes $(DIST)/thesaurus $(DIST)/sophia $(DIST)/logos $(DIST)/iris-server.js
	@echo "Cleaned dist/"

# ─── Test (unit) ──────────────────────────────────────
.PHONY: test test-thesaurus test-logos test-sophia test-iris

test: test-thesaurus test-logos test-sophia test-iris
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
	@cd services/iris && PATH=$(NODE_BIN):$$PATH node server/__tests__/insights-routes.test.js

# ─── Eval (manual, gated by EVAL_OLLAMA=1) ────────────
.PHONY: eval-hallucination
eval-hallucination:
	@echo "Running insight hallucination eval (requires local Ollama + EVAL_OLLAMA=1)..."
	@cd services/sophia && EVAL_OLLAMA=1 go test -tags=eval -run HallucinationEval -v -count=1 -timeout 30m ./ai/...

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

# ─── Verify (localhost health checks) ─────────────────
.PHONY: verify

verify:
	@bash scripts/verify.sh localhost

define VERIFY_SERVICE
.PHONY: verify-$(1)
verify-$(1):
	@port=$$(echo "$(SERVICE_PORTS)" | tr ' ' '\n' | grep "^$(1):" | cut -d: -f2); \
	health="/health"; \
	if [ "$(1)" = "iris" ]; then health="/api/health"; fi; \
	code=$$(curl -sf -o /dev/null -w '%{http_code}' --connect-timeout 3 \
		"http://localhost:$$port$$health" 2>/dev/null || echo "000"); \
	if [ "$$code" = "200" ]; then \
		echo "✅ $(1) (port $$port) — healthy"; \
	else \
		echo "❌ $(1) (port $$port) — FAILED (HTTP $$code)"; \
		exit 1; \
	fi
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call VERIFY_SERVICE,$(svc))))
