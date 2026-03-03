# WiFi Provisioning

LocalFinance uses **Balena wifi-connect** for first-boot WiFi setup.

## How It Works

1. Device boots with no WiFi configured
2. Creates hotspot: `LocalFinance-Setup`
3. User connects phone/laptop to hotspot
4. Captive portal opens automatically
5. User selects their WiFi and enters password
6. Device connects and hotspot disappears
7. LocalFinance bot starts!

## Components

- `wifi-connect` - Balena's WiFi provisioning binary
- `install.sh` - Downloads and installs wifi-connect
- `localfinance.service` - Systemd service for the bot
- `wifi-provision.service` - Systemd service for WiFi check on boot

## Installation (on Raspberry Pi)

```bash
# Run as root
sudo ./install.sh
```

This will:
1. Install wifi-connect
2. Set up systemd services
3. Configure boot sequence

## Manual Testing

```bash
# Force provisioning mode (creates hotspot)
sudo wifi-connect --portal-ssid "LocalFinance-Setup"

# Check if WiFi is connected
nmcli -t -f STATE general
```
