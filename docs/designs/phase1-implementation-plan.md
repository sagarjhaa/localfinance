# Phase 1 Implementation Plan — LocalFinance Mac

**Date:** 2026-04-26
**Source:** `/plan-eng-review` output against `docs/designs/product-direction.md`
**Estimated effort:** 5-7 weeks solo with subagent-driven dispatch
**Branch base:** `main`
**Worktrees:** 6 parallel (with sequenced merges for dependencies)

## End-state for Phase 1

- User installs `LocalFinance.app` on macOS (Ollama assumed pre-installed via brew)
- App launches; UI opens at `http://localhost:3001`
- User uploads bank/credit-card statements (CSV/PDF)
- Logos parses → Thesaurus stores transactions
- Sophia analyzes via Ollama (configurable model via `--chat-model` CLI flag)
- User chats with their transactions in natural language
- Insights engine surfaces 5 rule-based detections + LLM-narrated Month in Review on each upload
- All data stays local. Zero cloud.

## Scope decisions locked in

| Decision | Choice | Reason |
|---|---|---|
| Target platform | macOS only | Jetson dropped this iteration |
| Default model | Decided by Phase 0 eval across `llama3.1:8b`, `qwen2.5:7b`, `qwen2.5:14b` | Empirical, not theoretical |
| Phase 0 gate | Slim eval (10 prompts × 3 models) | Right-sized; bigger models drop hallucination risk |
| Architecture | Keep 5 services for Phase 1; monolith refactor in Phase 1.5 | Don't compound installer risk with refactor |
| Insight persistence | Hybrid C — recompute insights, persist only dismissals | Minimal table, supports UX |
| Installer shape | Self-contained `.app`, embeds Postgres, assumes Ollama installed | Low friction for friends-and-family validation |
| Model selection | CLI flag `--chat-model` → env var to Sophia | No in-app picker; ship lean |

## Task 0 — Cleanup

**Worktree:** `cleanup` | **No deps** | **Owner:** main thread | **~3-4h**

- 0.1 Strip Jetson from Makefile (`deploy-*`, `jetson-*`, `build-arm64*` targets)
- 0.2 Strip Jetson section from `CLAUDE.md`
- 0.3 Update auto-memory `project_jetson_deploy.md` → mark deprecated
- 0.4 Rewrite `README.md` + `ROADMAP.md` to match actual Go/Postgres/React stack
- 0.5 Mark `BUSINESS.md` aspirational (banner at top)
- 0.6 Update `docs/designs/product-direction.md` for Mac-only scope

**DoD:** `grep -ri jetson` returns zero hits in source/docs.

## Task 1 — Phase 0 Eval Harness

**Worktree:** `eval` | **No deps** | **Owner:** subagent | **~6-8h**

- 1.1 `evals/insight_hallucination/fixtures.json` — 30 synthetic transactions
- 1.2 `evals/insight_hallucination/prompts.json` — 10 questions + 5 month-in-review prompts
- 1.3 `services/sophia/ai/hallucination_eval_test.go` behind `//go:build eval`
- 1.4 `make eval-hallucination` target running 3 models sequentially
- 1.5 Manual grade run + `evals/insight_hallucination/results-2026-04-26.md`
- 1.6 Pick winner; commit decision in product-direction.md

**DoD:** `make eval-hallucination` runs end-to-end; one model selected as Phase 1 default.

## Task 2 — Thesaurus: Insight Dismissals

**Worktree:** `thesaurus-dismissals` | **No deps** | **Owner:** subagent | **~6-8h**

- 2.1 `services/thesaurus/models/dismissed_insight.go` — UUID, UserID, InsightKey, RuleID, DismissedAt
- 2.2 GORM AutoMigrate + unique index `(user_id, insight_key)`
- 2.3 `services/thesaurus/repository/dismissed_insight_repo.go`
- 2.4 `services/thesaurus/api/handlers/internal/dismissed_insights.go`
- 2.5 Wire `/internal/` routes (no auth middleware)
- 2.6 Repo tests via testcontainers Postgres

**DoD:** `go test ./services/thesaurus/...` passes.

## Task 3 — Sophia: Insights Engine

**Worktree:** `sophia-insights` | **Blocked by Task 2** | **Owner:** subagent dispatch (split) | **~25-35h**

- 3.1 `services/sophia/insights/types.go`
- 3.2 `services/sophia/insights/rules.go` — 5 detector funcs
- 3.3 `services/sophia/insights/engine.go` — orchestrator with dismissal filter
- 3.4 `services/sophia/insights/keys.go` — deterministic key generator
- 3.5 `services/sophia/insights/keys_stability_test.go` — assert determinism
- 3.6 `services/sophia/insights/narrator.go` — single-call structured-output narration
- 3.7 `services/sophia/insights/narrator_test.go` — adversarial number/merchant guard
- 3.8 `services/sophia/insights/rules_test.go` — table-driven, 20 fixtures/rule
- 3.9 Replace stub at `services/sophia/api/handlers/insights_handler.go:78`
- 3.10 Sophia startup probe: ping Ollama for model, fail-fast

**DoD:** >90% coverage on `insights/`; e2e: upload → GET /insights returns insights.

## Task 4 — Month in Review

**Worktree:** `monthreview` | **Blocked by Task 3** | **Owner:** subagent | **~8-12h**

- 4.1 `services/sophia/monthreview/handler.go` — internal generate, public fetch
- 4.2 Logos calls `Sophia /internal/month-review/generate` after parse
- 4.3 Sophia caches reviews keyed by `(user_id, period)`
- 4.4 Unit + e2e tests

**DoD:** Upload statement → month-review available within 30s.

## Task 5 — Iris UI

**Worktree:** `iris-insights` | **Blocked by Tasks 3, 4** | **Owner:** subagent | **~10-15h**

- 5.1 Insights page — list with finding/evidence/confidence/dismiss
- 5.2 Month in Review page — narrative + section per insight
- 5.3 "View month in review" CTA on upload-success
- 5.4 Iris proxy routes (no business logic)

**DoD:** `make dev-up` → upload → insights → dismiss → reload → dismissal sticks.

## Task 6 — macOS Installer ✅

**Worktree:** `installer` | **Blocked by Tasks 2-5 functional** | **Owner:** subagent dispatch (split) | **~25-35h**

- 6.1 Audit Redis usage; decide drop or embed
- 6.2 `installer/build.sh` — produces `LocalFinance.app` (universal2 binaries, Iris static, embedded Postgres)
- 6.3 `installer/launcher.go` — entry binary: starts Postgres + 5 services, manages lifecycle, sets `OLLAMA_KEEP_ALIVE=24h`
- 6.4 Ollama detection at startup; clear error + brew hint if missing
- 6.5 `--chat-model` CLI flag → env var to Sophia
- 6.6 Smoke test on clean Mac
- 6.7 Document RAM tier → model recommendation table

**DoD:** Fresh Mac with Ollama → double-click `.app` → upload → insights generate.

## Worktree dependency graph

```
        ┌─ Task 0 (cleanup) ──────┐
main ───┤                         ├─→ all merged → Task 1.5 monolith refactor (separate plan, gates validation)
        ├─ Task 1 (eval) ─────────┤
        │                         │
        ├─ Task 2 (thesaurus) ──→ Task 3 (sophia) ──→ Task 4 (monthreview) ──→ Task 5 (iris) ──→ Task 6 (installer)
        │                                                                                       ↗
        └─ Task 6.1 Redis audit ───────────────────────────────────────────────────────────────┘
```

## Top 3 risks

1. **Phase 0 eval picks 14b but most Macs are 16GB.** Mitigation: document RAM tiers; default to 7b/8b, offer 14b opt-in.
2. **Embedded Postgres + signed `.app` cert run can be painful.** Mitigation: test on a second Mac early.
3. **5 services as `os/exec` children in `.app` — orphan-process risk on crash.** Mitigation: launcher kills children on SIGTERM; documented as known issue until Phase 1.5 monolith refactor.

## Success metrics (Phase 1 exit)

- `make dev-up` works on a fresh Mac clone
- `LocalFinance.app` installs and runs on a second Mac
- Your own 90-day data produces ≥3 unknown insights
- README accurately describes the project
- `grep -ri jetson` finds zero hits in source

## Out of scope for Phase 1

- Windows installer
- Linux installer
- Jetson deployment (removed)
- In-app model picker (CLI flag only)
- Multi-user data isolation
- Subscription Auditor (Phase 2)
- Fraud / Savings (Phase 3, validation-gated)
- Monolith refactor (Phase 1.5, separate plan)

## Task 6 — Build notes

Decisions made while shipping the macOS installer:

- **Postgres: not embedded** in Phase 1. The launcher resolves `initdb`,
  `pg_ctl`, `postgres`, `psql` from the runtime user's `PATH`. The Homebrew
  binaries hard-code dylib install_names (icu4c, openssl@1.1, krb5) that
  break when copied into a bundle — and macOS strips `DYLD_*` env vars when
  posix_spawn'ing into hardened-runtime binaries, so we can't paper over it
  with env vars from an unsigned launcher. The proper fix
  (`install_name_tool -change` plus shipping matching dylibs) is tracked as
  a Phase 1.5 packaging task. End users run `brew install postgresql@15`
  once before first launch. Documented in `installer/POSTGRES.md`.
  Opt-in flag `LOCALFINANCE_BUNDLE_POSTGRES=1` exists for experimenting.
- **Bundle size:** ~165 MB without bundled Postgres, ~275 MB with
  (LocalFinance.app/Contents/Resources contains the 4 Go binaries, Iris
  static, iris-server.js, a Node 18 binary).
- **Universal2:** off by default to keep build fast. Set `UNIVERSAL2=1` to
  build amd64 + arm64 and lipo-fuse. Falls back to native arch with a
  warning if `lipo` is missing.
- **Postgres port:** 15432 (avoids fighting brew's own postgres on 5432).
- **Service start order:** thesaurus → sophia → logos → hermes → iris,
  each gated on the previous service's `/health` endpoint. Iris readiness
  triggers the browser open (`http://localhost:3001`).
- **Ollama policy:** hard-exit if `/api/tags` fails or the requested model
  is missing. The error message points at `brew install ollama` and
  `ollama pull <model>`. No fall-through, no implicit auto-pull.
- **Crash handling:** any child exit triggers full teardown (10s SIGTERM
  grace, then SIGKILL). No restart loop in Phase 1 — surfaces the failing
  service's tail of log to stderr.
- **Redis:** confirmed unused (per `redis-audit-2026-04-26.md`); not
  started by the launcher.
- **MinIO:** Logos imports a MinIO client but `main.go` never instantiates
  it. Skipped — uploads use the local filesystem (`TEMP_DIR` env points at
  `~/Library/Application Support/LocalFinance/uploads`).
- **Go linker quirk:** `-ldflags=-linkmode=external` is required on this
  Mac to work around a Go 1.22 / new macOS SDK issue (binaries built with
  internal linkmode crash with "missing LC_UUID load command" on `go test`
  and on Gatekeeper). Baked into `build.sh` and `make installer-test`.

**Verified locally:** `make installer-clean && make installer` produces
`dist/LocalFinance.app/`; running the launcher with `--skip-browser` brings
all 5 services healthy in ~7 seconds; `curl localhost:3001/api/health`
returns 200; SIGTERM cleans up every child including the Postgres process.
`make installer-test` passes 8 unit tests for the launcher's testable
seams (`findResources`, `freePort`, `waitForHealthy`, `parseArgs`,
`hasModel`, `ollamaTags`).

---

**Update 2026-04-27**: services collapsed to single binary; see
`docs/superpowers/plans/2026-04-27-service-consolidation.md` for the
migration that supersedes the multi-service installer task.
