# AGENTS.md — LocalFinance Project

**Last Updated:** 2026-03-03

---

## 📊 Current Status

| Component | Status | Notes |
|-----------|--------|-------|
| Training Data | ✅ Complete | 2,944 examples |
| Model Training | ✅ Complete | Qwen2-0.5B fine-tuned |
| GGUF Conversion | ✅ Complete | F16 (948MB), Q4 (379MB) |
| Local Testing | ✅ Works | Generates SQL (needs prompt tuning) |
| Playground Deploy | ❌ Failed | Intel Mac too slow, see `docs/DEPLOYMENT_LOG.md` |
| **Jetson Hardware** | 🚚 Ordered | Arriving Mar 4! |

**Next Up (Mar 4):**
1. Unbox Jetson Orin Nano Super + case
2. Flash JetPack OS, install Ollama
3. Deploy model, test inference speed
4. Connect Telegram bot end-to-end

**Expected:** 2-3 second responses (vs 8+ minutes on Intel Mac)

---

## 🎯 What This Is

**LocalFinance** is a privacy-first personal finance system with a locally-running AI assistant. Target customers are non-technical users who get a pre-configured device, plug it in, and chat with it via Telegram/WhatsApp.

**Key differentiator:** Everything runs on the user's device. No cloud. No subscriptions. No bank login sharing.

---

## 📍 Project Locations

| Component | Path | Purpose |
|-----------|------|---------|
| LocalFinance (product) | `/Users/sagarjha/projects/localfinance` | Main product repo |
| Finance tools (internal) | `/Users/sagarjha/projects/finances` | Sagar's personal finance tools (statement processor, DB) |
| Dashboard integration | `/Users/sagarjha/clawd/dashboard` | Personal dashboard (not for customers) |

**Note:** The `finances` project has working code (statement parsers, CLI, Telegram bot) that will be adapted for LocalFinance.

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     USER'S DEVICE                           │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │  Telegram/  │→ │  Finance Bot │→ │  LocalFinance AI  │  │
│  │  WhatsApp   │  │  (Python)    │  │  (Qwen2-0.5B)     │  │
│  └─────────────┘  └──────┬───────┘  └─────────┬─────────┘  │
│                          │                     │            │
│                          ▼                     ▼            │
│                   ┌──────────────┐    ┌──────────────┐     │
│                   │   SQLite DB   │←──│ SQL Generator │     │
│                   │  (finances)   │    │              │     │
│                   └──────────────┘    └──────────────┘     │
│                          ▲                                  │
│              ┌───────────┴───────────┐                     │
│              │ Statement Processors   │                     │
│              │ (PDF/CSV → DB)         │                     │
│              └───────────────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

---

## 📊 Current Status (Mar 2, 2026)

### ✅ Done
- [x] Architecture planned (docs/)
- [x] User flow documented
- [x] Business model defined
- [x] Statement parsers working (Chase, Amex, Capital One, etc.)
- [x] SQLite schema designed
- [x] CLI interface (`finances.py`) with init/import/report/config
- [x] Telegram bot created (`finance_bot.py`) — needs token to test
- [x] Training data generated: **2,944 examples** in `training/data/combined.jsonl`
- [x] Training script ready (`training/train.py`)
- [x] **Model trained!** (Mar 2, 2026)
  - Runtime: 36h 13m on CPU
  - Final eval loss: 0.22
  - Model saved to `models/localfinance-v1/` (1.97GB)
- [x] **Test script with stop tokens** (`training/test_model.py`)
  - SQLStoppingCriteria halts at semicolons/newlines
  - Post-processing cleanup for edge cases
  - Results: 8/8 queries produce clean SQL
  - ~19s per query on CPU
- [x] **GGUF converter script** (`scripts/convert_to_gguf.py`)
- [x] **Telegram Bot Integration** (Mar 2, 2026 afternoon)
  - `src/inference.py`: AI inference module with SQL generation
  - `src/bot.py`: Full Telegram bot with /start, /help, /status, /test
  - End-to-end flow: Question → SQL → Database → Formatted response
  - Works with test_data.db (4 sample transactions)
- [x] **Device Setup Scripts**
  - `scripts/setup.sh`: One-command setup for new devices
  - `scripts/run.sh`: Easy bot launcher
  - `requirements.txt`: All dependencies
  - `config.json.template`: Configuration template

### 🔄 In Progress
- [ ] Device testing on Raspberry Pi / Mac Mini (tonight)
- [ ] WiFi provisioning for device (captive portal)

### ⏳ TODO (Next Steps)
- [ ] Run GGUF conversion (torch version issue — needs newer torch)
- [ ] Improve date handling in model (training data issue)
- [ ] Raspberry Pi / Mac Mini setup scripts
- [ ] First-run onboarding experience
- [ ] Generate more training data if accuracy needs improvement
- [ ] WhatsApp integration (requires Business API)

---

## 🤖 Model Training

**Base Model:** Qwen2-0.5B-Instruct (MIT license, ~400MB)

**Training Data:**
```
training/data/
├── combined.jsonl         # 2,944 examples (576KB)
├── spending_*.jsonl       # Spending queries by time/category
├── merchant.jsonl         # Merchant-specific queries
├── transactions.jsonl     # Transaction lookups
├── trends.jsonl           # Trend analysis
├── goals.jsonl            # Goal tracking
└── accounts.jsonl         # Account queries
```

**To Generate More Data:**
```bash
cd /Users/sagarjha/projects/localfinance
source .venv/bin/activate
python training/generate_100k.py  # or generate_more_data.py
```

**To Train:**
```bash
cd /Users/sagarjha/projects/localfinance
source .venv/bin/activate

# Run in background with logging:
nohup python training/train.py > training/train.log 2>&1 &
echo $!  # Save this PID!
```

**Output:** `models/localfinance-v1/`

---

## 📊 Monitoring Training

### Check if training is running:
```bash
ps aux | grep train.py | grep -v grep
```

### Watch live progress:
```bash
tail -f /Users/sagarjha/projects/localfinance/training/train.log
```

### Check progress summary:
```bash
tail -50 /Users/sagarjha/projects/localfinance/training/train.log | grep -E "loss|step|%"
```

### Ask Tiny to check:
Just say: "Check training status for localfinance"

### Training timeline (CPU):
- ~2-4 hours for 2,944 examples × 3 epochs
- ~1,989 steps total
- Progress shown as percentage in log

### When training completes:
1. Model saved to `models/localfinance-v1/`
2. Test with: `python training/test_model.py`
3. Convert to GGUF: `python scripts/convert_to_gguf.py` (TODO)

### If training crashes:
- Check log: `tail -100 training/train.log`
- Common issue: OOM on MPS → use CPU (current default)
- Restart with: `nohup python training/train.py > training/train.log 2>&1 &`

---

## 💬 Customer Experience

### Setup Flow (Non-Technical User)
1. **Unbox & plug in** → Device boots
2. **Connect to "LocalFinance-XXXX" WiFi** → Phone connects
3. **Open setup page** → Enter home WiFi credentials
4. **Scan QR code** → Links to Telegram bot
5. **Send statements** → Drop PDF/CSV in chat
6. **Ask questions** → "How much did I spend on groceries?"

### Chat Interface
- Natural language queries
- PDF/CSV file uploads
- Commands: /start, /help, /report, /balance

---

## 📁 Key Files

| File | Purpose |
|------|---------|
| `README.md` | Project overview |
| `docs/ARCHITECTURE.md` | Technical architecture |
| `docs/USER_FLOW.md` | Customer journey |
| `docs/ONBOARDING.md` | First-run experience |
| `docs/BUSINESS.md` | Business model |
| `src/prototype.py` | Working prototype |
| `training/train.py` | Model training script |
| `training/generate_100k.py` | Data generation (large scale) |
| `test_data.db` | Test SQLite database |

---

## 🔗 Related Projects

### `/Users/sagarjha/projects/finances`
Sagar's personal finance tools. Code here is being adapted for LocalFinance:
- `finances.py` — CLI interface (init, import, report, config)
- `finance_bot.py` — Telegram bot interface
- `schema.sql` — Database schema
- Statement parsers for Chase, Amex, Capital One, etc.

---

## 💡 Key Decisions

1. **Telegram-first** — Users already have it, no app to install
2. **WiFi captive portal** — Simplest non-technical setup
3. **Qwen2-0.5B** — Small enough for phones, smart enough for SQL
4. **Manual statement upload** — Privacy over convenience (no Plaid)
5. **SQLite** — Simple, portable, no server needed

---

## 🚨 Watch Out For

- **MPS memory limits** — Mac GPU can run out during training. Use `PYTORCH_MPS_HIGH_WATERMARK_RATIO=0.0` if needed
- **PDF parsing** — Different banks have wildly different formats
- **Telegram bot tokens** — Need BotFather setup per deployment
- **WiFi provisioning** — Test on actual Pi hardware

---

## 📝 Notes

- Target hardware: Raspberry Pi 4+ or any mini PC with 2GB+ RAM
- Model inference via llama.cpp for efficiency
- Could add WhatsApp later (needs Business API, more complex)

---

*This file helps me (Tiny) remember context across sessions. Update it when major progress happens.*
