# Architecture

## Overview

LocalFinance is a **fully offline personal finance system** with three main components:

```
┌─────────────────────────────────────────────────────────────────┐
│                        USER'S DEVICE                            │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │  Statement  │───▶│   SQLite    │◀───│   Finance AI        │  │
│  │  Processor  │    │   Database  │    │   (Local LLM)       │  │
│  └─────────────┘    └─────────────┘    └─────────────────────┘  │
│        ▲                   ▲                      ▲             │
│        │                   │                      │             │
│   PDF/CSV files      All queries            Natural language    │
│   (drag & drop)      stay local             questions           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   NEVER LEAVES  │
                    │   THE DEVICE    │
                    └─────────────────┘
```

---

## Component Details

### 1. Statement Processor

**Purpose:** Import financial data from bank/credit card statements.

**Supported Formats:**
- PDF statements (parsed with pdfplumber)
- CSV exports (from bank websites)
- OFX/QFX files (future)

**Bank Parsers:**
Each bank has slightly different statement formats. We maintain parsers for:
- Chase (checking, credit)
- Bank of America
- Capital One
- American Express
- Discover
- Wells Fargo
- *Extensible for new banks*

**Flow:**
```
User drops PDF ──▶ Auto-detect bank ──▶ Extract transactions ──▶ Categorize ──▶ Store in DB
```

**Category Rules:**
Auto-categorization based on merchant patterns:
```python
CATEGORIES = {
    "Groceries": ["SAFEWAY", "TRADER JOE", "WHOLE FOODS", "COSTCO"],
    "Dining": ["DOORDASH", "UBER EATS", "GRUBHUB", "RESTAURANT"],
    "Gas": ["SHELL", "CHEVRON", "76", "EXXON"],
    # ... etc
}
```

---

### 2. SQLite Database

**Purpose:** Store all financial data locally in a single file.

**Location:** `~/.localfinance/data/finances.db`

**Schema Overview:**

```sql
-- Accounts (checking, savings, credit cards, investments)
accounts(id, name, type, institution, last_four, is_active)

-- Transactions (every purchase, payment, transfer)
transactions(id, date, description, amount, category, account_id, source)

-- Account balances over time
account_balances(id, account_id, balance, as_of_date, source)

-- Financial goals (emergency fund, vacation, etc.)
goals(id, name, target_amount, current_amount, target_date)

-- Budget categories (50/30/20 mapping)
budget_categories(category, bucket, monthly_budget)

-- Net worth snapshots
net_worth_snapshots(date, assets, liabilities, net_worth)

-- Recurring transactions
recurring_transactions(id, name, amount, frequency, category)
```

**Why SQLite?**
- Single file = easy backup
- No server needed
- Fast queries on local data
- Portable across platforms

---

### 3. Finance AI (Local LLM)

**Purpose:** Natural language interface to query financial data.

**How It Works:**
```
User Question ──▶ LLM generates SQL ──▶ Execute on SQLite ──▶ LLM formats answer
```

**Example:**
```
Input: "How much did I spend on groceries this year?"

Generated SQL:
SELECT SUM(amount) as total
FROM transactions 
WHERE category = 'Groceries' 
AND date >= date('now', 'start of year')

Raw Result: 3847.52

Formatted Answer: "You've spent $3,847.52 on groceries so far this year."
```

**Model Options:**

| Model | Size | RAM | Speed | Quality |
|-------|------|-----|-------|---------|
| Qwen2 0.5B | 400MB | 1GB | ⚡⚡⚡ | ⭐⭐ |
| TinyLlama 1.1B | 700MB | 2GB | ⚡⚡ | ⭐⭐⭐ |
| Llama 3.2 1B | 800MB | 2GB | ⚡⚡ | ⭐⭐⭐ |
| Llama 3.2 3B | 2GB | 4GB | ⚡ | ⭐⭐⭐⭐ |
| Phi-3 Mini 3.8B | 2.5GB | 5GB | ⚡ | ⭐⭐⭐⭐⭐ |

**Fine-tuning:**
We fine-tune on finance-specific Q&A pairs:
```json
{
  "instruction": "How much did I spend on dining last month?",
  "input": "Schema: transactions(date, description, amount, category)...",
  "output": "SELECT SUM(amount) FROM transactions WHERE category='Dining' AND date >= date('now', '-1 month')"
}
```

Training data: ~5,000-10,000 examples covering:
- Spending queries
- Budget questions
- Goal tracking
- Trend analysis
- Credit card metrics

---

## Deployment Options

### Option A: CLI Tool (pip install)
```bash
pip install localfinance
localfinance init
localfinance import ~/Downloads/statement.pdf
localfinance ask "What's my spending trend?"
```

### Option B: Desktop App (Tauri)
- Native app for macOS/Windows/Linux
- Drag-and-drop interface
- Chat UI for AI queries

### Option C: Docker
```bash
docker run -v ~/.localfinance:/data localfinance
```

---

## Security Model

| Layer | Protection |
|-------|------------|
| Data Storage | All data in `~/.localfinance/` (user-owned) |
| Encryption | Optional: encrypt DB with user password |
| Network | Zero network calls. No telemetry. No cloud. |
| Model | Runs 100% on-device via llama.cpp |
| Backup | User controls their own backups |

---

## Directory Structure (User's Machine)

```
~/.localfinance/
├── config.yaml           # User preferences
├── data/
│   └── finances.db       # SQLite database
├── inbox/                # Drop statements here
├── processed/            # Processed statements archive
├── models/
│   └── finance-v1.gguf   # Fine-tuned LLM weights
└── logs/                 # Application logs
```

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Python 3.10+ |
| Database | SQLite |
| PDF Parsing | pdfplumber |
| LLM Inference | llama-cpp-python |
| Model Format | GGUF |
| CLI | Click or Typer |
| Desktop App | Tauri (Rust + Web) |
| Packaging | PyInstaller or cx_Freeze |
