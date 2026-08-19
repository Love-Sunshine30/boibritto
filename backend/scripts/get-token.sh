#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."   # ensure we're at backend/, regardless of where the script's called from

if [ -f .env.dev ]; then
  set -a
  source .env.dev
  set +a
fi

EMAIL="${1:-test@example.com}"
PASSWORD="${2:-testpassword123}"
WEB_API_KEY="${FIREBASE_WEB_API_KEY:?FIREBASE_WEB_API_KEY not set in .env.dev}"

curl -s -X POST \
  "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=${WEB_API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\",\"returnSecureToken\":true}" \
  | jq -r '.idToken'