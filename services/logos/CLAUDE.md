# Logos — Document Processing (port 8003)

Stateless document parser. Receives files from Thesaurus, parses
CSV/PDF, extracts transactions, sends back to Thesaurus.

## Build

make build-logos              # native
make build-arm64-logos        # cross-compile for Jetson

## Test

make test-logos               # unit tests (CSV parser, PDF parser, detector)
make test-e2e                 # integration tests

## Deploy

make deploy-logos             # build ARM64 + SCP + restart + verify

## Processing Flow

1. Thesaurus receives file upload -> stores document record -> fires POST to Logos
2. Logos detects file type (CSV/PDF) via processors/detector.go
3. Logos parses transactions via processors/csv_processor.go or pdf_processor.go
4. Logos fetches user's category rules from Thesaurus /internal/category-rules/
5. Applies matching rules, sends unmatched to Ollama for AI categorization
6. Sends all transactions to Thesaurus /internal/transactions/bulk
7. Updates document status to "processed" via Thesaurus /internal/documents/

## Adding a New File Format

1. Create processor in processors/ implementing the Processor interface
2. Register in processors/manager.go
3. Add format detection in processors/detector.go
4. Add test with fixture file in processors/*_test.go

## Key Files

- main.go — Gin router + inline processing logic (~440 lines)
- processors/csv_processor.go — CSV parsing
- processors/pdf_processor.go — PDF text extraction and transaction parsing
- processors/detector.go — statement type/institution detection
- processors/manager.go — processor factory
- models/models.go — ProcessRequest, Transaction types
- config/config.go — Thesaurus URL, Ollama URL
