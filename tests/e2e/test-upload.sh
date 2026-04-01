#!/usr/bin/env bash
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "=== Upload Tests ==="

register_user || { echo "FAIL: Could not register test user"; exit 1; }
echo "Registered test user: ${TEST_EMAIL}"

# Upload CSV
echo ""
echo "CSV upload flow:"
FIXTURE="$SCRIPT_DIR/../fixtures/sample.csv"
resp=$(curl -s -w "\n%{http_code}" -X POST "${IRIS_URL}/api/upload" \
  -H "Authorization: Bearer ${TOKEN}" \
  -F "file=@${FIXTURE}")
body=$(echo "$resp" | head -1)
code=$(echo "$resp" | tail -1)

assert_status "Upload CSV returns 2xx" "$(echo "$code" | grep -oE '2[0-9]{2}')" "$code"
doc_id=$(echo "$body" | jq -r '.document_id // .id // empty')
assert_not_empty "Upload returns document_id" "$doc_id"

# Poll for processing
echo ""
echo "Processing poll:"
PROCESSED=false
for i in $(seq 1 15); do
  status_resp=$(curl -s "${THESAURUS_URL}/internal/documents/${doc_id}/status" 2>/dev/null || echo "{}")
  doc_status=$(echo "$status_resp" | jq -r '.status // empty')
  if [ "$doc_status" = "processed" ]; then
    PROCESSED=true
    break
  fi
  sleep 2
done

if [ "$PROCESSED" = true ]; then
  log_pass "Document processed within 30s"
else
  log_fail "Document processing timed out" "processed" "$doc_status"
fi

# Fetch transactions
echo ""
echo "Transaction verification:"
txn_resp=$(curl -s "${THESAURUS_URL}/internal/transactions/by-document?document_id=${doc_id}" 2>/dev/null || echo "[]")
txn_count=$(echo "$txn_resp" | jq 'if type == "array" then length else .transactions | length // 0 end' 2>/dev/null || echo "0")

if [ "$txn_count" -ge 4 ]; then
  log_pass "Found $txn_count transactions (expected >= 4)"
else
  log_fail "Transaction count" ">= 4" "$txn_count"
fi

print_summary "Upload"
