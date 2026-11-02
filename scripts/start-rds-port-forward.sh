#!/usr/bin/env bash
# Start SSM port forward to RDS after ensuring AWS SSO login.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PROFILE="${AWS_PROFILE:-AdminAccess-Pradeep}"
REGION="${AWS_REGION:-ap-south-1}"
DB_HOST="${DB_HOST:-poultry-farm-dev.cx68k0g04oac.ap-south-1.rds.amazonaws.com}"
REMOTE_PORT="${REMOTE_PORT:-5432}"
LOCAL_PORT="${LOCAL_PORT:-15432}"
# kolipannai-dev-bastion. Do not pick "first online instance": another
# instance in this account is not the RDS bastion.
TARGET_INSTANCE_ID="${TARGET_INSTANCE_ID:-i-0a5c2678494e93072}"

"$SCRIPT_DIR/aws-sso-ensure.sh"

if [[ -z "$TARGET_INSTANCE_ID" || "$TARGET_INSTANCE_ID" == "None" ]]; then
  TARGET_INSTANCE_ID="$(
    aws ssm describe-instance-information \
      --profile "$PROFILE" \
      --region "$REGION" \
      --query 'InstanceInformationList[?PingStatus==`Online`]|[0].InstanceId' \
      --output text
  )"
fi

if [[ -z "$TARGET_INSTANCE_ID" || "$TARGET_INSTANCE_ID" == "None" ]]; then
  echo "No online SSM managed instance found. Set TARGET_INSTANCE_ID and retry." >&2
  exit 1
fi

echo "Using SSM target instance: $TARGET_INSTANCE_ID"
echo "Forwarding localhost:$LOCAL_PORT -> $DB_HOST:$REMOTE_PORT"

exec aws ssm start-session \
  --target "$TARGET_INSTANCE_ID" \
  --document-name AWS-StartPortForwardingSessionToRemoteHost \
  --parameters "{\"host\":[\"$DB_HOST\"],\"portNumber\":[\"$REMOTE_PORT\"],\"localPortNumber\":[\"$LOCAL_PORT\"]}" \
  --profile "$PROFILE" \
  --region "$REGION"
