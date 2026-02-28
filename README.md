# LocalFinance

**Your money. Your data. Your device.**

A fully local, privacy-first personal finance system with an AI assistant that runs entirely on your hardware. No cloud. No subscriptions. No data leaving your device.

---

## 🎯 Vision

Most finance apps (Mint, YNAB, Copilot) require you to:
- Connect your bank accounts to their servers
- Pay monthly subscriptions
- Trust them with your most sensitive data

**LocalFinance is different.** Everything runs on YOUR device:
- Import statements manually (drag & drop PDFs/CSVs)
- AI assistant answers questions about YOUR data
- Fine-tuned model runs locally on phone/laptop
- Zero internet required after setup

---

## 🏗️ What We're Building

### Core Product
1. **Statement Processor** — Import bank/credit card statements (PDF/CSV)
2. **SQLite Database** — Your financial data, stored locally
3. **Finance AI Model** — Small LLM fine-tuned for personal finance queries
4. **Query Interface** — Ask natural language questions, get answers

### Example Interactions
```
User: "How much did I spend on dining last month?"
AI: "You spent $847.32 on dining in January 2026, across 23 transactions. 
     That's 18% higher than your 3-month average of $718."

User: "Am I on track for my emergency fund goal?"
AI: "Your emergency fund is at $8,400 of your $15,000 goal (56%).
     At your current savings rate of $500/month, you'll reach it in 13 months."

User: "What's my credit utilization?"
AI: "Your total credit utilization is 23% ($2,760 of $12,000).
     Chase Sapphire: 31% | Amex Gold: 18% | Discover: 12%"
```

---

## 📁 Repository Structure

```
localfinance/
├── README.md                 # You are here
├── docs/
│   ├── ARCHITECTURE.md       # Technical architecture
│   ├── USER_FLOW.md          # User journey
│   ├── ONBOARDING.md         # Customer onboarding process
│   ├── BUSINESS.md           # Business model
│   └── ROADMAP.md            # Development roadmap
├── src/
│   ├── core/                 # Statement processing, DB management
│   ├── parsers/              # Bank-specific parsers (Chase, Amex, etc.)
│   ├── ai/                   # Model inference, query processing
│   └── ui/                   # CLI and/or desktop UI
├── models/                   # Fine-tuned model weights
├── scripts/                  # Setup, training, deployment scripts
├── tests/                    # Test suite
└── training/                 # Model training data and configs
```

---

## 🚀 Quick Links

- [Architecture](docs/ARCHITECTURE.md) — How it all fits together
- [User Flow](docs/USER_FLOW.md) — The customer experience
- [Onboarding](docs/ONBOARDING.md) — How we help users get started
- [Business Model](docs/BUSINESS.md) — How we make money

---

## 📊 Status

🟡 **Phase 1: Planning** ← We are here
- [ ] Architecture finalized
- [ ] User flow documented
- [ ] Business model defined

⚪ **Phase 2: Core Development**
- [ ] Statement processor generalized
- [ ] Database schema packaged
- [ ] CLI interface built

⚪ **Phase 3: AI Model**
- [ ] Training data generated
- [ ] Model fine-tuned
- [ ] Inference pipeline built

⚪ **Phase 4: Distribution**
- [ ] Installer/packaging
- [ ] Documentation
- [ ] Launch

---

*Built with privacy in mind. Your finances stay yours.*
