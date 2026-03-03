# LocalFinance

**Your money. Your data. Your device.**

A fully local, privacy-first personal finance system with an AI assistant that runs entirely on your hardware. No cloud. No subscriptions. No data leaving your device.

---

## 🚀 Quick Start

```bash
# Clone and setup
git clone https://github.com/sagarjhaa/localfinance.git
cd localfinance
./scripts/setup.sh

# Configure
cp config.json.template config.json
# Edit config.json with your Telegram bot token (from @BotFather)

# Run
./scripts/run.sh
```

**That's it!** Message your bot on Telegram with questions like:
- "How much did I spend on dining last month?"
- "Show me all Amazon purchases"
- "What are my subscriptions?"

---

## 📊 Current Status

| Component | Status |
|-----------|--------|
| AI Model (Qwen2-0.5B fine-tuned) | ✅ Trained |
| SQL Generation | ✅ Working (8/8 clean queries) |
| Telegram Bot | ✅ Integrated |
| Setup Scripts | ✅ Ready |
| GGUF Conversion | ⏳ Pending (torch version) |
| Raspberry Pi Testing | ⏳ Tonight |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     YOUR DEVICE                              │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │  Telegram   │→ │  Bot Server  │→ │  LocalFinance AI  │  │
│  │  (your app) │  │  (Python)    │  │  (Qwen2-0.5B)     │  │
│  └─────────────┘  └──────┬───────┘  └─────────┬─────────┘  │
│                          │                     │            │
│                          ▼                     ▼            │
│                   ┌──────────────┐    ┌──────────────┐     │
│                   │   SQLite DB   │←──│ SQL Generator │     │
│                   │ (your data)   │    │              │     │
│                   └──────────────┘    └──────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

**Privacy:** All processing happens locally. The only network connection is Telegram's API for the chat interface.

---

## 📁 Repository Structure

```
localfinance/
├── src/
│   ├── inference.py      # AI model loading & SQL generation
│   ├── bot.py            # Telegram bot integration
│   └── prototype.py      # Early prototype
├── models/
│   └── localfinance-v1/  # Fine-tuned Qwen2-0.5B model (1.97GB)
├── training/
│   ├── train.py          # Training script
│   ├── test_model.py     # Model testing with stop tokens
│   ├── data/             # 2,944 training examples
│   └── train.log         # Training logs
├── scripts/
│   ├── setup.sh          # One-command device setup
│   ├── run.sh            # Bot launcher
│   └── convert_to_gguf.py # GGUF conversion (for llama.cpp)
├── docs/                 # Architecture, user flow, business docs
├── requirements.txt      # Python dependencies
└── config.json.template  # Configuration template
```

---

## 💻 Requirements

- **Python:** 3.10+
- **RAM:** 4GB minimum (model uses ~2GB)
- **Storage:** ~3GB for model + dependencies
- **Tested on:** macOS, Linux (Raspberry Pi 4+ pending)

---

## 🤖 How the AI Works

1. You ask: *"How much did I spend on dining last month?"*
2. Model generates SQL: `SELECT SUM(amount) FROM transactions WHERE category = 'Dining' AND date >= date('now', '-1 month')`
3. Bot executes SQL on your local database
4. You get: *"💰 $847.32"*

**Model specs:**
- Base: Qwen2-0.5B-Instruct (MIT license)
- Fine-tuned on: 2,944 finance query examples
- Inference time: ~20s on CPU (faster with GGUF/llama.cpp)

---

## 📝 Roadmap

- [x] Architecture & planning
- [x] Training data generation
- [x] Model fine-tuning
- [x] Telegram bot integration
- [ ] GGUF conversion for faster inference
- [ ] Raspberry Pi deployment testing
- [ ] WiFi captive portal for easy setup
- [ ] Statement PDF/CSV import via chat
- [ ] WhatsApp integration (Business API)

---

## 🔒 Privacy Promise

- **No cloud:** All AI runs on your device
- **No tracking:** We don't see your data
- **No subscriptions:** One-time setup
- **Open source:** Audit the code yourself

---

*Built with privacy in mind. Your finances stay yours.*
