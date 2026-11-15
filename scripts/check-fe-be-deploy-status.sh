#!/usr/bin/env bash
# Combined FE+BE deployment status check for prod (default) or staging.
# Usage:
#   bash scripts/check-fe-be-deploy-status.sh
#   bash scripts/check-fe-be-deploy-status.sh --env staging
#   DEPLOY_ENV=staging bash scripts/check-fe-be-deploy-status.sh

set -u

DEPLOY_ENV="${DEPLOY_ENV:-prod}"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --env|-e)
      DEPLOY_ENV="${2:-prod}"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

PROFILE="${AWS_PROFILE:-AdminAccess-Pradeep}"
K8S_DEPLOYMENT="${K8S_DEPLOYMENT:-go-api}"

if [[ "$DEPLOY_ENV" == "staging" ]]; then
  CLOUDFRONT_DISTRIBUTION_ID="${CLOUDFRONT_DISTRIBUTION_ID:-E2NM4WGGES128L}"
  FRONTEND_URL="${FRONTEND_URL:-https://staging.mykolipannai.com}"
  S3_BUCKET="${S3_BUCKET:-app-staging.mykolipannai.com-frontend}"
  API_BASE="${API_BASE:-https://api-staging.mykolipannai.com}"
  K8S_NAMESPACE="${K8S_NAMESPACE:-poultry-staging}"
else
  CLOUDFRONT_DISTRIBUTION_ID="${CLOUDFRONT_DISTRIBUTION_ID:-E2TLR40Y95CSC6}"
  FRONTEND_URL="${FRONTEND_URL:-https://mykolipannai.com}"
  S3_BUCKET="${S3_BUCKET:-app-dev.mykolipannai.com-frontend}"
  API_BASE="${API_BASE:-https://api-dev.mykolipannai.com}"
  K8S_NAMESPACE="${K8S_NAMESPACE:-poultry-dev}"
fi

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  echo -e "${GREEN}PASS${NC} - $*"
}

warn() {
  WARN_COUNT=$((WARN_COUNT + 1))
  echo -e "${YELLOW}WARN${NC} - $*"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  echo -e "${RED}FAIL${NC} - $*"
}

check_cmd() {
  command -v "$1" >/dev/null 2>&1
}

print_header() {
  echo "===================================================="
  echo " FE + BE Deployment Status Check ($DEPLOY_ENV)"
  echo "===================================================="
  echo "AWS_PROFILE: $PROFILE"
  echo "Frontend: $FRONTEND_URL (S3: $S3_BUCKET, CF: $CLOUDFRONT_DISTRIBUTION_ID)"
  echo "API Base: $API_BASE"
  echo "K8s: namespace=$K8S_NAMESPACE deployment=$K8S_DEPLOYMENT"
  echo ""
}

print_section() {
  echo ""
  echo "--- $* ---"
}

check_latest_workflow() {
  local workflow_file="$1"
  local label="$2"
  local raw

  if ! check_cmd gh; then
    warn "gh CLI not installed; skipping $label workflow check"
    return
  fi

  raw=$(gh run list --workflow "$workflow_file" --limit 1 \
    --json databaseId,headSha,status,conclusion,updatedAt,displayTitle 2>/dev/null || true)

  if [[ -z "$raw" || "$raw" == "[]" ]]; then
    warn "$label: no workflow runs found"
    return
  fi

  local status conclusion sha updated title
  status=$(echo "$raw" | jq -r '.[0].status // "unknown"' 2>/dev/null || echo "unknown")
  conclusion=$(echo "$raw" | jq -r '.[0].conclusion // "unknown"' 2>/dev/null || echo "unknown")
  sha=$(echo "$raw" | jq -r '.[0].headSha // "unknown"' 2>/dev/null || echo "unknown")
  updated=$(echo "$raw" | jq -r '.[0].updatedAt // "unknown"' 2>/dev/null || echo "unknown")
  title=$(echo "$raw" | jq -r '.[0].displayTitle // "unknown"' 2>/dev/null || echo "unknown")

  echo "$label latest run:"
  echo "  title: $title"
  echo "  sha:   $sha"
  echo "  at:    $updated"
  echo "  state: $status / $conclusion"

  if [[ "$status" == "completed" && "$conclusion" == "success" ]]; then
    pass "$label workflow is successful"
  elif [[ "$status" == "in_progress" || "$status" == "queued" ]]; then
    warn "$label workflow is still running ($status)"
  else
    fail "$label workflow is not successful ($status/$conclusion)"
  fi
}

check_cloudfront() {
  print_section "Frontend CDN"

  if ! check_cmd aws; then
    warn "aws CLI not installed; skipping CloudFront check"
    return
  fi

  local out
  out=$(aws cloudfront list-invalidations \
    --distribution-id "$CLOUDFRONT_DISTRIBUTION_ID" \
    --max-items 1 \
    --profile "$PROFILE" \
    --query 'InvalidationList.Items[0].[Id,Status,CreateTime]' \
    --output text 2>/dev/null || true)

  if [[ -z "$out" || "$out" == "None" ]]; then
    warn "No invalidation found or CloudFront query failed"
    return
  fi

  local id status created
  id=$(echo "$out" | awk '{print $1}')
  status=$(echo "$out" | awk '{print $2}')
  created=$(echo "$out" | awk '{print $3}')

  echo "Latest invalidation:"
  echo "  id:      $id"
  echo "  status:  $status"
  echo "  created: $created"

  if [[ "$status" == "Completed" ]]; then
    pass "CloudFront invalidation completed"
  else
    warn "CloudFront invalidation status is $status"
  fi

  local s3_js live_js
  s3_js=$(aws s3 ls "s3://$S3_BUCKET/static/js/" --profile "$PROFILE" 2>/dev/null | grep -oE 'main\.[a-f0-9]+\.js' | head -n 1 || true)
  live_js=$(curl -sS "$FRONTEND_URL/" 2>/dev/null | grep -oE 'main\.[a-f0-9]+\.js' | head -n 1 || true)
  echo "S3 JS bundle:   ${s3_js:-unknown}"
  echo "Live JS bundle: ${live_js:-unknown}"
  if [[ -n "$s3_js" && "$s3_js" == "$live_js" ]]; then
    pass "Live frontend bundle matches S3 ($live_js)"
  else
    warn "Live frontend bundle (${live_js:-none}) does not match S3 (${s3_js:-none})"
  fi
}

check_api_health() {
  print_section "Backend API"

  if ! check_cmd curl; then
    fail "curl is required for API health check"
    return
  fi

  local code
  code=$(curl -sS -o /dev/null -w "%{http_code}" "$API_BASE/api/health" || echo "000")
  echo "GET $API_BASE/api/health -> $code"

  if [[ "$code" == "200" ]]; then
    pass "API health endpoint is reachable"
  else
    fail "API health endpoint expected 200, got $code"
  fi

  if [[ -f "scripts/api-post-deploy-regression.sh" ]]; then
    local regression_exit=0
    REGRESSION_API_BASE="$API_BASE" REGRESSION_MODE=smoke \
      bash scripts/api-post-deploy-regression.sh >/tmp/api-regression.log 2>&1 || regression_exit=$?

    if [[ "$regression_exit" -eq 0 ]]; then
      pass "API smoke regression passed"
    else
      fail "API smoke regression failed (see /tmp/api-regression.log)"
      echo "Last regression output:"
      tail -n 20 /tmp/api-regression.log
    fi
  else
    warn "scripts/api-post-deploy-regression.sh not found; skipping smoke regression"
  fi
}

check_k8s_rollout() {
  print_section "Kubernetes Runtime"

  if ! check_cmd kubectl; then
    warn "kubectl not installed; skipping K8s checks"
    return
  fi

  if ! kubectl -n "$K8S_NAMESPACE" get deploy "$K8S_DEPLOYMENT" >/dev/null 2>&1; then
    fail "Cannot access deployment $K8S_NAMESPACE/$K8S_DEPLOYMENT"
    return
  fi

  local image ready updated available
  image=$(kubectl -n "$K8S_NAMESPACE" get deploy "$K8S_DEPLOYMENT" -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || echo "unknown")
  ready=$(kubectl -n "$K8S_NAMESPACE" get deploy "$K8S_DEPLOYMENT" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
  updated=$(kubectl -n "$K8S_NAMESPACE" get deploy "$K8S_DEPLOYMENT" -o jsonpath='{.status.updatedReplicas}' 2>/dev/null || echo "0")
  available=$(kubectl -n "$K8S_NAMESPACE" get deploy "$K8S_DEPLOYMENT" -o jsonpath='{.status.availableReplicas}' 2>/dev/null || echo "0")

  echo "Deployment image: $image"
  echo "Replicas: ready=$ready updated=$updated available=$available"

  if [[ "${ready:-0}" -ge 1 && "${updated:-0}" -ge 1 && "${available:-0}" -ge 1 ]]; then
    pass "K8s deployment has healthy replicas"
  else
    fail "K8s deployment is not fully healthy"
  fi

  local rollout_output rollout_exit=0
  rollout_output=$(kubectl -n "$K8S_NAMESPACE" rollout status "deploy/$K8S_DEPLOYMENT" --timeout=20s 2>&1) || rollout_exit=$?
  echo "$rollout_output"

  if [[ "$rollout_exit" -eq 0 ]]; then
    pass "K8s rollout status is complete"
  else
    fail "K8s rollout status check failed"
  fi
}

print_summary() {
  print_section "Summary"
  echo "PASS: $PASS_COUNT"
  echo "WARN: $WARN_COUNT"
  echo "FAIL: $FAIL_COUNT"
  echo ""

  if [[ "$FAIL_COUNT" -gt 0 ]]; then
    echo -e "${RED}Deployment is NOT fully verified.${NC}"
    exit 1
  fi

  if [[ "$WARN_COUNT" -gt 0 ]]; then
    echo -e "${YELLOW}Deployment mostly verified with warnings.${NC}"
    exit 0
  fi

  echo -e "${GREEN}Deployment verified successfully.${NC}"
}

main() {
  print_header

  print_section "GitHub Actions"
  if check_cmd jq; then
    check_latest_workflow "staging-deploy.yml" "Staging + E2E + Prod deploy"
    check_latest_workflow "pr-ci.yml" "PR CI (Go, React, Functional E2E)"
  else
    warn "jq not installed; install jq to parse workflow output"
  fi

  check_cloudfront
  check_api_health
  check_k8s_rollout
  print_summary
}

main "$@"
