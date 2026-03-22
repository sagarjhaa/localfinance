#!/usr/bin/env python3
"""
Enhanced Hisab Finance Bot - With document processing
"""
import json
import requests
import asyncio
import sqlite3
import os
from pathlib import Path
from telegram import Update
from telegram.ext import Application, CommandHandler, MessageHandler, filters, ContextTypes

# Load configuration
with open('config.json') as f:
    config = json.load(f)

BOT_TOKEN = config['bot_token']
OLLAMA_HOST = config['ollama_host']
MODEL_NAME = config['model_name']
INBOX_DIR = Path('/home/sagar/localfinance/inbox')
DB_PATH = '/home/sagar/localfinance/data/finances.db'

def query_ollama(prompt):
    """Query Ollama API for finance advice"""
    try:
        response = requests.post(
            f"{OLLAMA_HOST}/api/generate",
            json={
                "model": MODEL_NAME,
                "prompt": f"You are Hisab, a personal finance assistant. Answer this question about finance: {prompt}",
                "stream": False
            },
            timeout=30
        )
        if response.status_code == 200:
            return response.json().get('response', 'Sorry, I could not process that.')
        else:
            return f"Error: {response.status_code}"
    except Exception as e:
        return f"Error connecting to AI: {str(e)}"

def get_recent_transactions(limit=10):
    """Get recent transactions from database"""
    try:
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        
        cursor.execute('''
            SELECT date, description, amount, category, file_source
            FROM transactions 
            ORDER BY created_at DESC 
            LIMIT ?
        ''', (limit,))
        
        transactions = cursor.fetchall()
        conn.close()
        
        return transactions
    except:
        return []

def get_spending_summary():
    """Get spending summary by category"""
    try:
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        
        cursor.execute('''
            SELECT category, COUNT(*) as count, SUM(amount) as total
            FROM transactions 
            WHERE date >= date('now', '-30 days')
            GROUP BY category
            ORDER BY total DESC
        ''')
        
        summary = cursor.fetchall()
        conn.close()
        
        return summary
    except:
        return []

async def start_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /start command"""
    welcome_text = """🏦 **Welcome to Hisab Finance Assistant!**

I'm your personal finance assistant running locally on your Jetson.

💰 **What I can help with:**
• Budgeting and expense planning
• Investment advice and strategies
• Savings goals and emergency funds
• **Document processing** (NEW!)

📄 **Document Processing:**
Drop files in these folders and I'll automatically process them:
• `inbox/statements/` - Bank statements (PDF/CSV)
• `inbox/receipts/` - Receipt files
• `inbox/invoices/` - Invoice files

🔒 **100% Private** - Everything stays on your device!

**Available commands:**
• `/help` - Show help
• `/transactions` - Recent transactions
• `/summary` - Spending summary
• `/status` - Bot and document processor status

Just ask me any finance question! 🚀"""
    
    await update.message.reply_text(welcome_text, parse_mode='Markdown')

async def help_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /help command"""
    help_text = """🆘 **Hisab Finance Assistant Help**

**💬 Chat Commands:**
• Ask any finance question in natural language
• "How should I budget my income?"
• "What's a good savings rate?"

**📋 Bot Commands:**
• `/start` - Welcome message
• `/help` - This help message
• `/transactions` - Show recent transactions
• `/summary` - Monthly spending by category
• `/status` - Check system status

**📄 Document Processing:**
Drop files in these folders on the Jetson:

📁 **`/home/sagar/localfinance/inbox/statements/`**
   → Bank statements (PDF, CSV)
   
📁 **`/home/sagar/localfinance/inbox/receipts/`**
   → Receipt files
   
📁 **`/home/sagar/localfinance/inbox/invoices/`**
   → Invoice files

Files are automatically:
✅ Parsed for transactions
✅ Saved to database  
✅ Moved to `processed/` folder
❌ Failed files go to `failed/` folder

**🔒 Privacy:** Everything runs locally on your Jetson!"""
    
    await update.message.reply_text(help_text, parse_mode='Markdown')

async def transactions_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Show recent transactions"""
    transactions = get_recent_transactions(10)
    
    if not transactions:
        await update.message.reply_text("📄 No transactions found. Drop some bank statements in the inbox folder!")
        return
    
    text = "💳 **Recent Transactions:**\n\n"
    
    for date, desc, amount, category, source in transactions:
        emoji = "💸" if amount < 0 else "💰"
        text += f"{emoji} **${abs(amount):.2f}** - {desc}\n"
        text += f"   📅 {date} | 🏷️ {category} | 📁 {source}\n\n"
    
    if len(text) > 4000:
        text = text[:3900] + "\n... (truncated)"
    
    await update.message.reply_text(text, parse_mode='Markdown')

async def summary_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Show spending summary"""
    summary = get_spending_summary()
    
    if not summary:
        await update.message.reply_text("📊 No spending data available yet. Upload some bank statements!")
        return
    
    text = "📊 **Monthly Spending Summary:**\n\n"
    
    total_spent = 0
    for category, count, amount in summary:
        if amount < 0:  # Expenses
            total_spent += abs(amount)
            text += f"🏷️ **{category}**: ${abs(amount):.2f} ({count} transactions)\n"
    
    text += f"\n💸 **Total Spent:** ${total_spent:.2f}"
    
    await update.message.reply_text(text, parse_mode='Markdown')

async def status_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Show system status"""
    # Check inbox folders
    inbox_status = {}
    for folder in ['statements', 'receipts', 'invoices', 'processed', 'failed']:
        folder_path = INBOX_DIR / folder
        if folder_path.exists():
            count = len(list(folder_path.iterdir()))
            inbox_status[folder] = count
        else:
            inbox_status[folder] = 0
    
    # Check database
    try:
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        cursor.execute('SELECT COUNT(*) FROM transactions')
        txn_count = cursor.fetchone()[0]
        cursor.execute('SELECT COUNT(*) FROM documents')
        doc_count = cursor.fetchone()[0]
        conn.close()
        db_status = f"✅ {txn_count} transactions, {doc_count} documents"
    except:
        db_status = "❌ Database error"
    
    # Check Ollama
    try:
        response = requests.get(f"{OLLAMA_HOST}/api/version", timeout=5)
        ollama_status = "✅ Connected" if response.status_code == 200 else "❌ Error"
    except:
        ollama_status = "❌ Offline"
    
    status_text = f"""📊 **Hisab Bot Status**

🤖 **Bot:** Running
🧠 **AI Model:** {MODEL_NAME}
🔗 **Ollama:** {ollama_status}
💾 **Database:** {db_status}

📁 **Document Inbox:**
• Statements: {inbox_status['statements']} files
• Receipts: {inbox_status['receipts']} files  
• Invoices: {inbox_status['invoices']} files
• Processed: {inbox_status['processed']} files
• Failed: {inbox_status['failed']} files

📂 **Drop files here:**
`/home/sagar/localfinance/inbox/`

Ready for finance questions and document processing! 🚀"""
    
    await update.message.reply_text(status_text, parse_mode='Markdown')

async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle user messages"""
    user_message = update.message.text
    
    # Check if asking about transactions
    if any(word in user_message.lower() for word in ['transaction', 'spending', 'expense', 'recent']):
        # Include transaction data in the query
        recent_txns = get_recent_transactions(5)
        if recent_txns:
            txn_summary = "Recent transactions: " + "; ".join([f"${abs(amt):.2f} on {desc}" for _, desc, amt, _, _ in recent_txns[:3]])
            user_message = f"{user_message}\n\nContext: {txn_summary}"
    
    # Send typing indicator
    await update.message.chat.send_action('typing')
    
    # Get AI response
    ai_response = query_ollama(user_message)
    
    # Send response
    await update.message.reply_text(ai_response)

def main():
    """Start the enhanced bot"""
    print(f"🤖 Starting Enhanced Hisab Finance Bot...")
    print(f"Model: {MODEL_NAME}")
    print(f"Ollama: {OLLAMA_HOST}")
    print(f"Inbox: {INBOX_DIR}")
    
    # Create application
    application = Application.builder().token(BOT_TOKEN).build()
    
    # Add handlers
    application.add_handler(CommandHandler('start', start_command))
    application.add_handler(CommandHandler('help', help_command))
    application.add_handler(CommandHandler('transactions', transactions_command))
    application.add_handler(CommandHandler('summary', summary_command))
    application.add_handler(CommandHandler('status', status_command))
    application.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))
    
    # Start bot
    print("✅ Enhanced Hisab Bot is running!")
    print("📄 Document processing available!")
    application.run_polling()

if __name__ == '__main__':
    main()
