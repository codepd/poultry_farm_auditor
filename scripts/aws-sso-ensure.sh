#!/usr/bin/env bash
# Ensure AWS SSO login is active for a profile.
# If expired, starts login with --no-browser and opens URL in a fixed Chrome profile.

set -euo pipefail

PROFILE="${AWS_PROFILE:-AdminAccess-Pradeep}"
REGION="${AWS_REGION:-ap-south-1}"
CHROME_BIN="${CHROME_BIN:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
CHROME_PROFILE_DIR="${CHROME_PROFILE_DIR:-Profile 8}"
OPEN_LOGIN_URL="${OPEN_LOGIN_URL:-1}"
STS_TIMEOUT_SECONDS="${STS_TIMEOUT_SECONDS:-8}"

sts_ok() {
  aws sts get-caller-identity \
    --profile "$PROFILE" \
    --region "$REGION" \
    >/dev/null 2>&1
}

extract_login_url() {
  local file="$1"
  python3 - "$file" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
text = path.read_text(errors="ignore")
match = re.search(r"https://oidc[^\s]+", text)
print(match.group(0) if match else "")
PY
}

open_url_in_profile() {
  local url="$1"
  if [[ ! -x "$CHROME_BIN" ]]; then
    echo "WARN: Chrome binary not found at: $CHROME_BIN" >&2
    return 0
  fi

  # Do not pass --new-window or `open -n`. A second Chrome instance cannot
  # attach to Profile 8 ("codepd") while that profile is already open.
  "$CHROME_BIN" \
    --profile-directory="$CHROME_PROFILE_DIR" \
    "$url" \
    >/dev/null 2>&1 || true
}

if sts_ok; then
  echo "AWS SSO session is already valid for profile: $PROFILE"
  exit 0
fi

echo "AWS SSO session expired for profile: $PROFILE"
echo "Starting login flow..."

tmp_out="$(mktemp -t aws-sso-login.XXXXXX.log)"
url_opened=0
url=""

cleanup() {
  rm -f "$tmp_out"
}
trap cleanup EXIT

aws sso login --profile "$PROFILE" --region "$REGION" --no-browser >"$tmp_out" 2>&1 &
login_pid=$!

while kill -0 "$login_pid" >/dev/null 2>&1; do
  if [[ -z "$url" ]]; then
    url="$(extract_login_url "$tmp_out" || true)"
  fi

  if [[ "$OPEN_LOGIN_URL" == "1" && "$url_opened" -eq 0 && -n "$url" ]]; then
    echo "Opening AWS SSO URL in Chrome profile: $CHROME_PROFILE_DIR"
    open_url_in_profile "$url"
    url_opened=1
  fi
  sleep 1
done

set +e
wait "$login_pid"
login_rc=$?
set -e

if [[ "$login_rc" -ne 0 ]]; then
  echo "AWS SSO login failed (exit $login_rc). Output:" >&2
  cat "$tmp_out" >&2
  exit "$login_rc"
fi

if ! sts_ok; then
  echo "AWS SSO login command succeeded, but STS validation failed." >&2
  exit 1
fi

echo "AWS SSO login is active for profile: $PROFILE"
