#!/bin/bash
# Script to get frontend deployment values for GitHub Secrets
# Run this after Terraform apply to get the values you need

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$SCRIPT_DIR/../infra/envs/aws/dev"

echo "=========================================="
echo "Frontend Deployment Values for GitHub Secrets"
echo "=========================================="
echo ""

cd "$TERRAFORM_DIR"

# Check if Terraform is initialized
if [ ! -f "terraform.tfstate" ] && [ ! -f ".terraform/terraform.tfstate" ]; then
    echo "⚠️  Warning: Terraform state not found. Make sure you've run 'terraform apply' first."
    echo ""
fi

echo "Getting Terraform outputs..."
echo ""

# Get outputs
BUCKET_NAME=$(terraform output -raw frontend_bucket_name 2>/dev/null || echo "N/A")
DISTRIBUTION_ID=$(terraform output -raw frontend_cloudfront_distribution_id 2>/dev/null || echo "N/A")
ROLE_ARN=$(terraform output -raw github_actions_role_arn 2>/dev/null || echo "N/A")

echo "📋 GitHub Secrets to Configure:"
echo ""
echo "1. AWS_ROLE_TO_ASSUME"
echo "   Value: $ROLE_ARN"
echo ""
echo "2. FRONTEND_S3_BUCKET"
echo "   Value: $BUCKET_NAME"
echo ""
echo "3. CLOUDFRONT_DISTRIBUTION_ID"
echo "   Value: $DISTRIBUTION_ID"
echo ""

if [ "$BUCKET_NAME" != "N/A" ] && [ "$DISTRIBUTION_ID" != "N/A" ] && [ "$ROLE_ARN" != "N/A" ]; then
    echo "✅ All values retrieved successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Go to GitHub repository → Settings → Secrets and variables → Actions"
    echo "2. Add each secret with the values above"
    echo "3. Push a change to react_frontend/ to trigger deployment"
else
    echo "⚠️  Some values could not be retrieved. Make sure:"
    echo "   - Terraform has been applied successfully"
    echo "   - You're in the correct directory"
    echo "   - Terraform outputs are available"
fi

echo ""
echo "=========================================="
