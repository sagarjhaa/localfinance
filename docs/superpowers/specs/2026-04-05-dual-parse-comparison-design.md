# Dual Parse Comparison Design Spec

**Date:** 2026-04-05
**Status:** Approved
**Goal:** Show static-parsed and AI-parsed transactions side by side after upload, so user can compare quality.

---

## Flow

1. User uploads file (existing flow)
2. Thesaurus stores document, fires to Logos (existing)
3. Logos static-parses, saves transactions to DB (existing)
4. Logos returns extracted text in its process response (new)
5. After static parse completes, Iris sends extracted text to Sophia for AI parsing (new)
6. Dashboard shows both results side by side

Static parse results persist (saved to DB). AI parse results are display-only.

---

## Backend Changes

### Logos — Return extracted text

Current Logos process response updates document status but doesn't return the raw text. Change: when Logos sends transactions to Thesaurus, also save the extracted text on the document record.

Add `extracted_text` field to the Document model in Thesaurus. Logos sets it when updating document status to "processed".

### Sophia — New parse endpoint

`POST /api/v1/parse`

**Request:**
```json
{
  "text": "<raw statement text>",
  "user_id": "uuid"
}
```

**Response:**
```json
{
  "transactions": [
    {"date": "2026-02-16", "description": "APPLE.COM/BILL", "amount": 18.99, "category": "Shopping"},
    {"date": "2026-02-17", "description": "Bollywood Salon", "amount": 36.00, "category": "Health"}
  ],
  "count": 58,
  "model": "llama3.2:1b"
}
```

Sophia sends the text to Ollama with a prompt asking it to extract transactions as JSON. Uses the user's preferred model.

### Thesaurus — Extracted text on document

Add `extracted_text` (text, nullable) to the Document model. Logos populates this via the existing status update endpoint. Frontend can fetch it to send to Sophia.

---

## Frontend Changes

### Dashboard after upload

Currently shows a single transaction table. Change to two panels:

```
┌─────────────────────────────┬─────────────────────────────┐
│ Static Parse (Logos)        │ AI Parse (Sophia)           │
│ 58 transactions · Saved     │ 52 transactions · Preview   │
├─────────────────────────────┼─────────────────────────────┤
│ Date  Description    Amount │ Date  Description    Amount │
│ 02/16 APPLE.COM/BILL 18.99 │ 02/16 Apple.com Bill 18.99  │
│ 02/17 SQ *BOLLYWOOD  36.00 │ 02/17 Bollywood Sal  36.00  │
│ 02/18 MADRAS GROCER   9.56 │ 02/18 Madras Grocer   9.56  │
│ ...                        │ ...                         │
└─────────────────────────────┴─────────────────────────────┘
```

Each panel shows full transaction details:
- Date
- Description
- Amount (color-coded: green for credits, red for debits)
- Category (chip/badge)

Header shows: count, source label, status (Saved vs Preview)

### Timing

- Static parse: shows as soon as Logos finishes (poll existing status endpoint)
- AI parse: starts after static parse completes (needs the extracted text), shows "AI parsing..." loading state, then renders when done

---

## Service Responsibilities

| Service | Change |
|---------|--------|
| **Thesaurus** | Add `extracted_text` to Document model |
| **Logos** | Save extracted text when updating document status |
| **Sophia** | New `POST /api/v1/parse` endpoint |
| **Iris** | Dual-panel UI on Dashboard after upload |

---

## What This Does NOT Include

- Saving AI-parsed transactions to DB
- User choosing between results
- Merging transactions from both sources
- Running AI parse on previously uploaded documents
