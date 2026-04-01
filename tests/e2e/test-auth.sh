#!/usr/bin/env bash
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "=== Auth Tests ==="

# Register
echo ""
echo "Register flow:"
resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${TEST_EMAIL}\",
    \"password\": \"${TEST_PASSWORD}\",
    \"first_name\": \"${TEST_FIRST}\",
    \"last_name\": \"${TEST_LAST}\"
  }")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)

assert_status "Register returns 2xx" "$(echo "$code" | grep -oE '2[0-9]{2}')" "$code"
TOKEN=$(echo "$body" | jq -r '.token // empty')
assert_not_empty "Register returns JWT token" "$TOKEN"

# /me with valid token
echo ""
echo "Token validation:"
resp=$(curl -s -w "\n%{http_code}" -X GET "${IRIS_URL}/api/auth/me" \
  -H "Authorization: Bearer ${TOKEN}")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)
assert_status "GET /me returns 200" "200" "$code"
user_email=$(echo "$body" | jq -r '.email // .user.email // empty')
assert_eq "GET /me returns correct email" "$TEST_EMAIL" "$user_email"

# /me without token
resp=$(curl -s -o /dev/null -w "%{http_code}" -X GET "${IRIS_URL}/api/auth/me")
assert_status "GET /me without token returns 401" "401" "$resp"

# /me with bad token
resp=$(curl -s -o /dev/null -w "%{http_code}" -X GET "${IRIS_URL}/api/auth/me" \
  -H "Authorization: Bearer invalidtoken123")
assert_status "GET /me with bad token returns 401" "401" "$resp"

# Login
echo ""
echo "Login flow:"
resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"${TEST_EMAIL}\", \"password\": \"${TEST_PASSWORD}\"}")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)
assert_status "Login returns 200" "200" "$code"
login_token=$(echo "$body" | jq -r '.token // empty')
assert_not_empty "Login returns JWT token" "$login_token"

# Wrong password
resp=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${IRIS_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"${TEST_EMAIL}\", \"password\": \"wrongpassword\"}")
assert_status "Login with wrong password returns 401" "401" "$resp"

# Duplicate registration
echo ""
echo "Edge cases:"
resp=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${IRIS_URL}/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${TEST_EMAIL}\",
    \"password\": \"${TEST_PASSWORD}\",
    \"first_name\": \"${TEST_FIRST}\",
    \"last_name\": \"${TEST_LAST}\"
  }")
if [ "$resp" != "200" ] && [ "$resp" != "201" ]; then
  log_pass "Duplicate registration rejected (HTTP $resp)"
else
  log_fail "Duplicate registration should be rejected" "4xx" "$resp"
fi

print_summary "Auth"
