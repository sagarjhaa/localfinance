# LocalFinance

Privacy-first personal finance system. Upload bank/credit-card statements, chat with your transactions, get rule-based spending insights — all running locally on your Mac. No cloud. No telemetry. No bank-account login sharing.

## What it does

- Upload CSV / PDF statements from any bank
- Statements are parsed locally (no cloud OCR)
- Chat with your transactions in natural language ("how much did I spend on groceries last month?")
- Get rule-based insights: category shifts, new recurring charges, unusual spending, anomalies
- Month-in-Review summary on every statement upload
- All data stays on your Mac in a local Postgres database

## Status

**Current:** Phase 1 in progress (insights engine + macOS installer). See `docs/designs/product-direction.md` and `docs/designs/phase1-implementation-plan.md`.

**Goal:** A `LocalFinance.app` you double-click on macOS that bundles everything except Ollama (which you install separately via brew).

## Architecture

| Service | Port | Responsibility |
|---|---|---|
| **Iris** | 3001 | React UI + Express proxy. ZERO business logic. |
| **Hermes** | 3000 | API Gateway (future routing/aggregation) |
| **Thesaurus** | 8001 | Auth, CRUD, PostgreSQL. Source of truth for all data. |
| **Sophia** | 8002 | AI/Ollama integration. Two-pass chat with real data. |
| **Logos** | 8003 | Document parsing (CSV, PDF). Stateless processor. |

**Infra:** PostgreSQL 15, Redis 7, Ollama (configurable model), MinIO

## Requirements

- macOS (Apple Silicon recommended)
- 16GB RAM minimum (8b model), 32GB+ for 14b models
- [Ollama](https://ollama.com) installed and running
- A model pulled: `ollama pull llama3.1:8b` (default)

## Quick start (local dev)

```bash
# Start everything in Docker
make dev-up

# Open the UI
open http://localhost:3001

# Tear down
make dev-down
```

## Run with a different model

```bash
ollama pull qwen2.5:7b
make dev-up CHAT_MODEL=qwen2.5:7b
```

## Tests

```bash
make test           # unit tests (Go + Node)
make test-e2e       # integration tests (requires make dev-up)
```

## Privacy

- All processing happens on your Mac
- Statements are parsed locally; nothing is sent to any cloud
- LLM inference runs entirely through your local Ollama
- The only network traffic is between your browser and `localhost:3001`

## Project layout

```
services/
  iris/         # React UI + Express proxy
  hermes/       # API gateway
  thesaurus/    # Auth + Postgres CRUD
  sophia/       # AI / Ollama integration
  logos/        # Statement parsers
docs/designs/   # Product direction + implementation plans
evals/          # Hallucination eval fixtures (Phase 0)
```

## License

TBD (see `docs/designs/product-direction.md` — likely MIT or Apache 2.0).
