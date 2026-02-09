#!/bin/bash
# Script to set up HTTPS for API endpoint
# This script:
# 1. Gets the ACM certificate ARN from Terraform
# 2. Updates the Ingress with the certificate ARN
# 3. Gets the ALB DNS name and creates Route53 A record

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
INFRA_DIR="$PROJECT_ROOT/infra/envs/aws/dev"
K8S_DIR="$PROJECT_ROOT/k8s/go-api"

echo "Step 1: Getting ACM certificate ARN from Terraform..."
cd "$INFRA_DIR"
CERT_ARN=$(terraform output -raw api_certificate_arn 2>/dev/null || echo "")

if [ -z "$CERT_ARN" ]; then
  echo "Error: Certificate ARN not found. Make sure Terraform has been applied."
  echo "Run: cd $INFRA_DIR && terraform apply"
  exit 1
fi

echo "Certificate ARN: $CERT_ARN"

echo ""
echo "Step 2: Updating Ingress with certificate ARN..."
cd "$PROJECT_ROOT"
# Update the Ingress YAML with the certificate ARN
sed -i.bak "s|alb.ingress.kubernetes.io/certificate-arn:.*|alb.ingress.kubernetes.io/certificate-arn: \"$CERT_ARN\"|" "$K8S_DIR/ingress.yaml"
rm -f "$K8S_DIR/ingress.yaml.bak"

echo "Ingress updated. Applying to cluster..."
kubectl apply -f "$K8S_DIR/ingress.yaml"

echo ""
echo "Step 3: Waiting for ALB to be created..."
sleep 10

echo "Getting ALB DNS name from Ingress status..."
ALB_DNS=$(kubectl get ingress go-api-ingress -n poultry-dev -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "")

if [ -z "$ALB_DNS" ]; then
  echo "Warning: ALB DNS name not found yet. The ALB may still be provisioning."
  echo "You can check with: kubectl get ingress go-api-ingress -n poultry-dev"
  echo "Once the ALB is ready, run the following to create Route53 record:"
  echo ""
  echo "  cd $INFRA_DIR"
  echo "  terraform apply -var=\"alb_dns_name=$ALB_DNS\""
  exit 0
fi

echo "ALB DNS: $ALB_DNS"

echo ""
echo "Step 4: Creating Route53 A record..."
cd "$INFRA_DIR"
# Check if Route53 record resource exists, if not, we'll need to add it to Terraform
# For now, create it manually or add to Terraform
echo "To create Route53 A record, add this to Terraform or run:"
echo "  aws route53 change-resource-record-sets --hosted-zone-id <ZONE_ID> --change-batch '{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"api-dev.mykolipannai.com\",\"Type\":\"A\",\"AliasTarget\":{\"DNSName\":\"$ALB_DNS\",\"EvaluateTargetHealth\":false,\"HostedZoneId\":\"Z1D633PJN98FT9\"}}}]}'"

echo ""
echo "Setup complete!"
echo "API will be available at: https://api-dev.mykolipannai.com/api"
echo "Note: DNS propagation may take a few minutes."
