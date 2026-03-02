#!/usr/bin/env python3
"""
LocalFinance AI Prototype - Test the concept TODAY
Uses Ollama + llama3.2 with prompt engineering (no fine-tuning needed)
"""

import subprocess
import json
import sqlite3
from pathlib import Path

# Our database schema (simplified)
SCHEMA = """
CREATE TABLE transactions (
    id INTEGER PRIMARY KEY,
    date TEXT,
    description TEXT,
    amount REAL,
    category TEXT,
    source TEXT
);

CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    name TEXT,
    type TEXT,  -- checking, savings, credit_card
    balance REAL
);

CREATE TABLE goals (
    id TEXT PRIMARY KEY,
    name TEXT,
    target_amount REAL,
    current_amount REAL
);

CREATE TABLE budget_categories (
    category TEXT PRIMARY KEY,
    bucket TEXT,  -- needs, wants, future
    monthly_budget REAL
);
"""

# Few-shot examples
EXAMPLES = """
Question: How much did I spend last month?
SQL: SELECT SUM(amount) as total FROM transactions WHERE date >= date('now', '-1 month', 'start of month') AND date < date('now', 'start of month')

Question: What are my top spending categories?
SQL: SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC LIMIT 5

Question: How much did I spend on groceries?
SQL: SELECT SUM(amount) as total FROM transactions WHERE category = 'Groceries'

Question: What's my checking account balance?
SQL: SELECT balance FROM accounts WHERE type = 'checking'

Question: Show me Amazon purchases
SQL: SELECT date, description, amount FROM transactions WHERE description LIKE '%AMAZON%' ORDER BY date DESC

Question: Am I over budget on dining?
SQL: SELECT SUM(t.amount) as spent, b.monthly_budget FROM transactions t JOIN budget_categories b ON t.category = b.category WHERE t.category = 'Dining' AND t.date >= date('now', 'start of month')

Question: How's my emergency fund?
SQL: SELECT name, current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as progress FROM goals WHERE name LIKE '%emergency%'
"""

def ask_ollama(question: str) -> str:
    """Send question to Ollama and get SQL back."""
    
    prompt = f"""You are a SQL expert for personal finance. Convert natural language questions to SQLite queries.

DATABASE SCHEMA:
{SCHEMA}

EXAMPLES:
{EXAMPLES}

Now convert this question to SQL. Return ONLY the SQL query, nothing else.

Question: {question}
SQL:"""

    try:
        result = subprocess.run(
            ["ollama", "run", "llama3.2", prompt],
            capture_output=True,
            text=True,
            timeout=60
        )
        sql = result.stdout.strip()
        # Clean up common issues
        sql = sql.replace("```sql", "").replace("```", "").strip()
        return sql
    except Exception as e:
        return f"Error: {e}"

def create_test_db():
    """Create a test database with sample data."""
    db_path = Path(__file__).parent.parent / "test_data.db"
    conn = sqlite3.connect(db_path)
    c = conn.cursor()
    
    # Create tables
    c.executescript("""
        DROP TABLE IF EXISTS transactions;
        DROP TABLE IF EXISTS accounts;
        DROP TABLE IF EXISTS goals;
        DROP TABLE IF EXISTS budget_categories;
        
        CREATE TABLE transactions (
            id INTEGER PRIMARY KEY,
            date TEXT,
            description TEXT,
            amount REAL,
            category TEXT,
            source TEXT
        );
        
        CREATE TABLE accounts (
            id TEXT PRIMARY KEY,
            name TEXT,
            type TEXT,
            balance REAL
        );
        
        CREATE TABLE goals (
            id TEXT PRIMARY KEY,
            name TEXT,
            target_amount REAL,
            current_amount REAL
        );
        
        CREATE TABLE budget_categories (
            category TEXT PRIMARY KEY,
            bucket TEXT,
            monthly_budget REAL
        );
        
        -- Sample transactions
        INSERT INTO transactions (date, description, amount, category, source) VALUES
        ('2026-02-01', 'TRADER JOES #123', 87.43, 'Groceries', 'Chase'),
        ('2026-02-03', 'UBER EATS', 34.21, 'Dining', 'Chase'),
        ('2026-02-05', 'SHELL OIL', 52.00, 'Gas', 'Chase'),
        ('2026-02-07', 'AMAZON.COM', 125.99, 'Shopping', 'Chase'),
        ('2026-02-10', 'STARBUCKS', 8.45, 'Dining', 'Chase'),
        ('2026-02-12', 'COSTCO', 234.56, 'Groceries', 'Chase'),
        ('2026-02-14', 'NETFLIX', 15.99, 'Subscriptions', 'Chase'),
        ('2026-02-15', 'WHOLE FOODS', 67.89, 'Groceries', 'Chase'),
        ('2026-02-18', 'CHIPOTLE', 14.25, 'Dining', 'Chase'),
        ('2026-02-20', 'AMAZON.COM', 45.00, 'Shopping', 'Chase'),
        ('2026-02-22', 'SPOTIFY', 9.99, 'Subscriptions', 'Chase'),
        ('2026-02-25', 'SAFEWAY', 98.32, 'Groceries', 'Chase'),
        ('2026-02-27', 'DOORDASH', 42.15, 'Dining', 'Chase');
        
        -- Sample accounts
        INSERT INTO accounts (id, name, type, balance) VALUES
        ('chase_checking', 'Chase Checking', 'checking', 4523.45),
        ('chase_savings', 'Chase Savings', 'savings', 12500.00),
        ('amex_gold', 'Amex Gold', 'credit_card', 1234.56);
        
        -- Sample goals
        INSERT INTO goals (id, name, target_amount, current_amount) VALUES
        ('emergency', 'Emergency Fund', 15000, 8500),
        ('vacation', 'Vacation Fund', 3000, 1200);
        
        -- Sample budgets
        INSERT INTO budget_categories (category, bucket, monthly_budget) VALUES
        ('Groceries', 'needs', 600),
        ('Dining', 'wants', 300),
        ('Gas', 'needs', 200),
        ('Shopping', 'wants', 200),
        ('Subscriptions', 'wants', 50);
    """)
    
    conn.commit()
    conn.close()
    return db_path

def run_sql(db_path: Path, sql: str):
    """Execute SQL and return results."""
    try:
        conn = sqlite3.connect(db_path)
        conn.row_factory = sqlite3.Row
        c = conn.cursor()
        c.execute(sql)
        rows = c.fetchall()
        conn.close()
        
        if not rows:
            return "No results"
        
        # Format results
        results = []
        for row in rows:
            results.append(dict(row))
        return results
    except Exception as e:
        return f"SQL Error: {e}"

def main():
    print("=" * 60)
    print("🏦 LocalFinance AI - Prototype Test")
    print("=" * 60)
    
    # Create test database
    print("\n📊 Creating test database with sample data...")
    db_path = create_test_db()
    print(f"   Database: {db_path}")
    
    # Test questions
    test_questions = [
        "How much did I spend on groceries?",
        "What are my top spending categories?",
        "Show me Amazon purchases",
        "What's my checking account balance?",
        "How's my emergency fund?",
        "Am I over budget on dining?",
        "How much did I spend at Starbucks?",
        "Total spending this month?",
    ]
    
    print("\n" + "=" * 60)
    print("🧪 Running Tests")
    print("=" * 60)
    
    for question in test_questions:
        print(f"\n❓ Question: {question}")
        
        # Get SQL from model
        sql = ask_ollama(question)
        print(f"📝 SQL: {sql}")
        
        # Run the SQL
        if not sql.startswith("Error"):
            result = run_sql(db_path, sql)
            print(f"✅ Result: {result}")
        else:
            print(f"❌ {sql}")
        
        print("-" * 40)
    
    print("\n" + "=" * 60)
    print("🎮 Interactive Mode")
    print("Type 'quit' to exit")
    print("=" * 60)
    
    while True:
        question = input("\n❓ Your question: ").strip()
        if question.lower() in ['quit', 'exit', 'q']:
            break
        if not question:
            continue
            
        sql = ask_ollama(question)
        print(f"📝 SQL: {sql}")
        
        if not sql.startswith("Error"):
            result = run_sql(db_path, sql)
            print(f"✅ Result: {result}")

if __name__ == "__main__":
    main()
