#!/usr/bin/env bash
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "=== Category Rules Tests ==="

register_user || { echo "FAIL: Could not register test user"; exit 1; }
echo "Registered test user: ${TEST_EMAIL}"

echo ""
echo "Rule CRUD:"
rule_resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/proxy/thesaurus/api/v1/category-rules" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{"pattern": "TRADER JOES", "category": "Groceries"}')
body=$(echo "$rule_resp" | head -1)
code=$(echo "$rule_resp" | tail -1)

assert_status "Create rule returns 2xx" "$(echo "$code" | grep -oE '2[0-9]{2}')" "$code"
rule_id=$(echo "$body" | jq -r '.id // empty')
assert_not_empty "Create rule returns id" "$rule_id"

list_resp=$(curl -s "${IRIS_URL}/api/proxy/thesaurus/api/v1/category-rules" \
  -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
rule_count=$(echo "$list_resp" | jq 'if type == "array" then length else 0 end' 2>/dev/null || echo "0")

if [ "$rule_count" -ge 1 ]; then
  log_pass "List rules returns $rule_count rule(s)"
else
  log_fail "List rules" ">= 1" "$rule_count"
fi

if [ -n "$rule_id" ]; then
  del_code=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE \
    "${IRIS_URL}/api/proxy/thesaurus/api/v1/category-rules/${rule_id}" \
    -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
  assert_status "Delete rule returns 2xx" "$(echo "$del_code" | grep -oE '2[0-9]{2}')" "$del_code"
fi

print_summary "Category Rules"
