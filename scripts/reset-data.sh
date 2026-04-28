#!/usr/bin/env bash
# reset-data.sh — wipe ingested data so you can re-import from scratch.
#
# Usage:
#   scripts/reset-data.sh           # wipe transactions, documents, accounts,
#                                   # statement_periods, conversations, chat_messages,
#                                   # dismissed_insights. Keeps users + preferences.
#   scripts/reset-data.sh --hard    # also drops users + preferences (full nuke).
#   scripts/reset-data.sh --uploads # also remove /tmp/localfinance/uploads/*
#
# Connects to the embedded Postgres the .app uses. Reads the port from
# the postmaster.pid file under the data dir. Safe to run while the .app
# is up — Postgres handles concurrent writers, our app just sees an
# empty result set on the next read.
set -euo pipefail

DATA_DIR="${LOCALFINANCE_DATA_DIR:-$HOME/Library/Application Support/LocalFinance}"
PIDFILE="$DATA_DIR/postgres/postmaster.pid"

HARD=0
WIPE_UPLOADS=0
for arg in "$@"; do
  case "$arg" in
    --hard) HARD=1 ;;
    --uploads) WIPE_UPLOADS=1 ;;
    -h|--help)
      sed -n '2,12p' "$0"; exit 0 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

if [ ! -f "$PIDFILE" ]; then
  echo "Embedded Postgres pidfile not found at $PIDFILE" >&2
  echo "Is LocalFinance running? Open the app first." >&2
  exit 1
fi

# Line 4 of postmaster.pid is the TCP port.
PORT=$(awk 'NR==4' "$PIDFILE")
if ! [[ "$PORT" =~ ^[0-9]+$ ]]; then
  echo "Could not read postgres port from $PIDFILE" >&2
  exit 1
fi

# Hardcoded credentials — set in main.go for the embedded path.
export PGPASSWORD=postgres
PSQL=(psql -h 127.0.0.1 -p "$PORT" -U postgres -d localfinance -v ON_ERROR_STOP=1)

if ! command -v psql >/dev/null 2>&1; then
  echo "psql not found. Install with: brew install libpq && brew link --force libpq" >&2
  exit 1
fi

echo "▸ Resetting data on embedded Postgres at port $PORT"

# Tables to wipe in soft mode. Order respects FK dependencies via
# CASCADE — TRUNCATE ... CASCADE handles it for us.
SOFT_TABLES=(
  transactions
  documents
  statement_periods
  accounts
  conversations
  chat_messages
  dismissed_insights
)

# Hard mode adds users + preferences. Doing this means you'll need to
# register again on next launch.
HARD_TABLES=(
  users
  user_preferences
  user_sessions
  budgets
  category_rules
)

if [ "$HARD" -eq 1 ]; then
  TABLES=("${SOFT_TABLES[@]}" "${HARD_TABLES[@]}")
else
  TABLES=("${SOFT_TABLES[@]}")
fi

# Build comma-joined list for one-shot TRUNCATE. RESTART IDENTITY resets
# any sequences (we mostly use UUIDs, but harmless).
JOINED=$(IFS=,; echo "${TABLES[*]}")
echo "  truncating: $JOINED"
"${PSQL[@]}" -c "TRUNCATE TABLE $JOINED RESTART IDENTITY CASCADE;" >/dev/null
echo "✓ tables truncated"

if [ "$WIPE_UPLOADS" -eq 1 ]; then
  if [ -d /tmp/localfinance/uploads ]; then
    rm -rf /tmp/localfinance/uploads/*
    echo "✓ /tmp/localfinance/uploads cleared"
  fi
fi

# parse-debug is just artifacts; harmless to nuke too.
DEBUG_DIR="$DATA_DIR/parse-debug"
if [ -d "$DEBUG_DIR" ]; then
  rm -f "$DEBUG_DIR"/*.txt "$DEBUG_DIR"/*.md 2>/dev/null || true
  echo "✓ parse-debug cleared"
fi

echo "Done. Re-upload statements to repopulate."
