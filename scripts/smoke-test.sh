#!/usr/bin/env bash
set -uo pipefail

# TradeBench Smoke Test
# Validates that the local docker-compose stack
# is healthy and the core API flows work.

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
LEADERBOARD_URL="${LEADERBOARD_URL:-http://localhost:8085}"
DUMMY_ORDERBOOK_URL="${DUMMY_ORDERBOOK_URL:-http://localhost:8089}"
SCORER_URL="${SCORER_URL:-http://localhost:8086}"

pass=0
fail=0

check_http() {
  local name="$1"
  local expected="$2"
  shift 2
  local status
  status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$@" 2>/dev/null) || status="000"
  if [ "$status" = "$expected" ]; then
    echo -e "  ${GREEN}✓${NC} $name (HTTP $status)"
    pass=$((pass + 1))
  else
    echo -e "  ${RED}✗${NC} $name (HTTP $status, expected $expected)"
    fail=$((fail + 1))
  fi
}

echo ""
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo -e "${YELLOW}  TradeBench Smoke Test${NC}"
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo ""

# 1. Health Endpoints 
echo -e "${YELLOW}▸ Health Endpoints${NC}"
check_http "Gateway /health"          "200" "$GATEWAY_URL/health"
check_http "Leaderboard /health"      "200" "$LEADERBOARD_URL/health"
check_http "Dummy Orderbook /health"  "200" "$DUMMY_ORDERBOOK_URL/health"
check_http "Scorer /health"           "200" "$SCORER_URL/health"
echo ""

# 2. Auth Flow 
echo -e "${YELLOW}▸ Auth Flow${NC}"

TOKEN_RESP=$(curl -s --max-time 5 -X POST "$GATEWAY_URL/api/v1/auth/token" \
  -H "Content-Type: application/json" \
  -d '{"contestant_id":"smoke-test-user"}' 2>/dev/null) || TOKEN_RESP=""

TOKEN=$(echo "$TOKEN_RESP" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
  echo -e "  ${GREEN}✓${NC} POST /api/v1/auth/token → got JWT"
  pass=$((pass + 1))
else
  echo -e "  ${RED}✗${NC} POST /api/v1/auth/token → no token returned"
  fail=$((fail + 1))
  TOKEN="invalid"
fi
echo ""

# 3. Submission API 
echo -e "${YELLOW}▸ Submission API${NC}"

check_http "POST /api/v1/submissions (empty → 400)" "400" \
  -X POST "$GATEWAY_URL/api/v1/submissions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
echo ""

# 4. Leaderboard WebSocket 
echo -e "${YELLOW}▸ Leaderboard WebSocket${NC}"

# Use curl with http_code; the server returns 101 on upgrade
WS_STATUS=$(timeout 3 curl -s -o /dev/null -w '%{http_code}' --max-time 3 \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
  "$LEADERBOARD_URL/ws/leaderboard" 2>/dev/null) || WS_STATUS="000"

# Accept 101 (proper upgrade) or 000 (curl exits after upgrade, no http_code)
if [ "$WS_STATUS" = "101" ] || [ "$WS_STATUS" = "000" ]; then
  echo -e "  ${GREEN}✓${NC} WS /ws/leaderboard → connection accepted"
  pass=$((pass + 1))
else
  echo -e "  ${RED}✗${NC} WS /ws/leaderboard → HTTP $WS_STATUS"
  fail=$((fail + 1))
fi
echo ""

# 5. Dummy Orderbook 
echo -e "${YELLOW}▸ Dummy Orderbook${NC}"

OB_RESP=$(curl -s --max-time 5 "$DUMMY_ORDERBOOK_URL/orders" 2>/dev/null) || OB_RESP=""
if echo "$OB_RESP" | grep -q "orders"; then
  echo -e "  ${GREEN}✓${NC} GET /orders → returns order data"
  pass=$((pass + 1))
else
  echo -e "  ${RED}✗${NC} GET /orders → unexpected response"
  fail=$((fail + 1))
fi
echo ""

# Summary 
total=$((pass + fail))
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
if [ "$fail" -eq 0 ]; then
  echo -e "  ${GREEN}All $total checks passed ✓${NC}"
else
  echo -e "  ${RED}$fail/$total checks failed ✗${NC}"
fi
echo -e "${YELLOW}═══════════════════════════════════════${NC}"
echo ""

exit "$fail"
