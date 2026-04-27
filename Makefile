# LocalFinance — single-binary build
# This Makefile is the only interface for building, testing, running locally.

DIST          = $(CURDIR)/dist
NODE_BIN      = $(shell /usr/local/bin/node --version >/dev/null 2>&1 && echo /usr/local/bin || echo $(shell dirname $(shell which node)))
NPM           = PATH=$(NODE_BIN):$$PATH npm

.PHONY: help
help:
	@echo "LocalFinance"
	@echo ""
	@echo "Run:"
	@echo "  make dev               Run the consolidated binary against host Postgres + Ollama"
	@echo ""
	@echo "Build:"
	@echo "  make build             Build dist/localfinance (embeds React UI)"
	@echo "  make webui-build       Build React only (writes internal/webui/dist)"
	@echo ""
	@echo "Test:"
	@echo "  make test              Go unit tests (./internal/... ./cmd/...)"
	@echo "  make test-e2e          Integration tests (requires make dev running)"
	@echo "  make eval-hallucination  Insight hallucination eval (live Ollama, EVAL_OLLAMA=1)"
	@echo ""
	@echo "Verify:"
	@echo "  make verify            curl localhost:3001/health"
	@echo ""
	@echo "macOS .app installer:"
	@echo "  make installer         Build dist/LocalFinance.app"
	@echo "  make installer-clean   Remove dist/LocalFinance.app"
	@echo "  make installer-run     Build + launch the .app"
	@echo "  make dmg               Build dist/LocalFinance-<version>.dmg"
	@echo ""
	@echo "Clean:"
	@echo "  make clean             Remove dist/"

# ─── Run ──────────────────────────────────────────────
.PHONY: dev
dev:
	@PORT=3001 go run ./cmd/localfinance

# ─── Build ────────────────────────────────────────────
.PHONY: build webui-build clean
webui-build:
	@if [ ! -d services/iris/client/node_modules ]; then \
		echo "Installing React deps (first run)..."; \
		cd services/iris/client && $(NPM) install --silent 2>&1 | tail -3 ; \
	fi
	@cd services/iris/client && CI=false $(NPM) run build 2>&1 | tail -3
	@rm -rf internal/webui/dist
	@cp -r services/iris/client/build internal/webui/dist
	@echo "  → internal/webui/dist/ (embed input)"

build: webui-build
	@go build -o $(DIST)/localfinance ./cmd/localfinance
	@echo "  → $(DIST)/localfinance ($$(du -h $(DIST)/localfinance | cut -f1))"

clean:
	@rm -rf $(DIST)
	@echo "Cleaned dist/"

# ─── Test ─────────────────────────────────────────────
.PHONY: test test-e2e eval-hallucination
test:
	@go test ./internal/... ./cmd/...

test-e2e:
	@bash tests/e2e/run-all.sh

eval-hallucination:
	@echo "Running insight hallucination eval (requires local Ollama + EVAL_OLLAMA=1)..."
	@EVAL_OLLAMA=1 go test -tags=eval -run HallucinationEval -v -count=1 -timeout 30m ./internal/ai/...

# ─── Verify ───────────────────────────────────────────
.PHONY: verify
verify:
	@code=$$(curl -sf -o /dev/null -w '%{http_code}' --connect-timeout 3 \
		http://localhost:3001/health 2>/dev/null || echo "000"); \
	if [ "$$code" = "200" ]; then \
		echo "localhost:3001/health — healthy"; \
	else \
		echo "localhost:3001/health — FAILED (HTTP $$code)"; exit 1; \
	fi

# ─── macOS Installer ──────────────────────────────────
.PHONY: installer installer-clean installer-run dmg
installer:
	@bash installer/build.sh

installer-clean:
	@rm -rf $(DIST)/LocalFinance.app

installer-run: installer
	@$(DIST)/LocalFinance.app/Contents/MacOS/LocalFinance

dmg:
	@bash installer/dmg.sh
