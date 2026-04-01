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
make dev-down-clean          # Tear down + delete volumes
make dev-logs                # Tail all service logs
make dev-logs-{service}      # Tail one service
make dev-restart-{service}   # Rebuild + restart one container
```

### Build
```
make build                   # Build all Go services (native)
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
