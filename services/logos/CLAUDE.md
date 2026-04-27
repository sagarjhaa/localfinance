# Logos — Document Processing (port 8003)

Stateless document text-extractor. Receives files from Thesaurus, extracts
raw text, hands the text to Sophia's AI parse endpoint, then ships the
returned transactions back to Thesaurus. **Logos no longer parses
transactions itself** — every upload is parsed by the local LLM via Sophia.

## Build

make build-logos              # native

## Test

make test-logos               # unit tests (text extraction, Sophia client, detector)
make test-e2e                 # integration tests

## Processing Flow

1. Thesaurus receives file upload, stores document record, fires `POST /api/v1/process` to Logos.
2. Logos extracts raw text from the file:
   - **PDF**: `pdftotext -layout` (poppler) with a `ledongthuc/pdf` Go fallback.
   - **CSV / TXT**: file bytes read verbatim as UTF-8.
   - **XLSX / XLS**: rejected — user must convert to CSV.
3. Logos POSTs `{ text, user_id }` to Sophia `POST /api/v1/parse` (5-minute timeout).
4. Sophia's local LLM returns `{ transactions, count, model }`. Logos converts each
   `map[string]interface{}` into a `models.Transaction`.
5. Logos applies the user's category rules from Thesaurus
   `/api/v1/internal/category-rules/match`.
6. Logos batch-categorizes any remaining uncategorized transactions via Ollama directly
   (`aiCategorize` in `main.go` — fast bulk prompt).
7. Logos detects statement metadata (institution, account number, period, balances)
   from the same extracted text via `processors.DetectStatementInfo`.
8. Logos POSTs transactions + metadata to Thesaurus `/api/v1/transactions/bulk`.
9. Logos PATCHes the document status to `processed` with the extracted text payload.

## Failure Modes

- **Text extraction fails** (PDF unreadable, file missing): document → `status=error`
  with message `Text extraction failed: <reason>`.
- **Sophia unreachable / 5xx / non-200**: document → `status=error` with message
  `AI parse failed: <reason>`. The file is preserved so it can be retried later.
- **Sophia returns 0 transactions**: document → `status=error` with message
  `AI returned no transactions from this file`. Extracted text is still saved
  so a human can inspect what the LLM saw.
- **XLSX uploaded**: rejected immediately with `Text extraction failed: excel
  binary format no longer supported — please convert to CSV before uploading`.

## Adding a New File Format

Only the text-extraction step is format-specific now. Add a case to
`extractTextForAI` in `main.go` that returns the file's contents as text.
Sophia's LLM handles every parsing detail downstream.

## Key Files

- `main.go` — Gin router, extract → Sophia parse → categorize → send to Thesaurus.
- `processors/pdf_processor.go` — `ExtractPDFText` and `normalizeDate` only.
- `processors/detector.go` — statement metadata detection (institution, period, etc.).
- `processors/types.go` — `ProcessResult` struct used by `main.go`.
- `models/models.go` — `ProcessRequest`, `Transaction`, `ProcessingResult`.
- `config/config.go` — Thesaurus URL, Sophia URL, Ollama URL.

## Environment

- `THESAURUS_URL` (default `http://localhost:8001`) — for category rules + saving transactions.
- `SOPHIA_URL` (default `http://localhost:8002`) — required, all parsing flows here.
- `OLLAMA_HOST` (default `http://127.0.0.1:11434`) — used by the bulk category prompt.
