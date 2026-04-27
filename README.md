# LocalFinance

Privacy-first personal finance for macOS. Upload bank/credit-card statements;
chat with your transactions; rule-based spending insights — all running locally
on your Mac. No cloud. No telemetry.

## Quick start (development)

```
brew install ollama postgresql@15
ollama serve &
ollama pull gemma3:4b
docker compose -f docker-compose.dev.yml up -d   # or run host Postgres on :5432
make dev
```

UI opens at http://localhost:3001. Default credentials are seeded on first run.

Default DB env (override as needed):
```
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=devpass DB_NAME=localfinance
```

If `DB_HOST` is unset, the binary boots an embedded Postgres under
`~/Library/Application Support/LocalFinance/postgres/` instead.

## Quick start (.app for end users)

```
make installer
open dist/LocalFinance.app
```

The first launch walks the user through Ollama install + model download.

## Architecture

Single Go binary (`cmd/localfinance`) serves the React UI, owns all data
and AI logic, and embeds Postgres. Internal packages:

- `internal/api` — HTTP routes (Gin)
- `internal/auth` — JWT, password hashing, default-user seed
- `internal/data` — GORM models, repositories, embedded Postgres lifecycle
- `internal/parse` — PDF/CSV ingest pipeline
- `internal/ai` — Ollama client + chat/insight logic
- `internal/insights` — rules engine + narrator
- `internal/monthreview` — period summary cache
- `internal/ollama` — host Ollama probes + model pull SSE
- `internal/webui` — `go:embed` React build

See `docs/superpowers/specs/2026-04-27-service-consolidation-design.md`
for the full design.

## Tests

```
make test                # unit tests
make test-e2e            # integration (requires make dev running)
make eval-hallucination  # Phase 0 hallucination eval (manual graded)
```

## License

TBD (see `docs/designs/product-direction.md` — likely MIT or Apache 2.0).
