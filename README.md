# 🏛️ LocalFinance

**Your money. Your data. Your device.**

A privacy-first personal finance system with enhanced UI and AI assistant that runs entirely on your hardware. No cloud. No subscriptions. No data leaving your device.

---

## 🚀 Quick Start (Enhanced UI)

```bash
# Clone the repository
git clone https://github.com/sagarjhaa/localfinance.git
cd localfinance

# Quick deployment to Jetson
./quick-deploy.sh

# Or use full deployment script
./scripts/deploy.sh

# Check status
./scripts/status.sh
```

**That's it!** Access your enhanced LocalFinance at: **http://10.0.0.16:8080**

---

## 🎨 Enhanced UI Features

### 📱 **Left Sidebar Navigation**
- **📊 Dashboard:** Financial overview with income, expenses, and recent transactions
- **📤 Upload Statement:** Drag & drop file upload with real-time processing feedback
- **💰 Transactions:** View and manage all financial transactions with categories
- **📄 Documents:** Manage uploaded financial documents and processing status
- **⚙️ Settings:** User preferences, currency, date format, auto-categorization

### 🔄 **Real-Time Processing**
When you upload bank statements:
```
Upload → "Starting analysis..." (25%)
      → "Analyzing file format..." (50%) 
      → "Extracting transactions..." (75%)
      → "Categorizing transactions..." (100%)
      → "Successfully processed X transactions"
```

### 🎯 **Smart Features**
- **🤖 Auto-Categorization:** Food, Transport, Shopping, Utilities, Income, etc.
- **💱 Multi-Currency Support:** USD, EUR, GBP, CAD
- **📅 Flexible Date Formats:** MM/DD/YYYY, DD/MM/YYYY, YYYY-MM-DD
- **📱 Responsive Design:** Works on desktop, tablet, and mobile
- **🔐 Secure Authentication:** JWT-based sessions with password hashing

---

## 📊 Current Status

| Component | Status | Description |
|-----------|--------|-------------|
| Enhanced Web UI | ✅ Complete | Modern sidebar navigation with real-time processing |
| Authentication System | ✅ Working | Secure login/register with JWT tokens |
| Document Processing | ✅ Working | CSV, PDF, Excel upload with auto-categorization |
| Transaction Management | ✅ Working | View, categorize, and manage financial transactions |
| Settings Configuration | ✅ Working | User preferences and system configuration |
| Deployment Scripts | ✅ Ready | Automated deployment, status check, and rollback |
| AI Model Integration | ⏳ Available | Qwen2-0.5B fine-tuned for SQL generation |
| Telegram Bot | ⏳ Available | Chat interface for AI queries |

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     YOUR DEVICE (JETSON)                     │
│  ┌─────────────────┐  ┌──────────────┐  ┌─────────────────┐ │
│  │   Enhanced UI   │→ │  Web Server  │→ │   Database      │ │
│  │ (React/Modern)  │  │  (Python)    │  │   (SQLite)      │ │
│  └─────────────────┘  └──────────────┘  └─────────────────┘ │
│                                │                             │
│  ┌─────────────────────────────▼──────────────────────────┐ │
│  │              Document Processing Engine                │ │
│  │    • CSV/PDF/Excel parsing                            │ │
│  │    • Smart transaction categorization                  │ │
│  │    • Real-time processing feedback                     │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                 LocalFinance AI                         │ │
│  │    • Qwen2-0.5B fine-tuned model                       │ │
│  │    • Natural language to SQL                           │ │
│  │    • Telegram bot integration                          │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**Privacy:** All processing happens locally. Enhanced UI provides modern web interface while maintaining complete data privacy.

---

## 📁 Repository Structure

```
localfinance/
├── enhanced_localfinance.py   # 🎨 Enhanced UI server (77KB)
├── services/                  # 🏗️ Microservices architecture
│   ├── hermes/               # Frontend & API Gateway
│   ├── thesaurus/            # CRUD & Database operations
│   ├── sophia/               # AI & Natural language processing
│   └── logos/                # Document processing
├── scripts/                  # 🚀 Deployment & management
│   ├── deploy.sh            # Automated deployment
│   ├── status.sh            # Status checking
│   ├── rollback.sh          # Safe rollback
│   └── build-and-deploy.sh  # Cross-compilation build
├── dashboard/                # 📊 Legacy dashboard components
├── production/              # 🏭 Production deployment configs
├── training/                # 🤖 AI model training
│   ├── train.py            # Training script
│   ├── test_model.py       # Model testing
│   └── data/               # Training examples
├── src/                     # 🔧 Core application logic
│   ├── inference.py        # AI model integration
│   ├── bot.py              # Telegram bot
│   └── core/               # Database & configuration
├── docs/                    # 📚 Documentation
├── DEPLOYMENT.md            # 📋 Deployment guide
└── quick-deploy.sh          # ⚡ One-command deployment
```

---

## 💻 Requirements

### **For Enhanced UI:**
- **Device:** Jetson Nano/Xavier or similar ARM64 device
- **OS:** Ubuntu 22.04 LTS
- **RAM:** 4GB minimum (8GB recommended)
- **Storage:** 10GB free space
- **Network:** WiFi/Ethernet for web access

### **For AI Features:**
- **Additional RAM:** +4GB for model loading
- **Python:** 3.10+ with ML libraries
- **Model Size:** ~2GB for Qwen2-0.5B

---

## 🚀 Deployment Options

### **1. Quick Interactive Deployment**
```bash
./quick-deploy.sh
```
Interactive script with status checks and user confirmation.

### **2. Full Automated Deployment**
```bash
./scripts/deploy.sh
```
Comprehensive deployment with error handling and verification.

### **3. Manual Deployment**
See `DEPLOYMENT.md` for step-by-step manual deployment instructions.

### **4. Status Monitoring**
```bash
./scripts/status.sh
```
Check server health, API endpoints, and UI features.

---

## 🎯 Usage Guide

### **First Time Setup:**
1. **Access:** Go to http://10.0.0.16:8080
2. **Register:** Create your account with email/password
3. **Dashboard:** View your financial overview
4. **Upload:** Use Upload Statement tab to process bank files
5. **Configure:** Adjust settings for currency, categories, etc.

### **Document Processing:**
- **Supported Formats:** CSV, PDF, Excel (.xlsx, .xls)
- **File Size Limit:** 10MB per file
- **Processing:** Real-time feedback with progress bar
- **Categorization:** Automatic smart categorization
- **Review:** View results in Transactions and Documents tabs

### **Settings Configuration:**
- **Currency:** USD, EUR, GBP, CAD
- **Date Format:** Multiple international formats
- **Categories:** Customize transaction categories
- **Auto-Categorization:** Enable/disable smart categorization

---

## 🤖 AI Integration

Ask questions via Telegram bot:
- *"How much did I spend on dining last month?"*
- *"Show me all Amazon purchases"*
- *"What are my subscriptions?"*

**Model generates SQL:**
```sql
SELECT SUM(amount) FROM transactions 
WHERE category = 'Dining' 
AND date >= date('now', '-1 month')
```

**Model specs:**
- **Base:** Qwen2-0.5B-Instruct (MIT license)
- **Fine-tuned:** 2,944 finance query examples
- **Inference:** ~20s CPU (optimizable with GGUF)

---

## 📝 Roadmap

- [x] Enhanced web UI with sidebar navigation
- [x] Real-time document processing feedback
- [x] User authentication and session management
- [x] Comprehensive deployment scripts
- [x] Settings and configuration management
- [x] Smart transaction categorization
- [ ] Mobile app (React Native)
- [ ] Advanced reporting and analytics
- [ ] Multi-user support
- [ ] API integrations (bank connections)
- [ ] Advanced AI features (spending insights)
- [ ] GGUF optimization for faster inference

---

## 🔒 Privacy Promise

- **✅ No cloud:** All processing on your device
- **✅ No tracking:** Your data never leaves your hardware
- **✅ No subscriptions:** One-time setup, yours forever
- **✅ Open source:** Full code transparency
- **✅ Local authentication:** Credentials stored locally
- **✅ Encrypted storage:** Secure password hashing

---

## 🛠️ Development & Contribution

### **Run Development Server:**
```bash
python3 enhanced_localfinance.py
```

### **Build Production:**
```bash
./scripts/build-and-deploy.sh
```

### **Testing:**
```bash
./scripts/status.sh
```

---

*Built with privacy in mind. Your finances stay yours. Now with a modern, intuitive interface.*

**🌐 Ready to use at: http://10.0.0.16:8080**