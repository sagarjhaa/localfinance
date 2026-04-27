#!/usr/bin/env bash
# LocalFinance one-line installer.
# Usage:  curl -fsSL https://raw.githubusercontent.com/sagarjhaa/localfinance/main/scripts/install.sh | sh
set -euo pipefail

REPO_URL="https://github.com/sagarjhaa/localfinance.git"
SRC_DIR="${HOME}/.localfinance/src"

if [ -t 1 ]; then BLUE=$'\033[1;34m'; RESET=$'\033[0m'; else BLUE=""; RESET=""; fi
say() { echo >&2 "${BLUE}==>${RESET} $*"; }

if [ "$(uname -s)" != "Darwin" ]; then
  echo >&2 "LocalFinance currently supports macOS only. Detected: $(uname -s)"; exit 1
fi

# Ollama install
if command -v ollama >/dev/null 2>&1; then
  say "Ollama already installed"
else
  say "Installing Ollama"
  if command -v brew >/dev/null 2>&1; then
    brew install ollama
  else
    curl -fsSL https://ollama.com/install.sh | sh
  fi
fi

# Ensure ollama serve is running
if pgrep -f "ollama serve" >/dev/null 2>&1; then
  say "Ollama already running"
else
  say "Starting ollama serve in background"
  nohup ollama serve >/dev/null 2>&1 &
  sleep 1
fi

# Resolve repo root: local clone if present, else clone into ~/.localfinance/src
# TODO: replace with release-download path once GitHub Releases are published.
if [ -d ".git" ] && [ -f "Makefile" ]; then
  REPO_ROOT="$(pwd)"
  say "Detected local clone at ${REPO_ROOT}"
else
  say "Cloning LocalFinance into ${SRC_DIR}"
  mkdir -p "$(dirname "$SRC_DIR")"
  if [ -d "$SRC_DIR/.git" ]; then
    git -C "$SRC_DIR" pull --ff-only
  else
    git clone "$REPO_URL" "$SRC_DIR"
  fi
  REPO_ROOT="$SRC_DIR"
fi

say "Building LocalFinance.app (this can take a couple of minutes)"
( cd "$REPO_ROOT" && make installer )

APP_SRC="${REPO_ROOT}/dist/LocalFinance.app"
[ -d "$APP_SRC" ] || { echo >&2 "Build failed: ${APP_SRC} not found"; exit 1; }

# Install to /Applications (fall back to ~/Applications)
DEST="/Applications"
if [ ! -w "$DEST" ]; then
  say "/Applications not writable, falling back to ~/Applications"
  DEST="${HOME}/Applications"; mkdir -p "$DEST"
fi
say "Installing LocalFinance.app to ${DEST}"
rm -rf "${DEST}/LocalFinance.app"
if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete "$APP_SRC/" "${DEST}/LocalFinance.app/"
else
  cp -R "$APP_SRC" "${DEST}/LocalFinance.app"
fi

# Pick a model based on RAM, pull in background
MEM_GB=$(( $(sysctl -n hw.memsize 2>/dev/null || echo 0) / 1073741824 ))
if   [ "$MEM_GB" -ge 32 ]; then MODEL="qwen2.5:7b"
elif [ "$MEM_GB" -ge 16 ]; then MODEL="gemma3:4b"
else                            MODEL="llama3.2:3b"
fi
say "Pulling ${MODEL} in background (${MEM_GB} GB RAM detected)"
nohup ollama pull "$MODEL" >/dev/null 2>&1 &

say "Opening ${DEST}/LocalFinance.app"
open "${DEST}/LocalFinance.app"

cat >&2 <<MSG

${BLUE}==>${RESET} LocalFinance is starting up.

  URL:      http://localhost:3001
  Email:    local@localfinance.app
  Password: localfinance

  Model ${MODEL} is downloading in the background; the first-run wizard
  will pick it up automatically.

  Docs: https://github.com/sagarjhaa/localfinance#readme

MSG
