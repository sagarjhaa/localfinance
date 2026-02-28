# Model Training

## What We Tell Customers

> **"LocalFinance AI"** — A custom-trained language model built specifically for personal finance queries. Runs 100% on your device.

### Technical Details (for the curious)
- Base: Qwen2-0.5B (open source, MIT licensed)
- Fine-tuned on 5,000+ finance Q&A examples
- Optimized for natural language → SQL translation
- Size: ~400MB (fits on any device)
- Runs offline via llama.cpp

### Why This Model?
- Small enough for phones/laptops
- Smart enough for finance queries
- Open source = no licensing issues
- We own the fine-tuned weights

---

## Training Data Location

All training data is in this folder:

```
training/
├── README.md              # You are here
├── data/
│   ├── spending_queries.jsonl    # "How much did I spend..."
│   ├── category_queries.jsonl    # "What category..." 
│   ├── trend_queries.jsonl       # "Am I spending more..."
│   ├── goal_queries.jsonl        # "How close am I to..."
│   ├── account_queries.jsonl     # "What's my balance..."
│   └── combined.jsonl            # All data merged
├── generate_data.py       # Script to generate examples
├── train.py               # Training script
└── progress.log           # Training progress
```

---

## Progress Tracking

### Data Generation Status
| Category | Target | Generated | Status |
|----------|--------|-----------|--------|
| Spending queries | 1,000 | 0 | 🔴 Not started |
| Category queries | 800 | 0 | 🔴 Not started |
| Trend queries | 800 | 0 | 🔴 Not started |
| Goal queries | 600 | 0 | 🔴 Not started |
| Account queries | 600 | 0 | 🔴 Not started |
| Edge cases | 200 | 0 | 🔴 Not started |
| **Total** | **5,000** | **0** | 🔴 |

### Training Status
- [ ] Data generation complete
- [ ] Base model downloaded
- [ ] Training started
- [ ] Training complete
- [ ] Model exported to GGUF
- [ ] Testing passed

---

## View Progress

```bash
# See generated data
cat training/data/spending_queries.jsonl | head -20

# Count examples
wc -l training/data/*.jsonl

# Watch training (when running)
tail -f training/progress.log
```
