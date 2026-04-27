#!/usr/bin/env bash
# installer/build.sh
# Produce dist/LocalFinance.app — a self-contained macOS bundle that runs
# the LocalFinance stack (Postgres + 4 Go services + Iris) on double-click.
#
# Usage:
#   bash installer/build.sh                # native arch only (fast)
#   UNIVERSAL2=1 bash installer/build.sh   # also build amd64 + lipo fuse
#
# The script is idempotent — re-run after edits and it will only rebuild what changed.
set -euo pipefail

REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"
DIST="${REPO_ROOT}/dist"
APP_DIR="${DIST}/LocalFinance.app"
CONTENTS="${APP_DIR}/Contents"
MACOS_DIR="${CONTENTS}/MacOS"
RES_DIR="${CONTENTS}/Resources"
BIN_DIR="${RES_DIR}/bin"
STATIC_DIR="${RES_DIR}/iris-static"
CACHE_DIR="${DIST}/cache"
VERSION="0.1.0"

UNIVERSAL2="${UNIVERSAL2:-0}"
GO_LDFLAGS=( -ldflags="-linkmode=external" )

log() { printf "\033[1;36m[installer]\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m[warn]\033[0m %s\n" "$*"; }

mkdir -p "${MACOS_DIR}" "${BIN_DIR}" "${STATIC_DIR}" "${CACHE_DIR}"

# ─── 1. Build Go services ──────────────────────────────────────────────
log "Building Go services (native)…"
( cd "${REPO_ROOT}" && make build >/dev/null )
for svc in thesaurus sophia logos hermes; do
  src="${DIST}/${svc}"
  if [[ ! -f "${src}" ]]; then
    echo "ERROR: ${src} not produced by 'make build'" >&2
    exit 1
  fi
  cp -f "${src}" "${BIN_DIR}/${svc}"
done

# Optional universal2 fusion.
if [[ "${UNIVERSAL2}" == "1" ]]; then
  if ! command -v lipo >/dev/null 2>&1; then
    warn "lipo not found — skipping universal2, shipping native arch only"
  else
    log "Building amd64 services for universal2 fusion…"
    AMD_DIR="${CACHE_DIR}/amd64"
    mkdir -p "${AMD_DIR}"
    for svc in thesaurus sophia logos hermes; do
      ( cd "${REPO_ROOT}/services/${svc}" && \
        GOOS=darwin GOARCH=amd64 go build "${GO_LDFLAGS[@]}" -o "${AMD_DIR}/${svc}" . )
    done
    log "Lipo-fusing services…"
    for svc in thesaurus sophia logos hermes; do
      lipo -create "${BIN_DIR}/${svc}" "${AMD_DIR}/${svc}" -output "${BIN_DIR}/${svc}.universal" \
        && mv "${BIN_DIR}/${svc}.universal" "${BIN_DIR}/${svc}"
    done
  fi
fi

# ─── 2. Build Iris (React + Node bundle) ──────────────────────────────
log "Building Iris (React client + Node server bundle)…"
( cd "${REPO_ROOT}" && make build-iris >/dev/null )
cp -f "${DIST}/iris-server.js" "${BIN_DIR}/iris-server.js"

CLIENT_BUILD="${REPO_ROOT}/services/iris/client/build"
if [[ ! -d "${CLIENT_BUILD}" ]]; then
  echo "ERROR: Iris client build dir missing: ${CLIENT_BUILD}" >&2
  exit 1
fi
log "Copying Iris static assets…"
rm -rf "${STATIC_DIR}"
mkdir -p "${STATIC_DIR}"
( cd "${CLIENT_BUILD}" && tar cf - . ) | ( cd "${STATIC_DIR}" && tar xf - )

# ─── 3. Build the launcher ────────────────────────────────────────────
log "Building launcher…"
( cd "${REPO_ROOT}/installer/launcher" && \
  go build "${GO_LDFLAGS[@]}" -o "${MACOS_DIR}/launcher" . )
chmod +x "${MACOS_DIR}/launcher"

if [[ "${UNIVERSAL2}" == "1" ]] && command -v lipo >/dev/null 2>&1; then
  log "Building amd64 launcher and lipo-fusing…"
  ( cd "${REPO_ROOT}/installer/launcher" && \
    GOOS=darwin GOARCH=amd64 go build "${GO_LDFLAGS[@]}" -o "${CACHE_DIR}/launcher.amd64" . )
  lipo -create "${MACOS_DIR}/launcher" "${CACHE_DIR}/launcher.amd64" -output "${MACOS_DIR}/launcher.universal" \
    && mv "${MACOS_DIR}/launcher.universal" "${MACOS_DIR}/launcher"
fi

# ─── 4. Bundle Node ───────────────────────────────────────────────────
# We copy the developer's local node binary into the bundle. End users running
# the .app don't need Node installed. (For a fully reproducible build, swap
# this for a node-darwin-{arch}.tar.gz download — see installer/POSTGRES.md
# notes for the same pattern.)
log "Bundling Node binary…"
NODE_BIN="$(command -v node || true)"
if [[ -z "${NODE_BIN}" ]]; then
  warn "No node found on PATH — bundle will fall back to system PATH at runtime"
else
  cp -f "${NODE_BIN}" "${BIN_DIR}/node"
  chmod +x "${BIN_DIR}/node"
fi

# ─── 5. Postgres binaries ─────────────────────────────────────────────
log "Acquiring Postgres binaries…"
bash "${REPO_ROOT}/installer/postgres-fetch.sh" "${BIN_DIR}/postgres" || \
  warn "Postgres fetch fell back to system mode — see installer/POSTGRES.md"

# ─── 6. Info.plist ────────────────────────────────────────────────────
log "Writing Info.plist…"
cat > "${CONTENTS}/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>             <string>LocalFinance</string>
  <key>CFBundleDisplayName</key>      <string>LocalFinance</string>
  <key>CFBundleIdentifier</key>       <string>com.localfinance.app</string>
  <key>CFBundleVersion</key>          <string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundlePackageType</key>      <string>APPL</string>
  <key>CFBundleExecutable</key>       <string>launcher</string>
  <key>LSMinimumSystemVersion</key>   <string>11.0</string>
  <key>LSUIElement</key>              <false/>
  <key>NSHighResolutionCapable</key>  <true/>
</dict>
</plist>
EOF
# Mirror it under Resources/ for tools that look there.
cp -f "${CONTENTS}/Info.plist" "${RES_DIR}/Info.plist"

log "Bundle ready: ${APP_DIR}"
du -sh "${APP_DIR}" | awk '{ printf "  size: %s\n", $1 }'
