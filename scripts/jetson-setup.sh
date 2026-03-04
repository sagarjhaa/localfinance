#!/bin/bash
# LocalFinance Jetson Setup Script
# Run this on a fresh Jetson Orin Nano with JetPack OS

set -e

echo "🚀 LocalFinance Jetson Setup"
echo "============================"

# Step 1: System update
echo ""
echo "📦 Step 1: Updating system..."
sudo apt update && sudo apt upgrade -y

# Step 2: Install Ollama
echo ""
echo "🦙 Step 2: Installing Ollama..."
curl -fsSL https://ollama.com/install.sh | sh

# Step 3: Start Ollama service
echo ""
echo "🔧 Step 3: Starting Ollama service..."
sudo systemctl enable ollama
sudo systemctl start ollama
sleep 5

# Step 4: Check if model file exists
MODEL_FILE="$HOME/localfinance-v1-q4_k_m.gguf"
if [ ! -f "$MODEL_FILE" ]; then
    echo ""
    echo "⚠️  Model file not found at $MODEL_FILE"
    echo "   Please copy the model first:"
    echo "   scp /path/to/localfinance-v1-q4_k_m.gguf jetson:~/"
    exit 1
fi

# Step 5: Create Modelfile
echo ""
echo "📝 Step 5: Creating Modelfile..."
cat > "$HOME/Modelfile" << 'EOF'
FROM ./localfinance-v1-q4_k_m.gguf

TEMPLATE """{{ if .System }}<|im_start|>system
{{ .System }}<|im_end|>
{{ end }}{{ if .Prompt }}<|im_start|>user
{{ .Prompt }}<|im_end|>
{{ end }}<|im_start|>assistant
{{ .Response }}<|im_end|>
"""

PARAMETER stop "<|im_end|>"
PARAMETER stop "<|endoftext|>"
PARAMETER temperature 0.1
PARAMETER num_ctx 2048
EOF

# Step 6: Create Ollama model
echo ""
echo "🔨 Step 6: Creating Ollama model..."
cd "$HOME"
ollama create localfinance -f Modelfile

# Step 7: Test the model
echo ""
echo "🧪 Step 7: Testing model..."
echo "Query: How much did I spend on groceries?"
time ollama run localfinance "How much did I spend on groceries in February?"

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Clone the LocalFinance repo"
echo "  2. Install Python dependencies"
echo "  3. Configure and start Telegram bot"
echo ""
echo "To test manually:"
echo "  ollama run localfinance \"Your question here\""
