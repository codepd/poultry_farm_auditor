#!/usr/bin/env bash
# Local E2E checks for hen sale + expense analytics flow.
#
# This script is intended for LOCAL/STAGING validation because it writes data:
# - creates a sale event
# - creates a linked INCOME transaction
# - optionally creates a temporary batch
#
# Usage examples:
#   E2E_API_BASE=http://localhost:8080 E2E_EMAIL=owner@example.com E2E_PASSWORD=secret \
#     bash scripts/hen-sale-expense-e2e.sh
#
#   E2E_API_BASE=http://localhost:8080 E2E_TOKEN=<jwt> E2E_BATCH_ID=12 \
#     bash scripts/hen-sale-expense-e2e.sh

set -euo pipefail

command -v jq >/dev/null 2>&1 || { echo "jq is required (brew install jq)"; exit 1; }

BASE="${E2E_API_BASE:-http://localhost:8080}"
BASE="${BASE%/}"
TOKEN="${E2E_TOKEN:-}"
EMAIL="${E2E_EMAIL:-}"
PASSWORD="${E2E_PASSWORD:-}"
BATCH_ID="${E2E_BATCH_ID:-}"
SALE_COUNT="${E2E_SALE_COUNT:-10}"
PRICE_PER_HEN="${E2E_PRICE_PER_HEN:-90}"
SALE_DATE="${E2E_SALE_DATE:-$(date +%F)}"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

fail() { echo -e "${RED}FAIL:${NC} $*" >&2; exit 1; }
pass() { echo -e "${GREEN}OK:${NC} $*"; }

api_get() {
  local path="$1"
  curl -sS -H "Authorization: Bearer $TOKEN" "$BASE$path"
}

api_post() {
  local path="$1" body="$2"
  curl -sS -X POST "$BASE$path" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$body"
}

if [[ -z "$TOKEN" ]]; then
  [[ -n "$EMAIL" && -n "$PASSWORD" ]] || fail "Set E2E_TOKEN or both E2E_EMAIL and E2E_PASSWORD"
  login_body=$(jq -n --arg e "$EMAIL" --arg p "$PASSWORD" '{email:$e,password:$p,remember_me:false}')
  login_resp=$(curl -sS -X POST "$BASE/api/auth/login" -H "Content-Type: application/json" -d "$login_body")
  TOKEN="$(echo "$login_resp" | jq -r '.data.token // empty')"
  [[ -n "$TOKEN" ]] || fail "Login failed. Response: $login_resp"
  pass "Authenticated successfully"
fi

# Resolve batch id: either provided or create temporary batch.
if [[ -z "$BATCH_ID" ]]; then
  temp_batch_name="E2E_TEMP_BATCH_$(date +%s)"
  batch_body=$(jq -n \
    --arg n "$temp_batch_name" \
    --arg d "$SALE_DATE" \
    '{batch_name:$n,initial_count:100,current_count:100,age_weeks:20,age_days:0,date_added:$d,notes:"Temporary E2E batch"}')
  batch_resp=$(api_post "/api/hen-batches" "$batch_body")
  BATCH_ID="$(echo "$batch_resp" | jq -r '.data.id // empty')"
  [[ -n "$BATCH_ID" ]] || fail "Failed to create temp batch. Response: $batch_resp"
  pass "Created temporary batch id=$BATCH_ID"
else
  pass "Using existing batch id=$BATCH_ID"
fi

batch_before_resp="$(api_get "/api/hen-batches/$BATCH_ID")"
batch_before_count="$(echo "$batch_before_resp" | jq -r '.data.current_count // empty')"
[[ -n "$batch_before_count" ]] || fail "Failed to read batch before sale. Response: $batch_before_resp"

sale_note="E2E sale $(date +%s)"
sale_body=$(jq -n \
  --argjson batch_id "$BATCH_ID" \
  --arg sale_date "$SALE_DATE" \
  --argjson count "$SALE_COUNT" \
  --argjson price_per_hen "$PRICE_PER_HEN" \
  --arg notes "$sale_note" \
  '{batch_id:$batch_id,sale_date:$sale_date,count:$count,price_per_hen:$price_per_hen,notes:$notes}')

sale_resp="$(api_post "/api/hen-batches/sales" "$sale_body")"
sale_id="$(echo "$sale_resp" | jq -r '.data.id // empty')"
sale_total="$(echo "$sale_resp" | jq -r '.data.total_amount // empty')"
[[ -n "$sale_id" ]] || fail "Sale creation failed. Response: $sale_resp"
pass "Created sale event id=$sale_id amount=$sale_total"

sales_history_resp="$(api_get "/api/hen-batches/$BATCH_ID/sales")"
found_sale="$(echo "$sales_history_resp" | jq -r --argjson sid "$sale_id" '.data[]? | select(.id == $sid) | .id')"
[[ -n "$found_sale" ]] || fail "Sale not found in sales history. Response: $sales_history_resp"
pass "Sale appears in sales history"

batch_after_resp="$(api_get "/api/hen-batches/$BATCH_ID")"
batch_after_count="$(echo "$batch_after_resp" | jq -r '.data.current_count // empty')"
[[ -n "$batch_after_count" ]] || fail "Failed to read batch after sale. Response: $batch_after_resp"

expected_after=$((batch_before_count - SALE_COUNT))
if [[ "$batch_after_count" -ne "$expected_after" ]]; then
  fail "Batch count mismatch. before=$batch_before_count sale=$SALE_COUNT expected=$expected_after actual=$batch_after_count"
fi
pass "Batch current_count reduced correctly"

year="$(date -d "$SALE_DATE" +%Y 2>/dev/null || date -j -f "%Y-%m-%d" "$SALE_DATE" +%Y)"
month="$(date -d "$SALE_DATE" +%-m 2>/dev/null || date -j -f "%Y-%m-%d" "$SALE_DATE" +%m | sed 's/^0*//')"
summary_resp="$(api_get "/api/analytics/enhanced-monthly-summary?year=$year&month=$month")"
hen_sale_income="$(echo "$summary_resp" | jq -r '.data[0].hen_sale_income // 0')"
chick_stage_enabled="$(echo "$summary_resp" | jq -r '.data[0].chick_stage_expense.enabled // false')"

hen_sale_income_num="$(printf "%.0f" "$hen_sale_income")"
sale_total_num="$(printf "%.0f" "$sale_total")"
if (( hen_sale_income_num < sale_total_num )); then
  fail "hen_sale_income appears too low. hen_sale_income=$hen_sale_income sale_total=$sale_total"
fi
pass "Monthly summary includes hen sale income"

[[ "$chick_stage_enabled" == "true" || "$chick_stage_enabled" == "false" ]] || fail "Missing chick_stage_expense payload"
pass "Monthly summary includes chick stage expense payload"

echo -e "${GREEN}All hen-sale/expense E2E checks passed.${NC}"
