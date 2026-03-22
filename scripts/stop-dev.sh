#!/bin/bash
#
# LocalFinance Development Stop Script
# Stops all running microservices
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🛑 Stopping LocalFinance Microservices${NC}"
echo "====================================="

# Function to stop a service
stop_service() {
    local service_name=$1
    local pid_file="logs/${service_name}.pid"
    
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            echo -e "${YELLOW}🛑 Stopping ${service_name} (PID: ${pid})...${NC}"
            kill "$pid"
            sleep 2
            
            # Force kill if still running
            if ps -p "$pid" > /dev/null 2>&1; then
                echo -e "${YELLOW}💀 Force killing ${service_name}...${NC}"
                kill -9 "$pid" 2>/dev/null || true
            fi
            
            echo -e "${GREEN}✅ ${service_name} stopped${NC}"
        else
            echo -e "${YELLOW}⚠️ ${service_name} was not running${NC}"
        fi
        rm -f "$pid_file"
    else
        echo -e "${YELLOW}⚠️ No PID file found for ${service_name}${NC}"
    fi
}

# Stop all services
stop_service "hermes"
stop_service "logos" 
stop_service "sophia"
stop_service "thesaurus"

# Clean up any remaining processes
echo ""
echo -e "${YELLOW}🧹 Cleaning up any remaining processes...${NC}"
pkill -f "bin/hermes" 2>/dev/null || true
pkill -f "bin/logos" 2>/dev/null || true
pkill -f "bin/sophia" 2>/dev/null || true
pkill -f "bin/thesaurus" 2>/dev/null || true

echo ""
echo -e "${GREEN}✅ All LocalFinance services stopped${NC}"

# Optional: Clean up build artifacts
read -p "🗑️ Remove build artifacts (bin/ directory)? [y/N]: " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf bin/
    echo -e "${GREEN}✅ Build artifacts removed${NC}"
fi