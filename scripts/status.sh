#!/bin/bash

# LocalFinance Status Check Script
# Checks the current status of LocalFinance deployment

# Configuration
JETSON_HOST="10.0.0.16"
SERVICE_PORT="8080"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

feature() {
    echo -e "${PURPLE}[FEATURE]${NC} $1"
}

check_server_health() {
    log "🔍 Checking server health..."
    
    local health_response=$(curl -s --connect-timeout 10 "http://${JETSON_HOST}:${SERVICE_PORT}/health" 2>/dev/null)
    
    if [ -n "$health_response" ]; then
        log "✅ Server is responding"
        
        # Parse health response
        local service=$(echo "$health_response" | python3 -c "import json,sys; data=json.load(sys.stdin); print(data.get('service', 'unknown'))" 2>/dev/null || echo "unknown")
        local status=$(echo "$health_response" | python3 -c "import json,sys; data=json.load(sys.stdin); print(data.get('status', 'unknown'))" 2>/dev/null || echo "unknown")
        local version=$(echo "$health_response" | python3 -c "import json,sys; data=json.load(sys.stdin); print(data.get('version', 'unknown'))" 2>/dev/null || echo "unknown")
        
        info "📊 Service: $service"
        info "🎯 Status: $status"
        info "🏷️  Version: $version"
        
        return 0
    else
        error "❌ Server is not responding"
        return 1
    fi
}

check_api_endpoints() {
    log "🔌 Testing API endpoints..."
    
    # Test upload endpoint
    local upload_response=$(curl -s -X POST "http://${JETSON_HOST}:${SERVICE_PORT}/api/v1/upload" 2>/dev/null)
    
    if echo "$upload_response" | grep -q "No file provided\|Missing required fields"; then
        feature "✅ Enhanced upload API available"
    else
        warn "⚠️  Basic upload API or not available"
    fi
    
    # Test auth endpoint
    local auth_response=$(curl -s -X POST "http://${JETSON_HOST}:${SERVICE_PORT}/api/auth/login" -H "Content-Type: application/json" -d '{}' 2>/dev/null)
    
    if echo "$auth_response" | grep -q "Missing email\|email or password"; then
        feature "✅ Authentication API available"
    else
        warn "⚠️  Authentication API not available"
    fi
}

check_frontend_features() {
    log "🎨 Checking frontend features..."
    
    local frontend_response=$(curl -s "http://${JETSON_HOST}:${SERVICE_PORT}/" 2>/dev/null)
    
    if [ -n "$frontend_response" ]; then
        feature "✅ Frontend is accessible"
        
        # Check for enhanced UI features
        if echo "$frontend_response" | grep -q "sidebar\|nav-menu"; then
            feature "✅ Enhanced UI with sidebar navigation"
        else
            warn "⚠️  Basic UI without sidebar"
        fi
        
        if echo "$frontend_response" | grep -q "upload-area\|drag.*drop"; then
            feature "✅ Enhanced upload interface"
        else
            warn "⚠️  Basic upload interface"
        fi
        
        if echo "$frontend_response" | grep -q "settings\|configuration"; then
            feature "✅ Settings/Configuration available"
        else
            warn "⚠️  Settings not detected"
        fi
        
        if echo "$frontend_response" | grep -q "processing.*status\|real.*time"; then
            feature "✅ Real-time processing feedback"
        else
            warn "⚠️  Basic processing feedback"
        fi
        
    else
        error "❌ Frontend is not accessible"
    fi
}

show_access_info() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                    📱 ACCESS INFORMATION                      ║${NC}"
    echo -e "${BLUE}╠══════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${BLUE}║                                                              ║${NC}"
    echo -e "${BLUE}║  🌐 URL: http://${JETSON_HOST}:${SERVICE_PORT}                           ║${NC}"
    echo -e "${BLUE}║                                                              ║${NC}"
    echo -e "${BLUE}║  📱 Available Features:                                      ║${NC}"
    echo -e "${BLUE}║    • User Registration & Login                               ║${NC}"
    echo -e "${BLUE}║    • Financial Dashboard                                     ║${NC}"
    echo -e "${BLUE}║    • Document Upload & Processing                           ║${NC}"
    echo -e "${BLUE}║    • Transaction Management                                 ║${NC}"
    echo -e "${BLUE}║    • Settings Configuration                                 ║${NC}"
    echo -e "${BLUE}║                                                              ║${NC}"
    echo -e "${BLUE}║  🔧 Navigation:                                              ║${NC}"
    echo -e "${BLUE}║    📊 Dashboard - Financial overview                        ║${NC}"
    echo -e "${BLUE}║    📤 Upload Statement - File processing                    ║${NC}"
    echo -e "${BLUE}║    💰 Transactions - View/manage transactions              ║${NC}"
    echo -e "${BLUE}║    📄 Documents - Manage uploaded files                    ║${NC}"
    echo -e "${BLUE}║    ⚙️  Settings - User preferences                          ║${NC}"
    echo -e "${BLUE}║                                                              ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

main() {
    log "📊 LocalFinance Status Check"
    echo "============================="
    
    if check_server_health; then
        check_api_endpoints
        check_frontend_features
        show_access_info
        log "🎯 Status check completed"
    else
        error "🚨 Service appears to be down"
        info "💡 Try running: ./scripts/deploy.sh"
        exit 1
    fi
}

main "$@"