#!/usr/bin/env python3
"""
LocalFinance Telegram Bot
Combines natural language AI with finance database queries.

Usage:
    BOT_TOKEN=your_token python3 bot.py
"""

import os
import sys
import json
import sqlite3
import logging
from datetime import datetime
from pathlib import Path

# Add parent to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

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
from src.inference import get_model, query as ai_query

# =============================================================================
# Configuration
# =============================================================================

logging.basicConfig(
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    level=logging.INFO
)
logger = logging.getLogger(__name__)

# Database path (configurable)
DB_PATH = os.environ.get('FINANCE_DB', str(Path(__file__).parent.parent / 'test_data.db'))


def get_bot_token():
    """Get bot token from environment or config."""
    token = os.environ.get('BOT_TOKEN') or os.environ.get('TELEGRAM_BOT_TOKEN')
    if token:
        return token
    
    config_path = Path(__file__).parent.parent / "config.json"
    if config_path.exists():
        with open(config_path) as f:
            config = json.load(f)
            return config.get('bot_token') or config.get('telegram_bot_token')
    
    return None


# =============================================================================
# Database Execution
# =============================================================================

def execute_sql(sql: str, db_path: str = None) -> dict:
    """Execute SQL and return results."""
    db = db_path or DB_PATH
    
    if not Path(db).exists():
        return {'success': False, 'error': f'Database not found: {db}'}
    
    try:
        conn = sqlite3.connect(db)
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
# Main Query Handler
# =============================================================================

def process_question(question: str) -> str:
    """Process a natural language question end-to-end."""
    
    # Generate SQL using AI
    ai_result = ai_query(question)
    
    if not ai_result['success']:
        return f"❌ AI Error: {ai_result.get('error', 'Unknown error')}"
    
    sql = ai_result['sql']
    inference_time = ai_result['inference_time']
    
    # Execute SQL
    db_result = execute_sql(sql)
    
    if not db_result['success']:
        return (
            f"❌ SQL Error: {db_result.get('error', 'Unknown error')}\n\n"
            f"Generated SQL:\n`{sql}`"
        )
    
    # Format results
    response = format_results(db_result['results'], question)
    response += f"\n\n_Query: `{sql[:60]}{'...' if len(sql) > 60 else ''}`_"
    response += f"\n_AI: {inference_time:.1f}s_"
    
    return response


# =============================================================================
# Telegram Handlers
# =============================================================================

async def cmd_start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /start command."""
    await update.message.reply_text(
        "👋 **Welcome to LocalFinance!**\n\n"
        "I'm your personal finance assistant, running 100% locally on your device.\n\n"
        "**Ask me anything about your finances:**\n"
        "• \"How much did I spend on dining last month?\"\n"
        "• \"Show me all Amazon purchases\"\n"
        "• \"What's my total spending this week?\"\n"
        "• \"Find all grocery transactions\"\n\n"
        "**Commands:**\n"
        "/help - Show this message\n"
        "/status - Check system status\n\n"
        "🔒 _All data stays on your device. No cloud. No tracking._",
        parse_mode='Markdown'
    )


async def cmd_help(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /help command."""
    await update.message.reply_text(
        "🤖 **LocalFinance Help**\n\n"
        "**Example Questions:**\n"
        "• How much did I spend on [category]?\n"
        "• Show me transactions over $100\n"
        "• What are my subscriptions?\n"
        "• Compare dining vs groceries\n"
        "• Find transactions from [merchant]\n\n"
        "**Categories I understand:**\n"
        "Dining, Groceries, Shopping, Entertainment, "
        "Transportation, Subscriptions, Utilities, etc.\n\n"
        "**Time periods:**\n"
        "• this week / this month / last month\n"
        "• January / February / etc.\n"
        "• 2024 / 2025 / etc.",
        parse_mode='Markdown'
    )


async def cmd_status(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /status command."""
    # Check model
    model = get_model()
    model_status = "✅ Loaded" if model._loaded else "⏳ Not loaded (loads on first query)"
    
    # Check database
    db_exists = Path(DB_PATH).exists()
    db_status = f"✅ {DB_PATH}" if db_exists else f"❌ Not found: {DB_PATH}"
    
    # Get transaction count if DB exists
    tx_count = 0
    if db_exists:
        try:
            conn = sqlite3.connect(DB_PATH)
            cursor = conn.cursor()
            cursor.execute("SELECT COUNT(*) FROM transactions")
            tx_count = cursor.fetchone()[0]
            conn.close()
        except:
            pass
    
    await update.message.reply_text(
        "📊 **LocalFinance Status**\n\n"
        f"**AI Model:** {model_status}\n"
        f"**Database:** {db_status}\n"
        f"**Transactions:** {tx_count:,}\n\n"
        "_All processing happens locally on this device._",
        parse_mode='Markdown'
    )


async def cmd_test(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /test command - run a quick end-to-end test."""
    await update.message.reply_text("🧪 Running end-to-end test...")
    
    test_questions = [
        "Show all transactions",
        "How much did I spend total?",
    ]
    
    results = []
    for q in test_questions:
        try:
            response = process_question(q)
            passed = "❌" not in response
            results.append(f"{'✅' if passed else '❌'} {q[:30]}")
        except Exception as e:
            results.append(f"❌ {q[:30]}: {str(e)[:20]}")
    
    msg = "🧪 **Test Results**\n\n"
    msg += "\n".join(results)
    msg += "\n\n_Test complete!_"
    
    await update.message.reply_text(msg, parse_mode='Markdown')


async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle natural language messages."""
    question = update.message.text.strip()
    
    if not question:
        return
    
    # Send "thinking" indicator
    thinking_msg = await update.message.reply_text("🤔 Thinking...")
    
    try:
        response = process_question(question)
        await thinking_msg.edit_text(response, parse_mode='Markdown')
    except Exception as e:
        logger.error(f"Error processing question: {e}")
        await thinking_msg.edit_text(f"❌ Error: {str(e)}")


# =============================================================================
# Main
# =============================================================================

def main():
    """Start the bot."""
    token = get_bot_token()
    
    if not token:
        print("❌ No bot token found!")
        print("   Set BOT_TOKEN environment variable or add to config.json")
        sys.exit(1)
    
    print("🚀 Starting LocalFinance Bot...")
    print(f"   Database: {DB_PATH}")
    
    # Pre-load model
    print("   Loading AI model...")
    model = get_model()
    model.load()
    
    # Build application
    app = Application.builder().token(token).build()
    
    # Add handlers
    app.add_handler(CommandHandler("start", cmd_start))
    app.add_handler(CommandHandler("help", cmd_help))
    app.add_handler(CommandHandler("status", cmd_status))
    app.add_handler(CommandHandler("test", cmd_test))
    app.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))
    
    print("✅ Bot is running! Press Ctrl+C to stop.")
    app.run_polling(allowed_updates=Update.ALL_TYPES)


if __name__ == "__main__":
    main()
