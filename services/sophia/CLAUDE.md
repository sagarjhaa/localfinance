# Sophia — AI Service (port 8002)

AI-powered insights using local Ollama (llama3.2:1b).
Two-pass chat: parse intent -> fetch real data from Thesaurus -> generate answer.

## Build

make build-sophia             # native
make build-arm64-sophia       # cross-compile for Jetson

## Test

make test-sophia              # unit tests
make test-e2e                 # integration tests (requires Ollama in Docker)

## Deploy

make deploy-sophia            # build ARM64 + SCP + restart + verify

## Two-Pass Chat Flow

1. Pass 1: User question -> Ollama generates query intent (JSON with filters)
2. Execute: Call Thesaurus /internal/ endpoints with intent params
3. Pass 2: Real data + original question -> Ollama generates natural language answer

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
