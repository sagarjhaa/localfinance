# LocalFinance Service Consolidation — Design

**Date:** 2026-04-27
**Author:** Brainstormed via `/Users/sagar/.claude/plugins/cache/claude-plugins-official/superpowers/5.0.7/skills/brainstorming`
**Status:** APPROVED — ready for implementation plan

## Goal

Collapse the current 5-service architecture (Iris/React, Iris/Express, Hermes, Thesaurus, Sophia, Logos) into a single Go binary that:

1. Serves the React UI from embedded static assets
2. Owns all auth, data, AI, parse, insights, and month-review logic in-process
3. Manages an embedded Postgres
4. Walks non-technical users through Ollama install and model download on first run
5. Ships as `LocalFinance.app` for macOS — double-click to launch, browser opens to dashboard

## Why

The 5-service split was built for a multi-host Jetson world. The product is now Mac-only, single-user, local-only. Today's runtime cost: 5 child processes, internal HTTP between every pair, JSON serialization overhead on every page load, more failure surface than the use case warrants. The split also makes the `.app` installer lifecycle harder than it needs to be (orphan-process risk on crash).

Moving to a single Go binary using a **modular monolith** internal layout preserves every service boundary as a Go package with a clean interface. If we ever need to re-split (e.g. for Couples Mode or a future cloud sync product), each package can be wrapped in a thin HTTP handler — work measured in days, not weeks.

## Architecture

### One binary, two processes

```
LocalFinance.app launched
  ├─ embedded Postgres (child via embedded-postgres-go)
  └─ localfinance http server on :3001
       ├─ /                → embedded React build (go:embed)
       ├─ /api/auth/*      → auth handlers
       ├─ /api/v1/*        → unified API
       ├─ /api/setup/*     → first-run wizard
       └─ Ollama @ host    → http calls to 127.0.0.1:11434 (not a child)
```

### Internal Go package layout

```
cmd/localfinance/main.go         entry point, lifecycle, signal handling
internal/
  webui/        embeds the React build (`go:embed`), serves at /
  api/          HTTP handlers, route table, JWT middleware
    setup/      first-run wizard endpoints (no auth)
  auth/         users, password hashing, JWT, default-user seed
  data/         GORM models, migrations, repositories
  parse/        PDF→images render + dispatch (PDF: vision; CSV/TXT: text)
  ai/           Ollama HTTP client, model select, probe, parse, narrate
  insights/     rules engine, narrator (lifted from current sophia/insights)
  monthreview/  service + cache (lifted from current sophia/monthreview)
  ollama/       installer wizard helpers, model pull progress streaming
  postgres/     embedded Postgres lifecycle
embeds/
  webui-build/  output of `npm run build` baked into the binary
```

### Mapping from today's services

| Today's service | After |
|---|---|
| `services/iris/server/` (Node/Express proxy) | DELETED. Browser talks directly to Go on `:3001` |
| `services/iris/client/` (React) | Same code, built once and embedded via `go:embed` |
| `services/hermes/` (gateway, currently minimal) | DELETED. Was a placeholder. |
| `services/thesaurus/` (auth + data) | → `internal/auth`, `internal/data`, handlers under `internal/api` |
| `services/sophia/` (AI) | → `internal/ai`, `internal/insights`, `internal/monthreview` |
| `services/logos/` (parsers) | → `internal/parse`, `internal/ai` |

### What's preserved

- Every Postgres table and GORM model, schema-identical
- React app — every component, page, theme, animation, route
- Default-credentials login flow (`local@localfinance.app` / `localfinance`)
- Insights engine, rules, narrator — pure Go, lift-and-shift
- The HTTP wire shapes the React app calls — same URLs, same JSON, just answered by Go directly now
- All current tests live in their packages and keep working
- Phase 0 hallucination eval harness (behind `//go:build eval`)

## Data flows

### Upload (PDF — the long path)

```
Browser POST /api/upload (multipart)
  → api.UploadHandler
      saves file to ~/Library/Application Support/LocalFinance/uploads/
      data.Documents.Create(...)
      go func() {
          parse.ProcessPDF(filePath, userID)
            → ai.RenderPDFToPNGs(filePath)
            → ai.ParseTransactionsFromImages(...)
            → ai.ApplyUserRules(...)
            → ai.AICategorize(...)
            → data.Transactions.BulkInsert(...)
            → monthreview.Service.Generate(...)
            → data.Documents.SetStatus("processed")
      }()
  ← 202 Accepted {document_id}

Browser polls GET /api/v1/documents/:id every 2s
  → data.Documents.Get(id)
  ← 200 {status, transactions[]}
```

What's gone: 4 internal HTTP calls (`Iris→Thesaurus`, `Thesaurus→Logos`, `Logos→Thesaurus×2`, `Logos→Sophia`). All replaced by Go function calls.

### Chat

```
Browser POST /api/v1/chat
  → api.ChatHandler
      ai.AnswerFinancialQuery(query)
        Pass 1 → ollama.Generate(intent_prompt)
        → data.Transactions.Filter(params)
        Pass 2 → ollama.Generate(answer_prompt + data)
      data.Conversations.SaveMessage(...)
  ← 200 {answer, conversation_id, model}
```

### Insights / Month-in-Review

```
Browser GET /api/v1/insights/:userId
  → api.InsightsHandler
      txns := data.Transactions.WindowSince(userID, 90d)
      results := insights.Engine.Run(ctx, txns, dismissedKeys)
      narrative := narrator.Narrate(results)
  ← 200 {insights[], narrative}

Dismiss
Browser POST /api/v1/insights/:userId/dismiss
  → api.DismissHandler
      data.DismissedInsights.Create(userID, key, ruleID)
  ← 201
```

### Couples Mode escape hatch

If multi-machine sync is ever needed, the natural extension point is `internal/data`. Today it's GORM + Postgres directly. A future phase could replace it with a network-aware layer (sync server speaking GORM) without touching `api/`, `parse/`, or `insights/`. The boundary that matters for re-splitting is **the `data` interface**, not HTTP.

## First-run Ollama wizard

State machine:

```
LAUNCH
  ↓
Probe Ollama at 127.0.0.1:11434/api/tags
  ├─ reachable → check installed models
  │     ├─ has any sweet-spot model → READY (auto-select, open dashboard)
  │     └─ no sweet-spot model       → MODEL_PULL screen
  └─ unreachable → INSTALL_OLLAMA screen
```

### Screen 1 — INSTALL_OLLAMA

Copy: "LocalFinance needs Ollama to run AI on your Mac. It's free and open-source."
Primary button: **Download Ollama** → opens `https://ollama.com/download/Ollama-darwin.zip` in the user's browser.
Status text refreshes every 2s: "Waiting for Ollama..." → "Ollama detected ✓".

Backend: `GET /api/setup/ollama-status` returns `{installed: bool, version}`. Frontend polls every 2s; flips to Screen 2 when installed.

### Screen 2 — MODEL_PULL

Recommended model based on RAM (`sysctl hw.memsize`):

| Mac RAM | Recommended model |
|---|---|
| 8 GB | `llama3.2:3b` (~2 GB) |
| 16 GB | `gemma3:4b` (~3 GB, vision-capable) |
| 32 GB+ | `qwen2.5:7b` or `llama3.1:8b` (~5 GB) |

Display: "We'll download `<model>` (~3 GB). Takes 3-10 minutes depending on your connection."
Primary button: **Download Model** → `POST /api/setup/pull-model` body `{model}`.
Backend streams Ollama's `POST /api/pull` progress as SSE to the React page.
React renders progress bar + "Downloading 1.2 GB / 3.0 GB · 12 MB/s".
On completion: 1-second confetti, redirect to login.

Override: small "Choose different model" link below shows all installed models so power users can pick.

### Screen 3 — READY

Backend marks setup complete in `~/Library/Application Support/LocalFinance/setup.json`. Subsequent launches go straight to login.

### Setup endpoints (no auth)

```
GET  /api/setup/state         → {step: "install_ollama"|"pull_model"|"ready", ...}
GET  /api/setup/ollama-status → {installed: bool, version, host_ram_gb}
POST /api/setup/pull-model    → SSE stream of Ollama /api/pull progress
GET  /api/setup/recommended   → {model, size_gb, reason}
```

### Error states

- Ollama detection times out (15s) → "Couldn't reach Ollama. If installed, run `ollama serve` in Terminal."
- Pull fails mid-stream → "Download failed. Check your internet and try again." Retry button.
- Disk full → fatal screen, suggest freeing space and relaunching.

### Why we don't bundle the Ollama binary

- Ollama is ~150 MB and macOS-version sensitive
- Official Ollama installer is signed + simple (60 seconds)
- Maintenance burden of tracking Ollama updates ourselves
- Revisit if non-technical users still get stuck after validation

## Migration path

Each phase keeps `make dev` working end-to-end. Stop at any phase if blocked.

### Phase 0 — Module skeleton (no logic moves)

1. Create `cmd/localfinance/main.go` with router init only
2. Create empty `internal/{api,auth,data,parse,ai,insights,monthreview,ollama,postgres,webui}` packages
3. Add a single root `go.mod`
4. Add `make dev` and `make build-mono` Makefile targets

### Phase 1 — Lift `insights/` and `monthreview/`

1. `git mv services/sophia/insights internal/insights`
2. `git mv services/sophia/monthreview internal/monthreview`
3. Define `internal/data/Repositories` interface for what these packages need (`ListDismissedKeys`, `FetchTransactionsSince`, etc). Implement against today's HTTP-to-Thesaurus client first.
4. Tests move with the packages.

### Phase 2 — Lift Thesaurus into `data/` + `auth/`

1. `git mv services/thesaurus/models internal/data/models`
2. `git mv services/thesaurus/repository internal/data/repository`
3. `git mv services/thesaurus/auth internal/auth`
4. `git mv services/thesaurus/database internal/data`
5. Each Thesaurus HTTP handler becomes a Go function in `internal/api/handlers`.
6. Old Thesaurus container retires from compose; new binary handles `:8001` routes.

### Phase 3 — Lift Sophia + Logos parse paths

1. `git mv services/sophia/ai internal/ai`
2. Lift Sophia handlers into `internal/api/handlers`
3. Lift Logos's `renderPDFToPNGs`, `parseViaSophiaImages`, `parseViaSophia`, `aiCategorize`, `applyUserRules`, `triggerMonthReview` into `internal/parse` and `internal/ai`
4. Sophia + Logos containers retire from compose
5. Update Iris's Express proxy routes to point at the new binary

### Phase 4 — Drop Iris/Node, embed React

1. `cd services/iris/client && npm run build`
2. `internal/webui/embed.go`: `//go:embed build/*`
3. Backend handles all routes the Express proxy used to handle (already true after Phase 3)
4. `services/iris/server/` deleted
5. Browser hits `http://localhost:3001/` directly

After Phase 4: just one binary + Postgres container.

### Phase 5 — Embed Postgres

Use [`embedded-postgres-go`](https://github.com/fergusstrange/embedded-postgres) — pure Go, downloads Postgres binaries on first run, caches under `~/Library/Application Support/LocalFinance/postgres/`. Subsequent launches are instant.

### Phase 6 — First-run Ollama wizard

1. Setup endpoints + state machine in `internal/api/setup/`
2. Three React routes (`/setup/install`, `/setup/pull`)
3. App-startup logic checks `setup.json`, redirects browser if needed
4. Recommended-model logic with RAM detection

### Phase 7 — Final cleanup

- Delete `services/` directory entirely
- Delete `docker-compose.dev.yml` (or keep stripped to just Ollama for an optional Docker dev path)
- Update `Makefile`: `make dev` is `go run ./cmd/localfinance` after a one-time `make webui-build`
- Update `LocalFinance.app` packaging to copy the single binary

## Reversibility

| Decision | Reversibility |
|---|---|
| Single Go binary | Easy. Each `internal/` package can be wrapped in HTTP handler. ~1-2 days per service. |
| Postgres schema unchanged | Trivial. The DB is the same; any future split reads from it directly. |
| `data` interface, not direct GORM imports across packages | Easy. The boundary that matters for sync products is `data`, which is interface-based. |
| Embedded React via `go:embed` | Easy. Replace `go:embed` with `http.FileServer` against a disk path. |
| Embedded Postgres via lib | Trivial. Disable lib, point at host Postgres via env vars. |

What WOULD lock you in (and is therefore explicitly avoided):
- Replacing Postgres with SQLite — no
- Hardcoding "single user, single tenant" assumptions deep in queries — no
- Sharing mutable global state across packages — no

## Error handling

| Today | After |
|---|---|
| Cross-service HTTP errors wrapped, logged, surfaced as JSON | Direct Go function returns. Errors are typed. |
| Each service has its own logger + correlation ID middleware | Single `slog` logger, `correlation_id` field set per-request via `context.Context`. |
| Background goroutines fire-and-forget log-on-fail | Same pattern via `worker.Submit(func)` helper that recovers panics. |
| Postgres unreachable → service won't start | Postgres unreachable → binary won't start. Health-check loop in `cmd/localfinance/main.go` retries 5× then exits with user-readable message. |
| Ollama 404 / unreachable | Same handling preserved (model fallback, transport-error wrapping). |

New failure mode: a panic anywhere kills the app. Mitigation:
- `recover()` in every goroutine helper
- HTTP middleware recovers, logs, returns 500
- Postgres + Ollama health probed continuously; UI banner if either drops mid-session

## Testing

| Layer | Where | Notes |
|---|---|---|
| Pure logic | `internal/insights/`, `internal/monthreview/`, `internal/auth/` | >90% coverage preserved from current state |
| Repositories | `internal/data/repository/*_test.go` | sqlite-in-memory pattern stays |
| HTTP handlers | `internal/api/*_test.go` | `httptest` against the unified router |
| AI parse paths | `internal/ai/*_test.go` | httptest stubs Ollama |
| Phase 0 hallucination eval | `internal/ai/hallucination_eval_test.go` | Behind `//go:build eval`, `make eval-hallucination` |
| End-to-end smoke | `tests/e2e/` | New: Go test that boots binary against a real Postgres + stubbed Ollama, uploads a CSV, asserts transactions land in DB. Replaces bash `tests/e2e/*.sh`. |

The 17 Iris proxy tests retire (no proxy now); replaced by integration tests against the unified router.

## Local dev workflow

```
# First time
brew install ollama postgresql@15
ollama pull gemma3:4b
make webui-build       # one-time React build (or: make webui-watch for hot reload)

# Daily
make dev               # go run ./cmd/localfinance — hits host Postgres + host Ollama
make test              # all Go unit tests
make eval-hallucination  # gated by EVAL_OLLAMA=1, hits live Ollama

# When shipping
make installer         # builds binary, packages .app with embedded Postgres + React
```

For React-only changes: `cd services/iris/client && npm start` runs CRA dev server on `:3000` proxying to Go on `:3001`. Hot reload.

## Packaging

```
LocalFinance.app/
  Contents/
    Info.plist                       (CFBundleIdentifier, version, icon)
    MacOS/
      localfinance                   (one universal2 Go binary, ~30 MB)
    Resources/
      AppIcon.icns
      setup.json                     (default empty; installer marker)
```

No `Resources/bin/`, no five binaries, no Iris-static, no node binary. React build is `go:embed`'d. Postgres is downloaded on first launch (~80 MB) and cached under `~/Library/Application Support/LocalFinance/postgres/`.

Total disk after first run: ~130 MB + the user's chosen Ollama model (~2-5 GB).

### Code signing + notarization (deferred)

Today the .app is unsigned (Gatekeeper requires right-click → Open). Signing is a separate phase:
- Apple Developer account ($99/yr)
- Sign the binary with `codesign`
- Notarize via `notarytool`
- Hardened runtime entitlements

Worth doing before broad distribution but **not in scope for this consolidation work**.

## Out of scope

- Multi-user / multi-tenant support (Phase 3 work, gated by validation)
- Couples Mode sync (Phase 3 work)
- Windows / Linux installer (separate plans)
- Code signing + notarization (separate plan)
- Bundling Ollama binary (revisit if non-tech users struggle after validation)

## Success criteria

- `make dev` runs the new binary against host Postgres + host Ollama in <5 seconds
- `make installer` produces a single `LocalFinance.app` <50 MB
- Fresh Mac: double-click `.app` → through Ollama install → through model pull → at the dashboard within ~5 minutes
- All 40+ existing tests still pass
- `git log` shows a clean phase-by-phase migration each commit of which keeps the system runnable

## Open questions

None as of approval. Future polish flagged in TODOS:
- `[P2]` In-binary Ollama bundling (revisit post-validation)
- `[P2]` Code signing pipeline
- `[P2]` SSE-based live progress for upload (currently 2s polling)
