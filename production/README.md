# LocalFinance Production Deployment

## 🚀 Complete Production Setup

Scripts and configuration for deploying LocalFinance to Jetson devices and production environments.

### Quick Installation
```bash
# Download and run installer
curl -sSL https://raw.githubusercontent.com/sagarjhaa/localfinance/main/production/scripts/installer.sh | bash -s YOUR_BOT_TOKEN

# Or manual installation
sudo ./scripts/installer.sh YOUR_BOT_TOKEN
```

### Production Scripts

#### scripts/installer.sh
- **Complete automated installer** for fresh Jetson devices
- **Dependencies**: Ollama, Python packages, SystemD services
- **Database setup**: Creates SQLite schema
- **Service configuration**: Auto-start on boot
- **Usage**: `./installer.sh <telegram_bot_token>`

#### scripts/os-integration.sh  
- **OS image integration** for manufacturing
- **Bakes LocalFinance** into custom OS distributions
- **Pre-configures** all dependencies
- **Ready-to-ship** Jetson devices

#### scripts/quick-install.sh
- **One-liner installer** for development
- **Minimal dependencies** for testing
- **Fast deployment** for demos

### Production Services

#### services/
- **hisab-bot.service**: SystemD service for Telegram bot
- **hisab-web.service**: SystemD service for web dashboard
- **Auto-restart** on failure
- **Logging** to systemd journal
- **User isolation** for security

### Configuration

#### config/
- **Production configuration** templates
- **Security settings** for production
- **Environment-specific** overrides

### Documentation

#### docs/
- **Deployment guides** for different environments
- **Security hardening** instructions
- **Troubleshooting** guides
- **OS integration** documentation

### Deployment Environments

#### Jetson Orin Nano
- **Primary target platform**
- **Optimized for ARM64 architecture**
- **GPU acceleration** for AI inference
- **Low power consumption**

#### Raspberry Pi 4/5
- **Alternative ARM platform**
- **CPU-only inference**
- **Budget-friendly option**

#### x86_64 Linux
- **Development and testing**
- **Desktop deployments**
- **Server installations**

### Security Features
- **Sandboxed services** with limited permissions
- **Local-only processing** - no cloud dependencies
- **Encrypted database** storage
- **User isolation** via SystemD
- **Firewall configuration** included

### Monitoring
- **SystemD logging** for all services
- **Health checks** via service status
- **Performance monitoring** scripts
- **Alert configuration** for failures

Built for mass deployment and manufacturing integration.