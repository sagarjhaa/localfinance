# LocalFinance

Privacy-first personal finance for macOS. Local Ollama AI, embedded Postgres,
single Go binary that serves the React UI.

## Architecture

One binary at `cmd/localfinance` does everything: serves the React UI from a
`go:embed`ed bundle, owns auth/data/parse/AI/insights, and embeds Postgres on
the .app build path (or talks to host Postgres in `make dev`).

Internal packages:

| Package | Responsibility |
|---------|---------------|
| `internal/api` | Gin router and HTTP handlers (auth, data, parse, AI, insights, monthreview, setup) |
| `internal/auth` | JWT, Argon2id password hashing, default-user seed |
| `internal/data` | GORM models, repositories, embedded Postgres lifecycle, config |
| `internal/parse` | PDF/CSV ingest pipeline (Sophia AI parse) |
| `internal/ai` | Ollama client + two-pass chat + insight narrator |
| `internal/insights` | Deterministic rules engine + Phase A/B narrator |
| `internal/monthreview` | Period summary cache |
| `internal/ollama` | Host Ollama probe + model pull SSE |
| `internal/webui` | `go:embed` of React build output |
| `internal/postgres` | Embedded Postgres manager (.app path) |

The React source lives at `services/iris/client/`. `make webui-build` writes
its `build/` into `internal/webui/dist/` for embedding.

## Development Cycle

Every task follows this cycle.

```
1. PLAN
   superpowers:brainstorming → feature design
   superpowers:writing-plans → implementation plan
   Get user approval before writing code.

2. CODE
   superpowers:subagent-driven-development for parallel execution.
   One logical unit at a time.

3. TEST
   make test                ← Go unit tests
   make dev                 ← run the binary against host Postgres + Ollama
   make test-e2e            ← integration tests
   If tests fail → superpowers:systematic-debugging.

4. COMMIT
   One commit per logical unit.
   Format: type(scope): what and why
   Types: feat, fix, refactor, test, docs, chore
   Never batch frontend + backend in one commit.

5. VERIFY
   make verify              ← curl localhost:3001/health
   superpowers:verification-before-completion before claiming done.

6. REPORT
   superpowers:requesting-code-review for self-review.
   Report: what was built, how to test, clean git log.
```

## Commands

### Run / build
```
make dev                     # go run ./cmd/localfinance against host PG + Ollama
make build                   # dist/localfinance (embeds React UI)
make webui-build             # rebuild React only → internal/webui/dist
make installer               # dist/LocalFinance.app
make installer-run           # build + open the .app
make installer-clean         # rm dist/LocalFinance.app
make clean                   # rm dist/
```

### Test / verify
```
make test                    # go test ./internal/... ./cmd/...
make test-e2e                # tests/e2e/run-all.sh (requires make dev running)
make eval-hallucination      # insight hallucination eval (live Ollama, EVAL_OLLAMA=1)
make verify                  # curl localhost:3001/health
```

## Commit Conventions

```
type(scope): what changed and why

Co-Authored-By: Claude <noreply@anthropic.com>
```

One logical unit per commit. If you can describe it with "and", split it.

Good:
```
feat(insights): add weekly cadence rule
fix(parse): handle BOM in Wells Fargo CSV
test(e2e): cover category-rules upload flow
```

Bad:
```
Add rules, settings page, and Makefile target    ← too many things
fix stuff                                          ← meaningless
```

## Debugging Protocol

When `make verify` fails:

1. `curl -v http://localhost:3001/health` — what does it say?
2. Tail server logs — `~/Library/Application Support/LocalFinance/logs/`
   when running the .app, stdout when running `make dev`.
3. Match pattern → known fix:
   - `address already in use` → kill the prior `localfinance` process
   - `connection refused` to DB → start Postgres (`docker compose -f docker-compose.dev.yml up -d`)
   - `ollama: connection refused` → `ollama serve &`
   - `model not found` → `ollama pull gemma3:4b`
4. Fix → rebuild → re-run → `make verify`.
5. Max 3 retries, then escalate with: what failed, logs, what was tried, suspected cause.

## Superpowers Skills

| Skill | When |
|-------|------|
| `superpowers:brainstorming` | Before any new feature — explore design with user |
| `superpowers:writing-plans` | After design approval — create implementation plan |
| `superpowers:subagent-driven-development` | Executing a plan — dispatch parallel agents per task |
| `superpowers:systematic-debugging` | When tests fail or behavior is unexpected |
| `superpowers:requesting-code-review` | After completing work — self-review before reporting |
| `superpowers:verification-before-completion` | Before claiming done |
| `superpowers:finishing-a-development-branch` | When work is complete — merge/PR/cleanup |
