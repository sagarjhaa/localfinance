#!/usr/bin/env bash
# Shared test helpers for e2e tests

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost}"
IRIS_URL="${BASE_URL}:3001"
THESAURUS_URL="${BASE_URL}:8001"
SOPHIA_URL="${BASE_URL}:8002"
LOGOS_URL="${BASE_URL}:8003"

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

TEST_ID="test_$(date +%s)_$$"
TEST_EMAIL="${TEST_ID}@test.localfinance.dev"
TEST_PASSWORD="TestPass123!"
TEST_FIRST="Test"
TEST_LAST="User"
TOKEN=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

log_pass() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_PASSED=$((TESTS_PASSED + 1))
  echo -e "  ${GREEN}PASS${NC} $1"
}

log_fail() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_FAILED=$((TESTS_FAILED + 1))
  echo -e "  ${RED}FAIL${NC} $1"
  if [ -n "${2:-}" ]; then echo -e "       Expected: $2"; fi
  if [ -n "${3:-}" ]; then echo -e "       Got:      $3"; fi
}

log_skip() {
  echo -e "  ${YELLOW}SKIP${NC} $1"
}

register_user() {
  local resp
  resp=$(curl -sf -X POST "${IRIS_URL}/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${TEST_EMAIL}\",
      \"password\": \"${TEST_PASSWORD}\",
      \"first_name\": \"${TEST_FIRST}\",
      \"last_name\": \"${TEST_LAST}\"
    }" 2>/dev/null) || return 1
  TOKEN=$(echo "$resp" | jq -r '.token // empty')
  if [ -z "$TOKEN" ]; then return 1; fi
}

login_user() {
  local resp
  resp=$(curl -sf -X POST "${IRIS_URL}/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${TEST_EMAIL}\",
      \"password\": \"${TEST_PASSWORD}\"
    }" 2>/dev/null) || return 1
  TOKEN=$(echo "$resp" | jq -r '.token // empty')
  if [ -z "$TOKEN" ]; then return 1; fi
}

assert_eq() {
  local description="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then log_pass "$description"
  else log_fail "$description" "$expected" "$actual"; fi
}

assert_contains() {
  local description="$1" haystack="$2" needle="$3"
  if echo "$haystack" | grep -q "$needle"; then log_pass "$description"
  else log_fail "$description" "contains '$needle'" "$(echo "$haystack" | head -c 200)"; fi
}

assert_status() {
  local description="$1" expected_code="$2" actual_code="$3"
  assert_eq "$description (HTTP $expected_code)" "$expected_code" "$actual_code"
}

assert_not_empty() {
  local description="$1" value="$2"
  if [ -n "$value" ]; then log_pass "$description"
  else log_fail "$description" "non-empty" "(empty)"; fi
}

retry_until() {
  local description="$1" max_attempts="${2:-10}" delay="${3:-3}"
  shift 3
  local cmd="$@"
  for attempt in $(seq 1 "$max_attempts"); do
    if eval "$cmd" 2>/dev/null; then return 0; fi
    sleep "$delay"
  done
  return 1
}

print_summary() {
  local suite_name="${1:-Tests}"
  echo ""
  echo "--- ${suite_name} Summary ---"
  echo "  Total:  $TESTS_RUN"
  echo -e "  Passed: ${GREEN}${TESTS_PASSED}${NC}"
  if [ "$TESTS_FAILED" -gt 0 ]; then
    echo -e "  Failed: ${RED}${TESTS_FAILED}${NC}"
    exit 1
  else
    echo "  Failed: $TESTS_FAILED"
  fi
}
