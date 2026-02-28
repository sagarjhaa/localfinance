#!/usr/bin/env python3
"""
Generate training data for LocalFinance AI model.

Each example is a (question, SQL) pair that teaches the model
how to translate natural language into database queries.

Run: python generate_data.py
Output: data/*.jsonl files
"""

import json
import random
from pathlib import Path
from datetime import datetime

OUTPUT_DIR = Path(__file__).parent / "data"
OUTPUT_DIR.mkdir(exist_ok=True)

# Track statistics
stats = {
    "spending_queries": 0,
    "category_queries": 0,
    "trend_queries": 0,
    "goal_queries": 0,
    "account_queries": 0,
}

def save_example(category: str, question: str, sql: str):
    """Save a training example to the appropriate file."""
    filepath = OUTPUT_DIR / f"{category}.jsonl"
    example = {
        "instruction": question,
        "output": sql,
        "category": category
    }
    with open(filepath, "a") as f:
        f.write(json.dumps(example) + "\n")
    stats[category] += 1

def generate_spending_queries():
    """Generate spending-related queries."""
    
    # Time periods
    periods = [
        ("last month", "date >= date('now', '-1 month')"),
        ("this month", "date >= date('now', 'start of month')"),
        ("last week", "date >= date('now', '-7 days')"),
        ("this year", "date >= date('now', 'start of year')"),
        ("in January", "strftime('%m', date) = '01'"),
        ("in February", "strftime('%m', date) = '02'"),
        ("in 2025", "strftime('%Y', date) = '2025'"),
        ("today", "date = date('now')"),
        ("yesterday", "date = date('now', '-1 day')"),
        ("last 30 days", "date >= date('now', '-30 days')"),
        ("last 90 days", "date >= date('now', '-90 days')"),
    ]
    
    # Question templates
    templates = [
        "How much did I spend {period}?",
        "What was my total spending {period}?",
        "Show me my spending {period}",
        "What did I spend {period}?",
        "Total expenses {period}?",
        "How much money did I spend {period}?",
        "What's my spending {period}?",
    ]
    
    for period_text, period_sql in periods:
        for template in templates:
            question = template.format(period=period_text)
            sql = f"SELECT SUM(amount) as total_spent FROM transactions WHERE {period_sql}"
            save_example("spending_queries", question, sql)
    
    # Category-specific spending
    categories = ["Groceries", "Dining", "Gas", "Shopping", "Entertainment", 
                  "Utilities", "Healthcare", "Transport", "Subscriptions", "Travel"]
    
    cat_templates = [
        "How much did I spend on {category}?",
        "What's my {category} spending?",
        "Show me {category} expenses",
        "Total {category} costs?",
        "{category} spending last month?",
        "How much on {category} this month?",
    ]
    
    for category in categories:
        for template in cat_templates:
            question = template.format(category=category.lower())
            sql = f"SELECT SUM(amount) as total FROM transactions WHERE category = '{category}'"
            save_example("spending_queries", question, sql)

def generate_category_queries():
    """Generate category-related queries."""
    
    templates = [
        ("What are my top spending categories?", 
         "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC LIMIT 5"),
        
        ("Which category do I spend the most on?",
         "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC LIMIT 1"),
        
        ("Show me spending by category",
         "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC"),
        
        ("Break down my expenses by category",
         "SELECT category, SUM(amount) as total, COUNT(*) as count FROM transactions GROUP BY category ORDER BY total DESC"),
        
        ("Where does my money go?",
         "SELECT category, SUM(amount) as total, ROUND(SUM(amount) * 100.0 / (SELECT SUM(amount) FROM transactions), 1) as percentage FROM transactions GROUP BY category ORDER BY total DESC"),
        
        ("What percentage do I spend on dining?",
         "SELECT ROUND(SUM(CASE WHEN category = 'Dining' THEN amount ELSE 0 END) * 100.0 / SUM(amount), 1) as dining_percentage FROM transactions"),
        
        ("How many transactions per category?",
         "SELECT category, COUNT(*) as transaction_count FROM transactions GROUP BY category ORDER BY transaction_count DESC"),
    ]
    
    for question, sql in templates:
        save_example("category_queries", question, sql)
    
    # Variations
    variations = [
        "top categories", "biggest expenses", "main spending areas",
        "expense breakdown", "spending distribution", "where money goes"
    ]
    
    for var in variations:
        question = f"Show me my {var}"
        sql = "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC"
        save_example("category_queries", question, sql)

def generate_trend_queries():
    """Generate trend and comparison queries."""
    
    templates = [
        ("Am I spending more than last month?",
         """SELECT 
            (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')) as this_month,
            (SELECT SUM(amount) FROM transactions WHERE date >= date('now', '-1 month', 'start of month') AND date < date('now', 'start of month')) as last_month"""),
        
        ("What's my spending trend?",
         """SELECT strftime('%Y-%m', date) as month, SUM(amount) as total 
            FROM transactions GROUP BY month ORDER BY month DESC LIMIT 6"""),
        
        ("Compare my spending month over month",
         """SELECT strftime('%Y-%m', date) as month, SUM(amount) as total 
            FROM transactions GROUP BY month ORDER BY month"""),
        
        ("Is my grocery spending increasing?",
         """SELECT strftime('%Y-%m', date) as month, SUM(amount) as total 
            FROM transactions WHERE category = 'Groceries' GROUP BY month ORDER BY month DESC LIMIT 6"""),
        
        ("How does this month compare to average?",
         """SELECT 
            (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')) as this_month,
            (SELECT AVG(monthly_total) FROM (SELECT SUM(amount) as monthly_total FROM transactions GROUP BY strftime('%Y-%m', date))) as average_month"""),
        
        ("What's my average monthly spending?",
         """SELECT AVG(monthly_total) as avg_monthly 
            FROM (SELECT SUM(amount) as monthly_total FROM transactions GROUP BY strftime('%Y-%m', date))"""),
        
        ("Show me spending over time",
         """SELECT strftime('%Y-%m', date) as month, SUM(amount) as total 
            FROM transactions GROUP BY month ORDER BY month"""),
    ]
    
    for question, sql in templates:
        save_example("trend_queries", question, sql)

def generate_goal_queries():
    """Generate goal and budget queries."""
    
    templates = [
        ("How close am I to my emergency fund goal?",
         "SELECT name, current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as progress_pct FROM goals WHERE name LIKE '%emergency%'"),
        
        ("Show me my savings goals",
         "SELECT name, current_amount, target_amount, target_date FROM goals WHERE is_active = 1"),
        
        ("What's my progress on all goals?",
         "SELECT name, current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as progress_pct FROM goals"),
        
        ("How much more do I need for my vacation fund?",
         "SELECT name, target_amount - current_amount as remaining FROM goals WHERE name LIKE '%vacation%'"),
        
        ("Am I on track for my goals?",
         """SELECT name, current_amount, target_amount, target_date,
            ROUND(current_amount * 100.0 / target_amount, 1) as progress_pct
            FROM goals WHERE is_active = 1"""),
        
        ("What goals do I have?",
         "SELECT name, target_amount, current_amount, target_date FROM goals"),
        
        ("How much have I saved this month?",
         """SELECT SUM(amount) as saved FROM transactions 
            WHERE category IN ('Savings', 'Transfer to Savings') 
            AND date >= date('now', 'start of month')"""),
    ]
    
    for question, sql in templates:
        save_example("goal_queries", question, sql)

def generate_account_queries():
    """Generate account and balance queries."""
    
    templates = [
        ("What's my checking account balance?",
         "SELECT balance FROM v_current_balances WHERE type = 'checking'"),
        
        ("Show me all my accounts",
         "SELECT name, type, balance FROM v_current_balances"),
        
        ("What's my credit card balance?",
         "SELECT name, balance FROM v_current_balances WHERE type = 'credit_card'"),
        
        ("What's my total credit card debt?",
         "SELECT SUM(balance) as total_debt FROM v_current_balances WHERE type = 'credit_card'"),
        
        ("What's my credit utilization?",
         """SELECT name, balance, credit_limit, 
            ROUND(balance * 100.0 / credit_limit, 1) as utilization 
            FROM v_credit_card_summary"""),
        
        ("How much do I have in savings?",
         "SELECT SUM(balance) as total_savings FROM v_current_balances WHERE type = 'savings'"),
        
        ("What's my net worth?",
         """SELECT 
            SUM(CASE WHEN type IN ('checking', 'savings', 'investment') THEN balance ELSE 0 END) as assets,
            SUM(CASE WHEN type IN ('credit_card', 'loan') THEN balance ELSE 0 END) as liabilities,
            SUM(CASE WHEN type IN ('checking', 'savings', 'investment') THEN balance ELSE -balance END) as net_worth
            FROM v_current_balances"""),
        
        ("Show me my net worth over time",
         "SELECT snapshot_date, net_worth FROM net_worth_snapshots ORDER BY snapshot_date"),
        
        ("What accounts do I have?",
         "SELECT name, type, institution FROM accounts WHERE is_active = 1"),
    ]
    
    for question, sql in templates:
        save_example("account_queries", question, sql)

def combine_all_data():
    """Merge all data into a single training file."""
    combined = OUTPUT_DIR / "combined.jsonl"
    all_examples = []
    
    for jsonl_file in OUTPUT_DIR.glob("*.jsonl"):
        if jsonl_file.name == "combined.jsonl":
            continue
        with open(jsonl_file) as f:
            for line in f:
                all_examples.append(json.loads(line))
    
    # Shuffle for training
    random.shuffle(all_examples)
    
    with open(combined, "w") as f:
        for example in all_examples:
            f.write(json.dumps(example) + "\n")
    
    return len(all_examples)

def update_readme_stats():
    """Update the README with current stats."""
    readme_path = Path(__file__).parent / "README.md"
    content = readme_path.read_text()
    
    # This is a simple update - in practice we'd use proper templating
    print("\n📊 Current Stats:")
    for category, count in stats.items():
        print(f"  {category}: {count}")
    print(f"  TOTAL: {sum(stats.values())}")

def main():
    print("🚀 Generating training data for LocalFinance AI...")
    print(f"📁 Output directory: {OUTPUT_DIR}\n")
    
    # Clear existing data
    for f in OUTPUT_DIR.glob("*.jsonl"):
        f.unlink()
    
    print("1️⃣  Generating spending queries...")
    generate_spending_queries()
    
    print("2️⃣  Generating category queries...")
    generate_category_queries()
    
    print("3️⃣  Generating trend queries...")
    generate_trend_queries()
    
    print("4️⃣  Generating goal queries...")
    generate_goal_queries()
    
    print("5️⃣  Generating account queries...")
    generate_account_queries()
    
    print("\n📦 Combining all data...")
    total = combine_all_data()
    
    print(f"\n✅ Generated {total} training examples!")
    update_readme_stats()
    
    print(f"\n📂 Files created in {OUTPUT_DIR}/")
    for f in sorted(OUTPUT_DIR.glob("*.jsonl")):
        count = sum(1 for _ in open(f))
        print(f"   {f.name}: {count} examples")

if __name__ == "__main__":
    main()
