#!/usr/bin/env python3
"""
LocalFinance Database Module
SQLite database operations for transaction storage and queries.
"""

import os
import sqlite3
from pathlib import Path
from typing import Optional

# Project root
PROJECT_ROOT = Path(__file__).parent.parent.parent
DEFAULT_DB_PATH = PROJECT_ROOT / "data" / "finances.db"


def get_db_path() -> Path:
    """Get database path from environment or default."""
    env_path = os.environ.get('FINANCE_DB')
    if env_path:
        return Path(env_path)
    return DEFAULT_DB_PATH


def execute_query(sql: str, db_path: Optional[Path] = None) -> dict:
    """
    Execute a SQL query and return results.
    
    Returns:
        dict with keys: success, results (list of dicts), row_count, error
    """
    db = db_path or get_db_path()
    
    if not db.exists():
        return {'success': False, 'error': f'Database not found: {db}'}
    
    try:
        conn = sqlite3.connect(str(db))
        conn.row_factory = sqlite3.Row
        cursor = conn.cursor()
        
        cursor.execute(sql)
        rows = cursor.fetchall()
        
        # Convert to list of dicts
        results = [dict(row) for row in rows]
        
        conn.close()
        
        return {
            'success': True,
            'results': results,
            'row_count': len(results)
        }
    except sqlite3.Error as e:
        return {
            'success': False,
            'error': str(e),
            'sql': sql
        }


def get_transaction_count(db_path: Optional[Path] = None) -> int:
    """Get total transaction count."""
    db = db_path or get_db_path()
    
    if not db.exists():
        return 0
    
    try:
        conn = sqlite3.connect(str(db))
        cursor = conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM transactions")
        count = cursor.fetchone()[0]
        conn.close()
        return count
    except:
        return 0


def get_categories(db_path: Optional[Path] = None) -> list:
    """Get list of unique categories."""
    result = execute_query(
        "SELECT DISTINCT category FROM transactions ORDER BY category",
        db_path
    )
    if result['success']:
        return [r['category'] for r in result['results']]
    return []


def init_database(db_path: Optional[Path] = None):
    """Initialize database with schema if it doesn't exist."""
    db = db_path or get_db_path()
    db.parent.mkdir(parents=True, exist_ok=True)
    
    conn = sqlite3.connect(str(db))
    cursor = conn.cursor()
    
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS transactions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            date TEXT NOT NULL,
            description TEXT NOT NULL,
            amount REAL NOT NULL,
            category TEXT,
            account TEXT,
            source_file TEXT,
            imported_at TEXT DEFAULT CURRENT_TIMESTAMP
        )
    """)
    
    cursor.execute("""
        CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date)
    """)
    
    cursor.execute("""
        CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(category)
    """)
    
    conn.commit()
    conn.close()


if __name__ == "__main__":
    # Quick test
    print(f"Database path: {get_db_path()}")
    print(f"Transaction count: {get_transaction_count()}")
    print(f"Categories: {get_categories()}")
