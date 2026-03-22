#!/bin/bash
#
# LocalFinance Production Installer
# Complete automated installation for Jetson devices
#

set -euo pipefail

echo "🚀 LocalFinance Production Installer"
echo "===================================="

# Configuration
INSTALL_DIR="/home/sagar/localfinance"
BOT_TOKEN="$1"
OLLAMA_MODEL="llama3.2:1b"

if [ -z "$BOT_TOKEN" ]; then
    echo "❌ Usage: $0 <telegram_bot_token>"
    exit 1
fi

echo "📦 Installing dependencies..."
sudo apt update
sudo apt install -y python3-pip sqlite3 curl

echo "🤖 Installing Ollama..."
curl -fsSL https://ollama.ai/install.sh | sh
systemctl --user enable ollama
systemctl --user start ollama

echo "📥 Downloading AI model..."
ollama pull $OLLAMA_MODEL

echo "📁 Setting up directories..."
mkdir -p $INSTALL_DIR/{data,logs,inbox/{statements,receipts,processed,failed}}

echo "🐍 Installing Python dependencies..."
pip3 install python-telegram-bot flask pdfplumber pandas watchdog requests

echo "⚙️ Creating configuration..."
cat > $INSTALL_DIR/config.json << EOFCONFIG
{
    "bot_token": "$BOT_TOKEN",
    "model_name": "$OLLAMA_MODEL",
    "ollama_host": "http://127.0.0.1:11434",
    "db_path": "data/finances.db"
}
EOFCONFIG

echo "🗄️ Initializing database..."
sqlite3 $INSTALL_DIR/data/finances.db << EOFSQL
CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    category TEXT,
    account TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    file_type TEXT NOT NULL,
    status TEXT NOT NULL,
    processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    transaction_count INTEGER DEFAULT 0
);
EOFSQL

echo "🔧 Creating SystemD services..."
sudo tee /etc/systemd/system/hisab-bot.service > /dev/null << EOFSERVICE
[Unit]
Description=Hisab Finance Bot
After=network.target ollama.service
Wants=ollama.service

[Service]
Type=simple
User=sagar
Group=sagar
WorkingDirectory=$INSTALL_DIR
ExecStart=/usr/bin/python3 enhanced_hisab_bot.py
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOFSERVICE

sudo tee /etc/systemd/system/hisab-web.service > /dev/null << EOFWEBSERVICE
[Unit]
Description=Hisab Web Dashboard
After=network.target

[Service]
Type=simple
User=sagar
Group=sagar
WorkingDirectory=$INSTALL_DIR
ExecStart=/usr/bin/python3 dashboard/app.py
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOFWEBSERVICE

echo "🚀 Enabling services..."
sudo systemctl daemon-reload
sudo systemctl enable hisab-bot.service
sudo systemctl enable hisab-web.service

echo "✅ LocalFinance installation complete!"
echo "📱 Start Telegram bot: sudo systemctl start hisab-bot"
echo "🌐 Start Web dashboard: sudo systemctl start hisab-web"
echo "🖥️ Web interface: http://localhost:5000"