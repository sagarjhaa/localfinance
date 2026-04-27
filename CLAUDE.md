# LocalFinance

Privacy-first personal finance system. Local Ollama AI, runs on macOS.

## Architecture

| Service | Port | Responsibility |
|---------|------|---------------|
| **Iris** | 3001 | React UI + Express proxy. ZERO business logic. |
| **Hermes** | 3000 | API Gateway (future routing/aggregation) |
| **Thesaurus** | 8001 | Auth, CRUD, PostgreSQL. Source of truth for all data. |
| **Sophia** | 8002 | AI/Ollama integration. Two-pass chat with real data. |
| **Logos** | 8003 | Document parsing (CSV, PDF). Stateless processor. |

**Infra:** PostgreSQL 15, Redis 7, Ollama (configurable model, default `llama3.1:8b`), MinIO

### Boundaries — NEVER violate these

- **Iris is DUMB.** No parsing, no data transformation, no business logic, no direct DB access. It proxies requests and renders UI.
- **All file processing in Logos.** CSV, PDF, Excel parsing happens only in Logos.
- **All data persistence through Thesaurus.** Every read/write goes through Thesaurus REST API.
- **All AI through Sophia.** Ollama calls only happen in Sophia.
- **Internal service calls use `/internal/` routes** (no auth middleware). User-facing routes use auth middleware.

### Data Flows

**Upload:** Browser → Iris proxy → Thesaurus (store doc) → Logos (parse) → Thesaurus (store transactions) → Sophia (generate Month in Review)

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
   Types: feat, fix, refactor, test, docs
   Never batch frontend + backend in one commit

5. VERIFY
   make verify              ← health check all services on localhost
   If verify fails → docker logs, fix, go back to step 3
   Use superpowers:verification-before-completion before claiming done

6. REPORT
   Use superpowers:requesting-code-review for self-review
   Tell user: what was built, which URLs to test
   Show clean git log of all commits
```

## Commands

### Local Development (Docker)
```
make dev-up                  # Start full local stack (all services + infra)
make dev-up CHAT_MODEL=qwen2.5:7b   # Override default model
make dev-down                # Tear down local stack
make dev-down-clean          # Tear down + delete volumes
make dev-logs                # Tail all service logs
make dev-logs-{service}      # Tail one service
make dev-restart-{service}   # Rebuild + restart one container
```

### Build
```
make build                   # Build all Go services
make build-{service}         # Build one (hermes|thesaurus|sophia|logos)
make build-iris              # Build React + bundle Node server
```

### Test
```
make test                    # All unit tests (Go + Node)
make test-{service}          # One service
make test-e2e                # Integration tests (requires make dev-up)
make test-e2e-{suite}        # One suite (auth|upload|chat|categories)
```

### Eval (Phase 0 hallucination gate)
```
make eval-hallucination      # Run live-Ollama eval — manual graded
```

### Verify
```
make verify                  # Health check all services on localhost
make verify-{service}        # Health check one
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
2. `make dev-logs-{service}` — read recent log output
3. Match pattern → known fix:
   - `address already in use` → `make dev-restart-{service}`
   - `connection refused` to DB → check `docker compose ps`, restart infra
   - `401 Unauthorized` on internal call → move endpoint to `/internal/` route group
   - `model not found` → `ollama pull <model-name>`
   - `timeout` → increase timeout in code, retry
4. Fix → rebuild → restart → `make verify`
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

## Runtime Target

- **Platform:** macOS (Apple Silicon recommended)
- **Mode:** Local Docker stack (`make dev-up`) for development; `LocalFinance.app` for end-user install (Phase 1 deliverable)
- **Ollama:** Assumed pre-installed via `brew install ollama`; default model `llama3.1:8b`
- **Data:** Postgres in Docker today; embedded in `.app` once installer ships

## Active plan

See `docs/designs/product-direction.md` and `docs/designs/phase1-implementation-plan.md` for current scope and execution plan.
