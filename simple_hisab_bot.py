#!/usr/bin/env python3
"""
Hisab Finance Bot - Simplified version using Ollama API directly
"""
import json
import requests
import asyncio
from telegram import Update
from telegram.ext import Application, CommandHandler, MessageHandler, filters, ContextTypes

# Load configuration
with open('config.json') as f:
    config = json.load(f)

BOT_TOKEN = config['bot_token']
OLLAMA_HOST = config['ollama_host']
MODEL_NAME = config['model_name']

def query_ollama(prompt):
    """Query Ollama API directly"""
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

async def start_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Start command handler"""
    welcome_text = """
🏦 **Welcome to Hisab Finance Assistant!**

I'm your personal finance assistant running locally on your Jetson. I can help you with:

💰 **Budgeting & Planning**
📊 **Expense Tracking** 
💳 **Investment Advice**
📈 **Financial Analysis**
🔒 **Complete Privacy** - Everything stays on your device!

Just ask me any finance question!

Examples:
• "How should I budget my salary?"
• "What's the 50/30/20 rule?"
• "Best investment strategy for beginners?"
    """
    await update.message.reply_text(welcome_text, parse_mode='Markdown')

async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle incoming messages"""
    user_message = update.message.text
    
    # Send typing indicator
    await update.message.chat.send_action('typing')
    
    # Get AI response
    ai_response = query_ollama(user_message)
    
    # Send response
    await update.message.reply_text(ai_response)

def main():
    """Start the bot"""
    print(f"🤖 Starting Hisab Finance Bot...")
    print(f"Model: {MODEL_NAME}")
    print(f"Ollama: {OLLAMA_HOST}")
    
    # Create application
    application = Application.builder().token(BOT_TOKEN).build()
    
    # Add handlers
    application.add_handler(CommandHandler('start', start_command))
    application.add_handler(MessageHandler(filters.TEXT & ~filters.COMMAND, handle_message))
    
    # Start bot
    print("✅ Bot is running! Send /start to begin.")
    application.run_polling()

if __name__ == '__main__':
    main()
