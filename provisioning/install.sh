#!/bin/bash
# LocalFinance WiFi Provisioning Installer
# Run on Raspberry Pi: sudo ./install.sh

set -e

echo "🏦 LocalFinance WiFi Provisioning Setup"
echo "========================================"

# Check root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root: sudo ./install.sh"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    aarch64) WIFI_CONNECT_ARCH="linux-aarch64" ;;
    armv7l)  WIFI_CONNECT_ARCH="linux-rpi" ;;
    x86_64)  WIFI_CONNECT_ARCH="linux-x86_64" ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "📍 Architecture: $ARCH → $WIFI_CONNECT_ARCH"

# Install dependencies
echo "📦 Installing dependencies..."
apt-get update
apt-get install -y network-manager dnsmasq

# Disable default networking (use NetworkManager instead)
systemctl disable dhcpcd 2>/dev/null || true
systemctl enable NetworkManager

# Download wifi-connect
WIFI_CONNECT_VERSION="v4.11.84"
WIFI_CONNECT_URL="https://github.com/balena-os/wifi-connect/releases/download/${WIFI_CONNECT_VERSION}/wifi-connect-${WIFI_CONNECT_VERSION}-${WIFI_CONNECT_ARCH}.tar.gz"

echo "📥 Downloading wifi-connect ${WIFI_CONNECT_VERSION}..."
curl -L -o /tmp/wifi-connect.tar.gz "$WIFI_CONNECT_URL"

echo "📂 Installing to /usr/local/bin..."
tar -xzf /tmp/wifi-connect.tar.gz -C /usr/local/bin
chmod +x /usr/local/bin/wifi-connect
rm /tmp/wifi-connect.tar.gz

# Verify installation
if ! /usr/local/bin/wifi-connect --version; then
    echo "❌ wifi-connect installation failed"
    exit 1
fi

# Create provisioning script
echo "📝 Creating provisioning script..."
cat > /usr/local/bin/localfinance-provision << 'EOF'
#!/bin/bash
# LocalFinance WiFi Provisioning Check
# Runs at boot - starts captive portal if no WiFi

SSID="LocalFinance-Setup"
PORTAL_GATEWAY="192.168.42.1"
PORTAL_DHCP_RANGE="192.168.42.2,192.168.42.254"

# Check if already connected to WiFi
if nmcli -t -f STATE general | grep -q "connected"; then
    echo "✅ WiFi connected, skipping provisioning"
    exit 0
fi

echo "📡 No WiFi connection. Starting provisioning portal..."
echo "   Connect to: $SSID"
echo "   Then open: http://$PORTAL_GATEWAY"

# Start wifi-connect (blocks until user configures WiFi)
/usr/local/bin/wifi-connect \
    --portal-ssid "$SSID" \
    --portal-gateway "$PORTAL_GATEWAY" \
    --portal-dhcp-range "$PORTAL_DHCP_RANGE" \
    --portal-passphrase "" \
    --timeout 0

echo "✅ WiFi configured!"
EOF
chmod +x /usr/local/bin/localfinance-provision

# Create systemd service for WiFi provisioning
echo "📝 Creating systemd services..."
cat > /etc/systemd/system/localfinance-wifi.service << 'EOF'
[Unit]
Description=LocalFinance WiFi Provisioning
Wants=NetworkManager.service
After=NetworkManager.service
Before=localfinance.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/localfinance-provision
RemainAfterExit=yes
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

# Create systemd service for LocalFinance bot
cat > /etc/systemd/system/localfinance.service << 'EOF'
[Unit]
Description=LocalFinance AI Assistant
After=localfinance-wifi.service network-online.target
Wants=network-online.target

[Service]
Type=simple
User=pi
WorkingDirectory=/home/pi/localfinance
Environment=BOT_TOKEN=YOUR_TOKEN_HERE
Environment=FINANCE_DB=/home/pi/localfinance/data/finances.db
ExecStart=/home/pi/localfinance/.venv/bin/python -m src.bot.telegram
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# Enable services
systemctl daemon-reload
systemctl enable localfinance-wifi.service
systemctl enable localfinance.service

echo ""
echo "✅ Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Edit /etc/systemd/system/localfinance.service"
echo "     - Set BOT_TOKEN to your Telegram bot token"
echo "     - Adjust paths if not using default /home/pi/localfinance"
echo ""
echo "  2. Reboot to test:"
echo "     sudo reboot"
echo ""
echo "  3. On boot (if no WiFi saved):"
echo "     - Device creates 'LocalFinance-Setup' hotspot"
echo "     - Connect your phone and configure WiFi"
echo "     - Bot starts automatically after connection"
echo ""
