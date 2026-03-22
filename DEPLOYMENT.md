# 🚀 LocalFinance Deployment Guide

This guide covers deploying LocalFinance with the enhanced UI to your Jetson device.

## 📋 Prerequisites

- ✅ Jetson device running Ubuntu 22.04
- ✅ SSH access to Jetson (user: `sagar`, password: `jetson`)
- ✅ Python 3, pip, and required packages installed on Jetson
- ✅ LocalFinance codebase downloaded

## 🛠️ Deployment Scripts

### 1. 📤 **Deploy Enhanced UI**
```bash
./scripts/deploy.sh
```
Deploys the enhanced LocalFinance with:
- Left sidebar navigation
- Upload Statement tab with drag & drop
- Real-time processing feedback
- Settings/Configuration tab
- Modern responsive design

### 2. 📊 **Check Status**
```bash
./scripts/status.sh
```
Comprehensive status check showing:
- Server health and version
- Available API endpoints
- Frontend features detected
- Access information

### 3. 🔄 **Rollback to Previous Version**
```bash
./scripts/rollback.sh
```
Rolls back to the previous working version if deployment fails.

## 🔧 Manual Deployment Steps

If the automated scripts have issues, you can deploy manually:

### Step 1: Upload Enhanced File
```bash
scp /tmp/enhanced_localfinance.py sagar@10.0.0.16:/tmp/
```

### Step 2: Deploy on Jetson
```bash
ssh sagar@10.0.0.16
cd /home/sagar/localfinance
sudo pkill -f python3  # Stop existing server
cp /tmp/enhanced_localfinance.py production_localfinance.py
chmod +x production_localfinance.py
nohup python3 production_localfinance.py > logs/enhanced.log 2>&1 &
```

### Step 3: Verify Deployment
```bash
curl http://localhost:8080/health
```

## 🌐 Access Your LocalFinance

Once deployed, access your enhanced LocalFinance at:
**http://10.0.0.16:8080**

### 🔐 First Time Setup
1. **Register Account:** Create your first user account
2. **Login:** Access your dashboard
3. **Upload Statements:** Use the Upload Statement tab
4. **Configure Settings:** Adjust preferences in Settings tab

## 🎯 Enhanced Features

### 📱 **Left Sidebar Navigation**
- 📊 **Dashboard:** Financial overview and recent transactions
- 📤 **Upload Statement:** Drag & drop file upload with real-time processing
- 💰 **Transactions:** View and manage all transactions
- 📄 **Documents:** Manage uploaded financial documents
- ⚙️ **Settings:** User preferences and configuration

### 🔄 **Real-Time Processing**
When you upload a financial statement:
1. **File Upload** (0-25%): "Starting analysis..."
2. **Format Detection** (25-50%): "Analyzing file format..."
3. **Data Extraction** (50-75%): "Extracting transactions..."
4. **Categorization** (75-100%): "Categorizing transactions..."
5. **Complete**: "Successfully processed X transactions"

### ⚙️ **Settings & Configuration**
- **Currency:** USD, EUR, GBP, CAD
- **Date Format:** MM/DD/YYYY, DD/MM/YYYY, YYYY-MM-DD
- **Default Category:** Set preferred transaction category
- **Auto-Categorization:** Enable/disable smart categorization
- **Notifications:** Configure user notifications

## 🔍 Troubleshooting

### Server Not Responding
1. Check if server is running: `./scripts/status.sh`
2. Restart server: `./scripts/deploy.sh`
3. Check logs on Jetson: `tail -f /home/sagar/localfinance/logs/enhanced.log`

### Basic UI Instead of Enhanced UI
1. Verify enhanced file was deployed: `ssh sagar@10.0.0.16 'ls -la /home/sagar/localfinance/production_localfinance.py'`
2. Check file size (should be ~77KB for enhanced version)
3. Restart server: `./scripts/deploy.sh`

### SSH Authentication Issues
If scripts fail with password issues:
1. Verify password: `ssh sagar@10.0.0.16` (password: `jetson`)
2. Check if `expect` is installed: `which expect`
3. Use manual deployment steps above

## 📊 Version Information

- **Basic Version:** ~16KB, basic authentication and upload
- **Enhanced Version:** ~77KB, full UI with sidebar navigation
- **Version Check:** Use `./scripts/status.sh` to see current version and features

## 🔗 Quick Access Links

- 🌐 **LocalFinance:** http://10.0.0.16:8080
- 📊 **Health Check:** http://10.0.0.16:8080/health
- 🔐 **Login API:** http://10.0.0.16:8080/api/auth/login
- 📤 **Upload API:** http://10.0.0.16:8080/api/v1/upload

---

**🎯 Your privacy-first personal finance system is ready!**