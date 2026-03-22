#!/bin/bash
#
# LocalFinance Development Startup Script
# Starts all 4 microservices for local development
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🏛️ LocalFinance Microservices Development Startup${NC}"
echo "=================================================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go 1.21 or later.${NC}"
    exit 1
fi

# Check if we're in the right directory
if [[ ! -f "go.mod" ]]; then
    echo -e "${RED}❌ Please run this script from the LocalFinance root directory.${NC}"
    exit 1
fi

# Set up environment variables
export ENV=development
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=localfinance
export REDIS_HOST=localhost
export REDIS_PORT=6379
export OLLAMA_HOST=http://localhost:11434
export MODEL_NAME=llama3.2:1b
export STORAGE_ENDPOINT=localhost:9000
export STORAGE_ACCESS_KEY=minioadmin
export STORAGE_SECRET_KEY=minioadmin
export STORAGE_BUCKET=documents

# Service URLs
export THESAURUS_URL=http://localhost:8001
export SOPHIA_URL=http://localhost:8002
export LOGOS_URL=http://localhost:8003

echo -e "${YELLOW}📋 Prerequisites Check:${NC}"
echo "  • Go: $(go version | cut -d' ' -f3)"
echo "  • PostgreSQL should be running on localhost:5432"
echo "  • Redis should be running on localhost:6379"
echo "  • Ollama should be running on localhost:11434"
echo "  • MinIO should be running on localhost:9000"
echo ""

# Function to start a service in the background
start_service() {
    local service_name=$1
    local service_path=$2
    local port=$3
    
    echo -e "${BLUE}🚀 Starting ${service_name}...${NC}"
    
    cd "services/${service_path}"
    
    # Build the service
    if ! go build -o "../../bin/${service_name}" .; then
        echo -e "${RED}❌ Failed to build ${service_name}${NC}"
        cd ../..
        return 1
    fi
    
    # Start the service
    cd ../..
    ./bin/${service_name} > "logs/${service_name}.log" 2>&1 &
    local pid=$!
    echo $pid > "logs/${service_name}.pid"
    
    echo -e "${GREEN}✅ ${service_name} started (PID: ${pid}) - http://localhost:${port}${NC}"
    
    # Wait a moment for startup
    sleep 2
}

# Create necessary directories
mkdir -p bin logs

echo -e "${YELLOW}🏗️ Building and starting services...${NC}"
echo ""

# Start services in order (dependencies first)
start_service "thesaurus" "thesaurus" "8001"
start_service "sophia" "sophia" "8002"
start_service "logos" "logos" "8003"
start_service "hermes" "hermes" "3000"

echo ""
echo -e "${GREEN}🎉 All services started successfully!${NC}"
echo ""
echo -e "${YELLOW}📡 Service URLs:${NC}"
echo "  🎭 Hermes (Frontend/Gateway): http://localhost:3000"
echo "  🏛️ Thesaurus (Database):      http://localhost:8001"
echo "  🦉 Sophia (AI):               http://localhost:8002"
echo "  📜 Logos (Processing):        http://localhost:8003"
echo ""
echo -e "${YELLOW}📊 Health Checks:${NC}"
echo "  curl http://localhost:3000/health"
echo "  curl http://localhost:8001/health"
echo "  curl http://localhost:8002/health"
echo "  curl http://localhost:8003/health"
echo ""
echo -e "${YELLOW}📋 Logs:${NC}"
echo "  tail -f logs/hermes.log"
echo "  tail -f logs/thesaurus.log"
echo "  tail -f logs/sophia.log"
echo "  tail -f logs/logos.log"
echo ""
echo -e "${BLUE}🛑 To stop all services: ./scripts/stop-dev.sh${NC}"