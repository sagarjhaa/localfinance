#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Build the binary (also runs webui-build)
make build

# Layout the .app
APP="dist/LocalFinance.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

cp dist/localfinance "$APP/Contents/MacOS/LocalFinance"
chmod +x "$APP/Contents/MacOS/LocalFinance"

# Optional icon — skip if not present
if [ -f installer/AppIcon.icns ]; then
  cp installer/AppIcon.icns "$APP/Contents/Resources/AppIcon.icns"
fi

# Info.plist
cat > "$APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleIdentifier</key><string>com.localfinance.app</string>
  <key>CFBundleName</key><string>LocalFinance</string>
  <key>CFBundleVersion</key><string>0.2.0</string>
  <key>CFBundleShortVersionString</key><string>0.2.0</string>
  <key>CFBundleExecutable</key><string>LocalFinance</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>LSUIElement</key><false/>
</dict>
</plist>
PLIST

echo "  -> $APP"
echo "  Run: open $APP"
echo "  Logs: ~/Library/Application Support/LocalFinance/logs/"
