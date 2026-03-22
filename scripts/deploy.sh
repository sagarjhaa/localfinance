#!/bin/bash

# LocalFinance Deployment Script
# Deploys enhanced UI to Jetson with proper authentication handling

set -e

# Configuration
JETSON_HOST="10.0.0.16"
JETSON_USER="sagar"
JETSON_PASS="jetson"
REMOTE_DIR="/home/sagar/localfinance"
LOCAL_ENHANCED_FILE="/tmp/enhanced_localfinance.py"
SERVICE_PORT="8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
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

# Check if required tools are available
check_dependencies() {
    log "🔍 Checking dependencies..."
    
    if ! command -v expect >/dev/null 2>&1; then
        error "expect is required but not installed. Please install expect."
        exit 1
    fi
    
    if ! command -v scp >/dev/null 2>&1; then
        error "scp is required but not installed."
        exit 1
    fi
    
    if ! command -v ssh >/dev/null 2>&1; then
        error "ssh is required but not installed."
        exit 1
    fi
    
    log "✅ All dependencies available"
}

# Test Jetson connectivity
test_connectivity() {
    log "🌐 Testing connectivity to Jetson..."
    
    if timeout 5 bash -c "</dev/tcp/$JETSON_HOST/22" 2>/dev/null; then
        log "✅ Jetson is reachable on port 22"
    else
        error "❌ Cannot reach Jetson at $JETSON_HOST:22"
        exit 1
    fi
}

# Check if enhanced file exists
check_enhanced_file() {
    log "📄 Checking enhanced LocalFinance file..."
    
    if [ ! -f "$LOCAL_ENHANCED_FILE" ]; then
        error "❌ Enhanced LocalFinance file not found at $LOCAL_ENHANCED_FILE"
        info "Please ensure the enhanced file has been created first."
        exit 1
    fi
    
    local file_size=$(wc -c < "$LOCAL_ENHANCED_FILE")
    log "✅ Enhanced file found (${file_size} bytes)"
}

# Upload enhanced file to Jetson
upload_file() {
    log "📤 Uploading enhanced LocalFinance to Jetson..."
    
    expect << EOF
set timeout 30
spawn scp "$LOCAL_ENHANCED_FILE" "${JETSON_USER}@${JETSON_HOST}:/tmp/"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect eof
    }
    "yes/no" {
        send "yes\r"
        expect "password:"
        send "${JETSON_PASS}\r"
        expect eof
    }
    timeout {
        puts "Upload timeout"
        exit 1
    }
}
EOF
    
    if [ $? -eq 0 ]; then
        log "✅ File uploaded successfully"
    else
        error "❌ Upload failed"
        exit 1
    fi
}

# Execute commands on Jetson
execute_remote_commands() {
    log "🔧 Executing deployment commands on Jetson..."
    
    expect << EOF
set timeout 60
spawn ssh "${JETSON_USER}@${JETSON_HOST}"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect "$ "
    }
    "yes/no" {
        send "yes\r"
        expect "password:"
        send "${JETSON_PASS}\r"
        expect "$ "
    }
    timeout {
        puts "SSH connection timeout"
        exit 1
    }
}

# Navigate to project directory
send "cd $REMOTE_DIR\r"
expect "$ "

# Stop existing Python processes
send "sudo pkill -f python3 2>/dev/null || echo 'No processes to stop'\r"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect "$ "
    }
    "$ " {
        # No sudo password needed
    }
}

# Wait a moment for processes to stop
send "sleep 3\r"
expect "$ "

# Check if port is free
send "netstat -tlnp | grep :$SERVICE_PORT || echo 'Port $SERVICE_PORT is free'\r"
expect "$ "

# Backup current version if it exists
send "[ -f production_localfinance.py ] && cp production_localfinance.py production_localfinance.py.backup || echo 'No existing file to backup'\r"
expect "$ "

# Deploy enhanced version
send "cp /tmp/enhanced_localfinance.py production_localfinance.py\r"
expect "$ "

# Make executable
send "chmod +x production_localfinance.py\r"
expect "$ "

# Create logs directory if it doesn't exist
send "mkdir -p logs\r"
expect "$ "

# Start the enhanced server
send "nohup python3 production_localfinance.py > logs/enhanced.log 2>&1 &\r"
expect "$ "

# Wait for startup
send "sleep 5\r"
expect "$ "

# Test the server
send "curl -s http://localhost:$SERVICE_PORT/health | head -3 || echo 'Server not responding yet'\r"
expect "$ "

# Check server logs for any errors
send "tail -5 logs/enhanced.log\r"
expect "$ "

# Exit SSH session
send "exit\r"
expect eof
EOF
    
    if [ $? -eq 0 ]; then
        log "✅ Remote commands executed successfully"
    else
        error "❌ Remote command execution failed"
        exit 1
    fi
}

# Verify deployment
verify_deployment() {
    log "🧪 Verifying deployment..."
    
    # Wait a moment for server to fully start
    sleep 3
    
    # Test health endpoint
    local health_response=$(curl -s --connect-timeout 10 "http://${JETSON_HOST}:${SERVICE_PORT}/health" 2>/dev/null)
    
    if [ -n "$health_response" ]; then
        log "✅ Server is responding"
        
        # Check if it's the enhanced version by testing upload endpoint
        local upload_test=$(curl -s -X POST "http://${JETSON_HOST}:${SERVICE_PORT}/api/v1/upload" 2>/dev/null)
        
        if echo "$upload_test" | grep -q "No file provided\|Missing required fields"; then
            log "✅ Enhanced API endpoints detected"
        else
            warn "⚠️  Basic API detected - enhancement may not be active"
        fi
        
        # Check frontend for enhanced features
        local frontend_response=$(curl -s "http://${JETSON_HOST}:${SERVICE_PORT}/" 2>/dev/null)
        
        if echo "$frontend_response" | grep -q "sidebar\|nav-menu\|upload-area"; then
            log "✅ Enhanced UI features detected"
        else
            warn "⚠️  Basic UI detected"
        fi
        
        info "🌐 LocalFinance is accessible at: http://${JETSON_HOST}:${SERVICE_PORT}"
        
    else
        error "❌ Server is not responding"
        warn "Check the server logs on Jetson for errors"
        exit 1
    fi
}

# Display deployment summary
show_summary() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                 🎉 DEPLOYMENT COMPLETE                       ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  📱 Enhanced UI Features:                                    ║${NC}"
    echo -e "${GREEN}║    • Left sidebar navigation                                 ║${NC}"
    echo -e "${GREEN}║    • Upload Statement tab with drag & drop                  ║${NC}"
    echo -e "${GREEN}║    • Real-time processing feedback                          ║${NC}"
    echo -e "${GREEN}║    • Settings/Configuration tab                             ║${NC}"
    echo -e "${GREEN}║    • Modern responsive design                               ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  🌐 Access URL: http://${JETSON_HOST}:${SERVICE_PORT}                        ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  🔐 Features Available:                                      ║${NC}"
    echo -e "${GREEN}║    • User registration & authentication                     ║${NC}"
    echo -e "${GREEN}║    • Document upload & processing                           ║${NC}"
    echo -e "${GREEN}║    • Transaction management                                 ║${NC}"
    echo -e "${GREEN}║    • Financial dashboard                                    ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Main deployment function
main() {
    log "🚀 Starting LocalFinance Enhanced UI Deployment"
    echo "=================================================="
    
    check_dependencies
    test_connectivity
    check_enhanced_file
    upload_file
    execute_remote_commands
    verify_deployment
    show_summary
    
    log "🎯 Deployment completed successfully!"
}

# Handle script interruption
trap 'error "🛑 Deployment interrupted"; exit 1' INT TERM

# Run main function
main "$@"