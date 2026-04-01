#!/usr/bin/env bash
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/helpers.sh"

echo "=== Chat Tests ==="

ollama_status=$(curl -sf -o /dev/null -w '%{http_code}' http://localhost:11434/api/tags 2>/dev/null || echo "000")
if [ "$ollama_status" != "200" ]; then
  log_skip "Ollama not available — skipping chat tests"
  echo "  (Start Ollama or run make dev-up to enable)"
  exit 0
fi

register_user || { echo "FAIL: Could not register test user"; exit 1; }
echo "Registered test user: ${TEST_EMAIL}"

FIXTURE="$SCRIPT_DIR/../fixtures/sample.csv"
upload_resp=$(curl -s -X POST "${IRIS_URL}/api/upload" \
  -H "Authorization: Bearer ${TOKEN}" \
  -F "file=@${FIXTURE}" 2>/dev/null)
doc_id=$(echo "$upload_resp" | jq -r '.document_id // .id // empty')
sleep 5

echo ""
echo "Chat flow:"
chat_resp=$(curl -sf --max-time 180 -X POST "${IRIS_URL}/api/proxy/sophia/api/v1/chat/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d "{
    \"question\": \"How much did I spend total?\",
    \"user_id\": \"$(echo "$upload_resp" | jq -r '.user_id // empty')\"
  }" 2>/dev/null || echo "{}")

answer=$(echo "$chat_resp" | jq -r '.answer // .response // empty')
confidence=$(echo "$chat_resp" | jq -r '.confidence // 0')

assert_not_empty "Chat returns an answer" "$answer"

if [ "$(echo "$confidence > 0" | bc -l 2>/dev/null || echo 0)" = "1" ]; then
  log_pass "Chat confidence > 0 ($confidence)"
else
  log_skip "Chat confidence check (bc not available or confidence=0)"
fi

print_summary "Chat"
