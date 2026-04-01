# Claude-First Repo Design Spec

**Date:** 2026-03-31
**Status:** Approved
**Goal:** Make localfinance fully autonomous for AI agents — an agent takes a feature request, plans, codes, tests, deploys to Jetson, verifies, and reports back with clean commits.

---

## Problem Statement

From real development sessions, agents working on this repo suffer from:

1. **No verification loop** — agent deploys, user tests in browser, reports back "it's broken", agent debugs. Repeat.
2. **Raw SSH/SCP everywhere** — every deploy is manual commands, killing processes, checking logs.
3. **No local testing** — all testing happens on the Jetson, making the feedback loop 5-10 minutes per change.
4. **No architectural guardrails** — agent put parsing logic in Iris (wrong service), hardcoded merchants, broke proxies.
5. **Giant commits** — 40-90 files per commit, impossible to review or revert.
6. **Stale processes** — old binaries kept running, agent couldn't tell.
7. **Permission prompt fatigue** — 130+ one-off permission approvals per session.

## Solution: Agent Playbook

Seven components that together make the repo fully autonomous for agents.

---

## 1. CLAUDE.md — The Agent Contract

Replace the current CLAUDE.md (outdated — references Python/SQLite/Flask) with an agent-oriented document.

### Structure

```
CLAUDE.md
├── Architecture
│   ├── Service map (name, port, responsibility, one-liner)
│   ├── Boundaries (what goes where, what Iris must NOT do)
│   └── Data flow (upload, chat, auth)
├── Agent Workflow
│   ├── Mandatory sequence: plan → code → test → commit → deploy → verify
│   ├── Never run raw ssh/scp/sshpass — use make targets
│   ├── Always run make test before deploying
│   ├── Always run make verify after deploying
│   └── Auto-debug failures (up to 3 retries before escalating)
├── Commands
│   ├── Local dev: make dev-up, dev-down, dev-logs, dev-restart-*
│   ├── Build: make build, build-*, build-arm64, build-arm64-*
│   ├── Test: make test, test-*, test-e2e, test-e2e-*
│   ├── Deploy: make deploy-*, deploy-all
│   └── Verify: make verify, verify-*, jetson-logs-*, jetson-status
├── Commit Conventions
│   ├── Format: type(service): what and why
│   ├── Types: feat, fix, refactor, test, docs, deploy
│   ├── One logical unit per commit
│   └── Never batch frontend + backend + deploy in one commit
├── Boundaries
│   ├── Iris is a DUMB frontend — no parsing, no business logic
│   ├── All file processing in Logos
│   ├── All data persistence through Thesaurus
│   ├── All AI through Sophia
│   └── Internal service calls use /internal/ routes (no auth)
└── Debugging Protocol
    ├── Step 1: make verify — identify which service
    ├── Step 2: make jetson-logs-* — read last 50 lines
    ├── Step 3: Match log pattern → known fix
    ├── Step 4: Fix, rebuild, redeploy, re-verify (max 3 retries)
    └── Step 5: Escalate to user with logs + diagnosis
```

### Key Rules

- Never run raw `ssh`, `scp`, `sshpass` — always use `make deploy-*`, `make jetson-logs-*`
- Never commit more than one logical unit per commit
- Always `make test` before deploy, always `make verify` after deploy
- If `make verify` fails, auto-debug: check logs, fix, retry (up to 3 times)
- Iris must NEVER: parse files, transform data, run business logic, store state beyond auth tokens
- Service-to-service calls use `/internal/` route group (no auth middleware)

---

## 2. Per-Service CLAUDE.md

Each service gets a `CLAUDE.md` (50-100 lines) answering:

1. What does this service do?
2. How do I build it?
3. How do I add/change things?
4. How do I test it?
5. How do I deploy it?

### services/thesaurus/CLAUDE.md

```markdown
# Thesaurus — Data & Auth Service (port 8001)

Source of truth for all persistent data. Owns PostgreSQL.
Handles: user auth (JWT/Argon2id), accounts, transactions, budgets,
documents, category rules, statement periods.

## Build
make build-thesaurus          # native (for local Docker)
make build-arm64-thesaurus    # cross-compile for Jetson

## Adding an Endpoint
1. Add/update model in models/ (with GORM tags)
2. Add handler in api/handlers/
3. Register route in api/routes.go
4. Add to AutoMigrate list in database/database.go
5. User-facing: add under protected group (requires AuthMiddleware)
6. Service-to-service: add under /internal/ group (no auth)

## Testing
make test-thesaurus    # unit tests (auth, models, handlers)
make test-e2e          # integration tests against local Docker

## Deploy
make deploy-thesaurus  # build ARM64 + SCP + restart + verify health

## Common Patterns
- User-scoped queries: always filter by user_id from JWT
- Bulk operations: use /internal/transactions/bulk (Logos calls this)
- Models use UUID primary keys via BeforeCreate hook
- JWT secret from env var JWT_SECRET
```

### services/iris/CLAUDE.md

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
make build-iris    # builds React client + bundles Node server

## Adding a Page
1. Create component in client/src/pages/
2. Add route in client/src/App.jsx
3. Add nav link in sidebar (Dashboard.jsx has the sidebar template)
4. API calls go through client/src/api/client.js

## Adding an API Proxy Route
1. All backend calls proxy through server/routes/proxy.js
2. Add service URL mapping if new service
3. Never add business logic in proxy — just forward request/response

## Testing
make test-iris    # unit tests (auth routes)
make test-e2e     # integration tests

## Deploy
make deploy-iris  # build React + bundle + SCP + restart + verify
```

### services/sophia/CLAUDE.md

```markdown
# Sophia — AI Service (port 8002)

AI-powered insights using local Ollama (llama3.2:1b).
Two-pass chat: parse intent → fetch real data from Thesaurus → generate answer.

## Build
make build-sophia          # native
make build-arm64-sophia    # cross-compile for Jetson

## Adding AI Features
1. Add handler in api/handlers/
2. Register route in api/routes.go
3. Use ai/service.go for Ollama calls
4. Always fetch real data from Thesaurus — never hallucinate numbers
5. Timeout for Ollama calls: 120s minimum (Jetson is slow)

## Two-Pass Chat Flow
1. Pass 1: User question → Ollama generates query intent (JSON)
2. Execute: Call Thesaurus /internal/ endpoints with intent params
3. Pass 2: Real data + question → Ollama generates natural language answer

## Testing
make test-sophia   # unit tests
make test-e2e      # integration tests (requires Ollama running)

## Deploy
make deploy-sophia # build ARM64 + SCP + restart + verify
```

### services/logos/CLAUDE.md

```markdown
# Logos — Document Processing (port 8003)

Stateless document parser. Receives files from Thesaurus, parses
CSV/PDF, extracts transactions, sends back to Thesaurus.

## Build
make build-logos          # native
make build-arm64-logos    # cross-compile for Jetson

## Processing Flow
1. Thesaurus receives file upload, stores document record, fires to Logos
2. Logos detects file type (CSV/PDF)
3. Logos parses transactions
4. Logos applies user's category rules (from Thesaurus /internal/category-rules/)
5. Unmatched transactions → batch AI categorization via Ollama
6. Logos sends transactions to Thesaurus /internal/transactions/bulk
7. Logos updates document status to "processed"

## Adding a New File Format
1. Create processor in processors/ implementing the Processor interface
2. Register in processors/manager.go
3. Add format detection in processors/detector.go
4. Add test with fixture file in processors/*_test.go

## Testing
make test-logos    # unit tests (CSV parser, PDF parser, detector)
make test-e2e     # integration tests

## Deploy
make deploy-logos  # build ARM64 + SCP + restart + verify
```

### services/hermes/CLAUDE.md

```markdown
# Hermes — API Gateway (port 3000)

Future routing and aggregation layer. Currently minimal — serves
health endpoint. Will eventually replace Iris's proxy role.

## Build
make build-hermes          # native
make build-arm64-hermes    # cross-compile for Jetson

## Deploy
make deploy-hermes # build ARM64 + SCP + restart + verify
```

---

## 3. Makefile — The Single Interface

The Makefile is the only interface agents use. No raw commands.

### Target Map

```makefile
# ──────────────────────────────────────────────
# Local Development (Docker)
# ──────────────────────────────────────────────
make dev-up                  # Start full local stack (all services + infra)
make dev-down                # Tear down local stack
make dev-logs                # Tail all service logs
make dev-logs-{service}      # Tail one service
make dev-restart-{service}   # Rebuild + restart one service container

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────
make build                   # Build all Go services (native, for Docker)
make build-{service}         # Build one service (native)
make build-arm64             # Cross-compile all for Jetson
make build-arm64-{service}   # Cross-compile one
make build-iris              # Build React client + bundle Node server

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────
make test                    # All unit tests (Go + Node)
make test-{service}          # One service unit tests
make test-e2e                # Integration tests against local Docker stack
make test-e2e-{suite}        # One integration test suite (auth, upload, chat, categories)
make lint                    # Go vet + staticcheck (future: eslint)

# ──────────────────────────────────────────────
# Deploy to Jetson
# ──────────────────────────────────────────────
make deploy-{service}        # Build ARM64 + SCP + restart + verify health
make deploy-all              # All services

# ──────────────────────────────────────────────
# Verify & Debug (Jetson)
# ──────────────────────────────────────────────
make verify                  # Health check all 5 services on Jetson
make verify-{service}        # Health check one
make jetson-logs-{service}   # Tail logs on Jetson
make jetson-status           # systemctl status all services
make jetson-kill-{service}   # Kill stale process by port
make jetson-infra-status     # Check Postgres, Redis, Ollama, MinIO
make jetson-ollama-pull      # Pull llama3.2:1b if missing
make jetson-ssh              # SSH into Jetson
```

### Design Decisions

- `deploy-*` chains: `build-arm64-* → scp → restart → verify`. If verify fails, the target fails.
- `verify` uses curl with retries: 3 attempts, 5 seconds apart. Handles slow startup.
- `test-e2e` requires `dev-up` running. Tests against localhost, not Jetson.
- `dev-up` waits for all health checks before returning.
- No raw SSH anywhere in agent workflow.

---

## 4. Docker Compose Local Stack

Full local environment mirroring the Jetson. Everything in Docker, nothing on Mac.

### docker-compose.dev.yml

**Infrastructure (same as Jetson):**
- PostgreSQL 15 (port 5432)
- Redis 7 (port 6379)
- Ollama (port 11434) — auto-pulls llama3.2:1b on first start
- MinIO (port 9000)

**Application services (new):**
- Thesaurus — Go service, built in multi-stage Dockerfile
- Sophia — Go service, connects to Ollama and Thesaurus
- Logos — Go service, connects to Thesaurus and Ollama
- Hermes — Go service
- Iris — Node + React, dev server with hot reload

### Design Decisions

1. **Multi-stage Dockerfiles for Go services.** Build stage uses golang:1.22, runtime uses scratch/alpine. Keeps images small.
2. **Same env vars as Jetson.** Services connect to `postgres:5432` not `localhost:5432`. Config is identical.
3. **Ollama model auto-pull.** An init container runs `ollama pull llama3.2:1b` on first start.
4. **Health checks on all services.** Docker-level health checks on `/health` endpoints. `make dev-up` script waits for all healthy.
5. **Named volumes for data persistence.** DB data survives `make dev-restart-*`. Only `make dev-down` with `--volumes` wipes data.
6. **Source mounts for Go services.** Code is mounted as volume. `make dev-restart-sophia` rebuilds inside container without recreating from scratch.

### Network

All services on a single Docker network `localfinance-net`. Services reference each other by container name:
- `http://thesaurus:8001`
- `http://sophia:8002`
- `http://logos:8003`
- `http://ollama:11434`
- `postgres://postgres:5432`

---

## 5. Integration Test Suite

Shell-script tests that run against the local Docker stack. Pure curl + jq + bash.

### Structure

```
tests/
├── e2e/
│   ├── run-all.sh           # Entry point (make test-e2e calls this)
│   ├── test-auth.sh         # Auth lifecycle
│   ├── test-upload.sh       # File upload → parse → transactions
│   ├── test-chat.sh         # AI chat with real data
│   ├── test-categories.sh   # User category rules
│   └── helpers.sh           # Shared: register, login, assert, retry
├── fixtures/
│   ├── sample.csv           # 5 fake transactions
│   └── sample-creditcard.csv # Credit card format
```

### Test Suites

**test-auth.sh:**
- Register new user → assert 201, JWT returned
- Login → assert 200, JWT returned
- GET /me with token → assert user data
- Refresh token → assert new JWT
- Logout → assert old token rejected
- Register duplicate email → assert 409

**test-upload.sh:**
- Login, upload sample.csv via Iris
- Poll document status until "processed" (retry_until, max 30s)
- Fetch transactions by document_id
- Assert transaction count matches CSV row count
- Assert categories are assigned (not all "Other")
- Assert amounts match CSV values

**test-chat.sh:**
- Login, upload fixture data
- POST chat: "how much did I spend on food?"
- Assert response contains a dollar amount
- Assert confidence > 0.5
- Assert response references real transaction data (not hallucinated)

**test-categories.sh:**
- Login, create rule: "TestMerchant" → "TestCategory"
- Upload CSV containing "TestMerchant" transaction
- Fetch transactions, assert "TestMerchant" categorized as "TestCategory"
- Delete rule, re-upload, assert AI categorization takes over

### helpers.sh

```bash
# Register a test user, return JWT
register_user() { ... }

# Assert equality with clear error message
assert_eq() { ... }

# Assert string contains substring
assert_contains() { ... }

# Retry a command until it succeeds (for polling)
retry_until() { ... }

# Cleanup: delete test user data
cleanup() { ... }
```

### Design Decisions

- Each test creates its own user — tests don't interfere with each other
- Tests clean up after themselves
- `retry_until` for async operations (document processing)
- Exit code 0 = pass, non-zero = fail with descriptive message
- No test framework to install — just bash, curl, jq
- Fixture files committed to repo (fake data only, never real statements)

---

## 6. Commit Conventions

### Format

```
<type>(<service>): <what changed and why>

<optional body — only if the "why" isn't obvious from the subject>

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Types

| Type | When |
|---|---|
| `feat` | New capability |
| `fix` | Bug fix |
| `refactor` | Restructure without behavior change |
| `test` | Add/update tests |
| `docs` | Documentation only |
| `deploy` | Deployment config, Makefile, Docker, systemd |

### Rules

1. **One logical unit per commit.** If you can describe it with "and", split it.
2. **Never batch frontend + backend + deploy** in one commit.
3. **Message explains why**, not just what. "Add PDF parser for Amex/Chase statement formats" not "create pdf_processor.go".
4. **Test commits can bundle with their feature** or be separate — agent's judgment.
5. **Deploy/config changes are always their own commit.**

### Examples

What happened in past sessions (bad):
```
# 91 files in one commit
"🚀 Add Makefile CD pipeline, remove legacy Python, fix shared middleware"
```

What should have happened (good):
```
deploy: add Makefile with cross-compile and deploy targets
deploy: add systemd units for all 5 services
refactor: remove legacy Python files and update .gitignore
fix(shared): remove unused net/http import from correlation middleware
```

---

## 7. Debugging Protocol

Baked into CLAUDE.md. Agent follows this when `make verify` fails.

### Decision Tree

```
make verify fails
  │
  ├─ Which service? → make verify-{service}
  │
  ├─ Read logs → make jetson-logs-{service}
  │
  ├─ Match pattern:
  │   ├─ "address already in use"  → make jetson-kill-{service}, redeploy
  │   ├─ "connection refused" to DB → make jetson-infra-status, restart infra
  │   ├─ "401 Unauthorized"        → endpoint needs /internal/ route (no auth)
  │   ├─ "model not found"         → make jetson-ollama-pull
  │   ├─ "no such file"            → binary not deployed, re-run make deploy-*
  │   ├─ "timeout"                 → increase timeout, retry
  │   └─ unknown                   → read full logs, diagnose
  │
  ├─ Apply fix → rebuild → redeploy → make verify
  │
  ├─ Still failing? → retry (max 3 attempts)
  │
  └─ 3 failures → escalate to user with:
      ├─ What failed
      ├─ Relevant log lines
      ├─ What was tried
      └─ Suspected root cause
```

### Helper Targets

```makefile
make jetson-kill-{service}     # Kill stale process by port number
make jetson-infra-status       # Check Postgres, Redis, Ollama, MinIO health
make jetson-ollama-pull        # Pull llama3.2:1b model if missing
```

---

## Implementation Order

1. **CLAUDE.md rewrite** — agent contract, boundaries, workflow, commands
2. **Per-service CLAUDE.md files** — 5 files, one per service
3. **Makefile overhaul** — replace current Makefile with full target map
4. **Docker Compose local stack** — docker-compose.dev.yml + Dockerfiles
5. **Integration test suite** — test scripts + fixtures
6. **Commit convention enforcement** — document in CLAUDE.md (no tooling needed, agent follows rules)
7. **Debugging protocol** — document in CLAUDE.md (no tooling needed, agent follows decision tree)

Steps 1-2 are documentation. Steps 3-5 are infrastructure. Steps 6-7 are conventions already covered by step 1.

---

## What This Enables

**Before (from real sessions):**
```
User: "Add category rules"
Agent: codes → deploys → broken → user tests in browser → reports error
     → agent checks logs → fixes → redeploys → still broken
     → user reports again → agent fixes again → finally works
     → commits 40 files in one commit
     → 3 hours elapsed
```

**After:**
```
User: "Add category rules"
Agent: writes spec → user approves
     → codes model + handler + UI + tests
     → make test (unit tests pass)
     → make dev-up && make test-e2e (integration tests pass)
     → commits: feat(thesaurus), feat(iris), test(e2e) — 3 clean commits
     → make deploy-all (build + deploy + verify)
     → verify passes → reports back with working URL
     → 1 hour elapsed, zero user involvement after approval
```
