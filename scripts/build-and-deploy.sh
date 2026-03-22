#!/bin/bash

# LocalFinance Build and Deploy Script
# Builds ARM64 binaries locally and deploys to Jetson

set -e

# Configuration
JETSON_IP="10.0.0.16"
JETSON_USER="sagar"
JETSON_PASS="jetson"
DEPLOY_DIR="/home/sagar/localfinance"
PROJECT_ROOT="/Users/sagarjha/projects/localfinance"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🏛️ LocalFinance Build and Deploy Pipeline${NC}"
echo "=============================================="

# Step 1: Clean and prepare
echo -e "${YELLOW}📁 Cleaning previous builds...${NC}"
cd "$PROJECT_ROOT"
rm -rf dist/
mkdir -p dist/{bin,config,scripts,systemd}

# Step 2: Cross-compile for ARM64
echo -e "${YELLOW}🔨 Cross-compiling services for ARM64 Linux...${NC}"

export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=0

echo -e "  🏛️ Building Thesaurus (Database service)..."
go build -ldflags="-s -w" -o dist/bin/thesaurus ./services/thesaurus

echo -e "  🦉 Building Sophia (AI service)..."
go build -ldflags="-s -w" -o dist/bin/sophia ./services/sophia

echo -e "  📜 Building Logos (Document processing)..."
go build -ldflags="-s -w" -o dist/bin/logos ./services/logos

echo -e "  🎭 Building Hermes (Gateway service)..."
go build -ldflags="-s -w" -o dist/bin/hermes ./services/hermes

# Verify binaries
echo -e "${GREEN}✅ Built binaries:${NC}"
ls -la dist/bin/

# Step 3: Create configuration files
echo -e "${YELLOW}⚙️ Creating configuration files...${NC}"

cat > dist/config/environment.env << 'EOF'
# LocalFinance Environment Configuration
ENV=production
LOG_LEVEL=info

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=localuser
DB_PASSWORD=localpass
DB_NAME=localfinance

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379

# Ollama Configuration
OLLAMA_HOST=http://localhost:11434
MODEL_NAME=llama3.2:1b

# Service Ports
THESAURUS_PORT=8001
SOPHIA_PORT=8002
LOGOS_PORT=8003
HERMES_PORT=3000

# JWT Configuration
JWT_SECRET=localfinance-production-secret-change-this
JWT_EXPIRY_DAYS=7

# Storage Configuration
STORAGE_ENDPOINT=localhost:9000
STORAGE_ACCESS_KEY=minioadmin
STORAGE_SECRET_KEY=minioadmin
STORAGE_BUCKET=documents
EOF

# Step 4: Create systemd service files
echo -e "${YELLOW}🔧 Creating systemd service files...${NC}"

create_service_file() {
    local service_name=$1
    local service_port=$2
    local service_description="$3"
    
    cat > "dist/systemd/localfinance-${service_name}.service" << EOF
[Unit]
Description=LocalFinance ${service_description}
After=network.target postgresql.service redis-server.service
Wants=postgresql.service redis-server.service

[Service]
Type=simple
User=sagar
Group=sagar
WorkingDirectory=${DEPLOY_DIR}
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin
EnvironmentFile=${DEPLOY_DIR}/config/environment.env
ExecStart=${DEPLOY_DIR}/bin/${service_name}
Restart=always
RestartSec=10
StandardOutput=append:${DEPLOY_DIR}/logs/${service_name}.log
StandardError=append:${DEPLOY_DIR}/logs/${service_name}_error.log

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=${DEPLOY_DIR}

[Install]
WantedBy=multi-user.target
EOF
}

create_service_file "thesaurus" "8001" "Database Service"
create_service_file "sophia" "8002" "AI Service" 
create_service_file "logos" "8003" "Document Processing Service"
create_service_file "hermes" "3000" "API Gateway Service"

# Step 5: Create deployment scripts
echo -e "${YELLOW}📝 Creating deployment scripts...${NC}"

cat > dist/scripts/setup-database.sh << 'EOF'
#!/bin/bash
# Database setup script

echo "🗄️ Setting up PostgreSQL database..."

# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << SQL
DROP DATABASE IF EXISTS localfinance;
DROP USER IF EXISTS localuser;
CREATE DATABASE localfinance;
CREATE USER localuser WITH PASSWORD 'localpass';
GRANT ALL PRIVILEGES ON DATABASE localfinance TO localuser;
\q
SQL

echo "✅ Database setup complete"
EOF

cat > dist/scripts/install-services.sh << 'EOF'
#!/bin/bash
# Service installation script

set -e

DEPLOY_DIR="/home/sagar/localfinance"

echo "🔧 Installing LocalFinance services..."

# Create directories
mkdir -p $DEPLOY_DIR/{logs,data,uploads}

# Copy systemd service files
sudo cp systemd/*.service /etc/systemd/system/
sudo systemctl daemon-reload

# Set proper permissions
chmod +x bin/*
chmod +x scripts/*.sh

# Install and enable services
services=("localfinance-thesaurus" "localfinance-sophia" "localfinance-logos" "localfinance-hermes")

for service in "${services[@]}"; do
    echo "  📦 Installing $service..."
    sudo systemctl enable $service
done

echo "✅ Services installed successfully"
EOF

cat > dist/scripts/start-services.sh << 'EOF'
#!/bin/bash
# Start all LocalFinance services

services=("localfinance-thesaurus" "localfinance-sophia" "localfinance-logos" "localfinance-hermes")

echo "🚀 Starting LocalFinance services..."

for service in "${services[@]}"; do
    echo "  ▶️ Starting $service..."
    sudo systemctl start $service
    sleep 2
done

echo ""
echo "📊 Service Status:"
for service in "${services[@]}"; do
    status=$(sudo systemctl is-active $service)
    if [ "$status" = "active" ]; then
        echo "  ✅ $service: $status"
    else
        echo "  ❌ $service: $status"
    fi
done

echo ""
echo "🌐 Service URLs:"
echo "  🎭 Hermes (API Gateway): http://10.0.0.16:3000"
echo "  🏛️ Thesaurus (Database): http://10.0.0.16:8001"
echo "  🦉 Sophia (AI): http://10.0.0.16:8002"  
echo "  📜 Logos (Processing): http://10.0.0.16:8003"
EOF

cat > dist/scripts/stop-services.sh << 'EOF'
#!/bin/bash
# Stop all LocalFinance services

services=("localfinance-hermes" "localfinance-logos" "localfinance-sophia" "localfinance-thesaurus")

echo "🛑 Stopping LocalFinance services..."

for service in "${services[@]}"; do
    echo "  ⏹️ Stopping $service..."
    sudo systemctl stop $service || true
done

echo "✅ All services stopped"
EOF

# Make scripts executable
chmod +x dist/scripts/*.sh

# Step 6: Create deployment archive
echo -e "${YELLOW}📦 Creating deployment archive...${NC}"
cd dist/
tar -czf ../localfinance-deploy.tar.gz .
cd ..

echo -e "${GREEN}✅ Build complete! Archive: $(pwd)/localfinance-deploy.tar.gz${NC}"

# Step 7: Deploy to Jetson
echo -e "${YELLOW}🚀 Deploying to Jetson...${NC}"

echo -e "  📤 Uploading deployment archive..."
expect -c "
    spawn scp localfinance-deploy.tar.gz ${JETSON_USER}@${JETSON_IP}:/tmp/
    expect \"password:\"
    send \"${JETSON_PASS}\\r\"
    expect eof
"

echo -e "  📋 Extracting and installing on Jetson..."
expect -c "
    spawn ssh ${JETSON_USER}@${JETSON_IP}
    expect \"password:\"
    send \"${JETSON_PASS}\\r\"
    expect \"$ \"
    
    # Clean previous installation
    send \"sudo systemctl stop localfinance-* 2>/dev/null || true\\r\"
    expect \"$ \"
    
    send \"rm -rf ${DEPLOY_DIR}\\r\"
    expect \"$ \"
    
    send \"mkdir -p ${DEPLOY_DIR}\\r\"
    expect \"$ \"
    
    # Extract deployment
    send \"cd ${DEPLOY_DIR}\\r\"
    expect \"$ \"
    
    send \"tar -xzf /tmp/localfinance-deploy.tar.gz\\r\"
    expect \"$ \"
    
    # Setup database
    send \"chmod +x scripts/*.sh\\r\"
    expect \"$ \"
    
    send \"./scripts/setup-database.sh\\r\"
    expect {
        \"password:\" {
            send \"${JETSON_PASS}\\r\"
            exp_continue
        }
        \"$ \" {
            send \"echo 'Database setup complete'\\r\"
            expect \"$ \"
        }
    }
    
    # Install services
    send \"./scripts/install-services.sh\\r\"
    expect {
        \"password:\" {
            send \"${JETSON_PASS}\\r\"
            exp_continue
        }
        \"$ \" {
            send \"echo 'Services installed'\\r\"
            expect \"$ \"
        }
    }
    
    # Start services
    send \"./scripts/start-services.sh\\r\"
    expect {
        \"password:\" {
            send \"${JETSON_PASS}\\r\"
            exp_continue
        }
        \"$ \" {
            send \"echo 'Services started'\\r\"
            expect \"$ \"
        }
    }
    
    send \"exit\\r\"
    expect eof
"

# Step 8: Verify deployment
echo -e "${YELLOW}🔍 Verifying deployment...${NC}"

sleep 5

echo -e "  🏛️ Testing Thesaurus service..."
curl -s http://${JETSON_IP}:8001/health || echo "❌ Thesaurus not responding"

echo -e "  🦉 Testing Sophia service..."
curl -s http://${JETSON_IP}:8002/health || echo "❌ Sophia not responding"

echo -e "  📜 Testing Logos service..."
curl -s http://${JETSON_IP}:8003/health || echo "❌ Logos not responding"

echo -e "  🎭 Testing Hermes gateway..."
curl -s http://${JETSON_IP}:3000/health || echo "❌ Hermes not responding"

echo ""
echo -e "${GREEN}🎉 Deployment Complete!${NC}"
echo ""
echo -e "${BLUE}📱 Access your LocalFinance application at:${NC}"
echo -e "  ${GREEN}http://${JETSON_IP}:3000${NC}"
echo ""
echo -e "${YELLOW}🔧 Management commands on Jetson:${NC}"
echo -e "  Start:   ./scripts/start-services.sh"
echo -e "  Stop:    ./scripts/stop-services.sh" 
echo -e "  Status:  systemctl status localfinance-*"
echo -e "  Logs:    tail -f logs/*.log"