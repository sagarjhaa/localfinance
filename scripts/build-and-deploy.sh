#!/bin/bash

# LocalFinance Go Microservices Build and Deploy Script
# Builds correlation ID-enabled services and deploys to Jetson

set -e

# Configuration
JETSON_HOST="10.0.0.16"
JETSON_USER="sagar"
JETSON_PASS="jetson"
REMOTE_DIR="/home/sagar/localfinance"

# Service configuration (using simple variables)
SERVICES="hermes thesaurus sophia logos"
HERMES_PORT="3000"
THESAURUS_PORT="8001" 
SOPHIA_PORT="8002"
LOGOS_PORT="8003"

get_service_port() {
    case "$1" in
        hermes) echo "3000" ;;
        thesaurus) echo "8001" ;;
        sophia) echo "8002" ;;
        logos) echo "8003" ;;
        *) echo "8080" ;;
    esac
}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
highlight() { echo -e "${CYAN}$1${NC}"; }

print_header() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║             🔨 LocalFinance Microservices Builder           ║"
    echo "║                                                              ║"
    echo "║  Building Go services with correlation ID support           ║"
    echo "║  Target: Jetson ARM64 (10.0.0.16)                          ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_local_go() {
    log "🔍 Checking for local Go installation..."
    
    if command -v go >/dev/null 2>&1; then
        GO_VERSION=$(go version)
        info "✅ Go found: $GO_VERSION"
        export HAS_GO=true
    else
        warn "❌ Go not found locally - will build on Jetson"
        export HAS_GO=false
    fi
}

prepare_source() {
    log "📦 Preparing source code for deployment..."
    
    # Create deployment package
    BUILD_DIR="/tmp/localfinance-build-$(date +%s)"
    mkdir -p "$BUILD_DIR"
    
    # Copy source files
    cp -r services/ shared/ "$BUILD_DIR/"
    
    # Create go.mod files if they don't exist
    for service in $SERVICES; do
        if [ ! -f "$BUILD_DIR/services/$service/go.mod" ]; then
            cat > "$BUILD_DIR/services/$service/go.mod" << EOF
module github.com/sagarjhaa/localfinance/services/$service

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/google/uuid v1.3.0
    modernc.org/sqlite v1.27.0
)
EOF
        fi
    done
    
    # Create shared go.mod
    if [ ! -f "$BUILD_DIR/shared/go.mod" ]; then
        cat > "$BUILD_DIR/shared/go.mod" << EOF
module github.com/sagarjhaa/localfinance/shared

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/google/uuid v1.3.0
)
EOF
    fi
    
    # Create deployment scripts
    cat > "$BUILD_DIR/build.sh" << 'EOFBUILD'
#!/bin/bash
set -e

echo "🔨 Building LocalFinance microservices on Jetson..."

# Install Go if not present
if ! command -v go >/dev/null 2>&1; then
    echo "📥 Installing Go..."
    wget -q -O /tmp/go.tar.gz https://go.dev/dl/go1.21.6.linux-arm64.tar.gz
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    export PATH=/usr/local/go/bin:$PATH
    echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
fi

export PATH=/usr/local/go/bin:$PATH

# Create bin directory
mkdir -p bin

# Build each service
for service in hermes thesaurus sophia logos; do
    echo "🎯 Building $service..."
    cd "services/$service"
    go mod tidy
    go build -o "../../bin/$service" main.go
    cd ../..
    echo "✅ $service built successfully"
done

echo "📦 All services built!"
ls -la bin/
EOFBUILD

    chmod +x "$BUILD_DIR/build.sh"
    
    # Create systemd service files
    mkdir -p "$BUILD_DIR/systemd"
    
    for service in $SERVICES; do
        port=$(get_service_port "$service")
        cat > "$BUILD_DIR/systemd/localfinance-$service.service" << EOF
[Unit]
Description=LocalFinance $service Service
After=network.target

[Service]
Type=simple
User=sagar
WorkingDirectory=$REMOTE_DIR
ExecStart=$REMOTE_DIR/bin/$service
Environment=PORT=$port
Environment=DATABASE_PATH=$REMOTE_DIR/data/localfinance.db
Environment=LOG_LEVEL=info
Restart=always
RestartSec=10
StandardOutput=append:$REMOTE_DIR/logs/$service.log
StandardError=append:$REMOTE_DIR/logs/$service-error.log

[Install]
WantedBy=multi-user.target
EOF
    done
    
    # Create service management script
    cat > "$BUILD_DIR/manage-services.sh" << 'EOFMANAGE'
#!/bin/bash

SERVICES="hermes thesaurus sophia logos"
ACTION=${1:-status}

case "$ACTION" in
    start)
        echo "🚀 Starting LocalFinance microservices..."
        for service in $SERVICES; do
            echo "Starting localfinance-$service..."
            sudo systemctl start localfinance-$service
        done
        ;;
    stop)
        echo "🛑 Stopping LocalFinance microservices..."
        for service in $SERVICES; do
            echo "Stopping localfinance-$service..."
            sudo systemctl stop localfinance-$service || true
        done
        ;;
    restart)
        echo "🔄 Restarting LocalFinance microservices..."
        $0 stop
        sleep 3
        $0 start
        ;;
    status)
        echo "📊 LocalFinance microservices status:"
        for service in $SERVICES; do
            status=$(systemctl is-active localfinance-$service 2>/dev/null || echo "inactive")
            if [ "$status" = "active" ]; then
                echo "✅ localfinance-$service: $status"
            else
                echo "❌ localfinance-$service: $status"
            fi
        done
        ;;
    logs)
        service=${2:-hermes}
        echo "📋 Logs for localfinance-$service:"
        journalctl -u localfinance-$service -f
        ;;
    install)
        echo "📦 Installing systemd services..."
        sudo cp systemd/*.service /etc/systemd/system/
        sudo systemctl daemon-reload
        for service in $SERVICES; do
            sudo systemctl enable localfinance-$service
        done
        echo "✅ Services installed and enabled"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|install} [service-name]"
        exit 1
        ;;
esac
EOFMANAGE

    chmod +x "$BUILD_DIR/manage-services.sh"
    
    # Create testing script
    cat > "$BUILD_DIR/test-services.sh" << 'EOFTEST'
#!/bin/bash

echo "🧪 Testing LocalFinance microservices..."

BASE_URL="http://localhost"
SERVICES=("hermes:3000" "thesaurus:8001" "sophia:8002" "logos:8003")

for service_port in "${SERVICES[@]}"; do
    service=$(echo $service_port | cut -d: -f1)
    port=$(echo $service_port | cut -d: -f2)
    
    echo "🔍 Testing $service on port $port..."
    
    # Test health endpoint
    response=$(curl -s --connect-timeout 5 "$BASE_URL:$port/health" || echo "ERROR")
    
    if [ "$response" = "ERROR" ]; then
        echo "❌ $service: Not responding"
    else
        echo "✅ $service: Responding"
        # Try to parse correlation ID
        correlation_id=$(echo "$response" | grep -o '"correlation_id":"[^"]*"' || echo "")
        if [ -n "$correlation_id" ]; then
            echo "   🔗 Correlation ID: $(echo "$correlation_id" | cut -d'"' -f4)"
        fi
    fi
done

echo ""
echo "🌐 Testing gateway (Hermes)..."
curl -s "$BASE_URL:3000/health" | head -3

echo ""
echo "🎯 Microservices testing complete!"
EOFTEST

    chmod +x "$BUILD_DIR/test-services.sh"
    
    echo "$BUILD_DIR"
}

upload_and_build() {
    local BUILD_DIR="$1"
    log "📤 Uploading source code to Jetson..."
    
    # Create tarball
    tar -czf "/tmp/localfinance-microservices.tar.gz" -C "$(dirname "$BUILD_DIR")" "$(basename "$BUILD_DIR")"
    
    # Upload to Jetson
    expect << EOF
set timeout 60
spawn scp /tmp/localfinance-microservices.tar.gz ${JETSON_USER}@${JETSON_HOST}:/tmp/
expect "password:"
send "${JETSON_PASS}\r"
expect eof
EOF

    log "🔨 Building microservices on Jetson..."
    
    expect << EOF
set timeout 300
spawn ssh ${JETSON_USER}@${JETSON_HOST}
expect "password:"
send "${JETSON_PASS}\r"
expect "$ "

# Extract source
send "cd /tmp && tar -xzf localfinance-microservices.tar.gz\r"
expect "$ "

# Find the build directory
send "BUILD_DIR=\$(ls -d localfinance-build-* | head -1)\r"
expect "$ "

send "cd \$BUILD_DIR\r"
expect "$ "

# Run build script
send "./build.sh\r"
expect {
    "All services built!" {
        expect "$ "
    }
    timeout {
        send "\003\r"
        exit 1
    }
}

# Copy to localfinance directory
send "cp -r bin systemd *.sh ${REMOTE_DIR}/\r"
expect "$ "

send "cd ${REMOTE_DIR}\r"
expect "$ "

send "exit\r"
expect eof
EOF

    if [ $? -eq 0 ]; then
        log "✅ Build and copy successful"
    else
        error "❌ Build failed"
        return 1
    fi
}

stop_existing_services() {
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
send "sudo pkill -f python3 || echo 'No Python services to stop'\r"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect "$ "
    }
    "$ " {}
}

# Stop Go services if they exist
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

    log "✅ Existing services stopped"
}

install_services() {
    log "📦 Installing microservices as systemd services..."
    
    expect << EOF
set timeout 60
spawn ssh ${JETSON_USER}@${JETSON_HOST}
expect "password:"
send "${JETSON_PASS}\r"
expect "$ "

send "cd ${REMOTE_DIR}\r"
expect "$ "

# Install systemd services
send "./manage-services.sh install\r"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect "$ "
    }
    "Services installed" {
        expect "$ "
    }
}

send "exit\r"
expect eof
EOF

    log "✅ Services installed"
}

start_services() {
    log "🚀 Starting microservices..."
    
    expect << EOF
set timeout 60
spawn ssh ${JETSON_USER}@${JETSON_HOST}
expect "password:"
send "${JETSON_PASS}\r"
expect "$ "

send "cd ${REMOTE_DIR}\r"
expect "$ "

# Start services
send "./manage-services.sh start\r"
expect {
    "password:" {
        send "${JETSON_PASS}\r"
        expect "Starting LocalFinance microservices"
    }
    "Starting LocalFinance microservices" {
        expect "$ "
    }
}

send "sleep 10\r"
expect "$ "

# Check status
send "./manage-services.sh status\r"
expect "$ "

send "exit\r"
expect eof
EOF

    log "✅ Services started"
}

test_deployment() {
    log "🧪 Testing microservices deployment..."
    
    sleep 5
    
    # Test each service
    for service in $SERVICES; do
        port=$(get_service_port "$service")
        info "Testing $service on port $port..."
        
        response=$(curl -s --connect-timeout 10 "http://${JETSON_HOST}:${port}/health" 2>/dev/null || echo "ERROR")
        
        if [ "$response" = "ERROR" ]; then
            warn "❌ $service: Not responding"
        else
            highlight "✅ $service: Responding"
            # Try to extract correlation ID
            if echo "$response" | grep -q "correlation_id"; then
                correlation_id=$(echo "$response" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    print(data.get('correlation_id', 'none'))
except:
    print('parse_error')
" 2>/dev/null)
                if [ "$correlation_id" != "none" ] && [ "$correlation_id" != "parse_error" ]; then
                    echo "   🔗 Correlation ID: $correlation_id"
                fi
            fi
        fi
    done
    
    # Test gateway specifically
    echo ""
    info "🌐 Testing main gateway (Hermes)..."
    gateway_response=$(curl -s "http://${JETSON_HOST}:3000/health" 2>/dev/null)
    if [ -n "$gateway_response" ]; then
        echo "$gateway_response" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    print(f'Service: {data.get(\"service\", \"unknown\")}')
    print(f'Version: {data.get(\"version\", \"unknown\")}')
    print(f'Correlation ID: {data.get(\"correlation_id\", \"none\")}')
except:
    print('Gateway response (raw):')
    print('$gateway_response')
"
    fi
}

show_summary() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║               🎉 MICROSERVICES DEPLOYED                      ║${NC}"
    echo -e "${GREEN}╠══════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  🏗️  LocalFinance Go Microservices Architecture             ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  🎭 Hermes (Gateway):     http://${JETSON_HOST}:3000        ║${NC}"
    echo -e "${GREEN}║  🏛️  Thesaurus (Database): http://${JETSON_HOST}:8001        ║${NC}"
    echo -e "${GREEN}║  🦉 Sophia (AI):         http://${JETSON_HOST}:8002        ║${NC}"
    echo -e "${GREEN}║  📜 Logos (Processing):   http://${JETSON_HOST}:8003        ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  ✨ Features:                                                ║${NC}"
    echo -e "${GREEN}║    🔗 Correlation ID threading across all services         ║${NC}"
    echo -e "${GREEN}║    📊 Structured logging with request tracing              ║${NC}"
    echo -e "${GREEN}║    🔐 JWT authentication and sessions                      ║${NC}"
    echo -e "${GREEN}║    📤 Document upload and processing                       ║${NC}"
    echo -e "${GREEN}║    🗄️  SQLite database with transactions                    ║${NC}"
    echo -e "${GREEN}║    ⚙️  Systemd service management                          ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  🛠️  Management:                                            ║${NC}"
    echo -e "${GREEN}║    sudo systemctl status localfinance-hermes               ║${NC}"
    echo -e "${GREEN}║    ./manage-services.sh {start|stop|restart|status}        ║${NC}"
    echo -e "${GREEN}║    ./test-services.sh                                      ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}║  📋 Logs: journalctl -u localfinance-hermes -f             ║${NC}"
    echo -e "${GREEN}║                                                              ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

main() {
    print_header
    
    check_local_go
    
    BUILD_DIR=$(prepare_source)
    info "📁 Source prepared in: $BUILD_DIR"
    
    upload_and_build "$BUILD_DIR"
    stop_existing_services
    install_services
    start_services
    test_deployment
    
    # Cleanup
    rm -rf "$BUILD_DIR"
    rm -f "/tmp/localfinance-microservices.tar.gz"
    
    show_summary
    log "🎯 Microservices deployment completed successfully!"
}

# Handle interruption
trap 'error "🛑 Deployment interrupted"; exit 1' INT TERM

# Check dependencies
for cmd in expect curl; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        error "❌ Required command not found: $cmd"
        exit 1
    fi
done

# Run main function
main "$@"