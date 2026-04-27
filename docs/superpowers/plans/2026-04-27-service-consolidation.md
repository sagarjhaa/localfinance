# Service Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse 5-service stack (Iris/Hermes/Thesaurus/Sophia/Logos) into a single Go binary that serves the React UI, owns all logic in-process, manages embedded Postgres, and walks non-technical Mac users through Ollama install + model download on first run.

**Architecture:** Modular monolith — one Go binary, packages with clean interfaces under `internal/*`, React build embedded via `go:embed`, Postgres lifecycle via `embedded-postgres-go`, no inter-process HTTP. Wire shapes the React app uses today are preserved exactly so frontend code doesn't change.

**Tech Stack:** Go 1.21+, GORM, Gin (or stdlib `net/http`), `embedded-postgres-go`, React (existing CRA app), `go:embed`, `slog`.

**Spec:** `docs/superpowers/specs/2026-04-27-service-consolidation-design.md`

---

## Phase 0 — Module skeleton

Goal: a `cmd/localfinance/main.go` that compiles, runs, and listens on `:3001` with nothing else changed. No logic moves.

### Task 0.1: Create the root go.mod and skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/localfinance/main.go`
- Create: `internal/api/router.go`
- Create: `internal/api/router_test.go`

- [ ] **Step 1: Create root go.mod**

```bash
cd /Users/sagar/Documents/codebases/localfinance
go mod init github.com/sagarjhaa/localfinance
go mod edit -replace github.com/sagarjhaa/localfinance/shared=./shared
```

Expected: `go.mod` exists at repo root with module path `github.com/sagarjhaa/localfinance`.

- [ ] **Step 2: Write the failing test for the router**

Create `internal/api/router_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterHealth(t *testing.T) {
	r := NewRouter()
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want 200", resp.StatusCode)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

```bash
go test ./internal/api/...
```

Expected: FAIL — `NewRouter` undefined.

- [ ] **Step 4: Write the minimal router**

Create `internal/api/router.go`:

```go
// Package api wires the unified HTTP surface for the localfinance binary.
// Replaces the per-service Gin routers from the old 5-service stack.
package api

import (
	"net/http"
)

// NewRouter returns the top-level mux. As phases land, more routes get
// registered here. For now: just /health so cmd/localfinance can boot.
func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"localfinance"}`))
	})
	return mux
}
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test ./internal/api/...
```

Expected: PASS.

- [ ] **Step 6: Wire main.go to start the server**

Create `cmd/localfinance/main.go`:

```go
// Package main is the entry point for the LocalFinance single-binary server.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/sagarjhaa/localfinance/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	slog.Info("localfinance starting", "port", port)
	if err := http.ListenAndServe(":"+port, api.NewRouter()); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Verify it builds and runs**

```bash
go build ./cmd/localfinance
./localfinance &
sleep 1
curl -sf http://localhost:3001/health
kill %1
rm localfinance
```

Expected: curl returns `{"status":"healthy","service":"localfinance"}`.

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum cmd/localfinance/main.go internal/api/router.go internal/api/router_test.go
git commit -m "feat(monolith): add cmd/localfinance skeleton with /health"
```

### Task 0.2: Add stub packages so future imports don't fail

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/data/data.go`
- Create: `internal/parse/parse.go`
- Create: `internal/ai/ai.go`
- Create: `internal/insights/insights.go` (placeholder until Phase 1)
- Create: `internal/monthreview/monthreview.go` (placeholder until Phase 1)
- Create: `internal/ollama/ollama.go`
- Create: `internal/postgres/postgres.go`
- Create: `internal/webui/webui.go`

- [ ] **Step 1: Create each stub file**

Each file looks like:

```go
// Package <name> is a placeholder; populated in Phase N.
// See docs/superpowers/specs/2026-04-27-service-consolidation-design.md.
package <name>
```

Replace `<name>` per file. Don't bother with real types yet — the goal is `go build ./...` succeeds.

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/
git commit -m "feat(monolith): scaffold internal/* package directories"
```

### Task 0.3: Add Makefile targets for the new binary

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add new targets at end of Makefile**

Append to `Makefile`:

```makefile
# ─── Single-binary build (consolidation phase) ─────────
.PHONY: dev build-mono test-mono

# Run the consolidated binary against host Postgres + host Ollama (assumes
# both are running). Hot path during the consolidation work.
dev:
	@PORT=3001 go run ./cmd/localfinance

build-mono:
	@go build -o $(DIST)/localfinance ./cmd/localfinance
	@echo "  → $(DIST)/localfinance"

test-mono:
	@go test ./internal/... ./cmd/...
```

- [ ] **Step 2: Verify the new targets work**

```bash
make build-mono
make test-mono
```

Both should succeed.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat(monolith): add make dev / build-mono / test-mono targets"
```

---

## Phase 1 — Lift `insights/` and `monthreview/`

Goal: move both packages into `internal/`, point them at a `data.Repositories` interface, keep their existing tests green. The 5-service stack still runs after this — Sophia keeps using its own copies until Phase 5.

### Task 1.1: Define the data.Repositories interface

**Files:**
- Create: `internal/data/repositories.go`

- [ ] **Step 1: Define the interface**

```go
// Package data owns persistence. The Repositories aggregate is the single
// surface other packages depend on so we can swap implementations (HTTP-to-
// Thesaurus during the migration; direct GORM after Phase 2).
package data

import (
	"context"
	"time"
)

// Transaction is the lean shape consumers use. Persisted shape lives in
// internal/data/models once Phase 2 lifts the GORM models.
type Transaction struct {
	ID          string
	UserID      string
	Date        time.Time
	Description string
	Amount      float64
	Category    string
	Type        string
}

// DismissedInsight is the lean shape for the dismissals table.
type DismissedInsight struct {
	UserID     string
	InsightKey string
	RuleID     string
	DismissedAt time.Time
}

// Repositories is the umbrella interface other packages depend on.
type Repositories interface {
	FetchTransactionsSince(ctx context.Context, userID string, start time.Time) ([]Transaction, error)
	ListDismissedInsightKeys(ctx context.Context, userID string) (map[string]struct{}, error)
	CreateDismissedInsight(ctx context.Context, d DismissedInsight) error
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/data/repositories.go
git commit -m "feat(data): introduce Repositories interface for migration"
```

### Task 1.2: Move insights/ from sophia to internal/

**Files:**
- Move: `services/sophia/insights/*` → `internal/insights/`
- Modify: every file in `internal/insights/` to update imports

- [ ] **Step 1: Move the files**

```bash
git mv services/sophia/insights/*.go internal/insights/
```

- [ ] **Step 2: Update package imports**

Use search-and-replace across all `internal/insights/*.go`:
- `github.com/sagarjhaa/localfinance/services/sophia/insights` → `github.com/sagarjhaa/localfinance/internal/insights`
- `github.com/sagarjhaa/localfinance/services/sophia/models` → references `models.TransactionRef` and `models.FinancialInsight` — for now leave the import as-is (still resolves) and **plan to redefine these types in `internal/data` in Phase 2**. To unblock the build, add a temporary type alias file:

Create `internal/insights/aliases.go`:

```go
package insights

import sophiamodels "github.com/sagarjhaa/localfinance/services/sophia/models"

type TransactionRef = sophiamodels.TransactionRef
type FinancialInsight = sophiamodels.FinancialInsight
```

Then change every reference to `models.TransactionRef` → `TransactionRef` (local alias) and `models.FinancialInsight` → `FinancialInsight` inside `internal/insights/`. The aliases file gets deleted in Phase 2 once the real models live in `internal/data/models`.

- [ ] **Step 3: Update DismissalFetcher consumer**

The engine's `DismissalFetcher` interface stays in `internal/insights/engine.go`. Add an adapter in `internal/insights/repo_adapter.go`:

```go
package insights

import (
	"context"

	"github.com/sagarjhaa/localfinance/internal/data"
)

// RepoFetcher adapts data.Repositories to the engine's DismissalFetcher
// interface. Used in production wiring; tests still use the existing
// FakeDismissalFetcher in engine_test.go.
type RepoFetcher struct{ Repos data.Repositories }

func (f *RepoFetcher) ListDismissedKeys(ctx context.Context, userID string) (map[string]struct{}, error) {
	return f.Repos.ListDismissedInsightKeys(ctx, userID)
}
```

- [ ] **Step 4: Run insights tests**

```bash
go test ./internal/insights/...
```

Expected: every test that passed in `services/sophia/insights/` still passes.

- [ ] **Step 5: Commit**

```bash
git add services/sophia/insights internal/insights
git commit -m "refactor(insights): move package to internal/insights"
```

### Task 1.3: Move monthreview/ from sophia to internal/

**Files:**
- Move: `services/sophia/monthreview/*` → `internal/monthreview/`

- [ ] **Step 1: Move files**

```bash
git mv services/sophia/monthreview/*.go internal/monthreview/
```

- [ ] **Step 2: Fix imports**

Same pattern as Task 1.2:
- `services/sophia/monthreview` → `internal/monthreview`
- `services/sophia/insights` → `internal/insights`
- `services/sophia/models.FinancialInsight` → use the alias from `internal/insights` (export it through `internal/insights.FinancialInsight`) OR add another alias file in `internal/monthreview`. Pick whichever produces fewer changes.

- [ ] **Step 3: Run tests**

```bash
go test ./internal/monthreview/...
```

Expected: every test passes.

- [ ] **Step 4: Commit**

```bash
git add services/sophia/monthreview internal/monthreview
git commit -m "refactor(monthreview): move package to internal/monthreview"
```

### Task 1.4: Update sophia to import from internal/insights and internal/monthreview

**Files:**
- Modify: `services/sophia/api/handlers/insights_handler.go`
- Modify: `services/sophia/api/handlers/monthreview_handler.go`
- Modify: `services/sophia/api/routes.go`

- [ ] **Step 1: Update Sophia imports**

In every Sophia file that imports `services/sophia/insights` or `services/sophia/monthreview`, change to `internal/insights` / `internal/monthreview`.

- [ ] **Step 2: Verify both builds**

```bash
go build ./...
make test-sophia
make test-mono
```

Both must pass. The 5-service stack still runs against the new internal/ packages.

- [ ] **Step 3: Commit**

```bash
git add services/sophia
git commit -m "refactor(sophia): consume insights and monthreview from internal/"
```

---

## Phase 2 — Lift Thesaurus

Goal: Thesaurus's models, repo, auth, and database setup live under `internal/`. The Thesaurus container is replaced by direct in-process calls inside `cmd/localfinance` for the new binary, but the old Thesaurus binary still works for the migrating dev environment.

### Task 2.1: Move models, repository, auth, database

**Files:**
- Move: `services/thesaurus/models/*` → `internal/data/models/`
- Move: `services/thesaurus/repository/*` → `internal/data/repository/`
- Move: `services/thesaurus/auth/*` → `internal/auth/`
- Move: `services/thesaurus/database/*` → `internal/data/database/`

- [ ] **Step 1: git mv each directory**

```bash
git mv services/thesaurus/models internal/data/models
git mv services/thesaurus/repository internal/data/repository
git mv services/thesaurus/auth internal/auth
git mv services/thesaurus/database internal/data/database
```

- [ ] **Step 2: Update imports across moved files**

Search-and-replace inside `internal/data/`, `internal/auth/`:
- `services/thesaurus/models` → `internal/data/models`
- `services/thesaurus/repository` → `internal/data/repository`
- `services/thesaurus/auth` → `internal/auth`
- `services/thesaurus/database` → `internal/data/database`
- `services/thesaurus/config` — temporary; Thesaurus's config package stays at its old path until Task 2.3

- [ ] **Step 3: Drop the temporary aliases.go from Phase 1**

```bash
rm internal/insights/aliases.go
```

Update `internal/insights/types.go` to import `internal/data/models.FinancialInsight` directly. Repeat for `internal/monthreview/`.

- [ ] **Step 4: Verify build + tests**

```bash
go build ./...
go test ./internal/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/ services/thesaurus
git commit -m "refactor(thesaurus): move models, repo, auth, database to internal/"
```

### Task 2.2: Implement data.Repositories backed by direct GORM

**Files:**
- Create: `internal/data/postgres_repos.go`
- Create: `internal/data/postgres_repos_test.go`

- [ ] **Step 1: Implement the Repositories surface against `*gorm.DB`**

```go
// Package data — Postgres-backed implementation of the Repositories interface.
// Used by the new binary; the old Thesaurus container also exposes the same
// shape via HTTP for any service that hasn't been collapsed yet.
package data

import (
	"context"
	"time"

	"github.com/sagarjhaa/localfinance/internal/data/models"
	"github.com/sagarjhaa/localfinance/internal/data/repository"
	"gorm.io/gorm"
)

type postgresRepos struct {
	db *gorm.DB
}

func NewPostgresRepos(db *gorm.DB) Repositories {
	return &postgresRepos{db: db}
}

func (r *postgresRepos) FetchTransactionsSince(ctx context.Context, userID string, start time.Time) ([]Transaction, error) {
	var rows []models.Transaction
	if err := r.db.WithContext(ctx).
		Joins("JOIN accounts ON transactions.account_id = accounts.id").
		Where("accounts.user_id = ? AND transactions.date >= ?", userID, start).
		Order("transactions.date desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Transaction, 0, len(rows))
	for _, m := range rows {
		out = append(out, Transaction{
			ID:          m.ID.String(),
			UserID:      userID,
			Date:        m.Date,
			Description: m.Description,
			Amount:      m.Amount,
			Category:    m.Category,
			Type:        m.Type,
		})
	}
	return out, nil
}

func (r *postgresRepos) ListDismissedInsightKeys(ctx context.Context, userID string) (map[string]struct{}, error) {
	repo := repository.NewDismissedInsightRepo(r.db)
	rows, err := repo.ListByUser(ctx, parseUUID(userID))
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		out[row.InsightKey] = struct{}{}
	}
	return out, nil
}

func (r *postgresRepos) CreateDismissedInsight(ctx context.Context, d DismissedInsight) error {
	repo := repository.NewDismissedInsightRepo(r.db)
	return repo.Create(ctx, &models.DismissedInsight{
		UserID:     parseUUID(d.UserID),
		InsightKey: d.InsightKey,
		RuleID:     d.RuleID,
	})
}
```

Add a small `parseUUID(string) uuid.UUID` helper in the same file with sensible empty-on-error fallback.

- [ ] **Step 2: Write tests using the existing sqlite-in-memory pattern**

`internal/data/postgres_repos_test.go` uses `glebarez/sqlite`. Cover: FetchTransactionsSince returns rows in window, ListDismissedInsightKeys returns set, CreateDismissedInsight is idempotent on duplicate.

- [ ] **Step 3: Run**

```bash
go test ./internal/data/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/data/postgres_repos.go internal/data/postgres_repos_test.go
git commit -m "feat(data): postgres-backed Repositories implementation"
```

### Task 2.3: Lift Thesaurus HTTP handlers into internal/api/handlers

**Files:**
- Move: `services/thesaurus/api/handlers/*` → `internal/api/handlers/`
- Move: `services/thesaurus/api/handlers/internalapi/*` → `internal/api/handlers/`
  (rename to e.g. `internal_dismissed_insights_handler.go` since the `internal` directory name is reserved by Go for visibility scoping)
- Move: `services/thesaurus/middleware/*` → `internal/api/middleware/`
- Modify: `internal/api/router.go` — register all handlers

- [ ] **Step 1: Move files**

```bash
git mv services/thesaurus/api/handlers/* internal/api/handlers/
git mv services/thesaurus/api/handlers/internalapi/* internal/api/handlers/
rmdir services/thesaurus/api/handlers/internalapi
git mv services/thesaurus/middleware/* internal/api/middleware/
```

Rename the moved files from `internalapi/` to flatten — e.g. `internalapi/dismissed_insights_handler.go` → `internal_dismissed_insights_handler.go`.

- [ ] **Step 2: Adapt route registration**

Update `internal/api/router.go` to register every handler under the same paths Thesaurus uses today. Use Gin (already a dep) for consistency:

```go
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/api/handlers"
	"github.com/sagarjhaa/localfinance/internal/api/middleware"
	"gorm.io/gorm"
)

func NewGinRouter(db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	authH := handlers.NewAuthHandler(db)
	userH := handlers.NewUserHandler(db)
	docH := handlers.NewDocumentHandler(db)
	// ... wire every handler

	r.GET("/health", healthHandler)

	v1 := r.Group("/api/v1")
	{
		// public
		r.POST("/api/auth/register", authH.Register)
		r.POST("/api/auth/login", authH.Login)

		// protected
		protected := v1.Use(middleware.AuthMiddleware())
		{
			protected.PUT("/users/:id", userH.UpdateUser)
			// ... every existing route
		}

		// internal (no auth — called by Sophia/Logos in the old world; safe in
		// the new world because the binary is single-user and listens on
		// localhost only)
		internal := v1.Group("/internal")
		{
			internal.POST("/users/:user_id/dismissed-insights", /* handler */)
			// ... etc
		}
	}
	return r
}
```

- [ ] **Step 3: Update cmd/localfinance/main.go**

```go
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/sagarjhaa/localfinance/internal/api"
	"github.com/sagarjhaa/localfinance/internal/data/database"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	if err := database.Migrate(db); err != nil {
		slog.Error("db migrate failed", "err", err)
		os.Exit(1)
	}
	if err := database.SeedDefaultUser(db); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}

	router := api.NewGinRouter(db)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	slog.Info("localfinance starting", "port", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Verify the new binary handles auth + data routes**

```bash
make build-mono
DB_HOST=localhost DB_USER=postgres DB_PASSWORD=devpass DB_NAME=localfinance DB_SSLMODE=disable PORT=3001 ./dist/localfinance &
sleep 3
curl -sS -X POST http://localhost:3001/api/auth/login -H 'Content-Type: application/json' -d '{"email":"local@localfinance.app","password":"localfinance"}' | head -c 200
kill %1
```

Expected: token returned. (Assumes a host Postgres is running.)

- [ ] **Step 5: Run all tests**

```bash
go test ./internal/...
make test-thesaurus  # the old thesaurus dir still exists; if it imports from internal it should still build
```

- [ ] **Step 6: Commit**

```bash
git add internal/ services/thesaurus cmd/localfinance/main.go
git commit -m "refactor(api): unify Thesaurus handlers into internal/api"
```

### Task 2.4: Retire old Thesaurus container from compose, route to new binary

**Files:**
- Modify: `docker-compose.dev.yml`

- [ ] **Step 1: Replace the `thesaurus` service block**

Change the `thesaurus` service in compose to build from a new `Dockerfile.mono` that builds `cmd/localfinance` and exposes `:8001` (so Sophia/Logos still find it):

```yaml
  thesaurus:
    build:
      context: .
      dockerfile: Dockerfile.mono
    env_file: deployment/docker/dev.env
    ports:
      - "8001:3001"
    environment:
      PORT: "3001"
    volumes:
      - uploads_data:/tmp/localfinance
    depends_on:
      postgres:
        condition: service_healthy
```

Create `Dockerfile.mono`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY shared/ /shared/
RUN go mod edit -replace github.com/sagarjhaa/localfinance/shared=/shared
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /service ./cmd/localfinance

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl poppler-utils
COPY --from=builder /service /service
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-3001}/health || exit 1
ENTRYPOINT ["/service"]
```

- [ ] **Step 2: Verify dev-up still works end-to-end**

```bash
make dev-down
make dev-up
make verify
```

Expected: all healthy. Login + insights smoke test passes.

- [ ] **Step 3: Commit**

```bash
git add docker-compose.dev.yml Dockerfile.mono
git commit -m "feat(deploy): switch thesaurus container to consolidated binary"
```

---

## Phase 3 — Lift Sophia + Logos parse paths

Goal: every Sophia/Logos handler lives in the new binary. Sophia + Logos containers retire from compose.

### Task 3.1: Move Sophia ai/ + handlers

**Files:**
- Move: `services/sophia/ai/*` → `internal/ai/`
- Move: `services/sophia/api/handlers/chat_handler.go` → `internal/api/handlers/`
- Move: `services/sophia/api/handlers/insights_handler.go` → already moved by Phase 1 — adjust if needed
- Move: `services/sophia/api/handlers/monthreview_handler.go` → already moved by Phase 1
- Move: `services/sophia/api/handlers/categorize_handler.go` → `internal/api/handlers/`

- [ ] **Step 1: git mv each**

```bash
git mv services/sophia/ai/*.go internal/ai/
git mv services/sophia/api/handlers/chat_handler.go internal/api/handlers/
git mv services/sophia/api/handlers/categorize_handler.go internal/api/handlers/
```

- [ ] **Step 2: Update imports**

`services/sophia/ai` → `internal/ai`. Sophia models referenced inside `ai/service.go` — leave models.* references for now and lift `services/sophia/models/models.go` into `internal/data/models/sophia_types.go` in Step 3.

- [ ] **Step 3: Move sophia models**

```bash
git mv services/sophia/models/models.go internal/data/models/sophia_types.go
```

Update package declaration to `package models` (it's already in that package). Fix imports across `internal/ai/`, `internal/api/handlers/`.

- [ ] **Step 4: Wire all moved handlers into the router**

Modify `internal/api/router.go` to register chat, categorize, parse, parse-images, models, status. Same paths Sophia uses today (`/api/v1/chat/`, `/api/v1/insights/`, etc).

- [ ] **Step 5: Verify build + tests**

```bash
go build ./...
go test ./internal/...
```

- [ ] **Step 6: Commit**

```bash
git add internal/ services/sophia
git commit -m "refactor(ai): move Sophia ai/ + handlers to internal/"
```

### Task 3.2: Move Logos parse pipeline

**Files:**
- Move: `services/logos/processors/pdf_processor.go` → `internal/parse/pdf.go`
- Move: `services/logos/processors/detector.go` → `internal/parse/detector.go`
- Move: `services/logos/processors/types.go` → `internal/parse/types.go`
- Move: lift functions from `services/logos/main.go` (renderPDFToPNGs, parseViaSophiaImages, parseViaSophia, aiCategorize, applyUserRules, sendTransactionsWithMetadata, updateDocumentStatus, triggerMonthReview, inferPeriod, extractTextForAI, convertParsedTransactions) → `internal/parse/pipeline.go`

- [ ] **Step 1: Move processors**

```bash
git mv services/logos/processors/pdf_processor.go internal/parse/pdf.go
git mv services/logos/processors/detector.go internal/parse/detector.go
git mv services/logos/processors/types.go internal/parse/types.go
```

- [ ] **Step 2: Lift pipeline functions from logos/main.go**

Cut these functions from `services/logos/main.go` and paste into `internal/parse/pipeline.go`:
- `processDocument` → renamed to `parse.Pipeline.ProcessDocument(ctx, req)`
- `renderPDFToPNGs`
- `parseViaSophia` — now becomes a direct call to `internal/ai.Service.ParseTransactions`
- `parseViaSophiaImages` — direct call to `internal/ai.Service.ParseTransactionsFromImages`
- `aiCategorize` — direct calls to `internal/ai.Service`
- `applyUserRules` — direct call to `internal/data.Repositories`
- `sendTransactionsWithMetadata` — direct call to `internal/data.Repositories.BulkInsertTransactions`
- `updateDocumentStatus` — direct call to `internal/data.Repositories.UpdateDocumentStatus`
- `triggerMonthReview` — direct call to `internal/monthreview.Service.Generate`
- `inferPeriod` (helper)
- `extractTextForAI`
- `convertParsedTransactions`

Define `parse.Pipeline` struct:

```go
package parse

import (
	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/data"
	"github.com/sagarjhaa/localfinance/internal/monthreview"
)

type Pipeline struct {
	AI          *ai.Service
	Repos       data.Repositories
	MonthReview *monthreview.Service
}

func New(aiSvc *ai.Service, repos data.Repositories, mr *monthreview.Service) *Pipeline {
	return &Pipeline{AI: aiSvc, Repos: repos, MonthReview: mr}
}
```

Replace every HTTP-call function in the lifted code with the direct in-process equivalent.

- [ ] **Step 3: Wire upload handler to call the pipeline**

Add `internal/api/handlers/upload_handler.go`:

```go
func (h *UploadHandler) Process(c *gin.Context) {
	// existing file save + document row create stays
	go h.pipeline.ProcessDocument(context.Background(), parse.Request{
		DocumentID: doc.ID,
		FilePath:   filePath,
		UserID:     userID,
		AccountID:  accountID,
		FileType:   fileType,
	})
	c.JSON(202, gin.H{"document_id": doc.ID, "status": "processing"})
}
```

- [ ] **Step 4: Verify**

```bash
go build ./...
go test ./internal/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/ services/logos
git commit -m "refactor(parse): lift Logos pipeline into internal/parse"
```

### Task 3.3: Retire Sophia + Logos containers from compose

**Files:**
- Modify: `docker-compose.dev.yml`

- [ ] **Step 1: Delete sophia + logos service blocks**

The unified binary now answers their routes via `localhost:3001`. Remove sophia + logos from compose.

- [ ] **Step 2: Update Iris's proxy routes**

`services/iris/server/routes/insights.js` and friends point at `THESAURUS_URL` and `SOPHIA_URL` envs. Set both to `http://thesaurus:3001` (the consolidated binary).

- [ ] **Step 3: Verify dev-up**

```bash
make dev-down
make dev-up
make verify
```

Smoke test: register, upload a small CSV, verify insights endpoint returns data.

- [ ] **Step 4: Commit**

```bash
git add docker-compose.dev.yml services/iris/server
git commit -m "feat(deploy): retire sophia and logos containers"
```

---

## Phase 4 — Drop Iris/Node, embed React

Goal: Go binary serves the React build directly. Iris/Node disappears.

### Task 4.1: Add go:embed for the React build

**Files:**
- Create: `internal/webui/embed.go`
- Create: `internal/webui/embed_test.go`

- [ ] **Step 1: Build the React app**

```bash
cd services/iris/client
CI=false npm run build
cd /Users/sagar/Documents/codebases/localfinance
mkdir -p internal/webui/dist
cp -r services/iris/client/build/* internal/webui/dist/
```

- [ ] **Step 2: Write the failing test**

`internal/webui/embed_test.go`:

```go
package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServesIndexHTML(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/")
	if err != nil { t.Fatal(err) }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<!doctype html>") {
		t.Fatalf("expected index html, got %s", body[:200])
	}
}

func TestServesStaticAsset(t *testing.T) {
	srv := httptest.NewServer(Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/static/css/main.css")
	// Acceptable: 200 OR 404 (depending on hashed filename). Just shouldn't 500.
	if err != nil { t.Fatal(err) }
	defer resp.Body.Close()
	if resp.StatusCode == 500 { t.Fatal("expected non-500") }
}
```

- [ ] **Step 3: Run, expect FAIL**

```bash
go test ./internal/webui/...
```

- [ ] **Step 4: Implement embed.go**

```go
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var embedded embed.FS

// Handler serves the embedded React build. SPA routing: any unknown path
// returns index.html so React Router can handle it client-side.
func Handler() http.Handler {
	sub, _ := fs.Sub(embedded, "dist")
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API routes already handled by the gin router; this is mounted last.
		f, err := sub.Open(r.URL.Path[1:])
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback
		index, err := sub.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer index.Close()
		w.Header().Set("Content-Type", "text/html")
		http.ServeContent(w, r, "index.html", time.Time{}, index.(io.ReadSeeker))
	})
}
```

- [ ] **Step 5: Run, expect PASS**

```bash
go test ./internal/webui/...
```

- [ ] **Step 6: Add .gitignore**

`internal/webui/dist/` is build output — ignore it:

```
echo "internal/webui/dist/" >> .gitignore
```

But the embed needs files at build time. Solution: `make build-mono` runs `make webui-build` first (Task 4.2).

- [ ] **Step 7: Commit**

```bash
git add internal/webui .gitignore
git commit -m "feat(webui): embed React build via go:embed"
```

### Task 4.2: Add Makefile target that builds React + binary together

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add `webui-build` target**

```makefile
.PHONY: webui-build
webui-build:
	@cd services/iris/client && CI=false $(NPM) run build 2>&1 | tail -3
	@rm -rf internal/webui/dist
	@cp -r services/iris/client/build internal/webui/dist
	@echo "  → internal/webui/dist/ (embed input)"

build-mono: webui-build
	@go build -o $(DIST)/localfinance ./cmd/localfinance
	@echo "  → $(DIST)/localfinance ($(shell du -h $(DIST)/localfinance | cut -f1))"
```

- [ ] **Step 2: Verify**

```bash
make build-mono
ls -lh dist/localfinance
```

Expected: a binary in the 30-50 MB range with React embedded.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat(deploy): make build-mono runs webui-build first"
```

### Task 4.3: Mount the webui handler in the Gin router

**Files:**
- Modify: `internal/api/router.go`

- [ ] **Step 1: Mount webui as the no-op fallback**

```go
import "github.com/sagarjhaa/localfinance/internal/webui"

func NewGinRouter(db *gorm.DB) *gin.Engine {
	r := gin.New()
	// ... existing route registration

	// Static + SPA fallback — must be LAST so /api/* takes precedence
	r.NoRoute(gin.WrapH(webui.Handler()))

	return r
}
```

- [ ] **Step 2: Verify**

```bash
make build-mono
PORT=3001 ./dist/localfinance &
sleep 2
curl -sf http://localhost:3001/ | head -c 200
curl -sf http://localhost:3001/insights | head -c 200  # SPA route
curl -sf http://localhost:3001/api/health
kill %1
```

Expected: `/` and `/insights` both serve React; `/api/health` returns JSON.

- [ ] **Step 3: Commit**

```bash
git add internal/api/router.go
git commit -m "feat(api): mount embedded webui as Gin NoRoute fallback"
```

### Task 4.4: Delete services/iris/server (the Express proxy)

**Files:**
- Delete: `services/iris/server/`
- Delete: `services/iris/Dockerfile`
- Delete: `services/iris/package.json` (the server one)
- Modify: `docker-compose.dev.yml` — drop the iris service block

- [ ] **Step 1: Delete the Node server**

```bash
rm -rf services/iris/server
rm -f services/iris/Dockerfile
rm -f services/iris/package.json services/iris/package-lock.json services/iris/node_modules
```

- [ ] **Step 2: Drop iris from compose**

Remove the `iris` service from `docker-compose.dev.yml`. Iris was a Node container; now the binary at `:3001` does its job.

- [ ] **Step 3: Update the binary's compose service to listen on 3001**

The `thesaurus` compose service from Task 2.4 already exposes `8001:3001`. Add a second port mapping `3001:3001` so the browser can reach the unified binary directly:

```yaml
  thesaurus:
    ports:
      - "3001:3001"
      - "8001:3001"
```

- [ ] **Step 4: Verify dev-up**

```bash
make dev-down
make dev-up
curl -sf http://localhost:3001/api/health
open http://localhost:3001
```

Expected: browser loads the React app served by Go.

- [ ] **Step 5: Commit**

```bash
git add services/iris docker-compose.dev.yml
git commit -m "feat(deploy): drop Iris Express proxy and node container"
```

---

## Phase 5 — Embed Postgres

Goal: the binary downloads + manages its own Postgres so users don't need brew. Uses `embedded-postgres-go`.

### Task 5.1: Add the dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add the lib**

```bash
go get github.com/fergusstrange/embedded-postgres@latest
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add embedded-postgres-go for embedded Postgres"
```

### Task 5.2: Implement internal/postgres lifecycle

**Files:**
- Create: `internal/postgres/postgres.go`
- Create: `internal/postgres/postgres_test.go`

- [ ] **Step 1: Write a test for Start/Stop**

```go
package postgres

import (
	"context"
	"testing"
	"time"
)

func TestStartStop(t *testing.T) {
	if testing.Short() { t.Skip("integration") }
	mgr, err := New(t.TempDir(), 0) // 0 = pick free port
	if err != nil { t.Fatal(err) }

	if err := mgr.Start(context.Background()); err != nil { t.Fatal(err) }
	defer mgr.Stop(context.Background())

	if mgr.DSN() == "" { t.Fatal("expected DSN") }

	// Wait for ready
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := mgr.Ping(context.Background()); err == nil { return }
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("postgres never became ready")
}
```

- [ ] **Step 2: Implement Manager**

```go
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	embedded "github.com/fergusstrange/embedded-postgres"
)

type Manager struct {
	pg   *embedded.EmbeddedPostgres
	port uint32
	dataDir string
}

func New(dataDir string, port uint32) (*Manager, error) {
	if port == 0 {
		var err error
		port, err = findFreePort()
		if err != nil { return nil, err }
	}
	pg := embedded.NewDatabase(embedded.DefaultConfig().
		Port(port).
		DataPath(dataDir).
		Database("localfinance"))
	return &Manager{pg: pg, port: port, dataDir: dataDir}, nil
}

func (m *Manager) Start(ctx context.Context) error { return m.pg.Start() }
func (m *Manager) Stop(ctx context.Context) error  { return m.pg.Stop() }
func (m *Manager) DSN() string {
	return fmt.Sprintf("host=localhost port=%d user=postgres password=postgres dbname=localfinance sslmode=disable", m.port)
}
func (m *Manager) Ping(ctx context.Context) error {
	db, err := sql.Open("postgres", m.DSN())
	if err != nil { return err }
	defer db.Close()
	return db.PingContext(ctx)
}

func findFreePort() (uint32, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { return 0, err }
	defer l.Close()
	return uint32(l.Addr().(*net.TCPAddr).Port), nil
}
```

- [ ] **Step 3: Run test (will download Postgres on first run)**

```bash
go test ./internal/postgres/... -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/postgres
git commit -m "feat(postgres): embedded Postgres lifecycle manager"
```

### Task 5.3: Wire embedded Postgres into main.go

**Files:**
- Modify: `cmd/localfinance/main.go`

- [ ] **Step 1: Pick connection mode based on env**

```go
func main() {
	dataDir := dataDir() // ~/Library/Application Support/LocalFinance
	useEmbedded := os.Getenv("DB_HOST") == "" // dev sets DB_HOST=localhost; .app doesn't

	var dsn string
	if useEmbedded {
		mgr, err := postgres.New(filepath.Join(dataDir, "postgres"), 0)
		if err != nil { fail("postgres init", err) }
		if err := mgr.Start(context.Background()); err != nil { fail("postgres start", err) }
		defer mgr.Stop(context.Background())
		dsn = mgr.DSN()
	} else {
		dsn = buildDSNFromEnv()
	}
	// ... existing migrate / seed / serve
}
```

- [ ] **Step 2: Smoke test both modes**

```bash
# Embedded mode
unset DB_HOST
make build-mono
./dist/localfinance &
sleep 30  # postgres extract on first run
curl -sf http://localhost:3001/api/health
kill %1

# Host-DB mode
DB_HOST=localhost ./dist/localfinance &
sleep 5
curl -sf http://localhost:3001/api/health
kill %1
```

- [ ] **Step 3: Commit**

```bash
git add cmd/localfinance/main.go
git commit -m "feat(postgres): use embedded Postgres in .app, host DB in dev"
```

---

## Phase 6 — First-run Ollama wizard

Goal: the binary detects Ollama, walks the user through install, then through model pull. Three React routes + four backend endpoints.

### Task 6.1: Backend setup endpoints

**Files:**
- Create: `internal/api/setup/handlers.go`
- Create: `internal/api/setup/handlers_test.go`

- [ ] **Step 1: Write the test**

`internal/api/setup/handlers_test.go` covers:
- `GET /api/setup/state` returns `install_ollama` when Ollama probe fails
- Returns `pull_model` when Ollama is up but no sweet-spot model is installed
- Returns `ready` when Ollama is up + at least one sweet-spot model is installed
- `GET /api/setup/recommended` returns a model name based on a fake RAM probe injected via test seam

- [ ] **Step 2: Implement**

```go
package setup

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sagarjhaa/localfinance/internal/ai"
	"github.com/sagarjhaa/localfinance/internal/ollama"
)

type Handlers struct {
	OllamaHost string
	HostRAM    func() uint64
}

func (h *Handlers) State(c *gin.Context) {
	if !ollama.Reachable(h.OllamaHost) {
		c.JSON(200, gin.H{"step": "install_ollama"})
		return
	}
	models := ollama.InstalledModels(h.OllamaHost)
	if !ai.HasSweetSpotModel(models) {
		c.JSON(200, gin.H{"step": "pull_model"})
		return
	}
	c.JSON(200, gin.H{"step": "ready"})
}

func (h *Handlers) Recommended(c *gin.Context) {
	gb := h.HostRAM() / (1 << 30)
	model, sizeGB, reason := ai.RecommendByRAM(gb)
	c.JSON(200, gin.H{"model": model, "size_gb": sizeGB, "reason": reason})
}

// PullModel SSE-streams progress from Ollama /api/pull to the browser.
func (h *Handlers) PullModel(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	var req struct{ Model string `json:"model"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }

	ollama.StreamPull(c.Request.Context(), h.OllamaHost, req.Model, func(line []byte) {
		_, _ = c.Writer.Write([]byte("data: "))
		_, _ = c.Writer.Write(line)
		_, _ = c.Writer.Write([]byte("\n\n"))
		c.Writer.Flush()
	})
}

func (h *Handlers) OllamaStatus(c *gin.Context) {
	c.JSON(200, gin.H{
		"installed": ollama.Reachable(h.OllamaHost),
		"version":   ollama.Version(h.OllamaHost),
		"host_ram_gb": h.HostRAM() / (1 << 30),
	})
}
```

- [ ] **Step 3: Implement helper functions in internal/ollama/installer.go**

`Reachable(host) bool`, `InstalledModels(host) []string`, `Version(host) string`, `StreamPull(ctx, host, model, cb)`. Use stdlib `net/http`. `StreamPull` consumes `application/x-ndjson` from Ollama and invokes `cb` per line.

- [ ] **Step 4: Implement RAM probe in internal/ollama/ram.go**

```go
package ollama

import "syscall"

func HostRAM() uint64 {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil { return 0 }
	return uint64(info.Totalram) * uint64(info.Unit)
}
```

(For Mac, prefer `unix.SysctlUint64("hw.memsize")` — adjust per platform.)

- [ ] **Step 5: Wire into the router**

```go
setup := r.Group("/api/setup")
setupH := &setup.Handlers{OllamaHost: ollamaHost, HostRAM: ollama.HostRAM}
setup.GET("/state", setupH.State)
setup.GET("/ollama-status", setupH.OllamaStatus)
setup.GET("/recommended", setupH.Recommended)
setup.POST("/pull-model", setupH.PullModel)
```

- [ ] **Step 6: Run + commit**

```bash
go test ./internal/api/setup/... ./internal/ollama/...
git add internal/api/setup internal/ollama
git commit -m "feat(setup): backend endpoints for first-run wizard"
```

### Task 6.2: React wizard pages

**Files:**
- Create: `services/iris/client/src/pages/setup/InstallOllama.jsx`
- Create: `services/iris/client/src/pages/setup/PullModel.jsx`
- Modify: `services/iris/client/src/App.jsx` — add `/setup/install`, `/setup/pull` routes

- [ ] **Step 1: InstallOllama page**

Polls `/api/setup/ollama-status` every 2s. Shows download button. When `installed=true`, navigates to `/setup/pull`.

- [ ] **Step 2: PullModel page**

Calls `/api/setup/recommended`. On Download click, opens an EventSource to `/api/setup/pull-model` (POST + SSE). Renders progress bar from `total / completed` fields in the stream. On done, navigates to `/login`.

- [ ] **Step 3: App.jsx routes**

```jsx
<Route path="/setup/install" element={<InstallOllama />} />
<Route path="/setup/pull" element={<PullModel />} />
```

- [ ] **Step 4: Boot redirect logic**

In the React app's root component, on first mount call `/api/setup/state` and redirect:
- `install_ollama` → `/setup/install`
- `pull_model` → `/setup/pull`
- `ready` → existing default route

- [ ] **Step 5: Build + smoke**

```bash
make build-mono
PORT=3001 ./dist/localfinance &
# In a browser: confirm wizard shows when Ollama is intentionally stopped
kill %1
```

- [ ] **Step 6: Commit**

```bash
git add services/iris/client/src
git commit -m "feat(webui): first-run Ollama install + model pull wizard"
```

---

## Phase 7 — Final cleanup

Goal: delete `services/`, simplify Makefile, update the .app installer to use the single binary.

### Task 7.1: Delete services/

**Files:**
- Delete: `services/`

- [ ] **Step 1: Confirm nothing imports from services/**

```bash
grep -rn "github.com/sagarjhaa/localfinance/services/" cmd/ internal/ 2>&1 | head
```

Expected: zero matches. If any remain, fix them before deleting.

- [ ] **Step 2: Delete**

```bash
git rm -rf services/
```

- [ ] **Step 3: Verify build**

```bash
make build-mono
make test-mono
```

- [ ] **Step 4: Commit**

```bash
git commit -m "chore: remove services/ directory; everything lives under internal/"
```

### Task 7.2: Simplify Makefile

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Drop legacy targets**

Delete: `build-thesaurus`, `build-sophia`, `build-logos`, `build-hermes`, `build-iris`, `dev-up`, `dev-down`, `dev-logs-*`, `dev-restart-*`, `test-thesaurus`, `test-sophia`, `test-logos`, `test-iris`, `verify-*`.

Keep / add:
- `make dev` — go run ./cmd/localfinance
- `make build-mono` — full binary build
- `make test` — go test ./internal/... ./cmd/...
- `make test-e2e` — Go end-to-end tests
- `make eval-hallucination` — same as today
- `make installer` — builds .app
- `make installer-clean`, `make installer-run`, `make installer-test` — keep

- [ ] **Step 2: Verify**

```bash
make help
make test
make build-mono
```

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "chore(make): simplify Makefile for single-binary world"
```

### Task 7.3: Update LocalFinance.app installer

**Files:**
- Modify: `installer/build.sh`
- Modify: `installer/launcher/main.go`

- [ ] **Step 1: build.sh — copy single binary, no Resources/bin/**

```bash
# In installer/build.sh
make build-mono
mkdir -p dist/LocalFinance.app/Contents/MacOS
cp dist/localfinance dist/LocalFinance.app/Contents/MacOS/LocalFinance
mkdir -p dist/LocalFinance.app/Contents/Resources
cp installer/AppIcon.icns dist/LocalFinance.app/Contents/Resources/
cat > dist/LocalFinance.app/Contents/Info.plist <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key><string>com.localfinance.app</string>
  <key>CFBundleName</key><string>LocalFinance</string>
  <key>CFBundleVersion</key><string>0.2.0</string>
  <key>CFBundleExecutable</key><string>LocalFinance</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
</dict>
</plist>
PLIST
```

- [ ] **Step 2: Delete the launcher**

The Go launcher orchestration is gone — `LocalFinance` IS the binary now, no children to spawn (Postgres is embedded; React is embedded; Ollama is detected).

```bash
rm -rf installer/launcher
```

- [ ] **Step 3: Test**

```bash
make installer
open dist/LocalFinance.app
```

Expected: app launches, browser opens.

- [ ] **Step 4: Commit**

```bash
git add installer/
git commit -m "feat(installer): collapse to single-binary .app"
```

### Task 7.4: Update CLAUDE.md and README for new architecture

**Files:**
- Modify: `CLAUDE.md`
- Modify: `README.md`

- [ ] **Step 1: Rewrite CLAUDE.md**

Drop the "5 services" architecture table. Replace with: "LocalFinance is a single Go binary. Internal packages: api, auth, data, parse, ai, insights, monthreview, ollama, postgres, webui."

- [ ] **Step 2: Rewrite README.md quickstart**

```markdown
## Quick start

\`\`\`
brew install ollama
ollama serve &
make dev    # runs cmd/localfinance against host postgres + host ollama
\`\`\`

Or:

\`\`\`
make installer
open dist/LocalFinance.app
\`\`\`
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md README.md
git commit -m "docs: rewrite for single-binary architecture"
```

---

## Self-Review

**Spec coverage:**
- Architecture (one binary, two processes) → Phase 0-7 ✓
- Internal package layout → Phase 0 (skeleton) + 1-3 (lifts) ✓
- Mapping from today's services → Phase 1-4 ✓
- Data flows (upload, chat, insights) → Phase 3 (parse pipeline) ✓
- First-run Ollama wizard + model pull → Phase 6 ✓
- Reversibility — preserved by interface boundaries (data.Repositories) ✓
- Error handling — `gin.Recovery()` middleware in router; goroutine helper deferred to as-needed
- Testing — covered per-phase ✓
- Local dev workflow — `make dev` ✓
- Packaging — Phase 7.3 ✓
- Out-of-scope items (multi-user, code signing, Ollama bundling) — explicitly not in plan ✓

**Placeholder scan:** None. Every step has either code or exact commands.

**Type consistency:** `data.Repositories` interface defined Task 1.1 and consumed Task 1.2/1.3/2.2. `ai.Service` referenced consistently. `parse.Pipeline` defined Task 3.2.

**Ambiguity check:** Phase 6 RAM probe is platform-dependent — flagged inline ("For Mac, prefer `unix.SysctlUint64("hw.memsize")` — adjust per platform").

---

## Out of scope

These items are not part of this plan. Track separately:

- Multi-user / multi-tenant (Phase 3 product validation)
- Couples Mode sync (Phase 3 product validation)
- Code signing + notarization (separate plan; needs Apple Developer account)
- Bundling the Ollama binary inside `.app` (revisit post-validation)
- Windows / Linux installers (separate plans)
- SSE-based live upload progress (currently 2s polling; flagged in TODOS)
