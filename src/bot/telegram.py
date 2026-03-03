#!/usr/bin/env python3
"""
LocalFinance Telegram Bot
Combines natural language AI with finance database queries.

Features:
- Natural language queries → SQL → results
- PDF/CSV statement import via file upload
- User settings (date format, categories)
- Comprehensive error handling and logging

Usage:
    BOT_TOKEN=your_token python -m src.bot.telegram
"""

import os
import sys
import json
import sqlite3
import tempfile
from datetime import datetime
from pathlib import Path

# Add project root to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))

# Telegram imports
try:
    from telegram import Update
    from telegram.ext import (
        Application, 
        CommandHandler, 
        MessageHandler, 
        ContextTypes,
        filters
    )
except ImportError:
    print("❌ python-telegram-bot not installed.")
    print("   Install with: pip install python-telegram-bot")
    sys.exit(1)

# Local imports
from src.ai.inference import get_model, query as ai_query
from src.core.config import get_config, list_date_formats, DATE_FORMATS
from src.core.statements import parse_statement
from src.core.database import execute_query, get_transaction_count, init_database
from src.core.logging_config import setup_logging, user_friendly_error, get_logger, log_error_with_code

# =============================================================================
# Configuration
# =============================================================================

PROJECT_ROOT = Path(__file__).parent.parent.parent
DB_PATH = Path(os.environ.get('FINANCE_DB', str(PROJECT_ROOT / 'data' / 'finances.db')))
UPLOAD_DIR = PROJECT_ROOT / "uploads"

# Setup logging
logger = setup_logging("localfinance.bot")


def get_bot_token():
    """Get bot token from environment or config."""
    token = os.environ.get('BOT_TOKEN') or os.environ.get('TELEGRAM_BOT_TOKEN')
    if token:
        return token
    
    config_path = PROJECT_ROOT / "config.json"
    if config_path.exists():
        with open(config_path) as f:
            config = json.load(f)
            return config.get('bot_token') or config.get('telegram_bot_token')
    
    return None


# =============================================================================
# Database Operations
# =============================================================================

def save_transactions(transactions: list) -> tuple[int, int]:
    """
    Save transactions to database.
    Returns (saved_count, duplicate_count)
    """
    init_database(DB_PATH)
    
    saved = 0
    duplicates = 0
    
    conn = sqlite3.connect(str(DB_PATH))
    cursor = conn.cursor()
    
    for tx in transactions:
        try:
            cursor.execute("""
                INSERT INTO transactions (date, description, amount, category, account)
                VALUES (?, ?, ?, ?, ?)
            """, (
                tx['date'],
                tx['description'],
                tx['amount'],
                tx['category'],
                tx.get('source', 'Import')
            ))
            saved += 1
        except sqlite3.IntegrityError:
            duplicates += 1
    
    conn.commit()
    conn.close()
    
    return saved, duplicates


def format_results(results: list, question: str) -> str:
    """Format SQL results for display."""
    if not results:
        return "📭 No results found."
    
    # Single aggregate result (SUM, COUNT, etc.)
    if len(results) == 1 and len(results[0]) == 1:
        key = list(results[0].keys())[0]
        value = results[0][key]
        
        # Handle NULL results
        if value is None:
            return "📭 No matching transactions found."
        
        if isinstance(value, (int, float)):
            if 'total' in key.lower() or 'sum' in key.lower() or 'amount' in key.lower():
                return f"💰 **${value:,.2f}**"
            return f"📊 **{value:,}**"
        return f"📊 {value}"
    
    # Single row with multiple columns
    if len(results) == 1:
        row = results[0]
        msg = "📊 **Result:**\n"
        for key, value in row.items():
            if isinstance(value, float):
                msg += f"• {key}: ${value:,.2f}\n"
            else:
                msg += f"• {key}: {value}\n"
        return msg
    
    # Multiple rows - table format
    msg = f"📊 **{len(results)} results:**\n\n"
    
    for i, row in enumerate(results[:10]):  # Limit to 10 rows
        row_str = " | ".join(
            f"${v:,.2f}" if isinstance(v, float) else str(v)[:30]
            for v in row.values()
        )
        msg += f"{i+1}. {row_str}\n"
    
    if len(results) > 10:
        msg += f"\n_...and {len(results) - 10} more_"
    
    return msg


# =============================================================================
# Query Processing
# =============================================================================

def process_question(question: str) -> str:
    """Process a natural language question end-to-end."""
    
    # Generate SQL using AI
    ai_result = ai_query(question)
    
    if not ai_result['success']:
        logger.error(f"AI query failed: {ai_result.get('error')}")
        return user_friendly_error("model_error", ai_result.get('error'))
    
    sql = ai_result['sql']
    inference_time = ai_result['inference_time']
    
    logger.info(f"Query: {question[:50]} → SQL: {sql[:50]}")
    
    # Execute SQL
    db_result = execute_query(sql, DB_PATH)
    
    if not db_result['success']:
        logger.error(f"SQL execution failed: {db_result.get('error')}")
        return (
            f"❌ Query error: {db_result.get('error', 'Unknown')}\n\n"
            f"Generated SQL:\n`{sql}`"
        )
    
    # Format results
    response = format_results(db_result['results'], question)
    response += f"\n\n_Query: `{sql[:60]}{'...' if len(sql) > 60 else ''}`_"
    response += f"\n_AI: {inference_time:.1f}s_"
    
    return response


# =============================================================================
# Telegram Command Handlers
# =============================================================================

async def cmd_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /start command."""
    logger.info(f"New user: {update.effective_user.id}")
    
    await update.message.reply_text(
        "👋 **Welcome to LocalFinance!**\n\n"
        "I'm your personal finance assistant, running 100% locally on your device.\n\n"
        "**Getting Started:**\n"
        "1️⃣ Send me a PDF or CSV bank statement\n"
        "2️⃣ I'll import your transactions\n"
        "3️⃣ Ask me anything about your spending!\n\n"
        "**Example questions:**\n"
        "• \"How much did I spend on dining?\"\n"
        "• \"Show me Amazon purchases\"\n"
        "• \"What are my subscriptions?\"\n\n"
        "**Commands:**\n"
        "/help — Full help\n"
        "/status — Check system status\n"
        "/settings — Date format & preferences\n\n"
        "🔒 _All data stays on this device. No cloud._",
        parse_mode='Markdown'
    )


async def cmd_help(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /help command."""
    await update.message.reply_text(
        "🤖 **LocalFinance Help**\n\n"
        "**📄 Importing Statements**\n"
        "Just send me a PDF or CSV! I support:\n"
        "• Chase (credit & checking)\n"
        "• Capital One (Savor, Quicksilver, Venture)\n"
        "• American Express\n"
        "• Generic CSV files\n\n"
        "**💬 Asking Questions**\n"
        "• \"How much did I spend on [category]?\"\n"
        "• \"Show me transactions over $100\"\n"
        "• \"Compare dining vs groceries\"\n"
        "• \"Find Amazon purchases\"\n\n"
        "**⚙️ Commands**\n"
        "/status — System status & transaction count\n"
        "/settings — Configure date format\n"
        "/categories — View spending by category\n"
        "/support — Get help with issues\n"
        "/test — Run diagnostic test\n\n"
        "**🔒 Privacy**\n"
        "Everything runs locally. Your data never leaves this device.",
        parse_mode='Markdown'
    )


async def cmd_status(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /status command."""
    # Check model
    model = get_model()
    model_status = "✅ Loaded" if model._loaded else "⏳ Not loaded (loads on first query)"
    
    # Check database
    db_exists = DB_PATH.exists()
    db_status = f"✅ Connected" if db_exists else f"⏳ Will be created on first import"
    
    # Get transaction count
    tx_count = get_transaction_count(DB_PATH) if db_exists else 0
    
    # Get config
    config = get_config()
    
    await update.message.reply_text(
        "📊 **LocalFinance Status**\n\n"
        f"**AI Model:** {model_status}\n"
        f"**Database:** {db_status}\n"
        f"**Transactions:** {tx_count:,}\n"
        f"**Date Format:** {config.date_format_name}\n\n"
        "_All processing happens locally on this device._",
        parse_mode='Markdown'
    )


async def cmd_settings(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /settings command."""
    args = context.args
    config = get_config()
    
    if not args:
        # Show current settings
        msg = "⚙️ **Settings**\n\n"
        msg += f"**Date Format:** {config.date_format_name}\n\n"
        msg += "To change date format:\n"
        msg += "`/settings date us` — US (MM/DD/YYYY)\n"
        msg += "`/settings date eu` — European (DD/MM/YYYY)\n"
        msg += "`/settings date iso` — ISO (YYYY-MM-DD)\n\n"
        msg += list_date_formats()
        
        await update.message.reply_text(msg, parse_mode='Markdown')
        return
    
    if args[0] == "date" and len(args) > 1:
        format_key = args[1].lower()
        if config.set_date_format(format_key):
            await update.message.reply_text(
                f"✅ Date format set to **{config.date_format_name}**",
                parse_mode='Markdown'
            )
            logger.info(f"User changed date format to: {format_key}")
        else:
            await update.message.reply_text(
                f"❌ Unknown format: `{format_key}`\n\n{list_date_formats()}",
                parse_mode='Markdown'
            )
    else:
        await update.message.reply_text("❓ Unknown setting. Use `/settings` to see options.", parse_mode='Markdown')


async def cmd_categories(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /categories command - show spending by category."""
    result = execute_query("""
        SELECT category, SUM(amount) as total, COUNT(*) as count
        FROM transactions
        GROUP BY category
        ORDER BY total DESC
        LIMIT 15
    """, DB_PATH)
    
    if not result['success'] or not result['results']:
        await update.message.reply_text("📭 No transactions yet. Send a statement to get started!")
        return
    
    msg = "📊 **Spending by Category**\n\n"
    total = sum(r['total'] for r in result['results'])
    
    for r in result['results']:
        pct = (r['total'] / total * 100) if total > 0 else 0
        bar = "█" * int(pct / 5) + "░" * (20 - int(pct / 5))
        msg += f"**{r['category']}**\n"
        msg += f"${r['total']:,.2f} ({r['count']} txns)\n"
        msg += f"`{bar}` {pct:.1f}%\n\n"
    
    msg += f"**Total:** ${total:,.2f}"
    
    await update.message.reply_text(msg, parse_mode='Markdown')


async def cmd_support(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /support command - show how to get help."""
    await update.message.reply_text(
        "🆘 **Need Help?**\n\n"
        "If you're having issues:\n\n"
        "1️⃣ **Screenshot the error message** (include the error code like `LF-A1B2`)\n"
        "2️⃣ **Forward it to support**\n"
        "3️⃣ **Describe what you were trying to do**\n\n"
        "**Common issues:**\n"
        "• _PDF not importing_ → Try CSV export from your bank\n"
        "• _Wrong amounts_ → Check /settings for date format\n"
        "• _Missing transactions_ → Some banks need specific PDF downloads\n\n"
        "The error code helps us find exactly what went wrong in the logs.",
        parse_mode='Markdown'
    )


async def cmd_test(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /test command - run diagnostic test."""
    await update.message.reply_text("🧪 Running diagnostics...")
    
    results = []
    
    # Test 1: Model load
    try:
        model = get_model()
        if not model._loaded:
            model.load()
        results.append("✅ AI model loaded")
    except Exception as e:
        results.append(f"❌ AI model: {str(e)[:30]}")
    
    # Test 2: Database
    try:
        tx_count = get_transaction_count(DB_PATH)
        results.append(f"✅ Database: {tx_count} transactions")
    except Exception as e:
        results.append(f"❌ Database: {str(e)[:30]}")
    
    # Test 3: Query
    try:
        result = ai_query("Show all transactions")
        if result['success']:
            results.append(f"✅ AI query: {result['inference_time']:.1f}s")
        else:
            results.append(f"❌ AI query: {result.get('error', 'Failed')[:30]}")
    except Exception as e:
        results.append(f"❌ AI query: {str(e)[:30]}")
    
    msg = "🧪 **Diagnostic Results**\n\n"
    msg += "\n".join(results)
    
    await update.message.reply_text(msg, parse_mode='Markdown')


# =============================================================================
# File Upload Handler
# =============================================================================

async def handle_document(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle uploaded PDF/CSV files."""
    document = update.message.document
    file_name = document.file_name or "unknown"
    
    logger.info(f"Received file: {file_name} ({document.file_size} bytes)")
    
    # Check file type
    suffix = Path(file_name).suffix.lower()
    if suffix not in ['.pdf', '.csv']:
        await update.message.reply_text(
            "📄 Please send a **PDF** or **CSV** bank statement.\n\n"
            "Most banks let you download statements from their website.",
            parse_mode='Markdown'
        )
        return
    
    # Send processing message
    status_msg = await update.message.reply_text(f"📥 Processing `{file_name}`...", parse_mode='Markdown')
    
    try:
        # Download file
        UPLOAD_DIR.mkdir(parents=True, exist_ok=True)
        file_path = UPLOAD_DIR / f"{datetime.now().strftime('%Y%m%d_%H%M%S')}_{file_name}"
        
        file = await document.get_file()
        await file.download_to_drive(str(file_path))
        
        logger.info(f"Downloaded to: {file_path}")
        
        # Parse statement
        config = get_config()
        result = parse_statement(file_path, config.categorize)
        
        if not result.success:
            logger.error(f"Parse failed: {result.error}")
            await status_msg.edit_text(
                f"❌ **Import Failed**\n\n{result.error}\n\n"
                "💡 _Tip: Try downloading a CSV from your bank's website._",
                parse_mode='Markdown'
            )
            return
        
        # Save transactions
        saved, duplicates = save_transactions(result.transactions)
        
        # Build response
        msg = f"✅ **Import Complete**\n\n"
        msg += f"📄 **Source:** {result.source}\n"
        msg += f"📊 **Transactions:** {len(result.transactions)} found\n"
        msg += f"💾 **Saved:** {saved} new\n"
        
        if duplicates > 0:
            msg += f"🔄 **Duplicates:** {duplicates} skipped\n"
        
        if result.warnings:
            msg += f"\n⚠️ **Warnings:**\n"
            for w in result.warnings[:3]:
                msg += f"• {w}\n"
        
        msg += f"\n_Try asking: \"How much did I spend this month?\"_"
        
        await status_msg.edit_text(msg, parse_mode='Markdown')
        logger.info(f"Import complete: {saved} saved, {duplicates} duplicates")
        
    except Exception as e:
        error_code = log_error_with_code("parse_failed", str(e), exception=e)
        await status_msg.edit_text(
            f"❌ **Error processing file**\n\n"
            f"Something went wrong while reading your statement.\n\n"
            f"📋 **Error code:** `{error_code}`\n"
            f"_Forward this message to support if you need help._\n\n"
            f"💡 _Tip: Try downloading a CSV export from your bank's website._",
            parse_mode='Markdown'
        )


# =============================================================================
# Message Handler
# =============================================================================

async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle natural language messages."""
    question = update.message.text.strip()
    
    if not question:
        return
    
    logger.info(f"Question: {question[:50]}...")
    
    # Send "thinking" indicator
    thinking_msg = await update.message.reply_text("🤔 Thinking...")
    
    try:
        response = process_question(question)
        await thinking_msg.edit_text(response, parse_mode='Markdown')
    except Exception as e:
        error_code = log_error_with_code("model_error", f"Question: {question[:50]} | Error: {str(e)}", exception=e)
        await thinking_msg.edit_text(
            f"❌ **Something went wrong**\n\n"
            f"I couldn't process your question. Try rephrasing it.\n\n"
            f"📋 **Error code:** `{error_code}`\n"
            f"_Forward this message to support if you need help._",
            parse_mode='Markdown'
        )


# =============================================================================
# Main
# =============================================================================

def main():
    """Start the bot."""
    token = get_bot_token()
    
    if not token:
        logger.error("No bot token found!")
        print("❌ No bot token found!")
        print("   Set BOT_TOKEN environment variable or add to config.json")
        sys.exit(1)
    
    logger.info("Starting LocalFinance Bot...")
    print("🚀 Starting LocalFinance Bot...")
    print(f"   Database: {DB_PATH}")
    print(f"   Logs: {PROJECT_ROOT / 'logs'}")
    
    # Pre-load model
    print("   Loading AI model...")
    model = get_model()
    model.load()
    
    # Initialize database
    init_database(DB_PATH)
    
    # Build application
    app = Application.builder().token(token).build()
    
    # Add handlers
    app.add_handler(CommandHandler("start", cmd_start))
    app.add_handler(CommandHandler("help", cmd_help))
    app.add_handler(CommandHandler("status", cmd_status))
    app.add_handler(CommandHandler("settings", cmd_settings))
    app.add_handler(CommandHandler("categories", cmd_categories))
    app.add_handler(CommandHandler("support", cmd_support))
    app.add_handler(CommandHandler("test", cmd_test))
    app.add_handler(MessageHandler(filters.Document.ALL, handle_document))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))
    
    logger.info("Bot is running!")
    print("✅ Bot is running! Press Ctrl+C to stop.")
    app.run_polling(allowed_updates=Update.ALL_TYPES)


if __name__ == "__main__":
    main()
