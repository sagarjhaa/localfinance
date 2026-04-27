#!/usr/bin/env bash
# Build an unsigned, compressed .dmg for friend-and-family distribution.
#   dist/LocalFinance-<version>.dmg
# Drag-to-Applications layout. No code signing, no notarization.
set -euo pipefail

cd "$(dirname "$0")/.."

DIST="$(pwd)/dist"
APP="${DIST}/LocalFinance.app"

# 1. Make sure the .app exists and is fresh.
make installer

[ -d "$APP" ] || { echo >&2 "Expected ${APP} after make installer"; exit 1; }

# 2. Resolve version from Info.plist (PlistBuddy is on every macOS).
PLIST_BUDDY="/usr/libexec/PlistBuddy"
if [ -x "$PLIST_BUDDY" ]; then
  VERSION="$("$PLIST_BUDDY" -c "Print :CFBundleVersion" "${APP}/Contents/Info.plist")"
else
  # Fallback if PlistBuddy is unavailable for any reason.
  VERSION="0.2.0"
fi

VOL_NAME="LocalFinance ${VERSION}"
DMG_PATH="${DIST}/LocalFinance-${VERSION}.dmg"

# 3. Stage the contents of the .dmg in a temp dir.
STAGE="$(mktemp -d -t localfinance-dmg)"
trap 'rm -rf "$STAGE"' EXIT

cp -R "$APP" "${STAGE}/LocalFinance.app"
ln -s /Applications "${STAGE}/Applications"

# TODO(v2): add a background image + AppleScript-driven window layout
# (icon size, positions) via osascript so the volume opens with a
# polished drag-target view. Skipped for v1 — drag-to-Applications still
# works without it.

# 4. Replace any previous .dmg and build the compressed image.
rm -f "$DMG_PATH"
hdiutil create \
  -volname "$VOL_NAME" \
  -srcfolder "$STAGE" \
  -ov \
  -format UDZO \
  "$DMG_PATH" >/dev/null

echo "  -> ${DMG_PATH} ($(du -h "$DMG_PATH" | cut -f1))"
echo "  Open: open \"$DMG_PATH\""
