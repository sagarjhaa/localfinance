#!/bin/bash

# Cleanup script for Jetson to remove any conflicting services

JETSON_IP="10.0.0.16"
JETSON_USER="sagar"
JETSON_PASS="jetson"

echo "🧹 Cleaning up Jetson for fresh deployment..."

expect -c "
    spawn ssh ${JETSON_USER}@${JETSON_IP}
    expect \"password:\"
    send \"${JETSON_PASS}\\r\"
    expect \"$ \"
    
    # Kill any running Python processes on port 5000
    send \"sudo pkill -f 'python.*5000' || true\\r\"
    expect {
        \"password:\" {
            send \"${JETSON_PASS}\\r\"
            expect \"$ \"
        }
        \"$ \" {}
    }
    
    # Kill any processes using port 5000
    send \"sudo fuser -k 5000/tcp || true\\r\"
    expect \"$ \"
    
    # Stop any existing LocalFinance services
    send \"sudo systemctl stop localfinance-* 2>/dev/null || true\\r\"
    expect \"$ \"
    
    # Remove old service files
    send \"sudo rm -f /etc/systemd/system/localfinance-*.service\\r\"
    expect \"$ \"
    
    send \"sudo systemctl daemon-reload\\r\"
    expect \"$ \"
    
    # Clean up old deployment
    send \"rm -rf /home/sagar/localfinance*\\r\"
    expect \"$ \"
    
    # Check what's using ports
    send \"sudo netstat -tulpn | grep ':300[0-9]\\|:500[0-9]\\|:800[0-9]' || echo 'No conflicting ports'\\r\"
    expect \"$ \"
    
    send \"echo 'Cleanup complete'\\r\"
    expect \"$ \"
    
    send \"exit\\r\"
    expect eof
"

echo "✅ Jetson cleanup complete"