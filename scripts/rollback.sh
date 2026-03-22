#!/bin/bash

# LocalFinance Rollback Script
# Rolls back to previous version if deployment fails

set -e

# Configuration
JETSON_HOST="10.0.0.16"
JETSON_USER="sagar"
JETSON_PASS="jetson"
REMOTE_DIR="/home/sagar/localfinance"
SERVICE_PORT="8080"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

rollback_to_previous() {
    log "🔄 Rolling back to previous version..."
    
    expect << EOF
set timeout 30
spawn ssh "${JETSON_USER}@${JETSON_HOST}"
expect "password:"
send "${JETSON_PASS}\r"
expect "$ "

# Navigate to project directory
send "cd $REMOTE_DIR\r"
expect "$ "

# Stop current server
send "sudo pkill -f python3 || echo 'No processes to stop'\r"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect "$ "
    }
    "$ " {}
}

send "sleep 2\r"
expect "$ "

# Check if backup exists
send "[ -f production_localfinance.py.backup ] && echo 'Backup found' || echo 'No backup available'\r"
expect "$ "

# Restore backup
send "[ -f production_localfinance.py.backup ] && cp production_localfinance.py.backup production_localfinance.py || echo 'Using current version'\r"
expect "$ "

# Start server
send "nohup python3 production_localfinance.py > logs/rollback.log 2>&1 &\r"
expect "$ "

# Wait for startup
send "sleep 4\r"
expect "$ "

# Test server
send "curl -s http://localhost:$SERVICE_PORT/health || echo 'Server not responding'\r"
expect "$ "

send "exit\r"
expect eof
EOF

    log "✅ Rollback completed"
}

verify_rollback() {
    log "🧪 Verifying rollback..."
    
    sleep 3
    
    local health_response=$(curl -s --connect-timeout 10 "http://${JETSON_HOST}:${SERVICE_PORT}/health" 2>/dev/null)
    
    if [ -n "$health_response" ]; then
        log "✅ Server is responding after rollback"
        info "🌐 LocalFinance is accessible at: http://${JETSON_HOST}:${SERVICE_PORT}"
    else
        error "❌ Server is not responding after rollback"
        exit 1
    fi
}

main() {
    log "🔄 Starting LocalFinance Rollback"
    echo "=================================="
    
    rollback_to_previous
    verify_rollback
    
    log "🎯 Rollback completed successfully!"
}

main "$@"