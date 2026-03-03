#!/bin/bash
# LocalFinance Setup Script
# Run this on a new device (Raspberry Pi, Mac Mini, etc.)

set -e

echo "🏦 LocalFinance Setup"
echo "===================="

# Check Python version
PYTHON_VERSION=$(python3 --version 2>&1 | cut -d' ' -f2 | cut -d'.' -f1,2)
echo "Python version: $PYTHON_VERSION"

if [[ "$PYTHON_VERSION" < "3.10" ]]; then
    echo "❌ Python 3.10+ required. Please upgrade."
    exit 1
fi

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"
echo "📁 Project directory: $PROJECT_DIR"

# Create virtual environment if needed
if [ ! -d ".venv" ]; then
    echo "📦 Creating virtual environment..."
    python3 -m venv .venv
fi

# Activate venv
source .venv/bin/activate

# Install dependencies
echo "📥 Installing dependencies..."
pip install --upgrade pip
pip install -r requirements.txt

# Check for model
if [ ! -d "models/localfinance-v1" ]; then
    echo ""
    echo "⚠️  Model not found at models/localfinance-v1/"
    echo "   Copy the trained model directory here, or download from:"
    echo "   (Model hosting TBD)"
    echo ""
fi

# Check for database
if [ ! -f "test_data.db" ]; then
    echo ""
    echo "⚠️  No database found (test_data.db)"
    echo "   The bot needs transaction data to work."
    echo "   Copy your finances database or create with sample data."
    echo ""
fi

# Check for bot token
if [ -z "$BOT_TOKEN" ]; then
    echo ""
    echo "⚠️  BOT_TOKEN not set"
    echo "   Get a token from @BotFather on Telegram"
    echo "   Then run: export BOT_TOKEN=your_token_here"
    echo ""
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "To start the bot:"
echo "  cd $PROJECT_DIR"
echo "  source .venv/bin/activate"
echo "  export BOT_TOKEN=your_token"
echo "  python src/bot.py"
echo ""
