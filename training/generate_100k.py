#!/usr/bin/env python3
"""
Generate 100,000 training examples for LocalFinance AI.

Strategy:
1. Templates × Variations × Time periods × Categories = Combinatorial explosion
2. Synonyms and paraphrases for natural language
3. Edge cases and complex queries
4. Real-world scenarios people actually ask

Target: 100,000 examples
"""

import json
import random
import itertools
from pathlib import Path
from datetime import datetime, timedelta

OUTPUT_DIR = Path(__file__).parent / "data"
OUTPUT_DIR.mkdir(exist_ok=True)

# Clear existing data
for f in OUTPUT_DIR.glob("*.jsonl"):
    f.unlink()

EXAMPLES = []

def add(question: str, sql: str, category: str = "general"):
    EXAMPLES.append({"instruction": question, "output": sql, "category": category})

# =============================================================================
# BUILDING BLOCKS
# =============================================================================

# Time periods with SQL
TIME_PERIODS = [
    ("today", "date = date('now')"),
    ("yesterday", "date = date('now', '-1 day')"),
    ("this week", "date >= date('now', 'weekday 0', '-7 days')"),
    ("last week", "date >= date('now', '-14 days') AND date < date('now', '-7 days')"),
    ("this month", "date >= date('now', 'start of month')"),
    ("last month", "date >= date('now', '-1 month', 'start of month') AND date < date('now', 'start of month')"),
    ("this year", "date >= date('now', 'start of year')"),
    ("last year", "strftime('%Y', date) = strftime('%Y', date('now', '-1 year'))"),
    ("the past 7 days", "date >= date('now', '-7 days')"),
    ("the past 30 days", "date >= date('now', '-30 days')"),
    ("the past 90 days", "date >= date('now', '-90 days')"),
    ("the past 6 months", "date >= date('now', '-6 months')"),
    ("Q1", "date >= date('now', 'start of year') AND date < date('now', 'start of year', '+3 months')"),
    ("Q2", "date >= date('now', 'start of year', '+3 months') AND date < date('now', 'start of year', '+6 months')"),
    ("Q3", "date >= date('now', 'start of year', '+6 months') AND date < date('now', 'start of year', '+9 months')"),
    ("Q4", "date >= date('now', 'start of year', '+9 months') AND date < date('now', 'start of year', '+12 months')"),
    ("in January", "strftime('%m', date) = '01'"),
    ("in February", "strftime('%m', date) = '02'"),
    ("in March", "strftime('%m', date) = '03'"),
    ("in April", "strftime('%m', date) = '04'"),
    ("in May", "strftime('%m', date) = '05'"),
    ("in June", "strftime('%m', date) = '06'"),
    ("in July", "strftime('%m', date) = '07'"),
    ("in August", "strftime('%m', date) = '08'"),
    ("in September", "strftime('%m', date) = '09'"),
    ("in October", "strftime('%m', date) = '10'"),
    ("in November", "strftime('%m', date) = '11'"),
    ("in December", "strftime('%m', date) = '12'"),
    ("on weekends", "strftime('%w', date) IN ('0', '6')"),
    ("on weekdays", "strftime('%w', date) NOT IN ('0', '6')"),
    ("on Mondays", "strftime('%w', date) = '1'"),
    ("on Fridays", "strftime('%w', date) = '5'"),
    ("since January 1st", "date >= '2026-01-01'"),
    ("before tax day", "date < '2026-04-15'"),
]

# Categories
CATEGORIES = [
    "Groceries", "Dining", "Gas", "Transport", "Utilities", "Healthcare",
    "Entertainment", "Shopping", "Travel", "Subscriptions", "Insurance",
    "Rent", "Mortgage", "Kids", "Pets", "Education", "Fitness", "Beauty",
    "Home", "Gifts", "Charity", "Business", "Fees", "Interest", "Other"
]

# Merchants (realistic)
MERCHANTS = [
    ("Amazon", "AMAZON"), ("Costco", "COSTCO"), ("Target", "TARGET"),
    ("Walmart", "WALMART"), ("Whole Foods", "WHOLE FOODS"), ("Trader Joe's", "TRADER JOE"),
    ("Safeway", "SAFEWAY"), ("Kroger", "KROGER"), ("Publix", "PUBLIX"),
    ("Starbucks", "STARBUCKS"), ("Dunkin", "DUNKIN"), ("McDonald's", "MCDONALD"),
    ("Chipotle", "CHIPOTLE"), ("Panera", "PANERA"), ("Chick-fil-A", "CHICK-FIL-A"),
    ("Uber", "UBER"), ("Lyft", "LYFT"), ("DoorDash", "DOORDASH"),
    ("Grubhub", "GRUBHUB"), ("Instacart", "INSTACART"), ("Postmates", "POSTMATES"),
    ("Netflix", "NETFLIX"), ("Spotify", "SPOTIFY"), ("Apple", "APPLE"),
    ("Google", "GOOGLE"), ("Microsoft", "MICROSOFT"), ("Adobe", "ADOBE"),
    ("Hulu", "HULU"), ("Disney+", "DISNEY"), ("HBO", "HBO"),
    ("Amazon Prime", "PRIME"), ("YouTube", "YOUTUBE"), ("Dropbox", "DROPBOX"),
    ("Gym", "GYM"), ("Planet Fitness", "PLANET FITNESS"), ("Equinox", "EQUINOX"),
    ("CVS", "CVS"), ("Walgreens", "WALGREENS"), ("Rite Aid", "RITE AID"),
    ("Home Depot", "HOME DEPOT"), ("Lowes", "LOWES"), ("IKEA", "IKEA"),
    ("Best Buy", "BEST BUY"), ("Apple Store", "APPLE STORE"), ("Nordstrom", "NORDSTROM"),
    ("Macy's", "MACYS"), ("TJ Maxx", "TJ MAXX"), ("Ross", "ROSS"),
    ("Shell", "SHELL"), ("Chevron", "CHEVRON"), ("Exxon", "EXXON"),
    ("BP", "BP"), ("76", "76"), ("Costco Gas", "COSTCO GAS"),
    ("Verizon", "VERIZON"), ("AT&T", "ATT"), ("T-Mobile", "T-MOBILE"),
    ("Comcast", "COMCAST"), ("Spectrum", "SPECTRUM"), ("PG&E", "PGE"),
    ("Electric", "ELECTRIC"), ("Water", "WATER"), ("Gas Company", "GAS CO"),
    ("Venmo", "VENMO"), ("PayPal", "PAYPAL"), ("Zelle", "ZELLE"),
    ("Chase", "CHASE"), ("Bank of America", "BANK OF AMERICA"), ("Wells Fargo", "WELLS FARGO"),
    ("Airline", "AIRLINE"), ("Delta", "DELTA"), ("United", "UNITED"),
    ("Southwest", "SOUTHWEST"), ("JetBlue", "JETBLUE"), ("American Airlines", "AMERICAN"),
    ("Hotel", "HOTEL"), ("Marriott", "MARRIOTT"), ("Hilton", "HILTON"),
    ("Airbnb", "AIRBNB"), ("VRBO", "VRBO"), ("Expedia", "EXPEDIA"),
]

# Question starters (synonyms)
SPEND_VERBS = ["spend", "spent", "pay", "paid", "use", "used", "drop", "dropped", "blow", "blew"]
SHOW_VERBS = ["show", "display", "list", "give me", "what are", "tell me", "find", "get"]
HOW_MUCH = ["how much", "what's the total", "what did I spend", "total", "sum of", "what's my spending", "what was"]

# =============================================================================
# GENERATORS
# =============================================================================

def generate_spending_by_time():
    """Generate spending queries for all time periods."""
    count = 0
    
    templates = [
        "How much did I spend {time}?",
        "What did I spend {time}?",
        "Total spending {time}?",
        "What's my spending {time}?",
        "How much have I spent {time}?",
        "Spending {time}?",
        "What was my total {time}?",
        "{time} spending?",
        "Money spent {time}?",
        "Show me spending {time}",
        "What'd I spend {time}?",
        "Damage {time}?",
    ]
    
    for time_text, time_sql in TIME_PERIODS:
        for template in templates:
            question = template.format(time=time_text)
            sql = f"SELECT SUM(amount) as total_spent FROM transactions WHERE {time_sql}"
            add(question, sql, "spending_time")
            count += 1
    
    return count

def generate_spending_by_category():
    """Generate spending queries for all categories."""
    count = 0
    
    templates = [
        "How much did I spend on {cat}?",
        "What's my {cat} spending?",
        "{cat} expenses?",
        "Total {cat} costs?",
        "How much on {cat}?",
        "Show me {cat} spending",
        "{cat} total?",
        "What did I pay for {cat}?",
        "Money spent on {cat}?",
        "{cat}?",  # Just the category name
    ]
    
    for category in CATEGORIES:
        cat_lower = category.lower()
        for template in templates:
            question = template.format(cat=cat_lower)
            sql = f"SELECT SUM(amount) as total FROM transactions WHERE category = '{category}'"
            add(question, sql, "spending_category")
            count += 1
    
    return count

def generate_spending_by_category_and_time():
    """Cross product of categories and time periods."""
    count = 0
    
    templates = [
        "How much did I spend on {cat} {time}?",
        "{cat} spending {time}?",
        "What did I spend on {cat} {time}?",
        "{cat} {time}?",
        "Total {cat} {time}?",
    ]
    
    # Use subset to avoid explosion (25 cats × 35 times × 5 templates = 4375)
    for category in CATEGORIES[:15]:  # Top 15 categories
        cat_lower = category.lower()
        for time_text, time_sql in TIME_PERIODS[:15]:  # Top 15 time periods
            for template in templates:
                question = template.format(cat=cat_lower, time=time_text)
                sql = f"SELECT SUM(amount) as total FROM transactions WHERE category = '{category}' AND {time_sql}"
                add(question, sql, "spending_cat_time")
                count += 1
    
    return count

def generate_merchant_queries():
    """Queries about specific merchants."""
    count = 0
    
    templates = [
        "How much did I spend at {name}?",
        "Show me all {name} purchases",
        "{name} spending?",
        "Total at {name}?",
        "How many times did I go to {name}?",
        "What's my average {name} purchase?",
        "{name} this month?",
        "{name} transactions?",
        "Last {name} purchase?",
        "Find {name}",
        "{name}",  # Just merchant name
    ]
    
    for name, pattern in MERCHANTS:
        for template in templates:
            question = template.format(name=name)
            if "how many" in template.lower() or "times" in template.lower():
                sql = f"SELECT COUNT(*) as visits FROM transactions WHERE description LIKE '%{pattern}%'"
            elif "average" in template.lower():
                sql = f"SELECT AVG(amount) as avg_purchase FROM transactions WHERE description LIKE '%{pattern}%'"
            elif "last" in template.lower():
                sql = f"SELECT date, amount FROM transactions WHERE description LIKE '%{pattern}%' ORDER BY date DESC LIMIT 1"
            elif "show" in template.lower() or "all" in template.lower() or "transactions" in template.lower() or "find" in template.lower():
                sql = f"SELECT date, description, amount FROM transactions WHERE description LIKE '%{pattern}%' ORDER BY date DESC"
            else:
                sql = f"SELECT SUM(amount) as total FROM transactions WHERE description LIKE '%{pattern}%'"
            add(question, sql, "merchant")
            count += 1
    
    return count

def generate_category_analysis():
    """Queries about category breakdowns."""
    count = 0
    
    queries = [
        ("What are my top spending categories?", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC LIMIT 5"),
        ("Top 10 categories?", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC LIMIT 10"),
        ("Where does my money go?", "SELECT category, SUM(amount) as total, ROUND(SUM(amount) * 100.0 / (SELECT SUM(amount) FROM transactions), 1) as pct FROM transactions GROUP BY category ORDER BY total DESC"),
        ("Spending breakdown", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC"),
        ("Category breakdown", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC"),
        ("Expense breakdown", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC"),
        ("Which category do I spend most on?", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC LIMIT 1"),
        ("Biggest expense category?", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC LIMIT 1"),
        ("Show me spending by category", "SELECT category, SUM(amount) as total FROM transactions GROUP BY category ORDER BY total DESC"),
        ("Categorize my spending", "SELECT category, SUM(amount) as total, COUNT(*) as transactions FROM transactions GROUP BY category ORDER BY total DESC"),
        ("What percentage is dining?", "SELECT ROUND(SUM(CASE WHEN category = 'Dining' THEN amount ELSE 0 END) * 100.0 / SUM(amount), 1) as dining_pct FROM transactions"),
        ("What percentage is groceries?", "SELECT ROUND(SUM(CASE WHEN category = 'Groceries' THEN amount ELSE 0 END) * 100.0 / SUM(amount), 1) as grocery_pct FROM transactions"),
        ("How many transactions per category?", "SELECT category, COUNT(*) as count FROM transactions GROUP BY category ORDER BY count DESC"),
    ]
    
    for question, sql in queries:
        add(question, sql, "category_analysis")
        count += 1
    
    # Time-based category analysis
    for time_text, time_sql in TIME_PERIODS[:10]:
        add(f"Category breakdown {time_text}",
            f"SELECT category, SUM(amount) as total FROM transactions WHERE {time_sql} GROUP BY category ORDER BY total DESC",
            "category_analysis")
        add(f"Top categories {time_text}",
            f"SELECT category, SUM(amount) as total FROM transactions WHERE {time_sql} GROUP BY category ORDER BY total DESC LIMIT 5",
            "category_analysis")
        count += 2
    
    return count

def generate_budget_queries():
    """50/30/20 and budget queries."""
    count = 0
    
    queries = [
        # 50/30/20 rule
        ("How am I doing on 50/30/20?", "SELECT bucket, SUM(t.amount) as spent, ROUND(SUM(t.amount) * 100.0 / (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')), 1) as pct FROM transactions t JOIN budget_categories bc ON t.category = bc.category WHERE t.date >= date('now', 'start of month') GROUP BY bucket"),
        ("50 30 20 breakdown", "SELECT bucket, SUM(t.amount) as spent FROM transactions t JOIN budget_categories bc ON t.category = bc.category WHERE t.date >= date('now', 'start of month') GROUP BY bucket"),
        ("Needs vs wants vs savings?", "SELECT bucket, SUM(t.amount) as total FROM transactions t JOIN budget_categories bc ON t.category = bc.category GROUP BY bucket"),
        ("Am I overspending on wants?", "SELECT SUM(amount) as wants_total FROM transactions t JOIN budget_categories bc ON t.category = bc.category WHERE bucket = 'wants' AND date >= date('now', 'start of month')"),
        ("What percentage goes to needs?", "SELECT ROUND(SUM(t.amount) * 100.0 / (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')), 1) as needs_pct FROM transactions t JOIN budget_categories bc ON t.category = bc.category WHERE bucket = 'needs' AND t.date >= date('now', 'start of month')"),
        
        # Budget tracking
        ("Am I over budget?", "SELECT category, SUM(amount) as spent, bc.monthly_budget, SUM(amount) - bc.monthly_budget as over_under FROM transactions t JOIN budget_categories bc ON t.category = bc.category WHERE date >= date('now', 'start of month') GROUP BY category HAVING monthly_budget IS NOT NULL"),
        ("Which categories am I over budget?", "SELECT category, SUM(amount) as spent, bc.monthly_budget FROM transactions t JOIN budget_categories bc ON t.category = bc.category WHERE date >= date('now', 'start of month') GROUP BY category HAVING SUM(amount) > bc.monthly_budget"),
        ("Budget remaining?", "SELECT bc.category, bc.monthly_budget - COALESCE(SUM(t.amount), 0) as remaining FROM budget_categories bc LEFT JOIN transactions t ON t.category = bc.category AND t.date >= date('now', 'start of month') WHERE bc.monthly_budget IS NOT NULL GROUP BY bc.category"),
        ("How much dining budget left?", "SELECT bc.monthly_budget - COALESCE(SUM(t.amount), 0) as remaining FROM budget_categories bc LEFT JOIN transactions t ON t.category = bc.category AND t.date >= date('now', 'start of month') WHERE bc.category = 'Dining'"),
        ("How much grocery budget left?", "SELECT bc.monthly_budget - COALESCE(SUM(t.amount), 0) as remaining FROM budget_categories bc LEFT JOIN transactions t ON t.category = bc.category AND t.date >= date('now', 'start of month') WHERE bc.category = 'Groceries'"),
        ("What's my dining budget?", "SELECT monthly_budget FROM budget_categories WHERE category = 'Dining'"),
        ("Show all budgets", "SELECT category, monthly_budget FROM budget_categories WHERE monthly_budget IS NOT NULL"),
    ]
    
    for question, sql in queries:
        add(question, sql, "budget")
        count += 1
    
    return count

def generate_account_queries():
    """Account balance and credit card queries."""
    count = 0
    
    queries = [
        # Balances
        ("What's my checking balance?", "SELECT balance FROM v_current_balances WHERE type = 'checking'"),
        ("Checking account balance?", "SELECT balance FROM v_current_balances WHERE type = 'checking'"),
        ("How much in checking?", "SELECT balance FROM v_current_balances WHERE type = 'checking'"),
        ("What's my savings balance?", "SELECT balance FROM v_current_balances WHERE type = 'savings'"),
        ("How much in savings?", "SELECT SUM(balance) FROM v_current_balances WHERE type = 'savings'"),
        ("Total savings?", "SELECT SUM(balance) FROM v_current_balances WHERE type = 'savings'"),
        ("Show all accounts", "SELECT name, type, balance FROM v_current_balances"),
        ("Account balances?", "SELECT name, type, balance FROM v_current_balances ORDER BY balance DESC"),
        ("How much money do I have?", "SELECT SUM(balance) FROM v_current_balances WHERE type IN ('checking', 'savings')"),
        ("Total cash?", "SELECT SUM(balance) FROM v_current_balances WHERE type IN ('checking', 'savings')"),
        ("Available funds?", "SELECT SUM(balance) FROM v_current_balances WHERE type IN ('checking', 'savings')"),
        
        # Credit cards
        ("Credit card balance?", "SELECT name, balance FROM v_current_balances WHERE type = 'credit_card'"),
        ("How much do I owe on credit cards?", "SELECT SUM(balance) as total_debt FROM v_current_balances WHERE type = 'credit_card'"),
        ("Credit card debt?", "SELECT SUM(balance) as total_debt FROM v_current_balances WHERE type = 'credit_card'"),
        ("What's my credit utilization?", "SELECT name, balance, credit_limit, ROUND(balance * 100.0 / credit_limit, 1) as utilization FROM v_credit_card_summary"),
        ("Am I near my credit limit?", "SELECT name, balance, credit_limit, ROUND(balance * 100.0 / credit_limit, 1) as utilization FROM v_credit_card_summary WHERE balance > credit_limit * 0.7"),
        ("Credit limit remaining?", "SELECT name, credit_limit - balance as available FROM v_credit_card_summary"),
        ("Total credit available?", "SELECT SUM(credit_limit - balance) as total_available FROM v_credit_card_summary"),
        
        # Net worth
        ("What's my net worth?", "SELECT SUM(CASE WHEN type IN ('checking', 'savings', 'investment') THEN balance ELSE -balance END) as net_worth FROM v_current_balances"),
        ("Net worth?", "SELECT SUM(CASE WHEN type IN ('checking', 'savings', 'investment') THEN balance ELSE -balance END) as net_worth FROM v_current_balances"),
        ("Assets vs liabilities?", "SELECT SUM(CASE WHEN type IN ('checking', 'savings', 'investment') THEN balance ELSE 0 END) as assets, SUM(CASE WHEN type IN ('credit_card', 'loan') THEN balance ELSE 0 END) as liabilities FROM v_current_balances"),
        ("Net worth over time", "SELECT snapshot_date, net_worth FROM net_worth_snapshots ORDER BY snapshot_date"),
        ("Net worth trend", "SELECT snapshot_date, net_worth FROM net_worth_snapshots ORDER BY snapshot_date DESC LIMIT 12"),
    ]
    
    for question, sql in queries:
        add(question, sql, "accounts")
        count += 1
    
    return count

def generate_goal_queries():
    """Savings goals queries."""
    count = 0
    
    queries = [
        ("Show my savings goals", "SELECT name, current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as progress FROM goals WHERE is_active = 1"),
        ("Goals progress?", "SELECT name, current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as progress FROM goals WHERE is_active = 1"),
        ("How's my emergency fund?", "SELECT current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as progress FROM goals WHERE name LIKE '%emergency%'"),
        ("Emergency fund progress?", "SELECT current_amount, target_amount, target_amount - current_amount as remaining FROM goals WHERE name LIKE '%emergency%'"),
        ("How much more for emergency fund?", "SELECT target_amount - current_amount as remaining FROM goals WHERE name LIKE '%emergency%'"),
        ("Vacation fund status?", "SELECT current_amount, target_amount, ROUND(current_amount * 100.0 / target_amount, 1) as progress FROM goals WHERE name LIKE '%vacation%'"),
        ("Am I on track for my goals?", "SELECT name, current_amount, target_amount, target_date, ROUND(current_amount * 100.0 / target_amount, 1) as progress FROM goals WHERE is_active = 1"),
        ("Which goals am I closest to?", "SELECT name, ROUND(current_amount * 100.0 / target_amount, 1) as progress FROM goals WHERE is_active = 1 ORDER BY progress DESC"),
        ("What goals do I have?", "SELECT name, target_amount, target_date FROM goals WHERE is_active = 1"),
        ("Total saved toward goals?", "SELECT SUM(current_amount) as total_saved FROM goals"),
        ("Total goal amount?", "SELECT SUM(target_amount) as total_target FROM goals WHERE is_active = 1"),
    ]
    
    for question, sql in queries:
        add(question, sql, "goals")
        count += 1
    
    return count

def generate_trend_queries():
    """Trend and comparison queries."""
    count = 0
    
    queries = [
        # Month comparisons
        ("Am I spending more than last month?", "SELECT (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')) as this_month, (SELECT SUM(amount) FROM transactions WHERE date >= date('now', '-1 month', 'start of month') AND date < date('now', 'start of month')) as last_month"),
        ("Compare this month to last month", "SELECT (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')) as this_month, (SELECT SUM(amount) FROM transactions WHERE date >= date('now', '-1 month', 'start of month') AND date < date('now', 'start of month')) as last_month"),
        ("Month over month spending", "SELECT strftime('%Y-%m', date) as month, SUM(amount) as total FROM transactions GROUP BY month ORDER BY month DESC LIMIT 6"),
        ("Monthly spending trend", "SELECT strftime('%Y-%m', date) as month, SUM(amount) as total FROM transactions GROUP BY month ORDER BY month"),
        ("Spending trend", "SELECT strftime('%Y-%m', date) as month, SUM(amount) as total FROM transactions GROUP BY month ORDER BY month"),
        
        # Category trends
        ("Is my grocery spending going up?", "SELECT strftime('%Y-%m', date) as month, SUM(amount) as total FROM transactions WHERE category = 'Groceries' GROUP BY month ORDER BY month DESC LIMIT 6"),
        ("Dining trend", "SELECT strftime('%Y-%m', date) as month, SUM(amount) as total FROM transactions WHERE category = 'Dining' GROUP BY month ORDER BY month DESC LIMIT 6"),
        
        # Averages
        ("What's my average monthly spending?", "SELECT AVG(monthly_total) as avg_monthly FROM (SELECT SUM(amount) as monthly_total FROM transactions GROUP BY strftime('%Y-%m', date))"),
        ("Average weekly spending?", "SELECT AVG(weekly_total) as avg_weekly FROM (SELECT SUM(amount) as weekly_total FROM transactions GROUP BY strftime('%Y-%W', date))"),
        ("Average daily spending?", "SELECT AVG(daily_total) as avg_daily FROM (SELECT SUM(amount) as daily_total FROM transactions GROUP BY date)"),
        ("How does this month compare to average?", "SELECT (SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')) as this_month, (SELECT AVG(monthly_total) FROM (SELECT SUM(amount) as monthly_total FROM transactions GROUP BY strftime('%Y-%m', date))) as average"),
        
        # Year over year
        ("Year over year comparison", "SELECT strftime('%Y', date) as year, SUM(amount) as total FROM transactions GROUP BY year"),
        ("This year vs last year", "SELECT (SELECT SUM(amount) FROM transactions WHERE strftime('%Y', date) = strftime('%Y', 'now')) as this_year, (SELECT SUM(amount) FROM transactions WHERE strftime('%Y', date) = strftime('%Y', date('now', '-1 year'))) as last_year"),
    ]
    
    for question, sql in queries:
        add(question, sql, "trends")
        count += 1
    
    # Category trends for each category
    for category in CATEGORIES[:10]:
        add(f"{category} spending trend",
            f"SELECT strftime('%Y-%m', date) as month, SUM(amount) as total FROM transactions WHERE category = '{category}' GROUP BY month ORDER BY month DESC LIMIT 6",
            "trends")
        add(f"Is {category.lower()} spending going up?",
            f"SELECT strftime('%Y-%m', date) as month, SUM(amount) as total FROM transactions WHERE category = '{category}' GROUP BY month ORDER BY month DESC LIMIT 3",
            "trends")
        count += 2
    
    return count

def generate_insight_queries():
    """Pattern recognition and insight queries."""
    count = 0
    
    queries = [
        ("What's my biggest expense this month?", "SELECT description, amount, date FROM transactions WHERE date >= date('now', 'start of month') ORDER BY amount DESC LIMIT 1"),
        ("Biggest purchase?", "SELECT description, amount, date FROM transactions ORDER BY amount DESC LIMIT 1"),
        ("Largest transaction?", "SELECT description, amount, date FROM transactions ORDER BY amount DESC LIMIT 1"),
        ("Top 5 biggest expenses", "SELECT description, amount, date FROM transactions ORDER BY amount DESC LIMIT 5"),
        ("Most frequent purchases?", "SELECT description, COUNT(*) as frequency, SUM(amount) as total FROM transactions GROUP BY description ORDER BY frequency DESC LIMIT 10"),
        ("Where am I wasting money?", "SELECT category, SUM(amount) as total FROM transactions WHERE category IN ('Dining', 'Entertainment', 'Shopping', 'Subscriptions') AND date >= date('now', '-3 months') GROUP BY category ORDER BY total DESC"),
        ("What subscriptions am I paying?", "SELECT DISTINCT description, amount FROM transactions WHERE category = 'Subscriptions' ORDER BY amount DESC"),
        ("Monthly subscriptions?", "SELECT description, amount FROM recurring_transactions WHERE frequency = 'monthly'"),
        ("Find recurring charges", "SELECT description, amount, COUNT(*) as occurrences FROM transactions GROUP BY description, amount HAVING occurrences > 2 ORDER BY occurrences DESC"),
        ("Daily average?", "SELECT ROUND(AVG(daily_total), 2) as avg_daily FROM (SELECT date, SUM(amount) as daily_total FROM transactions GROUP BY date)"),
        ("What day do I spend the most?", "SELECT CASE strftime('%w', date) WHEN '0' THEN 'Sunday' WHEN '1' THEN 'Monday' WHEN '2' THEN 'Tuesday' WHEN '3' THEN 'Wednesday' WHEN '4' THEN 'Thursday' WHEN '5' THEN 'Friday' WHEN '6' THEN 'Saturday' END as day, SUM(amount) as total FROM transactions GROUP BY strftime('%w', date) ORDER BY total DESC LIMIT 1"),
        ("Spending by day of week", "SELECT CASE strftime('%w', date) WHEN '0' THEN 'Sunday' WHEN '1' THEN 'Monday' WHEN '2' THEN 'Tuesday' WHEN '3' THEN 'Wednesday' WHEN '4' THEN 'Thursday' WHEN '5' THEN 'Friday' WHEN '6' THEN 'Saturday' END as day, SUM(amount) as total FROM transactions GROUP BY strftime('%w', date) ORDER BY strftime('%w', date)"),
        ("Unusual transactions?", "SELECT * FROM transactions WHERE amount > (SELECT AVG(amount) * 3 FROM transactions) ORDER BY amount DESC"),
        ("Find potential fraud", "SELECT * FROM transactions WHERE amount > 1000 OR (description LIKE '%WIRE%' OR description LIKE '%TRANSFER%') ORDER BY date DESC LIMIT 10"),
    ]
    
    for question, sql in queries:
        add(question, sql, "insights")
        count += 1
    
    return count

def generate_transaction_queries():
    """Transaction search and listing queries."""
    count = 0
    
    # Basic listing
    listings = [
        ("Show recent transactions", "SELECT date, description, amount, category FROM transactions ORDER BY date DESC LIMIT 10"),
        ("Last 10 transactions", "SELECT date, description, amount, category FROM transactions ORDER BY date DESC LIMIT 10"),
        ("Last 20 transactions", "SELECT date, description, amount, category FROM transactions ORDER BY date DESC LIMIT 20"),
        ("Today's transactions", "SELECT date, description, amount, category FROM transactions WHERE date = date('now')"),
        ("Yesterday's transactions", "SELECT date, description, amount, category FROM transactions WHERE date = date('now', '-1 day')"),
        ("Recent activity", "SELECT date, description, amount, category FROM transactions ORDER BY date DESC LIMIT 15"),
    ]
    
    for question, sql in listings:
        add(question, sql, "transactions")
        count += 1
    
    # Amount filters
    amounts = [10, 20, 50, 100, 200, 500, 1000]
    for amt in amounts:
        add(f"Transactions over ${amt}", 
            f"SELECT date, description, amount FROM transactions WHERE amount > {amt} ORDER BY amount DESC",
            "transactions")
        add(f"Purchases over ${amt}",
            f"SELECT date, description, amount FROM transactions WHERE amount > {amt} ORDER BY amount DESC",
            "transactions")
        add(f"Find transactions under ${amt}",
            f"SELECT date, description, amount FROM transactions WHERE amount < {amt} AND amount > 0 ORDER BY date DESC",
            "transactions")
        count += 3
    
    # Search
    searches = [
        ("Show refunds", "SELECT date, description, amount FROM transactions WHERE amount < 0"),
        ("Find credits", "SELECT date, description, amount FROM transactions WHERE amount < 0"),
        ("Pending transactions", "SELECT date, description, amount FROM transactions WHERE status = 'pending'"),
        ("Find ATM withdrawals", "SELECT date, description, amount FROM transactions WHERE description LIKE '%ATM%' OR description LIKE '%WITHDRAW%'"),
        ("Cash withdrawals", "SELECT date, description, amount FROM transactions WHERE description LIKE '%ATM%' OR description LIKE '%CASH%'"),
        ("Find transfers", "SELECT date, description, amount FROM transactions WHERE description LIKE '%TRANSFER%' OR description LIKE '%XFER%'"),
        ("Uncategorized transactions", "SELECT date, description, amount FROM transactions WHERE category IS NULL OR category = 'Other'"),
        ("Find duplicates", "SELECT description, amount, COUNT(*) as count FROM transactions GROUP BY description, amount HAVING count > 1"),
    ]
    
    for question, sql in searches:
        add(question, sql, "transactions")
        count += 1
    
    return count

def generate_comparison_queries():
    """Comparison between different dimensions."""
    count = 0
    
    queries = [
        ("Dining vs groceries?", "SELECT category, SUM(amount) as total FROM transactions WHERE category IN ('Dining', 'Groceries') GROUP BY category"),
        ("Needs vs wants?", "SELECT bucket, SUM(t.amount) as total FROM transactions t JOIN budget_categories bc ON t.category = bc.category GROUP BY bucket"),
        ("Weekday vs weekend spending?", "SELECT CASE WHEN strftime('%w', date) IN ('0', '6') THEN 'Weekend' ELSE 'Weekday' END as day_type, SUM(amount) as total, AVG(amount) as avg_transaction FROM transactions GROUP BY day_type"),
        ("Which card do I use most?", "SELECT source, COUNT(*) as transactions, SUM(amount) as total FROM transactions GROUP BY source ORDER BY transactions DESC"),
        ("Cash vs card spending?", "SELECT CASE WHEN source LIKE '%CASH%' OR source LIKE '%ATM%' THEN 'Cash' ELSE 'Card' END as payment_type, SUM(amount) as total FROM transactions GROUP BY payment_type"),
        ("Online vs in-store?", "SELECT CASE WHEN description LIKE '%.COM%' OR description LIKE '%ONLINE%' OR description LIKE '%AMAZON%' THEN 'Online' ELSE 'In-Store' END as purchase_type, SUM(amount) as total FROM transactions GROUP BY purchase_type"),
        ("Morning vs evening spending?", "SELECT CASE WHEN CAST(strftime('%H', datetime) as INTEGER) < 12 THEN 'Morning' WHEN CAST(strftime('%H', datetime) as INTEGER) < 17 THEN 'Afternoon' ELSE 'Evening' END as time_of_day, SUM(amount) as total FROM transactions GROUP BY time_of_day"),
    ]
    
    for question, sql in queries:
        add(question, sql, "comparisons")
        count += 1
    
    return count

def generate_natural_language_variations():
    """Casual, informal, and varied ways to ask things."""
    count = 0
    
    # Casual spending queries
    casual = [
        ("What'd I spend?", "SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')"),
        ("Where'd my money go?", "SELECT category, SUM(amount) as total FROM transactions WHERE date >= date('now', 'start of month') GROUP BY category ORDER BY total DESC"),
        ("Damage report", "SELECT SUM(amount) as total_spent FROM transactions WHERE date >= date('now', 'start of month')"),
        ("How broke am I?", "SELECT balance FROM v_current_balances WHERE type = 'checking'"),
        ("Money left?", "SELECT SUM(balance) FROM v_current_balances WHERE type IN ('checking', 'savings')"),
        ("Can I afford...", "SELECT SUM(balance) as available FROM v_current_balances WHERE type IN ('checking', 'savings')"),
        ("Bank balance?", "SELECT SUM(balance) FROM v_current_balances WHERE type = 'checking'"),
        ("What's in my account?", "SELECT name, balance FROM v_current_balances"),
        ("Savings update", "SELECT name, balance FROM v_current_balances WHERE type = 'savings'"),
        ("Financial health check", "SELECT SUM(CASE WHEN type IN ('checking', 'savings', 'investment') THEN balance ELSE -balance END) as net_worth FROM v_current_balances"),
        ("Money status", "SELECT type, SUM(balance) as total FROM v_current_balances GROUP BY type"),
        
        # Casual credit
        ("Credit card damage?", "SELECT SUM(balance) FROM v_current_balances WHERE type = 'credit_card'"),
        ("How much debt?", "SELECT SUM(balance) FROM v_current_balances WHERE type IN ('credit_card', 'loan')"),
        ("Cards maxed?", "SELECT name, ROUND(balance * 100.0 / credit_limit, 1) as utilization FROM v_credit_card_summary WHERE balance > credit_limit * 0.8"),
        
        # Casual goals
        ("Savings goals?", "SELECT name, current_amount, target_amount FROM goals WHERE is_active = 1"),
        ("Goal check", "SELECT name, ROUND(current_amount * 100.0 / target_amount, 1) as pct FROM goals WHERE is_active = 1"),
        ("Emergency fund?", "SELECT current_amount, target_amount FROM goals WHERE name LIKE '%emergency%'"),
        
        # Casual specific
        ("Coffee spending", "SELECT SUM(amount) FROM transactions WHERE description LIKE '%COFFEE%' OR description LIKE '%STARBUCKS%' OR description LIKE '%DUNKIN%'"),
        ("Food delivery", "SELECT SUM(amount) FROM transactions WHERE description LIKE '%DOORDASH%' OR description LIKE '%UBEREATS%' OR description LIKE '%GRUBHUB%'"),
        ("Uber total", "SELECT SUM(amount) FROM transactions WHERE description LIKE '%UBER%'"),
        ("Streaming services", "SELECT description, amount FROM transactions WHERE category = 'Subscriptions' AND (description LIKE '%NETFLIX%' OR description LIKE '%HULU%' OR description LIKE '%DISNEY%' OR description LIKE '%HBO%' OR description LIKE '%SPOTIFY%')"),
    ]
    
    for question, sql in casual:
        add(question, sql, "natural")
        count += 1
    
    return count

def generate_action_queries():
    """Queries that modify data."""
    count = 0
    
    # Categorization
    for category in CATEGORIES:
        add(f"Categorize this as {category.lower()}",
            f"UPDATE transactions SET category = '{category}' WHERE id = ?",
            "actions")
        add(f"Mark as {category.lower()}",
            f"UPDATE transactions SET category = '{category}' WHERE id = ?",
            "actions")
        count += 2
    
    # Goals
    goals = [
        ("Create emergency fund goal for $10000", "INSERT INTO goals (id, name, target_amount, current_amount, category) VALUES (?, 'Emergency Fund', 10000, 0, 'emergency')"),
        ("Create vacation fund for $3000", "INSERT INTO goals (id, name, target_amount, current_amount, category) VALUES (?, 'Vacation Fund', 3000, 0, 'vacation')"),
        ("Create car fund for $5000", "INSERT INTO goals (id, name, target_amount, current_amount, category) VALUES (?, 'Car Fund', 5000, 0, 'car')"),
        ("Add $500 to emergency fund", "UPDATE goals SET current_amount = current_amount + 500 WHERE name LIKE '%emergency%'"),
        ("Add $100 to vacation fund", "UPDATE goals SET current_amount = current_amount + 100 WHERE name LIKE '%vacation%'"),
    ]
    
    for question, sql in goals:
        add(question, sql, "actions")
        count += 1
    
    # Budgets
    amounts = [100, 200, 300, 400, 500, 600, 750, 1000]
    for amt in amounts:
        add(f"Set dining budget to ${amt}",
            f"UPDATE budget_categories SET monthly_budget = {amt} WHERE category = 'Dining'",
            "actions")
        add(f"Set grocery budget to ${amt}",
            f"UPDATE budget_categories SET monthly_budget = {amt} WHERE category = 'Groceries'",
            "actions")
        count += 2
    
    return count

def generate_complex_queries():
    """More complex analytical queries."""
    count = 0
    
    queries = [
        # Complex analytics
        ("What's my savings rate?", "SELECT ROUND((SELECT SUM(amount) FROM transactions WHERE category IN ('Savings', 'Investments', '401k')) * 100.0 / (SELECT SUM(amount) FROM transactions WHERE amount > 0), 1) as savings_rate"),
        ("Discretionary vs essential spending", "SELECT CASE WHEN bucket = 'needs' THEN 'Essential' ELSE 'Discretionary' END as type, SUM(t.amount) as total FROM transactions t JOIN budget_categories bc ON t.category = bc.category GROUP BY type"),
        ("Spending velocity this month", "SELECT date, SUM(amount) OVER (ORDER BY date) as cumulative_spending FROM transactions WHERE date >= date('now', 'start of month')"),
        ("Days until I'm broke at this rate", "SELECT ROUND((SELECT SUM(balance) FROM v_current_balances WHERE type = 'checking') / (SELECT AVG(daily_total) FROM (SELECT SUM(amount) as daily_total FROM transactions WHERE date >= date('now', '-30 days') GROUP BY date)), 0) as days_remaining"),
        ("Projected month-end spending", "SELECT ROUND((SELECT SUM(amount) FROM transactions WHERE date >= date('now', 'start of month')) * 30.0 / CAST(strftime('%d', 'now') as REAL), 2) as projected_total"),
        
        # Rolling averages
        ("3-month average spending", "SELECT AVG(monthly_total) as avg_3mo FROM (SELECT SUM(amount) as monthly_total FROM transactions WHERE date >= date('now', '-3 months') GROUP BY strftime('%Y-%m', date))"),
        ("6-month average by category", "SELECT category, AVG(monthly_total) as avg_6mo FROM (SELECT category, strftime('%Y-%m', date) as month, SUM(amount) as monthly_total FROM transactions WHERE date >= date('now', '-6 months') GROUP BY category, month) GROUP BY category ORDER BY avg_6mo DESC"),
        
        # Specific scenarios
        ("Can I afford a $500 purchase?", "SELECT CASE WHEN (SELECT SUM(balance) FROM v_current_balances WHERE type = 'checking') > 500 THEN 'Yes' ELSE 'No' END as can_afford, (SELECT SUM(balance) FROM v_current_balances WHERE type = 'checking') as available"),
        ("What if I cut dining by 50%?", "SELECT SUM(amount) * 0.5 as potential_savings FROM transactions WHERE category = 'Dining' AND date >= date('now', '-3 months') / 3"),
        ("Impact of canceling Netflix", "SELECT SUM(amount) * 12 as yearly_savings FROM transactions WHERE description LIKE '%NETFLIX%' AND date >= date('now', '-1 month')"),
    ]
    
    for question, sql in queries:
        add(question, sql, "complex")
        count += 1
    
    return count

# =============================================================================
# MAIN
# =============================================================================

def main():
    print("🚀 Generating 100,000 training examples for LocalFinance AI")
    print("=" * 60)
    
    counts = {}
    
    print("\n📊 Generating queries...")
    
    counts["spending_time"] = generate_spending_by_time()
    print(f"  ✓ Spending by time: {counts['spending_time']}")
    
    counts["spending_category"] = generate_spending_by_category()
    print(f"  ✓ Spending by category: {counts['spending_category']}")
    
    counts["spending_cat_time"] = generate_spending_by_category_and_time()
    print(f"  ✓ Spending by category+time: {counts['spending_cat_time']}")
    
    counts["merchants"] = generate_merchant_queries()
    print(f"  ✓ Merchant queries: {counts['merchants']}")
    
    counts["category_analysis"] = generate_category_analysis()
    print(f"  ✓ Category analysis: {counts['category_analysis']}")
    
    counts["budget"] = generate_budget_queries()
    print(f"  ✓ Budget queries: {counts['budget']}")
    
    counts["accounts"] = generate_account_queries()
    print(f"  ✓ Account queries: {counts['accounts']}")
    
    counts["goals"] = generate_goal_queries()
    print(f"  ✓ Goal queries: {counts['goals']}")
    
    counts["trends"] = generate_trend_queries()
    print(f"  ✓ Trend queries: {counts['trends']}")
    
    counts["insights"] = generate_insight_queries()
    print(f"  ✓ Insight queries: {counts['insights']}")
    
    counts["transactions"] = generate_transaction_queries()
    print(f"  ✓ Transaction queries: {counts['transactions']}")
    
    counts["comparisons"] = generate_comparison_queries()
    print(f"  ✓ Comparison queries: {counts['comparisons']}")
    
    counts["natural"] = generate_natural_language_variations()
    print(f"  ✓ Natural language: {counts['natural']}")
    
    counts["actions"] = generate_action_queries()
    print(f"  ✓ Action queries: {counts['actions']}")
    
    counts["complex"] = generate_complex_queries()
    print(f"  ✓ Complex queries: {counts['complex']}")
    
    # Shuffle and save
    print("\n📦 Shuffling and saving...")
    random.shuffle(EXAMPLES)
    
    # Save combined
    combined_path = OUTPUT_DIR / "combined.jsonl"
    with open(combined_path, "w") as f:
        for ex in EXAMPLES:
            f.write(json.dumps(ex) + "\n")
    
    # Save by category
    by_category = {}
    for ex in EXAMPLES:
        cat = ex.get("category", "general")
        if cat not in by_category:
            by_category[cat] = []
        by_category[cat].append(ex)
    
    for cat, examples in by_category.items():
        path = OUTPUT_DIR / f"{cat}.jsonl"
        with open(path, "w") as f:
            for ex in examples:
                f.write(json.dumps(ex) + "\n")
    
    # Summary
    total = len(EXAMPLES)
    print("\n" + "=" * 60)
    print(f"🎉 TOTAL EXAMPLES: {total:,}")
    print("=" * 60)
    
    print("\n📂 Category breakdown:")
    for cat, examples in sorted(by_category.items(), key=lambda x: -len(x[1])):
        print(f"   {cat}: {len(examples):,}")
    
    # Progress to 100k
    target = 100000
    pct = (total / target) * 100
    print(f"\n📈 Progress to 100k: {pct:.1f}%")
    print(f"   Need {target - total:,} more examples")
    
    print(f"\n📁 Data saved to: {OUTPUT_DIR}")

if __name__ == "__main__":
    main()
