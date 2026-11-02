#!/bin/zsh
# Refresh the AdminAccess-Pradeep IAM Identity Center session.
# Opens the device-login page in the Chrome profile named by CHROME_PROFILE
# (default: codepd). The SSO token is cached by the AWS CLI in
# ~/.aws/sso/cache (mode 600). Identity Center does not accept a stored
# password; the browser approval is the renewal step.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
opener="$root/scripts/open_aws_sso_chrome.sh"
profile="${AWS_PROFILE:-AdminAccess-Pradeep}"

export BROWSER="$opener %s"
echo "Opening AWS SSO in Chrome profile codepd (${CHROME_PROFILE:-Profile 8})"
echo "Approve the request in that window. This command waits until you do."
exec aws sso login --profile "$profile"
