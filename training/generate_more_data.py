#!/usr/bin/env python3
"""
Extended training data generator - More natural queries people actually ask.
"""

import json
import random
from pathlib import Path

OUTPUT_DIR = Path(__file__).parent / "data"
OUTPUT_DIR.mkdir(exist_ok=True)

def save_example(category: str, question: str, sql: str):
    filepath = OUTPUT_DIR / f"{category}.jsonl"
    example = {"instruction": question, "output": sql, "category": category}
    with open(filepath, "a") as f:
        f.write(json.dumps(example) + "\n")

def generate_merchant_queries():
    """Queries about specific merchants/stores."""
    
    merchants = [
        ("Amazon", "AMAZON"),
        ("Costco", "COSTCO"),
        ("Target", "TARGET"),
        ("Walmart", "WALMART"),
        ("Starbucks", "STARBUCKS"),
        ("Uber", "UBER"),
        ("Lyft", "LYFT"),
        ("Netflix", "NETFLIX"),
        ("Spotify", "SPOTIFY"),
        ("Apple", "APPLE"),
        ("Google", "GOOGLE"),
        ("DoorDash", "DOORDASH"),
        ("Grubhub", "GRUBHUB"),
        ("Whole Foods", "WHOLE FOODS"),
        ("Trader Joe's", "TRADER JOE"),
        ("CVS", "CVS"),
        ("Walgreens", "WALGREENS"),
    ]
    
    templates = [
        ("How much did I spend at {name}?", 
         "SELECT SUM(amount) as total FROM transactions WHERE description LIKE '%{pattern}%'"),
        ("Show me all {name} purchases",
         "SELECT date, description, amount FROM transactions WHERE description LIKE '%{pattern}%' ORDER BY date DESC"),
        ("How many times did I shop at {name}?",
         "SELECT COUNT(*) as visits FROM transactions WHERE description LIKE '%{pattern}%'"),
        ("What's my average {name} purchase?",
         "SELECT AVG(amount) as avg_purchase FROM transactions WHERE description LIKE '%{pattern}%'"),
        ("{name} spending this month?",
         "SELECT SUM(amount) as total FROM transactions WHERE description LIKE '%{pattern}%' AND date >= date('now', 'start of month')"),
        ("Last {name} purchase?",
         "SELECT date, amount FROM transactions WHERE description LIKE '%{pattern}%' ORDER BY date DESC LIMIT 1"),
    ]
    
    count = 0
    for name, pattern in merchants:
        for template_q, template_sql in templates:
            question = template_q.format(name=name)
            sql = template_sql.format(pattern=pattern)
            save_example("merchant_queries", question, sql)
            count += 1
    
    return count

def generate_budget_queries():
    """50/30/20 and budget-related queries."""
    
    queries = [
        # 50/30/20 analysis
        ("How am I doing on the 50/30/20 rule?",
         """SELECT 
            bucket,
            SUM(t.amount) as spent,
            ROUND(SUM(t.amount) * 100.0 / (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')), 1) as pct
         FROM transactions t
         JOIN budget_categories bc ON t.category = bc.category
         WHERE t.date >= date('now', 'start of month')
         GROUP BY bucket"""),
        
        ("What percentage goes to needs vs wants?",
         """SELECT bucket, ROUND(SUM(amount) * 100.0 / (SELECT SUM(amount) FROM transactions), 1) as pct
         FROM transactions t JOIN budget_categories bc ON t.category = bc.category
         GROUP BY bucket"""),
        
        ("Am I overspending on wants?",
         """SELECT SUM(amount) as wants_spending,
            (SELECT SUM(amount) * 0.30 FROM transactions WHERE date >= date('now', 'start of month')) as wants_budget
         FROM transactions t JOIN budget_categories bc ON t.category = bc.category
         WHERE bucket = 'wants' AND date >= date('now', 'start of month')"""),
         
        # Budget tracking
        ("Am I over budget?",
         """SELECT category, SUM(amount) as spent, bc.monthly_budget,
            SUM(amount) - bc.monthly_budget as over_under
         FROM transactions t JOIN budget_categories bc ON t.category = bc.category
         WHERE date >= date('now', 'start of month')
         GROUP BY category
         HAVING monthly_budget IS NOT NULL"""),
        
        ("Which categories am I overspending on?",
         """SELECT category, SUM(amount) as spent, bc.monthly_budget
         FROM transactions t JOIN budget_categories bc ON t.category = bc.category
         WHERE date >= date('now', 'start of month')
         GROUP BY category
         HAVING SUM(amount) > bc.monthly_budget"""),
        
        ("How much budget do I have left for dining?",
         """SELECT bc.monthly_budget - COALESCE(SUM(t.amount), 0) as remaining
         FROM budget_categories bc
         LEFT JOIN transactions t ON t.category = bc.category AND t.date >= date('now', 'start of month')
         WHERE bc.category = 'Dining'"""),
        
        ("Set my grocery budget to $500",
         "UPDATE budget_categories SET monthly_budget = 500 WHERE category = 'Groceries'"),
        
        ("What's my dining budget?",
         "SELECT monthly_budget FROM budget_categories WHERE category = 'Dining'"),
    ]
    
    for question, sql in queries:
        save_example("budget_queries", question, sql)
    
    return len(queries)

def generate_time_queries():
    """Queries with various time expressions."""
    
    time_expressions = [
        ("this week", "date >= date('now', 'weekday 0', '-7 days')"),
        ("last week", "date >= date('now', 'weekday 0', '-14 days') AND date < date('now', 'weekday 0', '-7 days')"),
        ("this quarter", "date >= date('now', 'start of month', '-' || ((strftime('%m', 'now') - 1) % 3) || ' months')"),
        ("Q1", "date >= '2026-01-01' AND date < '2026-04-01'"),
        ("the last 6 months", "date >= date('now', '-6 months')"),
        ("since January", "date >= '2026-01-01'"),
        ("before December", "date < '2025-12-01'"),
        ("on my birthday", "strftime('%m-%d', date) = '03-15'"),  # example date
        ("on weekends", "strftime('%w', date) IN ('0', '6')"),
        ("on Fridays", "strftime('%w', date) = '5'"),
    ]
    
    count = 0
    for time_text, time_sql in time_expressions:
        question = f"How much did I spend {time_text}?"
        sql = f"SELECT SUM(amount) as total FROM transactions WHERE {time_sql}"
        save_example("time_queries", question, sql)
        count += 1
    
    return count

def generate_insight_queries():
    """Queries asking for insights and patterns."""
    
    queries = [
        ("What's my biggest expense this month?",
         "SELECT description, amount, date FROM transactions WHERE date >= date('now', 'start of month') ORDER BY amount DESC LIMIT 1"),
        
        ("What are my most frequent purchases?",
         "SELECT description, COUNT(*) as frequency, SUM(amount) as total FROM transactions GROUP BY description ORDER BY frequency DESC LIMIT 10"),
        
        ("Where am I wasting money?",
         """SELECT category, SUM(amount) as total 
         FROM transactions 
         WHERE category IN ('Dining', 'Entertainment', 'Shopping', 'Subscriptions')
         AND date >= date('now', '-3 months')
         GROUP BY category ORDER BY total DESC"""),
        
        ("What subscriptions am I paying for?",
         "SELECT DISTINCT description, amount FROM transactions WHERE category = 'Subscriptions' ORDER BY amount DESC"),
        
        ("How much do I spend on subscriptions monthly?",
         "SELECT SUM(amount) as monthly_subscriptions FROM recurring_transactions WHERE category = 'Subscriptions' AND frequency = 'monthly'"),
        
        ("What's my daily average spending?",
         "SELECT ROUND(AVG(daily_total), 2) as avg_daily FROM (SELECT date, SUM(amount) as daily_total FROM transactions GROUP BY date)"),
        
        ("What day do I spend the most?",
         """SELECT 
            CASE strftime('%w', date)
                WHEN '0' THEN 'Sunday'
                WHEN '1' THEN 'Monday'
                WHEN '2' THEN 'Tuesday'
                WHEN '3' THEN 'Wednesday'
                WHEN '4' THEN 'Thursday'
                WHEN '5' THEN 'Friday'
                WHEN '6' THEN 'Saturday'
            END as day_of_week,
            SUM(amount) as total
         FROM transactions GROUP BY strftime('%w', date) ORDER BY total DESC LIMIT 1"""),
        
        ("Find unusual transactions",
         """SELECT * FROM transactions 
         WHERE amount > (SELECT AVG(amount) + 2 * 
            (SELECT AVG(amount * amount) - AVG(amount) * AVG(amount) FROM transactions) 
         FROM transactions)
         ORDER BY amount DESC"""),
        
        ("What's my spending pattern?",
         """SELECT strftime('%w', date) as day, COUNT(*) as transactions, SUM(amount) as total
         FROM transactions GROUP BY day ORDER BY day"""),
        
        ("When do I shop the most?",
         """SELECT strftime('%H', datetime) as hour, COUNT(*) as count
         FROM transactions GROUP BY hour ORDER BY count DESC LIMIT 5"""),
    ]
    
    for question, sql in queries:
        save_example("insight_queries", question, sql)
    
    return len(queries)

def generate_comparison_queries():
    """Queries comparing time periods or categories."""
    
    queries = [
        ("Compare this month to last month",
         """SELECT 
            'This Month' as period, SUM(amount) as total FROM transactions WHERE date >= date('now', 'start of month')
         UNION ALL
         SELECT 
            'Last Month', SUM(amount) FROM transactions WHERE date >= date('now', '-1 month', 'start of month') AND date < date('now', 'start of month')"""),
        
        ("How does my spending compare to 3 months ago?",
         """SELECT strftime('%Y-%m', date) as month, SUM(amount) as total
         FROM transactions 
         WHERE date >= date('now', '-3 months')
         GROUP BY month ORDER BY month"""),
        
        ("Am I spending more on dining or groceries?",
         """SELECT category, SUM(amount) as total FROM transactions 
         WHERE category IN ('Dining', 'Groceries') GROUP BY category"""),
        
        ("Which credit card do I use most?",
         """SELECT source, COUNT(*) as transactions, SUM(amount) as total
         FROM transactions GROUP BY source ORDER BY transactions DESC"""),
        
        ("Compare weekday vs weekend spending",
         """SELECT 
            CASE WHEN strftime('%w', date) IN ('0', '6') THEN 'Weekend' ELSE 'Weekday' END as day_type,
            SUM(amount) as total, AVG(amount) as avg_transaction
         FROM transactions GROUP BY day_type"""),
         
        ("Year over year comparison",
         """SELECT strftime('%Y', date) as year, SUM(amount) as total
         FROM transactions GROUP BY year ORDER BY year"""),
    ]
    
    for question, sql in queries:
        save_example("comparison_queries", question, sql)
    
    return len(queries)

def generate_transaction_queries():
    """Queries about finding specific transactions."""
    
    queries = [
        ("Show me my last 10 transactions",
         "SELECT date, description, amount, category FROM transactions ORDER BY date DESC LIMIT 10"),
        
        ("Find all transactions over $100",
         "SELECT date, description, amount FROM transactions WHERE amount > 100 ORDER BY amount DESC"),
        
        ("Find all transactions under $10",
         "SELECT date, description, amount FROM transactions WHERE amount < 10 AND amount > 0 ORDER BY date DESC"),
        
        ("Show me refunds",
         "SELECT date, description, amount FROM transactions WHERE amount < 0 ORDER BY date DESC"),
        
        ("Find duplicate transactions",
         """SELECT description, amount, COUNT(*) as count FROM transactions 
         GROUP BY description, amount HAVING count > 1"""),
        
        ("Search for transactions with 'coffee'",
         "SELECT date, description, amount FROM transactions WHERE description LIKE '%coffee%' ORDER BY date DESC"),
        
        ("What did I buy on January 15th?",
         "SELECT description, amount, category FROM transactions WHERE date = '2026-01-15'"),
        
        ("Show me all uncategorized transactions",
         "SELECT date, description, amount FROM transactions WHERE category IS NULL OR category = 'Other'"),
        
        ("Large purchases over $500",
         "SELECT date, description, amount, category FROM transactions WHERE amount > 500 ORDER BY amount DESC"),
        
        ("Find all ATM withdrawals",
         "SELECT date, description, amount FROM transactions WHERE description LIKE '%ATM%' OR description LIKE '%WITHDRAW%'"),
        
        ("Show me pending transactions",
         "SELECT date, description, amount FROM transactions WHERE status = 'pending'"),
    ]
    
    for question, sql in queries:
        save_example("transaction_queries", question, sql)
    
    return len(queries)

def generate_natural_variations():
    """Natural language variations of common queries."""
    
    variations = [
        # Casual ways to ask about spending
        ("What'd I spend this month?", "SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')"),
        ("Where'd my money go?", "SELECT category, SUM(amount) as total FROM transactions WHERE date >= date('now', 'start of month') GROUP BY category ORDER BY total DESC"),
        ("Damage report", "SELECT SUM(amount) as total_spent FROM transactions WHERE date >= date('now', 'start of month')"),
        ("How broke am I?", "SELECT balance FROM v_current_balances WHERE type = 'checking'"),
        ("Money left?", "SELECT SUM(balance) FROM v_current_balances WHERE type IN ('checking', 'savings')"),
        
        # Goal-related casual
        ("Hows my savings looking?", "SELECT name, current_amount, target_amount FROM goals WHERE is_active = 1"),
        ("Emergency fund status", "SELECT current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as pct FROM goals WHERE name LIKE '%emergency%'"),
        
        # Credit card casual
        ("Credit card debt?", "SELECT SUM(balance) FROM v_current_balances WHERE type = 'credit_card'"),
        ("How much do I owe?", "SELECT name, balance FROM v_current_balances WHERE type IN ('credit_card', 'loan')"),
        ("Cards maxed out?", "SELECT name, balance, credit_limit, ROUND(balance * 100.0 / credit_limit, 1) as utilization FROM v_credit_card_summary WHERE utilization > 80"),
        
        # Trend casual
        ("Spending going up or down?", 
         """SELECT strftime('%Y-%m', date) as month, SUM(amount) as total 
         FROM transactions GROUP BY month ORDER BY month DESC LIMIT 3"""),
        ("Better or worse than last month?",
         """SELECT 
            (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')) as this_month,
            (SELECT SUM(amount) FROM transactions WHERE date >= date('now', '-1 month', 'start of month') AND date < date('now', 'start of month')) as last_month"""),
        
        # Typos and informal
        ("groceries spending", "SELECT SUM(amount) FROM transactions WHERE category = 'Groceries'"),
        ("amazon", "SELECT SUM(amount) FROM transactions WHERE description LIKE '%AMAZON%'"),
        ("uber eats total", "SELECT SUM(amount) FROM transactions WHERE description LIKE '%UBER EATS%'"),
        ("starbucks addiction", "SELECT SUM(amount) as total, COUNT(*) as visits FROM transactions WHERE description LIKE '%STARBUCKS%'"),
    ]
    
    for question, sql in variations:
        save_example("natural_queries", question, sql)
    
    return len(variations)

def generate_action_queries():
    """Queries that modify data or take action."""
    
    queries = [
        ("Categorize this as groceries",
         "UPDATE transactions SET category = 'Groceries' WHERE id = ?"),
        
        ("Mark this as a business expense",
         "UPDATE transactions SET category = 'Business', tags = tags || ',business' WHERE id = ?"),
        
        ("Add a note to my last transaction",
         "UPDATE transactions SET note = ? WHERE id = (SELECT id FROM transactions ORDER BY date DESC LIMIT 1)"),
        
        ("Delete duplicate transactions",
         """DELETE FROM transactions WHERE id NOT IN (
            SELECT MIN(id) FROM transactions GROUP BY date, description, amount
         )"""),
        
        ("Create a vacation fund for $3000",
         "INSERT INTO goals (id, name, target_amount, current_amount, category) VALUES (?, 'Vacation Fund', 3000, 0, 'vacation')"),
        
        ("Add $500 to my emergency fund",
         "UPDATE goals SET current_amount = current_amount + 500 WHERE name LIKE '%emergency%'"),
        
        ("Set dining budget to $400",
         "UPDATE budget_categories SET monthly_budget = 400 WHERE category = 'Dining'"),
    ]
    
    for question, sql in queries:
        save_example("action_queries", question, sql)
    
    return len(queries)

def combine_all():
    combined = OUTPUT_DIR / "combined.jsonl"
    all_examples = []
    
    for f in OUTPUT_DIR.glob("*.jsonl"):
        if f.name == "combined.jsonl":
            continue
        with open(f) as fp:
            for line in fp:
                all_examples.append(json.loads(line))
    
    random.shuffle(all_examples)
    
    with open(combined, "w") as f:
        for ex in all_examples:
            f.write(json.dumps(ex) + "\n")
    
    return len(all_examples)

def main():
    print("🚀 Generating MORE training data...\n")
    
    counts = {}
    counts["merchant"] = generate_merchant_queries()
    print(f"✅ Merchant queries: {counts['merchant']}")
    
    counts["budget"] = generate_budget_queries()
    print(f"✅ Budget queries: {counts['budget']}")
    
    counts["time"] = generate_time_queries()
    print(f"✅ Time queries: {counts['time']}")
    
    counts["insight"] = generate_insight_queries()
    print(f"✅ Insight queries: {counts['insight']}")
    
    counts["comparison"] = generate_comparison_queries()
    print(f"✅ Comparison queries: {counts['comparison']}")
    
    counts["transaction"] = generate_transaction_queries()
    print(f"✅ Transaction queries: {counts['transaction']}")
    
    counts["natural"] = generate_natural_variations()
    print(f"✅ Natural variations: {counts['natural']}")
    
    counts["action"] = generate_action_queries()
    print(f"✅ Action queries: {counts['action']}")
    
    print(f"\n📦 Combining all data...")
    total = combine_all()
    
    print(f"\n🎉 TOTAL: {total} training examples!")
    
    print(f"\n📂 Files:")
    for f in sorted(OUTPUT_DIR.glob("*.jsonl")):
        count = sum(1 for _ in open(f))
        print(f"   {f.name}: {count}")

if __name__ == "__main__":
    main()
