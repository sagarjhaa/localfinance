# User Flow

## The Customer Journey

### Phase 1: Discovery → Purchase

```
┌─────────────────────────────────────────────────────────────────┐
│  DISCOVERY                                                       │
│  "I want to track my finances but I don't trust Mint/YNAB       │
│   with my bank credentials"                                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  LANDING PAGE                                                    │
│  - See product demo                                              │
│  - "Your money. Your data. Your device."                        │
│  - See example AI conversations                                  │
│  - Compare: LocalFinance vs Mint vs YNAB                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  PURCHASE                                                        │
│  - Free tier: Statement import + basic queries                  │
│  - Pro ($29 one-time): Full AI assistant                        │
│  - Download installer (macOS/Windows/Linux)                     │
└─────────────────────────────────────────────────────────────────┘
```

---

### Phase 2: Setup (5 minutes)

```
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: INSTALL (1 min)                                        │
│                                                                  │
│  macOS: Drag to Applications                                    │
│  Windows: Run installer                                          │
│  Linux: AppImage or pip install                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 2: DOWNLOAD AI MODEL (2 min)                              │
│                                                                  │
│  "Downloading Finance AI (800MB)..."                            │
│  [████████████████████████] 100%                                │
│                                                                  │
│  ✓ Model ready! This runs 100% on your device.                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 3: IMPORT FIRST STATEMENT (2 min)                         │
│                                                                  │
│  "Drag a bank statement here, or click to browse"              │
│                                                                  │
│  [  📄 Drop PDF/CSV here  ]                                     │
│                                                                  │
│  Don't have one handy? [Use sample data]                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: VERIFY IMPORT                                          │
│                                                                  │
│  Found 47 transactions from Chase Checking                      │
│  Date range: Jan 1 - Jan 31, 2026                               │
│                                                                  │
│  Sample transactions:                                            │
│  ✓ Jan 03  TRADER JOES #123      -$87.43    Groceries          │
│  ✓ Jan 05  UBER EATS             -$34.21    Dining             │
│  ✓ Jan 07  SHELL OIL             -$52.00    Gas                │
│                                                                  │
│  [Looks good!]  [Edit categories]                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  🎉 SETUP COMPLETE!                                             │
│                                                                  │
│  Try asking:                                                     │
│  • "How much did I spend last month?"                           │
│  • "What are my top spending categories?"                       │
│  • "Show me all dining transactions"                            │
└─────────────────────────────────────────────────────────────────┘
```

---

### Phase 3: Daily Use

#### Importing New Statements

```
┌─────────────────────────────────────────────────────────────────┐
│  MONTHLY RITUAL (2 min/month)                                    │
│                                                                  │
│  1. Download statement from bank website (PDF or CSV)           │
│  2. Drag into LocalFinance                                       │
│  3. Review auto-categorization                                   │
│  4. Done!                                                        │
│                                                                  │
│  Pro tip: Set a monthly reminder to import statements           │
└─────────────────────────────────────────────────────────────────┘
```

#### Asking Questions

```
┌─────────────────────────────────────────────────────────────────┐
│  NATURAL LANGUAGE QUERIES                                        │
│                                                                  │
│  You: "Am I spending more on groceries this month?"             │
│                                                                  │
│  AI: "Yes, you've spent $623 on groceries so far this month,   │
│       which is 34% higher than your average of $465/month.      │
│                                                                  │
│       Top merchants this month:                                  │
│       • Whole Foods: $287 (3 visits)                            │
│       • Costco: $198 (1 visit)                                  │
│       • Safeway: $138 (4 visits)"                               │
│                                                                  │
│  You: "Why is Whole Foods so high?"                             │
│                                                                  │
│  AI: "Your 3 Whole Foods transactions this month:               │
│       • Jan 15: $142.33                                          │
│       • Jan 22: $87.21                                           │
│       • Jan 28: $57.46                                           │
│                                                                  │
│       The Jan 15 purchase was unusually large compared to       │
│       your typical $60-80 visits."                               │
└─────────────────────────────────────────────────────────────────┘
```

#### Setting Goals

```
┌─────────────────────────────────────────────────────────────────┐
│  GOAL TRACKING                                                   │
│                                                                  │
│  You: "Create an emergency fund goal for $15,000"               │
│                                                                  │
│  AI: "Got it! I've created an emergency fund goal:              │
│       Target: $15,000                                            │
│       Current: $0                                                │
│                                                                  │
│       Want to set a target date or monthly contribution?"       │
│                                                                  │
│  You: "I want to save $500/month toward it"                     │
│                                                                  │
│  AI: "Perfect. At $500/month, you'll reach $15,000 in 30        │
│       months (July 2028).                                        │
│                                                                  │
│       I'll track your progress and let you know each month      │
│       how you're doing."                                         │
└─────────────────────────────────────────────────────────────────┘
```

---

### Phase 4: Ongoing Value

#### Weekly Check-in
```
User opens app on Sunday evening:

AI: "Here's your week in review:
     
     💰 Spent: $847.32 (within your typical range)
     📊 Top category: Groceries ($234)
     ⚠️  Dining is 23% over your monthly budget
     ✅ Emergency fund: $8,900 / $15,000 (59%)
     
     Want details on anything?"
```

#### Month-End Summary
```
AI: "January 2026 Summary:
     
     Income: $6,200
     Expenses: $4,847
     Saved: $1,353 (21.8%)
     
     50/30/20 Breakdown:
     • Needs: $2,340 (48%) ✅
     • Wants: $1,607 (33%) ⚠️ slightly over
     • Savings: $900 (19%) ✅ close to target
     
     Net worth change: +$1,203
     
     Shall I show spending by category?"
```

---

## User Personas

### Persona 1: Privacy-Conscious Professional
- **Who:** 30-45, tech-savvy, income $80-150k
- **Pain:** Doesn't trust Mint/Plaid with bank credentials
- **Need:** Track spending without cloud dependency
- **Value:** Privacy, local control, no subscription

### Persona 2: FIRE Enthusiast
- **Who:** 25-40, saving aggressively, tracks net worth
- **Pain:** Spreadsheets are tedious, apps lack flexibility
- **Need:** Custom queries, goal tracking, net worth trends
- **Value:** Flexibility, detailed analysis, long-term tracking

### Persona 3: Small Business Owner
- **Who:** Self-employed, mixes personal/business finances
- **Pain:** Needs to separate expenses for taxes
- **Need:** Categorization, reporting, export for accountant
- **Value:** Organization, tax prep, simplicity

---

## Key Moments That Matter

| Moment | User Feeling | Our Goal |
|--------|--------------|----------|
| First import | "Will this actually work?" | Instant success, clear feedback |
| First AI query | "Is this magic?" | Accurate, helpful answer |
| Finding an insight | "I didn't know that!" | Surface useful patterns |
| Monthly ritual | "This is easy" | < 5 min to update everything |
| Recommending to friend | "You should try this" | Shareable, quotable value |
