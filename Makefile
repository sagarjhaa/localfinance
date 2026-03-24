# LocalFinance - Continuous Deployment Pipeline
# Cross-compile Go services on Mac, deploy to Jetson Nano Orin

JETSON_USER   ?= sagar
JETSON_HOST   ?= 10.0.0.16
JETSON_PASS   ?= jetson
JETSON_DIR    ?= /home/sagar/localfinance
JETSON        = sshpass -p $(JETSON_PASS) ssh -o StrictHostKeyChecking=no $(JETSON_USER)@$(JETSON_HOST)
JETSON_SUDO   = $(JETSON) 'echo $(JETSON_PASS) | sudo -S'
JSCP          = sshpass -p $(JETSON_PASS) scp -o StrictHostKeyChecking=no

GO_SERVICES   = hermes thesaurus sophia logos
GOOS          = linux
GOARCH        = arm64
CGO_ENABLED   = 0
DIST          = $(CURDIR)/dist
NODE_VERSION  = 18.20.4
NODE_TARBALL  = node-v$(NODE_VERSION)-linux-arm64.tar.xz
NODE_URL      = https://nodejs.org/dist/v$(NODE_VERSION)/$(NODE_TARBALL)
NODE_CACHE    = $(DIST)/node-arm64
# Use brew Node for builds (nvm may have an old version in PATH)
NODE_BIN      = $(shell /usr/local/bin/node --version >/dev/null 2>&1 && echo /usr/local/bin || echo $(shell dirname $(shell which node)))
NPM           = PATH=$(NODE_BIN):$$PATH npm

# ─── Build ──────────────────────────────────────────────

.PHONY: build-all $(addprefix build-,$(GO_SERVICES)) build-iris clean

build-all: $(addprefix build-,$(GO_SERVICES)) build-iris
	@echo "✅ All services built"

define BUILD_SERVICE
build-$(1):
	@echo "🔨 Building $(1) (linux/arm64)..."
	@cd services/$(1) && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -ldflags="-s -w" -o $(DIST)/$(1) .
	@echo "  → dist/$(1) ($$(du -h $(DIST)/$(1) | cut -f1))"
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call BUILD_SERVICE,$(svc))))

build-iris: fetch-node
	@echo "🔨 Building Iris (React + Node)..."
	@cd services/iris && $(NPM) install --silent 2>/dev/null
	@cd services/iris/client && $(NPM) install --silent 2>/dev/null && $(NPM) run build 2>&1 | tail -3
	@echo "  → services/iris/client/build/"

fetch-node:
	@if [ ! -f "$(NODE_CACHE)/bin/node" ]; then \
		echo "📦 Downloading Node.js $(NODE_VERSION) (linux/arm64)..."; \
		mkdir -p $(NODE_CACHE); \
		curl -sL $(NODE_URL) | tar -xJ -C $(NODE_CACHE) --strip-components=1; \
		echo "  → $(NODE_CACHE)/bin/node"; \
	fi

clean:
	@rm -rf $(DIST)/hermes $(DIST)/thesaurus $(DIST)/sophia $(DIST)/logos
	@echo "🧹 Cleaned dist/"

# ─── Deploy ─────────────────────────────────────────────

.PHONY: deploy-all $(addprefix deploy-,$(GO_SERVICES)) deploy-iris deploy-infra

deploy-all: $(addprefix deploy-,$(GO_SERVICES)) deploy-iris
	@echo "✅ All services deployed"

define DEPLOY_SERVICE
deploy-$(1): build-$(1)
	@echo "🚀 Deploying $(1) to Jetson..."
	@$(JSCP) $(DIST)/$(1) $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/bin/$(1)
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl restart localfinance-$(1)' 2>/dev/null || \
		echo "  ⚠️  systemd unit not installed yet — run 'make setup-jetson' first"
	@echo "  → deployed $(1)"
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call DEPLOY_SERVICE,$(svc))))

deploy-iris: build-iris
	@echo "🚀 Deploying Iris to Jetson..."
	@$(JETSON) 'mkdir -p $(JETSON_DIR)/iris/{server,client,node/bin}'
	@$(JSCP) $(NODE_CACHE)/bin/node $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/node/bin/node
	@$(JSCP) -r services/iris/server $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/
	@$(JSCP) -r services/iris/client/build $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/client/
	@$(JSCP) services/iris/package.json $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/
	@echo "  → syncing npm deps..."
	@cd services/iris && $(NPM) install --production --silent 2>/dev/null
	@$(JSCP) -r services/iris/node_modules $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl restart localfinance-iris' 2>/dev/null || \
		echo "  ⚠️  systemd unit not installed yet — run 'make setup-jetson' first"
	@echo "  → deployed iris"

deploy-infra:
	@echo "🐳 Deploying infrastructure (Docker Compose) to Jetson..."
	@$(JSCP) deployment/docker/docker-compose.yml $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/docker-compose.yml
	@$(JETSON) 'cd $(JETSON_DIR) && docker compose up -d'
	@echo "  → infrastructure running"

# ─── Jetson Setup (first-time) ──────────────────────────

.PHONY: setup-jetson

setup-jetson:
	@echo "🔧 Setting up Jetson..."
	@$(JETSON) 'mkdir -p $(JETSON_DIR)/{bin,logs,data,uploads,iris}'
	@echo "  → directories created"
	@$(JSCP) deployment/jetson.env $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/.env 2>/dev/null || \
		$(JSCP) deployment/jetson.env.template $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/.env
	@echo "  → .env deployed"
	@for unit in deployment/systemd/*.service; do \
		$(JSCP) $$unit $(JETSON_USER)@$(JETSON_HOST):/tmp/; \
		$(JETSON) "echo $(JETSON_PASS) | sudo -S cp /tmp/$$(basename $$unit) /etc/systemd/system/"; \
	done
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl daemon-reload'
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S systemctl enable localfinance-hermes localfinance-thesaurus localfinance-sophia localfinance-logos localfinance-iris'
	@echo "  → systemd units installed and enabled"
	@echo "✅ Jetson setup complete. Run 'make deploy-all' to deploy services."

# ─── Status & Monitoring ────────────────────────────────

.PHONY: status logs ssh

status:
	@echo "📡 Checking Jetson services..."
	@for svc in hermes:3000 thesaurus:8001 sophia:8002 logos:8003 iris:3001; do \
		name=$${svc%%:*}; port=$${svc##*:}; \
		resp=$$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 2 http://$(JETSON_HOST):$$port/health 2>/dev/null); \
		if [ "$$resp" = "200" ]; then \
			echo "  ✅ $$name (port $$port) — healthy"; \
		else \
			echo "  ❌ $$name (port $$port) — down (HTTP $$resp)"; \
		fi; \
	done

logs:
	@echo "Usage: make logs-<service>"
	@echo "  e.g. make logs-hermes"

define LOGS_SERVICE
logs-$(1):
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S journalctl -u localfinance-$(1) --no-pager -n 50'
endef
$(foreach svc,$(GO_SERVICES) iris,$(eval $(call LOGS_SERVICE,$(svc))))

ssh:
	@sshpass -p $(JETSON_PASS) ssh -o StrictHostKeyChecking=no $(JETSON_USER)@$(JETSON_HOST)

# ─── Help ───────────────────────────────────────────────

.PHONY: help
help:
	@echo "LocalFinance Deployment"
	@echo ""
	@echo "Build:"
	@echo "  make build-all          Build all services for ARM64"
	@echo "  make build-<service>    Build a single service (hermes|thesaurus|sophia|logos|iris)"
	@echo "  make clean              Remove build artifacts"
	@echo ""
	@echo "Deploy:"
	@echo "  make deploy-all         Build + deploy all services to Jetson"
	@echo "  make deploy-<service>   Build + deploy a single service"
	@echo "  make deploy-infra       Deploy Docker Compose infra to Jetson"
	@echo ""
	@echo "Setup:"
	@echo "  make setup-jetson       First-time Jetson setup (dirs, systemd, .env)"
	@echo ""
	@echo "Monitor:"
	@echo "  make status             Check health of all services"
	@echo "  make logs-<service>     Tail logs for a service"
	@echo "  make ssh                SSH into Jetson"
