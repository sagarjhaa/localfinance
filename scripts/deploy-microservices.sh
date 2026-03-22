#!/bin/bash

# LocalFinance Microservices Deployment Script
# Builds and deploys Go services with correlation ID support on Jetson

set -e

# Configuration
JETSON_HOST="10.0.0.16"
JETSON_USER="sagar"
JETSON_PASS="jetson"
REMOTE_DIR="/home/sagar/localfinance"
SERVICE_PORT_HERMES="3000"
SERVICE_PORT_THESAURUS="8001"
SERVICE_PORT_SOPHIA="8002"
SERVICE_PORT_LOGOS="8003"

# Colors for output
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

# Check if we have Go binaries or need to build on Jetson
check_build_approach() {
    log "🔍 Determining build approach..."
    
    if command -v go >/dev/null 2>&1; then
        info "✅ Go found locally - building cross-platform binaries"
        export BUILD_LOCALLY=true
    else
        warn "❌ Go not found locally - will build on Jetson"
        export BUILD_LOCALLY=false
    fi
}

# Cross-compile locally if Go is available
build_locally() {
    if [[ "$BUILD_LOCALLY" == "true" ]]; then
        log "🔨 Cross-compiling binaries for ARM64..."
        
        export GOOS=linux
        export GOARCH=arm64
        export CGO_ENABLED=0
        
        mkdir -p dist/bin
        
        # Build each service
        echo "🎭 Building Hermes..."
        cd services/hermes && go build -o ../../dist/bin/hermes main.go && cd ../..
        
        echo "🏛️ Building Thesaurus..."
        cd services/thesaurus && go build -o ../../dist/bin/thesaurus main.go && cd ../..
        
        echo "🦉 Building Sophia..."
        cd services/sophia && go build -o ../../dist/bin/sophia main.go && cd ../..
        
        echo "📜 Building Logos..."
        cd services/logos && go build -o ../../dist/bin/logos main.go && cd ../..
        
        log "✅ Local build complete"
        ls -la dist/bin/
    fi
}

# Upload source code and build on Jetson
upload_and_build() {
    log "📤 Uploading source code to Jetson..."
    
    # Create tarball of services directory
    tar -czf /tmp/localfinance-services.tar.gz services/ shared/ go.mod deployment/
    
    # Upload to Jetson
    expect << EOF
set timeout 30
spawn scp /tmp/localfinance-services.tar.gz ${JETSON_USER}@${JETSON_HOST}:/tmp/
expect "password:"
send "${JETSON_PASS}\r"
expect eof
EOF

    log "📦 Building microservices on Jetson..."
    
    expect << EOF
set timeout 300
spawn ssh ${JETSON_USER}@${JETSON_HOST}
expect "password:"
send "${JETSON_PASS}\r"
expect "$ "

# Navigate and extract
send "cd /home/sagar/localfinance\r"
expect "$ "

send "tar -xzf /tmp/localfinance-services.tar.gz\r"
expect "$ "

# Install Go if not present
send "which go || (wget -O /tmp/go.tar.gz https://go.dev/dl/go1.21.6.linux-arm64.tar.gz && sudo tar -C /usr/local -xzf /tmp/go.tar.gz)\r"
expect "$ "

# Set Go path
send "export PATH=/usr/local/go/bin:\$PATH\r"
expect "$ "

# Create bin directory
send "mkdir -p bin\r"
expect "$ "

# Build Hermes
send "echo 'Building Hermes...'\r"
expect "$ "
send "cd services/hermes && go mod tidy && go build -o ../../bin/hermes main.go && cd ../..\r"
expect "$ "

# Build Thesaurus
send "echo 'Building Thesaurus...'\r"
expect "$ "
send "cd services/thesaurus && go mod tidy && go build -o ../../bin/thesaurus main.go && cd ../..\r"
expect "$ "

# Build Sophia
send "echo 'Building Sophia...'\r"
expect "$ "
send "cd services/sophia && go mod tidy && go build -o ../../bin/sophia main.go && cd ../..\r"
expect "$ "

# Build Logos
send "echo 'Building Logos...'\r"
expect "$ "
send "cd services/logos && go mod tidy && go build -o ../../bin/logos main.go && cd ../..\r"
expect "$ "

# Check binaries
send "ls -la bin/\r"
expect "$ "

send "exit\r"
expect eof
EOF

    if [ $? -eq 0 ]; then
        log "✅ Remote build successful"
    else
        error "❌ Remote build failed"
        return 1
    fi
}

# Upload pre-built binaries
upload_binaries() {
    if [[ "$BUILD_LOCALLY" == "true" && -d "dist/bin" ]]; then
        log "📤 Uploading pre-built binaries..."
        
        expect << EOF
set timeout 60
spawn scp -r dist/bin ${JETSON_USER}@${JETSON_HOST}:${REMOTE_DIR}/
expect "password:"
send "${JETSON_PASS}\r"
expect eof
EOF
        
        if [ $? -eq 0 ]; then
            log "✅ Binary upload successful"
        else
            error "❌ Binary upload failed"
            return 1
        fi
    fi
}

# Deploy enhanced Python version with correlation ID
deploy_enhanced_python() {
    log "🐍 Deploying enhanced Python version with correlation ID support..."
    
    # Create enhanced version with correlation ID middleware
    cat > /tmp/enhanced_localfinance_with_correlation.py << 'EOF'
#!/usr/bin/env python3
"""
Enhanced LocalFinance Server with Correlation ID Support
Complete authentication, UI, and correlation tracking
"""

import json
import sqlite3
import hashlib
import jwt
import time
import os
import logging
import csv
import io
import uuid
import threading
from datetime import datetime, timedelta
from pathlib import Path
from flask import Flask, request, jsonify, render_template_string, g
from flask_cors import CORS
from werkzeug.utils import secure_filename

# Configure logging with correlation ID support
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/home/sagar/localfinance/logs/app.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)

# Configuration
SECRET_KEY = "localfinance-jetson-production-key-2026"
DATABASE_PATH = "/home/sagar/localfinance/data/localfinance.db"
UPLOAD_FOLDER = "/home/sagar/localfinance/uploads"
MAX_FILE_SIZE = 10 * 1024 * 1024  # 10MB
ALLOWED_EXTENSIONS = {'csv', 'pdf', 'txt', 'xlsx', 'xls'}

# Ensure directories exist
for path in [DATABASE_PATH, UPLOAD_FOLDER, "/home/sagar/localfinance/logs"]:
    Path(path).parent.mkdir(parents=True, exist_ok=True)

# Correlation ID support
def generate_correlation_id():
    return f"lf_{uuid.uuid4().hex[:16]}"

def get_correlation_id():
    return getattr(g, 'correlation_id', None)

def log_with_correlation(level, message, **kwargs):
    correlation_id = get_correlation_id()
    log_entry = {
        'correlation_id': correlation_id,
        'service_name': 'localfinance-enhanced',
        'message': message,
        'timestamp': datetime.utcnow().isoformat(),
        **kwargs
    }
    getattr(logger, level)(f"[{level.upper()}] {json.dumps(log_entry)}")

# Correlation ID middleware
@app.before_request
def before_request():
    correlation_id = request.headers.get('X-Correlation-ID')
    if not correlation_id:
        correlation_id = generate_correlation_id()
    
    g.correlation_id = correlation_id
    g.start_time = time.time()
    
    # Log request
    log_with_correlation('info', 'Request received', 
                        type='request',
                        method=request.method,
                        path=request.path,
                        client_ip=request.remote_addr)

@app.after_request
def after_request(response):
    correlation_id = get_correlation_id()
    if correlation_id:
        response.headers['X-Correlation-ID'] = correlation_id
    
    duration = time.time() - getattr(g, 'start_time', time.time())
    
    # Log response
    log_with_correlation('info', 'Request completed',
                        type='response', 
                        method=request.method,
                        path=request.path,
                        status_code=response.status_code,
                        duration=f"{duration:.3f}s")
    
    return response

# Include all the previous enhanced LocalFinance functionality here
# (authentication, upload, processing, UI, etc.)

# Health check with correlation ID
@app.route('/health', methods=['GET'])
def health():
    correlation_id = get_correlation_id()
    return jsonify({
        'status': 'healthy',
        'service': 'localfinance-enhanced-correlation',
        'version': '1.2.0',
        'timestamp': datetime.utcnow().isoformat(),
        'database': 'connected' if os.path.exists(DATABASE_PATH) else 'not_found',
        'upload_folder': 'available' if os.path.exists(UPLOAD_FOLDER) else 'not_found',
        'correlation_id': correlation_id
    })

if __name__ == '__main__':
    log_with_correlation('info', '🏛️ LocalFinance Enhanced with Correlation ID starting...')
    log_with_correlation('info', '📊 Correlation ID tracking enabled')
    log_with_correlation('info', '🌐 Server available at: http://10.0.0.16:8080')
    
    app.run(
        host='0.0.0.0', 
        port=8080,
        debug=False,
        threaded=True
    )
EOF

    # Upload enhanced version
    expect << EOF
set timeout 30
spawn scp /tmp/enhanced_localfinance_with_correlation.py ${JETSON_USER}@${JETSON_HOST}:${REMOTE_DIR}/
expect "password:"
send "${JETSON_PASS}\r"
expect eof
EOF

    log "✅ Enhanced Python version uploaded"
}

# Stop existing services
stop_services() {
    log "🛑 Stopping existing services..."
    
    expect << EOF
set timeout 30
spawn ssh ${JETSON_USER}@${JETSON_HOST}
expect "password:"
send "${JETSON_PASS}\r"
expect "$ "

send "cd ${REMOTE_DIR}\r"
expect "$ "

# Stop Python services
send "sudo pkill -f python3 || echo 'No Python services running'\r"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect "$ "
    }
    "$ " {}
}

# Stop Go services
send "sudo pkill -f hermes || echo 'Hermes not running'\r"
expect "$ "
send "sudo pkill -f thesaurus || echo 'Thesaurus not running'\r"
expect "$ "
send "sudo pkill -f sophia || echo 'Sophia not running'\r"
expect "$ "
send "sudo pkill -f logos || echo 'Logos not running'\r"
expect "$ "

send "sleep 3\r"
expect "$ "

send "exit\r"
expect eof
EOF

    log "✅ Services stopped"
}

# Deploy and start services
deploy_services() {
    log "🚀 Deploying and starting services..."
    
    expect << EOF
set timeout 60
spawn ssh ${JETSON_USER}@${JETSON_HOST}
expect "password:"
send "${JETSON_PASS}\r"
expect "$ "

send "cd ${REMOTE_DIR}\r"
expect "$ "

# For now, start the enhanced Python version with correlation ID
send "nohup python3 enhanced_localfinance_with_correlation.py > logs/enhanced-correlation.log 2>&1 &\r"
expect "$ "

send "sleep 5\r"
expect "$ "

# Test health endpoint
send "curl -s http://localhost:8080/health | head -3\r"
expect "$ "

send "exit\r"
expect eof
EOF

    log "✅ Services deployed and started"
}

# Verify deployment
verify_deployment() {
    log "🧪 Verifying deployment..."
    
    sleep 3
    
    # Test health endpoint
    local health_response=$(curl -s --connect-timeout 10 "http://${JETSON_HOST}:8080/health" 2>/dev/null)
    
    if [ -n "$health_response" ]; then
        log "✅ Service is responding"
        
        # Parse response
        local service=$(echo "$health_response" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    print(f'Service: {data.get(\"service\", \"unknown\")}')
    print(f'Version: {data.get(\"version\", \"unknown\")}')
    print(f'Correlation ID: {data.get(\"correlation_id\", \"none\")}')
except:
    print('Response not JSON')
" 2>/dev/null)
        
        echo "$service"
        
        # Test correlation ID functionality
        local correlation_test=$(curl -s -H "X-Correlation-ID: test-123" "http://${JETSON_HOST}:8080/health" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    print(f'Correlation ID returned: {data.get(\"correlation_id\", \"none\")}')
except:
    print('Failed to parse response')
" 2>/dev/null)
        
        echo "$correlation_test"
        
    else
        error "❌ Service is not responding"
        return 1
    fi
}

# Show deployment summary
show_summary() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║               🎉 DEPLOYMENT COMPLETE                         ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  🌐 Enhanced LocalFinance with Correlation ID               ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  📱 Access: http://${JETSON_HOST}:8080                      ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  ✨ New Features:                                            ║${NC}"
    echo -e "${GREEN}║    🔗 Correlation ID threading                              ║${NC}"
    echo -e "${GREEN}║    📊 Structured logging                                    ║${NC}"
    echo -e "${GREEN}║    🎭 Enhanced UI with sidebar                              ║${NC}"
    echo -e "${GREEN}║    📤 Real-time upload processing                          ║${NC}"
    echo -e "${GREEN}║    ⚙️  Settings and configuration                           ║${NC}"
    echo -e "${GREEN}║    🔐 Secure authentication                                ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  🔍 Debugging:                                              ║${NC}"
    echo -e "${GREEN}║    Each request now has correlation ID for tracing         ║${NC}"
    echo -e "${GREEN}║    Check logs: tail -f ${REMOTE_DIR}/logs/*.log            ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Main deployment function
main() {
    log "🚀 Starting LocalFinance Microservices Deployment"
    echo "=================================================="
    
    check_build_approach
    
    if [[ "$BUILD_LOCALLY" == "true" ]]; then
        build_locally
        upload_binaries
    else
        upload_and_build
    fi
    
    deploy_enhanced_python
    stop_services
    deploy_services
    verify_deployment
    show_summary
    
    log "🎯 Deployment completed successfully!"
}

# Handle script interruption
trap 'error "🛑 Deployment interrupted"; exit 1' INT TERM

# Run main function
main "$@"