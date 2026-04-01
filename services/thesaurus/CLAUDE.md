# Thesaurus — Data & Auth Service (port 8001)

Source of truth for all persistent data. Owns PostgreSQL.
Handles: user auth (JWT/Argon2id), accounts, transactions, budgets,
documents, category rules, statement periods.

## Build

make build-thesaurus          # native (for local Docker)
make build-arm64-thesaurus    # cross-compile for Jetson

## Test

make test-thesaurus           # unit tests (auth, models)
make test-e2e                 # integration tests against local Docker

## Deploy

make deploy-thesaurus         # build ARM64 + SCP + restart + verify health

## Adding an Endpoint

1. Add/update model in models/ (with GORM tags, UUID primary key via BeforeCreate)
2. Add handler in api/handlers/
3. Register route in api/routes.go
4. Add to AutoMigrate list in database/database.go
5. User-facing: add under protected group (requires AuthMiddleware)
6. Service-to-service: add under /internal/ group (no auth)

## Key Files

- main.go — Gin router setup, DB init, migrations
- api/routes.go — all route registration
- api/handlers/ — one file per resource (auth, user, account, transaction, etc.)
- models/ — GORM models (User, Account, Transaction, Budget, Document, etc.)
- auth/jwt.go — JWT generation/validation
- auth/password.go — Argon2id hashing
- database/database.go — PostgreSQL connection + AutoMigrate
- middleware/auth_middleware.go — JWT auth enforcement

## Patterns

- User-scoped queries: always filter by user_id from JWT claims
- Bulk operations: /internal/transactions/bulk (Logos calls this, no auth)
- Models use UUID primary keys via BeforeCreate hook
- JWT secret from env var JWT_SECRET
- Config loaded from env vars: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
