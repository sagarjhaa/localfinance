#!/usr/bin/env python3
"""
Hisab Document Processor - Parse financial documents
"""
import os
import json
import sqlite3
import time
from pathlib import Path
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
import pdfplumber
import pandas as pd
from datetime import datetime
import re

class DocumentProcessor(FileSystemEventHandler):
    def __init__(self, inbox_dir, db_path):
        self.inbox_dir = Path(inbox_dir)
        self.db_path = db_path
        self.setup_database()
    
    def setup_database(self):
        """Create database tables for financial data"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        # Transactions table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS transactions (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                date TEXT,
                description TEXT,
                amount REAL,
                category TEXT,
                account TEXT,
                file_source TEXT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        ''')
        
        # Documents table
        cursor.execute('''
            CREATE TABLE IF NOT EXISTS documents (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                filename TEXT,
                file_type TEXT,
                status TEXT,
                processed_at TIMESTAMP,
                transactions_count INTEGER DEFAULT 0,
                error_message TEXT
            )
        ''')
        
        conn.commit()
        conn.close()
    
    def on_created(self, event):
        if not event.is_dir:
            print(f"📄 New file detected: {event.src_path}")
            time.sleep(1)  # Wait for file to be fully written
            self.process_file(event.src_path)
    
    def process_file(self, file_path):
        """Process a financial document"""
        file_path = Path(file_path)
        filename = file_path.name
        
        print(f"🔍 Processing: {filename}")
        
        try:
            if file_path.suffix.lower() == '.pdf':
                transactions = self.process_pdf(file_path)
            elif file_path.suffix.lower() == '.csv':
                transactions = self.process_csv(file_path)
            else:
                raise Exception(f"Unsupported file type: {file_path.suffix}")
            
            # Save to database
            if transactions:
                self.save_transactions(transactions, filename)
                self.move_file(file_path, self.inbox_dir / 'processed')
                print(f"✅ Processed {len(transactions)} transactions from {filename}")
            else:
                self.move_file(file_path, self.inbox_dir / 'failed')
                print(f"⚠️ No transactions found in {filename}")
                
        except Exception as e:
            print(f"❌ Error processing {filename}: {str(e)}")
            self.move_file(file_path, self.inbox_dir / 'failed')
            self.log_error(filename, str(e))
    
    def process_pdf(self, file_path):
        """Extract transactions from PDF bank statement"""
        transactions = []
        
        try:
            with pdfplumber.open(file_path) as pdf:
                for page in pdf.pages:
                    text = page.extract_text()
                    if not text:
                        continue
                    
                    # Look for transaction patterns
                    lines = text.split('\n')
                    for line in lines:
                        # Pattern: Date Amount Description
                        match = re.search(r'(\d{1,2}/\d{1,2}/\d{2,4}).*?([+-]?\$?[\d,]+\.\d{2})\s+(.+)', line)
                        if match:
                            date_str, amount_str, description = match.groups()
                            
                            # Clean up amount
                            amount = float(re.sub(r'[\$,]', '', amount_str))
                            
                            transactions.append({
                                'date': self.parse_date(date_str),
                                'amount': amount,
                                'description': description.strip(),
                                'category': self.categorize_transaction(description),
                                'account': 'Bank Statement'
                            })
        except Exception as e:
            print(f"PDF processing error: {e}")
        
        return transactions
    
    def process_csv(self, file_path):
        """Extract transactions from CSV file"""
        transactions = []
        
        try:
            df = pd.read_csv(file_path)
            
            # Try to identify columns
            date_cols = ['date', 'Date', 'Transaction Date', 'Posting Date']
            amount_cols = ['amount', 'Amount', 'Debit', 'Credit']
            desc_cols = ['description', 'Description', 'Transaction Description']
            
            date_col = next((col for col in date_cols if col in df.columns), None)
            amount_col = next((col for col in amount_cols if col in df.columns), None)
            desc_col = next((col for col in desc_cols if col in df.columns), None)
            
            if not all([date_col, amount_col, desc_col]):
                raise Exception("Could not identify required columns in CSV")
            
            for _, row in df.iterrows():
                transactions.append({
                    'date': str(row[date_col]),
                    'amount': float(str(row[amount_col]).replace(',', '').replace('$', '')),
                    'description': str(row[desc_col]),
                    'category': self.categorize_transaction(str(row[desc_col])),
                    'account': 'CSV Import'
                })
                
        except Exception as e:
            raise Exception(f"CSV processing error: {str(e)}")
        
        return transactions
    
    def categorize_transaction(self, description):
        """Simple transaction categorization"""
        desc_lower = description.lower()
        
        if any(word in desc_lower for word in ['grocery', 'food', 'restaurant', 'cafe']):
            return 'Food & Dining'
        elif any(word in desc_lower for word in ['gas', 'fuel', 'shell', 'chevron']):
            return 'Transportation'
        elif any(word in desc_lower for word in ['amazon', 'store', 'shop']):
            return 'Shopping'
        elif any(word in desc_lower for word in ['electric', 'water', 'internet', 'phone']):
            return 'Utilities'
        elif any(word in desc_lower for word in ['rent', 'mortgage']):
            return 'Housing'
        else:
            return 'Other'
    
    def parse_date(self, date_str):
        """Parse date string to standard format"""
        try:
            # Try common formats
            for fmt in ['%m/%d/%Y', '%m/%d/%y', '%Y-%m-%d']:
                try:
                    return datetime.strptime(date_str, fmt).strftime('%Y-%m-%d')
                except ValueError:
                    continue
            return date_str
        except:
            return date_str
    
    def save_transactions(self, transactions, filename):
        """Save transactions to database"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        for txn in transactions:
            cursor.execute('''
                INSERT INTO transactions (date, description, amount, category, account, file_source)
                VALUES (?, ?, ?, ?, ?, ?)
            ''', (txn['date'], txn['description'], txn['amount'], 
                  txn['category'], txn['account'], filename))
        
        # Log the document
        cursor.execute('''
            INSERT INTO documents (filename, file_type, status, processed_at, transactions_count)
            VALUES (?, ?, ?, ?, ?)
        ''', (filename, Path(filename).suffix, 'processed', 
              datetime.now().isoformat(), len(transactions)))
        
        conn.commit()
        conn.close()
    
    def log_error(self, filename, error_msg):
        """Log processing error"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        cursor.execute('''
            INSERT INTO documents (filename, file_type, status, processed_at, error_message)
            VALUES (?, ?, ?, ?, ?)
        ''', (filename, Path(filename).suffix, 'failed', 
              datetime.now().isoformat(), error_msg))
        
        conn.commit()
        conn.close()
    
    def move_file(self, src, dest_dir):
        """Move processed file to appropriate directory"""
        dest_dir.mkdir(exist_ok=True)
        dest_path = dest_dir / Path(src).name
        Path(src).rename(dest_path)

def main():
    inbox_dir = Path('/home/sagar/localfinance/inbox')
    db_path = '/home/sagar/localfinance/data/finances.db'
    
    print(f"📁 Monitoring inbox: {inbox_dir}")
    print(f"💾 Database: {db_path}")
    
    # Create processor
    processor = DocumentProcessor(inbox_dir, db_path)
    
    # Setup file watcher
    observer = Observer()
    observer.schedule(processor, str(inbox_dir), recursive=False)
    observer.start()
    
    print("✅ Document processor started! Drop files in inbox/")
    
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        observer.stop()
        print("\n📄 Document processor stopped")
    
    observer.join()

if __name__ == '__main__':
    main()
