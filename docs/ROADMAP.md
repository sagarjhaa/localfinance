# Development Roadmap

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│  PHASE 1        PHASE 2        PHASE 3        PHASE 4          │
│  Foundation     AI Model       Polish         Launch           │
│  (2 weeks)      (2 weeks)      (2 weeks)      (1 week)         │
│                                                                  │
│  ▓▓▓▓▓▓▓▓▓▓    ░░░░░░░░░░    ░░░░░░░░░░    ░░░░░░░░░░        │
│  ← We are here                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Foundation (Weeks 1-2)

### Goals
- Generalize existing code from personal setup
- Create installable package
- Basic CLI working

### Tasks

#### Week 1: Core Restructure
- [ ] **P0** Extract code from `~/projects/finances/`
- [ ] **P0** Remove hardcoded paths → use `~/.localfinance/`
- [ ] **P0** Create `config.yaml` for user settings
- [ ] **P1** Package as pip-installable (`pip install localfinance`)
- [ ] **P1** Create `localfinance init` command
- [ ] **P1** Create `localfinance import <file>` command

#### Week 2: Parser Generalization
- [ ] **P0** Refactor parser architecture (pluggable)
- [ ] **P0** Chase checking/credit parser
- [ ] **P0** Bank of America parser
- [ ] **P1** Capital One parser
- [ ] **P1** American Express parser
- [ ] **P2** Generic CSV fallback parser
- [ ] **P2** Parser auto-detection

### Deliverable
```bash
pip install localfinance
localfinance init
localfinance import ~/Downloads/chase-statement.pdf
localfinance query "SELECT * FROM transactions LIMIT 5"
```

---

## Phase 2: AI Model (Weeks 3-4)

### Goals
- Fine-tune small model for finance queries
- Natural language → SQL pipeline
- Local inference working

### Tasks

#### Week 3: Training Data + Fine-tuning
- [ ] **P0** Generate training data (500+ Q&A pairs)
- [ ] **P0** Set up fine-tuning pipeline (Unsloth or llama.cpp)
- [ ] **P0** Fine-tune Qwen2 0.5B or TinyLlama 1.1B
- [ ] **P1** Evaluate model accuracy
- [ ] **P1** Export to GGUF format

#### Week 4: Inference Pipeline
- [ ] **P0** Integrate llama-cpp-python
- [ ] **P0** Create `localfinance ask "<question>"` command
- [ ] **P0** NL → SQL → Result → Human answer pipeline
- [ ] **P1** Context injection (schema + recent data)
- [ ] **P2** Conversation memory (follow-up questions)

### Deliverable
```bash
localfinance ask "How much did I spend on groceries last month?"
# Output: "You spent $487.32 on groceries in January 2026."
```

---

## Phase 3: Polish (Weeks 5-6)

### Goals
- Desktop UI (optional but nice)
- Error handling + edge cases
- Documentation

### Tasks

#### Week 5: User Experience
- [ ] **P1** Better error messages
- [ ] **P1** Progress indicators for imports
- [ ] **P1** Category management commands
- [ ] **P2** Goal tracking commands
- [ ] **P2** Budget tracking commands

#### Week 6: Distribution Prep
- [ ] **P0** Create installer (PyInstaller or similar)
- [ ] **P1** macOS app bundle
- [ ] **P1** Windows installer
- [ ] **P2** Linux AppImage
- [ ] **P0** Write user documentation
- [ ] **P1** Create demo video

### Deliverable
- Downloadable installers
- Documentation site
- Demo video

---

## Phase 4: Launch (Week 7)

### Goals
- Soft launch to small audience
- Collect feedback
- Fix critical issues

### Tasks
- [ ] **P0** Landing page
- [ ] **P0** Payment integration (Stripe)
- [ ] **P0** Download delivery system
- [ ] **P1** ProductHunt preparation
- [ ] **P1** Reddit/HN posts drafted
- [ ] **P2** Email capture for updates

### Deliverable
- Live product
- First paying customers
- Feedback loop established

---

## Post-Launch (Ongoing)

### Month 2-3
- [ ] Additional bank parsers (by request)
- [ ] Improve AI accuracy based on real usage
- [ ] Mobile companion app (view-only)
- [ ] Investment account support

### Month 4-6
- [ ] Net worth tracking
- [ ] Recurring transaction detection
- [ ] Tax category reports
- [ ] Desktop app (Tauri)

### Month 6-12
- [ ] Multi-currency support
- [ ] Shared household finances
- [ ] API for integrations
- [ ] Enterprise/advisor features

---

## Technical Debt to Address

| Item | Priority | Notes |
|------|----------|-------|
| Test coverage | P1 | Need >80% coverage |
| Error handling | P1 | User-friendly messages |
| Logging | P2 | Debug support issues |
| Performance | P2 | Large statement imports |
| Security audit | P1 | Before launch |

---

## Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Model accuracy too low | Medium | High | More training data, larger model |
| Bank format changes | Medium | Medium | Parser versioning, quick updates |
| Install issues | Medium | High | Thorough testing, good docs |
| Low conversion rate | Medium | Medium | Free tier value, testimonials |
| Support overload | Low | Medium | Good docs, FAQ, community |

---

## Resource Needs

### Development
- 1 developer (full-time equivalent for 7 weeks)
- Access to GPU for training (rent if needed)

### Design
- Logo + branding (can outsource)
- Landing page design
- App icon

### Content
- Documentation
- Demo video
- Launch posts

---

## Definition of Done (v1.0)

- [ ] User can install with one command/click
- [ ] User can import statements from 5+ major banks
- [ ] AI answers spending questions accurately (>90%)
- [ ] Works fully offline after setup
- [ ] Documentation covers all features
- [ ] Payment flow works
- [ ] 10 beta users have successfully used it
