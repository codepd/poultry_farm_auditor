#!/usr/bin/env bash
# Post-deployment API regression checks (smoke + optional authenticated integration).
# Usage:
#   REGRESSION_API_BASE=https://api-dev.example.com ./scripts/api-post-deploy-regression.sh
#   REGRESSION_MODE=full REGRESSION_EMAIL=... REGRESSION_PASSWORD=... ./scripts/api-post-deploy-regression.sh
#
# REGRESSION_MODE:
#   smoke (default) — no secrets; public routes + unauthenticated expectations
#   full — login, tenant GET, refresh rotation, logout (needs jq + credentials)

set -euo pipefail

MODE="${REGRESSION_MODE:-smoke}"
BASE="${REGRESSION_API_BASE:-https://api-dev.mykolipannai.com}"
# Strip trailing slash
BASE="${BASE%/}"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

fail() {
  echo -e "${RED}FAIL:${NC} $*" >&2
  exit 1
}

pass() {
  echo -e "${GREEN}OK:${NC} $*"
}

expect_code() {
  local name="$1" want="$2" got="$3"
  if [[ "$got" != "$want" ]]; then
    fail "$name: expected HTTP $want, got $got"
  fi
  pass "$name → $got"
}

run_smoke() {
  local code

  # Some ingress setups only forward /api/* to the app; root /health may 404.
  code=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE/health")
  if [[ "$code" == "200" ]]; then
    pass "GET /health → 200"
  elif [[ "$code" == "404" ]]; then
    pass "GET /health → 404 (skipped; ingress may only expose /api)"
  else
    fail "GET /health: expected HTTP 200 or 404, got $code"
  fi

  code=$(curl -sS -o /dev/null -w "%{http_code}" "$BASE/api/health")
  expect_code "GET /api/health" "200" "$code"

  code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$BASE/api/auth/refresh" \
    -H "Content-Type: application/json" -d '{}')
  expect_code "POST /api/auth/refresh (no cookie)" "401" "$code"

  code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$BASE/api/auth/logout" \
    -H "Content-Type: application/json" -d '{}')
  expect_code "POST /api/auth/logout" "200" "$code"

  code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$BASE/api/users/change-password" \
    -H "Content-Type: application/json" \
    -d '{"current_password":"x","new_password":"abcdef"}')
  expect_code "POST /api/users/change-password (no auth)" "401" "$code"

  code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$BASE/api/users/logout-other-devices" \
    -H "Content-Type: application/json" -d '{}')
  expect_code "POST /api/users/logout-other-devices (no auth)" "401" "$code"
}

run_full() {
  command -v jq >/dev/null 2>&1 || fail "full mode requires jq (brew install jq)"

  local email password tenant_id
  email="${REGRESSION_EMAIL:-}"
  password="${REGRESSION_PASSWORD:-}"
  tenant_id="${REGRESSION_TENANT_ID:-}"

  if [[ -z "$email" || -z "$password" ]]; then
    fail "full mode requires REGRESSION_EMAIL and REGRESSION_PASSWORD"
  fi

  local jar
  jar=$(mktemp)
  trap 'rm -f "$jar"' EXIT

  local login_body
  login_body=$(jq -n --arg e "$email" --arg p "$password" \
    '{email:$e, password:$p, remember_me:false}')

  local login_http
  login_http=$(curl -sS -w "\n%{http_code}" -c "$jar" -b "$jar" \
    -X POST "$BASE/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "$login_body")
  local login_code
  login_code=$(echo "$login_http" | tail -n1)
  local login_json
  login_json=$(echo "$login_http" | sed '$d')
  if [[ "$login_code" != "200" ]]; then
    echo "$login_json" >&2
    fail "login: expected HTTP 200, got $login_code"
  fi
  pass "POST /api/auth/login → 200"

  local token
  token=$(echo "$login_json" | jq -r '.data.token // empty')
  if [[ -z "$token" ]]; then
    echo "$login_json" >&2
    fail "login: missing .data.token"
  fi

  if [[ -z "$tenant_id" ]]; then
    tenant_id=$(echo "$login_json" | jq -r '.data.tenants[0].tenant_id // empty')
  fi
  if [[ -z "$tenant_id" || "$tenant_id" == "null" ]]; then
    fail "set REGRESSION_TENANT_ID or ensure user has at least one tenant"
  fi

  local tcode
  tcode=$(curl -sS -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $token" \
    "$BASE/api/tenants/$tenant_id")
  expect_code "GET /api/tenants/{id}" "200" "$tcode"

  local ref_http ref_code new_tok
  ref_http=$(curl -sS -w "\n%{http_code}" -c "$jar" -b "$jar" \
    -X POST "$BASE/api/auth/refresh" \
    -H "Content-Type: application/json" -d '{}')
  ref_code=$(echo "$ref_http" | tail -n1)
  ref_json=$(echo "$ref_http" | sed '$d')
  if [[ "$ref_code" != "200" ]]; then
    echo "$ref_json" >&2
    fail "refresh: expected HTTP 200, got $ref_code"
  fi
  new_tok=$(echo "$ref_json" | jq -r '.data.token // empty')
  if [[ -z "$new_tok" ]]; then
    echo "$ref_json" >&2
    fail "refresh: missing .data.token"
  fi
  pass "POST /api/auth/refresh (with cookie) → 200 + new access token"

  local tcode2
  tcode2=$(curl -sS -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $new_tok" \
    "$BASE/api/tenants/$tenant_id")
  expect_code "GET /api/tenants/{id} (after refresh)" "200" "$tcode2"

  local lcode
  lcode=$(curl -sS -o /dev/null -w "%{http_code}" -c "$jar" -b "$jar" \
    -X POST "$BASE/api/auth/logout" \
    -H "Content-Type: application/json" -d '{}')
  expect_code "POST /api/auth/logout (with session)" "200" "$lcode"

  local rcode
  rcode=$(curl -sS -o /dev/null -w "%{http_code}" -b "$jar" \
    -X POST "$BASE/api/auth/refresh" \
    -H "Content-Type: application/json" -d '{}')
  expect_code "POST /api/auth/refresh (after logout)" "401" "$rcode"
}

echo "Regression API base: $BASE (mode=$MODE)"

run_smoke

if [[ "$MODE" == "full" ]]; then
  run_full
fi

echo -e "${GREEN}All checks passed.${NC}"
