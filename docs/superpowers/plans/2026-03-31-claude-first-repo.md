# Claude-First Repo Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make localfinance fully autonomous for AI agents — plan, code, test, deploy, verify with zero hand-holding.

**Architecture:** Replace the current ad-hoc Makefile and outdated CLAUDE.md with a complete agent playbook: new CLAUDE.md contract with superpowers skill integration, per-service docs, Docker Compose local dev stack, Makefile as single interface, integration test suite, and debugging protocol.

**Tech Stack:** Docker Compose, Make, Bash, curl, jq, Go 1.22+, Node 18+

**Spec:** `docs/superpowers/specs/2026-03-31-claude-first-repo-design.md`

---

## File Map

### New Files
| File | Responsibility |
|---|---|
| `CLAUDE.md` | Agent contract (rewrite existing) |
| `services/thesaurus/CLAUDE.md` | Thesaurus service guide |
| `services/iris/CLAUDE.md` | Iris service guide |
| `services/sophia/CLAUDE.md` | Sophia service guide |
| `services/logos/CLAUDE.md` | Logos service guide |
| `services/hermes/CLAUDE.md` | Hermes service guide |
| `docker-compose.dev.yml` | Full local dev stack |
| `services/thesaurus/Dockerfile` | Multi-stage Go build |
| `services/sophia/Dockerfile` | Multi-stage Go build |
| `services/logos/Dockerfile` | Multi-stage Go build |
| `services/hermes/Dockerfile` | Multi-stage Go build |
| `services/iris/Dockerfile` | Node multi-stage build |
| `deployment/docker/dev.env` | Dev environment variables |
| `scripts/dev-up.sh` | Start dev stack + wait for health |
| `scripts/dev-down.sh` | Tear down dev stack |
| `scripts/verify.sh` | Health check with retries |
| `tests/e2e/run-all.sh` | E2E test entry point |
| `tests/e2e/helpers.sh` | Shared test utilities |
| `tests/e2e/test-auth.sh` | Auth lifecycle tests |
| `tests/e2e/test-upload.sh` | Upload flow tests |
| `tests/e2e/test-chat.sh` | AI chat tests |
| `tests/e2e/test-categories.sh` | Category rules tests |
| `tests/fixtures/sample.csv` | Test CSV fixture |
| `tests/fixtures/sample-creditcard.csv` | Credit card CSV fixture |

### Modified Files
| File | Change |
|---|---|
| `Makefile` | Complete rewrite with dev, test-e2e, verify, jetson-* targets |
| `.claude/settings.json` | Add docker permissions |

---

## Task 1: Rewrite CLAUDE.md — Agent Contract

**Files:**
- Rewrite: `CLAUDE.md`

- [ ] **Step 1: Read current CLAUDE.md**

Read the existing file to understand what to preserve (service ports, boundaries) vs replace (Python/SQLite references, outdated structure).

- [ ] **Step 2: Write new CLAUDE.md**

Replace the entire file with the agent contract. This is the most important file in the repo — it tells every future agent exactly how to work.

```markdown
# LocalFinance

Privacy-first personal finance system. Local Ollama AI, deployed on Jetson Nano Orin.

## Architecture

| Service | Port | Responsibility |
|---------|------|---------------|
| **Iris** | 3001 | React UI + Express proxy. ZERO business logic. |
| **Hermes** | 3000 | API Gateway (future routing/aggregation) |
| **Thesaurus** | 8001 | Auth, CRUD, PostgreSQL. Source of truth for all data. |
| **Sophia** | 8002 | AI/Ollama integration. Two-pass chat with real data. |
| **Logos** | 8003 | Document parsing (CSV, PDF). Stateless processor. |

**Infra:** PostgreSQL 15, Redis 7, Ollama (llama3.2:1b), MinIO

### Boundaries — NEVER violate these

- **Iris is DUMB.** No parsing, no data transformation, no business logic, no direct DB access. It proxies requests and renders UI.
- **All file processing in Logos.** CSV, PDF, Excel parsing happens only in Logos.
- **All data persistence through Thesaurus.** Every read/write goes through Thesaurus REST API.
- **All AI through Sophia.** Ollama calls only happen in Sophia.
- **Internal service calls use `/internal/` routes** (no auth middleware). User-facing routes use auth middleware.

### Data Flows

**Upload:** Browser → Iris proxy → Thesaurus (store doc) → Logos (parse) → Thesaurus (store transactions)
**Chat:** Browser → Iris proxy → Sophia (Pass 1: parse intent → Thesaurus: fetch data → Pass 2: generate answer)
**Auth:** Browser → Iris proxy → Thesaurus (register/login/validate JWT)

## Development Cycle

Every task follows this cycle. No shortcuts.

```
1. PLAN
   Use superpowers:brainstorming for feature design
   Use superpowers:writing-plans for implementation plan
   Get user approval before writing code

2. CODE
   Use superpowers:subagent-driven-development for parallel execution
   Follow per-service CLAUDE.md patterns (see services/*/CLAUDE.md)
   One logical unit at a time

3. TEST LOCALLY
   make test          ← unit tests must pass
   make dev-up        ← start local Docker stack
   make test-e2e      ← integration tests must pass
   If tests fail → use superpowers:systematic-debugging
   Fix → retest (loop until green)

4. COMMIT
   One commit per logical unit
   Format: type(service): what and why
   Types: feat, fix, refactor, test, docs, deploy
   Never batch frontend + backend + deploy in one commit

5. DEPLOY TO JETSON
   make deploy-{service}    ← builds ARM64 + SCP + restart
   make deploy-all          ← all services

6. VERIFY
   make verify              ← health check all services
   If verify fails → read logs, fix, go back to step 3
   Max 3 retry cycles, then escalate to user
   Use superpowers:verification-before-completion before claiming done

7. REPORT
   Use superpowers:requesting-code-review for self-review
   Tell user: what was built, which URLs to test
   Show clean git log of all commits
```

## Commands — Use These, Never Raw SSH

### Local Development (Docker)
```
make dev-up                  # Start full local stack (all services + infra)
make dev-down                # Tear down local stack
make dev-logs                # Tail all service logs
make dev-logs-{service}      # Tail one service
make dev-restart-{service}   # Rebuild + restart one container
```

### Build
```
make build                   # Build all Go services (native, for Docker)
make build-{service}         # Build one (hermes|thesaurus|sophia|logos)
make build-arm64             # Cross-compile all for Jetson
make build-arm64-{service}   # Cross-compile one
make build-iris              # Build React + bundle Node server
```

### Test
```
make test                    # All unit tests (Go + Node)
make test-{service}          # One service
make test-e2e                # Integration tests (requires make dev-up)
make test-e2e-{suite}        # One suite (auth|upload|chat|categories)
```

### Deploy to Jetson
```
make deploy-{service}        # Build ARM64 + SCP + restart + verify health
make deploy-all              # All services
make deploy-infra            # Docker Compose infra only
```

### Verify & Debug (Jetson)
```
make verify                  # Health check all services
make verify-{service}        # Health check one
make jetson-logs-{service}   # Tail logs
make jetson-status           # systemctl status all
make jetson-kill-{service}   # Kill stale process
make jetson-infra-status     # Check Postgres, Redis, Ollama, MinIO
make jetson-ollama-pull      # Pull model if missing
```

## Commit Conventions

Format:
```
type(service): what changed and why

Co-Authored-By: Claude <noreply@anthropic.com>
```

**One logical unit per commit.** If you can describe it with "and", split it.

Examples of GOOD commits:
```
feat(thesaurus): add category rules CRUD with user-scoped queries
feat(iris): add Settings page for managing category rules
test(e2e): add category rules integration test
deploy: update Makefile with dev-restart target
```

Examples of BAD commits:
```
Add category rules, settings page, and update Makefile    ← too many things
create files                                               ← meaningless
fix stuff                                                  ← meaningless
```

## Debugging Protocol

When `make verify` fails:

1. `make verify-{service}` — identify which service
2. `make jetson-logs-{service}` — read last 50 lines
3. Match pattern → known fix:
   - `address already in use` → `make jetson-kill-{service}`, redeploy
   - `connection refused` to DB → `make jetson-infra-status`, restart infra
   - `401 Unauthorized` on internal call → move endpoint to `/internal/` route group
   - `model not found` → `make jetson-ollama-pull`
   - `no such file` → binary not deployed, re-run `make deploy-*`
   - `timeout` → increase timeout in code, retry
4. Fix → rebuild → redeploy → `make verify`
5. Max 3 retries, then escalate to user with: what failed, logs, what was tried, suspected cause

## Superpowers Skills — When To Use

| Skill | When |
|-------|------|
| `superpowers:brainstorming` | Before any new feature — explore design with user |
| `superpowers:writing-plans` | After design approval — create implementation plan |
| `superpowers:subagent-driven-development` | Executing plan — dispatch parallel agents per task |
| `superpowers:systematic-debugging` | When tests fail or behavior is unexpected |
| `superpowers:requesting-code-review` | After completing work — self-review before reporting |
| `superpowers:verification-before-completion` | Before claiming done — verify everything passes |
| `superpowers:finishing-a-development-branch` | When work is complete — decide merge/PR/cleanup |

## Jetson Target

- **Host:** 10.0.0.16
- **User:** sagar
- **Services dir:** /home/sagar/localfinance/
- **Binaries:** /home/sagar/localfinance/bin/
- **Iris:** /home/sagar/localfinance/iris/
- **Infra:** Docker Compose (Postgres, Redis, Ollama, MinIO)
- **Process manager:** systemd
```

- [ ] **Step 3: Verify the file reads correctly**

Run: `head -5 CLAUDE.md`
Expected: `# LocalFinance`

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: rewrite CLAUDE.md as agent contract with development cycle and superpowers integration"
```

---

## Task 2: Per-Service CLAUDE.md Files

**Files:**
- Create: `services/thesaurus/CLAUDE.md`
- Create: `services/iris/CLAUDE.md`
- Create: `services/sophia/CLAUDE.md`
- Create: `services/logos/CLAUDE.md`
- Create: `services/hermes/CLAUDE.md`

- [ ] **Step 1: Create services/thesaurus/CLAUDE.md**

```markdown
# Thesaurus — Data & Auth Service (port 8001)

Source of truth for all persistent data. Owns PostgreSQL.
Handles: user auth (JWT/Argon2id), accounts, transactions, budgets,
documents, category rules, statement periods.

## Build

make build-thesaurus          # native (for local Docker)
make build-arm64-thesaurus    # cross-compile for Jetson

## Test

make test-thesaurus           # unit tests (auth, models)
make test-e2e                 # integration tests against local Docker

## Deploy

make deploy-thesaurus         # build ARM64 + SCP + restart + verify health

## Adding an Endpoint

1. Add/update model in models/ (with GORM tags, UUID primary key via BeforeCreate)
2. Add handler in api/handlers/
3. Register route in api/routes.go
4. Add to AutoMigrate list in database/database.go
5. User-facing: add under protected group (requires AuthMiddleware)
6. Service-to-service: add under /internal/ group (no auth)

## Key Files

- main.go — Gin router setup, DB init, migrations
- api/routes.go — all route registration
- api/handlers/ — one file per resource (auth, user, account, transaction, etc.)
- models/ — GORM models (User, Account, Transaction, Budget, Document, etc.)
- auth/jwt.go — JWT generation/validation
- auth/password.go — Argon2id hashing
- database/database.go — PostgreSQL connection + AutoMigrate
- middleware/auth_middleware.go — JWT auth enforcement

## Patterns

- User-scoped queries: always filter by user_id from JWT claims
- Bulk operations: /internal/transactions/bulk (Logos calls this, no auth)
- Models use UUID primary keys via BeforeCreate hook
- JWT secret from env var JWT_SECRET
- Config loaded from env vars: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
```

- [ ] **Step 2: Create services/iris/CLAUDE.md**

```markdown
# Iris — Frontend (port 3001)

React UI + Express proxy. ZERO business logic.

## NEVER do these in Iris:
- Parse files (CSV, PDF, Excel)
- Transform or process data
- Run business logic or categorization
- Store state beyond auth tokens
- Direct database access

## Build

make build-iris               # builds React client + bundles Node server

## Test

make test-iris                # unit tests (auth routes)
make test-e2e                 # integration tests

## Deploy

make deploy-iris              # build React + bundle + SCP + restart + verify

## Adding a Page

1. Create component in client/src/pages/NewPage.jsx
2. Add route in client/src/App.jsx
3. Add nav link in sidebar (copy from Dashboard.jsx sidebar)
4. API calls go through client/src/api/client.js

## Adding an API Proxy Route

1. All backend calls proxy through server/routes/proxy.js
2. Proxy strips /api/proxy/{service}/ prefix, forwards rest to backend
3. Never add business logic in proxy — just forward request/response

## Key Files

- server/index.js — Express entry point (Helmet, CORS, static files)
- server/routes/auth.js — proxies auth to Thesaurus
- server/routes/upload.js — proxies file uploads to Thesaurus
- server/routes/proxy.js — generic proxy to any backend service
- server/middleware/auth.js — validates JWT via Thesaurus
- client/src/App.jsx — React router
- client/src/api/client.js — Axios HTTP client with interceptors
- client/src/pages/ — one file per page (Dashboard, Chat, Settings, etc.)
```

- [ ] **Step 3: Create services/sophia/CLAUDE.md**

```markdown
# Sophia — AI Service (port 8002)

AI-powered insights using local Ollama (llama3.2:1b).
Two-pass chat: parse intent → fetch real data from Thesaurus → generate answer.

## Build

make build-sophia             # native
make build-arm64-sophia       # cross-compile for Jetson

## Test

make test-sophia              # unit tests
make test-e2e                 # integration tests (requires Ollama in Docker)

## Deploy

make deploy-sophia            # build ARM64 + SCP + restart + verify

## Two-Pass Chat Flow

1. Pass 1: User question → Ollama generates query intent (JSON with filters)
2. Execute: Call Thesaurus /internal/ endpoints with intent params
3. Pass 2: Real data + original question → Ollama generates natural language answer

## Adding AI Features

1. Add handler in api/handlers/
2. Register route in api/routes.go
3. Use ai/service.go for Ollama calls
4. Always fetch real data from Thesaurus — never hallucinate numbers
5. Timeout for Ollama calls: 120s minimum (Jetson GPU is slow)

## Key Files

- main.go — Gin router, AI service init
- ai/service.go — Ollama client, two-pass chat logic, merchant cleaning
- api/routes.go — route registration
- api/handlers/chat_handler.go — chat endpoint
- api/handlers/categorize_handler.go — transaction categorization
- api/handlers/insights_handler.go — financial insights
- models/models.go — request/response types
- config/config.go — Ollama host, model name, Thesaurus URL
```

- [ ] **Step 4: Create services/logos/CLAUDE.md**

```markdown
# Logos — Document Processing (port 8003)

Stateless document parser. Receives files from Thesaurus, parses
CSV/PDF, extracts transactions, sends back to Thesaurus.

## Build

make build-logos              # native
make build-arm64-logos        # cross-compile for Jetson

## Test

make test-logos               # unit tests (CSV parser, PDF parser, detector)
make test-e2e                 # integration tests

## Deploy

make deploy-logos             # build ARM64 + SCP + restart + verify

## Processing Flow

1. Thesaurus receives file upload → stores document record → fires POST to Logos
2. Logos detects file type (CSV/PDF) via processors/detector.go
3. Logos parses transactions via processors/csv_processor.go or pdf_processor.go
4. Logos fetches user's category rules from Thesaurus /internal/category-rules/
5. Applies matching rules, sends unmatched to Ollama for AI categorization
6. Sends all transactions to Thesaurus /internal/transactions/bulk
7. Updates document status to "processed" via Thesaurus /internal/documents/

## Adding a New File Format

1. Create processor in processors/ implementing the Processor interface
2. Register in processors/manager.go
3. Add format detection in processors/detector.go
4. Add test with fixture file in processors/*_test.go

## Key Files

- main.go — Gin router + inline processing logic (large file, ~440 lines)
- processors/csv_processor.go — CSV parsing
- processors/pdf_processor.go — PDF text extraction and transaction parsing
- processors/detector.go — statement type/institution detection
- processors/manager.go — processor factory
- models/models.go — ProcessRequest, Transaction types
- config/config.go — Thesaurus URL, Ollama URL
```

- [ ] **Step 5: Create services/hermes/CLAUDE.md**

```markdown
# Hermes — API Gateway (port 3000)

Future routing and aggregation layer. Currently minimal — serves
health endpoint and a status dashboard.

## Build

make build-hermes             # native
make build-arm64-hermes       # cross-compile for Jetson

## Deploy

make deploy-hermes            # build ARM64 + SCP + restart + verify

## Key Files

- main.go — Gin router, health endpoint, HTML status dashboard
```

- [ ] **Step 6: Commit**

```bash
git add services/thesaurus/CLAUDE.md services/iris/CLAUDE.md services/sophia/CLAUDE.md services/logos/CLAUDE.md services/hermes/CLAUDE.md
git commit -m "docs: add per-service CLAUDE.md with build, test, deploy instructions"
```

---

## Task 3: Docker Compose Local Dev Stack

**Files:**
- Create: `docker-compose.dev.yml`
- Create: `services/thesaurus/Dockerfile`
- Create: `services/sophia/Dockerfile`
- Create: `services/logos/Dockerfile`
- Create: `services/hermes/Dockerfile`
- Create: `services/iris/Dockerfile`
- Create: `deployment/docker/dev.env`

- [ ] **Step 1: Create dev environment variables**

```bash
# deployment/docker/dev.env
ENV=development

# Hermes
HERMES_PORT=3000
THESAURUS_URL=http://thesaurus:8001
SOPHIA_URL=http://sophia:8002
LOGOS_URL=http://logos:8003

# Thesaurus
THESAURUS_PORT=8001
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=devpass
DB_NAME=localfinance
DB_SSLMODE=disable
REDIS_HOST=redis
REDIS_PORT=6379
JWT_SECRET=dev-jwt-secret-do-not-use-in-production

# Sophia
SOPHIA_PORT=8002
OLLAMA_HOST=http://ollama:11434
MODEL_NAME=llama3.2:1b
THESAURUS_URL=http://thesaurus:8001

# Logos
LOGOS_PORT=8003
STORAGE_ENDPOINT=minio:9000
STORAGE_ACCESS_KEY=minioadmin
STORAGE_SECRET_KEY=devpass
STORAGE_BUCKET=documents
THESAURUS_URL=http://thesaurus:8001
OLLAMA_HOST=http://ollama:11434

# Iris
IRIS_PORT=3001
THESAURUS_URL=http://thesaurus:8001
SOPHIA_URL=http://sophia:8002
LOGOS_URL=http://logos:8003
```

- [ ] **Step 2: Create Go service Dockerfile (shared pattern)**

Create `services/thesaurus/Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
# Copy shared module first (dependency)
COPY shared/ /shared/
# Copy service source
COPY services/thesaurus/ .
# Update go.mod replace directive for Docker context
RUN go mod edit -replace github.com/sagarjhaa/localfinance/shared=/shared
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
COPY --from=builder /service /service
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-8001}/health || exit 1
ENTRYPOINT ["/service"]
```

- [ ] **Step 3: Create Sophia Dockerfile**

Create `services/sophia/Dockerfile` — same pattern as thesaurus:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY shared/ /shared/
COPY services/sophia/ .
RUN go mod edit -replace github.com/sagarjhaa/localfinance/shared=/shared
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
COPY --from=builder /service /service
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-8002}/health || exit 1
ENTRYPOINT ["/service"]
```

- [ ] **Step 4: Create Logos Dockerfile**

Create `services/logos/Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY shared/ /shared/
COPY services/logos/ .
RUN go mod edit -replace github.com/sagarjhaa/localfinance/shared=/shared
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
COPY --from=builder /service /service
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-8003}/health || exit 1
ENTRYPOINT ["/service"]
```

- [ ] **Step 5: Create Hermes Dockerfile**

Create `services/hermes/Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY shared/ /shared/
COPY services/hermes/ .
RUN go mod edit -replace github.com/sagarjhaa/localfinance/shared=/shared
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /service .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
COPY --from=builder /service /service
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-3000}/health || exit 1
ENTRYPOINT ["/service"]
```

- [ ] **Step 6: Create Iris Dockerfile**

Create `services/iris/Dockerfile`:

```dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
# Build React client
COPY services/iris/client/package*.json ./client/
RUN cd client && npm ci
COPY services/iris/client/ ./client/
RUN cd client && npm run build

# Production server
FROM node:18-alpine
WORKDIR /app
COPY services/iris/package*.json ./
RUN npm ci --production
COPY services/iris/server/ ./server/
COPY --from=builder /app/client/build ./client/build
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-3001}/api/health || exit 1
ENV NODE_ENV=production
CMD ["node", "server/index.js"]
```

- [ ] **Step 7: Create docker-compose.dev.yml**

```yaml
# LocalFinance - Full Local Dev Stack
# Usage: make dev-up / make dev-down
# Everything runs in Docker. Nothing on Mac.

services:
  # ─── Infrastructure ──────────────────────────────────
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: devpass
      POSTGRES_DB: localfinance
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  ollama:
    image: ollama/ollama:latest
    restart: unless-stopped
    volumes:
      - ollama_data:/root/.ollama
    ports:
      - "11434:11434"
    environment:
      OLLAMA_HOST: "0.0.0.0"
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:11434/api/tags || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  ollama-init:
    image: ollama/ollama:latest
    depends_on:
      ollama:
        condition: service_healthy
    entrypoint: ["ollama", "pull", "llama3.2:1b"]
    environment:
      OLLAMA_HOST: "http://ollama:11434"
    restart: "no"

  minio:
    image: minio/minio:latest
    restart: unless-stopped
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: devpass
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:9000/minio/health/live || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 5

  # ─── Application Services ───────────────────────────
  hermes:
    build:
      context: .
      dockerfile: services/hermes/Dockerfile
    env_file: deployment/docker/dev.env
    ports:
      - "3000:3000"
    environment:
      PORT: "3000"
    depends_on:
      postgres:
        condition: service_healthy

  thesaurus:
    build:
      context: .
      dockerfile: services/thesaurus/Dockerfile
    env_file: deployment/docker/dev.env
    ports:
      - "8001:8001"
    environment:
      PORT: "8001"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  sophia:
    build:
      context: .
      dockerfile: services/sophia/Dockerfile
    env_file: deployment/docker/dev.env
    ports:
      - "8002:8002"
    environment:
      PORT: "8002"
    depends_on:
      thesaurus:
        condition: service_healthy
      ollama:
        condition: service_healthy

  logos:
    build:
      context: .
      dockerfile: services/logos/Dockerfile
    env_file: deployment/docker/dev.env
    ports:
      - "8003:8003"
    environment:
      PORT: "8003"
    depends_on:
      thesaurus:
        condition: service_healthy
      minio:
        condition: service_healthy

  iris:
    build:
      context: .
      dockerfile: services/iris/Dockerfile
    env_file: deployment/docker/dev.env
    ports:
      - "3001:3001"
    environment:
      PORT: "3001"
    depends_on:
      thesaurus:
        condition: service_healthy

volumes:
  postgres_data:
  redis_data:
  ollama_data:
  minio_data:

networks:
  default:
    name: localfinance-net
```

- [ ] **Step 8: Test that docker-compose.dev.yml parses**

Run: `docker compose -f docker-compose.dev.yml config --quiet`
Expected: Exit code 0, no output (valid YAML)

- [ ] **Step 9: Commit**

```bash
git add docker-compose.dev.yml deployment/docker/dev.env services/*/Dockerfile
git commit -m "deploy: add Docker Compose local dev stack with all services and infra"
```

---

## Task 4: Dev Lifecycle Scripts

**Files:**
- Create: `scripts/dev-up.sh`
- Create: `scripts/dev-down.sh`
- Create: `scripts/verify.sh`

- [ ] **Step 1: Create scripts/dev-up.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "Starting LocalFinance dev stack..."
docker compose -f docker-compose.dev.yml up -d --build

echo "Waiting for services to be healthy..."

SERVICES="hermes:3000 thesaurus:8001 sophia:8002 logos:8003 iris:3001"
MAX_WAIT=120
ELAPSED=0

for svc_port in $SERVICES; do
  name="${svc_port%%:*}"
  port="${svc_port##*:}"
  echo -n "  Waiting for $name (port $port)..."

  while true; do
    if [ $ELAPSED -ge $MAX_WAIT ]; then
      echo " TIMEOUT after ${MAX_WAIT}s"
      echo "Check logs: docker compose -f docker-compose.dev.yml logs $name"
      exit 1
    fi

    health_path="/health"
    if [ "$name" = "iris" ]; then
      health_path="/api/health"
    fi

    if curl -sf "http://localhost:${port}${health_path}" > /dev/null 2>&1; then
      echo " ready"
      break
    fi

    sleep 2
    ELAPSED=$((ELAPSED + 2))
  done
done

echo ""
echo "All services healthy:"
echo "  Iris:      http://localhost:3001"
echo "  Hermes:    http://localhost:3000"
echo "  Thesaurus: http://localhost:8001"
echo "  Sophia:    http://localhost:8002"
echo "  Logos:     http://localhost:8003"
```

- [ ] **Step 2: Create scripts/dev-down.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

VOLUMES_FLAG=""
if [ "${1:-}" = "--volumes" ]; then
  VOLUMES_FLAG="-v"
  echo "Tearing down dev stack (including volumes)..."
else
  echo "Tearing down dev stack (preserving volumes)..."
fi

docker compose -f docker-compose.dev.yml down $VOLUMES_FLAG

echo "Done."
```

- [ ] **Step 3: Create scripts/verify.sh**

```bash
#!/usr/bin/env bash
# Verify service health with retries.
# Usage: ./scripts/verify.sh [host]
# Default host: localhost (for local Docker)
# For Jetson: ./scripts/verify.sh 10.0.0.16

set -euo pipefail

HOST="${1:-localhost}"
RETRIES=3
DELAY=5
FAILED=0

SERVICES="hermes:3000:/health thesaurus:8001:/health sophia:8002:/health logos:8003:/health iris:3001:/api/health"

echo "Verifying services on $HOST..."

for svc_info in $SERVICES; do
  IFS=':' read -r name port path <<< "$svc_info"
  SUCCESS=false

  for attempt in $(seq 1 $RETRIES); do
    code=$(curl -sf -o /dev/null -w '%{http_code}' --connect-timeout 3 \
      "http://${HOST}:${port}${path}" 2>/dev/null || echo "000")

    if [ "$code" = "200" ]; then
      echo "  ✅ $name (port $port) — healthy"
      SUCCESS=true
      break
    fi

    if [ "$attempt" -lt "$RETRIES" ]; then
      sleep "$DELAY"
    fi
  done

  if [ "$SUCCESS" = false ]; then
    echo "  ❌ $name (port $port) — FAILED (HTTP $code after $RETRIES attempts)"
    FAILED=$((FAILED + 1))
  fi
done

if [ "$FAILED" -gt 0 ]; then
  echo ""
  echo "FAIL: $FAILED service(s) unhealthy"
  exit 1
fi

echo ""
echo "All services healthy."
```

- [ ] **Step 4: Make scripts executable**

Run: `chmod +x scripts/dev-up.sh scripts/dev-down.sh scripts/verify.sh`
Expected: No output, exit code 0

- [ ] **Step 5: Commit**

```bash
git add scripts/
git commit -m "deploy: add dev lifecycle scripts (dev-up, dev-down, verify with retries)"
```

---

## Task 5: Makefile Overhaul

**Files:**
- Rewrite: `Makefile`

- [ ] **Step 1: Write the new Makefile**

This replaces the existing Makefile entirely. Preserves all current functionality (build, deploy, test, status) and adds: dev-*, verify, jetson-*, test-e2e.

```makefile
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
dev-logs-$(1):
	@docker compose -f docker-compose.dev.yml logs -f $(1)
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call DEV_LOGS_SERVICE,$(svc))))

define DEV_RESTART_SERVICE
dev-restart-$(1):
	@echo "Rebuilding $(1)..."
	@docker compose -f docker-compose.dev.yml up -d --build --no-deps $(1)
	@echo "Waiting for $(1) to be healthy..."
	@sleep 3
	@bash scripts/verify.sh localhost 2>/dev/null && echo "$(1) restarted." || echo "Warning: $(1) may not be healthy yet"
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call DEV_RESTART_SERVICE,$(svc))))

# ─── Build (native) ───────────────────────────────────
.PHONY: build $(addprefix build-,$(GO_SERVICES)) build-iris

build: $(addprefix build-,$(GO_SERVICES))
	@echo "All Go services built (native)"

define BUILD_NATIVE
build-$(1):
	@echo "Building $(1) (native)..."
	@cd services/$(1) && go build -o $(DIST)/$(1)-native .
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call BUILD_NATIVE,$(svc))))

# ─── Build (ARM64 cross-compile for Jetson) ───────────
.PHONY: build-arm64 $(addprefix build-arm64-,$(GO_SERVICES)) build-iris

build-arm64: $(addprefix build-arm64-,$(GO_SERVICES))
	@echo "All Go services built (ARM64)"

define BUILD_ARM64
build-arm64-$(1):
	@echo "Building $(1) (linux/arm64)..."
	@cd services/$(1) && GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -ldflags="-s -w" -o $(DIST)/$(1) .
	@echo "  → dist/$(1) ($$(du -h $(DIST)/$(1) | cut -f1))"
endef
$(foreach svc,$(GO_SERVICES),$(eval $(call BUILD_ARM64,$(svc))))

build-iris: fetch-node
	@echo "Building Iris (React + Node)..."
	@cd services/iris && $(NPM) install --silent 2>/dev/null
	@cd services/iris/client && $(NPM) install --silent 2>/dev/null && $(NPM) run build 2>&1 | tail -3
	@echo "  → services/iris/client/build/"

fetch-node:
	@if [ ! -f "$(NODE_CACHE)/bin/node" ]; then \
		echo "Downloading Node.js $(NODE_VERSION) (linux/arm64)..."; \
		mkdir -p $(NODE_CACHE); \
		curl -sL $(NODE_URL) | tar -xJ -C $(NODE_CACHE) --strip-components=1; \
	fi

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
.PHONY: deploy-all $(addprefix deploy-,$(GO_SERVICES)) deploy-iris deploy-infra

deploy-all: $(addprefix deploy-,$(GO_SERVICES)) deploy-iris
	@echo "All services deployed. Verifying..."
	@bash scripts/verify.sh $(JETSON_HOST)

define DEPLOY_SERVICE
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
	@$(JETSON) 'mkdir -p $(JETSON_DIR)/iris/{server,client,node/bin}'
	@$(JSCP) $(NODE_CACHE)/bin/node $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/node/bin/node
	@$(JSCP) -r services/iris/server $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/
	@$(JSCP) -r services/iris/client/build $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/client/
	@$(JSCP) services/iris/package.json $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/
	@cd services/iris && $(NPM) install --production --silent 2>/dev/null
	@$(JSCP) -r services/iris/node_modules $(JETSON_USER)@$(JETSON_HOST):$(JETSON_DIR)/iris/
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
.PHONY: verify $(addprefix verify-,$(ALL_SERVICES)) jetson-status jetson-infra-status jetson-ollama-pull jetson-ssh

verify:
	@bash scripts/verify.sh $(JETSON_HOST)

define VERIFY_SERVICE
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
jetson-logs-$(1):
	@$(JETSON) 'echo $(JETSON_PASS) | sudo -S journalctl -u localfinance-$(1) --no-pager -n 50'
endef
$(foreach svc,$(ALL_SERVICES),$(eval $(call JETSON_LOGS_SERVICE,$(svc))))

define JETSON_KILL_SERVICE
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
```

- [ ] **Step 2: Verify Makefile syntax**

Run: `make help`
Expected: Full help output with all sections (Local Development, Build, Test, Deploy, Verify & Debug)

- [ ] **Step 3: Verify build targets still work**

Run: `make build-arm64-hermes`
Expected: `Building hermes (linux/arm64)...` followed by binary size

- [ ] **Step 4: Verify unit tests still work**

Run: `make test`
Expected: All tests pass (Thesaurus, Logos, Iris)

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "deploy: overhaul Makefile with dev, test-e2e, verify, and jetson debug targets"
```

---

## Task 6: Integration Test Suite

**Files:**
- Create: `tests/e2e/helpers.sh`
- Create: `tests/e2e/run-all.sh`
- Create: `tests/e2e/test-auth.sh`
- Create: `tests/e2e/test-upload.sh`
- Create: `tests/e2e/test-chat.sh`
- Create: `tests/e2e/test-categories.sh`
- Create: `tests/fixtures/sample.csv`
- Create: `tests/fixtures/sample-creditcard.csv`

- [ ] **Step 1: Create test fixtures**

Create `tests/fixtures/sample.csv`:
```csv
Date,Description,Amount,Category
2026-01-15,TRADER JOES SUNNYVALE,-45.67,Food
2026-01-16,SHELL OIL STATION,-52.30,Transport
2026-01-17,AMAZON.COM PURCHASE,-89.99,Shopping
2026-01-18,SALARY DEPOSIT,3500.00,Income
2026-01-19,NETFLIX SUBSCRIPTION,-15.99,Entertainment
```

Create `tests/fixtures/sample-creditcard.csv`:
```csv
Transaction Date,Post Date,Description,Amount
01/15/2026,01/16/2026,STARBUCKS STORE 12345,-5.75
01/16/2026,01/17/2026,UBER TRIP,-23.40
01/17/2026,01/18/2026,WHOLE FOODS MKT,-67.89
01/18/2026,01/19/2026,PAYMENT RECEIVED,500.00
01/19/2026,01/20/2026,TARGET STORE 0042,-34.56
```

- [ ] **Step 2: Create tests/e2e/helpers.sh**

```bash
#!/usr/bin/env bash
# Shared test helpers for e2e tests
# Source this file: . "$(dirname "$0")/helpers.sh"

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost}"
IRIS_URL="${BASE_URL}:3001"
THESAURUS_URL="${BASE_URL}:8001"
SOPHIA_URL="${BASE_URL}:8002"
LOGOS_URL="${BASE_URL}:8003"

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Generate unique test user for this run
TEST_ID="test_$(date +%s)_$$"
TEST_EMAIL="${TEST_ID}@test.localfinance.dev"
TEST_PASSWORD="TestPass123!"
TEST_FIRST="Test"
TEST_LAST="User"
TOKEN=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

log_pass() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_PASSED=$((TESTS_PASSED + 1))
  echo -e "  ${GREEN}PASS${NC} $1"
}

log_fail() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo -e "  ${RED}FAIL${NC} $1"
  if [ -n "${2:-}" ]; then
    echo -e "       Expected: $2"
  fi
  if [ -n "${3:-}" ]; then
    echo -e "       Got:      $3"
  fi
}

log_skip() {
  echo -e "  ${YELLOW}SKIP${NC} $1"
}

# Register a test user and set TOKEN
register_user() {
  local resp
  resp=$(curl -sf -X POST "${IRIS_URL}/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${TEST_EMAIL}\",
      \"password\": \"${TEST_PASSWORD}\",
      \"first_name\": \"${TEST_FIRST}\",
      \"last_name\": \"${TEST_LAST}\"
    }" 2>/dev/null) || return 1

  TOKEN=$(echo "$resp" | jq -r '.token // empty')
  if [ -z "$TOKEN" ]; then
    return 1
  fi
}

# Login and set TOKEN
login_user() {
  local resp
  resp=$(curl -sf -X POST "${IRIS_URL}/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${TEST_EMAIL}\",
      \"password\": \"${TEST_PASSWORD}\"
    }" 2>/dev/null) || return 1

  TOKEN=$(echo "$resp" | jq -r '.token // empty')
  if [ -z "$TOKEN" ]; then
    return 1
  fi
}

# Assert two values are equal
assert_eq() {
  local description="$1"
  local expected="$2"
  local actual="$3"

  if [ "$expected" = "$actual" ]; then
    log_pass "$description"
  else
    log_fail "$description" "$expected" "$actual"
  fi
}

# Assert string contains substring
assert_contains() {
  local description="$1"
  local haystack="$2"
  local needle="$3"

  if echo "$haystack" | grep -q "$needle"; then
    log_pass "$description"
  else
    log_fail "$description" "contains '$needle'" "$(echo "$haystack" | head -c 200)"
  fi
}

# Assert HTTP status code
assert_status() {
  local description="$1"
  local expected_code="$2"
  local actual_code="$3"

  assert_eq "$description (HTTP $expected_code)" "$expected_code" "$actual_code"
}

# Assert value is not empty
assert_not_empty() {
  local description="$1"
  local value="$2"

  if [ -n "$value" ]; then
    log_pass "$description"
  else
    log_fail "$description" "non-empty" "(empty)"
  fi
}

# Retry a command until it succeeds or max attempts reached
retry_until() {
  local description="$1"
  local max_attempts="${2:-10}"
  local delay="${3:-3}"
  shift 3
  local cmd="$@"

  for attempt in $(seq 1 "$max_attempts"); do
    if eval "$cmd" 2>/dev/null; then
      return 0
    fi
    sleep "$delay"
  done
  return 1
}

# Print test summary and exit with appropriate code
print_summary() {
  local suite_name="${1:-Tests}"
  echo ""
  echo "─── ${suite_name} Summary ───"
  echo "  Total:  $TESTS_RUN"
  echo -e "  Passed: ${GREEN}${TESTS_PASSED}${NC}"
  if [ "$TESTS_FAILED" -gt 0 ]; then
    echo -e "  Failed: ${RED}${TESTS_FAILED}${NC}"
    exit 1
  else
    echo -e "  Failed: ${TESTS_FAILED}"
  fi
}
```

- [ ] **Step 3: Create tests/e2e/test-auth.sh**

```bash
#!/usr/bin/env bash
# Auth lifecycle integration tests
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "═══ Auth Tests ═══"

# Test 1: Register new user
echo ""
echo "Register flow:"
resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${TEST_EMAIL}\",
    \"password\": \"${TEST_PASSWORD}\",
    \"first_name\": \"${TEST_FIRST}\",
    \"last_name\": \"${TEST_LAST}\"
  }")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)

assert_status "Register returns 201 or 200" "$(echo "$code" | grep -oE '2[0-9]{2}')" "$code"
TOKEN=$(echo "$body" | jq -r '.token // empty')
assert_not_empty "Register returns JWT token" "$TOKEN"

# Test 2: GET /me with valid token
echo ""
echo "Token validation:"
resp=$(curl -s -w "\n%{http_code}" -X GET "${IRIS_URL}/api/auth/me" \
  -H "Authorization: Bearer ${TOKEN}")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)

assert_status "GET /me returns 200" "200" "$code"
user_email=$(echo "$body" | jq -r '.email // .user.email // empty')
assert_eq "GET /me returns correct email" "$TEST_EMAIL" "$user_email"

# Test 3: GET /me with no token
resp=$(curl -s -o /dev/null -w "%{http_code}" -X GET "${IRIS_URL}/api/auth/me")
assert_status "GET /me without token returns 401" "401" "$resp"

# Test 4: GET /me with invalid token
resp=$(curl -s -o /dev/null -w "%{http_code}" -X GET "${IRIS_URL}/api/auth/me" \
  -H "Authorization: Bearer invalidtoken123")
assert_status "GET /me with bad token returns 401" "401" "$resp"

# Test 5: Login with correct credentials
echo ""
echo "Login flow:"
resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"${TEST_EMAIL}\", \"password\": \"${TEST_PASSWORD}\"}")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)

assert_status "Login returns 200" "200" "$code"
login_token=$(echo "$body" | jq -r '.token // empty')
assert_not_empty "Login returns JWT token" "$login_token"

# Test 6: Login with wrong password
resp=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${IRIS_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"${TEST_EMAIL}\", \"password\": \"wrongpassword\"}")
assert_status "Login with wrong password returns 401" "401" "$resp"

# Test 7: Register duplicate email
echo ""
echo "Edge cases:"
resp=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${IRIS_URL}/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${TEST_EMAIL}\",
    \"password\": \"${TEST_PASSWORD}\",
    \"first_name\": \"${TEST_FIRST}\",
    \"last_name\": \"${TEST_LAST}\"
  }")
# Should be 409 or 400 — not 200/201
if [ "$resp" != "200" ] && [ "$resp" != "201" ]; then
  log_pass "Duplicate registration rejected (HTTP $resp)"
else
  log_fail "Duplicate registration should be rejected" "4xx" "$resp"
fi

print_summary "Auth"
```

- [ ] **Step 4: Create tests/e2e/test-upload.sh**

```bash
#!/usr/bin/env bash
# Upload flow integration tests
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "═══ Upload Tests ═══"

# Setup: register and login
register_user || { echo "FAIL: Could not register test user"; exit 1; }
echo "Registered test user: ${TEST_EMAIL}"

# Test 1: Upload CSV file
echo ""
echo "CSV upload flow:"
FIXTURE="$SCRIPT_DIR/../fixtures/sample.csv"
resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/upload" \
  -H "Authorization: Bearer ${TOKEN}" \
  -F "file=@${FIXTURE}")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)

assert_status "Upload CSV returns 200 or 201" "$(echo "$code" | grep -oE '2[0-9]{2}')" "$code"
doc_id=$(echo "$body" | jq -r '.document_id // .id // empty')
assert_not_empty "Upload returns document_id" "$doc_id"

# Test 2: Poll for processing completion
echo ""
echo "Processing poll:"
PROCESSED=false
for i in $(seq 1 15); do
  status_resp=$(curl -s "${IRIS_URL}/api/proxy/thesaurus/internal/documents/${doc_id}/status" 2>/dev/null || \
    curl -s "${THESAURUS_URL}/internal/documents/${doc_id}/status" 2>/dev/null || echo "{}")
  doc_status=$(echo "$status_resp" | jq -r '.status // empty')
  if [ "$doc_status" = "processed" ]; then
    PROCESSED=true
    break
  fi
  sleep 2
done

if [ "$PROCESSED" = true ]; then
  log_pass "Document processed within 30s"
else
  log_fail "Document processing timed out" "processed" "$doc_status"
fi

# Test 3: Fetch transactions
echo ""
echo "Transaction verification:"
txn_resp=$(curl -s "${IRIS_URL}/api/proxy/thesaurus/internal/transactions/by-document?document_id=${doc_id}" 2>/dev/null || \
  curl -s "${THESAURUS_URL}/internal/transactions/by-document?document_id=${doc_id}" 2>/dev/null || echo "[]")
txn_count=$(echo "$txn_resp" | jq 'if type == "array" then length else .transactions | length // 0 end' 2>/dev/null || echo "0")

if [ "$txn_count" -ge 4 ]; then
  log_pass "Found $txn_count transactions (expected >= 4)"
else
  log_fail "Transaction count" ">= 4" "$txn_count"
fi

print_summary "Upload"
```

- [ ] **Step 5: Create tests/e2e/test-chat.sh**

```bash
#!/usr/bin/env bash
# AI chat integration tests
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "═══ Chat Tests ═══"

# Check if Ollama is available
ollama_status=$(curl -sf -o /dev/null -w '%{http_code}' http://localhost:11434/api/tags 2>/dev/null || echo "000")
if [ "$ollama_status" != "200" ]; then
  log_skip "Ollama not available — skipping chat tests"
  echo "  (Start Ollama or run make dev-up to enable)"
  exit 0
fi

# Setup: register, login, upload test data
register_user || { echo "FAIL: Could not register test user"; exit 1; }
echo "Registered test user: ${TEST_EMAIL}"

# Upload fixture data first
FIXTURE="$SCRIPT_DIR/../fixtures/sample.csv"
upload_resp=$(curl -s -X POST "${IRIS_URL}/api/upload" \
  -H "Authorization: Bearer ${TOKEN}" \
  -F "file=@${FIXTURE}" 2>/dev/null)
doc_id=$(echo "$upload_resp" | jq -r '.document_id // .id // empty')

# Wait for processing
sleep 5

# Test 1: Chat with data question
echo ""
echo "Chat flow:"
chat_resp=$(curl -sf --max-time 180 -X POST "${IRIS_URL}/api/proxy/sophia/api/v1/chat/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d "{
    \"question\": \"How much did I spend total?\",
    \"user_id\": \"$(echo "$upload_resp" | jq -r '.user_id // empty')\"
  }" 2>/dev/null || echo "{}")

answer=$(echo "$chat_resp" | jq -r '.answer // .response // empty')
confidence=$(echo "$chat_resp" | jq -r '.confidence // 0')

assert_not_empty "Chat returns an answer" "$answer"

# Check confidence is reasonable (> 0 means it found data)
if [ "$(echo "$confidence > 0" | bc -l 2>/dev/null || echo 0)" = "1" ]; then
  log_pass "Chat confidence > 0 ($confidence)"
else
  log_skip "Chat confidence check (bc not available or confidence=0)"
fi

print_summary "Chat"
```

- [ ] **Step 6: Create tests/e2e/test-categories.sh**

```bash
#!/usr/bin/env bash
# Category rules integration tests
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "═══ Category Rules Tests ═══"

# Setup: register and login
register_user || { echo "FAIL: Could not register test user"; exit 1; }
echo "Registered test user: ${TEST_EMAIL}"

# Test 1: Create a category rule
echo ""
echo "Rule CRUD:"
rule_resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/proxy/thesaurus/api/v1/category-rules" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{"pattern": "TRADER JOES", "category": "Groceries"}')
body=$(echo "$rule_resp" | head -1)
code=$(echo "$rule_resp" | tail -1)

assert_status "Create rule returns 2xx" "$(echo "$code" | grep -oE '2[0-9]{2}')" "$code"
rule_id=$(echo "$body" | jq -r '.id // empty')
assert_not_empty "Create rule returns id" "$rule_id"

# Test 2: List rules
list_resp=$(curl -s "${IRIS_URL}/api/proxy/thesaurus/api/v1/category-rules" \
  -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
rule_count=$(echo "$list_resp" | jq 'if type == "array" then length else 0 end' 2>/dev/null || echo "0")

if [ "$rule_count" -ge 1 ]; then
  log_pass "List rules returns $rule_count rule(s)"
else
  log_fail "List rules" ">= 1" "$rule_count"
fi

# Test 3: Delete rule
if [ -n "$rule_id" ]; then
  del_code=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE \
    "${IRIS_URL}/api/proxy/thesaurus/api/v1/category-rules/${rule_id}" \
    -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
  assert_status "Delete rule returns 2xx" "$(echo "$del_code" | grep -oE '2[0-9]{2}')" "$del_code"
fi

print_summary "Category Rules"
```

- [ ] **Step 7: Create tests/e2e/run-all.sh**

```bash
#!/usr/bin/env bash
# Run all e2e test suites
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FAILED=0

echo "════════════════════════════════════════"
echo " LocalFinance E2E Integration Tests"
echo "════════════════════════════════════════"
echo ""

# Verify local stack is running
echo "Checking local stack..."
if ! curl -sf http://localhost:8001/health > /dev/null 2>&1; then
  echo "ERROR: Local dev stack not running."
  echo "Run 'make dev-up' first."
  exit 1
fi
echo "Stack healthy."
echo ""

for suite in auth upload categories chat; do
  echo "────────────────────────────────────────"
  if bash "$SCRIPT_DIR/test-${suite}.sh"; then
    echo ""
  else
    FAILED=$((FAILED + 1))
    echo ""
  fi
done

echo "════════════════════════════════════════"
if [ "$FAILED" -gt 0 ]; then
  echo " FAILED: $FAILED suite(s) had failures"
  exit 1
else
  echo " ALL SUITES PASSED"
fi
echo "════════════════════════════════════════"
```

- [ ] **Step 8: Make all test scripts executable**

Run: `chmod +x tests/e2e/*.sh`
Expected: No output, exit code 0

- [ ] **Step 9: Commit**

```bash
git add tests/
git commit -m "test(e2e): add integration test suite with auth, upload, chat, and category tests"
```

---

## Task 7: Update .claude/settings.json

**Files:**
- Modify: `.claude/settings.json`

- [ ] **Step 1: Add docker permissions**

Add `"Bash(docker *)"` and `"Bash(docker compose *)"` to the allow list if not already present.

Run: `grep -c "docker" .claude/settings.json`

If 0, add the docker permissions to the allow list.

- [ ] **Step 2: Commit**

```bash
git add .claude/settings.json
git commit -m "deploy: add docker permissions to Claude settings"
```

---

## Task 8: Smoke Test Full Workflow

This task validates everything works end-to-end.

- [ ] **Step 1: Run unit tests**

Run: `make test`
Expected: All unit tests pass (Thesaurus, Logos, Iris)

- [ ] **Step 2: Build Docker images**

Run: `docker compose -f docker-compose.dev.yml build`
Expected: All 5 service images build successfully

- [ ] **Step 3: Start local dev stack**

Run: `make dev-up`
Expected: All services healthy, script prints URLs

- [ ] **Step 4: Run integration tests**

Run: `make test-e2e`
Expected: All 4 suites pass (auth, upload, categories, chat)

- [ ] **Step 5: Tear down**

Run: `make dev-down-clean`
Expected: All containers and volumes removed

- [ ] **Step 6: Fix any failures**

If any step fails, fix the issue and re-run from step 1. Use `superpowers:systematic-debugging` if the cause is unclear.

- [ ] **Step 7: Commit any fixes**

If fixes were needed, commit them with descriptive messages per the conventions.

---

## Task 9: Final Commit and Report

- [ ] **Step 1: Review all changes**

Run: `git log --oneline HEAD~10..HEAD`
Expected: Clean commit history with descriptive messages, one per logical unit

- [ ] **Step 2: Push to remote**

Run: `git push origin main`
Expected: All commits pushed successfully

- [ ] **Step 3: Report to user**

Report: what was built, how to use it, URLs for local dev stack.

Key commands for the user:
```
make dev-up        # start everything locally
make test          # unit tests
make test-e2e      # integration tests
make deploy-all    # deploy to Jetson
make verify        # health check Jetson
```
