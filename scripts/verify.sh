#!/usr/bin/env bash
# Verify service health with retries.
# Usage: ./scripts/verify.sh [host]
# Default host: localhost (for local Docker)

set -euo pipefail

HOST="${1:-localhost}"
RETRIES=3
DELAY=5
FAILED=0

SERVICES="hermes:3000:/health thesaurus:8001:/health sophia:8002:/health logos:8003:/health iris:3001:/api/health"

echo "Verifying services on $HOST..."

for svc_info in $SERVICES; do
  IFS=':' read -r name port path <<< "$svc_info"
  SUCCESS=false

  for attempt in $(seq 1 $RETRIES); do
    code=$(curl -sf -o /dev/null -w '%{http_code}' --connect-timeout 3 \
      "http://${HOST}:${port}${path}" 2>/dev/null || echo "000")

    if [ "$code" = "200" ]; then
      echo "  ✅ $name (port $port) — healthy"
      SUCCESS=true
      break
    fi

    if [ "$attempt" -lt "$RETRIES" ]; then
      sleep "$DELAY"
    fi
  done

  if [ "$SUCCESS" = false ]; then
    echo "  ❌ $name (port $port) — FAILED (HTTP $code after $RETRIES attempts)"
    FAILED=$((FAILED + 1))
  fi
done

if [ "$FAILED" -gt 0 ]; then
  echo ""
  echo "FAIL: $FAILED service(s) unhealthy"
  exit 1
fi

echo ""
echo "All services healthy."
