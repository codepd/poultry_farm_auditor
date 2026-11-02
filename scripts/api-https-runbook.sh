#!/bin/bash
# API HTTPS Setup Runbook - Run this after: aws sso login --profile AdminAccess-Pradeep
# Domain mykolipannai.com is registered. Route53 hosted zone should exist.

set -e
export AWS_PROFILE=AdminAccess-Pradeep
export AWS_SDK_LOAD_CONFIG=1

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
INFRA_DIR="$PROJECT_ROOT/infra/envs/aws/dev"
K8S_DIR="$PROJECT_ROOT/k8s/go-api"

echo "=== Step 1: Apply Terraform (ACM cert + Route53 validation) ==="
cd "$INFRA_DIR"
terraform plan -out=tfplan
echo ""
read -p "Apply this plan? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  terraform apply tfplan
else
  echo "Skipped. Run: terraform apply tfplan when ready."
  exit 0
fi

echo ""
echo "=== Step 2: Get certificate ARN ==="
CERT_ARN=$(terraform output -raw api_certificate_arn 2>/dev/null || echo "")
if [ -z "$CERT_ARN" ]; then
  echo "Error: Certificate ARN not found. Check Terraform outputs."
  exit 1
fi
echo "Certificate ARN: $CERT_ARN"

echo ""
echo "=== Step 3: Update Ingress with certificate ARN ==="
cd "$PROJECT_ROOT"
sed -i.bak "s|alb.ingress.kubernetes.io/certificate-arn:.*|alb.ingress.kubernetes.io/certificate-arn: \"$CERT_ARN\"|" "$K8S_DIR/ingress.yaml"
rm -f "$K8S_DIR/ingress.yaml.bak"
echo "Ingress updated."

echo ""
echo "=== Step 4: Apply Ingress to cluster ==="
kubectl apply -f "$K8S_DIR/ingress.yaml"

echo ""
echo "=== Step 5: Wait for ALB (2-3 min). Checking status... ==="
for i in {1..24}; do
  ALB_DNS=$(kubectl get ingress go-api-ingress -n poultry-dev -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || echo "")
  if [ -n "$ALB_DNS" ]; then
    echo "ALB ready: $ALB_DNS"
    break
  fi
  echo "  Waiting... ($i/24)"
  sleep 10
done

if [ -z "$ALB_DNS" ]; then
  echo "ALB not ready yet. Check: kubectl get ingress go-api-ingress -n poultry-dev"
  echo "Once ready: kubectl get ingress go-api-ingress -n poultry-dev -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'"
  exit 1
fi

echo ""
echo "=== Step 6: Create Route53 A record (update tfvars and apply) ==="
echo "Add to $INFRA_DIR/terraform.tfvars:"
echo "  alb_dns_name = \"$ALB_DNS\""
echo ""
read -p "Apply Terraform to create Route53 A record? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  # Add alb_dns_name to tfvars if not present
  if ! grep -q "alb_dns_name" "$INFRA_DIR/terraform.tfvars"; then
    echo "alb_dns_name = \"$ALB_DNS\"" >> "$INFRA_DIR/terraform.tfvars"
  else
    # Update existing
    sed -i.bak "s|alb_dns_name = .*|alb_dns_name = \"$ALB_DNS\"|" "$INFRA_DIR/terraform.tfvars"
    rm -f "$INFRA_DIR/terraform.tfvars.bak"
  fi
  cd "$INFRA_DIR"
  terraform plan -out=tfplan
  terraform apply tfplan
fi

echo ""
echo "=== Done! ==="
echo "API will be at: https://api-dev.mykolipannai.com/api"
echo "Test: curl https://api-dev.mykolipannai.com/api/health"
echo "DNS propagation may take a few minutes."
