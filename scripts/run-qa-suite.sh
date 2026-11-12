#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "=================================================================="
echo "  MyKoliPannai Automated QA Test Suite"
echo "=================================================================="

echo ""
echo "[1/3] Running Backend Go Unit & Regression Tests..."
go -C "$ROOT_DIR/go_api" test -v ./handlers -run "TestClassifyFeedStage|TestAnalyticsCacheLifecycleAndInvalidation|TestSanitizeFloat|TestSeptember2026ProfitAndPerEggConsistency|TestOTPVaultEncryptionAndHashing"

echo ""
echo "[2/3] Running Frontend React / Jest Automated QA Suites..."
CI=true npm --prefix "$ROOT_DIR/react_frontend" test -- --watchAll=false --ci

echo ""
echo "[3/3] Verifying Frontend Production Bundle Compilation..."
npm --prefix "$ROOT_DIR/react_frontend" run build

echo ""
echo "=================================================================="
echo "  All Automated QA Suites Passed Successfully!"
echo "=================================================================="
