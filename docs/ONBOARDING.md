# Onboarding

## How We Help Users Get Started

### Onboarding Philosophy

> **Goal:** User asks their first AI question within 5 minutes of download.

We optimize for:
1. **Speed to value** — Don't make them wait
2. **Confidence** — They know it's working
3. **Delight** — First AI response should impress

---

## Onboarding Flow

### Step 1: Download & Install (1-2 min)

**What User Does:**
- Downloads from our website
- Runs installer

**What We Do:**
- Single installer (no dependencies to install separately)
- Model bundled OR downloaded on first run
- No sign-up required (no email, no account)

**Key Decision:**
| Option | Pros | Cons |
|--------|------|------|
| Bundle model in installer | Instant setup | 800MB download |
| Download model on first run | Smaller initial download | Extra step |

**Recommendation:** Offer both. Default to bundled, option for "lite" download.

---

### Step 2: First Run Experience

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│                    💰 Welcome to LocalFinance                    │
│                                                                  │
│         Your money. Your data. Your device.                     │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                                                          │    │
│  │   Let's get you set up in under 5 minutes.              │    │
│  │                                                          │    │
│  │   All your data stays on this device. Always.           │    │
│  │                                                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│                      [ Get Started ]                             │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

### Step 3: Model Setup (if not bundled)

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│                    🧠 Downloading AI Model                       │
│                                                                  │
│  This is a one-time download (800MB).                           │
│  The AI runs 100% on your device — no internet needed after.   │
│                                                                  │
│  [████████████████░░░░░░░░] 67%                                 │
│  Estimated time: 45 seconds                                      │
│                                                                  │
│                                                                  │
│  While you wait:                                                 │
│  ───────────────                                                 │
│  📄 Download a recent bank statement (PDF or CSV)               │
│     from your bank's website. You'll import it next.            │
│                                                                  │
│  🔒 Your statements never leave your computer.                  │
│     We don't have servers. There's nowhere to send it.          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

### Step 4: First Import

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│                    📄 Import Your First Statement                │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                                                          │    │
│  │                                                          │    │
│  │         Drag a bank statement here                       │    │
│  │              (PDF or CSV)                                │    │
│  │                                                          │    │
│  │         or click to browse                               │    │
│  │                                                          │    │
│  │                                                          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  Supported banks: Chase, Bank of America, Capital One,          │
│  American Express, Discover, Wells Fargo, and more.             │
│                                                                  │
│  ─────────────────────────────────────────────────────────────  │
│                                                                  │
│  Don't have a statement handy?                                  │
│  [ Use sample data to try it out ]                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Sample Data Option:**
- Pre-loaded with 3 months of realistic (fake) transactions
- Lets user try AI queries immediately
- Can be cleared when they import real data

---

### Step 5: Review Import

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│  ✅ Found 47 transactions                                       │
│                                                                  │
│  Account: Chase Checking ••4892                                 │
│  Period: January 1 - January 31, 2026                           │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Date     Description              Amount    Category     │    │
│  │ ─────────────────────────────────────────────────────── │    │
│  │ Jan 03   TRADER JOES #123        -$87.43   🛒 Groceries │    │
│  │ Jan 05   UBER EATS               -$34.21   🍕 Dining    │    │
│  │ Jan 07   SHELL OIL 12345         -$52.00   ⛽ Gas       │    │
│  │ Jan 08   NETFLIX                  -$15.99   📺 Subscr.  │    │
│  │ Jan 10   VENMO PAYMENT           -$50.00   ❓ Transfer  │ ← │
│  │ ...                                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  ⚠️  5 transactions need categories. [Review now] or [Later]   │
│                                                                  │
│                    [ Import & Continue ]                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

### Step 6: First AI Interaction

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│  🎉 You're all set!                                             │
│                                                                  │
│  Try asking me something:                                        │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ How much did I spend last month?                     🎤 │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  Or try one of these:                                            │
│  ┌───────────────────────────────────────┐                      │
│  │ What are my top spending categories?  │                      │
│  └───────────────────────────────────────┘                      │
│  ┌───────────────────────────────────────┐                      │
│  │ Show me all my dining transactions   │                      │
│  └───────────────────────────────────────┘                      │
│  ┌───────────────────────────────────────┐                      │
│  │ Am I spending more than last month?   │                      │
│  └───────────────────────────────────────┘                      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Support Channels

### Self-Service (Free)
- In-app help tooltips
- FAQ / Knowledge base
- Video tutorials (YouTube)
- Community Discord

### Assisted Onboarding (Pro)
- Email support
- Screen-share setup sessions (first week)
- Custom parser requests (for unsupported banks)

---

## Onboarding Metrics

| Metric | Target |
|--------|--------|
| Time to first import | < 5 minutes |
| Time to first AI query | < 7 minutes |
| Import success rate | > 95% |
| First-week retention | > 70% |
| Support tickets (week 1) | < 10% of users |

---

## Handling Edge Cases

### Unsupported Bank
```
"We don't recognize this statement format yet.

Options:
1. [Request support for this bank] — We'll add it within 1 week
2. [Import as CSV] — Export from your bank as CSV instead
3. [Manual entry] — Add transactions manually"
```

### Import Errors
```
"We had trouble reading this PDF.

This sometimes happens with:
• Scanned/image PDFs (we need text-based)
• Password-protected files
• Very old statement formats

Try: Download a fresh statement from your bank's website,
or [contact support] for help."
```

### Model Download Fails
```
"Download interrupted. 

[ Retry ] or [ Download manually ]

Manual download: 
1. Download from: https://localfinance.app/models/finance-v1.gguf
2. Place in: ~/.localfinance/models/
3. Restart the app"
```

---

## Post-Onboarding Nurturing

### Day 2: Check-in
```
"Welcome back! You've imported 47 transactions.

Quick insight: Your top spending category is Dining ($312).

Try asking: 'How does my dining spending compare to last month?'"
```

### Day 7: Monthly Ritual Reminder
```
"Tip: For the best insights, import statements from all your accounts.

You've imported:
✅ Chase Checking

Consider adding:
• Credit card statements
• Savings accounts
• Investment accounts (for net worth tracking)"
```

### Day 30: Feature Discovery
```
"Did you know you can set savings goals?

Try: 'Create an emergency fund goal for $10,000'

I'll track your progress and help you stay on target."
```
