#!/usr/bin/env bash
#
# Runs all showcase scenarios against a ProteusMock server and pretty-prints
# the request and response for each one.
#
# Usage:
#   ./scripts/showcase.sh          # default: localhost:8080
#   ./scripts/showcase.sh 9090     # custom port

set -euo pipefail

PORT="${1:-8080}"
BASE="http://localhost:${PORT}"

# Colors
BOLD='\033[1m'
DIM='\033[2m'
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
MAGENTA='\033[35m'
RESET='\033[0m'

# Check if jq is available for pretty-printing JSON.
if command -v jq &>/dev/null; then
  PRETTY="jq ."
else
  PRETTY="python3 -m json.tool 2>/dev/null || cat"
fi

divider() {
  echo ""
  echo -e "${DIM}$(printf '%.0s─' {1..72})${RESET}"
  echo ""
}

banner() {
  local engine="$1"
  local title="$2"
  local tag

  if [ "$engine" = "expr" ]; then
    tag="${GREEN}[expr]${RESET}"
  else
    tag="${MAGENTA}[jinja2]${RESET}"
  fi

  echo -e "${BOLD}${tag} ${title}${RESET}"
}

run_curl() {
  local label="$1"
  shift

  echo -e "  ${DIM}${label}${RESET}"
  echo -e "  ${CYAN}curl $*${RESET}"
  echo ""

  local http_code body headers
  local tmpfile
  tmpfile=$(mktemp)

  # Run curl: capture body + headers + status code.
  http_code=$(curl -s -o "$tmpfile" -w "%{http_code}" "$@" -D /dev/stderr 2>/dev/null) || true
  body=$(cat "$tmpfile")
  rm -f "$tmpfile"

  echo -e "  ${YELLOW}HTTP ${http_code}${RESET}"

  # Pretty-print body.
  if [ -n "$body" ]; then
    echo "$body" | eval "$PRETTY" | sed 's/^/  /'
  fi
}

# ──────────────────────────────────────────────────────────────────────
#  Wait for server to be ready
# ──────────────────────────────────────────────────────────────────────

echo -e "${BOLD}Waiting for ProteusMock on port ${PORT}...${RESET}"

for i in $(seq 1 30); do
  if curl -s -o /dev/null -w "" "${BASE}/api/v1/health" 2>/dev/null; then
    echo -e "${GREEN}Server is ready.${RESET}"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo -e "\033[31mServer did not start within 30 seconds.\033[0m" >&2
    exit 1
  fi
  sleep 1
done

# ──────────────────────────────────────────────────────────────────────
#  Expr Engine Scenarios
# ──────────────────────────────────────────────────────────────────────

divider
banner "expr" "Basic Interpolation — pathParam, queryParam, header, uuid, now, randomInt"
run_curl "GET /api/v1/users/42?fields=name,email" \
  "${BASE}/api/v1/users/42?fields=name,email" \
  -H "Authorization: Bearer tok_abc"

divider
banner "expr" "Conditional — ternary operator based on X-Env header"

run_curl "X-Env: production" \
  "${BASE}/api/v1/config" \
  -H "X-Env: production"
echo ""
run_curl "X-Env: staging" \
  "${BASE}/api/v1/config" \
  -H "X-Env: staging"
echo ""
run_curl "(no header = development)" \
  "${BASE}/api/v1/config"

divider
banner "expr" "List Generation — seq() + toJSON()"
run_curl "GET /api/v1/catalog" \
  "${BASE}/api/v1/catalog"

divider
banner "expr" "Echo Body & jsonPath — extract fields from JSON request"
run_curl "POST /api/v1/echo" \
  -X POST "${BASE}/api/v1/echo" \
  -H "Content-Type: application/json" \
  -d '{"user": {"name": "Alice", "role": "admin"}}'

# ──────────────────────────────────────────────────────────────────────
#  Jinja2 Engine Scenarios
# ──────────────────────────────────────────────────────────────────────

divider
banner "jinja2" "Basic Interpolation — method, path, queryParam, header, uuid, now"
run_curl "POST /api/v1/submit?source=web" \
  -X POST "${BASE}/api/v1/submit?source=web" \
  -H "X-Request-Id: req-001" \
  -H "User-Agent: TestBot/2.0"

divider
banner "jinja2" "Conditional — if/elif/else based on X-Tier header"

run_curl "X-Tier: premium" \
  "${BASE}/api/v1/feature-flags" \
  -H "X-Tier: premium"
echo ""
run_curl "X-Tier: basic" \
  "${BASE}/api/v1/feature-flags" \
  -H "X-Tier: basic"
echo ""
run_curl "(no header = free)" \
  "${BASE}/api/v1/feature-flags"

divider
banner "jinja2" "Loops — for loop with seq(), forloop.First, randomInt"
run_curl "GET /api/v1/products" \
  "${BASE}/api/v1/products"

divider
banner "jinja2" "jsonPath Extraction — extract fields from JSON request body"
run_curl "POST /api/v1/process" \
  -X POST "${BASE}/api/v1/process" \
  -H "Content-Type: application/json" \
  -d '{"order": {"id": "ORD-999", "amount": 42.50}}'

divider
echo -e "${BOLD}${GREEN}All showcase scenarios completed.${RESET}"
echo ""
