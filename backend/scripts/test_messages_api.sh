#!/usr/bin/env bash
# End-to-end test for the messages API: accepting a request creates a
# thread -> both participants can send/list messages -> non-participants
# and premature access are rejected.
#
# Usage: ./test-messages-api.sh
# Requires: jq, curl, scripts/get-token.sh
# Run from backend/, with the API already running (go run ./cmd/api).
#
# Optional: set USERC_EMAIL / USERC_PASSWORD to a THIRD real Firebase user
# to test the "non-participant is forbidden" case. If unset, that check is
# skipped (noted in the output) rather than failing.

set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0
SKIP=0

green() { echo -e "\033[32m$1\033[0m"; }
red()   { echo -e "\033[31m$1\033[0m"; }
yellow(){ echo -e "\033[33m$1\033[0m"; }

check() {
  local desc="$1" expected="$2" actual="$3" body="$4"
  if [ "$actual" = "$expected" ]; then
    green "  PASS  $desc (got $actual)"
    PASS=$((PASS + 1))
  else
    red   "  FAIL  $desc (expected $expected, got $actual)"
    echo "        body: $body"
    FAIL=$((FAIL + 1))
  fi
}

skip() {
  yellow "  SKIP  $1"
  SKIP=$((SKIP + 1))
}

req() {
  local method="$1" path="$2" token="$3" body="${4:-}"
  local response status payload
  if [ -n "$body" ]; then
    response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$path" \
      -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "$body")
  else
    response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$path" \
      -H "Authorization: Bearer $token")
  fi
  status=$(echo "$response" | tail -n1)
  payload=$(echo "$response" | sed '$d')
  echo "$status"
  echo "$payload"
}

field() { echo "$1" | jq -r "$2" 2>/dev/null; }

echo "=== Setting up tokens ==="
TOKEN_A=$(./scripts/get-token.sh usera@example.com passwordA)
TOKEN_B=$(./scripts/get-token.sh userb@example.com passwordB)

if [ -z "$TOKEN_A" ] || [ "$TOKEN_A" = "null" ]; then
  red "Could not get a token for usera@example.com"; exit 1
fi
if [ -z "$TOKEN_B" ] || [ "$TOKEN_B" = "null" ]; then
  red "Could not get a token for userb@example.com"; exit 1
fi

TOKEN_C=""
if [ -n "${USERC_EMAIL:-}" ] && [ -n "${USERC_PASSWORD:-}" ]; then
  TOKEN_C=$(./scripts/get-token.sh "$USERC_EMAIL" "$USERC_PASSWORD")
  if [ -z "$TOKEN_C" ] || [ "$TOKEN_C" = "null" ]; then
    yellow "Could not get a token for USERC_EMAIL — non-participant test will be skipped."
    TOKEN_C=""
  fi
fi

green "Tokens acquired."
echo

# ---------------------------------------------------------------------------
# 1. Set up: A creates book, B requests, but NOT yet accepted
# ---------------------------------------------------------------------------
echo "=== 1. Setup: create book + pending request (not yet accepted) ==="
RESULT=$(req POST /api/v1/books "$TOKEN_A" '{"title":"Msg Test '"$(date +%s)"'","author":"Someone"}')
BODY=$(echo "$RESULT" | tail -n +2)
BOOK_ID=$(field "$BODY" '.data.id')
echo "  book_id=$BOOK_ID"

RESULT=$(req POST "/api/v1/books/$BOOK_ID/requests" "$TOKEN_B" '{}')
BODY=$(echo "$RESULT" | tail -n +2)
REQ_ID=$(field "$BODY" '.data.id')
echo "  request_id=$REQ_ID"
echo

# ---------------------------------------------------------------------------
# 2. Before acceptance, no thread exists yet — sending should fail
# ---------------------------------------------------------------------------
echo "=== 2. B tries to message before request is accepted (no thread yet) ==="
RESULT=$(req POST "/api/v1/threads/$REQ_ID/messages" "$TOKEN_B" '{"body":"hello?"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
if [ "$STATUS" = "403" ] || [ "$STATUS" = "404" ]; then
  green "  PASS  message before acceptance rejected (got $STATUS)"
  PASS=$((PASS + 1))
else
  red "  FAIL  expected 403 or 404, got $STATUS"
  echo "        body: $BODY"
  FAIL=$((FAIL + 1))
fi
echo

# ---------------------------------------------------------------------------
# 3. A accepts — this should create the thread
# ---------------------------------------------------------------------------
echo "=== 3. A accepts the request (thread should be created) ==="
RESULT=$(req PATCH "/api/v1/requests/$REQ_ID" "$TOKEN_A" '{"status":"accepted"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "accept returns 200" "200" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# 4. B sends the first message
# ---------------------------------------------------------------------------
echo "=== 4. B sends a message ==="
RESULT=$(req POST "/api/v1/threads/$REQ_ID/messages" "$TOKEN_B" '{"body":"Hi! When can I pick this up?"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "send message returns 201" "201" "$STATUS" "$BODY"
check "message body round-trips" "Hi! When can I pick this up?" "$(field "$BODY" '.data.body')" "$BODY"
echo

# ---------------------------------------------------------------------------
# 5. A replies
# ---------------------------------------------------------------------------
echo "=== 5. A replies ==="
RESULT=$(req POST "/api/v1/threads/$REQ_ID/messages" "$TOKEN_A" '{"body":"How about tomorrow evening?"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "reply returns 201" "201" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# 6. Empty message should be rejected
# ---------------------------------------------------------------------------
echo "=== 6. Empty message body is rejected ==="
RESULT=$(req POST "/api/v1/threads/$REQ_ID/messages" "$TOKEN_B" '{"body":"   "}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "whitespace-only body returns 400" "400" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# 7. Overly long message should be rejected
# ---------------------------------------------------------------------------
echo "=== 7. Overly long message is rejected ==="
LONG_BODY=$(printf 'a%.0s' {1..2001})
RESULT=$(req POST "/api/v1/threads/$REQ_ID/messages" "$TOKEN_B" "{\"body\":\"$LONG_BODY\"}")
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "2001-char body returns 400" "400" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# 8. Thread appears in both participants' thread lists
# ---------------------------------------------------------------------------
echo "=== 8. Thread appears in both participants' lists ==="
RESULT=$(req GET "/api/v1/threads" "$TOKEN_A")
BODY=$(echo "$RESULT" | tail -n +2)
FOUND=$(echo "$BODY" | jq -r ".data[] | select(.request_id==$REQ_ID) | .request_id")
check "thread appears in A's list" "$REQ_ID" "$FOUND" "$BODY"

RESULT=$(req GET "/api/v1/threads" "$TOKEN_B")
BODY=$(echo "$RESULT" | tail -n +2)
FOUND=$(echo "$BODY" | jq -r ".data[] | select(.request_id==$REQ_ID) | .request_id")
check "thread appears in B's list" "$REQ_ID" "$FOUND" "$BODY"

RESULT=$(req GET "/api/v1/threads" "$TOKEN_A")
BODY=$(echo "$RESULT" | tail -n +2)
PREVIEW=$(echo "$BODY" | jq -r ".data[] | select(.request_id==$REQ_ID) | .last_message_preview")
check "last_message_preview reflects most recent message" "How about tomorrow evening?" "$PREVIEW" "$BODY"
echo

# ---------------------------------------------------------------------------
# 9. Non-participant cannot read or send to this thread
# ---------------------------------------------------------------------------
echo "=== 9. Non-participant is forbidden ==="
if [ -n "$TOKEN_C" ]; then
  RESULT=$(req POST "/api/v1/threads/$REQ_ID/messages" "$TOKEN_C" '{"body":"can I join?"}')
  STATUS=$(echo "$RESULT" | head -n1)
  BODY=$(echo "$RESULT" | tail -n +2)
  check "non-participant send returns 403" "403" "$STATUS" "$BODY"
else
  skip "non-participant test — set USERC_EMAIL / USERC_PASSWORD to enable"
fi
echo

# ---------------------------------------------------------------------------
# 10. Unauthenticated request is rejected
# ---------------------------------------------------------------------------
echo "=== 10. Unauthenticated request is rejected ==="
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/threads/$REQ_ID/messages" \
  -H "Content-Type: application/json" -d '{"body":"no auth"}')
check "no token returns 401" "401" "$STATUS" "(no body captured)"
echo

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo "==============================="
if [ "$FAIL" -eq 0 ]; then
  green "ALL $PASS CHECKS PASSED${SKIP:+ ($SKIP skipped)}"
else
  red "$FAIL FAILED, $PASS PASSED, $SKIP SKIPPED"
fi
echo "==============================="
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)