# LocalFinance - Privacy-First Personal Finance Assistant

## Project Overview

LocalFinance is a complete privacy-first personal finance management system designed for local deployment on NVIDIA Jetson devices. The system provides AI-powered financial insights while ensuring all data processing remains local, with no cloud dependencies.

## Core Architecture

### Components
- **Telegram Bot**: Natural language interface for financial queries
- **Web Dashboard**: Modern drag-and-drop interface for document uploads
- **Document Processor**: Automated parsing of bank statements (CSV/PDF)
- **AI Integration**: Local Ollama inference with llama3.2:1b model
- **Database**: SQLite for local transaction storage

### Technology Stack
```
Backend: Python 3.8+, Flask, SQLite
AI: Ollama (llama3.2:1b) - Local inference
Bot: python-telegram-bot framework
Processing: PDFplumber, Pandas, Watchdog
Services: SystemD integration for production
```

## Repository Structure

```
localfinance/
├── dashboard/                   # Web Interface
│   ├── app.py                  # Main Flask application
│   ├── enhanced_web_upload.py  # Modern drag-drop interface
│   └── README.md               # Dashboard documentation
├── production/                 # Deployment & Services
│   ├── scripts/
│   │   ├── installer.sh        # Automated installer
│   │   └── manage.sh           # Service management
│   └── services/               # SystemD service files
├── src/                        # Original Framework
│   ├── ai/                     # AI inference modules
│   ├── bot/                    # Bot framework
│   └── core/                   # Database and config
├── enhanced_hisab_bot.py       # Working Telegram bot
├── document_processor.py       # File monitoring system
└── simple_hisab_bot.py         # Lightweight bot version
```

## Development Workflow

### Mac Development Environment
- **Location**: `/Users/sagarjha/projects/localfinance/`
- **Purpose**: Code development, Git operations, documentation
- **Git**: All commits and pushes from Mac

### Jetson Production Environment
- **Location**: `sagar@10.0.0.16:~/localfinance/`
- **Purpose**: Live deployment, hardware testing, production validation
- **Services**: Running bot, web dashboard, AI inference

## Key Features

### 🤖 Telegram Bot (enhanced_hisab_bot.py)
- Natural language financial queries using local AI
- Commands: `/transactions`, `/summary`, `/status`
- Real-time database integration
- Privacy-first: All processing local via Ollama

### 🌐 Web Dashboard (dashboard/)
- Modern drag-and-drop file upload
- Real-time processing statistics
- Mobile responsive design
- Transaction history and analytics

### 📄 Document Processing
- Support for CSV, PDF, Excel bank statements
- Smart automatic categorization
- Real-time file monitoring with Watchdog
- Robust error handling and validation

### 🚀 Production Deployment
- One-command Jetson installation
- SystemD service integration
- Auto-start/restart capabilities
- Service management scripts

## Privacy & Security

### Local-Only Architecture
- **No Cloud Dependencies**: All AI inference via local Ollama
- **No External APIs**: Complete offline operation
- **No Data Transmission**: Financial data never leaves device
- **Encrypted Storage**: SQLite with proper file permissions

### Security Measures
- Strong .gitignore prevents sensitive data commits
- Sandboxed services with limited permissions
- User isolation (services run as non-root)
- File permission restrictions

## Database Schema

### Transactions
```sql
CREATE TABLE transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date TEXT NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    category TEXT,
    account TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Documents
```sql
CREATE TABLE documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    file_type TEXT NOT NULL,
    status TEXT NOT NULL,
    processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    transaction_count INTEGER DEFAULT 0
);
```

## Common Tasks

### Local Development
```bash
# Start web dashboard
cd dashboard && python3 app.py

# Run Telegram bot
python3 enhanced_hisab_bot.py

# Process documents
python3 document_processor.py
```

### Production Deployment
```bash
# Install on fresh Jetson
sudo production/scripts/installer.sh BOT_TOKEN

# Manage services
production/scripts/manage.sh start|stop|status|restart|logs
```

### Debugging
```bash
# Check service status
systemctl status hisab-bot hisab-web

# View logs
journalctl -u hisab-bot -f

# Test Ollama
curl http://127.0.0.1:11434/api/tags

# Database queries
sqlite3 data/finances.db "SELECT * FROM transactions LIMIT 5;"
```

## Configuration Files

### Bot Configuration (config.json)
```json
{
    "bot_token": "YOUR_TELEGRAM_BOT_TOKEN",
    "model_name": "llama3.2:1b",
    "ollama_host": "http://127.0.0.1:11434",
    "db_path": "data/finances.db"
}
```

### Dependencies (requirements.txt)
```
python-telegram-bot==20.7
flask==3.1.3
pdfplumber==0.10.3
pandas==2.1.4
watchdog==4.0.0
requests==2.31.0
```

## AI Integration

### Local Model Setup
- **Model**: llama3.2:1b (1.3GB, optimized for Jetson)
- **Inference**: Ollama API at http://127.0.0.1:11434
- **Privacy**: Completely local, no external model calls
- **Context**: Financial domain knowledge and transaction analysis

### Bot Intelligence
- Natural language processing for financial queries
- Smart transaction categorization
- Context-aware responses using real data
- Financial insights and spending analysis

## Security Guidelines

### Data Protection
- **Never commit real financial data** to repository
- Use .gitignore patterns for sensitive files
- Test with sample data only
- Verify clean commits before pushing

### File Patterns to Ignore
```gitignore
data/
inbox/**/*.pdf
inbox/**/*.csv
*.db
personal_*
private_*
sensitive_*
```

## Deployment Scenarios

### Development Testing
- Local Python execution
- Sample data testing
- Feature development

### Production Deployment
- Automated Jetson installation
- SystemD service management
- Production monitoring

### OS Integration
- Custom OS image baking
- Manufacturing ready
- Plug-and-play deployment

## Future Enhancements
- Multi-bank format support
- Advanced reporting and analytics
- Investment tracking capabilities
- Budget planning and alerts
- Multi-currency support
- Data export capabilities

---

**Status**: Production-ready system for privacy-first local finance management  
**Repository**: https://github.com/sagarjhaa/localfinance  
**Last Updated**: March 21, 2026