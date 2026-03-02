#!/bin/bash
# Quick test of LocalFinance AI concept
# Run: ./test_today.sh

echo "=============================================="
echo "🏦 LocalFinance AI - Quick Test"
echo "=============================================="

# Test question
QUESTION="$1"
if [ -z "$QUESTION" ]; then
    QUESTION="How much did I spend on groceries?"
fi

echo ""
echo "❓ Question: $QUESTION"
echo ""

# Send to Ollama with our schema
PROMPT="You are a SQL expert. Convert this question to SQLite:

Schema:
- transactions(date, description, amount, category)
- accounts(name, type, balance)
- goals(name, target_amount, current_amount)

Examples:
Q: How much did I spend on groceries?
A: SELECT SUM(amount) FROM transactions WHERE category='Groceries'

Q: Show Amazon purchases
A: SELECT date, amount FROM transactions WHERE description LIKE '%AMAZON%'

Now convert this question to SQL only (no explanation):
Q: $QUESTION
A:"

echo "📝 Sending to Ollama..."
echo ""

SQL=$(ollama run llama3.2 "$PROMPT" 2>/dev/null | head -5)

echo "SQL Generated:"
echo "━━━━━━━━━━━━━━"
echo "$SQL"
echo "━━━━━━━━━━━━━━"
