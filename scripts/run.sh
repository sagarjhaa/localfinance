#!/bin/bash
# LocalFinance Bot Runner
# Usage: ./scripts/run.sh [--bg]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"
source .venv/bin/activate

# Check for token
if [ -z "$BOT_TOKEN" ]; then
    if [ -f "config.json" ]; then
        echo "📋 Using token from config.json"
    else
        echo "❌ No BOT_TOKEN set and no config.json found"
        echo "   Set with: export BOT_TOKEN=your_token"
        exit 1
    fi
fi

# Optional background mode
if [ "$1" == "--bg" ]; then
    echo "🚀 Starting LocalFinance bot in background..."
    nohup python -m src.bot.telegram > logs/bot.log 2>&1 &
    echo $! > .bot.pid
    echo "   PID: $(cat .bot.pid)"
    echo "   Logs: logs/bot.log"
else
    echo "🚀 Starting LocalFinance bot..."
    python -m src.bot.telegram
fi
