# Sophia — AI Service (port 8002)

AI-powered insights using local Ollama (configurable model, default `llama3.1:8b`).
Two-pass chat: parse intent -> fetch real data from Thesaurus -> generate answer.

## Build

make build-sophia             # native

## Test

make test-sophia              # unit tests
make test-e2e                 # integration tests (requires Ollama in Docker)
make eval-hallucination       # Phase 0 eval against live Ollama (manual graded)

## Two-Pass Chat Flow

1. Pass 1: User question -> Ollama generates query intent (JSON with filters)
2. Execute: Call Thesaurus /internal/ endpoints with intent params
3. Pass 2: Real data + original question -> Ollama generates natural language answer

## Adding AI Features

1. Add handler in api/handlers/
2. Register route in api/routes.go
3. Use ai/service.go for Ollama calls
4. Always fetch real data from Thesaurus — never hallucinate numbers
5. Timeout for Ollama calls: 120s minimum (covers cold-start of larger models)

## Key Files

- main.go — Gin router, AI service init, Ollama startup probe (fail-fast)
- ai/service.go — Ollama client, two-pass chat logic, merchant cleaning
- ai/probe.go — `Probe(ctx, client, host, model)` — verifies Ollama up + model installed at boot
- api/routes.go — route registration
- api/handlers/chat_handler.go — chat endpoint
- api/handlers/categorize_handler.go — transaction categorization
- api/handlers/insights_handler.go — financial insights (engine + narrator wired here)
- insights/ — deterministic rules engine (rules.go, engine.go, keys.go)
- insights/dismissals.go — `ThesaurusDismissalFetcher` HTTP impl of `DismissalFetcher`
- insights/narrator.go — `Narrator` interface; Phase A template + Phase B LLM polish with allow-set guard
- models/models.go — request/response types
- config/config.go — Ollama host, model name, Thesaurus URL

## Patterns

- Insights handler: 90d window filter is the handler's responsibility (not the engine).
- Narrator: never let unvalidated LLM output reach the user — `validateLLMOutput`
  enforces a numbers/dates/merchants allow-set; on failure we fall back to Phase A.
- LLM polish is opt-in via `INSIGHTS_LLM_POLISH=1`; default is template-only.
- Startup probe can be skipped in tests via `SKIP_OLLAMA_PROBE=1`.
