#!/usr/bin/env bash
# End-to-end test for the requests API: create -> request -> accept ->
# dual-confirm -> return, plus the ownership/authorization checks.
#
# Usage: ./test-requests-api.sh
# Requires: jq, curl, and scripts/get-token.sh (from earlier in this project)
# Run from backend/, with the API already running (go run ./cmd/api).

set -uo pipefail  # NOT -e — we want to keep going and report failures, not abort on the first one

BASE_URL="${BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

green() { echo -e "\033[32m$1\033[0m"; }
red()   { echo -e "\033[31m$1\033[0m"; }

# check <description> <expected_status> <actual_status> <body>
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

# req <method> <path> <token> [json_body]
# Prints "STATUS\nBODY" separated by a newline — callers split on first line.
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
  red "Could not get a token for usera@example.com — create this user in Firebase console first."
  exit 1
fi
if [ -z "$TOKEN_B" ] || [ "$TOKEN_B" = "null" ]; then
  red "Could not get a token for userb@example.com — create this user in Firebase console first."
  exit 1
fi
green "Tokens acquired for user A (owner) and user B (borrower)."
echo

# ---------------------------------------------------------------------------
# 1. A creates a book
# ---------------------------------------------------------------------------
echo "=== 1. Create book (as A) ==="
RESULT=$(req POST /api/v1/books "$TOKEN_A" '{"title":"Dune '"$(date +%s)"'","author":"Frank Herbert"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "create book returns 201" "201" "$STATUS" "$BODY"
BOOK_ID=$(field "$BODY" '.data.id')
echo "  book_id=$BOOK_ID"
echo

# ---------------------------------------------------------------------------
# 2. B requests to borrow it, with a message
# ---------------------------------------------------------------------------
echo "=== 2. B requests the book, with a message ==="
RESULT=$(req POST "/api/v1/books/$BOOK_ID/requests" "$TOKEN_B" '{"message":"Could I borrow this for a week?"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "create request returns 201" "201" "$STATUS" "$BODY"
check "status is pending" "pending" "$(field "$BODY" '.data.status')" "$BODY"
check "message round-trips" "Could I borrow this for a week?" "$(field "$BODY" '.data.message')" "$BODY"
REQ_ID=$(field "$BODY" '.data.id')
echo "  request_id=$REQ_ID"
echo

# ---------------------------------------------------------------------------
# 3. Duplicate request should be rejected with 409
# ---------------------------------------------------------------------------
echo "=== 3. B requests the SAME book again (should conflict) ==="
RESULT=$(req POST "/api/v1/books/$BOOK_ID/requests" "$TOKEN_B" '{}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "duplicate pending request returns 409" "409" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# 4. B cannot accept their own request
# ---------------------------------------------------------------------------
echo "=== 4. B tries to accept the request (should be forbidden — not the owner) ==="
RESULT=$(req PATCH "/api/v1/requests/$REQ_ID" "$TOKEN_B" '{"status":"accepted"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "non-owner accept returns 403" "403" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# 5. A accepts the request
# ---------------------------------------------------------------------------
echo "=== 5. A accepts the request ==="
RESULT=$(req PATCH "/api/v1/requests/$REQ_ID" "$TOKEN_A" '{"status":"accepted"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "accept returns 200" "200" "$STATUS" "$BODY"
check "status is accepted" "accepted" "$(field "$BODY" '.data.status')" "$BODY"
echo

# ---------------------------------------------------------------------------
# 6. Book should still be available — nobody has confirmed handoff yet
# ---------------------------------------------------------------------------
echo "=== 6. Book still available after accept (pre-confirmation) ==="
RESULT=$(req GET "/api/v1/books" "$TOKEN_A")
BODY=$(echo "$RESULT" | tail -n +2)
AVAILABLE=$(echo "$BODY" | jq -r ".data.books[] | select(.id==$BOOK_ID) | .available")
check "book available == true" "true" "$AVAILABLE" "$BODY"
echo

# ---------------------------------------------------------------------------
# 7. B confirms handoff first — book should STILL be available
# ---------------------------------------------------------------------------
echo "=== 7. B confirms handoff (one-sided — book should stay available) ==="
RESULT=$(req POST "/api/v1/requests/$REQ_ID/confirm" "$TOKEN_B" '')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "confirm returns 200" "200" "$STATUS" "$BODY"
check "status still accepted (not active yet)" "accepted" "$(field "$BODY" '.data.status')" "$BODY"
check "borrower_confirmed is true" "true" "$(field "$BODY" '.data.borrower_confirmed')" "$BODY"
check "owner_confirmed still false" "false" "$(field "$BODY" '.data.owner_confirmed')" "$BODY"

RESULT=$(req GET "/api/v1/books" "$TOKEN_A")
BODY=$(echo "$RESULT" | tail -n +2)
AVAILABLE=$(echo "$BODY" | jq -r ".data.books[] | select(.id==$BOOK_ID) | .available")
check "book STILL available (only one side confirmed)" "true" "$AVAILABLE" "$BODY"
echo

# ---------------------------------------------------------------------------
# 8. A confirms handoff — NOW both sides confirmed, book unavailable
# ---------------------------------------------------------------------------
echo "=== 8. A confirms handoff (both sides now — book should become unavailable) ==="
RESULT=$(req POST "/api/v1/requests/$REQ_ID/confirm" "$TOKEN_A" '')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "confirm returns 200" "200" "$STATUS" "$BODY"
check "status is now active" "active" "$(field "$BODY" '.data.status')" "$BODY"

RESULT=$(req GET "/api/v1/books" "$TOKEN_A")
BODY=$(echo "$RESULT" | tail -n +2)
AVAILABLE=$(echo "$BODY" | jq -r ".data.books[] | select(.id==$BOOK_ID) | .available")
check "book now UNAVAILABLE (both confirmed)" "false" "$AVAILABLE" "$BODY"
echo

# ---------------------------------------------------------------------------
# 9. B (borrower) cannot mark it returned — owner only
# ---------------------------------------------------------------------------
echo "=== 9. B tries to mark returned (should be forbidden — not the owner) ==="
RESULT=$(req POST "/api/v1/requests/$REQ_ID/return" "$TOKEN_B" '')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "non-owner return returns 403" "403" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# 10. A marks it returned — book becomes available again
# ---------------------------------------------------------------------------
echo "=== 10. A marks the book returned ==="
RESULT=$(req POST "/api/v1/requests/$REQ_ID/return" "$TOKEN_A" '')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "return returns 200" "200" "$STATUS" "$BODY"
check "status is returned" "returned" "$(field "$BODY" '.data.status')" "$BODY"

RESULT=$(req GET "/api/v1/books" "$TOKEN_A")
BODY=$(echo "$RESULT" | tail -n +2)
AVAILABLE=$(echo "$BODY" | jq -r ".data.books[] | select(.id==$BOOK_ID) | .available")
check "book available again after return" "true" "$AVAILABLE" "$BODY"
echo

# ---------------------------------------------------------------------------
# 11. Sent / incoming lists reflect the finished request
# ---------------------------------------------------------------------------
echo "=== 11. Sent/incoming lists ==="
RESULT=$(req GET "/api/v1/requests/sent" "$TOKEN_B")
BODY=$(echo "$RESULT" | tail -n +2)
FOUND=$(echo "$BODY" | jq -r ".data[] | select(.id==$REQ_ID) | .id")
check "request appears in B's sent list" "$REQ_ID" "$FOUND" "$BODY"

RESULT=$(req GET "/api/v1/requests/incoming" "$TOKEN_A")
BODY=$(echo "$RESULT" | tail -n +2)
FOUND=$(echo "$BODY" | jq -r ".data[] | select(.id==$REQ_ID) | .id")
check "request appears in A's incoming list" "$REQ_ID" "$FOUND" "$BODY"
echo

# ---------------------------------------------------------------------------
# 12. Rejection path — separate book, separate request
# ---------------------------------------------------------------------------
echo "=== 12. Rejection path (separate book) ==="
RESULT=$(req POST /api/v1/books "$TOKEN_A" '{"title":"Rejected Book '"$(date +%s)"'","author":"Someone"}')
BODY=$(echo "$RESULT" | tail -n +2)
BOOK_ID_2=$(field "$BODY" '.data.id')

RESULT=$(req POST "/api/v1/books/$BOOK_ID_2/requests" "$TOKEN_B" '{}')
BODY=$(echo "$RESULT" | tail -n +2)
REQ_ID_2=$(field "$BODY" '.data.id')

RESULT=$(req PATCH "/api/v1/requests/$REQ_ID_2" "$TOKEN_A" '{"status":"rejected"}')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "reject returns 200" "200" "$STATUS" "$BODY"
check "status is rejected" "rejected" "$(field "$BODY" '.data.status')" "$BODY"

RESULT=$(req POST "/api/v1/requests/$REQ_ID_2/confirm" "$TOKEN_A" '')
STATUS=$(echo "$RESULT" | head -n1)
BODY=$(echo "$RESULT" | tail -n +2)
check "confirming a rejected request is rejected (400)" "400" "$STATUS" "$BODY"
echo

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo "==============================="
if [ "$FAIL" -eq 0 ]; then
  green "ALL $PASS CHECKS PASSED"
else
  red "$FAIL FAILED, $PASS PASSED"
fi
echo "==============================="
exit $([ "$FAIL" -eq 0 ] && echo 0 || echo 1)