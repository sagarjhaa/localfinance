# TODOS

Deferred items from the CEO review on 2026-04-19. Each entry: what, why, pros, cons, context, effort, priority, blocked by.

Effort estimates show both scales: human team → CC+gstack.

---

## Phase 2.5 (post-refactor, post-core-insights)

### [P1] Tax Export (TurboTax CSV)
- **What:** Year-end export of transactions in TurboTax-compatible CSV format, grouped by tax-relevant categories (business, charitable, medical, interest, dividends).
- **Why:** High-value seasonal use case. Users who try this in Jan-Apr are most motivated to pay.
- **Pros:** Concrete dollar-value hook. Natural upsell moment. Relatively mechanical to build.
- **Cons:** Category mapping needs research per jurisdiction. TurboTax format changes annually.
- **Context:** Not in core vision but gives a paid product a seasonal cash spike. Should land by December 2026.
- **Effort:** M (human 1-2wk) → CC+gstack ~5-8h
- **Priority:** P1
- **Blocked by:** Phase 2 categories + insights working

### [P1] Windows Installer
- **What:** MSI installer for Windows 10/11. Parity with macOS and Linux.
- **Why:** Windows is ~60% of desktop share. Excluding it caps addressable market.
- **Pros:** Market expansion. Low product-risk (same binary, different packaging).
- **Cons:** Code signing certs ($300/yr), SmartScreen warnings, Windows Defender flags.
- **Context:** Deferred from Phase 1 since privacy-community users skew Linux/macOS. Revisit after beta feedback.
- **Effort:** M (human 2wk) → CC+gstack ~6-10h
- **Priority:** P1
- **Blocked by:** Phase 1 refactor landed + macOS/Linux installers proven

### [P2] Biweekly Upload Nudge (power user mode)
- **What:** Opt-in mode that reminds user every 14 days to upload their latest CSV export from Chase/Amex/etc., with one-click upload buttons.
- **Why:** Bridges monthly statement cadence to biweekly data freshness for users who want more proactive insights.
- **Pros:** Doubles data freshness without Plaid. Power-user feature that drives engagement.
- **Cons:** Friction is real — users will forget and disable it. Need deep-link CSV download URLs per bank.
- **Context:** Revisit after Month in Review ships and we see how often users actually upload.
- **Effort:** S (human 1wk) → CC+gstack ~3-5h
- **Priority:** P2
- **Blocked by:** Month in Review shipped + data cadence behavior observed

---

## Phase 3 (post-v1.0 launch)

### [P2] Merchant Normalization
- **What:** Map messy transaction descriptions ("AMZN MKTP US*A3R5K", "SQ *COFFEE SHOP DEN") to canonical merchants ("Amazon", "Coffee Shop").
- **Why:** Readability is table stakes. Every insight that mentions a merchant looks unprofessional without this.
- **Pros:** Massive readability improvement across entire product. Enables merchant-level logos.
- **Cons:** Requires either: a curated registry (tedious to maintain), scraping an open source (legal risk), or LLM per-transaction (slow, hallucinations).
- **Context:** Biggest "polish" win. Consider a community-curated YAML file in the repo.
- **Effort:** M (human 2wk) → CC+gstack ~8-12h
- **Priority:** P2

### [P2] Plugin Interface for Bank Parsers
- **What:** Define `ingest.Parser` Go interface; compile parsers as plugins or in-tree modules; allow community contributions without forking.
- **Why:** Parser maintenance is a long-tail problem. Every bank format change breaks a parser. Community help is the only scalable solution.
- **Pros:** Turns parser support into a distributed asset. Signals "open and extensible."
- **Cons:** Plugin security (loading arbitrary user code). Maintenance burden of stable plugin ABI.
- **Context:** Phase 3 because it needs the monolith stable first. Could be compile-time modules instead of runtime plugins.
- **Effort:** L (human 3-4wk) → CC+gstack ~10-15h
- **Priority:** P2

### [P2] Rules-as-YAML
- **What:** Insight engine rules stored as YAML in `~/.localfinance/rules/*.yaml`. Power users can write custom rules ("flag Amazon charges >$200 on weekends").
- **Why:** Differentiator for power users; makes the product feel extensible rather than a black box.
- **Pros:** Power user lock-in. Community rule sharing potential. Users who invest time in custom rules don't churn.
- **Cons:** YAML rule language design is a project of its own. Edge cases in composition/precedence.
- **Context:** Ship default built-in rules in Phase 2; add user rules in Phase 3 once the rule model has stabilized.
- **Effort:** M (human 2-3wk) → CC+gstack ~8-12h
- **Priority:** P2

### [P3] Standalone "What If" Simulator
- **What:** Dedicated page where user adjusts categories ("cut DoorDash 50%") and sees year-end impact.
- **Why:** Experimentation tool; helps users see the dollar value of behavior changes.
- **Pros:** Engagement. Shareable outputs ("I found $X I could save").
- **Cons:** Basic version ships in Savings Auto-Pilot; standalone page may not earn its complexity.
- **Context:** Revisit if users ask for it.
- **Effort:** S (human 1wk) → CC+gstack ~3-5h
- **Priority:** P3

### [P3] Couples Mode
- **What:** Two Macs in a household sync via end-to-end encrypted channel. Each partner has a personal view + joint view of shared accounts.
- **Why:** Huge market (couples manage money together). No privacy-first product solves this.
- **Pros:** Differentiation. Replaces Honeydue/Zeta. Opens to "household" not just "individual" market.
- **Cons:** E2E encryption design is hard. Sync conflict resolution. Support burden.
- **Context:** Phase 3. Requires careful design. Could drive real revenue.
- **Effort:** XL (human 6-8wk) → CC+gstack ~25-40h
- **Priority:** P3

### [P3] Sankey Monthly Flow Visualization
- **What:** Visual river diagram of income sources → categories → specific merchants. Beautiful "where did my money go."
- **Why:** Screenshot-worthy, evangelism fuel, unique in privacy-finance apps.
- **Pros:** Instagram/Twitter-ready output. Unique look.
- **Cons:** D3.js complexity. Doesn't fit small screens well.
- **Context:** Pure delight; not part of core value loop.
- **Effort:** M (human 1-2wk) → CC+gstack ~5-8h
- **Priority:** P3

### [P3] Voice Input on Chat
- **What:** Press-to-talk on the chat UI. Local speech-to-text (Whisper).
- **Why:** Conversational UX; "talk to your money while cooking."
- **Pros:** Delightful. Differentiator.
- **Cons:** Whisper + model memory on lower-RAM Macs is tight. Accuracy risk.
- **Context:** Depends on model memory budget post-insights.
- **Effort:** M (human 2wk) → CC+gstack ~6-10h
- **Priority:** P3

### [P3] Travel / FX Mode
- **What:** Transactions in foreign currencies auto-categorized, with FX rate capture at transaction date.
- **Why:** Travelers have messy transaction data. Specific pain point.
- **Pros:** Clear user segment it serves.
- **Cons:** FX rate source needs to be local (offline) or cached; adding a dependency.
- **Context:** Narrow audience; defer.
- **Effort:** M (human 2wk) → CC+gstack ~5-8h
- **Priority:** P3

### [P3] Cash Flow Forecast (30/60/90)
- **What:** Predict next 30/60/90 day cash position based on recurring income + recurring expenses + trend.
- **Why:** Forward-looking answers the "will I have enough?" question.
- **Pros:** High perceived value. Concrete decision support.
- **Cons:** Requires 6+ months of user data for any accuracy. Wrong forecasts erode trust fast.
- **Context:** Phase 3 since data maturity matters.
- **Effort:** M-L (human 2-3wk) → CC+gstack ~8-12h
- **Priority:** P3

### [P3] Receipt OCR
- **What:** Scan a receipt photo, extract line items, match to transaction.
- **Why:** Line-item detail enables richer insights (caught 3 coffees on what looked like a single restaurant charge).
- **Pros:** Power feature. Unique in privacy-finance.
- **Cons:** Local OCR quality. Matching algorithm. UI complexity.
- **Context:** Phase 3; depends on mobile companion.
- **Effort:** L (human 3-4wk) → CC+gstack ~12-18h
- **Priority:** P3

### [P3] Email Forwarding for Near-Real-Time Data
- **What:** User forwards Chase/Amex transaction emails to an address the LocalFinance app listens on; app parses alerts for real-time transactions.
- **Why:** Closes the data-freshness gap without Plaid.
- **Pros:** Real-time finance data while staying privacy-first (email is user-owned).
- **Cons:** Local mail server = attack surface. Email parsing is fragile. Not all banks send detailed alerts.
- **Context:** Phase 3 after security design.
- **Effort:** L (human 3-4wk) → CC+gstack ~15-20h
- **Priority:** P3

### [P2] Category rules UI + business-logic layer
- **What:** User-facing UI to create/edit category rules (regex/amount/merchant matchers). Extract category-decision logic from Logos into a shared package the chat, insights, and parse paths all use.
- **Why:** Today the schema and the Logos `applyUserRules` path exist but rules can only be added via direct API call. User wants to customize categories and have AI parsing respect those preferences.
- **Pros:** Closes the loop on "personalized finance." Makes the LLM's category guesses authoritative-but-overridable. One place to change category logic.
- **Cons:** Designing the rule expression language is a project of its own (see existing Rules-as-YAML TODO). Risk of accidentally tying chat narration to rule changes.
- **Context:** User flagged this as the eventual direction during Phase 1 AI-parse work on 2026-04-27. Prerequisites: AI parse path landed (Logos→Sophia /api/v1/parse).
- **Effort:** L (human 3-4wk) → CC+gstack ~10-15h
- **Priority:** P2
- **Blocked by:** AI parse path landed
