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

# Build deps — Go and Node are required for `make installer` since we don't
# yet ship prebuilt binaries. Once Releases land this whole block goes away.
ensure_dep() {
  local cmd="$1" pkg="$2"
  if command -v "$cmd" >/dev/null 2>&1; then return 0; fi
  if command -v brew >/dev/null 2>&1; then
    say "Installing $pkg via brew"
    brew install "$pkg"
  else
    echo >&2 "Need $cmd to build LocalFinance from source. Install Homebrew (https://brew.sh) or $pkg manually and re-run."
    exit 1
  fi
}
ensure_dep go go
ensure_dep node node

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

# Pick a model based on RAM. If anything in the sweet-spot range is already
# pulled, skip the download — the in-app auto-select will use it.
MEM_GB=$(( $(sysctl -n hw.memsize 2>/dev/null || echo 0) / 1073741824 ))
if   [ "$MEM_GB" -ge 32 ]; then MODEL="qwen2.5:7b"
elif [ "$MEM_GB" -ge 16 ]; then MODEL="gemma3:4b"
else                            MODEL="llama3.2:3b"
fi

# Quick check: any sweet-spot model already installed?
INSTALLED=$(ollama list 2>/dev/null | tail -n +2 | awk '{print $1}')
HAVE_SWEET_SPOT=0
for m in $INSTALLED; do
  case "$m" in
    *:3b|*:4b|*:7b|*:8b|*:9b|*:10b|*:11b|*:12b|*:13b|*:14b|gemma3:*|gemma3) HAVE_SWEET_SPOT=1 ;;
  esac
done

if [ "$HAVE_SWEET_SPOT" = "1" ]; then
  say "Compatible model already installed — skipping pull"
elif echo "$INSTALLED" | grep -Fxq "$MODEL"; then
  say "Recommended model ${MODEL} already installed — skipping pull"
else
  say "Pulling ${MODEL} in background (${MEM_GB} GB RAM detected)"
  nohup ollama pull "$MODEL" >/dev/null 2>&1 &
fi

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
