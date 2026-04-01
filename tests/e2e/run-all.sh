#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FAILED=0

echo "========================================"
echo " LocalFinance E2E Integration Tests"
echo "========================================"
echo ""

echo "Checking local stack..."
if ! curl -sf http://localhost:8001/health > /dev/null 2>&1; then
  echo "ERROR: Local dev stack not running."
  echo "Run 'make dev-up' first."
  exit 1
fi
echo "Stack healthy."
echo ""

for suite in auth upload categories chat; do
  echo "----------------------------------------"
  if bash "$SCRIPT_DIR/test-${suite}.sh"; then
    echo ""
  else
    FAILED=$((FAILED + 1))
    echo ""
  fi
done

echo "========================================"
if [ "$FAILED" -gt 0 ]; then
  echo " FAILED: $FAILED suite(s) had failures"
  exit 1
else
  echo " ALL SUITES PASSED"
fi
echo "========================================"
